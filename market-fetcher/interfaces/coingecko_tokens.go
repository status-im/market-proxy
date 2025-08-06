package interfaces

//go:generate mockgen -destination=mocks/coingecko_tokens.go . CoingeckoTokensService

// CoingeckoTokensService defines the interface for CoinGecko tokens service
type CoingeckoTokensService interface {
	// GetTokens returns cached tokens
	GetTokens() []Token

	// GetTokenIds returns cached token IDs
	GetTokenIds() []string

	// SubscribeOnTokensUpdate subscribes to tokens update notifications
	SubscribeOnTokensUpdate() chan struct{}

	// Unsubscribe unsubscribes from tokens update notifications
	Unsubscribe(ch chan struct{})
}

type Token struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Platforms map[string]string `json:"platforms"`
}
