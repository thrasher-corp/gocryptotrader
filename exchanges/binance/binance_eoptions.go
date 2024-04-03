package binance

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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Without subscriptions
	binanceEOptionWebsocketURL       = "wss://nbstream.binance.com/eoptions/ws"
	binanceEOptionWebsocketURLStream = "wss://nbstream.binance.com/eoptions/stream"
)

var (
	errUnderlyingIsRequired = errors.New("underlying is required")
)

// CheckEOptionsServerTime retrieves the server time.
func (b *Binance) CheckEOptionsServerTime(ctx context.Context) (convert.ExchangeTime, error) {
	resp := &struct {
		ServerTime convert.ExchangeTime `json:"serverTime"`
	}{}
	return resp.ServerTime, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/time", spotDefaultRate, &resp)
}

// GetOptionsExchangeInformation retrieves an exchange information through the options endpoint.
func (b *Binance) GetOptionsExchangeInformation(ctx context.Context) (*EOptionExchangeInfo, error) {
	var resp *EOptionExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/exchangeInfo", spotDefaultRate, &resp)
}

// GetEOptionsOrderbook retrieves european options orderbook information for specific symbol
func (b *Binance) GetEOptionsOrderbook(ctx context.Context, symbol string, limit int64) (*EOptionsOrderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *EOptionsOrderbook
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/depth", params), spotDefaultRate, &resp)
}

// GetEOptionsRecentTrades retrieves recent market trades
func (b *Binance) GetEOptionsRecentTrades(ctx context.Context, symbol string, limit int64) ([]EOptionsTradeItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EOptionsTradeItem
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/trades", params), spotDefaultRate, &resp)
}

// GetEOptionsTradeHistory retrieves older market historical trades.
func (b *Binance) GetEOptionsTradeHistory(ctx context.Context, symbol string, fromID, limit int64) ([]EOptionsTradeItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EOptionsTradeItem
	return resp, b.SendAPIKeyHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/historicalTrades", params), spotDefaultRate, &resp)
}

// GetEOptionsCandlesticks retrieves kline/candlestick bars for an option symbol. Klines are uniquely identified by their open time.
func (b *Binance) GetEOptionsCandlesticks(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]EOptionsCandlestick, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if interval == 0 || interval.String() == "" {
		return nil, kline.ErrInvalidInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", b.FormatExchangeKlineInterval(interval))
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
	var resp []EOptionsCandlestick
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/klines", params), spotDefaultRate, &resp)
}

// GetOptionMarkPrice option mark price and greek info.
func (b *Binance) GetOptionMarkPrice(ctx context.Context, symbol string) ([]OptionMarkPrice, error) {
	params := url.Values{}
	if symbol == "" {
		params.Set("symbol", symbol)
	}
	var resp []OptionMarkPrice
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/mark", params), spotDefaultRate, &resp)
}

// GetEOptions24hrTickerPriceChangeStatistics 24 hour rolling window price change statistics.
func (b *Binance) GetEOptions24hrTickerPriceChangeStatistics(ctx context.Context, symbol string) ([]EOptionTicker, error) {
	params := url.Values{}
	if symbol == "" {
		params.Set("symbol", symbol)
	}
	var resp []EOptionTicker
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/ticker", params), spotDefaultRate, &resp)
}

// GetEOptionsSymbolPriceTicker represents a symbol ticker instances.
func (b *Binance) GetEOptionsSymbolPriceTicker(ctx context.Context, underlying string) (*EOptionIndexSymbolPriceTicker, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var resp *EOptionIndexSymbolPriceTicker
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/index", params), spotDefaultRate, &resp)
}

// GetEOptionsHistoricalExerciseRecords retrieves historical exercise records.
func (b *Binance) GetEOptionsHistoricalExerciseRecords(ctx context.Context, underlying string, startTime, endTime time.Time, limit int64) ([]ExerciseHistoryItem, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ExerciseHistoryItem
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/exerciseHistory", params), spotDefaultRate, &resp)
}

// GetEOptionsOpenInterests retrieves  open interest for specific underlying asset on specific expiration date.
func (b *Binance) GetEOptionsOpenInterests(ctx context.Context, underlyingAsset string, expiration time.Time) ([]OpenInterest, error) {
	if underlyingAsset == "" {
		return nil, errUnderlyingIsRequired
	}
	if expiration.IsZero() {
		return nil, errors.New("expiration time is required")
	}
	params := url.Values{}
	params.Set("underlyingAsset", underlyingAsset)
	params.Set("expiration", fmt.Sprintf("%d%s%d", expiration.Day(), expiration.Month(), (expiration.Year()%2000)))
	var resp []OpenInterest
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/openInterest", params), spotDefaultRate, &resp)
}

// ----------------------------------------------------------- Account trade endpoints ---------------------------------------------------------------------

// GetOptionsAccountInformation retrieves the current account information.
func (b *Binance) GetOptionsAccountInformation(ctx context.Context) (*EOptionsAccountInformation, error) {
	var resp *EOptionsAccountInformation
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/account", spotDefaultRate, &resp)
}

// NewOptionsOrder places a new european options order instance.
func (b *Binance) NewOptionsOrder(ctx context.Context, arg *OptionsOrderParams) (*OptionOrder, error) {
	if arg == nil || *arg == (OptionsOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol.String())
	params.Set("side", arg.Side)
	params.Set("type", arg.OrderType)
	params.Set("quantity", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	arg.OrderType = strings.ToUpper(arg.OrderType)
	if arg.OrderType == "LIMIT" && arg.Price <= 0 {
		return nil, fmt.Errorf("%w, price is required for limit orders", order.ErrPriceBelowMin)
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.PostOnly {
		params.Set("postOnly", "true")
	}
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", arg.NewOrderResponseType)
	}
	if arg.ClientOrderID != "" {
		params.Set("clientOrderId", arg.ClientOrderID)
	}
	if arg.IsMarketMakerProtection {
		params.Set("isMmp", "true")
	}
	var resp *OptionOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/order", params, spotDefaultRate, &resp)
}

// PlaceEOptionsOrder send multiple option orders.
func (b *Binance) PlaceEOptionsOrder(ctx context.Context, args []OptionsOrderParams) ([]OptionOrder, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	for a := range args {
		if &args[a] == nil || args[a] == (OptionsOrderParams{}) {
			return nil, common.ErrNilPointer
		}
		if args[a].Symbol.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[a].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		if args[a].OrderType == "" {
			return nil, order.ErrTypeIsInvalid
		}
		if args[a].Amount <= 0 {
			return nil, order.ErrAmountBelowMin
		}
	}
	val, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("orders", string(val))
	var resp []OptionOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/batchOrders", params, spotDefaultRate, &resp)
}

// GetSingleEOptionsOrder retrieves a single order status.
func (b *Binance) GetSingleEOptionsOrder(ctx context.Context, symbol, clientOrderID string, orderID int64) (*OptionOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if clientOrderID != "" {
		params.Set("clientOrderId", clientOrderID)
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	var resp *OptionOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/order", params, spotDefaultRate, &resp)
}

// CancelOptionsOrder represents an options order instance
func (b *Binance) CancelOptionsOrder(ctx context.Context, symbol, clientOrderID string, orderID int64) (*OptionOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if clientOrderID != "" {
		params.Set("clientOrderId", clientOrderID)
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	var resp *OptionOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/eapi/v1/order", params, spotDefaultRate, &resp)
}

// CancelOptionsOrders cancel an active orders
func (b *Binance) CancelOptionsOrders(ctx context.Context, symbol string, orderIDs []int64, clientOrderIDs []string) ([]OptionOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if len(orderIDs) == 0 && len(clientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if len(orderIDs) > 0 {
		vals, err := json.Marshal(orderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("orderIds", string(vals))
	}
	if len(clientOrderIDs) > 0 {
		vals, err := json.Marshal(clientOrderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("clientOrderIds", string(vals))
	}
	var resp []OptionOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/batchOrders", params, spotDefaultRate, &resp)
}

// CancelAllOptionOrdersOnSpecificSymbol cancels all active order on a symbol
func (b *Binance) CancelAllOptionOrdersOnSpecificSymbol(ctx context.Context, symbol string) error {
	params := url.Values{}
	if symbol == "" {
		params.Set("symbol", symbol)
	}
	return b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/allOpenOrders", params, spotDefaultRate, &struct{}{})
}

// CancelAllOptionsOrdersByUnderlying cancel all active orders on specified underlying.
func (b *Binance) CancelAllOptionsOrdersByUnderlying(ctx context.Context, underlying string) (int64, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var resp int64
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/allOpenOrdersByUnderlying", params, spotDefaultRate, &resp)
}

// GetCurrentOpenOptionsOrders retrieves all open orders. Status: ACCEPTED PARTIALLY_FILLED
func (b *Binance) GetCurrentOpenOptionsOrders(ctx context.Context, symbol string, startTime, endTime time.Time, orderID, limit int64) ([]OptionOrder, error) {
	return b.getOptionsOrders(ctx, "/eapi/v1/openOrders", symbol, startTime, endTime, orderID, limit)
}

// GetOptionsOrdersHistory retrieves all finished orders within 5 days.
// Possible finished status values: CANCELLED, FILLED, REJECTED
func (b *Binance) GetOptionsOrdersHistory(ctx context.Context, symbol string, startTime, endTime time.Time, orderID, limit int64) ([]OptionOrder, error) {
	return b.getOptionsOrders(ctx, "/eapi/v1/historyOrders", symbol, startTime, endTime, orderID, limit)
}

func (b *Binance) getOptionsOrders(ctx context.Context, path, symbol string, startTime, endTime time.Time, orderID, limit int64) ([]OptionOrder, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []OptionOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, path, params, spotDefaultRate, &resp)
}

// GetOptionPositionInformation retrieves current position information.
func (b *Binance) GetOptionPositionInformation(ctx context.Context, symbol string) ([]OptionPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []OptionPosition
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/position", params, spotDefaultRate, &resp)
}

// GetEOptionsAccountTradeList retrieves trades for a specific account and symbol
func (b *Binance) GetEOptionsAccountTradeList(ctx context.Context, symbol string, fromID, limit int64, startTime, endTime time.Time) ([]OptionsAccountTradeItem, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []OptionsAccountTradeItem
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/eapi/v1/userTrades", params, spotDefaultRate, &resp)
}

// GetUserOptionsExerciseRecord retrives account exercise records
func (b *Binance) GetUserOptionsExerciseRecord(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) ([]UserOptionsExerciseRecord, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UserOptionsExerciseRecord
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/exerciseRecord", params, spotDefaultRate, &resp)
}

// GetAccountFundingFlow retrieves account funding flows
func (b *Binance) GetAccountFundingFlow(ctx context.Context, ccy currency.Code, recordID, limit int64, startTime, endTime time.Time) ([]AccountFunding, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	if recordID != 0 {
		params.Set("recordId", strconv.FormatInt(recordID, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountFunding
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/bill", params, spotDefaultRate, &resp)
}

// GetDownloadIDForOptionTransactionHistory retrieves options transaction history
func (b *Binance) GetDownloadIDForOptionTransactionHistory(ctx context.Context, startTime, endTime time.Time) (*DownloadIDOfOptionsTransaction, error) {
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *DownloadIDOfOptionsTransaction
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/eapi/v1/income/asyn", params, spotDefaultRate, &resp)
}

// GetOptionTransactionHistoryDownloadLinkByID retrieves an options transaction history download link by ID.
func (b *Binance) GetOptionTransactionHistoryDownloadLinkByID(ctx context.Context, downloadID string) (*DownloadIDTransactionHistory, error) {
	if downloadID == "" {
		return nil, errors.New("download ID is required")
	}
	params := url.Values{}
	params.Set("downloadId", downloadID)
	var resp *DownloadIDTransactionHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/income/asyn/id", params, spotDefaultRate, &resp)
}

// -----------------------------------------------------    Market Maker Endpoint   ----------------------------------------------------------------------------------
// Market maker endpoints only work for option market makers, api users will get error when send requests to these endpoints.

// GetOptionMarginAccountInformation retrieves current account information
func (b *Binance) GetOptionMarginAccountInformation(ctx context.Context) (*OptionMarginAccountInfo, error) {
	var resp *OptionMarginAccountInfo
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/marginAccount", nil, spotDefaultRate, &resp)
}

// SetOptionsMarketMakerProtectionConfig a sets config for market maker protection(MMP) is a set of protection mechanism for option market maker,
// this mechanism is able to prevent mass trading in short period time.
// Once market maker's account branches the threshold, the Market Maker Protection will be triggered.
// When Market Maker Protection triggers, all the current MMP orders will be canceled, new MMP orders will be rejected.
// Market maker can use this time to reevaluate market and modify order price.
func (b *Binance) SetOptionsMarketMakerProtectionConfig(ctx context.Context, arg *MarketMakerProtectionConfig) (*MarketMakerProtection, error) {
	if arg.Underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	if arg.WindowTimeInMilliseconds == 0 {
		return nil, errors.New("windowTimeInMilliseconds is required")
	}
	if arg.FrozenTimeInMilliseconds == 0 {
		return nil, errors.New("frozenTimeInMilliseconds is required")
	}
	if arg.QuantityLimit <= 0 {
		return nil, errors.New("quantity limit is required")
	}
	if arg.NetDeltaLimit <= 0 {
		return nil, errors.New("netDeltaLimit is required")
	}
	params := url.Values{}
	params.Set("underlying", arg.Underlying)
	params.Set("windowTimeInMilliseconds", strconv.FormatInt(arg.WindowTimeInMilliseconds, 10))
	params.Set("frozenTimeInMilliseconds", strconv.FormatInt(arg.FrozenTimeInMilliseconds, 10))
	params.Set("qtyLimit", strconv.FormatFloat(arg.QuantityLimit, 'f', -1, 64))
	params.Set("deltaLimit", strconv.FormatFloat(arg.NetDeltaLimit, 'f', -1, 64))
	var resp *MarketMakerProtection
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/mmpSet", params, spotDefaultRate, &resp)
}

// GetOptionsMarketMakerProtection retrieves the merket maker protection config
func (b *Binance) GetOptionsMarketMakerProtection(ctx context.Context, underlying string) (*MarketMakerProtection, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var resp *MarketMakerProtection
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/mmp", params, spotDefaultRate, &resp)
}

// ResetMarketMaketProtection reset MMP, start MMP order again.
func (b *Binance) ResetMarketMaketProtection(ctx context.Context, underlying string) (*MarketMakerProtection, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var resp *MarketMakerProtection
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/mmp", params, spotDefaultRate, &resp)
}

// SetOptionsAutoCancelAllOpenOrders sets the parameters of the auto-cancel feature which cancels all open orders
// (both market maker protection and non market maker protection order types) of the underlying symbol at the end of
// the specified countdown time period if no heartbeat message is sent. After the countdown time period,
// all open orders will be cancelled and new orders will be rejected with error code -2010 until either a
// heartbeat message is sent or the auto-cancel feature is turned off by setting countdownTime to 0.
//
// Countdown time in milliseconds (ex. 1,000 for 1 second). 0 to disable the timer. Negative values (ex. -10000) are not accepted.
// Minimum acceptable value is 5,000
func (b *Binance) SetOptionsAutoCancelAllOpenOrders(ctx context.Context, underlying string, countdownTime int64) (*UnderlyingCountdown, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	if countdownTime < 5000 {
		return nil, errors.New("countdown time in milliseconds must be greater than 5000")
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	var resp *UnderlyingCountdown
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/countdownCancelAll", params, spotDefaultRate, &resp)
}

// GetAutoCancelAllOpenOrdersConfig returns the auto-cancel parameters for each underlying symbol.
// Note only active auto-cancel parameters will be returned, if countdownTime is set to 0 (ie. countdownTime has been turned off),
// the underlying symbol and corresponding countdownTime parameter will not be returned in the response.
func (b *Binance) GetAutoCancelAllOpenOrdersConfig(ctx context.Context, underlying string) (*UnderlyingCountdown, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var resp *UnderlyingCountdown
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/countdownCancelAll", params, spotDefaultRate, &resp)
}

// GetOptionsAutoCancelAllOpenOrdersHeartbeat resets the time from which the countdown will begin to the time this messaged is received. It should be called repeatedly as heartbeats.
// Multiple heartbeats can be updated at once by specifying the underlying symbols as a list in the underlyings parameter.
func (b *Binance) GetOptionsAutoCancelAllOpenOrdersHeartbeat(ctx context.Context, underlyings []string) ([]string, error) {
	if len(underlyings) == 0 {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlyings", strings.Join(underlyings, ","))
	resp := &struct {
		Underlyings []string `json:"underlyings"`
	}{}
	return resp.Underlyings, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/countdownCancelAllHeartBeat", params, spotDefaultRate, &resp)
}
