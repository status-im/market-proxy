package coingecko_market_chart

import (
	"errors"
	"strconv"
	"strings"
)

type MarketChartParams struct {
	ID       string `json:"id"`
	Currency string `json:"vs_currency"`
	Days     string `json:"days"`
	// Interval specifies data interval (only for Enterprise plan)
	// Valid values: "5m", "hourly", "daily"
	// Leave empty for automatic granularity based on days
	Interval string `json:"interval,omitempty"`

	// DataFilter specifies which data fields to include in response
	// Available fields: "prices", "market_caps", "total_volumes"
	// If empty, all fields are included
	DataFilter string `json:"data_filter,omitempty"`
}

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

func (p *MarketChartParams) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("coin ID is required")
	}

	if p.Days != "" && p.Days != "max" {
		if days, err := strconv.Atoi(p.Days); err != nil {
			return errors.New("invalid days parameter, must be a number from 1 to 365 or 'max'")
		} else if days < 1 || days > 365 {
			return errors.New("invalid days parameter, must be a number from 1 to 365 or 'max'")
		}
	}

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

type MarketChartResponseData map[string]interface{}

type MarketChartData [2]float64

type MarketChartResponse struct {
	Prices       []MarketChartData `json:"prices"`
	MarketCaps   []MarketChartData `json:"market_caps"`
	TotalVolumes []MarketChartData `json:"total_volumes"`
}

type MarketChartAPIResponse struct {
	Data  *MarketChartResponse `json:"data,omitempty"`
	Error string               `json:"error,omitempty"`
}
