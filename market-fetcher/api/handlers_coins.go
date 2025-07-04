package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/coingecko_market_chart"
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

// handleMarketChart implements CoinGecko-compatible /api/v3/coins/{id}/market_chart endpoint
func (s *Server) handleMarketChart(w http.ResponseWriter, r *http.Request) {
	// Extract coin ID from URL path
	// Path format: /api/v1/coins/{id}/market_chart
	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathSegments) < 4 || pathSegments[3] == "" {
		http.Error(w, "Missing coin ID in path", http.StatusBadRequest)
		return
	}

	coinID := pathSegments[3]

	// Parse query parameters
	currency := r.URL.Query().Get("vs_currency")
	if currency == "" {
		currency = "usd"
	}

	days := r.URL.Query().Get("days")
	if days == "" {
		days = "30"
	}

	interval := r.URL.Query().Get("interval")
	dataFilter := r.URL.Query().Get("data_filter")

	// Create market chart params
	params := coingecko_market_chart.MarketChartParams{
		ID:         coinID,
		Currency:   currency,
		Days:       days,
		Interval:   interval,
		DataFilter: dataFilter,
	}

	// Fetch market chart data
	data, err := s.marketChartService.MarketChart(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching market chart: %v", err), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")

	// Write JSON response
	json.NewEncoder(w).Encode(data)
}

// handleCoinsRoutes routes different /api/v1/coins/* endpoints to appropriate handlers
func (s *Server) handleCoinsRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse the path to determine the endpoint
	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	// Handle different coins endpoints
	if len(pathSegments) >= 4 {
		switch pathSegments[3] {
		case "list":
			s.handleCoinsList(w, r)
			return
		case "markets":
			s.handleCoinsMarkets(w, r)
			return
		default:
			// Check if this is a market_chart request: /api/v1/coins/{id}/market_chart
			if len(pathSegments) >= 5 && pathSegments[4] == "market_chart" {
				s.handleMarketChart(w, r)
				return
			}
		}
	}

	// If no matching endpoint found
	http.Error(w, "Endpoint not found", http.StatusNotFound)
}
