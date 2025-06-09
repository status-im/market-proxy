package coingecko_leaderboard

import (
	cg "github.com/status-im/market-proxy/coingecko_common"
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
		hasValidPrice := false

		// Extract price for the currency - this is required
		if priceValue, exists := tokenData[currency]; exists {
			if price, ok := priceValue.(float64); ok && price > 0 {
				quote.Price = price
				hasValidPrice = true
			}
		}

		// Only continue processing if we have a valid price
		if !hasValidPrice {
			continue
		}

		// Extract market cap for the currency
		marketCapKey := currency + "_market_cap"
		if marketCapValue, exists := tokenData[marketCapKey]; exists {
			if marketCap, ok := marketCapValue.(float64); ok {
				quote.MarketCap = marketCap
			}
		}

		// Extract 24h volume for the currency
		volume24hKey := currency + "_24h_vol"
		if volume24hValue, exists := tokenData[volume24hKey]; exists {
			if volume24h, ok := volume24hValue.(float64); ok {
				quote.Volume24h = volume24h
			}
		}

		// Extract 24h change for the currency
		change24hKey := currency + "_24h_change"
		if change24hValue, exists := tokenData[change24hKey]; exists {
			if change24h, ok := change24hValue.(float64); ok {
				quote.PercentChange24h = change24h
			}
		}

		// Add the quote since it has a valid price
		currencyQuotes[tokenID] = quote
	}

	return currencyQuotes
}
