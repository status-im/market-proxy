package coingecko_prices

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cg "github.com/status-im/market-proxy/coingecko_common"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/stretchr/testify/assert"
)

func TestNewCoinGeckoClient(t *testing.T) {
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{"test-key"},
		},
	}

	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)
	client := NewCoinGeckoClient(cfg, metricsWriter)
	assert.NotNil(t, client)
	assert.NotNil(t, client.config)
	assert.NotNil(t, client.keyManager)
	assert.NotNil(t, client.httpClient)
	assert.False(t, client.Healthy())
}

func TestCoinGeckoClient_FetchPrices_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		assert.Equal(t, "/api/v3/simple/price", r.URL.Path)
		assert.Equal(t, "bitcoin,ethereum", r.URL.Query().Get("ids"))
		assert.Equal(t, "usd,eur", r.URL.Query().Get("vs_currencies"))

		// Return test response
		response := map[string]map[string]float64{
			"bitcoin": {
				"usd": 50000.0,
				"eur": 45000.0,
			},
			"ethereum": {
				"usd": 3000.0,
				"eur": 2700.0,
			},
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := &config.Config{
		OverrideCoingeckoPublicURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
	}
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)
	client := NewCoinGeckoClient(cfg, metricsWriter)

	// Test fetching prices
	params := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}
	tokenData, err := client.FetchPrices(params)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, tokenData)
	assert.True(t, client.Healthy())

	// Verify we got data for both tokens
	assert.Equal(t, 2, len(tokenData)) // 2 tokens
	assert.Contains(t, tokenData, "bitcoin")
	assert.Contains(t, tokenData, "ethereum")

	// Parse and verify bitcoin data
	var bitcoinData map[string]interface{}
	err = json.Unmarshal(tokenData["bitcoin"], &bitcoinData)
	assert.NoError(t, err)
	assert.Equal(t, 50000.0, bitcoinData["usd"])
	assert.Equal(t, 45000.0, bitcoinData["eur"])

	// Parse and verify ethereum data
	var ethereumData map[string]interface{}
	err = json.Unmarshal(tokenData["ethereum"], &ethereumData)
	assert.NoError(t, err)
	assert.Equal(t, 3000.0, ethereumData["usd"])
	assert.Equal(t, 2700.0, ethereumData["eur"])
}

func TestCoinGeckoClient_FetchPrices_Error(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := &config.Config{
		OverrideCoingeckoPublicURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
	}
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)
	client := NewCoinGeckoClient(cfg, metricsWriter)

	// Test fetching prices
	params := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}
	tokenData, err := client.FetchPrices(params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, tokenData)
	assert.False(t, client.Healthy())
}

func TestCoinGeckoClient_FetchPrices_InvalidJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("invalid json")); err != nil {
			return
		}
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := &config.Config{
		OverrideCoingeckoPublicURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
	}
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)
	client := NewCoinGeckoClient(cfg, metricsWriter)

	// Test fetching prices
	params := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}
	tokenData, err := client.FetchPrices(params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, tokenData)
	assert.False(t, client.Healthy())
}

func TestCoinGeckoClient_FetchPrices_ProKey(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		assert.Equal(t, "/api/v3/simple/price", r.URL.Path)
		assert.Equal(t, "bitcoin", r.URL.Query().Get("ids"))
		assert.Equal(t, "usd", r.URL.Query().Get("vs_currencies"))
		assert.Equal(t, "test-pro-key", r.URL.Query().Get("x_cg_pro_api_key"))

		// Return test response
		response := map[string]map[string]float64{
			"bitcoin": {
				"usd": 50000.0,
			},
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	// Create client with test server URL and Pro key
	cfg := &config.Config{
		OverrideCoingeckoProURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{"test-pro-key"},
		},
	}
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)
	client := NewCoinGeckoClient(cfg, metricsWriter)

	// Test fetching prices
	params := cg.PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd"},
	}
	tokenData, err := client.FetchPrices(params)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, tokenData)
	assert.True(t, client.Healthy())

	// Parse and verify bitcoin data
	var bitcoinData map[string]interface{}
	err = json.Unmarshal(tokenData["bitcoin"], &bitcoinData)
	assert.NoError(t, err)
	assert.Equal(t, 50000.0, bitcoinData["usd"])
}

func TestCoinGeckoClient_FetchPrices_WithMetadata(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters including optional ones
		assert.Equal(t, "/api/v3/simple/price", r.URL.Path)
		assert.Equal(t, "bitcoin", r.URL.Query().Get("ids"))
		assert.Equal(t, "usd", r.URL.Query().Get("vs_currencies"))
		assert.Equal(t, "true", r.URL.Query().Get("include_market_cap"))
		assert.Equal(t, "true", r.URL.Query().Get("include_24hr_vol"))
		assert.Equal(t, "true", r.URL.Query().Get("include_24hr_change"))
		assert.Equal(t, "true", r.URL.Query().Get("include_last_updated_at"))
		assert.Equal(t, "2", r.URL.Query().Get("precision"))

		// Return test response with metadata
		response := map[string]map[string]interface{}{
			"bitcoin": {
				"usd":             50000.0,
				"usd_market_cap":  950000000000.0,
				"usd_24h_vol":     25000000000.0,
				"usd_24h_change":  2.5,
				"last_updated_at": 1640995200,
			},
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := &config.Config{
		OverrideCoingeckoPublicURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
	}
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)
	client := NewCoinGeckoClient(cfg, metricsWriter)

	// Test fetching prices with all metadata
	params := cg.PriceParams{
		IDs:                  []string{"bitcoin"},
		Currencies:           []string{"usd"},
		IncludeMarketCap:     true,
		Include24hrVol:       true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
		Precision:            "2",
	}
	tokenData, err := client.FetchPrices(params)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, tokenData)
	assert.True(t, client.Healthy())

	// Parse and verify bitcoin data with metadata
	var bitcoinData map[string]interface{}
	err = json.Unmarshal(tokenData["bitcoin"], &bitcoinData)
	assert.NoError(t, err)
	assert.Equal(t, 50000.0, bitcoinData["usd"])
	assert.Equal(t, 950000000000.0, bitcoinData["usd_market_cap"])
	assert.Equal(t, 25000000000.0, bitcoinData["usd_24h_vol"])
	assert.Equal(t, 2.5, bitcoinData["usd_24h_change"])
	assert.Equal(t, float64(1640995200), bitcoinData["last_updated_at"])
}
