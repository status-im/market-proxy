package coingecko_exchange_rates

import "encoding/json"

// ExchangeRatesResponse is the CoinGecko /api/v3/exchange_rates body kept
// verbatim. It stays raw JSON so the served values are byte-for-byte the ones
// CoinGecko returned (Passthrough) - no float round-tripping.
type ExchangeRatesResponse = json.RawMessage
