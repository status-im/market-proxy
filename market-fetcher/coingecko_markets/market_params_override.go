package coingecko_markets

import (
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

// getParamsOverride normalizes MarketParams according to configuration
// This function applies parameter overrides from the configuration to ensure
// consistent caching behavior regardless of user input parameters.
func (s *Service) getParamsOverride(params cg.MarketsParams) cg.MarketsParams {
	// If no normalization config is provided, return params as is
	if s.config.CoingeckoMarkets.MarketParamsNormalize == nil {
		return params
	}

	normalize := s.config.CoingeckoMarkets.MarketParamsNormalize
	normalizedParams := params // Create a copy

	// Override vs_currency if configured
	if normalize.VsCurrency != nil {
		normalizedParams.Currency = *normalize.VsCurrency
	}

	// Override order if configured
	if normalize.Order != nil {
		normalizedParams.Order = *normalize.Order
	}

	// Override per_page if configured
	if normalize.PerPage != nil {
		normalizedParams.PerPage = *normalize.PerPage
	}

	// Override sparkline if configured
	if normalize.Sparkline != nil {
		normalizedParams.SparklineEnabled = *normalize.Sparkline
	}

	// Override price_change_percentage if configured
	if normalize.PriceChangePercentage != nil {
		normalizedParams.PriceChangePercentage = strings.Split(*normalize.PriceChangePercentage, ",")
	}

	// Override category if configured
	if normalize.Category != nil {
		normalizedParams.Category = *normalize.Category
	}

	return normalizedParams
}
