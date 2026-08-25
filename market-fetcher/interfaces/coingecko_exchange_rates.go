package interfaces

import "encoding/json"

//go:generate mockgen -destination=mocks/coingecko_exchange_rates.go . IExchangeRatesService

// IExchangeRatesService defines the interface for the CoinGecko exchange rates
// service
type IExchangeRatesService interface {
	IHealthReporter

	// ExchangeRates returns the cached exchange rates body, or an error when
	// nothing has been fetched yet
	ExchangeRates() (ExchangeRatesResponse, error)
}

// ExchangeRatesResponse is the CoinGecko /api/v3/exchange_rates body kept
// verbatim. It stays raw JSON so the served values are byte-for-byte the ones
// CoinGecko returned (Passthrough) - no float round-tripping.
type ExchangeRatesResponse = json.RawMessage
