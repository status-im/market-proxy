package api

import (
	"net/http"

	"github.com/status-im/market-proxy/coingecko_assets_platforms"
)

// handleAssetsPlatforms implements CoinGecko-compatible /api/v3/asset_platforms endpoint
func (s *Server) handleAssetsPlatforms(w http.ResponseWriter, r *http.Request) {
	params := coingecko_assets_platforms.AssetsPlatformsParams{}

	if filterParam := r.URL.Query().Get("filter"); filterParam != "" {
		params.Filter = filterParam
	}

	data, err := s.assetsPlatformsService.AssetsPlatforms(params)
	if err != nil {
		http.Error(w, "Failed to fetch assets platforms: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, data)
}
