package coingecko_prices

import (
	"net/http"
	"net/url"
	"testing"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/stretchr/testify/assert"
)

func TestNewPricesRequestBuilder(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	builder := NewPricesRequestBuilder(baseURL)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.CoingeckoRequestBuilder)
}

func TestPricesRequestBuilder_WithIds(t *testing.T) {
	builder := NewPricesRequestBuilder("https://api.coingecko.com")
	ids := []string{"bitcoin", "ethereum"}

	builder.WithIds(ids)
	builtURL := builder.BuildURL()
	parsedURL, err := url.Parse(builtURL)
	assert.NoError(t, err)

	query := parsedURL.Query()
	assert.Equal(t, "bitcoin,ethereum", query.Get("ids"))
}

func TestPricesRequestBuilder_WithCurrencies(t *testing.T) {
	builder := NewPricesRequestBuilder("https://api.coingecko.com")
	currencies := []string{"usd", "eur"}

	builder.WithCurrencies(currencies)
	builtURL := builder.BuildURL()
	parsedURL, err := url.Parse(builtURL)
	assert.NoError(t, err)

	query := parsedURL.Query()
	assert.Equal(t, "usd,eur", query.Get("vs_currencies"))
}

func TestPricesRequestBuilder_WithApiKey(t *testing.T) {
	builder := NewPricesRequestBuilder("https://api.coingecko.com")
	apiKey := "test-api-key"

	builder.WithApiKey(apiKey, cg.ProKey)
	key, keyType := builder.GetApiKey()

	assert.Equal(t, apiKey, key)
	assert.Equal(t, cg.ProKey, keyType)
}

func TestPricesRequestBuilder_WithHeader(t *testing.T) {
	builder := NewPricesRequestBuilder("https://api.coingecko.com")
	headerName := "X-Custom-Header"
	headerValue := "test-value"

	builder.WithHeader(headerName, headerValue)
	req, err := builder.Build()

	assert.NoError(t, err)
	assert.Equal(t, headerValue, req.Header.Get(headerName))
}

func TestPricesRequestBuilder_WithUserAgent(t *testing.T) {
	builder := NewPricesRequestBuilder("https://api.coingecko.com")
	userAgent := "test-user-agent"

	builder.WithUserAgent(userAgent)
	req, err := builder.Build()

	assert.NoError(t, err)
	assert.Equal(t, userAgent, req.Header.Get("User-Agent"))
}

func TestPricesRequestBuilder_Build(t *testing.T) {
	builder := NewPricesRequestBuilder("https://api.coingecko.com")
	ids := []string{"bitcoin", "ethereum"}
	currencies := []string{"usd", "eur"}

	builder.WithIds(ids).WithCurrencies(currencies)
	req, err := builder.Build()

	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, req.Method)

	query := req.URL.Query()
	assert.Equal(t, "bitcoin,ethereum", query.Get("ids"))
	assert.Equal(t, "usd,eur", query.Get("vs_currencies"))
}

func TestPricesRequestBuilder_BuildURL(t *testing.T) {
	builder := NewPricesRequestBuilder("https://api.coingecko.com")
	ids := []string{"bitcoin", "ethereum"}
	currencies := []string{"usd", "eur"}

	builtURL := builder.WithIds(ids).WithCurrencies(currencies).BuildURL()
	parsedURL, err := url.Parse(builtURL)
	assert.NoError(t, err)

	assert.Equal(t, "https://api.coingecko.com", parsedURL.Scheme+"://"+parsedURL.Host)
	assert.Equal(t, PRICES_API_PATH, parsedURL.Path)

	query := parsedURL.Query()
	assert.Equal(t, "bitcoin,ethereum", query.Get("ids"))
	assert.Equal(t, "usd,eur", query.Get("vs_currencies"))
}

func TestPricesRequestBuilder_WithMetadataParameters(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	builder := NewPricesRequestBuilder(baseURL)

	// Test individual metadata parameters
	builder.WithIds([]string{"bitcoin", "ethereum"}).
		WithCurrencies([]string{"usd", "eur"}).
		WithIncludeMarketCap(true).
		WithInclude24hVolume(true).
		WithInclude24hChange(true).
		WithIncludeLastUpdatedAt(true)

	url := builder.BuildURL()

	// Verify all parameters are included (note: comma is URL encoded as %2C)
	assert.Contains(t, url, "ids=bitcoin%2Cethereum")
	assert.Contains(t, url, "vs_currencies=usd%2Ceur")
	assert.Contains(t, url, "include_market_cap=true")
	assert.Contains(t, url, "include_24hr_vol=true")
	assert.Contains(t, url, "include_24hr_change=true")
	assert.Contains(t, url, "include_last_updated_at=true")
}

func TestPricesRequestBuilder_WithAllMetadata(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	builder := NewPricesRequestBuilder(baseURL)

	// Test convenience method for all metadata
	builder.WithIds([]string{"bitcoin"}).
		WithCurrencies([]string{"usd"}).
		WithAllMetadata()

	url := builder.BuildURL()

	// Verify all metadata parameters are included
	assert.Contains(t, url, "include_market_cap=true")
	assert.Contains(t, url, "include_24hr_vol=true")
	assert.Contains(t, url, "include_24hr_change=true")
	assert.Contains(t, url, "include_last_updated_at=true")
}

func TestPricesRequestBuilder_WithMetadataFalse(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	builder := NewPricesRequestBuilder(baseURL)

	// Test that false parameters are not added
	builder.WithIds([]string{"bitcoin"}).
		WithCurrencies([]string{"usd"}).
		WithIncludeMarketCap(false).
		WithInclude24hVolume(false).
		WithInclude24hChange(false).
		WithIncludeLastUpdatedAt(false)

	url := builder.BuildURL()

	// Verify metadata parameters are NOT included when set to false
	assert.NotContains(t, url, "include_market_cap")
	assert.NotContains(t, url, "include_24hr_vol")
	assert.NotContains(t, url, "include_24hr_change")
	assert.NotContains(t, url, "include_last_updated_at")
}
