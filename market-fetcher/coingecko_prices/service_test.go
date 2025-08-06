package coingecko_prices

import (
	"context"
	"testing"
	"time"

	cg "github.com/status-im/market-proxy/interfaces"
	mock_interfaces "github.com/status-im/market-proxy/interfaces/mocks"

	"github.com/status-im/market-proxy/cache"
	cache_mocks "github.com/status-im/market-proxy/cache/mocks"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Test data constants
var (
	samplePriceData1 = []byte(`{"usd":45000,"eur":38000}`)
	samplePriceData2 = []byte(`{"usd":3000,"eur":2500}`)
)

func createMockTokensService(ctrl *gomock.Controller) *mock_interfaces.MockCoingeckoTokensService {
	mockTokensService := mock_interfaces.NewMockCoingeckoTokensService(ctrl)
	mockTokensService.EXPECT().GetTokens().Return([]cg.Token{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}).AnyTimes()
	mockTokensService.EXPECT().GetTokenIds().Return([]string{
		"bitcoin", "ethereum",
	}).AnyTimes()
	mockTokensService.EXPECT().SubscribeOnTokensUpdate().Return(make(chan struct{})).AnyTimes()
	mockTokensService.EXPECT().Unsubscribe(gomock.Any()).AnyTimes()
	return mockTokensService
}

func createTestConfig() *config.Config {
	return &config.Config{
		CoingeckoPrices: config.CoingeckoPricesFetcher{
			Currencies: []string{"usd", "eur"},
			TTL:        30 * time.Second,
			Tiers: []config.PriceTier{
				{
					Name:           "top-1000",
					TokenFrom:      1,
					TokenTo:        1000,
					UpdateInterval: 30 * time.Second,
				},
				{
					Name:           "top-1001-10000",
					TokenFrom:      1001,
					TokenTo:        10000,
					UpdateInterval: 5 * time.Minute,
				},
			},
		},
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
	}
}

func TestService_Basic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock cache service - expect possible cache operations from periodic updater
	mockCache := cache_mocks.NewMockCache(ctrl)
	mockCache.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Create mock markets service
	mockMarketsService := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
	mockMarketsService.EXPECT().SubscribeTopMarketsUpdate().Return(make(chan struct{})).AnyTimes()
	mockMarketsService.EXPECT().TopMarketIds(10000).Return([]string{"bitcoin", "ethereum"}, nil).AnyTimes()
	mockMarketsService.EXPECT().Unsubscribe(gomock.Any()).AnyTimes()

	// Create test config
	cfg := createTestConfig()

	// Create price service
	mockTokensService := createMockTokensService(ctrl)
	priceService := NewService(mockCache, cfg, mockMarketsService, mockTokensService)

	// Test Start method
	err := priceService.Start(context.Background())
	assert.NoError(t, err)
	defer priceService.Stop()

	// Test SimplePrices with empty IDs - no cache calls should be made
	response, _, err := priceService.SimplePrices(context.Background(), cg.PriceParams{
		IDs:        []string{},
		Currencies: []string{"usd"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response, 0)
}

func TestService_SimplePricesWithMissingData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock cache service
	mockCache := cache_mocks.NewMockCache(ctrl)

	// Create mock markets service
	mockMarketsService := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
	mockMarketsService.EXPECT().SubscribeTopMarketsUpdate().Return(make(chan struct{})).AnyTimes()
	mockMarketsService.EXPECT().TopMarketIds(10000).Return([]string{"bitcoin", "ethereum"}, nil).AnyTimes()
	mockMarketsService.EXPECT().Unsubscribe(gomock.Any()).AnyTimes()

	// Create test config
	cfg := createTestConfig()

	// Create price service
	mockTokensService := createMockTokensService(ctrl)
	priceService := NewService(mockCache, cfg, mockMarketsService, mockTokensService)

	// Test SimplePrices with data not in cache - should return empty results
	params := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}

	// Cache returns missing keys (not found)
	expectedCacheKeys := []string{"price:id:bitcoin", "price:id:ethereum"}
	mockCache.EXPECT().Get(expectedCacheKeys).Return(
		map[string][]byte{}, // No cached data
		expectedCacheKeys,   // All keys are missing
		nil,
	)

	response, _, err := priceService.SimplePrices(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestService_CacheKeys(t *testing.T) {
	// Test cache key generation for individual token IDs
	key1 := createTokenIDCacheKey("bitcoin")
	key2 := createTokenIDCacheKey("ethereum")
	key3 := createTokenIDCacheKey("bitcoin")

	// Test single token key format
	assert.Equal(t, "price:id:bitcoin", key1)
	assert.Equal(t, "price:id:ethereum", key2)

	// Same token should create same key
	assert.Equal(t, key1, key3)

	// Different tokens should create different keys
	assert.NotEqual(t, key1, key2)

	// All keys should contain the prefix
	assert.Contains(t, key1, "price:id:")
	assert.Contains(t, key2, "price:id:")
}

func TestService_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock cache service - expect possible cache operations from periodic updater
	mockCache := cache_mocks.NewMockCache(ctrl)
	mockCache.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Create test config
	cfg := createTestConfig()

	// Create mock markets service
	mockMarketsService := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
	mockMarketsService.EXPECT().SubscribeTopMarketsUpdate().Return(make(chan struct{})).AnyTimes()
	mockMarketsService.EXPECT().TopMarketIds(10000).Return([]string{"bitcoin", "ethereum"}, nil).AnyTimes()
	mockMarketsService.EXPECT().Unsubscribe(gomock.Any()).AnyTimes()

	// Create price service
	mockTokensService := createMockTokensService(ctrl)
	priceService := NewService(mockCache, cfg, mockMarketsService, mockTokensService)

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
	cfg := createTestConfig()

	// Create price service without cache
	priceService := NewService(nil, cfg, nil, nil)

	// Test Start should fail
	err := priceService.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache dependency not provided")
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
