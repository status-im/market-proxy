package coingecko_market_chart

import (
	"errors"
	"strconv"
	"strings"
)

// MarketChartParams represents parameters for market chart requests
type MarketChartParams struct {
	// ID is the coin id (required) - can be obtained from /coins/list
	ID string `json:"id"`

	// Currency to compare against (e.g., "usd", "eur", "btc")
	Currency string `json:"vs_currency"`

	// Days specifies the data up to number of days ago (1/7/14/30/90/180/365/max)
	Days string `json:"days"`

	// Interval specifies data interval (only for Enterprise plan)
	// Valid values: "5m" (5-minutely), "hourly", "daily"
	// Leave empty for automatic granularity:
	// 1 day = 5-minutely data
	// 2-90 days = hourly data
	// above 90 days = daily data
	Interval string `json:"interval,omitempty"`

	// DataFilter specifies which data fields to include in response
	// Comma-separated list (e.g., "prices,market_caps")
	// Available fields: "prices", "market_caps", "total_volumes"
	// If empty, all fields are included
	DataFilter string `json:"data_filter,omitempty"`
}

// ParseDataFilters parses comma-separated data_filter string into a slice of strings
// Returns an empty slice if dataFilter is empty or contains only whitespace
func ParseDataFilters(dataFilter string) []string {
	if strings.TrimSpace(dataFilter) == "" {
		return []string{}
	}

	filters := strings.Split(dataFilter, ",")
	var result []string

	for _, filter := range filters {
		filter = strings.TrimSpace(filter)
		if filter != "" {
			result = append(result, filter)
		}
	}

	return result
}

// Validate validates the MarketChartParams
func (p *MarketChartParams) Validate() error {
	// ID is required and cannot be empty or whitespace
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("coin ID is required")
	}

	// Validate days parameter if provided
	if p.Days != "" {
		if p.Days == "max" {
			// "max" is valid
		} else {
			// Try to parse as integer
			if days, err := strconv.Atoi(p.Days); err != nil {
				return errors.New("invalid days parameter, must be a number from 1 to 365 or 'max'")
			} else if days < 1 || days > 365 {
				return errors.New("invalid days parameter, must be a number from 1 to 365 or 'max'")
			}
		}
	}

	// Validate interval parameter if provided
	if p.Interval != "" {
		validIntervals := []string{"5m", "hourly", "daily"}
		isValid := false
		for _, validInterval := range validIntervals {
			if p.Interval == validInterval {
				isValid = true
				break
			}
		}
		if !isValid {
			return errors.New("invalid interval parameter, must be one of: 5m, hourly, daily")
		}
	}

	// Validate data_filter parameter if provided
	if p.DataFilter != "" {
		validFields := []string{"prices", "market_caps", "total_volumes"}
		filters := ParseDataFilters(p.DataFilter)

		for _, filter := range filters {
			isValid := false
			for _, validField := range validFields {
				if filter == validField {
					isValid = true
					break
				}
			}
			if !isValid {
				return errors.New("invalid data_filter parameter, must contain only: prices, market_caps, total_volumes")
			}
		}
	}

	return nil
}

// MarketChartResponseData represents the market chart response data structure
// analogous to SimplePriceResponse for consistent API response handling
type MarketChartResponseData map[string]interface{}

// MarketChartData represents a single data point [timestamp, value]
type MarketChartData [2]float64

// MarketChartResponse represents the market chart API response structure
type MarketChartResponse struct {
	// Prices contains historical price data as [timestamp, price] pairs
	Prices []MarketChartData `json:"prices"`

	// MarketCaps contains historical market cap data as [timestamp, market_cap] pairs
	MarketCaps []MarketChartData `json:"market_caps"`

	// TotalVolumes contains historical volume data as [timestamp, total_volume] pairs
	TotalVolumes []MarketChartData `json:"total_volumes"`
}

// MarketChartAPIResponse represents a full API response with possible error handling
type MarketChartAPIResponse struct {
	Data  *MarketChartResponse `json:"data,omitempty"`
	Error string               `json:"error,omitempty"`
}
