package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// TopMarketsUpdater handles periodic updates of markets leaderboard data
type TopMarketsUpdater struct {
	config         *config.Config
	scheduler      *scheduler.Scheduler
	marketsFetcher coingecko_common.MarketsFetcher
	metricsWriter  *metrics.MetricsWriter
	onUpdate       func()

	// Cache for markets data
	cache struct {
		sync.RWMutex
		data *APIResponse
	}
}

// NewTopMarketsUpdater creates a new top markets updater
func NewTopMarketsUpdater(cfg *config.Config, marketsFetcher coingecko_common.MarketsFetcher) *TopMarketsUpdater {
	updater := &TopMarketsUpdater{
		config:         cfg,
		marketsFetcher: marketsFetcher,
		metricsWriter:  metrics.NewMetricsWriter(metrics.ServiceLBMarkets),
	}

	return updater
}

// SetOnUpdateCallback sets a callback function that will be called when data is updated
func (u *TopMarketsUpdater) SetOnUpdateCallback(onUpdate func()) {
	u.onUpdate = onUpdate
}

// GetCacheData returns the current cached markets data
func (u *TopMarketsUpdater) GetCacheData() *APIResponse {
	u.cache.RLock()
	defer u.cache.RUnlock()
	return u.cache.data
}

// GetTopTokenIDs extracts token IDs from cached data for use by other components
func (u *TopMarketsUpdater) GetTopTokenIDs() []string {
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

// Start starts the top markets updater with periodic updates
func (u *TopMarketsUpdater) Start(ctx context.Context) error {
	updateInterval := u.config.CoingeckoLeaderboard.TopMarketsUpdateInterval

	// If interval is 0 or negative, skip periodic updates
	if updateInterval <= 0 {
		log.Printf("Top markets updater: periodic updates disabled (interval: %v)", updateInterval)
		return nil
	}

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
	log.Printf("Started top markets updater with update interval: %v", updateInterval)

	return nil
}

// Stop stops the top markets updater
func (u *TopMarketsUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

// fetchAndUpdate fetches markets data from markets service and updates cache
func (u *TopMarketsUpdater) fetchAndUpdate(ctx context.Context) error {
	// Record start time for metrics
	startTime := time.Now()

	// Get top tokens limit from config, use default if not set
	limit := u.config.CoingeckoLeaderboard.TopPricesLimit
	if limit <= 0 {
		limit = 500 // Default top tokens limit
	}

	// Use TopMarkets to get top markets data and cache individual tokens
	// Get currency from config, use "usd" as default
	currency := u.config.CoingeckoLeaderboard.Currency
	if currency == "" {
		currency = "usd"
	}

	data, err := u.marketsFetcher.TopMarkets(limit, currency)
	if err != nil {
		log.Printf("Error fetching top markets data from fetcher: %v", err)
		// Record metrics even on error
		u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
		return err
	}

	// MarketsResponse is already []interface{}, no need for type assertion
	marketsData := []interface{}(data)

	// Convert raw markets data directly to CoinData using the new utility method
	convertedData := ConvertMarketsResponseToCoinData(marketsData)

	localData := &APIResponse{
		Data: convertedData,
	}

	// Update cache
	u.cache.Lock()
	u.cache.data = localData
	u.cache.Unlock()

	// Record metrics after successful update
	u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
	u.metricsWriter.RecordCacheSize(len(localData.Data))

	log.Printf("Updated top markets cache with %d tokens (limit: %d)", len(localData.Data), limit)

	// Signal update through callback
	if u.onUpdate != nil {
		u.onUpdate()
	}

	return nil
}

// Healthy checks if the top markets updater can fetch data
func (u *TopMarketsUpdater) Healthy() bool {
	// Check if we already have some data in cache
	if u.GetCacheData() != nil && len(u.GetCacheData().Data) > 0 {
		return true
	}

	// Since MarketsFetcher doesn't have Healthy() method,
	// we consider it healthy if we have a fetcher instance
	return u.marketsFetcher != nil
}
