package bybit

import (
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
	bybitSpotGetSymbols = "/spot/v1/symbols"
	bybitOrderBook = "/spot/quote/v1/depth"
	bybitMergedOrderBook = "/spot/quote/v1/depth/merged"
	bybitRecentTrades = "/spot/quote/v1/trades"
	bybitCandlestickChart = "/spot/quote/v1/kline"
	bybit24HrsChange = "/spot/quote/v1/ticker/24hr"
	bybitLastTradedPrice = "/spot/quote/v1/ticker/price"
	bybitBestBidAskPrice = "/spot/quote/v1/ticker/book_ticker"

	// Authenticated endpoints
	// TODO:
	bybitAuthenticatedSpotOrder = "/spot/v1/order" // create, query, cancel
	bybitBatchCancelSpotOrder = "/spot/order/batch-cancel"
	bybitOpenOrder = "/spot/v1/open-orders"
	bybitPastOrder = "/spot/v1/history-orders"
	bybitTradeHistory = "/spot/v1/myTrades"

	bybitWalletBalance = "/spot/v1/account"
	bybitServerTime = "/spot/v1/time"
)

// GetAllPairs gets all pairs on the exchange
func (b *Bybit) GetAllPairs() ([]PairData, error) {
	resp := struct {
		Data []PairData `json:"result"`
	}{}
	path := bybitSpotGetSymbols
	return resp.Data, b.SendHTTPRequest(exchange.RestSpot, path, spotPairs, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (b *Bybit) SendHTTPRequest(ep exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpoint, err := b.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var resp json.RawMessage
	errCap := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}

	if err := b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        &resp,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f,
	}); err != nil {
		return err
	}

	if err := json.Unmarshal(resp, &errCap); err == nil {
		if errCap.Code != 200 && errCap.Message != "" {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(resp, result)
}