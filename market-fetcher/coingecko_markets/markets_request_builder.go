package coingecko_markets

import (
	"net/http"
	"strconv"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	// Complete path for markets API endpoint
	MARKETS_API_PATH = "/api/v3/coins/markets"
)

// MarketsRequestBuilder implements the Builder pattern for CoinGecko markets API requests
type MarketsRequestBuilder struct {
	// Composition with base request builder
	builder *cg.CoingeckoRequestBuilder
}

// NewMarketRequestBuilder creates a new request builder for markets endpoint
func NewMarketRequestBuilder(baseURL string) *MarketsRequestBuilder {
	rb := &MarketsRequestBuilder{
		builder: cg.NewCoingeckoRequestBuilder(baseURL, MARKETS_API_PATH),
	}

	// Add default market parameters
	rb.WithCurrency("usd")
	rb.WithOrder("market_cap_desc")

	return rb
}

// WithPage adds page parameter for pagination
func (rb *MarketsRequestBuilder) WithPage(page int) *MarketsRequestBuilder {
	rb.builder.With("page", strconv.Itoa(page))
	return rb
}

// WithPerPage adds per_page parameter
func (rb *MarketsRequestBuilder) WithPerPage(perPage int) *MarketsRequestBuilder {
	rb.builder.With("per_page", strconv.Itoa(perPage))
	return rb
}

// WithCurrency adds currency parameter
func (rb *MarketsRequestBuilder) WithCurrency(currency string) *MarketsRequestBuilder {
	rb.builder.With("vs_currency", currency)
	return rb
}

// WithOrder adds ordering parameter
func (rb *MarketsRequestBuilder) WithOrder(order string) *MarketsRequestBuilder {
	rb.builder.With("order", order)
	return rb
}

// WithCategory adds category parameter
func (rb *MarketsRequestBuilder) WithCategory(category string) *MarketsRequestBuilder {
	rb.builder.With("category", category)
	return rb
}

// WithIDs adds ids parameter (comma-separated list of coin IDs)
func (rb *MarketsRequestBuilder) WithIDs(ids []string) *MarketsRequestBuilder {
	if len(ids) > 0 {
		// Join the IDs with commas
		idsStr := ""
		for i, id := range ids {
			if i > 0 {
				idsStr += ","
			}
			idsStr += id
		}
		rb.builder.With("ids", idsStr)
	}
	return rb
}

// WithSparkline adds sparkline parameter
func (rb *MarketsRequestBuilder) WithSparkline(enabled bool) *MarketsRequestBuilder {
	rb.builder.With("sparkline", strconv.FormatBool(enabled))
	return rb
}

// WithPriceChangePercentage adds price_change_percentage parameter
func (rb *MarketsRequestBuilder) WithPriceChangePercentage(percentages []string) *MarketsRequestBuilder {
	if len(percentages) > 0 {
		// Join the percentages with commas
		percentagesStr := ""
		for i, percentage := range percentages {
			if i > 0 {
				percentagesStr += ","
			}
			percentagesStr += percentage
		}
		rb.builder.With("price_change_percentage", percentagesStr)
	}
	return rb
}

// WithApiKey sets the API key and its type (delegated to base builder)
func (rb *MarketsRequestBuilder) WithApiKey(apiKey string, keyType cg.KeyType) *MarketsRequestBuilder {
	rb.builder.WithApiKey(apiKey, keyType)
	return rb
}

// WithHeader adds a custom HTTP header (delegated to base builder)
func (rb *MarketsRequestBuilder) WithHeader(name, value string) *MarketsRequestBuilder {
	rb.builder.WithHeader(name, value)
	return rb
}

// WithUserAgent sets the User-Agent header (delegated to base builder)
func (rb *MarketsRequestBuilder) WithUserAgent(userAgent string) *MarketsRequestBuilder {
	rb.builder.WithUserAgent(userAgent)
	return rb
}

// GetApiKey returns the API key and its type (delegated to base builder)
func (rb *MarketsRequestBuilder) GetApiKey() (string, cg.KeyType) {
	return rb.builder.GetApiKey()
}

// BuildURL builds the complete URL for the request (delegated to base builder)
func (rb *MarketsRequestBuilder) BuildURL() string {
	return rb.builder.BuildURL()
}

// Build creates an http.Request object (delegated to base builder)
func (rb *MarketsRequestBuilder) Build() (*http.Request, error) {
	return rb.builder.Build()
}
