package coingecko_exchange_rates

import (
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

const logPrefix = "CoinGecko-ExchangeRates"

//go:generate mockgen -destination=mocks/client.go . IClient

// IClient defines the API operations of the exchange rates service
type IClient interface {
	FetchExchangeRates() (ExchangeRatesResponse, error)
	Healthy() bool
}

// CoinGeckoClient implements IClient for CoinGecko
type CoinGeckoClient struct {
	config          *config.Config
	keyManager      cg.IAPIKeyManager
	httpClient      *cg.HTTPClientWithRetries
	successfulFetch atomic.Bool
}

// NewCoinGeckoClient creates a new CoinGecko exchange rates client
func NewCoinGeckoClient(cfg *config.Config, metricsWriter *metrics.MetricsWriter) *CoinGeckoClient {
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = logPrefix

	return &CoinGeckoClient{
		config:     cfg,
		keyManager: cg.NewAPIKeyManager(cfg.APITokens),
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsWriter, cg.GetRateLimiterManagerInstance()),
	}
}

// Healthy reports whether at least one fetch has succeeded
func (c *CoinGeckoClient) Healthy() bool {
	return c.successfulFetch.Load()
}

// FetchExchangeRates fetches the exchange rates body verbatim
func (c *CoinGeckoClient) FetchExchangeRates() (ExchangeRatesResponse, error) {
	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		requestBuilder := NewExchangeRatesRequestBuilder(baseURL)
		requestBuilder.WithApiKey(apiKey.Key, apiKey.Type)

		request, err := requestBuilder.Build()
		if err != nil {
			log.Printf("%s: Error building request with key type %v: %v", logPrefix, apiKey.Type, err)
			return nil, false, err
		}

		resp, body, _, err := c.httpClient.ExecuteRequest(request)
		if err != nil {
			return nil, false, err
		}
		resp.Body.Close()

		// Validate it is JSON without touching the values, then keep the raw bytes
		if !json.Valid(body) {
			err := fmt.Errorf("invalid JSON response")
			log.Printf("%s: %v", logPrefix, err)
			return nil, false, err
		}

		c.successfulFetch.Store(true)

		return ExchangeRatesResponse(body), true, nil
	}

	onFailed := cg.CreateFailCallback(c.keyManager)
	availableKeys := c.keyManager.GetAvailableKeys()

	result, err := cg.TryWithKeys(availableKeys, logPrefix, executor, onFailed)
	if err != nil {
		return nil, err
	}

	return result.(ExchangeRatesResponse), nil
}
