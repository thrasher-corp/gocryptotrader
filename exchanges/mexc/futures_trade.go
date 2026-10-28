package mexc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// futuresBatchLimit is the most orders one batch request places, cancels or reads
	futuresBatchLimit = 50
	// futuresExternalOrderIDLimit is the longest external order ID the venue takes
	futuresExternalOrderIDLimit = 32
	// futuresDealsPageSizeLimit is the page size cap of the deals and closed orders lists
	futuresDealsPageSizeLimit = 1000
)

// validFuturesSide reports whether a side is one of the four futures order sides
func validFuturesSide(side uint8) bool {
	return side >= FuturesSideOpenLong && side <= FuturesSideCloseLong
}

// validFuturesOpenType reports whether an open type is isolated or cross margin
func validFuturesOpenType(openType uint8) bool {
	return openType == FuturesOpenTypeIsolated || openType == FuturesOpenTypeCross
}

// validFuturesPriceType reports whether a trigger price type is the latest, fair or index price; 0 leaves the
// venue's default where the type is optional
func validFuturesPriceType(priceType uint8, optional bool) bool {
	return (optional && priceType == 0) || (priceType >= FuturesPriceTypeLatest && priceType <= FuturesPriceTypeIndex)
}

// isFuturesLimitOrderType reports whether an order type rests at a limit price
func isFuturesLimitOrderType(orderType uint8) bool {
	return orderType >= FuturesOrderTypeLimit && orderType <= FuturesOrderTypeFillOrKill
}

// validateFuturesOrder checks an order before it is placed alone or in a batch
func validateFuturesOrder(o *FuturesOrderRequest) error {
	if err := common.NilGuard(o); err != nil {
		return err
	}
	if o.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if !validFuturesSide(o.Side) {
		return fmt.Errorf("%w: %d", errInvalidOrderSide, o.Side)
	}
	if o.SideType != "" && o.SideType != FuturesSideTypeBuy && o.SideType != FuturesSideTypeSell {
		return fmt.Errorf("%w: %q", errInvalidSideType, o.SideType)
	}
	if o.Type < FuturesOrderTypeLimit || o.Type > FuturesOrderTypeMarketToLimit {
		return fmt.Errorf("%w: %d", order.ErrUnsupportedOrderType, o.Type)
	}
	if !validFuturesOpenType(o.OpenType) {
		return fmt.Errorf("%w: %d", errInvalidOpenType, o.OpenType)
	}
	if o.Volume <= 0 {
		return fmt.Errorf("%w: volume %v", limits.ErrAmountBelowMin, o.Volume)
	}
	// Only the limit family is priced; the venue ignores a market order's price
	if isFuturesLimitOrderType(o.Type) && o.BBOTypeNumber == 0 && o.Price <= 0 {
		return fmt.Errorf("%w: price %v", limits.ErrPriceBelowMin, o.Price)
	}
	// An isolated position takes its leverage from the order that opens it
	if o.OpenType == FuturesOpenTypeIsolated && (o.Side == FuturesSideOpenLong || o.Side == FuturesSideOpenShort) && o.Leverage == 0 {
		return fmt.Errorf("%w: an isolated position needs leverage to open", errInvalidLeverage)
	}
	if len(o.ExternalOrderID) > futuresExternalOrderIDLimit {
		return fmt.Errorf("%w: %q", errExternalOrderIDTooLong, o.ExternalOrderID)
	}
	if o.BBOTypeNumber > 4 {
		return fmt.Errorf("%w: BBO type %d", order.ErrUnsupportedOrderType, o.BBOTypeNumber)
	}
	if o.BBOTypeNumber > 0 && o.ExternalOrderID != "" {
		return fmt.Errorf("%w: a BBO order takes no external order id", order.ErrClientOrderIDNotSupported)
	}
	if !validFuturesPriceType(o.LossTrend, true) || !validFuturesPriceType(o.ProfitTrend, true) {
		return fmt.Errorf("%w: stop-loss %d, take-profit %d", errInvalidPriceType, o.LossTrend, o.ProfitTrend)
	}
	if o.PositionMode != 0 && o.PositionMode != FuturesPositionModeHedge && o.PositionMode != FuturesPositionModeOneWay {
		return fmt.Errorf("%w: %d", errInvalidPositionMode, o.PositionMode)
	}
	if o.SelfTradePreventionMode > 3 {
		return fmt.Errorf("%w: %d", errInvalidSelfTradePrevention, o.SelfTradePreventionMode)
	}
	return nil
}

// parseFuturesOrderID returns an order ID as the number the venue's request bodies take
func parseFuturesOrderID(orderID string) (uint64, error) {
	if orderID == "" {
		return 0, order.ErrOrderIDNotSet
	}
	id, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: %q", errInvalidFuturesID, orderID)
	}
	return id, nil
}

// joinFuturesStates renders order states as the comma separated list the venue takes
func joinFuturesStates(states []uint8) string {
	parts := make([]string, len(states))
	for i := range states {
		parts[i] = strconv.FormatUint(uint64(states[i]), 10)
	}
	return strings.Join(parts, ",")
}

// PlaceFuturesOrder places a futures order, which must name its side type
func (e *Exchange) PlaceFuturesOrder(ctx context.Context, req *FuturesOrderRequest) (*FuturesOrderPlacementResponse, error) {
	if err := validateFuturesOrder(req); err != nil {
		return nil, err
	}
	if req.SideType == "" {
		return nil, fmt.Errorf("%w: unset", errInvalidSideType)
	}
	var resp *FuturesOrderPlacementResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresPlaceOrderEPL, http.MethodPost, "private/order/create", nil, req, &resp, true)
}

// PlaceFuturesBatchOrders places up to 50 futures orders in one request. Only market maker accounts can use it,
// and each order's outcome is reported separately
func (e *Exchange) PlaceFuturesBatchOrders(ctx context.Context, orders []FuturesOrderRequest) ([]*FuturesBatchOrderResponse, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(orders) > futuresBatchLimit {
		return nil, fmt.Errorf("%w: %d orders, at most %d", errTooManyOrders, len(orders), futuresBatchLimit)
	}
	for i := range orders {
		if err := validateFuturesOrder(&orders[i]); err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
	}
	var resp []*FuturesBatchOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresBatchPlaceOrdersEPL, http.MethodPost, "private/order/submit_batch", nil, orders, &resp, true)
}

// ChaseFuturesOrder moves a limit order's price to the best price on its side of the book
func (e *Exchange) ChaseFuturesOrder(ctx context.Context, orderID uint64) error {
	if orderID == 0 {
		return order.ErrOrderIDNotSet
	}
	body := &FuturesChaseOrderRequest{OrderID: orderID}
	return e.SendFuturesHTTPRequest(ctx, futuresChaseOrderEPL, http.MethodPost, "private/order/chase_limit_order", nil, body, nil, true)
}

// ModifyFuturesOrder changes an unfilled limit order's price and volume and returns the order's ID
func (e *Exchange) ModifyFuturesOrder(ctx context.Context, req *FuturesModifyOrderRequest) (FuturesID, error) {
	if err := common.NilGuard(req); err != nil {
		return "", err
	}
	if req.OrderID == 0 {
		return "", order.ErrOrderIDNotSet
	}
	if req.Price <= 0 {
		return "", fmt.Errorf("%w: price %v", limits.ErrPriceBelowMin, req.Price)
	}
	if req.Volume <= 0 {
		return "", fmt.Errorf("%w: volume %v", limits.ErrAmountBelowMin, req.Volume)
	}
	var resp FuturesID
	return resp, e.SendFuturesHTTPRequest(ctx, futuresModifyOrderEPL, http.MethodPost, "private/order/change_limit_order", nil, req, &resp, true)
}

// CancelFuturesOrders cancels up to 50 orders; each order's outcome is reported separately
func (e *Exchange) CancelFuturesOrders(ctx context.Context, orderIDs []uint64) ([]*FuturesCancelOrderResponse, error) {
	if len(orderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if len(orderIDs) > futuresBatchLimit {
		return nil, fmt.Errorf("%w: %d orders, at most %d", errTooManyOrders, len(orderIDs), futuresBatchLimit)
	}
	if slices.Contains(orderIDs, 0) {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*FuturesCancelOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresCancelOrdersEPL, http.MethodPost, "private/order/cancel", nil, orderIDs, &resp, true)
}

// validateExternalOrders checks orders named by contract and external order ID
func validateExternalOrders(orders []FuturesExternalOrderRequest) error {
	if len(orders) == 0 {
		return errExternalOrderIDRequired
	}
	if len(orders) > futuresBatchLimit {
		return fmt.Errorf("%w: %d orders, at most %d", errTooManyOrders, len(orders), futuresBatchLimit)
	}
	for i := range orders {
		if orders[i].Symbol == "" {
			return currency.ErrSymbolStringEmpty
		}
		if orders[i].ExternalOrderID == "" {
			return errExternalOrderIDRequired
		}
	}
	return nil
}

// CancelFuturesOrdersByExternalID cancels orders named by contract and external order ID; each order's outcome is
// reported separately
func (e *Exchange) CancelFuturesOrdersByExternalID(ctx context.Context, orders []FuturesExternalOrderRequest) ([]*FuturesCancelOrderResponse, error) {
	if err := validateExternalOrders(orders); err != nil {
		return nil, err
	}
	var resp []*FuturesCancelOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresBatchCancelByExternalIDEPL, http.MethodPost, "private/order/batch_cancel_with_external", nil, orders, &resp, true)
}

// CancelFuturesOrderByExternalID cancels an open order named by contract and external order ID
func (e *Exchange) CancelFuturesOrderByExternalID(ctx context.Context, symbol currency.Pair, externalOrderID string) ([]*FuturesCancelOrderResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if externalOrderID == "" {
		return nil, errExternalOrderIDRequired
	}
	body := &FuturesExternalOrderRequest{Symbol: symbol.String(), ExternalOrderID: externalOrderID}
	var resp oneOrMany[*FuturesCancelOrderResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresCancelByExternalIDEPL, http.MethodPost, "private/order/cancel_with_external", nil, body, &resp, true)
}

// CancelAllFuturesOrders cancels every open order of a contract, or of every contract when symbol is unset
func (e *Exchange) CancelAllFuturesOrders(ctx context.Context, symbol currency.Pair) error {
	// The venue cancels every contract's orders for an empty symbol, so the field is always sent
	body := &FuturesCancelAllOrdersRequest{}
	if !symbol.IsEmpty() {
		body.Symbol = symbol.String()
	}
	return e.SendFuturesHTTPRequest(ctx, futuresCancelAllOrdersEPL, http.MethodPost, "private/order/cancel_all", nil, body, nil, true)
}

// ReverseFuturesPosition closes a position and opens one of the same volume on the other side
func (e *Exchange) ReverseFuturesPosition(ctx context.Context, req *FuturesReversePositionRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if req.PositionID == 0 {
		return errPositionIDRequired
	}
	if req.Volume <= 0 {
		return fmt.Errorf("%w: volume %v", limits.ErrAmountBelowMin, req.Volume)
	}
	return e.SendFuturesHTTPRequest(ctx, futuresReversePositionEPL, http.MethodPost, "private/position/reverse", nil, req, nil, true)
}

// CloseAllFuturesPositions closes every open position at the market price
func (e *Exchange) CloseAllFuturesPositions(ctx context.Context) error {
	return e.SendFuturesHTTPRequest(ctx, futuresCloseAllPositionsEPL, http.MethodPost, "private/position/close_all", nil, struct{}{}, nil, true)
}

// GetFuturesOpenOrderCounts returns the numbers of the account's open orders by kind
func (e *Exchange) GetFuturesOpenOrderCounts(ctx context.Context) (*FuturesOpenOrderCountsResponse, error) {
	var resp *FuturesOpenOrderCountsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOpenOrderCountsEPL, http.MethodGet, "private/order/open_order_total_count", nil, nil, &resp, true)
}

// GetFuturesOrderByExternalID returns an order named by contract and external order ID
func (e *Exchange) GetFuturesOrderByExternalID(ctx context.Context, symbol currency.Pair, externalOrderID string) (*FuturesOrderResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if externalOrderID == "" {
		return nil, errExternalOrderIDRequired
	}
	var resp *FuturesOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrderByExternalIDEPL, http.MethodGet, "private/order/external/"+symbol.String()+"/"+url.PathEscape(externalOrderID), nil, nil, &resp, true)
}

// GetFuturesOrder returns an order named by its order ID
func (e *Exchange) GetFuturesOrder(ctx context.Context, orderID string) (*FuturesOrderResponse, error) {
	if _, err := parseFuturesOrderID(orderID); err != nil {
		return nil, err
	}
	var resp *FuturesOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrderEPL, http.MethodGet, "private/order/get/"+orderID, nil, nil, &resp, true)
}

// GetFuturesOrdersByID returns up to 50 orders named by their order IDs
func (e *Exchange) GetFuturesOrdersByID(ctx context.Context, orderIDs []string) ([]*FuturesOrderResponse, error) {
	if len(orderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if len(orderIDs) > futuresBatchLimit {
		return nil, fmt.Errorf("%w: %d orders, at most %d", errTooManyOrders, len(orderIDs), futuresBatchLimit)
	}
	for _, id := range orderIDs {
		if _, err := parseFuturesOrderID(id); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("order_ids", strings.Join(orderIDs, ","))
	var resp []*FuturesOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrdersByIDEPL, http.MethodGet, "private/order/batch_query", params, nil, &resp, true)
}

// GetFuturesOrdersByExternalID returns orders named by contract and external order ID
func (e *Exchange) GetFuturesOrdersByExternalID(ctx context.Context, orders []FuturesExternalOrderRequest) ([]*FuturesOrderResponse, error) {
	if err := validateExternalOrders(orders); err != nil {
		return nil, err
	}
	var resp []*FuturesOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrdersByExternalIDEPL, http.MethodPost, "private/order/batch_query_with_external", nil, orders, &resp, true)
}

// GetFuturesOpenOrders returns a page of the account's open orders across every contract; the page size is at most
// 100
func (e *Exchange) GetFuturesOpenOrders(ctx context.Context, page *FuturesPageRequest) ([]*FuturesOrderResponse, error) {
	params := url.Values{}
	if err := setFuturesPage(params, page, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesOrderResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOpenOrdersEPL, http.MethodGet, "private/order/list/open_orders", params, nil, &resp, true)
}

// GetFuturesOrderHistory returns a page of a contract's orders in the states asked for, newest first; the page size
// is at most 100, and without a window the venue returns the last 7 days
func (e *Exchange) GetFuturesOrderHistory(ctx context.Context, req *FuturesOrderHistoryRequest) ([]*FuturesOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if len(req.States) == 0 {
		return nil, errOrderStatesRequired
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol.String())
	params.Set("states", joinFuturesStates(req.States))
	if req.Category != 0 {
		params.Set("category", strconv.FormatUint(uint64(req.Category), 10))
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if err := setFuturesTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := setFuturesPage(params, &req.FuturesPageRequest, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesOrderResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrderHistoryEPL, http.MethodGet, "private/order/list/history_orders", params, nil, &resp, true)
}

// futuresDealsParams returns the parameters of a deals or closed orders request
func futuresDealsParams(req *FuturesOrderDealsRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol.String())
	if err := setFuturesTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := setFuturesPage(params, &req.FuturesPageRequest, futuresDealsPageSizeLimit); err != nil {
		return nil, err
	}
	return params, nil
}

// GetFuturesOrderDeals returns a page of a contract's fills, newest first; the page size is at most 1000
func (e *Exchange) GetFuturesOrderDeals(ctx context.Context, req *FuturesOrderDealsRequest) ([]*FuturesOrderDealResponse, error) {
	params, err := futuresDealsParams(req)
	if err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesOrderDealResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrderDealsEPL, http.MethodGet, "private/order/list/order_deals/v3", params, nil, &resp, true)
}

// GetFuturesOrderDealDetails returns an order's fills
func (e *Exchange) GetFuturesOrderDealDetails(ctx context.Context, orderID string) ([]*FuturesOrderDealResponse, error) {
	if _, err := parseFuturesOrderID(orderID); err != nil {
		return nil, err
	}
	var resp []*FuturesOrderDealResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOrderDealDetailsEPL, http.MethodGet, "private/order/deal_details/"+orderID, nil, nil, &resp, true)
}

// GetFuturesClosedOrders returns a page of a contract's finished orders, newest first; the page size is at most 1000
func (e *Exchange) GetFuturesClosedOrders(ctx context.Context, req *FuturesOrderDealsRequest) ([]*FuturesOrderResponse, error) {
	params, err := futuresDealsParams(req)
	if err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesOrderResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresClosedOrdersEPL, http.MethodGet, "private/order/list/close_orders", params, nil, &resp, true)
}

// GetFuturesPlanOrders returns a page of the account's plan orders within a window; the page size is at most 100
func (e *Exchange) GetFuturesPlanOrders(ctx context.Context, req *FuturesPlanOrdersRequest) ([]*FuturesPlanOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Side != 0 && !validFuturesSide(req.Side) {
		return nil, fmt.Errorf("%w: %d", errInvalidOrderSide, req.Side)
	}
	params := url.Values{}
	if !req.Symbol.IsEmpty() {
		params.Set("symbol", req.Symbol.String())
	}
	if len(req.States) > 0 {
		params.Set("states", joinFuturesStates(req.States))
	}
	if req.Side != 0 {
		params.Set("side", strconv.FormatUint(uint64(req.Side), 10))
	}
	if err := setFuturesTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := setFuturesPage(params, &req.FuturesPageRequest, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesPlanOrderResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresPlanOrdersEPL, http.MethodGet, "private/planorder/list/orders", params, nil, &resp, true)
}

// PlaceFuturesPlanOrder places a plan order, which places an order once its trigger price is reached, and returns
// its ID
func (e *Exchange) PlaceFuturesPlanOrder(ctx context.Context, req *FuturesPlanOrderRequest) (FuturesID, error) {
	if err := common.NilGuard(req); err != nil {
		return "", err
	}
	if req.Symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	if !validFuturesSide(req.Side) {
		return "", fmt.Errorf("%w: %d", errInvalidOrderSide, req.Side)
	}
	if !validFuturesOpenType(req.OpenType) {
		return "", fmt.Errorf("%w: %d", errInvalidOpenType, req.OpenType)
	}
	if req.Volume <= 0 {
		return "", fmt.Errorf("%w: volume %v", limits.ErrAmountBelowMin, req.Volume)
	}
	if req.Leverage == 0 {
		return "", errInvalidLeverage
	}
	if req.TriggerPrice <= 0 {
		return "", fmt.Errorf("%w: trigger price %v", limits.ErrPriceBelowMin, req.TriggerPrice)
	}
	if req.TriggerType != FuturesTriggerGreaterOrEqual && req.TriggerType != FuturesTriggerLessOrEqual {
		return "", fmt.Errorf("%w: %d", errInvalidTriggerType, req.TriggerType)
	}
	if req.ExecuteCycle != FuturesExecuteCycle24Hours && req.ExecuteCycle != FuturesExecuteCycle7Days {
		return "", fmt.Errorf("%w: %d", errInvalidExecutionCycle, req.ExecuteCycle)
	}
	if req.OrderType < FuturesOrderTypeLimit || req.OrderType > FuturesOrderTypeMarket {
		return "", fmt.Errorf("%w: %d", order.ErrUnsupportedOrderType, req.OrderType)
	}
	if isFuturesLimitOrderType(req.OrderType) && req.Price <= 0 {
		return "", fmt.Errorf("%w: price %v", limits.ErrPriceBelowMin, req.Price)
	}
	if !validFuturesPriceType(req.Trend, false) || !validFuturesPriceType(req.LossTrend, true) || !validFuturesPriceType(req.ProfitTrend, true) {
		return "", fmt.Errorf("%w: trigger %d, stop-loss %d, take-profit %d", errInvalidPriceType, req.Trend, req.LossTrend, req.ProfitTrend)
	}
	var resp FuturesID
	return resp, e.SendFuturesHTTPRequest(ctx, futuresPlacePlanOrderEPL, http.MethodPost, "private/planorder/place/v2", nil, req, &resp, true)
}

// ModifyFuturesPlanOrder changes a plan order's trigger and execution
func (e *Exchange) ModifyFuturesPlanOrder(ctx context.Context, req *FuturesModifyPlanOrderRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if req.OrderID == 0 {
		return order.ErrOrderIDNotSet
	}
	if req.TriggerPrice <= 0 {
		return fmt.Errorf("%w: trigger price %v", limits.ErrPriceBelowMin, req.TriggerPrice)
	}
	if req.OrderType < FuturesOrderTypeLimit || req.OrderType > FuturesOrderTypeMarket {
		return fmt.Errorf("%w: %d", order.ErrUnsupportedOrderType, req.OrderType)
	}
	if isFuturesLimitOrderType(req.OrderType) && req.Price <= 0 {
		return fmt.Errorf("%w: price %v", limits.ErrPriceBelowMin, req.Price)
	}
	if req.TriggerType != FuturesTriggerGreaterOrEqual && req.TriggerType != FuturesTriggerLessOrEqual {
		return fmt.Errorf("%w: %d", errInvalidTriggerType, req.TriggerType)
	}
	if !validFuturesPriceType(req.Trend, false) {
		return fmt.Errorf("%w: %d", errInvalidPriceType, req.Trend)
	}
	return e.SendFuturesHTTPRequest(ctx, futuresModifyPlanOrderEPL, http.MethodPost, "private/planorder/change_price", nil, req, nil, true)
}

// CancelFuturesPlanOrders cancels up to 50 plan orders
func (e *Exchange) CancelFuturesPlanOrders(ctx context.Context, orders []FuturesCancelPlanOrderRequest) error {
	if len(orders) == 0 {
		return order.ErrOrderIDNotSet
	}
	if len(orders) > futuresBatchLimit {
		return fmt.Errorf("%w: %d orders, at most %d", errTooManyOrders, len(orders), futuresBatchLimit)
	}
	for i := range orders {
		if orders[i].Symbol == "" {
			return currency.ErrSymbolStringEmpty
		}
		if orders[i].OrderID == "" {
			return order.ErrOrderIDNotSet
		}
	}
	return e.SendFuturesHTTPRequest(ctx, futuresCancelPlanOrdersEPL, http.MethodPost, "private/planorder/cancel", nil, orders, nil, true)
}

// CancelAllFuturesPlanOrders cancels every plan order of a contract, or of every contract when symbol is unset
func (e *Exchange) CancelAllFuturesPlanOrders(ctx context.Context, symbol currency.Pair) error {
	body := &FuturesCancelAllPlanOrdersRequest{}
	if !symbol.IsEmpty() {
		body.Symbol = symbol.String()
	}
	return e.SendFuturesHTTPRequest(ctx, futuresCancelAllPlanOrdersEPL, http.MethodPost, "private/planorder/cancel_all", nil, body, nil, true)
}

// PlaceFuturesTakeProfitStopLossOrder places a take-profit and stop-loss order on a position and returns its ID.
// When an open order of the same prices exists the venue returns its ID and updates its volume
func (e *Exchange) PlaceFuturesTakeProfitStopLossOrder(ctx context.Context, req *FuturesTakeProfitStopLossRequest) (FuturesID, error) {
	if err := common.NilGuard(req); err != nil {
		return "", err
	}
	if req.PositionID == 0 {
		return "", errPositionIDRequired
	}
	if req.Volume <= 0 {
		return "", fmt.Errorf("%w: volume %v", limits.ErrAmountBelowMin, req.Volume)
	}
	if req.StopLossPrice <= 0 && req.TakeProfitPrice <= 0 {
		return "", errInvalidTakeProfitStopLoss
	}
	if !validFuturesPriceType(req.LossTrend, false) || !validFuturesPriceType(req.ProfitTrend, false) {
		return "", fmt.Errorf("%w: stop-loss %d, take-profit %d", errInvalidPriceType, req.LossTrend, req.ProfitTrend)
	}
	var resp FuturesID
	return resp, e.SendFuturesHTTPRequest(ctx, futuresPlaceTakeProfitStopLossEPL, http.MethodPost, "private/stoporder/place", nil, req, &resp, true)
}

// CancelFuturesTakeProfitStopLossOrders cancels up to 50 take-profit and stop-loss orders
func (e *Exchange) CancelFuturesTakeProfitStopLossOrders(ctx context.Context, stopPlanOrderIDs []uint64) error {
	if len(stopPlanOrderIDs) == 0 || slices.Contains(stopPlanOrderIDs, 0) {
		return errStopPlanOrderIDRequired
	}
	if len(stopPlanOrderIDs) > futuresBatchLimit {
		return fmt.Errorf("%w: %d orders, at most %d", errTooManyOrders, len(stopPlanOrderIDs), futuresBatchLimit)
	}
	body := make([]FuturesCancelTakeProfitStopLossRequest, len(stopPlanOrderIDs))
	for i, id := range stopPlanOrderIDs {
		body[i].StopPlanOrderID = id
	}
	return e.SendFuturesHTTPRequest(ctx, futuresCancelTakeProfitStopLossEPL, http.MethodPost, "private/stoporder/cancel", nil, body, nil, true)
}

// CancelAllFuturesTakeProfitStopLossOrders cancels the take-profit and stop-loss orders of a position when
// positionID is set, otherwise of a contract when symbol is, and otherwise of every contract
func (e *Exchange) CancelAllFuturesTakeProfitStopLossOrders(ctx context.Context, positionID uint64, symbol currency.Pair) error {
	body := &FuturesCancelAllTakeProfitStopLossRequest{PositionID: positionID}
	if !symbol.IsEmpty() {
		body.Symbol = symbol.String()
	}
	return e.SendFuturesHTTPRequest(ctx, futuresCancelAllTakeProfitStopLossEPL, http.MethodPost, "private/stoporder/cancel_all", nil, body, nil, true)
}

// ModifyFuturesLimitOrderTakeProfitStopLoss changes a limit order's take-profit and stop-loss prices; leaving both
// unset cancels them
func (e *Exchange) ModifyFuturesLimitOrderTakeProfitStopLoss(ctx context.Context, req *FuturesModifyTakeProfitStopLossRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.OrderID == 0 {
		return order.ErrOrderIDNotSet
	}
	if !validFuturesPriceType(req.LossTrend, true) || !validFuturesPriceType(req.ProfitTrend, true) {
		return fmt.Errorf("%w: stop-loss %d, take-profit %d", errInvalidPriceType, req.LossTrend, req.ProfitTrend)
	}
	body := *req
	body.StopPlanOrderID = 0
	return e.SendFuturesHTTPRequest(ctx, futuresModifyLimitOrderTakeProfitStopLossEPL, http.MethodPost, "private/stoporder/change_price", nil, &body, nil, true)
}

// ModifyFuturesTakeProfitStopLossOrder changes a take-profit and stop-loss order's prices; at least one must be set
func (e *Exchange) ModifyFuturesTakeProfitStopLossOrder(ctx context.Context, req *FuturesModifyTakeProfitStopLossRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.StopPlanOrderID == 0 {
		return errStopPlanOrderIDRequired
	}
	if req.StopLossPrice <= 0 && req.TakeProfitPrice <= 0 {
		return errInvalidTakeProfitStopLoss
	}
	if !validFuturesPriceType(req.LossTrend, true) || !validFuturesPriceType(req.ProfitTrend, true) {
		return fmt.Errorf("%w: stop-loss %d, take-profit %d", errInvalidPriceType, req.LossTrend, req.ProfitTrend)
	}
	body := *req
	body.OrderID = 0
	return e.SendFuturesHTTPRequest(ctx, futuresModifyTakeProfitStopLossEPL, http.MethodPost, "private/stoporder/change_plan_price", nil, &body, nil, true)
}

// ModifyFuturesPlanOrderTakeProfitStopLoss changes a plan order's take-profit and stop-loss prices; a zero price
// cancels that side
func (e *Exchange) ModifyFuturesPlanOrderTakeProfitStopLoss(ctx context.Context, req *FuturesModifyPlanOrderTakeProfitStopLossRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if req.OrderID == 0 {
		return order.ErrOrderIDNotSet
	}
	if !validFuturesPriceType(req.LossTrend, true) || !validFuturesPriceType(req.ProfitTrend, true) {
		return fmt.Errorf("%w: stop-loss %d, take-profit %d", errInvalidPriceType, req.LossTrend, req.ProfitTrend)
	}
	return e.SendFuturesHTTPRequest(ctx, futuresModifyPlanOrderTakeProfitStopLossEPL, http.MethodPost, "private/planorder/change_stop_order", nil, req, nil, true)
}

// GetFuturesTakeProfitStopLossOrders returns a page of the account's take-profit and stop-loss orders; the page size
// is at most 100
func (e *Exchange) GetFuturesTakeProfitStopLossOrders(ctx context.Context, req *FuturesTakeProfitStopLossOrdersRequest) ([]*FuturesTakeProfitStopLossOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PositionType != 0 && !validPositionType(req.PositionType) {
		return nil, fmt.Errorf("%w: %d", errInvalidPositionType, req.PositionType)
	}
	params := url.Values{}
	if !req.Symbol.IsEmpty() {
		params.Set("symbol", req.Symbol.String())
	}
	if req.IsFinished != nil {
		params.Set("is_finished", "0")
		if *req.IsFinished {
			params.Set("is_finished", "1")
		}
	}
	if req.State != 0 {
		params.Set("state", strconv.FormatUint(uint64(req.State), 10))
	}
	if req.PositionType != 0 {
		params.Set("type", strconv.FormatUint(uint64(req.PositionType), 10))
	}
	if err := setFuturesTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := setFuturesPage(params, &req.FuturesPageRequest, futuresDefaultPageSizeLimit); err != nil {
		return nil, err
	}
	var resp pagedList[*FuturesTakeProfitStopLossOrderResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresTakeProfitStopLossOrdersEPL, http.MethodGet, "private/stoporder/list/orders", params, nil, &resp, true)
}

// GetFuturesOpenTakeProfitStopLossOrders returns the account's open take-profit and stop-loss orders, of a contract
// when symbol is set
func (e *Exchange) GetFuturesOpenTakeProfitStopLossOrders(ctx context.Context, symbol currency.Pair) ([]*FuturesTakeProfitStopLossOrderResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*FuturesTakeProfitStopLossOrderResponse
	return resp, e.SendFuturesHTTPRequest(ctx, futuresOpenTakeProfitStopLossOrdersEPL, http.MethodGet, "private/stoporder/open_orders", params, nil, &resp, true)
}

// validateTrailing checks a trailing order's price type and callback
func validateTrailing(trend, backType uint8, backValue float64) error {
	if !validFuturesPriceType(trend, false) {
		return fmt.Errorf("%w: %d", errInvalidPriceType, trend)
	}
	if backType != FuturesCallbackPercentage && backType != FuturesCallbackAbsolute {
		return fmt.Errorf("%w: %d", errInvalidCallbackType, backType)
	}
	if backValue <= 0 {
		return fmt.Errorf("%w: callback value %v", limits.ErrPriceBelowMin, backValue)
	}
	return nil
}

// PlaceFuturesTrailingOrder places a trailing order and returns its ID
func (e *Exchange) PlaceFuturesTrailingOrder(ctx context.Context, req *FuturesTrailingOrderRequest) (FuturesID, error) {
	if err := common.NilGuard(req); err != nil {
		return "", err
	}
	if req.Symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	if req.Leverage == 0 {
		return "", errInvalidLeverage
	}
	if !validFuturesSide(req.Side) {
		return "", fmt.Errorf("%w: %d", errInvalidOrderSide, req.Side)
	}
	if req.Volume <= 0 {
		return "", fmt.Errorf("%w: volume %v", limits.ErrAmountBelowMin, req.Volume)
	}
	if !validFuturesOpenType(req.OpenType) {
		return "", fmt.Errorf("%w: %d", errInvalidOpenType, req.OpenType)
	}
	if req.PositionMode != FuturesPositionModeHedge && req.PositionMode != FuturesPositionModeOneWay {
		return "", fmt.Errorf("%w: %d", errInvalidPositionMode, req.PositionMode)
	}
	if err := validateTrailing(req.Trend, req.BackType, req.BackValue); err != nil {
		return "", err
	}
	var resp FuturesID
	return resp, e.SendFuturesHTTPRequest(ctx, futuresPlaceTrailingOrderEPL, http.MethodPost, "private/trackorder/place", nil, req, &resp, true)
}

// CancelFuturesTrailingOrder cancels a trailing order
func (e *Exchange) CancelFuturesTrailingOrder(ctx context.Context, symbol currency.Pair, trackOrderID uint64) error {
	if trackOrderID == 0 {
		return errTrailingOrderIDRequired
	}
	body := &FuturesCancelTrailingOrderRequest{TrackOrderID: trackOrderID}
	if !symbol.IsEmpty() {
		body.Symbol = symbol.String()
	}
	return e.SendFuturesHTTPRequest(ctx, futuresCancelTrailingOrderEPL, http.MethodPost, "private/trackorder/cancel", nil, body, nil, true)
}

// ModifyFuturesTrailingOrder changes a trailing order
func (e *Exchange) ModifyFuturesTrailingOrder(ctx context.Context, req *FuturesModifyTrailingOrderRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if req.TrackOrderID == 0 {
		return errTrailingOrderIDRequired
	}
	if req.Volume <= 0 {
		return fmt.Errorf("%w: volume %v", limits.ErrAmountBelowMin, req.Volume)
	}
	if err := validateTrailing(req.Trend, req.BackType, req.BackValue); err != nil {
		return err
	}
	return e.SendFuturesHTTPRequest(ctx, futuresModifyTrailingOrderEPL, http.MethodPost, "private/trackorder/change_order", nil, req, nil, true)
}

// GetFuturesTrailingOrders returns a page of the account's trailing orders in the states asked for
func (e *Exchange) GetFuturesTrailingOrders(ctx context.Context, req *FuturesTrailingOrdersRequest) ([]*FuturesTrailingOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.States) == 0 {
		return nil, errOrderStatesRequired
	}
	if req.Side != 0 && !validFuturesSide(req.Side) {
		return nil, fmt.Errorf("%w: %d", errInvalidOrderSide, req.Side)
	}
	params := url.Values{}
	if !req.Symbol.IsEmpty() {
		params.Set("symbol", req.Symbol.String())
	}
	params.Set("states", joinFuturesStates(req.States))
	if req.Side != 0 {
		params.Set("side", strconv.FormatUint(uint64(req.Side), 10))
	}
	if err := setFuturesTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(req.PageIndex, 10))
	}
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp pagedList[*FuturesTrailingOrderResponse]
	return resp, e.SendFuturesHTTPRequest(ctx, futuresTrailingOrdersEPL, http.MethodGet, "private/trackorder/list/orders", params, nil, &resp, true)
}
