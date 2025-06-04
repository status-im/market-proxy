package coingecko_prices

import (
	"context"
	"testing"

	"github.com/status-im/market-proxy/cache"
	"github.com/stretchr/testify/assert"
)

func TestService_Basic(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create price service
	priceService := NewService(cacheService)

	// Test Start method
	err := priceService.Start(context.Background())
	assert.NoError(t, err)

	// Test SimplePrices with empty IDs
	response, err := priceService.SimplePrices(PriceParams{
		IDs:        []string{},
		Currencies: []string{"usd"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 0)
	assert.Len(t, response.RequestedIDs, 0)
	assert.Len(t, response.FoundIDs, 0)
	assert.Len(t, response.MissingIDs, 0)
}

func TestService_SimplePricesWithMissingData(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create price service
	priceService := NewService(cacheService)

	// Test SimplePrices with data not in cache
	params := PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}

	response, err := priceService.SimplePrices(params)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, params.IDs, response.RequestedIDs)
	assert.Len(t, response.FoundIDs, 0)   // No data should be found since loader returns empty
	assert.Len(t, response.MissingIDs, 2) // All IDs should be missing
	assert.Len(t, response.Data, 0)       // No data should be returned
}

func TestService_CacheKeys(t *testing.T) {
	// Test cache key generation
	key1 := createCacheKey("bitcoin", []string{"usd"})
	key2 := createCacheKey("bitcoin", []string{"usd", "eur"})
	key3 := createCacheKey("ethereum", []string{"usd"})

	assert.Equal(t, "prices:bitcoin:usd", key1)
	assert.Equal(t, "prices:bitcoin:usd,eur", key2)
	assert.Equal(t, "prices:ethereum:usd", key3)

	// Keys should be different for different currencies
	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key1, key3)
}

func TestService_StartStop(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create price service
	priceService := NewService(cacheService)

	// Test Start
	err := priceService.Start(context.Background())
	assert.NoError(t, err)

	// Test Stop (should not panic)
	assert.NotPanics(t, func() {
		priceService.Stop()
	})
}

func TestService_StartWithoutCache(t *testing.T) {
	// Create price service without cache
	priceService := NewService(nil)

	// Test Start should fail
	err := priceService.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache dependency not provided")
}
