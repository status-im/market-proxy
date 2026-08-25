package currency_ratios

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// samplePayload is a simple/price response shaped like CoinGecko's, with
// bitcoin priced in usd, eur and btc plus 24h changes.
const samplePayload = `{
  "bitcoin": {
    "usd": 100000.0,
    "usd_24h_change": 10.0,
    "eur": 90000.0,
    "eur_24h_change": 20.0,
    "btc": 1.0,
    "btc_24h_change": 0.0
  },
  "ethereum": {
    "usd": 4000.0,
    "usd_24h_change": 10.0,
    "eur": 3600.0,
    "eur_24h_change": 20.0
  }
}`

func mustDecode(t *testing.T, raw string) SimplePricePayload {
	t.Helper()
	var payload SimplePricePayload
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	return payload
}

func TestComputeRatios_FromReferenceCoin(t *testing.T) {
	snapshot, err := ComputeRatios(mustDecode(t, samplePayload), []string{"bitcoin", "ethereum"}, []string{"usd", "eur", "btc"})
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	assert.Equal(t, "bitcoin", snapshot.ReferenceCoin)
	assert.False(t, snapshot.UpdatedAt.IsZero())

	// usd is the base: both ratios are exactly 1
	usd, ok := snapshot.Ratio("usd")
	require.True(t, ok)
	assert.Equal(t, 1.0, usd.Now)
	assert.Equal(t, 1.0, usd.H24)

	// eur spot ratio: 90000 / 100000
	eur, ok := snapshot.Ratio("eur")
	require.True(t, ok)
	assert.InDelta(t, 0.9, eur.Now, 1e-12)

	// eur 24h-ago ratio: (90000 / 1.2) / (100000 / 1.1) = 75000 / 90909.0909... = 0.825
	assert.InDelta(t, 0.825, eur.H24, 1e-12)

	// btc spot ratio: 1 / 100000
	btc, ok := snapshot.Ratio("btc")
	require.True(t, ok)
	assert.InDelta(t, 1e-5, btc.Now, 1e-17)
	// btc 24h-ago ratio: (1 / 1.0) / (100000 / 1.1) = 1.1e-5
	assert.InDelta(t, 1.1e-5, btc.H24, 1e-17)
}

// TestComputeRatios_ReferenceCoinDoesNotMatter is the empirical claim the ADR
// rests on: the reference coin's own price cancels out in the division.
func TestComputeRatios_ReferenceCoinDoesNotMatter(t *testing.T) {
	payload := mustDecode(t, samplePayload)
	currencies := []string{"usd", "eur"}

	fromBitcoin, err := ComputeRatios(payload, []string{"bitcoin"}, currencies)
	require.NoError(t, err)
	fromEthereum, err := ComputeRatios(payload, []string{"ethereum"}, currencies)
	require.NoError(t, err)

	// ethereum: spot 3600/4000 = 0.9; 24h-ago (3600/1.2)/(4000/1.1) = 3000/3636.36 = 0.825
	assert.InDelta(t, fromBitcoin.Ratios["eur"].Now, fromEthereum.Ratios["eur"].Now, 1e-12)
	assert.InDelta(t, fromBitcoin.Ratios["eur"].H24, fromEthereum.Ratios["eur"].H24, 1e-12)
}

func TestComputeRatios_FallsBackToNextReferenceCoin(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "first coin missing entirely",
			payload: `{"ethereum": {"usd": 4000.0, "usd_24h_change": 10.0, "eur": 3600.0, "eur_24h_change": 20.0}}`,
		},
		{
			name:    "first coin has null base price",
			payload: `{"bitcoin": {"usd": null, "usd_24h_change": 10.0}, "ethereum": {"usd": 4000.0, "usd_24h_change": 10.0, "eur": 3600.0, "eur_24h_change": 20.0}}`,
		},
		{
			name:    "first coin missing base 24h change",
			payload: `{"bitcoin": {"usd": 100000.0}, "ethereum": {"usd": 4000.0, "usd_24h_change": 10.0, "eur": 3600.0, "eur_24h_change": 20.0}}`,
		},
		{
			name:    "first coin has empty row",
			payload: `{"bitcoin": {}, "ethereum": {"usd": 4000.0, "usd_24h_change": 10.0, "eur": 3600.0, "eur_24h_change": 20.0}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, err := ComputeRatios(mustDecode(t, tt.payload), []string{"bitcoin", "ethereum"}, []string{"usd", "eur"})
			require.NoError(t, err)
			assert.Equal(t, "ethereum", snapshot.ReferenceCoin)
			assert.InDelta(t, 0.9, snapshot.Ratios["eur"].Now, 1e-12)
		})
	}
}

func TestComputeRatios_SkipsIncompleteCurrencies(t *testing.T) {
	payload := mustDecode(t, `{
	  "bitcoin": {
	    "usd": 100000.0, "usd_24h_change": 10.0,
	    "eur": 90000.0, "eur_24h_change": 20.0,
	    "jpy": null, "jpy_24h_change": 1.0,
	    "gbp": 80000.0,
	    "chf": 0.0, "chf_24h_change": 1.0,
	    "sek": 90000.0, "sek_24h_change": -100.0
	  }
	}`)

	snapshot, err := ComputeRatios(payload, []string{"bitcoin"}, []string{"usd", "eur", "jpy", "gbp", "chf", "sek", "nok"})
	require.NoError(t, err)

	assert.Len(t, snapshot.Ratios, 2)
	assert.Contains(t, snapshot.Ratios, "usd")
	assert.Contains(t, snapshot.Ratios, "eur")
	assert.NotContains(t, snapshot.Ratios, "jpy", "null price must be skipped")
	assert.NotContains(t, snapshot.Ratios, "gbp", "missing 24h change must be skipped")
	assert.NotContains(t, snapshot.Ratios, "chf", "zero price must be skipped")
	assert.NotContains(t, snapshot.Ratios, "sek", "-100% change would divide by zero")
	assert.NotContains(t, snapshot.Ratios, "nok", "currency absent from the row must be skipped")
}

func TestComputeRatios_Errors(t *testing.T) {
	tests := []struct {
		name           string
		payload        string
		referenceCoins []string
	}{
		{
			name:           "no reference coins configured",
			payload:        samplePayload,
			referenceCoins: nil,
		},
		{
			name:           "no reference coin present",
			payload:        `{"dogecoin": {"usd": 0.4, "usd_24h_change": 1.0}}`,
			referenceCoins: []string{"bitcoin", "ethereum"},
		},
		{
			name:           "empty payload",
			payload:        `{}`,
			referenceCoins: []string{"bitcoin"},
		},
		{
			name:           "reference coin has no usable base price",
			payload:        `{"bitcoin": {"usd": "not-a-number", "usd_24h_change": 1.0}}`,
			referenceCoins: []string{"bitcoin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, err := ComputeRatios(mustDecode(t, tt.payload), tt.referenceCoins, []string{"usd", "eur"})
			assert.Error(t, err)
			assert.Nil(t, snapshot)
		})
	}
}

// TestConvertPercentChange24h_HandComputed pins the honest percent-change
// formula against values worked out by hand.
func TestConvertPercentChange24h_HandComputed(t *testing.T) {
	tests := []struct {
		name     string
		pctBase  float64
		ratio    Ratio
		expected float64
	}{
		{
			// Base currency: ratios are 1, so the change passes through untouched.
			name:     "base currency passes through",
			pctBase:  7.5,
			ratio:    IdentityRatio,
			expected: 7.5,
		},
		{
			// From samplePayload: eur ratio_now = 0.9, ratio_24h = 0.825.
			// A coin up 10% in usd: ((1.10 * 0.9 / 0.825) - 1) * 100 = 20%
			name:     "usd gain amplified by a weakening base",
			pctBase:  10.0,
			ratio:    Ratio{Now: 0.9, H24: 0.825},
			expected: 20.0,
		},
		{
			// Bitcoin measured in btc must come out ~0, whatever its usd change.
			// btc ratio_now = 1e-5, ratio_24h = 1.1e-5, bitcoin +10% in usd:
			// ((1.10 * 1e-5 / 1.1e-5) - 1) * 100 = 0
			name:     "bitcoin in btc terms is flat",
			pctBase:  10.0,
			ratio:    Ratio{Now: 1e-5, H24: 1.1e-5},
			expected: 0.0,
		},
		{
			// A flat usd price still moves when the target currency moves:
			// ((1.0 * 0.9 / 0.825) - 1) * 100 = 9.0909...%
			name:     "flat in usd, up in eur",
			pctBase:  0.0,
			ratio:    Ratio{Now: 0.9, H24: 0.825},
			expected: 100.0 * (0.9/0.825 - 1),
		},
		{
			name:     "a spot-only conversion would be wrong: loss in usd, loss in eur",
			pctBase:  -20.0,
			ratio:    Ratio{Now: 0.9, H24: 0.9},
			expected: -20.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, ConvertPercentChange24h(tt.pctBase, tt.ratio), 1e-9)
		})
	}
}

// TestConvertPercentChange24h_ConsistentWithRawPrices cross-checks the formula
// against converting the raw prices directly.
func TestConvertPercentChange24h_ConsistentWithRawPrices(t *testing.T) {
	priceUSDNow := 250.0
	pctUSD := -12.5
	priceUSD24h := priceUSDNow / (1 + pctUSD/100)

	ratio := Ratio{Now: 0.9, H24: 0.825}

	priceEURNow := priceUSDNow * ratio.Now
	priceEUR24h := priceUSD24h * ratio.H24
	expected := (priceEURNow/priceEUR24h - 1) * 100

	got := ConvertPercentChange24h(pctUSD, ratio)
	assert.InDelta(t, expected, got, 1e-9)
	assert.False(t, math.IsNaN(got))
}

func TestConvertPercentChange24h_ZeroRatioFallsBack(t *testing.T) {
	assert.Equal(t, 3.0, ConvertPercentChange24h(3.0, Ratio{Now: 0.9, H24: 0}))
	assert.Equal(t, 3.0, ConvertPercentChange(3.0, 0.9, 0))
}

// TestConvertAbsoluteChange24h_HandComputed pins the absolute-delta conversion.
// Each end of the delta is converted with the ratio that applied at that end.
func TestConvertAbsoluteChange24h_HandComputed(t *testing.T) {
	tests := []struct {
		name     string
		valueNow float64
		delta    float64
		ratio    Ratio
		expected float64
	}{
		{
			// 100000 usd now, +5000 over 24h, so 95000 usd a day ago.
			// 100000*0.9 - 95000*0.825 = 90000 - 78375 = 11625
			name:     "gain in usd, larger gain in eur",
			valueNow: 100000,
			delta:    5000,
			ratio:    Ratio{Now: 0.9, H24: 0.825},
			expected: 11625,
		},
		{
			name:     "base currency passes through",
			valueNow: 100000,
			delta:    5000,
			ratio:    IdentityRatio,
			expected: 5000,
		},
		{
			// bitcoin measured in btc: 100000*1e-5 - 95000*1.1e-5 = 1 - 1.045
			name:     "bitcoin in btc terms",
			valueNow: 100000,
			delta:    5000,
			ratio:    Ratio{Now: 1e-5, H24: 1.1e-5},
			expected: -0.045,
		},
		{
			// a flat usd price still moves when the target currency moves:
			// 200*0.9 - 200*0.825 = 15
			name:     "flat in usd, up in eur",
			valueNow: 200,
			delta:    0,
			ratio:    Ratio{Now: 0.9, H24: 0.825},
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, ConvertAbsoluteChange24h(tt.valueNow, tt.delta, tt.ratio), 1e-9)
		})
	}
}

// TestConvertAbsoluteChange24h_ExactForUnmovedRatio guards the numerically stable
// form: a small delta against a large value must survive bit-for-bit when the
// ratio did not move, so convert_currency=usd matches Passthrough exactly.
func TestConvertAbsoluteChange24h_ExactForUnmovedRatio(t *testing.T) {
	tests := []struct {
		name     string
		valueNow float64
		delta    float64
		ratio    Ratio
	}{
		{name: "large value, small delta", valueNow: 78402, delta: 895.16, ratio: IdentityRatio},
		{name: "tiny delta", valueNow: 1.0001, delta: -4.7642098751077e-05, ratio: IdentityRatio},
		{name: "unmoved non-unit ratio", valueNow: 78402, delta: 895.16, ratio: Ratio{Now: 0.9, H24: 0.9}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.delta*tt.ratio.H24, ConvertAbsoluteChange24h(tt.valueNow, tt.delta, tt.ratio))
		})
	}
}

// TestConvertAbsoluteChange24h_ConsistentWithPercentChange cross-checks the two
// honest conversions against each other.
func TestConvertAbsoluteChange24h_ConsistentWithPercentChange(t *testing.T) {
	ratio := Ratio{Now: 0.9, H24: 0.825}
	priceNow := 100000.0
	delta := 5000.0

	convertedDelta := ConvertAbsoluteChange24h(priceNow, delta, ratio)
	convertedPrice := priceNow * ratio.Now
	pctFromDelta := convertedDelta / (convertedPrice - convertedDelta) * 100

	pctUSD := delta / (priceNow - delta) * 100
	assert.InDelta(t, ConvertPercentChange24h(pctUSD, ratio), pctFromDelta, 1e-9)
}

func TestSnapshot_RatioOnNilSnapshot(t *testing.T) {
	var snapshot *Snapshot
	ratio, ok := snapshot.Ratio("eur")
	assert.False(t, ok)
	assert.Equal(t, Ratio{}, ratio)
}
