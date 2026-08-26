package coingecko_assets_platforms

import (
	"encoding/json"
	"log"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/proxy-common/apikeys"
)

type CoinGeckoClient struct {
	config          *config.Config
	keyManager      apikeys.IAPIKeyManager
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
		httpClient: cg.NewHTTPClientWithRetries(retryOpts, metricsWriter, cg.GetRateLimiterManagerInstance()),
	}
}

func (c *CoinGeckoClient) Healthy() bool {
	return c.successfulFetch.Load()
}

func (c *CoinGeckoClient) FetchAssetsPlatforms(params AssetsPlatformsParams) (AssetsPlatformsResponse, error) {
	executor := func(apiKey apikeys.APIKey) (interface{}, bool, error) {
		baseURL := cg.GetApiBaseUrl(c.config, apiKey.Type)

		requestBuilder := NewAssetsPlatformsRequestBuilder(baseURL).
			WithFilter(params.Filter).
			WithApiKey(apiKey.Key, apiKey.Type)

		request, err := requestBuilder.Build()
		if err != nil {
			log.Printf("CoinGecko-AssetsPlatforms: Error building request with key type %v: %v", apiKey.Type, err)
			return nil, false, err
		}

		resp, body, _, err := c.httpClient.ExecuteRequest(request)
		if err != nil {
			return nil, false, err
		}

		resp.Body.Close()

		c.successfulFetch.Store(true)

		var result interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("CoinGecko-AssetsPlatforms: Error parsing JSON response: %v", err)
			return nil, false, err
		}

		return result, true, nil
	}

	onFailed := apikeys.CreateFailCallback(c.keyManager)
	availableKeys := c.keyManager.GetAvailableKeys()

	result, err := apikeys.TryWithKeys(availableKeys, "CoinGecko-AssetsPlatforms", executor, onFailed)
	if err != nil {
		return nil, err
	}

	return result.(AssetsPlatformsResponse), nil
}
