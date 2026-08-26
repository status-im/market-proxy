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
		CurrentPrice:             floatPtr(100000),
		MarketCap:                floatPtr(2000000000),
		TotalVolume:              floatPtr(50000000),
		PriceChangePercentage24h: floatPtr(10),
	}}}
	service.topMarketsUpdater.cache.Unlock()

	service.topPricesUpdater.topPricesCache.Lock()
	service.topPricesUpdater.topPricesCache.data = map[string]PriceQuotes{
		"usd": {"bitcoin": {Price: 100000, Volume24h: floatPtr(50000000), MarketCap: floatPtr(2000000000), PercentChange24h: floatPtr(10)}},
		"eur": {"bitcoin": {Price: 12345, Volume24h: floatPtr(1), MarketCap: floatPtr(2), PercentChange24h: floatPtr(3)}},
	}
	service.topPricesUpdater.topPricesCache.Unlock()

	return service
}

func TestService_GetCacheData_ConvertCurrency(t *testing.T) {
	service := newConvertingLeaderboard(t, eurLeaderboardSnapshot())

	converted := service.GetCacheData("eur")
	require.NotNil(t, converted)
	require.Len(t, converted.Data, 1)

	assert.InDelta(t, 90000.0, requireFloat(t, converted.Data[0].CurrentPrice), 1e-9)
	assert.InDelta(t, 1800000000.0, requireFloat(t, converted.Data[0].MarketCap), 1e-3)
	assert.InDelta(t, 20.0, requireFloat(t, converted.Data[0].PriceChangePercentage24h), 1e-9)

	// the cached Passthrough row is untouched
	passthrough := service.GetCacheData("")
	require.NotNil(t, passthrough)
	assert.Equal(t, floatPtr(100000.0), passthrough.Data[0].CurrentPrice)
	assert.Equal(t, floatPtr(10.0), passthrough.Data[0].PriceChangePercentage24h)
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
	assert.InDelta(t, 45000000.0, requireFloat(t, converted["bitcoin"].Volume24h), 1e-3)
	assert.InDelta(t, 20.0, requireFloat(t, converted["bitcoin"].PercentChange24h), 1e-9)
}

// TestService_GetTopPricesQuotes_CurrencyKeepsPassthroughSemantics pins that the
// two parameters mean different things: `currency` selects a cached currency,
// `convert_currency` computes an Estimate.
func TestService_GetTopPricesQuotes_CurrencyKeepsPassthroughSemantics(t *testing.T) {
	service := newConvertingLeaderboard(t, eurLeaderboardSnapshot())

	passthrough := service.GetTopPricesQuotes("eur", "")
	require.Contains(t, passthrough, "bitcoin")
	assert.Equal(t, 12345.0, passthrough["bitcoin"].Price, "the cached eur row is served verbatim")
	assert.Equal(t, floatPtr(3.0), passthrough["bitcoin"].PercentChange24h)

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

// TestService_EstimatesCurrency - the leaderboard caches exactly the one
// currency it is configured to fetch, so only that one is provider data.
func TestService_EstimatesCurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := mock_interfaces.NewMockICurrencyRatiosProvider(ctrl)

	usdConfigured := &config.Config{CoingeckoLeaderboard: config.LeaderboardFetcherConfig{Currency: "usd"}}
	service := NewService(usdConfigured, nil, nil, provider)

	assert.False(t, service.EstimatesCurrency("usd"))
	assert.False(t, service.EstimatesCurrency("USD"), "the check is case-insensitive")
	assert.True(t, service.EstimatesCurrency("eur"))

	// an unset currency falls back to the base currency, matching the updaters
	unset := NewService(&config.Config{}, nil, nil, provider)
	assert.False(t, unset.EstimatesCurrency(currency_ratios.BaseCurrency))
}

// TestService_CachedCurrencyIsServedAsProviderData - asking for the cached
// currency must not compute anything, even with no Ratio snapshot at all.
func TestService_CachedCurrencyIsServedAsProviderData(t *testing.T) {
	service := newConvertingLeaderboard(t, nil) // nil snapshot: any conversion would fail

	markets := service.GetCacheData(currency_ratios.BaseCurrency)
	require.NotNil(t, markets, "the cached rows are already in this currency")
	assert.Equal(t, 100000.0, *markets.Data[0].CurrentPrice)

	quotes := service.GetTopPricesQuotes("", currency_ratios.BaseCurrency)
	require.Contains(t, quotes, "bitcoin")
	assert.Equal(t, 100000.0, quotes["bitcoin"].Price, "the provider's own value, not a converted copy")
}

// TestService_ConvertsFromTheConfiguredCacheCurrency guards a config the shipped
// one does not use: if the leaderboard were configured to cache eur, the
// conversion source would be eur, and Ratios - all expressed against the base
// currency - would have to be crossed rather than applied directly.
func TestService_ConvertsFromTheConfiguredCacheCurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := mock_interfaces.NewMockICurrencyRatiosProvider(ctrl)
	provider.EXPECT().GetSnapshot().Return(&interfaces.CurrencyRatiosSnapshot{
		Ratios: map[string]currency_ratios.Ratio{
			"usd": currency_ratios.IdentityRatio,
			"eur": {Now: 0.9, H24: 0.825},
			"chf": {Now: 0.8, H24: 0.75},
		},
	}).AnyTimes()

	service := NewService(
		&config.Config{CoingeckoLeaderboard: config.LeaderboardFetcherConfig{Currency: "eur"}},
		nil, nil, provider,
	)

	service.topPricesUpdater.topPricesCache.Lock()
	service.topPricesUpdater.topPricesCache.data = map[string]PriceQuotes{
		"eur": {"bitcoin": {Price: 90000, PercentChange24h: floatPtr(10)}},
	}
	service.topPricesUpdater.topPricesCache.Unlock()

	quotes := service.GetTopPricesQuotes("", "chf")
	require.Contains(t, quotes, "bitcoin", "the cache is read in its own currency, not the base one")

	// eur -> chf spot ratio is 0.8/0.9
	assert.InDelta(t, 90000*(0.8/0.9), quotes["bitcoin"].Price, 1e-9)
	// and the honest change uses the crossed 24h ratio, 0.75/0.825
	expected := (1.1*(0.8/0.9)/(0.75/0.825) - 1) * 100
	assert.InDelta(t, expected, *quotes["bitcoin"].PercentChange24h, 1e-9)
}
