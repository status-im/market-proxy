package coingecko_common

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
}

// MarketsResponse represents markets data response structure
type MarketsResponse []interface{}

// MarketsFetcher interface for fetching markets data
type MarketsFetcher interface {
	// Markets returns markets data for specified parameters
	Markets(params MarketsParams) (MarketsResponse, error)

	// TopMarkets fetches top markets data for specified number of tokens,
	// caches individual tokens by their coingecko id and returns the response
	TopMarkets(limit int, currency string) (MarketsResponse, error)
}
