package coingecko_markets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/status-im/market-proxy/cache"
	cfg "github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/metrics"
)

const (
	MARKETS_DEFAULT_CHUNK_SIZE    = 250  // CoinGecko's API max per_page value
	MARKETS_DEFAULT_REQUEST_DELAY = 1000 // 1 second in milliseconds
	ID_FIELD                      = "id"
)

// Service provides markets data fetching functionality with caching
type Service struct {
	cache               cache.Cache
	config              *cfg.Config
	metricsWriter       *metrics.MetricsWriter
	apiClient           APIClient
	subscriptionManager *events.SubscriptionManager
	periodicUpdater     *PeriodicUpdater
	tokensService       interfaces.CoingeckoTokensService
	tokenUpdateCh       chan struct{}
	cancelFunc          context.CancelFunc
}

func NewService(cache cache.Cache, config *cfg.Config, tokensService interfaces.CoingeckoTokensService) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarkets)
	apiClient := NewCoinGeckoClient(config)

	service := &Service{
		cache:               cache,
		config:              config,
		metricsWriter:       metricsWriter,
		apiClient:           apiClient,
		subscriptionManager: events.NewSubscriptionManager(),
		tokensService:       tokensService,
	}

	// Create periodic updater
	service.periodicUpdater = NewPeriodicUpdater(&config.CoingeckoMarkets, apiClient)

	// Set onUpdateTierPages callback to cache tokens and emit events to subscription manager
	service.periodicUpdater.SetOnUpdateTierPagesCallback(func(ctx context.Context, tier cfg.MarketTier, pagesData []PageData) {
		// Cache by individual ids
		for _, pageData := range pagesData {
			_, err := service.cacheTokensByID(pageData.Data)
			if err != nil {
				log.Printf("Failed to cache markets data by id: %v", err)
			}
		}

		// Cache pages
		_, err := service.cacheTokensPage(tier, pagesData)
		if err != nil {
			log.Printf("Failed to cache page data: %v", err)
		}
		service.subscriptionManager.Emit(ctx)
	})

	// Set onUpdateMissingExtraIds callback to cache missing tokens by ID
	service.periodicUpdater.SetOnUpdateMissingExtraIdsCallback(func(ctx context.Context, tokensData [][]byte) {
		// Cache missing tokens by their IDs
		_, err := service.cacheTokensByID(tokensData)
		if err != nil {
			log.Printf("Failed to cache missing extra IDs: %v", err)
		} else {
			log.Printf("Successfully cached %d missing extra IDs", len(tokensData))
		}
		service.subscriptionManager.Emit(ctx)
	})

	return service
}

// onTokenListChanged is called when token list is updated
func (s *Service) onTokenListChanged() {
	if s.tokensService == nil {
		return
	}

	tokenIDs := s.tokensService.GetTokenIds()

	log.Printf("Token list changed, updating periodic updater with %d extra IDs", len(tokenIDs))

	if s.periodicUpdater != nil {
		s.periodicUpdater.SetExtraIds(tokenIDs)
	}
}

// handleTokenUpdates handles token update notifications
func (s *Service) handleTokenUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.tokenUpdateCh:
			s.onTokenListChanged()
		}
	}
}

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}

	// Subscribe to token list updates
	if s.tokensService != nil {
		s.tokenUpdateCh = s.tokensService.SubscribeOnTokensUpdate()

		// Create cancelable context for the goroutine
		goroutineCtx, cancel := context.WithCancel(ctx)
		s.cancelFunc = cancel
		go s.handleTokenUpdates(goroutineCtx)

		// Initial call to set extra IDs
		s.onTokenListChanged()
	}

	// Start periodic updater
	if s.periodicUpdater != nil {
		if err := s.periodicUpdater.Start(ctx); err != nil {
			return fmt.Errorf("failed to start periodic updater: %w", err)
		}
	}

	return nil
}

// Stop implements core.Interface
func (s *Service) Stop() {
	// Cancel the goroutine first
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}

	// Unsubscribe from token updates
	if s.tokensService != nil && s.tokenUpdateCh != nil {
		s.tokensService.Unsubscribe(s.tokenUpdateCh)
		s.tokenUpdateCh = nil
	}

	// Stop periodic updater
	if s.periodicUpdater != nil {
		s.periodicUpdater.Stop()
	}
	// Cache will handle its own cleanup
}

// cacheTokensByID parses tokens data and caches each token by its CoinGecko ID
func (s *Service) cacheTokensByID(tokensData [][]byte) ([]interface{}, error) {
	// Parse tokens data
	marketData, cacheData, err := parseTokensData(tokensData)
	if err != nil {
		return nil, err
	}

	// Cache tokens directly
	if len(cacheData) > 0 {
		err := s.cache.Set(cacheData, s.config.CoingeckoMarkets.GetTTL())
		if err != nil {
			log.Printf("Failed to cache tokens data: %v", err)
			return nil, fmt.Errorf("failed to cache tokens data: %w", err)
		}
		log.Printf("Successfully cached %d tokens by their coingecko id", len(cacheData))
	}

	return marketData, nil
}

// cacheTokensPage caches page data for page-based requests
func (s *Service) cacheTokensPage(tier cfg.MarketTier, pagesData []PageData) (map[int]interface{}, error) {
	// Parse pages data
	pageMapping, cacheData, err := parsePagesData(pagesData)
	if err != nil {
		return nil, err
	}

	// Cache pages data
	if len(cacheData) > 0 {
		err := s.cache.Set(cacheData, s.config.CoingeckoMarkets.GetTTL())
		if err != nil {
			log.Printf("Failed to cache page data: %v", err)
			return nil, fmt.Errorf("failed to cache page data: %w", err)
		}
		log.Printf("Successfully cached %d pages for tier '%s'", len(cacheData), tier.Name)
	}

	return pageMapping, nil
}

// Markets fetches markets data using cache with specified parameters
// Returns full CoinGecko markets response in APIResponse format
func (s *Service) Markets(params interfaces.MarketsParams) (interfaces.MarketsResponse, interfaces.CacheStatus, error) {
	// Check if specific IDs are requested
	if len(params.IDs) > 0 {
		return s.MarketsByIds(params)
	}

	// TODO: Implement general markets fetching without specific IDs
	log.Printf("Markets called without specific IDs - returning empty array (TODO: implement general fetching)")
	return interfaces.MarketsResponse([]interface{}{}), interfaces.CacheStatusMiss, nil
}

// MarketsByIds fetches markets data for specific token IDs using cache
func (s *Service) MarketsByIds(params interfaces.MarketsParams) (response interfaces.MarketsResponse, cacheStatus interfaces.CacheStatus, err error) {
	log.Printf("Loading markets data for %d specific IDs with currency=%s", len(params.IDs), params.Currency)
	params = s.getParamsOverride(params)
	cacheKeys := createCacheKeys(params)

	cachedData, missingKeys, err := s.cache.Get(cacheKeys)
	if err != nil {
		log.Printf("Failed to check cache: %v", err)
	}

	// If all tokens are in cache, return cached data preserving order from cacheKeys
	if len(missingKeys) == 0 && len(cachedData) == len(cacheKeys) {
		marketData := make([]interface{}, 0, len(cacheKeys))
		for _, cacheKey := range cacheKeys {
			if tokenBytes, exists := cachedData[cacheKey]; exists {
				var tokenData interface{}
				if err := json.Unmarshal(tokenBytes, &tokenData); err == nil {
					marketData = append(marketData, tokenData)
				}
			}
		}
		log.Printf("Returning cached data for all %d tokens", len(marketData))
		return interfaces.MarketsResponse(marketData), interfaces.CacheStatusFull, nil
	}

	if len(cachedData) > 0 {
		cacheStatus = interfaces.CacheStatusPartial
	} else {
		cacheStatus = interfaces.CacheStatusMiss
	}

	// Some tokens are missing from cache - fetch from API
	log.Printf("Cache miss for some tokens, fetching from API")
	tokensData, err := s.apiClient.FetchPage(params)
	if err != nil {
		log.Printf("apiClient.FetchPage failed to fetch markets data: %v", err)
		return nil, cacheStatus, fmt.Errorf("failed to fetch markets data: %w", err)
	}

	// Cache tokens by their IDs
	marketData, err := s.cacheTokensByID(tokensData)
	if err != nil {
		return nil, cacheStatus, err
	}

	log.Printf("Loaded and cached markets data with %d coins", len(marketData))
	return interfaces.MarketsResponse(marketData), cacheStatus, nil
}

// Healthy checks if the service is operational
func (s *Service) Healthy() bool {
	// Check if we can fetch at least one page
	if s.apiClient != nil {
		return s.apiClient.Healthy()
	}
	return false
}

// TopMarkets fetches top markets data for specified number of tokens,
// caches individual tokens by their coingecko id and returns the response
func (s *Service) TopMarkets(limit int, currency string) (interfaces.MarketsResponse, error) {
	log.Printf("Loading top %d markets data from CoinGecko API with currency=%s", limit, currency)

	// Set default limit if not provided
	if limit <= 0 {
		limit = 100
	}

	requestDelay := s.config.CoingeckoMarkets.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = MARKETS_DEFAULT_REQUEST_DELAY
	}

	// Create parameters for top markets request
	params := interfaces.MarketsParams{
		Currency: currency,
		Order:    "market_cap_desc", // Order by market cap to get top tokens
		PerPage:  MARKETS_DEFAULT_CHUNK_SIZE,
	}

	// Apply parameters normalization from config
	params = s.getParamsOverride(params)

	// Calculate page range based on limit
	perPage := params.PerPage
	pageFrom := 1
	pageTo := (limit + perPage - 1) / perPage // Ceiling division

	// Create PaginatedFetcher with parameters
	fetcher := NewPaginatedFetcher(s.apiClient, pageFrom, pageTo, requestDelayMs, params)

	// Use PaginatedFetcher to get markets data as [][]byte
	tokensData, err := fetcher.FetchData()
	if err != nil {
		log.Printf("PaginatedFetcher failed to fetch top markets data: %v", err)
		return nil, fmt.Errorf("failed to fetch top markets data: %w", err)
	}

	// Cache tokens by their IDs
	marketData, err := s.cacheTokensByID(tokensData)
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded and cached top markets data with %d coins", len(marketData))
	return interfaces.MarketsResponse(marketData), nil
}

func (s *Service) SubscribeTopMarketsUpdate() chan struct{} {
	return s.subscriptionManager.Subscribe()
}

func (s *Service) Unsubscribe(ch chan struct{}) {
	s.subscriptionManager.Unsubscribe(ch)
}
