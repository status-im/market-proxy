package interfaces

//go:generate mockgen -destination=mocks/coingecko_leaderboard.go . ILeaderboardService

// ILeaderboardService defines the interface for the leaderboard service: the
// proxy's own aggregated API over top-N tokens.
type ILeaderboardService interface {
	IHealthReporter

	// GetCacheData returns the cached top markets rows, converted to
	// convertCurrency when that is non-empty. nil means there is nothing to
	// serve: either the cache is empty or no Ratio is available yet.
	GetCacheData(convertCurrency string) *LeaderboardResponse

	// GetTopPricesQuotes returns cached price quotes for top tokens.
	// currency selects a Passthrough currency from the cache; convertCurrency,
	// when non-empty, instead returns an Estimate computed at request time from
	// the base currency rows. An empty result means no Ratio is available yet.
	GetTopPricesQuotes(currency string, convertCurrency string) LeaderboardQuotes
}

// LeaderboardQuote represents price data for one token in a single currency
// (matching the CoinMarketCap quote structure)
type LeaderboardQuote struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	MarketCap        float64 `json:"market_cap"`
	PercentChange24h float64 `json:"percent_change_24h"`
}

// LeaderboardQuotes maps a token ID to its quote data
type LeaderboardQuotes = map[string]LeaderboardQuote

// LeaderboardCoinData is a CoinGecko market row reduced to the fields the
// leaderboard serves
type LeaderboardCoinData struct {
	ID                       string  `json:"id"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	Image                    string  `json:"image"`
	CurrentPrice             float64 `json:"current_price"`
	MarketCap                float64 `json:"market_cap"`
	TotalVolume              float64 `json:"total_volume"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
}

// LeaderboardResponse is the response structure of /v1/leaderboard/markets
type LeaderboardResponse struct {
	Data []LeaderboardCoinData `json:"data"`
}
