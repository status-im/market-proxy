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
func (s *Service) SimplePrices(params PriceParams) (*PriceResponse, error) {
	if len(params.IDs) == 0 {
		return &PriceResponse{
			Data:         make(map[string]PriceData),
			RequestedIDs: params.IDs,
			FoundIDs:     []string{},
			MissingIDs:   []string{},
		}, nil
	}

	// Create cache keys based on token IDs and currencies
	cacheKeys := make([]string, len(params.IDs))
	for i, id := range params.IDs {
		// Create a cache key that includes currencies for uniqueness
		cacheKeys[i] = createCacheKey(id, params.Currencies)
	}

	// Create a placeholder loader that does nothing for now
	// This will be implemented in future steps
	loader := func(missingKeys []string) (map[string][]byte, error) {
		// For now, return empty data for missing keys
		// TODO: Implement actual price loading from Coingecko API
		return make(map[string][]byte), nil
	}

	// Get data from cache
	cachedData, err := s.cache.GetOrLoad(cacheKeys, loader, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices from cache: %w", err)
	}

	// Parse cached data and build response
	response := &PriceResponse{
		Data:         make(map[string]PriceData),
		RequestedIDs: params.IDs,
		FoundIDs:     []string{},
		MissingIDs:   []string{},
	}

	for i, id := range params.IDs {
		cacheKey := cacheKeys[i]
		if data, found := cachedData[cacheKey]; found {
			var priceData PriceData
			if err := json.Unmarshal(data, &priceData); err == nil {
				response.Data[id] = priceData
				response.FoundIDs = append(response.FoundIDs, id)
			} else {
				response.MissingIDs = append(response.MissingIDs, id)
			}
		} else {
			response.MissingIDs = append(response.MissingIDs, id)
		}
	}

	return response, nil
}

// createCacheKey creates a cache key for a token ID and currencies
func createCacheKey(tokenID string, currencies []string) string {
	// Create a deterministic key that includes currencies
	// Format: "prices:{tokenID}:{currency1,currency2,...}"
	currenciesStr := ""
	for i, currency := range currencies {
		if i > 0 {
			currenciesStr += ","
		}
		currenciesStr += currency
	}
	return fmt.Sprintf("prices:%s:%s", tokenID, currenciesStr)
}
