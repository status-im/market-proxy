package coingecko_token_list

// TokenListInfo represents basic information about a token in the token list
type TokenListInfo struct {
	ChainID  int    `json:"chainId"`
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	LogoURI  string `json:"logoURI,omitempty"`
}

// TokenListVersion represents the version information of a token list
type TokenListVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// TokenList represents the complete token list response from CoinGecko
type TokenList struct {
	Name      string           `json:"name"`
	LogoURI   string           `json:"logoURI,omitempty"`
	Keywords  []string         `json:"keywords,omitempty"`
	Version   TokenListVersion `json:"version"`
	Tokens    []TokenListInfo  `json:"tokens"`
	Timestamp string           `json:"timestamp,omitempty"`
}

// TokenListCache represents cached token list data for a specific platform
type TokenListCache struct {
	Platform  string    `json:"platform"`
	TokenList TokenList `json:"token_list"`
	UpdatedAt int64     `json:"updated_at"`
}

// TokenListResponse represents the response from GetTokenList method
type TokenListResponse struct {
	TokenList *TokenList `json:"token_list,omitempty"`
	Error     error      `json:"-"`
}
