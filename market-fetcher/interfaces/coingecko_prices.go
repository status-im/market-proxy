package interfaces

import (
	"context"

	"github.com/status-im/market-proxy/events"
)

//go:generate mockgen -destination=mocks/coingecko_prices.go . IPricesService

// IPricesService interface for fetching prices of top tokens
type IPricesService interface {
	IHealthReporter

	// SimplePrices returns cached prices using PriceParams structure
	SimplePrices(ctx context.Context, params PriceParams) (SimplePriceResponse, CacheStatus, error)

	// TopPrices fetches prices for top tokens with specified limit and currencies
	// Similar to TopMarkets in markets service, provides clean interface for token price fetching
	TopPrices(ctx context.Context, limit int, currencies []string) (SimplePriceResponse, CacheStatus, error)

	// SubscribeTopPricesUpdate subscribes to prices update notifications
	SubscribeTopPricesUpdate() events.ISubscription
}

// PriceParams represents parameters for price requests
type PriceParams struct {
	// IDs list of token/coin IDs to fetch prices for
	IDs []string `json:"ids"`

	// Currencies list of target currencies (e.g., "usd", "eur")
	Currencies []string `json:"vs_currencies"`

	// Include additional data fields
	IncludeMarketCap     bool `json:"include_market_cap"`
	Include24hrVol       bool `json:"include_24hr_vol"`
	Include24hrChange    bool `json:"include_24hr_change"`
	IncludeLastUpdatedAt bool `json:"include_last_updated_at"`

	// Precision for decimal places (empty means full precision)
	Precision string `json:"precision,omitempty"`

	// ConvertCurrency, when set, adds realtime Estimate keys for that currency
	// alongside the Passthrough keys selected by Currencies. It must not also
	// appear in Currencies - one key cannot be both. An empty response means no
	// Ratio is available yet.
	ConvertCurrency string `json:"convert_currency,omitempty"`
}

// SimplePriceResponse represents the response format compatible with CoinGecko simple/price API
// This is the raw JSON structure that CoinGecko returns and what we store in cache
type SimplePriceResponse map[string]interface{}
