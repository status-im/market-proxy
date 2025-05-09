package metrics

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsPrefix is the prefix used for all metrics
const MetricsPrefix = "market_fetcher_"

var (
	// FetchDurationHistogram tracks the duration of fetch operations
	FetchDurationHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: MetricsPrefix + "fetch_duration_seconds",
			Help: "Time taken to fetch data from external APIs",
		},
		[]string{"service", "operation"},
	)

	// CacheSizeGauge tracks the number of tokens in cache
	CacheSizeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "cache_size_tokens",
			Help: "Number of tokens in cache",
		},
		[]string{"service"},
	)

	// TokensByPlatformGauge tracks the number of tokens per platform
	TokensByPlatformGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "tokens_by_platform",
			Help: "Number of tokens per blockchain platform",
		},
		[]string{"platform"},
	)

	// RequestsCounter tracks total number of HTTP requests
	RequestsCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "http_requests_total",
			Help: "Total number of HTTP requests made",
		},
		[]string{"service", "status"},
	)

	// RequestsCycleCounter tracks number of HTTP requests per cycle, resets each cycle
	RequestsCycleCounter = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "http_requests_per_cycle",
			Help: "Number of HTTP requests per fetch cycle",
		},
		[]string{"service", "status"},
	)

	// RetryCounter tracks number of retry attempts
	RetryCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "http_retry_attempts_total",
			Help: "Total number of HTTP retry attempts",
		},
		[]string{"service"},
	)
)

// RecordFetchMarketDataCycle measures and records the duration of a market data fetch cycle
func RecordFetchMarketDataCycle(service string, start time.Time) {
	duration := time.Since(start)
	FetchDurationHistogram.WithLabelValues(service, "fetchAndUpdate").Observe(duration.Seconds())
	log.Printf("Metrics: %s fetchAndUpdate took %.2fs", service, duration.Seconds())
}

// RecordTokensCacheSize records the number of tokens in cache
func RecordTokensCacheSize(service string, size int) {
	CacheSizeGauge.WithLabelValues(service).Set(float64(size))
	log.Printf("Metrics: %s cache size is %d tokens", service, size)
}

// RecordTokensByPlatform records the number of tokens for each platform
func RecordTokensByPlatform(tokensByPlatform map[string]int) {
	// Reset all previous values first to handle platforms that no longer have tokens
	TokensByPlatformGauge.Reset()

	// Record the count for each platform
	for platform, count := range tokensByPlatform {
		TokensByPlatformGauge.WithLabelValues(platform).Set(float64(count))
		log.Printf("Metrics: Platform %s has %d tokens", platform, count)
	}
}

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(service, status string) {
	RequestsCounter.WithLabelValues(service, status).Inc()
	RequestsCycleCounter.WithLabelValues(service, status).Inc()
	log.Printf("Metrics: Recorded HTTP request for %s with status %s", service, status)
}

// RecordHTTPRetry records a retry attempt
func RecordHTTPRetry(service string) {
	RetryCounter.WithLabelValues(service).Inc()
	log.Printf("Metrics: Recorded HTTP retry for %s", service)
}

// ResetCycleCounters resets the per-cycle counters after a fetch cycle is complete
func ResetCycleCounters(service string) {
	RequestsCycleCounter.WithLabelValues(service, "success").Set(0)
	RequestsCycleCounter.WithLabelValues(service, "rate_limited").Set(0)
	RequestsCycleCounter.WithLabelValues(service, "error").Set(0)
	log.Printf("Metrics: Reset cycle counters for %s", service)
}
