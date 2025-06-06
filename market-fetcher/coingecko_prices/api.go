package coingecko_prices

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
	// FetchPrices fetches prices for the given parameters
	// Returns a map where key is token ID and value is raw JSON response for that token
	FetchPrices(params PriceParams) (map[string][]byte, error)
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

// FetchPrices fetches prices for the given parameters
func (c *CoinGeckoClient) FetchPrices(params PriceParams) (map[string][]byte, error) {
	// Get raw HTTP response and body using private function
	resp, body, err := c.executeFetchRequest(params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response using RawMessage to avoid unnecessary marshaling
	var rawPrices map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawPrices); err != nil {
		log.Printf("CoinGecko: Error parsing JSON response: %v", err)
		return nil, err
	}

	// Create result map with individual token responses
	result := make(map[string][]byte)
	for tokenId, tokenData := range rawPrices {
		// Use RawMessage directly as bytes (no additional marshaling needed)
		result[tokenId] = []byte(tokenData)
	}

	log.Printf("CoinGecko: Successfully fetched prices for %d tokens in %d currencies",
		len(params.IDs), len(params.Currencies))

	// Mark that we've had at least one successful fetch
	c.successfulFetch.Store(true)

	return result, nil
}

// executeFetchRequest is a private function that handles the actual request execution
// and returns the raw HTTP response and body
func (c *CoinGeckoClient) executeFetchRequest(params PriceParams) (*http.Response, []byte, error) {
	// Get available API keys from the key manager
	availableKeys := c.keyManager.GetAvailableKeys()

	// Track errors to return if all keys fail
	var lastError error

	// Try each key until one succeeds
	for _, apiKey := range availableKeys {
		// Get the appropriate base URL for this key type
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		// Create request builder for prices endpoint
		requestBuilder := NewPricesRequestBuilder(baseURL)

		// Configure request with token IDs and currencies
		requestBuilder.WithIds(params.IDs).WithCurrencies(params.Currencies)

		// Add optional parameters
		if params.IncludeMarketCap {
			requestBuilder.WithIncludeMarketCap(true)
		}
		if params.Include24hrVol {
			requestBuilder.WithInclude24hVolume(true)
		}
		if params.Include24hrChange {
			requestBuilder.WithInclude24hChange(true)
		}
		if params.IncludeLastUpdatedAt {
			requestBuilder.WithIncludeLastUpdatedAt(true)
		}
		if params.Precision != "" {
			requestBuilder.WithPrecision(params.Precision)
		}

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
		log.Printf("CoinGecko: Attempting request for %d tokens with key type %v", len(params.IDs), apiKey.Type)

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
		log.Printf("CoinGecko: Raw request successful for %d tokens with key type %v in %.2fs",
			len(params.IDs), apiKey.Type, duration.Seconds())

		return resp, body, nil
	}

	// If we got here, all keys failed
	return nil, nil, fmt.Errorf("all API keys failed, last error: %v", lastError)
}
