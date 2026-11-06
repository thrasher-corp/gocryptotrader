package kraken

import (
	"context"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesHistoricalFundingRates calls Historical funding rates, returning every funding rate Kraken holds for a
// perpetual, which spans about a year of hourly rates for an established market
func (e *Exchange) GetFuturesHistoricalFundingRates(ctx context.Context, pair currency.Pair) (*FuturesHistoricalFundingRatesResponse, error) {
	symbol, err := e.futuresMarketSymbol(pair)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesHistoricalFundingRatesResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/api/v3/historical-funding-rates", params), &resp)
}
