package coingecko_leaderboard

import (
	"github.com/status-im/market-proxy/currency_ratios"
)

// ConvertAPIResponse returns a copy of the markets response with every monetary
// field converted to the target currency using the spot ratio, and the 24h
// percent change converted honestly with both ratios.
//
// The input is Passthrough data owned by the cache and is never mutated: the
// result is an Estimate computed at request time.
func ConvertAPIResponse(response *APIResponse, ratio currency_ratios.Ratio) *APIResponse {
	if response == nil {
		return nil
	}

	converted := &APIResponse{
		Data: make([]CoinData, 0, len(response.Data)),
	}

	for _, coin := range response.Data {
		coin.CurrentPrice *= ratio.Now
		coin.MarketCap *= ratio.Now
		coin.TotalVolume *= ratio.Now
		coin.PriceChangePercentage24h = currency_ratios.ConvertPercentChange24h(coin.PriceChangePercentage24h, ratio)
		converted.Data = append(converted.Data, coin)
	}

	return converted
}

// ConvertQuotes returns a copy of the price quotes converted to the target
// currency. The input map is never mutated.
//
// Price and MarketCap are point-in-time amounts and scale exactly with the spot
// ratio; PercentChange24h has two known endpoints in time and is converted
// honestly with both ratios. Volume24h is spot-scaled for the reason given on
// ConvertAPIResponse: converting it honestly would need the rate at each trade.
func ConvertQuotes(quotes map[string]Quote, ratio currency_ratios.Ratio) map[string]Quote {
	converted := make(map[string]Quote, len(quotes))

	for tokenID, quote := range quotes {
		quote.Price *= ratio.Now
		quote.Volume24h *= ratio.Now
		quote.MarketCap *= ratio.Now
		quote.PercentChange24h = currency_ratios.ConvertPercentChange24h(quote.PercentChange24h, ratio)
		converted[tokenID] = quote
	}

	return converted
}
