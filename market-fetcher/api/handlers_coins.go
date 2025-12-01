package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/coingecko_market_chart"
)

// handleCoinsList responds with the list of tokens filtered by supported platforms
// This endpoint mimics the CoinGecko API endpoint /api/v3/coins/list with platform information
func (s *Server) handleCoinsList(w http.ResponseWriter, r *http.Request) {
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
	params := interfaces.MarketsParams{}

	currency := getParamLowercase(r, "vs_currency")
	if currency != "" {
		params.Currency = currency
	}

	order := getParamLowercase(r, "order")
	if order != "" {
		params.Order = order
	}

	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if page, err := strconv.Atoi(pageParam); err == nil && page > 0 {
			params.Page = page
		}
	}

	if perPageParam := r.URL.Query().Get("per_page"); perPageParam != "" {
		if perPage, err := strconv.Atoi(perPageParam); err == nil && perPage > 0 {
			params.PerPage = perPage
		}
	}

	if idsParam := getParamLowercase(r, "ids"); idsParam != "" {
		params.IDs = splitParamLowercase(idsParam)
	}

	if categoryParam := getParamLowercase(r, "category"); categoryParam != "" {
		params.Category = categoryParam
	}

	if sparklineParam := r.URL.Query().Get("sparkline"); sparklineParam != "" {
		if sparkline, err := strconv.ParseBool(sparklineParam); err == nil {
			params.SparklineEnabled = sparkline
		}
	}

	if priceChangeParam := getParamLowercase(r, "price_change_percentage"); priceChangeParam != "" {
		params.PriceChangePercentage = splitParamLowercase(priceChangeParam)
	}

	data, cacheStatus, err := s.marketsService.Markets(params)
	if err != nil {
		http.Error(w, "Failed to fetch markets data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.setCacheStatusHeader(w, cacheStatus.String())
	s.sendJSONResponse(w, data)
}

// handleSimplePrice implements CoinGecko-compatible /api/v3/simple/price endpoint
func (s *Server) handleSimplePrice(w http.ResponseWriter, r *http.Request) {
	params := interfaces.PriceParams{}

	idsParam := getParamLowercase(r, "ids")
	if idsParam == "" {
		http.Error(w, "Parameter 'ids' is required", http.StatusBadRequest)
		return
	}
	params.IDs = splitParamLowercase(idsParam)

	currenciesParam := getParamLowercase(r, "vs_currencies")
	if currenciesParam == "" {
		http.Error(w, "Parameter 'vs_currencies' is required", http.StatusBadRequest)
		return
	}
	params.Currencies = splitParamLowercase(currenciesParam)

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

	response, cacheStatus, err := s.pricesService.SimplePrices(r.Context(), params)
	if err != nil {
		http.Error(w, "Failed to fetch prices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.setCacheStatusHeader(w, cacheStatus.String())
	s.sendJSONResponse(w, response)
}

// handleMarketChart implements CoinGecko-compatible /api/v3/coins/{id}/market_chart endpoint
func (s *Server) handleMarketChart(w http.ResponseWriter, r *http.Request) {
	// Path format: /api/v1/coins/{id}/market_chart
	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathSegments) < 4 || pathSegments[3] == "" {
		http.Error(w, "Missing coin ID in path", http.StatusBadRequest)
		return
	}

	coinID := strings.ToLower(pathSegments[3])

	currency := getParamLowercase(r, "vs_currency")
	days := getParamLowercase(r, "days")
	interval := getParamLowercase(r, "interval")
	dataFilter := getParamLowercase(r, "data_filter")

	params := coingecko_market_chart.MarketChartParams{
		ID:         coinID,
		Currency:   currency,
		Days:       days,
		Interval:   interval,
		DataFilter: dataFilter,
	}

	data, err := s.marketChartService.MarketChart(params)
	if err != nil {
		if strings.Contains(err.Error(), "invalid parameters") {
			http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
		} else {
			http.Error(w, fmt.Sprintf("Error fetching market chart: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	s.sendJSONResponse(w, data)
}

// handleCoinsID implements CoinGecko-compatible /api/v3/coins/{id} endpoint
func (s *Server) handleCoinsID(w http.ResponseWriter, r *http.Request) {
	// Path format: /api/v1/coins/{id}
	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathSegments) < 4 || pathSegments[3] == "" {
		http.Error(w, "Missing coin ID in path", http.StatusBadRequest)
		return
	}

	coinID := strings.ToLower(pathSegments[3])

	data, cacheStatus, err := s.coinsService.GetCoin(coinID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("Coin not found: %s", coinID), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Error fetching coin data: %v", err), http.StatusInternalServerError)
		}
		return
	}

	s.setCacheStatusHeader(w, cacheStatus.String())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleCoinsRoutes routes different /api/v1/coins/* endpoints to appropriate handlers
func (s *Server) handleCoinsRoutes(w http.ResponseWriter, r *http.Request) {
	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

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
			// Otherwise, treat it as a coin ID request: /api/v1/coins/{id}
			s.handleCoinsID(w, r)
			return
		}
	}

	http.Error(w, "Endpoint not found", http.StatusNotFound)
}
