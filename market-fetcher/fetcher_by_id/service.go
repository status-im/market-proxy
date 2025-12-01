package fetcher_by_id

import (
	"context"
	"fmt"
	"log"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/metrics"
)

// Service manages id-parametrized data fetching with caching
type Service struct {
	cfg                 *config.FetcherByIdConfig
	globalCfg           *config.Config
	client              *Client
	cache               cache.ICache
	metricsWriter       *metrics.MetricsWriter
	subscriptionManager *events.SubscriptionManager
	periodicUpdater     *PeriodicUpdater
}

func NewService(globalCfg *config.Config, fetcherCfg *config.FetcherByIdConfig, cacheService cache.ICache) *Service {
	metricsWriter := metrics.NewMetricsWriter(fetcherCfg.Name)
	client := NewClient(globalCfg, fetcherCfg, metricsWriter)

	service := &Service{
		cfg:                 fetcherCfg,
		globalCfg:           globalCfg,
		client:              client,
		cache:               cacheService,
		metricsWriter:       metricsWriter,
		subscriptionManager: events.NewSubscriptionManager(),
	}

	service.periodicUpdater = NewPeriodicUpdater(
		fetcherCfg,
		client,
		metricsWriter,
		service.onDataUpdated,
	)

	return service
}

func (s *Service) SetIdsProvider(provider IIdsProvider) {
	s.periodicUpdater.SetIdsProvider(provider)
}

// SetExtraIdsProvider sets the extra IDs provider (from coinslist service)
func (s *Service) SetExtraIdsProvider(provider IIdsProvider) {
	s.periodicUpdater.SetExtraIdsProvider(provider)
}

func (s *Service) onDataUpdated(ctx context.Context, data map[string][]byte) error {
	if err := s.cacheByID(data); err != nil {
		log.Printf("%s: Failed to cache data: %v", s.cfg.Name, err)
		return err
	}

	log.Printf("%s: Cache update complete - items: %d", s.cfg.Name, len(data))
	s.subscriptionManager.Emit(ctx)

	return nil
}

func (s *Service) cacheByID(data map[string][]byte) error {
	if len(data) == 0 {
		return nil
	}

	cacheData := make(map[string][]byte)
	for id, rawData := range data {
		cacheKey := s.cfg.BuildCacheKey(id)
		cacheData[cacheKey] = rawData
	}

	err := s.cache.Set(cacheData, s.cfg.GetTTL())
	if err != nil {
		return fmt.Errorf("failed to store in cache: %w", err)
	}

	return nil
}

func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}

	log.Printf("%s: Starting service (mode: %s)", s.cfg.Name, s.cfg.GetFetchMode())
	return s.periodicUpdater.Start(ctx)
}

func (s *Service) Stop() {
	s.periodicUpdater.Stop()
	log.Printf("%s: Service stopped", s.cfg.Name)
}

// GetByID returns cached data for a specific ID (for HTTP API)
func (s *Service) GetByID(id string) ([]byte, interfaces.CacheStatus, error) {
	cacheKey := s.cfg.BuildCacheKey(id)

	cachedData, missingKeys, err := s.cache.Get([]string{cacheKey})
	if err != nil {
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("failed to get from cache: %w", err)
	}

	if len(missingKeys) > 0 {
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("item not found: %s", id)
	}

	data, exists := cachedData[cacheKey]
	if !exists {
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("item not found: %s", id)
	}

	return data, interfaces.CacheStatusFull, nil
}

func (s *Service) GetMultiple(ids []string) (map[string][]byte, []string, interfaces.CacheStatus) {
	if len(ids) == 0 {
		return make(map[string][]byte), nil, interfaces.CacheStatusFull
	}

	cacheKeys := make([]string, len(ids))
	keyToID := make(map[string]string)
	for i, id := range ids {
		cacheKey := s.cfg.BuildCacheKey(id)
		cacheKeys[i] = cacheKey
		keyToID[cacheKey] = id
	}

	cachedData, missingKeys, err := s.cache.Get(cacheKeys)
	if err != nil {
		log.Printf("%s: Failed to get from cache: %v", s.cfg.Name, err)
		return nil, ids, interfaces.CacheStatusMiss
	}

	result := make(map[string][]byte)
	for cacheKey, data := range cachedData {
		id := keyToID[cacheKey]
		result[id] = data
	}

	var missing []string
	for _, cacheKey := range missingKeys {
		if id, exists := keyToID[cacheKey]; exists {
			missing = append(missing, id)
		}
	}

	var status interfaces.CacheStatus
	if len(missing) == 0 {
		status = interfaces.CacheStatusFull
	} else if len(result) > 0 {
		status = interfaces.CacheStatusPartial
	} else {
		status = interfaces.CacheStatusMiss
	}

	return result, missing, status
}

func (s *Service) Healthy() bool {
	return s.periodicUpdater.IsInitialized()
}

func (s *Service) SubscribeOnUpdate() events.ISubscription {
	return s.subscriptionManager.Subscribe()
}

func (s *Service) ForceUpdate(ctx context.Context) error {
	return s.periodicUpdater.ForceUpdate(ctx)
}

func (s *Service) GetName() string {
	return s.cfg.Name
}

func (s *Service) GetConfig() *config.FetcherByIdConfig {
	return s.cfg
}
