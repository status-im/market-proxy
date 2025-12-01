package fetcher_by_id

import (
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

// Client handles API requests to CoinGecko for generic endpoints
type Client struct {
	cfg             *config.Config
	fetcherCfg      *config.FetcherByIdConfig
	httpClient      *cg.HTTPClientWithRetries
	keyManager      cg.IAPIKeyManager
	metricsWriter   *metrics.MetricsWriter
	successfulFetch atomic.Bool
}

// NewClient creates a new generic CoinGecko client
func NewClient(cfg *config.Config, fetcherCfg *config.FetcherByIdConfig, metricsWriter *metrics.MetricsWriter) *Client {
	retryOpts := cg.DefaultRetryOptions()
	retryOpts.LogPrefix = fmt.Sprintf("CoinGecko-%s", fetcherCfg.Name)

	return &Client{
		cfg:           cfg,
		fetcherCfg:    fetcherCfg,
		httpClient:    cg.NewHTTPClientWithRetries(retryOpts, metricsWriter, cg.GetRateLimiterManagerInstance()),
		keyManager:    cg.NewAPIKeyManager(cfg.APITokens),
		metricsWriter: metricsWriter,
	}
}

// Healthy returns true if at least one fetch was successful
func (c *Client) Healthy() bool {
	return c.successfulFetch.Load()
}

// FetchSingle fetches data for a single ID
// Returns the raw JSON response
func (c *Client) FetchSingle(id string) ([]byte, error) {
	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		baseURL := cg.GetApiBaseUrl(c.cfg, apiKey.Type)

		reqBuilder := NewRequestBuilder(baseURL, c.fetcherCfg).
			WithAPIKey(apiKey.Key, apiKey.Type)

		req, err := reqBuilder.BuildSingleRequest(id)
		if err != nil {
			return nil, false, fmt.Errorf("failed to build request: %w", err)
		}

		resp, body, _, err := c.httpClient.ExecuteRequest(req)
		if err != nil {
			return nil, false, err
		}
		defer resp.Body.Close()

		return body, true, nil
	}

	onFailed := cg.CreateFailCallback(c.keyManager)
	availableKeys := c.keyManager.GetAvailableKeys()

	result, err := cg.TryWithKeys(availableKeys, fmt.Sprintf("CoinGecko-%s", c.fetcherCfg.Name), executor, onFailed)
	if err != nil {
		return nil, err
	}

	c.successfulFetch.Store(true)
	return result.([]byte), nil
}

// FetchBatch fetches data for multiple IDs in a single request
// Returns a map of ID -> raw JSON response
func (c *Client) FetchBatch(ids []string) (map[string][]byte, error) {
	if len(ids) == 0 {
		return make(map[string][]byte), nil
	}

	executor := func(apiKey cg.APIKey) (interface{}, bool, error) {
		baseURL := cg.GetApiBaseUrl(c.cfg, apiKey.Type)

		reqBuilder := NewRequestBuilder(baseURL, c.fetcherCfg).
			WithAPIKey(apiKey.Key, apiKey.Type)

		req, err := reqBuilder.BuildBatchRequest(ids)
		if err != nil {
			return nil, false, fmt.Errorf("failed to build request: %w", err)
		}

		resp, body, _, err := c.httpClient.ExecuteRequest(req)
		if err != nil {
			return nil, false, err
		}
		defer resp.Body.Close()

		// Parse response - expecting a map where keys are IDs
		result, err := c.parseBatchResponse(body)
		if err != nil {
			return nil, false, err
		}

		return result, true, nil
	}

	onFailed := cg.CreateFailCallback(c.keyManager)
	availableKeys := c.keyManager.GetAvailableKeys()

	result, err := cg.TryWithKeys(availableKeys, fmt.Sprintf("CoinGecko-%s", c.fetcherCfg.Name), executor, onFailed)
	if err != nil {
		return nil, err
	}

	c.successfulFetch.Store(true)
	return result.(map[string][]byte), nil
}

// parseBatchResponse parses a batch response into a map of ID -> raw JSON
func (c *Client) parseBatchResponse(body []byte) (map[string][]byte, error) {
	// Try to parse as a map (most common batch response format)
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawMap); err != nil {
		log.Printf("%s: Failed to parse batch response as map: %v", c.fetcherCfg.Name, err)
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	result := make(map[string][]byte)
	for id, rawData := range rawMap {
		result[id] = []byte(rawData)
	}

	return result, nil
}

// FetchBatchInChunks fetches data for multiple IDs, splitting into chunks
// Returns a map of ID -> raw JSON response
func (c *Client) FetchBatchInChunks(ids []string, onChunk func(data map[string][]byte)) (map[string][]byte, error) {
	if len(ids) == 0 {
		return make(map[string][]byte), nil
	}

	chunkSize := c.fetcherCfg.GetChunkSize()
	result := make(map[string][]byte)

	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}

		chunk := ids[i:end]
		chunkData, err := c.FetchBatch(chunk)
		if err != nil {
			log.Printf("%s: Failed to fetch chunk %d-%d: %v", c.fetcherCfg.Name, i, end, err)
			continue
		}

		// Merge chunk data into result
		for id, data := range chunkData {
			result[id] = data
		}

		// Call chunk callback if provided
		if onChunk != nil {
			onChunk(chunkData)
		}
	}

	return result, nil
}
