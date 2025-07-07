package coingecko_market_chart

import (
	"fmt"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	MARKET_CHART_API_PATH_TEMPLATE = "/api/v3/coins/%s/market_chart"
)

type MarketChartRequestBuilder struct {
	*cg.CoingeckoRequestBuilder
	coinID string
}

func NewMarketChartRequestBuilder(baseURL, coinID string) *MarketChartRequestBuilder {
	apiPath := fmt.Sprintf(MARKET_CHART_API_PATH_TEMPLATE, coinID)

	rb := &MarketChartRequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, apiPath),
		coinID:                  coinID,
	}

	rb.WithCurrency("usd")
	rb.WithDays("90")

	return rb
}

func (rb *MarketChartRequestBuilder) WithDays(days string) *MarketChartRequestBuilder {
	if days != "" {
		rb.With("days", days)
	}
	return rb
}

func (rb *MarketChartRequestBuilder) WithInterval(interval string) *MarketChartRequestBuilder {
	if interval != "" {
		rb.With("interval", interval)
	}
	return rb
}
