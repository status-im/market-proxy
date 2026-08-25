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

// convertibleTokenRow is a cached simple/price row carrying base currency values
// plus one other Passthrough currency
const convertibleTokenRow = `{"usd":100000,"usd_market_cap":2000000000,"usd_24h_vol":50000000,` +
	`"usd_24h_change":10,"chf":88000,"last_updated_at":1703097600}`

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

func eurPricesSnapshot() *interfaces.CurrencyRatiosSnapshot {
	return &interfaces.CurrencyRatiosSnapshot{
		Ratios: map[string]currency_ratios.Ratio{
			"usd": currency_ratios.IdentityRatio,
			"eur": {Now: 0.9, H24: 0.825},
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

func TestService_SimplePrices_ConvertCurrencyAddsEstimateKeys(t *testing.T) {
	service := newConvertingPricesService(t, eurPricesSnapshot())

	response, _, err := service.SimplePrices(context.Background(), interfaces.PriceParams{
		IDs:               []string{"bitcoin"},
		Currencies:        []string{"usd"},
		IncludeMarketCap:  true,
		Include24hrVol:    true,
		Include24hrChange: true,
		ConvertCurrency:   "eur",
	})
	require.NoError(t, err)

	row := bitcoinRow(t, response)

	// Passthrough keys survive untouched
	assert.Equal(t, 100000.0, row["usd"])
	assert.Equal(t, 2000000000.0, row["usd_market_cap"])
	assert.Equal(t, 10.0, row["usd_24h_change"])

	// Estimate keys are added
	assert.InDelta(t, 90000.0, row["eur"], 1e-9)
	assert.InDelta(t, 1800000000.0, row["eur_market_cap"], 1e-3)
	assert.InDelta(t, 45000000.0, row["eur_24h_vol"], 1e-3)
	assert.InDelta(t, 20.0, row["eur_24h_change"], 1e-9)
}

// TestService_SimplePrices_ConvertCurrencyReadsBaseWithoutLeakingIt covers
// vs_currencies=chf&convert_currency=eur: the base currency has to be read from
// cache to compute the Estimate, but the caller never asked for it.
func TestService_SimplePrices_ConvertCurrencyReadsBaseWithoutLeakingIt(t *testing.T) {
	service := newConvertingPricesService(t, eurPricesSnapshot())

	response, _, err := service.SimplePrices(context.Background(), interfaces.PriceParams{
		IDs:             []string{"bitcoin"},
		Currencies:      []string{"chf"},
		ConvertCurrency: "eur",
	})
	require.NoError(t, err)

	row := bitcoinRow(t, response)
	assert.Equal(t, 88000.0, row["chf"], "the requested Passthrough currency is served")
	assert.InDelta(t, 90000.0, row["eur"], 1e-9, "the Estimate is computed from the base currency")
	assert.NotContains(t, row, "usd", "the base currency was not requested")
	assert.NotContains(t, row, "usd_market_cap")
}

func TestService_SimplePrices_ConvertToBaseCurrencyPassesValuesThrough(t *testing.T) {
	service := newConvertingPricesService(t, eurPricesSnapshot())

	response, _, err := service.SimplePrices(context.Background(), interfaces.PriceParams{
		IDs:               []string{"bitcoin"},
		Currencies:        []string{"chf"},
		IncludeMarketCap:  true,
		Include24hrChange: true,
		ConvertCurrency:   "usd",
	})
	require.NoError(t, err)

	row := bitcoinRow(t, response)
	assert.Equal(t, 100000.0, row["usd"])
	assert.Equal(t, 2000000000.0, row["usd_market_cap"])
	assert.Equal(t, 10.0, row["usd_24h_change"])
	assert.Equal(t, 88000.0, row["chf"])
}

func TestService_SimplePrices_ConvertCurrencyWithoutRatio(t *testing.T) {
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

			response, cacheStatus, err := service.SimplePrices(context.Background(), interfaces.PriceParams{
				IDs:             []string{"bitcoin"},
				Currencies:      []string{"usd"},
				ConvertCurrency: "eur",
			})
			require.NoError(t, err)
			assert.Empty(t, response)
			assert.Equal(t, interfaces.CacheStatusMiss, cacheStatus)
		})
	}
}

func TestService_SimplePrices_WithoutConvertCurrency(t *testing.T) {
	service := newConvertingPricesService(t, eurPricesSnapshot())

	response, _, err := service.SimplePrices(context.Background(), interfaces.PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd"},
	})
	require.NoError(t, err)

	row := bitcoinRow(t, response)
	assert.Equal(t, 100000.0, row["usd"])
	assert.NotContains(t, row, "eur")
	assert.NotContains(t, row, "chf", "vs_currencies still filters the response")
}
