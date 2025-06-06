package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

const (
	// Maximum items per page
	MAX_PER_PAGE = 250 // CoinGecko's API max per_page value
)

type CacheData struct {
	sync.RWMutex
	Data interface{}
}

// Service represents the CoinGecko service
type Service struct {
	config   *config.Config
	onUpdate func()
	cache    struct {
		sync.RWMutex
		data *APIResponse
	}
	scheduler        *scheduler.Scheduler
	apiClient        *CoinGeckoClient
	fetcher          *PaginatedFetcher
	topPricesUpdater *TopPricesUpdater
}

// NewService creates a new CoinGecko service
func NewService(cfg *config.Config, priceFetcher coingecko_common.PriceFetcher) *Service {
	// Create API client
	apiClient := NewCoinGeckoClient(cfg)

	// Create paginated fetcher with the API client
	requestDelayMs := int(cfg.CoingeckoLeaderboard.RequestDelay.Milliseconds())
	fetcher := NewPaginatedFetcher(apiClient, cfg.CoingeckoLeaderboard.Limit, MAX_PER_PAGE, requestDelayMs)

	// Create top prices updater
	topPricesUpdater := NewTopPricesUpdater(cfg, priceFetcher)

	service := &Service{
		config:           cfg,
		apiClient:        apiClient,
		fetcher:          fetcher,
		topPricesUpdater: topPricesUpdater,
	}

	return service
}

// SetOnUpdateCallback sets a callback function that will be called when data is updated
func (s *Service) SetOnUpdateCallback(onUpdate func()) {
	s.onUpdate = onUpdate
}

// GetTopPricesQuotes returns cached prices quotes for top tokens in specified currency
func (s *Service) GetTopPricesQuotes(currency string) map[string]Quote {
	return s.topPricesUpdater.GetTopPricesQuotes(currency)
}

// Start starts the CoinGecko service
func (s *Service) Start(ctx context.Context) error {
	// Use update interval directly as it's already a time.Duration
	updateInterval := s.config.CoingeckoLeaderboard.UpdateInterval

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

	// Start top prices updater
	if err := s.topPricesUpdater.Start(ctx); err != nil {
		log.Printf("Error starting top prices updater: %v", err)
		return err
	}

	return nil
}

func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
	if s.topPricesUpdater != nil {
		s.topPricesUpdater.Stop()
	}
}

// fetchAndUpdate fetches data from CoinGecko and signals update
func (s *Service) fetchAndUpdate(ctx context.Context) error {
	// Reset request cycle counters
	metrics.ResetCycleCounters("coingecko_common")

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

	// Update top prices updater with new token IDs
	if data != nil && data.Data != nil && s.topPricesUpdater != nil {
		tokenIDs := make([]string, len(data.Data))
		for i, coin := range data.Data {
			tokenIDs[i] = coin.ID
		}
		s.topPricesUpdater.SetTopTokenIDs(tokenIDs)
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
