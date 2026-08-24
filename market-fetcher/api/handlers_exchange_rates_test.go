package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/status-im/market-proxy/coingecko_exchange_rates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubExchangeRatesService returns a fixed body or error
type stubExchangeRatesService struct {
	body coingecko_exchange_rates.ExchangeRatesResponse
	err  error
}

func (s *stubExchangeRatesService) ExchangeRates() (coingecko_exchange_rates.ExchangeRatesResponse, error) {
	return s.body, s.err
}

const exchangeRatesBody = `{"rates":{"btc":{"name":"Bitcoin","unit":"BTC","value":1.0,"type":"crypto"},` +
	`"eur":{"name":"Euro","unit":"€","value":92123.456789012345,"type":"fiat"}}}`

func TestHandleExchangeRates_ServesBodyVerbatim(t *testing.T) {
	server := &Server{exchangeRatesService: &stubExchangeRatesService{
		body: coingecko_exchange_rates.ExchangeRatesResponse(exchangeRatesBody),
	}}

	recorder := httptest.NewRecorder()
	server.handleExchangeRates(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/exchange_rates", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.NotEmpty(t, recorder.Header().Get("ETag"))

	// Passthrough: the body is byte-for-byte what the provider returned,
	// including the full precision of the values.
	assert.Equal(t, exchangeRatesBody, recorder.Body.String())

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &parsed))
	assert.Contains(t, parsed, "rates")
}

func TestHandleExchangeRates_NotAvailableYet(t *testing.T) {
	server := &Server{exchangeRatesService: &stubExchangeRatesService{
		err: fmt.Errorf("exchange rates are not available yet"),
	}}

	recorder := httptest.NewRecorder()
	server.handleExchangeRates(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/exchange_rates", nil))

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "not available yet")
}

func TestHandleExchangeRates_NoService(t *testing.T) {
	server := &Server{}

	recorder := httptest.NewRecorder()
	server.handleExchangeRates(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/exchange_rates", nil))

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
