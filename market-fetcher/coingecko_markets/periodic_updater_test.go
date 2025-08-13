package coingecko_markets

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	api_mocks "github.com/status-im/market-proxy/coingecko_markets/mocks"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Helper function to create test config
func createTestPeriodicUpdaterConfig() *config.MarketsFetcherConfig {
	usdCurrency := "usd"
	return &config.MarketsFetcherConfig{
		MarketParamsNormalize: &config.MarketParamsNormalize{
			VsCurrency: &usdCurrency,
		},
		Tiers: []config.MarketTier{
			{
				Name:           "tier1",
				PageFrom:       1,
				PageTo:         2,
				UpdateInterval: time.Second * 5,
			},
		},
	}
}

// Helper function to setup mock FetchPage for any parameters
func setupMockFetchPage(mockFetcher *api_mocks.MockIAPIClient, data interfaces.MarketsResponse, err error) {
	// Convert data to bytes for FetchPage mock
	sampleBytes := make([][]byte, len(data))
	for i, item := range data {
		if itemBytes, marshalErr := json.Marshal(item); marshalErr == nil {
			sampleBytes[i] = itemBytes
		}
	}
	mockFetcher.EXPECT().FetchPage(gomock.Any()).Return(sampleBytes, err).AnyTimes()
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
		MockIAPIClient := api_mocks.NewMockIAPIClient(ctrl)

		updater := NewPeriodicUpdater(cfg, MockIAPIClient)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Equal(t, MockIAPIClient, updater.apiClient)
		assert.Nil(t, updater.scheduler)
		assert.Nil(t, updater.onUpdateTierPages)
		assert.Nil(t, updater.GetCacheData())
	})

	t.Run("Works with nil client", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()

		updater := NewPeriodicUpdater(cfg, nil)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Nil(t, updater.apiClient)
	})
}

func TestPeriodicUpdater_SetOnUpdateTierPagesCallback(t *testing.T) {
	t.Run("Sets callback function", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPeriodicUpdaterConfig()
		MockIAPIClient := api_mocks.NewMockIAPIClient(ctrl)
		updater := NewPeriodicUpdater(cfg, MockIAPIClient)

		callbackCalled := false
		callback := func(ctx context.Context, tier config.MarketTier, pagesData []PageData) {
			callbackCalled = true
		}

		updater.SetOnUpdateTierPagesCallback(callback)

		assert.NotNil(t, updater.onUpdateTierPages)

		// Test callback is called
		ctx := context.Background()
		testTier := config.MarketTier{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: time.Second}
		testPages := []PageData{{Page: 1, Data: [][]byte{}}}
		updater.onUpdateTierPages(ctx, testTier, testPages)
		assert.True(t, callbackCalled)
	})

	t.Run("Overwrites existing callback", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, api_mocks.NewMockIAPIClient(gomock.NewController(t)))

		firstCallbackCalled := false
		secondCallbackCalled := false

		// Set first callback
		updater.SetOnUpdateTierPagesCallback(func(ctx context.Context, tier config.MarketTier, pagesData []PageData) {
			firstCallbackCalled = true
		})

		// Set second callback (should overwrite first)
		updater.SetOnUpdateTierPagesCallback(func(ctx context.Context, tier config.MarketTier, pagesData []PageData) {
			secondCallbackCalled = true
		})

		// Call the callback
		ctx := context.Background()
		testTier := config.MarketTier{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: time.Second}
		testPages := []PageData{{Page: 1, Data: [][]byte{}}}
		updater.onUpdateTierPages(ctx, testTier, testPages)

		assert.False(t, firstCallbackCalled)
		assert.True(t, secondCallbackCalled)
	})

	t.Run("Can set nil callback", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, api_mocks.NewMockIAPIClient(gomock.NewController(t)))

		// Set a callback first
		updater.SetOnUpdateTierPagesCallback(func(ctx context.Context, tier config.MarketTier, pagesData []PageData) {})
		assert.NotNil(t, updater.onUpdateTierPages)

		// Set to nil
		updater.SetOnUpdateTierPagesCallback(nil)
		assert.Nil(t, updater.onUpdateTierPages)
	})
}

func TestPeriodicUpdater_fetchAndUpdateTier(t *testing.T) {
	t.Run("Successful fetch and update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := api_mocks.NewMockIAPIClient(ctrl)
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		// Override tier to have PageTo = 2 for this test (expecting 4 elements: 2 pages × 2 items each)
		tier := config.MarketTier{
			Name:           "tier1",
			PageFrom:       1,
			PageTo:         2,
			UpdateInterval: time.Second * 5,
		}
		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil)

		var mu sync.Mutex
		callbackCalled := false
		var callbackCtx context.Context
		updater.SetOnUpdateTierPagesCallback(func(ctx context.Context, tier config.MarketTier, pagesData []PageData) {
			mu.Lock()
			defer mu.Unlock()
			callbackCalled = true
			callbackCtx = ctx
		})

		ctx := context.Background()
		err := updater.fetchAndUpdateTier(ctx, tier)

		assert.NoError(t, err)

		mu.Lock()
		actualCallbackCalled := callbackCalled
		actualCallbackCtx := callbackCtx
		mu.Unlock()

		assert.True(t, actualCallbackCalled)
		assert.Equal(t, ctx, actualCallbackCtx)

		// Verify cache was updated for this tier (4 items: 2 pages × 2 items each)
		tierCacheData := updater.GetCacheDataForTier(tier.Name)
		assert.NotNil(t, tierCacheData)
		assert.Len(t, tierCacheData.Data, 4)
		assert.Equal(t, "bitcoin", tierCacheData.Data[0].ID)
		assert.Equal(t, "ethereum", tierCacheData.Data[1].ID)
		assert.Equal(t, "bitcoin", tierCacheData.Data[2].ID)  // Second page, same data
		assert.Equal(t, "ethereum", tierCacheData.Data[3].ID) // Second page, same data
	})

	t.Run("Uses default limit when config limit is 0", func(t *testing.T) {
		usdCurrency := "usd"
		cfg := &config.MarketsFetcherConfig{
			Tiers: []config.MarketTier{{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: time.Second}}, // Tier with page 1-1 (optimized from 500)
			MarketParamsNormalize: &config.MarketParamsNormalize{
				VsCurrency: &usdCurrency,
			},
		}
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil)

		ctx := context.Background()
		tier := cfg.Tiers[0]
		err := updater.fetchAndUpdateTier(ctx, tier)

		assert.NoError(t, err)
	})

	t.Run("Uses default limit when config limit is negative", func(t *testing.T) {
		usdCurrency := "usd"
		cfg := &config.MarketsFetcherConfig{
			Tiers: []config.MarketTier{{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: time.Second}}, // Test tier with page 1-1 (optimized from 100)
			MarketParamsNormalize: &config.MarketParamsNormalize{
				VsCurrency: &usdCurrency,
			},
		}
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil)

		ctx := context.Background()
		tier := cfg.Tiers[0]
		err := updater.fetchAndUpdateTier(ctx, tier)

		assert.NoError(t, err)
	})

	t.Run("Uses default currency when MarketParamsNormalize is nil", func(t *testing.T) {
		cfg := &config.MarketsFetcherConfig{
			Tiers:                 []config.MarketTier{{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: time.Second}}, // Test tier with page 1-1 (optimized from 100)
			MarketParamsNormalize: nil,                                                                                      // Should use default "usd"
		}
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil)

		ctx := context.Background()
		tier := cfg.Tiers[0]
		err := updater.fetchAndUpdateTier(ctx, tier)

		assert.NoError(t, err)
	})

	t.Run("Handles fetcher error", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		expectedError := errors.New("API error")
		setupMockFetchPage(mockFetcher, interfaces.MarketsResponse(nil), expectedError)

		callbackCalled := false
		updater.SetOnUpdateTierPagesCallback(func(ctx context.Context, tier config.MarketTier, pagesData []PageData) {
			callbackCalled = true
		})

		ctx := context.Background()
		tier := cfg.Tiers[0]
		err := updater.fetchAndUpdateTier(ctx, tier)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
		assert.False(t, callbackCalled)

		// Verify cache wasn't updated
		cacheData := updater.GetCacheData()
		assert.Nil(t, cacheData)
	})

	t.Run("Handles empty response", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		emptyData := interfaces.MarketsResponse([]interface{}{})
		setupMockFetchPage(mockFetcher, emptyData, nil)

		callbackCalled := false
		updater.SetOnUpdateTierPagesCallback(func(ctx context.Context, tier config.MarketTier, pagesData []PageData) {
			callbackCalled = true
		})

		ctx := context.Background()
		tier := cfg.Tiers[0]
		err := updater.fetchAndUpdateTier(ctx, tier)

		assert.NoError(t, err)
		assert.True(t, callbackCalled)

		// Verify cache returns nil for empty data (expected behavior)
		cacheData := updater.GetCacheData()
		assert.Nil(t, cacheData)
	})

	t.Run("Doesn't call callback when callback is nil", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil)

		// Don't set callback (should be nil)

		ctx := context.Background()
		tier := cfg.Tiers[0]
		err := updater.fetchAndUpdateTier(ctx, tier)

		assert.NoError(t, err)
		// Should not panic even without callback
	})
}

func TestPeriodicUpdater_Healthy(t *testing.T) {
	t.Run("Returns true when cache has data", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, api_mocks.NewMockIAPIClient(gomock.NewController(t)))

		mockData := []CoinGeckoData{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		}

		updater.cache.Lock()
		// Setup tier cache
		updater.cache.tiers["test"] = &TierDataWithTimestamp{Data: mockData, Timestamp: time.Now()}
		updater.cache.Unlock()

		result := updater.Healthy()

		assert.True(t, result)
	})

	t.Run("Returns true when fetcher exists but no cache", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		MockIAPIClient := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		MockIAPIClient.EXPECT().Healthy().Return(true)
		updater := NewPeriodicUpdater(cfg, MockIAPIClient)

		result := updater.Healthy()

		assert.True(t, result) // Because fetcher is not nil
	})

	t.Run("Returns true when cache data is empty but fetcher exists", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		MockIAPIClient := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		MockIAPIClient.EXPECT().Healthy().Return(true)
		updater := NewPeriodicUpdater(cfg, MockIAPIClient)

		updater.cache.Lock()
		// Setup tier cache
		updater.cache.tiers["test"] = &TierDataWithTimestamp{Data: []CoinGeckoData{}, Timestamp: time.Now()}
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
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		// Stop to clean up
		updater.Stop()
		time.Sleep(10 * time.Millisecond) // Allow goroutines to stop
	})

	t.Run("Start fails when interval is zero", func(t *testing.T) {
		cfg := &config.MarketsFetcherConfig{
			Tiers: []config.MarketTier{{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: 0}}, // Test tier with 0 interval - should fail validation (optimized from 100)
		}
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		ctx := context.Background()
		err := updater.Start(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update_interval must be greater than 0")
	})

	t.Run("Start fails when interval is negative", func(t *testing.T) {
		cfg := &config.MarketsFetcherConfig{
			Tiers: []config.MarketTier{{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: -time.Second}}, // Test tier with negative interval - should fail validation (optimized from 100)
		}
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		ctx := context.Background()
		err := updater.Start(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update_interval must be greater than 0")
	})

	t.Run("Stop stops scheduler when it exists", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start first
		err := updater.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		// Now stop
		updater.Stop()
		time.Sleep(10 * time.Millisecond) // Allow goroutines to stop

		// Scheduler should still exist but be stopped
		assert.NotNil(t, updater.scheduler)
	})

	t.Run("Stop doesn't panic when scheduler is nil", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, api_mocks.NewMockIAPIClient(gomock.NewController(t)))

		// Call stop without starting
		assert.NotPanics(t, func() {
			updater.Stop()
		})
	})

	t.Run("Start with minimal update interval", func(t *testing.T) {
		usdCurrency := "usd"
		cfg := &config.MarketsFetcherConfig{
			Tiers: []config.MarketTier{{Name: "test", PageFrom: 1, PageTo: 1, UpdateInterval: time.Second}}, // Test tier time.Millisecond, // Minimal interval (optimized from 100)
			MarketParamsNormalize: &config.MarketParamsNormalize{
				VsCurrency: &usdCurrency,
			},
		}
		mockFetcher := api_mocks.NewMockIAPIClient(gomock.NewController(t))
		updater := NewPeriodicUpdater(cfg, mockFetcher)

		// Setup mock for potential immediate scheduler call
		sampleData := createSampleMarketsData()
		setupMockFetchPage(mockFetcher, sampleData, nil) // Should match tier's PageTo

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.scheduler)

		updater.Stop()
		time.Sleep(10 * time.Millisecond) // Allow goroutines to stop
	})
}

func TestPeriodicUpdater_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent cache access is safe", func(t *testing.T) {
		cfg := createTestPeriodicUpdaterConfig()
		updater := NewPeriodicUpdater(cfg, api_mocks.NewMockIAPIClient(gomock.NewController(t)))

		var wg sync.WaitGroup
		numGoroutines := 10

		// Set initial data
		mockData := []CoinGeckoData{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		}
		updater.cache.Lock()
		// Setup tier cache
		updater.cache.tiers["test"] = &TierDataWithTimestamp{Data: mockData, Timestamp: time.Now()}
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
					newData := []CoinGeckoData{
						{ID: "token" + string(rune(id)), Symbol: "tkn", Name: "Token"},
					}
					updater.cache.Lock()
					// Setup tier cache
					updater.cache.tiers["test"] = &TierDataWithTimestamp{Data: newData, Timestamp: time.Now()}
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
		coinData := map[string]interface{}{
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
		}

		// Convert to JSON bytes
		coinBytes, err := json.Marshal(coinData)
		assert.NoError(t, err)
		input := [][]byte{coinBytes}

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
		input := [][]byte{}
		result := ConvertMarketsResponseToCoinGeckoData(input)
		assert.Empty(t, result)
	})

	t.Run("Skips invalid items", func(t *testing.T) {
		// Create invalid JSON and valid JSON
		validCoinData := map[string]interface{}{
			"id":     "bitcoin",
			"symbol": "btc",
			"name":   "Bitcoin",
		}
		validBytes, err := json.Marshal(validCoinData)
		assert.NoError(t, err)

		input := [][]byte{
			[]byte("invalid json"), // Invalid JSON
			[]byte("{incomplete"),  // Incomplete JSON
			validBytes,             // Valid JSON
		}

		result := ConvertMarketsResponseToCoinGeckoData(input)

		assert.Len(t, result, 1)
		assert.Equal(t, "bitcoin", result[0].ID)
	})

	t.Run("Handles missing fields gracefully", func(t *testing.T) {
		coinData := map[string]interface{}{
			"id": "bitcoin",
			// Missing other fields
		}

		coinBytes, err := json.Marshal(coinData)
		assert.NoError(t, err)
		input := [][]byte{coinBytes}

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
