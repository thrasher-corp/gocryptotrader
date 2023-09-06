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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const (
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com/api"
	kucoinWebsocketURL  = "wss://ws-api.kucoin.com/endpoint"

	// Public market endpoints
	kucoinFuturesOpenContracts    = "/v1/contracts/active"
	kucoinFuturesContract         = "/v1/contracts/"
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
	kucoinFuturesCancelOrder               = "/v1/orders/"
	kucoinFuturesStopOrder                 = "/v1/stopOrders"
	kucoinFuturesRecentCompletedOrder      = "/v1/recentDoneOrders"
	kucoinFuturesGetOrderDetails           = "/v1/orders/"
	kucoinFuturesGetOrderDetailsByClientID = "/v1/orders/byClientOid"

	kucoinFuturesFills               = "/v1/fills"
	kucoinFuturesRecentFills         = "/v1/recentFills"
	kucoinFuturesOpenOrderStats      = "/v1/openOrderStatistics"
	kucoinFuturesPosition            = "/v1/position"
	kucoinFuturesPositionList        = "/v1/positions"
	kucoinFuturesSetAutoDeposit      = "/v1/position/margin/auto-deposit-status"
	kucoinFuturesAddMargin           = "/v1/position/margin/deposit-margin"
	kucoinFuturesRiskLimitLevel      = "/v1/contracts/risk-limit/"
	kucoinFuturesUpdateRiskLmitLevel = "/v1/position/risk-limit-level/change"
	kucoinFuturesFundingHistory      = "/v1/funding-history"

	kucoinFuturesAccountOverview              = "/v1/account-overview"
	kucoinFuturesTransactionHistory           = "/v1/transaction-history"
	kucoinFuturesSubAccountAPI                = "/v1/sub/api-key"
	kucoinFuturesDepositAddress               = "/v1/deposit-address"
	kucoinFuturesDepositsList                 = "/v1/deposit-list"
	kucoinFuturesWithdrawalLimit              = "/v1/withdrawals/quotas"
	kucoinFuturesWithdrawalList               = "/v1/withdrawal-list"
	kucoinFuturesCancelWithdrawal             = "/v1/withdrawals/"
	kucoinFuturesTransferFundtoMainAccount    = "/v3/transfer-out"
	kucoinFuturesTransferFundtoFuturesAccount = "/v1/transfer-in"
	kucoinFuturesTransferOutList              = "/v1/transfer-list"
	kucoinFuturesCancelTransferOut            = "/v1/cancel/transfer-out"
)

// GetFuturesOpenContracts gets all open futures contract with its details
func (ku *Kucoin) GetFuturesOpenContracts(ctx context.Context) ([]Contract, error) {
	var resp []Contract
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, kucoinFuturesOpenContracts, &resp)
}

// GetFuturesContract get contract details
func (ku *Kucoin) GetFuturesContract(ctx context.Context, symbol string) (*Contract, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	var resp *Contract
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, kucoinFuturesContract+symbol, &resp)
}

// GetFuturesRealTimeTicker get real time ticker
func (ku *Kucoin) GetFuturesRealTimeTicker(ctx context.Context, symbol string) (*FuturesTicker, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesTicker
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesRealTimeTicker, params), &resp)
}

// GetFuturesOrderbook gets full orderbook for a specified symbol
func (ku *Kucoin) GetFuturesOrderbook(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o futuresOrderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveFullOrderbookLevel2EPL, common.EncodeURLValues(kucoinFuturesFullOrderbook, params), &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o)
}

// GetFuturesPartOrderbook20 gets orderbook for a specified symbol with depth 20
func (ku *Kucoin) GetFuturesPartOrderbook20(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o futuresOrderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesPartOrderbook20, params), &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o)
}

// GetFuturesPartOrderbook100 gets orderbook for a specified symbol with depth 100
func (ku *Kucoin) GetFuturesPartOrderbook100(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o futuresOrderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesPartOrderbook100, params), &o)
	if err != nil {
		return nil, err
	}
	return constructFuturesOrderbook(&o)
}

// GetFuturesTradeHistory get last 100 trades for symbol
func (ku *Kucoin) GetFuturesTradeHistory(ctx context.Context, symbol string) ([]FuturesTrade, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []FuturesTrade
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesTradeHistory, params), &resp)
}

// GetFuturesInterestRate get interest rate
func (ku *Kucoin) GetFuturesInterestRate(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) (*FundingInterestRateResponse, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)

	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FundingInterestRateResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesInterestRate, params), &resp)
}

// GetFuturesIndexList retrieves futures index information for a symbol
func (ku *Kucoin) GetFuturesIndexList(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) (*FuturesIndexResponse, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesIndexResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesIndex, params), &resp)
}

// GetFuturesCurrentMarkPrice get current mark price
func (ku *Kucoin) GetFuturesCurrentMarkPrice(ctx context.Context, symbol string) (*FuturesMarkPrice, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	var resp *FuturesMarkPrice
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, fmt.Sprintf(kucoinFuturesMarkPrice, symbol), &resp)
}

// GetFuturesPremiumIndex get premium index
func (ku *Kucoin) GetFuturesPremiumIndex(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) (*FuturesInterestRateResponse, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesInterestRateResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesPremiumIndex, params), &resp)
}

// GetFuturesCurrentFundingRate get current funding rate
func (ku *Kucoin) GetFuturesCurrentFundingRate(ctx context.Context, symbol string) (*FuturesFundingRate, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	var resp *FuturesFundingRate
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, fmt.Sprintf(kucoinFuturesFundingRate, symbol), &resp)
}

// GetFuturesServerTime get server time
func (ku *Kucoin) GetFuturesServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Data convert.ExchangeTime `json:"data"`
		Error
	}{}
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, kucoinFuturesServerTime, &resp)
	if err != nil {
		return time.Time{}, err
	}
	return resp.Data.Time(), nil
}

// GetFuturesServiceStatus get service status
func (ku *Kucoin) GetFuturesServiceStatus(ctx context.Context) (*FuturesServiceStatus, error) {
	var resp *FuturesServiceStatus
	return resp, ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, kucoinFuturesServiceStatus, &resp)
}

// GetFuturesKline get contract's kline data
func (ku *Kucoin) GetFuturesKline(ctx context.Context, granularity int64, symbol string, from, to time.Time) ([]FuturesKline, error) {
	if granularity == 0 {
		return nil, errors.New("granularity can not be empty")
	}
	if !common.StringDataContains(validGranularity, strconv.FormatInt(granularity, 10)) {
		return nil, errors.New("invalid granularity")
	}
	params := url.Values{}
	// The granularity (granularity parameter of K-line) represents the number of minutes, the available granularity scope is: 1,5,15,30,60,120,240,480,720,1440,10080. Requests beyond the above range will be rejected.
	params.Set("granularity", strconv.FormatInt(granularity, 10))
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
	var resp [][6]float64
	err := ku.SendHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, common.EncodeURLValues(kucoinFuturesKline, params), &resp)
	if err != nil {
		return nil, err
	}
	kline := make([]FuturesKline, len(resp))
	for i := range resp {
		kline[i] = FuturesKline{
			StartTime: time.UnixMilli(int64(resp[i][0])),
			Open:      resp[i][1],
			High:      resp[i][2],
			Low:       resp[i][3],
			Close:     resp[i][4],
			Volume:    resp[i][5],
		}
	}
	return kline, nil
}

// PostFuturesOrder used to place two types of futures orders: limit and market
func (ku *Kucoin) PostFuturesOrder(ctx context.Context, arg *FuturesOrderParam) (string, error) {
	if arg.Leverage < 0.01 {
		return "", fmt.Errorf("%w must be greater than 0.01", errInvalidLeverage)
	}
	if arg.ClientOrderID == "" {
		return "", errInvalidClientOrderID
	}
	if arg.Side == "" {
		return "", fmt.Errorf("%w, empty order side", order.ErrSideIsInvalid)
	}
	if arg.Symbol.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	if arg.Stop != "" {
		if arg.StopPriceType == "" {
			return "", errInvalidStopPriceType
		}
		if arg.StopPrice <= 0 {
			return "", fmt.Errorf("%w, stopPrice is required", errInvalidPrice)
		}
	}
	switch arg.OrderType {
	case "limit", "":
		if arg.Price <= 0 {
			return "", fmt.Errorf("%w %f", errInvalidPrice, arg.Price)
		}
		if arg.Size <= 0 {
			return "", fmt.Errorf("%w, must be non-zero positive value", errInvalidSize)
		}
		if arg.VisibleSize < 0 {
			return "", fmt.Errorf("%w, visible size must be non-zero positive value", errInvalidSize)
		}
	case "market":
		if arg.Size <= 0 {
			return "", fmt.Errorf("%w, market size must be > 0", errInvalidSize)
		}
	default:
		return "", fmt.Errorf("%w, order type= %s", order.ErrTypeIsInvalid, arg.OrderType)
	}
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresPlaceOrderEPL, http.MethodPost, kucoinFuturesOrder, &arg, &resp)
}

// CancelFuturesOrder used to cancel single order previously placed including a stop order
func (ku *Kucoin) CancelFuturesOrder(ctx context.Context, orderID string) ([]string, error) {
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
	}{}

	if orderID == "" {
		return resp.CancelledOrderIDs, errors.New("orderID can't be empty")
	}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresCancelAnOrderEPL, http.MethodDelete, kucoinFuturesCancelOrder+orderID, nil, &resp)
}

// CancelAllFuturesOpenOrders used to cancel all futures order excluding stop orders
func (ku *Kucoin) CancelAllFuturesOpenOrders(ctx context.Context, symbol string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresLimitOrderMassCancelationEPL, http.MethodDelete, common.EncodeURLValues(kucoinFuturesOrder, params), nil, &resp)
}

// CancelAllFuturesStopOrders used to cancel all untriggered stop orders
func (ku *Kucoin) CancelAllFuturesStopOrders(ctx context.Context, symbol string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodDelete, common.EncodeURLValues(kucoinFuturesStopOrder, params), nil, &resp)
}

// GetFuturesOrders gets the user current futures order list
func (ku *Kucoin) GetFuturesOrders(ctx context.Context, status, symbol, side, orderType string, startAt, endAt time.Time) (*FutureOrdersResponse, error) {
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
	var resp *FutureOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveOrderListEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesOrder, params), nil, &resp)
}

// GetUntriggeredFuturesStopOrders gets the untriggered stop orders list
func (ku *Kucoin) GetUntriggeredFuturesStopOrders(ctx context.Context, symbol, side, orderType string, startAt, endAt time.Time) (*FutureOrdersResponse, error) {
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
	var resp *FutureOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesStopOrder, params), nil, &resp)
}

// GetFuturesRecentCompletedOrders gets list of recent 1000 orders in the last 24 hours
func (ku *Kucoin) GetFuturesRecentCompletedOrders(ctx context.Context) ([]FuturesOrder, error) {
	var resp []FuturesOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, kucoinFuturesRecentCompletedOrder, nil, &resp)
}

// GetFuturesOrderDetails gets single order details by order ID
func (ku *Kucoin) GetFuturesOrderDetails(ctx context.Context, orderID string) (*FuturesOrder, error) {
	var resp *FuturesOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, kucoinFuturesGetOrderDetails+orderID, nil, &resp)
}

// GetFuturesOrderDetailsByClientID gets single order details by client ID
func (ku *Kucoin) GetFuturesOrderDetailsByClientID(ctx context.Context, clientID string) (*FuturesOrder, error) {
	if clientID == "" {
		return nil, errors.New("clientID can't be empty")
	}
	params := url.Values{}
	params.Set("clientOid", clientID)
	var resp *FuturesOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesGetOrderDetailsByClientID, params), nil, &resp)
}

// GetFuturesFills gets list of recent fills
func (ku *Kucoin) GetFuturesFills(ctx context.Context, orderID, symbol, side, orderType string, startAt, endAt time.Time) (*FutureFillsResponse, error) {
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
	var resp *FutureFillsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveFillsEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesFills, params), nil, &resp)
}

// GetFuturesRecentFills gets list of 1000 recent fills in the last 24 hrs
func (ku *Kucoin) GetFuturesRecentFills(ctx context.Context) ([]FuturesFill, error) {
	var resp []FuturesFill
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRecentFillsEPL, http.MethodGet, kucoinFuturesRecentFills, nil, &resp)
}

// GetFuturesOpenOrderStats gets the total number and value of the all your active orders
func (ku *Kucoin) GetFuturesOpenOrderStats(ctx context.Context, symbol string) (*FuturesOpenOrderStats, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesOpenOrderStats
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesOpenOrderStats, params), nil, &resp)
}

// GetFuturesPosition gets the position details of a specified position
func (ku *Kucoin) GetFuturesPosition(ctx context.Context, symbol string) (*FuturesPosition, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesPosition
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesPosition, params), nil, &resp)
}

// GetFuturesPositionList gets the list of position with details
func (ku *Kucoin) GetFuturesPositionList(ctx context.Context) ([]FuturesPosition, error) {
	var resp []FuturesPosition
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrievePositionListEPL, http.MethodGet, kucoinFuturesPositionList, nil, &resp)
}

// SetAutoDepositMargin enable/disable of auto-deposit margin
func (ku *Kucoin) SetAutoDepositMargin(ctx context.Context, symbol string, status bool) (bool, error) {
	params := make(map[string]interface{})
	if symbol == "" {
		return false, errors.New("symbol can't be empty")
	}
	params["symbol"] = symbol
	params["status"] = status
	var resp bool
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodPost, kucoinFuturesSetAutoDeposit, params, &resp)
}

// AddMargin is used to add margin manually
func (ku *Kucoin) AddMargin(ctx context.Context, symbol, uniqueID string, margin float64) (*FuturesPosition, error) {
	params := make(map[string]interface{})
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params["symbol"] = symbol
	if uniqueID == "" {
		return nil, errors.New("uniqueID can't be empty")
	}
	params["bizNo"] = uniqueID
	if margin <= 0 {
		return nil, errors.New("margin can't be zero or negative")
	}
	params["margin"] = strconv.FormatFloat(margin, 'f', -1, 64)
	var resp *FuturesPosition
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodPost, kucoinFuturesAddMargin, params, &resp)
}

// GetFuturesRiskLimitLevel gets information about risk limit level of a specific contract
func (ku *Kucoin) GetFuturesRiskLimitLevel(ctx context.Context, symbol string) ([]FuturesRiskLimitLevel, error) {
	var resp []FuturesRiskLimitLevel
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, kucoinFuturesRiskLimitLevel+symbol, nil, &resp)
}

// FuturesUpdateRiskLmitLevel is used to adjustment the risk limit level
func (ku *Kucoin) FuturesUpdateRiskLmitLevel(ctx context.Context, symbol string, level int64) (bool, error) {
	params := make(map[string]interface{})
	if symbol == "" {
		return false, errors.New("symbol can't be empty")
	}
	params["symbol"] = symbol
	params["level"] = strconv.FormatInt(level, 10)
	var resp bool
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodPost, kucoinFuturesUpdateRiskLmitLevel, params, &resp)
}

// GetFuturesFundingHistory gets information about funding history
func (ku *Kucoin) GetFuturesFundingHistory(ctx context.Context, symbol string, offset, maxCount int64, reverse, forward bool, startAt, endAt time.Time) (*FuturesFundingHistoryResponse, error) {
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesFundingHistoryResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveFundingHistoryEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesFundingHistory, params), nil, &resp)
}

// GetFuturesAccountOverview gets future account overview
func (ku *Kucoin) GetFuturesAccountOverview(ctx context.Context, currency string) (FuturesAccount, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	resp := FuturesAccount{}
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveAccountOverviewEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesAccountOverview, params), nil, &resp)
}

// GetFuturesTransactionHistory gets future transaction history
func (ku *Kucoin) GetFuturesTransactionHistory(ctx context.Context, currency, txType string, offset, maxCount int64, forward bool, startAt, endAt time.Time) (*FuturesTransactionHistoryResponse, error) {
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
	params.Set("forward", strconv.FormatBool(forward))
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesTransactionHistoryResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresRetrieveTransactionHistoryEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesTransactionHistory, params), nil, &resp)
}

// CreateFuturesSubAccountAPIKey is used to create Futures APIs for sub-accounts
func (ku *Kucoin) CreateFuturesSubAccountAPIKey(ctx context.Context, ipWhitelist, passphrase, permission, remark, subName string) (*APIKeyDetail, error) {
	params := make(map[string]interface{})
	if ipWhitelist != "" {
		params["ipWhitelist"] = ipWhitelist
	}
	if passphrase == "" {
		return nil, errors.New("passphrase can't be empty")
	}
	params["passphrase"] = passphrase
	if permission != "" {
		params["permission"] = permission
	}
	if remark == "" {
		return nil, errors.New("remark can't be empty")
	}
	params["remark"] = remark
	if subName == "" {
		return nil, errors.New("subName can't be empty")
	}
	params["subName"] = subName
	var resp *APIKeyDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodPost, kucoinFuturesSubAccountAPI, params, &resp)
}

// GetFuturesDepositAddress gets deposit address for currency
func (ku *Kucoin) GetFuturesDepositAddress(ctx context.Context, currency string) (*DepositAddress, error) {
	if currency == "" {
		return nil, errors.New("currency can't be empty")
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp *DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesDepositAddress, params), nil, &resp)
}

// GetFuturesDepositsList gets deposits list
func (ku *Kucoin) GetFuturesDepositsList(ctx context.Context, currency, status string, startAt, endAt time.Time) (*FuturesDepositDetailsResponse, error) {
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
	var resp *FuturesDepositDetailsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesDepositsList, params), nil, &resp)
}

// GetFuturesWithdrawalLimit gets withdrawal limits for currency
func (ku *Kucoin) GetFuturesWithdrawalLimit(ctx context.Context, currency string) (*FuturesWithdrawalLimit, error) {
	if currency == "" {
		return nil, errors.New("currency can't be empty")
	}
	params := url.Values{}
	params.Set("currency", currency)
	var resp *FuturesWithdrawalLimit
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesWithdrawalLimit, params), nil, &resp)
}

// GetFuturesWithdrawalList gets withdrawal list
func (ku *Kucoin) GetFuturesWithdrawalList(ctx context.Context, currency, status string, startAt, endAt time.Time) (*FuturesWithdrawalsListResponse, error) {
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
	var resp *FuturesWithdrawalsListResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesWithdrawalList, params), nil, &resp)
}

// CancelFuturesWithdrawal is used to cancel withdrawal request of only PROCESSING status
func (ku *Kucoin) CancelFuturesWithdrawal(ctx context.Context, withdrawalID string) (bool, error) {
	var resp bool
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodDelete, kucoinFuturesCancelWithdrawal+withdrawalID, nil, &resp)
}

// TransferFuturesFundsToMainAccount helps in transferring funds from futures to main/trade account
func (ku *Kucoin) TransferFuturesFundsToMainAccount(ctx context.Context, amount float64, currency, recAccountType string) (*TransferRes, error) {
	params := make(map[string]interface{})
	if amount <= 0 {
		return nil, errors.New("amount can't be zero or negative")
	}
	params["amount"] = amount
	if currency == "" {
		return nil, errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if recAccountType == "" {
		return nil, errors.New("recAccountType can't be empty")
	}
	params["recAccountType"] = recAccountType
	var resp *TransferRes
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodPost, kucoinFuturesTransferFundtoMainAccount, params, &resp)
}

// TransferFundsToFuturesAccount helps in transferring funds from payee account to futures account
func (ku *Kucoin) TransferFundsToFuturesAccount(ctx context.Context, amount float64, currency, payAccountType string) error {
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
	resp := struct {
		Error
	}{}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodPost, kucoinFuturesTransferFundtoFuturesAccount, params, &resp)
}

// GetFuturesTransferOutList gets list of transfer out
func (ku *Kucoin) GetFuturesTransferOutList(ctx context.Context, currency, status string, startAt, endAt time.Time) (*TransferListsResponse, error) {
	if currency == "" {
		return nil, errors.New("currency can't be empty")
	}
	params := url.Values{}
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
	var resp *TransferListsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodGet, common.EncodeURLValues(kucoinFuturesTransferOutList, params), nil, &resp)
}

// CancelFuturesTransferOut is used to cancel transfer out request of only PROCESSING status
func (ku *Kucoin) CancelFuturesTransferOut(ctx context.Context, applyID string) error {
	if applyID == "" {
		return errors.New("applyID can't be empty")
	}
	params := url.Values{}
	params.Set("applyId", applyID)
	resp := struct {
		Error
	}{}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, defaultFuturesEPL, http.MethodDelete, common.EncodeURLValues(kucoinFuturesCancelTransferOut, params), nil, &resp)
}

func processFuturesOB(ob [][2]float64) []orderbook.Item {
	o := make([]orderbook.Item, len(ob))
	for x := range ob {
		o[x] = orderbook.Item{
			Price:  ob[x][0],
			Amount: ob[x][1],
		}
	}
	return o
}

func constructFuturesOrderbook(o *futuresOrderbookResponse) (*Orderbook, error) {
	var (
		s   Orderbook
		err error
	)
	s.Bids = processFuturesOB(o.Bids)
	if err != nil {
		return nil, err
	}
	s.Asks = processFuturesOB(o.Asks)
	if err != nil {
		return nil, err
	}
	s.Sequence = o.Sequence
	s.Time = o.Time.Time()
	return &s, err
}
