package coingecko_leaderboard

import (
	"github.com/status-im/market-proxy/currency_ratios"
)

// scaled returns a pointer to value*ratio, or nil when the value is absent.
//
// An absent Passthrough value has no Estimate: there is nothing to convert, and
// inventing a zero would hand the client a number the provider never reported.
func scaled(value *float64, spotRatio float64) *float64 {
	if value == nil {
		return nil
	}
	converted := *value * spotRatio
	return &converted
}

// convertedPercent returns a pointer to the honestly converted 24h percent
// change, or nil when the change is absent.
//
// This is the case that makes the nil handling load-bearing rather than tidy:
// ConvertPercentChange24h of a substituted 0 is not 0, it is whatever the fx
// ratio moved by over the window, so a missing change would come back as a
// plausible-looking nonzero percentage.
func convertedPercent(pct *float64, ratio currency_ratios.Ratio) *float64 {
	if pct == nil {
		return nil
	}
	converted := currency_ratios.ConvertPercentChange24h(*pct, ratio)
	return &converted
}

// ConvertAPIResponse returns a copy of the markets response with every monetary
// field converted to the target currency using the spot ratio, and the 24h
// percent change converted honestly with both ratios.
//
// The input is Passthrough data owned by the cache and is never mutated: the
// result is an Estimate computed at request time.
//
// current_price and market_cap are measured at one instant, so the spot ratio is
// exact for them. price_change_percentage_24h spans a window with two known
// endpoints, and we know the Ratio at both, so it is converted honestly.
// total_volume is the odd one out: it is an integral of every trade over the
// window, so an honest conversion would need the exchange rate at each trade.
// That is not available, and CoinGecko itself reports non-usd volume by scaling
// with the current rate, so spot scaling is both the only option and the one
// that matches upstream.
func ConvertAPIResponse(response *APIResponse, ratio currency_ratios.Ratio) *APIResponse {
	if response == nil {
		return nil
	}

	converted := &APIResponse{
		Data: make([]CoinData, 0, len(response.Data)),
	}

	for _, coin := range response.Data {
		coin.CurrentPrice = scaled(coin.CurrentPrice, ratio.Now)
		coin.MarketCap = scaled(coin.MarketCap, ratio.Now)
		coin.TotalVolume = scaled(coin.TotalVolume, ratio.Now)
		coin.PriceChangePercentage24h = convertedPercent(coin.PriceChangePercentage24h, ratio)
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
		quote.Volume24h = scaled(quote.Volume24h, ratio.Now)
		quote.MarketCap = scaled(quote.MarketCap, ratio.Now)
		quote.PercentChange24h = convertedPercent(quote.PercentChange24h, ratio)
		converted[tokenID] = quote
	}

	return converted
}
