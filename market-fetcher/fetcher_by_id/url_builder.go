package fetcher_by_id

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
)

const (
	templatePlaceholderID      = "{{id}}"
	templatePlaceholderIDsList = "{{ids_list}}"
)

// URLBuilder builds URLs from endpoint templates
type URLBuilder struct {
	baseURL      string
	endpointPath string
	queryParams  map[string]string
	apiKey       string
	keyType      cg.KeyType
}

// NewURLBuilder creates a new URL builder
func NewURLBuilder(baseURL string, cfg *config.FetcherByIdConfig) *URLBuilder {
	return &URLBuilder{
		baseURL:      strings.TrimRight(baseURL, "/"),
		endpointPath: cfg.EndpointPath,
		queryParams:  cfg.BuildQueryParams(),
	}
}

// WithAPIKey sets the API key for the request
func (b *URLBuilder) WithAPIKey(apiKey string, keyType cg.KeyType) *URLBuilder {
	b.apiKey = apiKey
	b.keyType = keyType
	return b
}

// BuildSingleURL builds a URL for a single ID request
// Replaces {{id}} with the provided ID
func (b *URLBuilder) BuildSingleURL(id string) (string, error) {
	if !strings.Contains(b.endpointPath, templatePlaceholderID) {
		return "", fmt.Errorf("endpoint path does not contain %s placeholder", templatePlaceholderID)
	}

	// Replace the placeholder with the actual ID
	path := strings.Replace(b.endpointPath, templatePlaceholderID, url.PathEscape(id), 1)

	return b.buildFinalURL(path)
}

// BuildBatchURL builds a URL for a batch request
// Replaces {{ids_list}} with comma-separated IDs
func (b *URLBuilder) BuildBatchURL(ids []string) (string, error) {
	if !strings.Contains(b.endpointPath, templatePlaceholderIDsList) {
		return "", fmt.Errorf("endpoint path does not contain %s placeholder", templatePlaceholderIDsList)
	}

	// Join IDs with comma and replace the placeholder
	idsList := strings.Join(ids, ",")
	path := strings.Replace(b.endpointPath, templatePlaceholderIDsList, idsList, 1)

	return b.buildFinalURL(path)
}

// buildFinalURL combines base URL, path, and query parameters
func (b *URLBuilder) buildFinalURL(path string) (string, error) {
	// Parse the path to handle any existing query params in the template
	parsedPath, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("failed to parse path: %w", err)
	}

	// Combine with base URL
	fullURL := b.baseURL + parsedPath.Path

	// Merge query params: template params + config params + API key
	query := parsedPath.Query()

	// Add config query params
	for key, value := range b.queryParams {
		query.Set(key, value)
	}

	// Add API key
	if b.apiKey != "" {
		switch b.keyType {
		case cg.ProKey:
			query.Set("x_cg_pro_api_key", b.apiKey)
		case cg.DemoKey:
			query.Set("x_cg_demo_api_key", b.apiKey)
		}
	}

	// Build final URL
	if len(query) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, query.Encode())
	}

	return fullURL, nil
}

// RequestBuilder wraps URLBuilder to create http.Request objects
type RequestBuilder struct {
	urlBuilder *URLBuilder
	userAgent  string
	headers    map[string]string
}

// NewRequestBuilder creates a new request builder
func NewRequestBuilder(baseURL string, cfg *config.FetcherByIdConfig) *RequestBuilder {
	return &RequestBuilder{
		urlBuilder: NewURLBuilder(baseURL, cfg),
		userAgent:  "Mozilla/5.0 Market-Proxy",
		headers: map[string]string{
			"Accept": "application/json",
		},
	}
}

// WithAPIKey sets the API key for the request
func (rb *RequestBuilder) WithAPIKey(apiKey string, keyType cg.KeyType) *RequestBuilder {
	rb.urlBuilder.WithAPIKey(apiKey, keyType)
	return rb
}

// BuildSingleRequest builds an HTTP request for a single ID
func (rb *RequestBuilder) BuildSingleRequest(id string) (*http.Request, error) {
	urlStr, err := rb.urlBuilder.BuildSingleURL(id)
	if err != nil {
		return nil, err
	}

	return rb.createRequest(urlStr)
}

// BuildBatchRequest builds an HTTP request for multiple IDs
func (rb *RequestBuilder) BuildBatchRequest(ids []string) (*http.Request, error) {
	urlStr, err := rb.urlBuilder.BuildBatchURL(ids)
	if err != nil {
		return nil, err
	}

	return rb.createRequest(urlStr)
}

// createRequest creates an http.Request with common headers
func (rb *RequestBuilder) createRequest(urlStr string) (*http.Request, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", rb.userAgent)
	for key, value := range rb.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
