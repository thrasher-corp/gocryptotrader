package bitstamp

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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Order subtypes accepted by the limit and market order endpoints
var (
	limitOrderSubtypes  = []string{OrderSubtypeLimit, OrderSubtypeStopLossLimit, OrderSubtypeTakeProfitLimit, OrderSubtypeTrailingStopLossLimit, OrderSubtypeTrailingTakeProfitLimit}
	marketOrderSubtypes = []string{OrderSubtypeMarket, OrderSubtypeStopLoss, OrderSubtypeTakeProfit, OrderSubtypeTrailingStopLoss, OrderSubtypeTrailingTakeProfit}
)

// GetOrderStatus returns the status of an order
// Closed orders are only available for 30 days
func (e *Exchange) GetOrderStatus(ctx context.Context, req *OrderStatusRequest) (*OrderStatusResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := validateClientOrderID(req.ClientOrderID); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != 0 {
		params.Set("id", strconv.FormatUint(req.OrderID, 10))
	}
	if req.ClientOrderID != "" {
		params.Set("client_order_id", req.ClientOrderID)
	}
	if req.OmitTransactions {
		params.Set("omit_transactions", trueValue)
	}
	var resp *OrderStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/order_status/", nil, params, &resp)
}

// GetAccountOrderData returns up to 1000 of the account's order events for a market, oldest first, to recover events
// missed on the private orders websocket channel; events are retained for 30 days
func (e *Exchange) GetAccountOrderData(ctx context.Context, req *OrderEventsRequest) ([]OrderEventResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderSource == "" {
		return nil, errOrderSourceRequired
	}
	params, err := orderEventParams(req)
	if err != nil {
		return nil, err
	}
	params.Set("order_source", req.OrderSource)
	var resp []OrderEventResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/account_order_data/", nil, params, &resp)
}

// GetOrderData returns up to 1000 public order events for a market, oldest first, to recover events missed on the
// live orders websocket channel; events are retained for 30 days
func (e *Exchange) GetOrderData(ctx context.Context, req *OrderEventsRequest) ([]OrderEventResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := orderEventParams(req)
	if err != nil {
		return nil, err
	}
	var resp []OrderEventResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/order_data/", nil, params, &resp)
}

func orderEventParams(req *OrderEventsRequest) (url.Values, error) {
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("market", formatMarketName(req.Pair))
	if req.SinceID != "" {
		params.Set("since_id", req.SinceID)
	}
	if req.UntilID != "" {
		params.Set("until_id", req.UntilID)
	}
	return params, nil
}

// GetOpenOrders returns the account's open orders
// pair is optional and restricts the results to a single market
func (e *Exchange) GetOpenOrders(ctx context.Context, pair currency.Pair) ([]OpenOrderResponse, error) {
	path := "/v2/open_orders/"
	if !pair.IsEmpty() {
		path += formatMarketSymbol(pair) + "/"
	}
	var resp []OpenOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nil, &resp)
}

// PlaceLimitOrder places a buy or sell limit order
func (e *Exchange) PlaceLimitOrder(ctx context.Context, req *LimitOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if req.Amount < 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.LimitPrice < 0 {
		return nil, fmt.Errorf("%w: limit price %v", errInvalidPrice, req.LimitPrice)
	}
	var flags int
	for _, set := range []bool{req.DailyOrder, req.ImmediateOrCancel, req.FillOrKill, req.MakerOrCancel, req.GoodTillDate} {
		if set {
			flags++
		}
	}
	if flags > 1 {
		return nil, errOrderFlagsConflict
	}
	if flags > 0 && req.LimitPrice != 0 {
		return nil, errLimitPriceConflict
	}
	if req.GoodTillDate && req.ExpireTime.IsZero() {
		return nil, fmt.Errorf("%w: expire time is required for good till date orders", common.ErrDateUnset)
	}
	orderParams := &derivativesOrderParams{
		ClientOrderID:   req.ClientOrderID,
		Subtype:         req.Subtype,
		MarginMode:      req.MarginMode,
		Leverage:        req.Leverage,
		StopPrice:       req.StopPrice,
		Trigger:         req.Trigger,
		ActivationPrice: req.ActivationPrice,
		TrailingDelta:   req.TrailingDelta,
		ReduceOnly:      req.ReduceOnly,
	}
	if err := validateDerivativesOrderParams(req.Pair, orderParams, limitOrderSubtypes); err != nil {
		return nil, err
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	setFloat(params, "price", req.Price)
	setFloat(params, "limit_price", req.LimitPrice)
	setTrue(params, "daily_order", req.DailyOrder)
	setTrue(params, "ioc_order", req.ImmediateOrCancel)
	setTrue(params, "fok_order", req.FillOrKill)
	setTrue(params, "moc_order", req.MakerOrCancel)
	setTrue(params, "gtd_order", req.GoodTillDate)
	if req.GoodTillDate {
		params.Set("expire_time", strconv.FormatInt(req.ExpireTime.UnixMilli(), 10))
	}
	setDerivativesOrderParams(params, orderParams)
	var resp *OrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+side+"/"+formatMarketSymbol(req.Pair)+"/", nil, params, &resp)
}

// PlaceInstantOrder places a buy or sell instant order
// Buy amounts are in the counter currency and sell amounts are in the base currency unless AmountInCounter is set
func (e *Exchange) PlaceInstantOrder(ctx context.Context, req *InstantOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if err := validateClientOrderID(req.ClientOrderID); err != nil {
		return nil, err
	}
	if req.Leverage < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidLeverage, req.Leverage)
	}
	// Only sell orders whose amount is in the base currency accept the reduce only flag
	if req.ReduceOnly && (req.Side.IsLong() || req.AmountInCounter) {
		return nil, errReduceOnlyUnsupported
	}
	if isPerpetualPair(req.Pair) && req.Side.IsLong() {
		if err := validateMarginParams(req.MarginMode, req.Leverage); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	if req.AmountInCounter && req.Side.IsShort() {
		params.Set("amount_in_counter", trueValue)
	}
	setDerivativesOrderParams(params, &derivativesOrderParams{
		ClientOrderID: req.ClientOrderID,
		MarginMode:    req.MarginMode,
		Leverage:      req.Leverage,
		ReduceOnly:    req.ReduceOnly,
	})
	var resp *OrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+side+"/instant/"+formatMarketSymbol(req.Pair)+"/", nil, params, &resp)
}

// PlaceMarketOrder places a buy or sell market order
func (e *Exchange) PlaceMarketOrder(ctx context.Context, req *MarketOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Amount < 0 {
		return nil, order.ErrAmountIsInvalid
	}
	orderParams := &derivativesOrderParams{
		ClientOrderID:   req.ClientOrderID,
		Subtype:         req.Subtype,
		MarginMode:      req.MarginMode,
		Leverage:        req.Leverage,
		StopPrice:       req.StopPrice,
		Trigger:         req.Trigger,
		ActivationPrice: req.ActivationPrice,
		TrailingDelta:   req.TrailingDelta,
		ReduceOnly:      req.ReduceOnly,
	}
	if err := validateDerivativesOrderParams(req.Pair, orderParams, marketOrderSubtypes); err != nil {
		return nil, err
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	setDerivativesOrderParams(params, orderParams)
	var resp *OrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+side+"/market/"+formatMarketSymbol(req.Pair)+"/", nil, params, &resp)
}

// validateDerivativesOrderParams checks the parameters shared by limit and market orders against the documented limits
// Margin mode and leverage are required for derivatives markets
func validateDerivativesOrderParams(pair currency.Pair, p *derivativesOrderParams, subtypes []string) error {
	if err := validateClientOrderID(p.ClientOrderID); err != nil {
		return err
	}
	if p.Subtype != "" && !slices.Contains(subtypes, p.Subtype) {
		return fmt.Errorf("%w: %q", errInvalidOrderSubtype, p.Subtype)
	}
	if p.Leverage < 0 {
		return fmt.Errorf("%w: %v", errInvalidLeverage, p.Leverage)
	}
	if isPerpetualPair(pair) {
		if err := validateMarginParams(p.MarginMode, p.Leverage); err != nil {
			return err
		}
	}
	if p.StopPrice < 0 {
		return fmt.Errorf("%w: stop price %v", errInvalidPrice, p.StopPrice)
	}
	if p.ActivationPrice < 0 {
		return fmt.Errorf("%w: activation price %v", errInvalidPrice, p.ActivationPrice)
	}
	if p.TrailingDelta > maxTrailingDelta {
		return fmt.Errorf("%w: %d", errInvalidTrailingDelta, p.TrailingDelta)
	}
	if p.ActivationPrice > 0 && p.TrailingDelta == 0 {
		return errTrailingDeltaRequired
	}
	return nil
}

// validateMarginParams checks the margin mode and leverage required by derivatives orders
func validateMarginParams(marginMode string, leverage float64) error {
	if marginMode == "" {
		return errMarginModeRequired
	}
	if leverage <= 0 {
		return errLeverageRequired
	}
	return nil
}

// validateClientOrderID checks a client order ID against the documented maximum length
func validateClientOrderID(clientOrderID string) error {
	if len(clientOrderID) > maxClientOrderIDLength {
		return fmt.Errorf("%w: %d characters", errClientOrderIDTooLong, len(clientOrderID))
	}
	return nil
}

func setDerivativesOrderParams(params url.Values, p *derivativesOrderParams) {
	if p.ClientOrderID != "" {
		params.Set("client_order_id", p.ClientOrderID)
	}
	if p.Subtype != "" {
		params.Set("subtype", p.Subtype)
	}
	if p.MarginMode != "" {
		params.Set("margin_mode", p.MarginMode)
	}
	setFloat(params, "leverage", p.Leverage)
	setFloat(params, "stop_price", p.StopPrice)
	if p.Trigger != "" {
		params.Set("trigger", p.Trigger)
	}
	setFloat(params, "activation_price", p.ActivationPrice)
	if p.TrailingDelta != 0 {
		params.Set("trailing_delta", strconv.FormatUint(p.TrailingDelta, 10))
	}
	setTrue(params, "reduce_only", p.ReduceOnly)
}

// CancelExistingOrder cancels an order
func (e *Exchange) CancelExistingOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := validateClientOrderID(req.ClientOrderID); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != 0 {
		params.Set("id", strconv.FormatUint(req.OrderID, 10))
	}
	if req.ClientOrderID != "" {
		params.Set("client_order_id", req.ClientOrderID)
	}
	var resp *CancelOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/cancel_order/", nil, params, &resp)
}

// CancelAllExistingOrders cancels all open orders
// pair is optional and restricts the cancellation to a single market
func (e *Exchange) CancelAllExistingOrders(ctx context.Context, pair currency.Pair) (*CancelAllOrdersResponse, error) {
	path := "/v2/cancel_all_orders/"
	if !pair.IsEmpty() {
		path += formatMarketSymbol(pair) + "/"
	}
	var resp *CancelAllOrdersResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nil, &resp)
}

// ReplaceOrder atomically cancels an order and places a new order with the supplied amount and price
func (e *Exchange) ReplaceOrder(ctx context.Context, req *ReplaceOrderRequest) (*ReplaceOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	for _, id := range []string{req.OriginalClientOrderID, req.ClientOrderID} {
		if err := validateClientOrderID(id); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.OrderID != 0 {
		params.Set("id", strconv.FormatUint(req.OrderID, 10))
	}
	if req.OriginalClientOrderID != "" {
		params.Set("orig_client_order_id", req.OriginalClientOrderID)
	}
	if req.ClientOrderID != "" {
		params.Set("client_order_id", req.ClientOrderID)
	}
	setFloat(params, "amount", req.Amount)
	setFloat(params, "price", req.Price)
	var resp *ReplaceOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/replace_order/", nil, params, &resp)
}

// GetTradingMarkets returns the markets which can be traded by the account
func (e *Exchange) GetTradingMarkets(ctx context.Context) ([]TradingMarketResponse, error) {
	var resp []TradingMarketResponse
	// The API documents this endpoint as a GET, however it only accepts POST requests
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/my_markets/", nil, nil, &resp)
}

// GetMaxOrderAmount returns the maximum amount and value of an order on a derivatives market
func (e *Exchange) GetMaxOrderAmount(ctx context.Context, req *MaxOrderAmountRequest) (*MaxOrderAmountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.MarginMode == "" {
		return nil, errMarginModeRequired
	}
	if err := validateLeverage(req.Leverage); err != nil {
		return nil, err
	}
	if req.OrderType == "" {
		return nil, errOrderTypeRequired
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Price < 0 || req.StopPrice < 0 || req.ActivationPrice < 0 {
		return nil, errInvalidPrice
	}
	if req.TrailingDelta > maxTrailingDelta {
		return nil, fmt.Errorf("%w: %d", errInvalidTrailingDelta, req.TrailingDelta)
	}
	if err := validateCollateral(req.AdditionalCollateral); err != nil {
		return nil, err
	}
	body := &maxOrderAmountBody{
		Market:               formatMarketName(req.Pair),
		MarginMode:           req.MarginMode,
		Leverage:             req.Leverage,
		OrderType:            req.OrderType,
		Side:                 strings.ToUpper(side),
		Price:                req.Price,
		StopPrice:            req.StopPrice,
		ActivationPrice:      req.ActivationPrice,
		TrailingDelta:        req.TrailingDelta,
		AdditionalCollateral: formatCollateral(req.AdditionalCollateral),
	}
	var resp *MaxOrderAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/get_max_order_amount/", nil, body, &resp)
}

// validateLeverage checks a derivatives leverage against the documented range
func validateLeverage(leverage float64) error {
	if leverage == 0 {
		return errLeverageRequired
	}
	if leverage < minLeverage || leverage > maxLeverage {
		return fmt.Errorf("%w: %v", errInvalidLeverage, leverage)
	}
	return nil
}

// validateCollateral checks that collateral amounts are not negative
func validateCollateral(collateral map[currency.Code]float64) error {
	for c, amount := range collateral {
		if amount < 0 {
			return fmt.Errorf("%w: %s %v", errInvalidCollateral, c, amount)
		}
	}
	return nil
}
