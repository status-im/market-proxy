package metrics

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsPrefix is the prefix used for all metrics
const MetricsPrefix = "market_fetcher_"

// Service constants
const (
	ServiceLBMarkets = "lb-markets"
	ServiceLBPrices  = "lb-prices"
	ServiceCoins     = "coins"
	ServicePrices    = "prices"
)

var (
	// TokensByPlatformGauge tracks the number of tokens per platform
	// Cardinality: ~15 (number of supported blockchain platforms)
	TokensByPlatformGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "tokens_by_platform",
			Help: "Number of tokens per blockchain platform",
		},
		[]string{"platform"},
	)

	// Global Coingecko request counter (all services)
	// Cardinality: ~5 (success, error, rate_limited, timeout, etc.)
	CoingeckoRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "coingecko_requests_total",
			Help: "Total number of HTTP requests to Coingecko API across all services",
		},
		[]string{"status"},
	)

	// Service-specific Coingecko request counter
	// Cardinality: ~20 (4 services × 5 statuses)
	ServiceCoingeckoRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "service_coingecko_requests_total",
			Help: "Total number of HTTP requests to Coingecko API per service",
		},
		[]string{"service", "status"},
	)

	// Data fetch cycle duration per service
	// Cardinality: ~4 (number of services)
	DataFetchCycleDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: MetricsPrefix + "data_fetch_cycle_duration_seconds",
			Help: "Time taken to complete a full data fetch cycle",
		},
		[]string{"service"},
	)

	// Requests per cycle gauge (resets each cycle)
	// Cardinality: ~4 (number of services)
	RequestsPerCycleGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "requests_per_cycle",
			Help: "Number of HTTP requests in current cycle",
		},
		[]string{"service"},
	)

	// Request status counts per cycle
	// Cardinality: ~20 (4 services × 5 statuses)
	CycleRequestStatusGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "cycle_request_status_count",
			Help: "Number of requests by status in current cycle",
		},
		[]string{"service", "status"},
	)

	// Service cache size
	// Cardinality: ~4 (number of services)
	ServiceCacheSizeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "service_cache_size",
			Help: "Number of items in service cache",
		},
		[]string{"service"},
	)

	// Request latency per endpoint
	// Cardinality: ~20 (4 services × 5 endpoints per service)
	RequestLatencyHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: MetricsPrefix + "request_latency_seconds",
			Help: "HTTP request latency by service and endpoint",
		},
		[]string{"service", "endpoint"},
	)

	// Retry attempts counter
	// Cardinality: ~4 (number of services)
	ServiceRetryCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "service_retry_attempts_total",
			Help: "Total number of retry attempts per service",
		},
		[]string{"service"},
	)

	// Rate limit hits counter
	// Cardinality: ~4 (number of services)
	RateLimitCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "rate_limit_hits_total",
			Help: "Total number of rate limit hits per service",
		},
		[]string{"service"},
	)
)

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

// MetricsWriter provides a unified interface for recording service metrics
type MetricsWriter struct {
	serviceName string
}

// NewMetricsWriter creates a new MetricsWriter for the specified service
func NewMetricsWriter(serviceName string) *MetricsWriter {
	return &MetricsWriter{
		serviceName: serviceName,
	}
}

// GetServiceName returns the service name
func (mw *MetricsWriter) GetServiceName() string {
	return mw.serviceName
}

// RecordServiceCoingeckoRequest records a service-specific Coingecko API request
func (mw *MetricsWriter) RecordServiceCoingeckoRequest(status string) {
	CoingeckoRequestsTotal.WithLabelValues(status).Inc()
	ServiceCoingeckoRequestsTotal.WithLabelValues(mw.serviceName, status).Inc()
	log.Printf("Metrics: %s Coingecko request recorded with status %s", mw.serviceName, status)
}

// RecordDataFetchCycle records the duration of a data fetch cycle
func (mw *MetricsWriter) RecordDataFetchCycle(duration time.Duration) {
	DataFetchCycleDuration.WithLabelValues(mw.serviceName).Observe(duration.Seconds())
	log.Printf("Metrics: %s data fetch cycle took %.2fs", mw.serviceName, duration.Seconds())
}

// RecordCacheSize records the number of items in service cache
func (mw *MetricsWriter) RecordCacheSize(size int) {
	ServiceCacheSizeGauge.WithLabelValues(mw.serviceName).Set(float64(size))
	log.Printf("Metrics: %s cache size is %d items", mw.serviceName, size)
}

// RecordRetryAttempt records a retry attempt
func (mw *MetricsWriter) RecordRetryAttempt() {
	ServiceRetryCounter.WithLabelValues(mw.serviceName).Inc()
	log.Printf("Metrics: %s recorded a retry attempt", mw.serviceName)
}

// ResetCycleMetrics resets all cycle-related metrics
func (mw *MetricsWriter) ResetCycleMetrics() {
	RequestsPerCycleGauge.WithLabelValues(mw.serviceName).Set(0)

	// Reset common status counters
	statuses := []string{"success", "error", "rate_limited", "timeout"}
	for _, status := range statuses {
		CycleRequestStatusGauge.WithLabelValues(mw.serviceName, status).Set(0)
	}

	log.Printf("Metrics: Reset cycle metrics for %s", mw.serviceName)
}

// Implement HttpStatusHandler interface for MetricsWriter
// OnRequest records an HTTP request with its status
func (mw *MetricsWriter) OnRequest(status string) {
	mw.RecordServiceCoingeckoRequest(status)
}

// OnRetry records an HTTP retry attempt
func (mw *MetricsWriter) OnRetry() {
	mw.RecordRetryAttempt()
}
