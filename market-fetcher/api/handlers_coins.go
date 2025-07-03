package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/status-im/market-proxy/coingecko_common"
)

// handleCoinsList responds with the list of tokens filtered by supported platforms
// This endpoint mimics the CoinGecko API endpoint /api/v3/coins/list with platform information
func (s *Server) handleCoinsList(w http.ResponseWriter, r *http.Request) {
	// Check for include_platform parameter, though we always include platforms
	includePlatform := r.URL.Query().Get("include_platform")
	if includePlatform != "" {
		include, err := strconv.ParseBool(includePlatform)
		if err != nil || !include {
			http.Error(w, "include_platform parameter must be a valid boolean value representing 'true'", http.StatusBadRequest)
			return
		}
	}

	tokens := s.tokensService.GetTokens()
	if len(tokens) == 0 {
		http.Error(w, "No token data available", http.StatusServiceUnavailable)
		return
	}

	s.sendJSONResponse(w, tokens)
}

// handleCoinsMarkets implements CoinGecko-compatible /api/v3/coins/markets endpoint
func (s *Server) handleCoinsMarkets(w http.ResponseWriter, r *http.Request) {
	params := coingecko_common.MarketsParams{}

	// Parse vs_currency (required in CoinGecko, but we'll default to USD)
	currency := r.URL.Query().Get("vs_currency")
	if currency != "" {
		params.Currency = currency
	}

	// Parse order parameter
	order := r.URL.Query().Get("order")
	if order != "" {
		params.Order = order
	}

	// Parse page parameter
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if page, err := strconv.Atoi(pageParam); err == nil && page > 0 {
			params.Page = page
		}
	}

	// Parse per_page parameter
	if perPageParam := r.URL.Query().Get("per_page"); perPageParam != "" {
		if perPage, err := strconv.Atoi(perPageParam); err == nil && perPage > 0 {
			params.PerPage = perPage
		}
	}

	// Parse ids parameter (optional)
	if idsParam := r.URL.Query().Get("ids"); idsParam != "" {
		params.IDs = strings.Split(idsParam, ",")
	}

	// Parse category parameter (optional)
	if categoryParam := r.URL.Query().Get("category"); categoryParam != "" {
		params.Category = categoryParam
	}

	// Parse sparkline parameter
	if sparklineParam := r.URL.Query().Get("sparkline"); sparklineParam != "" {
		if sparkline, err := strconv.ParseBool(sparklineParam); err == nil {
			params.SparklineEnabled = sparkline
		}
	}

	// Parse price_change_percentage parameter
	if priceChangeParam := r.URL.Query().Get("price_change_percentage"); priceChangeParam != "" {
		params.PriceChangePercentage = strings.Split(priceChangeParam, ",")
	}

	// Call markets service
	data, err := s.marketsService.Markets(params)
	if err != nil {
		http.Error(w, "Failed to fetch markets data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.Error(w, "No data available", http.StatusServiceUnavailable)
		return
	}

	s.sendJSONResponse(w, data)
}

// handleSimplePrice implements CoinGecko-compatible /api/v3/simple/price endpoint
func (s *Server) handleSimplePrice(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	params := coingecko_common.PriceParams{}

	// Parse IDs (required)
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		http.Error(w, "Parameter 'ids' is required", http.StatusBadRequest)
		return
	}
	params.IDs = strings.Split(idsParam, ",")

	// Parse currencies (vs_currencies, required)
	currenciesParam := r.URL.Query().Get("vs_currencies")
	if currenciesParam == "" {
		http.Error(w, "Parameter 'vs_currencies' is required", http.StatusBadRequest)
		return
	}
	params.Currencies = strings.Split(currenciesParam, ",")

	// Parse optional boolean parameters
	if marketCapParam := r.URL.Query().Get("include_market_cap"); marketCapParam != "" {
		if marketCap, err := strconv.ParseBool(marketCapParam); err == nil {
			params.IncludeMarketCap = marketCap
		}
	}

	if volParam := r.URL.Query().Get("include_24hr_vol"); volParam != "" {
		if vol, err := strconv.ParseBool(volParam); err == nil {
			params.Include24hrVol = vol
		}
	}

	if changeParam := r.URL.Query().Get("include_24hr_change"); changeParam != "" {
		if change, err := strconv.ParseBool(changeParam); err == nil {
			params.Include24hrChange = change
		}
	}

	if updatedParam := r.URL.Query().Get("include_last_updated_at"); updatedParam != "" {
		if updated, err := strconv.ParseBool(updatedParam); err == nil {
			params.IncludeLastUpdatedAt = updated
		}
	}

	// Call price service
	response, err := s.pricesService.SimplePrices(params)
	if err != nil {
		http.Error(w, "Failed to fetch prices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send raw JSON response from cache (CoinGecko format)
	s.sendJSONResponse(w, response)
}
