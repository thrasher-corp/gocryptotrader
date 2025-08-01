package poloniex

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetAccountBalance get information about your Futures account.
func (e *Exchange) GetAccountBalance(ctx context.Context) (*FuturesAccountBalance, error) {
	var resp *FuturesAccountBalance
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, "/v3/account/balance", nil, nil, &resp, true)
}

// GetAccountBills retrieve the account’s bills.
func (e *Exchange) GetAccountBills(ctx context.Context, startTime, endTime time.Time, offset, limit int64, direction, billType string) ([]BillDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if billType != "" {
		params.Set("type", billType)
	}
	var resp []BillDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, "/v3/account/bills", params, nil, &resp, true)
}

// PlaceV3FuturesOrder place an order in futures trading.
func (e *Exchange) PlaceV3FuturesOrder(ctx context.Context, arg *FuturesParams) (*FuturesV3OrderIDResponse, error) {
	if arg == nil || *arg == (FuturesParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.PositionSide == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *FuturesV3OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/order", nil, arg, &resp, true)
}

// PlaceV3FuturesMultipleOrders place orders in a batch. A maximum of 10 orders can be placed per request.
func (e *Exchange) PlaceV3FuturesMultipleOrders(ctx context.Context, args []FuturesParams) ([]FuturesV3OrderItem, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		err := validationOrderCreationParam(&args[x])
		if err != nil {
			return nil, err
		}
	}
	var resp []FuturesV3OrderItem
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/orders", nil, args, &resp, true)
}

func validationOrderCreationParam(arg *FuturesParams) error {
	if *arg == (FuturesParams{}) {
		return common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.PositionSide == "" {
		return order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return order.ErrAmountBelowMin
	}
	return nil
}

// CancelV3FuturesOrder cancels an order in futures trading.
func (e *Exchange) CancelV3FuturesOrder(ctx context.Context, arg *CancelOrderParams) (*FuturesV3OrderIDResponse, error) {
	if arg == nil || *arg == (CancelOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *FuturesV3OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/order", nil, arg, &resp, true)
}

// CancelMultipleV3FuturesOrders cancel orders in a batch. A maximum of 10 orders can be cancelled per request.
func (e *Exchange) CancelMultipleV3FuturesOrders(ctx context.Context, args *CancelOrdersParams) ([]FuturesV3OrderIDResponse, error) {
	if args == nil {
		return nil, common.ErrEmptyParams
	}
	if args.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []FuturesV3OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/batchOrders", nil, args, &resp, true)
}

// CancelAllV3FuturesOrders cancel all current pending orders.
func (e *Exchange) CancelAllV3FuturesOrders(ctx context.Context, symbol, side string) ([]FuturesV3OrderIDResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	arg := &struct {
		Symbol string `json:"symbol"`
		Side   string `json:"side,omitempty"`
	}{
		Symbol: symbol,
		Side:   side,
	}
	var resp []FuturesV3OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/allOrders", nil, arg, &resp, true)
}

// CloseAtMarketPrice close orders at market price.
func (e *Exchange) CloseAtMarketPrice(ctx context.Context, symbol, marginMode, positionSide, clientOrderID string) (*FuturesV3OrderIDResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	arg := &struct {
		Symbol       string `json:"symbol"`
		MgnMode      string `json:"mgnMode"`
		ClOrdID      string `json:"clOrdId"`
		PositionSide string `json:"posSide,omitempty"`
	}{
		Symbol:       symbol,
		MgnMode:      marginMode,
		ClOrdID:      clientOrderID,
		PositionSide: positionSide,
	}
	var resp *FuturesV3OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/position", nil, arg, &resp, true)
}

// CloseAllAtMarketPrice close all orders at market price.
func (e *Exchange) CloseAllAtMarketPrice(ctx context.Context) ([]FuturesV3OrderIDResponse, error) {
	var resp []FuturesV3OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/positionAll", nil, nil, &resp, true)
}

// GetCurrentFuturesOrders get unfilled futures orders. If no request parameters are specified, you will get all open orders sorted on the creation time in chronological order.
func (e *Exchange) GetCurrentFuturesOrders(ctx context.Context, symbol, side, orderID, clientOrderID, direction string, offset, limit int64) ([]FuturesV3OrderDetail, error) {
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesV3OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/opens", params, nil, &resp, true)
}

// GetOrderExecutionDetails retrieves detailed information about your executed futures order
func (e *Exchange) GetOrderExecutionDetails(ctx context.Context, symbol, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit int64) ([]FuturesTradeFill, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesTradeFill
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/trades", params, nil, &resp, true)
}

// GetV3FuturesOrderHistory retrieves previous futures orders. Orders that are completely canceled (no transaction has occurred) initiated through the API can only be queried for 4 hours.
func (e *Exchange) GetV3FuturesOrderHistory(ctx context.Context, symbol, orderType, side, orderState, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit int64) ([]FuturesV3OrderDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if side != "" {
		params.Set("side", side)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderState != "" {
		params.Set("state", orderState)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesV3OrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/history", params, nil, &resp, true)
}

// GetV3FuturesCurrentPosition retrieves  information about your current position.
func (e *Exchange) GetV3FuturesCurrentPosition(ctx context.Context, symbol string) ([]V3FuturesPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []V3FuturesPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/position/opens", params, nil, &resp, true)
}

// GetV3FuturesPositionHistory get information about previous positions.
func (e *Exchange) GetV3FuturesPositionHistory(ctx context.Context, symbol, marginMode, positionSide, direction string, startTime, endTime time.Time, offset, limit int64) ([]V3FuturesPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	if positionSide != "" {
		params.Set("posSide", positionSide)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/position/history", params, nil, &resp, true)
}

// AdjustMarginForIsolatedMarginTradingPositions add or reduce margin for positions in isolated margin mode.
func (e *Exchange) AdjustMarginForIsolatedMarginTradingPositions(ctx context.Context, symbol, positionSide, adjustType string, amount float64) (*AdjustV3FuturesMarginResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if adjustType == "" {
		return nil, errMarginAdjustTypeMissing
	}
	arg := &struct {
		Symbol       string  `json:"symbol"`
		PositionSide string  `json:"posSide,omitempty"`
		Amount       float64 `json:"amt,string"`
		Type         string  `json:"type"`
	}{
		Symbol:       symbol,
		PositionSide: positionSide,
		Amount:       amount,
		Type:         adjustType,
	}
	var resp *AdjustV3FuturesMarginResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/position/margin", nil, arg, &resp, true)
}

// GetV3FuturesLeverage retrieves the list of leverage.
func (e *Exchange) GetV3FuturesLeverage(ctx context.Context, symbol, marginMode string) ([]V3FuturesLeverage, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	var resp []V3FuturesLeverage
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/position/leverages", params, nil, &resp, true)
}

// SetV3FuturesLeverage change leverage
func (e *Exchange) SetV3FuturesLeverage(ctx context.Context, symbol, marginMode, positionSide string, leverage int64) (*V3FuturesLeverage, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if positionSide == "" {
		return nil, order.ErrSideIsInvalid
	}
	if leverage <= 0 {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	var resp *V3FuturesLeverage
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/position/leverage", nil, &map[string]string{
		"symbol":  symbol,
		"mgnMode": marginMode,
		"posSide": positionSide,
		"lever":   strconv.FormatInt(leverage, 10),
	}, &resp, true)
}

// SwitchPositionMode switch the current position mode. Please ensure you do not have open positions and open orders under this mode before the switch.
// Position mode, HEDGE: LONG/SHORT, ONE_WAY: BOTH
func (e *Exchange) SwitchPositionMode(ctx context.Context, positionMode string) error {
	if positionMode == "" {
		return errPositionModeInvalid
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/position/mode", nil, map[string]string{"posMode": positionMode}, &struct{}{}, true)
}

// GetPositionMode get the current position mode.
func (e *Exchange) GetPositionMode(ctx context.Context) (string, error) {
	resp := &struct {
		PositionMode string `json:"posMode"`
	}{}
	return resp.PositionMode, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/position/mode", nil, nil, &resp, true)
}

// GetV3FuturesOrderBook get market depth data of the designated trading pair
func (e *Exchange) GetV3FuturesOrderBook(ctx context.Context, symbol string, depth, limit int64) (*FuturesV3Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if depth > 0 {
		params.Set("scale", strconv.FormatInt(depth, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FuturesV3Orderbook
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/orderBook", params), &resp, true)
}

var intervalToStringMap = map[kline.Interval]string{kline.OneMin: "MINUTE_1", kline.FiveMin: "MINUTE_5", kline.FifteenMin: "MINUTE_15", kline.ThirtyMin: "MINUTE_30", kline.OneHour: "HOUR_1", kline.TwoHour: "HOUR_2", kline.FourHour: "HOUR_4", kline.TwelveHour: "HOUR_12", kline.OneDay: "DAY_1", kline.ThreeDay: "DAY_3", kline.OneWeek: "WEEK_1"}

// IntervalString returns a string representation of kline.Interval instance.
func IntervalString(interval kline.Interval) (string, error) {
	intervalString, ok := intervalToStringMap[interval]
	if !ok {
		return "", kline.ErrUnsupportedInterval
	}
	return intervalString, nil
}

// GetV3FuturesKlineData retrieves K-line data of the designated trading pair
func (e *Exchange) GetV3FuturesKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]V3FuturesCandle, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	intervalString, err := IntervalString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("interval", intervalString)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	resp := &struct {
		Code    int64             `json:"code"`
		Message string            `json:"msg"`
		Data    TypeFuturesCandle `json:"data"`
	}{}
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/candles", params), &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("code: %d, msg: %s", resp.Code, resp.Message)
	}
	return resp.Data, nil
}

// TypeFuturesCandle holds and handles futures candle data
type TypeFuturesCandle []V3FuturesCandle

// UnmarshalJSON deserializes byte data into list of V3FuturesCandle
func (t *TypeFuturesCandle) UnmarshalJSON(data []byte) error {
	var result []V3FuturesCandle
	err := json.Unmarshal(data, &result)
	if err != nil {
		result = []V3FuturesCandle{}
	}
	*t = result
	return nil
}

// GetV3FuturesExecutionInfo get the latest execution information. The default limit is 500, with a maximum of 1,000.
func (e *Exchange) GetV3FuturesExecutionInfo(ctx context.Context, symbol string, limit int64) ([]V3FuturesExecutionInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesExecutionInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/trades", params), &resp, true)
}

// GetV3LiquidiationOrder get Liquidation Order Interface
func (e *Exchange) GetV3LiquidiationOrder(ctx context.Context, symbol, direction string, startTime, endTime time.Time, offset, limit int64) ([]LiquidiationPriceInfo, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LiquidiationPriceInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/liquidationOrder", params), &resp, true)
}

// GetV3FuturesMarketInfo get the market information of trading pairs in the past 24 hours.
func (e *Exchange) GetV3FuturesMarketInfo(ctx context.Context, symbol string) ([]V3FuturesTickerDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []V3FuturesTickerDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/tickers", params), &resp, true)
}

// GetV3FuturesIndexPrice get the current index price.
func (e *Exchange) GetV3FuturesIndexPrice(ctx context.Context, symbol string) (*InstrumentIndexPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *InstrumentIndexPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPrice", params), &resp, true)
}

// GetV3IndexPriceComponents get the index price components for a trading pair.
func (e *Exchange) GetV3IndexPriceComponents(ctx context.Context, symbol string) (*IndexPriceComponent, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *IndexPriceComponent
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPriceComponents", params), &resp, true)
}

// GetIndexPriceKlineData obtain the K-line data for the index price.
func (e *Exchange) GetIndexPriceKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	intervalString, err := IntervalString(interval)
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
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesIndexPriceData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPriceCandlesticks", params), &resp, true)
}

// GetV3FuturesMarkPrice get the current mark price.
func (e *Exchange) GetV3FuturesMarkPrice(ctx context.Context, symbol string) (*V3FuturesMarkPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *V3FuturesMarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/markPrice", params), &resp, true)
}

// GetMarkPriceKlineData obtain the K-line data for the mark price.
func (e *Exchange) GetMarkPriceKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]V3FuturesMarkPriceCandle, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	intervalString, err := IntervalString(interval)
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
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesMarkPriceCandle
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/markPriceCandlesticks", params), &resp, true)
}

// GetV3FuturesAllProductInfo inquire about the basic information of the all product.
func (e *Exchange) GetV3FuturesAllProductInfo(ctx context.Context, symbol string) ([]ProductInfo, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []ProductInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/allInstruments", params), &resp, true)
}

// GetV3FuturesProductInfo inquire about the basic information of the product.
func (e *Exchange) GetV3FuturesProductInfo(ctx context.Context, symbol string) (*ProductInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *ProductInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/instruments", params), &resp, true)
}

// GetV3FuturesCurrentFundingRate retrieve the current funding rate of the contract.
func (e *Exchange) GetV3FuturesCurrentFundingRate(ctx context.Context, symbol string) (*V3FuturesFundingRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *V3FuturesFundingRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/fundingRate", params), &resp, true)
}

// GetV3FuturesHistoricalFundingRates retrieve the previous funding rates of a contract.
func (e *Exchange) GetV3FuturesHistoricalFundingRates(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) ([]V3FuturesFundingRate, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesFundingRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/fundingRate/history", params), &resp, true)
}

// GetV3FuturesCurrentOpenPositions retrieve all current open interest in the market.
func (e *Exchange) GetV3FuturesCurrentOpenPositions(ctx context.Context, symbol string) (*OpenInterestData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *OpenInterestData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/openInterest", params), &resp, true)
}

// GetInsuranceFundInformation query insurance fund information
func (e *Exchange) GetInsuranceFundInformation(ctx context.Context) ([]InsuranceFundInfo, error) {
	var resp []InsuranceFundInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "/v3/market/insurance", &resp, true)
}

// GetV3FuturesRiskLimit retrieve information from the Futures Risk Limit Table.
func (e *Exchange) GetV3FuturesRiskLimit(ctx context.Context, symbol string) ([]RiskLimit, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []RiskLimit
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/riskLimit", params), &resp, true)
}
