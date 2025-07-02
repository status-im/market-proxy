package coingecko_assets_platforms

import (
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

type CoinGeckoClient struct {
	config          *config.Config
	keyManager      cg.APIKeyManagerInterface
	httpClient      *cg.HTTPClientWithRetries
	successfulFetch atomic.Bool
}

func NewCoinGeckoClient(cfg *config.Config) *CoinGeckoClient {
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = "CoinGecko-AssetsPlatforms"

	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePlatforms)

	return &CoinGeckoClient{
		config:     cfg,
		keyManager: cg.NewAPIKeyManager(cfg.APITokens),
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsWriter),
	}
}

func (c *CoinGeckoClient) Healthy() bool {
	return c.successfulFetch.Load()
}

func (c *CoinGeckoClient) FetchAssetsPlatforms(params AssetsPlatformsParams) (AssetsPlatformsResponse, error) {
	availableKeys := c.keyManager.GetAvailableKeys()

	var lastError error

	for _, apiKey := range availableKeys {
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		requestBuilder := NewAssetsPlatformsRequestBuilder(baseURL)

		if params.Filter != "" {
			requestBuilder.WithFilter(params.Filter)
		}

		if apiKey.Key != "" {
			requestBuilder.builder.WithApiKey(apiKey.Key, apiKey.Type)
		}

		request, err := requestBuilder.builder.Build()
		if err != nil {
			log.Printf("CoinGecko-AssetsPlatforms: Error building request with key type %v: %v", apiKey.Type, err)
			lastError = err
			continue
		}

		resp, body, _, err := c.httpClient.ExecuteRequest(request)

		if err != nil {
			log.Printf("CoinGecko-AssetsPlatforms: Request failed with key type %v: %v", apiKey.Type, err)

			if apiKey.Key != "" {
				log.Printf("CoinGecko-AssetsPlatforms: Marking key as failed and adding to backoff")
				c.keyManager.MarkKeyAsFailed(apiKey.Key)
			}

			lastError = err
			continue
		}

		resp.Body.Close()

		c.successfulFetch.Store(true)

		var result interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("CoinGecko-AssetsPlatforms: Error parsing JSON response: %v", err)
			return nil, err
		}

		return result, nil
	}

	return nil, fmt.Errorf("all API keys failed, last error: %v", lastError)
}
