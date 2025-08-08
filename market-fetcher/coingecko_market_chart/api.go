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

type IAPIClient interface {
	FetchMarketChart(params MarketChartParams) (map[string][]byte, error)
	Healthy() bool
}

type CoinGeckoClient struct {
	config          *config.Config
	keyManager      cg.IAPIKeyManager
	httpClient      *cg.HTTPClientWithRetries
	successfulFetch atomic.Bool
}

func NewCoinGeckoClient(cfg *config.Config) *CoinGeckoClient {
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGecko-MarketChart"

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarketCharts)

	return &CoinGeckoClient{
		config:     cfg,
		keyManager: cg.NewAPIKeyManager(cfg.APITokens),
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsWriter, cg.GetRateLimiterManagerInstance()),
	}
}

// prependFreeKey moves the NoKey type to the beginning
func prependFreeKey(keys []cg.APIKey) []cg.APIKey {
	for i, key := range keys {
		if key.Type == cg.NoKey {
			return append([]cg.APIKey{key}, append(keys[:i], keys[i+1:]...)...)
		}
	}
	return keys
}

func (c *CoinGeckoClient) Healthy() bool {
	return c.successfulFetch.Load()
}

func (c *CoinGeckoClient) FetchMarketChart(params MarketChartParams) (map[string][]byte, error) {
	if params.ID == "" {
		return nil, fmt.Errorf("coin ID is required")
	}

	resp, body, err := c.executeFetchRequest(params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rawPrices map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawPrices); err != nil {
		log.Printf("CoinGecko-MarketChart: Error parsing JSON response: %v", err)
		return nil, err
	}

	result := make(map[string][]byte)
	for tokenId, tokenData := range rawPrices {
		result[tokenId] = []byte(tokenData)
	}

	log.Printf("CoinGecko-MarketChart: Successfully fetched market chart for coin %s",
		params.ID)

	c.successfulFetch.Store(true)

	return result, nil
}

func (c *CoinGeckoClient) executeFetchRequest(params MarketChartParams) (*http.Response, []byte, error) {
	// Create executor function that attempts to fetch with a given API key
	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		requestBuilder := NewMarketChartRequestBuilder(baseURL, params.ID).
			WithDays(params.Days).
			WithInterval(params.Interval).
			WithCurrency(params.Currency).
			WithApiKey(apiKey.Key, apiKey.Type)

		request, err := requestBuilder.Build()
		if err != nil {
			log.Printf("CoinGecko-MarketChart: Error building request with key type %v: %v", apiKey.Type, err)
			return nil, false, err
		}

		resp, body, _, err := c.httpClient.ExecuteRequest(request)
		if err != nil {
			return nil, false, err
		}

		// Return both response and body as a struct
		result := struct {
			Response *http.Response
			Body     []byte
		}{resp, body}

		return result, true, nil
	}

	// Create the onFailed callback
	onFailed := cg.CreateFailCallback(c.keyManager)

	availableKeys := c.keyManager.GetAvailableKeys()

	if c.config.CoingeckoMarketChart.TryFreeApiFirst && params.Interval == "" {
		availableKeys = prependFreeKey(availableKeys)
	}

	result, err := cg.TryWithKeys(availableKeys, "CoinGecko-MarketChart", executor, onFailed)
	if err != nil {
		return nil, nil, err
	}

	responseData := result.(struct {
		Response *http.Response
		Body     []byte
	})

	return responseData.Response, responseData.Body, nil
}
