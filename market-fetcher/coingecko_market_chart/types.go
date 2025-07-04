package coingecko_market_chart

// MarketChartParams represents parameters for market chart requests
type MarketChartParams struct {
	// ID is the coin id (required) - can be obtained from /coins/list
	ID string `json:"id"`

	// Currency to compare against (e.g., "usd", "eur", "btc")
	Currency string `json:"vs_currency"`

	// Days specifies the data up to number of days ago (1/7/14/30/90/180/365/max)
	Days string `json:"days"`

	// Interval specifies data interval (only for Enterprise plan)
	// Valid values: "5m" (5-minutely), "hourly", "daily"
	// Leave empty for automatic granularity:
	// 1 day = 5-minutely data
	// 2-90 days = hourly data
	// above 90 days = daily data
	Interval string `json:"interval,omitempty"`

	// DataFilter specifies which data fields to include in response
	// Comma-separated list (e.g., "prices,market_caps")
	// Available fields: "prices", "market_caps", "total_volumes"
	// If empty, all fields are included
	DataFilter string `json:"data_filter,omitempty"`
}

// MarketChartData represents a single data point [timestamp, value]
type MarketChartData [2]float64

// MarketChartResponse represents the market chart API response structure
type MarketChartResponse struct {
	// Prices contains historical price data as [timestamp, price] pairs
	Prices []MarketChartData `json:"prices"`

	// MarketCaps contains historical market cap data as [timestamp, market_cap] pairs
	MarketCaps []MarketChartData `json:"market_caps"`

	// TotalVolumes contains historical volume data as [timestamp, total_volume] pairs
	TotalVolumes []MarketChartData `json:"total_volumes"`
}

// MarketChartAPIResponse represents a full API response with possible error handling
type MarketChartAPIResponse struct {
	Data  *MarketChartResponse `json:"data,omitempty"`
	Error string               `json:"error,omitempty"`
}
