package currency_ratios

import (
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func snapshotAt(t time.Time, eurNow float64) *Snapshot {
	return &Snapshot{
		Ratios: map[string]Ratio{
			"usd": IdentityRatio,
			"eur": {Now: eurNow, H24: eurNow},
		},
		ReferenceCoin: "bitcoin",
		UpdatedAt:     t,
	}
}

func TestHistoryCapacity(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		expected int
	}{
		{name: "one minute covers 70 minutes", interval: time.Minute, expected: 72},
		{name: "five minutes", interval: 5 * time.Minute, expected: 16},
		{name: "longer than retention still keeps a pair", interval: 2 * time.Hour, expected: minRatioHistoryEntries},
		{name: "zero interval falls back to the minimum", interval: 0, expected: minRatioHistoryEntries},
		{name: "negative interval falls back to the minimum", interval: -time.Minute, expected: minRatioHistoryEntries},
		{name: "tiny interval is capped", interval: time.Millisecond, expected: maxRatioHistoryEntries},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, historyCapacity(tt.interval))
		})
	}
}

func TestRatioHistory_EmptyRing(t *testing.T) {
	history := newRatioHistory(3)

	assert.Equal(t, 0, history.len())
	assert.Nil(t, history.oldest())
	assert.Nil(t, history.closestTo(time.Now()))
}

func TestRatioHistory_AddAndOldest(t *testing.T) {
	now := time.Now()
	history := newRatioHistory(3)

	for i := 0; i < 3; i++ {
		history.add(snapshotAt(now.Add(time.Duration(i)*time.Minute), 0.9))
	}

	assert.Equal(t, 3, history.len())
	require.NotNil(t, history.oldest())
	assert.Equal(t, now, history.oldest().UpdatedAt)
}

func TestRatioHistory_EvictsOldestWhenFull(t *testing.T) {
	now := time.Now()
	history := newRatioHistory(3)

	for i := 0; i < 5; i++ {
		history.add(snapshotAt(now.Add(time.Duration(i)*time.Minute), 0.9))
	}

	assert.Equal(t, 3, history.len(), "the ring never grows past its capacity")
	// entries 0 and 1 were overwritten, so the oldest retained one is at +2m
	assert.Equal(t, now.Add(2*time.Minute), history.oldest().UpdatedAt)
}

func TestRatioHistory_ClosestTo(t *testing.T) {
	now := time.Now()
	history := newRatioHistory(5)

	for i := 0; i < 5; i++ {
		history.add(snapshotAt(now.Add(time.Duration(i)*time.Minute), 0.9+0.01*float64(i)))
	}

	tests := []struct {
		name     string
		target   time.Time
		expected time.Time
	}{
		{name: "exact match", target: now.Add(2 * time.Minute), expected: now.Add(2 * time.Minute)},
		{name: "rounds to the nearer entry", target: now.Add(2*time.Minute + 20*time.Second), expected: now.Add(2 * time.Minute)},
		{name: "rounds up past the midpoint", target: now.Add(2*time.Minute + 40*time.Second), expected: now.Add(3 * time.Minute)},
		{name: "clamps before the first entry", target: now.Add(-time.Hour), expected: now},
		{name: "clamps after the last entry", target: now.Add(time.Hour), expected: now.Add(4 * time.Minute)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := history.closestTo(tt.target)
			require.NotNil(t, snapshot)
			assert.Equal(t, tt.expected, snapshot.UpdatedAt)
		})
	}
}

func TestRatioHistory_IgnoresNil(t *testing.T) {
	history := newRatioHistory(2)
	history.add(nil)
	assert.Equal(t, 0, history.len())
}

// --- service level lookup ---

func newHistoryService(t *testing.T, interval time.Duration) *Service {
	t.Helper()

	cfg := testConfig()
	cfg.UpdateInterval = interval

	return NewServiceWithClient(cfg, &stubClient{}, metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios))
}

// TestService_GetSpotRatioAgo_FallsBackUntilHistoryCoversTheWindow is the
// documented fresh-start behaviour: no 1h ratio means the 1h percent change is
// left unconverted rather than computed against the current ratio.
func TestService_GetSpotRatioAgo_FallsBackUntilHistoryCoversTheWindow(t *testing.T) {
	service := newHistoryService(t, time.Minute)
	now := time.Now()

	// nothing recorded yet
	_, ok := service.GetSpotRatioAgo("eur", time.Hour)
	assert.False(t, ok)

	// only 10 minutes of history
	for i := 10; i >= 0; i-- {
		service.history.add(snapshotAt(now.Add(-time.Duration(i)*time.Minute), 0.9))
	}
	_, ok = service.GetSpotRatioAgo("eur", time.Hour)
	assert.False(t, ok, "history that only reaches back 10 minutes cannot answer a 1h lookup")
}

func TestService_GetSpotRatioAgo_ReturnsHistoricalRatio(t *testing.T) {
	service := newHistoryService(t, time.Minute)
	now := time.Now()

	// 70 minutes of history, ratio drifting by a hundredth per minute
	for i := 70; i >= 0; i-- {
		service.history.add(snapshotAt(now.Add(-time.Duration(i)*time.Minute), 0.9-0.001*float64(i)))
	}

	ratio, ok := service.GetSpotRatioAgo("eur", time.Hour)
	require.True(t, ok)
	assert.InDelta(t, 0.9-0.001*60, ratio, 1e-9)
}

func TestService_GetSpotRatioAgo_AcceptsSlackOfOneInterval(t *testing.T) {
	service := newHistoryService(t, 5*time.Minute)
	now := time.Now()

	// the oldest entry is 57 minutes old - short of an hour, but within one
	// update interval of it
	for i := 57; i >= 0; i -= 5 {
		service.history.add(snapshotAt(now.Add(-time.Duration(i)*time.Minute), 0.9))
	}

	ratio, ok := service.GetSpotRatioAgo("eur", time.Hour)
	require.True(t, ok)
	assert.InDelta(t, 0.9, ratio, 1e-9)
}

func TestService_GetSpotRatioAgo_UnknownCurrency(t *testing.T) {
	service := newHistoryService(t, time.Minute)
	now := time.Now()

	for i := 70; i >= 0; i-- {
		service.history.add(snapshotAt(now.Add(-time.Duration(i)*time.Minute), 0.9))
	}

	_, ok := service.GetSpotRatioAgo("chf", time.Hour)
	assert.False(t, ok)
}

func TestService_GetSpotRatioAgo_ZeroAgeUsesCurrentSnapshot(t *testing.T) {
	service := newHistoryService(t, time.Minute)

	_, ok := service.GetSpotRatioAgo("eur", 0)
	assert.False(t, ok, "no current snapshot yet")

	service.snapshot.Store(snapshotAt(time.Now(), 0.9))

	ratio, ok := service.GetSpotRatioAgo("eur", 0)
	require.True(t, ok)
	assert.Equal(t, 0.9, ratio)
}

func TestService_FetchAndUpdateRecordsHistory(t *testing.T) {
	client := &stubClient{payloads: []SimplePricePayload{
		decodePayload(t, samplePayload),
		decodePayload(t, samplePayload),
	}}
	service := NewServiceWithClient(testConfig(), client, metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios))

	require.NoError(t, service.fetchAndUpdate())
	assert.Equal(t, 1, service.history.len())

	require.NoError(t, service.fetchAndUpdate())
	assert.Equal(t, 2, service.history.len())
}

func TestNewService_HistorySizedFromInterval(t *testing.T) {
	cfg := config.CurrencyRatiosConfig{
		UpdateInterval: time.Minute,
		ReferenceCoins: []string{"bitcoin"},
		Currencies:     []string{"usd"},
	}

	service := NewServiceWithClient(cfg, &stubClient{}, metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios))
	assert.Len(t, service.history.entries, 72)
}
