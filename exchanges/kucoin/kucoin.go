package kucoin

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	kucoinAPIURL = "https://api.kucoin.com/api"
	tradeBaseURL = "https://www.kucoin.com/"
	tradeSpot    = "trade/"
	tradeMargin  = "margin/"
	tradeFutures = "futures/"

	symbolQuery = "?symbol="
)

// GetSymbols gets pairs details on the exchange
// For market details see endpoint: https://www.kucoin.com/docs/rest/spot-trading/market-data/get-market-list
func (ku *Kucoin) GetSymbols(ctx context.Context, market string) ([]SymbolInfo, error) {
	params := url.Values{}
	if market != "" {
		params.Set(order.Market.Lower(), market)
	}
	var resp []SymbolInfo
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, symbolsEPL, common.EncodeURLValues("/v2/symbols", params), &resp)
}

// GetTicker gets pair ticker information
func (ku *Kucoin) GetTicker(ctx context.Context, symbol string) (*Ticker, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *Ticker
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, tickersEPL, common.EncodeURLValues("/v1/market/orderbook/level1", params), &resp)
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
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, allTickersEPL, "/v1/market/allTickers", &resp)
}

// Get24hrStats get the statistics of the specified pair in the last 24 hours
func (ku *Kucoin) Get24hrStats(ctx context.Context, symbol string) (*Stats24hrs, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *Stats24hrs
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, statistics24HrEPL, common.EncodeURLValues("/v1/market/stats", params), &resp)
}

// GetMarketList get the transaction currency for the entire trading market
func (ku *Kucoin) GetMarketList(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, marketListEPL, "/v1/markets", &resp)
}

// processOB constructs an orderbook.Level instances from slice of numbers.
func processOB(ob [][2]types.Number) []orderbook.Level {
	o := make([]orderbook.Level, len(ob))
	for x := range ob {
		o[x].Amount = ob[x][1].Float64()
		o[x].Price = ob[x][0].Float64()
	}
	return o
}

// constructOrderbook parse checks and constructs an *Orderbook instance from *orderbookResponse.
func constructOrderbook(o *orderbookResponse) (*Orderbook, error) {
	s := Orderbook{
		Bids: processOB(o.Bids),
		Asks: processOB(o.Asks),
		Time: o.Time.Time(),
	}
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
func (ku *Kucoin) GetPartOrderbook20(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o *orderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, partOrderbook20EPL, common.EncodeURLValues("/v1/market/orderbook/level2_20", params), &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetPartOrderbook100 gets orderbook for a specified pair with depth 100
func (ku *Kucoin) GetPartOrderbook100(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o *orderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, partOrderbook100EPL, common.EncodeURLValues("/v1/market/orderbook/level2_100", params), &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetOrderbook gets full orderbook for a specified pair
func (ku *Kucoin) GetOrderbook(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var o *orderbookResponse
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, fullOrderbookEPL, http.MethodGet, common.EncodeURLValues("/v3/market/orderbook/level2", params), nil, &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetTradeHistory gets trade history of the specified pair
func (ku *Kucoin) GetTradeHistory(ctx context.Context, symbol string) ([]Trade, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []Trade
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, tradeHistoryEPL, common.EncodeURLValues("/v1/market/histories", params), &resp)
}

// GetKlines gets kline of the specified pair
func (ku *Kucoin) GetKlines(ctx context.Context, symbol, period string, start, end time.Time) ([]Kline, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if period == "" {
		return nil, fmt.Errorf("%w, period can not be empty", errInvalidPeriod)
	}
	if !slices.Contains(validPeriods, period) {
		return nil, errInvalidPeriod
	}
	params.Set("type", period)
	if !start.IsZero() {
		params.Set("startAt", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		params.Set("endAt", strconv.FormatInt(end.Unix(), 10))
	}
	var resp []Kline
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, klinesEPL, common.EncodeURLValues("/v1/market/candles", params), &resp)
}

// GetCurrenciesV3 the V3 of retrieving list of currencies
func (ku *Kucoin) GetCurrenciesV3(ctx context.Context) ([]CurrencyDetail, error) {
	var resp []CurrencyDetail
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, spotCurrenciesV3EPL, "/v3/currencies", &resp)
}

// GetCurrencyDetailV3 V3 endpoint to gets currency detail using currency code and chain information.
func (ku *Kucoin) GetCurrencyDetailV3(ctx context.Context, ccy currency.Code, chain string) (*CurrencyDetail, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *CurrencyDetail
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, spotCurrencyDetailEPL, common.EncodeURLValues("/v3/currencies/"+ccy.Upper().String(), params), &resp)
}

// GetFiatPrice gets fiat prices of currencies, default base currency is USD
func (ku *Kucoin) GetFiatPrice(ctx context.Context, base, currencies string) (map[string]types.Number, error) {
	params := url.Values{}
	if base != "" {
		params.Set("base", base)
	}
	if currencies != "" {
		params.Set("currencies", currencies)
	}
	var resp map[string]types.Number
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, fiatPriceEPL, common.EncodeURLValues("/v1/prices", params), &resp)
}

// GetLeveragedTokenInfo returns leveraged token information
func (ku *Kucoin) GetLeveragedTokenInfo(ctx context.Context, ccy currency.Code) ([]LeveragedTokenInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []LeveragedTokenInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, leveragedTokenInfoEPL, http.MethodGet, common.EncodeURLValues("/v3/etf/info", params), nil, &resp)
}

// GetMarkPrice gets index price of the specified pair
func (ku *Kucoin) GetMarkPrice(ctx context.Context, symbol string) (*MarkPrice, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *MarkPrice
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, getMarkPriceEPL, "/v1/mark-price/"+symbol+"/current", &resp)
}

// GetAllMarginTradingPairsMarkPrices retrieves all margin trading pairs ticker mark price information
func (ku *Kucoin) GetAllMarginTradingPairsMarkPrices(ctx context.Context) ([]MarkPrice, error) {
	var resp []MarkPrice
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, getAllMarginMarkPriceEPL, "/v3/mark-price/all-symbols", &resp)
}

// GetMarginConfiguration gets configure info of the margin
func (ku *Kucoin) GetMarginConfiguration(ctx context.Context) (*MarginConfiguration, error) {
	var resp *MarginConfiguration
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, getMarginConfigurationEPL, "/v1/margin/config", &resp)
}

// GetMarginAccount gets configure info of the margin
func (ku *Kucoin) GetMarginAccount(ctx context.Context) (*MarginAccounts, error) {
	var resp *MarginAccounts
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, marginAccountDetailEPL, http.MethodGet, "/v1/margin/account", nil, &resp)
}

// GetCrossMarginRiskLimitCurrencyConfig risk limit and currency configuration of cross margin account
// isIsolated: true - isolated, false - cross ; default false
func (ku *Kucoin) GetCrossMarginRiskLimitCurrencyConfig(ctx context.Context, symbol string, ccy currency.Code) ([]CrossMarginRiskLimitCurrencyConfig, error) {
	var resp []CrossMarginRiskLimitCurrencyConfig
	return resp, ku.getCrossOrIsolatedMarginRiskLimitCurrencyConfig(ctx, false, symbol, ccy, &resp)
}

// GetIsolatedMarginRiskLimitCurrencyConfig risk limit and currency configuration of cross isolated margin
func (ku *Kucoin) GetIsolatedMarginRiskLimitCurrencyConfig(ctx context.Context, symbol string, ccy currency.Code) ([]IsolatedMarginRiskLimitCurrencyConfig, error) {
	var resp []IsolatedMarginRiskLimitCurrencyConfig
	return resp, ku.getCrossOrIsolatedMarginRiskLimitCurrencyConfig(ctx, true, symbol, ccy, &resp)
}

func (ku *Kucoin) getCrossOrIsolatedMarginRiskLimitCurrencyConfig(ctx context.Context, isIsolated bool, symbol string, ccy currency.Code, resp any) error {
	params := url.Values{}
	if isIsolated {
		params.Set("isIsolated", "true")
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, crossIsolatedMarginRiskLimitCurrencyConfigEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/currencies", params), nil, &resp)
}

// PostMarginBorrowOrder used to post borrow order
func (ku *Kucoin) PostMarginBorrowOrder(ctx context.Context, arg *MarginBorrowParam) (*BorrowAndRepaymentOrderResp, error) {
	if *arg == (MarginBorrowParam{}) {
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, postMarginBorrowOrderEPL, http.MethodPost, "/v3/margin/borrow", arg, &resp)
}

// GetMarginBorrowingHistory retrieves the borrowing orders for cross and isolated margin accounts
func (ku *Kucoin) GetMarginBorrowingHistory(ctx context.Context, ccy currency.Code, isIsolated bool,
	symbol, orderNo string,
	startTime, endTime time.Time,
	currentPage, pageSize int64,
) (*BorrowRepayDetailResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if isIsolated {
		params.Set("isIsolated", "true")
	}
	params.Set("symbol", symbol)
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
	var resp *BorrowRepayDetailResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, marginBorrowingHistoryEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/borrow", params), nil, &resp)
}

// PostRepayment used to initiate an application for the repayment of cross or isolated margin borrowing.
func (ku *Kucoin) PostRepayment(ctx context.Context, arg *RepayParam) (*BorrowAndRepaymentOrderResp, error) {
	if *arg == (RepayParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Size <= 0 {
		return nil, fmt.Errorf("%w , size = %f", order.ErrAmountBelowMin, arg.Size)
	}
	var resp *BorrowAndRepaymentOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, postMarginRepaymentEPL, http.MethodPost, "/v3/margin/repay", arg, &resp)
}

// GetCrossIsolatedMarginInterestRecords request via this endpoint to get the interest records of the cross/isolated margin lending
func (ku *Kucoin) GetCrossIsolatedMarginInterestRecords(ctx context.Context, isIsolated bool, symbol string, ccy currency.Code, startTime, endTime time.Time, currentPage, pageSize int64) (*MarginInterestRecords, error) {
	params := url.Values{}
	if isIsolated {
		params.Set("isIsolated", "true")
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if currentPage > 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *MarginInterestRecords
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getCrossIsolatedMarginInterestRecordsEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/interest", params), nil, &resp)
}

// GetRepaymentHistory retrieves the repayment orders for cross and isolated margin accounts.
func (ku *Kucoin) GetRepaymentHistory(ctx context.Context, ccy currency.Code, isIsolated bool,
	symbol, orderNo string,
	startTime, endTime time.Time,
	currentPage, pageSize int64,
) (*BorrowRepayDetailResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if isIsolated {
		params.Set("isIsolated", "true")
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
	var resp *BorrowRepayDetailResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, marginRepaymentHistoryEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/repay", params), nil, &resp)
}

// GetIsolatedMarginPairConfig get the current isolated margin trading pair configuration
func (ku *Kucoin) GetIsolatedMarginPairConfig(ctx context.Context) ([]IsolatedMarginPairConfig, error) {
	var resp []IsolatedMarginPairConfig
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, isolatedMarginPairConfigEPL, http.MethodGet, "/v1/isolated/symbols", nil, &resp)
}

// GetIsolatedMarginAccountInfo get all isolated margin accounts of the current user
func (ku *Kucoin) GetIsolatedMarginAccountInfo(ctx context.Context, balanceCurrency string) (*IsolatedMarginAccountInfo, error) {
	params := url.Values{}
	if balanceCurrency != "" {
		params.Set("balanceCurrency", balanceCurrency)
	}
	var resp *IsolatedMarginAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, isolatedMarginAccountInfoEPL, http.MethodGet, common.EncodeURLValues("/v1/isolated/accounts", params), nil, &resp)
}

// GetSingleIsolatedMarginAccountInfo get single isolated margin accounts of the current user
func (ku *Kucoin) GetSingleIsolatedMarginAccountInfo(ctx context.Context, symbol string) (*AssetInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *AssetInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, singleIsolatedMarginAccountInfoEPL, http.MethodGet, "/v1/isolated/account/"+symbol, nil, &resp)
}

// GetCurrentServerTime gets the server time
func (ku *Kucoin) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Timestamp types.Time `json:"data"`
		Error
	}{}
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, currentServerTimeEPL, "/v1/timestamp", &resp)
	if err != nil {
		return time.Time{}, err
	}
	return resp.Timestamp.Time(), nil
}

// GetServiceStatus gets the service status
func (ku *Kucoin) GetServiceStatus(ctx context.Context) (*ServiceStatus, error) {
	var resp *ServiceStatus
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, serviceStatusEPL, "/v1/status", &resp)
}

// --------------------------------------------- Spot High Frequency(HF) Pro Account ---------------------------

// HFSpotPlaceOrder places a high frequency spot order
// There are two types of orders: (limit) order: set price and quantity for the transaction. (market) order : set amount or quantity for the transaction.
func (ku *Kucoin) HFSpotPlaceOrder(ctx context.Context, arg *PlaceHFParam) (string, error) {
	return ku.SendSpotHFPlaceOrder(ctx, arg, "/v1/hf/orders")
}

// SpotPlaceHFOrderTest order test endpoint, the request parameters and return parameters of this endpoint are exactly the same as the order endpoint,
// and can be used to verify whether the signature is correct and other operations.
func (ku *Kucoin) SpotPlaceHFOrderTest(ctx context.Context, arg *PlaceHFParam) (string, error) {
	return ku.SendSpotHFPlaceOrder(ctx, arg, "/v1/hf/orders/test")
}

// ValidatePlaceOrderParams validates an order placement parameters.
func (a *PlaceHFParam) ValidatePlaceOrderParams() error {
	if *a == (PlaceHFParam{}) {
		return common.ErrNilPointer
	}
	if a.Symbol.IsEmpty() {
		return currency.ErrSymbolStringEmpty
	}
	if a.OrderType == "" {
		return order.ErrTypeIsInvalid
	}
	if a.Side == "" {
		return order.ErrSideIsInvalid
	}
	a.Side = strings.ToLower(a.Side)
	if a.Price <= 0 {
		return order.ErrPriceBelowMin
	}
	if a.Size <= 0 {
		return order.ErrAmountBelowMin
	}
	return nil
}

// SendSpotHFPlaceOrder sends a spot high-frequency order to the specified path
// Use HFSpotPlaceOrder to place an order or SpotPlaceHFOrderTest to send a test order
func (ku *Kucoin) SendSpotHFPlaceOrder(ctx context.Context, arg *PlaceHFParam, path string) (string, error) {
	err := arg.ValidatePlaceOrderParams()
	if err != nil {
		return "", err
	}
	resp := &struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfPlaceOrderEPL, http.MethodPost, path, arg, &resp)
}

// SyncPlaceHFOrder this interface will synchronously return the order information after the order matching is completed.
func (ku *Kucoin) SyncPlaceHFOrder(ctx context.Context, arg *PlaceHFParam) (*SyncPlaceHFOrderResp, error) {
	err := arg.ValidatePlaceOrderParams()
	if err != nil {
		return nil, err
	}
	var resp *SyncPlaceHFOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfSyncPlaceOrderEPL, http.MethodPost, "/v1/hf/orders/sync", arg, &resp)
}

// PlaceMultipleOrders endpoint supports sequential batch order placement from a single endpoint. A maximum of 5 orders can be placed simultaneously.
func (ku *Kucoin) PlaceMultipleOrders(ctx context.Context, args []PlaceHFParam) ([]PlaceOrderResp, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for i := range args {
		err := args[i].ValidatePlaceOrderParams()
		if err != nil {
			return nil, err
		}
	}
	var resp []PlaceOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfMultipleOrdersEPL, http.MethodPost, "/v1/hf/orders/multi", &PlaceOrderParams{OrderList: args}, &resp)
}

// SyncPlaceMultipleHFOrders this interface will synchronously return the order information after the order matching is completed
func (ku *Kucoin) SyncPlaceMultipleHFOrders(ctx context.Context, args []PlaceHFParam) ([]SyncPlaceHFOrderResp, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for i := range args {
		err := args[i].ValidatePlaceOrderParams()
		if err != nil {
			return nil, err
		}
	}
	var resp []SyncPlaceHFOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfSyncPlaceMultipleHFOrdersEPL, http.MethodPost, "/v1/hf/orders/multi/sync", args, &resp)
}

// ModifyHFOrder modifies a high frequency order.
func (ku *Kucoin) ModifyHFOrder(ctx context.Context, arg *ModifyHFOrderParam) (string, error) {
	if *arg == (ModifyHFOrderParam{}) {
		return "", common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	resp := &struct {
		NewOrderID string `json:"newOrderId"`
	}{}
	return resp.NewOrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfModifyOrderEPL, http.MethodPost, "/v1/hf/orders/alter", arg, &resp)
}

// CancelHFOrder used to cancel a high-frequency order by orderId.
func (ku *Kucoin) CancelHFOrder(ctx context.Context, orderID, symbol string) (string, error) {
	if orderID == "" {
		return "", order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	resp := &struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelHFOrderEPL, http.MethodDelete, "/v1/hf/orders/"+orderID+symbolQuery+symbol, nil, &resp)
}

// SyncCancelHFOrder this interface will synchronously return the order information after the order canceling is completed.
func (ku *Kucoin) SyncCancelHFOrder(ctx context.Context, orderID, symbol string) (*SyncCancelHFOrderResp, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	return ku.SendSyncCancelHFOrder(ctx, orderID, symbol, "/v1/hf/orders/sync/")
}

// SyncCancelHFOrderByClientOrderID this interface will synchronously return the order information after the order canceling is completed.
func (ku *Kucoin) SyncCancelHFOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) (*SyncCancelHFOrderResp, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	return ku.SendSyncCancelHFOrder(ctx, clientOrderID, symbol, "/v1/hf/orders/sync/client-order/")
}

// SendSyncCancelHFOrder sends a sync-cancel high-frequency order by order ID or client supplied order ID.
func (ku *Kucoin) SendSyncCancelHFOrder(ctx context.Context, id, symbol, path string) (*SyncCancelHFOrderResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *SyncCancelHFOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfSyncCancelOrderEPL, http.MethodDelete, path+id+symbolQuery+symbol, nil, &resp)
}

// CancelHFOrderByClientOrderID sends out a request to cancel a high-frequency order using clientOid.
func (ku *Kucoin) CancelHFOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) (string, error) {
	if clientOrderID == "" {
		return "", order.ErrClientOrderIDMustBeSet
	}
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	resp := &struct {
		ClientOrderID string `json:"clientOid"`
	}{}
	return resp.ClientOrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfCancelOrderByClientOrderIDEPL, http.MethodDelete, "/v1/hf/orders/client-order/"+clientOrderID+symbolQuery+symbol, nil, &resp)
}

// CancelSpecifiedNumberHFOrdersByOrderID cancel the specified quantity of the order according to the orderId.
func (ku *Kucoin) CancelSpecifiedNumberHFOrdersByOrderID(ctx context.Context, orderID, symbol string, cancelSize float64) (*CancelOrderByNumberResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if cancelSize == 0 {
		return nil, fmt.Errorf("%w, cancel size is required", order.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("cancelSize", strconv.FormatFloat(cancelSize, 'f', -1, 64))
	var resp *CancelOrderByNumberResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelSpecifiedNumberHFOrdersByOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/hf/orders/cancel/"+orderID, params), nil, &resp)
}

// CancelAllHFOrdersBySymbol cancel all open high-frequency orders
func (ku *Kucoin) CancelAllHFOrdersBySymbol(ctx context.Context, symbol string) (string, error) {
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	var resp string
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfCancelAllOrdersBySymbolEPL, http.MethodDelete, "/v1/hf/orders?symbol="+symbol, nil, &resp)
}

// CancelAllHFOrders cancels all high-frequency orders for all symbols
func (ku *Kucoin) CancelAllHFOrders(ctx context.Context) (*CancelAllHFOrdersResponse, error) {
	var resp *CancelAllHFOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfCancelAllOrdersEPL, http.MethodDelete, "/v1/hf/orders/cancelAll", nil, &resp)
}

// GetActiveHFOrders retrieves all high-frequency active orders
func (ku *Kucoin) GetActiveHFOrders(ctx context.Context, symbol string) ([]OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfGetAllActiveOrdersEPL, http.MethodGet, "/v1/hf/orders/active?symbol="+symbol, nil, &resp)
}

// GetSymbolsWithActiveHFOrderList retrieves all trading pairs that the user has active orders
func (ku *Kucoin) GetSymbolsWithActiveHFOrderList(ctx context.Context) ([]string, error) {
	resp := &struct {
		Symbols []string `json:"symbols"`
	}{}
	return resp.Symbols, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfSymbolsWithActiveOrdersEPL, http.MethodGet, "/v1/hf/orders/active/symbols", nil, &resp)
}

// GetHFCompletedOrderList obtains a list of filled HF orders and returns paginated data. The returned data is sorted in descending order based on the latest order update times.
func (ku *Kucoin) GetHFCompletedOrderList(ctx context.Context, symbol, side, orderType, lastID string, startAt, endAt time.Time, limit int64) (*CompletedHFOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if side != "" {
		params.Set("side", strings.ToLower(side))
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
	if lastID == "" {
		params.Set("lastId", lastID)
	}
	if limit > 0 {
		params.Set(order.Limit.Lower(), strconv.FormatInt(limit, 10))
	}
	var resp *CompletedHFOrder
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfCompletedOrderListEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/orders/done", params), nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetHFOrderDetailsByOrderID obtain information for a single HF order using the order id.
// If the order is not an active order, you can only get data within the time range of 3 _ 24 hours (ie: from the current time to 3 _ 24 hours ago).
func (ku *Kucoin) GetHFOrderDetailsByOrderID(ctx context.Context, orderID, symbol string) (*OrderDetail, error) {
	return ku.GetHFOrderDetailsByID(ctx, orderID, symbol, "/v1/hf/orders/")
}

// GetHFOrderDetailsByClientOrderID used to obtain information about a single order using clientOid. If the order does not exist, then there will be a prompt saying that the order does not exist.
func (ku *Kucoin) GetHFOrderDetailsByClientOrderID(ctx context.Context, clientOrderID, symbol string) (*OrderDetail, error) {
	return ku.GetHFOrderDetailsByID(ctx, clientOrderID, symbol, "/v1/hf/orders/client-order/")
}

// GetHFOrderDetailsByID retrieves a high-frequency order by order ID or client supplied ID.
func (ku *Kucoin) GetHFOrderDetailsByID(ctx context.Context, orderID, symbol, path string) (*OrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *OrderDetail
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfOrderDetailByOrderIDEPL, http.MethodGet, path+orderID+symbolQuery+symbol, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, order.ErrOrderNotFound
	}
	return resp, nil
}

// AutoCancelHFOrderSetting automatically cancel all orders of the set trading pair after the specified time.
// If this interface is not called again for renewal or cancellation before the set time,
// the system will help the user to cancel the order of the corresponding trading pair. Otherwise it will not.
func (ku *Kucoin) AutoCancelHFOrderSetting(ctx context.Context, timeout int64, symbols []string) (*AutoCancelHFOrderResponse, error) {
	if timeout == 0 {
		return nil, errTimeoutRequired
	}
	arg := &struct {
		Timeout int64    `json:"timeout,string"`
		Symbols []string `json:"symbols,omitempty"`
	}{
		Timeout: timeout,
		Symbols: symbols,
	}
	var resp *AutoCancelHFOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, autoCancelHFOrderSettingEPL, http.MethodPost, "/v1/hf/orders/dead-cancel-all", arg, &resp)
}

// AutoCancelHFOrderSettingQuery query the settings of automatic order cancellation
func (ku *Kucoin) AutoCancelHFOrderSettingQuery(ctx context.Context) (*AutoCancelHFOrderResponse, error) {
	var resp *AutoCancelHFOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, autoCancelHFOrderSettingQueryEPL, http.MethodGet, "/v1/hf/orders/dead-cancel-all/query", nil, &resp)
}

// GetHFFilledList retrieves a list of the latest HF transaction details. The returned results are paginated. The data is sorted in descending order according to time.
func (ku *Kucoin) GetHFFilledList(ctx context.Context, orderID, symbol, side, orderType, lastID string, startAt, endAt time.Time, limit int64) (*HFOrderFills, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if side != "" {
		params.Set("side", strings.ToLower(side))
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
	if lastID != "" {
		params.Set("lastId", lastID)
	}
	if limit > 0 {
		params.Set(order.Limit.Lower(), strconv.FormatInt(limit, 10))
	}
	var resp *HFOrderFills
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfFilledListEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/fills", params), nil, &resp)
}

// PostOrder used to place two types of orders: limit and market
// Note: use this only for SPOT trades
func (ku *Kucoin) PostOrder(ctx context.Context, arg *SpotOrderParam) (string, error) {
	return ku.HandlePostOrder(ctx, arg, "/v1/orders")
}

// PostOrderTest used to verify whether the signature is correct and other operations.
// After placing an order, the order will not enter the matching system, and the order cannot be queried.
func (ku *Kucoin) PostOrderTest(ctx context.Context, arg *SpotOrderParam) (string, error) {
	return ku.HandlePostOrder(ctx, arg, "/v1/orders/test")
}

// HandlePostOrder applies a spot order placement or tests the order placement process.
func (ku *Kucoin) HandlePostOrder(ctx context.Context, arg *SpotOrderParam, path string) (string, error) {
	if arg.ClientOrderID == "" {
		// NOTE: 128 bit max length character string. UUID recommended.
		return "", order.ErrClientOrderIDMustBeSet
	}
	if arg.Side == "" {
		return "", order.ErrSideIsInvalid
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Symbol.IsEmpty() {
		return "", fmt.Errorf("%w, empty symbol", currency.ErrCurrencyPairEmpty)
	}
	switch arg.OrderType {
	case order.Limit.Lower(), "":
		if arg.Price <= 0 {
			return "", fmt.Errorf("%w, price =%.3f", order.ErrPriceBelowMin, arg.Price)
		}
		if arg.Size <= 0 {
			return "", order.ErrAmountBelowMin
		}
		if arg.VisibleSize < 0 {
			return "", fmt.Errorf("%w, visible size must be non-zero positive value", order.ErrAmountBelowMin)
		}
	case order.Market.Lower():
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
	return resp.Data.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, path, &arg, &resp)
}

// PostMarginOrderTest a test endpoint used to place two types of margin orders: limit and margin.
func (ku *Kucoin) PostMarginOrderTest(ctx context.Context, arg *MarginOrderParam) (*PostMarginOrderResp, error) {
	return ku.SendPostMarginOrder(ctx, arg, "/v1/margin/order/test")
}

// PostMarginOrder used to place two types of margin orders: limit and market
func (ku *Kucoin) PostMarginOrder(ctx context.Context, arg *MarginOrderParam) (*PostMarginOrderResp, error) {
	return ku.SendPostMarginOrder(ctx, arg, "/v1/margin/order")
}

// SendPostMarginOrder applies a margin order placement or tests the order placement process.
func (ku *Kucoin) SendPostMarginOrder(ctx context.Context, arg *MarginOrderParam, path string) (*PostMarginOrderResp, error) {
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	switch arg.OrderType {
	case order.Limit.Lower(), "":
		if arg.Price <= 0 {
			return nil, fmt.Errorf("%w, price=%.3f", order.ErrPriceBelowMin, arg.Price)
		}
		if arg.Size <= 0 {
			return nil, order.ErrAmountBelowMin
		}
		if arg.VisibleSize < 0 {
			return nil, fmt.Errorf("%w, visible size must be non-zero positive value", order.ErrAmountBelowMin)
		}
	case order.Market.Lower():
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
	return &resp.PostMarginOrderResp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginOrdersEPL, http.MethodPost, path, &arg, &resp)
}

// PostBulkOrder used to place 5 orders at the same time. The order type must be a limit order of the same symbol
// Note: it supports only SPOT trades
// Note: To check if order was posted successfully, check status field in response
func (ku *Kucoin) PostBulkOrder(ctx context.Context, symbol string, orderList []OrderRequest) ([]PostBulkOrderResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if len(orderList) == 0 {
		return nil, common.ErrEmptyParams
	}
	for i := range orderList {
		if orderList[i].ClientOID == "" {
			return nil, order.ErrClientOrderIDMustBeSet
		}
		if orderList[i].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		orderList[i].Side = strings.ToLower(orderList[i].Side)
		if orderList[i].Price <= 0 {
			return nil, order.ErrPriceBelowMin
		}
		if orderList[i].Size <= 0 {
			return nil, order.ErrAmountBelowMin
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
		return nil, order.ErrOrderIDNotSet
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOrderEPL, http.MethodDelete, "/v1/orders/"+orderID, nil, &resp)
}

// CancelOrderByClientOID used to cancel order via the clientOid
func (ku *Kucoin) CancelOrderByClientOID(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOrderByClientOrderIDEPL, http.MethodDelete, "/v1/order/client-order/"+orderID, nil, &resp)
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
	}{
		CancelledOrderIDs: []string{},
	}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelAllOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v1/orders", params), nil, &resp)
}

// ListOrders gets the user order list
func (ku *Kucoin) ListOrders(ctx context.Context, status, symbol, side, orderType, tradeType string, startAt, endAt time.Time) (*OrdersListResponse, error) {
	params := FillParams(symbol, side, orderType, tradeType, startAt, endAt)
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

// FillParams fills request parameters for orders and order fills.
func FillParams(symbol, side, orderType, tradeType string, startAt, endAt time.Time) url.Values {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", strings.ToLower(side))
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, recentOrdersEPL, http.MethodGet, "/v1/limit/orders", nil, &resp)
}

// GetOrderByID get a single order info by order ID
func (ku *Kucoin) GetOrderByID(ctx context.Context, orderID string) (*OrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, orderDetailByIDEPL, http.MethodGet, "/v1/orders/"+orderID, nil, &resp)
}

// GetOrderByClientSuppliedOrderID get a single order info by client order ID
func (ku *Kucoin) GetOrderByClientSuppliedOrderID(ctx context.Context, clientOID string) (*OrderDetail, error) {
	if clientOID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	var resp *OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOrderByClientSuppliedOrderIDEPL, http.MethodGet, "/v1/order/client-order/"+clientOID, nil, &resp)
}

// GetFills get fills
func (ku *Kucoin) GetFills(ctx context.Context, orderID, symbol, side, orderType, tradeType string, startAt, endAt time.Time) (*ListFills, error) {
	params := FillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	var resp *ListFills
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, listFillsEPL, http.MethodGet, common.EncodeURLValues("/v1/fills", params), nil, &resp)
}

// GetRecentFills get a list of 1000 fills in last 24 hours
func (ku *Kucoin) GetRecentFills(ctx context.Context) ([]Fill, error) {
	var resp []Fill
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getRecentFillsEPL, http.MethodGet, "/v1/limit/fills", nil, &resp)
}

// PostStopOrder used to place two types of stop orders: limit and market
func (ku *Kucoin) PostStopOrder(ctx context.Context, clientOID, side, symbol, orderType, remark, stop, stp,
	tradeType, timeInForce string, size, price, stopPrice, cancelAfter, visibleSize,
	funds float64, postOnly, hidden, iceberg bool,
) (string, error) {
	if clientOID == "" {
		return "", order.ErrClientOrderIDMustBeSet
	}
	if side == "" {
		return "", fmt.Errorf("%w order side cannot be empty", order.ErrSideIsInvalid)
	}
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	arg := make(map[string]any)
	arg["clientOid"] = clientOID
	arg["side"] = strings.ToLower(side)
	arg["symbol"] = symbol
	if remark != "" {
		arg["remark"] = remark
	}
	if stop != "" {
		arg["stop"] = stop
		if stopPrice <= 0 {
			return "", fmt.Errorf("%w, stopPrice is required", order.ErrPriceBelowMin)
		}
		arg["stopPrice"] = strconv.FormatFloat(stopPrice, 'f', -1, 64)
	}
	if stp != "" {
		arg["stp"] = stp
	}
	if tradeType != "" {
		arg["tradeType"] = tradeType
	}
	orderType = strings.ToLower(orderType)
	switch orderType {
	case order.Limit.Lower(), "":
		if price <= 0 {
			return "", order.ErrPriceBelowMin
		}
		arg["price"] = strconv.FormatFloat(price, 'f', -1, 64)
		if size <= 0 {
			return "", fmt.Errorf("%w, size is required", order.ErrAmountBelowMin)
		}
		arg["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		if timeInForce != "" {
			arg["timeInForce"] = timeInForce
		}
		if cancelAfter > 0 && timeInForce == order.GoodTillTime.String() {
			arg["cancelAfter"] = strconv.FormatFloat(cancelAfter, 'f', -1, 64)
		}
		arg["postOnly"] = postOnly
		arg["hidden"] = hidden
		arg["iceberg"] = iceberg
		if visibleSize > 0 {
			arg["visibleSize"] = strconv.FormatFloat(visibleSize, 'f', -1, 64)
		}
	case order.Market.Lower():
		switch {
		case size > 0:
			arg["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		case funds > 0:
			arg["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
		default:
			return "", errSizeOrFundIsRequired
		}
	default:
		return "", fmt.Errorf("%w, order type: %s", order.ErrTypeIsInvalid, orderType)
	}
	if orderType != "" {
		arg["type"] = orderType
	}
	resp := struct {
		OrderID string `json:"orderId"`
		Error
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeStopOrderEPL, http.MethodPost, "/v1/stop-order", arg, &resp)
}

// CancelStopOrder used to cancel single stop order previously placed
func (ku *Kucoin) CancelStopOrder(ctx context.Context, orderID string) ([]string, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	resp := struct {
		Data []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelStopOrderEPL, http.MethodDelete, "/v1/stop-order/"+orderID, nil, &resp)
}

// CancelStopOrderByClientOrderID used to cancel single stop order previously placed by client supplied order ID.
func (ku *Kucoin) CancelStopOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) ([]string, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOrderID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	resp := struct {
		Data []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelStopOrderEPL, http.MethodDelete, common.EncodeURLValues("/v1/stop-order/cancelOrderByClientOid", params), nil, &resp)
}

// CancelStopOrders used to cancel all order based upon the parameters passed
func (ku *Kucoin) CancelStopOrders(ctx context.Context, symbol, tradeType string, orderIDs []string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	if len(orderIDs) > 0 {
		params.Set("orderIds", strings.Join(orderIDs, ","))
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{
		CancelledOrderIDs: []string{},
	}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelStopOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v1/stop-order/cancel", params), nil, &resp)
}

// GetStopOrder used to cancel single stop order previously placed
func (ku *Kucoin) GetStopOrder(ctx context.Context, orderID string) (*StopOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	resp := struct {
		StopOrder
		Error
	}{}
	return &resp.StopOrder, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getStopOrderDetailEPL, http.MethodGet, "/v1/stop-order/"+orderID, nil, &resp)
}

// ListStopOrders get all current untriggered stop orders
func (ku *Kucoin) ListStopOrders(ctx context.Context, symbol, side, orderType, tradeType string, orderIDs []string, startAt, endAt time.Time, currentPage, pageSize int64) (*StopOrderListResponse, error) {
	params := FillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if len(orderIDs) > 0 {
		params.Set("orderIds", strings.Join(orderIDs, ","))
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *StopOrderListResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, listStopOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/stop-order", params), nil, &resp)
}

// GetStopOrderByClientID get a stop order information via the clientOID
func (ku *Kucoin) GetStopOrderByClientID(ctx context.Context, symbol, clientOID string) ([]StopOrder, error) {
	if clientOID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []StopOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getStopOrderByClientIDEPL, http.MethodGet, common.EncodeURLValues("/v1/stop-order/queryOrderByClientOid", params), nil, &resp)
}

// CancelStopOrderByClientID used to cancel a stop order via the clientOID.
func (ku *Kucoin) CancelStopOrderByClientID(ctx context.Context, symbol, clientOID string) (*CancelOrderResponse, error) {
	if clientOID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("clientOid", clientOID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *CancelOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelStopOrderByClientIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/stop-order/cancelOrderByClientOid", params), nil, &resp)
}

// ------------------------------------------------ OCO Order -----------------------------------------------------------------

// PlaceOCOOrder creates a new One cancel other(OCO) order.
func (ku *Kucoin) PlaceOCOOrder(ctx context.Context, arg *OCOOrderParams) (string, error) {
	if *arg == (OCOOrderParams{}) {
		return "", common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return "", order.ErrSideIsInvalid
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Price <= 0 {
		return "", order.ErrPriceBelowMin
	}
	if arg.Size <= 0 {
		return "", order.ErrAmountBelowMin
	}
	if arg.StopPrice <= 0 {
		return "", fmt.Errorf("%w stop price = %f", order.ErrPriceBelowMin, arg.StopPrice)
	}
	if arg.LimitPrice <= 0 {
		return "", fmt.Errorf("%w limit price = %f", order.ErrPriceBelowMin, arg.LimitPrice)
	}
	if arg.ClientOrderID == "" {
		return "", order.ErrClientOrderIDMustBeSet
	}
	arg.Side = strings.ToLower(arg.Side)
	resp := &struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeOCOOrderEPL, http.MethodPost, "/v3/oco/order", &arg, &resp)
}

// CancelOCOOrderByOrderID cancels a single oco order previously placed by order ID.
func (ku *Kucoin) CancelOCOOrderByOrderID(ctx context.Context, orderID string) (*OCOOrderCancellationResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	return ku.CancelOCOOrderByID(ctx, "/v3/oco/order/", orderID)
}

// CancelOCOOrderByClientOrderID cancels a single oco order previously placed by client order ID.
func (ku *Kucoin) CancelOCOOrderByClientOrderID(ctx context.Context, clientOrderID string) (*OCOOrderCancellationResponse, error) {
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	return ku.CancelOCOOrderByID(ctx, "/v3/oco/client-order/", clientOrderID)
}

// CancelOCOOrderByID sends a cancel OCO order by order ID or client supplied order ID.
func (ku *Kucoin) CancelOCOOrderByID(ctx context.Context, path, id string) (*OCOOrderCancellationResponse, error) {
	var resp *OCOOrderCancellationResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOCOOrderByIDEPL, http.MethodDelete, path+id, nil, &resp)
}

// CancelOCOMultipleOrders batch cancel OCO orders through orderIds.
func (ku *Kucoin) CancelOCOMultipleOrders(ctx context.Context, orderIDs []string, symbol string) (*OCOOrderCancellationResponse, error) {
	params := url.Values{}
	if len(orderIDs) > 0 {
		params.Set("orderIds", strings.Join(orderIDs, ","))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *OCOOrderCancellationResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMultipleOCOOrdersEPL, http.MethodDelete, common.EncodeURLValues("/v3/oco/orders", params), nil, &resp)
}

// GetOCOOrderInfoByOrderID to get a oco order information via the order ID.
func (ku *Kucoin) GetOCOOrderInfoByOrderID(ctx context.Context, orderID string) (*OCOOrderInfo, error) {
	return ku.GetOCOOrderInfoByID(ctx, orderID, "/v3/oco/order/")
}

// GetOCOOrderInfoByClientOrderID to get a oco order information via the client order ID.
func (ku *Kucoin) GetOCOOrderInfoByClientOrderID(ctx context.Context, clientOrderID string) (*OCOOrderInfo, error) {
	return ku.GetOCOOrderInfoByID(ctx, clientOrderID, "/v3/oco/client-order/")
}

// GetOCOOrderInfoByID sends a request to get an OCO order by order ID or client supplied order ID.
func (ku *Kucoin) GetOCOOrderInfoByID(ctx context.Context, id, path string) (*OCOOrderInfo, error) {
	if id == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OCOOrderInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOCOOrderByIDEPL, http.MethodGet, path+id, nil, &resp)
}

// GetOCOOrderDetailsByOrderID get a oco order detail via the order ID.
func (ku *Kucoin) GetOCOOrderDetailsByOrderID(ctx context.Context, orderID string) (*OCOOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OCOOrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOCOOrderDetailsByOrderIDEPL, http.MethodGet, "/v3/oco/order/details/"+orderID, nil, &resp)
}

// GetOCOOrderList retrieves list of OCO orders.
func (ku *Kucoin) GetOCOOrderList(ctx context.Context, pageSize, currentPage int64, symbol string, startAt, endAt time.Time, orderIDs []string) (*OCOOrders, error) {
	if pageSize < 10 {
		return nil, fmt.Errorf("%w, pageSize must be between 10 and 500", errPageSizeRequired)
	}
	if currentPage <= 0 {
		return nil, fmt.Errorf("%w, must be greater than 1", errCurrentPageRequired)
	}
	params := url.Values{}
	params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if len(orderIDs) == 0 {
		params.Set("orderIds", strings.Join(orderIDs, ","))
	}
	var resp *OCOOrders
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getOCOOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/oco/orders", params), nil, &resp)
}

// ----------------------------------------------------------- Margin HF Trade -------------------------------------------------------------

// PlaceMarginHFOrder used to place cross-margin or isolated-margin high-frequency margin trading
func (ku *Kucoin) PlaceMarginHFOrder(ctx context.Context, arg *PlaceMarginHFOrderParam) (*MarginHFOrderResponse, error) {
	return ku.SendPlaceMarginHFOrder(ctx, arg, "/v3/hf/margin/order")
}

// PlaceMarginHFOrderTest used to verify whether the signature is correct and other operations. After placing an order,
// the order will not enter the matching system, and the order cannot be queried.
func (ku *Kucoin) PlaceMarginHFOrderTest(ctx context.Context, arg *PlaceMarginHFOrderParam) (*MarginHFOrderResponse, error) {
	return ku.SendPlaceMarginHFOrder(ctx, arg, "/v3/hf/margin/order/test")
}

// SendPlaceMarginHFOrder applies a high-frequency margin order placement or tests the order placement process.
func (ku *Kucoin) SendPlaceMarginHFOrder(ctx context.Context, arg *PlaceMarginHFOrderParam, path string) (*MarginHFOrderResponse, error) {
	if *arg == (PlaceMarginHFOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDNotSupported
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.Size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *MarginHFOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginOrderEPL, http.MethodPost, path, arg, &resp)
}

// CancelMarginHFOrderByOrderID cancels a single order by orderId. If the order cannot be canceled (sold or canceled),
// an error message will be returned, and the reason can be obtained according to the returned msg.
func (ku *Kucoin) CancelMarginHFOrderByOrderID(ctx context.Context, orderID, symbol string) (string, error) {
	return ku.CancelMarginHFOrderByID(ctx, orderID, symbol, "/v3/hf/margin/orders/")
}

// CancelMarginHFOrderByClientOrderID to cancel a single order by clientOid.
func (ku *Kucoin) CancelMarginHFOrderByClientOrderID(ctx context.Context, clientOrderID, symbol string) (string, error) {
	return ku.CancelMarginHFOrderByID(ctx, clientOrderID, symbol, "/v3/hf/margin/orders/client-order/")
}

// CancelMarginHFOrderByID sends a cancel order high frequency margin orders by order ID or client supplied order ID.
func (ku *Kucoin) CancelMarginHFOrderByID(ctx context.Context, id, symbol, path string) (string, error) {
	if id == "" {
		return "", order.ErrOrderIDNotSet
	}
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	resp := &struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelMarginHFOrderByIDEPL, http.MethodDelete, path+id+symbolQuery+symbol, nil, &resp)
}

// CancelAllMarginHFOrdersBySymbol cancel all open high-frequency Margin orders(orders created through POST /api/v3/hf/margin/order).
// Transaction type: MARGIN_TRADE - cross margin trade, MARGIN_ISOLATED_TRADE - isolated margin trade
func (ku *Kucoin) CancelAllMarginHFOrdersBySymbol(ctx context.Context, symbol, tradeType string) (string, error) {
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	if tradeType == "" {
		return "", errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("tradeType", tradeType)
	var resp string
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelAllMarginHFOrdersBySymbolEPL, http.MethodDelete, common.EncodeURLValues("/v3/hf/margin/orders", params), nil, &resp)
}

// GetActiveMarginHFOrders retrieves list if active high-frequency margin orders
func (ku *Kucoin) GetActiveMarginHFOrders(ctx context.Context, symbol, tradeType string) ([]OrderDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	var resp []OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getActiveMarginHFOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/orders/active", params), nil, &resp)
}

// GetFilledHFMarginOrders list of filled margin HF orders and returns paginated data.
// The returned data is sorted in descending order based on the latest order update times.
func (ku *Kucoin) GetFilledHFMarginOrders(ctx context.Context, symbol, tradeType, side, orderType string, startAt, endAt time.Time, lastID, limit int64) (*FilledMarginHFOrdersResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("tradeType", tradeType)
	if side != "" {
		params.Set("side", strings.ToLower(side))
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
	if lastID > 0 {
		params.Set("lastId", strconv.FormatInt(lastID, 10))
	}
	if limit > 0 {
		params.Set(order.Limit.Lower(), strconv.FormatInt(limit, 10))
	}
	var resp *FilledMarginHFOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getFilledHFMarginOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/orders/done", params), nil, &resp)
}

// GetMarginHFOrderDetailByOrderID retrieves the detail of a HF margin order by order ID.
func (ku *Kucoin) GetMarginHFOrderDetailByOrderID(ctx context.Context, orderID, symbol string) (*OrderDetail, error) {
	return ku.GetMarginHFOrderDetailByID(ctx, orderID, symbol, "/v3/hf/margin/orders/")
}

// GetMarginHFOrderDetailByClientOrderID retrieves the detaul of a HF margin order by client order ID.
func (ku *Kucoin) GetMarginHFOrderDetailByClientOrderID(ctx context.Context, clientOrderID, symbol string) (*OrderDetail, error) {
	return ku.GetMarginHFOrderDetailByID(ctx, clientOrderID, symbol, "/v3/hf/margin/orders/client-order/")
}

// GetMarginHFOrderDetailByID sends an HTTP request to fetch margin high frequency orders by order ID or client supplied order ID.
func (ku *Kucoin) GetMarginHFOrderDetailByID(ctx context.Context, orderID, symbol, path string) (*OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderID)
	var resp *OrderDetail
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginHFOrderDetailByOrderIDEPL, http.MethodGet, path+orderID+symbolQuery+symbol, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%w, where orderId %s and symbol %s", order.ErrOrderNotFound, orderID, symbol)
	}
	return resp, nil
}

// GetMarginHFTradeFills to obtain a list of the latest margin HF transaction details. The returned results are paginated. The data is sorted in descending order according to time.
func (ku *Kucoin) GetMarginHFTradeFills(ctx context.Context, orderID, symbol, tradeType, side, orderType string, startAt, endAt time.Time, lastID, limit int64) (*HFMarginOrderTransaction, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if side != "" {
		params.Set("side", strings.ToLower(side))
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
	if lastID > 0 {
		params.Set("lastId", strconv.FormatInt(lastID, 10))
	}
	if limit > 0 {
		params.Set(order.Limit.Lower(), strconv.FormatInt(limit, 10))
	}
	var resp *HFMarginOrderTransaction
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginHFTradeFillsEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/fills", params), nil, &resp)
}

// CreateSubUser creates a new sub-user for the account.
func (ku *Kucoin) CreateSubUser(ctx context.Context, subAccountName, password, remarks, access string) (*SubAccount, error) {
	if subAccountName == "" {
		return nil, fmt.Errorf("%w, subaccount name is required", errInvalidSubAccountName)
	}
	if password == "" {
		return nil, errInvalidPassPhraseInstance
	}
	arg := &struct {
		SubAccountName string `json:"subName"`
		Password       string `json:"password"`
		Remarks        string `json:"remarks,omitempty"`
		Access         string `json:"access,omitempty"`
	}{
		SubAccountName: subAccountName,
		Password:       password,
		Remarks:        remarks,
		Access:         access,
	}
	var resp *SubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, createSubUserEPL, http.MethodPost, "/v2/sub/user/created", arg, &resp)
}

// GetSubAccountSpotAPIList used to obtain a list of Spot APIs pertaining to a sub-account.
func (ku *Kucoin) GetSubAccountSpotAPIList(ctx context.Context, subAccountName, apiKeys string) ([]SpotAPISubAccount, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subName", subAccountName)
	if apiKeys != "" {
		params.Set("apiKey", apiKeys)
	}
	var resp []SpotAPISubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, subAccountSpotAPIListEPL, http.MethodGet, common.EncodeURLValues("/v1/sub/api-key", params), nil, &resp)
}

// CreateSpotAPIsForSubAccount can be used to create Spot APIs for sub-accounts.
func (ku *Kucoin) CreateSpotAPIsForSubAccount(ctx context.Context, arg *SpotAPISubAccountParams) (*SpotAPISubAccount, error) {
	if arg.SubAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if arg.Passphrase == "" {
		return nil, fmt.Errorf("%w, must contain 7-32 characters. cannot contain any spaces", errInvalidPassPhraseInstance)
	}
	if arg.Remark == "" {
		return nil, errRemarkIsRequired
	}
	var resp *SpotAPISubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, createSpotAPIForSubAccountEPL, http.MethodPost, "/v1/sub/api-key", &arg, &resp)
}

// ModifySubAccountSpotAPIs modifies sub-account Spot APIs.
func (ku *Kucoin) ModifySubAccountSpotAPIs(ctx context.Context, arg *SpotAPISubAccountParams) (*SpotAPISubAccount, error) {
	if arg.SubAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if arg.APIKey == "" {
		return nil, errAPIKeyRequired
	}
	if arg.Passphrase == "" {
		return nil, fmt.Errorf("%w, must contain 7-32 characters. cannot contain any spaces", errInvalidPassPhraseInstance)
	}
	var resp *SpotAPISubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifySubAccountSpotAPIEPL, http.MethodPut, "/v1/sub/api-key/update", &arg, &resp)
}

// DeleteSubAccountSpotAPI delete sub-account Spot APIs.
func (ku *Kucoin) DeleteSubAccountSpotAPI(ctx context.Context, apiKey, subAccountName, passphrase string) (*DeleteSubAccountResponse, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if apiKey == "" {
		return nil, errAPIKeyRequired
	}
	if passphrase == "" {
		return nil, errInvalidPassPhraseInstance
	}
	params := url.Values{}
	params.Set("apiKey", apiKey)
	params.Set("subName", subAccountName)
	params.Set("passphrase", passphrase)
	var resp *DeleteSubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, deleteSubAccountSpotAPIEPL, http.MethodDelete, common.EncodeURLValues("/v1/sub/api-key", params), nil, &resp)
}

// GetUserInfoOfAllSubAccounts get the user info of all sub-users via this interface.
func (ku *Kucoin) GetUserInfoOfAllSubAccounts(ctx context.Context) (*SubAccountResponse, error) {
	var resp *SubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, allUserSubAccountsV2EPL, http.MethodGet, "/v2/sub/user", nil, &resp)
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifySubAccountAPIEPL, http.MethodGet, common.EncodeURLValues("/v1/sub/api-key/update", params), nil, &resp)
}

// GetAllAccounts get all accounts
// accountType possible values are main、trade、margin、trade_hf
func (ku *Kucoin) GetAllAccounts(ctx context.Context, ccy currency.Code, accountType string) ([]AccountInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if accountType != "" {
		params.Set("type", accountType)
	}
	var resp []AccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, allAccountEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts", params), nil, &resp)
}

// GetAccountDetail get information of single account
func (ku *Kucoin) GetAccountDetail(ctx context.Context, accountID string) (*AccountInfo, error) {
	if accountID == "" {
		return nil, errAccountIDMissing
	}
	var resp *AccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, accountDetailEPL, http.MethodGet, "/v1/accounts/"+accountID, nil, &resp)
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, crossMarginAccountsDetailEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/accounts", params), nil, &resp)
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, isolatedMarginAccountDetailEPL, http.MethodGet, common.EncodeURLValues("/v3/isolated/accounts", params), nil, &resp)
}

// GetFuturesAccountDetail retrieves futures account detail information
func (ku *Kucoin) GetFuturesAccountDetail(ctx context.Context, ccy currency.Code) (*FuturesAccountOverview, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp *FuturesAccountOverview
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresAccountsDetailEPL, http.MethodGet, common.EncodeURLValues("/v1/account-overview", params), nil, &resp)
}

// GetSubAccounts retrieves all sub-account information
func (ku *Kucoin) GetSubAccounts(ctx context.Context, subUserID string, includeBaseAmount bool) (*SubAccounts, error) {
	if subUserID == "" {
		return nil, fmt.Errorf("%w, sub-users ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if includeBaseAmount {
		params.Set("includeBaseAmount", "true")
	} else {
		params.Set("includeBaseAmount", "false")
	}
	var resp *SubAccounts
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, subAccountsEPL, http.MethodGet, common.EncodeURLValues("/v1/sub-accounts/"+subUserID, params), nil, &resp)
}

// GetAllFuturesSubAccountBalances retrieves all futures subaccount balances
func (ku *Kucoin) GetAllFuturesSubAccountBalances(ctx context.Context, ccy currency.Code) (*FuturesSubAccountBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp *FuturesSubAccountBalance
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, allFuturesSubAccountBalancesEPL, http.MethodGet, common.EncodeURLValues("/v1/account-overview-all", params), nil, &resp)
}

// populateParams populates account ledger request parameters.
func populateParams(ccy currency.Code, direction, bizType string, lastID, limit int64, startTime, endTime time.Time) url.Values {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
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
		params.Set(order.Limit.Lower(), strconv.FormatInt(limit, 10))
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
func (ku *Kucoin) GetAccountLedgers(ctx context.Context, ccy currency.Code, direction, bizType string, startAt, endAt time.Time) (*AccountLedgerResponse, error) {
	params := populateParams(ccy, direction, bizType, 0, 0, time.Time{}, time.Time{})
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *AccountLedgerResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, accountLedgersEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts/ledgers", params), nil, &resp)
}

// GetAccountLedgersHFTrade returns all transfer (in and out) records in high-frequency trading account and supports multi-coin queries.
// The query results are sorted in descending order by createdAt and id.
func (ku *Kucoin) GetAccountLedgersHFTrade(ctx context.Context, ccy currency.Code, direction, bizType string, lastID, limit int64, startTime, endTime time.Time) ([]LedgerInfo, error) {
	params := populateParams(ccy, direction, bizType, lastID, limit, startTime, endTime)
	var resp []LedgerInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfAccountLedgersEPL, http.MethodGet, common.EncodeURLValues("/v1/hf/accounts/ledgers", params), nil, &resp)
}

// GetAccountLedgerHFMargin returns all transfer (in and out) records in high-frequency margin trading account and supports multi-coin queries.
func (ku *Kucoin) GetAccountLedgerHFMargin(ctx context.Context, ccy currency.Code, direction, bizType string, lastID, limit int64, startTime, endTime time.Time) ([]LedgerInfo, error) {
	params := populateParams(ccy, direction, bizType, lastID, limit, startTime, endTime)
	var resp []LedgerInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, hfAccountLedgersMarginEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/account/ledgers", params), nil, &resp)
}

// GetFuturesAccountLedgers If there are open positions, the status of the first page returned will be Pending,
// indicating the realised profit and loss in the current 8-hour settlement period.
// Type RealisedPNL-Realised profit and loss, Deposit-Deposit, Withdrawal-withdraw, Transferin-Transfer in, TransferOut-Transfer out
func (ku *Kucoin) GetFuturesAccountLedgers(ctx context.Context, ccy currency.Code, forward bool, startAt, endAt time.Time, offset, maxCount int64) (*FuturesLedgerInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if forward {
		params.Set("forward", "true")
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount != 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesLedgerInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresAccountLedgersEPL, http.MethodGet, common.EncodeURLValues("/v1/transaction-history", params), nil, &resp)
}

// GetAllSubAccountsInfoV1 retrieves the user info of all sub-account via this interface.
func (ku *Kucoin) GetAllSubAccountsInfoV1(ctx context.Context) ([]SubAccount, error) {
	var resp []SubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, subAccountInfoV1EPL, http.MethodGet, "/v1/sub/user", nil, &resp)
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, allSubAccountsInfoV2EPL, http.MethodGet, "/v2/sub/user", nil, &resp)
}

// GetAccountSummaryInformation this can be used to obtain account summary information.
func (ku *Kucoin) GetAccountSummaryInformation(ctx context.Context) (*AccountSummaryInformation, error) {
	var resp *AccountSummaryInformation
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, accountSummaryInfoEPL, http.MethodGet, "/v2/user-info", nil, &resp)
}

// GetAggregatedSubAccountBalance get the account info of all sub-users
func (ku *Kucoin) GetAggregatedSubAccountBalance(ctx context.Context) ([]SubAccountInfo, error) {
	var resp []SubAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, subAccountBalancesEPL, http.MethodGet, "/v1/sub-accounts", nil, &resp)
}

// GetAllSubAccountsBalanceV2 retrieves sub-account balance information through the V2 API
func (ku *Kucoin) GetAllSubAccountsBalanceV2(ctx context.Context) (*SubAccountsBalanceV2, error) {
	var resp *SubAccountsBalanceV2
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, allSubAccountBalancesV2EPL, http.MethodGet, "/v2/sub-accounts", nil, &resp)
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, allSubAccountsBalanceEPL, http.MethodGet, common.EncodeURLValues("/v1/sub-accounts", params), nil, &resp)
}

// GetTransferableBalance get the transferable balance of a specified account
// The account type:MAIN、TRADE、TRADE_HF、MARGIN、ISOLATED
func (ku *Kucoin) GetTransferableBalance(ctx context.Context, ccy currency.Code, accountType, tag string) (*TransferableBalanceInfo, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if accountType == "" {
		return nil, errAccountTypeMissing
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("type", accountType)
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp *TransferableBalanceInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getTransferablesEPL, http.MethodGet, common.EncodeURLValues("/v1/accounts/transferable", params), nil, &resp)
}

// GetUniversalTransfer support transfer between master and sub accounts (only applicable to master account APIKey).
func (ku *Kucoin) GetUniversalTransfer(ctx context.Context, arg *UniversalTransferParam) (string, error) {
	if *arg == (UniversalTransferParam{}) {
		return "", common.ErrNilPointer
	}
	if arg.ClientSuppliedOrderID == "" {
		return "", order.ErrClientOrderIDMustBeSet
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, flexiTransferEPL, http.MethodPost, "/v3/accounts/universal-transfer", arg, &resp)
}

// TransferMainToSubAccount used to transfer funds from main account to sub-account
func (ku *Kucoin) TransferMainToSubAccount(ctx context.Context, ccy currency.Code, amount float64, clientOID, direction, accountType, subAccountType, subUserID string) (string, error) {
	if clientOID == "" {
		return "", order.ErrClientOrderIDMustBeSet
	}
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	if direction == "" {
		return "", errTransferDirectionRequired
	}
	if subUserID == "" {
		return "", fmt.Errorf("%w, sub-user ID is required", errSubUserIDRequired)
	}
	arg := &struct {
		ClientOrderID  string        `json:"clientOid"`
		SubUserID      string        `json:"subUserId"`
		Currency       currency.Code `json:"currency"`
		Amount         float64       `json:"amount,string"`
		Direction      string        `json:"direction"`
		AccountType    string        `json:"accountType,omitempty"`
		SubAccountType string        `json:"subAccountType,omitempty"`
	}{
		ClientOrderID:  clientOID,
		SubUserID:      subUserID,
		Currency:       ccy,
		Amount:         amount,
		Direction:      direction,
		AccountType:    accountType,
		SubAccountType: subAccountType,
	}
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, masterSubUserTransferEPL, http.MethodPost, "/v2/accounts/sub-transfer", arg, &resp)
}

// MakeInnerTransfer used to transfer funds between accounts internally
// possible account types: main, trade, trade_hf, margin, isolated, margin_v2, isolated_v2, contract
func (ku *Kucoin) MakeInnerTransfer(ctx context.Context, amount float64, ccy currency.Code, clientOID, paymentAccountType, receivingAccountType, fromTag, toTag string) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if clientOID == "" {
		return "", order.ErrClientOrderIDMustBeSet
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	if paymentAccountType == "" {
		return "", fmt.Errorf("%w sending account type is required", errAccountTypeMissing)
	}
	if receivingAccountType == "" {
		return "", fmt.Errorf("%w receiving account type is required", errAccountTypeMissing)
	}
	arg := &struct {
		ClientOrderID        string        `json:"clientOid"`
		Currency             currency.Code `json:"currency"`
		Amount               float64       `json:"amount,string"`
		PaymentAccountType   string        `json:"from"`
		ReceivingAccountType string        `json:"to"`
		FromTag              string        `json:"fromTag,omitempty"`
		ToTag                string        `json:"toTag,omitempty"`
	}{
		ClientOrderID:        clientOID,
		Amount:               amount,
		Currency:             ccy,
		PaymentAccountType:   paymentAccountType,
		ReceivingAccountType: receivingAccountType,
		FromTag:              fromTag,
		ToTag:                toTag,
	}
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, innerTransferEPL, http.MethodPost, "/v2/accounts/inner-transfer", arg, &resp)
}

// TransferToMainOrTradeAccount transfers fund from KuCoin Futures account to Main or Trade accounts.
func (ku *Kucoin) TransferToMainOrTradeAccount(ctx context.Context, arg *FundTransferFuturesParam) (*InnerTransferToMainAndTradeResponse, error) {
	if *arg == (FundTransferFuturesParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.RecieveAccountType != "MAIN" && arg.RecieveAccountType != SpotTradeType {
		return nil, fmt.Errorf("invalid receive account type %s, only TRADE and MAIN are supported", arg.RecieveAccountType)
	}
	var resp *InnerTransferToMainAndTradeResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, toMainOrTradeAccountEPL, http.MethodPost, "/v3/transfer-out", arg, &resp)
}

// TransferToFuturesAccount transfers fund from KuCoin Futures account to Main or Trade accounts.
func (ku *Kucoin) TransferToFuturesAccount(ctx context.Context, arg *FundTransferToFuturesParam) (*FundTransferToFuturesResponse, error) {
	if *arg == (FundTransferToFuturesParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.PaymentAccountType != "MAIN" && arg.PaymentAccountType != SpotTradeType {
		return nil, fmt.Errorf("invalid receive account type %s, only TRADE and MAIN are supported", arg.PaymentAccountType)
	}
	var resp *FundTransferToFuturesResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, toFuturesAccountEPL, http.MethodPost, "/v1/transfer-in", arg, &resp)
}

// GetFuturesTransferOutRequestRecords retrieves futures transfers out requests.
func (ku *Kucoin) GetFuturesTransferOutRequestRecords(ctx context.Context, startAt, endAt time.Time, status, queryStatus string, ccy currency.Code, currentPage, pageSize int64) (*FuturesTransferOutResponse, error) {
	params := url.Values{}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if status != "" {
		params.Set("status", status)
	}
	if queryStatus != "" {
		params.Set("queryStatus", queryStatus)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *FuturesTransferOutResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestFutures, futuresTransferOutRequestRecordsEPL, http.MethodGet, common.EncodeURLValues("/v1/transfer-list", params), nil, &resp)
}

// CreateDepositAddress create a deposit address for a currency you intend to deposit
func (ku *Kucoin) CreateDepositAddress(ctx context.Context, arg *DepositAddressParams) (*DepositAddress, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, createDepositAddressEPL, http.MethodPost, "/v1/deposit-addresses", arg, &resp)
}

// GetDepositAddressesV2 get all deposit addresses for the currency you intend to deposit
func (ku *Kucoin) GetDepositAddressesV2(ctx context.Context, ccy currency.Code) ([]DepositAddress, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp []DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, depositAddressesV2EPL, http.MethodGet, common.EncodeURLValues("/v2/deposit-addresses", params), nil, &resp)
}

// GetDepositAddressV1 get a deposit address for the currency you intend to deposit
func (ku *Kucoin) GetDepositAddressV1(ctx context.Context, ccy currency.Code, chain string) (*DepositAddress, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, depositAddressesV1EPL, http.MethodGet, common.EncodeURLValues("/v1/deposit-addresses", params), nil, &resp)
}

// GetDepositList get deposit list items and sorted to show the latest first
// Status. Available value: PROCESSING, SUCCESS, and FAILURE
func (ku *Kucoin) GetDepositList(ctx context.Context, ccy currency.Code, status string, startAt, endAt time.Time) (*DepositResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, depositListEPL, http.MethodGet, common.EncodeURLValues("/v1/deposits", params), nil, &resp)
}

// GetHistoricalDepositList get historical deposit list items
func (ku *Kucoin) GetHistoricalDepositList(ctx context.Context, ccy currency.Code, status string, startAt, endAt time.Time) (*HistoricalDepositWithdrawalResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, historicDepositListEPL, http.MethodGet, common.EncodeURLValues("/v1/hist-deposits", params), nil, &resp)
}

// GetWithdrawalList get withdrawal list items
func (ku *Kucoin) GetWithdrawalList(ctx context.Context, ccy currency.Code, status string, startAt, endAt time.Time) (*WithdrawalsResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
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
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, withdrawalListEPL, http.MethodGet, common.EncodeURLValues("/v1/withdrawals", params), nil, &resp)
}

// GetHistoricalWithdrawalList get historical withdrawal list items
func (ku *Kucoin) GetHistoricalWithdrawalList(ctx context.Context, ccy currency.Code, status string, startAt, endAt time.Time) (*HistoricalDepositWithdrawalResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
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
func (ku *Kucoin) GetWithdrawalQuotas(ctx context.Context, ccy currency.Code, chain string) (*WithdrawalQuota, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *WithdrawalQuota
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, withdrawalQuotaEPL, http.MethodGet, common.EncodeURLValues("/v1/withdrawals/quotas", params), nil, &resp)
}

// ApplyWithdrawal create a withdrawal request
// The endpoint was deprecated for futures, please transfer assets from the FUTURES account to the MAIN account first, and then withdraw from the MAIN account
// Withdrawal fee deduct types are: INTERNAL and EXTERNAL
//
// TIP: On the WEB end, you can open the switch of specified favorite addresses for withdrawal, and when it is turned on,
// it will verify whether your withdrawal address(including chain) is a favorite address(it is case sensitive); if it fails validation,
// it will respond with the error message {"msg":"Already set withdraw whitelist, this address is not favorite address","code":"260325"}.
func (ku *Kucoin) ApplyWithdrawal(ctx context.Context, ccy currency.Code, address, memo, remark, chain, feeDeductType string, isInner bool, amount float64) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if address == "" {
		return "", fmt.Errorf("%w, empty withdrawal address", errAddressRequired)
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	arg := &struct {
		Currency      currency.Code `json:"currency"`
		Address       string        `json:"address"`
		Amount        float64       `json:"amount"`
		IsInner       bool          `json:"isInner"`
		Memo          string        `json:"memo,omitempty"`
		Remark        string        `json:"remark,omitempty"`
		Chain         string        `json:"chain,omitempty"`
		FeeDeductType string        `json:"feeDeductType,omitempty"`
	}{
		Currency:      ccy,
		Address:       address,
		Amount:        amount,
		Memo:          memo,
		IsInner:       isInner,
		Remark:        remark,
		Chain:         chain,
		FeeDeductType: feeDeductType,
	}
	resp := struct {
		WithdrawalID string `json:"withdrawalId"`
		Error
	}{}
	return resp.WithdrawalID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, applyWithdrawalEPL, http.MethodPost, "/v1/withdrawals", arg, &resp)
}

// CancelWithdrawal used to cancel a withdrawal request
func (ku *Kucoin) CancelWithdrawal(ctx context.Context, withdrawalID string) error {
	if withdrawalID == "" {
		return fmt.Errorf("%w withdrawal ID is required", order.ErrOrderIDNotSet)
	}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalsEPL, http.MethodDelete, "/v1/withdrawals/"+withdrawalID, nil, &struct{}{})
}

// GetBasicFee get basic fee rate of users
// Currency type: '0'-crypto currency, '1'-fiat currency. default is '0'-crypto currency
func (ku *Kucoin) GetBasicFee(ctx context.Context, currencyType string) (*Fees, error) {
	params := url.Values{}
	if currencyType != "" {
		params.Set("currencyType", currencyType)
	}
	var resp *Fees
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, basicFeesEPL, http.MethodGet, common.EncodeURLValues("/v1/base-fee", params), nil, &resp)
}

// GetTradingFee get fee rate of trading pairs
// WARNING: There is a limit of 10 currency pairs allowed to be requested per call.
func (ku *Kucoin) GetTradingFee(ctx context.Context, pairs currency.Pairs) ([]Fees, error) {
	if len(pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	var resp []Fees
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, tradeFeesEPL, http.MethodGet, "/v1/trade-fees?symbols="+pairs.Upper().Join(), nil, &resp)
}

// ----------------------------------------------------------  Lending Market ----------------------------------------------------------------------------

// GetLendingCurrencyInformation retrieves a lending currency information.
func (ku *Kucoin) GetLendingCurrencyInformation(ctx context.Context, ccy currency.Code) ([]LendingCurrencyInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []LendingCurrencyInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, lendingCurrencyInfoEPL, http.MethodGet, common.EncodeURLValues("/v3/project/list", params), nil, &resp)
}

// GetInterestRate retrieves the interest rates of the margin lending market over the past 7 days.
func (ku *Kucoin) GetInterestRate(ctx context.Context, ccy currency.Code) ([]InterestRate, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []InterestRate
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, interestRateEPL, http.MethodGet, "/v3/project/marketInterestRate?currency="+ccy.String(), nil, &resp)
}

// MarginLendingSubscription retrieves margin lending subscription information.
func (ku *Kucoin) MarginLendingSubscription(ctx context.Context, ccy currency.Code, size, interestRate float64) (*OrderNumberResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if interestRate <= 0 {
		return nil, errMissingInterestRate
	}
	arg := &struct {
		Currency     currency.Code `json:"currency"`
		Size         float64       `json:"size,string"`
		InterestRate float64       `json:"interestRate"`
	}{
		Currency:     ccy,
		Size:         size,
		InterestRate: interestRate,
	}
	var resp *OrderNumberResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, marginLendingSubscriptionEPL, http.MethodPost, "/v3/purchase", arg, &resp)
}

// Redemption initiate redemptions of margin lending.
func (ku *Kucoin) Redemption(ctx context.Context, ccy currency.Code, size float64, purchaseOrderNo string) (*OrderNumberResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if purchaseOrderNo == "" {
		return nil, errMissingPurchaseOrderNumber
	}
	arg := &struct {
		Currency            currency.Code `json:"currency"`
		Size                float64       `json:"size,string"`
		PurchaseOrderNumber string        `json:"purchaseOrderNo"`
	}{
		Currency:            ccy,
		Size:                size,
		PurchaseOrderNumber: purchaseOrderNo,
	}
	var resp *OrderNumberResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, redemptionEPL, http.MethodPost, "/v3/redeem", arg, &resp)
}

// ModifySubscriptionOrder is used to update the interest rates of subscription orders, which will take effect at the beginning of the next hour.
func (ku *Kucoin) ModifySubscriptionOrder(ctx context.Context, ccy currency.Code, purchaseOrderNo string, interestRate float64) (*ModifySubscriptionOrderResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if interestRate <= 0 {
		return nil, errMissingInterestRate
	}
	if purchaseOrderNo == "" {
		return nil, errMissingPurchaseOrderNumber
	}
	arg := map[string]any{
		"currency":        ccy.String(),
		"interestRate":    interestRate,
		"purchaseOrderNo": purchaseOrderNo,
	}
	var resp *ModifySubscriptionOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifySubscriptionEPL, http.MethodPost, "/v3/lend/purchase/update", arg, &resp)
}

// GetRedemptionOrders query for the redemption orders.
// Status: DONE-completed; PENDING-settling
func (ku *Kucoin) GetRedemptionOrders(ctx context.Context, ccy currency.Code, status, redeemOrderNo string, currentPage, pageSize int64) (*RedemptionOrdersResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if status == "" {
		return nil, errStatusMissing
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("status", status)
	if redeemOrderNo != "" {
		params.Set("redeemOrderNo", redeemOrderNo)
	}
	if currentPage > 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *RedemptionOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getRedemptionOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/redeem/orders", params), nil, &resp)
}

// GetSubscriptionOrders provides pagination query for the subscription orders.
func (ku *Kucoin) GetSubscriptionOrders(ctx context.Context, ccy currency.Code, purchaseOrderNo, status string, currentPage, pageSize int64) (*PurchaseSubscriptionOrdersResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if status == "" {
		return nil, errStatusMissing
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("status", status)
	if purchaseOrderNo != "" {
		params.Set("purchaseOrderNo", purchaseOrderNo)
	}
	if currentPage > 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *PurchaseSubscriptionOrdersResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, getSubscriptionOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/purchase/orders", params), nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (ku *Kucoin) SendHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, path string, result any) error {
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
			HTTPRecording: ku.HTTPRecording,
		}, nil
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
func (ku *Kucoin) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, method, path string, arg, result any) error {
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
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		signHash, err := crypto.GetHMAC(crypto.HashSHA256, []byte(ts+method+"/api"+path+string(payload)), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		passPhraseHash, err := crypto.GetHMAC(crypto.HashSHA256, []byte(creds.ClientID), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"KC-API-KEY":         creds.Key,
			"KC-API-SIGN":        base64.StdEncoding.EncodeToString(signHash),
			"KC-API-TIMESTAMP":   ts,
			"KC-API-PASSPHRASE":  base64.StdEncoding.EncodeToString(passPhraseHash),
			"KC-API-KEY-VERSION": "3",
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
			HTTPRecording: ku.HTTPRecording,
		}, nil
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

// IntervalToString returns a string from kline.Interval input.
func IntervalToString(interval kline.Interval) (string, error) {
	intervalString, okay := intervalMap[interval]
	if okay {
		return intervalString, nil
	}
	return "", fmt.Errorf("%w interval: %v", kline.ErrUnsupportedInterval, interval)
}

// StringToOrderStatus returns an order.Status instance from string.
func (ku *Kucoin) StringToOrderStatus(status string) (order.Status, error) {
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

// AccountToTradeTypeString returns the account trade type given the asset type and margin mode information for spot and margin assets.
func (ku *Kucoin) AccountToTradeTypeString(a asset.Item, marginMode string) string {
	switch a {
	case asset.Spot:
		return SpotTradeType
	case asset.Margin:
		if strings.EqualFold(marginMode, "isolated") {
			return IsolatedMarginTradeType
		}
		return CrossMarginTradeType
	default:
		return ""
	}
}

// OrderSideString converts an order.Side instance to a string representation
func (ku *Kucoin) OrderSideString(side order.Side) (string, error) {
	switch {
	case side.IsLong():
		return order.Buy.Lower(), nil
	case side.IsShort():
		return order.Sell.Lower(), nil
	case side == order.AnySide:
		return "", nil
	default:
		return "", fmt.Errorf("%w, side:%s", order.ErrSideIsInvalid, side.Lower())
	}
}

// GetTradingPairActualFees retrieves list of trading pairs and fees.
func (ku *Kucoin) GetTradingPairActualFees(ctx context.Context, symbols []string) ([]TradingPairFee, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp []TradingPairFee
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, tradingPairActualFeeEPL, http.MethodGet, common.EncodeURLValues("/v1/trade-fees", params), nil, &resp)
}

// -----------------------------------------------------------  Earn Endpoints  ----------------------------------------------------------------

// SubscribeToEarnFixedIncomeProduct allows subscribing to fixed income products. If the subscription fails, it returns the corresponding error code.
func (ku *Kucoin) SubscribeToEarnFixedIncomeProduct(ctx context.Context, productID, accountType string, amount float64) (*SusbcribeEarn, error) {
	if productID == "" {
		return nil, errProductIDMissing
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if accountType == "" {
		return nil, errAccountTypeMissing
	}
	var resp *SusbcribeEarn
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, subscribeToEarnEPL, http.MethodPost, "/v1/earn/orders",
		&map[string]any{
			"productId":     productID,
			"accountType":   accountType,
			"amount,string": amount,
		}, &resp)
}

// RedeemByEarnHoldingID allows initiating redemption by holding ID.
// If the current holding is fully redeemed or in the process of being redeemed, it indicates that the holding does not exist.
// Confirmation field for early redemption penalty: 1 (confirm early redemption, and the current holding will be fully redeemed).
// This parameter is valid only for fixed-term products
func (ku *Kucoin) RedeemByEarnHoldingID(ctx context.Context, orderID, fromAccountType, confirmPunishRedeem string, amount float64) (*EarnRedeem, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if fromAccountType != "" {
		params.Set("fromAccountType", fromAccountType)
	}
	if confirmPunishRedeem != "" {
		params.Set("confirmPunishRedeem", confirmPunishRedeem)
	}
	var resp *EarnRedeem
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, earnRedemptionEPL, http.MethodDelete, common.EncodeURLValues("/v1/earn/orders", params), nil, &resp)
}

// GetEarnRedeemPreviewByHoldingID retrieves redemption preview information by holding ID.
// If the current holding is fully redeemed or in the process of being redeemed, it indicates that the holding does not exist.
func (ku *Kucoin) GetEarnRedeemPreviewByHoldingID(ctx context.Context, orderID, fromAccountType string) (*EarnRedemptionPreview, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	if fromAccountType != "" {
		params.Set("fromAccountType", fromAccountType)
	}
	var resp *EarnRedemptionPreview
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, earnRedemptionPreviewEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/redeem-preview", params), nil, &resp)
}

// ---------------------------------------------------------------- Kucoin Earn ----------------------------------------------------------------

// GetEarnSavingsProducts retrieves savings products. If no savings products are available, an empty list is returned.
func (ku *Kucoin) GetEarnSavingsProducts(ctx context.Context, ccy currency.Code) ([]EarnSavingProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []EarnSavingProduct
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, kucoinEarnSavingsProductsEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/saving/products", params), nil, &resp)
}

// GetEarnFixedIncomeCurrentHoldings retrieves current holding assets of fixed income products. If no current holding assets are available, an empty list is returned.
func (ku *Kucoin) GetEarnFixedIncomeCurrentHoldings(ctx context.Context, productID, productCategory string, ccy currency.Code, currentPage, pageSize int64) (*FixedIncomeEarnHoldings, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	if productCategory != "" {
		params.Set("productCategory", productCategory)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if currentPage > 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *FixedIncomeEarnHoldings
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, kucoinEarnFixedIncomeCurrentHoldingEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/hold-assets", params), nil, &resp)
}

// GetLimitedTimePromotionProducts retrieves limited-time promotion products. If no products are available, an empty list is returned.
func (ku *Kucoin) GetLimitedTimePromotionProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []EarnProduct
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, earnLimitedTimePromotionProductEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/promotion/products", params), nil, &resp)
}

// ---------------------------------------------------------------- Staking Endpoints ----------------------------------------------------------------

// GetEarnKCSStakingProducts retrieves KCS Staking products. If no KCS Staking products are available, an empty list is returned.
func (ku *Kucoin) GetEarnKCSStakingProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []EarnProduct
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, earnKCSStakingProductEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/kcs-staking/products", params), nil, &resp)
}

// GetEarnStakingProducts retrieves staking products. If no staking products are available, an empty list is returned.
func (ku *Kucoin) GetEarnStakingProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []EarnProduct
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, earnStakingProductEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/staking/products", params), nil, &resp)
}

// GetEarnETHStakingProducts retrieves ETH Staking products. If no ETH Staking products are available, an empty list is returned.
func (ku *Kucoin) GetEarnETHStakingProducts(ctx context.Context) ([]EarnProduct, error) {
	var resp []EarnProduct
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, earnStakingProductEPL, http.MethodGet, "/v1/earn/eth-staking/products", nil, &resp)
}

// ---------------------------------------------------------------- VIP Lending ----------------------------------------------------------------

// GetInformationOnOffExchangeFundingAndLoans retrieves accounts that are currently involved in loans.
func (ku *Kucoin) GetInformationOnOffExchangeFundingAndLoans(ctx context.Context) (*OffExchangeFundingAndLoan, error) {
	var resp *OffExchangeFundingAndLoan
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, vipLendingEPL, http.MethodGet, "/v1/otc-loan/loan", nil, &resp)
}

// GetInformationOnAccountInvolvedInOffExchangeLoans retrieves accounts that are currently involved in off-exchange loans.
func (ku *Kucoin) GetInformationOnAccountInvolvedInOffExchangeLoans(ctx context.Context) ([]VIPLendingAccounts, error) {
	var resp []VIPLendingAccounts
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, vipLendingEPL, http.MethodGet, "/v1/otc-loan/accounts", nil, &resp)
}

// GetAffilateUserRebateInformation allows getting affiliate user rebate information.
func (ku *Kucoin) GetAffilateUserRebateInformation(ctx context.Context, date time.Time, offset string, maxCount int64) ([]UserRebateInfo, error) {
	if date.IsZero() {
		return nil, errQueryDateIsRequired
	}
	if offset == "" {
		return nil, errOffsetIsRequired
	}
	params := url.Values{}
	formattedDate := date.Format("20060102")
	params.Set("date", formattedDate)
	params.Set("offset", offset)
	if maxCount > 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp []UserRebateInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, affilateUserRebateInfoEPL, http.MethodGet, common.EncodeURLValues("/v2/affiliate/inviter/statistics", params), nil, &resp)
}

// GetMarginPairsConfigurations allows querying the configuration of cross margin trading pairs.
func (ku *Kucoin) GetMarginPairsConfigurations(ctx context.Context, symbol string) (*MarginPairConfigs, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *MarginPairConfigs
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, marginPairsConfigurationEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/symbols", params), nil, &resp)
}

// ModifyLeverageMultiplier this endpoint allows modifying the leverage multiplier for cross margin or isolated margin
func (ku *Kucoin) ModifyLeverageMultiplier(ctx context.Context, symbol string, leverage int64, isIsolated bool) error {
	if leverage <= 0 {
		return errInvalidLeverage
	}
	arg := &struct {
		Symbol     string `json:"symbol,omitempty"`
		Leverage   int64  `json:"leverage"`
		IsIsolated bool   `json:"isIsolated,omitempty"`
	}{
		Symbol:     symbol,
		Leverage:   leverage,
		IsIsolated: isIsolated,
	}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyLeverageMultiplierEPL, http.MethodPost, "/v3/position/update-user-leverage", arg, &struct{}{})
}

// GetActiveHFOrderSymbols retrieves the symbols of active high-frequency orders.
// Possible values for tradeType are MARGIN_TRADE for cross-margin trading
// and MARGIN_ISOLATED_TRADE for isolated margin trading.
func (ku *Kucoin) GetActiveHFOrderSymbols(ctx context.Context, tradeType string) (*MarginActiveSymbolDetail, error) {
	if tradeType == "" {
		return nil, errTradeTypeMissing
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	var resp *MarginActiveSymbolDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, marginActiveHFOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/hf/margin/order/active/symbols", params), nil, &resp)
}
