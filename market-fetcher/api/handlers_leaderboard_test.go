package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubLeaderboardService serves fixed leaderboard cache contents
type stubLeaderboardService struct {
	markets           *coingecko_leaderboard.APIResponse
	quotes            map[string]coingecko_leaderboard.PriceQuotes
	requestedCurrency string
}

func (s *stubLeaderboardService) GetCacheData() *coingecko_leaderboard.APIResponse {
	return s.markets
}

func (s *stubLeaderboardService) GetTopPricesQuotes(currency string) map[string]coingecko_leaderboard.Quote {
	s.requestedCurrency = currency
	if quotes, ok := s.quotes[currency]; ok {
		return quotes
	}
	return map[string]coingecko_leaderboard.Quote{}
}

func (s *stubLeaderboardService) Healthy() bool { return true }

// stubRatiosProvider is a interfaces.ICurrencyRatiosProvider with a fixed snapshot
type stubRatiosProvider struct {
	supported map[string]bool
	snapshot  *currency_ratios.Snapshot
	// spotAgo holds the historical spot ratios per currency; a currency absent
	// from the map behaves like history that does not reach back far enough.
	spotAgo map[string]float64
}

func (s *stubRatiosProvider) GetSnapshot() *currency_ratios.Snapshot { return s.snapshot }

func (s *stubRatiosProvider) IsCurrencySupported(currency string) bool {
	return s.supported[currency]
}

func (s *stubRatiosProvider) GetSpotRatioAgo(currency string, _ time.Duration) (float64, bool) {
	ratio, ok := s.spotAgo[currency]
	return ratio, ok
}

// eurRatio: eur is 0.9 usd now and was 0.825 usd 24h ago
var eurRatio = currency_ratios.Ratio{Now: 0.9, H24: 0.825}

func readySnapshot() *currency_ratios.Snapshot {
	return &currency_ratios.Snapshot{
		Ratios: map[string]currency_ratios.Ratio{
			"usd": currency_ratios.IdentityRatio,
			"eur": eurRatio,
		},
		ReferenceCoin: "bitcoin",
		UpdatedAt:     time.Now(),
	}
}

func supportedCurrencies() map[string]bool {
	return map[string]bool{"usd": true, "eur": true, "btc": true}
}

func testMarkets() *coingecko_leaderboard.APIResponse {
	return &coingecko_leaderboard.APIResponse{
		Data: []coingecko_leaderboard.CoinData{
			{
				ID:                       "bitcoin",
				Symbol:                   "btc",
				Name:                     "Bitcoin",
				CurrentPrice:             100000,
				MarketCap:                2000000000,
				TotalVolume:              50000000,
				PriceChangePercentage24h: 10,
			},
		},
	}
}

func testQuotes() map[string]coingecko_leaderboard.PriceQuotes {
	return map[string]coingecko_leaderboard.PriceQuotes{
		"usd": {
			"bitcoin": {Price: 100000, Volume24h: 50000000, MarketCap: 2000000000, PercentChange24h: 10},
		},
		"eur": {
			"bitcoin": {Price: 12345, Volume24h: 1, MarketCap: 2, PercentChange24h: 3},
		},
	}
}

func newTestServer(leaderboard interfaces.ILeaderboardService, ratios interfaces.ICurrencyRatiosProvider) *Server {
	return &Server{cgService: leaderboard, currencyRatiosService: ratios}
}

func doRequest(t *testing.T, handler http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func decodeMarkets(t *testing.T, recorder *httptest.ResponseRecorder) coingecko_leaderboard.APIResponse {
	t.Helper()
	var response coingecko_leaderboard.APIResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func decodeQuotes(t *testing.T, recorder *httptest.ResponseRecorder) map[string]coingecko_leaderboard.Quote {
	t.Helper()
	var response map[string]coingecko_leaderboard.Quote
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

// --- markets ---

func TestHandleLeaderboardMarkets_NoConversionIsUntouched(t *testing.T) {
	leaderboard := &stubLeaderboardService{markets: testMarkets()}
	server := newTestServer(leaderboard, &stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()})

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets")

	assert.Equal(t, http.StatusOK, recorder.Code)
	response := decodeMarkets(t, recorder)
	require.Len(t, response.Data, 1)
	assert.Equal(t, 100000.0, response.Data[0].CurrentPrice)
	assert.Equal(t, 10.0, response.Data[0].PriceChangePercentage24h)
}

func TestHandleLeaderboardMarkets_ConvertCurrency(t *testing.T) {
	leaderboard := &stubLeaderboardService{markets: testMarkets()}
	server := newTestServer(leaderboard, &stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()})

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	response := decodeMarkets(t, recorder)
	require.Len(t, response.Data, 1)
	assert.InDelta(t, 90000.0, response.Data[0].CurrentPrice, 1e-9)
	assert.InDelta(t, 1800000000.0, response.Data[0].MarketCap, 1e-3)
	assert.InDelta(t, 45000000.0, response.Data[0].TotalVolume, 1e-3)
	assert.InDelta(t, 20.0, response.Data[0].PriceChangePercentage24h, 1e-9)

	// the cached Passthrough data was not mutated
	assert.Equal(t, 100000.0, leaderboard.markets.Data[0].CurrentPrice)
	assert.Equal(t, 10.0, leaderboard.markets.Data[0].PriceChangePercentage24h)
}

func TestHandleLeaderboardMarkets_ConvertCurrencyIsCaseInsensitive(t *testing.T) {
	server := newTestServer(
		&stubLeaderboardService{markets: testMarkets()},
		&stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()},
	)

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=EUR")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.InDelta(t, 90000.0, decodeMarkets(t, recorder).Data[0].CurrentPrice, 1e-9)
}

func TestHandleLeaderboardMarkets_ConvertToBaseCurrencyPassesValuesThrough(t *testing.T) {
	server := newTestServer(
		&stubLeaderboardService{markets: testMarkets()},
		&stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()},
	)

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=usd")

	assert.Equal(t, http.StatusOK, recorder.Code)
	response := decodeMarkets(t, recorder)
	assert.Equal(t, testMarkets().Data, response.Data)
}

func TestHandleLeaderboardMarkets_UnknownCurrencyReturns400(t *testing.T) {
	server := newTestServer(
		&stubLeaderboardService{markets: testMarkets()},
		&stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()},
	)

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "unsupported convert_currency: xyz", body["error"])
}

// A configured currency that is not in the snapshot yet is not an error - the
// endpoint answers with its empty response shape, exactly as for an empty cache.
func TestHandleLeaderboardMarkets_EmptyResponseBeforeFirstSnapshot(t *testing.T) {
	tests := []struct {
		name   string
		ratios interfaces.ICurrencyRatiosProvider
	}{
		{
			name:   "no snapshot at all",
			ratios: &stubRatiosProvider{supported: supportedCurrencies(), snapshot: nil},
		},
		{
			name: "currency missing from snapshot",
			ratios: &stubRatiosProvider{
				supported: supportedCurrencies(),
				snapshot: &currency_ratios.Snapshot{
					Ratios: map[string]currency_ratios.Ratio{"usd": currency_ratios.IdentityRatio},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(&stubLeaderboardService{markets: testMarkets()}, tt.ratios)

			recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=eur")

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.JSONEq(t, `{"data":[]}`, recorder.Body.String())
		})
	}
}

func TestHandleLeaderboardMarkets_EmptyCacheKeepsEmptyShape(t *testing.T) {
	server := newTestServer(
		&stubLeaderboardService{markets: nil},
		&stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()},
	)

	for _, target := range []string{
		"/api/v1/leaderboard/markets",
		"/api/v1/leaderboard/markets?convert_currency=eur",
	} {
		recorder := doRequest(t, server.handleLeaderboardMarkets, target)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{"data":[]}`, recorder.Body.String())
	}
}

func TestHandleLeaderboardMarkets_NoRatiosServiceRejectsConversion(t *testing.T) {
	server := newTestServer(&stubLeaderboardService{markets: testMarkets()}, nil)

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=eur")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	// without the parameter the endpoint is unaffected
	recorder = doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets")
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Len(t, decodeMarkets(t, recorder).Data, 1)
}

// --- prices ---

func TestHandleLeaderboardPrices_CurrencyParamSemanticsUnchanged(t *testing.T) {
	leaderboard := &stubLeaderboardService{quotes: testQuotes()}
	server := newTestServer(leaderboard, &stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()})

	// `currency` still selects a Passthrough currency straight from the cache
	recorder := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices?currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "eur", leaderboard.requestedCurrency)

	quotes := decodeQuotes(t, recorder)
	assert.Equal(t, 12345.0, quotes["bitcoin"].Price)
	assert.Equal(t, 3.0, quotes["bitcoin"].PercentChange24h)
}

func TestHandleLeaderboardPrices_ConvertCurrency(t *testing.T) {
	leaderboard := &stubLeaderboardService{quotes: testQuotes()}
	server := newTestServer(leaderboard, &stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()})

	recorder := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices?convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	// Estimates are computed from the usd Passthrough rows, not the eur ones
	assert.Equal(t, "usd", leaderboard.requestedCurrency)

	quotes := decodeQuotes(t, recorder)
	require.Contains(t, quotes, "bitcoin")
	assert.InDelta(t, 90000.0, quotes["bitcoin"].Price, 1e-9)
	assert.InDelta(t, 45000000.0, quotes["bitcoin"].Volume24h, 1e-3)
	assert.InDelta(t, 1800000000.0, quotes["bitcoin"].MarketCap, 1e-3)
	assert.InDelta(t, 20.0, quotes["bitcoin"].PercentChange24h, 1e-9)

	// cached Passthrough quotes untouched
	assert.Equal(t, 100000.0, leaderboard.quotes["usd"]["bitcoin"].Price)
}

func TestHandleLeaderboardPrices_UnknownCurrencyReturns400(t *testing.T) {
	server := newTestServer(
		&stubLeaderboardService{quotes: testQuotes()},
		&stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()},
	)

	recorder := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices?convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "unsupported convert_currency: xyz", body["error"])
}

func TestHandleLeaderboardPrices_EmptyResponseBeforeFirstSnapshot(t *testing.T) {
	server := newTestServer(
		&stubLeaderboardService{quotes: testQuotes()},
		&stubRatiosProvider{supported: supportedCurrencies(), snapshot: nil},
	)

	recorder := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices?convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{}`, recorder.Body.String())
}

func TestHandleLeaderboardPrices_ConvertToBaseCurrencyPassesValuesThrough(t *testing.T) {
	server := newTestServer(
		&stubLeaderboardService{quotes: testQuotes()},
		&stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()},
	)

	recorder := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices?convert_currency=usd")

	assert.Equal(t, http.StatusOK, recorder.Code)
	quotes := decodeQuotes(t, recorder)
	assert.Equal(t, map[string]coingecko_leaderboard.Quote(testQuotes()["usd"]), quotes)
}

func TestHandleLeaderboardSimplePrices_ConvertCurrency(t *testing.T) {
	leaderboard := &stubLeaderboardService{quotes: testQuotes()}
	server := newTestServer(leaderboard, &stubRatiosProvider{supported: supportedCurrencies(), snapshot: readySnapshot()})

	recorder := doRequest(t, server.handleLeaderboardSimplePrices, "/api/v1/leaderboard/simpleprices?convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.InDelta(t, 90000.0, decodeQuotes(t, recorder)["bitcoin"].Price, 1e-9)
}
