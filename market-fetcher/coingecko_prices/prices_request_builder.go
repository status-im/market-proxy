package coingecko_prices

import (
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	// Complete path for simple price API endpoint
	PRICES_API_PATH = "/api/v3/simple/price"
)

// PricesRequestBuilder implements the Builder pattern for CoinGecko simple price API requests
type PricesRequestBuilder struct {
	*cg.CoingeckoRequestBuilder
}

// NewPricesRequestBuilder creates a new request builder for simple price endpoint
func NewPricesRequestBuilder(baseURL string) *PricesRequestBuilder {
	return &PricesRequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, PRICES_API_PATH),
	}
}

// WithIds adds coin IDs parameter
func (rb *PricesRequestBuilder) WithIds(ids []string) *PricesRequestBuilder {
	rb.With("ids", strings.Join(ids, ","))
	return rb
}

// WithCurrencies adds vs_currencies parameter
func (rb *PricesRequestBuilder) WithCurrencies(currencies []string) *PricesRequestBuilder {
	rb.With("vs_currencies", strings.Join(currencies, ","))
	return rb
}

// WithIncludeMarketCap adds include_market_cap parameter
func (rb *PricesRequestBuilder) WithIncludeMarketCap(include bool) *PricesRequestBuilder {
	if include {
		rb.With("include_market_cap", "true")
	}
	return rb
}

// WithInclude24hVolume adds include_24hr_vol parameter
func (rb *PricesRequestBuilder) WithInclude24hVolume(include bool) *PricesRequestBuilder {
	if include {
		rb.With("include_24hr_vol", "true")
	}
	return rb
}

// WithInclude24hChange adds include_24hr_change parameter
func (rb *PricesRequestBuilder) WithInclude24hChange(include bool) *PricesRequestBuilder {
	if include {
		rb.With("include_24hr_change", "true")
	}
	return rb
}

// WithIncludeLastUpdatedAt adds include_last_updated_at parameter
func (rb *PricesRequestBuilder) WithIncludeLastUpdatedAt(include bool) *PricesRequestBuilder {
	if include {
		rb.With("include_last_updated_at", "true")
	}
	return rb
}

// WithPrecision adds precision parameter for decimal places
func (rb *PricesRequestBuilder) WithPrecision(precision string) *PricesRequestBuilder {
	if precision != "" {
		rb.With("precision", precision)
	}
	return rb
}

// WithAllMetadata adds all metadata parameters (market cap, 24hr volume, 24hr change, last updated)
func (rb *PricesRequestBuilder) WithAllMetadata() *PricesRequestBuilder {
	return rb.WithIncludeMarketCap(true).
		WithInclude24hVolume(true).
		WithInclude24hChange(true).
		WithIncludeLastUpdatedAt(true)
}
