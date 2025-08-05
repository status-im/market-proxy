package coingecko_markets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/status-im/market-proxy/cache"
	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

const (
	MARKETS_DEFAULT_CHUNK_SIZE    = 250  // CoinGecko's API max per_page value
	MARKETS_DEFAULT_REQUEST_DELAY = 1000 // 1 second in milliseconds
	ID_FIELD                      = "id"
)

// Service provides markets data fetching functionality with caching
type Service struct {
	cache         cache.Cache
	config        *config.Config
	metricsWriter *metrics.MetricsWriter
	apiClient     APIClient
}

func NewService(cache cache.Cache, config *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarkets)
	apiClient := NewCoinGeckoClient(config)

	return &Service{
		cache:         cache,
		config:        config,
		metricsWriter: metricsWriter,
		apiClient:     apiClient,
	}
}

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}
	return nil
}

// Stop implements core.Interface
func (s *Service) Stop() {
	// Markets service doesn't need shutdown logic for now
	// Cache will handle its own cleanup
}

// parseTokensData parses tokens data and extracts market data with cache keys
func (s *Service) parseTokensData(tokensData [][]byte) ([]interface{}, map[string][]byte, error) {
	marketData := make([]interface{}, 0, len(tokensData))
	cacheData := make(map[string][]byte)

	for _, tokenBytes := range tokensData {
		var tokenData interface{}
		if err := json.Unmarshal(tokenBytes, &tokenData); err != nil {
			log.Printf("Failed to unmarshal token data: %v", err)
			continue
		}

		// Extract ID and create cache key directly
		if tokenMap, ok := tokenData.(map[string]interface{}); ok {
			if id, exists := tokenMap[ID_FIELD]; exists {
				if tokenID, ok := id.(string); ok && tokenID != "" {
					cacheKey := getCacheKey(tokenID)
					cacheData[cacheKey] = tokenBytes
				} else {
					log.Printf("Invalid id type in market data: %T", id)
					continue
				}
			} else {
				log.Printf("Missing '%s' field in market data", ID_FIELD)
				continue
			}
		} else {
			log.Printf("Invalid market data format: %T", tokenData)
			continue
		}

		marketData = append(marketData, tokenData)
	}

	return marketData, cacheData, nil
}

// cacheTokensByID parses tokens data and caches each token by its CoinGecko ID
func (s *Service) cacheTokensByID(tokensData [][]byte) ([]interface{}, error) {
	// Parse tokens data
	marketData, cacheData, err := s.parseTokensData(tokensData)
	if err != nil {
		return nil, err
	}

	// Cache tokens directly
	if len(cacheData) > 0 {
		err := s.cache.Set(cacheData, s.config.CoingeckoMarkets.TTL)
		if err != nil {
			log.Printf("Failed to cache tokens data: %v", err)
			return nil, fmt.Errorf("failed to cache tokens data: %w", err)
		}
		log.Printf("Successfully cached %d tokens by their coingecko id", len(cacheData))
	}

	return marketData, nil
}

// Markets fetches markets data using cache with specified parameters
// Returns full CoinGecko markets response in APIResponse format
func (s *Service) Markets(params cg.MarketsParams) (cg.MarketsResponse, cg.CacheStatus, error) {
	// Check if specific IDs are requested
	if len(params.IDs) > 0 {
		return s.MarketsByIds(params)
	}

	// TODO: Implement general markets fetching without specific IDs
	log.Printf("Markets called without specific IDs - returning empty array (TODO: implement general fetching)")
	return cg.MarketsResponse([]interface{}{}), cg.CacheStatusMiss, nil
}

// MarketsByIds fetches markets data for specific token IDs using cache
func (s *Service) MarketsByIds(params cg.MarketsParams) (response cg.MarketsResponse, cacheStatus cg.CacheStatus, err error) {
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
		return cg.MarketsResponse(marketData), cg.CacheStatusFull, nil
	}

	if len(cachedData) > 0 {
		cacheStatus = cg.CacheStatusPartial
	} else {
		cacheStatus = cg.CacheStatusMiss
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
	return cg.MarketsResponse(marketData), cacheStatus, nil
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
func (s *Service) TopMarkets(limit int, currency string) (cg.MarketsResponse, error) {
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
	params := cg.MarketsParams{
		Currency: currency,
		Order:    "market_cap_desc", // Order by market cap to get top tokens
		PerPage:  MARKETS_DEFAULT_CHUNK_SIZE,
	}

	// Apply parameters normalization from config
	params = s.getParamsOverride(params)

	// Create PaginatedFetcher with parameters
	fetcher := NewPaginatedFetcher(s.apiClient, limit, requestDelayMs, params)

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
	return cg.MarketsResponse(marketData), nil
}
