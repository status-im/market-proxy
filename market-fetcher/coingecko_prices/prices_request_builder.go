package coingecko_prices

import (
	"net/http"
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	// Complete path for simple price API endpoint
	PRICES_API_PATH = "/api/v3/simple/price"
)

// PricesRequestBuilder implements the Builder pattern for CoinGecko simple price API requests
type PricesRequestBuilder struct {
	// Composition with base request builder
	builder *cg.CoingeckoRequestBuilder
}

// NewPricesRequestBuilder creates a new request builder for simple price endpoint
func NewPricesRequestBuilder(baseURL string) *PricesRequestBuilder {
	return &PricesRequestBuilder{
		builder: cg.NewCoingeckoRequestBuilder(baseURL, PRICES_API_PATH),
	}
}

// WithIds adds coin IDs parameter
func (rb *PricesRequestBuilder) WithIds(ids []string) *PricesRequestBuilder {
	rb.builder.With("ids", strings.Join(ids, ","))
	return rb
}

// WithCurrencies adds vs_currencies parameter
func (rb *PricesRequestBuilder) WithCurrencies(currencies []string) *PricesRequestBuilder {
	rb.builder.With("vs_currencies", strings.Join(currencies, ","))
	return rb
}

// WithApiKey sets the API key and its type (delegated to base builder)
func (rb *PricesRequestBuilder) WithApiKey(apiKey string, keyType cg.KeyType) *PricesRequestBuilder {
	rb.builder.WithApiKey(apiKey, keyType)
	return rb
}

// WithHeader adds a custom HTTP header (delegated to base builder)
func (rb *PricesRequestBuilder) WithHeader(name, value string) *PricesRequestBuilder {
	rb.builder.WithHeader(name, value)
	return rb
}

// WithUserAgent sets the User-Agent header (delegated to base builder)
func (rb *PricesRequestBuilder) WithUserAgent(userAgent string) *PricesRequestBuilder {
	rb.builder.WithUserAgent(userAgent)
	return rb
}

// GetApiKey returns the API key and its type (delegated to base builder)
func (rb *PricesRequestBuilder) GetApiKey() (string, cg.KeyType) {
	return rb.builder.GetApiKey()
}

// BuildURL builds the complete URL for the request (delegated to base builder)
func (rb *PricesRequestBuilder) BuildURL() string {
	return rb.builder.BuildURL()
}

// Build creates an http.Request object (delegated to base builder)
func (rb *PricesRequestBuilder) Build() (*http.Request, error) {
	return rb.builder.Build()
}
