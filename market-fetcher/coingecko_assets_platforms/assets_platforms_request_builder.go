package coingecko_assets_platforms

import (
	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	ASSETS_PLATFORMS_API_PATH = "/api/v3/asset_platforms"
)

type AssetsPlatformsRequestBuilder struct {
	*cg.CoingeckoRequestBuilder
}

func NewAssetsPlatformsRequestBuilder(baseURL string) *AssetsPlatformsRequestBuilder {
	return &AssetsPlatformsRequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, ASSETS_PLATFORMS_API_PATH),
	}
}

func (rb *AssetsPlatformsRequestBuilder) WithFilter(filter string) *AssetsPlatformsRequestBuilder {
	if filter != "" {
		rb.With("filter", filter)
	}
	return rb
}
