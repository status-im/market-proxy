package coingecko_markets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/status-im/market-proxy/cache"
	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCache implements cache.Cache interface for testing
type MockCache struct {
	mock.Mock
}

func (m *MockCache) GetOrLoad(keys []string, loader cache.LoaderFunc, loadOnlyMissingKeys bool, ttl time.Duration) (map[string][]byte, error) {
	args := m.Called(keys, loader, loadOnlyMissingKeys, ttl)
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (m *MockCache) Get(keys []string) (map[string][]byte, []string, error) {
	args := m.Called(keys)
	return args.Get(0).(map[string][]byte), args.Get(1).([]string), args.Error(2)
}

func (m *MockCache) Set(data map[string][]byte, ttl time.Duration) error {
	args := m.Called(data, ttl)
	return args.Error(0)
}

// MockServiceAPIClient implements APIClient interface for testing service
type MockServiceAPIClient struct {
	mock.Mock
}

func (m *MockServiceAPIClient) FetchPage(params cg.MarketsParams) ([][]byte, error) {
	args := m.Called(params)
	return args.Get(0).([][]byte), args.Error(1)
}

func (m *MockServiceAPIClient) Healthy() bool {
	args := m.Called()
	return args.Bool(0)
}

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
			ChunkSize:    250,
			RequestDelay: 1000 * time.Millisecond,
			TTL:          300 * time.Second,
		},
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
	}
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name   string
		cache  cache.Cache
		config *config.Config
	}{
		{
			name:   "Valid service creation",
			cache:  &MockCache{},
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
			service := NewService(tt.cache, tt.config)
			assert.NotNil(t, service)
			assert.Equal(t, tt.cache, service.cache)
			assert.Equal(t, tt.config, service.config)
			assert.NotNil(t, service.metricsWriter)
			assert.NotNil(t, service.apiClient)
		})
	}
}

func TestService_Start(t *testing.T) {
	tests := []struct {
		name        string
		cache       cache.Cache
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Start with valid cache",
			cache:       &MockCache{},
			expectError: false,
		},
		{
			name:        "Start with nil cache",
			cache:       nil,
			expectError: true,
			errorMsg:    "cache dependency not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &Service{
				cache:  tt.cache,
				config: createTestConfig(),
			}

			err := service.Start(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_Stop(t *testing.T) {
	service := NewService(&MockCache{}, createTestConfig())

	// Should not panic
	assert.NotPanics(t, func() {
		service.Stop()
	})
}

func TestService_parseTokensData(t *testing.T) {
	service := NewService(&MockCache{}, createTestConfig())

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
			marketData, cacheData, err := service.parseTokensData(tt.tokensData)

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
			mockCache := &MockCache{}
			service := NewService(mockCache, createTestConfig())

			if len(tt.tokensData) > 0 {
				mockCache.On("Set", mock.AnythingOfType("map[string][]uint8"), mock.AnythingOfType("time.Duration")).Return(tt.cacheSetError)
			}

			marketData, err := service.cacheTokensByID(tt.tokensData)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, marketData, tt.expectedLen)
			}

			mockCache.AssertExpectations(t)
		})
	}
}

func TestService_Healthy(t *testing.T) {
	tests := []struct {
		name             string
		apiClientHealthy bool
		expected         bool
	}{
		{
			name:             "Healthy API client",
			apiClientHealthy: true,
			expected:         true,
		},
		{
			name:             "Unhealthy API client",
			apiClientHealthy: false,
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPIClient := &MockServiceAPIClient{}
			service := &Service{
				apiClient: mockAPIClient,
			}

			mockAPIClient.On("Healthy").Return(tt.apiClientHealthy)

			result := service.Healthy()
			assert.Equal(t, tt.expected, result)

			mockAPIClient.AssertExpectations(t)
		})
	}
}

func TestService_Markets(t *testing.T) {
	tests := []struct {
		name        string
		params      cg.MarketsParams
		expectCall  bool
		expectedLen int
	}{
		{
			name: "Markets with specific IDs - should call MarketsByIds",
			params: cg.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum"},
				Currency: "usd",
			},
			expectCall:  true,
			expectedLen: 2,
		},
		{
			name: "Markets without IDs - should return empty",
			params: cg.MarketsParams{
				Currency: "usd",
			},
			expectCall:  false,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := &MockCache{}
			mockAPIClient := &MockServiceAPIClient{}
			service := NewService(mockCache, createTestConfig())
			service.apiClient = mockAPIClient

			if tt.expectCall {
				// Mock cache behavior for MarketsByIds
				mockCache.On("Get", mock.AnythingOfType("[]string")).Return(
					map[string][]byte{},
					[]string{"markets:bitcoin", "markets:ethereum"},
					nil,
				)
				mockAPIClient.On("FetchPage", mock.AnythingOfType("coingecko_common.MarketsParams")).Return(
					[][]byte{sampleMarketData1, sampleMarketData2},
					nil,
				)
				mockCache.On("Set", mock.AnythingOfType("map[string][]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)
			}

			result, err := service.Markets(tt.params)

			assert.NoError(t, err)
			assert.Len(t, result, tt.expectedLen)

			if tt.expectCall {
				mockCache.AssertExpectations(t)
				mockAPIClient.AssertExpectations(t)
			}
		})
	}
}

func TestService_MarketsByIds(t *testing.T) {
	tests := []struct {
		name          string
		params        cg.MarketsParams
		cachedData    map[string][]byte
		missingKeys   []string
		cacheError    error
		apiData       [][]byte
		apiError      error
		expectedLen   int
		expectedError bool
	}{
		{
			name: "All data in cache",
			params: cg.MarketsParams{
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
			name: "Partial cache miss - fetch from API",
			params: cg.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum"},
				Currency: "usd",
			},
			cachedData: map[string][]byte{
				"markets:bitcoin": sampleMarketData1,
			},
			missingKeys:   []string{"markets:ethereum"},
			apiData:       [][]byte{sampleMarketData1, sampleMarketData2},
			expectedLen:   2,
			expectedError: false,
		},
		{
			name: "API fetch error",
			params: cg.MarketsParams{
				IDs:      []string{"bitcoin"},
				Currency: "usd",
			},
			cachedData:    map[string][]byte{},
			missingKeys:   []string{"markets:bitcoin"},
			apiError:      errors.New("API error"),
			expectedLen:   0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := &MockCache{}
			mockAPIClient := &MockServiceAPIClient{}
			service := NewService(mockCache, createTestConfig())
			service.apiClient = mockAPIClient

			// Setup cache mock
			mockCache.On("Get", mock.AnythingOfType("[]string")).Return(
				tt.cachedData,
				tt.missingKeys,
				tt.cacheError,
			)

			// Setup API mock if needed
			if len(tt.missingKeys) > 0 {
				mockAPIClient.On("FetchPage", mock.AnythingOfType("coingecko_common.MarketsParams")).Return(
					tt.apiData,
					tt.apiError,
				)

				if tt.apiError == nil {
					mockCache.On("Set", mock.AnythingOfType("map[string][]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)
				}
			}

			result, err := service.MarketsByIds(tt.params)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
			}

			mockCache.AssertExpectations(t)
			if len(tt.missingKeys) > 0 {
				mockAPIClient.AssertExpectations(t)
			}
		})
	}
}

func TestService_TopMarkets(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		currency      string
		apiData       [][]byte
		apiError      error
		expectedLen   int
		expectedError bool
	}{
		{
			name:          "Successful top markets fetch",
			limit:         2,
			currency:      "usd",
			apiData:       [][]byte{sampleMarketData1, sampleMarketData2},
			expectedLen:   2,
			expectedError: false,
		},
		{
			name:          "API error",
			limit:         1,
			currency:      "usd",
			apiError:      errors.New("API error"),
			expectedLen:   0,
			expectedError: true,
		},
		{
			name:          "Default parameters",
			limit:         0,  // Should default to 100
			currency:      "", // Should default to "usd"
			apiData:       [][]byte{sampleMarketData1},
			expectedLen:   1,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := &MockCache{}
			mockAPIClient := &MockServiceAPIClient{}
			service := NewService(mockCache, createTestConfig())
			service.apiClient = mockAPIClient

			// Mock the paginated fetcher behavior through API client
			mockAPIClient.On("FetchPage", mock.AnythingOfType("coingecko_common.MarketsParams")).Return(
				tt.apiData,
				tt.apiError,
			).Maybe()

			if tt.apiError == nil {
				mockCache.On("Set", mock.AnythingOfType("map[string][]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)
			}

			result, err := service.TopMarkets(tt.limit, tt.currency)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
			}

			if tt.apiError == nil {
				mockCache.AssertExpectations(t)
			}
		})
	}
}

func TestService_MarketsByIds_DefaultParams(t *testing.T) {
	mockCache := &MockCache{}
	mockAPIClient := &MockServiceAPIClient{}
	service := NewService(mockCache, createTestConfig())
	service.apiClient = mockAPIClient

	// Test that default parameters are applied
	params := cg.MarketsParams{
		IDs: []string{"bitcoin"},
		// Currency, Order, PerPage, Page not set - should get defaults
	}

	mockCache.On("Get", mock.AnythingOfType("[]string")).Return(
		map[string][]byte{},
		[]string{"markets:bitcoin"},
		nil,
	)

	mockAPIClient.On("FetchPage", mock.MatchedBy(func(p cg.MarketsParams) bool {
		return p.Currency == "usd" &&
			p.Order == "market_cap_desc" &&
			p.PerPage == MARKETS_DEFAULT_CHUNK_SIZE &&
			p.Page == 1
	})).Return([][]byte{sampleMarketData1}, nil)

	mockCache.On("Set", mock.AnythingOfType("map[string][]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)

	result, err := service.MarketsByIds(params)

	assert.NoError(t, err)
	assert.Len(t, result, 1)

	mockCache.AssertExpectations(t)
	mockAPIClient.AssertExpectations(t)
}
