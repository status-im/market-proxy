package interfaces

// ICurrencySourceReporter tells the caller whether a currency it asked for is
// served as provider data or computed by the proxy.
//
// `convert_currency=X` asks for values in X, not for an Estimate: which source
// is better is the proxy's decision, made with knowledge the client does not
// have (which currencies it fetches, what its caches are normalized to). The
// client must never have to mirror that configuration to phrase a request, so
// the answer travels back in the response instead.
type ICurrencySourceReporter interface {
	// EstimatesCurrency reports whether values in this currency would be
	// computed by the proxy rather than passed through from the provider
	EstimatesCurrency(currency string) bool
}
