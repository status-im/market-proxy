package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubMarketsService records the params the handler built and replays a fixed
// answer. Conversion is the service's job and is covered by
// coingecko_markets/service_convert_test.go.
type stubMarketsService struct {
	response   interfaces.MarketsResponse
	lastParams interfaces.MarketsParams
	estimated  map[string]bool
}

func (s *stubMarketsService) TopMarkets(int, string) (interfaces.MarketsResponse, error) {
	return s.response, nil
}

func (s *stubMarketsService) TopMarketIds(int) ([]string, error) { return nil, nil }

func (s *stubMarketsService) Markets(params interfaces.MarketsParams) (interfaces.MarketsResponse, interfaces.CacheStatus, error) {
	s.lastParams = params
	return s.response, interfaces.CacheStatusFull, nil
}

func (s *stubMarketsService) SubscribeTopMarketsUpdate() events.ISubscription { return nil }
func (s *stubMarketsService) SubscribeInitialized() events.ISubscription      { return nil }
func (s *stubMarketsService) Healthy() bool                                   { return true }

// estimated lists the currencies this stub claims to compute rather than pass through
func (s *stubMarketsService) EstimatesCurrency(currency string) bool {
	return s.estimated[currency]
}

// stubPricesService records the params the handler built
type stubPricesService struct {
	response   interfaces.SimplePriceResponse
	lastParams interfaces.PriceParams
	estimated  map[string]bool
}

func (s *stubPricesService) SimplePrices(_ context.Context, params interfaces.PriceParams) (interfaces.SimplePriceResponse, interfaces.CacheStatus, error) {
	s.lastParams = params
	return s.response, interfaces.CacheStatusFull, nil
}

func (s *stubPricesService) TopPrices(context.Context, int, []string) (interfaces.SimplePriceResponse, interfaces.CacheStatus, error) {
	return s.response, interfaces.CacheStatusFull, nil
}

func (s *stubPricesService) SubscribeTopPricesUpdate() events.ISubscription { return nil }
func (s *stubPricesService) Healthy() bool                                  { return true }

func (s *stubPricesService) EstimatesCurrency(currency string) bool {
	return s.estimated[currency]
}

func marketsRow() map[string]interface{} {
	return map[string]interface{}{
		"id":            "bitcoin",
		"current_price": 100000.0,
	}
}

func simplePriceRow() map[string]interface{} {
	return map[string]interface{}{"usd": 100000.0}
}

func newCoinsTestServer(markets *stubMarketsService, prices *stubPricesService, ratios interfaces.ICurrencyRatiosProvider) *Server {
	return &Server{marketsService: markets, pricesService: prices, currencyRatiosService: ratios}
}

// --- /api/v1/coins/markets ---

func TestHandleCoinsMarkets_ForwardsParams(t *testing.T) {
	tests := []struct {
		name                    string
		target                  string
		expectedCurrency        string
		expectedConvertCurrency string
	}{
		{
			name:             "vs_currency keeps its passthrough meaning",
			target:           "/api/v1/coins/markets?ids=bitcoin&vs_currency=eur",
			expectedCurrency: "eur",
		},
		{
			name:                    "convert_currency is forwarded",
			target:                  "/api/v1/coins/markets?ids=bitcoin&convert_currency=eur",
			expectedConvertCurrency: "eur",
		},
		{
			name:                    "both are forwarded; the service decides what to do with them",
			target:                  "/api/v1/coins/markets?ids=bitcoin&vs_currency=jpy&convert_currency=eur",
			expectedCurrency:        "jpy",
			expectedConvertCurrency: "eur",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
			server := newCoinsTestServer(markets, nil, readyRatiosProvider())

			recorder := doRequest(t, server.handleCoinsMarkets, tt.target)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tt.expectedCurrency, markets.lastParams.Currency)
			assert.Equal(t, tt.expectedConvertCurrency, markets.lastParams.ConvertCurrency)
		})
	}
}

func TestHandleCoinsMarkets_UnknownCurrencyReturns400(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	server := newCoinsTestServer(markets, nil, readyRatiosProvider())

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "unsupported convert_currency: xyz", decodeErrorBody(t, recorder)["error"])
	assert.Empty(t, markets.lastParams.IDs, "a rejected request never reaches the service")
}

// An empty response from the service - empty cache, or a conversion with no
// Ratio yet - stays the endpoint's empty shape.
func TestHandleCoinsMarkets_EmptyResponseStaysAnArray(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{}}
	server := newCoinsTestServer(markets, nil, readyRatiosProvider())

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `[]`, recorder.Body.String())
}

// --- /api/v1/simple/price ---

func TestHandleSimplePrice_ForwardsParams(t *testing.T) {
	tests := []struct {
		name                    string
		target                  string
		expectedCurrencies      []string
		expectedConvertCurrency string
	}{
		{
			name:               "vs_currencies only",
			target:             "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd",
			expectedCurrencies: []string{"usd"},
		},
		{
			name:                    "convert_currency is forwarded untouched",
			target:                  "/api/v1/simple/price?ids=bitcoin&vs_currencies=chf&convert_currency=eur",
			expectedCurrencies:      []string{"chf"},
			expectedConvertCurrency: "eur",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
			server := newCoinsTestServer(nil, prices, readyRatiosProvider())

			recorder := doRequest(t, server.handleSimplePrice, tt.target)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tt.expectedCurrencies, prices.lastParams.Currencies)
			assert.Equal(t, tt.expectedConvertCurrency, prices.lastParams.ConvertCurrency)
		})
	}
}

// Naming the same currency twice used to be a 400, which forced the client to
// mirror the proxy's cached-currency configuration. It is now a duplicate the
// service resolves.
func TestHandleSimplePrice_SameCurrencyInBothParamsIsAccepted(t *testing.T) {
	for _, target := range []string{
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd,eur&convert_currency=eur",
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=usd",
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=EUR&convert_currency=eur",
	} {
		prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
		server := newCoinsTestServer(nil, prices, readyRatiosProvider())

		recorder := doRequest(t, server.handleSimplePrice, target)

		assert.Equal(t, http.StatusOK, recorder.Code, "%s should be accepted", target)
		assert.Equal(t, []string{"bitcoin"}, prices.lastParams.IDs, "the request reaches the service")
	}
}

// --- X-Estimated-Currencies ---

func TestHandleSimplePrice_HeaderNamesOnlyComputedCurrencies(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		estimated      map[string]bool
		expectedHeader string
	}{
		{
			name:           "computed currency is named",
			target:         "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=chf",
			estimated:      map[string]bool{"chf": true},
			expectedHeader: "chf",
		},
		{
			name:      "provider currency is not named",
			target:    "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=eur",
			estimated: map[string]bool{"chf": true},
		},
		{
			name:      "no conversion requested",
			target:    "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd",
			estimated: map[string]bool{"chf": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prices := &stubPricesService{
				response:  interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()},
				estimated: tt.estimated,
			}
			server := newCoinsTestServer(nil, prices, readyRatiosProvider())

			recorder := doRequest(t, server.handleSimplePrice, tt.target)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, tt.expectedHeader, recorder.Header().Get(estimatedCurrenciesHeader),
				"an empty expectation means the header must be absent")
		})
	}
}

func TestHandleCoinsMarkets_HeaderNamesOnlyComputedCurrencies(t *testing.T) {
	markets := &stubMarketsService{
		response:  interfaces.MarketsResponse{marketsRow()},
		estimated: map[string]bool{"chf": true},
	}
	server := newCoinsTestServer(markets, nil, readyRatiosProvider())

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=chf")
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "chf", recorder.Header().Get(estimatedCurrenciesHeader))

	recorder = doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=usd")
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Header().Get(estimatedCurrenciesHeader),
		"a currency the cache already holds is provider data")
}

func TestHandleSimplePrice_UnknownCurrencyReturns400(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatiosProvider())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "unsupported convert_currency: xyz", decodeErrorBody(t, recorder)["error"])
}

func TestHandleSimplePrice_EmptyResponseStaysAnObject(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{}}
	server := newCoinsTestServer(nil, prices, readyRatiosProvider())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{}`, recorder.Body.String())
}

func TestHandleSimplePrice_RequiredParams(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatiosProvider())

	recorder := doRequest(t, server.handleSimplePrice, "/api/v1/simple/price?vs_currencies=usd&convert_currency=eur")
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "ids is still required")

	recorder = doRequest(t, server.handleSimplePrice, "/api/v1/simple/price?ids=bitcoin")
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "naming no currency at all is still a bad request")

	// convert_currency names a currency to answer in, so it satisfies the
	// requirement on its own
	recorder = doRequest(t, server.handleSimplePrice, "/api/v1/simple/price?ids=bitcoin&convert_currency=eur")
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, prices.lastParams.Currencies)
	assert.Equal(t, "eur", prices.lastParams.ConvertCurrency)
}

func TestHandleSimplePrice_ResponseIsServedUnchanged(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{
		"bitcoin": map[string]interface{}{"usd": 100000.0, "eur": 90000.0},
	}}
	server := newCoinsTestServer(nil, prices, readyRatiosProvider())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=eur")

	require.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]map[string]float64
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, 100000.0, response["bitcoin"]["usd"])
	assert.Equal(t, 90000.0, response["bitcoin"]["eur"])
}
