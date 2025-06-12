package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

const (
	// Maximum items per page
	MAX_PER_PAGE = 250 // CoinGecko's API max per_page value
)

// MarketsUpdater handles periodic updates of markets leaderboard data
type MarketsUpdater struct {
	config        *config.Config
	scheduler     *scheduler.Scheduler
	apiClient     *CoinGeckoClient
	fetcher       *PaginatedFetcher
	onUpdate      func()
	metricsWriter *metrics.MetricsWriter

	// Cache for markets data
	cache struct {
		sync.RWMutex
		data *APIResponse
	}
}

// NewMarketsUpdater creates a new markets updater
func NewMarketsUpdater(cfg *config.Config) *MarketsUpdater {
	// Create API client
	apiClient := NewCoinGeckoClient(cfg)

	// Create paginated fetcher with the API client
	requestDelayMs := int(cfg.CoingeckoLeaderboard.RequestDelay.Milliseconds())
	fetcher := NewPaginatedFetcher(apiClient, cfg.CoingeckoLeaderboard.Limit, MAX_PER_PAGE, requestDelayMs)

	updater := &MarketsUpdater{
		config:        cfg,
		apiClient:     apiClient,
		fetcher:       fetcher,
		metricsWriter: metrics.NewMetricsWriter(metrics.ServiceLBMarkets),
	}

	return updater
}

// SetOnUpdateCallback sets a callback function that will be called when data is updated
func (u *MarketsUpdater) SetOnUpdateCallback(onUpdate func()) {
	u.onUpdate = onUpdate
}

// GetCacheData returns the current cached markets data
func (u *MarketsUpdater) GetCacheData() *APIResponse {
	u.cache.RLock()
	defer u.cache.RUnlock()
	return u.cache.data
}

// GetTopTokenIDs extracts token IDs from cached data for use by other components
func (u *MarketsUpdater) GetTopTokenIDs() []string {
	cacheData := u.GetCacheData()
	if cacheData == nil || cacheData.Data == nil {
		return nil
	}

	// Extract token IDs from cached data
	tokenIDs := make([]string, 0, len(cacheData.Data))
	for _, coinData := range cacheData.Data {
		if coinData.ID != "" {
			tokenIDs = append(tokenIDs, coinData.ID)
		}
	}

	return tokenIDs
}

// Start starts the markets updater with periodic updates
func (u *MarketsUpdater) Start(ctx context.Context) error {
	updateInterval := u.config.CoingeckoLeaderboard.UpdateInterval

	// Create scheduler for periodic updates
	u.scheduler = scheduler.New(
		updateInterval,
		func(ctx context.Context) {
			if err := u.fetchAndUpdate(ctx); err != nil {
				log.Printf("Error updating markets data: %v", err)
			}
		},
	)

	// Start the scheduler with context
	u.scheduler.Start(ctx, true)
	log.Printf("Started markets updater with update interval: %v", updateInterval)

	return nil
}

// Stop stops the markets updater
func (u *MarketsUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

// fetchAndUpdate fetches markets data from CoinGecko and updates cache
func (u *MarketsUpdater) fetchAndUpdate(ctx context.Context) error {
	// Reset request cycle counters
	u.metricsWriter.ResetCycleMetrics()

	// Record start time for metrics
	startTime := time.Now()

	// Perform the fetch operation
	data, err := u.fetcher.FetchData()

	// Record metrics regardless of success or failure
	u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))

	if err != nil {
		log.Printf("Error fetching markets data: %v", err)
		return err
	}

	// Update cache
	u.cache.Lock()
	u.cache.data = data
	u.cache.Unlock()

	// Record cache size metric
	if data != nil && data.Data != nil {
		u.metricsWriter.RecordCacheSize(len(data.Data))
	}

	log.Printf("Updated markets cache with %d tokens", len(data.Data))

	// Signal update through callback
	if u.onUpdate != nil {
		u.onUpdate()
	}

	return nil
}

// Healthy checks if the markets updater can fetch data
func (u *MarketsUpdater) Healthy() bool {
	// Check if we already have some data in cache
	if u.GetCacheData() != nil && len(u.GetCacheData().Data) > 0 {
		return true
	}

	// If not, try to fetch at least one page
	return u.apiClient.Healthy()
}
