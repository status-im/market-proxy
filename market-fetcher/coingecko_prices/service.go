package coingecko_prices

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/metrics"

	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
)

// Service provides price fetching functionality with caching
type Service struct {
	cache                          cache.ICache
	fetcher                        *ChunksFetcher
	config                         *config.Config
	metricsWriter                  *metrics.MetricsWriter
	subscriptionManager            *events.SubscriptionManager
	periodicUpdater                IPeriodicUpdater
	marketsService                 interfaces.IMarketsService
	tokensService                  interfaces.ITokensService
	marketUpdateSubscription       events.ISubscription
	tokenUpdateSubscription        events.ISubscription
	marketsInitializedSubscription events.ISubscription
}

// NewService creates a new price service with the given cache and config
func NewService(cache cache.ICache, config *config.Config, marketsService interfaces.IMarketsService, tokensService interfaces.ITokensService) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePrices)
	apiClient := NewCoinGeckoClient(config, metricsWriter)

	chunkSize := config.CoingeckoPrices.ChunkSize
	if chunkSize <= 0 {
		chunkSize = DEFAULT_CHUNK_SIZE
	}

	requestDelay := config.CoingeckoPrices.RequestDelay
	requestDelayMs := int(requestDelay.Milliseconds())
	if requestDelayMs < 0 {
		requestDelayMs = DEFAULT_REQUEST_DELAY
	}

	fetcher := NewChunksFetcher(apiClient, chunkSize, requestDelayMs)

	service := &Service{
		cache:               cache,
		fetcher:             fetcher,
		config:              config,
		metricsWriter:       metricsWriter,
		subscriptionManager: events.NewSubscriptionManager(),
		marketsService:      marketsService,
		tokensService:       tokensService,
	}

	// Create periodic updater
	service.periodicUpdater = NewPeriodicUpdater(&config.CoingeckoPrices, apiClient)

	// Set callbacks to handle data updates
	service.periodicUpdater.SetOnTopPricesUpdatedCallback(service.handleTopPricesUpdate)
	service.periodicUpdater.SetOnMissingExtraIdsUpdatedCallback(service.handleMissingExtraIdsUpdate)

	return service
}

// handleTopPricesUpdate handles top prices update by caching tokens and emitting events
func (s *Service) handleTopPricesUpdate(ctx context.Context, tier config.PriceTier, pricesData map[string][]byte) {
	// ICache prices by individual token IDs
	err := s.cachePricesByID(pricesData)
	if err != nil {
		log.Printf("Failed to cache prices data by id: %v", err)
	}

	log.Printf("Prices service cache update complete: %d", len(pricesData))
	s.subscriptionManager.Emit(ctx)
}

// handleMissingExtraIdsUpdate handles missing extra IDs update by caching tokens and emitting events
func (s *Service) handleMissingExtraIdsUpdate(ctx context.Context, pricesData map[string][]byte) {
	// ICache missing tokens by their IDs
	err := s.cachePricesByID(pricesData)
	if err != nil {
		log.Printf("Failed to cache missing extra IDs: %v", err)
	}

	log.Printf("Prices service cache update complete: extra ids %d", len(pricesData))
	s.subscriptionManager.Emit(ctx)
}

// getMaxTokenLimit calculates the maximum token limit from prices tiers configuration
func (s *Service) getMaxTokenLimit() int {
	if s.config == nil {
		return 100000 // fallback
	}

	maxTokenTo := 0
	for _, tier := range s.config.CoingeckoPrices.Tiers {
		if tier.TokenTo > maxTokenTo {
			maxTokenTo = tier.TokenTo
		}
	}

	if maxTokenTo == 0 {
		return 100000 // fallback if no tiers configured
	}

	return maxTokenTo
}

// onMarketListChanged is called when market list is updated
func (s *Service) onMarketListChanged() {
	if s.marketsService == nil {
		return
	}

	// Get maximum available top market IDs from markets service
	// Calculate limit based on the maximum TokenTo from prices tiers configuration
	maxLimit := s.getMaxTokenLimit()
	topMarketIds, err := s.marketsService.TopMarketIds(maxLimit)
	if err != nil {
		log.Printf("Failed to get top market IDs: %v", err)
		return
	}

	if s.periodicUpdater != nil {
		s.periodicUpdater.SetTopMarketIds(topMarketIds)
	}
}

// onTokenListChanged is called when token list is updated
func (s *Service) onTokenListChanged() {
	if s.tokensService == nil {
		return
	}

	tokenIDs := s.tokensService.GetTokenIds()

	if s.periodicUpdater != nil {
		s.periodicUpdater.SetExtraIds(tokenIDs)
	}
}

// cachePricesByID caches price data by individual token IDs
func (s *Service) cachePricesByID(pricesData map[string][]byte) error {
	if len(pricesData) == 0 {
		return nil
	}

	// Prepare cache data
	cacheData := make(map[string][]byte)
	for tokenID, data := range pricesData {
		// Create cache key for this token
		cacheKey := createTokenIDCacheKey(tokenID)
		cacheData[cacheKey] = data
	}

	// ICache prices data
	err := s.cache.Set(cacheData, s.config.CoingeckoPrices.GetTTL())
	if err != nil {
		log.Printf("Failed to cache prices data: %v", err)
		return fmt.Errorf("failed to cache prices data: %w", err)
	}

	return nil
}

// createTokenIDCacheKey creates a cache key for individual token ID
func createTokenIDCacheKey(tokenID string) string {
	return fmt.Sprintf("price:id:%s", tokenID)
}

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}

	// Subscribe to market list updates
	if s.marketsService != nil {
		s.marketUpdateSubscription = s.marketsService.SubscribeTopMarketsUpdate().
			Watch(ctx, s.onMarketListChanged, true)

		// Subscribe to markets initialization events
		s.marketsInitializedSubscription = s.marketsService.SubscribeInitialized().
			Watch(ctx, func() {
				log.Printf("Markets service initialized - triggering force update of prices")
				if s.periodicUpdater != nil {
					s.periodicUpdater.ForceUpdate(ctx)
				}
			}, false)
	}

	// Subscribe to token list updates
	if s.tokensService != nil {
		s.tokenUpdateSubscription = s.tokensService.SubscribeOnTokensUpdate().
			Watch(ctx, s.onTokenListChanged, true)
	}

	// Start periodic updater
	if s.periodicUpdater != nil {
		if err := s.periodicUpdater.Start(ctx); err != nil {
			return fmt.Errorf("failed to start periodic updater: %w", err)
		}
	}

	return nil
}

// Stop implements core.Interface
func (s *Service) Stop() {
	// Cancel all subscriptions
	if s.marketUpdateSubscription != nil {
		s.marketUpdateSubscription.Cancel()
		s.marketUpdateSubscription = nil
	}

	if s.marketsInitializedSubscription != nil {
		s.marketsInitializedSubscription.Cancel()
		s.marketsInitializedSubscription = nil
	}

	if s.tokenUpdateSubscription != nil {
		s.tokenUpdateSubscription.Cancel()
		s.tokenUpdateSubscription = nil
	}

	// Stop periodic updater
	if s.periodicUpdater != nil {
		s.periodicUpdater.Stop()
	}
	// ICache will handle its own cleanup
}

// SimplePrices fetches prices for the given parameters using cache only
// Returns raw CoinGecko JSON response with cache status
func (s *Service) SimplePrices(ctx context.Context, params interfaces.PriceParams) (resp interfaces.SimplePriceResponse, cacheStatus interfaces.CacheStatus, err error) {
	if len(params.IDs) == 0 {
		return interfaces.SimplePriceResponse{}, interfaces.CacheStatusFull, nil
	}

	// Create cache keys for individual token IDs
	cacheKeys := make([]string, len(params.IDs))
	for i, tokenID := range params.IDs {
		cacheKeys[i] = createTokenIDCacheKey(tokenID)
	}

	// Get data from cache only
	cachedData, missingKeys, err := s.cache.Get(cacheKeys)
	if err != nil {
		log.Printf("Failed to check cache: %v", err)
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("failed to check cache: %w", err)
	}

	// Build response from cached data, preserving order from params.IDs
	fullResponse := make(interfaces.SimplePriceResponse)
	for i, tokenID := range params.IDs {
		cacheKey := cacheKeys[i]
		if tokenBytes, exists := cachedData[cacheKey]; exists {
			var tokenData map[string]interface{}
			if err := json.Unmarshal(tokenBytes, &tokenData); err == nil {
				fullResponse[tokenID] = tokenData
			}
		}
	}

	// Log missing keys but don't fetch from API - only return cached data
	if len(missingKeys) > 0 {
		log.Printf("Missing %d tokens in cache", len(missingKeys))
	}

	// Determine cache status
	if len(missingKeys) == 0 && len(cachedData) == len(cacheKeys) {
		cacheStatus = interfaces.CacheStatusFull
	} else if len(cachedData) > 0 {
		cacheStatus = interfaces.CacheStatusPartial
	} else {
		cacheStatus = interfaces.CacheStatusMiss
	}

	// Filter the response according to user parameters
	filteredResponse := stripResponse(fullResponse, params)

	return filteredResponse, cacheStatus, nil
}

// getConfigCurrencies returns the currencies from config, with fallback to default
func (s *Service) getConfigCurrencies() []string {
	if s.config != nil && len(s.config.CoingeckoPrices.Currencies) > 0 {
		return s.config.CoingeckoPrices.Currencies
	}
	// Fallback to default currencies if config is not available or empty
	return []string{"usd", "eur", "btc", "eth"}
}

// TopPrices fetches prices for top tokens with specified limit and currencies
// Similar to TopMarkets in markets service, provides clean interface for token price fetching
func (s *Service) TopPrices(ctx context.Context, limit int, currencies []string) (interfaces.SimplePriceResponse, interfaces.CacheStatus, error) {
	// Get top market IDs based on the limit
	if s.marketsService == nil {
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("markets service not available")
	}

	topMarketIds, err := s.marketsService.TopMarketIds(limit)
	if err != nil {
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("failed to get top market IDs: %w", err)
	}

	params := interfaces.PriceParams{
		IDs:                  topMarketIds,
		Currencies:           currencies,
		IncludeMarketCap:     true,
		Include24hrVol:       true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
	}

	return s.SimplePrices(ctx, params)
}

// Healthy checks if the service is operational
func (s *Service) Healthy() bool {
	// Check if the periodic updater is healthy
	if s.periodicUpdater != nil {
		return s.periodicUpdater.Healthy()
	}
	return false
}

// SubscribeTopPricesUpdate subscribes to prices update notifications
func (s *Service) SubscribeTopPricesUpdate() events.ISubscription {
	return s.subscriptionManager.Subscribe()
}
