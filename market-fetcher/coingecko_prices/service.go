package coingecko_prices

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/status-im/market-proxy/cache"
)

// Service provides price fetching functionality with caching
type Service struct {
	cache cache.Cache
}

// NewService creates a new price service with the given cache
func NewService(cache cache.Cache) *Service {
	return &Service{
		cache: cache,
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

	// Create a placeholder loader that does nothing for now
	// This will be implemented in future steps
	loader := func(missingKeys []string) (map[string][]byte, error) {
		// For now, return empty data for missing keys
		// TODO: Implement actual price loading from Coingecko API
		return make(map[string][]byte), nil
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

// createCacheKeys creates cache keys for each token ID
func createCacheKeys(params PriceParams) []string {
	keys := make([]string, len(params.IDs))

	// Create currencies string for the key
	currenciesStr := ""
	for i, currency := range params.Currencies {
		if i > 0 {
			currenciesStr += ","
		}
		currenciesStr += currency
	}

	// Create a key for each token ID
	for i, tokenID := range params.IDs {
		keys[i] = fmt.Sprintf("simple_price:%s:%s", tokenID, currenciesStr)
	}

	return keys
}
