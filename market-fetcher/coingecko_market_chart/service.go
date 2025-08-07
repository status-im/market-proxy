package coingecko_market_chart

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

const (
	MARKET_CHART_CACHE_PREFIX = "market_chart"
)

type Service struct {
	cache         cache.ICache
	config        *config.Config
	metricsWriter *metrics.MetricsWriter
	apiClient     IAPIClient
}

func NewService(cache cache.ICache, config *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarketCharts)
	apiClient := NewCoinGeckoClient(config)

	return &Service{
		cache:         cache,
		config:        config,
		metricsWriter: metricsWriter,
		apiClient:     apiClient,
	}
}

func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}
	return nil
}

func (s *Service) Stop() {
}

func (s *Service) MarketChart(params MarketChartParams) (MarketChartResponseData, error) {
	log.Printf("Loading market chart data for coin %s, currency=%s, days=%s",
		params.ID, params.Currency, params.Days)

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	originalParams := params

	if params.Currency == "" {
		params.Currency = "usd"
	}
	if params.Days == "" {
		params.Days = "30"
	}

	// Round up parameters to maximize cache utilization
	roundedParams := RoundUpMarketChartParams(params, s.config.CoingeckoMarketChart.DailyDataThreshold)

	cacheKey := s.createCacheKey(roundedParams)

	var chartData map[string]interface{}

	cachedData, err := s.getCachedData(cacheKey)
	if err == nil && cachedData != nil {
		log.Printf("Returning cached market chart data for coin %s", params.ID)
		chartData = cachedData
	} else {
		log.Printf("ICache miss for market chart %s, fetching from API with rounded params", params.ID)
		fetchedData, err := s.apiClient.FetchMarketChart(roundedParams)
		if err != nil {
			log.Printf("apiClient.FetchMarketChart failed: %v", err)
			return nil, fmt.Errorf("failed to fetch market chart data: %w", err)
		}

		if err := s.cacheData(cacheKey, fetchedData, roundedParams); err != nil {
			log.Printf("Failed to cache market chart data: %v", err)
		}

		chartData = s.convertBytesToInterface(fetchedData)
	}

	// Strip the data to match original request
	strippedData, err := StripMarketChartResponse(originalParams, chartData)
	if err != nil {
		log.Printf("Failed to strip data: %v", err)
		return MarketChartResponseData(chartData), nil
	}

	return MarketChartResponseData(strippedData), nil
}

func (s *Service) Healthy() bool {
	if s.apiClient != nil {
		return s.apiClient.Healthy()
	}
	return false
}

func (s *Service) createCacheKey(params MarketChartParams) string {
	baseKey := fmt.Sprintf("%s:%s:%s", MARKET_CHART_CACHE_PREFIX, params.ID, params.Currency)

	baseKey += fmt.Sprintf(":days:%s", params.Days)

	if params.Interval != "" {
		baseKey += fmt.Sprintf(":interval:%s", params.Interval)
	}

	return baseKey
}

func (s *Service) getCachedData(cacheKey string) (map[string]interface{}, error) {
	cacheKeys := []string{cacheKey}
	cachedData, _, err := s.cache.Get(cacheKeys)
	if err != nil {
		return nil, err
	}

	if data, exists := cachedData[cacheKey]; exists {
		var rawData map[string]json.RawMessage
		if err := json.Unmarshal(data, &rawData); err != nil {
			return nil, err
		}

		byteData := make(map[string][]byte)
		for key, value := range rawData {
			byteData[key] = []byte(value)
		}

		return s.convertBytesToInterface(byteData), nil
	}

	return nil, fmt.Errorf("data not found in cache")
}

func (s *Service) cacheData(cacheKey string, chartData map[string][]byte, params MarketChartParams) error {
	rawData := make(map[string]json.RawMessage)
	for key, value := range chartData {
		rawData[key] = json.RawMessage(value)
	}

	dataBytes, err := json.Marshal(rawData)
	if err != nil {
		return err
	}

	cacheDataMap := map[string][]byte{
		cacheKey: dataBytes,
	}

	ttl := s.selectTTL(params)

	return s.cache.Set(cacheDataMap, ttl)
}

func (s *Service) selectTTL(params MarketChartParams) time.Duration {
	if params.Days == "max" {
		return s.config.CoingeckoMarketChart.DailyTTL
	}

	if daysStr := params.Days; daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil {
			if days <= s.config.CoingeckoMarketChart.DailyDataThreshold {
				return s.config.CoingeckoMarketChart.HourlyTTL
			} else {
				return s.config.CoingeckoMarketChart.DailyTTL
			}
		}
	}

	return s.config.CoingeckoMarketChart.DailyTTL
}

func (s *Service) convertBytesToInterface(data map[string][]byte) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		var jsonValue interface{}
		if err := json.Unmarshal(value, &jsonValue); err == nil {
			result[key] = jsonValue
		}
	}

	return result
}
