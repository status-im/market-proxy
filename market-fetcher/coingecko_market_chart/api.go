package coingecko_market_chart

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

// APIClient defines interface for market chart API operations
type APIClient interface {
	// FetchMarketChart fetches market chart data for a specific coin
	FetchMarketChart(params MarketChartParams) (map[string][]byte, error)
	// Healthy checks if the API is responsive
	Healthy() bool
}

// CoinGeckoClient implements APIClient for CoinGecko market chart API
type CoinGeckoClient struct {
	config          *config.Config
	keyManager      cg.APIKeyManagerInterface
	httpClient      *cg.HTTPClientWithRetries
	successfulFetch atomic.Bool // Flag indicating if at least one fetch was successful
}

// NewCoinGeckoClient creates a new CoinGecko market chart API client
func NewCoinGeckoClient(cfg *config.Config) *CoinGeckoClient {
	// Create retry options with CoinGecko specific settings
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGecko-MarketChart"

	// Create metrics writer for this service
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarkets) // Reuse markets service metrics

	return &CoinGeckoClient{
		config:     cfg,
		keyManager: cg.NewAPIKeyManager(cfg.APITokens),
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsWriter),
	}
}

// Healthy checks if the API has had at least one successful fetch
func (c *CoinGeckoClient) Healthy() bool {
	return c.successfulFetch.Load()
}

// FetchMarketChart fetches market chart data from CoinGecko with retry capability
func (c *CoinGeckoClient) FetchMarketChart(params MarketChartParams) (map[string][]byte, error) {
	// Validate required parameters
	if params.ID == "" {
		return nil, fmt.Errorf("coin ID is required")
	}

	// Get raw HTTP response and body using private function
	resp, body, err := c.executeFetchRequest(params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response using RawMessage to avoid unnecessary marshaling
	var rawPrices map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawPrices); err != nil {
		log.Printf("CoinGecko-MarketChart: Error parsing JSON response: %v", err)
		return nil, err
	}

	// Create result map with individual token responses
	result := make(map[string][]byte)
	for tokenId, tokenData := range rawPrices {
		// Use RawMessage directly as bytes (no additional marshaling needed)
		result[tokenId] = []byte(tokenData)
	}

	log.Printf("CoinGecko-MarketChart: Successfully fetched market chart for coin %s",
		params.ID)

	// Mark that we've had at least one successful fetch
	c.successfulFetch.Store(true)

	return result, nil
}

// executeFetchRequest is a private function that handles the actual request execution
// and returns the raw HTTP response and body
func (c *CoinGeckoClient) executeFetchRequest(params MarketChartParams) (*http.Response, []byte, error) {
	// Get available API keys from the key manager
	availableKeys := c.keyManager.GetAvailableKeys()

	// Track errors to return if all keys fail
	var lastError error

	// Try each key until one succeeds
	for _, apiKey := range availableKeys {
		// Get the appropriate base URL for this key type
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		// Create request builder for market chart endpoint
		requestBuilder := NewMarketChartRequestBuilder(baseURL, params.ID)

		// Configure request with parameters
		if params.Days != "" {
			requestBuilder.WithDays(params.Days)
		}

		if params.Currency != "" {
			requestBuilder.builder.WithCurrency(params.Currency)
		}

		if params.Interval != "" {
			requestBuilder.WithInterval(params.Interval)
		}

		// Add API key if available
		if apiKey.Key != "" {
			requestBuilder.builder.WithApiKey(apiKey.Key, apiKey.Type)
		}

		// Build the HTTP request
		request, err := requestBuilder.builder.Build()
		if err != nil {
			log.Printf("CoinGecko-MarketChart: Error building request with key type %v: %v", apiKey.Type, err)
			lastError = err
			continue
		}

		// Log the attempt
		log.Printf("CoinGecko-MarketChart: Attempting request for coin %s with key type %v", params.ID, apiKey.Type)

		// Execute the request with retries
		resp, body, duration, err := c.httpClient.ExecuteRequest(request)

		// If the request failed
		if err != nil {
			log.Printf("CoinGecko-MarketChart: Request failed with key type %v: %v", apiKey.Type, err)

			// Mark the key as failed if it's not the NoKey
			if apiKey.Key != "" {
				log.Printf("CoinGecko-MarketChart: Marking key as failed and adding to backoff")
				c.keyManager.MarkKeyAsFailed(apiKey.Key)
			}

			lastError = err
			continue
		}

		// If we got here, the request succeeded
		log.Printf("CoinGecko-MarketChart: Raw request successful for coin %s with key type %v in %.2fs",
			params.ID, apiKey.Type, duration.Seconds())

		return resp, body, nil
	}

	// If we got here, all keys failed
	return nil, nil, fmt.Errorf("all API keys failed, last error: %v", lastError)
}
