package poloniex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidPositionMode     = errors.New("invalid position mode")
	errInvalidMarginAdjustType = errors.New("invalid margin adjust type")
)

const (
	tradePathV3    = v3Path + "trade/"
	marketsPathV3  = v3Path + "market/"
	accountPathV3  = v3Path + "account/"
	positionPathV3 = v3Path + "position/"
)

// GetAccountBalance gets information about your Futures account.
func (e *Exchange) GetAccountBalance(ctx context.Context) (*FuturesAccountBalance, error) {
	var resp *FuturesAccountBalance
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetAccountBalanceEPL, http.MethodGet, accountPathV3+"balance", nil, nil, &resp)
}

// GetAccountBills retrieves the account’s bills.
func (e *Exchange) GetAccountBills(ctx context.Context, startTime, endTime time.Time, offset, limit uint64, direction, billType string) ([]*BillDetails, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
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
	var resp []*BillDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetBillsDetailsEPL, http.MethodGet, accountPathV3+"bills", params, nil, &resp)
}

// PlaceFuturesOrder places an order in futures trading.
func (e *Exchange) PlaceFuturesOrder(ctx context.Context, arg *FuturesOrderRequest) (*FuturesOrderIDResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *FuturesOrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fOrderEPL, http.MethodPost, tradePathV3+"order", nil, arg, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, err)
	}
	return resp, nil
}

// PlaceFuturesMultipleOrders places orders in a batch. A maximum of 10 orders can be placed per request.
func (e *Exchange) PlaceFuturesMultipleOrders(ctx context.Context, args []FuturesOrderRequest) ([]*FuturesOrderIDResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for i := range args {
		if err := args[i].validate(); err != nil {
			return nil, err
		}
	}
	resp, err := SendBatchValidatedAuthenticatedHTTPRequest[*FuturesOrderIDResponse](ctx, e, exchange.RestSpot, fBatchOrdersEPL, http.MethodPost, tradePathV3+"orders", nil, args)
	if err != nil {
		return resp, fmt.Errorf("%w: %w", order.ErrPlaceFailed, err)
	}
	return resp, nil
}

func (o *FuturesOrderRequest) validate() error {
	if o.Symbol.IsEmpty() {
		return currency.ErrSymbolStringEmpty
	}
	if o.Side == "" {
		return order.ErrSideIsInvalid
	}
	if (o.MarginMode != MarginMode(margin.Unset) && o.PositionSide == order.UnknownSide) ||
		(o.MarginMode == MarginMode(margin.Unset) && o.PositionSide != order.UnknownSide) {
		return fmt.Errorf("%w: %w: either both margin mode and position side fields are filled or left blank", order.ErrSideIsInvalid, margin.ErrInvalidMarginType)
	}
	if o.OrderType == OrderType(order.UnknownType) {
		return order.ErrTypeIsInvalid
	}
	if o.Size <= 0 {
		return limits.ErrAmountBelowMin
	}
	return nil
}

// CancelFuturesOrder cancels an order in futures trading.
func (e *Exchange) CancelFuturesOrder(ctx context.Context, arg *CancelOrderRequest) (*FuturesOrderIDResponse, error) {
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *FuturesOrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fCancelOrderEPL, http.MethodDelete, tradePathV3+"order", nil, arg, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// CancelMultipleFuturesOrders cancels orders in a batch. A maximum of 10 orders can be cancelled per request.
func (e *Exchange) CancelMultipleFuturesOrders(ctx context.Context, args *CancelFuturesOrdersRequest) ([]*FuturesOrderIDResponse, error) {
	if args.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(args.OrderIDs) == 0 && len(args.ClientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	resp, err := SendBatchValidatedAuthenticatedHTTPRequest[*FuturesOrderIDResponse](ctx, e, exchange.RestSpot, fCancelBatchOrdersEPL, http.MethodDelete, tradePathV3+"batchOrders", nil, args)
	if err != nil {
		return resp, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// CancelFuturesOrders cancels all current pending orders.
func (e *Exchange) CancelFuturesOrders(ctx context.Context, symbol currency.Pair, side string) ([]*FuturesOrderIDResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	arg := &struct {
		Symbol string `json:"symbol"`
		Side   string `json:"side,omitempty"`
	}{
		Symbol: symbol.String(),
		Side:   side,
	}
	resp, err := SendBatchValidatedAuthenticatedHTTPRequest[*FuturesOrderIDResponse](ctx, e, exchange.RestSpot, fCancelAllLimitOrdersEPL, http.MethodDelete, tradePathV3+"allOrders", nil, arg)
	if err != nil {
		return resp, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// CloseAtMarketPrice closes orders at market price.
func (e *Exchange) CloseAtMarketPrice(ctx context.Context, symbol currency.Pair, marginMode, positionSide, clientOrderID string) (*FuturesOrderIDResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	arg := &struct {
		Symbol       string `json:"symbol"`
		MgnMode      string `json:"mgnMode"`
		ClOrdID      string `json:"clOrdId,omitempty"`
		PositionSide string `json:"posSide,omitempty"`
	}{
		Symbol:       symbol.String(),
		MgnMode:      marginMode,
		ClOrdID:      clientOrderID,
		PositionSide: positionSide,
	}
	var resp *FuturesOrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fCancelPositionAtMarketPriceEPL, http.MethodPost, tradePathV3+"position", nil, arg, &resp)
}

// CloseAllPositionsAtMarketPrice closes all orders at market price.
func (e *Exchange) CloseAllPositionsAtMarketPrice(ctx context.Context) ([]*FuturesOrderIDResponse, error) {
	resp, err := SendBatchValidatedAuthenticatedHTTPRequest[*FuturesOrderIDResponse](ctx, e, exchange.RestSpot, fCancelAllPositionsAtMarketPriceEPL, http.MethodPost, tradePathV3+"positionAll", nil, nil)
	if err != nil {
		return resp, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	return resp, nil
}

// GetCurrentFuturesOrders gets unfilled futures orders. If no request parameters are specified, you will get all open orders sorted by creation time in chronological order.
func (e *Exchange) GetCurrentFuturesOrders(ctx context.Context, symbol currency.Pair, side, clientOrderID, direction string, orderID, offsetOrderID, limit uint64) ([]*FuturesOrderDetails, error) {
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if orderID > 0 {
		params.Set("ordId", strconv.FormatUint(orderID, 10))
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offsetOrderID > 0 {
		params.Set("from", strconv.FormatUint(offsetOrderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesOrderDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetOrdersEPL, http.MethodGet, tradePathV3+"order/opens", params, nil, &resp)
}

// GetOrderExecutionDetails retrieves detailed information about your executed futures order
func (e *Exchange) GetOrderExecutionDetails(ctx context.Context, side order.Side, symbol currency.Pair, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*FuturesTradeFill, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if side != order.UnknownSide {
		params.Set("side", side.String())
	}
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetFillsV2EPL, http.MethodGet, tradePathV3+"order/trades", params, nil, &resp)
}

// GetFuturesOrderHistory retrieves previous futures orders. Orders initiated through the API that are completely canceled (no transaction has occurred) can only be queried for 4 hours.
func (e *Exchange) GetFuturesOrderHistory(ctx context.Context, symbol currency.Pair, side order.Side, orderType, orderState, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*FuturesOrderDetails, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if side != order.UnknownSide {
		params.Set("side", side.String())
	}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
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
	var resp []*FuturesOrderDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetOrderHistoryEPL, http.MethodGet, tradePathV3+"order/history", params, nil, &resp)
}

// GetFuturesCurrentPosition retrieves information about your current position.
func (e *Exchange) GetFuturesCurrentPosition(ctx context.Context, symbol currency.Pair) ([]*OpenFuturesPosition, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*OpenFuturesPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetPositionOpenEPL, http.MethodGet, tradePathV3+"position/opens", params, nil, &resp)
}

// GetFuturesPositionHistory gets information about previous positions.
func (e *Exchange) GetFuturesPositionHistory(ctx context.Context, symbol currency.Pair, marginMode, positionSide, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*FuturesPosition, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	if positionSide != "" {
		params.Set("posSide", positionSide)
	}
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetPositionHistoryEPL, http.MethodGet, tradePathV3+"position/history", params, nil, &resp)
}

// AdjustMarginForIsolatedMarginTradingPositions adds or reduces margin for positions in isolated margin mode.
func (e *Exchange) AdjustMarginForIsolatedMarginTradingPositions(ctx context.Context, symbol currency.Pair, positionSide, adjustType string, amount float64) (*AdjustFuturesMarginResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if adjustType == "" {
		return nil, errInvalidMarginAdjustType
	}
	arg := &struct {
		Symbol       currency.Pair `json:"symbol"`
		PositionSide string        `json:"posSide,omitempty"`
		Amount       float64       `json:"amt,string"`
		Type         string        `json:"type"`
	}{
		Symbol:       symbol,
		PositionSide: positionSide,
		Amount:       amount,
		Type:         adjustType,
	}
	var resp *AdjustFuturesMarginResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fAdjustMarginEPL, http.MethodPost, tradePathV3+"position/margin", nil, arg, &resp)
}

// GetFuturesLeverage retrieves the list of leverage.
func (e *Exchange) GetFuturesLeverage(ctx context.Context, symbol currency.Pair, marginMode string) ([]*FuturesLeverage, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	var resp []*FuturesLeverage
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetPositionLeverageEPL, http.MethodGet, positionPathV3+"leverages", params, nil, &resp)
}

// SetFuturesLeverage changes leverage
func (e *Exchange) SetFuturesLeverage(ctx context.Context, symbol currency.Pair, marginMode, positionSide string, leverage uint64) (*FuturesLeverage, error) {
	if symbol.IsEmpty() {
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
	arg := &struct {
		Symbol       string `json:"symbol"`
		MarginMode   string `json:"mgnMode"`
		PositionSide string `json:"posSide"`
		Leverage     uint64 `json:"lever,string"`
	}{
		Symbol:       symbol.String(),
		MarginMode:   marginMode,
		PositionSide: positionSide,
		Leverage:     leverage,
	}
	var resp *FuturesLeverage
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fSetPositionLeverageEPL, http.MethodPost, positionPathV3+"leverage", nil, arg, &resp)
}

// SwitchPositionMode switches the current position mode. Please ensure you do not have open positions and open orders under this mode before the switch.
// Position mode, HEDGE: LONG/SHORT, ONE_WAY: BOTH
func (e *Exchange) SwitchPositionMode(ctx context.Context, positionMode string) error {
	if positionMode != "HEDGE" && positionMode != "ONE_WAY" {
		return errInvalidPositionMode
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fSwitchPositionModeEPL, http.MethodPost, positionPathV3+"mode", nil, map[string]string{"posMode": positionMode}, nil)
}

// GetPositionMode gets the current position mode.
func (e *Exchange) GetPositionMode(ctx context.Context) (string, error) {
	resp := &struct {
		PositionMode string `json:"posMode"`
	}{}
	return resp.PositionMode, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetPositionModeEPL, http.MethodGet, positionPathV3+"mode", nil, nil, &resp)
}

// GetUserPositionRiskLimit retrieves the user's position risk limit.
func (e *Exchange) GetUserPositionRiskLimit(ctx context.Context, symbol currency.Pair, marginMode, positionSide string) ([]*UserPositionRiskLimit, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	if positionSide != "" {
		params.Set("posSide", positionSide)
	}
	var resp []*UserPositionRiskLimit
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetPositionRiskLimitEPL, http.MethodGet, positionPathV3+"riskLimit", params, nil, &resp)
}

// GetFuturesOrderBook gets market depth data for the designated trading pair
func (e *Exchange) GetFuturesOrderBook(ctx context.Context, symbol currency.Pair, depth, limit uint64) (*FuturesOrderbook, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if depth > 0 {
		params.Set("scale", strconv.FormatUint(depth, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *FuturesOrderbook
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"orderBook", params), &resp)
}

// GetFuturesKlineData retrieves K-line data for the designated trading pair
func (e *Exchange) GetFuturesKlineData(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*FuturesCandle, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("interval", intervalString)
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesCandle
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fCandlestickEPL, common.EncodeURLValues(marketsPathV3+"candles", params), &resp)
}

// GetFuturesExecution gets the latest execution information. The default limit is 500, with a maximum of 1,000.
func (e *Exchange) GetFuturesExecution(ctx context.Context, symbol currency.Pair, limit uint64) ([]*FuturesExecutionInfo, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesExecutionInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"trades", params), &resp)
}

// GetLiquidationOrder gets details of previous liquidation events
func (e *Exchange) GetLiquidationOrder(ctx context.Context, symbol currency.Pair, direction string, startTime, endTime time.Time, offset, limit uint64) ([]*LiquidationPrice, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if offset > 0 {
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*LiquidationPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"liquidationOrder", params), &resp)
}

// GetFuturesMarket get the market information of trading pairs in the past 24 hours.
func (e *Exchange) GetFuturesMarket(ctx context.Context, symbol currency.Pair) ([]*FuturesTickerDetails, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*FuturesTickerDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"tickers", params), &resp)
}

// GetFuturesIndexPrice get the current index price.
func (e *Exchange) GetFuturesIndexPrice(ctx context.Context, symbol currency.Pair) (*InstrumentIndexPrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *InstrumentIndexPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"indexPrice", params), &resp)
}

// GetFuturesIndexPrices get the current index price for all instruments
func (e *Exchange) GetFuturesIndexPrices(ctx context.Context) ([]*InstrumentIndexPrice, error) {
	var resp []*InstrumentIndexPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, marketsPathV3+"indexPrice", &resp)
}

// GetIndexPriceComponents get the index price components for a trading pair.
func (e *Exchange) GetIndexPriceComponents(ctx context.Context, symbol currency.Pair) (*IndexPriceComponent, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp []*IndexPriceComponent
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"indexPriceComponents", params), &resp); err != nil {
		return nil, fmt.Errorf("%w %w", order.ErrGetFailed, err)
	}
	if len(resp) != 1 {
		return nil, fmt.Errorf("%w %w", order.ErrGetFailed, common.ErrInvalidResponse)
	}
	return resp[0], nil
}

// GetAllIndexPriceComponents returns index price components for all pairs.
func (e *Exchange) GetAllIndexPriceComponents(ctx context.Context) ([]*IndexPriceComponent, error) {
	var resp []*IndexPriceComponent
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, marketsPathV3+"indexPriceComponents", &resp)
}

// GetIndexPriceKlineData obtains the K-line data for the index price.
func (e *Exchange) GetIndexPriceKlineData(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*FuturesIndexPriceData, error) {
	return e.getIndexPriceKlineData(ctx, symbol, interval, startTime, endTime, limit, "indexPriceCandlesticks", fCandlestickEPL)
}

// GetPremiumIndexKlineData retrieves the K-line information of the premium index.
func (e *Exchange) GetPremiumIndexKlineData(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*FuturesIndexPriceData, error) {
	return e.getIndexPriceKlineData(ctx, symbol, interval, startTime, endTime, limit, "premiumIndexCandlesticks", fPremiumCandlestickEPL)
}

func (e *Exchange) getIndexPriceKlineData(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit uint64, path string, epl request.EndpointLimit) ([]*FuturesIndexPriceData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("interval", intervalString)
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesIndexPriceData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, common.EncodeURLValues(marketsPathV3+path, params), &resp)
}

// GetFuturesMarkPrice get the current mark price.
func (e *Exchange) GetFuturesMarkPrice(ctx context.Context, symbol currency.Pair) (*FuturesMarkPrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *FuturesMarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"markPrice", params), &resp)
}

// GetFuturesMarkPrices get the current mark price for all instruments
func (e *Exchange) GetFuturesMarkPrices(ctx context.Context) ([]*FuturesMarkPrice, error) {
	var resp []*FuturesMarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, marketsPathV3+"markPrice", &resp)
}

// GetMarkPriceKlineData obtains the K-line data for the mark price.
func (e *Exchange) GetMarkPriceKlineData(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*FuturesMarkPriceCandle, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("interval", intervalString)
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesMarkPriceCandle
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fCandlestickEPL, common.EncodeURLValues(marketsPathV3+"markPriceCandlesticks", params), &resp)
}

// GetFuturesAllProducts inquires about the basic information of all products.
func (e *Exchange) GetFuturesAllProducts(ctx context.Context, symbol currency.Pair) ([]*ProductDetail, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*ProductDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"allInstruments", params), &resp)
}

// GetFuturesProduct inquires about the basic information of one product.
func (e *Exchange) GetFuturesProduct(ctx context.Context, symbol currency.Pair) (*ProductDetail, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *ProductDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"instruments", params), &resp)
}

// GetFuturesCurrentFundingRate retrieves the current funding rate of the contract.
func (e *Exchange) GetFuturesCurrentFundingRate(ctx context.Context, symbol currency.Pair) (*FuturesFundingRate, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *FuturesFundingRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"fundingRate", params), &resp)
}

// GetFuturesHistoricalFundingRates retrieves the previous funding rates of a contract.
func (e *Exchange) GetFuturesHistoricalFundingRates(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, limit uint64) ([]*FuturesFundingRate, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if !startTime.IsZero() {
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*FuturesFundingRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fCandlestickEPL, common.EncodeURLValues(marketsPathV3+"fundingRate/history", params), &resp)
}

// GetContractOpenInterest retrieve all current open interest in the market.
func (e *Exchange) GetContractOpenInterest(ctx context.Context, symbol currency.Pair) (*OpenInterestData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *OpenInterestData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"openInterest", params), &resp)
}

// GetInsuranceFund query insurance fund information
func (e *Exchange) GetInsuranceFund(ctx context.Context) ([]*InsuranceFundInfo, error) {
	var resp []*InsuranceFundInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, marketsPathV3+"insurance", &resp)
}

// GetFuturesRiskLimit retrieve information from the Futures Risk Limit Table.
func (e *Exchange) GetFuturesRiskLimit(ctx context.Context, symbol currency.Pair, marginMode string, tier uint8) ([]*RiskLimit, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	if tier > 0 {
		params.Set("tier", strconv.Itoa(int(tier)))
	}
	var resp []*RiskLimit
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"riskLimit", params), &resp)
}

// GetContractLimitPrice query the highest buy price and the lowest sell price of the current contract trading pair.
func (e *Exchange) GetContractLimitPrice(ctx context.Context, symbols currency.Pairs) ([]ContractLimitPrice, error) {
	if len(symbols) == 0 || (len(symbols) == 1 && symbols[0].IsEmpty()) {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbols.Join())
	var resp []ContractLimitPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fMarketEPL, common.EncodeURLValues(marketsPathV3+"limitPrice", params), &resp)
}
