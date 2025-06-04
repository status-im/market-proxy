package coingecko_prices

import (
	"context"
	"testing"

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
		},
	}

	// Create price service
	priceService := NewService(cacheService, cfg)

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

	// Create test config
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
		},
	}

	// Create price service
	priceService := NewService(cacheService, cfg)

	// Test SimplePrices with data not in cache
	params := PriceParams{
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
		},
	}

	// Create price service
	priceService := NewService(cacheService, cfg)

	// Test parameters
	params := PriceParams{
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
