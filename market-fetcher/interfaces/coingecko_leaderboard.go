package interfaces

//go:generate mockgen -destination=mocks/coingecko_leaderboard.go . ILeaderboardService

// ILeaderboardService defines the interface for the leaderboard service: the
// proxy's own aggregated API over top-N tokens.
type ILeaderboardService interface {
	IHealthReporter
	ICurrencySourceReporter

	// GetCacheData returns the cached top markets rows, converted to
	// convertCurrency when that is non-empty. nil means there is nothing to
	// serve: either the cache is empty or no Ratio is available yet.
	GetCacheData(convertCurrency string) *LeaderboardResponse

	// GetTopPricesQuotes returns cached price quotes for top tokens.
	// currency selects a Passthrough currency from the cache; convertCurrency,
	// when non-empty, instead returns an Estimate computed at request time from
	// the base currency rows. An empty result means no Ratio is available yet.
	GetTopPricesQuotes(currency string, convertCurrency string) LeaderboardQuotes
}

// LeaderboardQuote represents price data for one token in a single currency
// (matching the CoinMarketCap quote structure).
//
// Every field the provider may omit is a pointer, and nil means "upstream did
// not report this", not zero. Substituting zero would be actively wrong: a
// missing percent change turned into 0 does not stay 0 through a currency
// conversion, it becomes whatever the fx ratio moved by - a fabricated number
// that looks like real data. The nil fields are dropped from the JSON, so a
// client decoding into plain floats still sees zero and nothing breaks.
type LeaderboardQuote struct {
	// Price is always present: a quote without a usable price is not emitted
	Price float64 `json:"price"`

	// Volume24h is absent unless the upstream request asked for 24h volumes
	Volume24h *float64 `json:"volume_24h,omitempty"`

	// MarketCap is absent unless the upstream request asked for market caps
	MarketCap *float64 `json:"market_cap,omitempty"`

	// PercentChange24h is absent unless the upstream request asked for 24h
	// changes, and for tokens too new to have one
	PercentChange24h *float64 `json:"percent_change_24h,omitempty"`
}

// LeaderboardQuotes maps a token ID to its quote data
type LeaderboardQuotes = map[string]LeaderboardQuote

// LeaderboardCoinData is a CoinGecko market row reduced to the fields the
// leaderboard serves.
//
// The numeric fields are pointers for the reason given on LeaderboardQuote:
// CoinGecko returns null for any of them (an unranked token has no market cap,
// a freshly listed one has no 24h change), and a null must not become a zero
// that a currency conversion can turn into a plausible-looking wrong value.
type LeaderboardCoinData struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Image  string `json:"image"`

	CurrentPrice             *float64 `json:"current_price,omitempty"`
	MarketCap                *float64 `json:"market_cap,omitempty"`
	TotalVolume              *float64 `json:"total_volume,omitempty"`
	PriceChangePercentage24h *float64 `json:"price_change_percentage_24h,omitempty"`
}

// LeaderboardResponse is the response structure of /v1/leaderboard/markets
type LeaderboardResponse struct {
	Data []LeaderboardCoinData `json:"data"`
}
