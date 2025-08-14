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

// PeriodicUpdater handles periodic updates of markets data
type PeriodicUpdater struct {
	config                  *config.MarketsFetcherConfig
	scheduler               *scheduler.Scheduler // Single scheduler for all tiers
	apiClient               IAPIClient
	metricsWriter           *metrics.MetricsWriter
	onUpdateTierPages       func(ctx context.Context, tier config.MarketTier, pagesData []PageData)
	onUpdateMissingExtraIds func(ctx context.Context, tokensData [][]byte)
	onInitialLoadCompleted  func(ctx context.Context)

	// Cache for markets data per tier with timestamps
	cache struct {
		sync.RWMutex
		tiers map[string]*TierDataWithTimestamp // tier name -> data with timestamp
	}

	// Extra IDs to fetch (from coins/list)
	extraIds struct {
		sync.RWMutex
		ids []string
	}

	// Track initial load completion per tier
	initialLoad struct {
		sync.RWMutex
		completedTiers map[string]bool
		allCompleted   bool
	}
}

// NewPeriodicUpdater creates a new periodic markets updater
func NewPeriodicUpdater(cfg *config.MarketsFetcherConfig, apiClient IAPIClient) *PeriodicUpdater {
	updater := &PeriodicUpdater{
		config:        cfg,
		apiClient:     apiClient,
		metricsWriter: metrics.NewMetricsWriter(metrics.ServiceMarkets),
	}

	updater.cache.tiers = make(map[string]*TierDataWithTimestamp)
	updater.initialLoad.completedTiers = make(map[string]bool)

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

// SetOnInitialLoadCompletedCallback sets a callback function that will be called when all tiers complete their initial load
func (u *PeriodicUpdater) SetOnInitialLoadCompletedCallback(onInitialLoadCompleted func(ctx context.Context)) {
	u.onInitialLoadCompleted = onInitialLoadCompleted
}

// SetExtraIds sets the list of extra token IDs to fetch
func (u *PeriodicUpdater) SetExtraIds(ids []string) {
	u.extraIds.Lock()
	defer u.extraIds.Unlock()
	u.extraIds.ids = make([]string, len(ids))
	copy(u.extraIds.ids, ids)
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

func (u *PeriodicUpdater) Start(ctx context.Context) error {
	if err := u.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return u.startAllTiers(ctx)
}

// startAllTiers starts a single scheduler that manages all tiers
func (u *PeriodicUpdater) startAllTiers(ctx context.Context) error {
	// Create single scheduler that runs every 2 seconds
	u.scheduler = scheduler.New(
		2*time.Second,
		func(ctx context.Context) {
			u.checkAndUpdateTiers(ctx)
		},
	)

	// Start the scheduler with context
	u.scheduler.Start(ctx, true)

	return nil
}

// Stop stops the periodic updater
func (u *PeriodicUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

// checkAndUpdateTiers checks all tiers and starts updates if needed
func (u *PeriodicUpdater) checkAndUpdateTiers(ctx context.Context) {
	now := time.Now()

	for _, tier := range u.config.Tiers {
		shouldUpdate := false
		var lastUpdate time.Time
		var isUpdating bool

		// Check tier cache status
		u.cache.RLock()
		tierData := u.cache.tiers[tier.Name]
		if tierData == nil {
			// Never updated before
			shouldUpdate = true
		} else {
			lastUpdate = tierData.Timestamp
			isUpdating = tierData.UpdateStartTime != nil

			// Check for stuck updates
			if isUpdating && tierData.UpdateStartTime != nil {
				updateDuration := now.Sub(*tierData.UpdateStartTime)
				// Consider stuck if running longer than max(10 minutes, 3x interval)
				maxUpdateDuration := 10 * time.Minute
				if tier.UpdateInterval*3 > maxUpdateDuration {
					maxUpdateDuration = tier.UpdateInterval * 3
				}

				if updateDuration > maxUpdateDuration {
					log.Printf("WARNING: Markets tier '%s' update stuck for %v (max: %v), resetting...",
						tier.Name, updateDuration, maxUpdateDuration)
					isUpdating = false
					// Reset the stuck state
					go u.setTierUpdateStartTime(tier.Name, nil)
				}
			}

			// Check if enough time has passed since last update
			if !isUpdating && now.Sub(lastUpdate) >= tier.UpdateInterval {
				shouldUpdate = true
			}
		}
		u.cache.RUnlock()

		if shouldUpdate {
			log.Printf("Starting update for markets tier '%s' (last update: %v, interval: %v, updating: %v)",
				tier.Name, lastUpdate.Format("15:04:05"), tier.UpdateInterval, isUpdating)

			// Start update in goroutine to avoid blocking other tiers
			go func(t config.MarketTier) {
				if err := u.fetchAndUpdateTier(ctx, t); err != nil {
					log.Printf("Error updating markets tier '%s' data: %v", t.Name, err)
				}
			}(tier)
		}
	}
}

// fetchAndUpdateTier fetches markets data for a specific tier and updates cache
func (u *PeriodicUpdater) fetchAndUpdateTier(ctx context.Context, tier config.MarketTier) error {
	defer u.metricsWriter.TrackDataFetchCycle()()

	// Mark update start time
	updateStartTime := time.Now()
	u.setTierUpdateStartTime(tier.Name, &updateStartTime)

	// Ensure we clear the update start time when done (with panic protection)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in fetchAndUpdateTier for markets tier '%s': %v", tier.Name, r)
		}
		u.setTierUpdateStartTime(tier.Name, nil)
	}()

	requestDelay := u.config.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = MARKETS_DEFAULT_REQUEST_DELAY
	}

	params := interfaces.MarketsParams{}
	params = ApplyParamsOverride(params, u.config)

	fetcher := NewPaginatedFetcher(u.apiClient, tier.PageFrom, tier.PageTo, requestDelayMs, params)

	// Create onPage callback to update cache with partial data (non-blocking)
	onPageCallback := func(pageData PageData) {
		if u.onUpdateTierPages != nil {
			go u.onUpdateTierPages(ctx, tier, []PageData{pageData})
		}
	}

	pagesData, err := fetcher.FetchPages(onPageCallback)
	if err != nil {
		log.Printf("PaginatedFetcher failed to fetch top markets data for tier '%s': %v", tier.Name, err)
		return err
	}

	// Flatten pages data for further processing
	var tokensData [][]byte
	for _, pageData := range pagesData {
		tokensData = append(tokensData, pageData.Data...)
	}

	// Final cache update - replace with complete data to ensure consistency
	localData := &TierDataWithTimestamp{
		Data:            ConvertMarketsResponseToCoinGeckoData(tokensData),
		Timestamp:       time.Now(),
		UpdateStartTime: nil, // Will be cleared by defer
	}

	u.cache.Lock()
	u.cache.tiers[tier.Name] = localData
	u.cache.Unlock()

	// Fetch missing coinslist IDs if enabled for this tier
	if tier.FetchCoinslistIds {
		_, err := u.fetchMissingExtraIds(ctx, tier)
		if err != nil {
			log.Printf("Failed to fetch missing extra IDs for tier '%s': %v", tier.Name, err)
		}
	}

	// Record metrics after successful update
	u.metricsWriter.RecordCacheSize(len(localData.Data))

	log.Printf("Updated tier '%s' cache with %d tokens (page: %d-%d)",
		tier.Name, len(localData.Data), tier.PageFrom, tier.PageTo)

	// Call final callback to notify tier update completion (even for empty data)
	if u.onUpdateTierPages != nil {
		u.onUpdateTierPages(ctx, tier, pagesData)
	}

	// Check if this is the first time this tier completed and all tiers are now complete
	u.checkAndTriggerInitialLoadCompleted(ctx, tier.Name)

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

	// Create onChunk callback to update partial missing extra IDs immediately (non-blocking)
	onChunkCallback := func(chunkData [][]byte) {
		if u.onUpdateMissingExtraIds != nil {
			go u.onUpdateMissingExtraIds(ctx, chunkData)
		}
	}

	tokensData, err := chunksFetcher.FetchMarkets(ctx, params, onChunkCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch missing extra IDs: %w", err)
	}

	// The fetched data will be cached by the service layer through the callback
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

			for _, coinData := range tierData.Data {
				if coinData.ID == id {
					found = true
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

// checkAndTriggerInitialLoadCompleted checks if all tiers have completed their initial load
func (u *PeriodicUpdater) checkAndTriggerInitialLoadCompleted(ctx context.Context, tierName string) {
	u.initialLoad.Lock()
	defer u.initialLoad.Unlock()

	// Mark this tier as completed
	if !u.initialLoad.completedTiers[tierName] {
		u.initialLoad.completedTiers[tierName] = true
		log.Printf("Tier '%s' completed initial load", tierName)
	}

	// Check if all tiers are completed and we haven't triggered the callback yet
	if !u.initialLoad.allCompleted && len(u.initialLoad.completedTiers) == len(u.config.Tiers) {
		allCompleted := true
		for _, tier := range u.config.Tiers {
			if !u.initialLoad.completedTiers[tier.Name] {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			u.initialLoad.allCompleted = true
			log.Printf("All %d tiers completed initial load", len(u.config.Tiers))

			if u.onInitialLoadCompleted != nil {
				go u.onInitialLoadCompleted(ctx)
			}
		}
	}
}

// Healthy checks if the periodic updater can fetch data
func (u *PeriodicUpdater) Healthy() bool {
	// Check if we already have some data in cache
	if u.GetCacheData() != nil && len(u.GetCacheData().Data) > 0 {
		return true
	}

	return u.apiClient != nil && u.apiClient.Healthy()
}

// setTierUpdateStartTime sets or clears the update start time for a tier
func (u *PeriodicUpdater) setTierUpdateStartTime(tierName string, startTime *time.Time) {
	u.cache.Lock()
	defer u.cache.Unlock()

	tierData := u.cache.tiers[tierName]
	if tierData == nil {
		// Create empty tier data if it doesn't exist
		tierData = &TierDataWithTimestamp{
			Data:            make([]CoinGeckoData, 0),
			Timestamp:       time.Time{}, // Zero time indicates never updated
			UpdateStartTime: startTime,
		}
		u.cache.tiers[tierName] = tierData
	} else {
		// Update existing tier data
		tierData.UpdateStartTime = startTime
	}
}
