package coingecko_prices

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/metrics"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
)

// Service provides price fetching functionality with caching
type Service struct {
	cache         cache.Cache
	fetcher       *ChunksFetcher
	config        *config.Config
	metricsWriter *metrics.MetricsWriter
}

// NewService creates a new price service with the given cache and config
func NewService(cache cache.Cache, config *config.Config) *Service {
	// Create metrics writer
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)

	// Create API client
	apiClient := NewCoinGeckoClient(config, metricsWriter)

	// Get configuration values or use defaults
	chunkSize := config.CoingeckoPrices.ChunkSize
	if chunkSize <= 0 {
		chunkSize = DEFAULT_CHUNK_SIZE
	}

	requestDelay := config.CoingeckoPrices.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = DEFAULT_REQUEST_DELAY
	}

	fetcher := NewChunksFetcher(apiClient, chunkSize, requestDelayMs)

	return &Service{
		cache:         cache,
		fetcher:       fetcher,
		config:        config,
		metricsWriter: metricsWriter,
	}
}

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	// Price service doesn't need startup logic for now
	// Just validate that cache is available
	if s.cache == nil {
		return fmt.Errorf("price service: cache dependency not provided")
	}
	return nil
}

// Stop implements core.Interface
func (s *Service) Stop() {
	// Price service doesn't need shutdown logic for now
	// Cache will handle its own cleanup
}

// SimplePrices fetches prices for the given parameters using cache
// Returns raw CoinGecko JSON response
func (s *Service) SimplePrices(params cg.PriceParams) (cg.SimplePriceResponse, error) {
	if len(params.IDs) == 0 {
		return cg.SimplePriceResponse{}, nil
	}

	// Create cache keys for each token ID
	cacheKeys := createCacheKeys(params)

	// Create loader that uses ChunksFetcher to fetch missing data
	loader := func(missingKeys []string) (map[string][]byte, error) {
		return s.loadMissingPrices(missingKeys, params)
	}

	// Get data from cache for all keys
	cachedData, err := s.cache.GetOrLoad(cacheKeys, loader, true, s.config.CoingeckoPrices.TTL)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices from cache: %w", err)
	}

	// Combine results from all cache keys
	fullResponse := make(cg.SimplePriceResponse)

	for i, tokenID := range params.IDs {
		cacheKey := cacheKeys[i]
		if data, found := cachedData[cacheKey]; found {
			var tokenData map[string]interface{}
			if err := json.Unmarshal(data, &tokenData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal cached price data for %s: %w", tokenID, err)
			}

			// Add token data directly to full response (will be filtered later)
			fullResponse[tokenID] = tokenData
		}
	}

	// Filter the response according to user parameters
	filteredResponse := stripResponse(fullResponse, params)

	return filteredResponse, nil
}

// loadMissingPrices loads price data for missing cache keys using ChunksFetcher
func (s *Service) loadMissingPrices(missingKeys []string, params cg.PriceParams) (map[string][]byte, error) {
	log.Printf("Loading missing price data for %d cache keys", len(missingKeys))

	// Extract token IDs from missing cache keys
	missingTokens := extractTokensFromKeys(missingKeys)

	if len(missingTokens) == 0 {
		return make(map[string][]byte), nil
	}

	// Merge config currencies with user-requested currencies
	allCurrencies := s.mergeCurrencies(params.Currencies)

	// Use ChunksFetcher to get prices from CoinGecko API
	fetchParams := cg.PriceParams{
		IDs:                  missingTokens,
		Currencies:           allCurrencies,
		IncludeMarketCap:     true,
		Include24hrVol:       true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
	}
	tokenData, err := s.fetcher.FetchPrices(fetchParams)
	if err != nil {
		log.Printf("ChunksFetcher failed to fetch prices: %v", err)
		return make(map[string][]byte), nil // Return empty data instead of error
	}

	// Map token data to cache keys
	result := make(map[string][]byte)
	for _, cacheKey := range missingKeys {
		tokenID := extractTokenIDFromKey(cacheKey)
		if data, found := tokenData[tokenID]; found {
			result[cacheKey] = data
		}
	}

	log.Printf("Loaded price data for %d tokens, cached %d keys", len(missingTokens), len(result))
	return result, nil
}

// mergeCurrencies merges config currencies with user-requested currencies
// Config currencies come first, then any additional user currencies that aren't in config
func (s *Service) mergeCurrencies(userCurrencies []string) []string {
	// Start with config currencies
	configCurrencies := s.getConfigCurrencies()
	allCurrencies := make([]string, len(configCurrencies))
	copy(allCurrencies, configCurrencies)

	// Create a set of existing currencies for fast lookup
	currencySet := make(map[string]bool)
	for _, currency := range configCurrencies {
		currencySet[currency] = true
	}

	// Add user currencies that aren't already in config
	for _, currency := range userCurrencies {
		if !currencySet[currency] {
			allCurrencies = append(allCurrencies, currency)
			currencySet[currency] = true
		}
	}

	return allCurrencies
}

// getConfigCurrencies returns the currencies from config, with fallback to default
func (s *Service) getConfigCurrencies() []string {
	if s.config != nil && len(s.config.CoingeckoPrices.Currencies) > 0 {
		return s.config.CoingeckoPrices.Currencies
	}
	// Fallback to default currencies if config is not available or empty
	return []string{"usd", "eur", "btc", "eth"}
}

// Healthy checks if the service is operational
func (s *Service) Healthy() bool {
	return true
}
