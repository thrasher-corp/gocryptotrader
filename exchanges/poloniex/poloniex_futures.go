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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	v3Path         = "/v3/"
	tradePathV3    = v3Path + "trade/"
	marketsPathV3  = v3Path + "market/"
	accountPathV3  = v3Path + "account/"
	positionPathV3 = v3Path + "position/"
)

// GetAccountBalance get information about your Futures account.
func (e *Exchange) GetAccountBalance(ctx context.Context) (*FuturesAccountBalance, error) {
	var resp *FuturesAccountBalance
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, accountPathV3+"balance", nil, nil, &resp)
}

// GetAccountBills retrieve the account’s bills.
func (e *Exchange) GetAccountBills(ctx context.Context, startTime, endTime time.Time, offset, limit uint64, direction, billType string) ([]*BillDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if offset > 0 {
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if billType != "" {
		params.Set("type", billType)
	}
	var resp []*BillDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, accountPathV3+"bills", params, nil, &resp)
}

// PlaceFuturesOrder place an order in futures trading.
func (e *Exchange) PlaceFuturesOrder(ctx context.Context, arg *FuturesOrderRequest) (*FuturesOrderIDResponse, error) {
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
		return nil, limits.ErrAmountBelowMin
	}
	var resp *FuturesOrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, tradePathV3+"order", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// PlaceFuturesMultipleOrders place orders in a batch. A maximum of 10 orders can be placed per request.
func (e *Exchange) PlaceFuturesMultipleOrders(ctx context.Context, args []FuturesOrderRequest) ([]*FuturesOrderIDResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		if err := validationOrderCreationParam(&args[x]); err != nil {
			return nil, err
		}
	}
	var resp []*FuturesOrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, tradePathV3+"orders", nil, args, &resp)
}

func validationOrderCreationParam(arg *FuturesOrderRequest) error {
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
		return limits.ErrAmountBelowMin
	}
	return nil
}

// CancelFuturesOrder cancels an order in futures trading.
func (e *Exchange) CancelFuturesOrder(ctx context.Context, arg *CancelOrderRequest) (*FuturesOrderIDResponse, error) {
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *FuturesOrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, tradePathV3+"order", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// CancelMultipleFuturesOrders cancel orders in a batch. A maximum of 10 orders can be cancelled per request.
func (e *Exchange) CancelMultipleFuturesOrders(ctx context.Context, args *CancelOrdersRequest) ([]*FuturesOrderIDResponse, error) {
	if args.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(args.OrderIDs) == 0 && len(args.ClientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*FuturesOrderIDResponse
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, tradePathV3+"batchOrders", nil, args, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

// CancelAllFuturesOrders cancel all current pending orders.
func (e *Exchange) CancelAllFuturesOrders(ctx context.Context, symbol, side string) ([]*FuturesOrderIDResponse, error) {
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
	var resp []*FuturesOrderIDResponse
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, tradePathV3+"allOrders", nil, arg, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

// CloseAtMarketPrice close orders at market price.
func (e *Exchange) CloseAtMarketPrice(ctx context.Context, symbol, marginMode, positionSide, clientOrderID string) (*FuturesOrderIDResponse, error) {
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
	var resp *FuturesOrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, tradePathV3+"position", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// CloseAllAtMarketPrice close all orders at market price.
func (e *Exchange) CloseAllAtMarketPrice(ctx context.Context) ([]*FuturesOrderIDResponse, error) {
	var resp []*FuturesOrderIDResponse
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, tradePathV3+"positionAll", nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

// GetCurrentFuturesOrders get unfilled futures orders. If no request parameters are specified, you will get all open orders sorted on the creation time in chronological order.
func (e *Exchange) GetCurrentFuturesOrders(ctx context.Context, symbol, side, orderID, clientOrderID, direction string, offset, limit uint64) ([]*FuturesOrderDetail, error) {
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
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, tradePathV3+"order/opens", params, nil, &resp)
}

// GetOrderExecutionDetails retrieves detailed information about your executed futures order
func (e *Exchange) GetOrderExecutionDetails(ctx context.Context, symbol, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*FuturesTradeFill, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
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
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesTradeFill
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, tradePathV3+"order/trades", params, nil, &resp)
}

// GetFuturesOrderHistory retrieves previous futures orders. Orders that are completely canceled (no transaction has occurred) initiated through the API can only be queried for 4 hours.
func (e *Exchange) GetFuturesOrderHistory(ctx context.Context, symbol, orderType, side, orderState, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*FuturesOrderDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
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
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, tradePathV3+"order/history", params, nil, &resp)
}

// GetFuturesCurrentPosition retrieves information about your current position.
func (e *Exchange) GetFuturesCurrentPosition(ctx context.Context, symbol string) ([]*FuturesPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []*FuturesPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, tradePathV3+"position/opens", params, nil, &resp)
}

// GetFuturesPositionHistory get information about previous positions.
func (e *Exchange) GetFuturesPositionHistory(ctx context.Context, symbol, marginMode, positionSide, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*FuturesPosition, error) {
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
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, tradePathV3+"position/history", params, nil, &resp)
}

// AdjustMarginForIsolatedMarginTradingPositions add or reduce margin for positions in isolated margin mode.
func (e *Exchange) AdjustMarginForIsolatedMarginTradingPositions(ctx context.Context, symbol, positionSide, adjustType string, amount float64) (*AdjustFuturesMarginResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
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
	var resp *AdjustFuturesMarginResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, tradePathV3+"position/margin", nil, arg, &resp)
}

// GetFuturesLeverage retrieves the list of leverage.
func (e *Exchange) GetFuturesLeverage(ctx context.Context, symbol, marginMode string) ([]*FuturesLeverage, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	var resp []*FuturesLeverage
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, positionPathV3+"leverages", params, nil, &resp)
}

// SetFuturesLeverage change leverage
func (e *Exchange) SetFuturesLeverage(ctx context.Context, symbol, marginMode, positionSide string, leverage int64) (*FuturesLeverage, error) {
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
	var resp *FuturesLeverage
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, positionPathV3+"leverage", nil, &map[string]string{
		"symbol":  symbol,
		"mgnMode": marginMode,
		"posSide": positionSide,
		"lever":   strconv.FormatInt(leverage, 10),
	}, &resp)
}

// SwitchPositionMode switch the current position mode. Please ensure you do not have open positions and open orders under this mode before the switch.
// Position mode, HEDGE: LONG/SHORT, ONE_WAY: BOTH
func (e *Exchange) SwitchPositionMode(ctx context.Context, positionMode string) error {
	if positionMode == "" {
		return errPositionModeInvalid
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, positionPathV3+"mode", nil, map[string]string{"posMode": positionMode}, &struct{}{})
}

// GetPositionMode get the current position mode.
func (e *Exchange) GetPositionMode(ctx context.Context) (string, error) {
	resp := &struct {
		PositionMode string `json:"posMode"`
	}{}
	return resp.PositionMode, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, positionPathV3+"mode", nil, nil, &resp)
}

// GetFuturesOrderBook get market depth data of the designated trading pair
func (e *Exchange) GetFuturesOrderBook(ctx context.Context, symbol string, depth, limit uint64) (*FuturesOrderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if depth > 0 {
		params.Set("scale", strconv.FormatUint(depth, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *FuturesOrderbook
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"orderBook", params), &resp)
}

// GetFuturesKlineData retrieves K-line data of the designated trading pair
func (e *Exchange) GetFuturesKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*FuturesCandle, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("interval", intervalString)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp FuturesCandles
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"candles", params), &resp)
}

// FuturesCandles holds and handles futures candles
type FuturesCandles []*FuturesCandle

// UnmarshalJSON deserializes byte data into list of FuturesCandle
func (t *FuturesCandles) UnmarshalJSON(data []byte) error {
	var result []*FuturesCandle
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*t = result
	return nil
}

// GetFuturesExecutionInfo get the latest execution information. The default limit is 500, with a maximum of 1,000.
func (e *Exchange) GetFuturesExecutionInfo(ctx context.Context, symbol string, limit uint64) ([]*FuturesExecutionInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesExecutionInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"trades", params), &resp)
}

// GetLiquidiationOrder get Liquidation Order Interface
func (e *Exchange) GetLiquidiationOrder(ctx context.Context, symbol, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*LiquidiationPrice, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
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
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*LiquidiationPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"liquidationOrder", params), &resp)
}

// GetFuturesMarketInfo get the market information of trading pairs in the past 24 hours.
func (e *Exchange) GetFuturesMarketInfo(ctx context.Context, symbol string) ([]*FuturesTickerDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []*FuturesTickerDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"tickers", params), &resp)
}

// GetFuturesIndexPrice get the current index price.
func (e *Exchange) GetFuturesIndexPrice(ctx context.Context, symbol string) (*InstrumentIndexPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *InstrumentIndexPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"indexPrice", params), &resp)
}

// GetIndexPriceComponents get the index price components for a trading pair.
func (e *Exchange) GetIndexPriceComponents(ctx context.Context, symbol string) (*IndexPriceComponent, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *IndexPriceComponent
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"indexPriceComponents", params), &resp)
}

// GetIndexPriceKlineData obtain the K-line data for the index price.
func (e *Exchange) GetIndexPriceKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*FuturesIndexPriceData, error) {
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
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesIndexPriceData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"indexPriceCandlesticks", params), &resp)
}

// GetFuturesMarkPrice get the current mark price.
func (e *Exchange) GetFuturesMarkPrice(ctx context.Context, symbol string) (*FuturesMarkPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FuturesMarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"markPrice", params), &resp)
}

// GetMarkPriceKlineData obtain the K-line data for the mark price.
func (e *Exchange) GetMarkPriceKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*FuturesMarkPriceCandle, error) {
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
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesMarkPriceCandle
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"markPriceCandlesticks", params), &resp)
}

// GetFuturesAllProductInfo inquire about the basic information of the all product.
func (e *Exchange) GetFuturesAllProductInfo(ctx context.Context, symbol string) ([]*ProductInfo, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []*ProductInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"allInstruments", params), &resp)
}

// GetFuturesProductInfo inquire about the basic information of the product.
func (e *Exchange) GetFuturesProductInfo(ctx context.Context, symbol string) (*ProductInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *ProductInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"instruments", params), &resp)
}

// GetFuturesCurrentFundingRate retrieve the current funding rate of the contract.
func (e *Exchange) GetFuturesCurrentFundingRate(ctx context.Context, symbol string) (*FuturesFundingRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesFundingRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"fundingRate", params), &resp)
}

// GetFuturesHistoricalFundingRates retrieve the previous funding rates of a contract.
func (e *Exchange) GetFuturesHistoricalFundingRates(ctx context.Context, symbol string, startTime, endTime time.Time, limit uint64) ([]*FuturesFundingRate, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesFundingRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"fundingRate/history", params), &resp)
}

// GetFuturesCurrentOpenPositions retrieve all current open interest in the market.
func (e *Exchange) GetFuturesCurrentOpenPositions(ctx context.Context, symbol string) (*OpenInterestData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *OpenInterestData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"openInterest", params), &resp)
}

// GetInsuranceFundInformation query insurance fund information
func (e *Exchange) GetInsuranceFundInformation(ctx context.Context) ([]*InsuranceFundInfo, error) {
	var resp []*InsuranceFundInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, marketsPathV3+"insurance", &resp)
}

// GetFuturesRiskLimit retrieve information from the Futures Risk Limit Table.
func (e *Exchange) GetFuturesRiskLimit(ctx context.Context, symbol string) ([]*RiskLimit, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []*RiskLimit
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues(marketsPathV3+"riskLimit", params), &resp)
}
