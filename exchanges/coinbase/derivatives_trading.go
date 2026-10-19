package coinbase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	// derivativesMaximumTradingCount is the most items a trading history method returns in a page
	derivativesMaximumTradingCount = 1000
	// derivativesBuyMethod and derivativesSellMethod place orders, taking the same params
	derivativesBuyMethod  = "private/buy"
	derivativesSellMethod = "private/sell"
	// derivativesOrderNotFound and derivativesNotOwnerOfOrder are the error messages the gateway answers a request
	// naming an order by its ID with when it does not find the order in the scope the request names
	derivativesOrderNotFound   = "order_not_found"
	derivativesNotOwnerOfOrder = "not_owner_of_order"
	// derivativesLiquidityMaker is a trade's liquidity when the account's order made it
	derivativesLiquidityMaker = "M"
)

var (
	errDerivativesOrderIDRequired                = errors.New("order ID required")
	errDerivativesLabelRequired                  = errors.New("label required")
	errDerivativesCurrencyPairRequired           = errors.New("currency pair required")
	errDerivativesInvalidTradingKind             = errors.New("invalid instrument kind")
	errDerivativesInvalidTradingCount            = errors.New("invalid count")
	errDerivativesInvalidOrderType               = errors.New("invalid order type")
	errDerivativesInvalidTimeInForce             = errors.New("invalid time in force")
	errDerivativesInvalidTrigger                 = errors.New("invalid trigger")
	errDerivativesTriggerRequired                = errors.New("trigger required for a trigger order")
	errDerivativesTriggerPriceRequired           = errors.New("trigger price required for a stop or take order")
	errDerivativesInvalidAdvancedType            = errors.New("invalid advanced order type")
	errDerivativesInvalidOrderDirection          = errors.New("invalid order direction")
	errDerivativesInvalidLinkedOrderType         = errors.New("invalid linked order type")
	errDerivativesInvalidTriggerFillCondition    = errors.New("invalid trigger fill condition")
	errDerivativesLinkedOrderTypeRequired        = errors.New("linked order type required for secondary orders and a trigger fill condition")
	errDerivativesInvalidSecondaryOrderCount     = errors.New("invalid number of secondary orders")
	errDerivativesPostOnlyTimeInForce            = errors.New("post only needs a good til cancelled order")
	errDerivativesRejectPostOnlyWithoutPostOnly  = errors.New("reject post only needs post only")
	errDerivativesIsolatedOrderUnsupported       = errors.New("isolated margin is not supported for trigger, advanced or linked orders")
	errDerivativesAllocatedMarginWithoutIsolated = errors.New("allocated margin needs isolated margin")
	errDerivativesInvalidCancelOrderType         = errors.New("invalid cancel order type")
	errDerivativesInvalidOpenOrderType           = errors.New("invalid open order type")
	errDerivativesInvalidSettlementHistoryType   = errors.New("invalid settlement type")
	errDerivativesInvalidUserTradesSorting       = errors.New("invalid sorting")
	errDerivativesInvalidLeverage                = errors.New("leverage must be at least 1")
	errDerivativesInvalidTakerFeeRate            = errors.New("taker fee rate must not be negative")
	errDerivativesOrderHistoryOrdersMissing      = errors.New("order history page has no orders field")
	errDerivativesInvalidCancelAllResult         = errors.New("invalid cancel all result")
)

// Buy places a buy order and returns it with the trades it made at once; a spot order routed to Coinbase Exchange
// fills asynchronously, so its trades arrive on the user.trades channel instead
func (g GlobalDerivativesAPI) Buy(ctx context.Context, req *DerivativesOrderRequest) (*DerivativesOrderResponse, error) {
	return g.placeOrder(ctx, false, derivativesBuyMethod, req)
}

// placeOrder places a buy or sell order, which take the same params
func (g GlobalDerivativesAPI) placeOrder(ctx context.Context, overWebsocket bool, method string, req *DerivativesOrderRequest) (*DerivativesOrderResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	params := &derivativesOrderParams{DerivativesOrderRequest: req}
	if !req.ValidUntil.IsZero() {
		params.ValidUntil = req.ValidUntil.UnixMilli()
	}
	var resp *DerivativesOrderResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, method, params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOrderByID cancels an open order or trigger order and returns it; an isolated margin order is only found with
// Isolated set
func (g GlobalDerivativesAPI) CancelOrderByID(ctx context.Context, req *DerivativesCancelOrderByIDRequest) (*DerivativesOrder, error) {
	return g.cancelOrderByID(ctx, false, req)
}

func (g GlobalDerivativesAPI) cancelOrderByID(ctx context.Context, overWebsocket bool, req *DerivativesCancelOrderByIDRequest) (*DerivativesOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, errDerivativesOrderIDRequired
	}
	var resp *DerivativesOrder
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/cancel", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAll cancels every open order and trigger order of the account, and of its isolated margin subaccounts with
// IncludeIsolated, and returns the number cancelled, or with Detailed the execution reports
func (g GlobalDerivativesAPI) CancelAll(ctx context.Context, req *DerivativesCancelAllRequest) (*DerivativesCancelAllResponse, error) {
	return g.cancelAll(ctx, false, req)
}

func (g GlobalDerivativesAPI) cancelAll(ctx context.Context, overWebsocket bool, req *DerivativesCancelAllRequest) (*DerivativesCancelAllResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp *DerivativesCancelAllResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/cancel_all", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByCurrency cancels the open orders of a currency's instruments, by instrument kind and order type, and
// returns the number cancelled, or with Detailed the execution reports
func (g GlobalDerivativesAPI) CancelAllByCurrency(ctx context.Context, req *DerivativesCancelAllByCurrencyRequest) (*DerivativesCancelAllResponse, error) {
	return g.cancelAllByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) cancelAllByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesCancelAllByCurrencyRequest) (*DerivativesCancelAllResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesTradingKind(req.Kind, true); err != nil {
		return nil, err
	}
	if err := validateDerivativesCancelOrderType(req.Type); err != nil {
		return nil, err
	}
	params := &derivativesCancelAllByCurrencyParams{DerivativesCancelAllByCurrencyRequest: req, Currency: req.Currency.Upper().String()}
	var resp *DerivativesCancelAllResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/cancel_all_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByCurrencyPair cancels the open orders of a currency pair's instruments, by instrument kind and order type,
// and returns the number cancelled, or with Detailed the execution reports
func (g GlobalDerivativesAPI) CancelAllByCurrencyPair(ctx context.Context, req *DerivativesCancelAllByCurrencyPairRequest) (*DerivativesCancelAllResponse, error) {
	return g.cancelAllByCurrencyPair(ctx, false, req)
}

func (g GlobalDerivativesAPI) cancelAllByCurrencyPair(ctx context.Context, overWebsocket bool, req *DerivativesCancelAllByCurrencyPairRequest) (*DerivativesCancelAllResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.CurrencyPair == "" {
		return nil, errDerivativesCurrencyPairRequired
	}
	if err := validateDerivativesTradingKind(req.Kind, true); err != nil {
		return nil, err
	}
	if err := validateDerivativesCancelOrderType(req.Type); err != nil {
		return nil, err
	}
	var resp *DerivativesCancelAllResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/cancel_all_by_currency_pair", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByInstrument cancels the open orders of an instrument, by order type, and returns the number cancelled, or
// with Detailed the execution reports
func (g GlobalDerivativesAPI) CancelAllByInstrument(ctx context.Context, req *DerivativesCancelAllByInstrumentRequest) (*DerivativesCancelAllResponse, error) {
	return g.cancelAllByInstrument(ctx, false, req)
}

func (g GlobalDerivativesAPI) cancelAllByInstrument(ctx context.Context, overWebsocket bool, req *DerivativesCancelAllByInstrumentRequest) (*DerivativesCancelAllResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesCancelOrderType(req.Type); err != nil {
		return nil, err
	}
	var resp *DerivativesCancelAllResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/cancel_all_by_instrument", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByKindOrType cancels the open orders of one or more currencies, by instrument kind and order type, and
// returns the number cancelled, or with Detailed the execution reports
func (g GlobalDerivativesAPI) CancelAllByKindOrType(ctx context.Context, req *DerivativesCancelAllByKindOrTypeRequest) (*DerivativesCancelAllResponse, error) {
	return g.cancelAllByKindOrType(ctx, false, req)
}

func (g GlobalDerivativesAPI) cancelAllByKindOrType(ctx context.Context, overWebsocket bool, req *DerivativesCancelAllByKindOrTypeRequest) (*DerivativesCancelAllResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	currencies, err := derivativesKindOrTypeCurrencies(req.Currencies)
	if err != nil {
		return nil, err
	}
	if err := validateDerivativesTradingKind(req.Kind, true); err != nil {
		return nil, err
	}
	if err := validateDerivativesCancelOrderType(req.Type); err != nil {
		return nil, err
	}
	params := &derivativesCancelAllByKindOrTypeParams{DerivativesCancelAllByKindOrTypeRequest: req, Currency: currencies}
	var resp *DerivativesCancelAllResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/cancel_all_by_kind_or_type", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelByLabel cancels the open orders and trigger orders with a label, in every currency or one, and returns the
// number cancelled; labels are not unique, so every order with the label is cancelled
func (g GlobalDerivativesAPI) CancelByLabel(ctx context.Context, req *DerivativesCancelByLabelRequest) (*DerivativesCancelAllResponse, error) {
	return g.cancelByLabel(ctx, false, req)
}

func (g GlobalDerivativesAPI) cancelByLabel(ctx context.Context, overWebsocket bool, req *DerivativesCancelByLabelRequest) (*DerivativesCancelAllResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Label == "" {
		return nil, errDerivativesLabelRequired
	}
	params := &derivativesCancelByLabelParams{DerivativesCancelByLabelRequest: req}
	if !req.Currency.IsEmpty() {
		params.Currency = req.Currency.Upper().String()
	}
	var resp *DerivativesCancelAllResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/cancel_by_label", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ClosePosition closes a position with a reduce only limit or market order and returns the order with the trades it
// made at once; with Isolated it closes the isolated margin subaccount's position in full
func (g GlobalDerivativesAPI) ClosePosition(ctx context.Context, req *DerivativesClosePositionRequest) (*DerivativesOrderResponse, error) {
	return g.closePosition(ctx, false, req)
}

func (g GlobalDerivativesAPI) closePosition(ctx context.Context, overWebsocket bool, req *DerivativesClosePositionRequest) (*DerivativesOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if req.Type != DerivativesOrderTypeLimit && req.Type != DerivativesOrderTypeMarket {
		return nil, fmt.Errorf("%w: %q, want limit or market", errDerivativesInvalidOrderType, req.Type)
	}
	var resp *DerivativesOrderResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/close_position", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// EditOrder edits an open order's size, price and flags and returns it with the trades it made at once; an isolated
// margin order is only found with Isolated set
func (g GlobalDerivativesAPI) EditOrder(ctx context.Context, req *DerivativesEditOrderRequest) (*DerivativesOrderResponse, error) {
	return g.editOrder(ctx, false, req)
}

func (g GlobalDerivativesAPI) editOrder(ctx context.Context, overWebsocket bool, req *DerivativesEditOrderRequest) (*DerivativesOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, errDerivativesOrderIDRequired
	}
	if err := validateDerivativesOrderEdit(req.Amount, req.Contracts, req.Advanced, req.PostOnly, req.RejectPostOnly, req.Isolated, req.AllocatedMargin); err != nil {
		return nil, err
	}
	params := &derivativesEditOrderParams{DerivativesEditOrderRequest: req}
	if !req.ValidUntil.IsZero() {
		params.ValidUntil = req.ValidUntil.UnixMilli()
	}
	var resp *DerivativesOrderResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/edit", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// EditOrderByLabel edits the only open order of an instrument with a label, as EditOrder does; labels are not unique,
// and the gateway rejects the edit when more than one open order has the label
func (g GlobalDerivativesAPI) EditOrderByLabel(ctx context.Context, req *DerivativesEditOrderByLabelRequest) (*DerivativesOrderResponse, error) {
	return g.editOrderByLabel(ctx, false, req)
}

func (g GlobalDerivativesAPI) editOrderByLabel(ctx context.Context, overWebsocket bool, req *DerivativesEditOrderByLabelRequest) (*DerivativesOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesOrderEdit(req.Amount, req.Contracts, req.Advanced, req.PostOnly, req.RejectPostOnly, req.Isolated, req.AllocatedMargin); err != nil {
		return nil, err
	}
	params := &derivativesEditOrderByLabelParams{DerivativesEditOrderByLabelRequest: req}
	if !req.ValidUntil.IsZero() {
		params.ValidUntil = req.ValidUntil.UnixMilli()
	}
	var resp *DerivativesOrderResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesMatchingEPL, "private/edit_by_label", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMargins returns the margins and fees a hypothetical order would need, and the prices orders are clamped to; with
// Isolated it previews the margin an isolated margin order needs
func (g GlobalDerivativesAPI) GetMargins(ctx context.Context, req *DerivativesMarginsRequest) (*DerivativesMarginsResponse, error) {
	return g.getMargins(ctx, false, req)
}

func (g GlobalDerivativesAPI) getMargins(ctx context.Context, overWebsocket bool, req *DerivativesMarginsRequest) (*DerivativesMarginsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	var resp *DerivativesMarginsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_margins", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMaxOrderSize returns the largest buy, sell and reduce only orders a future can take given the account's funds,
// position and open orders, optionally at a hypothetical leverage, price and taker fee rate; it needs the standard
// margin model
func (g GlobalDerivativesAPI) GetMaxOrderSize(ctx context.Context, req *DerivativesMaxOrderSizeRequest) (*DerivativesMaxOrderSizeResponse, error) {
	return g.getMaxOrderSize(ctx, false, req)
}

func (g GlobalDerivativesAPI) getMaxOrderSize(ctx context.Context, overWebsocket bool, req *DerivativesMaxOrderSizeRequest) (*DerivativesMaxOrderSizeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if req.Leverage != 0 && req.Leverage < 1 {
		return nil, fmt.Errorf("%w: %v", errDerivativesInvalidLeverage, req.Leverage)
	}
	if req.TakerFeeRate != nil && *req.TakerFeeRate < 0 {
		return nil, fmt.Errorf("%w: %v", errDerivativesInvalidTakerFeeRate, *req.TakerFeeRate)
	}
	if req.Price < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPrice, req.Price)
	}
	var resp *DerivativesMaxOrderSizeResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_max_order_size", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOpenOrders returns the account's open orders in every currency, by instrument kind and order type; the hidden
// isolated margin subaccounts' orders are returned with IncludeIsolated
func (g GlobalDerivativesAPI) GetOpenOrders(ctx context.Context, req *DerivativesOpenOrdersRequest) ([]DerivativesOrder, error) {
	return g.getOpenOrders(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOpenOrders(ctx context.Context, overWebsocket bool, req *DerivativesOpenOrdersRequest) ([]DerivativesOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validateDerivativesTradingKind(req.Kind, false); err != nil {
		return nil, err
	}
	if err := validateDerivativesOpenOrderType(req.Type); err != nil {
		return nil, err
	}
	var resp []DerivativesOrder
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_open_orders", req, &resp)
}

// GetOpenOrdersByCurrency returns the open orders of a currency's instruments, by instrument kind and order type
func (g GlobalDerivativesAPI) GetOpenOrdersByCurrency(ctx context.Context, req *DerivativesOpenOrdersByCurrencyRequest) ([]DerivativesOrder, error) {
	return g.getOpenOrdersByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOpenOrdersByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesOpenOrdersByCurrencyRequest) ([]DerivativesOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesTradingKind(req.Kind, false); err != nil {
		return nil, err
	}
	if err := validateDerivativesOpenOrderType(req.Type); err != nil {
		return nil, err
	}
	params := &derivativesOpenOrdersByCurrencyParams{DerivativesOpenOrdersByCurrencyRequest: req, Currency: req.Currency.Upper().String()}
	var resp []DerivativesOrder
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_open_orders_by_currency", params, &resp)
}

// GetOpenOrdersByInstrument returns the open orders of an instrument, by order type
func (g GlobalDerivativesAPI) GetOpenOrdersByInstrument(ctx context.Context, req *DerivativesOpenOrdersByInstrumentRequest) ([]DerivativesOrder, error) {
	return g.getOpenOrdersByInstrument(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOpenOrdersByInstrument(ctx context.Context, overWebsocket bool, req *DerivativesOpenOrdersByInstrumentRequest) ([]DerivativesOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesOpenOrderType(req.Type); err != nil {
		return nil, err
	}
	var resp []DerivativesOrder
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_open_orders_by_instrument", req, &resp)
}

// GetOpenOrdersByLabel returns the open orders with a label in a currency's instruments
func (g GlobalDerivativesAPI) GetOpenOrdersByLabel(ctx context.Context, req *DerivativesOpenOrdersByLabelRequest) ([]DerivativesOrder, error) {
	return g.getOpenOrdersByLabel(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOpenOrdersByLabel(ctx context.Context, overWebsocket bool, req *DerivativesOpenOrdersByLabelRequest) ([]DerivativesOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Label == "" {
		return nil, errDerivativesLabelRequired
	}
	params := &derivativesOpenOrdersByLabelParams{DerivativesOpenOrdersByLabelRequest: req, Currency: req.Currency.Upper().String()}
	var resp []DerivativesOrder
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_open_orders_by_label", params, &resp)
}

// GetOrderHistoryByCurrency returns a page of the order history of a currency's instruments; recent orders are kept
// for 30 minutes, and Historical returns older ones. An isolated margin subaccount's history needs its SubaccountID
func (g GlobalDerivativesAPI) GetOrderHistoryByCurrency(ctx context.Context, req *DerivativesOrderHistoryByCurrencyRequest) (*DerivativesOrderHistoryResponse, error) {
	return g.getOrderHistoryByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOrderHistoryByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesOrderHistoryByCurrencyRequest) (*DerivativesOrderHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesTradingKind(req.Kind, true); err != nil {
		return nil, err
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	params := &derivativesOrderHistoryByCurrencyParams{DerivativesOrderHistoryByCurrencyRequest: req, Currency: req.Currency.Upper().String()}
	var resp *DerivativesOrderHistoryResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_order_history_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderHistoryByInstrument returns a page of an instrument's order history; recent orders are kept for 30 minutes,
// and Historical returns older ones
func (g GlobalDerivativesAPI) GetOrderHistoryByInstrument(ctx context.Context, req *DerivativesOrderHistoryByInstrumentRequest) (*DerivativesOrderHistoryResponse, error) {
	return g.getOrderHistoryByInstrument(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOrderHistoryByInstrument(ctx context.Context, overWebsocket bool, req *DerivativesOrderHistoryByInstrumentRequest) (*DerivativesOrderHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	var resp *DerivativesOrderHistoryResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_order_history_by_instrument", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderMarginByIDs returns the initial margin each order holds
func (g GlobalDerivativesAPI) GetOrderMarginByIDs(ctx context.Context, req *DerivativesOrderMarginByIDsRequest) ([]DerivativesOrderMargin, error) {
	return g.getOrderMarginByIDs(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOrderMarginByIDs(ctx context.Context, overWebsocket bool, req *DerivativesOrderMarginByIDsRequest) ([]DerivativesOrderMargin, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.OrderIDs) == 0 {
		return nil, errDerivativesOrderIDRequired
	}
	for i := range req.OrderIDs {
		if req.OrderIDs[i] == "" {
			return nil, fmt.Errorf("%w: order ID %d is empty", errDerivativesOrderIDRequired, i)
		}
	}
	var resp []DerivativesOrderMargin
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_order_margin_by_ids", req, &resp)
}

// GetOrderState returns an order that is open or closed within the last 30 minutes; for a quote's order, Amount is the
// amount remaining. Older orders are in the order history, and an isolated margin order is only found with Isolated
// set
func (g GlobalDerivativesAPI) GetOrderState(ctx context.Context, req *DerivativesOrderStateRequest) (*DerivativesOrder, error) {
	return g.getOrderState(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOrderState(ctx context.Context, overWebsocket bool, req *DerivativesOrderStateRequest) (*DerivativesOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, errDerivativesOrderIDRequired
	}
	var resp *DerivativesOrder
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_order_state", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderStateByLabel returns the orders with a label in a currency's instruments that are open or closed within the
// last 30 minutes, which a trigger order and the order it placed both share
func (g GlobalDerivativesAPI) GetOrderStateByLabel(ctx context.Context, req *DerivativesOrderStateByLabelRequest) ([]DerivativesOrder, error) {
	return g.getOrderStateByLabel(ctx, false, req)
}

func (g GlobalDerivativesAPI) getOrderStateByLabel(ctx context.Context, overWebsocket bool, req *DerivativesOrderStateByLabelRequest) ([]DerivativesOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Label == "" {
		return nil, errDerivativesLabelRequired
	}
	params := &derivativesOrderStateByLabelParams{DerivativesOrderStateByLabelRequest: req, Currency: req.Currency.Upper().String()}
	var resp []DerivativesOrder
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_order_state_by_label", params, &resp)
}

// GetSettlementHistoryByCurrency returns a page of the settlement, delivery and bankruptcy events that affected the
// account in a currency; the profit and loss fields are platform wide aggregates, not the account's. An isolated
// margin subaccount's perpetual settlements need its SubaccountID
func (g GlobalDerivativesAPI) GetSettlementHistoryByCurrency(ctx context.Context, req *DerivativesSettlementHistoryByCurrencyRequest) (*DerivativesSettlementsResponse, error) {
	return g.getSettlementHistoryByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getSettlementHistoryByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesSettlementHistoryByCurrencyRequest) (*DerivativesSettlementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesSettlementHistoryType(req.Type); err != nil {
		return nil, err
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	params := &derivativesSettlementHistoryByCurrencyParams{DerivativesSettlementHistoryByCurrencyRequest: req, Currency: req.Currency.Upper().String()}
	if !req.SearchStartTime.IsZero() {
		params.SearchStartTimestamp = req.SearchStartTime.UnixMilli()
	}
	var resp *DerivativesSettlementsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_settlement_history_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetSettlementHistoryByInstrument returns a page of the settlement, delivery and bankruptcy events that affected the
// account in an instrument; the profit and loss fields are platform wide aggregates, not the account's
func (g GlobalDerivativesAPI) GetSettlementHistoryByInstrument(ctx context.Context, req *DerivativesSettlementHistoryByInstrumentRequest) (*DerivativesSettlementsResponse, error) {
	return g.getSettlementHistoryByInstrument(ctx, false, req)
}

func (g GlobalDerivativesAPI) getSettlementHistoryByInstrument(ctx context.Context, overWebsocket bool, req *DerivativesSettlementHistoryByInstrumentRequest) (*DerivativesSettlementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesSettlementHistoryType(req.Type); err != nil {
		return nil, err
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	params := &derivativesSettlementHistoryByInstrumentParams{DerivativesSettlementHistoryByInstrumentRequest: req}
	if !req.SearchStartTime.IsZero() {
		params.SearchStartTimestamp = req.SearchStartTime.UnixMilli()
	}
	var resp *DerivativesSettlementsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_settlement_history_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTriggerOrderHistory returns a page of the account's trigger order log in a currency, recording each trigger
// order's creation, triggering, execution and cancellation
func (g GlobalDerivativesAPI) GetTriggerOrderHistory(ctx context.Context, req *DerivativesTriggerOrderHistoryRequest) (*DerivativesTriggerOrderHistoryResponse, error) {
	return g.getTriggerOrderHistory(ctx, false, req)
}

func (g GlobalDerivativesAPI) getTriggerOrderHistory(ctx context.Context, overWebsocket bool, req *DerivativesTriggerOrderHistoryRequest) (*DerivativesTriggerOrderHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	params := &derivativesTriggerOrderHistoryParams{DerivativesTriggerOrderHistoryRequest: req, Currency: req.Currency.Upper().String()}
	var resp *DerivativesTriggerOrderHistoryResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_trigger_order_history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByCurrency returns the account's latest trades in a currency's instruments; recent trades are kept for
// 24 hours, and Historical returns older ones
func (g GlobalDerivativesAPI) GetUserTradesByCurrency(ctx context.Context, req *DerivativesUserTradesByCurrencyRequest) (*DerivativesUserTradesResponse, error) {
	return g.getUserTradesByCurrency(ctx, false, req)
}

func (g GlobalDerivativesAPI) getUserTradesByCurrency(ctx context.Context, overWebsocket bool, req *DerivativesUserTradesByCurrencyRequest) (*DerivativesUserTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateDerivativesTradingKind(req.Kind, true); err != nil {
		return nil, err
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	if err := validateDerivativesUserTradesSorting(req.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := derivativesUserTradesTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	params := &derivativesUserTradesByCurrencyParams{DerivativesUserTradesByCurrencyRequest: req, Currency: req.Currency.Upper().String(), StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *DerivativesUserTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_user_trades_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByCurrencyAndTime returns the account's trades in a currency's instruments within a time range, which
// needs a start or an end; recent trades are kept for 24 hours, and Historical returns older ones
func (g GlobalDerivativesAPI) GetUserTradesByCurrencyAndTime(ctx context.Context, req *DerivativesUserTradesByCurrencyAndTimeRequest) (*DerivativesUserTradesResponse, error) {
	return g.getUserTradesByCurrencyAndTime(ctx, false, req)
}

func (g GlobalDerivativesAPI) getUserTradesByCurrencyAndTime(ctx context.Context, overWebsocket bool, req *DerivativesUserTradesByCurrencyAndTimeRequest) (*DerivativesUserTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.StartTime.IsZero() && req.EndTime.IsZero() {
		return nil, fmt.Errorf("%w: a start or an end time is required", common.ErrDateUnset)
	}
	if err := validateDerivativesTradingKind(req.Kind, true); err != nil {
		return nil, err
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	if err := validateDerivativesUserTradesSorting(req.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := derivativesUserTradesTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	params := &derivativesUserTradesByCurrencyAndTimeParams{DerivativesUserTradesByCurrencyAndTimeRequest: req, Currency: req.Currency.Upper().String(), StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *DerivativesUserTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_user_trades_by_currency_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByInstrument returns the account's latest trades in an instrument; recent trades are kept for 24 hours,
// and Historical returns older ones
func (g GlobalDerivativesAPI) GetUserTradesByInstrument(ctx context.Context, req *DerivativesUserTradesByInstrumentRequest) (*DerivativesUserTradesResponse, error) {
	return g.getUserTradesByInstrument(ctx, false, req)
}

func (g GlobalDerivativesAPI) getUserTradesByInstrument(ctx context.Context, overWebsocket bool, req *DerivativesUserTradesByInstrumentRequest) (*DerivativesUserTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	if err := validateDerivativesUserTradesSorting(req.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := derivativesUserTradesTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	params := &derivativesUserTradesByInstrumentParams{DerivativesUserTradesByInstrumentRequest: req, StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *DerivativesUserTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_user_trades_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByInstrumentAndTime returns the account's trades in an instrument within a time range, which needs a
// start or an end; recent trades are kept for 24 hours, and Historical returns older ones
func (g GlobalDerivativesAPI) GetUserTradesByInstrumentAndTime(ctx context.Context, req *DerivativesUserTradesByInstrumentAndTimeRequest) (*DerivativesUserTradesResponse, error) {
	return g.getUserTradesByInstrumentAndTime(ctx, false, req)
}

func (g GlobalDerivativesAPI) getUserTradesByInstrumentAndTime(ctx context.Context, overWebsocket bool, req *DerivativesUserTradesByInstrumentAndTimeRequest) (*DerivativesUserTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if req.StartTime.IsZero() && req.EndTime.IsZero() {
		return nil, fmt.Errorf("%w: a start or an end time is required", common.ErrDateUnset)
	}
	if err := validateDerivativesTradingCount(req.Count); err != nil {
		return nil, err
	}
	if err := validateDerivativesUserTradesSorting(req.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := derivativesUserTradesTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	params := &derivativesUserTradesByInstrumentAndTimeParams{DerivativesUserTradesByInstrumentAndTimeRequest: req, StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *DerivativesUserTradesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_user_trades_by_instrument_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByOrder returns the trades an order made; recent trades are kept for 24 hours, and Historical returns
// older ones. An isolated margin order's trades are only found with Isolated set
func (g GlobalDerivativesAPI) GetUserTradesByOrder(ctx context.Context, req *DerivativesUserTradesByOrderRequest) ([]DerivativesUserTrade, error) {
	return g.getUserTradesByOrder(ctx, false, req)
}

func (g GlobalDerivativesAPI) getUserTradesByOrder(ctx context.Context, overWebsocket bool, req *DerivativesUserTradesByOrderRequest) ([]DerivativesUserTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, errDerivativesOrderIDRequired
	}
	if err := validateDerivativesUserTradesSorting(req.Sorting); err != nil {
		return nil, err
	}
	var resp []DerivativesUserTrade
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_user_trades_by_order", req, &resp)
}

// Sell places a sell order and returns it with the trades it made at once; a spot order routed to Coinbase Exchange
// fills asynchronously, so its trades arrive on the user.trades channel instead
func (g GlobalDerivativesAPI) Sell(ctx context.Context, req *DerivativesOrderRequest) (*DerivativesOrderResponse, error) {
	return g.placeOrder(ctx, false, derivativesSellMethod, req)
}

// SetLeverage sets a future's custom leverage, or with a nil Leverage removes it, and returns the custom leverage the
// gateway stored, or zero once it is removed; isolated margin subaccounts take no custom leverage of their own, so an
// isolated margin scope's is set with Isolated from the main account
func (g GlobalDerivativesAPI) SetLeverage(ctx context.Context, req *DerivativesSetLeverageRequest) (float64, error) {
	return g.setLeverage(ctx, false, req)
}

func (g GlobalDerivativesAPI) setLeverage(ctx context.Context, overWebsocket bool, req *DerivativesSetLeverageRequest) (float64, error) {
	if err := common.NilGuard(req); err != nil {
		return 0, err
	}
	if req.InstrumentName == "" {
		return 0, errInstrumentNameRequired
	}
	if req.Leverage != nil && *req.Leverage < 1 {
		return 0, fmt.Errorf("%w: %v", errDerivativesInvalidLeverage, *req.Leverage)
	}
	var resp *float64
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/set_leverage", req, &resp); err != nil {
		return 0, err
	}
	if resp == nil {
		// The gateway answers null once a custom leverage is removed, which is no answer to setting one
		if req.Leverage != nil {
			return 0, common.ErrNoResponse
		}
		return 0, nil
	}
	return *resp, nil
}

// validate checks an order request carries an instrument, a size and documented enum values, that a trigger order
// carries its trigger, a linked order its secondary orders, and an isolated margin order only what isolated margin
// supports
func (o *DerivativesOrderRequest) validate() error {
	if err := common.NilGuard(o); err != nil {
		return err
	}
	if o.InstrumentName == "" {
		return errInstrumentNameRequired
	}
	if err := validateDerivativesOrderSize(o.Amount, o.Contracts); err != nil {
		return err
	}
	if err := validateDerivativesOrderType(o.Type, o.Trigger, o.TriggerPrice); err != nil {
		return err
	}
	if err := validateDerivativesTimeInForce(o.TimeInForce); err != nil {
		return err
	}
	if err := validateDerivativesTrigger(o.Trigger); err != nil {
		return err
	}
	if err := validateDerivativesAdvancedType(o.Advanced); err != nil {
		return err
	}
	if o.PostOnly && o.TimeInForce != "" && o.TimeInForce != DerivativesTimeInForceGoodTilCancelled {
		return fmt.Errorf("%w: time in force %s", errDerivativesPostOnlyTimeInForce, o.TimeInForce)
	}
	if o.RejectPostOnly && !o.PostOnly {
		return errDerivativesRejectPostOnlyWithoutPostOnly
	}
	if err := o.validateLinkedOrder(); err != nil {
		return err
	}
	if o.AllocatedMargin < 0 {
		return fmt.Errorf("%w: allocated margin %v", errInvalidAmount, o.AllocatedMargin)
	}
	if !o.Isolated {
		if o.AllocatedMargin != 0 {
			return errDerivativesAllocatedMarginWithoutIsolated
		}
		return nil
	}
	switch {
	case isDerivativesTriggerOrderType(o.Type):
		return fmt.Errorf("%w: %s order", errDerivativesIsolatedOrderUnsupported, o.Type)
	case o.Advanced != "":
		return fmt.Errorf("%w: advanced %s order", errDerivativesIsolatedOrderUnsupported, o.Advanced)
	case o.LinkedOrderType != "":
		return fmt.Errorf("%w: %s order", errDerivativesIsolatedOrderUnsupported, o.LinkedOrderType)
	}
	return nil
}

// validateLinkedOrder checks a linked order type and trigger fill condition are documented, that they are set together
// with secondary orders, as many as the linked order type takes, and that each secondary order is valid
func (o *DerivativesOrderRequest) validateLinkedOrder() error {
	switch o.TriggerFillCondition {
	case "", DerivativesTriggerFillConditionFirstHit, DerivativesTriggerFillConditionCompleteFill, DerivativesTriggerFillConditionIncremental:
	default:
		return fmt.Errorf("%w: %q", errDerivativesInvalidTriggerFillCondition, o.TriggerFillCondition)
	}
	switch o.LinkedOrderType {
	case "":
		if len(o.OTOCOConfig) != 0 || o.TriggerFillCondition != "" {
			return errDerivativesLinkedOrderTypeRequired
		}
		return nil
	case DerivativesLinkedOrderTypeOneTriggersOther:
		if len(o.OTOCOConfig) == 0 {
			return fmt.Errorf("%w: %s takes one or more, got none", errDerivativesInvalidSecondaryOrderCount, o.LinkedOrderType)
		}
	case DerivativesLinkedOrderTypeOneCancelsOther:
		if len(o.OTOCOConfig) != 1 {
			return fmt.Errorf("%w: %s takes one, got %d", errDerivativesInvalidSecondaryOrderCount, o.LinkedOrderType, len(o.OTOCOConfig))
		}
	case DerivativesLinkedOrderTypeOneTriggersOneCancelsOther:
		if len(o.OTOCOConfig) != 2 {
			return fmt.Errorf("%w: %s takes two, got %d", errDerivativesInvalidSecondaryOrderCount, o.LinkedOrderType, len(o.OTOCOConfig))
		}
	default:
		return fmt.Errorf("%w: %q", errDerivativesInvalidLinkedOrderType, o.LinkedOrderType)
	}
	for i := range o.OTOCOConfig {
		if err := o.OTOCOConfig[i].validate(); err != nil {
			return fmt.Errorf("secondary order %d: %w", i, err)
		}
	}
	return nil
}

// validate checks a secondary order carries its required amount and direction and documented enum values, and that a
// trigger order carries its trigger
func (s *DerivativesSecondaryOrder) validate() error {
	if s.Amount <= 0 {
		return fmt.Errorf("%w: %v", errInvalidAmount, s.Amount)
	}
	if s.Direction != DerivativesDirectionBuy && s.Direction != DerivativesDirectionSell {
		return fmt.Errorf("%w: %q", errDerivativesInvalidOrderDirection, s.Direction)
	}
	if err := validateDerivativesOrderType(s.Type, s.Trigger, s.TriggerPrice); err != nil {
		return err
	}
	if err := validateDerivativesTimeInForce(s.TimeInForce); err != nil {
		return err
	}
	return validateDerivativesTrigger(s.Trigger)
}

// validateDerivativesOrderEdit checks an edit carries a size and a documented advanced order type, that reject post only
// comes with post only, and that allocated margin is not negative and goes to an isolated margin order alone, as only
// an isolated margin subaccount takes margin from the main account
func validateDerivativesOrderEdit(amount, contracts float64, advanced string, postOnly, rejectPostOnly, isolated bool, allocatedMargin float64) error {
	if err := validateDerivativesOrderSize(amount, contracts); err != nil {
		return err
	}
	if err := validateDerivativesAdvancedType(advanced); err != nil {
		return err
	}
	if rejectPostOnly && !postOnly {
		return errDerivativesRejectPostOnlyWithoutPostOnly
	}
	if allocatedMargin < 0 {
		return fmt.Errorf("%w: allocated margin %v", errInvalidAmount, allocatedMargin)
	}
	if allocatedMargin != 0 && !isolated {
		return errDerivativesAllocatedMarginWithoutIsolated
	}
	return nil
}

// validateDerivativesOrderSize checks an order is sized by a positive amount, a positive number of contracts or both
func validateDerivativesOrderSize(amount, contracts float64) error {
	if amount < 0 || contracts < 0 || (amount == 0 && contracts == 0) {
		return fmt.Errorf("%w: amount %v, contracts %v", errInvalidAmount, amount, contracts)
	}
	return nil
}

// isDerivativesTriggerOrderType reports whether an order type is a trigger, or algo, order type
func isDerivativesTriggerOrderType(orderType string) bool {
	switch orderType {
	case DerivativesOrderTypeStopLimit, DerivativesOrderTypeStopMarket, DerivativesOrderTypeTakeLimit, DerivativesOrderTypeTakeMarket, DerivativesOrderTypeTrailingStop:
		return true
	}
	return false
}

// validateDerivativesOrderType checks an order type is empty or documented, and that a trigger order carries its
// trigger and, for a stop or take order, its trigger price
func validateDerivativesOrderType(orderType, trigger string, triggerPrice float64) error {
	switch orderType {
	case "", DerivativesOrderTypeLimit, DerivativesOrderTypeMarket, DerivativesOrderTypeMarketLimit:
		return nil
	}
	if !isDerivativesTriggerOrderType(orderType) {
		return fmt.Errorf("%w: %q", errDerivativesInvalidOrderType, orderType)
	}
	if trigger == "" {
		return fmt.Errorf("%w: %s", errDerivativesTriggerRequired, orderType)
	}
	if orderType != DerivativesOrderTypeTrailingStop && triggerPrice == 0 {
		return fmt.Errorf("%w: %s", errDerivativesTriggerPriceRequired, orderType)
	}
	return nil
}

// validateDerivativesTimeInForce checks a time in force is empty or documented
func validateDerivativesTimeInForce(timeInForce string) error {
	switch timeInForce {
	case "", DerivativesTimeInForceGoodTilCancelled, DerivativesTimeInForceGoodTilDay, DerivativesTimeInForceFillOrKill, DerivativesTimeInForceImmediateOrCancel:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidTimeInForce, timeInForce)
}

// validateDerivativesTrigger checks a trigger is empty or documented
func validateDerivativesTrigger(trigger string) error {
	switch trigger {
	case "", DerivativesTriggerIndexPrice, DerivativesTriggerMarkPrice, DerivativesTriggerLastPrice:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidTrigger, trigger)
}

// validateDerivativesAdvancedType checks an advanced option order type is empty, usd or implv
func validateDerivativesAdvancedType(advanced string) error {
	switch advanced {
	case "", DerivativesAdvancedUSD, DerivativesAdvancedImpliedVolatility:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidAdvancedType, advanced)
}

// validateDerivativesTradingKind checks an instrument kind is empty or one a trading method accepts; aggregates selects
// whether combo, which selects every combo, and any are accepted
func validateDerivativesTradingKind(kind string, aggregates bool) error {
	switch kind {
	case "", DerivativesKindFuture, DerivativesKindOption, DerivativesKindSpot, DerivativesKindFutureCombo, DerivativesKindOptionCombo:
		return nil
	case DerivativesKindCombo, DerivativesKindAny:
		if aggregates {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidTradingKind, kind)
}

// validateDerivativesCancelOrderType checks the order type a cancel all method selects is empty or documented
func validateDerivativesCancelOrderType(orderType string) error {
	switch orderType {
	case "", DerivativesCancelOrderTypeAll, DerivativesCancelOrderTypeLimit, DerivativesCancelOrderTypeTriggerAll, DerivativesCancelOrderTypeStop, DerivativesCancelOrderTypeTake, DerivativesCancelOrderTypeTrailingStop:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidCancelOrderType, orderType)
}

// validateDerivativesOpenOrderType checks the order type an open orders method selects is empty or documented
func validateDerivativesOpenOrderType(orderType string) error {
	switch orderType {
	case "", DerivativesOpenOrderTypeAll, DerivativesOpenOrderTypeLimit, DerivativesOpenOrderTypeTriggerAll, DerivativesOpenOrderTypeStopAll,
		DerivativesOpenOrderTypeStopLimit, DerivativesOpenOrderTypeStopMarket, DerivativesOpenOrderTypeTakeAll, DerivativesOpenOrderTypeTakeLimit,
		DerivativesOpenOrderTypeTakeMarket, DerivativesOpenOrderTypeTrailingAll, DerivativesOpenOrderTypeTrailingStop:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidOpenOrderType, orderType)
}

// validateDerivativesSettlementHistoryType checks a settlement event type is empty, settlement, delivery or bankruptcy
func validateDerivativesSettlementHistoryType(settlementType string) error {
	switch settlementType {
	case "", "settlement", "delivery", "bankruptcy":
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidSettlementHistoryType, settlementType)
}

// validateDerivativesUserTradesSorting checks a user trades sorting is empty or documented
func validateDerivativesUserTradesSorting(sorting string) error {
	switch sorting {
	case "", DerivativesSortingAscending, DerivativesSortingDescending, DerivativesSortingDefault:
		return nil
	}
	return fmt.Errorf("%w: %q", errDerivativesInvalidUserTradesSorting, sorting)
}

// validateDerivativesTradingCount checks a page size is within the 1000 items a trading history method returns at
// most; zero leaves the gateway's default
func validateDerivativesTradingCount(count uint64) error {
	if count > derivativesMaximumTradingCount {
		return fmt.Errorf("%w: %d exceeds %d", errDerivativesInvalidTradingCount, count, derivativesMaximumTradingCount)
	}
	return nil
}

// derivativesKindOrTypeCurrencies returns the currencies a cancel by kind or type request selects as the gateway takes
// them: one currency on its own and several as a list, each upper case but for the lower case any
func derivativesKindOrTypeCurrencies(codes []currency.Code) (any, error) {
	if len(codes) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencies := make([]string, len(codes))
	for i := range codes {
		switch {
		case codes[i].IsEmpty():
			return nil, fmt.Errorf("%w: currency %d", currency.ErrCurrencyCodeEmpty, i)
		case strings.EqualFold(codes[i].String(), DerivativesCurrencyAny):
			currencies[i] = DerivativesCurrencyAny
		default:
			currencies[i] = codes[i].Upper().String()
		}
	}
	if len(currencies) == 1 {
		return currencies[0], nil
	}
	return currencies, nil
}

// derivativesUserTradesTimeRange returns a trade query's bounds in Unix milliseconds, zero for an unset bound, once
// checkWindow accepts them
func derivativesUserTradesTimeRange(start, end time.Time) (startTimestamp, endTimestamp int64, err error) {
	if err := checkWindow(start, end); err != nil {
		return 0, 0, err
	}
	if !start.IsZero() {
		startTimestamp = start.UnixMilli()
	}
	if !end.IsZero() {
		endTimestamp = end.UnixMilli()
	}
	return startTimestamp, endTimestamp, nil
}
