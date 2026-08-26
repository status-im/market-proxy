package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubLeaderboardService records what the handler asked for and replays a fixed
// answer. The conversion itself is the service's job and is covered by
// coingecko_leaderboard/service_convert_test.go; these tests only check that the
// handler parses, validates and maps to HTTP correctly.
type stubLeaderboardService struct {
	markets *interfaces.LeaderboardResponse
	quotes  interfaces.LeaderboardQuotes

	estimated map[string]bool

	gotMarketsConvertCurrency string
	gotPricesCurrency         string
	gotPricesConvertCurrency  string
}

func (s *stubLeaderboardService) GetCacheData(convertCurrency string) *interfaces.LeaderboardResponse {
	s.gotMarketsConvertCurrency = convertCurrency
	return s.markets
}

func (s *stubLeaderboardService) GetTopPricesQuotes(currency string, convertCurrency string) interfaces.LeaderboardQuotes {
	s.gotPricesCurrency = currency
	s.gotPricesConvertCurrency = convertCurrency
	return s.quotes
}

func (s *stubLeaderboardService) Healthy() bool { return true }

// estimated lists the currencies this stub claims to compute rather than pass through
func (s *stubLeaderboardService) EstimatesCurrency(currency string) bool {
	return s.estimated[currency]
}

// stubRatiosProvider is an ICurrencyRatiosProvider with a fixed allow-list
type stubRatiosProvider struct {
	supported map[string]bool
	snapshot  *interfaces.CurrencyRatiosSnapshot
	spotAgo   map[string]float64
}

func (s *stubRatiosProvider) GetSnapshot() *interfaces.CurrencyRatiosSnapshot { return s.snapshot }

func (s *stubRatiosProvider) IsCurrencySupported(currency string) bool {
	return s.supported[currency]
}

func (s *stubRatiosProvider) GetSpotRatioAgo(currency string, _ time.Duration) (float64, bool) {
	ratio, ok := s.spotAgo[currency]
	return ratio, ok
}

// floatPtr builds the optional-value pointers the leaderboard response structs use
func floatPtr(value float64) *float64 {
	return &value
}

func supportedCurrencies() map[string]bool {
	return map[string]bool{"usd": true, "eur": true, "btc": true, "chf": true}
}

func testMarkets() *interfaces.LeaderboardResponse {
	return &interfaces.LeaderboardResponse{
		Data: []interfaces.LeaderboardCoinData{{
			ID:           "bitcoin",
			Symbol:       "btc",
			CurrentPrice: floatPtr(100000),
		}},
	}
}

func testQuotes() interfaces.LeaderboardQuotes {
	return interfaces.LeaderboardQuotes{"bitcoin": {Price: 100000, PercentChange24h: floatPtr(10)}}
}

func newTestServer(leaderboard interfaces.ILeaderboardService, ratios interfaces.ICurrencyRatiosProvider) *Server {
	return &Server{cgService: leaderboard, currencyRatiosService: ratios}
}

func readyRatiosProvider() *stubRatiosProvider {
	return &stubRatiosProvider{supported: supportedCurrencies()}
}

func doRequest(t *testing.T, handler http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func decodeErrorBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

// --- markets ---

func TestHandleLeaderboardMarkets_PassesConvertCurrencyToService(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{name: "absent", target: "/api/v1/leaderboard/markets", expected: ""},
		{name: "present", target: "/api/v1/leaderboard/markets?convert_currency=eur", expected: "eur"},
		{name: "lowercased", target: "/api/v1/leaderboard/markets?convert_currency=EUR", expected: "eur"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaderboard := &stubLeaderboardService{markets: testMarkets()}
			server := newTestServer(leaderboard, readyRatiosProvider())

			recorder := doRequest(t, server.handleLeaderboardMarkets, tt.target)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tt.expected, leaderboard.gotMarketsConvertCurrency)
		})
	}
}

func TestHandleLeaderboardMarkets_UnknownCurrencyReturns400(t *testing.T) {
	leaderboard := &stubLeaderboardService{markets: testMarkets()}
	server := newTestServer(leaderboard, readyRatiosProvider())

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "unsupported convert_currency: xyz", decodeErrorBody(t, recorder)["error"])
}

// A service returning nil means "nothing to serve" - an empty cache or a
// conversion with no Ratio yet. Both map to the endpoint's empty shape.
func TestHandleLeaderboardMarkets_NilDataBecomesEmptyShape(t *testing.T) {
	server := newTestServer(&stubLeaderboardService{markets: nil}, readyRatiosProvider())

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
	leaderboard := &stubLeaderboardService{markets: testMarkets()}
	server := newTestServer(leaderboard, nil)

	recorder := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets?convert_currency=eur")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	// without the parameter the endpoint is unaffected
	recorder = doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets")
	assert.Equal(t, http.StatusOK, recorder.Code)
}

// --- prices ---

func TestHandleLeaderboardPrices_PassesBothCurrencyParamsToService(t *testing.T) {
	tests := []struct {
		name            string
		target          string
		currency        string
		convertCurrency string
	}{
		{name: "neither", target: "/api/v1/leaderboard/prices"},
		{name: "currency only", target: "/api/v1/leaderboard/prices?currency=eur", currency: "eur"},
		{name: "convert only", target: "/api/v1/leaderboard/prices?convert_currency=eur", convertCurrency: "eur"},
		{
			name:            "both are forwarded untouched",
			target:          "/api/v1/leaderboard/prices?currency=btc&convert_currency=eur",
			currency:        "btc",
			convertCurrency: "eur",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaderboard := &stubLeaderboardService{quotes: testQuotes()}
			server := newTestServer(leaderboard, readyRatiosProvider())

			recorder := doRequest(t, server.handleLeaderboardPrices, tt.target)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tt.currency, leaderboard.gotPricesCurrency)
			assert.Equal(t, tt.convertCurrency, leaderboard.gotPricesConvertCurrency)
		})
	}
}

func TestHandleLeaderboardPrices_UnknownCurrencyReturns400(t *testing.T) {
	server := newTestServer(&stubLeaderboardService{quotes: testQuotes()}, readyRatiosProvider())

	recorder := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices?convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "unsupported convert_currency: xyz", decodeErrorBody(t, recorder)["error"])
}

func TestHandleLeaderboardPrices_EmptyQuotesStayAnEmptyObject(t *testing.T) {
	for _, quotes := range []interfaces.LeaderboardQuotes{nil, {}} {
		server := newTestServer(&stubLeaderboardService{quotes: quotes}, readyRatiosProvider())

		recorder := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices?convert_currency=eur")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{}`, recorder.Body.String())
	}
}

func TestHandleLeaderboardSimplePrices_SharesTheSameHandling(t *testing.T) {
	leaderboard := &stubLeaderboardService{quotes: testQuotes()}
	server := newTestServer(leaderboard, readyRatiosProvider())

	recorder := doRequest(t, server.handleLeaderboardSimplePrices,
		"/api/v1/leaderboard/simpleprices?convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "eur", leaderboard.gotPricesConvertCurrency)
}

// --- X-Estimated-Currencies ---

func TestHandleLeaderboard_HeaderNamesOnlyComputedCurrencies(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		expectedHeader string
	}{
		{
			name:           "computed currency is named",
			target:         "?convert_currency=chf",
			expectedHeader: "chf",
		},
		{
			name:   "a currency the cache already holds is not named",
			target: "?convert_currency=usd",
		},
		{
			name:   "no conversion requested",
			target: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// the leaderboard caches one currency (usd); everything else is computed
			leaderboard := &stubLeaderboardService{
				markets:   testMarkets(),
				quotes:    testQuotes(),
				estimated: map[string]bool{"chf": true, "eur": true, "btc": true},
			}
			server := newTestServer(leaderboard, readyRatiosProvider())

			markets := doRequest(t, server.handleLeaderboardMarkets, "/api/v1/leaderboard/markets"+tt.target)
			require.Equal(t, http.StatusOK, markets.Code)
			assert.Equal(t, tt.expectedHeader, markets.Header().Get(estimatedCurrenciesHeader))

			prices := doRequest(t, server.handleLeaderboardPrices, "/api/v1/leaderboard/prices"+tt.target)
			require.Equal(t, http.StatusOK, prices.Code)
			assert.Equal(t, tt.expectedHeader, prices.Header().Get(estimatedCurrenciesHeader))
		})
	}
}

func TestSetEstimatedCurrenciesHeader(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name       string
		currencies []string
		expected   string
	}{
		{name: "none", currencies: nil},
		{name: "empty strings are dropped", currencies: []string{"", ""}},
		{name: "one", currencies: []string{"chf"}, expected: "chf"},
		{name: "several are comma separated", currencies: []string{"chf", "sek"}, expected: "chf,sek"},
		{name: "a mix keeps only the computed ones", currencies: []string{"chf", ""}, expected: "chf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.setEstimatedCurrenciesHeader(recorder, tt.currencies...)
			assert.Equal(t, tt.expected, recorder.Header().Get(estimatedCurrenciesHeader))
		})
	}
}
