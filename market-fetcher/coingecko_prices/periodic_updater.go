package coingecko_prices

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

//go:generate mockgen -destination=mocks/periodic_updater.go . PeriodicUpdaterInterface

// PeriodicUpdaterInterface defines the interface for periodic price updater
type PeriodicUpdaterInterface interface {
	Start(ctx context.Context) error
	Stop()
	SetTopMarketIds(ids []string)
	SetExtraIds(ids []string)
	SetOnTopPricesUpdatedCallback(callback func(ctx context.Context, tier config.PriceTier, pricesData map[string][]byte))
	SetOnMissingExtraIdsUpdatedCallback(callback func(ctx context.Context, pricesData map[string][]byte))
	GetCacheData() map[string][]byte
	GetCacheDataForTier(tierName string) map[string][]byte
	Healthy() bool
}

// TierScheduler represents a scheduler for a specific tier
type TierScheduler struct {
	tier      config.PriceTier
	scheduler *scheduler.Scheduler
}

// TierDataWithTimestamp holds tier data with timestamp for staleness checks
type TierDataWithTimestamp struct {
	Data      map[string][]byte
	Timestamp time.Time
}

// PeriodicUpdater handles periodic updates of prices data
type PeriodicUpdater struct {
	config                   *config.CoingeckoPricesFetcher
	schedulers               []*TierScheduler // Multiple schedulers for different tiers
	apiClient                APIClient
	metricsWriter            *metrics.MetricsWriter
	onTopPricesUpdated       func(ctx context.Context, tier config.PriceTier, pricesData map[string][]byte)
	onMissingExtraIdsUpdated func(ctx context.Context, pricesData map[string][]byte)

	// Cache for prices data per tier with timestamps
	cache struct {
		sync.RWMutex
		tiers map[string]*TierDataWithTimestamp // tier name -> data with timestamp
	}

	// Top market IDs to fetch for tiers
	topMarketIds struct {
		sync.RWMutex
		ids []string
	}

	// Extra IDs to fetch
	extraIds struct {
		sync.RWMutex
		ids []string
	}
}

// NewPeriodicUpdater creates a new periodic prices updater
func NewPeriodicUpdater(cfg *config.CoingeckoPricesFetcher, apiClient APIClient) *PeriodicUpdater {
	updater := &PeriodicUpdater{
		config:        cfg,
		apiClient:     apiClient,
		metricsWriter: metrics.NewMetricsWriter(metrics.ServicePrices),
	}

	// Initialize tier cache
	updater.cache.tiers = make(map[string]*TierDataWithTimestamp)

	return updater
}

// SetOnTopPricesUpdatedCallback sets a callback function that will be called when tier data is updated
func (u *PeriodicUpdater) SetOnTopPricesUpdatedCallback(callback func(ctx context.Context, tier config.PriceTier, pricesData map[string][]byte)) {
	u.onTopPricesUpdated = callback
}

// SetOnMissingExtraIdsUpdatedCallback sets a callback function that will be called when missing extra IDs are updated
func (u *PeriodicUpdater) SetOnMissingExtraIdsUpdatedCallback(callback func(ctx context.Context, pricesData map[string][]byte)) {
	u.onMissingExtraIdsUpdated = callback
}

// SetTopMarketIds sets the list of top market token IDs to fetch for tiers
func (u *PeriodicUpdater) SetTopMarketIds(ids []string) {
	u.topMarketIds.Lock()
	defer u.topMarketIds.Unlock()
	u.topMarketIds.ids = make([]string, len(ids))
	copy(u.topMarketIds.ids, ids)
	log.Printf("Updated top market IDs list with %d tokens", len(ids))
}

// SetExtraIds sets the list of extra token IDs to fetch
func (u *PeriodicUpdater) SetExtraIds(ids []string) {
	u.extraIds.Lock()
	defer u.extraIds.Unlock()
	u.extraIds.ids = make([]string, len(ids))
	copy(u.extraIds.ids, ids)
	log.Printf("Updated extra IDs list with %d tokens", len(ids))
}

// GetCacheData returns the current cached prices data
// Combines data from all tiers
func (u *PeriodicUpdater) GetCacheData() map[string][]byte {
	u.cache.RLock()
	defer u.cache.RUnlock()

	// Combine data from all tiers
	allData := make(map[string][]byte)
	for _, tierData := range u.cache.tiers {
		if tierData != nil && tierData.Data != nil {
			for tokenID, data := range tierData.Data {
				allData[tokenID] = data
			}
		}
	}

	return allData
}

// GetCacheDataForTier returns cached data for a specific tier
func (u *PeriodicUpdater) GetCacheDataForTier(tierName string) map[string][]byte {
	u.cache.RLock()
	defer u.cache.RUnlock()
	tierData := u.cache.tiers[tierName]
	if tierData == nil {
		return nil
	}
	return tierData.Data
}

// GetCacheDataForTierWithTimestamp returns cached data with timestamp for a specific tier
func (u *PeriodicUpdater) GetCacheDataForTierWithTimestamp(tierName string) *TierDataWithTimestamp {
	u.cache.RLock()
	defer u.cache.RUnlock()
	return u.cache.tiers[tierName]
}

// Start starts the periodic updater
func (u *PeriodicUpdater) Start(ctx context.Context) error {
	if err := u.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return u.startAllTiers(ctx)
}

// startAllTiers starts multiple schedulers
func (u *PeriodicUpdater) startAllTiers(ctx context.Context) error {
	log.Printf("Starting prices periodic updater in tier mode with %d tiers", len(u.config.Tiers))

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
		log.Printf("Started tier '%s' scheduler: token [%d-%d], interval: %v",
			tierCopy.Name, tierCopy.TokenFrom, tierCopy.TokenTo, tierCopy.UpdateInterval)
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

// fetchAndUpdateTier fetches prices data for a specific tier and updates cache
func (u *PeriodicUpdater) fetchAndUpdateTier(ctx context.Context, tier config.PriceTier) error {
	startTime := time.Now()

	// Get top market IDs for this tier
	u.topMarketIds.RLock()
	topMarketIds := make([]string, len(u.topMarketIds.ids))
	copy(topMarketIds, u.topMarketIds.ids)
	u.topMarketIds.RUnlock()

	if len(topMarketIds) == 0 {
		log.Printf("No top market IDs available for tier '%s'", tier.Name)
		return nil
	}

	// Calculate range for this tier
	fromIndex := tier.TokenFrom - 1 // Convert to 0-based
	toIndex := tier.TokenTo - 1

	if fromIndex >= len(topMarketIds) {
		log.Printf("Tier '%s' token_from (%d) exceeds available tokens (%d)", tier.Name, tier.TokenFrom, len(topMarketIds))
		return nil
	}

	if toIndex >= len(topMarketIds) {
		toIndex = len(topMarketIds) - 1
		log.Printf("Tier '%s' token_to (%d) exceeds available tokens (%d), adjusted to %d", tier.Name, tier.TokenTo, len(topMarketIds), toIndex+1)
	}

	// Get token IDs for this tier
	tierTokenIds := topMarketIds[fromIndex : toIndex+1]

	log.Printf("Fetching prices for tier '%s' with %d tokens", tier.Name, len(tierTokenIds))

	// Merge config currencies
	allCurrencies := u.getConfigCurrencies()

	// Prepare parameters for fetching prices
	fetchParams := interfaces.PriceParams{
		IDs:                  tierTokenIds,
		Currencies:           allCurrencies,
		IncludeMarketCap:     true,
		Include24hrVol:       true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
	}

	// Fetch prices using chunks fetcher
	requestDelay := u.config.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = DEFAULT_REQUEST_DELAY
	}

	chunkSize := u.config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = DEFAULT_CHUNK_SIZE
	}

	fetcher := NewChunksFetcher(u.apiClient, chunkSize, requestDelayMs)
	pricesData, err := fetcher.FetchPrices(ctx, fetchParams)
	if err != nil {
		log.Printf("ChunksFetcher failed to fetch prices data for tier '%s': %v", tier.Name, err)
		u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
		return err
	}

	localData := &TierDataWithTimestamp{
		Data:      pricesData,
		Timestamp: time.Now(),
	}

	// Update cache for this tier
	u.cache.Lock()
	u.cache.tiers[tier.Name] = localData
	u.cache.Unlock()

	// Fetch missing coinslist IDs if enabled for this tier
	if tier.FetchCoinslistIds {
		missingPricesData, err := u.fetchMissingExtraIds(ctx, tier)
		if err != nil {
			log.Printf("Failed to fetch missing extra IDs for tier '%s': %v", tier.Name, err)
		} else if len(missingPricesData) > 0 && u.onMissingExtraIdsUpdated != nil {
			// Signal update through callback with missing prices data
			u.onMissingExtraIdsUpdated(ctx, missingPricesData)
		}
	}

	// Record metrics after successful update
	u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
	u.metricsWriter.RecordCacheSize(len(pricesData))

	log.Printf("Updated tier '%s' cache with %d tokens (token range: %d-%d)",
		tier.Name, len(pricesData), tier.TokenFrom, tier.TokenTo)

	// Signal update through callback with prices data and tier information
	if u.onTopPricesUpdated != nil {
		u.onTopPricesUpdated(ctx, tier, pricesData)
	}

	return nil
}

// fetchMissingExtraIds fetches extra IDs that are missing or stale in cache
func (u *PeriodicUpdater) fetchMissingExtraIds(ctx context.Context, tier config.PriceTier) (map[string][]byte, error) {
	u.extraIds.RLock()
	extraIds := make([]string, len(u.extraIds.ids))
	copy(extraIds, u.extraIds.ids)
	u.extraIds.RUnlock()

	if len(extraIds) == 0 {
		return nil, nil
	}

	// Find IDs that are missing or have stale data (older than half TTL)
	missingIds := u.findMissingOrStaleIds(extraIds)
	if len(missingIds) == 0 {
		log.Printf("All extra IDs are fresh in cache for tier '%s'", tier.Name)
		return nil, nil
	}

	log.Printf("Fetching %d missing/stale extra IDs for tier '%s'", len(missingIds), tier.Name)

	// Merge config currencies
	allCurrencies := u.getConfigCurrencies()

	// Prepare parameters for fetching missing IDs
	fetchParams := interfaces.PriceParams{
		IDs:                  missingIds,
		Currencies:           allCurrencies,
		IncludeMarketCap:     true,
		Include24hrVol:       true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
	}

	// Use chunks fetcher to handle large number of missing IDs
	requestDelay := u.config.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = DEFAULT_REQUEST_DELAY
	}

	chunkSize := u.config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = DEFAULT_CHUNK_SIZE
	}

	chunksFetcher := NewChunksFetcher(u.apiClient, chunkSize, requestDelayMs)

	// Fetch missing IDs data using chunks fetcher
	pricesData, err := chunksFetcher.FetchPrices(ctx, fetchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch missing extra IDs: %w", err)
	}

	// The fetched data will be cached by the service layer through the callback
	if len(pricesData) > 0 {
		log.Printf("Successfully fetched %d missing extra IDs for tier '%s' using chunks fetcher", len(pricesData), tier.Name)
	}

	return pricesData, nil
}

// findMissingOrStaleIds finds IDs that are missing from all tiers or have stale data
func (u *PeriodicUpdater) findMissingOrStaleIds(extraIds []string) []string {
	u.cache.RLock()
	defer u.cache.RUnlock()

	halfTTL := u.config.GetTTL() / 2
	now := time.Now()
	missingIds := make([]string, 0)

	for _, id := range extraIds {
		found := false
		isStale := false

		// Check all tiers for this ID
		for _, tierData := range u.cache.tiers {
			if tierData == nil {
				continue
			}

			// Check if ID exists in this tier
			if _, exists := tierData.Data[id]; exists {
				found = true
				// Check if data is stale (older than half TTL)
				if now.Sub(tierData.Timestamp) > halfTTL {
					isStale = true
				}
				break
			}
		}

		// Add to missing list if not found or stale
		if !found || isStale {
			missingIds = append(missingIds, id)
		}
	}

	return missingIds
}

// getConfigCurrencies returns the currencies from config, with fallback to default
func (u *PeriodicUpdater) getConfigCurrencies() []string {
	if u.config != nil && len(u.config.Currencies) > 0 {
		return u.config.Currencies
	}
	// Fallback to default currencies if config is not available or empty
	return []string{"usd", "eur", "btc", "eth"}
}

// Healthy checks if the periodic updater can fetch data
func (u *PeriodicUpdater) Healthy() bool {
	// Check if we already have some data in cache
	if cacheData := u.GetCacheData(); len(cacheData) > 0 {
		return true
	}

	// Check if apiClient is healthy
	return u.apiClient != nil && u.apiClient.Healthy()
}
