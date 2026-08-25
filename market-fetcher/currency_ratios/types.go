package currency_ratios

import (
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
)

// BaseCurrency is the currency all ratios are relative to; leaderboard
// Passthrough data is cached in it.
const BaseCurrency = config.CurrencyRatiosBaseCurrency

// The canonical definitions live in the interfaces package, next to the
// ICurrencyRatiosProvider contract that returns them. These aliases keep the
// package-local names readable inside the service.
type (
	// Ratio is an exchange rate between the base currency and one target
	// currency, spot and as of 24 hours ago
	Ratio = interfaces.CurrencyRatio

	// Snapshot is the latest successfully computed set of ratios
	Snapshot = interfaces.CurrencyRatiosSnapshot
)

// IdentityRatio is the ratio of the base currency to itself
var IdentityRatio = Ratio{Now: 1, H24: 1}
