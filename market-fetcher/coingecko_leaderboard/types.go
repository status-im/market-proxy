package coingecko_leaderboard

// Original CoinGecko response structure

// Quote represents price data in a specific currency (matching CoinMarketCap structure)
type Quote struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	MarketCap        float64 `json:"market_cap"`
	PercentChange24h float64 `json:"percent_change_24h"`
}

// PriceQuotes maps symbol to its quote data
type PriceQuotes = map[string]Quote
