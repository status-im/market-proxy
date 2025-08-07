package coingecko_market_chart

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockIAPIClient implements IAPIClient for testing
type MockIAPIClient struct {
	mock.Mock
}

func (m *MockIAPIClient) FetchMarketChart(params MarketChartParams) (map[string][]byte, error) {
	args := m.Called(params)
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (m *MockIAPIClient) Healthy() bool {
	args := m.Called()
	return args.Bool(0)
}

// createTestConfig creates a test configuration with market chart settings
func createTestConfig() *config.Config {
	return &config.Config{
		CoingeckoMarketChart: config.MarketChartFetcherConfig{
			HourlyTTL:          30 * time.Minute,
			DailyTTL:           12 * time.Hour,
			DailyDataThreshold: 90,
		},
	}
}

// createTestResponseMapForService creates test data for service tests (returns map[string][]byte)
func createTestResponseMapForService(days int) map[string][]byte {
	now := time.Now()
	var prices []MarketChartData
	var marketCaps []MarketChartData
	var totalVolumes []MarketChartData

	// Create data for the specified number of days
	for i := 0; i < days; i++ {
		timestamp := now.AddDate(0, 0, -days+i).Unix() * 1000 // milliseconds
		price := float64(50000 + i*100)                       // Mock price data
		marketCap := float64(1000000000 + i*1000000)          // Mock market cap data
		volume := float64(10000000 + i*100000)                // Mock volume data

		prices = append(prices, MarketChartData{float64(timestamp), price})
		marketCaps = append(marketCaps, MarketChartData{float64(timestamp), marketCap})
		totalVolumes = append(totalVolumes, MarketChartData{float64(timestamp), volume})
	}

	// Marshal to JSON bytes as expected by the API client
	pricesBytes, _ := json.Marshal(prices)
	marketCapsBytes, _ := json.Marshal(marketCaps)
	totalVolumesBytes, _ := json.Marshal(totalVolumes)

	return map[string][]byte{
		"prices":        pricesBytes,
		"market_caps":   marketCapsBytes,
		"total_volumes": totalVolumesBytes,
	}
}

// Test data
var sampleMarketChartData = createTestResponseMapForService(90)

func TestService_Basic(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := &config.Config{
		CoingeckoMarkets: config.MarketsFetcherConfig{
			TTL: 30 * time.Second,
		},
	}

	// Create service
	service := NewService(cacheService, cfg)

	// Test Start method
	err := service.Start(context.Background())
	assert.NoError(t, err)

	// Test Stop method (should not panic)
	assert.NotPanics(t, func() {
		service.Stop()
	})
}

func TestService_StartWithoutCache(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		CoingeckoMarkets: config.MarketsFetcherConfig{
			TTL: 30 * time.Second,
		},
	}

	// Create service without cache
	service := NewService(nil, cfg)

	// Test Start should fail
	err := service.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache dependency not provided")
}

func TestService_Healthy(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Mock API client
	mockClient := new(MockIAPIClient)
	mockClient.On("Healthy").Return(true)
	service.apiClient = mockClient

	// Test Healthy
	assert.True(t, service.Healthy())
	mockClient.AssertExpectations(t)
}

func TestService_SelectTTL(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config with market chart settings
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	tests := []struct {
		name     string
		params   MarketChartParams
		expected time.Duration
	}{
		{
			name: "Days <= 90 should use 30 minutes TTL",
			params: MarketChartParams{
				Days: "30",
			},
			expected: 30 * time.Minute,
		},
		{
			name: "Days = 90 should use 30 minutes TTL",
			params: MarketChartParams{
				Days: "90",
			},
			expected: 30 * time.Minute,
		},
		{
			name: "Days > 90 should use 12 hours TTL",
			params: MarketChartParams{
				Days: "180",
			},
			expected: 12 * time.Hour,
		},
		{
			name: "Days = 365 should use 12 hours TTL",
			params: MarketChartParams{
				Days: "365",
			},
			expected: 12 * time.Hour,
		},
		{
			name: "Days = max should use 12 hours TTL",
			params: MarketChartParams{
				Days: "max",
			},
			expected: 12 * time.Hour,
		},
		{
			name: "Invalid days should use daily TTL",
			params: MarketChartParams{
				Days: "invalid",
			},
			expected: 12 * time.Hour, // From config DailyTTL
		},
		{
			name: "Empty days should use daily TTL",
			params: MarketChartParams{
				Days: "",
			},
			expected: 12 * time.Hour, // From config DailyTTL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.selectTTL(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_CreateCacheKey(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	tests := []struct {
		name     string
		params   MarketChartParams
		expected string
	}{
		{
			name: "Basic cache key",
			params: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "30",
			},
			expected: "market_chart:bitcoin:usd:days:30",
		},
		{
			name: "ICache key with interval",
			params: MarketChartParams{
				ID:       "ethereum",
				Currency: "eur",
				Days:     "90",
				Interval: "hourly",
			},
			expected: "market_chart:ethereum:eur:days:90:interval:hourly",
		},
		{
			name: "ICache key without interval",
			params: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "365",
			},
			expected: "market_chart:bitcoin:usd:days:365",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.createCacheKey(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_MarketChart_CacheMiss(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config with market chart settings
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Mock API client
	mockClient := new(MockIAPIClient)
	roundedParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "90", // Rounded up from 30 to 90
	}
	mockClient.On("FetchMarketChart", roundedParams).Return(sampleMarketChartData, nil)
	service.apiClient = mockClient

	// Test parameters (original request)
	params := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "30",
	}

	// Call MarketChart
	result, err := service.MarketChart(params)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "prices")
	assert.Contains(t, result, "market_caps")
	assert.Contains(t, result, "total_volumes")

	// Verify that API was called with rounded params
	mockClient.AssertExpectations(t)
}

func TestService_MarketChart_CacheHit(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Mock API client (should not be called)
	mockClient := new(MockIAPIClient)
	service.apiClient = mockClient

	// Pre-populate cache with rounded data
	roundedParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "90",
	}
	cacheKey := service.createCacheKey(roundedParams)

	// Marshal the sample data to cache format
	cacheData, _ := json.Marshal(sampleMarketChartData)
	cachemap := map[string][]byte{
		cacheKey: cacheData,
	}
	if err := cacheService.Set(cachemap, 30*time.Minute); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test parameters (original request that should hit cache)
	params := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "30",
	}

	// Call MarketChart
	result, err := service.MarketChart(params)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "prices")

	// Verify that API was NOT called
	mockClient.AssertNotCalled(t, "FetchMarketChart")
}

func TestService_MarketChart_APIError(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Mock API client to return error
	mockClient := new(MockIAPIClient)
	roundedParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "90",
	}
	mockClient.On("FetchMarketChart", roundedParams).Return(map[string][]byte{}, fmt.Errorf("API error"))
	service.apiClient = mockClient

	// Test parameters
	params := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "30",
	}

	// Call MarketChart
	result, err := service.MarketChart(params)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch market chart data")

	// Verify that API was called
	mockClient.AssertExpectations(t)
}

func TestService_MarketChart_DefaultValues(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Mock API client
	mockClient := new(MockIAPIClient)
	roundedParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd", // Default currency
		Days:     "90",  // Rounded up from default 30
	}
	mockClient.On("FetchMarketChart", roundedParams).Return(sampleMarketChartData, nil)
	service.apiClient = mockClient

	// Test parameters with empty values
	params := MarketChartParams{
		ID: "bitcoin",
		// Currency and Days are empty, should use defaults
	}

	// Call MarketChart
	result, err := service.MarketChart(params)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify that API was called with defaults
	mockClient.AssertExpectations(t)
}

func TestService_MarketChart_RoundUpLogic(t *testing.T) {
	tests := []struct {
		name         string
		originalDays string
		roundedDays  string
		expectedTTL  time.Duration
	}{
		{
			name:         "30 days should be rounded up to 90",
			originalDays: "30",
			roundedDays:  "90",
			expectedTTL:  30 * time.Minute,
		},
		{
			name:         "60 days should be rounded up to 90",
			originalDays: "60",
			roundedDays:  "90",
			expectedTTL:  30 * time.Minute,
		},
		{
			name:         "90 days should stay 90",
			originalDays: "90",
			roundedDays:  "90",
			expectedTTL:  30 * time.Minute,
		},
		{
			name:         "180 days should be rounded up to 365",
			originalDays: "180",
			roundedDays:  "365",
			expectedTTL:  12 * time.Hour,
		},
		{
			name:         "365 days should stay 365",
			originalDays: "365",
			roundedDays:  "365",
			expectedTTL:  12 * time.Hour,
		},
		{
			name:         "max days should stay max",
			originalDays: "max",
			roundedDays:  "max",
			expectedTTL:  12 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh cache service for each test
			cacheConfig := cache.DefaultCacheConfig()
			cacheService := cache.NewService(cacheConfig)

			// Create test config
			cfg := createTestConfig()

			// Create service
			service := NewService(cacheService, cfg)

			// Mock API client
			mockClient := new(MockIAPIClient)
			roundedParams := MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     tt.roundedDays,
			}
			mockClient.On("FetchMarketChart", roundedParams).Return(sampleMarketChartData, nil)
			service.apiClient = mockClient

			// Test parameters
			params := MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     tt.originalDays,
			}

			// Call MarketChart
			result, err := service.MarketChart(params)

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Verify TTL selection
			ttl := service.selectTTL(roundedParams)
			assert.Equal(t, tt.expectedTTL, ttl)

			// Verify API was called with rounded params
			mockClient.AssertExpectations(t)
		})
	}
}

func TestService_MarketChart_DataFilter(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Mock API client
	mockClient := new(MockIAPIClient)
	roundedParams := MarketChartParams{
		ID:         "bitcoin",
		Currency:   "usd",
		Days:       "90",
		DataFilter: "prices", // This should be passed through without rounding
	}
	mockClient.On("FetchMarketChart", roundedParams).Return(sampleMarketChartData, nil)
	service.apiClient = mockClient

	// Test parameters with data filter
	params := MarketChartParams{
		ID:         "bitcoin",
		Currency:   "usd",
		Days:       "30",
		DataFilter: "prices",
	}

	// Call MarketChart
	result, err := service.MarketChart(params)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// The result should be filtered to only include prices
	// (This depends on the strip function working correctly)
	assert.Contains(t, result, "prices")

	// Verify that API was called
	mockClient.AssertExpectations(t)
}

func TestService_GetCachedData_NotFound(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Try to get non-existent data from cache
	result, err := service.getCachedData("non-existent-key")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "data not found in cache")
}

func TestService_CacheData_Success(t *testing.T) {
	// Create cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewService(cacheConfig)

	// Create test config
	cfg := createTestConfig()

	// Create service
	service := NewService(cacheService, cfg)

	// Test caching
	cacheKey := "test-key"
	params := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "30",
	}

	// Use sampleMarketChartData directly (it's already map[string][]byte)
	err := service.cacheData(cacheKey, sampleMarketChartData, params)

	// Verify
	assert.NoError(t, err)

	// Verify data was cached
	cachedData, err := service.getCachedData(cacheKey)
	assert.NoError(t, err)
	assert.NotNil(t, cachedData)
	assert.Contains(t, cachedData, "prices")
}
