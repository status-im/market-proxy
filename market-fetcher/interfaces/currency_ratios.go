package interfaces

import "time"

//go:generate mockgen -destination=mocks/currency_ratios.go . ICurrencyRatiosProvider

// ICurrencyRatiosProvider exposes the Ratios needed to compute realtime currency
// Estimates from Passthrough values.
type ICurrencyRatiosProvider interface {
	// GetSnapshot returns the latest successful snapshot, or nil if none was
	// computed yet. The returned snapshot must not be mutated.
	GetSnapshot() *CurrencyRatiosSnapshot

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

// CurrencyRatio is an exchange rate between the base currency (usd) and one
// target currency, in two flavours: spot (now) and as it stood 24 hours ago.
//
// Both are needed because a percent change carried over from the base currency
// has to be re-expressed in the target currency honestly:
//
//	pct_X = ((1 + pct_usd/100) * Now / H24 - 1) * 100
type CurrencyRatio struct {
	// Now is the spot ratio: price_X / price_usd for the reference coin
	Now float64 `json:"now"`
	// H24 is the same ratio as it was 24 hours ago
	H24 float64 `json:"h24"`
}

// CurrencyRatiosSnapshot is a set of Ratios computed from one provider response
type CurrencyRatiosSnapshot struct {
	// Ratios maps a currency (lowercase) to its ratio against the base currency
	Ratios map[string]CurrencyRatio
	// ReferenceCoin is the coin id the ratios were derived from
	ReferenceCoin string
	// UpdatedAt is when the snapshot was computed
	UpdatedAt time.Time
}

// Ratio returns the ratio for a currency and whether it is present
func (s *CurrencyRatiosSnapshot) Ratio(currency string) (CurrencyRatio, bool) {
	if s == nil {
		return CurrencyRatio{}, false
	}
	ratio, ok := s.Ratios[currency]
	return ratio, ok
}
