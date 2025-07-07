package coingecko_markets

import (
	"strconv"
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	// Complete path for markets API endpoint
	MARKETS_API_PATH = "/api/v3/coins/markets"
)

// MarketsRequestBuilder implements the Builder pattern for CoinGecko markets API requests
type MarketsRequestBuilder struct {
	*cg.CoingeckoRequestBuilder
}

// NewMarketRequestBuilder creates a new request builder for markets endpoint
func NewMarketRequestBuilder(baseURL string) *MarketsRequestBuilder {
	rb := &MarketsRequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, MARKETS_API_PATH),
	}

	// Add default market parameters
	rb.WithCurrency("usd")
	rb.WithOrder("market_cap_desc")

	return rb
}

// WithPage adds page parameter for pagination
func (rb *MarketsRequestBuilder) WithPage(page int) *MarketsRequestBuilder {
	rb.With("page", strconv.Itoa(page))
	return rb
}

// WithPerPage adds per_page parameter
func (rb *MarketsRequestBuilder) WithPerPage(perPage int) *MarketsRequestBuilder {
	rb.With("per_page", strconv.Itoa(perPage))
	return rb
}

// WithOrder adds ordering parameter
func (rb *MarketsRequestBuilder) WithOrder(order string) *MarketsRequestBuilder {
	if order != "" {
		rb.With("order", order)
	}
	return rb
}

// WithCategory adds category parameter
func (rb *MarketsRequestBuilder) WithCategory(category string) *MarketsRequestBuilder {
	if category != "" {
		rb.With("category", category)
	}
	return rb
}

// WithIDs adds ids parameter (comma-separated list of coin IDs)
func (rb *MarketsRequestBuilder) WithIDs(ids []string) *MarketsRequestBuilder {
	if len(ids) > 0 {
		rb.With("ids", strings.Join(ids, ","))
	}
	return rb
}

// WithSparkline adds sparkline parameter
func (rb *MarketsRequestBuilder) WithSparkline(enabled bool) *MarketsRequestBuilder {
	if enabled {
		rb.With("sparkline", strconv.FormatBool(enabled))
	}

	return rb
}

// WithPriceChangePercentage adds price_change_percentage parameter
func (rb *MarketsRequestBuilder) WithPriceChangePercentage(percentages []string) *MarketsRequestBuilder {
	if len(percentages) > 0 {
		rb.With("price_change_percentage", strings.Join(percentages, ","))
	}
	return rb
}
