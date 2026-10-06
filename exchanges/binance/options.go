package binance

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errUnderlyingIsRequired  = errors.New("underlying is required")
	errQuantityLimitRequired = errors.New("quantity limit is required")
	errDeltaLimitRequired    = errors.New("delta limit is required")
)

// GetAccountFundingFlow returns the options Account Funding Flow of a currency
func (e *Exchange) GetAccountFundingFlow(ctx context.Context, req *OptionsAccountFundingFlowRequest) ([]OptionsAccountFunding, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("currency", req.Currency.Upper().String())
	if req.RecordID != 0 {
		params.Set("recordId", strconv.FormatUint(req.RecordID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []OptionsAccountFunding
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/bill", params, optionsDefaultRate, &resp)
}

// GetOptionMarginAccountInformation returns the Option Margin Account Information
func (e *Exchange) GetOptionMarginAccountInformation(ctx context.Context) (*OptionsMarginAccountInformationResponse, error) {
	var resp *OptionsMarginAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/marginAccount", nil, optionsMarginAccountInfoRate, &resp)
}

// GetDownloadIDForOptionTransactionHistory requests an options transaction history file for a time window and
// returns its download ID. Binance no longer documents this endpoint, but it still answers
func (e *Exchange) GetDownloadIDForOptionTransactionHistory(ctx context.Context, startTime, endTime time.Time) (*OptionsTransactionHistoryDownloadIDResponse, error) {
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *OptionsTransactionHistoryDownloadIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/income/asyn", params, optionsDownloadIDForOptionTransactionHistoryRate, &resp)
}

// GetOptionTransactionHistoryDownloadLinkByID returns the download link of an options transaction history file.
// Binance no longer documents this endpoint, but it still answers
func (e *Exchange) GetOptionTransactionHistoryDownloadLinkByID(ctx context.Context, downloadID string) (*OptionsTransactionHistoryDownloadLinkResponse, error) {
	if downloadID == "" {
		return nil, errDownloadIDRequired
	}
	params := url.Values{}
	params.Set("downloadId", downloadID)
	var resp *OptionsTransactionHistoryDownloadLinkResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/income/asyn/id", params, optionsGetTransHistoryDownloadLinkByIDRate, &resp)
}

// CheckEOptionsServerTime returns the options Check Server Time
func (e *Exchange) CheckEOptionsServerTime(ctx context.Context) (*OptionsServerTimeResponse, error) {
	var resp *OptionsServerTimeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/time", optionsDefaultRate, &resp)
}

// GetOptionsExchangeInformation returns the options Exchange Information
func (e *Exchange) GetOptionsExchangeInformation(ctx context.Context) (*OptionsExchangeInformationResponse, error) {
	var resp *OptionsExchangeInformationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/exchangeInfo", optionsDefaultRate, &resp)
}

// GetEOptionsHistoricalExerciseRecords returns the options Historical Exercise Records
func (e *Exchange) GetEOptionsHistoricalExerciseRecords(ctx context.Context, req *OptionsExerciseHistoryRequest) ([]ExerciseHistoryItem, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.Underlying != "" {
		params.Set("underlying", req.Underlying)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []ExerciseHistoryItem
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/exerciseHistory", params), optionsHistoricalExerciseRecordsRate, &resp)
}

// GetOptionsIndexPrice returns the spot Index Price of an options underlying such as BTCUSDT
func (e *Exchange) GetOptionsIndexPrice(ctx context.Context, underlying string) (*OptionsIndexPriceResponse, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var resp *OptionsIndexPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/index", params), optionsDefaultRate, &resp)
}

// GetEOptionsCandlesticks returns the options Kline/Candlestick Data of a symbol
func (e *Exchange) GetEOptionsCandlesticks(ctx context.Context, req *OptionsKlineRequest) ([]EOptionsCandlestick, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Interval == 0 {
		return nil, kline.ErrInvalidInterval
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", e.FormatExchangeKlineInterval(req.Interval))
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []EOptionsCandlestick
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/klines", params), optionsDefaultRate, &resp)
}

// GetEOptionsOpenInterests returns the options Open Interest of an underlying asset for an expiry date
func (e *Exchange) GetEOptionsOpenInterests(ctx context.Context, underlyingAsset currency.Code, expiration time.Time) ([]OptionsOpenInterest, error) {
	if underlyingAsset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if expiration.IsZero() {
		return nil, errExpirationTimeRequired
	}
	params := url.Values{}
	params.Set("underlyingAsset", underlyingAsset.Upper().String())
	// Expiry dates are written YYMMDD in UTC, as in option symbols such as BTC-261225-85000-C
	params.Set("expiration", expiration.UTC().Format("060102"))
	var resp []OptionsOpenInterest
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/openInterest", params), optionsDefaultRate, &resp)
}

// GetOptionMarkPrice returns the Option Mark Price and greeks of a symbol, or of every symbol when it is empty
func (e *Exchange) GetOptionMarkPrice(ctx context.Context, symbol currency.Pair) ([]OptionMarkPrice, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		s, err := e.FormatSymbol(symbol, asset.Options)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	var resp []OptionMarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/mark", params), optionsMarkPriceRate, &resp)
}

// GetEOptionsOrderbook returns the options Order Book of a symbol
//
// Binance documents limits of 10, 20, 50, 100, 500 and 1000, 100 when zero, but rejects 500 with -4021
func (e *Exchange) GetEOptionsOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*OptionsOrderBookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *OptionsOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/depth", params), optionsOrderbookLimit(limit), &resp)
}

// GetEOptionsRecentTrades returns the options Recent Trades List of a symbol
func (e *Exchange) GetEOptionsRecentTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]OptionsTrade, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []OptionsTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/trades", params), optionsRecentTradesRate, &resp)
}

// GetEOptions24hrTickerPriceChangeStatistics returns the options 24hr Ticker Price Change Statistics of a symbol, or
// of every symbol when it is empty
func (e *Exchange) GetEOptions24hrTickerPriceChangeStatistics(ctx context.Context, symbol currency.Pair) ([]EOptionTicker, error) {
	params := url.Values{}
	rateLimit := optionsAllTickerPriceStatisticsRate
	if !symbol.IsEmpty() {
		rateLimit = optionsDefaultRate
		s, err := e.FormatSymbol(symbol, asset.Options)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	var resp []EOptionTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestOptions, common.EncodeURLValues("/eapi/v1/ticker", params), rateLimit, &resp)
}

// SendOptionsAutoCancelAllOpenOrdersHeartbeat sends an options Auto-Cancel All Open Orders (Kill-Switch) Heartbeat,
// which restarts the countdown of each underlying; the response lists the underlyings whose countdown restarted
func (e *Exchange) SendOptionsAutoCancelAllOpenOrdersHeartbeat(ctx context.Context, underlyings []string) (*OptionsAutoCancelAllOpenOrdersHeartbeatResponse, error) {
	if len(underlyings) == 0 {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlyings", strings.Join(underlyings, ","))
	var resp *OptionsAutoCancelAllOpenOrdersHeartbeatResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/countdownCancelAllHeartBeat", params, optionsAutoCancelAllOpenOrdersHeartbeatRate, &resp)
}

// GetAutoCancelAllOpenOrdersConfig returns the options Auto-Cancel All Open Orders (Kill-Switch) Config of an
// underlying, or of every underlying when it is empty; underlyings without an active countdown are not returned
func (e *Exchange) GetAutoCancelAllOpenOrdersConfig(ctx context.Context, underlying string) (*OptionsAutoCancelAllOpenOrdersConfigResponse, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var resp *OptionsAutoCancelAllOpenOrdersConfigResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/countdownCancelAll", params, optionsDefaultRate, &resp)
}

// SetOptionsAutoCancelAllOpenOrders sets the options Auto-Cancel All Open Orders (Kill-Switch) Config of an
// underlying. Once the countdown elapses without a heartbeat, its open orders are cancelled and new orders are
// rejected until a heartbeat arrives or the countdown is turned off. A zero countdown turns it off; otherwise Binance
// requires at least five seconds
func (e *Exchange) SetOptionsAutoCancelAllOpenOrders(ctx context.Context, underlying string, countdown time.Duration) (*OptionsAutoCancelAllOpenOrdersConfigResponse, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	params.Set("countdownTime", strconv.FormatInt(countdown.Milliseconds(), 10))
	var resp *OptionsAutoCancelAllOpenOrdersConfigResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/countdownCancelAll", params, optionsDefaultRate, &resp)
}

// GetOptionsMarketMakerProtection returns the options Market Maker Protection Config of an underlying
func (e *Exchange) GetOptionsMarketMakerProtection(ctx context.Context, underlying string) (*OptionsMarketMakerProtectionResponse, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var resp *OptionsMarketMakerProtectionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/mmp", params, optionsDefaultRate, &resp)
}

// ResetOptionsMarketMakerProtection resets the options Market Maker Protection of an underlying after it triggered,
// so market maker protection orders are accepted again
func (e *Exchange) ResetOptionsMarketMakerProtection(ctx context.Context, underlying string) (*OptionsMarketMakerProtectionResponse, error) {
	if underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var resp *OptionsMarketMakerProtectionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/mmpReset", params, optionsDefaultRate, &resp)
}

// SetOptionsMarketMakerProtectionConfig sets the options Market Maker Protection Config of an underlying. Once the
// quantity or net delta traded within the window reaches its limit, protection triggers: open market maker protection
// orders are cancelled and new ones are rejected for the frozen time
func (e *Exchange) SetOptionsMarketMakerProtectionConfig(ctx context.Context, req *OptionsMarketMakerProtectionConfigRequest) (*OptionsMarketMakerProtectionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Underlying == "" {
		return nil, errUnderlyingIsRequired
	}
	if req.QuantityLimit <= 0 {
		return nil, errQuantityLimitRequired
	}
	if req.DeltaLimit <= 0 {
		return nil, errDeltaLimitRequired
	}
	params := url.Values{}
	params.Set("underlying", req.Underlying)
	params.Set("windowTimeInMilliseconds", strconv.FormatInt(req.WindowTime.Milliseconds(), 10))
	params.Set("frozenTimeInMilliseconds", strconv.FormatInt(req.FrozenTime.Milliseconds(), 10))
	params.Set("qtyLimit", strconv.FormatFloat(req.QuantityLimit, 'f', -1, 64))
	params.Set("deltaLimit", strconv.FormatFloat(req.DeltaLimit, 'f', -1, 64))
	var resp *OptionsMarketMakerProtectionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/mmpSet", params, optionsDefaultRate, &resp)
}

// GetEOptionsAccountTradeList returns the options Account Trade List of a symbol
func (e *Exchange) GetEOptionsAccountTradeList(ctx context.Context, req *OptionsAccountTradeListRequest) ([]OptionsAccountTradeItem, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []OptionsAccountTradeItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/userTrades", params, optionsAccountTradeListRate, &resp)
}

// CancelAllOptionsOrdersByUnderlying cancels every active options order of an underlying through Cancel All Option
// Orders By Underlying. The success reply carries no data, so only an error is returned
func (e *Exchange) CancelAllOptionsOrdersByUnderlying(ctx context.Context, underlying string) error {
	if underlying == "" {
		return errUnderlyingIsRequired
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	return e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/allOpenOrdersByUnderlying", params, optionsCancelAllByUnderlyingRate, nil)
}

// CancelAllOptionOrdersOnSpecificSymbol cancels every active options order of a symbol through Cancel all Option
// orders on specific symbol. The success reply carries no data, so only an error is returned
func (e *Exchange) CancelAllOptionOrdersOnSpecificSymbol(ctx context.Context, symbol currency.Pair) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.Options)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", s)
	return e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/allOpenOrders", params, optionsDefaultRate, nil)
}

// PlaceBatchEOptionsOrder places up to ten options orders through Place Multiple Orders
func (e *Exchange) PlaceBatchEOptionsOrder(ctx context.Context, orders []OptionsOrderRequest) ([]OptionsOrderResponse, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
	}
	batch := make([]map[string]string, len(orders))
	for i := range orders {
		params, err := e.optionsOrderParams(&orders[i])
		if err != nil {
			return nil, err
		}
		batch[i] = make(map[string]string, len(params))
		for k := range params {
			batch[i][k] = params.Get(k)
		}
	}
	batchOrders, err := json.Marshal(batch)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("orders", string(batchOrders))
	var resp []OptionsOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/batchOrders", params, optionsBatchOrderRate, &resp)
}

// CancelBatchOptionsOrders cancels options orders of a symbol by order ID or client order ID through Cancel Multiple
// Option Orders
func (e *Exchange) CancelBatchOptionsOrders(ctx context.Context, symbol currency.Pair, orderIDs []uint64, clientOrderIDs []string) ([]OptionsCancelledOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orderIDs) == 0 && len(clientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	s, err := e.FormatSymbol(symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if len(orderIDs) != 0 {
		ids, err := json.Marshal(orderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("orderIds", string(ids))
	}
	if len(clientOrderIDs) != 0 {
		ids, err := json.Marshal(clientOrderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("clientOrderIds", string(ids))
	}
	var resp []OptionsCancelledOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/batchOrders", params, optionsCancelBatchOrdersRate, &resp)
}

// optionsOrderIdentityParams builds the symbol and order identity parameters of the single options order endpoints,
// which require an order ID or a client order ID
func (e *Exchange) optionsOrderIdentityParams(symbol currency.Pair, clientOrderID string, orderID uint64) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	s, err := e.FormatSymbol(symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if clientOrderID != "" {
		params.Set("clientOrderId", clientOrderID)
	}
	return params, nil
}

// GetSingleEOptionsOrder returns an options order through Query Single Order
func (e *Exchange) GetSingleEOptionsOrder(ctx context.Context, symbol currency.Pair, clientOrderID string, orderID uint64) (*OptionsOrderStatusResponse, error) {
	params, err := e.optionsOrderIdentityParams(symbol, clientOrderID, orderID)
	if err != nil {
		return nil, err
	}
	var resp *OptionsOrderStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/order", params, optionsDefaultRate, &resp)
}

// NewOptionsOrder places an options order through New Order
func (e *Exchange) NewOptionsOrder(ctx context.Context, req *OptionsOrderRequest) (*OptionsOrderResponse, error) {
	params, err := e.optionsOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *OptionsOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/order", params, optionsDefaultOrderRate, &resp)
}

// optionsOrderParams builds the parameters of an options order, which New Order sends as they are and Place Multiple
// Orders sends as one JSON object per order
func (e *Exchange) optionsOrderParams(req *OptionsOrderRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if strings.EqualFold(req.OrderType, order.Limit.String()) && req.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", req.Side)
	params.Set("type", req.OrderType)
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	if req.Price != 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}
	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if req.PostOnly {
		params.Set("postOnly", "true")
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.ClientOrderID != "" {
		params.Set("clientOrderId", req.ClientOrderID)
	}
	if req.IsMarketMakerProtection {
		params.Set("isMmp", "true")
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	return params, nil
}

// CancelOptionsOrder cancels an options order through Cancel Option Order
func (e *Exchange) CancelOptionsOrder(ctx context.Context, symbol currency.Pair, clientOrderID string, orderID uint64) (*OptionsCancelOrderResponse, error) {
	params, err := e.optionsOrderIdentityParams(symbol, clientOrderID, orderID)
	if err != nil {
		return nil, err
	}
	var resp *OptionsCancelOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/order", params, optionsDefaultRate, &resp)
}

// GetOptionPositionInformation returns the Option Position Information of a symbol, or of every symbol when it is
// empty
func (e *Exchange) GetOptionPositionInformation(ctx context.Context, symbol currency.Pair) ([]OptionPosition, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		s, err := e.FormatSymbol(symbol, asset.Options)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	var resp []OptionPosition
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/position", params, optionsPositionInformationRate, &resp)
}

// GetCurrentOpenOptionsOrders returns the Current Open Option Orders of a symbol, or of every symbol when it is empty
func (e *Exchange) GetCurrentOpenOptionsOrders(ctx context.Context, req *OptionsOpenOrdersRequest) ([]OptionsOpenOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	rateLimit := optionsAllQueryOpenOrdersRate
	if !req.Symbol.IsEmpty() {
		rateLimit = optionsDefaultRate
		symbol, err := e.FormatSymbol(req.Symbol, asset.Options)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp []OptionsOpenOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/openOrders", params, rateLimit, &resp)
}

// GetOptionsOrdersHistory returns the Option Order History of a symbol, which only covers orders finished within
// the last five days
func (e *Exchange) GetOptionsOrdersHistory(ctx context.Context, req *OptionsOrderHistoryRequest) ([]OptionsOrderHistoryItem, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Options)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []OptionsOrderHistoryItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/historyOrders", params, optionsGetOrderHistoryRate, &resp)
}

// GetUserOptionsExerciseRecord returns the options User Exercise Record
func (e *Exchange) GetUserOptionsExerciseRecord(ctx context.Context, req *OptionsUserExerciseRecordRequest) ([]UserOptionsExerciseRecord, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(req.Symbol, asset.Options)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []UserOptionsExerciseRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodGet, "/eapi/v1/exerciseRecord", params, optionsUserExerciseRecordRate, &resp)
}

// KeepaliveOptionsUserDataStream extends the validity of the account's options user data stream listen key by 60
// minutes
func (e *Exchange) KeepaliveOptionsUserDataStream(ctx context.Context) error {
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestOptions, http.MethodPut, "/eapi/v1/listenKey", optionsDefaultRate, nil)
}

// StartOptionsUserDataStream starts an options user data stream. An account with an active listen key gets that key
// back with its validity extended by 60 minutes
func (e *Exchange) StartOptionsUserDataStream(ctx context.Context) (*OptionsListenKeyResponse, error) {
	var resp *OptionsListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, "/eapi/v1/listenKey", optionsDefaultRate, &resp)
}

// CloseOptionsUserDataStream closes the account's options user data stream and invalidates its listen key
func (e *Exchange) CloseOptionsUserDataStream(ctx context.Context) error {
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestOptions, http.MethodDelete, "/eapi/v1/listenKey", optionsDefaultRate, nil)
}
