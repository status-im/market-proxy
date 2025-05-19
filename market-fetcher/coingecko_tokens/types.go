package coingecko_tokens

// Token represents a cryptocurrency token with its id, symbol, name and supported platforms
type Token struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Platforms map[string]string `json:"platforms"`
}
