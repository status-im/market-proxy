package coingecko_leaderboard

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/status-im/market-proxy/currency_ratios"
)

func benchResponse(n int) *APIResponse {
	resp := &APIResponse{Data: make([]CoinData, 0, n)}
	for i := 0; i < n; i++ {
		resp.Data = append(resp.Data, CoinData{
			ID:                       fmt.Sprintf("coin-%d", i),
			Symbol:                   fmt.Sprintf("sym%d", i),
			Name:                     fmt.Sprintf("Coin %d", i),
			Image:                    "https://example.com/img.png",
			CurrentPrice:             12345.678,
			MarketCap:                9.8e10,
			TotalVolume:              3.2e9,
			PriceChangePercentage24h: 1.234,
		})
	}
	return resp
}

func BenchmarkConvertAPIResponse5000(b *testing.B) {
	resp := benchResponse(5000)
	ratio := currency_ratios.Ratio{Now: 0.857, H24: 0.855}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := ConvertAPIResponse(resp, ratio)
		if len(out.Data) != 5000 {
			b.Fatal("bad len")
		}
	}
}

func BenchmarkConvertAndMarshal5000(b *testing.B) {
	resp := benchResponse(5000)
	ratio := currency_ratios.Ratio{Now: 0.857, H24: 0.855}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := ConvertAPIResponse(resp, ratio)
		if _, err := json.Marshal(out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalOnly5000(b *testing.B) {
	resp := benchResponse(5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(resp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertQuotes500(b *testing.B) {
	quotes := make(map[string]Quote, 500)
	for i := 0; i < 500; i++ {
		quotes[fmt.Sprintf("coin-%d", i)] = Quote{Price: 123.4, Volume24h: 1e8, MarketCap: 1e10, PercentChange24h: 2.5}
	}
	ratio := currency_ratios.Ratio{Now: 0.857, H24: 0.855}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := ConvertQuotes(quotes, ratio)
		if len(out) != 500 {
			b.Fatal("bad len")
		}
	}
}
