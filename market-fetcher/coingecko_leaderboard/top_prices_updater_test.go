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
		assert.Nil(t, updater.updateSubscription)
		assert.Nil(t, updater.cancelFunc)
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

		// Setup mock response - TopPrices returns top tokens directly
		sampleResponse := createSamplePriceResponse()

		// TopPrices method is called with limit and currencies
		limit := cfg.CoingeckoLeaderboard.TopPricesLimit
		currencies := []string{"usd"}
		mockFetcher.EXPECT().TopPrices(gomock.Any(), limit, currencies).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

		// Verify cache was updated
		result := updater.GetTopPricesQuotes("usd")
		assert.Len(t, result, 2)
		assert.Equal(t, 50000.0, result["bitcoin"].Price)
		assert.Equal(t, 3000.0, result["ethereum"].Price)
	})

	t.Run("Handles empty price response", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Mock empty price response
		emptyResponse := interfaces.SimplePriceResponse{}
		limit := cfg.CoingeckoLeaderboard.TopPricesLimit
		currencies := []string{"usd"}
		mockFetcher.EXPECT().TopPrices(gomock.Any(), limit, currencies).Return(emptyResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

		// Cache should be empty due to empty response
		result := updater.GetTopPricesQuotes("usd")
		assert.Empty(t, result)
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

		// TopPrices should be called with the configured limit
		limit := 2
		currencies := []string{"usd"}
		sampleResponse := createSamplePriceResponse()
		mockFetcher.EXPECT().TopPrices(gomock.Any(), limit, currencies).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)

	})

	t.Run("Handles price fetcher error", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// Mock error response
		expectedError := errors.New("API error")
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, expectedError)

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

		// Mock empty response
		emptyResponse := interfaces.SimplePriceResponse{}
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(emptyResponse, interfaces.CacheStatusMiss, nil)

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

		sampleResponse := createSamplePriceResponse()
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)

		assert.NoError(t, err)
		// Metrics should be recorded (we can't easily test the actual metrics values)

	})
}

func TestTopPricesUpdater_StartStop(t *testing.T) {
	t.Run("Start subscribes to price updates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(ctrl)
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// TopPrices will be called for top tokens

		// Setup mock for subscription and initial fetch
		updateCh := make(chan struct{}, 1)
		mockFetcher.EXPECT().SubscribeTopPricesUpdate().Return(updateCh).Times(1)
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, nil).Times(1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, updater.updateSubscription)
		assert.NotNil(t, updater.cancelFunc)

		// Stop to clean up
		mockFetcher.EXPECT().Unsubscribe(updateCh).Times(1)
		updater.Stop()
	})

	t.Run("Start handles nil fetcher gracefully", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := updater.Start(ctx)

		assert.NoError(t, err)
		assert.Nil(t, updater.updateSubscription)
		assert.Nil(t, updater.cancelFunc)
	})

	t.Run("Stop unsubscribes and cancels goroutine", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(ctrl)
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// TopPrices will be called for top tokens

		// Setup mock for subscription and initial fetch
		updateCh := make(chan struct{}, 1)
		mockFetcher.EXPECT().SubscribeTopPricesUpdate().Return(updateCh).Times(1)
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, nil).Times(1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start first
		err := updater.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, updater.updateSubscription)

		// Now stop
		mockFetcher.EXPECT().Unsubscribe(updateCh).Times(1)
		updater.Stop()

		// Should be cleaned up
		assert.Nil(t, updater.updateSubscription)
		assert.Nil(t, updater.cancelFunc)
	})

	t.Run("Stop doesn't panic when not started", func(t *testing.T) {
		cfg := createTestPricesConfig()
		updater := NewTopPricesUpdater(cfg, mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t)))

		// Call stop without starting
		assert.NotPanics(t, func() {
			updater.Stop()
		})
	})

	t.Run("Subscription handler responds to price updates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(ctrl)
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// TopPrices will be called for top tokens

		updateCh := make(chan struct{}, 2)
		mockFetcher.EXPECT().SubscribeTopPricesUpdate().Return(updateCh).Times(1)
		// Expect initial fetch + one more when we send update signal
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.SimplePriceResponse{}, interfaces.CacheStatusMiss, nil).Times(2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		err := updater.Start(ctx)
		assert.NoError(t, err)

		// Send update signal
		updateCh <- struct{}{}

		// Give some time for the goroutine to process the signal
		time.Sleep(time.Millisecond * 100)

		// Stop to clean up
		mockFetcher.EXPECT().Unsubscribe(updateCh).Times(1)
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

}

func TestTopPricesUpdater_Integration(t *testing.T) {
	t.Run("Full workflow: fetch top prices, get quotes", func(t *testing.T) {
		cfg := createTestPricesConfig()
		mockFetcher := mock_interfaces.NewMockCoingeckoPricesService(gomock.NewController(t))
		updater := NewTopPricesUpdater(cfg, mockFetcher)

		// 1. Setup mock for fetchAndUpdateTopPrices using TopPrices
		sampleResponse := createSamplePriceResponse()
		limit := cfg.CoingeckoLeaderboard.TopPricesLimit
		currencies := []string{"usd"}
		mockFetcher.EXPECT().TopPrices(gomock.Any(), limit, currencies).Return(sampleResponse, interfaces.CacheStatusMiss, nil)

		// 2. Fetch and update prices
		ctx := context.Background()
		err := updater.fetchAndUpdateTopPrices(ctx)
		assert.NoError(t, err)

		// 3. Get cached quotes
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

		// First update
		firstResponse := interfaces.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd": 50000.0,
			},
		}
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(firstResponse, interfaces.CacheStatusMiss, nil).Times(1)

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
		mockFetcher.EXPECT().TopPrices(gomock.Any(), gomock.Any(), gomock.Any()).Return(secondResponse, interfaces.CacheStatusMiss, nil).Times(1)

		err = updater.fetchAndUpdateTopPrices(ctx)
		assert.NoError(t, err)

		secondQuotes := updater.GetTopPricesQuotes("usd")
		assert.Equal(t, 55000.0, secondQuotes["bitcoin"].Price)

	})
}
