package coingecko_markets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/market-proxy/interfaces"
	interface_mocks "github.com/status-im/market-proxy/interfaces/mocks"

	"github.com/status-im/market-proxy/cache"
	cache_mocks "github.com/status-im/market-proxy/cache/mocks"
	api_mocks "github.com/status-im/market-proxy/coingecko_markets/mocks"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Test data constants
var (
	sampleMarketData1 = []byte(`{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":45000,"market_cap":850000000000}`)
	sampleMarketData2 = []byte(`{"id":"ethereum","symbol":"eth","name":"Ethereum","current_price":3000,"market_cap":360000000000}`)
	invalidMarketData = []byte(`{"symbol":"btc","name":"Bitcoin"}`)                // Missing id field
	malformedData     = []byte(`{"id":"bitcoin","symbol":"btc","current_price":}`) // Invalid JSON
)

func createTestConfig() *config.Config {
	return &config.Config{
		CoingeckoMarkets: config.CoingeckoMarketsFetcher{
			RequestDelay: 1000 * time.Millisecond,
			TTL:          300 * time.Second,
			Tiers: []config.MarketTier{
				{
					Name:           "test-tier",
					PageFrom:       1,
					PageTo:         2,
					UpdateInterval: 5 * time.Second,
				},
			},
		},
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
	}
}

func createMockSubscription() events.SubscriptionInterface {
	// Create a subscription that can be used in tests
	mgr := events.NewSubscriptionManager()
	return mgr.Subscribe()
}

func createMockTokensService(ctrl *gomock.Controller) *interface_mocks.MockCoingeckoTokensService {
	mockTokensService := interface_mocks.NewMockCoingeckoTokensService(ctrl)
	mockTokensService.EXPECT().GetTokens().Return([]interfaces.Token{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}).AnyTimes()
	mockTokensService.EXPECT().GetTokenIds().Return([]string{
		"bitcoin", "ethereum",
	}).AnyTimes()
	mockTokensService.EXPECT().SubscribeOnTokensUpdate().Return(createMockSubscription()).AnyTimes()
	return mockTokensService
}

func TestNewService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTokensService := createMockTokensService(ctrl)

	tests := []struct {
		name   string
		cache  cache.Cache
		config *config.Config
	}{
		{
			name:   "Valid service creation",
			cache:  cache_mocks.NewMockCache(ctrl),
			config: createTestConfig(),
		},
		{
			name:   "Service with nil cache",
			cache:  nil,
			config: createTestConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.cache, tt.config, mockTokensService)
			assert.NotNil(t, service)
			assert.Equal(t, tt.cache, service.cache)
			assert.Equal(t, tt.config, service.config)
			assert.NotNil(t, service.metricsWriter)
			assert.NotNil(t, service.periodicUpdater)
			assert.Equal(t, mockTokensService, service.tokensService)
		})
	}
}

func TestService_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		cache         cache.Cache
		tokensService interfaces.CoingeckoTokensService
		expectError   bool
		errorMsg      string
		expectCancel  bool
	}{
		{
			name:          "Start with valid cache and tokens service",
			cache:         cache_mocks.NewMockCache(ctrl),
			tokensService: createMockTokensService(ctrl),
			expectError:   false,
			expectCancel:  true,
		},
		{
			name:          "Start with valid cache but no tokens service",
			cache:         cache_mocks.NewMockCache(ctrl),
			tokensService: nil,
			expectError:   false,
			expectCancel:  false,
		},
		{
			name:          "Start with nil cache",
			cache:         nil,
			tokensService: createMockTokensService(ctrl),
			expectError:   true,
			errorMsg:      "cache dependency not provided",
			expectCancel:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &Service{
				cache:         tt.cache,
				config:        createTestConfig(),
				tokensService: tt.tokensService,
			}

			err := service.Start(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if tt.expectCancel {
					assert.NotNil(t, service.tokenUpdateSubscription, "tokenUpdateSubscription should be set when tokens service is provided")
				} else {
					assert.Nil(t, service.tokenUpdateSubscription, "tokenUpdateSubscription should be nil when tokens service is not provided")
				}
			}
		})
	}
}

func TestService_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		setupService func() *Service
		expectCancel bool
	}{
		{
			name: "Stop service with tokens service and active goroutine",
			setupService: func() *Service {
				mockCache := cache_mocks.NewMockCache(ctrl)
				mockTokensService := createMockTokensService(ctrl)
				service := NewService(mockCache, createTestConfig(), mockTokensService)

				// Start the service to initialize the goroutine and cancelFunc
				err := service.Start(context.Background())
				assert.NoError(t, err)

				return service
			},
			expectCancel: true,
		},
		{
			name: "Stop service without tokens service",
			setupService: func() *Service {
				mockCache := cache_mocks.NewMockCache(ctrl)
				service := NewService(mockCache, createTestConfig(), nil)

				// Start the service
				err := service.Start(context.Background())
				assert.NoError(t, err)

				return service
			},
			expectCancel: false,
		},
		{
			name: "Stop service that was never started",
			setupService: func() *Service {
				mockCache := cache_mocks.NewMockCache(ctrl)
				mockTokensService := createMockTokensService(ctrl)
				return NewService(mockCache, createTestConfig(), mockTokensService)
			},
			expectCancel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := tt.setupService()

			// Verify initial state
			if tt.expectCancel {
				assert.NotNil(t, service.tokenUpdateSubscription, "tokenUpdateSubscription should be set before Stop")
			}

			// Should not panic
			assert.NotPanics(t, func() {
				service.Stop()
			})

			// Verify subscription is cleared after Stop
			assert.Nil(t, service.tokenUpdateSubscription, "tokenUpdateSubscription should be nil after Stop")
		})
	}
}

func TestService_onTokenListChanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		tokensService  interfaces.CoingeckoTokensService
		expectedTokens []string
		expectCall     bool
	}{
		{
			name: "Update with valid tokens service",
			tokensService: func() interfaces.CoingeckoTokensService {
				mockTokensService := interface_mocks.NewMockCoingeckoTokensService(ctrl)
				mockTokensService.EXPECT().GetTokenIds().Return([]string{
					"bitcoin", "ethereum", "cardano",
				}).Times(1)
				return mockTokensService
			}(),
			expectedTokens: []string{"bitcoin", "ethereum", "cardano"},
			expectCall:     true,
		},
		{
			name:           "Update with nil tokens service",
			tokensService:  nil,
			expectedTokens: nil,
			expectCall:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := cache_mocks.NewMockCache(ctrl)
			service := NewService(mockCache, createTestConfig(), tt.tokensService)

			// Create a mock periodic updater to verify SetExtraIds is called
			if tt.expectCall {
				mockPeriodicUpdater := &PeriodicUpdater{}
				service.periodicUpdater = mockPeriodicUpdater

				// Call onTokenListChanged
				service.onTokenListChanged()

				// We can't easily mock the PeriodicUpdater, but we can ensure no panic occurs
				// In a real implementation, you might want to make PeriodicUpdater an interface
			} else {
				// Should not panic with nil tokens service
				assert.NotPanics(t, func() {
					service.onTokenListChanged()
				})
			}
		})
	}
}

func TestService_parseTokensData(t *testing.T) {

	tests := []struct {
		name              string
		tokensData        [][]byte
		expectedLen       int
		expectedError     bool
		expectedCacheKeys []string
	}{
		{
			name:              "Valid tokens data",
			tokensData:        [][]byte{sampleMarketData1, sampleMarketData2},
			expectedLen:       2,
			expectedError:     false,
			expectedCacheKeys: []string{"markets:bitcoin", "markets:ethereum"},
		},
		{
			name:              "Mixed valid and invalid data",
			tokensData:        [][]byte{sampleMarketData1, invalidMarketData, sampleMarketData2},
			expectedLen:       2,
			expectedError:     false,
			expectedCacheKeys: []string{"markets:bitcoin", "markets:ethereum"},
		},
		{
			name:              "Empty tokens data",
			tokensData:        [][]byte{},
			expectedLen:       0,
			expectedError:     false,
			expectedCacheKeys: []string{},
		},
		{
			name:              "Malformed JSON data",
			tokensData:        [][]byte{malformedData},
			expectedLen:       0,
			expectedError:     false,
			expectedCacheKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marketData, cacheData, err := parseTokensData(tt.tokensData)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, marketData, tt.expectedLen)
				assert.Len(t, cacheData, len(tt.expectedCacheKeys))

				// Check cache keys
				for _, expectedKey := range tt.expectedCacheKeys {
					assert.Contains(t, cacheData, expectedKey)
				}
			}
		})
	}
}

func TestService_cacheTokensByID(t *testing.T) {
	tests := []struct {
		name          string
		tokensData    [][]byte
		cacheSetError error
		expectedError bool
		expectedLen   int
	}{
		{
			name:          "Successful caching",
			tokensData:    [][]byte{sampleMarketData1, sampleMarketData2},
			cacheSetError: nil,
			expectedError: false,
			expectedLen:   2,
		},
		{
			name:          "Cache set error",
			tokensData:    [][]byte{sampleMarketData1},
			cacheSetError: errors.New("cache error"),
			expectedError: true,
			expectedLen:   0,
		},
		{
			name:          "Empty tokens data",
			tokensData:    [][]byte{},
			cacheSetError: nil,
			expectedError: false,
			expectedLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCache := cache_mocks.NewMockCache(ctrl)
			mockTokensService := createMockTokensService(ctrl)
			service := NewService(mockCache, createTestConfig(), mockTokensService)

			if len(tt.tokensData) > 0 {
				mockCache.EXPECT().Set(gomock.Any(), gomock.Any()).Return(tt.cacheSetError)
			}

			marketData, err := service.cacheTokensByID(tt.tokensData)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, marketData, tt.expectedLen)
			}
		})
	}
}

func TestService_Healthy(t *testing.T) {
	tests := []struct {
		name                   string
		periodicUpdaterHealthy bool
		hasPeriodicUpdater     bool
		expected               bool
	}{
		{
			name:                   "Healthy periodic updater",
			periodicUpdaterHealthy: true,
			hasPeriodicUpdater:     true,
			expected:               true,
		},
		{
			name:                   "Unhealthy periodic updater",
			periodicUpdaterHealthy: false,
			hasPeriodicUpdater:     true,
			expected:               false,
		},
		{
			name:               "No periodic updater",
			hasPeriodicUpdater: false,
			expected:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service := &Service{}

			if tt.hasPeriodicUpdater {
				mockAPIClient := api_mocks.NewMockAPIClient(ctrl)
				mockAPIClient.EXPECT().Healthy().Return(tt.periodicUpdaterHealthy).AnyTimes()

				periodicUpdater := &PeriodicUpdater{
					apiClient: mockAPIClient,
				}
				service.periodicUpdater = periodicUpdater
			}

			result := service.Healthy()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_Markets(t *testing.T) {
	tests := []struct {
		name        string
		params      interfaces.MarketsParams
		expectCall  bool
		expectedLen int
	}{
		{
			name: "Markets with specific IDs - should call MarketsByIds",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum"},
				Currency: "usd",
			},
			expectCall:  true,
			expectedLen: 2,
		},
		{
			name: "Markets without IDs - should return empty",
			params: interfaces.MarketsParams{
				Currency: "usd",
			},
			expectCall:  false,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCache := cache_mocks.NewMockCache(ctrl)
			mockTokensService := createMockTokensService(ctrl)
			service := NewService(mockCache, createTestConfig(), mockTokensService)

			if tt.expectCall {
				// Mock cache behavior for MarketsByIds - return cached data
				mockCache.EXPECT().Get(gomock.Any()).Return(
					map[string][]byte{
						"markets:bitcoin":  sampleMarketData1,
						"markets:ethereum": sampleMarketData2,
					},
					[]string{},
					nil,
				)
			}

			result, _, err := service.Markets(tt.params)

			assert.NoError(t, err)
			assert.Len(t, result, tt.expectedLen)
		})
	}
}

func TestService_MarketsByIds(t *testing.T) {
	tests := []struct {
		name          string
		params        interfaces.MarketsParams
		cachedData    map[string][]byte
		missingKeys   []string
		cacheError    error
		expectedLen   int
		expectedError bool
	}{
		{
			name: "All data in cache",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum"},
				Currency: "usd",
			},
			cachedData: map[string][]byte{
				"markets:bitcoin":  sampleMarketData1,
				"markets:ethereum": sampleMarketData2,
			},
			missingKeys:   []string{},
			expectedLen:   2,
			expectedError: false,
		},
		{
			name: "Partial cache miss - only returns cached data",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum"},
				Currency: "usd",
			},
			cachedData: map[string][]byte{
				"markets:bitcoin": sampleMarketData1,
			},
			missingKeys:   []string{"markets:ethereum"},
			expectedLen:   1, // Only returns cached data
			expectedError: false,
		},
		{
			name: "Complete cache miss - returns empty",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin"},
				Currency: "usd",
			},
			cachedData:    map[string][]byte{},
			missingKeys:   []string{"markets:bitcoin"},
			expectedLen:   0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCache := cache_mocks.NewMockCache(ctrl)
			mockTokensService := createMockTokensService(ctrl)
			service := NewService(mockCache, createTestConfig(), mockTokensService)

			// Setup cache mock
			mockCache.EXPECT().Get(gomock.Any()).Return(
				tt.cachedData,
				tt.missingKeys,
				tt.cacheError,
			)

			result, _, err := service.MarketsByIds(tt.params)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
			}
		})
	}
}

func TestService_TopMarkets(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		currency      string
		topMarketsIDs []string
		cachedData    map[string][]byte
		expectedLen   int
		expectedError bool
	}{
		{
			name:          "Successful top markets from cache",
			limit:         2,
			currency:      "usd",
			topMarketsIDs: []string{"bitcoin", "ethereum"},
			cachedData: map[string][]byte{
				"markets_page:1": func() []byte {
					// Page data is stored as [][]byte, not []interface{}
					pageData := [][]byte{
						[]byte(`{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":45000,"market_cap":850000000000}`),
						[]byte(`{"id":"ethereum","symbol":"eth","name":"Ethereum","current_price":3000,"market_cap":360000000000}`),
					}
					bytes, _ := json.Marshal(pageData)
					return bytes
				}(),
			},
			expectedLen:   2,
			expectedError: false,
		},
		{
			name:          "No cached top market IDs",
			limit:         1,
			currency:      "usd",
			topMarketsIDs: []string{},
			cachedData:    map[string][]byte{},
			expectedLen:   0,
			expectedError: false,
		},
		{
			name:          "Default parameters",
			limit:         0, // Should default to 100
			currency:      "",
			topMarketsIDs: []string{"bitcoin"},
			cachedData:    map[string][]byte{},
			expectedLen:   0, // No data in cache, so returns empty
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCache := cache_mocks.NewMockCache(ctrl)
			mockTokensService := createMockTokensService(ctrl)
			service := NewService(mockCache, createTestConfig(), mockTokensService)

			// Initialize TopIdsManager with test data if we have topMarketsIDs
			if len(tt.topMarketsIDs) > 0 {
				mockTokenData := make([][]byte, len(tt.topMarketsIDs))
				for i, tokenID := range tt.topMarketsIDs {
					mockTokenData[i] = []byte(fmt.Sprintf(`{"id":"%s","symbol":"%s","name":"%s"}`, tokenID, tokenID, tokenID))
				}

				pageData := []PageData{
					{
						Page: 1,
						Data: mockTokenData,
					},
				}

				service.topIdsManager.UpdatePagesFromPageData(pageData)
			}

			// Mock cache reads for MarketsByIds
			mockCache.EXPECT().Get(gomock.Any()).DoAndReturn(func(keys []string) (map[string][]byte, []string, error) {
				result := make(map[string][]byte)
				var missingKeys []string

				for _, key := range keys {
					if data, exists := tt.cachedData[key]; exists {
						result[key] = data
					} else {
						missingKeys = append(missingKeys, key)
					}
				}

				return result, missingKeys, nil
			}).AnyTimes()

			result, err := service.TopMarkets(tt.limit, tt.currency)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
			}
		})
	}
}

func TestService_MarketsByIds_DefaultParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Helper functions for creating pointers
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }

	mockCache := cache_mocks.NewMockCache(ctrl)

	// Create config with MarketParamsNormalize to test default values
	cfg := createTestConfig()
	cfg.CoingeckoMarkets.MarketParamsNormalize = &config.MarketParamsNormalize{
		VsCurrency: strPtr("usd"),
		Order:      strPtr("market_cap_desc"),
		PerPage:    intPtr(MARKETS_DEFAULT_CHUNK_SIZE),
	}

	mockTokensService := createMockTokensService(ctrl)
	service := NewService(mockCache, cfg, mockTokensService)

	// Test that service works with cached data and default parameters
	params := interfaces.MarketsParams{
		IDs: []string{"bitcoin"},
		// Currency, Order, PerPage, Page not set - should get defaults
	}

	// Return cached data so no API call is needed
	mockCache.EXPECT().Get(gomock.Any()).Return(
		map[string][]byte{
			"markets:bitcoin": sampleMarketData1,
		},
		[]string{}, // No missing keys
		nil,
	)

	result, _, err := service.MarketsByIds(params)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestService_TopMarketIds(t *testing.T) {
	t.Run("TopMarketIds with limit 0 returns all available tokens", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create fresh mocks for this test
		mockCache := cache_mocks.NewMockCache(ctrl)
		mockTokensService := interface_mocks.NewMockCoingeckoTokensService(ctrl)
		mockTokensService.EXPECT().SubscribeOnTokensUpdate().Return(events.NewSubscriptionManager().Subscribe()).AnyTimes()
		mockTokensService.EXPECT().GetTokenIds().Return([]string{}).AnyTimes()

		// Create config and service
		cfg := createTestConfig()
		service := NewService(mockCache, cfg, mockTokensService)

		// Create test data for TopIdsManager
		tokens := []string{"bitcoin", "ethereum", "ada", "sol", "dot"}

		// Create mock token data
		mockTokenData := [][]byte{
			[]byte(`{"id":"bitcoin","symbol":"btc","name":"Bitcoin"}`),
			[]byte(`{"id":"ethereum","symbol":"eth","name":"Ethereum"}`),
			[]byte(`{"id":"ada","symbol":"ada","name":"Cardano"}`),
			[]byte(`{"id":"sol","symbol":"sol","name":"Solana"}`),
			[]byte(`{"id":"dot","symbol":"dot","name":"Polkadot"}`),
		}

		// Simulate page data update to populate TopIdsManager
		pageData := []PageData{
			{
				Page: 1,
				Data: mockTokenData,
			},
		}

		// Initialize TopIdsManager with test data
		service.topIdsManager.UpdatePagesFromPageData(pageData)

		result, err := service.TopMarketIds(0) // limit 0 should use default (250)

		assert.NoError(t, err)
		assert.Equal(t, tokens, result)
	})

	t.Run("TopMarketIds with positive limit returns limited tokens", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create fresh mocks for this test
		mockCache := cache_mocks.NewMockCache(ctrl)
		mockTokensService := interface_mocks.NewMockCoingeckoTokensService(ctrl)
		mockTokensService.EXPECT().SubscribeOnTokensUpdate().Return(events.NewSubscriptionManager().Subscribe()).AnyTimes()
		mockTokensService.EXPECT().GetTokenIds().Return([]string{}).AnyTimes()

		// Create config and service
		cfg := createTestConfig()
		service := NewService(mockCache, cfg, mockTokensService)

		// Create test data for TopIdsManager
		tokens := []string{"bitcoin", "ethereum", "ada", "sol", "dot"}

		// Create mock token data
		mockTokenData := [][]byte{
			[]byte(`{"id":"bitcoin","symbol":"btc","name":"Bitcoin"}`),
			[]byte(`{"id":"ethereum","symbol":"eth","name":"Ethereum"}`),
			[]byte(`{"id":"ada","symbol":"ada","name":"Cardano"}`),
			[]byte(`{"id":"sol","symbol":"sol","name":"Solana"}`),
			[]byte(`{"id":"dot","symbol":"dot","name":"Polkadot"}`),
		}

		// Simulate page data update to populate TopIdsManager
		pageData := []PageData{
			{
				Page: 1,
				Data: mockTokenData,
			},
		}

		// Initialize TopIdsManager with test data
		service.topIdsManager.UpdatePagesFromPageData(pageData)

		result, err := service.TopMarketIds(3) // limit to 3

		assert.NoError(t, err)
		assert.Equal(t, tokens[:3], result) // Should return first 3 tokens
	})
}

func TestMarketsByPageAndMarketsByIdsReturnSameFormat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create test data for 3 tokens
	testData1 := []byte(`{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":45000,"market_cap":850000000000}`)
	testData2 := []byte(`{"id":"ethereum","symbol":"eth","name":"Ethereum","current_price":3000,"market_cap":360000000000}`)
	testData3 := []byte(`{"id":"cardano","symbol":"ada","name":"Cardano","current_price":1.5,"market_cap":50000000000}`)

	// Setup cache mock
	mockCache := cache_mocks.NewMockCache(ctrl)
	config := createTestConfig()

	// Mock tokens service
	mockTokensService := interface_mocks.NewMockCoingeckoTokensService(ctrl)
	mockTokensService.EXPECT().SubscribeOnTokensUpdate().Return(createMockSubscription()).AnyTimes()

	service := NewService(mockCache, config, mockTokensService)

	t.Run("should return same format for page and ids", func(t *testing.T) {
		// Setup cache expectations for IDs
		expectedIDCache := map[string][]byte{
			"markets:bitcoin":  testData1,
			"markets:ethereum": testData2,
			"markets:cardano":  testData3,
		}

		// Setup cache expectations for page
		pageData := [][]byte{testData1, testData2, testData3}
		pageDataBytes, _ := json.Marshal(pageData)
		expectedPageCache := map[string][]byte{
			"markets_page:1": pageDataBytes,
		}

		// Test MarketsByIds
		mockCache.EXPECT().Get([]string{"markets:bitcoin", "markets:ethereum", "markets:cardano"}).
			Return(expectedIDCache, []string{}, nil)

		idsParams := interfaces.MarketsParams{
			IDs:      []string{"bitcoin", "ethereum", "cardano"},
			Currency: "usd",
		}

		idsResponse, _, err := service.MarketsByIds(idsParams)
		assert.NoError(t, err)
		assert.Len(t, idsResponse, 3)

		// Test MarketsByPage
		mockCache.EXPECT().Get([]string{"markets_page:1"}).
			Return(expectedPageCache, []string{}, nil)

		pageParams := interfaces.MarketsParams{
			Currency: "usd",
		}

		pageResponse, _, err := service.MarketsByPage(1, 1, pageParams)
		assert.NoError(t, err)
		assert.Len(t, pageResponse, 3)

		// Verify both responses have the same structure
		// Both should return []interface{} where each element is a map[string]interface{}
		for i := 0; i < 3; i++ {
			idsItem := idsResponse[i]
			pageItem := pageResponse[i]

			// Both should be map[string]interface{}
			idsMap, idsOk := idsItem.(map[string]interface{})
			pageMap, pageOk := pageItem.(map[string]interface{})

			assert.True(t, idsOk, "MarketsByIds should return map[string]interface{}")
			assert.True(t, pageOk, "MarketsByPage should return map[string]interface{}")

			// Compare specific fields to ensure they're the same
			assert.Equal(t, idsMap["id"], pageMap["id"], "IDs should match")
			assert.Equal(t, idsMap["symbol"], pageMap["symbol"], "Symbols should match")
			assert.Equal(t, idsMap["name"], pageMap["name"], "Names should match")
			assert.Equal(t, idsMap["current_price"], pageMap["current_price"], "Prices should match")
		}
	})
}
