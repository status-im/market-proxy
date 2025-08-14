package coingecko_leaderboard

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	cg "github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/metrics"
)

// TopPricesUpdater handles subscription-based price updates for top tokens
type TopPricesUpdater struct {
	config             *config.LeaderboardFetcherConfig
	priceFetcher       cg.IPricesService
	metricsWriter      *metrics.MetricsWriter
	updateSubscription events.ISubscription
	// Cache for top tokens prices
	topPricesCache struct {
		sync.RWMutex
		data map[string]PriceQuotes // currency -> tokenID -> Quote
	}
}

func NewTopPricesUpdater(cfg *config.LeaderboardFetcherConfig, priceFetcher cg.IPricesService) *TopPricesUpdater {
	updater := &TopPricesUpdater{
		config:        cfg,
		priceFetcher:  priceFetcher,
		metricsWriter: metrics.NewMetricsWriter(metrics.ServiceLBPrices),
	}

	updater.topPricesCache.data = make(map[string]PriceQuotes)

	return updater
}

// GetTopPricesQuotes returns cached prices quotes for top tokens in specified currency
func (u *TopPricesUpdater) GetTopPricesQuotes(currency string) map[string]Quote {
	u.topPricesCache.RLock()
	defer u.topPricesCache.RUnlock()

	if currencyQuotes, exists := u.topPricesCache.data[currency]; exists {
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
	if u.priceFetcher != nil {
		u.updateSubscription = u.priceFetcher.SubscribeTopPricesUpdate().
			Watch(ctx, func() {
				if err := u.fetchAndUpdateTopPrices(ctx); err != nil {
					log.Printf("Error updating price data on subscription signal: %v", err)
				}
			}, true)

		log.Printf("Started top prices updater with subscription to price updates")
	} else {
		log.Printf("No price fetcher available, price updater will not be active")
	}

	return nil
}

// Stop stops the price updater
func (u *TopPricesUpdater) Stop() {
	if u.updateSubscription != nil {
		u.updateSubscription.Cancel()
		u.updateSubscription = nil
	}
}

// fetchAndUpdateTopPrices fetches prices for top tokens and updates cache
func (u *TopPricesUpdater) fetchAndUpdateTopPrices(ctx context.Context) error {
	defer u.metricsWriter.TrackDataFetchCycle()()

	currency := u.config.Currency
	if currency == "" {
		currency = "usd"
	}
	currencies := []string{currency}
	limit := u.config.TopPricesLimit

	var newPricesData map[string]PriceQuotes
	var fetchStatus string
	var fetchDuration time.Duration

	if u.priceFetcher != nil {
		newPricesData = make(map[string]PriceQuotes)

		startTime := time.Now()
		priceResponse, _, err := u.priceFetcher.TopPrices(ctx, limit, currencies)
		fetchDuration = time.Since(startTime)

		if err != nil {
			fetchStatus = fmt.Sprintf("fetch error: %v", err)
		} else if len(priceResponse) > 0 {
			fetchStatus = "fetch successful"
			// Parse response for each currency
			for _, currency := range currencies {
				currencyQuotes := ConvertPriceResponseToPriceQuotes(priceResponse, currency)
				if len(currencyQuotes) > 0 {
					newPricesData[currency] = currencyQuotes
				}
			}
		} else {
			fetchStatus = "fetch successful but empty response"
		}
	} else {
		fetchStatus = "no price fetcher available"
		newPricesData = make(map[string]PriceQuotes)
		fetchDuration = 0
	}

	u.topPricesCache.Lock()
	u.topPricesCache.data = newPricesData
	currencyCount := len(newPricesData)
	u.topPricesCache.Unlock()

	totalTokens := 0
	for _, quotes := range newPricesData {
		totalTokens += len(quotes)
	}
	u.metricsWriter.RecordCacheSize(totalTokens)

	// Consolidated log with all diagnostic information in one line
	log.Printf("Leaderboard prices service cache update complete - cached currencies: %d, total price entries: %d (limit: %d) - %s - fetch duration: %v",
		currencyCount, totalTokens, limit, fetchStatus, fetchDuration)

	return nil
}
