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
	assert.NotNil(t, builder.builder)
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
