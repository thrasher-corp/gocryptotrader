package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const (
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com"

	// Public market endpoints
	kucoinFuturesOpenContracts    = "/api/v1/contracts/active"
	kucoinFuturesContract         = "/api/v1/contracts/%s"
	kucoinFuturesRealTimeTicker   = "/api/v1/ticker"
	kucoinFuturesFullOrderbook    = "/api/v1/level2/snapshot"
	kucoinFuturesPartOrderbook20  = "/api/v1/level2/depth20"
	kucoinFuturesPartOrderbook100 = "/api/v1/level2/depth100"
	kucoinFuturesTradeHistory     = "/api/v1/trade/history"
	kucoinFuturesInterestRate     = "/api/v1/interest/query"
	kucoinFuturesIndex            = "/api/v1/index/query"
	kucoinFuturesMarkPrice        = "/api/v1/mark-price/%s/current"
	kucoinFuturesPremiumIndex     = "/api/v1/premium/query"
	kucoinFuturesFundingRate      = "/api/v1/funding-rate/%s/current"
	kucoinFuturesServerTime       = "/api/v1/timestamp"
	kucoinFuturesServiceStatus    = "/api/v1/status"
	kucoinFuturesKline            = "/api/v1/kline/query"

	// Authenticated endpoints
	kucoinFuturesOrder                     = "/api/v1/orders"
	kucoinFuturesCancelOrder               = "/api/v1/orders/%s"
	kucoinFuturesStopOrder                 = "/api/v1/stopOrders"
	kucoinFuturesRecentCompletedOrder      = "/api/v1/recentDoneOrders"
	kucoinFuturesGetOrderDetails           = "/api/v1/orders/%s"
	kucoinFuturesGetOrderDetailsByClientID = "/api/v1/orders/byClientOid"
)

// GetFuturesOpenContracts gets all open futures contract with its details
func (k *Kucoin) GetFuturesOpenContracts(ctx context.Context) ([]Contract, error) {
	resp := struct {
		Data []Contract `json:"data"`
		Error
	}{}

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, kucoinFuturesOpenContracts, publicSpotRate, &resp)
}

// GetFuturesContract get contract details
func (k *Kucoin) GetFuturesContract(ctx context.Context, symbol string) (Contract, error) {
	resp := struct {
		Data Contract `json:"data"`
		Error
	}{}

	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesContract, symbol), publicSpotRate, &resp)
}

// GetFuturesRealTimeTicker get real time ticker
func (k *Kucoin) GetFuturesRealTimeTicker(ctx context.Context, symbol string) (FuturesTicker, error) {
	resp := struct {
		Data FuturesTicker `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesRealTimeTicker, params), publicSpotRate, &resp)
}

// GetFuturesOrderbook gets full orderbook for a specified symbol
func (k *Kucoin) GetFuturesOrderbook(ctx context.Context, symbol string) (*Orderbook, error) {
	var o futuresOrderbookResponse
	params := url.Values{}
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	err := k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesFullOrderbook, params), publicSpotRate, &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o)
}

// GetFuturesPartOrderbook20 gets orderbook for a specified symbol with depth 20
func (k *Kucoin) GetFuturesPartOrderbook20(ctx context.Context, symbol string) (*Orderbook, error) {
	var o futuresOrderbookResponse
	params := url.Values{}
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	err := k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesPartOrderbook20, params), publicSpotRate, &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o)
}

// GetFuturesPartOrderbook100 gets orderbook for a specified symbol with depth 100
func (k *Kucoin) GetFuturesPartOrderbook100(ctx context.Context, symbol string) (*Orderbook, error) {
	var o futuresOrderbookResponse
	params := url.Values{}
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	err := k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesPartOrderbook100, params), publicSpotRate, &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o)
}

// GetFuturesTradeHistory get last 100 trades for symbol
func (k *Kucoin) GetFuturesTradeHistory(ctx context.Context, symbol string) ([]FuturesTrade, error) {
	resp := struct {
		Data []FuturesTrade `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesTradeHistory, params), publicSpotRate, &resp)
}

// GetFuturesInterestRate get interest rate
func (k *Kucoin) GetFuturesInterestRate(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) ([]FuturesInterestRate, error) {
	resp := struct {
		Data struct {
			List    []FuturesInterestRate `json:"dataList"`
			HasMore bool                  `json:"hasMore"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data.List, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)

	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if reverse {
		params.Set("reverse", "true")
	} else {
		params.Set("reverse", "false")
	}
	if forward {
		params.Set("forward", "true")
	} else {
		params.Set("forward", "false")
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	return resp.Data.List, k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesInterestRate, params), publicSpotRate, &resp)
}

// GetFuturesInterestRate get interest rate
func (k *Kucoin) GetFuturesIndexList(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) ([]FuturesIndex, error) {
	resp := struct {
		Data struct {
			List    []FuturesIndex `json:"dataList"`
			HasMore bool           `json:"hasMore"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data.List, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)

	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if reverse {
		params.Set("reverse", "true")
	} else {
		params.Set("reverse", "false")
	}
	if forward {
		params.Set("forward", "true")
	} else {
		params.Set("forward", "false")
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	return resp.Data.List, k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesIndex, params), publicSpotRate, &resp)
}

// GetFuturesCurrentMarkPrice get current mark price
func (k *Kucoin) GetFuturesCurrentMarkPrice(ctx context.Context, symbol string) (FuturesMarkPrice, error) {
	resp := struct {
		Data FuturesMarkPrice `json:"data"`
		Error
	}{}

	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesMarkPrice, symbol), publicSpotRate, &resp)
}

// GetFuturesPremiumIndex get premium index
func (k *Kucoin) GetFuturesPremiumIndex(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) ([]FuturesInterestRate, error) {
	resp := struct {
		Data struct {
			List    []FuturesInterestRate `json:"dataList"`
			HasMore bool                  `json:"hasMore"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data.List, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)

	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if reverse {
		params.Set("reverse", "true")
	} else {
		params.Set("reverse", "false")
	}
	if forward {
		params.Set("forward", "true")
	} else {
		params.Set("forward", "false")
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	return resp.Data.List, k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesPremiumIndex, params), publicSpotRate, &resp)
}

// GetFuturesCurrentFundingRate get current funding rate
func (k *Kucoin) GetFuturesCurrentFundingRate(ctx context.Context, symbol string) (FuturesFundingRate, error) {
	resp := struct {
		Data FuturesFundingRate `json:"data"`
		Error
	}{}

	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesFundingRate, symbol), publicSpotRate, &resp)
}

// GetFuturesServerTime get server time
func (k *Kucoin) GetFuturesServerTime(ctx context.Context, symbol string) (time.Time, error) {
	resp := struct {
		Data kucoinTimeMilliSec `json:"data"`
		Error
	}{}
	return resp.Data.Time(), k.SendHTTPRequest(ctx, exchange.RestFutures, kucoinFuturesServerTime, publicSpotRate, &resp)
}

// GetFuturesServiceStatus get service status
func (k *Kucoin) GetFuturesServiceStatus(ctx context.Context, symbol string) (string, string, error) {
	resp := struct {
		Data struct {
			Status  string `json:"status"`
			Message string `json:"msg"`
		} `json:"data"`
		Error
	}{}
	return resp.Data.Status, resp.Data.Message, k.SendHTTPRequest(ctx, exchange.RestFutures, kucoinFuturesServiceStatus, publicSpotRate, &resp)
}

// GetFuturesKline get contract's kline data
func (k *Kucoin) GetFuturesKline(ctx context.Context, granularity, symbol string, from, to time.Time) ([]FuturesKline, error) {
	resp := struct {
		Data [][]float64 `json:"data"`
		Error
	}{}

	params := url.Values{}
	if granularity == "" {
		return nil, errors.New("granularity can't be empty")
	}
	if !common.StringDataContains(validGranularity, granularity) {
		return nil, errors.New("invalid granularity")
	}
	params.Set("granularity", granularity)
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(from.UnixMilli(), 10))
	}

	err := k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesKline, params), publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	kline := make([]FuturesKline, len(resp.Data))
	for i := range resp.Data {
		kline[i] = FuturesKline{
			StartTime: time.UnixMilli(int64(resp.Data[i][0])),
			Open:      resp.Data[i][1],
			High:      resp.Data[i][2],
			Low:       resp.Data[i][3],
			Close:     resp.Data[i][4],
			Volume:    resp.Data[i][5],
		}
	}
	return kline, nil
}

// PostOrder used to place two types of futures orders: limit and market
func (k *Kucoin) PostFuturesOrder(ctx context.Context, clientOID, side, symbol, orderType, leverage, remark, stop, stopPriceType, stopPrice, price, timeInForce string, size, visibleSize float64, reduceOnly, closeOrder, forceHold, postOnly, hidden, iceberg bool) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
		Error
	}{}

	params := make(map[string]interface{})
	if clientOID == "" {
		return resp.Data.OrderID, errors.New("clientOid can't be empty")
	}
	params["clientOid"] = clientOID
	if side == "" {
		return resp.Data.OrderID, errors.New("side can't be empty")
	}
	params["side"] = side
	if symbol == "" {
		return resp.Data.OrderID, errors.New("symbol can't be empty")
	}
	params["symbol"] = symbol
	if leverage == "" {
		return resp.Data.OrderID, errors.New("leverage can't be empty")
	}
	params["leverage"] = leverage
	if remark != "" {
		params["remark"] = remark
	}
	if stop != "" {
		params["stp"] = stop
		if stopPriceType == "" {
			return resp.Data.OrderID, errors.New("stopPriceType can't be empty")
		}
		params["stopPriceType"] = stopPriceType
		if stopPrice == "" {
			return resp.Data.OrderID, errors.New("stopPrice can't be empty")
		}
		params["stopPrice"] = stopPrice
	}
	if orderType == "limit" || orderType == "" {
		if price == "" {
			return resp.Data.OrderID, errors.New("price can't be empty")
		}
		params["price"] = price
		if size <= 0 {
			return resp.Data.OrderID, errors.New("size can't be zero or negative")
		}
		params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		if timeInForce != "" {
			params["timeInForce"] = timeInForce
		}
		params["postOnly"] = postOnly
		params["hidden"] = hidden
		params["iceberg"] = iceberg
		if visibleSize > 0 {
			params["visibleSize"] = strconv.FormatFloat(visibleSize, 'f', -1, 64)
		}
	} else if orderType == "market" {
		if size > 0 {
			params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		}
	} else {
		return resp.Data.OrderID, errors.New("invalid orderType")
	}

	if orderType != "" {
		params["type"] = orderType
	}
	params["reduceOnly"] = reduceOnly
	params["closeOrder"] = closeOrder
	params["forceHold"] = forceHold
	return resp.Data.OrderID, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, kucoinFuturesOrder, params, publicSpotRate, &resp)
}

// CancelFuturesOrder used to cancel single order previously placed including a stop order
func (k *Kucoin) CancelFuturesOrder(ctx context.Context, orderID string) ([]string, error) {
	resp := struct {
		Data struct {
			CancelledOrderIDs []string `json:"cancelledOrderIds"`
		} `json:"data"`
		Error
	}{}

	if orderID == "" {
		return resp.Data.CancelledOrderIDs, errors.New("orderID can't be empty")
	}
	return resp.Data.CancelledOrderIDs, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodDelete, fmt.Sprintf(kucoinFuturesCancelOrder, orderID), nil, publicSpotRate, &resp)
}

// CancelAllFuturesOpenOrders used to cancel all futures order excluding stop orders
func (k *Kucoin) CancelAllFuturesOpenOrders(ctx context.Context, symbol string) ([]string, error) {
	resp := struct {
		Data struct {
			CancelledOrderIDs []string `json:"cancelledOrderIds"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp.Data.CancelledOrderIDs, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodDelete, common.EncodeURLValues(kucoinFuturesOrder, params), nil, publicSpotRate, &resp)
}

// CancelAllFuturesStopOrders used to cancel all untriggered stop orders
func (k *Kucoin) CancelAllFuturesStopOrders(ctx context.Context, symbol string) ([]string, error) {
	resp := struct {
		Data struct {
			CancelledOrderIDs []string `json:"cancelledOrderIds"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return resp.Data.CancelledOrderIDs, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodDelete, common.EncodeURLValues(kucoinFuturesStopOrder, params), nil, publicSpotRate, &resp)
}

// GetFuturesOrders gets the user current futures order list
func (k *Kucoin) GetFuturesOrders(ctx context.Context, status, symbol, side, orderType string, startAt, endAt time.Time) ([]FuturesOrder, error) {
	resp := struct {
		Data struct {
			CurrentPage int64          `json:"currentPage"`
			PageSize    int64          `json:"pageSize"`
			TotalNum    int64          `json:"totalNum"`
			TotalPage   int64          `json:"totalPage"`
			Items       []FuturesOrder `json:"items"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	return resp.Data.Items, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesOrder, params), nil, publicSpotRate, &resp)
}

// GetUntriggeredFuturesStopOrders gets the untriggered stop orders list
func (k *Kucoin) GetUntriggeredFuturesStopOrders(ctx context.Context, symbol, side, orderType string, startAt, endAt time.Time) ([]FuturesOrder, error) {
	resp := struct {
		Data struct {
			CurrentPage int64          `json:"currentPage"`
			PageSize    int64          `json:"pageSize"`
			TotalNum    int64          `json:"totalNum"`
			TotalPage   int64          `json:"totalPage"`
			Items       []FuturesOrder `json:"items"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	return resp.Data.Items, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesStopOrder, params), nil, publicSpotRate, &resp)
}

// GetFuturesRecentCompletedOrders gets list of recent 1000 orders in the last 24 hours
func (k *Kucoin) GetFuturesRecentCompletedOrders(ctx context.Context) ([]FuturesOrder, error) {
	resp := struct {
		Data []FuturesOrder `json:"data"`
		Error
	}{}

	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, kucoinFuturesRecentCompletedOrder, nil, publicSpotRate, &resp)
}

// GetFuturesOrderDetails gets single order details by order ID
func (k *Kucoin) GetFuturesOrderDetails(ctx context.Context, orderID string) (FuturesOrder, error) {
	resp := struct {
		Data FuturesOrder `json:"data"`
		Error
	}{}

	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, fmt.Sprintf(kucoinFuturesGetOrderDetails, orderID), nil, publicSpotRate, &resp)
}

// GetFuturesOrderDetailsByClientID gets single order details by client ID
func (k *Kucoin) GetFuturesOrderDetailsByClientID(ctx context.Context, clientID string) (FuturesOrder, error) {
	resp := struct {
		Data FuturesOrder `json:"data"`
		Error
	}{}

	params := url.Values{}
	if clientID == "" {
		return resp.Data, errors.New("clientID can't be empty")
	}
	params.Set("clientOid", clientID)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesGetOrderDetailsByClientID, params), nil, publicSpotRate, &resp)
}

func processFuturesOB(ob [][2]float64) ([]orderbook.Item, error) {
	o := make([]orderbook.Item, len(ob))
	for x := range ob {
		o[x] = orderbook.Item{
			Price:  ob[x][0],
			Amount: ob[x][1],
		}
	}
	return o, nil
}

func constructFuturesOrderbook(o *futuresOrderbookResponse) (*Orderbook, error) {
	var (
		s   Orderbook
		err error
	)
	s.Bids, err = processFuturesOB(o.Data.Bids)
	if err != nil {
		return nil, err
	}
	s.Asks, err = processFuturesOB(o.Data.Asks)
	if err != nil {
		return nil, err
	}
	s.Time = o.Data.Time.Time()
	return &s, err
}
