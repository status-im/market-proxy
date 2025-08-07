package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/metrics"
)

// TopMarketsUpdater handles subscription-based updates of markets leaderboard data
type TopMarketsUpdater struct {
	config             *config.LeaderboardFetcherConfig
	marketsFetcher     interfaces.IMarketsService
	metricsWriter      *metrics.MetricsWriter
	onUpdate           func()
	updateSubscription events.ISubscription

	cache struct {
		sync.RWMutex
		data *APIResponse
	}
}

func NewTopMarketsUpdater(cfg *config.LeaderboardFetcherConfig, marketsFetcher interfaces.IMarketsService) *TopMarketsUpdater {
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

// Start starts the top markets updater by subscribing to market updates
func (u *TopMarketsUpdater) Start(ctx context.Context) error {
	u.updateSubscription = u.marketsFetcher.SubscribeTopMarketsUpdate().
		Watch(ctx, func() {
			if err := u.fetchAndUpdate(ctx); err != nil {
				log.Printf("Error updating markets data on subscription signal: %v", err)
			}
		}, true)

	log.Printf("Started top markets updater with subscription to market updates")

	return nil
}

// Stop stops the top markets updater
func (u *TopMarketsUpdater) Stop() {
	if u.updateSubscription != nil {
		u.updateSubscription.Cancel()
		u.updateSubscription = nil
	}
}

// fetchAndUpdate fetches markets data from markets service and updates cache
func (u *TopMarketsUpdater) fetchAndUpdate(ctx context.Context) error {
	defer u.metricsWriter.TrackDataFetchCycle()()

	limit := u.config.TopMarketsLimit
	if limit <= 0 {
		limit = 500 // Default limit
	}
	data, err := u.marketsFetcher.TopMarkets(limit, u.config.Currency)
	if err != nil {
		log.Printf("Error fetching top markets data from fetcher: %v", err)
		return err
	}

	marketsData := []interface{}(data)
	convertedData := ConvertMarketsResponseToCoinData(marketsData)

	localData := &APIResponse{
		Data: convertedData,
	}

	u.cache.Lock()
	u.cache.data = localData
	u.cache.Unlock()

	u.metricsWriter.RecordCacheSize(len(localData.Data))

	log.Printf("Updated top markets cache with %d tokens (limit: %d)", len(localData.Data), limit)
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

	return u.marketsFetcher != nil
}
