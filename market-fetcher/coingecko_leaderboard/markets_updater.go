package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/scheduler"
)

// MarketsUpdater handles periodic updates of markets leaderboard data
type MarketsUpdater struct {
	config         *config.Config
	scheduler      *scheduler.Scheduler
	marketsFetcher coingecko_common.MarketsFetcher
	onUpdate       func()

	// Cache for markets data
	cache struct {
		sync.RWMutex
		data *APIResponse
	}
}

// NewMarketsUpdater creates a new markets updater
func NewMarketsUpdater(cfg *config.Config, marketsFetcher coingecko_common.MarketsFetcher) *MarketsUpdater {
	updater := &MarketsUpdater{
		config:         cfg,
		marketsFetcher: marketsFetcher,
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
	updateInterval := u.config.CoingeckoLeaderboard.TopMarketsUpdateInterval

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

// fetchAndUpdate fetches markets data from markets service and updates cache
func (u *MarketsUpdater) fetchAndUpdate(ctx context.Context) error {
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

	log.Printf("Updated top markets cache with %d tokens (limit: %d)", len(localData.Data), limit)

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

	// Since MarketsFetcher doesn't have Healthy() method,
	// we consider it healthy if we have a fetcher instance
	return u.marketsFetcher != nil
}
