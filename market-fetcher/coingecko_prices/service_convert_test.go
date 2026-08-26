package coingecko_prices

import (
	"context"
	"testing"

	cache_mocks "github.com/status-im/market-proxy/cache/mocks"
	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
	mock_interfaces "github.com/status-im/market-proxy/interfaces/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// convertibleTokenRow is a cached simple/price row. createTestConfig fetches
// usd and eur from the provider, so both appear here as real provider values;
// chf is not fetched and can only ever be computed.
const convertibleTokenRow = `{"usd":100000,"usd_market_cap":2000000000,"usd_24h_vol":50000000,` +
	`"usd_24h_change":10,"eur":88000,"eur_market_cap":1760000000,"eur_24h_vol":44000000,` +
	`"eur_24h_change":7.5,"last_updated_at":1703097600}`

func newConvertingPricesService(t *testing.T, snapshot *interfaces.CurrencyRatiosSnapshot) *Service {
	t.Helper()

	ctrl := gomock.NewController(t)

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockCache.EXPECT().Get(gomock.Any()).DoAndReturn(func(keys []string) (map[string][]byte, []string, error) {
		result := make(map[string][]byte, len(keys))
		for _, key := range keys {
			result[key] = []byte(convertibleTokenRow)
		}
		return result, nil, nil
	}).AnyTimes()

	provider := mock_interfaces.NewMockICurrencyRatiosProvider(ctrl)
	provider.EXPECT().GetSnapshot().Return(snapshot).AnyTimes()

	return NewService(mockCache, createTestConfig(), nil, createMockTokensService(ctrl), provider)
}

func ratiosSnapshot() *interfaces.CurrencyRatiosSnapshot {
	return &interfaces.CurrencyRatiosSnapshot{
		Ratios: map[string]currency_ratios.Ratio{
			"usd": currency_ratios.IdentityRatio,
			"eur": {Now: 0.9, H24: 0.825},
			"chf": {Now: 0.8, H24: 0.78},
		},
		ReferenceCoin: "bitcoin",
	}
}

func bitcoinRow(t *testing.T, response interfaces.SimplePriceResponse) map[string]interface{} {
	t.Helper()
	row, ok := response["bitcoin"].(map[string]interface{})
	require.True(t, ok, "response should contain a bitcoin row: %v", response)
	return row
}

func allMetadata(currencies []string, convert string) interfaces.PriceParams {
	return interfaces.PriceParams{
		IDs:               []string{"bitcoin"},
		Currencies:        currencies,
		IncludeMarketCap:  true,
		Include24hrVol:    true,
		Include24hrChange: true,
		ConvertCurrency:   convert,
	}
}

// TestService_EstimatesCurrency pins the rule the header and the source choice
// both hang off: a currency the proxy fetches is never estimated.
func TestService_EstimatesCurrency(t *testing.T) {
	service := newConvertingPricesService(t, ratiosSnapshot())

	assert.False(t, service.EstimatesCurrency("usd"), "usd is fetched from the provider")
	assert.False(t, service.EstimatesCurrency("eur"), "eur is fetched from the provider")
	assert.False(t, service.EstimatesCurrency("EUR"), "the check is case-insensitive")
	assert.True(t, service.EstimatesCurrency("chf"), "chf is not fetched and can only be computed")
}

// TestService_SimplePrices_CachedCurrencyIsServedAsProviderData is the point of
// the design: asking for a currency the proxy already holds must return the
// provider's own values, not a copy derived from usd.
func TestService_SimplePrices_CachedCurrencyIsServedAsProviderData(t *testing.T) {
	service := newConvertingPricesService(t, ratiosSnapshot())

	response, _, err := service.SimplePrices(context.Background(), allMetadata([]string{"usd"}, "eur"))
	require.NoError(t, err)

	row := bitcoinRow(t, response)

	// the provider's eur values, not 100000 * 0.9
	assert.Equal(t, 88000.0, row["eur"])
	assert.Equal(t, 1760000000.0, row["eur_market_cap"])
	assert.Equal(t, 44000000.0, row["eur_24h_vol"])
	assert.Equal(t, 7.5, row["eur_24h_change"])

	assert.Equal(t, 100000.0, row["usd"], "the requested passthrough currency is still there")
}

// TestService_SimplePrices_SameCurrencyInBothParamsIsDeduped removes the rule
// that forced clients to know which currencies the proxy caches.
func TestService_SimplePrices_SameCurrencyInBothParamsIsDeduped(t *testing.T) {
	service := newConvertingPricesService(t, ratiosSnapshot())

	both, _, err := service.SimplePrices(context.Background(), allMetadata([]string{"usd", "eur"}, "eur"))
	require.NoError(t, err)

	onlyVsCurrencies, _, err := service.SimplePrices(context.Background(), allMetadata([]string{"usd", "eur"}, ""))
	require.NoError(t, err)

	assert.Equal(t, onlyVsCurrencies, both,
		"naming eur twice is a duplicate, not a conflict, and yields one set of values")
}

// TestService_SimplePrices_NonCachedCurrencyIsComputed keeps the original path
// working for a currency the proxy does not fetch.
func TestService_SimplePrices_NonCachedCurrencyIsComputed(t *testing.T) {
	service := newConvertingPricesService(t, ratiosSnapshot())

	response, _, err := service.SimplePrices(context.Background(), allMetadata([]string{"usd"}, "chf"))
	require.NoError(t, err)

	row := bitcoinRow(t, response)
	assert.InDelta(t, 80000.0, row["chf"], 1e-9, "100000 * 0.8")
	assert.InDelta(t, 1600000000.0, row["chf_market_cap"], 1e-3)
	assert.InDelta(t, 40000000.0, row["chf_24h_vol"], 1e-3)
	// ((1 + 0.10) * 0.8 / 0.78 - 1) * 100
	assert.InDelta(t, (1.1*0.8/0.78-1)*100, row["chf_24h_change"], 1e-9)

	assert.Equal(t, 100000.0, row["usd"])
}

// TestService_SimplePrices_ComputedCurrencyReadsBaseWithoutLeakingIt covers
// vs_currencies=eur&convert_currency=chf: usd has to be read to compute chf,
// but the caller never asked for it.
func TestService_SimplePrices_ComputedCurrencyReadsBaseWithoutLeakingIt(t *testing.T) {
	service := newConvertingPricesService(t, ratiosSnapshot())

	response, _, err := service.SimplePrices(context.Background(), interfaces.PriceParams{
		IDs:             []string{"bitcoin"},
		Currencies:      []string{"eur"},
		ConvertCurrency: "chf",
	})
	require.NoError(t, err)

	row := bitcoinRow(t, response)
	assert.Equal(t, 88000.0, row["eur"], "the requested passthrough currency is served")
	assert.InDelta(t, 80000.0, row["chf"], 1e-9, "the computed currency is added")
	assert.NotContains(t, row, "usd", "the base currency was not requested")
	assert.NotContains(t, row, "usd_market_cap")
}

// TestService_SimplePrices_BaseCurrencyIsPassthrough - usd is fetched, so
// convert_currency=usd is a plain read and the values are the provider's own.
func TestService_SimplePrices_BaseCurrencyIsPassthrough(t *testing.T) {
	service := newConvertingPricesService(t, ratiosSnapshot())

	response, _, err := service.SimplePrices(context.Background(), allMetadata([]string{"eur"}, "usd"))
	require.NoError(t, err)

	row := bitcoinRow(t, response)
	assert.Equal(t, 100000.0, row["usd"])
	assert.Equal(t, 2000000000.0, row["usd_market_cap"])
	assert.Equal(t, 10.0, row["usd_24h_change"])
	assert.Equal(t, 88000.0, row["eur"])
}

// TestService_SimplePrices_ComputedCurrencyWithoutRatio - only the computed path
// can fail for want of a Ratio; the passthrough path never needs one.
func TestService_SimplePrices_ComputedCurrencyWithoutRatio(t *testing.T) {
	tests := []struct {
		name     string
		snapshot *interfaces.CurrencyRatiosSnapshot
	}{
		{name: "no snapshot at all", snapshot: nil},
		{
			name: "currency missing from the snapshot",
			snapshot: &interfaces.CurrencyRatiosSnapshot{
				Ratios: map[string]currency_ratios.Ratio{"usd": currency_ratios.IdentityRatio},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newConvertingPricesService(t, tt.snapshot)

			response, cacheStatus, err := service.SimplePrices(context.Background(),
				allMetadata([]string{"usd"}, "chf"))
			require.NoError(t, err)
			assert.Empty(t, response)
			assert.Equal(t, interfaces.CacheStatusMiss, cacheStatus)

			// a cached currency is unaffected by the missing snapshot
			cached, _, err := service.SimplePrices(context.Background(), allMetadata([]string{"usd"}, "eur"))
			require.NoError(t, err)
			assert.Equal(t, 88000.0, bitcoinRow(t, cached)["eur"])
		})
	}
}

func TestService_SimplePrices_WithoutConvertCurrency(t *testing.T) {
	service := newConvertingPricesService(t, ratiosSnapshot())

	response, _, err := service.SimplePrices(context.Background(), interfaces.PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd"},
	})
	require.NoError(t, err)

	row := bitcoinRow(t, response)
	assert.Equal(t, 100000.0, row["usd"])
	assert.NotContains(t, row, "eur", "vs_currencies still filters the response")
	assert.NotContains(t, row, "chf")
}

func TestAppendCurrency(t *testing.T) {
	assert.Equal(t, []string{"usd", "eur"}, appendCurrency([]string{"usd"}, "eur"))
	assert.Equal(t, []string{"usd", "eur"}, appendCurrency([]string{"usd", "eur"}, "eur"))
	assert.Equal(t, []string{"USD"}, appendCurrency([]string{"USD"}, "usd"), "case-insensitive")
	assert.Equal(t, []string{"usd"}, appendCurrency(nil, "usd"))
}
