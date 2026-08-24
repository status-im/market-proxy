package coingecko_exchange_rates

import (
	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	// EXCHANGE_RATES_API_PATH is the CoinGecko exchange rates endpoint
	EXCHANGE_RATES_API_PATH = "/api/v3/exchange_rates"
)

// ExchangeRatesRequestBuilder builds requests to the CoinGecko exchange rates endpoint
type ExchangeRatesRequestBuilder struct {
	*cg.CoingeckoRequestBuilder
}

// NewExchangeRatesRequestBuilder creates a request builder for the exchange rates endpoint
func NewExchangeRatesRequestBuilder(baseURL string) *ExchangeRatesRequestBuilder {
	return &ExchangeRatesRequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, EXCHANGE_RATES_API_PATH),
	}
}
