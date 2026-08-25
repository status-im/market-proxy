package coingecko_leaderboard

import (
	markets "github.com/status-im/market-proxy/coingecko_markets"
	"github.com/status-im/market-proxy/interfaces"
)

// The canonical definitions of the leaderboard response types live in the
// interfaces package, next to the ILeaderboardService contract that returns
// them. These aliases keep the package-local names readable inside the service.
type (
	// Quote represents price data in a specific currency (matching CoinMarketCap structure)
	Quote = interfaces.LeaderboardQuote

	// PriceQuotes maps token ID to its quote data
	PriceQuotes = interfaces.LeaderboardQuotes

	// CoinData represents a cleaned CoinGecko coin with minimal fields for leaderboard
	CoinData = interfaces.LeaderboardCoinData

	// APIResponse represents the filtered response structure for leaderboard
	APIResponse = interfaces.LeaderboardResponse
)

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
