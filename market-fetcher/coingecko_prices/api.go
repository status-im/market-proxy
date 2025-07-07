package coingecko_prices

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

// APIClient defines interface for API operations
type APIClient interface {
	// FetchPrices fetches prices for the given parameters
	// Returns a map where key is token ID and value is raw JSON response for that token
	FetchPrices(params cg.PriceParams) (map[string][]byte, error)
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
func NewCoinGeckoClient(cfg *config.Config, metricsWriter *metrics.MetricsWriter) *CoinGeckoClient {
	// Create retry options with CoinGecko specific settings
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGeckoPrices"

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

// FetchPrices fetches prices for the given parameters
func (c *CoinGeckoClient) FetchPrices(params cg.PriceParams) (map[string][]byte, error) {
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

func (c *CoinGeckoClient) executeFetchRequest(params cg.PriceParams) (*http.Response, []byte, error) {
	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		// Get the appropriate base URL for this key type
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		// Create request builder for prices endpoint and configure with chaining
		requestBuilder := NewPricesRequestBuilder(baseURL).
			WithIds(params.IDs).
			WithCurrencies(params.Currencies).
			WithIncludeMarketCap(params.IncludeMarketCap).
			WithInclude24hVolume(params.Include24hrVol).
			WithInclude24hChange(params.Include24hrChange).
			WithIncludeLastUpdatedAt(params.IncludeLastUpdatedAt).
			WithPrecision(params.Precision).
			WithApiKey(apiKey.Key, apiKey.Type)

		// Build the HTTP request
		request, err := requestBuilder.Build()
		if err != nil {
			log.Printf("CoinGecko: Error building request with key type %v: %v", apiKey.Type, err)
			return nil, false, err
		}

		// Log the attempt
		log.Printf("CoinGecko: Attempting request for %d tokens with key type %v", len(params.IDs), apiKey.Type)

		// Execute the request with retries
		resp, body, duration, err := c.httpClient.ExecuteRequest(request)

		if err != nil {
			return nil, false, err
		}

		// If we got here, the request succeeded
		log.Printf("CoinGecko: Raw request successful for %d tokens with key type %v in %.2fs",
			len(params.IDs), apiKey.Type, duration.Seconds())

		// Return both response and body as a struct
		result := struct {
			Response *http.Response
			Body     []byte
		}{resp, body}

		return result, true, nil
	}

	// Use TryWithKeys iterator
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
