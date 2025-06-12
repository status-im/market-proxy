package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"
	"time"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// TopPricesUpdater handles periodic price updates for top tokens
type TopPricesUpdater struct {
	config         *config.Config
	priceFetcher   cg.PriceFetcher
	priceScheduler *scheduler.Scheduler
	metricsWriter  *metrics.MetricsWriter
	// Cache for top tokens prices
	topPricesCache struct {
		sync.RWMutex
		data map[string]PriceQuotes // currency -> tokenID -> Quote
	}
	// List of top token IDs to fetch prices for
	topTokenIDs struct {
		sync.RWMutex
		ids []string
	}
}

// NewTopPricesUpdater creates a new top prices updater
func NewTopPricesUpdater(cfg *config.Config, priceFetcher cg.PriceFetcher) *TopPricesUpdater {
	updater := &TopPricesUpdater{
		config:        cfg,
		priceFetcher:  priceFetcher,
		metricsWriter: metrics.NewMetricsWriter(metrics.ServiceLBPrices),
	}

	// Initialize prices cache
	updater.topPricesCache.data = make(map[string]PriceQuotes)

	return updater
}

// GetTopPricesQuotes returns cached prices quotes for top tokens in specified currency
func (u *TopPricesUpdater) GetTopPricesQuotes(currency string) map[string]Quote {
	u.topPricesCache.RLock()
	defer u.topPricesCache.RUnlock()

	if currencyQuotes, exists := u.topPricesCache.data[currency]; exists {
		// Return a copy to avoid race conditions
		result := make(map[string]Quote)
		for tokenID, quote := range currencyQuotes {
			result[tokenID] = quote
		}
		return result
	}

	return make(map[string]Quote)
}

// SetTopTokenIDs sets the list of top token IDs to fetch prices for
func (u *TopPricesUpdater) SetTopTokenIDs(tokenIDs []string) {
	u.topTokenIDs.Lock()
	defer u.topTokenIDs.Unlock()
	u.topTokenIDs.ids = make([]string, len(tokenIDs))
	copy(u.topTokenIDs.ids, tokenIDs)
}

// getTopTokenIDs returns a copy of the current top token IDs list
func (u *TopPricesUpdater) getTopTokenIDs() []string {
	u.topTokenIDs.RLock()
	defer u.topTokenIDs.RUnlock()
	if len(u.topTokenIDs.ids) == 0 {
		return nil
	}
	result := make([]string, len(u.topTokenIDs.ids))
	copy(result, u.topTokenIDs.ids)
	return result
}

// Start starts the price updater with periodic updates
func (u *TopPricesUpdater) Start(ctx context.Context) error {
	// Start price scheduler if configured
	pricesUpdateInterval := u.config.CoingeckoLeaderboard.PricesUpdateInterval
	if pricesUpdateInterval > 0 {
		u.priceScheduler = scheduler.New(
			pricesUpdateInterval,
			func(ctx context.Context) {
				if err := u.fetchAndUpdateTopPrices(ctx); err != nil {
					log.Printf("Error updating top prices: %v", err)
				}
			},
		)

		u.priceScheduler.Start(ctx, true)
		log.Printf("Started price scheduler with update interval: %v", pricesUpdateInterval)
	}

	return nil
}

// Stop stops the price updater
func (u *TopPricesUpdater) Stop() {
	if u.priceScheduler != nil {
		u.priceScheduler.Stop()
	}
}

// fetchAndUpdateTopPrices fetches prices for top tokens and updates cache
func (u *TopPricesUpdater) fetchAndUpdateTopPrices(ctx context.Context) error {
	// Default currencies to fetch
	currencies := []string{"usd"}

	// Record start time for metrics
	startTime := time.Now()

	var newPricesData map[string]PriceQuotes

	// Use price fetcher if available
	if u.priceFetcher != nil {
		log.Printf("Using price fetcher for top tokens prices")

		// Fetch prices for each currency using the price fetcher
		newPricesData = make(map[string]PriceQuotes)

		// Get the current list of top token IDs
		topTokenIDs := u.getTopTokenIDs()
		if len(topTokenIDs) == 0 {
			log.Printf("No top token IDs configured, skipping price update")
			newPricesData = make(map[string]PriceQuotes)
		} else {
			// Limit the number of tokens according to config
			if u.config.CoingeckoLeaderboard.TopTokensLimit > 0 && len(topTokenIDs) > u.config.CoingeckoLeaderboard.TopTokensLimit {
				topTokenIDs = topTokenIDs[:u.config.CoingeckoLeaderboard.TopTokensLimit]
				log.Printf("Limited top tokens to %d tokens as per config", u.config.CoingeckoLeaderboard.TopTokensLimit)
			}
			// Make single request with all currencies
			params := cg.PriceParams{
				IDs:                  topTokenIDs,
				Currencies:           currencies,
				IncludeMarketCap:     true,
				Include24hrVol:       true,
				Include24hrChange:    true,
				IncludeLastUpdatedAt: true,
			}

			priceResponse, err := u.priceFetcher.SimplePrices(params)
			if err != nil {
				log.Printf("Error fetching prices from price fetcher: %v", err)
			} else if len(priceResponse) > 0 {
				// Parse response for each currency
				for _, currency := range currencies {
					currencyQuotes := ConvertPriceResponseToPriceQuotes(priceResponse, currency)
					if len(currencyQuotes) > 0 {
						newPricesData[currency] = currencyQuotes
					}
				}
			}
		}
	} else {
		log.Printf("No price fetcher available, skipping price update")
		newPricesData = make(map[string]PriceQuotes)
	}

	// Record metrics regardless of success or failure
	u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))

	// Update cache
	u.topPricesCache.Lock()
	u.topPricesCache.data = newPricesData
	u.topPricesCache.Unlock()

	// Record cache size metric
	totalTokens := 0
	for _, quotes := range newPricesData {
		totalTokens += len(quotes)
	}
	u.metricsWriter.RecordCacheSize(totalTokens)

	log.Printf("Updated prices cache with %d currency-token pairs", totalTokens)
	return nil
}
