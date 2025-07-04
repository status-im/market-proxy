package coingecko_market_chart

import (
	"testing"
)

func TestEnrichMarketChartParams(t *testing.T) {
	tests := []struct {
		name           string
		inputParams    MarketChartParams
		expectedParams MarketChartParams
	}{
		{
			name: "Days 1 should be enriched to 90",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "1",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "90",
			},
		},
		{
			name: "Days 30 should be enriched to 90",
			inputParams: MarketChartParams{
				ID:       "ethereum",
				Currency: "usd",
				Days:     "30",
			},
			expectedParams: MarketChartParams{
				ID:       "ethereum",
				Currency: "usd",
				Days:     "90",
			},
		},
		{
			name: "Days 90 should stay 90",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "90",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "90",
			},
		},
		{
			name: "Days 180 should be enriched to 365",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "180",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "365",
			},
		},
		{
			name: "Days 365 should stay 365",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "365",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "365",
			},
		},
		{
			name: "Days max should stay max",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "max",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "max",
			},
		},
		{
			name: "Empty days should stay empty",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "",
			},
		},
		{
			name: "Invalid days should stay unchanged",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "invalid",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "invalid",
			},
		},
		{
			name: "Days 7 should be enriched to 90",
			inputParams: MarketChartParams{
				ID:       "cardano",
				Currency: "eur",
				Days:     "7",
			},
			expectedParams: MarketChartParams{
				ID:       "cardano",
				Currency: "eur",
				Days:     "90",
			},
		},
		{
			name: "Days 14 should be enriched to 90",
			inputParams: MarketChartParams{
				ID:       "polkadot",
				Currency: "btc",
				Days:     "14",
			},
			expectedParams: MarketChartParams{
				ID:       "polkadot",
				Currency: "btc",
				Days:     "90",
			},
		},
		{
			name: "Days 91 should be enriched to 365",
			inputParams: MarketChartParams{
				ID:       "chainlink",
				Currency: "usd",
				Days:     "91",
			},
			expectedParams: MarketChartParams{
				ID:       "chainlink",
				Currency: "usd",
				Days:     "365",
			},
		},
		{
			name: "Other fields should remain unchanged",
			inputParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "30",
				Interval: "hourly",
			},
			expectedParams: MarketChartParams{
				ID:       "bitcoin",
				Currency: "usd",
				Days:     "90",
				Interval: "hourly",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnrichMarketChartParams(tt.inputParams)

			// Check each field
			if result.ID != tt.expectedParams.ID {
				t.Errorf("Expected ID %s, got %s", tt.expectedParams.ID, result.ID)
			}
			if result.Currency != tt.expectedParams.Currency {
				t.Errorf("Expected Currency %s, got %s", tt.expectedParams.Currency, result.Currency)
			}
			if result.Days != tt.expectedParams.Days {
				t.Errorf("Expected Days %s, got %s", tt.expectedParams.Days, result.Days)
			}
			if result.Interval != tt.expectedParams.Interval {
				t.Errorf("Expected Interval %s, got %s", tt.expectedParams.Interval, result.Interval)
			}
		})
	}
}

func TestEnrichMarketChartParamsInPlace(t *testing.T) {
	// Test that in-place modification works correctly
	params := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "30",
	}

	EnrichMarketChartParamsInPlace(&params)

	if params.Days != "90" {
		t.Errorf("Expected Days to be enriched to 90, got %s", params.Days)
	}
	if params.ID != "bitcoin" {
		t.Errorf("Expected ID to remain bitcoin, got %s", params.ID)
	}
	if params.Currency != "usd" {
		t.Errorf("Expected Currency to remain usd, got %s", params.Currency)
	}
}

func TestEnrichMarketChartParams_OriginalUnchanged(t *testing.T) {
	// Test that the original params are not modified when using the non-in-place function
	original := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "30",
	}

	result := EnrichMarketChartParams(original)

	// Original should remain unchanged
	if original.Days != "30" {
		t.Errorf("Expected original Days to remain 30, got %s", original.Days)
	}

	// Result should be enriched
	if result.Days != "90" {
		t.Errorf("Expected result Days to be enriched to 90, got %s", result.Days)
	}
}
