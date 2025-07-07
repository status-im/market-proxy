package coingecko_market_chart

import (
	"log"
	"strconv"
)

// RoundUpMarketChartParams rounds up the days parameter to get maximum data:
// - If days <= dailyDataThreshold, round up to dailyDataThreshold
// - If days > dailyDataThreshold, round up to 365 days
// - If days == "max", keep "max"
func RoundUpMarketChartParams(params MarketChartParams, dailyDataThreshold int) MarketChartParams {
	roundedParams := params

	if params.Days == "" || params.Days == "max" {
		return roundedParams
	}

	daysInt, err := strconv.Atoi(params.Days)
	if err != nil {
		log.Printf("RoundUpMarketChartParams: Unable to parse days '%s' as integer, keeping original value", params.Days)
		return roundedParams
	}

	originalDays := params.Days
	if daysInt <= dailyDataThreshold {
		roundedParams.Days = strconv.Itoa(dailyDataThreshold)
	} else if daysInt > dailyDataThreshold {
		roundedParams.Days = "365"
	}

	if originalDays != roundedParams.Days {
		log.Printf("RoundUpMarketChartParams: Rounded up days from %s to %s for coin %s to get maximum data",
			originalDays, roundedParams.Days, params.ID)
	}

	return roundedParams
}

func RoundUpMarketChartParamsInPlace(params *MarketChartParams, dailyDataThreshold int) {
	rounded := RoundUpMarketChartParams(*params, dailyDataThreshold)
	*params = rounded
}
