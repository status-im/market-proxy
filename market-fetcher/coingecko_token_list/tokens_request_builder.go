package coingecko_token_list

import (
	"fmt"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	TOKEN_LISTS_API_PATH = "/api/v3/token_lists/%s/all.json"
)

// TokensRequestBuilder implements the Builder pattern for CoinGecko token lists API requests
type TokensRequestBuilder struct {
	*cg.CoingeckoRequestBuilder
}

func NewTokensRequestBuilder(baseURL, platform string) *TokensRequestBuilder {
	apiPath := fmt.Sprintf(TOKEN_LISTS_API_PATH, platform)

	rb := &TokensRequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, apiPath),
	}

	return rb
}
