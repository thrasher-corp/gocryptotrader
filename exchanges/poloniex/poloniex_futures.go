package poloniex

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	poloniexFuturesAPIURL = "https://futures-api.poloniex.com"
)

// GetFuturesServerTime get the API server time. This is the Unix timestamp.
func (p *Poloniex) GetFuturesServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	var resp *ServerTimeResponse
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/timestamp", &resp)
}

// GetFuturesUntriggeredStopOrderList retrieves list of untriggered futures orders.
func (p *Poloniex) GetFuturesUntriggeredStopOrderList(ctx context.Context, symbol, side, orderType string, startAt, endAt time.Time, marginType margin.Type) (*FuturesOrders, error) {
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
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	switch marginType {
	case margin.Multi:
		params.Set("marginType", "0")
	case margin.Isolated:
		params.Set("marginType", "1")
	}
	var resp *FuturesOrders
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetUntriggeredStopOrderEPL, http.MethodGet, "/api/v1/stopOrders", params, nil, &resp)
}

// GetAccountBalance get information about your Futures account.
func (p *Poloniex) GetAccountBalance(ctx context.Context) (*FuturesAccountBalance, error) {
	var resp *FuturesAccountBalance
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, "/v3/account/balance", nil, nil, &resp, true)
}

// GetAccountBills retrieve the account’s bills.
func (p *Poloniex) GetAccountBills(ctx context.Context, startTime, endTime time.Time, offset, limit int64, direction, billType string) ([]BillDetail, error) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, "/v3/account/bills", params, nil, &resp, true)
}

// PlaceV3FuturesOrder place an order in futures trading.
func (p *Poloniex) PlaceV3FuturesOrder(ctx context.Context, arg *FuturesV2Params) (*FuturesV3OrderIDResponse, error) {
	if arg == nil || *arg == (FuturesV2Params{}) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/order", nil, arg, &resp, true)
}

// PlaceV3FuturesMultipleOrders place orders in a batch. A maximum of 10 orders can be placed per request.
func (p *Poloniex) PlaceV3FuturesMultipleOrders(ctx context.Context, args []FuturesV2Params) ([]FuturesV3OrderIDResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		err := validationOrderCreationParam(&args[x])
		if err != nil {
			return nil, err
		}
	}
	var resp []FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/orders", nil, args, &resp, true)
}

func validationOrderCreationParam(arg *FuturesV2Params) error {
	if *arg == (FuturesV2Params{}) {
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
func (p *Poloniex) CancelV3FuturesOrder(ctx context.Context, arg *CancelOrderParams) (*FuturesV3OrderIDResponse, error) {
	if arg == nil || *arg == (CancelOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/order", nil, arg, &resp, true)
}

// CancelMultipleV3FuturesOrders cancel orders in a batch. A maximum of 10 orders can be cancelled per request.
func (p *Poloniex) CancelMultipleV3FuturesOrders(ctx context.Context, args *CancelOrdersParams) ([]FuturesV3OrderIDResponse, error) {
	if args == nil {
		return nil, common.ErrEmptyParams
	}
	if args.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/batchOrders", nil, args, &resp, true)
}

// CancelAllV3FuturesOrders cancel all current pending orders.
func (p *Poloniex) CancelAllV3FuturesOrders(ctx context.Context, symbol, side string) ([]FuturesV3OrderIDResponse, error) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/allOrders", nil, arg, &resp, true)
}

// CloseAtMarketPrice close orders at market price.
func (p *Poloniex) CloseAtMarketPrice(ctx context.Context, symbol, marginMode, positionSide, clientOrderID string) (*FuturesV3OrderIDResponse, error) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/position", nil, arg, &resp, true)
}

// CloseAllAtMarketPrice close all orders at market price.
func (p *Poloniex) CloseAllAtMarketPrice(ctx context.Context) ([]FuturesV3OrderIDResponse, error) {
	var resp []FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/positionAll", nil, nil, &resp, true)
}

// GetCurrentOrders get unfilled futures orders. If no request parameters are specified, you will get all open orders sorted on the creation time in chronological order.
func (p *Poloniex) GetCurrentOrders(ctx context.Context, symbol, side, orderID, clientOrderID, direction string, offset, limit int64) ([]FuturesV3Order, error) {
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
	var resp []FuturesV3Order
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/opens", params, nil, &resp, true)
}

// GetOrderExecutionDetails retrieves detailed information about your executed futures order
func (p *Poloniex) GetOrderExecutionDetails(ctx context.Context, symbol, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit int64) ([]FuturesTradeFill, error) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/trades", params, nil, &resp, true)
}

// GetV3FuturesOrderHistory retrieves previous futures orders. Orders that are completely canceled (no transaction has occurred) initiated through the API can only be queried for 4 hours.
func (p *Poloniex) GetV3FuturesOrderHistory(ctx context.Context, symbol, orderType, side, orderState, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit int64) ([]FuturesV3Order, error) {
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
	var resp []FuturesV3Order
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/history", params, nil, &resp, true)
}

// ------------------------------------------------- Position Endpoints ----------------------------------------------------

// GetV3FuturesCurrentPosition retrieves  information about your current position.
func (p *Poloniex) GetV3FuturesCurrentPosition(ctx context.Context, symbol string) ([]V3FuturesPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []V3FuturesPosition
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/position/opens", params, nil, &resp, true)
}

// GetV3FuturesPositionHistory get information about previous positions.
func (p *Poloniex) GetV3FuturesPositionHistory(ctx context.Context, symbol, marginMode, positionSide, direction string, startTime, endTime time.Time, offset, limit int64) ([]V3FuturesPosition, error) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/position/history", params, nil, &resp, true)
}

// AdjustMarginForIsolatedMarginTradingPositions add or reduce margin for positions in isolated margin mode.
func (p *Poloniex) AdjustMarginForIsolatedMarginTradingPositions(ctx context.Context, symbol, positionSide, adjustType string, amount float64) (*AdjustV3FuturesMarginResponse, error) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/position/margin", nil, arg, &resp, true)
}

// GetV3FuturesLeverage retrieves the list of leverage.
func (p *Poloniex) GetV3FuturesLeverage(ctx context.Context, symbol, marginMode string) ([]V3FuturesLeverage, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	var resp []V3FuturesLeverage
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/position/leverages", params, nil, &resp, true)
}

// SetV3FuturesLeverage change leverage
func (p *Poloniex) SetV3FuturesLeverage(ctx context.Context, symbol, marginMode, positionSide string, leverage int64) (*V3FuturesLeverage, error) {
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
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/position/leverage", nil, map[string]string{
		"symbol":  symbol,
		"mgnMode": marginMode,
		"posSide": positionSide,
		"lever":   strconv.FormatInt(leverage, 10),
	}, &resp, true)
}

// SwitchPositionMode switch the current position mode. Please ensure you do not have open positions and open orders under this mode before the switch.
// Position mode, HEDGE: LONG/SHORT, ONE_WAY: BOTH
func (p *Poloniex) SwitchPositionMode(ctx context.Context, positionMode string) error {
	if positionMode == "" {
		return errPositionModeInvalid
	}
	return p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/position/mode", nil, map[string]string{"posMode": positionMode}, &struct{}{}, true)
}

// GetPositionMode get the current position mode.
func (p *Poloniex) GetPositionMode(ctx context.Context) (string, error) {
	resp := &struct {
		PositionMode string `json:"posMode"`
	}{}
	return resp.PositionMode, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/position/mode", nil, nil, &resp, true)
}

// GetV3FuturesOrderBook get market depth data of the designated trading pair
func (p *Poloniex) GetV3FuturesOrderBook(ctx context.Context, symbol string, depth, limit int64) (*FuturesV3Orderbook, error) {
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
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/orderBook", params), &resp, true)
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
func (p *Poloniex) GetV3FuturesKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]V3FuturesCandle, error) {
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
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []V3FuturesCandle
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/candles", params), &resp, true)
}

// GetV3FuturesExecutionInfo get the latest execution information. The default limit is 500, with a maximum of 1,000.
func (p *Poloniex) GetV3FuturesExecutionInfo(ctx context.Context, symbol string, limit int64) ([]V3FuturesExecutionInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesExecutionInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/trades", params), &resp, true)
}

// GetV3LiquidiationOrder get Liquidation Order Interface
func (p *Poloniex) GetV3LiquidiationOrder(ctx context.Context, symbol, direction string, startTime, endTime time.Time, offset, limit int64) ([]LiquidiationPriceInfo, error) {
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
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/liquidationOrder", params), &resp, true)
}

// GetV3FuturesMarketInfo get the market information of trading pairs in the past 24 hours.
func (p *Poloniex) GetV3FuturesMarketInfo(ctx context.Context, symbol string) ([]V3FuturesTickerDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []V3FuturesTickerDetail
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/tickers", params), &resp, true)
}

// GetV3FuturesIndexPrice get the current index price.
func (p *Poloniex) GetV3FuturesIndexPrice(ctx context.Context, symbol string) (*InstrumentIndexPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *InstrumentIndexPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPrice", params), &resp, true)
}

// GetV3IndexPriceComponents get the index price components for a trading pair.
func (p *Poloniex) GetV3IndexPriceComponents(ctx context.Context, symbol string) (*IndexPriceComponent, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *IndexPriceComponent
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPriceComponents", params), &resp, true)
}

// GetIndexPriceKlineData obtain the K-line data for the index price.
func (p *Poloniex) GetIndexPriceKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) (interface{}, error) {
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
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPriceCandlesticks", params), &resp, true)
}

// GetV3FuturesMarkPrice get the current mark price.
func (p *Poloniex) GetV3FuturesMarkPrice(ctx context.Context, symbol string) (*V3FuturesMarkPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *V3FuturesMarkPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/markPrice", params), &resp, true)
}

// GetMarkPriceKlineData obtain the K-line data for the mark price.
func (p *Poloniex) GetMarkPriceKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]V3FuturesMarkPriceCandle, error) {
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
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/markPriceCandlesticks", params), &resp, true)
}

// GetV3FuturesAllProductInfo inquire about the basic information of the all product.
func (p *Poloniex) GetV3FuturesAllProductInfo(ctx context.Context, symbol string) ([]ProductInfo, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []ProductInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/allInstruments", params), &resp, true)
}

// GetV3FuturesProductInfo inquire about the basic information of the product.
func (p *Poloniex) GetV3FuturesProductInfo(ctx context.Context, symbol string) (*ProductInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *ProductInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/instruments", params), &resp, true)
}

// GetV3FuturesCurrentFundingRate retrieve the current funding rate of the contract.
func (p *Poloniex) GetV3FuturesCurrentFundingRate(ctx context.Context, symbol string) (*V3FuturesFundingRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *V3FuturesFundingRate
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/fundingRate", params), &resp, true)
}

// GetV3FuturesHistoricalFundingRates retrieve the previous funding rates of a contract.
func (p *Poloniex) GetV3FuturesHistoricalFundingRates(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) ([]V3FuturesFundingRate, error) {
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
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/fundingRate/history", params), &resp, true)
}

// GetV3FuturesCurrentOpenPositions retrieve all current open interest in the market.
func (p *Poloniex) GetV3FuturesCurrentOpenPositions(ctx context.Context, symbol string) (*OpenInterestData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *OpenInterestData
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/openInterest", params), &resp, true)
}

// GetInsuranceFundInformation query insurance fund information
func (p *Poloniex) GetInsuranceFundInformation(ctx context.Context) ([]InsuranceFundInfo, error) {
	var resp []InsuranceFundInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "/v3/market/insurance", &resp, true)
}

// GetV3FuturesRiskLimit retrieve information from the Futures Risk Limit Table.
func (p *Poloniex) GetV3FuturesRiskLimit(ctx context.Context, symbol string) ([]RiskLimit, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []RiskLimit
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/riskLimit", params), &resp, true)
}
