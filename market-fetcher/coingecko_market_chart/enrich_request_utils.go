package coingecko_market_chart

import (
	"log"
	"strconv"
)

// EnrichMarketChartParams enriches the request parameters to get maximum data
// by rounding up the days parameter according to CoinGecko's data availability and provided config:
// - If days <= dailyDataThreshold, round up to dailyDataThreshold
// - If days > dailyDataThreshold, round up to 365 days
// - If days == "max", keep "max"
func EnrichMarketChartParams(params MarketChartParams, dailyDataThreshold int) MarketChartParams {
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

	// Apply enrichment logic using config
	originalDays := params.Days
	if daysInt <= dailyDataThreshold {
		enrichedParams.Days = strconv.Itoa(dailyDataThreshold)
	} else if daysInt > dailyDataThreshold {
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
func EnrichMarketChartParamsInPlace(params *MarketChartParams, dailyDataThreshold int) {
	enriched := EnrichMarketChartParams(*params, dailyDataThreshold)
	*params = enriched
}
