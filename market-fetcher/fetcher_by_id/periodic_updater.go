package fetcher_by_id

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// TierState holds the state for a tier
type TierState struct {
	LastUpdate      time.Time
	UpdateStartTime *time.Time
	IsUpdating      bool
}

// PeriodicUpdater handles periodic fetching and updating of data
type PeriodicUpdater struct {
	cfg           *config.FetcherByIdConfig
	client        *Client
	chunksFetcher *ChunksFetcher
	metricsWriter *metrics.MetricsWriter
	onUpdated     UpdateCallback
	scheduler     *scheduler.Scheduler
	initialized   atomic.Bool

	// IDs provider for fetching (from markets service)
	idsProvider   IIdsProvider
	idsProviderMu sync.RWMutex

	// Extra IDs provider (from coinslist service)
	extraIdsProvider   IIdsProvider
	extraIdsProviderMu sync.RWMutex

	// Tier states
	tierStates   map[string]*TierState
	tierStatesMu sync.RWMutex
}

// NewPeriodicUpdater creates a new periodic updater
func NewPeriodicUpdater(
	cfg *config.FetcherByIdConfig,
	client *Client,
	metricsWriter *metrics.MetricsWriter,
	onUpdated UpdateCallback,
) *PeriodicUpdater {
	chunksFetcher := NewChunksFetcher(
		client,
		cfg.Name,
		cfg.GetChunkSize(),
		ChunksDefaultRequestDelay, // Use default delay
		cfg.IsBatchMode(),
	)

	u := &PeriodicUpdater{
		cfg:           cfg,
		client:        client,
		chunksFetcher: chunksFetcher,
		metricsWriter: metricsWriter,
		onUpdated:     onUpdated,
		tierStates:    make(map[string]*TierState),
	}

	// Initialize tier states
	for _, tier := range cfg.Tiers {
		u.tierStates[tier.Name] = &TierState{}
	}

	return u
}

// SetIdsProvider sets the main IDs provider (from markets service)
func (u *PeriodicUpdater) SetIdsProvider(provider IIdsProvider) {
	u.idsProviderMu.Lock()
	defer u.idsProviderMu.Unlock()
	u.idsProvider = provider
}

// SetExtraIdsProvider sets the extra IDs provider (from coinslist service)
func (u *PeriodicUpdater) SetExtraIdsProvider(provider IIdsProvider) {
	u.extraIdsProviderMu.Lock()
	defer u.extraIdsProviderMu.Unlock()
	u.extraIdsProvider = provider
}

// Start begins periodic updates
func (u *PeriodicUpdater) Start(ctx context.Context) error {
	if err := u.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if u.cfg.HasTiers() {
		return u.startWithTiers(ctx)
	}

	return u.startSimple(ctx)
}

// startSimple starts the updater in simple mode (no tiers)
func (u *PeriodicUpdater) startSimple(ctx context.Context) error {
	updateInterval := u.cfg.UpdateInterval
	if updateInterval <= 0 {
		log.Printf("%s: Periodic updates disabled (interval: %v)", u.cfg.Name, updateInterval)
		return nil
	}

	u.scheduler = scheduler.New(updateInterval, func(ctx context.Context) {
		if err := u.fetchAndUpdate(ctx); err != nil {
			log.Printf("%s: Error updating data: %v", u.cfg.Name, err)
		} else {
			u.initialized.Store(true)
		}
	})

	u.scheduler.Start(ctx, true)

	log.Printf("%s: Started periodic updater with interval %v", u.cfg.Name, updateInterval)
	return nil
}

// startWithTiers starts the updater in tier-based mode
func (u *PeriodicUpdater) startWithTiers(ctx context.Context) error {
	log.Printf("%s: Starting periodic updater with %d tiers", u.cfg.Name, len(u.cfg.Tiers))

	// Single scheduler that checks all tiers every 2 seconds
	u.scheduler = scheduler.New(2*time.Second, func(ctx context.Context) {
		u.checkAndUpdateTiers(ctx, false)
	})

	u.scheduler.Start(ctx, true)

	return nil
}

// Stop stops periodic updates
func (u *PeriodicUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

// IsInitialized returns true if updater has successfully fetched data at least once
func (u *PeriodicUpdater) IsInitialized() bool {
	return u.initialized.Load()
}

// ForceUpdate triggers an immediate update
func (u *PeriodicUpdater) ForceUpdate(ctx context.Context) error {
	if u.cfg.HasTiers() {
		u.checkAndUpdateTiers(ctx, true)
		return nil
	}
	return u.fetchAndUpdate(ctx)
}

// checkAndUpdateTiers checks all tiers and updates those that need updating
func (u *PeriodicUpdater) checkAndUpdateTiers(ctx context.Context, force bool) {
	// Get all IDs
	allIds, err := u.getAllIds()
	if err != nil {
		log.Printf("%s: Failed to get IDs: %v", u.cfg.Name, err)
		return
	}

	if len(allIds) == 0 && !force {
		return
	}

	now := time.Now()

	for _, tier := range u.cfg.Tiers {
		shouldUpdate := false
		var isUpdating bool

		// Check tier state
		u.tierStatesMu.RLock()
		state := u.tierStates[tier.Name]
		if state == nil {
			state = &TierState{}
			u.tierStates[tier.Name] = state
		}
		lastUpdate := state.LastUpdate
		isUpdating = state.IsUpdating

		// Check for stuck updates
		if isUpdating && state.UpdateStartTime != nil {
			updateDuration := now.Sub(*state.UpdateStartTime)
			maxUpdateDuration := 10 * time.Minute
			if tier.UpdateInterval*3 > maxUpdateDuration {
				maxUpdateDuration = tier.UpdateInterval * 3
			}

			if updateDuration > maxUpdateDuration {
				log.Printf("%s: WARNING: Tier '%s' update stuck for %v, resetting...",
					u.cfg.Name, tier.Name, updateDuration)
				isUpdating = false
				go u.setTierUpdating(tier.Name, false)
			}
		}

		if force {
			shouldUpdate = !isUpdating
		} else if !isUpdating && now.Sub(lastUpdate) >= tier.UpdateInterval {
			shouldUpdate = true
		}
		u.tierStatesMu.RUnlock()

		if shouldUpdate {
			go func(t config.GenericTier) {
				if err := u.fetchAndUpdateTier(ctx, t, allIds); err != nil {
					log.Printf("%s: Error updating tier '%s': %v", u.cfg.Name, t.Name, err)
				}
			}(tier)
		}
	}
}

// fetchAndUpdateTier fetches data for a specific tier
func (u *PeriodicUpdater) fetchAndUpdateTier(ctx context.Context, tier config.GenericTier, allIds []string) error {
	defer u.metricsWriter.TrackDataFetchCycle()()

	// Mark tier as updating
	u.setTierUpdating(tier.Name, true)
	defer u.setTierUpdating(tier.Name, false)

	// Calculate ID range for this tier
	fromIndex := tier.IdFrom - 1 // Convert to 0-based
	toIndex := tier.IdTo - 1

	// Validate indices
	if fromIndex < 0 {
		log.Printf("%s: Tier '%s' has invalid id_from (%d), must be >= 1",
			u.cfg.Name, tier.Name, tier.IdFrom)
		return nil
	}

	if fromIndex >= len(allIds) {
		log.Printf("%s: Tier '%s' id_from (%d) exceeds available IDs (%d)",
			u.cfg.Name, tier.Name, tier.IdFrom, len(allIds))
		return nil
	}

	if toIndex < fromIndex {
		log.Printf("%s: Tier '%s' has invalid id_to (%d), must be >= id_from (%d)",
			u.cfg.Name, tier.Name, tier.IdTo, tier.IdFrom)
		return nil
	}

	if toIndex >= len(allIds) {
		toIndex = len(allIds) - 1
	}

	tierIds := allIds[fromIndex : toIndex+1]

	log.Printf("%s: Fetching data for tier '%s' with %d IDs (range: %d-%d)",
		u.cfg.Name, tier.Name, len(tierIds), tier.IdFrom, tier.IdTo)

	// Fetch data using chunks fetcher with onChunk callback to cache immediately
	data, err := u.chunksFetcher.FetchData(ctx, tierIds, func(chunkData map[string][]byte) {
		// Cache each chunk immediately as it's fetched
		if u.onUpdated != nil {
			_ = u.onUpdated(ctx, chunkData)
		}
	})
	if err != nil {
		return err
	}

	// Fetch extra IDs from coinslist if enabled
	if tier.FetchCoinslistIds {
		extraData, extraErr := u.fetchExtraIds(ctx, data)
		if extraErr != nil {
			log.Printf("%s: Failed to fetch extra IDs for tier '%s': %v",
				u.cfg.Name, tier.Name, extraErr)
		} else if len(extraData) > 0 {
			// Merge extra data
			for id, d := range extraData {
				data[id] = d
			}
			log.Printf("%s: Fetched %d extra IDs for tier '%s'",
				u.cfg.Name, len(extraData), tier.Name)
		}
	}

	if len(data) == 0 {
		return fmt.Errorf("failed to fetch any data for tier '%s'", tier.Name)
	}

	u.metricsWriter.RecordCacheSize(len(data))

	// Update tier timestamp
	u.tierStatesMu.Lock()
	if state, exists := u.tierStates[tier.Name]; exists {
		state.LastUpdate = time.Now()
	}
	u.tierStatesMu.Unlock()

	// Call the callback with updated data
	if u.onUpdated != nil {
		if err := u.onUpdated(ctx, data); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	u.initialized.Store(true)

	log.Printf("%s: Updated tier '%s' with %d items", u.cfg.Name, tier.Name, len(data))
	return nil
}

// fetchExtraIds fetches IDs from coinslist that are not in the current tier's data
func (u *PeriodicUpdater) fetchExtraIds(ctx context.Context, existingData map[string][]byte) (map[string][]byte, error) {
	u.extraIdsProviderMu.RLock()
	provider := u.extraIdsProvider
	u.extraIdsProviderMu.RUnlock()

	if provider == nil {
		return nil, nil
	}

	// Get extra IDs from provider (fresh data on each call)
	extraIds, err := provider.GetIds(0) // 0 = all available
	if err != nil {
		return nil, err
	}

	if len(extraIds) == 0 {
		return nil, nil
	}

	// Filter to IDs that are not in the current tier's data
	var missingIds []string
	for _, id := range extraIds {
		if _, exists := existingData[id]; !exists {
			missingIds = append(missingIds, id)
		}
	}

	if len(missingIds) == 0 {
		log.Printf("%s: All extra IDs already fetched in tier", u.cfg.Name)
		return nil, nil
	}

	log.Printf("%s: Fetching %d extra IDs", u.cfg.Name, len(missingIds))

	// Fetch with onChunk callback to cache immediately
	return u.chunksFetcher.FetchData(ctx, missingIds, func(chunkData map[string][]byte) {
		if u.onUpdated != nil {
			_ = u.onUpdated(ctx, chunkData)
		}
	})
}

// setTierUpdating sets the updating state for a tier
func (u *PeriodicUpdater) setTierUpdating(tierName string, updating bool) {
	u.tierStatesMu.Lock()
	defer u.tierStatesMu.Unlock()

	state := u.tierStates[tierName]
	if state == nil {
		state = &TierState{}
		u.tierStates[tierName] = state
	}

	state.IsUpdating = updating
	if updating {
		now := time.Now()
		state.UpdateStartTime = &now
	} else {
		state.UpdateStartTime = nil
	}
}

// getAllIds retrieves IDs from the provider
func (u *PeriodicUpdater) getAllIds() ([]string, error) {
	u.idsProviderMu.RLock()
	provider := u.idsProvider
	u.idsProviderMu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("IDs provider not set")
	}

	limit := u.cfg.GetMaxIdLimit()
	return provider.GetIds(limit)
}

// fetchAndUpdate fetches data and calls the callback (simple mode)
func (u *PeriodicUpdater) fetchAndUpdate(ctx context.Context) error {
	u.metricsWriter.ResetCycleMetrics()
	defer u.metricsWriter.TrackDataFetchCycle()()

	// Get IDs to fetch
	ids, err := u.getIds()
	if err != nil {
		return fmt.Errorf("failed to get IDs: %w", err)
	}

	if len(ids) == 0 {
		log.Printf("%s: No IDs to fetch, skipping update", u.cfg.Name)
		return nil
	}

	log.Printf("%s: Fetching data for %d IDs (mode: %s)", u.cfg.Name, len(ids), u.cfg.GetFetchMode())

	// Fetch data using chunks fetcher with onChunk callback to cache immediately
	data, err := u.chunksFetcher.FetchData(ctx, ids, func(chunkData map[string][]byte) {
		// Cache each chunk immediately as it's fetched
		if u.onUpdated != nil {
			_ = u.onUpdated(ctx, chunkData)
		}
	})
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return fmt.Errorf("failed to fetch any data")
	}

	u.metricsWriter.RecordCacheSize(len(data))

	// Call the callback with updated data
	if u.onUpdated != nil {
		if err := u.onUpdated(ctx, data); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	log.Printf("%s: Updated cache with %d items", u.cfg.Name, len(data))
	return nil
}

// getIds retrieves IDs from the provider (simple mode)
func (u *PeriodicUpdater) getIds() ([]string, error) {
	u.idsProviderMu.RLock()
	provider := u.idsProvider
	u.idsProviderMu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("IDs provider not set")
	}

	limit := u.cfg.TopIdsLimit
	if limit <= 0 {
		limit = 1000 // default limit
	}

	return provider.GetIds(limit)
}
