package coingecko_exchange_rates

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ratesBody = `{"rates":{"btc":{"name":"Bitcoin","unit":"BTC","value":1.0,"type":"crypto"}}}`

// stubClient is a scripted IClient
type stubClient struct {
	mu    sync.Mutex
	body  string
	err   error
	calls int
}

func (c *stubClient) FetchExchangeRates() (ExchangeRatesResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	return ExchangeRatesResponse(c.body), nil
}

func (c *stubClient) Healthy() bool { return c.err == nil }

func (c *stubClient) setErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = err
}

func (c *stubClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func newTestService(client IClient, interval time.Duration) *Service {
	return NewServiceWithClient(
		config.ExchangeRatesFetcherConfig{UpdateInterval: interval},
		client,
		metrics.NewMetricsWriter(metrics.ServiceExchangeRates),
	)
}

func TestService_ErrorsBeforeFirstFetch(t *testing.T) {
	service := newTestService(&stubClient{body: ratesBody}, time.Minute)

	rates, err := service.ExchangeRates()
	assert.Error(t, err)
	assert.Nil(t, rates)
	assert.False(t, service.Healthy())
}

func TestService_ServesBodyVerbatim(t *testing.T) {
	service := newTestService(&stubClient{body: ratesBody}, time.Minute)

	require.NoError(t, service.fetchAndUpdate())

	rates, err := service.ExchangeRates()
	require.NoError(t, err)
	assert.Equal(t, ratesBody, string(rates))
	assert.True(t, service.Healthy())
}

func TestService_KeepsPreviousBodyOnFailure(t *testing.T) {
	client := &stubClient{body: ratesBody}
	service := newTestService(client, time.Minute)

	require.NoError(t, service.fetchAndUpdate())

	client.setErr(fmt.Errorf("upstream is down"))
	assert.Error(t, service.fetchAndUpdate())

	rates, err := service.ExchangeRates()
	require.NoError(t, err)
	assert.Equal(t, ratesBody, string(rates))
}

func TestService_StartFetchesImmediately(t *testing.T) {
	client := &stubClient{body: ratesBody}
	service := newTestService(client, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, service.Start(ctx))
	defer service.Stop()

	assert.Eventually(t, service.Healthy, 2*time.Second, 10*time.Millisecond)
}

func TestService_StartDisabledWhenIntervalNotPositive(t *testing.T) {
	client := &stubClient{body: ratesBody}
	service := newTestService(client, 0)

	require.NoError(t, service.Start(context.Background()))
	service.Stop()

	assert.Equal(t, 0, client.callCount())
	assert.False(t, service.Healthy())
}
