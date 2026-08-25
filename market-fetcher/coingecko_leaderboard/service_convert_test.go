package coingecko_leaderboard

import (
	"testing"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
	mock_interfaces "github.com/status-im/market-proxy/interfaces/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func eurLeaderboardSnapshot() *interfaces.CurrencyRatiosSnapshot {
	return &interfaces.CurrencyRatiosSnapshot{
		Ratios: map[string]currency_ratios.Ratio{
			"usd": currency_ratios.IdentityRatio,
			"eur": {Now: 0.9, H24: 0.825},
		},
		ReferenceCoin: "bitcoin",
	}
}

// newConvertingLeaderboard builds a service with cached markets rows and quotes
// and a ratios provider returning the given snapshot
func newConvertingLeaderboard(t *testing.T, snapshot *interfaces.CurrencyRatiosSnapshot) *Service {
	t.Helper()

	ctrl := gomock.NewController(t)
	provider := mock_interfaces.NewMockICurrencyRatiosProvider(ctrl)
	provider.EXPECT().GetSnapshot().Return(snapshot).AnyTimes()

	service := NewService(&config.Config{}, nil, nil, provider)

	service.topMarketsUpdater.cache.Lock()
	service.topMarketsUpdater.cache.data = &APIResponse{Data: []CoinData{{
		ID:                       "bitcoin",
		Symbol:                   "btc",
		CurrentPrice:             100000,
		MarketCap:                2000000000,
		TotalVolume:              50000000,
		PriceChangePercentage24h: 10,
	}}}
	service.topMarketsUpdater.cache.Unlock()

	service.topPricesUpdater.topPricesCache.Lock()
	service.topPricesUpdater.topPricesCache.data = map[string]PriceQuotes{
		"usd": {"bitcoin": {Price: 100000, Volume24h: 50000000, MarketCap: 2000000000, PercentChange24h: 10}},
		"eur": {"bitcoin": {Price: 12345, Volume24h: 1, MarketCap: 2, PercentChange24h: 3}},
	}
	service.topPricesUpdater.topPricesCache.Unlock()

	return service
}

func TestService_GetCacheData_ConvertCurrency(t *testing.T) {
	service := newConvertingLeaderboard(t, eurLeaderboardSnapshot())

	converted := service.GetCacheData("eur")
	require.NotNil(t, converted)
	require.Len(t, converted.Data, 1)

	assert.InDelta(t, 90000.0, converted.Data[0].CurrentPrice, 1e-9)
	assert.InDelta(t, 1800000000.0, converted.Data[0].MarketCap, 1e-3)
	assert.InDelta(t, 20.0, converted.Data[0].PriceChangePercentage24h, 1e-9)

	// the cached Passthrough row is untouched
	passthrough := service.GetCacheData("")
	require.NotNil(t, passthrough)
	assert.Equal(t, 100000.0, passthrough.Data[0].CurrentPrice)
	assert.Equal(t, 10.0, passthrough.Data[0].PriceChangePercentage24h)
}

func TestService_GetCacheData_ConvertCurrencyWithoutRatio(t *testing.T) {
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
			service := newConvertingLeaderboard(t, tt.snapshot)

			assert.Nil(t, service.GetCacheData("eur"), "no ratio reads as no data")
			assert.NotNil(t, service.GetCacheData(""), "Passthrough is unaffected")
		})
	}
}

func TestService_GetCacheData_ConvertCurrencyWithoutProvider(t *testing.T) {
	service := NewService(&config.Config{}, nil, nil, nil)
	assert.Nil(t, service.GetCacheData("eur"))
}

func TestService_GetTopPricesQuotes_ConvertCurrency(t *testing.T) {
	service := newConvertingLeaderboard(t, eurLeaderboardSnapshot())

	converted := service.GetTopPricesQuotes("", "eur")
	require.Contains(t, converted, "bitcoin")

	// Estimates come from the base currency rows, not from the cached eur ones
	assert.InDelta(t, 90000.0, converted["bitcoin"].Price, 1e-9)
	assert.InDelta(t, 45000000.0, converted["bitcoin"].Volume24h, 1e-3)
	assert.InDelta(t, 20.0, converted["bitcoin"].PercentChange24h, 1e-9)
}

// TestService_GetTopPricesQuotes_CurrencyKeepsPassthroughSemantics pins that the
// two parameters mean different things: `currency` selects a cached currency,
// `convert_currency` computes an Estimate.
func TestService_GetTopPricesQuotes_CurrencyKeepsPassthroughSemantics(t *testing.T) {
	service := newConvertingLeaderboard(t, eurLeaderboardSnapshot())

	passthrough := service.GetTopPricesQuotes("eur", "")
	require.Contains(t, passthrough, "bitcoin")
	assert.Equal(t, 12345.0, passthrough["bitcoin"].Price, "the cached eur row is served verbatim")
	assert.Equal(t, 3.0, passthrough["bitcoin"].PercentChange24h)

	defaulted := service.GetTopPricesQuotes("", "")
	assert.Equal(t, 100000.0, defaulted["bitcoin"].Price, "an empty currency falls back to the base currency")
}

func TestService_GetTopPricesQuotes_ConvertCurrencyWithoutRatio(t *testing.T) {
	service := newConvertingLeaderboard(t, nil)

	assert.Empty(t, service.GetTopPricesQuotes("", "eur"))
	assert.NotEmpty(t, service.GetTopPricesQuotes("", ""), "Passthrough is unaffected")
}

func TestService_GetTopPricesQuotes_ConvertCurrencyWithoutProvider(t *testing.T) {
	service := NewService(&config.Config{}, nil, nil, nil)
	assert.Empty(t, service.GetTopPricesQuotes("", "eur"))
}
