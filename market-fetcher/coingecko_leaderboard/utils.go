package coingecko_leaderboard

import (
	cg "github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/jsonutil"
)

// ConvertPriceResponseToPriceQuotes converts SimplePriceResponse to PriceQuotes for the given currency
// Only includes tokens that have a valid price (> 0)
//
// Fields the provider did not report stay nil rather than becoming zero: the
// per-currency market cap, volume and 24h change keys only exist when the
// upstream request asked for them, and a substituted zero would survive into a
// currency conversion as a fabricated value.
func ConvertPriceResponseToPriceQuotes(priceResponse cg.SimplePriceResponse, currency string) PriceQuotes {
	currencyQuotes := make(PriceQuotes)

	for tokenID, tokenDataInterface := range priceResponse {
		tokenData, ok := tokenDataInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract price for the currency - this is required
		price, ok := jsonutil.FloatField(tokenData, currency)
		if !ok || price <= 0 {
			continue // Only continue processing if we have a valid price
		}

		currencyQuotes[tokenID] = Quote{
			Price:            price,
			MarketCap:        optionalFloatField(tokenData, currency+"_market_cap"),
			Volume24h:        optionalFloatField(tokenData, currency+"_24h_vol"),
			PercentChange24h: optionalFloatField(tokenData, currency+"_24h_change"),
		}
	}

	return currencyQuotes
}

// ConvertMarketsResponseToCoinData converts raw markets response data to CoinData slice
// This function directly processes the interface{} slice from coins/markets API
//
// As above, a field CoinGecko returned as null stays nil instead of becoming a
// zero that later reads as real data.
func ConvertMarketsResponseToCoinData(marketsData []interface{}) []CoinData {
	result := make([]CoinData, 0, len(marketsData))

	for _, item := range marketsData {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		coinData := CoinData{
			ID:                       getStringFromMap(itemMap, "id"),
			Symbol:                   getStringFromMap(itemMap, "symbol"),
			Name:                     getStringFromMap(itemMap, "name"),
			Image:                    getStringFromMap(itemMap, "image"),
			CurrentPrice:             optionalFloatField(itemMap, "current_price"),
			MarketCap:                optionalFloatField(itemMap, "market_cap"),
			TotalVolume:              optionalFloatField(itemMap, "total_volume"),
			PriceChangePercentage24h: optionalFloatField(itemMap, "price_change_percentage_24h"),
		}

		result = append(result, coinData)
	}

	return result
}

// optionalFloatField returns a pointer to the field's value, or nil when the
// field is missing, null or not numeric
func optionalFloatField(m map[string]interface{}, key string) *float64 {
	value, ok := jsonutil.FloatField(m, key)
	if !ok {
		return nil
	}
	return &value
}

// getStringFromMap extracts a string field, falling back to "" when it is
// missing or not a string.
//
// Identity fields are not optional in the same way a price is: an empty id or
// symbol is as useless as an absent one, and the response shape depends on
// them always being present.
func getStringFromMap(m map[string]interface{}, key string) string {
	value, _ := jsonutil.StringField(m, key)
	return value
}
