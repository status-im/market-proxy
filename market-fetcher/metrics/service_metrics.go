package metrics

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Service constants
const (
	ServiceLBMarkets = "lb-markets"
	ServiceLBPrices  = "lb-prices"
	ServiceCoins     = "coins"
	ServicePrices    = "prices"
)

var (
	// Global Coingecko request counter (all services)
	CoingeckoRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "coingecko_requests_total",
			Help: "Total number of HTTP requests to Coingecko API across all services",
		},
		[]string{"status"},
	)

	// Service-specific Coingecko request counter
	ServiceCoingeckoRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "service_coingecko_requests_total",
			Help: "Total number of HTTP requests to Coingecko API per service",
		},
		[]string{"service", "status"},
	)

	// Data fetch cycle duration per service
	DataFetchCycleDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: MetricsPrefix + "data_fetch_cycle_duration_seconds",
			Help: "Time taken to complete a full data fetch cycle",
		},
		[]string{"service"},
	)

	// Requests per cycle gauge (resets each cycle)
	RequestsPerCycleGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "requests_per_cycle",
			Help: "Number of HTTP requests in current cycle",
		},
		[]string{"service"},
	)

	// Request status counts per cycle
	CycleRequestStatusGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "cycle_request_status_count",
			Help: "Number of requests by status in current cycle",
		},
		[]string{"service", "status"},
	)

	// Service cache size
	ServiceCacheSizeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MetricsPrefix + "service_cache_size",
			Help: "Number of items in service cache",
		},
		[]string{"service"},
	)

	// Request latency per endpoint
	RequestLatencyHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: MetricsPrefix + "request_latency_seconds",
			Help: "HTTP request latency by service and endpoint",
		},
		[]string{"service", "endpoint"},
	)

	// Retry attempts counter
	ServiceRetryCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "service_retry_attempts_total",
			Help: "Total number of retry attempts per service",
		},
		[]string{"service"},
	)

	// Rate limit hits counter
	RateLimitCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricsPrefix + "rate_limit_hits_total",
			Help: "Total number of rate limit hits per service",
		},
		[]string{"service"},
	)
)

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

// RecordCoingeckoRequestTotal records a global Coingecko API request
func (mw *MetricsWriter) RecordCoingeckoRequestTotal(status string) {
	CoingeckoRequestsTotal.WithLabelValues(status).Inc()
	log.Printf("Metrics: Global Coingecko request recorded with status %s", status)
}

// RecordServiceCoingeckoRequest records a service-specific Coingecko API request
func (mw *MetricsWriter) RecordServiceCoingeckoRequest(status string) {
	ServiceCoingeckoRequestsTotal.WithLabelValues(mw.serviceName, status).Inc()
	log.Printf("Metrics: %s Coingecko request recorded with status %s", mw.serviceName, status)
}

// RecordDataFetchCycle records the duration of a data fetch cycle
func (mw *MetricsWriter) RecordDataFetchCycle(duration time.Duration) {
	DataFetchCycleDuration.WithLabelValues(mw.serviceName).Observe(duration.Seconds())
	log.Printf("Metrics: %s data fetch cycle took %.2fs", mw.serviceName, duration.Seconds())
}

// RecordCycleRequestCount records the total number of requests in a cycle
func (mw *MetricsWriter) RecordCycleRequestCount(count int) {
	RequestsPerCycleGauge.WithLabelValues(mw.serviceName).Set(float64(count))
	log.Printf("Metrics: %s cycle had %d requests", mw.serviceName, count)
}

// RecordCycleRequestsByStatus records the number of requests by status in current cycle
func (mw *MetricsWriter) RecordCycleRequestsByStatus(statusCounts map[string]int) {
	for status, count := range statusCounts {
		CycleRequestStatusGauge.WithLabelValues(mw.serviceName, status).Set(float64(count))
		log.Printf("Metrics: %s cycle had %d requests with status %s", mw.serviceName, count, status)
	}
}

// RecordCacheSize records the number of items in service cache
func (mw *MetricsWriter) RecordCacheSize(size int) {
	ServiceCacheSizeGauge.WithLabelValues(mw.serviceName).Set(float64(size))
	log.Printf("Metrics: %s cache size is %d items", mw.serviceName, size)
}

// RecordRequestLatency records the latency for a specific endpoint
func (mw *MetricsWriter) RecordRequestLatency(duration time.Duration, endpoint string) {
	RequestLatencyHistogram.WithLabelValues(mw.serviceName, endpoint).Observe(duration.Seconds())
	log.Printf("Metrics: %s request to %s took %.2fs", mw.serviceName, endpoint, duration.Seconds())
}

// RecordRetryAttempt records a retry attempt
func (mw *MetricsWriter) RecordRetryAttempt() {
	ServiceRetryCounter.WithLabelValues(mw.serviceName).Inc()
	log.Printf("Metrics: %s recorded a retry attempt", mw.serviceName)
}

// RecordRateLimitHit records a rate limit hit
func (mw *MetricsWriter) RecordRateLimitHit() {
	RateLimitCounter.WithLabelValues(mw.serviceName).Inc()
	log.Printf("Metrics: %s hit rate limit", mw.serviceName)
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
	mw.RecordCoingeckoRequestTotal(status)
	mw.RecordServiceCoingeckoRequest(status)
}

// OnRetry records an HTTP retry attempt
func (mw *MetricsWriter) OnRetry() {
	mw.RecordRetryAttempt()
}
