package coingecko_leaderboard

import (
	cg "github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/jsonutil"
)

// ConvertPriceResponseToPriceQuotes converts SimplePriceResponse to PriceQuotes for the given currency
// Only includes tokens that have a valid price (> 0)
func ConvertPriceResponseToPriceQuotes(priceResponse cg.SimplePriceResponse, currency string) PriceQuotes {
	currencyQuotes := make(PriceQuotes)

	for tokenID, tokenDataInterface := range priceResponse {
		tokenData, ok := tokenDataInterface.(map[string]interface{})
		if !ok {
			continue
		}

		quote := Quote{}

		// Extract price for the currency - this is required
		price := getFloatFromMap(tokenData, currency)
		if price <= 0 {
			continue // Only continue processing if we have a valid price
		}
		quote.Price = price

		// Extract market cap for the currency
		marketCapKey := currency + "_market_cap"
		quote.MarketCap = getFloatFromMap(tokenData, marketCapKey)

		// Extract 24h volume for the currency
		volume24hKey := currency + "_24h_vol"
		quote.Volume24h = getFloatFromMap(tokenData, volume24hKey)

		// Extract 24h change for the currency
		change24hKey := currency + "_24h_change"
		quote.PercentChange24h = getFloatFromMap(tokenData, change24hKey)

		// Add the quote since it has a valid price
		currencyQuotes[tokenID] = quote
	}

	return currencyQuotes
}

// ConvertMarketsResponseToCoinData converts raw markets response data to CoinData slice
// This function directly processes the interface{} slice from coins/markets API
func ConvertMarketsResponseToCoinData(marketsData []interface{}) []CoinData {
	result := make([]CoinData, 0, len(marketsData))

	for _, item := range marketsData {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Convert map[string]interface{} to CoinData directly
		coinData := CoinData{
			ID:                       getStringFromMap(itemMap, "id"),
			Symbol:                   getStringFromMap(itemMap, "symbol"),
			Name:                     getStringFromMap(itemMap, "name"),
			Image:                    getStringFromMap(itemMap, "image"),
			CurrentPrice:             getFloatFromMap(itemMap, "current_price"),
			MarketCap:                getFloatFromMap(itemMap, "market_cap"),
			TotalVolume:              getFloatFromMap(itemMap, "total_volume"),
			PriceChangePercentage24h: getFloatFromMap(itemMap, "price_change_percentage_24h"),
		}

		result = append(result, coinData)
	}

	return result
}

// getStringFromMap extracts a string field, falling back to "" when it is
// missing or not a string.
//
// The fallback makes a missing field indistinguishable from an empty one. That
// is the long-standing behaviour of this conversion and the response shape
// depends on it, so the collapse is done here explicitly rather than changed.
func getStringFromMap(m map[string]interface{}, key string) string {
	value, _ := jsonutil.StringField(m, key)
	return value
}

// getFloatFromMap extracts a float64 field, falling back to 0 when it is
// missing or not numeric.
//
// As above: a missing price and a real zero come out the same. Preserved
// deliberately - changing it would change what these endpoints return.
func getFloatFromMap(m map[string]interface{}, key string) float64 {
	value, _ := jsonutil.FloatField(m, key)
	return value
}
