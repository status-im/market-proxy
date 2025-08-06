package coingecko_leaderboard

import (
	"context"
	"log"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
)

// Service represents the CoinGecko service
type Service struct {
	config            *config.Config
	onUpdate          func()
	topMarketsUpdater *TopMarketsUpdater
	topPricesUpdater  *TopPricesUpdater
}

// NewService creates a new CoinGecko service
func NewService(cfg *config.Config, priceFetcher interfaces.CoingeckoPricesService, marketsFetcher interfaces.CoingeckoMarketsService) *Service {
	// Create top markets updater
	topMarketsUpdater := NewTopMarketsUpdater(cfg, marketsFetcher)

	// Create top prices updater
	topPricesUpdater := NewTopPricesUpdater(cfg, priceFetcher)

	service := &Service{
		config:            cfg,
		topMarketsUpdater: topMarketsUpdater,
		topPricesUpdater:  topPricesUpdater,
	}

	return service
}

// SetOnUpdateCallback sets a callback function that will be called when data is updated
func (s *Service) SetOnUpdateCallback(onUpdate func()) {
	s.onUpdate = onUpdate

	// Set callback for top markets updater
	s.topMarketsUpdater.SetOnUpdateCallback(func() {
		if s.onUpdate != nil {
			s.onUpdate()
		}
	})
}

// GetTopPricesQuotes returns cached prices quotes for top tokens with default currency fallback
func (s *Service) GetTopPricesQuotes(currency string) map[string]Quote {
	// Set default currency if not provided
	if currency == "" {
		currency = "usd"
	}

	return s.topPricesUpdater.GetTopPricesQuotes(currency)
}

// Start starts the CoinGecko service
func (s *Service) Start(ctx context.Context) error {
	// Start top markets updater
	if err := s.topMarketsUpdater.Start(ctx); err != nil {
		log.Printf("Error starting top markets updater: %v", err)
		return err
	}

	// Start top prices updater
	if err := s.topPricesUpdater.Start(ctx); err != nil {
		log.Printf("Error starting top prices updater: %v", err)
		return err
	}

	return nil
}

func (s *Service) Stop() {
	if s.topMarketsUpdater != nil {
		s.topMarketsUpdater.Stop()
	}
	if s.topPricesUpdater != nil {
		s.topPricesUpdater.Stop()
	}
}

func (s *Service) GetCacheData() *APIResponse {
	return s.topMarketsUpdater.GetCacheData()
}

// Healthy checks if the service can fetch at least one page of data
func (s *Service) Healthy() bool {
	return s.topMarketsUpdater.Healthy()
}
