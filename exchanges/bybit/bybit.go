package bybit

import (
	"context"
	"net/http"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bybit is the overarching type across this package
type Bybit struct {
	exchange.Base
}

const (
	bybitAPIURL     = "https://api.bybit.com"
	bybitAPIVersion = "v1"

	// Public endpoints
	// TODO:
	bybitSpotGetSymbols   = "/spot/v1/symbols"
	bybitOrderBook        = "/spot/quote/v1/depth"
	bybitMergedOrderBook  = "/spot/quote/v1/depth/merged"
	bybitRecentTrades     = "/spot/quote/v1/trades"
	bybitCandlestickChart = "/spot/quote/v1/kline"
	bybit24HrsChange      = "/spot/quote/v1/ticker/24hr"
	bybitLastTradedPrice  = "/spot/quote/v1/ticker/price"
	bybitBestBidAskPrice  = "/spot/quote/v1/ticker/book_ticker"

	// Authenticated endpoints
	// TODO:
	bybitAuthenticatedSpotOrder = "/spot/v1/order" // create, query, cancel
	bybitBatchCancelSpotOrder   = "/spot/order/batch-cancel"
	bybitOpenOrder              = "/spot/v1/open-orders"
	bybitPastOrder              = "/spot/v1/history-orders"
	bybitTradeHistory           = "/spot/v1/myTrades"
	bybitWalletBalance          = "/spot/v1/account"
	bybitServerTime             = "/spot/v1/time"
)

// GetAllPairs gets all pairs on the exchange
func (by *Bybit) GetAllPairs() ([]PairData, error) {
	resp := struct {
		Data []PairData `json:"result"`
	}{}
	path := bybitSpotGetSymbols
	return resp.Data, by.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// SendHTTPRequest sends an unauthenticated request
func (by *Bybit) SendHTTPRequest(ePath exchange.URL, path string, result interface{}) error {
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	return by.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       by.Verbose,
		HTTPDebugging: by.HTTPDebugging,
		HTTPRecording: by.HTTPRecording,
	})
}
