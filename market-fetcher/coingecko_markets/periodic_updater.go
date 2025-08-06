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
	config                  *config.CoingeckoMarketsFetcher
	schedulers              []*TierScheduler // Multiple schedulers for different tiers
	apiClient               APIClient
	metricsWriter           *metrics.MetricsWriter
	onUpdateTierPages       func(ctx context.Context, tier config.MarketTier, pagesData []PageData)
	onUpdateMissingExtraIds func(ctx context.Context, tokensData [][]byte)

	// Cache for markets data per tier with timestamps
	cache struct {
		sync.RWMutex
		tiers map[string]*TierDataWithTimestamp // tier name -> data with timestamp
	}

	// Extra IDs to fetch
	extraIds struct {
		sync.RWMutex
		ids []string
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
	updater.cache.tiers = make(map[string]*TierDataWithTimestamp)

	return updater
}

// SetOnUpdateTierPagesCallback sets a callback function that will be called when tier data is updated
func (u *PeriodicUpdater) SetOnUpdateTierPagesCallback(onUpdateTierPages func(ctx context.Context, tier config.MarketTier, pagesData []PageData)) {
	u.onUpdateTierPages = onUpdateTierPages
}

// SetOnUpdateMissingExtraIdsCallback sets a callback function that will be called when missing extra IDs are updated
func (u *PeriodicUpdater) SetOnUpdateMissingExtraIdsCallback(onUpdateMissingExtraIds func(ctx context.Context, tokensData [][]byte)) {
	u.onUpdateMissingExtraIds = onUpdateMissingExtraIds
}

// SetExtraIds sets the list of extra token IDs to fetch
func (u *PeriodicUpdater) SetExtraIds(ids []string) {
	u.extraIds.Lock()
	defer u.extraIds.Unlock()
	u.extraIds.ids = make([]string, len(ids))
	copy(u.extraIds.ids, ids)
	log.Printf("Updated extra IDs list with %d tokens", len(ids))
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
	tierData := u.cache.tiers[tierName]
	if tierData == nil {
		return nil
	}
	return &APIResponse{Data: tierData.Data}
}

// GetCacheDataForTierWithTimestamp returns cached data with timestamp for a specific tier
func (u *PeriodicUpdater) GetCacheDataForTierWithTimestamp(tierName string) *TierDataWithTimestamp {
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
	startTime := time.Now()

	requestDelay := u.config.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = MARKETS_DEFAULT_REQUEST_DELAY
	}

	params := interfaces.MarketsParams{}
	params = ApplyParamsOverride(params, u.config)

	fetcher := NewPaginatedFetcher(u.apiClient, tier.PageFrom, tier.PageTo, requestDelayMs, params)

	pagesData, err := fetcher.FetchPages()
	if err != nil {
		log.Printf("PaginatedFetcher failed to fetch top markets data for tier '%s': %v", tier.Name, err)
		u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
		return err
	}

	// Flatten pages data for cache storage
	var tokensData [][]byte
	for _, pageData := range pagesData {
		tokensData = append(tokensData, pageData.Data...)
	}

	// Convert raw markets data directly to CoinGeckoData using utility method
	convertedData := ConvertMarketsResponseToCoinGeckoData(tokensData)

	localData := &TierDataWithTimestamp{
		Data:      convertedData,
		Timestamp: time.Now(),
	}

	// Update cache for this tier
	u.cache.Lock()
	u.cache.tiers[tier.Name] = localData
	u.cache.Unlock()

	// Fetch missing coinslist IDs if enabled for this tier
	if tier.FetchCoinslistIds {
		missingTokensData, err := u.fetchMissingExtraIds(ctx, tier)
		if err != nil {
			log.Printf("Failed to fetch missing extra IDs for tier '%s': %v", tier.Name, err)
		} else if len(missingTokensData) > 0 && u.onUpdateMissingExtraIds != nil {
			// Signal update through callback with missing tokens data
			u.onUpdateMissingExtraIds(ctx, missingTokensData)
		}
	}

	// Record metrics after successful update
	u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
	u.metricsWriter.RecordCacheSize(len(convertedData))

	log.Printf("Updated tier '%s' cache with %d tokens (page: %d-%d)",
		tier.Name, len(convertedData), tier.PageFrom, tier.PageTo)

	// Signal update through callback with pages data and tier information
	if u.onUpdateTierPages != nil {
		u.onUpdateTierPages(ctx, tier, pagesData)
	}

	return nil
}

// fetchMissingExtraIds fetches extra IDs that are missing or stale in cache
func (u *PeriodicUpdater) fetchMissingExtraIds(ctx context.Context, tier config.MarketTier) ([][]byte, error) {
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

	// Prepare parameters for fetching missing IDs
	params := interfaces.MarketsParams{
		IDs: missingIds,
	}
	params = ApplyParamsOverride(params, u.config)

	// Use chunks fetcher to handle large number of missing IDs
	requestDelay := u.config.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = MARKETS_DEFAULT_REQUEST_DELAY
	}

	// Create chunks fetcher with appropriate chunk size
	chunkSize := CHUNKS_DEFAULT_CHUNK_SIZE
	if params.PerPage > 0 && params.PerPage < chunkSize {
		chunkSize = params.PerPage
	}

	chunksFetcher := NewChunksFetcher(u.apiClient, chunkSize, requestDelayMs)

	// Fetch missing IDs data using chunks fetcher
	tokensData, err := chunksFetcher.FetchMarkets(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch missing extra IDs: %w", err)
	}

	// The fetched data will be cached by the service layer through the callback
	// Here we just update our tier data if needed
	if len(tokensData) > 0 {
		log.Printf("Successfully fetched %d missing extra IDs for tier '%s' using chunks fetcher", len(tokensData), tier.Name)
	}

	return tokensData, nil
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
			for _, coinData := range tierData.Data {
				if coinData.ID == id {
					found = true
					// Check if data is stale (older than half TTL)
					if now.Sub(tierData.Timestamp) > halfTTL {
						isStale = true
					}
					break
				}
			}

			if found && !isStale {
				break // ID found and not stale, no need to check other tiers
			}
		}

		// Add to missing list if not found or stale
		if !found || isStale {
			missingIds = append(missingIds, id)
		}
	}

	return missingIds
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
