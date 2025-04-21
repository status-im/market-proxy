package metrics

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// FetchDurationHistogram tracks the duration of fetch operations
	FetchDurationHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "fetch_duration_seconds",
			Help: "Time taken to fetch data from external APIs",
		},
		[]string{"service", "operation"},
	)
)

// RecordFetchMarketDataCycle measures and records the duration of a market data fetch cycle
func RecordFetchMarketDataCycle(service string, start time.Time) {
	duration := time.Since(start)
	FetchDurationHistogram.WithLabelValues(service, "fetchAndUpdate").Observe(duration.Seconds())
	log.Printf("Metrics: %s fetchAndUpdate took %.2fs", service, duration.Seconds())
}
