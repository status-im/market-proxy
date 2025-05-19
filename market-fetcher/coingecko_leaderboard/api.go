package coingecko_leaderboard

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
)

// APIClient defines interface for API operations
type APIClient interface {
	// FetchPage fetches a single page of data
	FetchPage(page, limit int) ([]CoinData, error)
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

	metricsHandler := cg.NewHttpRequestMetricsWriter("coingecko")

	return &CoinGeckoClient{
		config:     cfg,
		keyManager: cg.NewAPIKeyManager(cfg.APITokens),
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsHandler),
	}
}

// Healthy checks if the API has had at least one successful fetch
func (c *CoinGeckoClient) Healthy() bool {
	return c.successfulFetch.Load()
}

// FetchPage fetches a single page of data from CoinGecko with retry capability
func (c *CoinGeckoClient) FetchPage(page, limit int) ([]CoinData, error) {
	// Get raw HTTP response and body using private function
	resp, body, err := c.executeFetchRequest(page, limit)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response
	var data []CoinGeckoData
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("CoinGecko: Error parsing JSON response: %v", err)
		return nil, err
	}

	// Convert CoinGeckoData to CoinData
	result := ConvertCoinGeckoData(data)

	log.Printf("CoinGecko: Successfully processed page %d with %d items",
		page, len(result))

	// Mark that we've had at least one successful fetch
	c.successfulFetch.Store(true)

	return result, nil
}

// executeFetchRequest is a private function that handles the actual request execution
// and returns the raw HTTP response and body
func (c *CoinGeckoClient) executeFetchRequest(page, limit int) (*http.Response, []byte, error) {
	// Get available API keys from the key manager
	availableKeys := c.keyManager.GetAvailableKeys()

	// Track errors to return if all keys fail
	var lastError error

	// Try each key until one succeeds
	for _, apiKey := range availableKeys {
		// Get the appropriate base URL for this key type
		baseURL := c.getApiBaseUrl(apiKey.Type)

		// Create request builder for markets endpoint
		requestBuilder := NewMarketRequestBuilder(baseURL)

		// Configure request with pagination parameters
		requestBuilder.WithPage(page).WithPerPage(limit)

		// Add API key if available
		if apiKey.Key != "" {
			requestBuilder.WithApiKey(apiKey.Key, apiKey.Type)
		}

		// Build the HTTP request
		request, err := requestBuilder.Build()
		if err != nil {
			log.Printf("CoinGecko: Error building request with key type %v: %v", apiKey.Type, err)
			lastError = err
			continue
		}

		// Log the attempt
		log.Printf("CoinGecko: Attempting request for page %d with key type %v", page, apiKey.Type)

		// Execute the request with retries
		resp, body, duration, err := c.httpClient.ExecuteRequest(request)

		// If the request failed
		if err != nil {
			log.Printf("CoinGecko: Request failed with key type %v: %v", apiKey.Type, err)

			// Mark the key as failed if it's not the NoKey
			if apiKey.Key != "" {
				log.Printf("CoinGecko: Marking key as failed and adding to backoff")
				c.keyManager.MarkKeyAsFailed(apiKey.Key)
			}

			lastError = err
			continue
		}

		// If we got here, the request succeeded
		log.Printf("CoinGecko: Raw request successful for page %d with key type %v in %.2fs",
			page, apiKey.Type, duration.Seconds())

		return resp, body, nil
	}

	// If we got here, all keys failed
	return nil, nil, fmt.Errorf("all API keys failed, last error: %v", lastError)
}

// getApiBaseUrl returns the appropriate API URL based on the key type
func (c *CoinGeckoClient) getApiBaseUrl(keyType cg.KeyType) string {
	// Use Pro URL only if we're using a Pro key
	if keyType == cg.ProKey {
		log.Printf("CoinGecko: Using Pro API URL based on key type")
		if c.config.OverrideCoingeckoProURL != "" {
			log.Printf("CoinGecko: Using overridden Pro API URL: %s", c.config.OverrideCoingeckoProURL)
			return c.config.OverrideCoingeckoProURL
		}
		return cg.COINGECKO_PRO_URL
	}

	log.Printf("CoinGecko: Using Public API URL based on key type")
	// if OverrideCoingeckoPublicURL is set, use that
	if c.config.OverrideCoingeckoPublicURL != "" {
		log.Printf("CoinGecko: Using overridden public API URL: %s", c.config.OverrideCoingeckoPublicURL)
		return c.config.OverrideCoingeckoPublicURL
	}
	// Otherwise, use the default public URL
	return cg.COINGECKO_PUBLIC_URL
}
