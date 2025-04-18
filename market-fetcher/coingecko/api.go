package coingecko

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/status-im/market-proxy/config"
)

// APIClient defines interface for API operations
type APIClient interface {
	// FetchPage fetches a single page of data
	FetchPage(page, limit int) ([]interface{}, error)
}

// CoinGeckoClient implements APIClient for CoinGecko
type CoinGeckoClient struct {
	config     *config.Config
	apiTokens  *config.APITokens
	httpClient *HTTPClientWithRetries
}

// NewCoinGeckoClient creates a new CoinGecko API client
func NewCoinGeckoClient(cfg *config.Config, apiTokens *config.APITokens) *CoinGeckoClient {
	// Create retry options with CoinGecko specific settings
	retryOpts := DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGecko"

	return &CoinGeckoClient{
		config:     cfg,
		apiTokens:  apiTokens,
		httpClient: NewHTTPClientWithRetries(retryOpts),
	}
}

// FetchPage fetches a single page of data from CoinGecko with retry capability
func (c *CoinGeckoClient) FetchPage(page, limit int) ([]interface{}, error) {
	// Build request
	req, err := c.buildRequest(page, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Execute request with retries
	logContext := fmt.Sprintf("page %d", page)
	_, responseBody, requestDuration, err := c.httpClient.ExecuteRequest(req, logContext)
	if err != nil {
		return nil, err
	}

	// Parse the JSON
	var coinsOriginal []CoinGeckoData
	if err := json.Unmarshal(responseBody, &coinsOriginal); err != nil {
		return nil, fmt.Errorf("error parsing JSON for page %d: %v", page, err)
	}

	// Convert to CoinMarketCap-compatible format
	coins := ConvertCoinGeckoData(coinsOriginal)

	// Log how many coins we received
	log.Printf("CoinGecko: Received %d coins from page %d in %.2fs", len(coins), page, requestDuration.Seconds())

	// Convert to interface slice for generic handling
	result := make([]interface{}, len(coins))
	for i, coin := range coins {
		result[i] = coin
	}

	return result, nil
}

// buildRequest builds an HTTP request for the CoinGecko API
func (c *CoinGeckoClient) buildRequest(page, limit, attempt int) (*http.Request, error) {
	// Get the appropriate base URL
	baseUrl := c.getApiBaseUrl()

	// Build URL with pagination parameters
	url := fmt.Sprintf("%s?vs_currency=usd&order=market_cap_desc&per_page=%d&page=%d",
		baseUrl,
		limit,
		page)

	// Add API key to URL if available
	if len(c.apiTokens.Tokens) > 0 {
		// Use the correct parameter name based on key type
		if c.isUsingDemoKey() {
			url = fmt.Sprintf("%s&x_cg_demo_api_key=%s", url, c.apiTokens.Tokens[0])
			if attempt == 0 {
				log.Printf("CoinGecko: Using Public API with Demo key for page %d request", page)
			}
		} else {
			url = fmt.Sprintf("%s&x_cg_pro_api_key=%s", url, c.apiTokens.Tokens[0])
			if attempt == 0 {
				log.Printf("CoinGecko: Using Pro API with Pro key for page %d request", page)
			}
		}
	} else if attempt == 0 {
		log.Printf("CoinGecko: No API key available, using public API for page %d", page)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 Market-Proxy")

	return req, nil
}

// Returns the appropriate API URL based on whether we have an API key
func (c *CoinGeckoClient) getApiBaseUrl() string {
	if c.apiTokens == nil || len(c.apiTokens.Tokens) == 0 {
		log.Printf("CoinGecko: No API tokens provided, using public API URL")
		return COINGECKO_PUBLIC_URL
	}

	if c.isUsingDemoKey() {
		log.Printf("CoinGecko: Detected Demo API key, using public API URL")
		return COINGECKO_PUBLIC_URL
	} else {
		log.Printf("CoinGecko: Using Pro API with API key")
		return COINGECKO_PRO_URL
	}
}

// Determines if the API key is a demo key
func (c *CoinGeckoClient) isUsingDemoKey() bool {
	if c.apiTokens == nil || len(c.apiTokens.Tokens) == 0 {
		return false
	}

	apiKey := c.apiTokens.Tokens[0]
	// Check if this is a demo key
	return strings.HasPrefix(apiKey, "demo_") ||
		strings.HasPrefix(apiKey, "CG-") ||
		strings.Contains(strings.ToLower(apiKey), "demo")
}
