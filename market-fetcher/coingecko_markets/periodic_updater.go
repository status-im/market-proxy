package coingecko_markets

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// PeriodicUpdater handles periodic updates of markets data
type PeriodicUpdater struct {
	config         *config.CoingeckoMarketsFetcher
	scheduler      *scheduler.Scheduler
	marketsFetcher interfaces.CoingeckoMarketsService
	metricsWriter  *metrics.MetricsWriter
	onUpdate       func(ctx context.Context)

	// Cache for markets data
	cache struct {
		sync.RWMutex
		data *APIResponse
	}
}

// NewPeriodicUpdater creates a new periodic markets updater
func NewPeriodicUpdater(cfg *config.CoingeckoMarketsFetcher, marketsFetcher interfaces.CoingeckoMarketsService) *PeriodicUpdater {
	updater := &PeriodicUpdater{
		config:         cfg,
		marketsFetcher: marketsFetcher,
		metricsWriter:  metrics.NewMetricsWriter(metrics.ServiceMarkets),
	}

	return updater
}

// SetOnUpdateCallback sets a callback function that will be called when data is updated
func (u *PeriodicUpdater) SetOnUpdateCallback(onUpdate func(ctx context.Context)) {
	u.onUpdate = onUpdate
}

// GetCacheData returns the current cached markets data
func (u *PeriodicUpdater) GetCacheData() *APIResponse {
	u.cache.RLock()
	defer u.cache.RUnlock()
	return u.cache.data
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

// Start starts the periodic updater with periodic updates
func (u *PeriodicUpdater) Start(ctx context.Context) error {
	updateInterval := u.config.TopMarketsUpdateInterval

	// If interval is 0 or negative, skip periodic updates
	if updateInterval <= 0 {
		log.Printf("Markets periodic updater: periodic updates disabled (interval: %v)", updateInterval)
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
	log.Printf("Started markets periodic updater with update interval: %v", updateInterval)

	return nil
}

// Stop stops the periodic updater
func (u *PeriodicUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

// fetchAndUpdate fetches markets data from markets service and updates cache
func (u *PeriodicUpdater) fetchAndUpdate(ctx context.Context) error {
	// Record start time for metrics
	startTime := time.Now()

	// Get top tokens limit from config, use default if not set
	limit := u.config.TopMarketsLimit
	if limit <= 0 {
		limit = 500 // Default top tokens limit
	}

	// Get currency from config, use "usd" as default
	currency := u.config.Currency
	if currency == "" {
		currency = "usd"
	}

	// Use TopMarkets to get top markets data and cache individual tokens
	data, err := u.marketsFetcher.TopMarkets(limit, currency)
	if err != nil {
		log.Printf("Error fetching top markets data from fetcher: %v", err)
		// Record metrics even on error
		u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
		return err
	}

	// MarketsResponse is already []interface{}, no need for type assertion
	marketsData := []interface{}(data)

	// Convert raw markets data directly to CoinGeckoData using utility method
	convertedData := ConvertMarketsResponseToCoinGeckoData(marketsData)

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
		u.onUpdate(ctx)
	}

	return nil
}

// Healthy checks if the periodic updater can fetch data
func (u *PeriodicUpdater) Healthy() bool {
	// Check if we already have some data in cache
	if u.GetCacheData() != nil && len(u.GetCacheData().Data) > 0 {
		return true
	}

	// Since MarketsFetcher doesn't have Healthy() method,
	// we consider it healthy if we have a fetcher instance
	return u.marketsFetcher != nil
}

// ConvertMarketsResponseToCoinGeckoData converts raw markets response data to CoinGeckoData slice
// This function directly processes the interface{} slice from coins/markets API
func ConvertMarketsResponseToCoinGeckoData(marketsData []interface{}) []CoinGeckoData {
	result := make([]CoinGeckoData, 0, len(marketsData))

	for _, item := range marketsData {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Convert map[string]interface{} to CoinGeckoData directly
		coinData := CoinGeckoData{
			ID:                           getStringFromMap(itemMap, "id"),
			Symbol:                       getStringFromMap(itemMap, "symbol"),
			Name:                         getStringFromMap(itemMap, "name"),
			Image:                        getStringFromMap(itemMap, "image"),
			CurrentPrice:                 getFloatFromMap(itemMap, "current_price"),
			MarketCap:                    getFloatFromMap(itemMap, "market_cap"),
			MarketCapRank:                getIntFromMap(itemMap, "market_cap_rank"),
			FullyDilutedValuation:        getFloatFromMap(itemMap, "fully_diluted_valuation"),
			TotalVolume:                  getFloatFromMap(itemMap, "total_volume"),
			High24h:                      getFloatFromMap(itemMap, "high_24h"),
			Low24h:                       getFloatFromMap(itemMap, "low_24h"),
			PriceChange24h:               getFloatFromMap(itemMap, "price_change_24h"),
			PriceChangePercentage24h:     getFloatFromMap(itemMap, "price_change_percentage_24h"),
			MarketCapChange24h:           getFloatFromMap(itemMap, "market_cap_change_24h"),
			MarketCapChangePercentage24h: getFloatFromMap(itemMap, "market_cap_change_percentage_24h"),
			CirculatingSupply:            getFloatFromMap(itemMap, "circulating_supply"),
			TotalSupply:                  getFloatFromMap(itemMap, "total_supply"),
			MaxSupply:                    getFloatFromMap(itemMap, "max_supply"),
			ATH:                          getFloatFromMap(itemMap, "ath"),
			ATHChangePercentage:          getFloatFromMap(itemMap, "ath_change_percentage"),
			ATHDate:                      getStringFromMap(itemMap, "ath_date"),
			ATL:                          getFloatFromMap(itemMap, "atl"),
			ATLChangePercentage:          getFloatFromMap(itemMap, "atl_change_percentage"),
			ATLDate:                      getStringFromMap(itemMap, "atl_date"),
			ROI:                          itemMap["roi"], // Keep as interface{}
			LastUpdated:                  getStringFromMap(itemMap, "last_updated"),
		}

		result = append(result, coinData)
	}

	return result
}

// Helper function to safely extract string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// Helper function to safely extract float64 from map
func getFloatFromMap(m map[string]interface{}, key string) float64 {
	if value, exists := m[key]; exists {
		if f, ok := value.(float64); ok {
			return f
		}
	}
	return 0.0
}

// Helper function to safely extract int from map
func getIntFromMap(m map[string]interface{}, key string) int {
	if value, exists := m[key]; exists {
		if i, ok := value.(float64); ok {
			return int(i)
		}
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}
