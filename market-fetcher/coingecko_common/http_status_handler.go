package coingecko_common

import (
	"github.com/status-im/market-proxy/metrics"
)

// HttpRequestMetricsWriter implements HttpStatusHandler by writing to metrics
type HttpRequestMetricsWriter struct {
	metricsWriter *metrics.MetricsWriter
}

// NewHttpRequestMetricsWriter creates a new metrics writer using MetricsWriter
func NewHttpRequestMetricsWriter(metricsWriter *metrics.MetricsWriter) *HttpRequestMetricsWriter {
	return &HttpRequestMetricsWriter{
		metricsWriter: metricsWriter,
	}
}

// OnRequest records an HTTP request with its status
func (h *HttpRequestMetricsWriter) OnRequest(status string) {
	// Record both global and service-specific metrics
	h.metricsWriter.RecordCoingeckoRequestTotal(status)
	h.metricsWriter.RecordServiceCoingeckoRequest(status)
}

// OnRetry records an HTTP retry attempt
func (h *HttpRequestMetricsWriter) OnRetry() {
	h.metricsWriter.RecordRetryAttempt()
}
