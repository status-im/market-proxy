package coingecko_markets

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// TierScheduler represents a scheduler for a specific tier
type TierScheduler struct {
	tier      config.MarketTier
	scheduler *scheduler.Scheduler
}

// PeriodicUpdater handles periodic updates of markets data
type PeriodicUpdater struct {
	config        *config.CoingeckoMarketsFetcher
	schedulers    []*TierScheduler // Multiple schedulers for different tiers
	apiClient     APIClient
	metricsWriter *metrics.MetricsWriter
	onUpdate      func(ctx context.Context, tokensData [][]byte)

	// Cache for markets data per tier
	cache struct {
		sync.RWMutex
		tiers map[string]*APIResponse // tier name -> data
	}
}

// NewPeriodicUpdater creates a new periodic markets updater
func NewPeriodicUpdater(cfg *config.CoingeckoMarketsFetcher, apiClient APIClient) *PeriodicUpdater {
	updater := &PeriodicUpdater{
		config:        cfg,
		apiClient:     apiClient,
		metricsWriter: metrics.NewMetricsWriter(metrics.ServiceMarkets),
	}

	// Initialize tier cache
	updater.cache.tiers = make(map[string]*APIResponse)

	return updater
}

// SetOnUpdateCallback sets a callback function that will be called when data is updated
func (u *PeriodicUpdater) SetOnUpdateCallback(onUpdate func(ctx context.Context, tokensData [][]byte)) {
	u.onUpdate = onUpdate
}

// GetCacheData returns the current cached markets data
// Combines data from all tiers
func (u *PeriodicUpdater) GetCacheData() *APIResponse {
	u.cache.RLock()
	defer u.cache.RUnlock()

	// Combine data from all tiers
	var allData []CoinGeckoData
	for _, tierData := range u.cache.tiers {
		if tierData != nil && tierData.Data != nil {
			allData = append(allData, tierData.Data...)
		}
	}

	if len(allData) == 0 {
		return nil
	}

	return &APIResponse{Data: allData}
}

// GetCacheDataForTier returns cached data for a specific tier
func (u *PeriodicUpdater) GetCacheDataForTier(tierName string) *APIResponse {
	u.cache.RLock()
	defer u.cache.RUnlock()
	return u.cache.tiers[tierName]
}

// GetTopTokenIDs extracts token IDs from cached data for use by other components
func (u *PeriodicUpdater) GetTopTokenIDs() []string {
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

func (u *PeriodicUpdater) Start(ctx context.Context) error {
	if err := u.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return u.startAllTiers(ctx)
}

// startAllTiers starts multiple schedulers
func (u *PeriodicUpdater) startAllTiers(ctx context.Context) error {
	log.Printf("Starting markets periodic updater in tier mode with %d tiers", len(u.config.Tiers))

	u.schedulers = make([]*TierScheduler, 0, len(u.config.Tiers))

	for _, tier := range u.config.Tiers {
		// Create a copy of the tier for closure
		tierCopy := tier

		tierScheduler := &TierScheduler{
			tier: tierCopy,
			scheduler: scheduler.New(
				tierCopy.UpdateInterval,
				func(ctx context.Context) {
					if err := u.fetchAndUpdateTier(ctx, tierCopy); err != nil {
						log.Printf("Error updating tier '%s' data: %v", tierCopy.Name, err)
					}
				},
			),
		}

		u.schedulers = append(u.schedulers, tierScheduler)

		// Start the scheduler with context
		tierScheduler.scheduler.Start(ctx, true)
		log.Printf("Started tier '%s' scheduler: page [%d-%d], interval: %v",
			tierCopy.Name, tierCopy.PageFrom, tierCopy.PageTo, tierCopy.UpdateInterval)
	}

	return nil
}

// Stop stops the periodic updater
func (u *PeriodicUpdater) Stop() {
	for _, tierScheduler := range u.schedulers {
		if tierScheduler.scheduler != nil {
			tierScheduler.scheduler.Stop()
		}
	}
}

// fetchAndUpdateTier fetches markets data for a specific tier and updates cache
func (u *PeriodicUpdater) fetchAndUpdateTier(ctx context.Context, tier config.MarketTier) error {
	// Record start time for metrics
	startTime := time.Now()

	// Get request delay from config
	requestDelay := u.config.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = MARKETS_DEFAULT_REQUEST_DELAY
	}

	// Create parameters for top markets request
	params := interfaces.MarketsParams{}

	// Apply parameters normalization from config
	params = ApplyParamsOverride(params, u.config)

	// Create PaginatedFetcher with parameters
	fetcher := NewPaginatedFetcher(u.apiClient, tier.PageFrom, tier.PageTo, requestDelayMs, params)

	// Use PaginatedFetcher to get markets data as [][]byte
	tokensData, err := fetcher.FetchData()
	if err != nil {
		log.Printf("PaginatedFetcher failed to fetch top markets data for tier '%s': %v", tier.Name, err)
		// Record metrics even on error
		u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
		return err
	}

	// Convert raw markets data directly to CoinGeckoData using utility method
	convertedData := ConvertMarketsResponseToCoinGeckoData(tokensData)

	localData := &APIResponse{
		Data: convertedData,
	}

	// Update cache for this tier
	u.cache.Lock()
	u.cache.tiers[tier.Name] = localData
	u.cache.Unlock()

	// Record metrics after successful update
	u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
	u.metricsWriter.RecordCacheSize(len(convertedData))

	log.Printf("Updated tier '%s' cache with %d tokens (page: %d-%d)",
		tier.Name, len(convertedData), tier.PageFrom, tier.PageTo)

	// Signal update through callback with raw token data
	if u.onUpdate != nil {
		u.onUpdate(ctx, tokensData)
	}

	return nil
}

// Healthy checks if the periodic updater can fetch data
func (u *PeriodicUpdater) Healthy() bool {
	// Check if we already have some data in cache
	if u.GetCacheData() != nil && len(u.GetCacheData().Data) > 0 {
		return true
	}

	// Check if apiClient is healthy
	return u.apiClient != nil && u.apiClient.Healthy()
}
