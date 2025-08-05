package coingecko_markets

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

//go:generate mockgen -destination=mocks/api_client.go . APIClient

// APIClient defines interface for API operations
type APIClient interface {
	// FetchPage fetches a single page of data with given parameters
	FetchPage(params cg.MarketsParams) ([][]byte, error)
	// Healthy checks if the API is responsive by fetching a minimal amount of data
	Healthy() bool
}

// CoinGeckoClient implements APIClient for CoinGecko
type CoinGeckoClient struct {
	config          *config.Config
	keyManager      cg.APIKeyManagerInterface
	httpClient      *cg.HTTPClientWithRetries
	successfulFetch atomic.Bool // Flag indicating if at least one fetch was successful
}

// NewCoinGeckoClient creates a new CoinGecko API client
func NewCoinGeckoClient(cfg *config.Config) *CoinGeckoClient {
	// Create retry options with CoinGecko specific settings
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGecko"

	// Create metrics writer for this service
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarkets)

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

// FetchPage fetches a single page of data from CoinGecko with retry capability
func (c *CoinGeckoClient) FetchPage(params cg.MarketsParams) ([][]byte, error) {
	// Get raw HTTP response and body using private function
	resp, body, err := c.executeFetchRequest(params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response as array of RawMessage
	var rawData []json.RawMessage
	if err := json.Unmarshal(body, &rawData); err != nil {
		log.Printf("CoinGecko: Error parsing JSON response: %v", err)
		return nil, err
	}

	// Convert each RawMessage to []byte (no additional marshaling needed)
	tokensData := make([][]byte, 0, len(rawData))
	for _, tokenData := range rawData {
		tokensData = append(tokensData, []byte(tokenData))
	}

	log.Printf("CoinGecko: Successfully processed page %d with %d items",
		params.Page, len(tokensData))

	// Mark that we've had at least one successful fetch
	c.successfulFetch.Store(true)

	return tokensData, nil
}

// executeFetchRequest is a private function that handles the actual request execution
// and returns the raw HTTP response and body
func (c *CoinGeckoClient) executeFetchRequest(params cg.MarketsParams) (*http.Response, []byte, error) {
	// Create executor function that attempts to fetch with a given API key
	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		// Get the appropriate base URL for this key type
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		// Create request builder for markets endpoint
		requestBuilder := NewMarketRequestBuilder(baseURL)

		// Add pagination parameters only if values are not 0
		if params.Page > 0 {
			requestBuilder = requestBuilder.WithPage(params.Page)
		}
		if params.PerPage > 0 {
			requestBuilder = requestBuilder.WithPerPage(params.PerPage)
		}

		// Configure remaining parameters with chaining
		requestBuilder.
			WithOrder(params.Order).
			WithCategory(params.Category).
			WithIDs(params.IDs).
			WithSparkline(params.SparklineEnabled).
			WithPriceChangePercentage(params.PriceChangePercentage).
			WithCurrency(params.Currency).
			WithApiKey(apiKey.Key, apiKey.Type)

		// Build the HTTP request
		request, err := requestBuilder.Build()
		if err != nil {
			log.Printf("CoinGecko: Error building request with key type %v: %v", apiKey.Type, err)
			return nil, false, err
		}

		// Log the attempt
		log.Printf("CoinGecko: Attempting request for page %d with key type %v", params.Page, apiKey.Type)

		// Execute the request with retries
		resp, body, duration, err := c.httpClient.ExecuteRequest(request)

		if err != nil {
			return nil, false, err
		}

		// If we got here, the request succeeded
		log.Printf("CoinGecko: Raw request successful for page %d with key type %v in %.2fs",
			params.Page, apiKey.Type, duration.Seconds())

		// Return both response and body as a struct
		result := struct {
			Response *http.Response
			Body     []byte
		}{resp, body}

		return result, true, nil
	}

	onFailed := cg.CreateFailCallback(c.keyManager)
	availableKeys := c.keyManager.GetAvailableKeys()

	result, err := cg.TryWithKeys(availableKeys, "CoinGecko", executor, onFailed)
	if err != nil {
		return nil, nil, err
	}

	responseData := result.(struct {
		Response *http.Response
		Body     []byte
	})

	return responseData.Response, responseData.Body, nil
}
