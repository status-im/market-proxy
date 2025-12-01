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

	idsProvider   IIdsProvider
	idsProviderMu sync.RWMutex

	extraIdsProvider   IIdsProvider
	extraIdsProviderMu sync.RWMutex

	tierStates   map[string]*TierState
	tierStatesMu sync.RWMutex
}

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
		ChunksDefaultRequestDelay,
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

	for _, tier := range cfg.Tiers {
		u.tierStates[tier.Name] = &TierState{}
	}

	return u
}

func (u *PeriodicUpdater) SetIdsProvider(provider IIdsProvider) {
	u.idsProviderMu.Lock()
	defer u.idsProviderMu.Unlock()
	u.idsProvider = provider
}

func (u *PeriodicUpdater) SetExtraIdsProvider(provider IIdsProvider) {
	u.extraIdsProviderMu.Lock()
	defer u.extraIdsProviderMu.Unlock()
	u.extraIdsProvider = provider
}

func (u *PeriodicUpdater) Start(ctx context.Context) error {
	if err := u.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	log.Printf("%s: Starting periodic updater with %d tiers", u.cfg.Name, len(u.cfg.Tiers))

	u.scheduler = scheduler.New(2*time.Second, func(ctx context.Context) {
		u.checkAndUpdateTiers(ctx, false)
	})

	u.scheduler.Start(ctx, true)

	return nil
}

func (u *PeriodicUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

func (u *PeriodicUpdater) IsInitialized() bool {
	return u.initialized.Load()
}

func (u *PeriodicUpdater) ForceUpdate(ctx context.Context) error {
	u.checkAndUpdateTiers(ctx, true)
	return nil
}

func (u *PeriodicUpdater) checkAndUpdateTiers(ctx context.Context, force bool) {
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

		u.tierStatesMu.RLock()
		state := u.tierStates[tier.Name]
		if state == nil {
			state = &TierState{}
			u.tierStates[tier.Name] = state
		}
		lastUpdate := state.LastUpdate
		isUpdating = state.IsUpdating

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

func (u *PeriodicUpdater) fetchAndUpdateTier(ctx context.Context, tier config.GenericTier, allIds []string) error {
	defer u.metricsWriter.TrackDataFetchCycle()()

	u.setTierUpdating(tier.Name, true)
	defer u.setTierUpdating(tier.Name, false)

	fromIndex := tier.IdFrom - 1
	toIndex := tier.IdTo - 1

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

	data, err := u.chunksFetcher.FetchData(ctx, tierIds, func(chunkData map[string][]byte) {
		if u.onUpdated != nil {
			_ = u.onUpdated(ctx, chunkData)
		}
	})
	if err != nil {
		return err
	}

	if tier.FetchCoinslistIds {
		extraData, extraErr := u.fetchExtraIds(ctx, data)
		if extraErr != nil {
			log.Printf("%s: Failed to fetch extra IDs for tier '%s': %v",
				u.cfg.Name, tier.Name, extraErr)
		} else if len(extraData) > 0 {
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

	u.tierStatesMu.Lock()
	if state, exists := u.tierStates[tier.Name]; exists {
		state.LastUpdate = time.Now()
	}
	u.tierStatesMu.Unlock()

	if u.onUpdated != nil {
		if err := u.onUpdated(ctx, data); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	u.initialized.Store(true)

	log.Printf("%s: Updated tier '%s' with %d items", u.cfg.Name, tier.Name, len(data))
	return nil
}

func (u *PeriodicUpdater) fetchExtraIds(ctx context.Context, existingData map[string][]byte) (map[string][]byte, error) {
	u.extraIdsProviderMu.RLock()
	provider := u.extraIdsProvider
	u.extraIdsProviderMu.RUnlock()

	if provider == nil {
		return nil, nil
	}

	extraIds, err := provider.GetIds(0)
	if err != nil {
		return nil, err
	}

	if len(extraIds) == 0 {
		return nil, nil
	}

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

	return u.chunksFetcher.FetchData(ctx, missingIds, func(chunkData map[string][]byte) {
		if u.onUpdated != nil {
			_ = u.onUpdated(ctx, chunkData)
		}
	})
}

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
