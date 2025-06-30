package coingecko_markets

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

// APIResponse represents the full response structure with all CoinGecko data
type APIResponse struct {
	Data []CoinGeckoData `json:"data"`
}
