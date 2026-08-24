package coingecko_prices

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
)

// CoinGecko simple/price field suffixes
const (
	marketCapSuffix = "_market_cap"
	volume24hSuffix = "_24h_vol"
	change24hSuffix = "_24h_change"
)

// ConvertSimplePrices returns a copy of a simple/price response with Estimate
// keys for the target currency added, computed from the base currency keys.
//
// For a target currency X the keys `x`, `x_market_cap` and `x_24h_vol` are scaled
// by the spot ratio and `x_24h_change` is converted with the honest 24h formula.
// A key is only produced when the corresponding base currency key is present, so
// the include_* flags the caller applied are respected implicitly.
//
// When keepBase is false the base currency keys are dropped afterwards: they were
// only read in to compute the Estimate and the caller did not ask for them.
// `last_updated_at` and any other non-currency key are left untouched.
//
// The input response is never mutated - every token row is rebuilt.
func ConvertSimplePrices(
	response interfaces.SimplePriceResponse,
	currency string,
	ratio currency_ratios.Ratio,
	precision string,
	keepBase bool,
) interfaces.SimplePriceResponse {
	base := currency_ratios.BaseCurrency
	decimals := parsePrecision(precision)

	result := make(interfaces.SimplePriceResponse, len(response))

	for tokenID, tokenData := range response {
		row, ok := tokenData.(map[string]interface{})
		if !ok {
			result[tokenID] = tokenData
			continue
		}

		converted := make(map[string]interface{}, len(row)+4)
		for key, value := range row {
			if !keepBase && isCurrencyField(key, base) {
				continue
			}
			converted[key] = value
		}

		// Spot-scaled fields
		for _, suffix := range []string{"", marketCapSuffix, volume24hSuffix} {
			if value, ok := asFloat(row[base+suffix]); ok {
				converted[currency+suffix] = applyPrecision(value*ratio.Now, decimals)
			}
		}

		// Honest 24h change
		if value, ok := asFloat(row[base+change24hSuffix]); ok {
			converted[currency+change24hSuffix] = applyPrecision(currency_ratios.ConvertPercentChange24h(value, ratio), decimals)
		}

		if len(converted) > 0 {
			result[tokenID] = converted
		}
	}

	return result
}

// isCurrencyField reports whether a simple/price key belongs to the given currency
func isCurrencyField(key, currency string) bool {
	if key == currency {
		return true
	}
	for _, suffix := range []string{marketCapSuffix, volume24hSuffix, change24hSuffix} {
		if key == currency+suffix {
			return true
		}
	}
	return false
}

// parsePrecision mirrors the precision handling of stripResponse
func parsePrecision(precision string) int {
	if precision == "" {
		return 0
	}
	if decimals, err := strconv.Atoi(precision); err == nil && decimals > 0 {
		return decimals
	}
	return 0
}

// applyPrecision rounds an Estimate the same way Passthrough values are rounded
func applyPrecision(value float64, decimals int) float64 {
	if decimals <= 0 {
		return value
	}
	return roundToPrecision(value, decimals)
}

// SourceCurrencies returns the currency list to read from cache so that the base
// currency values needed for a conversion are present, without changing the
// currencies the caller actually asked for.
func SourceCurrencies(requested []string) []string {
	base := currency_ratios.BaseCurrency
	for _, currency := range requested {
		if strings.EqualFold(currency, base) {
			return requested
		}
	}

	source := make([]string, 0, len(requested)+1)
	source = append(source, requested...)

	return append(source, base)
}

// ContainsCurrency reports whether a currency is present in the list, ignoring case
func ContainsCurrency(currencies []string, currency string) bool {
	for _, candidate := range currencies {
		if strings.EqualFold(candidate, currency) {
			return true
		}
	}
	return false
}

// asFloat converts a decoded JSON value to float64
func asFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case json.Number:
		number, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}
