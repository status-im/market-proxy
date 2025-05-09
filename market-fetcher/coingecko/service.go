package coingecko

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

const (
	// Base URL for public API
	COINGECKO_PUBLIC_URL = "https://api.coingecko.com"
	// Base URL for Pro API
	COINGECKO_PRO_URL = "https://pro-api.coingecko.com"
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
	fetcher := NewPaginatedFetcher(apiClient, cfg.CoinGeckoFetcher.Limit, MAX_PER_PAGE, cfg.CoinGeckoFetcher.RequestDelayMs)

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
	// Convert update interval from milliseconds to time.Duration
	updateInterval := time.Duration(s.config.CoinGeckoFetcher.UpdateIntervalMs) * time.Millisecond

	// Create scheduler for periodic updates
	s.scheduler = scheduler.New(
		updateInterval,
		func(ctx context.Context) {
			if err := s.fetchAndUpdate(ctx); err != nil {
				log.Printf("Error updating data: %v", err)
			}
		},
	)

	// Start the scheduler with context
	s.scheduler.Start(ctx, true)
	return nil
}

func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

// fetchAndUpdate fetches data from CoinGecko and signals update
func (s *Service) fetchAndUpdate(ctx context.Context) error {
	// Reset request cycle counters
	metrics.ResetCycleCounters("coingecko")

	// Record start time for metrics
	startTime := time.Now()

	// Perform the fetch operation
	data, err := s.fetcher.FetchData()

	// Record metrics regardless of success or failure
	metrics.RecordFetchMarketDataCycle("markets-leaderboard", startTime)

	if err != nil {
		return err
	}

	s.cache.Lock()
	s.cache.data = data
	s.cache.Unlock()

	// Record cache size metric
	if data != nil && data.Data != nil {
		metrics.RecordTokensCacheSize("markets-leaderboard", len(data.Data))
	}

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

// Healthy checks if the service can fetch at least one page of data
func (s *Service) Healthy() bool {
	// Check if we already have some data in cache
	if s.GetCacheData() != nil && len(s.GetCacheData().Data) > 0 {
		return true
	}

	// If not, try to fetch at least one page
	return s.apiClient.Healthy()
}
