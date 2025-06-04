package coingecko_prices

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
)

// Service provides price fetching functionality with caching
type Service struct {
	cache   cache.Cache
	fetcher *ChunksFetcher
}

// NewService creates a new price service with the given cache and config
func NewService(cache cache.Cache, config *config.Config) *Service {
	// Create API client
	apiClient := NewCoinGeckoClient(config)

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
		cache:   cache,
		fetcher: fetcher,
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
func (s *Service) SimplePrices(params PriceParams) (SimplePriceResponse, error) {
	if len(params.IDs) == 0 {
		return SimplePriceResponse{}, nil
	}

	// Create cache keys for each token ID
	cacheKeys := createCacheKeys(params)

	// Create loader that uses ChunksFetcher to fetch missing data
	loader := func(missingKeys []string) (map[string][]byte, error) {
		return s.loadMissingPrices(missingKeys, params)
	}

	// Get data from cache for all keys
	cachedData, err := s.cache.GetOrLoad(cacheKeys, loader, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices from cache: %w", err)
	}

	// Combine results from all cache keys
	result := make(SimplePriceResponse)

	for i, tokenID := range params.IDs {
		cacheKey := cacheKeys[i]
		if data, found := cachedData[cacheKey]; found {
			var tokenData map[string]interface{}
			if err := json.Unmarshal(data, &tokenData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal cached price data for %s: %w", tokenID, err)
			}
			// Add token data to result
			for key, value := range tokenData {
				result[key] = value
			}
		}
	}

	return result, nil
}

// loadMissingPrices loads price data for missing cache keys using ChunksFetcher
func (s *Service) loadMissingPrices(missingKeys []string, params PriceParams) (map[string][]byte, error) {
	log.Printf("Loading missing price data for %d cache keys", len(missingKeys))

	// Extract token IDs from missing cache keys
	missingTokens := extractTokensFromKeys(missingKeys)

	if len(missingTokens) == 0 {
		return make(map[string][]byte), nil
	}

	// Use ChunksFetcher to get prices from CoinGecko API
	prices, err := s.fetcher.FetchPrices(missingTokens, params.Currencies)
	if err != nil {
		log.Printf("ChunksFetcher failed to fetch prices: %v", err)
		return make(map[string][]byte), nil // Return empty data instead of error
	}

	// Convert fetched prices to cache format
	result := make(map[string][]byte)

	// For each missing key, create the corresponding cache data
	for _, cacheKey := range missingKeys {
		tokenID := extractTokenIDFromKey(cacheKey)

		// Create token-specific response in CoinGecko format
		tokenResponse := make(map[string]interface{})

		// Add prices for each currency
		for _, currency := range params.Currencies {
			if currencyPrices, exists := prices[currency]; exists {
				if price, hasPrice := currencyPrices[tokenID]; hasPrice {
					tokenResponse[currency] = price
				}
			}
		}

		// If we have any data for this token, marshal it
		if len(tokenResponse) > 0 {
			// Wrap it in the expected CoinGecko format: {tokenID: {currency: price, ...}}
			wrappedResponse := map[string]interface{}{
				tokenID: tokenResponse,
			}

			data, err := json.Marshal(wrappedResponse)
			if err != nil {
				log.Printf("Failed to marshal price data for token %s: %v", tokenID, err)
				continue
			}

			result[cacheKey] = data
		}
	}

	log.Printf("Loaded price data for %d tokens, cached %d keys", len(missingTokens), len(result))
	return result, nil
}
