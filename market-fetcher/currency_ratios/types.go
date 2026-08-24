package currency_ratios

import (
	"time"

	"github.com/status-im/market-proxy/config"
)

// BaseCurrency is the currency all ratios are relative to; leaderboard
// Passthrough data is cached in it.
const BaseCurrency = config.CurrencyRatiosBaseCurrency

// Ratio is an exchange rate between the base currency (usd) and one target
// currency, in two flavours: spot (now) and as it stood 24 hours ago.
//
// Both are needed because a percent change carried over from the base currency
// has to be re-expressed in the target currency honestly:
//
//	pct_X = ((1 + pct_usd/100) * Now / H24 - 1) * 100
type Ratio struct {
	// Now is the spot ratio: price_X / price_usd for the reference coin
	Now float64 `json:"now"`
	// H24 is the same ratio as it was 24 hours ago
	H24 float64 `json:"h24"`
}

// IdentityRatio is the ratio of the base currency to itself
var IdentityRatio = Ratio{Now: 1, H24: 1}

// Snapshot is the latest successfully computed set of ratios
type Snapshot struct {
	// Ratios maps a currency (lowercase) to its ratio against the base currency
	Ratios map[string]Ratio
	// ReferenceCoin is the coin id the ratios were derived from
	ReferenceCoin string
	// UpdatedAt is when the snapshot was computed
	UpdatedAt time.Time
}

// Ratio returns the ratio for a currency and whether it is present
func (s *Snapshot) Ratio(currency string) (Ratio, bool) {
	if s == nil {
		return Ratio{}, false
	}
	ratio, ok := s.Ratios[currency]
	return ratio, ok
}

// IProvider exposes the ratios needed to compute realtime currency Estimates
type IProvider interface {
	// GetSnapshot returns the latest successful snapshot, or nil if none was
	// computed yet. The returned snapshot must not be mutated.
	GetSnapshot() *Snapshot

	// IsCurrencySupported reports whether the currency is part of the configured
	// currency list. It answers from configuration alone, so it is meaningful
	// before the first snapshot exists.
	IsCurrencySupported(currency string) bool

	// GetSpotRatioAgo returns the spot ratio the currency had approximately `ago`
	// before now, taken from the retained snapshot history. ok is false when the
	// history does not reach back that far yet (e.g. shortly after a restart) or
	// when the currency is absent from the matching snapshot.
	GetSpotRatioAgo(currency string, ago time.Duration) (float64, bool)
}
