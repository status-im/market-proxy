package coingecko_common

import (
	"github.com/status-im/market-proxy/metrics"
)

// HttpRequestMetricsWriter implements HttpStatusHandler by writing to metrics
type HttpRequestMetricsWriter struct {
	serviceName string
}

// NewHttpRequestMetricsWriter creates a new metrics writer for the given service
func NewHttpRequestMetricsWriter(serviceName string) *HttpRequestMetricsWriter {
	return &HttpRequestMetricsWriter{
		serviceName: serviceName,
	}
}

// OnRequest records an HTTP request with its status
func (h *HttpRequestMetricsWriter) OnRequest(status string) {
	metrics.RecordHTTPRequest(h.serviceName, status)
}

// OnRetry records an HTTP retry attempt
func (h *HttpRequestMetricsWriter) OnRetry() {
	metrics.RecordHTTPRetry(h.serviceName)
}
