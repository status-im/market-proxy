package coingecko_markets

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
	mock_interfaces "github.com/status-im/market-proxy/interfaces/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Helper function to create test config
func createTestPeriodicUpdaterConfig() *config.CoingeckoMarketsFetcher {
	return &config.CoingeckoMarketsFetcher{
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
			"market_cap_rank":             1,
			"total_volume":                25000000000.0,
			"price_change_percentage_24h": 2.5,
			"ath":                         69000.0,
			"ath_date":                    "2021-11-10T14:24:11.849Z",
			"atl":                         67.81,
			"atl_date":                    "2013-07-06T00:00:00.000Z",
			"circulating_supply":          19500000.0,
			"last_updated":                "2023-01-01T00:00:00.000Z",
		},
		map[string]interface{}{
			"id":                          "ethereum",
			"symbol":                      "eth",
			"name":                        "Ethereum",
			"image":                       "https://coin-images.coingecko.com/coins/images/279/large/ethereum.png",
			"current_price":               3000.0,
			"market_cap":                  360000000000.0,
			"market_cap_rank":             2,
			"total_volume":                15000000000.0,
			"price_change_percentage_24h": -1.2,
			"ath":                         4878.26,
			"ath_date":                    "2021-11-10T14:24:19.604Z",
			"atl":                         0.432979,
			"atl_date":                    "2015-10-20T00:00:00.000Z",
			"circulating_supply":          120000000.0,
			"last_updated":                "2023-01-01T00:00:00.000Z",
		},
	})
}

func TestNewPeriodicUpdater(t *testing.T) {
	t.Run("Creates new periodic updater with correct dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)

		updater := NewPeriodicUpdater(cfg, mockFetcher)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Equal(t, mockFetcher, updater.marketsFetcher)
		assert.Nil(t, updater.scheduler)
		assert.Nil(t, updater.onUpdate)
		assert.Nil(t, updater.GetCacheData())
	})

	t.Run("Works with nil fetcher", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()

		updater := NewPeriodicUpdater(cfg, nil)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Nil(t, updater.marketsFetcher)
	})
}

func TestPeriodicUpdater_SetOnUpdateCallback(t *testing.T) {
	t.Run("Sets callback function", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		callbackCalled := false
		callback := func(ctx context.Context) {
			callbackCalled = true
		}

		updater.SetOnUpdateCallback(callback)

		assert.NotNil(t, updater.onUpdate)

		// Test callback is called
		ctx := context.Background()
		updater.onUpdate(ctx)
		assert.True(t, callbackCalled)
	})

	t.Run("Overwrites existing callback", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		firstCallbackCalled := false
		secondCallbackCalled := false

		// Set first callback
		updater.SetOnUpdateCallback(func(ctx context.Context) {
			firstCallbackCalled = true
		})

		// Set second callback (should overwrite first)
		updater.SetOnUpdateCallback(func(ctx context.Context) {
			secondCallbackCalled = true
		})

		// Call the callback
		ctx := context.Background()
		updater.onUpdate(ctx)

		assert.False(t, firstCallbackCalled)
		assert.True(t, secondCallbackCalled)
	})

	t.Run("Can set nil callback", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		// Set a callback first
		updater.SetOnUpdateCallback(func(ctx context.Context) {})
		assert.NotNil(t, updater.onUpdate)

		// Set to nil
		updater.SetOnUpdateCallback(nil)
		assert.Nil(t, updater.onUpdate)
	})
}

func TestPeriodicUpdater_GetTopTokenIDs(t *testing.T) {
	t.Run("Returns nil when no cache data", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		result := updater.GetTopTokenIDs()

		assert.Nil(t, result)
	})

	t.Run("Returns nil when cache data is nil", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		updater.cache.Lock()
		updater.cache.data = nil
		updater.cache.Unlock()

		result := updater.GetTopTokenIDs()

		assert.Nil(t, result)
	})

	t.Run("Returns nil when cache data.Data is nil", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: nil}
		updater.cache.Unlock()

		result := updater.GetTopTokenIDs()

		assert.Nil(t, result)
	})

	t.Run("Returns empty slice when no coins have IDs", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		mockData := []CoinGeckoData{
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
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		mockData := []CoinGeckoData{
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
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		mockData := []CoinGeckoData{
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

func TestPeriodicUpdater_fetchAndUpdate(t *testing.T) {
	t.Run("Successful fetch and update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(ctrl)
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil)

		callbackCalled := false
		var callbackCtx context.Context
		updater.SetOnUpdateCallback(func(ctx context.Context) {
			callbackCalled = true
			callbackCtx = ctx
		})

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		assert.True(t, callbackCalled)
		assert.Equal(t, ctx, callbackCtx)

		// Verify cache was updated
		cacheData := updater.GetCacheData()
		assert.NotNil(t, cacheData)
		assert.Len(t, cacheData.Data, 2)
		assert.Equal(t, "bitcoin", cacheData.Data[0].ID)
		assert.Equal(t, "ethereum", cacheData.Data[1].ID)
	})

	t.Run("Uses default limit when config limit is 0", func(t *testing.T) {
		cfg := &config.CoingeckoMarketsFetcher{
			TopMarketsLimit: 0, // Should use default 500
			Currency:        "usd",
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(500, "usd").Return(sampleData, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
	})

	t.Run("Uses default limit when config limit is negative", func(t *testing.T) {
		cfg := &config.CoingeckoMarketsFetcher{
			TopMarketsLimit: -10, // Should use default 500
			Currency:        "usd",
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(500, "usd").Return(sampleData, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
	})

	t.Run("Uses default currency when config currency is empty", func(t *testing.T) {
		cfg := &config.CoingeckoMarketsFetcher{
			TopMarketsLimit: 10,
			Currency:        "", // Should use default "usd"
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
	})

	t.Run("Handles fetcher error", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		expectedError := errors.New("API error")
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(interfaces.MarketsResponse(nil), expectedError)

		callbackCalled := false
		updater.SetOnUpdateCallback(func(ctx context.Context) {
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
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		emptyData := interfaces.MarketsResponse([]interface{}{})
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(emptyData, nil)

		callbackCalled := false
		updater.SetOnUpdateCallback(func(ctx context.Context) {
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
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil)

		// Don't set callback (should be nil)

		ctx := context.Background()
		err := updater.fetchAndUpdate(ctx)

		assert.NoError(t, err)
		// Should not panic even without callback
	})
}

func TestPeriodicUpdater_Healthy(t *testing.T) {
	t.Run("Returns true when cache has data", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		mockData := []CoinGeckoData{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		}

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: mockData}
		updater.cache.Unlock()

		result := updater.Healthy()

		assert.True(t, result)
	})

	t.Run("Returns true when fetcher exists but no cache", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil
	})

	t.Run("Returns true when cache data is empty but fetcher exists", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		updater.cache.Lock()
		updater.cache.data = &APIResponse{Data: []CoinGeckoData{}}
		updater.cache.Unlock()

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil, even if cache is empty
	})

	t.Run("Returns false when fetcher is nil and no cache", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, nil)

		result := updater.Healthy()

		assert.False(t, result)
	})
}

func TestPeriodicUpdater_StartStop(t *testing.T) {
	t.Run("Start creates and starts scheduler", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil).AnyTimes()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		// Stop to clean up
		updater.Stop()
	})

	t.Run("Start skips periodic updates when interval is zero", func(t *testing.T) {
		cfg := &config.CoingeckoMarketsFetcher{
			TopMarketsUpdateInterval: 0, // Should skip periodic updates
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		ctx := context.Background()
		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.Nil(t, updater.scheduler) // Should not create scheduler
	})

	t.Run("Start skips periodic updates when interval is negative", func(t *testing.T) {
		cfg := &config.CoingeckoMarketsFetcher{
			TopMarketsUpdateInterval: -time.Second, // Should skip periodic updates
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		ctx := context.Background()
		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.Nil(t, updater.scheduler) // Should not create scheduler
	})

	t.Run("Stop stops scheduler when it exists", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(10, "usd").Return(sampleData, nil).AnyTimes()

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
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		// Call stop without starting
		assert.NotPanics(t, func() {
			updater.Stop()
		})
	})

	t.Run("Start with minimal update interval", func(t *testing.T) {
		cfg := &config.CoingeckoMarketsFetcher{
			TopMarketsUpdateInterval: time.Millisecond, // Minimal interval
			Currency:                 "usd",
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		mockFetcher.EXPECT().TopMarkets(500, "usd").Return(sampleData, nil).AnyTimes() // Default limit is 500

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		updater.Stop()
	})
}

func TestPeriodicUpdater_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent cache access is safe", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, mock_interfaces.NewMockCoingeckoMarketsService(gomock.NewController(t)))

		var wg sync.WaitGroup
		numGoroutines := 10

		// Set initial data
		mockData := []CoinGeckoData{
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
					newData := []CoinGeckoData{
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

func TestConvertMarketsResponseToCoinGeckoData(t *testing.T) {
	t.Run("Converts valid market data", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"id":                          "bitcoin",
				"symbol":                      "btc",
				"name":                        "Bitcoin",
				"image":                       "https://example.com/bitcoin.png",
				"current_price":               50000.0,
				"market_cap":                  950000000000.0,
				"market_cap_rank":             1.0, // JSON numbers are floats
				"total_volume":                25000000000.0,
				"price_change_percentage_24h": 2.5,
				"ath":                         69000.0,
				"ath_date":                    "2021-11-10T14:24:11.849Z",
				"atl":                         67.81,
				"atl_date":                    "2013-07-06T00:00:00.000Z",
				"circulating_supply":          19500000.0,
				"last_updated":                "2023-01-01T00:00:00.000Z",
				"roi":                         nil,
			},
		}

		result := ConvertMarketsResponseToCoinGeckoData(input)

		assert.Len(t, result, 1)
		coin := result[0]
		assert.Equal(t, "bitcoin", coin.ID)
		assert.Equal(t, "btc", coin.Symbol)
		assert.Equal(t, "Bitcoin", coin.Name)
		assert.Equal(t, "https://example.com/bitcoin.png", coin.Image)
		assert.Equal(t, 50000.0, coin.CurrentPrice)
		assert.Equal(t, 950000000000.0, coin.MarketCap)
		assert.Equal(t, 1, coin.MarketCapRank)
		assert.Equal(t, 25000000000.0, coin.TotalVolume)
		assert.Equal(t, 2.5, coin.PriceChangePercentage24h)
		assert.Equal(t, 69000.0, coin.ATH)
		assert.Equal(t, "2021-11-10T14:24:11.849Z", coin.ATHDate)
		assert.Equal(t, 67.81, coin.ATL)
		assert.Equal(t, "2013-07-06T00:00:00.000Z", coin.ATLDate)
		assert.Equal(t, 19500000.0, coin.CirculatingSupply)
		assert.Equal(t, "2023-01-01T00:00:00.000Z", coin.LastUpdated)
		assert.Nil(t, coin.ROI)
	})

	t.Run("Handles empty input", func(t *testing.T) {
		input := []interface{}{}
		result := ConvertMarketsResponseToCoinGeckoData(input)
		assert.Empty(t, result)
	})

	t.Run("Skips invalid items", func(t *testing.T) {
		input := []interface{}{
			"invalid",
			123,
			map[string]interface{}{
				"id":     "bitcoin",
				"symbol": "btc",
				"name":   "Bitcoin",
			},
		}

		result := ConvertMarketsResponseToCoinGeckoData(input)

		assert.Len(t, result, 1)
		assert.Equal(t, "bitcoin", result[0].ID)
	})

	t.Run("Handles missing fields gracefully", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"id": "bitcoin",
				// Missing other fields
			},
		}

		result := ConvertMarketsResponseToCoinGeckoData(input)

		assert.Len(t, result, 1)
		coin := result[0]
		assert.Equal(t, "bitcoin", coin.ID)
		assert.Equal(t, "", coin.Symbol)
		assert.Equal(t, "", coin.Name)
		assert.Equal(t, 0.0, coin.CurrentPrice)
		assert.Equal(t, 0, coin.MarketCapRank)
	})
}
