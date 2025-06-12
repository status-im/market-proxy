package coingecko_prices

import (
	"context"
	"testing"
	"time"

	cg "github.com/status-im/market-proxy/coingecko_common"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

func TestService_Basic(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        30 * time.Second,
		},
	}

	// Create price service
	priceService := NewService(cacheService, cfg)

	// Test Start method
	err := priceService.Start(context.Background())
	assert.NoError(t, err)

	// Test SimplePrices with empty IDs
	response, err := priceService.SimplePrices(cg.PriceParams{
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

	// Create test config
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        30 * time.Second,
		},
	}

	// Create price service
	priceService := NewService(cacheService, cfg)

	// Test SimplePrices with data not in cache
	params := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}

	response, err := priceService.SimplePrices(params)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	// Response might be empty since we're using real API client that might fail in tests
}

func TestService_CacheKeys(t *testing.T) {
	// Test cache key generation with different parameter combinations
	params1 := cg.PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd"},
	}
	keys1 := createCacheKeys(params1)

	params2 := cg.PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd", "eur"},
	}
	keys2 := createCacheKeys(params2)

	params3 := cg.PriceParams{
		IDs:        []string{"ethereum"},
		Currencies: []string{"usd"},
	}
	keys3 := createCacheKeys(params3)

	params4 := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd"},
	}
	keys4 := createCacheKeys(params4)

	// Test single token
	assert.Len(t, keys1, 1)
	assert.Equal(t, "simple_price:bitcoin", keys1[0])

	// Different currencies should create same keys (currencies not in key anymore)
	assert.Len(t, keys2, 1)
	assert.Equal(t, "simple_price:bitcoin", keys2[0])
	assert.Equal(t, keys1[0], keys2[0])

	// Different token should create different key
	assert.Len(t, keys3, 1)
	assert.Equal(t, "simple_price:ethereum", keys3[0])
	assert.NotEqual(t, keys1[0], keys3[0])

	// Multiple tokens should create multiple keys
	assert.Len(t, keys4, 2)
	assert.Equal(t, "simple_price:bitcoin", keys4[0])
	assert.Equal(t, "simple_price:ethereum", keys4[1])

	// All keys should contain the prefix
	for _, key := range keys4 {
		assert.Contains(t, key, "simple_price:")
	}
}

func TestService_StartStop(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        30 * time.Second,
		},
	}

	// Create price service
	priceService := NewService(cacheService, cfg)

	// Test Start
	err := priceService.Start(context.Background())
	assert.NoError(t, err)

	// Test Stop (should not panic)
	assert.NotPanics(t, func() {
		priceService.Stop()
	})
}

func TestService_StartWithoutCache(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        30 * time.Second,
		},
	}

	// Create price service without cache
	priceService := NewService(nil, cfg)

	// Test Start should fail
	err := priceService.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache dependency not provided")
}

func TestService_LoadMissingPrices(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        30 * time.Second,
		},
	}

	// Create price service
	priceService := NewService(cacheService, cfg)

	// Test parameters
	params := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd"},
	}

	// Test missing keys (simplified format)
	missingKeys := []string{
		"simple_price:bitcoin",
		"simple_price:ethereum",
	}

	// Call loadMissingPrices
	result, err := priceService.loadMissingPrices(missingKeys, params)

	// Should not return error even if API fails
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Result might be empty since we're using real API that might fail in tests

	// Test with empty missing keys
	emptyResult, err := priceService.loadMissingPrices([]string{}, params)
	assert.NoError(t, err)
	assert.NotNil(t, emptyResult)
	assert.Len(t, emptyResult, 0)
}

func TestService_MergeCurrencies(t *testing.T) {
	// Create test config with some currencies
	cfg := &config.Config{
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur", "btc"},
			TTL:        30 * time.Second,
		},
	}

	// Create price service
	priceService := &Service{config: cfg}

	// Test merging with no user currencies
	result1 := priceService.mergeCurrencies([]string{})
	expected1 := []string{"usd", "eur", "btc"}
	assert.Equal(t, expected1, result1)

	// Test merging with user currencies that are already in config
	result2 := priceService.mergeCurrencies([]string{"usd", "eur"})
	expected2 := []string{"usd", "eur", "btc"}
	assert.Equal(t, expected2, result2)

	// Test merging with new user currencies
	result3 := priceService.mergeCurrencies([]string{"eth", "ada"})
	expected3 := []string{"usd", "eur", "btc", "eth", "ada"}
	assert.Equal(t, expected3, result3)

	// Test merging with mix of existing and new currencies
	result4 := priceService.mergeCurrencies([]string{"usd", "eth", "eur", "dot"})
	expected4 := []string{"usd", "eur", "btc", "eth", "dot"}
	assert.Equal(t, expected4, result4)
}

func TestService_GetConfigCurrencies(t *testing.T) {
	// Test with config currencies
	cfg1 := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        30 * time.Second,
		},
	}
	service1 := &Service{config: cfg1}
	result1 := service1.getConfigCurrencies()
	assert.Equal(t, []string{"usd", "eur"}, result1)

	// Test with empty config currencies (fallback)
	cfg2 := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{},
			TTL:        30 * time.Second,
		},
	}
	service2 := &Service{config: cfg2}
	result2 := service2.getConfigCurrencies()
	assert.Equal(t, []string{"usd", "eur", "btc", "eth"}, result2)

	// Test with nil config (fallback)
	service3 := &Service{config: nil}
	result3 := service3.getConfigCurrencies()
	assert.Equal(t, []string{"usd", "eur", "btc", "eth"}, result3)
}

// TestService_TTLCaching verifies that cache respects TTL (Time To Live) setting:
// 1. First request loads data from network (loader called)
// 2. Subsequent requests use cached data while TTL is valid (loader not called)
// 3. After TTL expires, data is loaded from network again (loader called)
func TestService_TTLCaching(t *testing.T) {
	// Create cache service with short TTL for testing
	cacheConfig := cache.Config{
		GoCache: cache.GoCacheConfig{
			DefaultExpiration: 1 * time.Second, // Short default expiration
			CleanupInterval:   500 * time.Millisecond,
		},
	}
	cacheService := cache.NewService(cacheConfig)

	// Create test config with short TTL
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        500 * time.Millisecond, // Very short TTL for testing
		},
	}

	// Track loader calls
	var loaderCallCount int

	// Create a mock loader function
	mockLoader := func(missingKeys []string) (map[string][]byte, error) {
		loaderCallCount++
		result := make(map[string][]byte)
		for _, key := range missingKeys {
			// Return mock price data
			result[key] = []byte(`{"usd": 50000, "eur": 42000}`)
		}
		return result, nil
	}

	// Test using cache directly
	keys := []string{"simple_price:bitcoin"}

	// Test 1: First call should trigger loader
	loaderCallCount = 0
	data1, err := cacheService.GetOrLoad(keys, mockLoader, true, cfg.CoingeckoPrices.TTL)
	assert.NoError(t, err)
	assert.NotNil(t, data1)
	assert.Equal(t, 1, loaderCallCount, "First call should trigger loader")
	assert.Contains(t, data1, "simple_price:bitcoin")

	// Test 2: Second call immediately should use cache (no loader call)
	data2, err := cacheService.GetOrLoad(keys, mockLoader, true, cfg.CoingeckoPrices.TTL)
	assert.NoError(t, err)
	assert.NotNil(t, data2)
	assert.Equal(t, 1, loaderCallCount, "Second call should use cache, not trigger loader")
	assert.Equal(t, data1, data2, "Data should be identical")

	// Test 3: Third call immediately should still use cache
	data3, err := cacheService.GetOrLoad(keys, mockLoader, true, cfg.CoingeckoPrices.TTL)
	assert.NoError(t, err)
	assert.NotNil(t, data3)
	assert.Equal(t, 1, loaderCallCount, "Third call should still use cache")

	// Test 4: Wait for TTL to expire and call again - should trigger loader
	time.Sleep(600 * time.Millisecond) // Wait for TTL to expire (500ms + buffer)

	data4, err := cacheService.GetOrLoad(keys, mockLoader, true, cfg.CoingeckoPrices.TTL)
	assert.NoError(t, err)
	assert.NotNil(t, data4)
	assert.Equal(t, 2, loaderCallCount, "Fourth call after TTL expiry should trigger loader again")
}

// TestService_TTLCachingWithDifferentKeys verifies cache behavior with multiple keys:
// 1. Different keys are cached independently
// 2. Mixed requests (some keys cached, some not) work correctly
// 3. TTL expiry affects all cached keys uniformly
func TestService_TTLCachingWithDifferentKeys(t *testing.T) {
	// Create cache service with short TTL for testing
	cacheConfig := cache.Config{
		GoCache: cache.GoCacheConfig{
			DefaultExpiration: 1 * time.Second,
			CleanupInterval:   500 * time.Millisecond,
		},
	}
	cacheService := cache.NewService(cacheConfig)

	// Track loader calls and which keys were requested
	var loaderCallCount int
	var requestedKeys [][]string

	// Create a mock loader function
	mockLoader := func(missingKeys []string) (map[string][]byte, error) {
		loaderCallCount++
		requestedKeys = append(requestedKeys, append([]string(nil), missingKeys...))

		result := make(map[string][]byte)
		for _, key := range missingKeys {
			result[key] = []byte(`{"usd": 50000, "eur": 42000}`)
		}
		return result, nil
	}

	ttl := 500 * time.Millisecond

	// Test with first key
	keys1 := []string{"simple_price:bitcoin"}
	_, err := cacheService.GetOrLoad(keys1, mockLoader, true, ttl)
	assert.NoError(t, err)
	assert.Equal(t, 1, loaderCallCount)
	assert.Equal(t, []string{"simple_price:bitcoin"}, requestedKeys[0])

	// Test with second key (different from first)
	keys2 := []string{"simple_price:ethereum"}
	_, err = cacheService.GetOrLoad(keys2, mockLoader, true, ttl)
	assert.NoError(t, err)
	assert.Equal(t, 2, loaderCallCount)
	assert.Equal(t, []string{"simple_price:ethereum"}, requestedKeys[1])

	// Test with both keys - should only load missing one
	keysBoth := []string{"simple_price:bitcoin", "simple_price:ethereum"}
	dataBoth, err := cacheService.GetOrLoad(keysBoth, mockLoader, true, ttl)
	assert.NoError(t, err)
	assert.Equal(t, 2, loaderCallCount, "Should not call loader when all keys are in cache")
	assert.Len(t, dataBoth, 2)

	// Wait for expiry and test again
	time.Sleep(600 * time.Millisecond)

	dataBothExpired, err := cacheService.GetOrLoad(keysBoth, mockLoader, true, ttl)
	assert.NoError(t, err)
	assert.Equal(t, 3, loaderCallCount, "Should call loader again after TTL expiry")
	assert.Equal(t, []string{"simple_price:bitcoin", "simple_price:ethereum"}, requestedKeys[2])
	assert.Len(t, dataBothExpired, 2)
}
