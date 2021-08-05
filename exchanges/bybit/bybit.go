package bybit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
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

	sideBuy  = "BUY"
	sideSell = "SELL"

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
	return resp.Data, by.SendHTTPRequest(exchange.RestSpot, bybitSpotGetSymbols, &resp)
}

// GetOrderbook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetOrderBook(symbol string, depth int64) (Orderbook, error) {
	resp := struct {
		Data struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
			Time int64      `json:"time"`
		} `json:"result"`
	}{}

	strDepth := "100" // default depth
	if depth > 0 && depth < 100 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitOrderBook, params)
	err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	processOB := func(ob [][]string) ([]OrderbookItem, error) {
		var o []OrderbookItem
		for x := range ob {
			var price, amount float64
			amount, err = strconv.ParseFloat(ob[x][1], 64)
			if err != nil {
				return nil, err
			}
			price, err = strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, err
			}
			o = append(o, OrderbookItem{
				Price:  price,
				Amount: amount,
			})
		}
		return o, nil
	}

	var s Orderbook
	s.Bids, err = processOB(resp.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(resp.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = time.Unix(0, resp.Data.Time*int64(time.Millisecond))
	return s, nil
}

// GetMergedOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetMergedOrderBook(symbol string, scale, depth int64) (Orderbook, error) {
	resp := struct {
		Data struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
			Time int64      `json:"time"`
		} `json:"result"`
	}{}

	params := url.Values{}
	if scale > 0 {
		params.Set("scale", strconv.FormatInt(scale, 10))
	}

	strDepth := "100" // default depth
	if depth > 0 && depth < 100 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitMergedOrderBook, params)
	err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	processOB := func(ob [][]string) ([]OrderbookItem, error) {
		var o []OrderbookItem
		for x := range ob {
			var price, amount float64
			amount, err = strconv.ParseFloat(ob[x][1], 64)
			if err != nil {
				return nil, err
			}
			price, err = strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, err
			}
			o = append(o, OrderbookItem{
				Price:  price,
				Amount: amount,
			})
		}
		return o, nil
	}

	var s Orderbook
	s.Bids, err = processOB(resp.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(resp.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = time.Unix(0, resp.Data.Time*int64(time.Millisecond))
	return s, nil
}

// GetTrades gets recent trades from the exchange
func (by *Bybit) GetTrades(symbol string, limit int64) ([]TradeItem, error) {
	resp := struct {
		Data []struct {
			Price        float64 `json:"price,string"`
			Time         int64   `json:"time"`
			Quantity     float64 `json:"qty,string"`
			IsBuyerMaker bool    `json:"isBuyerMaker"`
		} `json:"result"`
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(limit, 10))
	path := common.EncodeURLValues(bybitRecentTrades, params)
	err := by.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return nil, err
	}

	var trades []TradeItem
	for x := range resp.Data {
		tradeSide := ""
		if resp.Data[x].IsBuyerMaker {
			tradeSide = sideBuy
		} else {
			tradeSide = sideSell
		}

		trades = append(trades, TradeItem{
			CurrencyPair: symbol,
			Price:        resp.Data[x].Price,
			Side:         tradeSide,
			Volume:       resp.Data[x].Quantity,
			TradeTime:    time.Unix(0, resp.Data[x].Time*int64(time.Millisecond)),
		})
	}
	return trades, nil
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
