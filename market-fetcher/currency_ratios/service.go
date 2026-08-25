package currency_ratios

import (
	"context"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// Service keeps an in-memory snapshot of currency Ratios, refreshed from a
// single CoinGecko simple/price call for the configured reference coins.
//
// On a failed refresh the previous snapshot is kept and served indefinitely -
// stale ratios are better than failing the whole market tab. The snapshot age
// is exported as a Prometheus metric so staleness stays visible.
type Service struct {
	config        config.CurrencyRatiosConfig
	client        IClient
	metricsWriter *metrics.MetricsWriter
	scheduler     *scheduler.Scheduler
	snapshot      atomic.Pointer[Snapshot]
	history       *ratioHistory
	supported     map[string]struct{}
}

// NewService creates a currency ratios service
func NewService(cfg *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios)
	return NewServiceWithClient(cfg.CurrencyRatios, NewCoinGeckoClient(cfg, metricsWriter), metricsWriter)
}

// NewServiceWithClient creates a currency ratios service with an explicit client
func NewServiceWithClient(cfg config.CurrencyRatiosConfig, client IClient, metricsWriter *metrics.MetricsWriter) *Service {
	supported := make(map[string]struct{}, len(cfg.Currencies))
	for _, currency := range cfg.Currencies {
		supported[strings.ToLower(currency)] = struct{}{}
	}

	return &Service{
		config:        cfg,
		client:        client,
		metricsWriter: metricsWriter,
		history:       newRatioHistory(historyCapacity(cfg.UpdateInterval)),
		supported:     supported,
	}
}

// Start begins periodic ratio updates
func (s *Service) Start(ctx context.Context) error {
	if s.config.UpdateInterval <= 0 {
		log.Printf("Currency ratios service: periodic updates disabled (interval: %v)", s.config.UpdateInterval)
		return nil
	}

	s.scheduler = scheduler.New(s.config.UpdateInterval, func(ctx context.Context) {
		if err := s.fetchAndUpdate(); err != nil {
			log.Printf("Currency ratios service: update failed, keeping previous snapshot: %v", err)
		}
	})

	s.scheduler.Start(ctx, true)

	return nil
}

// Stop terminates periodic updates
func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

// GetSnapshot returns the latest successful snapshot, or nil if none exists yet
func (s *Service) GetSnapshot() *Snapshot {
	return s.snapshot.Load()
}

// IsCurrencySupported reports whether the currency is in the configured list
func (s *Service) IsCurrencySupported(currency string) bool {
	_, ok := s.supported[strings.ToLower(currency)]
	return ok
}

// GetSpotRatioAgo returns the spot ratio the currency had approximately `ago`
// before now.
//
// It reports ok=false unless the retained history actually reaches back that far,
// so callers can leave a value unconverted rather than silently substituting the
// current ratio for a past one.
func (s *Service) GetSpotRatioAgo(currency string, ago time.Duration) (float64, bool) {
	if ago <= 0 {
		ratio, ok := s.GetSnapshot().Ratio(currency)
		return ratio.Now, ok
	}

	target := time.Now().Add(-ago)

	// The history must cover the requested window. One update interval of slack
	// keeps a snapshot taken just short of the target usable.
	oldest := s.history.oldest()
	if oldest == nil || oldest.UpdatedAt.After(target.Add(s.historySlack())) {
		return 0, false
	}

	snapshot := s.history.closestTo(target)
	if snapshot == nil {
		return 0, false
	}

	ratio, ok := snapshot.Ratio(currency)
	if !ok {
		return 0, false
	}

	return ratio.Now, true
}

// historySlack is how much younger than the requested age the oldest snapshot
// may be and still count as covering the window
func (s *Service) historySlack() time.Duration {
	if s.config.UpdateInterval > 0 {
		return s.config.UpdateInterval
	}
	return time.Minute
}

// Healthy reports whether a snapshot is available
func (s *Service) Healthy() bool {
	return s.GetSnapshot() != nil
}

// fetchAndUpdate refreshes the ratios snapshot
func (s *Service) fetchAndUpdate() error {
	s.metricsWriter.ResetCycleMetrics()
	defer s.metricsWriter.TrackDataFetchCycle()()

	payload, err := s.client.FetchReferencePrices(s.config.ReferenceCoins, s.config.Currencies)
	if err != nil {
		return err
	}

	snapshot, err := ComputeRatios(payload, s.config.ReferenceCoins, s.config.Currencies)
	if err != nil {
		return err
	}

	s.snapshot.Store(snapshot)
	s.history.add(snapshot)
	metrics.RecordCurrencyRatiosSnapshotTime(snapshot.UpdatedAt)
	s.metricsWriter.RecordCacheSize(len(snapshot.Ratios))

	log.Printf("Currency ratios updated from %s: %d/%d currencies",
		snapshot.ReferenceCoin, len(snapshot.Ratios), len(s.config.Currencies))

	return nil
}
