package coingecko

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/scheduler"
)

const (
	// Base URL for public API
	COINGECKO_PUBLIC_URL = "https://api.coingecko.com/api/v3/coins/markets"
	// Base URL for Pro API
	COINGECKO_PRO_URL = "https://pro-api.coingecko.com/api/v3/coins/markets"
	// Maximum items per page
	MAX_PER_PAGE = 250 // CoinGecko's API max per_page value
)

type CacheData struct {
	sync.RWMutex
	Data interface{}
}

// Service represents the CoinGecko service
type Service struct {
	config    *config.Config
	apiTokens *config.APITokens
	onUpdate  func()
	cache     struct {
		sync.RWMutex
		data *APIResponse
	}
	scheduler *scheduler.Scheduler
	apiClient *CoinGeckoClient
	fetcher   *PaginatedFetcher
}

// NewService creates a new CoinGecko service
func NewService(cfg *config.Config, apiTokens *config.APITokens, onUpdate func()) *Service {
	// Create API client
	apiClient := NewCoinGeckoClient(cfg, apiTokens)

	// Create paginated fetcher with the API client
	fetcher := NewPaginatedFetcher(apiClient, cfg.CoinGeckoFetcher.Limit, MAX_PER_PAGE)

	return &Service{
		config:    cfg,
		apiTokens: apiTokens,
		onUpdate:  onUpdate,
		apiClient: apiClient,
		fetcher:   fetcher,
	}
}

// Start starts the CoinGecko service
func (s *Service) Start(ctx context.Context) error {
	// Create scheduler for periodic updates
	s.scheduler = scheduler.New(
		time.Duration(s.config.CoinGeckoFetcher.UpdateInterval)*time.Second,
		func(ctx context.Context) {
			if err := s.fetchAndUpdate(ctx); err != nil {
				log.Printf("Error updating data: %v", err)
			}
		},
	)

	// Start the initial fetch
	if err := s.fetchAndUpdate(ctx); err != nil {
		return fmt.Errorf("initial fetch failed: %v", err)
	}

	// Start the scheduler with context
	s.scheduler.Start(ctx)
	return nil
}

func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

// fetchAndUpdate fetches data from CoinGecko and signals update
func (s *Service) fetchAndUpdate(ctx context.Context) error {
	data, err := s.fetcher.FetchData()
	if err != nil {
		return err
	}

	s.cache.Lock()
	s.cache.data = data
	s.cache.Unlock()

	// Signal update through callback
	if s.onUpdate != nil {
		s.onUpdate()
	}

	return nil
}

func (s *Service) GetCacheData() *APIResponse {
	s.cache.RLock()
	defer s.cache.RUnlock()
	return s.cache.data
}
