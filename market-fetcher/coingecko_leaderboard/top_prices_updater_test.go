package coingecko_leaderboard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/status-im/market-proxy/interfaces"
	mock_interfaces "github.com/status-im/market-proxy/interfaces/mocks"

	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Helper function to create test config for prices updater
func createTestPricesConfig() *config.Config {
	return &config.Config{
		CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
			TopPricesUpdateInterval: time.Second * 5,
			TopPricesLimit:          10,
			Currency:                "usd",
		},
	}
}

// Helper function to create sample price response
func createSamplePriceResponse() interfaces.SimplePriceResponse {
	return interfaces.SimplePriceResponse{
		"bitcoin": map[string]interface{}{
			"usd":            50000.0,
			"usd_market_cap": 950000000000.0,
			"usd_24h_vol":    25000000000.0,
			"usd_24h_change": 2.5,
		},
		"ethereum": map[string]interface{}{
			"usd":            3000.0,
			"usd_market_cap": 360000000000.0,
			"usd_24h_vol":    15000000000.0,
			"usd_24h_change": -1.2,
		},
	}
}

func TestNewTopPricesUpdater(t *testing.T) {
	t.Run("Creates new top prices updater with correct dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(ctrl)

		updater := NewTopPricesUpdater(cfg, mockFetcher)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Equal(t, mockFetcher, updater.priceFetcher)
		assert.NotNil(t, updater.metricsWriter)
		assert.Nil(t, updater.priceScheduler)
		assert.NotNil(t, updater.topPricesCache.data)
		assert.Empty(t, updater.topPricesCache.data)
	})

	t.Run("Works with nil fetcher", func(t *testing.T) {
		cfg := createTestPricesConfig()

		updater := NewTopPricesUpdater(cfg, nil)

		assert.NotNil(t, updater)
		assert.Equal(t, cfg, updater.config)
		assert.Nil(t, updater.priceFetcher)
		assert.NotNil(t, updater.metricsWriter)
		assert.NotNil(t, updater.topPricesCache.data)
	})
}

func TestTopPricesUpdater_GetTopPricesQuotes(t *testing.T) {
	t.Run("Returns empty map when no data cached", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		result := updater.GetTopPricesQuotes("usd")

		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("Returns empty map when currency not found", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		// Add data for EUR but request USD
		sampleQuotes := PriceQuotes{
			"bitcoin": Quote{
				Price:            42000.0,
				MarketCap:        798000000000.0,
				Volume24h:        21000000000.0,
				PercentChange24h: 1.8,
			},
		}

		updater.topPricesCache.Lock()
		updater.topPricesCache.data["eur"] = sampleQuotes
		updater.topPricesCache.Unlock()

		result := updater.GetTopPricesQuotes("usd")

		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("Returns cached data for correct currency", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		sampleQuotes := PriceQuotes{
			"bitcoin": Quote{
				Price:            50000.0,
				MarketCap:        950000000000.0,
				Volume24h:        25000000000.0,
				PercentChange24h: 2.5,
			},
			"ethereum": Quote{
				Price:            3000.0,
				MarketCap:        360000000000.0,
				Volume24h:        15000000000.0,
				PercentChange24h: -1.2,
			},
		}

		updater.topPricesCache.Lock()
		updater.topPricesCache.data["usd"] = sampleQuotes
		updater.topPricesCache.Unlock()

		result := updater.GetTopPricesQuotes("usd")

		assert.Len(t, result, 2)
		assert.Equal(t, 50000.0, result["bitcoin"].Price)
		assert.Equal(t, 3000.0, result["ethereum"].Price)
		assert.Equal(t, 950000000000.0, result["bitcoin"].MarketCap)
		assert.Equal(t, 360000000000.0, result["ethereum"].MarketCap)
	})

	t.Run("Returns independent copy to avoid race conditions", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		sampleQuotes := PriceQuotes{
			"bitcoin": Quote{Price: 50000.0},
		}

		updater.topPricesCache.Lock()
		updater.topPricesCache.data["usd"] = sampleQuotes
		updater.topPricesCache.Unlock()

		result := updater.GetTopPricesQuotes("usd")

		// Modify the returned copy
		result["bitcoin"] = Quote{Price: 60000.0}

		// Original data should be unchanged
		originalResult := updater.GetTopPricesQuotes("usd")
		assert.Equal(t, 50000.0, originalResult["bitcoin"].Price)
		assert.NotEqual(t, 60000.0, originalResult["bitcoin"].Price)
	})

	t.Run("Works with multiple currencies", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		usdQuotes := PriceQuotes{
			"bitcoin": Quote{Price: 50000.0},
		}
		eurQuotes := PriceQuotes{
			"bitcoin": Quote{Price: 42000.0},
		}

		updater.topPricesCache.Lock()
		updater.topPricesCache.data["usd"] = usdQuotes
		updater.topPricesCache.data["eur"] = eurQuotes
		updater.topPricesCache.Unlock()

		usdResult := updater.GetTopPricesQuotes("usd")
		eurResult := updater.GetTopPricesQuotes("eur")

		assert.Equal(t, 50000.0, usdResult["bitcoin"].Price)
		assert.Equal(t, 42000.0, eurResult["bitcoin"].Price)
	})
}

func TestTopPricesUpdater_fetchAndUpdateTopPrices(t *testing.T) {
	t.Run("Successful fetch and update with price fetcher", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(ctrl)
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Set token IDs
		tokenIDs := []string{"bitcoin", "ethereum"}
		updater.SetTopTokenIDs(tokenIDs)

		// Setup mock response
		sampleResponse := createSamplePriceResponse()
		expectedParams := interfaces.PriceParams{
			IDs:                  tokenIDs,
			Currencies:           []string{"usd"},
			IncludeMarketCap:     true,
			Include24hrVol:       true,
			Include24hrChange:    true,
			IncludeLastUpdatedAt: true,
		}
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), expectedParams).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

		// Verify cache was updated
		result := updater.GetTopPricesQuotes("usd")
		assert.Len(t, result, 2)
		assert.Equal(t, 50000.0, result["bitcoin"].Price)
		assert.Equal(t, 3000.0, result["ethereum"].Price)
	})

	t.Run("Handles no token IDs configured", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Don't set any token IDs
		// mockFetcher should not be called

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

		// Cache should be empty
		result := updater.GetTopPricesQuotes("usd")
		assert.Empty(t, result)

		// Mock should not have been called (gomock will fail if any unexpected call is made)
	})

	t.Run("Handles token limit configuration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := &config.Config{
			CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
				TopPricesLimit: 2, // Limit to 2 tokens
				Currency:       "usd",
			},
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(ctrl)
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Set more token IDs than the limit
		tokenIDs := []string{"bitcoin", "ethereum", "cardano", "polkadot"}
		updater.SetTopTokenIDs(tokenIDs)

		// Only first 2 tokens should be fetched
		expectedParams := interfaces.PriceParams{
			IDs:                  []string{"bitcoin", "ethereum"}, // Limited to 2
			Currencies:           []string{"usd"},
			IncludeMarketCap:     true,
			Include24hrVol:       true,
			Include24hrChange:    true,
			IncludeLastUpdatedAt: true,
		}

		sampleResponse := createSamplePriceResponse()
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), expectedParams).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

	})

	t.Run("Handles price fetcher error", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		tokenIDs := []string{"bitcoin"}
		updater.SetTopTokenIDs(tokenIDs)

		// Mock error response
		expectedError := errors.New("API error")
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, expectedError)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		// Should not return error, just log it
		assert.NoError(t, err)

		// Cache should be empty due to error
		result := updater.GetTopPricesQuotes("usd")
		assert.Empty(t, result)

	})

	t.Run("Handles empty price response", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		tokenIDs := []string{"bitcoin"}
		updater.SetTopTokenIDs(tokenIDs)

		// Mock empty response
		emptyResponse := interfaces.SimplePriceResponse{}
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(emptyResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

		// Cache should be empty
		result := updater.GetTopPricesQuotes("usd")
		assert.Empty(t, result)

	})

	t.Run("Handles nil price fetcher", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, nil) // No price fetcher

		tokenIDs := []string{"bitcoin"}
		updater.SetTopTokenIDs(tokenIDs)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

		// Cache should be empty
		result := updater.GetTopPricesQuotes("usd")
		assert.Empty(t, result)
	})

	t.Run("Updates metrics", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		tokenIDs := []string{"bitcoin"}
		updater.SetTopTokenIDs(tokenIDs)

		sampleResponse := createSamplePriceResponse()
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)
		// Metrics should be recorded (we can't easily test the actual metrics values)

	})
}

func TestTopPricesUpdater_StartStop(t *testing.T) {
	t.Run("Start creates and starts scheduler when interval is set", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Setup mock for potential scheduler calls
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, nil).AnyTimes()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.priceScheduler)

		// Stop to clean up
		updater.Stop()
	})

	t.Run("Start does not create scheduler when interval is zero", func(t *testing.T) {
		cfg := &config.Config{
			CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
				TopPricesUpdateInterval: 0, // Zero interval
				Currency:                "usd",
			},
		}
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.Nil(t, updater.priceScheduler)
	})

	t.Run("Start does not create scheduler when interval is negative", func(t *testing.T) {
		cfg := &config.Config{
			CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
				TopPricesUpdateInterval: -time.Second, // Negative interval
				Currency:                "usd",
			},
		}
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.Nil(t, updater.priceScheduler)
	})

	t.Run("Stop stops scheduler when it exists", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Setup mock for potential scheduler calls
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, nil).AnyTimes()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start first
		err := updater.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, updater.priceScheduler)

		// Now stop
		updater.Stop()

		// Scheduler should still exist but be stopped
		assert.NotNil(t, updater.priceScheduler)
	})

	t.Run("Stop doesn't panic when scheduler is nil", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		// Call stop without starting
		assert.NotPanics(t, func() {
			updater.Stop()
		})
	})

	t.Run("Start with minimal update interval", func(t *testing.T) {
		cfg := &config.Config{
			CoingeckoLeaderboard: config.CoingeckoLeaderboardFetcher{
				TopPricesUpdateInterval: time.Millisecond, // Minimal interval
				Currency:                "usd",
			},
		}
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Setup mock for potential scheduler calls
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, nil).AnyTimes()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.priceScheduler)

		updater.Stop()
	})
}

func TestTopPricesUpdater_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent cache access is safe", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		var wg sync.WaitGroup
		numGoroutines := 10

		// Set initial data
		sampleQuotes := PriceQuotes{
			"bitcoin": Quote{Price: 50000.0},
		}
		updater.topPricesCache.Lock()
		updater.topPricesCache.data["usd"] = sampleQuotes
		updater.topPricesCache.Unlock()

		// Start multiple readers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					quotes := updater.GetTopPricesQuotes("usd")
					_ = quotes
				}
			}()
		}

		// Start multiple writers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					newQuotes := PriceQuotes{
						"bitcoin": Quote{Price: float64(50000 + id)},
					}
					updater.topPricesCache.Lock()
					updater.topPricesCache.data["usd"] = newQuotes
					updater.topPricesCache.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Should not panic and should have some data
		finalQuotes := updater.GetTopPricesQuotes("usd")
		assert.NotEmpty(t, finalQuotes)
	})

	t.Run("Concurrent token IDs access is safe", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		var wg sync.WaitGroup
		numGoroutines := 10

		// Start multiple readers and writers for token IDs
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					// Read
					tokenIDs := updater.getTopTokenIDs()
					_ = tokenIDs

					// Write
					newTokenIDs := []string{"bitcoin", "ethereum", "token" + string(rune(id))}
					updater.SetTopTokenIDs(newTokenIDs)
				}
			}(i)
		}

		wg.Wait()

		// Should not panic and should have some data
		finalTokenIDs := updater.getTopTokenIDs()
		assert.NotNil(t, finalTokenIDs)
	})
}

func TestTopPricesUpdater_Integration(t *testing.T) {
	t.Run("Full workflow: set tokens, fetch prices, get quotes", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// 1. Set token IDs
		tokenIDs := []string{"bitcoin", "ethereum"}
		updater.SetTopTokenIDs(tokenIDs)

		// 2. Setup mock for fetchAndUpdateTopPrices
		sampleResponse := createSamplePriceResponse()
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		// 3. Fetch and update prices
		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)
		assert.NoError(t, err)

		// 4. Get cached quotes
		quotes := updater.GetTopPricesQuotes("usd")
		assert.Len(t, quotes, 2)
		assert.Equal(t, 50000.0, quotes["bitcoin"].Price)
		assert.Equal(t, 3000.0, quotes["ethereum"].Price)

		// 5. Verify metrics are recorded

	})

	t.Run("Cache survives multiple updates", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		tokenIDs := []string{"bitcoin"}
		updater.SetTopTokenIDs(tokenIDs)

		// First update
		firstResponse := interfaces.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd": 50000.0,
			},
		}
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(firstResponse, interfaces.CacheStatusMiss, nil).Times(1)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)
		assert.NoError(t, err)

		firstQuotes := updater.GetTopPricesQuotes("usd")
		assert.Equal(t, 50000.0, firstQuotes["bitcoin"].Price)

		// Second update with different price
		secondResponse := interfaces.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd": 55000.0,
			},
		}
		mockFetcher.EXPECT().SimplePrices(gomock.Any(), gomock.Any()).Return(secondResponse, interfaces.CacheStatusMiss, nil).Times(1)

		err = updater.fetchAndUpdateTopPrices(ctx)
		assert.NoError(t, err)

		secondQuotes := updater.GetTopPricesQuotes("usd")
		assert.Equal(t, 55000.0, secondQuotes["bitcoin"].Price)

	})
}
