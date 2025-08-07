package coingecko_leaderboard

import (
	"context"
	"log"
	"sync"

	cg "github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

// TopPricesUpdater handles subscription-based price updates for top tokens
type TopPricesUpdater struct {
	config             *config.CoingeckoLeaderboardFetcher
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

func NewTopPricesUpdater(cfg *config.CoingeckoLeaderboardFetcher, priceFetcher cg.CoingeckoPricesService) *TopPricesUpdater {
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
		u.updateSubscription = u.priceFetcher.SubscribeTopPricesUpdate()
		subscriptionCtx, cancel := context.WithCancel(ctx)
		u.cancelFunc = cancel

		go u.handlePriceUpdates(subscriptionCtx)

		log.Printf("Started top prices updater with subscription to price updates")

		// Perform initial fetch to populate cache
		if err := u.fetchAndUpdateTopPrices(ctx); err != nil {
			log.Printf("Error during initial price data fetch: %v", err)
		}
	} else {
		log.Printf("No price fetcher available, price updater will not be active")
	}

	return nil
}

// Stop stops the price updater
func (u *TopPricesUpdater) Stop() {
	if u.cancelFunc != nil {
		u.cancelFunc()
		u.cancelFunc = nil
	}

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
			if err := u.fetchAndUpdateTopPrices(ctx); err != nil {
				log.Printf("Error updating price data on subscription signal: %v", err)
			}
		}
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

	var newPricesData map[string]PriceQuotes

	if u.priceFetcher != nil {
		newPricesData = make(map[string]PriceQuotes)
		limit := u.config.TopPricesLimit

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

	u.topPricesCache.Lock()
	u.topPricesCache.data = newPricesData
	u.topPricesCache.Unlock()

	totalTokens := 0
	for _, quotes := range newPricesData {
		totalTokens += len(quotes)
	}
	u.metricsWriter.RecordCacheSize(totalTokens)

	log.Printf("Updated prices cache with %d currency-token pairs", totalTokens)
	return nil
}
