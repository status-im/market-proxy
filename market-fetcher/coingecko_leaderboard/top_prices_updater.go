package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"
	"time"

	cg "github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

// TopPricesUpdater handles subscription-based price updates for top tokens
type TopPricesUpdater struct {
	config             *config.Config
	priceFetcher       cg.CoingeckoPricesService
	metricsWriter      *metrics.MetricsWriter
	updateSubscription chan struct{}
	cancelFunc         context.CancelFunc
	// Cache for top tokens prices
	topPricesCache struct {
		sync.RWMutex
		data map[string]PriceQuotes // currency -> tokenID -> Quote
	}
}

// NewTopPricesUpdater creates a new top prices updater
func NewTopPricesUpdater(cfg *config.Config, priceFetcher cg.CoingeckoPricesService) *TopPricesUpdater {
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

// Start starts the price updater by subscribing to price updates
func (u *TopPricesUpdater) Start(ctx context.Context) error {
	// Subscribe to price updates from the prices service if available
	if u.priceFetcher != nil {
		u.updateSubscription = u.priceFetcher.SubscribeTopPricesUpdate()

		// Create cancelable context for the subscription handler
		subscriptionCtx, cancel := context.WithCancel(ctx)
		u.cancelFunc = cancel

		// Start subscription handler in a goroutine
		go u.handlePriceUpdates(subscriptionCtx)

		log.Printf("Started top prices updater with subscription to price updates")

		// Perform initial fetch to populate cache
		if err := u.fetchAndUpdateTopPrices(ctx); err != nil {
			log.Printf("Error during initial price data fetch: %v", err)
			// Don't return error - subscription can still work for future updates
		}
	} else {
		log.Printf("No price fetcher available, price updater will not be active")
	}

	return nil
}

// Stop stops the price updater
func (u *TopPricesUpdater) Stop() {
	// Cancel the subscription handler goroutine
	if u.cancelFunc != nil {
		u.cancelFunc()
		u.cancelFunc = nil
	}

	// Unsubscribe from price updates
	if u.updateSubscription != nil && u.priceFetcher != nil {
		u.priceFetcher.Unsubscribe(u.updateSubscription)
		u.updateSubscription = nil
	}
}

// handlePriceUpdates handles subscription to price updates
func (u *TopPricesUpdater) handlePriceUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-u.updateSubscription:
			// Price data has been updated, fetch new data
			if err := u.fetchAndUpdateTopPrices(ctx); err != nil {
				log.Printf("Error updating price data on subscription signal: %v", err)
			}
		}
	}
}

// fetchAndUpdateTopPrices fetches prices for top tokens and updates cache
func (u *TopPricesUpdater) fetchAndUpdateTopPrices(ctx context.Context) error {
	// Get currency from config, use "usd" as default
	currency := u.config.CoingeckoLeaderboard.Currency
	if currency == "" {
		currency = "usd"
	}
	currencies := []string{currency}

	// Record start time for metrics
	startTime := time.Now()

	var newPricesData map[string]PriceQuotes

	// Use price fetcher if available
	if u.priceFetcher != nil {
		log.Printf("Using price fetcher for top tokens prices")

		// Fetch prices for each currency using the price fetcher
		newPricesData = make(map[string]PriceQuotes)

		// Determine the limit for top tokens
		limit := u.config.CoingeckoLeaderboard.TopPricesLimit

		// Use TopPrices method directly
		priceResponse, _, err := u.priceFetcher.TopPrices(ctx, limit, currencies)
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
