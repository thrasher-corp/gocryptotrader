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
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com/api"

	// Public market endpoints
	kucoinFuturesOpenContracts    = "/v1/contracts/active"
	kucoinFuturesContract         = "/v1/contracts/%s"
	kucoinFuturesRealTimeTicker   = "/v1/ticker"
	kucoinFuturesFullOrderbook    = "/v1/level2/snapshot"
	kucoinFuturesPartOrderbook20  = "/v1/level2/depth20"
	kucoinFuturesPartOrderbook100 = "/v1/level2/depth100"
	kucoinFuturesTradeHistory     = "/v1/trade/history"
	kucoinFuturesInterestRate     = "/v1/interest/query"
	kucoinFuturesIndex            = "/v1/index/query"
	kucoinFuturesMarkPrice        = "/v1/mark-price/%s/current"
	kucoinFuturesPremiumIndex     = "/v1/premium/query"
	kucoinFuturesFundingRate      = "/v1/funding-rate/%s/current"
	kucoinFuturesServerTime       = "/v1/timestamp"
	kucoinFuturesServiceStatus    = "/v1/status"
	kucoinFuturesKline            = "/v1/kline/query"

	// Authenticated endpoints
	kucoinFuturesOrder                     = "/v1/orders"
	kucoinFuturesCancelOrder               = "/v1/orders/%s"
	kucoinFuturesStopOrder                 = "/v1/stopOrders"
	kucoinFuturesRecentCompletedOrder      = "/v1/recentDoneOrders"
	kucoinFuturesGetOrderDetails           = "/v1/orders/%s"
	kucoinFuturesGetOrderDetailsByClientID = "/v1/orders/byClientOid"

	kucoinFuturesFills               = "/v1/fills"
	kucoinFuturesRecentFills         = "/v1/recentFills"
	kucoinFuturesOpenOrderStats      = "/v1/openOrderStatistics"
	kucoinFuturesPosition            = "/v1/position"
	kucoinFuturesPositionList        = "/v1/positions"
	kucoinFuturesSetAutoDeposit      = "/v1/position/margin/auto-deposit-status"
	kucoinFuturesAddMargin           = "/v1/position/margin/deposit-margin"
	kucoinFuturesRiskLimitLevel      = "/v1/contracts/risk-limit/%s"
	kucoinFuturesUpdateRiskLmitLevel = "/v1/position/risk-limit-level/change"
	kucoinFuturesFundingHistory      = "/v1/funding-history"

	kucoinFuturesAccountOverview              = "/v1/account-overview"
	kucoinFuturesTransactionHistory           = "/v1/transaction-history"
	kucoinFuturesSubAccountAPIKey             = "/v1/sub/api-key"
	kucoinFuturesDepositAddress               = "/v1/deposit-address"
	kucoinFuturesDepositsList                 = "/v1/deposit-list"
	kucoinFuturesWithdrawalLimit              = "/v1/withdrawals/quotas"
	kucoinFuturesWithdrawalList               = "/v1/withdrawal-list"
	kucoinFuturesCancelWithdrawal             = "/v1/withdrawals/%s"
	kucoinFuturesTransferFundtoMainAccount    = "/v3/transfer-out"
	kucoinFuturesTransferFundtoFuturesAccount = "/v1/transfer-in"
	kucoinFuturesTransferOutList              = "/v1/transfer-list"
	kucoinFuturesCancelTransferOut            = "/v1/cancel/transfer-out"
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
func (k *Kucoin) GetFuturesRealTimeTicker(ctx context.Context, symbol string) (*FuturesTicker, error) {
	resp := struct {
		Data FuturesTicker `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return &resp.Data, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	return &resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesRealTimeTicker, params), publicSpotRate, &resp)
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
func (k *Kucoin) GetFuturesServiceStatus(ctx context.Context, symbol string) (*FuturesServiceStatus, error) {
	resp := struct {
		Data FuturesServiceStatus `json:"data"`
		Error
	}{}
	return &resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, kucoinFuturesServiceStatus, publicSpotRate, &resp)
}

// GetFuturesKline get contract's kline data
func (k *Kucoin) GetFuturesKline(ctx context.Context, granularity, symbol string, from, to time.Time) ([]FuturesKline, error) {
	resp := struct {
		Data [][]float64 `json:"data"`
		Error
	}{}

	params := url.Values{}
	if granularity == "" {
		return nil, errors.New("granularity can not be empty")
	}
	if !common.StringDataContains(validGranularity, granularity) {
		return nil, errors.New("invalid granularity")
	}
	// The granularity (granularity parameter of K-line) represents the number of minutes, the available granularity scope is: 1,5,15,30,60,120,240,480,720,1440,10080. Requests beyond the above range will be rejected.
	params.Set("granularity", granularity)
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
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
func (k *Kucoin) PostFuturesOrder(ctx context.Context, clientOID, side, symbol,
	orderType, remark,
	stop, stopPriceType, stopPrice, timeInForce string, size, price,
	leverage, visibleSize float64, reduceOnly, closeOrder, forceHold, postOnly, hidden, iceberg bool) (string, error) {
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
	if leverage != 0 {
		params["leverage"] = strconv.FormatFloat(leverage, 'f', -1, 64)
	}
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
		if price <= 0 {
			return resp.Data.OrderID, fmt.Errorf("%w %f", errInvalidPrice, price)
		}
		params["price"] = price
		if size <= 0 {
			return resp.Data.OrderID, fmt.Errorf("%w size must be non-zero positive value", errInvalidSize)
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

// GetFuturesFills gets list of recent fills
func (k *Kucoin) GetFuturesFills(ctx context.Context, orderID, symbol, side, orderType string, startAt, endAt time.Time) ([]FuturesFill, error) {
	resp := struct {
		Data struct {
			CurrentPage int64         `json:"currentPage"`
			PageSize    int64         `json:"pageSize"`
			TotalNum    int64         `json:"totalNum"`
			TotalPage   int64         `json:"totalPage"`
			Items       []FuturesFill `json:"items"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
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
	return resp.Data.Items, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesFills, params), nil, publicSpotRate, &resp)
}

// GetFuturesRecentFills gets list of 1000 recent fills in the last 24 hrs
func (k *Kucoin) GetFuturesRecentFills(ctx context.Context) ([]FuturesFill, error) {
	resp := struct {
		Data []FuturesFill `json:"data"`
		Error
	}{}

	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, kucoinFuturesRecentFills, nil, publicSpotRate, &resp)
}

// GetFuturesOpenOrderStats gets the total number and value of the all your active orders
func (k *Kucoin) GetFuturesOpenOrderStats(ctx context.Context, symbol string) (OpenOrderStats, error) {
	resp := struct {
		Data OpenOrderStats `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesOpenOrderStats, params), nil, publicSpotRate, &resp)
}

// GetFuturesPosition gets the position details of a specified position
func (k *Kucoin) GetFuturesPosition(ctx context.Context, symbol string) (Position, error) {
	resp := struct {
		Data Position `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesPosition, params), nil, publicSpotRate, &resp)
}

// GetFuturesPositionList gets the list of position with details
func (k *Kucoin) GetFuturesPositionList(ctx context.Context) ([]Position, error) {
	resp := struct {
		Data []Position `json:"data"`
		Error
	}{}

	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, kucoinFuturesPositionList, nil, publicSpotRate, &resp)
}

// SetAutoDepositMargin enable/disable of auto-deposit margin
func (k *Kucoin) SetAutoDepositMargin(ctx context.Context, symbol string, status bool) (bool, error) {
	resp := struct {
		Data bool `json:"data"`
		Error
	}{}

	params := make(map[string]interface{})
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	params["symbol"] = symbol
	params["status"] = status
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, kucoinFuturesSetAutoDeposit, params, publicSpotRate, &resp)
}

// AddMargin is used to add margin manually
func (k *Kucoin) AddMargin(ctx context.Context, symbol, uniqueID string, margin float64) (Position, error) {
	resp := struct {
		Data Position `json:"data"`
		Error
	}{}

	params := make(map[string]interface{})
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	params["symbol"] = symbol
	if uniqueID == "" {
		return resp.Data, errors.New("uniqueID can't be empty")
	}
	params["bizNo"] = uniqueID
	if margin <= 0 {
		return resp.Data, errors.New("margin can't be zero or negative")
	}
	params["margin"] = strconv.FormatFloat(margin, 'f', -1, 64)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, kucoinFuturesAddMargin, params, publicSpotRate, &resp)
}

// GetFuturesRiskLimitLevel gets information about risk limit level of a specific contract
func (k *Kucoin) GetFuturesRiskLimitLevel(ctx context.Context, symbol string) ([]RiskLimitLevel, error) {
	resp := struct {
		Data []RiskLimitLevel `json:"data"`
		Error
	}{}

	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, fmt.Sprintf(kucoinFuturesRiskLimitLevel, symbol), nil, publicSpotRate, &resp)
}

// UpdateRiskLmitLevel is used to adjustment the risk limit level
func (k *Kucoin) UpdateRiskLmitLevel(ctx context.Context, symbol string, level int64) (bool, error) {
	resp := struct {
		Data bool `json:"data"`
		Error
	}{}

	params := make(map[string]interface{})
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	params["symbol"] = symbol
	params["level"] = strconv.FormatInt(level, 10)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, kucoinFuturesUpdateRiskLmitLevel, params, publicSpotRate, &resp)
}

// GetFuturesFundingHistory gets information about funding history
func (k *Kucoin) GetFuturesFundingHistory(ctx context.Context, symbol string, offset, maxCount int64, reverse, forward bool, startAt, endAt time.Time) ([]FundingHistory, error) {
	resp := struct {
		Data []FundingHistory `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
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
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesFundingHistory, params), nil, publicSpotRate, &resp)
}

// GetFuturesAccountOverview gets future account overview
func (k *Kucoin) GetFuturesAccountOverview(ctx context.Context, currency string) (Account, error) {
	resp := struct {
		Data Account `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesAccountOverview, params), nil, publicSpotRate, &resp)
}

// GetFuturesTransactionHistory gets future transaction history
func (k *Kucoin) GetFuturesTransactionHistory(ctx context.Context, currency, txType string, offset, maxCount int64, forward bool, startAt, endAt time.Time) ([]TransactionHistory, error) {
	resp := struct {
		Data struct {
			List    []TransactionHistory `json:"dataList"`
			HasMore bool                 `json:"hasMore"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	if txType != "" {
		params.Set("type", txType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
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
	return resp.Data.List, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesTransactionHistory, params), nil, publicSpotRate, &resp)
}

// CreateFuturesSubAccountAPIKey is used to create Futures APIs for sub-accounts
func (k *Kucoin) CreateFuturesSubAccountAPIKey(ctx context.Context, ipWhitelist, passphrase, permission, remark, subName string) (APIKeyDetail, error) {
	resp := struct {
		Data APIKeyDetail `json:"data"`
		Error
	}{}

	params := make(map[string]interface{})
	if ipWhitelist != "" {
		params["ipWhitelist"] = ipWhitelist
	}
	if passphrase == "" {
		return resp.Data, errors.New("passphrase can't be empty")
	}
	params["passphrase"] = passphrase
	if permission != "" {
		params["permission"] = permission
	}
	if remark == "" {
		return resp.Data, errors.New("remark can't be empty")
	}
	params["remark"] = remark
	if subName == "" {
		return resp.Data, errors.New("subName can't be empty")
	}
	params["subName"] = subName
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, kucoinFuturesSubAccountAPIKey, params, publicSpotRate, &resp)
}

// GetFuturesDepositAddress gets deposit address for currency
func (k *Kucoin) GetFuturesDepositAddress(ctx context.Context, currency string) (DepositAddress, error) {
	resp := struct {
		Data DepositAddress `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency == "" {
		return resp.Data, errors.New("currency can't be empty")
	}
	params.Set("currency", currency)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesDepositAddress, params), nil, publicSpotRate, &resp)
}

// GetFuturesDepositsList gets deposits list
func (k *Kucoin) GetFuturesDepositsList(ctx context.Context, currency, status string, startAt, endAt time.Time) ([]DepositDetail, error) {
	resp := struct {
		Data struct {
			CurrentPage int64           `json:"currentPage"`
			PageSize    int64           `json:"pageSize"`
			TotalNum    int64           `json:"totalNum"`
			TotalPage   int64           `json:"totalPage"`
			Items       []DepositDetail `json:"items"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	return resp.Data.Items, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesDepositsList, params), nil, publicSpotRate, &resp)
}

// GetFuturesDepositsList gets withdrawal limits for currency
func (k *Kucoin) GetFuturesWithdrawalLimit(ctx context.Context, currency string) (WithdrawalLimit, error) {
	resp := struct {
		Data WithdrawalLimit `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency == "" {
		return resp.Data, errors.New("currency can't be empty")
	}
	params.Set("currency", currency)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesWithdrawalLimit, params), nil, publicSpotRate, &resp)
}

// GetFuturesWithdrawalList gets withdrawal list
func (k *Kucoin) GetFuturesWithdrawalList(ctx context.Context, currency, status string, startAt, endAt time.Time) ([]WithdrawalHistory, error) {
	resp := struct {
		Data struct {
			CurrentPage int64               `json:"currentPage"`
			PageSize    int64               `json:"pageSize"`
			TotalNum    int64               `json:"totalNum"`
			TotalPage   int64               `json:"totalPage"`
			Items       []WithdrawalHistory `json:"items"`
		} `json:"data"`
		Error
	}{}
	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	return resp.Data.Items, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesWithdrawalList, params), nil, publicSpotRate, &resp)
}

// CancelFuturesWithdrawal is used to cancel withdrawal request of only PROCESSING status
func (k *Kucoin) CancelFuturesWithdrawal(ctx context.Context, withdrawalID string) (bool, error) {
	resp := struct {
		Data bool `json:"data"`
		Error
	}{}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodDelete, fmt.Sprintf(kucoinFuturesCancelWithdrawal, withdrawalID), nil, publicSpotRate, &resp)
}

// TransferFuturesFundsToMainAccount helps in transfering funds from futures to main/trade account
func (k *Kucoin) TransferFuturesFundsToMainAccount(ctx context.Context, amount float64, currency, recAccountType string) (TransferRes, error) {
	resp := struct {
		Data TransferRes `json:"data"`
		Error
	}{}

	params := make(map[string]interface{})
	if amount <= 0 {
		return resp.Data, errors.New("amount can't be zero or negative")
	}
	params["amount"] = amount
	if currency == "" {
		return resp.Data, errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if recAccountType == "" {
		return resp.Data, errors.New("recAccountType can't be empty")
	}
	params["recAccountType"] = recAccountType
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, kucoinFuturesTransferFundtoMainAccount, params, publicSpotRate, &resp)
}

// TransferFundsToFuturesAccount helps in transfering funds from payee account to futures account
func (k *Kucoin) TransferFundsToFuturesAccount(ctx context.Context, amount float64, currency, payAccountType string) error {
	resp := struct {
		Error
	}{}

	params := make(map[string]interface{})
	if amount <= 0 {
		return errors.New("amount can't be zero or negative")
	}
	params["amount"] = amount
	if currency == "" {
		return errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if payAccountType == "" {
		return errors.New("payAccountType can't be empty")
	}
	params["payAccountType"] = payAccountType
	return k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, kucoinFuturesTransferFundtoFuturesAccount, params, publicSpotRate, &resp)
}

// GetFuturesTransferOutList gets list of trasfer out
func (k *Kucoin) GetFuturesTransferOutList(ctx context.Context, currency, status string, startAt, endAt time.Time) ([]Transfer, error) {
	resp := struct {
		Data struct {
			CurrentPage int64      `json:"currentPage"`
			PageSize    int64      `json:"pageSize"`
			TotalNum    int64      `json:"totalNum"`
			TotalPage   int64      `json:"totalPage"`
			Items       []Transfer `json:"items"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency == "" {
		return resp.Data.Items, errors.New("currency can't be empty")
	}
	params.Set("currency", currency)
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	return resp.Data.Items, k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, common.EncodeURLValues(kucoinFuturesTransferOutList, params), nil, publicSpotRate, &resp)
}

// CancelFuturesTransferOut is used to cancel transfer out request of only PROCESSING status
func (k *Kucoin) CancelFuturesTransferOut(ctx context.Context, applyID string) error {
	resp := struct {
		Error
	}{}

	params := url.Values{}
	if applyID == "" {
		return errors.New("applyID can't be empty")
	}
	params.Set("applyId", applyID)
	return k.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodDelete, common.EncodeURLValues(kucoinFuturesCancelTransferOut, params), nil, publicSpotRate, &resp)
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
	s.Sequence = o.Data.Sequence
	s.Time = o.Data.Time.Time()
	return &s, err
}
