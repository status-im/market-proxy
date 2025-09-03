package coingecko_token_list

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

const (
	TokenListsEndpoint = "/api/v3/token_lists/%s/all.json"
)

//go:generate mockgen -destination=mocks/client.go . IClient

// IClient defines interface for token list API operations
type IClient interface {
	FetchTokenList(platform string) (*TokenList, error)
	Healthy() bool
}

// CoinGeckoClient implements IClient for CoinGecko
type CoinGeckoClient struct {
	config          *config.Config
	keyManager      cg.IAPIKeyManager
	httpClient      *cg.HTTPClientWithRetries
	successfulFetch atomic.Bool // Flag indicating if at least one fetch was successful
}

// NewCoinGeckoClient creates a new CoinGecko token list API client
func NewCoinGeckoClient(cfg *config.Config) *CoinGeckoClient {
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGecko-TokenList"

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	return &CoinGeckoClient{
		config:     cfg,
		keyManager: cg.NewAPIKeyManager(cfg.APITokens),
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsWriter, cg.GetRateLimiterManagerInstance()),
	}
}

// Healthy checks if the API has had at least one successful fetch
func (c *CoinGeckoClient) Healthy() bool {
	return c.successfulFetch.Load()
}

// FetchTokenList retrieves token list for a specific platform from CoinGecko API
func (c *CoinGeckoClient) FetchTokenList(platform string) (*TokenList, error) {
	resp, body, err := c.executeFetchRequest(platform)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenList TokenList
	if err := json.Unmarshal(body, &tokenList); err != nil {
		log.Printf("CoinGecko-TokenList: Error parsing JSON response for platform %s: %v", platform, err)
		return nil, fmt.Errorf("error unmarshaling token list response for platform %s: %w", platform, err)
	}

	c.successfulFetch.Store(true)

	return &tokenList, nil
}

func (c *CoinGeckoClient) executeFetchRequest(platform string) (*http.Response, []byte, error) {
	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)
		requestBuilder := NewTokensRequestBuilder(baseURL, platform)
		requestBuilder.WithApiKey(apiKey.Key, apiKey.Type)

		req, err := requestBuilder.Build()
		if err != nil {
			log.Printf("CoinGecko-TokenList: Error building request for platform %s with key type %v: %v", platform, apiKey.Type, err)
			return nil, false, err
		}

		// Execute the request with retries
		resp, body, _, err := c.httpClient.ExecuteRequest(req)
		if err != nil {
			return nil, false, err
		}

		result := struct {
			Response *http.Response
			Body     []byte
		}{resp, body}

		return result, true, nil
	}

	onFailed := cg.CreateFailCallback(c.keyManager)
	availableKeys := c.keyManager.GetAvailableKeys()

	result, err := cg.TryWithKeys(availableKeys, "CoinGecko-TokenList", executor, onFailed)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching token list for platform %s: %w", platform, err)
	}

	responseData := result.(struct {
		Response *http.Response
		Body     []byte
	})

	return responseData.Response, responseData.Body, nil
}
