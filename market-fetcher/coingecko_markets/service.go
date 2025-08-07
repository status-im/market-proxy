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
	cache                          cache.Cache
	config                         *cfg.Config
	metricsWriter                  *metrics.MetricsWriter
	subscriptionManager            *events.SubscriptionManager
	initializedSubscriptionManager *events.SubscriptionManager
	periodicUpdater                *PeriodicUpdater
	tokensService                  interfaces.CoingeckoTokensService
	tokenUpdateSubscription        events.SubscriptionInterface
	topIdsManager                  *TopIdsManager
}

func NewService(cache cache.Cache, config *cfg.Config, tokensService interfaces.CoingeckoTokensService) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceMarkets)
	apiClient := NewCoinGeckoClient(config)

	service := &Service{
		cache:                          cache,
		config:                         config,
		metricsWriter:                  metricsWriter,
		subscriptionManager:            events.NewSubscriptionManager(),
		initializedSubscriptionManager: events.NewSubscriptionManager(),
		tokensService:                  tokensService,
		topIdsManager:                  NewTopIdsManager(),
	}

	service.periodicUpdater = NewPeriodicUpdater(&config.CoingeckoMarkets, apiClient)

	service.periodicUpdater.SetOnUpdateTierPagesCallback(service.handleTierPagesUpdate)
	service.periodicUpdater.SetOnUpdateMissingExtraIdsCallback(service.handleMissingExtraIdsUpdate)
	service.periodicUpdater.SetOnInitialLoadCompletedCallback(service.handleInitialLoadCompleted)

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

	// Update top IDs with new pages data
	s.topIdsManager.UpdatePagesFromPageData(pagesData)

	s.subscriptionManager.Emit(ctx)
}

// handleMissingExtraIdsUpdate handles missing extra IDs update by caching tokens and emitting events
func (s *Service) handleMissingExtraIdsUpdate(ctx context.Context, tokensData [][]byte) {
	_, err := s.cacheTokensByID(tokensData)
	if err != nil {
		log.Printf("Failed to cache missing extra IDs: %v", err)
	} else {
		log.Printf("Successfully cached %d missing extra IDs", len(tokensData))
	}
	s.subscriptionManager.Emit(ctx)
}

// handleInitialLoadCompleted handles initial load completion by emitting initialization event
func (s *Service) handleInitialLoadCompleted(ctx context.Context) {
	log.Printf("Markets service initial load completed - emitting Initialized event")
	s.initializedSubscriptionManager.Emit(ctx)
}

// onTokenListChanged is called when token list is updated (coins/list)
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

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache dependency not provided")
	}

	if s.tokensService != nil {
		s.tokenUpdateSubscription = s.tokensService.SubscribeOnTokensUpdate().
			Watch(ctx, s.onTokenListChanged, true)
	}

	if s.periodicUpdater != nil {
		if err := s.periodicUpdater.Start(ctx); err != nil {
			return fmt.Errorf("failed to start periodic updater: %w", err)
		}
	}

	return nil
}

// Stop implements core.Interface
func (s *Service) Stop() {
	if s.tokenUpdateSubscription != nil {
		s.tokenUpdateSubscription.Cancel()
		s.tokenUpdateSubscription = nil
	}

	if s.periodicUpdater != nil {
		s.periodicUpdater.Stop()
	}

	if s.topIdsManager != nil {
		s.topIdsManager.Clear()
	}
}

// cacheTokensByID parses tokens data and caches each token by its CoinGecko ID
func (s *Service) cacheTokensByID(tokensData [][]byte) ([]interface{}, error) {
	marketData, cacheData, err := parseTokensData(tokensData)
	if err != nil {
		return nil, err
	}

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
	pageMapping, cacheData, err := parsePagesData(pagesData)
	if err != nil {
		return nil, err
	}

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
	if len(params.IDs) > 0 {
		return s.MarketsByIds(params)
	}

	if params.Page > 0 {
		return s.MarketsByPage(params.Page, params.Page, params)
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

// MarketsByPage fetches markets data for a specific page range using cache only
func (s *Service) MarketsByPage(pageFrom, pageTo int, params interfaces.MarketsParams) (interfaces.MarketsResponse, interfaces.CacheStatus, error) {
	log.Printf("Loading markets data for pages %d-%d from cache with currency=%s", pageFrom, pageTo, params.Currency)

	// Validate page range
	if pageFrom <= 0 || pageTo <= 0 || pageFrom > pageTo {
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("invalid page range: from=%d, to=%d", pageFrom, pageTo)
	}

	pageCacheKeys := make([]string, 0, pageTo-pageFrom+1)
	for page := pageFrom; page <= pageTo; page++ {
		pageCacheKeys = append(pageCacheKeys, createPageCacheKey(page))
	}

	cachedData, missingKeys, err := s.cache.Get(pageCacheKeys)
	if err != nil {
		log.Printf("Failed to check cache for pages: %v", err)
		return nil, interfaces.CacheStatusMiss, fmt.Errorf("failed to check cache for pages: %w", err)
	}

	var allMarketData []interface{}
	for page := pageFrom; page <= pageTo; page++ {
		pageCacheKey := createPageCacheKey(page)
		if pageBytes, exists := cachedData[pageCacheKey]; exists {
			var pageData []interface{}
			if err := json.Unmarshal(pageBytes, &pageData); err != nil {
				log.Printf("Failed to unmarshal page %d data: %v", page, err)
				continue
			}
			allMarketData = append(allMarketData, pageData...)
		}
	}

	// Determine cache status
	var cacheStatus interfaces.CacheStatus
	if len(missingKeys) == 0 && len(cachedData) == len(pageCacheKeys) {
		cacheStatus = interfaces.CacheStatusFull
		log.Printf("Returning cached data for all %d pages (%d tokens)", len(pageCacheKeys), len(allMarketData))
	} else if len(cachedData) > 0 {
		cacheStatus = interfaces.CacheStatusPartial
		log.Printf("Returning partial cached data for %d/%d pages (%d tokens)", len(cachedData), len(pageCacheKeys), len(allMarketData))
	} else {
		cacheStatus = interfaces.CacheStatusMiss
		log.Printf("No cached data found for pages %d-%d", pageFrom, pageTo)
	}

	return interfaces.MarketsResponse(allMarketData), cacheStatus, nil
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

	// Calculate how many pages we need to fetch based on per_page size
	// Apply parameters normalization from config to get the actual per_page
	params := interfaces.MarketsParams{
		Currency: currency,
		Order:    "market_cap_desc",
		PerPage:  MARKETS_DEFAULT_CHUNK_SIZE,
	}
	params = s.getParamsOverride(params)
	perPage := params.PerPage

	pageTo := (limit + perPage - 1) / perPage
	if pageTo == 0 {
		pageTo = 1
	}

	log.Printf("Fetching top markets using page-based approach: need %d pages (per_page=%d) for limit=%d", pageTo, perPage, limit)

	// Use MarketsByPage to get the data more efficiently from page cache
	response, cacheStatus, err := s.MarketsByPage(1, pageTo, params)
	if err != nil {
		log.Printf("Failed to get markets data by pages: %v", err)
		return nil, fmt.Errorf("failed to get markets data by pages: %w", err)
	}

	// Trim the response to the exact limit requested since we might have fetched more
	if len(response) > limit {
		response = response[:limit]
		log.Printf("Trimmed response from %d to %d tokens to match requested limit", len(response)+limit-len(response), limit)
	}

	log.Printf("Returned top markets data with %d coins (cache status: %v)", len(response), cacheStatus)
	return response, nil
}

// TopMarketIds fetches top market token IDs for specified limit from top IDs manager
func (s *Service) TopMarketIds(limit int) ([]string, error) {
	log.Printf("Loading top %d market IDs from top IDs manager", limit)

	// Set default limit if not provided
	if limit <= 0 {
		limit = MARKETS_DEFAULT_CHUNK_SIZE
	}

	// Get top IDs from the manager
	topIds := s.topIdsManager.GetTopIds(limit)

	// Get manager statistics for logging
	pageCount, totalTokens, isDirty := s.topIdsManager.GetStats()
	log.Printf("Returning %d token IDs from top IDs manager (requested %d, %d pages, %d total tokens, dirty: %v)",
		len(topIds), limit, pageCount, totalTokens, isDirty)

	return topIds, nil
}

func (s *Service) SubscribeTopMarketsUpdate() events.SubscriptionInterface {
	return s.subscriptionManager.Subscribe()
}

func (s *Service) SubscribeInitialized() events.SubscriptionInterface {
	return s.initializedSubscriptionManager.Subscribe()
}
