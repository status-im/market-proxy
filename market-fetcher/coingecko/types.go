package coingecko

// Original CoinGecko response structure
type CoinGeckoData struct {
	ID                           string      `json:"id"`
	Symbol                       string      `json:"symbol"`
	Name                         string      `json:"name"`
	Image                        string      `json:"image"`
	CurrentPrice                 float64     `json:"current_price"`
	MarketCap                    float64     `json:"market_cap"`
	MarketCapRank                int         `json:"market_cap_rank"`
	FullyDilutedValuation        float64     `json:"fully_diluted_valuation"`
	TotalVolume                  float64     `json:"total_volume"`
	High24h                      float64     `json:"high_24h"`
	Low24h                       float64     `json:"low_24h"`
	PriceChange24h               float64     `json:"price_change_24h"`
	PriceChangePercentage24h     float64     `json:"price_change_percentage_24h"`
	MarketCapChange24h           float64     `json:"market_cap_change_24h"`
	MarketCapChangePercentage24h float64     `json:"market_cap_change_percentage_24h"`
	CirculatingSupply            float64     `json:"circulating_supply"`
	TotalSupply                  float64     `json:"total_supply"`
	MaxSupply                    float64     `json:"max_supply"`
	ATH                          float64     `json:"ath"`
	ATHChangePercentage          float64     `json:"ath_change_percentage"`
	ATHDate                      string      `json:"ath_date"`
	ATL                          float64     `json:"atl"`
	ATLChangePercentage          float64     `json:"atl_change_percentage"`
	ATLDate                      string      `json:"atl_date"`
	ROI                          interface{} `json:"roi"`
	LastUpdated                  string      `json:"last_updated"`
}

// Quote represents price data in a specific currency (matching CoinMarketCap structure)
type Quote struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	MarketCap        float64 `json:"market_cap"`
	PercentChange24h float64 `json:"percent_change_24h"`
}

// CoinData represents a cleaned CoinGecko coin with minimal fields
type CoinData struct {
	ID                       string  `json:"id"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	Image                    string  `json:"image"`
	CurrentPrice             float64 `json:"current_price"`
	MarketCap                float64 `json:"market_cap"`
	TotalVolume              float64 `json:"total_volume"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
}

// APIResponse represents the filtered response structure
type APIResponse struct {
	Data []CoinData `json:"data"`
}

// ConvertCoinGeckoData converts full CoinGecko data to minimal format
func ConvertCoinGeckoData(data []CoinGeckoData) []CoinData {
	result := make([]CoinData, 0, len(data))

	for _, item := range data {
		coin := CoinData{
			ID:                       item.ID,
			Symbol:                   item.Symbol,
			Name:                     item.Name,
			Image:                    item.Image,
			CurrentPrice:             item.CurrentPrice,
			MarketCap:                item.MarketCap,
			TotalVolume:              item.TotalVolume,
			PriceChangePercentage24h: item.PriceChangePercentage24h,
		}

		result = append(result, coin)
	}

	return result
}
