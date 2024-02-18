package kucoin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Kucoin is the overarching type across this package
type Kucoin struct {
	exchange.Base
	obm *orderbookManager
}

var locker sync.Mutex

const (
	kucoinAPIURL        = "https://api.kucoin.com/api"
	kucoinAPIKeyVersion = "2"
)

// GetSymbols gets pairs details on the exchange
// For market details see endpoint: https://www.kucoin.com/docs/rest/spot-trading/market-data/get-market-list
func (ku *Kucoin) GetSymbols(ctx context.Context, market string) ([]SymbolInfo, error) {
	params := url.Values{}
	if market != "" {
		params.Set("market", market)
	}
	var resp []SymbolInfo
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v2/symbols", params), &resp)
}

// GetTicker gets pair ticker information
func (ku *Kucoin) GetTicker(ctx context.Context, pair string) (*Ticker, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var resp *Ticker
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v1/market/orderbook/level1", params), &resp)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTickers gets all trading pair ticker information including 24h volume
func (ku *Kucoin) GetTickers(ctx context.Context) (*TickersResponse, error) {
	var resp *TickersResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, "/v1/market/allTickers", &resp)
}

// Get24hrStats get the statistics of the specified pair in the last 24 hours
func (ku *Kucoin) Get24hrStats(ctx context.Context, pair string) (*Stats24hrs, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var resp *Stats24hrs
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v1/market/stats", params), &resp)
}

// GetMarketList get the transaction currency for the entire trading market
func (ku *Kucoin) GetMarketList(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, "/v1/markets", &resp)
}

func processOB(ob [][2]types.Number) []orderbook.Item {
	o := make([]orderbook.Item, len(ob))
	for x := range ob {
		o[x].Amount = ob[x][1].Float64()
		o[x].Price = ob[x][0].Float64()
	}
	return o
}

func constructOrderbook(o *orderbookResponse) (*Orderbook, error) {
	var (
		s = Orderbook{
			Bids: processOB(o.Bids),
			Asks: processOB(o.Asks),
			Time: o.Time.Time(),
		}
	)
	if o.Sequence != "" {
		var err error
		s.Sequence, err = strconv.ParseInt(o.Sequence, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return &s, nil
}

// GetPartOrderbook20 gets orderbook for a specified pair with depth 20
func (ku *Kucoin) GetPartOrderbook20(ctx context.Context, pair string) (*Orderbook, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var o *orderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v1/market/orderbook/level2_20", params), &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetPartOrderbook100 gets orderbook for a specified pair with depth 100
func (ku *Kucoin) GetPartOrderbook100(ctx context.Context, pair string) (*Orderbook, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var o *orderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v1/market/orderbook/level2_100", params), &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetOrderbook gets full orderbook for a specified pair
func (ku *Kucoin) GetOrderbook(ctx context.Context, pair string) (*Orderbook, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var o *orderbookResponse
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveFullOrderbookEPL, http.MethodGet, common.EncodeURLValues("/v3/market/orderbook/level2", params), nil, &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetTradeHistory gets trade history of the specified pair
func (ku *Kucoin) GetTradeHistory(ctx context.Context, pair string) ([]Trade, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var resp []Trade
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v1/market/histories", params), &resp)
}

// GetKlines gets kline of the specified pair
func (ku *Kucoin) GetKlines(ctx context.Context, pair, period string, start, end time.Time) ([]Kline, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	if period == "" {
		return nil, errors.New("period can not be empty")
	}
	if !common.StringDataContains(validPeriods, period) {
		return nil, errors.New("invalid period")
	}
	params.Set("type", period)
	if !start.IsZero() {
		params.Set("startAt", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		params.Set("endAt", strconv.FormatInt(end.Unix(), 10))
	}
	var resp [][7]types.Number
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v1/market/candles", params), &resp)
	if err != nil {
		return nil, err
	}
	klines := make([]Kline, len(resp))
	for i := range resp {
		klines[i].StartTime = time.Unix(resp[i][0].Int64(), 0)
		klines[i].Open = resp[i][1].Float64()
		klines[i].Close = resp[i][2].Float64()
		klines[i].High = resp[i][3].Float64()
		klines[i].Low = resp[i][4].Float64()
		klines[i].Volume = resp[i][5].Float64()
		klines[i].Amount = resp[i][6].Float64()
	}
	return klines, nil
}

// GetCurrencies gets list of currencies
func (ku *Kucoin) GetCurrencies(ctx context.Context) ([]Currency, error) {
	var resp []Currency
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, "/v1/currencies", &resp)
}

// GetCurrencyDetail gets currency detail using currency code and chain information.
func (ku *Kucoin) GetCurrencyDetail(ctx context.Context, ccy, chain string) (*CurrencyDetail, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *CurrencyDetail
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v2/currencies/"+strings.ToUpper(ccy), params), &resp)
}

// GetFiatPrice gets fiat prices of currencies, default base currency is USD
func (ku *Kucoin) GetFiatPrice(ctx context.Context, base, currencies string) (map[string]string, error) {
	params := url.Values{}
	if base != "" {
		params.Set("base", base)
	}
	if currencies != "" {
		params.Set("currencies", currencies)
	}
	var resp map[string]string
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues("/v1/prices", params), &resp)
}

// GetLeveragedTokenInfo returns leveraged token information
func (ku *Kucoin) GetLeveragedTokenInfo(ctx context.Context, ccy string) ([]LeveragedTokenInfo, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp []LeveragedTokenInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("v3/etf/info", params), nil, &resp)
}

// GetMarkPrice gets index price of the specified pair
func (ku *Kucoin) GetMarkPrice(ctx context.Context, pair string) (*MarkPrice, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *MarkPrice
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, "/v1/mark-price/"+pair+"/current", &resp)
}

// GetMarginConfiguration gets configure info of the margin
func (ku *Kucoin) GetMarginConfiguration(ctx context.Context) (*MarginConfiguration, error) {
	var resp *MarginConfiguration
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, "/v1/margin/config", &resp)
}

// GetMarginAccount gets configure info of the margin
func (ku *Kucoin) GetMarginAccount(ctx context.Context) (*MarginAccounts, error) {
	var resp *MarginAccounts
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/margin/account", nil, &resp)
}

// GetMarginRiskLimit gets cross/isolated margin risk limit, default model is cross margin
func (ku *Kucoin) GetMarginRiskLimit(ctx context.Context, marginModel string) ([]MarginRiskLimit, error) {
	params := url.Values{}
	if marginModel != "" {
		params.Set("marginModel", marginModel)
	}
	var resp []MarginRiskLimit
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveMarginAccountEPL, http.MethodGet, common.EncodeURLValues("/v1/risk/limit/strategy", params), nil, &resp)
}

// PostMarginBorrowOrder used to post borrow order
func (ku *Kucoin) PostMarginBorrowOrder(ctx context.Context, arg *MarginBorrowParam) (*BorrowAndRepaymentOrderResp, error) {
	if arg == nil || *arg == (MarginBorrowParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.TimeInForce == "" {
		return nil, errTimeInForceRequired
	}
	if arg.Size <= 0 {
		return nil, fmt.Errorf("%w , size = %f", order.ErrAmountBelowMin, arg.Size)
	}
	var resp *BorrowAndRepaymentOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "/v3/margin/borrow", arg, &resp)
}

// GetMarginBorrowingHistory retrieves the borrowing orders for cross and isolated margin accounts
func (ku *Kucoin) GetMarginBorrowingHistory(ctx context.Context, ccy currency.Code, isIsolated bool,
	symbol currency.Pair, orderNo string,
	startTime, endTime time.Time,
	currentPage, pageSize int64) ([]BorrowRepayDetailItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if isIsolated {
		params.Set("isIsonalted", "true")
	}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if orderNo != "" {
		params.Set("orderNo", orderNo)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp []BorrowRepayDetailItem
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/borrow", params), nil, &resp)
}

// PostRepayment used to initiate an application for the repayment of cross or isolated margin borrowing.
func (ku *Kucoin) PostRepayment(ctx context.Context, arg *RepayParam) (*BorrowAndRepaymentOrderResp, error) {
	if arg == nil || *arg == (RepayParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Size <= 0 {
		return nil, fmt.Errorf("%w , size = %f", order.ErrAmountBelowMin, arg.Size)
	}
	var resp *BorrowAndRepaymentOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "/v3/margin/repay", arg, &resp)
}

// GetRepaymentHistory retrieves the repayment orders for cross and isolated margin accounts.
func (ku *Kucoin) GetRepaymentHistory(ctx context.Context, ccy currency.Code, isIsolated bool,
	symbol currency.Pair, orderNo string,
	startTime, endTime time.Time,
	currentPage, pageSize int64) ([]BorrowRepayDetailItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if isIsolated {
		params.Set("isIsonalted", "true")
	}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if orderNo != "" {
		params.Set("orderNo", orderNo)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp []BorrowRepayDetailItem
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/repay", params), nil, &resp)
}

// GetBorrowOrder gets borrow order information
func (ku *Kucoin) GetBorrowOrder(ctx context.Context, orderID string) (*BorrowOrder, error) {
	if orderID == "" {
		return nil, errors.New("empty orderID")
	}
	var resp *BorrowOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/margin/borrow?orderId="+orderID, nil, &resp)
}

// GetIsolatedMarginPairConfig get the current isolated margin trading pair configuration
func (ku *Kucoin) GetIsolatedMarginPairConfig(ctx context.Context) ([]IsolatedMarginPairConfig, error) {
	var resp []IsolatedMarginPairConfig
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/isolated/symbols", nil, &resp)
}

// GetIsolatedMarginAccountInfo get all isolated margin accounts of the current user
func (ku *Kucoin) GetIsolatedMarginAccountInfo(ctx context.Context, balanceCurrency string) (*IsolatedMarginAccountInfo, error) {
	params := url.Values{}
	if balanceCurrency != "" {
		params.Set("balanceCurrency", balanceCurrency)
	}
	var resp *IsolatedMarginAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/isolated/accounts", params), nil, &resp)
}

// GetSingleIsolatedMarginAccountInfo get single isolated margin accounts of the current user
func (ku *Kucoin) GetSingleIsolatedMarginAccountInfo(ctx context.Context, symbol string) (*AssetInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *AssetInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/isolated/account/"+symbol, nil, &resp)
}

// GetCurrentServerTime gets the server time
func (ku *Kucoin) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Timestamp convert.ExchangeTime `json:"data"`
		Error
	}{}
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, "/v1/timestamp", &resp)
	if err != nil {
		return time.Time{}, err
	}
	return resp.Timestamp.Time(), nil
}

// GetServiceStatus gets the service status
func (ku *Kucoin) GetServiceStatus(ctx context.Context) (*ServiceStatus, error) {
	var resp *ServiceStatus
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, "/v1/status", &resp)
}

// PostOrder used to place two types of orders: limit and market
// Note: use this only for SPOT trades
func (ku *Kucoin) PostOrder(ctx context.Context, arg *SpotOrderParam) (string, error) {
	if arg.ClientOrderID == "" {
		// NOTE: 128 bit max length character string. UUID recommended.
		return "", errInvalidClientOrderID
	}
	if arg.Side == "" {
		return "", order.ErrSideIsInvalid
	}
	if arg.Symbol.IsEmpty() {
		return "", fmt.Errorf("%w, empty symbol", currency.ErrCurrencyPairEmpty)
	}
	switch arg.OrderType {
	case "limit", "":
		if arg.Price <= 0 {
			return "", fmt.Errorf("%w, price =%.3f", errInvalidPrice, arg.Price)
		}
		if arg.Size <= 0 {
			return "", errInvalidSize
		}
		if arg.VisibleSize < 0 {
			return "", fmt.Errorf("%w, visible size must be non-zero positive value", errInvalidSize)
		}
	case "market":
		if arg.Size == 0 && arg.Funds == 0 {
			return "", errSizeOrFundIsRequired
		}
	default:
		return "", fmt.Errorf("%w %s", order.ErrTypeIsInvalid, arg.OrderType)
	}
	var resp struct {
		Data struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
		Error
	}
	return resp.Data.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, "/v1/orders", &arg, &resp)
}

// PostMarginOrder used to place two types of margin orders: limit and market
func (ku *Kucoin) PostMarginOrder(ctx context.Context, arg *MarginOrderParam) (*PostMarginOrderResp, error) {
	if arg.ClientOrderID == "" {
		return nil, errInvalidClientOrderID
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Symbol.IsEmpty() {
		return nil, fmt.Errorf("%w, empty symbol", currency.ErrCurrencyPairEmpty)
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	switch arg.OrderType {
	case "limit", "":
		if arg.Price <= 0 {
			return nil, fmt.Errorf("%w, price=%.3f", errInvalidPrice, arg.Price)
		}
		if arg.Size <= 0 {
			return nil, errInvalidSize
		}
		if arg.VisibleSize < 0 {
			return nil, fmt.Errorf("%w, visible size must be non-zero positive value", errInvalidSize)
		}
	case "market":
		sum := arg.Size + arg.Funds
		if sum <= 0 || (sum != arg.Size && sum != arg.Funds) {
			return nil, fmt.Errorf("%w, either 'size' or 'funds' has to be set, but not both", errSizeOrFundIsRequired)
		}
	default:
		return nil, fmt.Errorf("%w %s", order.ErrTypeIsInvalid, arg.OrderType)
	}
	resp := struct {
		PostMarginOrderResp
		Error
	}{}
	return &resp.PostMarginOrderResp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginOrdersEPL, http.MethodPost, "/v1/margin/order", &arg, &resp)
}

// PostBulkOrder used to place 5 orders at the same time. The order type must be a limit order of the same symbol
// Note: it supports only SPOT trades
// Note: To check if order was posted successfully, check status field in response
func (ku *Kucoin) PostBulkOrder(ctx context.Context, symbol string, orderList []OrderRequest) ([]PostBulkOrderResp, error) {
	if symbol == "" {
		return nil, errors.New("symbol can not be empty")
	}
	for i := range orderList {
		if orderList[i].ClientOID == "" {
			return nil, errors.New("clientOid can not be empty")
		}
		if orderList[i].Side == "" {
			return nil, errors.New("side can not be empty")
		}
		if orderList[i].Price <= 0 {
			return nil, errors.New("price must be positive")
		}
		if orderList[i].Size <= 0 {
			return nil, errors.New("size must be positive")
		}
	}
	arg := &struct {
		Symbol    string         `json:"symbol"`
		OrderList []OrderRequest `json:"orderList"`
	}{
		Symbol:    symbol,
		OrderList: orderList,
	}
	resp := &struct {
		Data []PostBulkOrderResp `json:"data"`
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeBulkOrdersEPL, http.MethodPost, "/v1/orders/multi", arg, &resp)
}

// CancelSingleOrder used to cancel single order previously placed
func (ku *Kucoin) CancelSingleOrder(ctx context.Context, orderID string) ([]string, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOrderEPL, http.MethodDelete, "/v1/orders/"+orderID, nil, &resp)
}

// CancelOrderByClientOID used to cancel order via the clientOid
func (ku *Kucoin) CancelOrderByClientOID(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	var resp *CancelOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, "/v1/order/client-order/"+orderID, nil, &resp)
}

// CancelAllOpenOrders used to cancel all order based upon the parameters passed
func (ku *Kucoin) CancelAllOpenOrders(ctx context.Context, symbol, tradeType string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelAllOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v1/orders", params), nil, &resp)
}

// ListOrders gets the user order list
func (ku *Kucoin) ListOrders(ctx context.Context, status, symbol, side, orderType, tradeType string, startAt, endAt time.Time) (*OrdersListResponse, error) {
	params := fillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if status != "" {
		params.Set("status", status)
	}
	var resp *OrdersListResponse
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, listOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/orders", params), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func fillParams(symbol, side, orderType, tradeType string, startAt, endAt time.Time) url.Values {
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
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	return params
}

// GetRecentOrders get orders in the last 24 hours.
func (ku *Kucoin) GetRecentOrders(ctx context.Context) ([]OrderDetail, error) {
	var resp []OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/limit/orders", nil, &resp)
}

// GetOrderByID get a single order info by order ID
func (ku *Kucoin) GetOrderByID(ctx context.Context, orderID string) (*OrderDetail, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	var resp *OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/orders/"+orderID, nil, &resp)
}

// GetOrderByClientSuppliedOrderID get a single order info by client order ID
func (ku *Kucoin) GetOrderByClientSuppliedOrderID(ctx context.Context, clientOID string) (*OrderDetail, error) {
	if clientOID == "" {
		return nil, errors.New("client order ID can not be empty")
	}
	var resp *OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/order/client-order/"+clientOID, nil, &resp)
}

// GetFills get fills
func (ku *Kucoin) GetFills(ctx context.Context, orderID, symbol, side, orderType, tradeType string, startAt, endAt time.Time) (*ListFills, error) {
	params := fillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	var resp *ListFills
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, listFillsEPL, http.MethodGet, common.EncodeURLValues("/v1/fills", params), nil, &resp)
}

// GetRecentFills get a list of 1000 fills in last 24 hours
func (ku *Kucoin) GetRecentFills(ctx context.Context) ([]Fill, error) {
	var resp []Fill
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/limit/fills", nil, &resp)
}

// PostStopOrder used to place two types of stop orders: limit and market
func (ku *Kucoin) PostStopOrder(ctx context.Context, clientOID, side, symbol, orderType, remark, stop, stp, tradeType, timeInForce string, size, price, stopPrice, cancelAfter, visibleSize, funds float64, postOnly, hidden, iceberg bool) (string, error) {
	if clientOID == "" {
		return "", errors.New("clientOid can not be empty")
	}
	if side == "" {
		return "", errors.New("side can not be empty")
	}
	if symbol == "" {
		return "", fmt.Errorf("%w, empty symbol", currency.ErrCurrencyPairEmpty)
	}
	params := make(map[string]interface{})
	params["clientOid"] = clientOID
	params["side"] = side
	params["symbol"] = symbol
	if remark != "" {
		params["remark"] = remark
	}
	if stop != "" {
		params["stop"] = stop
		if stopPrice <= 0 {
			return "", errors.New("stopPrice is required")
		}
		params["stopPrice"] = strconv.FormatFloat(stopPrice, 'f', -1, 64)
	}
	if stp != "" {
		params["stp"] = stp
	}
	if tradeType != "" {
		params["tradeType"] = tradeType
	}
	orderType = strings.ToLower(orderType)
	switch orderType {
	case "limit", "":
		if price <= 0 {
			return "", errors.New("price is required")
		}
		params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
		if size <= 0 {
			return "", errors.New("size can not be zero or negative")
		}
		params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		if timeInForce != "" {
			params["timeInForce"] = timeInForce
		}
		if cancelAfter > 0 && timeInForce == "GTT" {
			params["cancelAfter"] = strconv.FormatFloat(cancelAfter, 'f', -1, 64)
		}
		params["postOnly"] = postOnly
		params["hidden"] = hidden
		params["iceberg"] = iceberg
		if visibleSize > 0 {
			params["visibleSize"] = strconv.FormatFloat(visibleSize, 'f', -1, 64)
		}
	case "market":
		switch {
		case size > 0:
			params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		case funds > 0:
			params["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
		default:
			return "", errSizeOrFundIsRequired
		}
	default:
		return "", fmt.Errorf("%w, order type: %s", order.ErrTypeIsInvalid, orderType)
	}
	if orderType != "" {
		params["type"] = orderType
	}
	resp := struct {
		OrderID string `json:"orderId"`
		Error
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, "/v1/stop-order", params, &resp)
}

// CancelStopOrder used to cancel single stop order previously placed
func (ku *Kucoin) CancelStopOrder(ctx context.Context, orderID string) ([]string, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	resp := struct {
		Data []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, "/v1/stop-order/"+orderID, nil, &resp)
}

// CancelStopOrders used to cancel all order based upon the parameters passed
func (ku *Kucoin) CancelStopOrders(ctx context.Context, symbol, tradeType, orderIDs string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	if orderIDs != "" {
		params.Set("orderIds", orderIDs)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, common.EncodeURLValues("/v1/stop-order/cancel", params), nil, &resp)
}

// GetStopOrder used to cancel single stop order previously placed
func (ku *Kucoin) GetStopOrder(ctx context.Context, orderID string) (*StopOrder, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	resp := struct {
		StopOrder
		Error
	}{}
	return &resp.StopOrder, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/stop-order/"+orderID, nil, &resp)
}

// ListStopOrders get all current untriggered stop orders
func (ku *Kucoin) ListStopOrders(ctx context.Context, symbol, side, orderType, tradeType, orderIDs string, startAt, endAt time.Time, currentPage, pageSize int64) (*StopOrderListResponse, error) {
	params := fillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if orderIDs != "" {
		params.Set("orderIds", orderIDs)
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *StopOrderListResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/stop-order", params), nil, &resp)
}

// GetStopOrderByClientID get a stop order information via the clientOID
func (ku *Kucoin) GetStopOrderByClientID(ctx context.Context, symbol, clientOID string) ([]StopOrder, error) {
	if clientOID == "" {
		return nil, errors.New("clientOID can not be empty")
	}
	params := url.Values{}
	params.Set("clientOid", clientOID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []StopOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/stop-order/queryOrderByClientOid", params), nil, &resp)
}

// CancelStopOrderByClientID used to cancel a stop order via the clientOID.
func (ku *Kucoin) CancelStopOrderByClientID(ctx context.Context, symbol, clientOID string) (*CancelOrderResponse, error) {
	if clientOID == "" {
		return nil, errors.New("clientOID can not be empty")
	}
	params := url.Values{}
	params.Set("clientOid", clientOID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *CancelOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, common.EncodeURLValues("/v1/stop-order/cancelOrderByClientOid", params), nil, &resp)
}

// CreateSubUser creates a new sub-user for the account.
func (ku *Kucoin) CreateSubUser(ctx context.Context, subAccountName, password, remarks, access string) (*SubAccount, error) {
	if regexp.MustCompile("^[a-zA-Z0-9]{7-32}$").MatchString(subAccountName) {
		return nil, errors.New("invalid sub-account name")
	}
	if regexp.MustCompile("^[a-zA-Z0-9]{7-24}$").MatchString(password) {
		return nil, errInvalidPassPhraseInstance
	}
	params := make(map[string]interface{})
	params["subName"] = subAccountName
	params["password"] = password
	if remarks != "" {
		params["remarks"] = remarks
	}
	if access != "" {
		params["access"] = access
	}
	var resp *SubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "/v2/sub/user/created", params, &resp)
}

// GetSubAccountSpotAPIList used to obtain a list of Spot APIs pertaining to a sub-account.
func (ku *Kucoin) GetSubAccountSpotAPIList(ctx context.Context, subAccountName, apiKeys string) (*SubAccountResponse, error) {
	if subAccountRegExp.MatchString(subAccountName) {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subName", subAccountName)
	if apiKeys != "" {
		params.Set("apiKey", apiKeys)
	}
	var resp *SubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/sub/api-key", params), nil, &resp)
}

// CreateSpotAPIsForSubAccount can be used to create Spot APIs for sub-accounts.
func (ku *Kucoin) CreateSpotAPIsForSubAccount(ctx context.Context, arg *SpotAPISubAccountParams) (*SpotAPISubAccount, error) {
	if subAccountRegExp.MatchString(arg.SubAccountName) {
		return nil, errInvalidSubAccountName
	}
	if subAccountPassphraseRegExp.MatchString(arg.Passphrase) {
		return nil, fmt.Errorf("%w, must contain 7-32 characters. cannot contain any spaces", errInvalidPassPhraseInstance)
	}
	var resp *SpotAPISubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "/v1/sub/api-key", &arg, &resp)
}

// ModifySubAccountSpotAPIs modifies sub-account Spot APIs.
func (ku *Kucoin) ModifySubAccountSpotAPIs(ctx context.Context, arg *SpotAPISubAccountParams) (*SpotAPISubAccount, error) {
	if subAccountRegExp.MatchString(arg.SubAccountName) {
		return nil, errInvalidSubAccountName
	}
	if subAccountPassphraseRegExp.MatchString(arg.Passphrase) {
		return nil, fmt.Errorf("%w, must contain 7-32 characters. cannot contain any spaces", errInvalidPassPhraseInstance)
	}
	var resp *SpotAPISubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPut, "/v1/sub/api-key/update", &arg, &resp)
}

// DeleteSubAccountSpotAPI delete sub-account Spot APIs.
func (ku *Kucoin) DeleteSubAccountSpotAPI(ctx context.Context, apiKey, subAccountName string) (*DeleteSubAccountResponse, error) {
	if subAccountRegExp.MatchString(subAccountName) {
		return nil, errInvalidSubAccountName
	}
	if apiKey == "" {
		return nil, errors.New("apiKey is required")
	}
	params := url.Values{}
	params.Set("apiKey", apiKey)
	params.Set("subName", subAccountName)
	var resp *DeleteSubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, common.EncodeURLValues("/v1/sub/api-key", params), nil, &resp)
}

// GetUserInfoOfAllSubAccounts get the user info of all sub-users via this interface.
func (ku *Kucoin) GetUserInfoOfAllSubAccounts(ctx context.Context) (*SubAccountResponse, error) {
	var resp *SubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v2/sub/user", nil, &resp)
}

// GetPaginatedListOfSubAccounts to retrieve a paginated list of sub-accounts. Pagination is required.
func (ku *Kucoin) GetPaginatedListOfSubAccounts(ctx context.Context, currentPage, pageSize int64) (*SubAccountResponse, error) {
	params := url.Values{}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	if currentPage > 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	var resp *SubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/sub/api-key/update", params), nil, &resp)
}

// GetAllAccounts get all accounts
// accountType possible values are main、trade、margin、trade_hf
func (ku *Kucoin) GetAllAccounts(ctx context.Context, ccy, accountType string) ([]AccountInfo, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if accountType != "" {
		params.Set("type", accountType)
	}
	var resp []AccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts", params), nil, &resp)
}

// GetAccountDetail get information of single account
func (ku *Kucoin) GetAccountDetail(ctx context.Context, accountID string) (*AccountInfo, error) {
	if accountID == "" {
		return nil, errAccountIDMissing
	}
	var resp *AccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/accounts/"+accountID, nil, &resp)
}

// GetCrossMarginAccountsDetail retrieves the info of the cross margin account.
func (ku *Kucoin) GetCrossMarginAccountsDetail(ctx context.Context, quoteCurrency, queryType string) (*CrossMarginAccountDetail, error) {
	params := url.Values{}
	if quoteCurrency != "" {
		params.Set("quoteCurrency", quoteCurrency)
	}
	if queryType != "" {
		params.Set("queryType", queryType)
	}
	var resp *CrossMarginAccountDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/accounts", params), nil, &resp)
}

// GetIsolatedMarginAccountDetail to get the info of the isolated margin account.
func (ku *Kucoin) GetIsolatedMarginAccountDetail(ctx context.Context, symbol, queryCurrency, queryType string) (*IsolatedMarginAccountDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if queryCurrency != "" {
		params.Set("quoteCurrency", queryCurrency)
	}
	if queryType != "" {
		params.Set("queryType", queryType)
	}
	var resp *IsolatedMarginAccountDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v3/isolated/accounts", params), nil, &resp)
}

// GetFuturesAccountDetail retrieves futures account detail information
func (ku *Kucoin) GetFuturesAccountDetail(ctx context.Context, ccy string) (*FuturesAccountOverview, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp *FuturesAccountOverview
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/account-overview", params), nil, &resp)
}

// retrieves all sub-account informations
func (ku *Kucoin) GetSubAccounts(ctx context.Context, subUserID string, includeBaseAmount bool) (*SubAccounts, error) {
	if subUserID == "" {
		return nil, errors.New("sub users ID is required")
	}
	params := url.Values{}
	if includeBaseAmount {
		params.Set("includeBaseAmount", "true")
	} else {
		params.Set("includeBaseAmount", "false")
	}
	var resp *SubAccounts
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/sub-accounts/"+subUserID, params), nil, &resp)
}

// GetAllFuturesSubAccountBalances retrieves all futures subaccount balances
func (ku *Kucoin) GetAllFuturesSubAccountBalances(ctx context.Context, ccy string) (*FuturesSubAccountBalance, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp *FuturesSubAccountBalance
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/account-overview-all", params), nil, &resp)
}

func populateParams(ccy, direction, bizType string, lastID, limit int64, startTime, endTime time.Time) url.Values {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if bizType != "" {
		params.Set("bizType", bizType)
	}
	if lastID != 0 {
		params.Set("lastId", strconv.FormatInt(lastID, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return params
}

// GetAccountLedgers retrieves the transaction records from all types of your accounts, supporting inquiry of various currencies.
// bizType possible values: 'DEPOSIT' -deposit, 'WITHDRAW' -withdraw, 'TRANSFER' -transfer, 'SUB_TRANSFER' -subaccount transfer,'TRADE_EXCHANGE' -trade, 'MARGIN_EXCHANGE' -margin trade, 'KUCOIN_BONUS' -bonus
func (ku *Kucoin) GetAccountLedgers(ctx context.Context, ccy, direction, bizType string, startAt, endAt time.Time) (*AccountLedgerResponse, error) {
	params := populateParams(ccy, direction, bizType, 0, 0, time.Time{}, time.Time{})
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *AccountLedgerResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveAccountLedgerEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts/ledgers", params), nil, &resp)
}

// GetAccountLedgersHFTrade returns all transfer (in and out) records in high-frequency trading account and supports multi-coin queries.
// The query results are sorted in descending order by createdAt and id.
func (ku *Kucoin) GetAccountLedgersHFTrade(ctx context.Context, ccy, direction, bizType string, lastID, limit int64, startTime, endTime time.Time) ([]LedgerInfo, error) {
	params := populateParams(ccy, direction, bizType, lastID, limit, startTime, endTime)
	var resp []LedgerInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/accounts/ledgers", params), nil, &resp)
}

// GetAccountLedgerHFMargin returns all transfer (in and out) records in high-frequency margin trading account and supports multi-coin queries.
func (ku *Kucoin) GetAccountLedgerHFMargin(ctx context.Context, ccy, direction, bizType string, lastID, limit int64, startTime, endTime time.Time) ([]LedgerInfo, error) {
	params := populateParams(ccy, direction, bizType, lastID, limit, startTime, endTime)
	var resp []LedgerInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/account/ledgers", params), nil, &resp)
}

// GetFuturesAccountLedgers If there are open positions, the status of the first page returned will be Pending,
// indicating the realised profit and loss in the current 8-hour settlement period.
// Type RealisedPNL-Realised profit and loss, Deposit-Deposit, Withdrawal-withdraw, Transferin-Transfer in, TransferOut-Transfer out
func (ku *Kucoin) GetFuturesAccountLedgers(ctx context.Context, ccy string, forward bool, startAt, endAt time.Time, offset, maxCount int64) (*FuturesLedgerInfo, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if forward {
		params.Set("forward", "true")
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesLedgerInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/transaction-history", params), nil, &resp)
}

// GetAllSubAccountsInfoV1 retrieves the user info of all sub-account via this interface.
func (ku *Kucoin) GetAllSubAccountsInfoV1(ctx context.Context) ([]SubAccount, error) {
	var resp []SubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/sub/user", nil, &resp)
}

// GetAllSubAccountsInfoV2 retrieves list of sub-accounts.
func (ku *Kucoin) GetAllSubAccountsInfoV2(ctx context.Context, currentPage, pageSize int64) (*SubAccountV2Response, error) {
	params := url.Values{}
	if currentPage > 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *SubAccountV2Response
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v2/sub/user", nil, &resp)
}

// GetAccountSummaryInformation this can be used to obtain account summary information.
func (ku *Kucoin) GetAccountSummaryInformation(ctx context.Context) (*AccountSummaryInformation, error) {
	var resp *AccountSummaryInformation
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v2/user-info", nil, &resp)
}

// GetAggregatedSubAccountBalance get the account info of all sub-users
func (ku *Kucoin) GetAggregatedSubAccountBalance(ctx context.Context) ([]SubAccountInfo, error) {
	var resp []SubAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/sub-accounts", nil, &resp)
}

// GetAllSubAccountsBalanceV2 retrieves sub-account balance information through the V2 API
func (ku *Kucoin) GetAllSubAccountsBalanceV2(ctx context.Context) (*SubAccountBalanceV2, error) {
	var resp *SubAccountBalanceV2
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v2/sub-accounts", nil, &resp)
}

// GetPaginatedSubAccountInformation this endpoint can be used to get paginated sub-account information. Pagination is required.
func (ku *Kucoin) GetPaginatedSubAccountInformation(ctx context.Context, currentPage, pageSize int64) ([]SubAccountInfo, error) {
	params := url.Values{}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp []SubAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/sub-accounts", params), nil, &resp)
}

// GetTransferableBalance get the transferable balance of a specified account
func (ku *Kucoin) GetTransferableBalance(ctx context.Context, ccy, accountType, tag string) (*TransferableBalanceInfo, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if accountType == "" {
		return nil, errors.New("accountType can not be empty")
	}
	params := url.Values{}
	params.Set("currency", ccy)
	params.Set("type", accountType)
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp *TransferableBalanceInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts/transferable", params), nil, &resp)
}

// GetUniversalTransfer support transfer between master and sub accounts (only applicable to master account APIKey).
func (ku *Kucoin) GetUniversalTransfer(ctx context.Context, arg *UniversalTransferParam) (string, error) {
	if arg == nil || *arg == (UniversalTransferParam{}) {
		return "", common.ErrNilPointer
	}
	if arg.ClientSuppliedOrderID == "" {
		return "", errInvalidClientOrderID
	}
	if arg.Amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	if arg.FromAccountType == "" {
		return "", fmt.Errorf("%w, empty fromAccountType", errAccountTypeMissing)
	}
	if arg.TransferType == "" {
		return "", fmt.Errorf("%w, transfer type is empty", errTransferTypeMissing)
	}
	if arg.ToAccountType == "" {
		return "", fmt.Errorf("%w, toAccountType is empty", errAccountTypeMissing)
	}
	var resp string
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "v3/accounts/universal-transfer", arg, &resp)
}

// TransferMainToSubAccount used to transfer funds from main account to sub-account
func (ku *Kucoin) TransferMainToSubAccount(ctx context.Context, clientOID, ccy, amount, direction, accountType, subAccountType, subUserID string) (string, error) {
	if clientOID == "" {
		return "", errors.New("clientOID can not be empty")
	}
	if ccy == "" {
		return "", currency.ErrCurrencyPairEmpty
	}
	if amount == "" {
		return "", errors.New("amount can not be empty")
	}
	if direction == "" {
		return "", errors.New("direction can not be empty")
	}
	if subUserID == "" {
		return "", errors.New("subUserID can not be empty")
	}
	params := make(map[string]interface{})
	params["clientOid"] = clientOID
	params["currency"] = ccy
	params["amount"] = amount
	params["direction"] = direction
	if accountType != "" {
		params["accountType"] = accountType
	}
	if subAccountType != "" {
		params["subAccountType"] = subAccountType
	}
	params["subUserId"] = subUserID
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, masterSubUserTransferEPL, http.MethodPost, "/v2/accounts/sub-transfer", params, &resp)
}

// MakeInnerTransfer used to transfer funds between accounts internally
func (ku *Kucoin) MakeInnerTransfer(ctx context.Context, clientOID, ccy, from, to, amount, fromTag, toTag string) (string, error) {
	if clientOID == "" {
		return "", errors.New("clientOID can not be empty")
	}
	if ccy == "" {
		return "", currency.ErrCurrencyPairEmpty
	}
	if amount == "" {
		return "", errors.New("amount can not be empty")
	}
	if from == "" {
		return "", errors.New("from can not be empty")
	}
	if to == "" {
		return "", errors.New("to can not be empty")
	}
	params := make(map[string]interface{})
	params["clientOid"] = clientOID
	params["currency"] = ccy
	params["amount"] = amount
	params["from"] = from
	params["to"] = to
	if fromTag != "" {
		params["fromTag"] = fromTag
	}
	if toTag != "" {
		params["toTag"] = toTag
	}
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "/v2/accounts/inner-transfer", params, &resp)
}

// CreateDepositAddress create a deposit address for a currency you intend to deposit
func (ku *Kucoin) CreateDepositAddress(ctx context.Context, ccy, chain string) (*DepositAddress, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	if chain != "" {
		params["chain"] = chain
	}
	var resp *DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "/v1/deposit-addresses", params, &resp)
}

// GetDepositAddressesV2 get all deposit addresses for the currency you intend to deposit
func (ku *Kucoin) GetDepositAddressesV2(ctx context.Context, ccy string) ([]DepositAddress, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	var resp []DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v2/deposit-addresses", params), nil, &resp)
}

// GetDepositAddressV1 get a deposit address for the currency you intend to deposit
func (ku *Kucoin) GetDepositAddressV1(ctx context.Context, ccy, chain string) (*DepositAddress, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/deposit-addresses", params), nil, &resp)
}

// GetDepositList get deposit list items and sorted to show the latest first
// Status. Available value: PROCESSING, SUCCESS, and FAILURE
func (ku *Kucoin) GetDepositList(ctx context.Context, ccy, status string, startAt, endAt time.Time) (*DepositResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
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
	var resp *DepositResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveDepositListEPL, http.MethodGet, common.EncodeURLValues("/v1/deposits", params), nil, &resp)
}

// GetHistoricalDepositList get historical deposit list items
func (ku *Kucoin) GetHistoricalDepositList(ctx context.Context, ccy, status string, startAt, endAt time.Time) (*HistoricalDepositWithdrawalResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
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
	var resp *HistoricalDepositWithdrawalResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveV1HistoricalDepositListEPL, http.MethodGet, common.EncodeURLValues("/v1/hist-deposits", params), nil, &resp)
}

// GetWithdrawalList get withdrawal list items
func (ku *Kucoin) GetWithdrawalList(ctx context.Context, ccy, status string, startAt, endAt time.Time) (*WithdrawalsResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
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
	var resp *WithdrawalsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveWithdrawalListEPL, http.MethodGet, common.EncodeURLValues("/v1/withdrawals", params), nil, &resp)
}

// GetHistoricalWithdrawalList get historical withdrawal list items
func (ku *Kucoin) GetHistoricalWithdrawalList(ctx context.Context, ccy, status string, startAt, endAt time.Time) (*HistoricalDepositWithdrawalResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
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
	var resp *HistoricalDepositWithdrawalResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveV1HistoricalWithdrawalListEPL, http.MethodGet, common.EncodeURLValues("/v1/hist-withdrawals", params), nil, &resp)
}

// GetWithdrawalQuotas get withdrawal quota details
func (ku *Kucoin) GetWithdrawalQuotas(ctx context.Context, ccy, chain string) (*WithdrawalQuota, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *WithdrawalQuota
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/withdrawals/quotas", params), nil, &resp)
}

// ApplyWithdrawal create a withdrawal request
// The endpoint was deprecated for futures, please transfer assets from the FUTURES account to the MAIN account first, and then withdraw from the MAIN account
func (ku *Kucoin) ApplyWithdrawal(ctx context.Context, ccy, address, memo, remark, chain, feeDeductType string, isInner bool, amount float64) (string, error) {
	if ccy == "" {
		return "", currency.ErrCurrencyPairEmpty
	}
	if address == "" {
		return "", errors.New("address can not be empty")
	}
	if amount == 0 {
		return "", errors.New("amount can not be empty")
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	params["address"] = address
	params["amount"] = amount
	if memo != "" {
		params["memo"] = memo
	}
	params["isInner"] = isInner
	if remark != "" {
		params["remark"] = remark
	}
	if chain != "" {
		params["chain"] = chain
	}
	if feeDeductType != "" {
		params["feeDeductType"] = feeDeductType
	}
	resp := struct {
		WithdrawalID string `json:"withdrawalId"`
		Error
	}{}
	return resp.WithdrawalID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, "/v1/withdrawals", params, &resp)
}

// CancelWithdrawal used to cancel a withdrawal request
func (ku *Kucoin) CancelWithdrawal(ctx context.Context, withdrawalID string) error {
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, "/v1/withdrawals/"+withdrawalID, nil, &struct{}{})
}

// GetBasicFee get basic fee rate of users
func (ku *Kucoin) GetBasicFee(ctx context.Context, currencyType string) (*Fees, error) {
	params := url.Values{}
	if currencyType != "" {
		params.Set("currencyType", currencyType)
	}
	var resp *Fees
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues("/v1/base-fee", params), nil, &resp)
}

// GetTradingFee get fee rate of trading pairs
// WARNING: There is a limit of 10 currency pairs allowed to be requested per call.
func (ku *Kucoin) GetTradingFee(ctx context.Context, pairs currency.Pairs) ([]Fees, error) {
	if len(pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	var resp []Fees
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, "/v1/trade-fees?symbols="+pairs.Upper().Join(), nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (ku *Kucoin) SendHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, path string, result interface{}) error {
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Pointer {
		return errInvalidResultInterface
	}
	resp, okay := result.(UnmarshalTo)
	if !okay {
		resp = &Response{Data: result}
	}
	endpointPath, err := ku.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	err = ku.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + path,
			Result:        resp,
			Verbose:       ku.Verbose,
			HTTPDebugging: ku.HTTPDebugging,
			HTTPRecording: ku.HTTPRecording}, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}
	if result == nil {
		return errNoValidResponseFromServer
	}
	return resp.GetError()
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// Request parameters are added to path variable for GET and DELETE request and for other requests its passed in params variable
func (ku *Kucoin) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, method, path string, arg, result interface{}) error {
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Pointer {
		return errInvalidResultInterface
	}
	creds, err := ku.GetCredentials(ctx)
	if err != nil {
		return err
	}
	resp, okay := result.(UnmarshalTo)
	if !okay {
		resp = &Response{Data: result}
	}
	endpointPath, err := ku.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	if value.IsNil() || value.Kind() != reflect.Pointer {
		return fmt.Errorf("%w receiver has to be non-nil pointer", errInvalidResponseReceiver)
	}
	err = ku.SendPayload(ctx, epl, func() (*request.Item, error) {
		var (
			body    io.Reader
			payload []byte
		)
		if arg != nil {
			payload, err = json.Marshal(arg)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		timeStamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		var signHash, passPhraseHash []byte
		signHash, err = crypto.GetHMAC(crypto.HashSHA256, []byte(timeStamp+method+"/api"+path+string(payload)), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		passPhraseHash, err = crypto.GetHMAC(crypto.HashSHA256, []byte(creds.ClientID), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"KC-API-KEY":         creds.Key,
			"KC-API-SIGN":        crypto.Base64Encode(signHash),
			"KC-API-TIMESTAMP":   timeStamp,
			"KC-API-PASSPHRASE":  crypto.Base64Encode(passPhraseHash),
			"KC-API-KEY-VERSION": kucoinAPIKeyVersion,
			"Content-Type":       "application/json",
		}
		return &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Body:          body,
			Result:        &resp,
			Verbose:       ku.Verbose,
			HTTPDebugging: ku.HTTPDebugging,
			HTTPRecording: ku.HTTPRecording}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	if result == nil {
		return errNoValidResponseFromServer
	}
	return resp.GetError()
}

var intervalMap = map[kline.Interval]string{
	kline.OneMin: "1min", kline.ThreeMin: "3min", kline.FiveMin: "5min", kline.FifteenMin: "15min", kline.ThirtyMin: "30min", kline.OneHour: "1hour", kline.TwoHour: "2hour", kline.FourHour: "4hour", kline.SixHour: "6hour", kline.EightHour: "8hour", kline.TwelveHour: "12hour", kline.OneDay: "1day", kline.OneWeek: "1week",
}

func (ku *Kucoin) intervalToString(interval kline.Interval) (string, error) {
	intervalString, okay := intervalMap[interval]
	if okay {
		return intervalString, nil
	}
	return "", kline.ErrUnsupportedInterval
}

func (ku *Kucoin) stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "match":
		return order.Filled, nil
	case "open":
		return order.Open, nil
	case "done":
		return order.Closed, nil
	default:
		return order.StringToOrderStatus(status)
	}
}

func (ku *Kucoin) accountTypeToString(a asset.Item) string {
	switch a {
	case asset.Spot:
		return "trade"
	case asset.Margin:
		return "margin"
	case asset.Empty:
		return ""
	default:
		return "main"
	}
}

func (ku *Kucoin) accountToTradeTypeString(a asset.Item, marginMode string) string {
	switch a {
	case asset.Spot:
		return "TRADE"
	case asset.Margin:
		if strings.EqualFold(marginMode, "isolated") {
			return "MARGIN_ISOLATED_TRADE"
		}
		return "MARGIN_TRADE"
	default:
		return ""
	}
}

func (ku *Kucoin) orderTypeToString(orderType order.Type) (string, error) {
	switch orderType {
	case order.AnyType, order.UnknownType:
		return "", nil
	case order.Market, order.Limit:
		return orderType.Lower(), nil
	default:
		return "", order.ErrUnsupportedOrderType
	}
}

func (ku *Kucoin) orderSideString(side order.Side) (string, error) {
	switch side {
	case order.Buy, order.Sell:
		return side.Lower(), nil
	case order.AnySide:
		return "", nil
	default:
		return "", fmt.Errorf("%w, side:%s", order.ErrSideIsInvalid, side.String())
	}
}
