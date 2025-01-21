package mexc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// MEXC is the overarching type across this package
type MEXC struct {
	exchange.Base
}

const (
	mexcAPIURL     = "https://api.mexc.com"
	mexcWSAPIURL   = "https://api.mexc.com"
	mexcAPIVersion = "/v3/"

	// Public endpoints

	// Authenticated endpoints
)

var (
	errInvalidSubAccountName      = errors.New("invalid sub-account name")
	errInvalidSubAccountNote      = errors.New("invalid sub-account note")
	errUnsupportedPermissionValue = errors.New("permission is unsupported")
)

// Start implementing public and private exchange API funcs below

// GetSymbols retrieves current exchange trading rules and symbol information
func (me *MEXC) GetSymbols(ctx context.Context, symbols []string) (*ExchangeConfig, error) {
	params := url.Values{}
	if len(symbols) > 1 {
		params.Set("symbols", strings.Join(symbols, ","))
	} else if len(symbols) == 1 {
		params.Set("symbol", symbols[0])
	}
	var resp *ExchangeConfig
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "exchangeInfo", params, &resp)
}

// GetSystemTime check server time
func (me *MEXC) GetSystemTime(ctx context.Context) (types.Time, error) {
	resp := &struct {
		ServerTime types.Time `json:"serverTime"`
	}{}
	return resp.ServerTime, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "time", nil, &resp)
}

// GetDefaultSumbols retrieves all default symbols
func (me *MEXC) GetDefaultSumbols(ctx context.Context) ([]string, error) {
	resp := &struct {
		Symbols []string `json:"data"`
	}{}
	return resp.Symbols, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "defaultSymbols", nil, &resp)
}

// GetOrderbook retrieves orderbook data of a symbol
func (me *MEXC) GetOrderbook(ctx context.Context, symbol string, limit int64) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *Orderbook
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "depth", params, &resp)
}

// GetRecentTradesList retrieves recent trades list
func (me *MEXC) GetRecentTradesList(ctx context.Context, symbol string, limit int64) ([]TradeDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradeDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "trades", params, &resp)
}

// GetAggregatedTrades get compressed, aggregate trades. Trades that fill at the time, from the same order, with the same price will have the quantity aggregated.
func (me *MEXC) GetAggregatedTrades(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) ([]AggregatedTradeDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AggregatedTradeDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "aggTrades", params, &resp)
}

var intervalToStringMap = map[kline.Interval]string{kline.OneMin: "1m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m", kline.OneHour: "60m", kline.FourHour: "4h", kline.OneDay: "1d", kline.OneWeek: "1W", kline.OneMonth: "1M"}

func intervalToString(interval kline.Interval) (string, error) {
	intervalString, ok := intervalToStringMap[interval]
	if !ok {
		return "", kline.ErrUnsupportedInterval
	}
	return intervalString, nil
}

// GetCandlestick retrieves kline/candlestick bars for a symbol.
// Klines are uniquely identified by their open time.
func (me *MEXC) GetCandlestick(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]CandlestickData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", intervalString)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CandlestickData
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "klines", params, &resp)
}

// GetCurrentAveragePrice retrieves current average price of symbol
func (me *MEXC) GetCurrentAveragePrice(ctx context.Context, symbol string) (*SymbolAveragePrice, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *SymbolAveragePrice
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "avgPrice", params, &resp)
}

// Get24HourTickerPriceChangeStatistics retrieves ticker price change statistics
func (me *MEXC) Get24HourTickerPriceChangeStatistics(ctx context.Context, symbols []string) (TickerList, error) {
	params := url.Values{}
	if len(symbols) > 1 {
		params.Set("symbols", strings.Join(symbols, ","))
	} else if len(symbols) == 1 {
		params.Set("symbol", symbols[0])
	}
	var resp TickerList
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "ticker/24hr", params, &resp)
}

// GetSymbolPriceTicker represents a symbol price ticker detail
func (me *MEXC) GetSymbolPriceTicker(ctx context.Context, symbols []string) ([]SymbolPriceTicker, error) {
	params := url.Values{}
	if len(symbols) > 1 {
		params.Set("symbols", strings.Join(symbols, ","))
	} else if len(symbols) == 1 {
		params.Set("symbol", symbols[0])
	}
	var resp SymbolPriceTickers
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "ticker/price", params, &resp)
}

// GetSymbolOrderbookTicker represents an orderbook detail for a symbol
func (me *MEXC) GetSymbolOrderbookTicker(ctx context.Context, symbol string) ([]SymbolOrderbookTicker, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp SymbolOrderbookTickerList
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "ticker/bookTicker", params, &resp)
}

// CreateSubAccount create a sub-account from the master account.
func (me *MEXC) CreateSubAccount(ctx context.Context, subAccountName, note string) (*SubAccountCreationResponse, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if note == "" {
		return nil, errInvalidSubAccountNote
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	params.Set("note", note)
	var resp *SubAccountCreationResponse
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "sub-account/virtualSubAccount", params, &resp, true)
}

// GetSubAccountList get details of the sub-account list
func (me *MEXC) GetSubAccountList(ctx context.Context, subAccountName string, isFreeze bool, page, limit int64) (*SubAccounts, error) {
	params := url.Values{}
	if subAccountName != "" {
		params.Set("subAccount", subAccountName)
	}
	if isFreeze {
		params.Set("isFreeze", "true")
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *SubAccounts
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "sub-account/list", params, &resp, true)
}

// CreateAPIKeyForSubAccount creates an API key for sub-account
// Permission of APIKey: SPOT_ACCOUNT_READ, SPOT_ACCOUNT_WRITE, SPOT_DEAL_READ, SPOT_DEAL_WRITE, CONTRACT_ACCOUNT_READ, CONTRACT_ACCOUNT_WRITE, CONTRACT_DEAL_READ,
// CONTRACT_DEAL_WRITE, SPOT_TRANSFER_READ, SPOT_TRANSFER_WRITE
func (me *MEXC) CreateAPIKeyForSubAccount(ctx context.Context, subAccountName, note, permissions, ip string) (interface{}, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if note == "" {
		return nil, errInvalidSubAccountNote
	}
	if permissions == "" {
		return nil, errUnsupportedPermissionValue
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	params.Set("note", note)
	params.Set("permissions", permissions)
	if ip != "" {
		params.Set("ip", ip)
	}
	var resp *SubAccountAPIDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "sub-account/apiKey", params, &resp, true)
}

// GetSubAccountAPIKey applies to master accounts only
func (me *MEXC) GetSubAccountAPIKey(ctx context.Context, subAccountName string) (*SubAccountsAPIs, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	var resp *SubAccountsAPIs
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "sub-account/apiKey", params, &resp, true)
}

// DeleteAPIKeySubAccount delete the API Key of a sub-account
func (me *MEXC) DeleteAPIKeySubAccount(ctx context.Context, subAccountName string) (string, error) {
	if subAccountName == "" {
		return "", errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	resp := &struct {
		SubAccount string `json:"subAccount"`
	}{}
	return resp.SubAccount, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "sub-account/apiKey", params, &resp, true)
}

// UniversalTransfer requires SPOT_TRANSFER_WRITE permission
func (me *MEXC) UniversalTransfer(ctx context.Context, fromAccount, toAccount string, fromAccountType, toAccountType asset.Item, ccy currency.Code, amount float64) (*UniversalTransferResponse, error) {
	if !me.SupportsAsset(fromAccountType) {
		return nil, fmt.Errorf("%w fromAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	if !me.SupportsAsset(toAccountType) {
		return nil, fmt.Errorf("%w toAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	if ccy.IsEmpty() {
		return nil, fmt.Errorf("%w, asset %v", currency.ErrCurrencyCodeEmpty, ccy)
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType.String())
	params.Set("toAccountType", toAccountType.String())
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if fromAccount != "" {
		params.Set("fromAccount", fromAccount)
	}
	if toAccount != "" {
		params.Set("toAccount", toAccount)
	}
	var resp *UniversalTransferResponse
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "capital/sub-account/universalTransfer", params, &resp, true)
}

// GetUnversalTransferHistory retrieves universal assets transfer history of master account
func (me *MEXC) GetUnversalTransferHistory(ctx context.Context, fromAccount, toAccount string, fromAccountType, toAccountType asset.Item, startTime, endTime time.Time, page, limit int64) (*UniversalTransferHistoryData, error) {
	if !me.SupportsAsset(fromAccountType) {
		return nil, fmt.Errorf("%w fromAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	if !me.SupportsAsset(toAccountType) {
		return nil, fmt.Errorf("%w toAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("fromAccountType", fromAccountType.String())
	params.Set("toAccountType", toAccountType.String())
	if fromAccount != "" {
		params.Set("fromAccount", fromAccount)
	}
	if toAccount != "" {
		params.Set("toAccount", toAccount)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *UniversalTransferHistoryData
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "capital/sub-account/universalTransfer", params, resp, true)
}

// GetSubAccountAsset represents a sub-account asset balance detail
func (me *MEXC) GetSubAccountAsset(ctx context.Context, subAccount string, accountType asset.Item) (*SubAccountAssetBalances, error) {
	if subAccount == "" {
		return nil, errInvalidSubAccountName
	}
	if accountType == asset.Empty {
		return nil, asset.ErrNotSupported
	}
	params := url.Values{}
	params.Set("subAccount", subAccount)
	params.Set("accountType", accountType.String())
	var resp *SubAccountAssetBalances
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "sub-account/asset", params, &resp, true)
}

// GetKYCStatus retrieves accounts KYC(know your customer) status
func (me *MEXC) GetKYCStatus(ctx context.Context) (*KYCStatusInfo, error) {
	var resp *KYCStatusInfo
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "kyc/status", nil, &resp, true)
}

// UserAPIDefaultSymbols retrieves a default user API symbols
func (me *MEXC) UseAPIDefaultSymbols(ctx context.Context) (interface{}, error) {
	resp := &struct {
		Data []string `json:"data"`
	}{}
	return resp.Data, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "selfSymbols", nil, &resp, true)
}

// NewTestOrder creates and validates a new order but does not send it into the matching engine.
func (me *MEXC) NewTestOrder(ctx context.Context, symbol, newClientOrderID, side, orderType string, quantity, quoteOrderQty, price float64) (*OrderDetail, error) {
	return me.newOrder(ctx, symbol, newClientOrderID, side, orderType, "order/test", quantity, quoteOrderQty, price)
}

// NewOrder creates a new order
func (me *MEXC) NewOrder(ctx context.Context, symbol, newClientOrderID, side, orderType string, quantity, quoteOrderQty, price float64) (*OrderDetail, error) {
	return me.newOrder(ctx, symbol, newClientOrderID, side, orderType, "order", quantity, quoteOrderQty, price)
}

func (me *MEXC) newOrder(ctx context.Context, symbol, newClientOrderID, side, orderType, path string, quantity, quoteOrderQty, price float64) (*OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if orderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	orderType = strings.ToUpper(orderType)
	switch orderType {
	case "LIMIT_ORDER":
		if quantity <= 0 {
			return nil, fmt.Errorf("%w, quantity %v", order.ErrAmountBelowMin, quantity)
		}
		if price <= 0 {
			return nil, fmt.Errorf("%w, price %v", order.ErrPriceBelowMin, price)
		}
	case "MARKET_ORDER":
		if quantity <= 0 && quoteOrderQty <= 0 {
			return nil, fmt.Errorf("%w, either quantity or quote order quantity must be filled", order.ErrAmountBelowMin)
		}
	default:
		return nil, fmt.Errorf("%w, order type %s", order.ErrUnsupportedOrderType, orderType)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("type", orderType)
	if quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if quoteOrderQty != 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(quoteOrderQty, 'f', -1, 64))
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	var resp *OrderDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, path, params, &resp, true)
}

// OrderTypeString returns a string representation of an order.Type instance.
func (me *MEXC) OrderTypeString(oType order.Type) (string, error) {
	switch oType {
	case order.Limit:
		return "LIMIT_ORDER", nil
	case order.PostOnly:
		return "POST_ONLY", nil
	case order.Market:
		return "MARKET_ORDER", nil
	case order.ImmediateOrCancel:
		return "IMMEDIATE_OR_CANCEL", nil
	case order.FillOrKill:
		return "FILL_OR_KILL", nil
	case order.StopLimit:
		return "STOP_LIMIT", nil
	default:
		return "", order.ErrUnsupportedOrderType
	}
}

// CreateBatchOrder creates utmost 30 orders with a same symbol in a batch,rate limit:2 times/s.
func (me *MEXC) CreateBatchOrder(ctx context.Context, args []BatchOrderCreationParam) ([]OrderDetail, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for a := range args {
		if args[a] == (BatchOrderCreationParam{}) {
			return nil, common.ErrEmptyParams
		}
		if args[a].Symbol == "" {
			return nil, currency.ErrSymbolStringEmpty
		}
		if args[a].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		args[a].OrderType = strings.ToUpper(args[a].OrderType)
		switch args[a].OrderType {
		case "LIMIT_ORDER":
			if args[a].Quantity <= 0 {
				return nil, fmt.Errorf("%w, quantity %v", order.ErrAmountBelowMin, args[a].Quantity)
			}
			if args[a].Price <= 0 {
				return nil, fmt.Errorf("%w, price %v", order.ErrPriceBelowMin, args[a].Price)
			}
		case "MARKET_ORDER":
			if args[a].Quantity <= 0 && args[a].QuoteOrderQty <= 0 {
				return nil, fmt.Errorf("%w, either quantity or quote order quantity must be filled", order.ErrAmountBelowMin)
			}
		default:
			return nil, fmt.Errorf("%w, order type %s", order.ErrUnsupportedOrderType, args[a].OrderType)
		}
	}
	jsonString, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(jsonString))
	var resp []OrderDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "batchOrders", params, &resp, true)
}

// CancelTradeOrder cancels an order
func (me *MEXC) CancelTradeOrder(ctx context.Context, symbol string, orderID, clientOrderID, newClientOrderID string) (*OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if clientOrderID != "" {
		params.Set("origClientOrderId", clientOrderID)
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	var resp *OrderDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "order", params, &resp, true)
}

// CancelAllOpenOrdersBySymbol cancel all pending orders for a single symbol, including OCO pending orders.
func (me *MEXC) CancelAllOpenOrdersBySymbol(ctx context.Context, symbol string) ([]OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []OrderDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "openOrders", params, &resp, true)
}

// GetOrderByID retrieves a single order
func (me *MEXC) GetOrderByID(ctx context.Context, symbol, clientOrderID, orderID string) (*OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if clientOrderID == "" && orderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if clientOrderID != "" {
		params.Set("origClientOrderId", clientOrderID)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	var resp *OrderDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "order", params, &resp, true)
}

// GetOpenOrders retrieves all open orders on a symbol. Careful when accessing this with no symbol.
func (me *MEXC) GetOpenOrders(ctx context.Context, symbol string) ([]OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []OrderDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "openOrders", params, &resp, true)
}

// GetAllOrders retrieves all account orders including active, cancelled or completed orders(the query period is the latest 24 hours by default).
// You can query a maximum of the latest 7 days.
func (me *MEXC) GetAllOrders(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) ([]OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []OrderDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "allOrders", params, &resp, true)
}

// GetAccountInformation retrieves current account information,rate limit:2 times/s.
func (me *MEXC) GetAccountInformation(ctx context.Context) (*AccountDetail, error) {
	var resp *AccountDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "account", nil, &resp, true)
}

// GetAccountTradeList retrieves trades for a specific account and symbol,Only the transaction records in the past 1 month can be queried.
func (me *MEXC) GetAccountTradeList(ctx context.Context, symbol, orderID string, startTime, endTime time.Time, limit int64) ([]AccountTrade, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountTrade
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "myTrades", params, &resp, true)
}

// EnableMXDeduct enable or disable MX deduct for spot commission fee
func (me *MEXC) EnableMXDeduct(ctx context.Context, mxDeductEnable bool) (*MXDeductResponse, error) {
	params := url.Values{}
	if mxDeductEnable {
		params.Set("mxDeductEnable", "true")
	} else {
		params.Set("mxDeductEnable", "false")
	}
	var resp *MXDeductResponse
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "mxDeduct/enable", params, &resp, true)
}

// GetMXDeductStatus retrieves MX deduct status detail
func (me *MEXC) GetMXDeductStatus(ctx context.Context) (*MXDeductResponse, error) {
	var resp *MXDeductResponse
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "mxDeduct/enable", nil, &resp, true)
}

// GetSymbolTradingFee retrieves symbol commissions
func (me *MEXC) GetSymbolTradingFee(ctx context.Context, symbol string) (*SymbolCommissionFee, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *SymbolCommissionFee
	return resp, me.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "tradeFee", params, &resp, true)
}

// SendHTTPRequest sends an http request to a desired path with a JSON payload (of present)
func (me *MEXC) SendHTTPRequest(ctx context.Context, ep exchange.URL, f request.EndpointLimit, method, requestPath string, values url.Values, result interface{}, auth ...bool) error {
	ePoint, err := me.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	var authType request.AuthType
	authType = request.UnauthenticatedRequest
	if len(auth) > 0 && auth[0] {
		authType = request.AuthenticatedRequest
		creds, err := me.GetCredentials(ctx)
		if err != nil {
			return err
		}
		headers["X-MEXC-APIKEY"] = creds.Key
		if values != nil {
			values = url.Values{}
		}
		values.Set("recvWindow", "5000")
		values.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		hmac, err := crypto.GetHMAC(crypto.HashSHA512,
			[]byte(values.Encode()),
			[]byte(creds.Secret))
		if err != nil {
			return err
		}
		values.Set("signature", crypto.HexEncodeToString(hmac))
	}
	return me.SendPayload(ctx, request.Auth, func() (*request.Item, error) {
		return &request.Item{
			Method:  method,
			Path:    ePoint + "/api" + mexcAPIVersion + common.EncodeURLValues(requestPath, values),
			Headers: headers,
			// Body:          bytes.NewBufferString(values.Encode()),
			Result:        result,
			NonceEnabled:  true,
			Verbose:       me.Verbose,
			HTTPDebugging: me.HTTPDebugging,
			HTTPRecording: me.HTTPRecording,
		}, nil
	}, authType)
}
