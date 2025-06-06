package coingecko_prices

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

func TestNewCoinGeckoClient(t *testing.T) {
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{"test-key"},
		},
	}

	client := NewCoinGeckoClient(cfg)
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
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := &config.Config{
		OverrideCoingeckoPublicURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
	}
	client := NewCoinGeckoClient(cfg)

	// Test fetching prices
	params := PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}
	prices, err := client.FetchPrices(params)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, prices)
	assert.True(t, client.Healthy())

	// Verify price structure (currency first, then token)
	assert.Equal(t, 2, len(prices))        // 2 currencies
	assert.Equal(t, 2, len(prices["usd"])) // 2 tokens in USD
	assert.Equal(t, 2, len(prices["eur"])) // 2 tokens in EUR

	// Verify specific prices
	assert.Equal(t, 50000.0, prices["usd"]["bitcoin"])
	assert.Equal(t, 3000.0, prices["usd"]["ethereum"])
	assert.Equal(t, 45000.0, prices["eur"]["bitcoin"])
	assert.Equal(t, 2700.0, prices["eur"]["ethereum"])
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
	client := NewCoinGeckoClient(cfg)

	// Test fetching prices
	params := PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}
	prices, err := client.FetchPrices(params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, prices)
	assert.False(t, client.Healthy())
}

func TestCoinGeckoClient_FetchPrices_InvalidJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := &config.Config{
		OverrideCoingeckoPublicURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
	}
	client := NewCoinGeckoClient(cfg)

	// Test fetching prices
	params := PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}
	prices, err := client.FetchPrices(params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, prices)
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
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test server URL and Pro key
	cfg := &config.Config{
		OverrideCoingeckoProURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{"test-pro-key"},
		},
	}
	client := NewCoinGeckoClient(cfg)

	// Test fetching prices
	params := PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd"},
	}
	prices, err := client.FetchPrices(params)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, prices)
	assert.True(t, client.Healthy())
	assert.Equal(t, 50000.0, prices["usd"]["bitcoin"])
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
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := &config.Config{
		OverrideCoingeckoPublicURL: server.URL,
		APITokens: &config.APITokens{
			Tokens: []string{},
		},
	}
	client := NewCoinGeckoClient(cfg)

	// Test fetching prices with all metadata
	params := PriceParams{
		IDs:                  []string{"bitcoin"},
		Currencies:           []string{"usd"},
		IncludeMarketCap:     true,
		Include24hrVol:       true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
		Precision:            "2",
	}
	prices, err := client.FetchPrices(params)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, prices)
	assert.True(t, client.Healthy())
	assert.Equal(t, 50000.0, prices["usd"]["bitcoin"])
}
