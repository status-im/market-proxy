package coingecko_leaderboard

import (
	markets "github.com/status-im/market-proxy/coingecko_markets"
)

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

// CoinData represents a cleaned CoinGecko coin with minimal fields for leaderboard
type CoinData struct {
	ID                       string  `json:"id"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	Image                    string  `json:"image"`
	CurrentPrice             float64 `json:"current_price"`
	MarketCap                float64 `json:"market_cap"`
	TotalVolume              float64 `json:"total_volume"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
}

// APIResponse represents the filtered response structure for leaderboard
type APIResponse struct {
	Data []CoinData `json:"data"`
}

// ConvertCoinGeckoData converts full CoinGecko data to minimal format for leaderboard
func ConvertCoinGeckoData(data []markets.CoinGeckoData) []CoinData {
	result := make([]CoinData, 0, len(data))

	for _, item := range data {
		coin := CoinData{
			ID:                       item.ID,
			Symbol:                   item.Symbol,
			Name:                     item.Name,
			Image:                    item.Image,
			CurrentPrice:             item.CurrentPrice,
			MarketCap:                item.MarketCap,
			TotalVolume:              item.TotalVolume,
			PriceChangePercentage24h: item.PriceChangePercentage24h,
		}

		result = append(result, coin)
	}

	return result
}
