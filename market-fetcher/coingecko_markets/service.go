package coingecko_markets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/status-im/market-proxy/cache"
	cfg "github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/status-im/market-proxy/metrics"
)

const (
	MARKETS_DEFAULT_CHUNK_SIZE    = 250  // CoinGecko's API max per_page value
	MARKETS_DEFAULT_REQUEST_DELAY = 1000 // 1 second in milliseconds
	ID_FIELD                      = "id"
)

// Service provides markets data fetching functionality with caching
type Service struct {
	cache               cache.Cache
	config              *cfg.Config
	metricsWriter       *metrics.MetricsWriter
	subscriptionManager *events.SubscriptionManager
	periodicUpdater     *PeriodicUpdater
	tokensService       interfaces.CoingeckoTokensService
	tokenUpdateCh       chan struct{}
	cancelFunc          context.CancelFunc
}

func NewService(cache cache.Cache, config *cfg.Config, tokensService interfaces.CoingeckoTokensService) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarkets)
	apiClient := NewCoinGeckoClient(config)

	service := &Service{
		cache:               cache,
		config:              config,
		metricsWriter:       metricsWriter,
		subscriptionManager: events.NewSubscriptionManager(),
		tokensService:       tokensService,
	}

	// Create periodic updater
	service.periodicUpdater = NewPeriodicUpdater(&config.CoingeckoMarkets, apiClient)

	// Set onUpdateTierPages callback to cache tokens and emit events to subscription manager
	service.periodicUpdater.SetOnUpdateTierPagesCallback(service.handleTierPagesUpdate)

	// Set onUpdateMissingExtraIds callback to cache missing tokens by ID
	service.periodicUpdater.SetOnUpdateMissingExtraIdsCallback(service.handleMissingExtraIdsUpdate)

	return service
}

// handleTierPagesUpdate handles tier pages update by caching tokens and emitting events
func (s *Service) handleTierPagesUpdate(ctx context.Context, tier cfg.MarketTier, pagesData []PageData) {
	// Cache by individual ids
	for _, pageData := range pagesData {
		_, err := s.cacheTokensByID(pageData.Data)
		if err != nil {
			log.Printf("Failed to cache markets data by id: %v", err)
		}
	}

	// Cache pages
	_, err := s.cacheTokensPage(tier, pagesData)
	if err != nil {
		log.Printf("Failed to cache page data: %v", err)
	}
	s.subscriptionManager.Emit(ctx)
}

// handleMissingExtraIdsUpdate handles missing extra IDs update by caching tokens and emitting events
func (s *Service) handleMissingExtraIdsUpdate(ctx context.Context, tokensData [][]byte) {
	// Cache missing tokens by their IDs
	_, err := s.cacheTokensByID(tokensData)
	if err != nil {
		log.Printf("Failed to cache missing extra IDs: %v", err)
	} else {
		log.Printf("Successfully cached %d missing extra IDs", len(tokensData))
	}
	s.subscriptionManager.Emit(ctx)
}

// onTokenListChanged is called when token list is updated
func (s *Service) onTokenListChanged() {
	if s.tokensService == nil {
		return
	}

	tokenIDs := s.tokensService.GetTokenIds()

	log.Printf("Token list changed, updating periodic updater with %d extra IDs", len(tokenIDs))

	if s.periodicUpdater != nil {
		s.periodicUpdater.SetExtraIds(tokenIDs)
	}
}

// handleTokenUpdates handles token update notifications
func (s *Service) handleTokenUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.tokenUpdateCh:
			s.onTokenListChanged()
		}
	}
}

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}

	// Subscribe to token list updates
	if s.tokensService != nil {
		s.tokenUpdateCh = s.tokensService.SubscribeOnTokensUpdate()

		// Create cancelable context for the goroutine
		goroutineCtx, cancel := context.WithCancel(ctx)
		s.cancelFunc = cancel
		go s.handleTokenUpdates(goroutineCtx)

		// Initial call to set extra IDs
		s.onTokenListChanged()
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
	// Cancel the goroutine first
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}

	// Unsubscribe from token updates
	if s.tokensService != nil && s.tokenUpdateCh != nil {
		s.tokensService.Unsubscribe(s.tokenUpdateCh)
		s.tokenUpdateCh = nil
	}

	// Stop periodic updater
	if s.periodicUpdater != nil {
		s.periodicUpdater.Stop()
	}
	// Cache will handle its own cleanup
}

// cacheTokensByID parses tokens data and caches each token by its CoinGecko ID
func (s *Service) cacheTokensByID(tokensData [][]byte) ([]interface{}, error) {
	// Parse tokens data
	marketData, cacheData, err := parseTokensData(tokensData)
	if err != nil {
		return nil, err
	}

	// Cache tokens directly
	if len(cacheData) > 0 {
		err := s.cache.Set(cacheData, s.config.CoingeckoMarkets.GetTTL())
		if err != nil {
			log.Printf("Failed to cache tokens data: %v", err)
			return nil, fmt.Errorf("failed to cache tokens data: %w", err)
		}
		log.Printf("Successfully cached %d tokens by their coingecko id", len(cacheData))
	}

	return marketData, nil
}

// cacheTokensPage caches page data for page-based requests
func (s *Service) cacheTokensPage(tier cfg.MarketTier, pagesData []PageData) (map[int]interface{}, error) {
	// Parse pages data
	pageMapping, cacheData, err := parsePagesData(pagesData)
	if err != nil {
		return nil, err
	}

	// Cache pages data
	if len(cacheData) > 0 {
		err := s.cache.Set(cacheData, s.config.CoingeckoMarkets.GetTTL())
		if err != nil {
			log.Printf("Failed to cache page data: %v", err)
			return nil, fmt.Errorf("failed to cache page data: %w", err)
		}
		log.Printf("Successfully cached %d pages for tier '%s'", len(cacheData), tier.Name)
	}

	return pageMapping, nil
}

// Markets fetches markets data using cache with specified parameters
// Returns full CoinGecko markets response in APIResponse format
func (s *Service) Markets(params interfaces.MarketsParams) (interfaces.MarketsResponse, interfaces.CacheStatus, error) {
	// Check if specific IDs are requested
	if len(params.IDs) > 0 {
		return s.MarketsByIds(params)
	}

	// TODO: Implement general markets fetching without specific IDs
	log.Printf("Markets called without specific IDs - returning empty array (TODO: implement general fetching)")
	return interfaces.MarketsResponse([]interface{}{}), interfaces.CacheStatusMiss, nil
}

// MarketsByIds fetches markets data for specific token IDs using cache only
func (s *Service) MarketsByIds(params interfaces.MarketsParams) (response interfaces.MarketsResponse, cacheStatus interfaces.CacheStatus, err error) {
	log.Printf("Loading markets data for %d specific IDs from cache with currency=%s", len(params.IDs), params.Currency)
	params = s.getParamsOverride(params)
	cacheKeys := createCacheKeys(params)

	cachedData, missingKeys, err := s.cache.Get(cacheKeys)
	if err != nil {
		log.Printf("Failed to check cache: %v", err)
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("failed to check cache: %w", err)
	}

	// Build response from cached data, preserving order from cacheKeys
	marketData := make([]interface{}, 0, len(cacheKeys))
	for _, cacheKey := range cacheKeys {
		if tokenBytes, exists := cachedData[cacheKey]; exists {
			var tokenData interface{}
			if err := json.Unmarshal(tokenBytes, &tokenData); err == nil {
				marketData = append(marketData, tokenData)
			}
		}
	}

	// Log missing keys but don't fetch from API - only return cached data
	if len(missingKeys) > 0 {
		log.Printf("Missing %d tokens in cache - service only returns pre-warmed data from periodic updater", len(missingKeys))
	}

	// Determine cache status
	if len(missingKeys) == 0 && len(cachedData) == len(cacheKeys) {
		cacheStatus = interfaces.CacheStatusFull
		log.Printf("Returning cached data for all %d tokens", len(marketData))
	} else if len(cachedData) > 0 {
		cacheStatus = interfaces.CacheStatusPartial
		log.Printf("Returning partial cached data for %d tokens (requested %d, missing %d)", len(marketData), len(cacheKeys), len(missingKeys))
	} else {
		cacheStatus = interfaces.CacheStatusMiss
		log.Printf("No tokens found in cache for %d requested tokens", len(cacheKeys))
	}

	return interfaces.MarketsResponse(marketData), cacheStatus, nil
}

// getMaxTokenLimit calculates the maximum token limit from tiers configuration
func (s *Service) getMaxTokenLimit() int {
	if s.config == nil {
		return MARKETS_DEFAULT_CHUNK_SIZE
	}

	// Apply parameters normalization from config to get the actual per_page
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  MARKETS_DEFAULT_CHUNK_SIZE,
	}
	params = s.getParamsOverride(params)
	perPage := params.PerPage

	maxPageTo := 0
	for _, tier := range s.config.CoingeckoMarkets.Tiers {
		if tier.PageTo > maxPageTo {
			maxPageTo = tier.PageTo
		}
	}

	if maxPageTo == 0 {
		return MARKETS_DEFAULT_CHUNK_SIZE
	}

	return maxPageTo * perPage
}

// Healthy checks if the service is operational
func (s *Service) Healthy() bool {
	// Check if the periodic updater is healthy
	if s.periodicUpdater != nil {
		return s.periodicUpdater.Healthy()
	}
	return false
}

// TopMarkets fetches top markets data for specified number of tokens from cache
func (s *Service) TopMarkets(limit int, currency string) (interfaces.MarketsResponse, error) {
	log.Printf("Loading top %d markets data from cache with currency=%s", limit, currency)

	// Set default limit if not provided or invalid
	if limit <= 0 {
		limit = 100
	}

	// Ensure limit doesn't exceed maximum available tokens
	maxLimit := s.getMaxTokenLimit()
	if limit > maxLimit {
		limit = maxLimit
		log.Printf("Limit adjusted to maximum available: %d", limit)
	}

	// Get top market token IDs from cache
	tokenIDs, err := s.TopMarketIds(limit)
	if err != nil {
		log.Printf("Failed to get top market IDs: %v", err)
		return nil, fmt.Errorf("failed to get top market IDs: %w", err)
	}

	if len(tokenIDs) == 0 {
		log.Printf("No top market IDs found in cache - service only returns pre-warmed data from periodic updater")
		return interfaces.MarketsResponse([]interface{}{}), nil
	}

	// Create parameters for markets request
	params := interfaces.MarketsParams{
		Currency: currency,
		IDs:      tokenIDs,
	}

	// Use MarketsByIds to get the actual market data from cache
	response, cacheStatus, err := s.MarketsByIds(params)
	if err != nil {
		log.Printf("Failed to get markets data by IDs: %v", err)
		return nil, fmt.Errorf("failed to get markets data by IDs: %w", err)
	}

	log.Printf("Returned top markets data with %d coins (cache status: %v)", len(response), cacheStatus)
	return response, nil
}

// TopMarketIds fetches top market token IDs for specified limit from cache
func (s *Service) TopMarketIds(limit int) ([]string, error) {
	log.Printf("Loading top %d market IDs from cache", limit)

	// Set default limit if not provided
	if limit <= 0 {
		limit = MARKETS_DEFAULT_CHUNK_SIZE
	}

	// Create default parameters to get per_page setting
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  MARKETS_DEFAULT_CHUNK_SIZE,
	}

	// Apply parameters normalization from config to get the actual per_page
	params = s.getParamsOverride(params)
	perPage := params.PerPage

	// Calculate how many pages we need
	pageTo := (limit + perPage - 1) / perPage // Ceiling division

	// Generate cache keys for all pages at once
	pageIdsCacheKeys := createPageIdsCacheKeys(1, pageTo)
	if len(pageIdsCacheKeys) == 0 {
		log.Printf("No cache keys generated for pages 1-%d", pageTo)
		return []string{}, nil
	}

	// Fetch all pages in one batch operation
	cachedData, missingKeys, err := s.cache.Get(pageIdsCacheKeys)
	if err != nil {
		log.Printf("Failed to check cache for pages 1-%d: %v", pageTo, err)
		return nil, fmt.Errorf("failed to check cache: %w", err)
	}

	if len(missingKeys) > 0 {
		log.Printf("Missing %d pages in cache out of %d requested", len(missingKeys), len(pageIdsCacheKeys))
	}

	// Collect token IDs from all cached pages in order
	var allTokenIDs []string
	for page := 1; page <= pageTo; page++ {
		pageIdsCacheKey := createPageIdsCacheKey(page)
		if tokenIDsBytes, exists := cachedData[pageIdsCacheKey]; exists {
			var pageTokenIDs []string
			if err := json.Unmarshal(tokenIDsBytes, &pageTokenIDs); err == nil {
				allTokenIDs = append(allTokenIDs, pageTokenIDs...)
			} else {
				log.Printf("Failed to unmarshal token IDs for page %d: %v", page, err)
			}
		}
	}

	// Limit the results to requested number
	if len(allTokenIDs) > limit {
		allTokenIDs = allTokenIDs[:limit]
	}

	log.Printf("Returning %d token IDs from cache (requested %d, %d pages)", len(allTokenIDs), limit, pageTo)
	return allTokenIDs, nil
}

func (s *Service) SubscribeTopMarketsUpdate() chan struct{} {
	return s.subscriptionManager.Subscribe()
}

func (s *Service) Unsubscribe(ch chan struct{}) {
	s.subscriptionManager.Unsubscribe(ch)
}
