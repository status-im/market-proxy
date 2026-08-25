package currency_ratios

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(cfg *config.Config) *CoinGeckoClient {
	return NewCoinGeckoClient(cfg, metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios))
}

func TestNewCoinGeckoClient(t *testing.T) {
	cfg := &config.Config{
		APITokens: &config.APITokens{Tokens: []string{"test-token"}},
	}

	client := newTestClient(cfg)

	require.NotNil(t, client)
	assert.Same(t, cfg, client.config)
	assert.NotNil(t, client.httpClient, "HTTP client should be initialized")
	assert.NotNil(t, client.keyManager, "key manager should be initialized")
}

func TestCoinGeckoClient_Healthy(t *testing.T) {
	client := newTestClient(&config.Config{})

	assert.False(t, client.Healthy(), "client should be unhealthy before the first fetch")

	client.successfulFetch.Store(true)

	assert.True(t, client.Healthy(), "client should be healthy after a successful fetch")
}

// TestCoinGeckoClient_FetchReferencePrices checks the request the ratios service
// makes: one simple/price call carrying every configured currency, the 24h
// changes the honest percent conversion needs, and full precision.
func TestCoinGeckoClient_FetchReferencePrices(t *testing.T) {
	const body = `{"bitcoin":{"usd":100000,"usd_24h_change":10,"eur":90000,"eur_24h_change":20}}`

	var gotPath string
	var gotQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := newTestClient(&config.Config{
		APITokens:                  &config.APITokens{Tokens: []string{}},
		OverrideCoingeckoPublicURL: server.URL,
	})

	payload, err := client.FetchReferencePrices([]string{"bitcoin", "ethereum"}, []string{"usd", "eur"})
	require.NoError(t, err)

	assert.Equal(t, SIMPLE_PRICE_API_PATH, gotPath)
	assert.Equal(t, "bitcoin,ethereum", gotQuery.Get("ids"))
	assert.Equal(t, "usd,eur", gotQuery.Get("vs_currencies"))
	assert.Equal(t, "true", gotQuery.Get("include_24hr_change"))
	assert.Equal(t, "full", gotQuery.Get("precision"))

	var expected SimplePricePayload
	require.NoError(t, json.Unmarshal([]byte(body), &expected))
	assert.Equal(t, expected, payload)

	assert.True(t, client.Healthy())
}

func TestCoinGeckoClient_FetchReferencePrices_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := newTestClient(&config.Config{
		APITokens:                  &config.APITokens{Tokens: []string{}},
		OverrideCoingeckoPublicURL: server.URL,
	})

	payload, err := client.FetchReferencePrices([]string{"bitcoin"}, []string{"usd"})

	assert.Error(t, err)
	assert.Nil(t, payload)
	assert.False(t, client.Healthy())
}

func TestRatiosRequestBuilder(t *testing.T) {
	builder := NewRatiosRequestBuilder("https://example.com").
		WithIds([]string{"bitcoin", "ethereum"}).
		WithCurrencies([]string{"usd", "eur", "btc"}).
		WithInclude24hChange().
		WithFullPrecision()
	builder.WithApiKey("secret", cg.DemoKey)

	request, err := builder.Build()
	require.NoError(t, err)

	query := request.URL.Query()
	assert.Equal(t, SIMPLE_PRICE_API_PATH, request.URL.Path)
	assert.Equal(t, "bitcoin,ethereum", query.Get("ids"))
	assert.Equal(t, "usd,eur,btc", query.Get("vs_currencies"))
	assert.Equal(t, "true", query.Get("include_24hr_change"))
	assert.Equal(t, "full", query.Get("precision"))
	assert.Equal(t, "secret", query.Get("x_cg_demo_api_key"))
}
