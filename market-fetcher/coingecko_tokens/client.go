package coingecko_tokens

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/metrics"
)

const (
	// Default URL for CoinGecko API
	DefaultCoinGeckoBaseURL = "https://api.coingecko.com"
	// Endpoint for coin list with platforms
	CoinsListEndpoint = "/api/v3/coins/list?include_platform=true"
	// Timeout for HTTP requests
	requestTimeout = 30 * time.Second
)

// Client handles HTTP communication with the CoinGecko API
type Client struct {
	baseURL    string
	httpClient *cg.HTTPClientWithRetries
}

// NewClient creates a new API client
func NewClient(baseURL string, metricsWriter *metrics.MetricsWriter) *Client {
	if baseURL == "" {
		baseURL = DefaultCoinGeckoBaseURL
	}

	// Create retry options
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGecko-Tokens"

	// Create metrics handler
	metricsHandler := cg.NewHttpRequestMetricsWriter(metricsWriter)

	return &Client{
		baseURL:    baseURL,
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsHandler),
	}
}

// FetchTokens retrieves the list of tokens from CoinGecko API
func (c *Client) FetchTokens() ([]Token, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, CoinsListEndpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, body, _, err := c.httpClient.ExecuteRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tokens []Token
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return tokens, nil
}
