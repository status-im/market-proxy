package coingecko_prices

// PriceParams represents parameters for price requests
type PriceParams struct {
	// IDs list of token/coin IDs to fetch prices for
	IDs []string `json:"ids"`

	// Currencies list of target currencies (e.g., "usd", "eur")
	Currencies []string `json:"currencies"`
}

// PriceData represents price information for a single token
type PriceData struct {
	// Prices map of currency -> price value
	Prices map[string]float64 `json:"prices"`

	// LastUpdated timestamp when price was last updated
	LastUpdated int64 `json:"last_updated"`
}

// PriceResponse represents the complete response from price service
type PriceResponse struct {
	// Data map of token ID -> price data
	Data map[string]PriceData `json:"data"`

	// RequestedIDs original list of requested token IDs
	RequestedIDs []string `json:"requested_ids"`

	// FoundIDs list of token IDs for which prices were found
	FoundIDs []string `json:"found_ids"`

	// MissingIDs list of token IDs for which prices were not found
	MissingIDs []string `json:"missing_ids"`
}
