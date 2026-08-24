package currency_ratios

import (
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	// SIMPLE_PRICE_API_PATH is the CoinGecko simple price endpoint
	SIMPLE_PRICE_API_PATH = "/api/v3/simple/price"
)

// RatiosRequestBuilder builds the single simple/price request the ratios are derived from
type RatiosRequestBuilder struct {
	*cg.CoingeckoRequestBuilder
}

// NewRatiosRequestBuilder creates a request builder for the reference coin prices
func NewRatiosRequestBuilder(baseURL string) *RatiosRequestBuilder {
	return &RatiosRequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, SIMPLE_PRICE_API_PATH),
	}
}

// WithIds adds the reference coin ids
func (rb *RatiosRequestBuilder) WithIds(ids []string) *RatiosRequestBuilder {
	rb.With("ids", strings.Join(ids, ","))
	return rb
}

// WithCurrencies adds the currencies ratios are computed for
func (rb *RatiosRequestBuilder) WithCurrencies(currencies []string) *RatiosRequestBuilder {
	rb.With("vs_currencies", strings.Join(currencies, ","))
	return rb
}

// WithInclude24hChange requests the 24h change fields needed for the 24h-ago ratio
func (rb *RatiosRequestBuilder) WithInclude24hChange() *RatiosRequestBuilder {
	rb.With("include_24hr_change", "true")
	return rb
}

// WithFullPrecision requests unrounded prices, so ratios stay accurate for
// currencies whose unit price is far from the base currency's
func (rb *RatiosRequestBuilder) WithFullPrecision() *RatiosRequestBuilder {
	rb.With("precision", "full")
	return rb
}
