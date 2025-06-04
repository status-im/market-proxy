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
	assert.Len(t, response, 0)
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
	assert.Len(t, response, 0) // Should be empty since loader returns empty data
}

func TestService_CacheKeys(t *testing.T) {
	// Test cache key generation with different parameter combinations
	params1 := PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd"},
	}
	keys1 := createCacheKeys(params1)

	params2 := PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd", "eur"},
	}
	keys2 := createCacheKeys(params2)

	params3 := PriceParams{
		IDs:        []string{"ethereum"},
		Currencies: []string{"usd"},
	}
	keys3 := createCacheKeys(params3)

	params4 := PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd"},
	}
	keys4 := createCacheKeys(params4)

	// Test single token
	assert.Len(t, keys1, 1)
	assert.Equal(t, "simple_price:bitcoin:usd", keys1[0])

	// Different currencies should create different keys
	assert.Len(t, keys2, 1)
	assert.Equal(t, "simple_price:bitcoin:usd,eur", keys2[0])
	assert.NotEqual(t, keys1[0], keys2[0])

	// Different token should create different key
	assert.Len(t, keys3, 1)
	assert.Equal(t, "simple_price:ethereum:usd", keys3[0])
	assert.NotEqual(t, keys1[0], keys3[0])

	// Multiple tokens should create multiple keys
	assert.Len(t, keys4, 2)
	assert.Equal(t, "simple_price:bitcoin:usd", keys4[0])
	assert.Equal(t, "simple_price:ethereum:usd", keys4[1])

	// All keys should contain the prefix
	for _, key := range keys4 {
		assert.Contains(t, key, "simple_price:")
	}
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
