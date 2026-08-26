package coingecko_markets

import (
	"encoding/json"
	"testing"

	cache_mocks "github.com/status-im/market-proxy/cache/mocks"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
	interface_mocks "github.com/status-im/market-proxy/interfaces/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// convertibleRow is a cached row with the fields a conversion touches
const convertibleRow = `{"id":"bitcoin","symbol":"btc","current_price":100000,` +
	`"market_cap":2000000000,"high_24h":101000,"low_24h":98000,` +
	`"price_change_24h":5000,"price_change_percentage_24h":10,"market_cap_rank":1,` +
	`"price_change_percentage_1h_in_currency":2}`

// newConvertingService wires a markets service over a cache holding one row and
// a ratios provider with the given snapshot
func newConvertingService(t *testing.T, snapshot *interfaces.CurrencyRatiosSnapshot, spot1hAgo *float64) *Service {
	t.Helper()

	ctrl := gomock.NewController(t)

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockCache.EXPECT().Get(gomock.Any()).DoAndReturn(func(keys []string) (map[string][]byte, []string, error) {
		result := make(map[string][]byte, len(keys))
		for _, key := range keys {
			result[key] = []byte(convertibleRow)
		}
		return result, nil, nil
	}).AnyTimes()

	provider := interface_mocks.NewMockICurrencyRatiosProvider(ctrl)
	provider.EXPECT().GetSnapshot().Return(snapshot).AnyTimes()
	if spot1hAgo != nil {
		provider.EXPECT().GetSpotRatioAgo(gomock.Any(), gomock.Any()).Return(*spot1hAgo, true).AnyTimes()
	} else {
		provider.EXPECT().GetSpotRatioAgo(gomock.Any(), gomock.Any()).Return(0.0, false).AnyTimes()
	}

	return NewService(mockCache, createTestConfig(), createMockTokensService(ctrl), provider)
}

func eurSnapshot() *interfaces.CurrencyRatiosSnapshot {
	return &interfaces.CurrencyRatiosSnapshot{
		Ratios: map[string]currency_ratios.Ratio{
			"usd": currency_ratios.IdentityRatio,
			"eur": {Now: 0.9, H24: 0.825},
		},
		ReferenceCoin: "bitcoin",
	}
}

func firstRow(t *testing.T, response interfaces.MarketsResponse) map[string]interface{} {
	t.Helper()
	require.Len(t, response, 1)
	row, ok := response[0].(map[string]interface{})
	require.True(t, ok)
	return row
}

func TestService_Markets_WithoutConvertCurrency(t *testing.T) {
	service := newConvertingService(t, eurSnapshot(), nil)

	response, _, err := service.Markets(interfaces.MarketsParams{IDs: []string{"bitcoin"}, Currency: "eur"})
	require.NoError(t, err)

	assert.Equal(t, 100000.0, firstRow(t, response)["current_price"], "Passthrough values are served as-is")
}

func TestService_Markets_ConvertCurrency(t *testing.T) {
	service := newConvertingService(t, eurSnapshot(), nil)

	response, cacheStatus, err := service.Markets(interfaces.MarketsParams{
		IDs:             []string{"bitcoin"},
		ConvertCurrency: "eur",
	})
	require.NoError(t, err)
	assert.Equal(t, interfaces.CacheStatusFull, cacheStatus, "conversion keeps the cache status of the source read")

	row := firstRow(t, response)
	assert.InDelta(t, 90000.0, row["current_price"], 1e-9)
	assert.InDelta(t, 1800000000.0, row["market_cap"], 1e-3)
	assert.InDelta(t, 20.0, row["price_change_percentage_24h"], 1e-9)
	assert.InDelta(t, 11625.0, row["price_change_24h"], 1e-6)
	assert.Equal(t, 1.0, row["market_cap_rank"], "non-money fields pass through")
}

// TestService_Markets_ConvertCurrencyIgnoresVsCurrency pins the rule that
// vs_currency carries no information once a conversion is requested: cached rows
// are always normalized to the base currency.
func TestService_Markets_ConvertCurrencyIgnoresVsCurrency(t *testing.T) {
	service := newConvertingService(t, eurSnapshot(), nil)

	withoutVsCurrency, _, err := service.Markets(interfaces.MarketsParams{
		IDs:             []string{"bitcoin"},
		ConvertCurrency: "eur",
	})
	require.NoError(t, err)

	withVsCurrency, _, err := service.Markets(interfaces.MarketsParams{
		IDs:             []string{"bitcoin"},
		Currency:        "jpy",
		ConvertCurrency: "eur",
	})
	require.NoError(t, err)

	assert.Equal(t, withoutVsCurrency, withVsCurrency)
}

func TestService_Markets_ConvertCurrencyUses1hHistoryWhenAvailable(t *testing.T) {
	spot1hAgo := 0.88
	withHistory := newConvertingService(t, eurSnapshot(), &spot1hAgo)
	withoutHistory := newConvertingService(t, eurSnapshot(), nil)

	params := interfaces.MarketsParams{IDs: []string{"bitcoin"}, ConvertCurrency: "eur"}

	converted, _, err := withHistory.Markets(params)
	require.NoError(t, err)
	assert.InDelta(t, (1.02*0.9/0.88-1)*100, firstRow(t, converted)[PercentChange1hField], 1e-9)

	unconverted, _, err := withoutHistory.Markets(params)
	require.NoError(t, err)
	assert.Equal(t, 2.0, firstRow(t, unconverted)[PercentChange1hField],
		"without an hour of history the 1h change is left alone")
}

// TestService_Markets_ConvertCurrencyWithoutRatio is the "no snapshot yet" case:
// the service reports it by returning nothing rather than serving base currency
// values under another currency's name.
func TestService_Markets_ConvertCurrencyWithoutRatio(t *testing.T) {
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
			service := newConvertingService(t, tt.snapshot, nil)

			response, cacheStatus, err := service.Markets(interfaces.MarketsParams{
				IDs:             []string{"bitcoin"},
				ConvertCurrency: "eur",
			})
			require.NoError(t, err)
			assert.Empty(t, response)
			assert.Equal(t, interfaces.CacheStatusMiss, cacheStatus)
		})
	}
}

func TestService_Markets_ConvertCurrencyWithoutProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := cache_mocks.NewMockICache(ctrl)
	service := NewService(mockCache, createTestConfig(), createMockTokensService(ctrl), nil)

	response, _, err := service.Markets(interfaces.MarketsParams{
		IDs:             []string{"bitcoin"},
		ConvertCurrency: "eur",
	})
	require.NoError(t, err)
	assert.Empty(t, response)
}

// TestService_Markets_ConvertCurrencyDoesNotMutateCache guards the ADR rule that
// Passthrough values are never mutated in cache
func TestService_Markets_ConvertCurrencyDoesNotMutateCache(t *testing.T) {
	service := newConvertingService(t, eurSnapshot(), nil)
	params := interfaces.MarketsParams{IDs: []string{"bitcoin"}}

	converted, _, err := service.Markets(interfaces.MarketsParams{IDs: []string{"bitcoin"}, ConvertCurrency: "eur"})
	require.NoError(t, err)
	assert.InDelta(t, 90000.0, firstRow(t, converted)["current_price"], 1e-9)

	passthrough, _, err := service.Markets(params)
	require.NoError(t, err)

	var pristine map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(convertibleRow), &pristine))
	assert.Equal(t, pristine, firstRow(t, passthrough))
}

// TestService_EstimatesCurrency pins which currency the cache can serve as
// provider data: the one market_params_normalize pins it to.
func TestService_EstimatesCurrency(t *testing.T) {
	usd := "usd"

	withNormalization := createTestConfig()
	withNormalization.CoingeckoMarkets.MarketParamsNormalize = &config.MarketParamsNormalize{VsCurrency: &usd}

	ctrl := gomock.NewController(t)
	service := NewService(cache_mocks.NewMockICache(ctrl), withNormalization, createMockTokensService(ctrl), nil)

	assert.False(t, service.EstimatesCurrency("usd"), "the cache is normalized to usd")
	assert.False(t, service.EstimatesCurrency("USD"), "the check is case-insensitive")
	assert.True(t, service.EstimatesCurrency("eur"))

	// With normalization off the cache's currency is whatever the last fetch
	// used, so nothing can be claimed as provider data.
	withoutNormalization := createTestConfig()
	withoutNormalization.CoingeckoMarkets.MarketParamsNormalize = nil
	service = NewService(cache_mocks.NewMockICache(ctrl), withoutNormalization, createMockTokensService(ctrl), nil)

	assert.True(t, service.EstimatesCurrency("usd"))
}

// TestService_Markets_NormalizedCurrencyIsServedAsProviderData - asking for the
// currency the cache is already normalized to must not compute anything.
func TestService_Markets_NormalizedCurrencyIsServedAsProviderData(t *testing.T) {
	usd := "usd"

	cfg := createTestConfig()
	cfg.CoingeckoMarkets.MarketParamsNormalize = &config.MarketParamsNormalize{VsCurrency: &usd}

	ctrl := gomock.NewController(t)
	mockCache := cache_mocks.NewMockICache(ctrl)
	mockCache.EXPECT().Get(gomock.Any()).DoAndReturn(func(keys []string) (map[string][]byte, []string, error) {
		result := make(map[string][]byte, len(keys))
		for _, key := range keys {
			result[key] = []byte(convertibleRow)
		}
		return result, nil, nil
	}).AnyTimes()

	// a provider that would fail any conversion: if the row still comes back,
	// nothing was computed
	provider := interface_mocks.NewMockICurrencyRatiosProvider(ctrl)
	provider.EXPECT().GetSnapshot().Return(nil).AnyTimes()

	service := NewService(mockCache, cfg, createMockTokensService(ctrl), provider)

	response, _, err := service.Markets(interfaces.MarketsParams{
		IDs:             []string{"bitcoin"},
		ConvertCurrency: "usd",
	})
	require.NoError(t, err)

	var pristine map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(convertibleRow), &pristine))
	assert.Equal(t, pristine, firstRow(t, response),
		"the provider's own row is served, untouched, without needing a Ratio")
}
