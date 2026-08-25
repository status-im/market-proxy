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

// stubPricesService records the params the handler built
type stubPricesService struct {
	response   interfaces.SimplePriceResponse
	lastParams interfaces.PriceParams
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

// One key cannot be both Passthrough and Estimate, so the combination is
// rejected before it reaches the service.
func TestHandleSimplePrice_AmbiguousCurrencyReturns400(t *testing.T) {
	for _, target := range []string{
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd,eur&convert_currency=eur",
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=usd",
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=EUR&convert_currency=eur",
	} {
		prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
		server := newCoinsTestServer(nil, prices, readyRatiosProvider())

		recorder := doRequest(t, server.handleSimplePrice, target)

		assert.Equal(t, http.StatusBadRequest, recorder.Code, "%s should be rejected", target)
		assert.Contains(t, decodeErrorBody(t, recorder)["error"], "must not be listed in vs_currencies")
		assert.Empty(t, prices.lastParams.IDs, "a rejected request never reaches the service")
	}
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

func TestHandleSimplePrice_RequiredParamsStillEnforced(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatiosProvider())

	recorder := doRequest(t, server.handleSimplePrice, "/api/v1/simple/price?vs_currencies=usd&convert_currency=eur")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	recorder = doRequest(t, server.handleSimplePrice, "/api/v1/simple/price?ids=bitcoin&convert_currency=eur")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
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
