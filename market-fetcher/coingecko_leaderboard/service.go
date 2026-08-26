package coingecko_leaderboard

import (
	"context"
	"log"
	"strings"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
)

// Service keeps data for size-optimized list of tokens and prices:
// api/v1/leaderboard/prices
// api/v1/leaderboard/markets
type Service struct {
	config            *config.Config
	ratiosProvider    interfaces.ICurrencyRatiosProvider
	onUpdate          func()
	topMarketsUpdater *TopMarketsUpdater
	topPricesUpdater  *TopPricesUpdater
}

func NewService(
	cfg *config.Config,
	priceFetcher interfaces.IPricesService,
	marketsFetcher interfaces.IMarketsService,
	ratiosProvider interfaces.ICurrencyRatiosProvider,
) *Service {
	topMarketsUpdater := NewTopMarketsUpdater(&cfg.CoingeckoLeaderboard, marketsFetcher)
	topPricesUpdater := NewTopPricesUpdater(&cfg.CoingeckoLeaderboard, priceFetcher)

	service := &Service{
		config:            cfg,
		ratiosProvider:    ratiosProvider,
		topMarketsUpdater: topMarketsUpdater,
		topPricesUpdater:  topPricesUpdater,
	}

	return service
}

// cacheCurrency is the currency the leaderboard caches are held in. Both
// updaters fetch exactly this one currency, so it is the only one that can be
// served as provider data.
func (s *Service) cacheCurrency() string {
	if s.config != nil && s.config.CoingeckoLeaderboard.Currency != "" {
		return strings.ToLower(s.config.CoingeckoLeaderboard.Currency)
	}

	return currency_ratios.BaseCurrency
}

// EstimatesCurrency reports whether values in this currency would be computed
// rather than passed through
func (s *Service) EstimatesCurrency(currency string) bool {
	return !strings.EqualFold(s.cacheCurrency(), currency)
}

// ratio looks up the Ratio converting the cached currency into the target one.
//
// Ratios are expressed against the base currency, so a cache held in some other
// currency needs the quotient of the two. ok is false when no snapshot exists
// yet or either currency is missing from it; callers then behave as if the
// cache were empty.
func (s *Service) ratio(currency string) (currency_ratios.Ratio, bool) {
	if s.ratiosProvider == nil {
		return currency_ratios.Ratio{}, false
	}

	snapshot := s.ratiosProvider.GetSnapshot()

	target, ok := snapshot.Ratio(currency)
	if !ok {
		return currency_ratios.Ratio{}, false
	}

	source, ok := snapshot.Ratio(s.cacheCurrency())
	if !ok {
		return currency_ratios.Ratio{}, false
	}

	return currency_ratios.CrossRatio(source, target)
}

// SetOnUpdateCallback sets a callback function that will be called when data is updated
// TODO: remove, along with binance fetcher (issue #43)
func (s *Service) SetOnUpdateCallback(onUpdate func()) {
	s.onUpdate = onUpdate

	s.topMarketsUpdater.SetOnUpdateCallback(func() {
		if s.onUpdate != nil {
			s.onUpdate()
		}
	})
}

// GetTopPricesQuotes returns cached prices quotes for top tokens.
//
// currency selects a Passthrough currency from the cache. convertCurrency,
// when set, instead returns an Estimate computed at request time from the base
// currency rows - the cached Passthrough values are never mutated. An empty
// result means no ratio is available yet.
func (s *Service) GetTopPricesQuotes(currency string, convertCurrency string) PriceQuotes {
	if convertCurrency != "" {
		// The cache holds one currency; when that is the one being asked for,
		// the provider's own values are served instead of a computed copy.
		if !s.EstimatesCurrency(convertCurrency) {
			return s.topPricesUpdater.GetTopPricesQuotes(s.cacheCurrency())
		}

		ratio, ok := s.ratio(convertCurrency)
		if !ok {
			return PriceQuotes{}
		}

		return ConvertQuotes(s.topPricesUpdater.GetTopPricesQuotes(s.cacheCurrency()), ratio)
	}

	if currency == "" {
		currency = s.cacheCurrency()
	}

	return s.topPricesUpdater.GetTopPricesQuotes(currency)
}

// Start starts the CoinGecko service
func (s *Service) Start(ctx context.Context) error {
	if err := s.topMarketsUpdater.Start(ctx); err != nil {
		log.Printf("Error starting top markets updater: %v", err)
		return err
	}

	if err := s.topPricesUpdater.Start(ctx); err != nil {
		log.Printf("Error starting top prices updater: %v", err)
		return err
	}

	return nil
}

func (s *Service) Stop() {
	if s.topMarketsUpdater != nil {
		s.topMarketsUpdater.Stop()
	}
	if s.topPricesUpdater != nil {
		s.topPricesUpdater.Stop()
	}
}

// GetCacheData returns the cached top markets rows.
//
// When convertCurrency is set the rows are converted at request time into a
// fresh copy; the cache keeps its Passthrough values. nil means there is
// nothing to serve - either the cache is empty or no ratio is available yet.
func (s *Service) GetCacheData(convertCurrency string) *APIResponse {
	data := s.topMarketsUpdater.GetCacheData()
	if convertCurrency == "" || data == nil {
		return data
	}

	// The cached rows are already in this currency - provider data beats a
	// computed copy of it.
	if !s.EstimatesCurrency(convertCurrency) {
		return data
	}

	ratio, ok := s.ratio(convertCurrency)
	if !ok {
		return nil
	}

	return ConvertAPIResponse(data, ratio)
}

// Healthy checks if the service can fetch at least one page of data
func (s *Service) Healthy() bool {
	return s.topMarketsUpdater.Healthy()
}
