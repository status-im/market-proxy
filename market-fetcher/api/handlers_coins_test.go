package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/status-im/market-proxy/coingecko_markets"
	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubMarketsService serves a fixed markets response and records the params it saw
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

// stubPricesService serves a fixed simple/price response and records the params it saw
type stubPricesService struct {
	response   interfaces.SimplePriceResponse
	lastParams interfaces.PriceParams
}

func (s *stubPricesService) SimplePrices(_ context.Context, params interfaces.PriceParams) (interfaces.SimplePriceResponse, interfaces.CacheStatus, error) {
	s.lastParams = params

	// Mimic stripResponse: only the requested currencies survive
	requested := map[string]bool{}
	for _, currency := range params.Currencies {
		requested[currency] = true
	}

	filtered := make(interfaces.SimplePriceResponse, len(s.response))
	for tokenID, row := range s.response {
		source, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		result := map[string]interface{}{}
		for key, value := range source {
			if key == "last_updated_at" || requested[currencyOfKey(key)] {
				result[key] = value
			}
		}
		filtered[tokenID] = result
	}

	return filtered, interfaces.CacheStatusFull, nil
}

// currencyOfKey extracts the currency prefix of a simple/price field
func currencyOfKey(key string) string {
	for _, suffix := range []string{"_market_cap", "_24h_vol", "_24h_change"} {
		if len(key) > len(suffix) && key[len(key)-len(suffix):] == suffix {
			return key[:len(key)-len(suffix)]
		}
	}
	return key
}

func (s *stubPricesService) TopPrices(context.Context, int, []string) (interfaces.SimplePriceResponse, interfaces.CacheStatus, error) {
	return s.response, interfaces.CacheStatusFull, nil
}

func (s *stubPricesService) SubscribeTopPricesUpdate() events.ISubscription { return nil }
func (s *stubPricesService) Healthy() bool                                  { return true }

func marketsRow() map[string]interface{} {
	return map[string]interface{}{
		"id":                                     "bitcoin",
		"symbol":                                 "btc",
		"current_price":                          100000.0,
		"market_cap":                             2000000000.0,
		"high_24h":                               101000.0,
		"low_24h":                                98000.0,
		"price_change_24h":                       5000.0,
		"price_change_percentage_24h":            10.0,
		"market_cap_rank":                        1.0,
		"price_change_percentage_1h_in_currency": 2.0,
		"price_change_percentage_24h_in_currency": 10.0,
	}
}

func simplePriceRow() map[string]interface{} {
	return map[string]interface{}{
		"usd":             100000.0,
		"usd_market_cap":  2000000000.0,
		"usd_24h_vol":     50000000.0,
		"usd_24h_change":  10.0,
		"chf":             88000.0,
		"last_updated_at": 1703097600.0,
	}
}

func newCoinsTestServer(markets *stubMarketsService, prices *stubPricesService, ratios currency_ratios.IProvider) *Server {
	return &Server{marketsService: markets, pricesService: prices, currencyRatiosService: ratios}
}

func readyRatios() *stubRatiosProvider {
	return &stubRatiosProvider{
		supported: supportedCurrencies(),
		snapshot:  readySnapshot(),
	}
}

// --- /api/v1/coins/markets ---

func TestHandleCoinsMarkets_NoConversionIsUntouched(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	server := newCoinsTestServer(markets, nil, readyRatios())

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?vs_currency=eur&ids=bitcoin")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "eur", markets.lastParams.Currency, "vs_currency keeps its passthrough meaning")

	var rows []map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, 100000.0, rows[0]["current_price"])
}

func TestHandleCoinsMarkets_ConvertCurrency(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	server := newCoinsTestServer(markets, nil, readyRatios())

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=eur")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "usd", markets.lastParams.Currency,
		"conversion always reads the base currency rows")

	var rows []map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rows))
	require.Len(t, rows, 1)

	row := rows[0]
	assert.InDelta(t, 90000.0, row["current_price"], 1e-9)
	assert.InDelta(t, 1800000000.0, row["market_cap"], 1e-3)
	assert.InDelta(t, 90900.0, row["high_24h"], 1e-6)
	assert.InDelta(t, 88200.0, row["low_24h"], 1e-6)
	assert.InDelta(t, 11625.0, row["price_change_24h"], 1e-6)
	assert.InDelta(t, 20.0, row["price_change_percentage_24h"], 1e-9)
	assert.Equal(t, 1.0, row["market_cap_rank"])

	// no 1h history in this provider, so the 1h change is left alone
	assert.Equal(t, 2.0, row[coingecko_markets.PercentChange1hField])

	// the source row is untouched
	assert.Equal(t, 100000.0, markets.response[0].(map[string]interface{})["current_price"])
}

// TestHandleCoinsMarkets_ConvertCurrencyIgnoresVsCurrency documents that
// vs_currency carries no information once convert_currency is set: cached rows
// are always normalized to the base currency.
func TestHandleCoinsMarkets_ConvertCurrencyIgnoresVsCurrency(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	server := newCoinsTestServer(markets, nil, readyRatios())

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&vs_currency=jpy&convert_currency=eur")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "usd", markets.lastParams.Currency)
}

func TestHandleCoinsMarkets_ConvertCurrencyWith1hHistory(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	ratios := readyRatios()
	ratios.spotAgo = map[string]float64{"eur": 0.88}

	server := newCoinsTestServer(markets, nil, ratios)

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=eur")
	require.Equal(t, http.StatusOK, recorder.Code)

	var rows []map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rows))

	// ((1 + 0.02) * 0.9 / 0.88 - 1) * 100
	assert.InDelta(t, (1.02*0.9/0.88-1)*100, rows[0][coingecko_markets.PercentChange1hField], 1e-9)
}

func TestHandleCoinsMarkets_ConvertToBaseCurrencyPassesValuesThrough(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	ratios := readyRatios()
	ratios.spotAgo = map[string]float64{"usd": 1}

	server := newCoinsTestServer(markets, nil, ratios)

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=usd")
	require.Equal(t, http.StatusOK, recorder.Code)

	var rows []map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &rows))
	require.Len(t, rows, 1)

	assert.Equal(t, marketsRow(), rows[0])
}

func TestHandleCoinsMarkets_UnknownCurrencyReturns400(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	server := newCoinsTestServer(markets, nil, readyRatios())

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "unsupported convert_currency: xyz", body["error"])
}

func TestHandleCoinsMarkets_EmptyResponseBeforeFirstSnapshot(t *testing.T) {
	markets := &stubMarketsService{response: interfaces.MarketsResponse{marketsRow()}}
	server := newCoinsTestServer(markets, nil, &stubRatiosProvider{supported: supportedCurrencies()})

	recorder := doRequest(t, server.handleCoinsMarkets, "/api/v1/coins/markets?ids=bitcoin&convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `[]`, recorder.Body.String())
}

// --- /api/v1/simple/price ---

func TestHandleSimplePrice_NoConversionIsUntouched(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatios())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&include_market_cap=true")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, []string{"usd"}, prices.lastParams.Currencies)

	var response map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, 100000.0, response["bitcoin"]["usd"])
	assert.NotContains(t, response["bitcoin"], "eur")
}

func TestHandleSimplePrice_ConvertCurrencyAddsEstimateKeys(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatios())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=eur&include_market_cap=true&include_24hr_vol=true&include_24hr_change=true")

	require.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	row := response["bitcoin"]
	// passthrough keys survive
	assert.Equal(t, 100000.0, row["usd"])
	assert.Equal(t, 2000000000.0, row["usd_market_cap"])
	assert.Equal(t, 10.0, row["usd_24h_change"])

	// estimate keys are added
	assert.InDelta(t, 90000.0, row["eur"], 1e-9)
	assert.InDelta(t, 1800000000.0, row["eur_market_cap"], 1e-3)
	assert.InDelta(t, 45000000.0, row["eur_24h_vol"], 1e-3)
	assert.InDelta(t, 20.0, row["eur_24h_change"], 1e-9)

	assert.Equal(t, 1703097600.0, row["last_updated_at"])
}

// TestHandleSimplePrice_ConvertCurrencyWithoutBaseRequested covers
// vs_currencies=chf&convert_currency=eur: usd is read in to compute the Estimate
// but must not appear in the response.
func TestHandleSimplePrice_ConvertCurrencyWithoutBaseRequested(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatios())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=chf&convert_currency=eur")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, []string{"chf", "usd"}, prices.lastParams.Currencies,
		"the base currency is added to the cache read")

	var response map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	row := response["bitcoin"]
	assert.Equal(t, 88000.0, row["chf"])
	assert.InDelta(t, 90000.0, row["eur"], 1e-9)
	assert.NotContains(t, row, "usd", "the base currency was not requested")
	assert.NotContains(t, row, "usd_market_cap")
}

func TestHandleSimplePrice_ConvertToBaseCurrencyPassesValuesThrough(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatios())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=chf&convert_currency=usd&include_market_cap=true&include_24hr_change=true")

	require.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	row := response["bitcoin"]
	assert.Equal(t, 100000.0, row["usd"])
	assert.Equal(t, 2000000000.0, row["usd_market_cap"])
	assert.Equal(t, 10.0, row["usd_24h_change"])
	assert.Equal(t, 88000.0, row["chf"])
}

func TestHandleSimplePrice_AmbiguousCurrencyReturns400(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatios())

	for _, target := range []string{
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd,eur&convert_currency=eur",
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=usd",
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=EUR&convert_currency=eur",
	} {
		recorder := doRequest(t, server.handleSimplePrice, target)

		assert.Equal(t, http.StatusBadRequest, recorder.Code, "%s should be rejected", target)

		var body map[string]string
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		assert.Contains(t, body["error"], "must not be listed in vs_currencies")
	}
}

// The ambiguity check must win even before a snapshot exists, so a bad request
// never gets a 200 with an empty body.
func TestHandleSimplePrice_AmbiguousCurrencyReturns400BeforeFirstSnapshot(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, &stubRatiosProvider{supported: supportedCurrencies()})

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=usd")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestHandleSimplePrice_UnknownCurrencyReturns400(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatios())

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=xyz")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "unsupported convert_currency: xyz", body["error"])
}

func TestHandleSimplePrice_EmptyResponseBeforeFirstSnapshot(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, &stubRatiosProvider{supported: supportedCurrencies()})

	recorder := doRequest(t, server.handleSimplePrice,
		"/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=eur")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{}`, recorder.Body.String())
}

func TestHandleSimplePrice_RequiredParamsStillEnforced(t *testing.T) {
	prices := &stubPricesService{response: interfaces.SimplePriceResponse{"bitcoin": simplePriceRow()}}
	server := newCoinsTestServer(nil, prices, readyRatios())

	recorder := doRequest(t, server.handleSimplePrice, "/api/v1/simple/price?vs_currencies=usd&convert_currency=eur")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	recorder = doRequest(t, server.handleSimplePrice, "/api/v1/simple/price?ids=bitcoin&convert_currency=eur")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
