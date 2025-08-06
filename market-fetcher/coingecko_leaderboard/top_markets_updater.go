package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/metrics"
)

// TopMarketsUpdater handles subscription-based updates of markets leaderboard data
type TopMarketsUpdater struct {
	config             *config.Config
	marketsFetcher     interfaces.CoingeckoMarketsService
	metricsWriter      *metrics.MetricsWriter
	onUpdate           func()
	updateSubscription chan struct{}
	cancelFunc         context.CancelFunc

	// Cache for markets data
	cache struct {
		sync.RWMutex
		data *APIResponse
	}
}

// NewTopMarketsUpdater creates a new top markets updater
func NewTopMarketsUpdater(cfg *config.Config, marketsFetcher interfaces.CoingeckoMarketsService) *TopMarketsUpdater {
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

// Start starts the top markets updater by subscribing to market updates
func (u *TopMarketsUpdater) Start(ctx context.Context) error {
	// Subscribe to market updates from the markets service
	u.updateSubscription = u.marketsFetcher.SubscribeTopMarketsUpdate()

	// Create cancelable context for the subscription handler
	subscriptionCtx, cancel := context.WithCancel(ctx)
	u.cancelFunc = cancel

	// Start subscription handler in a goroutine
	go u.handleMarketUpdates(subscriptionCtx)

	log.Printf("Started top markets updater with subscription to market updates")

	// Perform initial fetch to populate cache
	if err := u.fetchAndUpdate(ctx); err != nil {
		log.Printf("Error during initial markets data fetch: %v", err)
		// Don't return error - subscription can still work for future updates
	}

	return nil
}

// Stop stops the top markets updater
func (u *TopMarketsUpdater) Stop() {
	// Cancel the subscription handler goroutine
	if u.cancelFunc != nil {
		u.cancelFunc()
		u.cancelFunc = nil
	}

	// Unsubscribe from market updates
	if u.updateSubscription != nil && u.marketsFetcher != nil {
		u.marketsFetcher.Unsubscribe(u.updateSubscription)
		u.updateSubscription = nil
	}
}

// handleMarketUpdates handles subscription to market updates
func (u *TopMarketsUpdater) handleMarketUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-u.updateSubscription:
			// Market data has been updated, fetch new data
			if err := u.fetchAndUpdate(ctx); err != nil {
				log.Printf("Error updating markets data on subscription signal: %v", err)
			}
		}
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
