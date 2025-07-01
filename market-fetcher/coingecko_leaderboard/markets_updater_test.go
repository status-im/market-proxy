package coingecko_leaderboard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMarketsFetcher is a mock implementation of MarketsFetcher interface
type MockMarketsFetcher struct {
	mock.Mock
}

func (m *MockMarketsFetcher) Markets(params cg.MarketsParams) (cg.MarketsResponse, error) {
	args := m.Called(params)
	return args.Get(0).(cg.MarketsResponse), args.Error(1)
}

func (m *MockMarketsFetcher) TopMarkets(limit int, currency string) (cg.MarketsResponse, error) {
	args := m.Called(limit, currency)
	return args.Get(0).(cg.MarketsResponse), args.Error(1)
}

// Helper function to create test config
func createTestMarketsConfig() *config.Config {
	return &config.Config{
		CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
			TopPricesLimit:           10,
			TopMarketsUpdateInterval: time.Second * 5,
			Currency:                 "usd",
		},
	}
}

// Helper function to create sample markets data
func createSampleMarketsData() cg.MarketsResponse {
	return cg.MarketsResponse([]interface{}{
		map[string]interface{}{
			"id":                          "bitcoin",
			"symbol":                      "btc",
			"name":                        "Bitcoin",
			"image":                       "https://coin-images.coingecko.com/coins/images/1/large/bitcoin.png",
			"current_price":               50000.0,
			"market_cap":                  950000000000.0,
			"total_volume":                25000000000.0,
			"price_change_percentage_24h": 2.5,
		},
		map[string]interface{}{
			"id":                          "ethereum",
			"symbol":                      "eth",
			"name":                        "Ethereum",
			"image":                       "https://coin-images.coingecko.com/coins/images/279/large/ethereum.png",
			"current_price":               3000.0,
			"market_cap":                  360000000000.0,
			"total_volume":                15000000000.0,
			"price_change_percentage_24h": -1.2,
		},
	})
}

func TestNewMarketsUpdater(t *testing.T) {
	t.Run("Creates new markets updater with correct dependencies", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}

		updater := NewMarketsUpdater(cfg, mockFetcher)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Equal(t, mockFetcher, updater.marketsFetcher)
		assert.Nil(t, updater.scheduler)
		assert.Nil(t, updater.onUpdate)
		assert.Nil(t, updater.GetCacheData())
	})

	t.Run("Works with nil fetcher", func(t *testing.T) {
		cfg := createTestMarketsConfig()

		updater := NewMarketsUpdater(cfg, nil)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Nil(t, updater.marketsFetcher)
	})
}

func TestMarketsUpdater_SetOnUpdateCallback(t *testing.T) {
	t.Run("Sets callback function", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		callbackCalled := false
		callback := func() {
			callbackCalled = true
		}

		updater.SetOnUpdateCallback(callback)

		assert.NotNil(t, updater.onUpdate)

		// Test callback is called
		updater.onUpdate()
		assert.True(t, callbackCalled)
	})

	t.Run("Overwrites existing callback", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		firstCallbackCalled := false
		secondCallbackCalled := false

		// Set first callback
		updater.SetOnUpdateCallback(func() {
			firstCallbackCalled = true
		})

		// Set second callback (should overwrite first)
		updater.SetOnUpdateCallback(func() {
			secondCallbackCalled = true
		})

		// Call the callback
		updater.onUpdate()

		assert.False(t, firstCallbackCalled)
		assert.True(t, secondCallbackCalled)
	})

	t.Run("Can set nil callback", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		// Set a callback first
		updater.SetOnUpdateCallback(func() {})
		assert.NotNil(t, updater.onUpdate)

		// Set to nil
		updater.SetOnUpdateCallback(nil)
		assert.Nil(t, updater.onUpdate)
	})
}

func TestMarketsUpdater_GetTopTokenIDs(t *testing.T) {
	t.Run("Returns nil when no cache data", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		result := updater.GetTopTokenIDs()

		assert.Nil(t, result)
	})

	t.Run("Returns nil when cache data is nil", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		updater.cache.Lock()
		updater.cache.data = nil
		updater.cache.Unlock()

		result := updater.GetTopTokenIDs()

		assert.Nil(t, result)
	})

	t.Run("Returns nil when cache data.Data is nil", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: nil}
		updater.cache.Unlock()

		result := updater.GetTopTokenIDs()

		assert.Nil(t, result)
	})

	t.Run("Returns empty slice when no coins have IDs", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		mockData := []CoinData{
			{ID: "", Symbol: "btc", Name: "Bitcoin"},
			{ID: "", Symbol: "eth", Name: "Ethereum"},
		}

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: mockData}
		updater.cache.Unlock()

		result := updater.GetTopTokenIDs()

		assert.Empty(t, result)
	})

	t.Run("Returns token IDs from cache data", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		mockData := []CoinData{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
			{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
			{ID: "cardano", Symbol: "ada", Name: "Cardano"},
		}

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: mockData}
		updater.cache.Unlock()

		result := updater.GetTopTokenIDs()

		expected := []string{"bitcoin", "ethereum", "cardano"}
		assert.Equal(t, expected, result)
	})

	t.Run("Filters out empty IDs", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		mockData := []CoinData{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
			{ID: "", Symbol: "eth", Name: "Ethereum"},
			{ID: "cardano", Symbol: "ada", Name: "Cardano"},
			{ID: "", Symbol: "sol", Name: "Solana"},
		}

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: mockData}
		updater.cache.Unlock()

		result := updater.GetTopTokenIDs()

		expected := []string{"bitcoin", "cardano"}
		assert.Equal(t, expected, result)
	})
}

func TestMarketsUpdater_fetchAndUpdate(t *testing.T) {
	t.Run("Successful fetch and update", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.On("TopMarkets", 10, "usd").Return(sampleData, nil)

		callbackCalled := false
		updater.SetOnUpdateCallback(func() {
			callbackCalled = true
		})

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		assert.True(t, callbackCalled)

		// Verify cache was updated
		cacheData := updater.GetCacheData()
		assert.NotNil(t, cacheData)
		assert.Len(t, cacheData.Data, 2)
		assert.Equal(t, "bitcoin", cacheData.Data[0].ID)
		assert.Equal(t, "ethereum", cacheData.Data[1].ID)

		mockFetcher.AssertExpectations(t)
	})

	t.Run("Uses default limit when config limit is 0", func(t *testing.T) {
		cfg := &config.Config{
			CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
				TopPricesLimit: 0, // Should use default 500
				Currency:       "usd",
			},
		}
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.On("TopMarkets", 500, "usd").Return(sampleData, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		mockFetcher.AssertExpectations(t)
	})

	t.Run("Uses default limit when config limit is negative", func(t *testing.T) {
		cfg := &config.Config{
			CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
				TopPricesLimit: -10, // Should use default 500
				Currency:       "usd",
			},
		}
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.On("TopMarkets", 500, "usd").Return(sampleData, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		mockFetcher.AssertExpectations(t)
	})

	t.Run("Handles fetcher error", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		expectedError := errors.New("API error")
		mockFetcher.On("TopMarkets", 10, "usd").Return(cg.MarketsResponse(nil), expectedError)

		callbackCalled := false
		updater.SetOnUpdateCallback(func() {
			callbackCalled = true
		})

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
		assert.False(t, callbackCalled)

		// Verify cache wasn't updated
		cacheData := updater.GetCacheData()
		assert.Nil(t, cacheData)

		mockFetcher.AssertExpectations(t)
	})

	t.Run("Handles empty response", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		emptyData := cg.MarketsResponse([]interface{}{})
		mockFetcher.On("TopMarkets", 10, "usd").Return(emptyData, nil)

		callbackCalled := false
		updater.SetOnUpdateCallback(func() {
			callbackCalled = true
		})

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		assert.True(t, callbackCalled)

		// Verify cache was updated with empty data
		cacheData := updater.GetCacheData()
		assert.NotNil(t, cacheData)
		assert.Len(t, cacheData.Data, 0)

		mockFetcher.AssertExpectations(t)
	})

	t.Run("Doesn't call callback when callback is nil", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.On("TopMarkets", 10, "usd").Return(sampleData, nil)

		// Don't set callback (should be nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		// Should not panic even without callback

		mockFetcher.AssertExpectations(t)
	})
}

func TestMarketsUpdater_Healthy(t *testing.T) {
	t.Run("Returns true when cache has data", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		mockData := []CoinData{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		}

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: mockData}
		updater.cache.Unlock()

		result := updater.Healthy()

		assert.True(t, result)
	})

	t.Run("Returns true when fetcher exists but no cache", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil
	})

	t.Run("Returns true when cache data is empty but fetcher exists", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: []CoinData{}}
		updater.cache.Unlock()

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil, even if cache is empty
	})

	t.Run("Returns true when fetcher is available but no cache", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil
	})

	t.Run("Returns false when fetcher is nil and no cache", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, nil)

		result := updater.Healthy()

		assert.False(t, result)
	})
}

func TestMarketsUpdater_StartStop(t *testing.T) {
	t.Run("Start creates and starts scheduler", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		mockFetcher.On("TopMarkets", 10, "usd").Return(sampleData, nil).Maybe()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		// Stop to clean up
		updater.Stop()
	})

	t.Run("Stop stops scheduler when it exists", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		mockFetcher.On("TopMarkets", 10, "usd").Return(sampleData, nil).Maybe()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start first
		err := updater.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		// Now stop
		updater.Stop()

		// Scheduler should still exist but be stopped
		assert.NotNil(t, updater.scheduler)
	})

	t.Run("Stop doesn't panic when scheduler is nil", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		// Call stop without starting
		assert.NotPanics(t, func() {
			updater.Stop()
		})
	})

	t.Run("Start with minimal update interval", func(t *testing.T) {
		cfg := &config.Config{
			CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
				TopMarketsUpdateInterval: time.Millisecond, // Minimal interval
				Currency:                 "usd",
			},
		}
		mockFetcher := &MockMarketsFetcher{}
		updater := NewMarketsUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		mockFetcher.On("TopMarkets", 500, "usd").Return(sampleData, nil).Maybe() // Default limit is 500

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		updater.Stop()
	})
}

func TestMarketsUpdater_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent cache access is safe", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewMarketsUpdater(cfg, &MockMarketsFetcher{})

		var wg sync.WaitGroup
		numGoroutines := 10

		// Set initial data
		mockData := []CoinData{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		}
		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: mockData}
		updater.cache.Unlock()

		// Start multiple readers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					data := updater.GetCacheData()
					tokenIDs := updater.GetTopTokenIDs()
					_ = data
					_ = tokenIDs
				}
			}()
		}

		// Start multiple writers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					newData := []CoinData{
						{ID: "token" + string(rune(id)), Symbol: "tkn", Name: "Token"},
					}
					updater.cache.Lock()
					updater.cache.data = &APIResponse{Data: newData}
					updater.cache.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Should not panic and should have some data
		finalData := updater.GetCacheData()
		assert.NotNil(t, finalData)
	})
}
