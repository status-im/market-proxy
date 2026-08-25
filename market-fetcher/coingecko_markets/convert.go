package coingecko_markets

import (
	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/jsonutil"
)

// Field names of a CoinGecko /coins/markets row, grouped by how a currency
// conversion has to treat them.
const (
	fieldCurrentPrice   = "current_price"
	fieldMarketCap      = "market_cap"
	fieldSparkline7d    = "sparkline_in_7d"
	fieldSparklinePrice = "price"
	// PercentChange1hField is the 1h change field, converted with the ratio from
	// one hour ago rather than the 24h-ago one
	PercentChange1hField = "price_change_percentage_1h_in_currency"
)

// spotScaledFields are amounts of money at a single point in time: multiplying
// by the spot ratio is exact.
var spotScaledFields = []string{
	fieldCurrentPrice,
	fieldMarketCap,
	"fully_diluted_valuation",
	"total_volume",
	"high_24h",
	"low_24h",
	"ath",
	"atl",
}

// absolute24hDeltaFields map an absolute 24h delta to the field holding the
// current value it was measured against. Both ends of the delta are converted
// with the ratio that applied at that end.
var absolute24hDeltaFields = map[string]string{
	"price_change_24h":      fieldCurrentPrice,
	"market_cap_change_24h": fieldMarketCap,
}

// percentChange24hFields are percent changes over the 24h window, converted with
// the spot and the 24h-ago ratio.
var percentChange24hFields = []string{
	"price_change_percentage_24h",
	"market_cap_change_percentage_24h",
	"price_change_percentage_24h_in_currency",
}

// MarketsConversion carries the ratios needed to convert one markets row.
type MarketsConversion struct {
	// Ratio holds the spot and 24h-ago ratios of the target currency
	Ratio currency_ratios.Ratio

	// Spot1hAgo is the target currency's spot ratio one hour ago
	Spot1hAgo float64

	// Has1hAgo is false when the ratio history does not reach back one hour yet;
	// the 1h percent change is then left unconverted.
	Has1hAgo bool
}

// ConvertMarketsResponse returns a copy of a /coins/markets response with money
// denominated fields converted to the target currency.
//
// The input rows are Passthrough data and are never mutated: every row is
// rebuilt as a fresh map, so the result is an Estimate computed at request time.
//
// Fields that are not money (ranks, supplies, dates, identifiers) pass through
// unchanged. ath_change_percentage and atl_change_percentage also pass through:
// converting them honestly would need the exchange rate as it stood on the ath /
// atl date, which the realtime ratio history does not carry.
// Percent changes over windows longer than 24h (7d, 14d, 30d, 200d, 1y) pass
// through for the same reason.
func ConvertMarketsResponse(response interfaces.MarketsResponse, conversion MarketsConversion) interfaces.MarketsResponse {
	converted := make([]interface{}, 0, len(response))

	for _, item := range response {
		row, ok := item.(map[string]interface{})
		if !ok {
			// Not a shape we understand - pass it through untouched
			converted = append(converted, item)
			continue
		}
		converted = append(converted, convertMarketsRow(row, conversion))
	}

	return interfaces.MarketsResponse(converted)
}

// convertMarketsRow builds a converted copy of a single markets row
func convertMarketsRow(row map[string]interface{}, conversion MarketsConversion) map[string]interface{} {
	ratio := conversion.Ratio
	result := make(map[string]interface{}, len(row))

	// Start from an untouched copy so non-money fields keep their exact values
	for key, value := range row {
		result[key] = value
	}

	for _, field := range spotScaledFields {
		if value, ok := jsonutil.Float(row[field]); ok {
			result[field] = value * ratio.Now
		}
	}

	for field, baseField := range absolute24hDeltaFields {
		delta, ok := jsonutil.Float(row[field])
		if !ok {
			continue
		}
		// The absolute delta needs the *base currency* current value, which is
		// read from the untouched row rather than the already-converted result.
		baseValue, ok := jsonutil.Float(row[baseField])
		if !ok {
			continue
		}
		result[field] = currency_ratios.ConvertAbsoluteChange24h(baseValue, delta, ratio)
	}

	for _, field := range percentChange24hFields {
		if value, ok := jsonutil.Float(row[field]); ok {
			result[field] = currency_ratios.ConvertPercentChange24h(value, ratio)
		}
	}

	if conversion.Has1hAgo {
		if value, ok := jsonutil.Float(row[PercentChange1hField]); ok {
			result[PercentChange1hField] = currency_ratios.ConvertPercentChange(value, ratio.Now, conversion.Spot1hAgo)
		}
	}

	if sparkline, ok := convertSparkline(row[fieldSparkline7d], ratio.Now); ok {
		result[fieldSparkline7d] = sparkline
	}

	return result
}

// convertSparkline scales the 7d sparkline prices by the spot ratio, returning a
// fresh structure. ok is false when the field is absent or not shaped as expected.
func convertSparkline(value interface{}, spotRatio float64) (map[string]interface{}, bool) {
	sparkline, ok := value.(map[string]interface{})
	if !ok {
		return nil, false
	}

	prices, ok := sparkline[fieldSparklinePrice].([]interface{})
	if !ok {
		return nil, false
	}

	convertedPrices := make([]interface{}, 0, len(prices))
	for _, price := range prices {
		if number, ok := jsonutil.Float(price); ok {
			convertedPrices = append(convertedPrices, number*spotRatio)
			continue
		}
		convertedPrices = append(convertedPrices, price)
	}

	result := make(map[string]interface{}, len(sparkline))
	for key, entry := range sparkline {
		result[key] = entry
	}
	result[fieldSparklinePrice] = convertedPrices

	return result, true
}
