package interfaces

import "github.com/status-im/market-proxy/events"

//go:generate mockgen -destination=mocks/coingecko_markets.go . IMarketsService

// IMarketsService defines the interface for CoinGecko markets service
type IMarketsService interface {
	// TopMarkets fetches top markets data for specified number of tokens,
	// caches individual tokens by their coingecko id and returns the response
	TopMarkets(limit int, currency string) (MarketsResponse, error)

	// TopMarketIds fetches top market token IDs for specified limit from cache
	TopMarketIds(limit int) ([]string, error)

	// Markets returns markets data for specified parameters
	Markets(params MarketsParams) (MarketsResponse, CacheStatus, error)

	// SubscribeTopMarketsUpdate subscribes to markets update notifications
	SubscribeTopMarketsUpdate() events.ISubscription

	// SubscribeInitialized subscribes to markets service initialization notifications
	SubscribeInitialized() events.ISubscription
}

// MarketsParams represents parameters for markets requests
type MarketsParams struct {
	// Currency to compare against (e.g., "usd", "eur", "btc")
	Currency string `json:"vs_currency"`

	// Order specifies sorting order (e.g., "market_cap_desc", "market_cap_asc", "volume_desc")
	Order string `json:"order"`

	// Page number for pagination (1-based)
	Page int `json:"page,omitempty"`

	// PerPage specifies number of results per page (1-250)
	PerPage int `json:"per_page,omitempty"`

	// Category filters by coin category
	Category string `json:"category,omitempty"`

	// IDs filters by specific coin IDs (comma-separated)
	IDs []string `json:"ids,omitempty"`

	// SparklineEnabled includes 7d price sparkline data
	SparklineEnabled bool `json:"sparkline,omitempty"`

	// PriceChangePercentage includes price change percentages for specific time periods
	PriceChangePercentage []string `json:"price_change_percentage,omitempty"`

	// IncludeRehypothecated includes rehypothecated tokens (e.g., stETH, WETH, WBTC) in the response
	IncludeRehypothecated bool `json:"include_rehypothecated,omitempty"`
}

// MarketsResponse represents markets data response structure
type MarketsResponse []interface{}
