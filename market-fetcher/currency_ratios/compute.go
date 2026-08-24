package currency_ratios

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/status-im/market-proxy/config"
)

// SimplePricePayload is the decoded CoinGecko simple/price response:
// coin id -> field name -> value. Fields are "<currency>" and
// "<currency>_24h_change"; either may be missing or null.
type SimplePricePayload map[string]map[string]interface{}

// changeSuffix is the field suffix CoinGecko uses for 24h changes
const changeSuffix = "_24h_change"

// ComputeRatios derives spot and 24h-ago ratios for every currency from a
// simple/price payload.
//
// The first reference coin with a present and complete row (base currency price
// and 24h change) wins; the reference coin's own price cancels out in the
// division, so the choice does not affect the result. Currencies with missing
// or unusable fields are skipped.
func ComputeRatios(payload SimplePricePayload, referenceCoins []string, currencies []string) (*Snapshot, error) {
	if len(referenceCoins) == 0 {
		return nil, fmt.Errorf("no reference coins configured")
	}

	for _, coin := range referenceCoins {
		row, ok := payload[coin]
		if !ok || len(row) == 0 {
			continue
		}

		basePrice, baseAgoPrice, ok := priceAndAgoPrice(row, config.CurrencyRatiosBaseCurrency)
		if !ok {
			continue
		}

		ratios := make(map[string]Ratio, len(currencies))
		for _, currency := range currencies {
			price, agoPrice, ok := priceAndAgoPrice(row, currency)
			if !ok {
				continue
			}

			ratios[currency] = Ratio{
				Now: price / basePrice,
				H24: agoPrice / baseAgoPrice,
			}
		}

		if len(ratios) == 0 {
			continue
		}

		return &Snapshot{
			Ratios:        ratios,
			ReferenceCoin: coin,
			UpdatedAt:     time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("no reference coin returned a complete row (tried %v)", referenceCoins)
}

// priceAndAgoPrice extracts the current and the 24h-ago price of the reference
// coin in the given currency. Returns false when any field is missing, null or
// unusable.
func priceAndAgoPrice(row map[string]interface{}, currency string) (price float64, agoPrice float64, ok bool) {
	price, ok = toFloat(row[currency])
	if !ok || price <= 0 {
		return 0, 0, false
	}

	change, ok := toFloat(row[currency+changeSuffix])
	if !ok {
		return 0, 0, false
	}

	// price_24h_ago = price_now / (1 + change/100)
	factor := 1 + change/100
	if factor <= 0 {
		return 0, 0, false
	}

	agoPrice = price / factor
	if agoPrice <= 0 {
		return 0, 0, false
	}

	return price, agoPrice, true
}

// toFloat converts a decoded JSON value to float64. Nulls, missing values and
// non-numeric values return false.
func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// ConvertPercentChange24h re-expresses a 24h percent change stated in the base
// currency into the target currency:
//
//	pct_X = ((1 + pct_base/100) * ratio_now / ratio_24h - 1) * 100
//
// A spot-ratio-only conversion would be wrong: bitcoin's 24h change measured in
// btc must come out ~0, not the usd change.
func ConvertPercentChange24h(pctBase float64, ratio Ratio) float64 {
	return ConvertPercentChange(pctBase, ratio.Now, ratio.H24)
}

// ConvertPercentChange re-expresses a percent change stated in the base currency
// into the target currency, given the target's ratio now and at the start of the
// same window:
//
//	pct_X = ((1 + pct_base/100) * ratioNow / ratioThen - 1) * 100
func ConvertPercentChange(pctBase, ratioNow, ratioThen float64) float64 {
	if ratioThen == 0 {
		return pctBase
	}
	// Unmoved ratio (always the case for the base currency): the change is
	// mathematically unchanged, so short-circuit instead of accumulating
	// floating point noise.
	if ratioNow == ratioThen {
		return pctBase
	}
	return ((1+pctBase/100)*ratioNow/ratioThen - 1) * 100
}

// ConvertAbsoluteChange24h re-expresses an absolute 24h delta (e.g.
// price_change_24h) stated in the base currency into the target currency.
//
// The delta is the difference of two amounts measured at different times, so
// each end is converted with the ratio that applied then:
//
//	delta_X = value_now_base * ratio_now - (value_now_base - delta_base) * ratio_24h
//
// It is evaluated in the algebraically identical but numerically stable form
// `value * (ratio_now - ratio_24h) + delta * ratio_24h`: the direct form
// subtracts two nearly equal large numbers whenever the ratio barely moved (the
// normal case for fiat), which would swamp a small delta in rounding error and
// make convert_currency=usd differ from Passthrough in the last digits.
func ConvertAbsoluteChange24h(valueNowBase, deltaBase float64, ratio Ratio) float64 {
	return valueNowBase*(ratio.Now-ratio.H24) + deltaBase*ratio.H24
}
