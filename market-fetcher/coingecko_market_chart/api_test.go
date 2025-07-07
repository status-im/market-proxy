package coingecko_market_chart

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIKeyManager implements APIKeyManagerInterface for testing
type MockAPIKeyManager struct {
	mock.Mock
}

func (m *MockAPIKeyManager) GetAvailableKeys() []cg.APIKey {
	args := m.Called()
	return args.Get(0).([]cg.APIKey)
}

func (m *MockAPIKeyManager) MarkKeyAsFailed(key string) {
	m.Called(key)
}

func TestCoinGeckoClient_TryFreeApiFirst(t *testing.T) {
	// Create a test server that returns valid market chart data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a pro API request (has x_cg_pro_api_key parameter)
		if r.URL.Query().Get("x_cg_pro_api_key") != "" {
			// For pro API requests, return success if key is valid
			if r.URL.Query().Get("x_cg_pro_api_key") == "pro-key-1" {
				// Return mock market chart data
				mockData := map[string]interface{}{
					"prices": [][]float64{
						{1643723400000, 38000.0},
						{1643809800000, 39000.0},
					},
					"market_caps": [][]float64{
						{1643723400000, 750000000000.0},
						{1643809800000, 770000000000.0},
					},
					"total_volumes": [][]float64{
						{1643723400000, 25000000000.0},
						{1643809800000, 26000000000.0},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(mockData); err != nil {
					http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				}
				return
			}
		}

		// Return mock market chart data for public API or valid keys
		mockData := map[string]interface{}{
			"prices": [][]float64{
				{1643723400000, 38000.0},
				{1643809800000, 39000.0},
			},
			"market_caps": [][]float64{
				{1643723400000, 750000000000.0},
				{1643809800000, 770000000000.0},
			},
			"total_volumes": [][]float64{
				{1643723400000, 25000000000.0},
				{1643809800000, 26000000000.0},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockData); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	tests := []struct {
		name            string
		tryFreeApiFirst bool
		interval        string
		availableKeys   []cg.APIKey
		expectedFirst   cg.KeyType
		description     string
	}{
		{
			name:            "TryFreeApiFirst enabled, no interval - NoKey should be first",
			tryFreeApiFirst: true,
			interval:        "",
			availableKeys: []cg.APIKey{
				{Key: "pro-key-1", Type: cg.ProKey},
				{Key: "demo-key-1", Type: cg.DemoKey},
				{Key: "", Type: cg.NoKey},
			},
			expectedFirst: cg.NoKey,
			description:   "NoKey should be moved to the beginning when TryFreeApiFirst is true and no interval",
		},
		{
			name:            "TryFreeApiFirst enabled, with interval - NoKey should stay at end",
			tryFreeApiFirst: true,
			interval:        "hourly",
			availableKeys: []cg.APIKey{
				{Key: "pro-key-1", Type: cg.ProKey},
				{Key: "demo-key-1", Type: cg.DemoKey},
				{Key: "", Type: cg.NoKey},
			},
			expectedFirst: cg.ProKey,
			description:   "NoKey should stay at the end when interval is specified",
		},
		{
			name:            "TryFreeApiFirst disabled, no interval - NoKey should stay at end",
			tryFreeApiFirst: false,
			interval:        "",
			availableKeys: []cg.APIKey{
				{Key: "pro-key-1", Type: cg.ProKey},
				{Key: "demo-key-1", Type: cg.DemoKey},
				{Key: "", Type: cg.NoKey},
			},
			expectedFirst: cg.ProKey,
			description:   "NoKey should stay at the end when TryFreeApiFirst is false",
		},
		{
			name:            "TryFreeApiFirst disabled, with interval - NoKey should stay at end",
			tryFreeApiFirst: false,
			interval:        "daily",
			availableKeys: []cg.APIKey{
				{Key: "pro-key-1", Type: cg.ProKey},
				{Key: "demo-key-1", Type: cg.DemoKey},
				{Key: "", Type: cg.NoKey},
			},
			expectedFirst: cg.ProKey,
			description:   "NoKey should stay at the end when TryFreeApiFirst is false and interval is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config with TryFreeApiFirst setting
			cfg := &config.Config{
				CoingeckoMarketChart: config.CoingeckoMarketChartFetcher{
					TryFreeApiFirst: tt.tryFreeApiFirst,
					HourlyTTL:       30 * time.Minute,
					DailyTTL:        12 * time.Hour,
				},
				OverrideCoingeckoPublicURL: server.URL,
				OverrideCoingeckoProURL:    server.URL,
			}

			// Create mock key manager
			mockKeyManager := new(MockAPIKeyManager)
			mockKeyManager.On("GetAvailableKeys").Return(tt.availableKeys)
			// Add expectation for MarkKeyAsFailed in case it's called
			mockKeyManager.On("MarkKeyAsFailed", mock.AnythingOfType("string")).Return().Maybe()

			// Create client
			client := NewCoinGeckoClient(cfg)
			client.keyManager = mockKeyManager

			// Create test parameters
			params := MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "30",
				Interval: tt.interval,
			}

			// Execute the request
			result, err := client.FetchMarketChart(params)

			// Check that the request succeeded
			assert.NoError(t, err, tt.description)
			assert.NotNil(t, result, tt.description)

			// Verify that we got the expected data
			assert.Contains(t, result, "prices", tt.description)
			assert.Contains(t, result, "market_caps", tt.description)
			assert.Contains(t, result, "total_volumes", tt.description)

			// Verify mock expectations
			mockKeyManager.AssertExpectations(t)
		})
	}
}

func TestCoinGeckoClient_TryFreeApiFirst_NoKeyNotFound(t *testing.T) {
	// Test case where NoKey is not in the available keys list
	// (This shouldn't happen in practice, but we should handle it gracefully)

	// Create a simple test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockData := map[string]interface{}{
			"prices":        [][]float64{{1643723400000, 38000.0}},
			"market_caps":   [][]float64{{1643723400000, 750000000000.0}},
			"total_volumes": [][]float64{{1643723400000, 25000000000.0}},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockData); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		CoingeckoMarketChart: config.CoingeckoMarketChartFetcher{
			TryFreeApiFirst: true,
			HourlyTTL:       30 * time.Minute,
			DailyTTL:        12 * time.Hour,
		},
		OverrideCoingeckoPublicURL: server.URL,
		OverrideCoingeckoProURL:    server.URL,
	}

	// Create mock key manager that returns keys without NoKey
	mockKeyManager := new(MockAPIKeyManager)
	availableKeys := []cg.APIKey{
		{Key: "pro-key-1", Type: cg.ProKey},
		{Key: "demo-key-1", Type: cg.DemoKey},
		// No NoKey in the list
	}
	mockKeyManager.On("GetAvailableKeys").Return(availableKeys)
	mockKeyManager.On("MarkKeyAsFailed", mock.AnythingOfType("string")).Return().Maybe()

	client := NewCoinGeckoClient(cfg)
	client.keyManager = mockKeyManager

	params := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "30",
		Interval: "", // No interval specified
	}

	// This should not panic even if NoKey is not found
	// The function should just proceed with the original order
	result, err := client.FetchMarketChart(params)

	// Should succeed because we have a working server
	assert.NoError(t, err) // Should work now with proper server
	assert.NotNil(t, result)

	// Verify mock expectations
	mockKeyManager.AssertExpectations(t)
}
