package currency_ratios

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubClient is a scripted IClient: each call pops the next scripted result
type stubClient struct {
	mu       sync.Mutex
	payloads []SimplePricePayload
	errors   []error
	calls    int
}

func (c *stubClient) FetchReferencePrices(_ []string, _ []string) (SimplePricePayload, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	index := c.calls
	c.calls++

	if index < len(c.errors) && c.errors[index] != nil {
		return nil, c.errors[index]
	}
	if index < len(c.payloads) {
		return c.payloads[index], nil
	}
	return nil, fmt.Errorf("no scripted result for call %d", index)
}

func (c *stubClient) Healthy() bool { return true }

func (c *stubClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func decodePayload(t *testing.T, raw string) SimplePricePayload {
	t.Helper()
	var payload SimplePricePayload
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	return payload
}

func testConfig() config.CurrencyRatiosConfig {
	return config.CurrencyRatiosConfig{
		UpdateInterval: time.Minute,
		ReferenceCoins: []string{"bitcoin", "ethereum"},
		Currencies:     []string{"usd", "eur", "btc"},
	}
}

func newTestService(client IClient) *Service {
	return NewServiceWithClient(testConfig(), client, metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios))
}

func TestService_NoSnapshotBeforeFirstFetch(t *testing.T) {
	service := newTestService(&stubClient{})

	assert.Nil(t, service.GetSnapshot())
	assert.False(t, service.Healthy())

	ratio, ok := service.GetSnapshot().Ratio("eur")
	assert.False(t, ok)
	assert.Equal(t, Ratio{}, ratio)
}

func TestService_IsCurrencySupported(t *testing.T) {
	service := newTestService(&stubClient{})

	assert.True(t, service.IsCurrencySupported("usd"))
	assert.True(t, service.IsCurrencySupported("eur"))
	assert.True(t, service.IsCurrencySupported("btc"))
	assert.False(t, service.IsCurrencySupported("xyz"))
	assert.False(t, service.IsCurrencySupported(""))

	// The list is matched case-insensitively, even though handlers lowercase first
	assert.True(t, service.IsCurrencySupported("EUR"))
}

func TestService_FetchAndUpdate_StoresSnapshot(t *testing.T) {
	client := &stubClient{payloads: []SimplePricePayload{decodePayload(t, samplePayload)}}
	service := newTestService(client)

	require.NoError(t, service.fetchAndUpdate())

	snapshot := service.GetSnapshot()
	require.NotNil(t, snapshot)
	assert.Equal(t, "bitcoin", snapshot.ReferenceCoin)
	assert.InDelta(t, 0.9, snapshot.Ratios["eur"].Now, 1e-12)
	assert.True(t, service.Healthy())
}

// TestService_KeepsPreviousSnapshotOnFailure is the ADR's availability rule:
// on upstream failure the last known Ratio is served indefinitely.
func TestService_KeepsPreviousSnapshotOnFailure(t *testing.T) {
	client := &stubClient{
		payloads: []SimplePricePayload{decodePayload(t, samplePayload), nil, nil},
		errors:   []error{nil, fmt.Errorf("upstream is down"), nil},
	}
	service := newTestService(client)

	require.NoError(t, service.fetchAndUpdate())
	first := service.GetSnapshot()
	require.NotNil(t, first)

	// Fetch error keeps the snapshot
	assert.Error(t, service.fetchAndUpdate())
	assert.Same(t, first, service.GetSnapshot())

	// A payload with no usable reference row also keeps the snapshot
	client.payloads[2] = decodePayload(t, `{"dogecoin": {"usd": 0.4, "usd_24h_change": 1.0}}`)
	assert.Error(t, service.fetchAndUpdate())
	assert.Same(t, first, service.GetSnapshot())
}

func TestService_StartRunsImmediatelyAndStops(t *testing.T) {
	client := &stubClient{payloads: []SimplePricePayload{
		decodePayload(t, samplePayload),
		decodePayload(t, samplePayload),
	}}

	cfg := testConfig()
	cfg.UpdateInterval = 50 * time.Millisecond
	service := NewServiceWithClient(cfg, client, metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, service.Start(ctx))
	defer service.Stop()

	assert.Eventually(t, func() bool {
		return service.GetSnapshot() != nil
	}, 2*time.Second, 10*time.Millisecond)

	assert.GreaterOrEqual(t, client.callCount(), 1)
}

func TestService_StartDisabledWhenIntervalNotPositive(t *testing.T) {
	client := &stubClient{payloads: []SimplePricePayload{decodePayload(t, samplePayload)}}

	cfg := testConfig()
	cfg.UpdateInterval = 0
	service := NewServiceWithClient(cfg, client, metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios))

	require.NoError(t, service.Start(context.Background()))
	service.Stop()

	assert.Equal(t, 0, client.callCount())
	assert.Nil(t, service.GetSnapshot())
}
