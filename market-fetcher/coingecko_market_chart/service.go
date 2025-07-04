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
	// Cache key prefix for market chart data
	MARKET_CHART_CACHE_PREFIX = "market_chart"
	// Default TTL for market chart cache (in seconds)
	MARKET_CHART_DEFAULT_TTL = 300 // 5 minutes
)

// Service provides market chart data fetching functionality with caching
type Service struct {
	cache         cache.Cache
	config        *config.Config
	metricsWriter *metrics.MetricsWriter
	apiClient     APIClient
}

// NewService creates a new market chart service with the given cache and config
func NewService(cache cache.Cache, config *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarkets) // Reuse markets service metrics
	apiClient := NewCoinGeckoClient(config)

	return &Service{
		cache:         cache,
		config:        config,
		metricsWriter: metricsWriter,
		apiClient:     apiClient,
	}
}

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}
	return nil
}

// Stop implements core.Interface
func (s *Service) Stop() {
	// Market chart service doesn't need shutdown logic for now
	// Cache will handle its own cleanup
}

// MarketChart fetches market chart data for a specific coin with caching
func (s *Service) MarketChart(params MarketChartParams) (map[string][]byte, error) {
	log.Printf("Loading market chart data for coin %s, currency=%s, days=%s",
		params.ID, params.Currency, params.Days)

	// Store original parameters for later strip operation
	originalParams := params

	// Set default values if not provided
	if params.Currency == "" {
		params.Currency = "usd"
	}
	if params.Days == "" {
		params.Days = "30" // Default to 30 days
	}

	// Enrich parameters to maximize cache utilization
	enrichedParams := EnrichMarketChartParams(params)

	// Create cache key based on enriched parameters
	cacheKey := s.createCacheKey(enrichedParams)

	// Check cache first
	cachedData, err := s.getCachedData(cacheKey)
	if err == nil && cachedData != nil {
		log.Printf("Returning cached market chart data for coin %s", params.ID)

		// Strip the cached data to match original request
		strippedData, err := StripMarketChartResponse(originalParams, cachedData)
		if err != nil {
			log.Printf("Failed to strip cached data: %v", err)
			// If stripping fails, return original cached data
			return cachedData, nil
		}

		return strippedData, nil
	}

	// Cache miss - fetch from API using enriched parameters
	log.Printf("Cache miss for market chart %s, fetching from API with enriched params", params.ID)
	chartData, err := s.apiClient.FetchMarketChart(enrichedParams)
	if err != nil {
		log.Printf("apiClient.FetchMarketChart failed: %v", err)
		return nil, fmt.Errorf("failed to fetch market chart data: %w", err)
	}

	// Cache the result using enriched parameters for TTL calculation
	if err := s.cacheData(cacheKey, chartData, enrichedParams); err != nil {
		log.Printf("Failed to cache market chart data: %v", err)
		// Don't fail the request if caching fails
	}

	// Strip the fetched data to match original request
	strippedData, err := StripMarketChartResponse(originalParams, chartData)
	if err != nil {
		log.Printf("Failed to strip fetched data: %v", err)
		// If stripping fails, return original data
		return chartData, nil
	}

	log.Printf("Successfully fetched, cached, and stripped market chart data for coin %s",
		params.ID)

	return strippedData, nil
}

// Healthy checks if the service is operational
func (s *Service) Healthy() bool {
	if s.apiClient != nil {
		return s.apiClient.Healthy()
	}
	return false
}

// createCacheKey creates a cache key based on request parameters
func (s *Service) createCacheKey(params MarketChartParams) string {
	baseKey := fmt.Sprintf("%s:%s:%s", MARKET_CHART_CACHE_PREFIX, params.ID, params.Currency)

	// Add time-based parameters
	baseKey += fmt.Sprintf(":days:%s", params.Days)

	// Add interval if specified
	if params.Interval != "" {
		baseKey += fmt.Sprintf(":interval:%s", params.Interval)
	}

	return baseKey
}

// getCachedData retrieves market chart data from cache
func (s *Service) getCachedData(cacheKey string) (map[string][]byte, error) {
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

		// Convert RawMessage to bytes
		result := make(map[string][]byte)
		for key, value := range rawData {
			result[key] = []byte(value)
		}

		return result, nil
	}

	return nil, fmt.Errorf("data not found in cache")
}

// cacheData stores market chart data in cache with smart TTL selection
func (s *Service) cacheData(cacheKey string, chartData map[string][]byte, params MarketChartParams) error {
	dataBytes, err := json.Marshal(chartData)
	if err != nil {
		return err
	}

	cacheData := map[string][]byte{
		cacheKey: dataBytes,
	}

	// Select TTL based on days parameter for smart caching
	ttl := s.selectTTL(params)

	return s.cache.Set(cacheData, ttl)
}

// selectTTL chooses appropriate TTL based on request parameters
func (s *Service) selectTTL(params MarketChartParams) time.Duration {
	// Parse days parameter to determine data granularity
	if params.Days == "max" {
		// For max days, use longer TTL (12 hours)
		return 12 * time.Hour
	}

	// Try to parse days as integer
	if daysStr := params.Days; daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil {
			// For days <= 90: hourly data, use shorter TTL (30 minutes)
			// For days > 90: daily data, use longer TTL (12 hours)
			if days <= 90 {
				return 30 * time.Minute
			} else {
				return 12 * time.Hour
			}
		}
	}

	// Default fallback TTL
	return time.Duration(MARKET_CHART_DEFAULT_TTL) * time.Second
}
