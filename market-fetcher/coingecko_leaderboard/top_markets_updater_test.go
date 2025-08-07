package coingecko_leaderboard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
	mock_interfaces "github.com/status-im/market-proxy/interfaces/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Helper function to create test config
func createTestMarketsConfig() *config.CoingeckoLeaderboardFetcher {
	return &config.CoingeckoLeaderboardFetcher{
		TopMarketsLimit:          10,
		TopMarketsUpdateInterval: time.Second * 5,
		Currency:                 "usd",
	}
}

// Helper function to create sample markets data
func createSampleMarketsData() interfaces.MarketsResponse {
	return interfaces.MarketsResponse([]interface{}{
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

func TestNewTopMarketsUpdater(t *testing.T) {
	t.Run("Creates new top markets updater with correct dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)

		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Equal(t, mockFetcher, updater.marketsFetcher)
		assert.Nil(t, updater.updateSubscription)
		assert.Nil(t, updater.updateSubscription)
		assert.Nil(t, updater.onUpdate)
		assert.Nil(t, updater.GetCacheData())
	})

	t.Run("Works with nil fetcher", func(t *testing.T) {
		cfg := createTestMarketsConfig()

		updater := NewTopMarketsUpdater(cfg, nil)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Nil(t, updater.marketsFetcher)
	})
}

func TestTopMarketsUpdater_SetOnUpdateCallback(t *testing.T) {
	t.Run("Sets callback function", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

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
		updater := NewTopMarketsUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

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
		updater := NewTopMarketsUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		// Set a callback first
		updater.SetOnUpdateCallback(func() {})
		assert.NotNil(t, updater.onUpdate)

		// Set to nil
		updater.SetOnUpdateCallback(nil)
		assert.Nil(t, updater.onUpdate)
	})
}

func TestTopMarketsUpdater_fetchAndUpdate(t *testing.T) {
	t.Run("Successful fetch and update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil)

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
	})

	t.Run("Uses default limit when config limit is 0", func(t *testing.T) {
		cfg := &config.CoingeckoLeaderboardFetcher{
			TopMarketsLimit: 0, // Should use default 500
			Currency:        "usd",
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(500, "usd").Return(sampleData, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)

	})

	t.Run("Uses default limit when config limit is negative", func(t *testing.T) {
		cfg := &config.CoingeckoLeaderboardFetcher{
			TopMarketsLimit: -10, // Should use default 500
			Currency:        "usd",
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(500, "usd").Return(sampleData, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)

	})

	t.Run("Handles fetcher error", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		expectedError := errors.New("API error")
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(interfaces.MarketsResponse(nil), expectedError)

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

	})

	t.Run("Handles empty response", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		emptyData := interfaces.MarketsResponse([]interface{}{})
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(emptyData, nil)

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

	})

	t.Run("Doesn't call callback when callback is nil", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil)

		// Don't set callback (should be nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		// Should not panic even without callback

	})
}

func TestTopMarketsUpdater_Healthy(t *testing.T) {
	t.Run("Returns true when cache has data", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewTopMarketsUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

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
		updater := NewTopMarketsUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil
	})

	t.Run("Returns true when cache data is empty but fetcher exists", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewTopMarketsUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: []CoinData{}}
		updater.cache.Unlock()

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil, even if cache is empty
	})

	t.Run("Returns true when fetcher is available but no cache", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil
	})

	t.Run("Returns false when fetcher is nil and no cache", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewTopMarketsUpdater(cfg, nil)

		result := updater.Healthy()

		assert.False(t, result)
	})
}

func TestTopMarketsUpdater_StartStop(t *testing.T) {
	t.Run("Start subscribes to market updates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		// Setup mock for subscription and initial fetch
		mockSubscription := events.NewSubscriptionManager().Subscribe()
		sampleData := createSampleMarketsData()

		mockFetcher.EXPECT().SubscribeTopMarketsUpdate().Return(mockSubscription).Times(1)
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil).AnyTimes() // Initial fetch

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.updateSubscription)

		// Stop to clean up

		updater.Stop()
	})

	t.Run("Stop unsubscribes and cancels goroutine", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		// Setup mock for subscription and initial fetch
		mockSubscription := events.NewSubscriptionManager().Subscribe()
		sampleData := createSampleMarketsData()

		mockFetcher.EXPECT().SubscribeTopMarketsUpdate().Return(mockSubscription).Times(1)
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil).AnyTimes()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start first
		err := updater.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, updater.updateSubscription)

		// Now stop
		updater.Stop()

		// Should be cleaned up
		assert.Nil(t, updater.updateSubscription)
	})

	t.Run("Stop doesn't panic when not started", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewTopMarketsUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		// Call stop without starting
		assert.NotPanics(t, func() {
			updater.Stop()
		})
	})

	t.Run("Subscription handler responds to market updates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestMarketsConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
		updater := NewTopMarketsUpdater(cfg, mockFetcher)

		mockSubscription := events.NewSubscriptionManager().Subscribe()
		sampleData := createSampleMarketsData()

		mockFetcher.EXPECT().SubscribeTopMarketsUpdate().Return(mockSubscription).Times(1)
		// Expect initial fetch (triggered by Watch with startNow: true)
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil).AnyTimes()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		err := updater.Start(ctx)
		assert.NoError(t, err)

		// Give some time for the initial processing
		time.Sleep(time.Millisecond * 100)

		// Stop to clean up
		updater.Stop()
	})
}

func TestTopMarketsUpdater_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent cache access is safe", func(t *testing.T) {
		cfg := createTestMarketsConfig()
		updater := NewTopMarketsUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

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
					_ = data
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
