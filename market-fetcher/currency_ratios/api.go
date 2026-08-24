package currency_ratios

import (
	"encoding/json"
	"log"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

const logPrefix = "CoinGecko-CurrencyRatios"

//go:generate mockgen -destination=mocks/client.go . IClient

// IClient defines the API operations the ratios service needs
type IClient interface {
	// FetchReferencePrices fetches the reference coin prices in every currency,
	// including 24h changes, in a single request
	FetchReferencePrices(referenceCoins []string, currencies []string) (SimplePricePayload, error)
	Healthy() bool
}

// CoinGeckoClient implements IClient for CoinGecko
type CoinGeckoClient struct {
	config          *config.Config
	keyManager      cg.IAPIKeyManager
	httpClient      *cg.HTTPClientWithRetries
	successfulFetch atomic.Bool
}

// NewCoinGeckoClient creates a new CoinGecko client for the ratios service
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

// FetchReferencePrices performs the single simple/price call ratios are derived from
func (c *CoinGeckoClient) FetchReferencePrices(referenceCoins []string, currencies []string) (SimplePricePayload, error) {
	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		requestBuilder := NewRatiosRequestBuilder(baseURL).
			WithIds(referenceCoins).
			WithCurrencies(currencies).
			WithInclude24hChange().
			WithFullPrecision()
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

		var payload SimplePricePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("%s: Error parsing JSON response: %v", logPrefix, err)
			return nil, false, err
		}

		c.successfulFetch.Store(true)

		return payload, true, nil
	}

	onFailed := cg.CreateFailCallback(c.keyManager)
	availableKeys := c.keyManager.GetAvailableKeys()

	result, err := cg.TryWithKeys(availableKeys, logPrefix, executor, onFailed)
	if err != nil {
		return nil, err
	}

	return result.(SimplePricePayload), nil
}
