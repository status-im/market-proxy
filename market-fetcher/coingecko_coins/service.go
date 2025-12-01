package coingecko_coins

import (
	"context"
	"log"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/fetcher_by_id"
	"github.com/status-im/market-proxy/interfaces"
)

// Service manages coin data with caching using the generic framework
type Service struct {
	cfg            *config.Config
	genericService *fetcher_by_id.Service
	marketsService interfaces.IMarketsService
}

// marketsIdsProvider adapts IMarketsService to IIdsProvider
type marketsIdsProvider struct {
	marketsService interfaces.IMarketsService
}

// GetIds implements fetcher_by_id.IIdsProvider
func (p *marketsIdsProvider) GetIds(limit int) ([]string, error) {
	return p.marketsService.TopMarketIds(limit)
}

// NewService creates a new coins service using the generic framework
func NewService(cfg *config.Config, marketsService interfaces.IMarketsService, cacheService cache.ICache) *Service {
	genericService := fetcher_by_id.NewService(cfg, &cfg.CoingeckoCoins, cacheService)
	genericService.SetIdsProvider(&marketsIdsProvider{marketsService: marketsService})

	return &Service{
		cfg:            cfg,
		genericService: genericService,
		marketsService: marketsService,
	}
}

// Start starts the service
func (s *Service) Start(ctx context.Context) error {
	log.Printf("Starting coins service")
	return s.genericService.Start(ctx)
}

// Stop stops the service
func (s *Service) Stop() {
	s.genericService.Stop()
}

// GetCoin returns cached coin data for a specific coin ID
func (s *Service) GetCoin(coinID string) ([]byte, interfaces.CacheStatus, error) {
	return s.genericService.GetByID(coinID)
}

// GetMultipleCoins returns cached coin data for multiple coin IDs
func (s *Service) GetMultipleCoins(coinIDs []string) (map[string][]byte, []string, interfaces.CacheStatus) {
	return s.genericService.GetMultiple(coinIDs)
}

// Healthy checks if service is initialized and has data
func (s *Service) Healthy() bool {
	return s.genericService.Healthy()
}

// SubscribeOnCoinsUpdate subscribes to coins update notifications
func (s *Service) SubscribeOnCoinsUpdate() events.ISubscription {
	return s.genericService.SubscribeOnUpdate()
}
