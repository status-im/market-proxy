package coingecko_market_chart

import (
	"log"
	"strconv"
)

// EnrichMarketChartParams enriches the request parameters to get maximum data
// by rounding up the days parameter according to CoinGecko's data availability:
// - If days <= 90, round up to 90 days
// - If days > 90, round up to 365 days
// - If days == "max", keep "max"
func EnrichMarketChartParams(params MarketChartParams) MarketChartParams {
	// Create a copy of the params to avoid modifying the original
	enrichedParams := params

	// If Days is empty or already "max", don't modify
	if params.Days == "" || params.Days == "max" {
		return enrichedParams
	}

	// Parse the days as integer
	daysInt, err := strconv.Atoi(params.Days)
	if err != nil {
		log.Printf("EnrichMarketChartParams: Unable to parse days '%s' as integer, keeping original value", params.Days)
		return enrichedParams
	}

	// Apply enrichment logic
	originalDays := params.Days
	if daysInt <= 90 {
		enrichedParams.Days = "90"
	} else if daysInt > 90 {
		enrichedParams.Days = "365"
	}

	// Log the enrichment if days were changed
	if originalDays != enrichedParams.Days {
		log.Printf("EnrichMarketChartParams: Enriched days from %s to %s for coin %s to get maximum data",
			originalDays, enrichedParams.Days, params.ID)
	}

	return enrichedParams
}

// EnrichMarketChartParamsInPlace enriches the request parameters in place
// This is a convenience function that modifies the original params struct
func EnrichMarketChartParamsInPlace(params *MarketChartParams) {
	enriched := EnrichMarketChartParams(*params)
	*params = enriched
}
