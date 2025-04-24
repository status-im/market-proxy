package coingecko

import (
	"fmt"
	cg "github.com/status-im/market-proxy/coingecko"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	// Complete path for markets API endpoint
	MARKETS_API_PATH = "/api/v3/coins/markets"
)

// buildURL safely combines a base URL with a path
func buildURL(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	trimmedPath := strings.TrimLeft(path, "/")

	return baseURL + "/" + trimmedPath
}

// MarketsRequestBuilder implements the Builder pattern for CoinGecko markets API requests
type MarketsRequestBuilder struct {
	// Basic request parameters
	baseURL    string
	httpMethod string

	// Request specific parameters
	params map[string]string

	// API key information
	apiKey  string
	keyType cg.KeyType

	// Other options
	userAgent string
	headers   map[string]string
}

// NewMarketRequestBuilder creates a new request builder for markets endpoint
func NewMarketRequestBuilder(baseURL string) *MarketsRequestBuilder {
	rb := &MarketsRequestBuilder{
		baseURL:    baseURL,
		httpMethod: "GET",
		params:     make(map[string]string),
		headers:    make(map[string]string),
		userAgent:  "Mozilla/5.0 Market-Proxy",
	}

	// Add default headers
	rb.headers["Accept"] = "application/json"

	// Add default market parameters
	rb.params["vs_currency"] = "usd"
	rb.params["order"] = "market_cap_desc"

	return rb
}

// WithPage adds page parameter for pagination
func (rb *MarketsRequestBuilder) WithPage(page int) *MarketsRequestBuilder {
	rb.params["page"] = strconv.Itoa(page)
	return rb
}

// WithPerPage adds per_page parameter
func (rb *MarketsRequestBuilder) WithPerPage(perPage int) *MarketsRequestBuilder {
	rb.params["per_page"] = strconv.Itoa(perPage)
	return rb
}

// WithCurrency adds currency parameter
func (rb *MarketsRequestBuilder) WithCurrency(currency string) *MarketsRequestBuilder {
	rb.params["vs_currency"] = currency
	return rb
}

// WithOrder adds ordering parameter
func (rb *MarketsRequestBuilder) WithOrder(order string) *MarketsRequestBuilder {
	rb.params["order"] = order
	return rb
}

// WithApiKey sets the API key and its type
func (rb *MarketsRequestBuilder) WithApiKey(apiKey string, keyType cg.KeyType) *MarketsRequestBuilder {
	rb.apiKey = apiKey
	rb.keyType = keyType
	return rb
}

// WithHeader adds a custom HTTP header
func (rb *MarketsRequestBuilder) WithHeader(name, value string) *MarketsRequestBuilder {
	rb.headers[name] = value
	return rb
}

// WithUserAgent sets the User-Agent header
func (rb *MarketsRequestBuilder) WithUserAgent(userAgent string) *MarketsRequestBuilder {
	rb.userAgent = userAgent
	return rb
}

// GetApiKey returns the API key and its type
func (rb *MarketsRequestBuilder) GetApiKey() (string, cg.KeyType) {
	return rb.apiKey, rb.keyType
}

// BuildURL builds the complete URL for the request
func (rb *MarketsRequestBuilder) BuildURL() string {
	// Build the full URL using the safe path combiner
	fullPath := buildURL(rb.baseURL, MARKETS_API_PATH)

	// Create query parameters
	query := url.Values{}

	// Add all parameters
	for key, value := range rb.params {
		query.Add(key, value)
	}

	// Add API key if available
	if rb.apiKey != "" {
		switch rb.keyType {
		case cg.ProKey:
			query.Add("x_cg_pro_api_key", rb.apiKey)
		case cg.DemoKey:
			query.Add("x_cg_demo_api_key", rb.apiKey)
		}
	}

	// Combine URL and query parameters
	finalURL := fullPath
	queryString := query.Encode()
	if queryString != "" {
		finalURL = fmt.Sprintf("%s?%s", finalURL, queryString)
	}

	return finalURL
}

// Build creates an http.Request object
func (rb *MarketsRequestBuilder) Build() (*http.Request, error) {
	// Build the URL
	finalURL := rb.BuildURL()

	// Create the request
	req, err := http.NewRequest(rb.httpMethod, finalURL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent
	req.Header.Set("User-Agent", rb.userAgent)

	// Add all headers
	for key, value := range rb.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
