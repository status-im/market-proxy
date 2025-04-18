package coingecko

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// RequestBuilder implements the Builder pattern for CoinGecko API requests
type RequestBuilder struct {
	// Basic request parameters
	baseURL    string
	httpMethod string

	// Request specific parameters
	params map[string]string

	// API key information
	apiKey  string
	keyType KeyType

	// Other options
	userAgent string
	headers   map[string]string
}

// NewMarketRequestBuilder creates a new request builder for markets endpoint
func NewMarketRequestBuilder(baseURL string) *RequestBuilder {
	rb := &RequestBuilder{
		baseURL:    fmt.Sprintf("%s/coins/markets", baseURL),
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
func (rb *RequestBuilder) WithPage(page int) *RequestBuilder {
	rb.params["page"] = strconv.Itoa(page)
	return rb
}

// WithPerPage adds per_page parameter
func (rb *RequestBuilder) WithPerPage(perPage int) *RequestBuilder {
	rb.params["per_page"] = strconv.Itoa(perPage)
	return rb
}

// WithCurrency adds currency parameter
func (rb *RequestBuilder) WithCurrency(currency string) *RequestBuilder {
	rb.params["vs_currency"] = currency
	return rb
}

// WithOrder adds ordering parameter
func (rb *RequestBuilder) WithOrder(order string) *RequestBuilder {
	rb.params["order"] = order
	return rb
}

// WithApiKey sets the API key and its type
func (rb *RequestBuilder) WithApiKey(apiKey string, keyType KeyType) *RequestBuilder {
	rb.apiKey = apiKey
	rb.keyType = keyType
	return rb
}

// WithHeader adds a custom HTTP header
func (rb *RequestBuilder) WithHeader(name, value string) *RequestBuilder {
	rb.headers[name] = value
	return rb
}

// WithUserAgent sets the User-Agent header
func (rb *RequestBuilder) WithUserAgent(userAgent string) *RequestBuilder {
	rb.userAgent = userAgent
	return rb
}

// GetApiKey returns the API key and its type
func (rb *RequestBuilder) GetApiKey() (string, KeyType) {
	return rb.apiKey, rb.keyType
}

// BuildURL builds the complete URL for the request
func (rb *RequestBuilder) BuildURL() string {
	// Start with base URL
	query := url.Values{}

	// Add all parameters
	for key, value := range rb.params {
		query.Add(key, value)
	}

	// Add API key if available
	if rb.apiKey != "" {
		switch rb.keyType {
		case ProKey:
			query.Add("x_cg_pro_api_key", rb.apiKey)
		case DemoKey:
			query.Add("x_cg_demo_api_key", rb.apiKey)
		}
	}

	// Combine URL and query parameters
	finalURL := rb.baseURL
	queryString := query.Encode()
	if queryString != "" {
		finalURL = fmt.Sprintf("%s?%s", finalURL, queryString)
	}

	return finalURL
}

// Build creates an http.Request object
func (rb *RequestBuilder) Build() (*http.Request, error) {
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
