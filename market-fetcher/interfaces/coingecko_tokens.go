package interfaces

import "github.com/status-im/market-proxy/events"

//go:generate mockgen -destination=mocks/coingecko_tokens.go . ITokensService

// ITokensService defines the interface for CoinGecko tokens service
type ITokensService interface {
	// GetTokens returns cached tokens
	GetTokens() []Token

	// GetTokenIds returns cached token IDs
	GetTokenIds() []string

	// SubscribeOnTokensUpdate subscribes to tokens update notifications
	SubscribeOnTokensUpdate() events.ISubscription
}

type Token struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Platforms map[string]string `json:"platforms"`
}
