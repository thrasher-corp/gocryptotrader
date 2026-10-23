package deribit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	// maximumTradingCount is the most items a trading history request returns in a page
	maximumTradingCount = 1000
	// maximumMMPDuration is the longest MMP interval and frozen time Deribit accepts
	maximumMMPDuration = time.Hour
	// maximumMMPTradeCountLimit is the highest block RFQ MMP trade count limit Deribit accepts
	maximumMMPTradeCountLimit = 1000
	// mmpIndexNameAll selects every index, which only block RFQ MMP resets accept
	mmpIndexNameAll = "all"
)

var (
	errInvalidOrderType             = errors.New("invalid order type")
	errInvalidOrderTimeInForce      = errors.New("invalid order time in force")
	errInvalidOrderTrigger          = errors.New("invalid order trigger")
	errOrderTriggerRequired         = errors.New("trigger is required for a trigger order")
	errOrderTriggerPriceRequired    = errors.New("trigger price is required for a stop or take order")
	errInvalidAdvancedOrderType     = errors.New("invalid advanced order type")
	errInvalidLinkedOrderType       = errors.New("invalid linked order type")
	errInvalidTriggerFillCondition  = errors.New("invalid trigger fill condition")
	errInvalidCancelOrderType       = errors.New("invalid cancel order type")
	errStarbaseOrdersWithDetailed   = errors.New("including Starbase orders cannot be combined with a detailed response")
	errCancelCurrencyPairRequired   = errors.New("currency pair is required")
	errInvalidCancelQuotesType      = errors.New("invalid cancel quotes type")
	errCancelQuoteSetIDRequired     = errors.New("quote set ID is required")
	errInvalidMarginAmountType      = errors.New("invalid margin amount type")
	errMMPIndexNameRequired         = errors.New("index name is required")
	errMMPIndexNameAllNeedsBlockRFQ = errors.New("index name all is only accepted for block RFQ MMP")
	errInvalidMMPInterval           = errors.New("MMP interval must be whole seconds up to an hour")
	errInvalidMMPFrozenTime         = errors.New("MMP frozen time must be whole seconds up to an hour")
	errInvalidMMPTradeCountLimit    = errors.New("MMP trade count limit must be at most 1000")
	errMMPGroupRequired             = errors.New("MMP group is required")
	errInvalidOpenOrderType         = errors.New("invalid open order type")
	errInvalidSettlementHistoryType = errors.New("invalid settlement history type")
	errInvalidUserTradesSorting     = errors.New("invalid user trades sorting")
	errMassQuoteIDRequired          = errors.New("quote ID is required")
	errMassQuotesRequired           = errors.New("at least one quote is required")
	errMassQuoteSideRequired        = errors.New("a quote needs a bid or an ask")
	errSourceUIDRequired            = errors.New("source subaccount ID is required")
	errTargetUIDRequired            = errors.New("target subaccount ID is required")
	errPositionMovesRequired        = errors.New("at least one position move is required")
	errOrderHistoryOrdersMissing    = errors.New("order history page has no orders field")
	errInvalidCancelAllResult       = errors.New("invalid cancel all result")
)

// Buy places a buy order and returns it with the trades it made at once; spot orders routed to Coinbase Exchange fill
// asynchronously, so their trades arrive on the user.trades channel instead
func (e *Exchange) Buy(ctx context.Context, arg *OrderRequest) (*OrderWithTradesResponse, error) {
	return e.placeOrder(ctx, "private/buy", arg, false)
}

// placeOrder places a buy or sell order, which take the same params
func (e *Exchange) placeOrder(ctx context.Context, method string, arg *OrderRequest, overWebsocket bool) (*OrderWithTradesResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	params := &OrderParams{OrderRequest: arg, ValidUntil: unixMilli(arg.ValidUntil)}
	var resp *OrderWithTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, method, params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOrderByID cancels an open order or trigger order and returns it
func (e *Exchange) CancelOrderByID(ctx context.Context, orderID string) (*OrderResponse, error) {
	return e.cancelOrderByID(ctx, orderID, false)
}

func (e *Exchange) cancelOrderByID(ctx context.Context, orderID string, overWebsocket bool) (*OrderResponse, error) {
	if orderID == "" {
		return nil, errOrderIDRequired
	}
	var resp *OrderResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel", &OrderIDRequest{OrderID: orderID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAll cancels every open order and trigger order of the account and returns the number cancelled, or with
// Detailed the execution reports
func (e *Exchange) CancelAll(ctx context.Context, arg *CancelAllRequest) (*CancelAllResponse, error) {
	return e.cancelAll(ctx, arg, false)
}

func (e *Exchange) cancelAll(ctx context.Context, arg *CancelAllRequest, overWebsocket bool) (*CancelAllResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.IncludeStarbaseOrders && arg.Detailed {
		return nil, errStarbaseOrdersWithDetailed
	}
	var resp *CancelAllResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel_all", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByCurrency cancels the open orders of a currency's instruments, by instrument kind and order type, and
// returns the number cancelled, or with Detailed the execution reports
func (e *Exchange) CancelAllByCurrency(ctx context.Context, arg *CancelAllByCurrencyRequest) (*CancelAllResponse, error) {
	return e.cancelAllByCurrency(ctx, arg, false)
}

func (e *Exchange) cancelAllByCurrency(ctx context.Context, arg *CancelAllByCurrencyRequest, overWebsocket bool) (*CancelAllResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateTradingKind(arg.Kind, true); err != nil {
		return nil, err
	}
	if err := validateCancelOrderType(arg.Type); err != nil {
		return nil, err
	}
	var resp *CancelAllResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel_all_by_currency", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByCurrencyPair cancels the open orders of a currency pair's instruments, by instrument kind and order type,
// and returns the number cancelled, or with Detailed the execution reports
func (e *Exchange) CancelAllByCurrencyPair(ctx context.Context, arg *CancelAllByCurrencyPairRequest) (*CancelAllResponse, error) {
	return e.cancelAllByCurrencyPair(ctx, arg, false)
}

func (e *Exchange) cancelAllByCurrencyPair(ctx context.Context, arg *CancelAllByCurrencyPairRequest, overWebsocket bool) (*CancelAllResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.CurrencyPair == "" {
		return nil, errCancelCurrencyPairRequired
	}
	if err := validateTradingKind(arg.Kind, true); err != nil {
		return nil, err
	}
	if err := validateCancelOrderType(arg.Type); err != nil {
		return nil, err
	}
	var resp *CancelAllResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel_all_by_currency_pair", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByInstrument cancels the open orders of an instrument, by order type, and returns the number cancelled, or
// with Detailed the execution reports
func (e *Exchange) CancelAllByInstrument(ctx context.Context, arg *CancelAllByInstrumentRequest) (*CancelAllResponse, error) {
	return e.cancelAllByInstrument(ctx, arg, false)
}

func (e *Exchange) cancelAllByInstrument(ctx context.Context, arg *CancelAllByInstrumentRequest, overWebsocket bool) (*CancelAllResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateCancelOrderType(arg.Type); err != nil {
		return nil, err
	}
	var resp *CancelAllResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel_all_by_instrument", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllByKindOrType cancels the open orders of one or more currencies, by instrument kind and order type, and
// returns the number cancelled, or with Detailed the execution reports
func (e *Exchange) CancelAllByKindOrType(ctx context.Context, arg *CancelAllByKindOrTypeRequest) (*CancelAllResponse, error) {
	return e.cancelAllByKindOrType(ctx, arg, false)
}

func (e *Exchange) cancelAllByKindOrType(ctx context.Context, arg *CancelAllByKindOrTypeRequest, overWebsocket bool) (*CancelAllResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.Currencies) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	for i := range arg.Currencies {
		if arg.Currencies[i].IsEmpty() {
			return nil, fmt.Errorf("%w: currency %d", currency.ErrCurrencyCodeEmpty, i)
		}
	}
	if err := validateTradingKind(arg.Kind, true); err != nil {
		return nil, err
	}
	if err := validateCancelOrderType(arg.Type); err != nil {
		return nil, err
	}
	if arg.IncludeStarbaseOrders && arg.Detailed {
		return nil, errStarbaseOrdersWithDetailed
	}
	params := &CancelAllByKindOrTypeParams{CancelAllByKindOrTypeRequest: arg, Currency: arg.Currencies}
	if len(arg.Currencies) == 1 {
		params.Currency = arg.Currencies[0]
	}
	var resp *CancelAllResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel_all_by_kind_or_type", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelByLabel cancels the open orders and trigger orders with a label, in every currency or one, and returns the
// number cancelled, or with Detailed the execution reports
func (e *Exchange) CancelByLabel(ctx context.Context, arg *CancelByLabelRequest) (*CancelAllResponse, error) {
	return e.cancelByLabel(ctx, arg, false)
}

func (e *Exchange) cancelByLabel(ctx context.Context, arg *CancelByLabelRequest, overWebsocket bool) (*CancelAllResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Label == "" {
		return nil, errLabelRequired
	}
	var resp *CancelAllResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel_by_label", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelQuotes cancels the mass quotes CancelType selects and returns the number cancelled, or with Detailed the
// execution reports; Deribit needs cancel on disconnect enabled on the connection
func (e *Exchange) CancelQuotes(ctx context.Context, arg *CancelQuotesRequest) (*CancelAllResponse, error) {
	return e.cancelQuotes(ctx, arg, false)
}

func (e *Exchange) cancelQuotes(ctx context.Context, arg *CancelQuotesRequest, overWebsocket bool) (*CancelAllResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	params := &CancelQuotesParams{CancelQuotesRequest: arg}
	if arg.CancelType == "delta" {
		params.MinDelta, params.MaxDelta = &arg.MinDelta, &arg.MaxDelta
	}
	var resp *CancelAllResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/cancel_quotes", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ClosePosition closes a position with a reduce only limit or market order and returns the order with the trades it
// made at once
func (e *Exchange) ClosePosition(ctx context.Context, arg *ClosePositionRequest) (*OrderWithTradesResponse, error) {
	return e.closePosition(ctx, arg, false)
}

func (e *Exchange) closePosition(ctx context.Context, arg *ClosePositionRequest, overWebsocket bool) (*OrderWithTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if arg.Type != OrderTypeLimit && arg.Type != OrderTypeMarket {
		return nil, fmt.Errorf("%w: %q", errInvalidOrderType, arg.Type)
	}
	var resp *OrderWithTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/close_position", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// EditOrder edits an open order's size, price and flags and returns it with the trades it made at once; Deribit
// answers order_not_found for an order a mass quote placed
func (e *Exchange) EditOrder(ctx context.Context, arg *EditOrderRequest) (*OrderWithTradesResponse, error) {
	return e.editOrder(ctx, arg, false)
}

func (e *Exchange) editOrder(ctx context.Context, arg *EditOrderRequest, overWebsocket bool) (*OrderWithTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" {
		return nil, errOrderIDRequired
	}
	if err := validateOrderSize(arg.Amount, arg.Contracts); err != nil {
		return nil, err
	}
	if err := validateAdvancedOrderType(arg.Advanced); err != nil {
		return nil, err
	}
	params := &EditOrderParams{EditOrderRequest: arg, ValidUntil: unixMilli(arg.ValidUntil)}
	var resp *OrderWithTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/edit", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// EditOrderByLabel edits the only open order of an instrument with a label, as EditOrder does; Deribit rejects the edit
// when more than one open order has the label
func (e *Exchange) EditOrderByLabel(ctx context.Context, arg *EditOrderByLabelRequest) (*OrderWithTradesResponse, error) {
	return e.editOrderByLabel(ctx, arg, false)
}

func (e *Exchange) editOrderByLabel(ctx context.Context, arg *EditOrderByLabelRequest, overWebsocket bool) (*OrderWithTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateOrderSize(arg.Amount, arg.Contracts); err != nil {
		return nil, err
	}
	if err := validateAdvancedOrderType(arg.Advanced); err != nil {
		return nil, err
	}
	params := &EditOrderByLabelParams{EditOrderByLabelRequest: arg, ValidUntil: unixMilli(arg.ValidUntil)}
	var resp *OrderWithTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, matchingEPL, "private/edit_by_label", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMargins returns the margins and fees a hypothetical order would need, and the prices orders are clamped to
func (e *Exchange) GetMargins(ctx context.Context, arg *MarginsRequest) (*MarginsResponse, error) {
	return e.getMargins(ctx, arg, false)
}

func (e *Exchange) getMargins(ctx context.Context, arg *MarginsRequest, overWebsocket bool) (*MarginsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, arg.Amount)
	}
	if arg.AmountType != "" && arg.AmountType != "base" && arg.AmountType != "quote" {
		return nil, fmt.Errorf("%w: %q", errInvalidMarginAmountType, arg.AmountType)
	}
	var resp *MarginsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_margins", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetMMPConfig returns the MMP configurations of an index's group, or of every group without an index; an empty list
// means none is configured
func (e *Exchange) GetMMPConfig(ctx context.Context, arg *MMPGroupRequest) ([]MMPConfigResponse, error) {
	return e.getMMPConfig(ctx, arg, false)
}

func (e *Exchange) getMMPConfig(ctx context.Context, arg *MMPGroupRequest, overWebsocket bool) ([]MMPConfigResponse, error) {
	if err := arg.validateQuery(); err != nil {
		return nil, err
	}
	var resp []MMPConfigResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_mmp_config", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetMMPStatus returns the status of an index's triggered MMP group, or of every triggered group without an index
func (e *Exchange) GetMMPStatus(ctx context.Context, arg *MMPGroupRequest) ([]MMPStatusResponse, error) {
	return e.getMMPStatus(ctx, arg, false)
}

func (e *Exchange) getMMPStatus(ctx context.Context, arg *MMPGroupRequest, overWebsocket bool) ([]MMPStatusResponse, error) {
	if err := arg.validateQuery(); err != nil {
		return nil, err
	}
	var resp []MMPStatusResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_mmp_status", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOpenOrders returns the account's open orders in every currency, by instrument kind and order type
func (e *Exchange) GetOpenOrders(ctx context.Context, arg *OpenOrdersRequest) ([]OrderResponse, error) {
	return e.getOpenOrders(ctx, arg, false)
}

func (e *Exchange) getOpenOrders(ctx context.Context, arg *OpenOrdersRequest, overWebsocket bool) ([]OrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if err := validateTradingKind(arg.Kind, false); err != nil {
		return nil, err
	}
	if err := validateOpenOrderType(arg.Type); err != nil {
		return nil, err
	}
	var resp []OrderResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_open_orders", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOpenOrdersByCurrency returns the open orders of a currency's instruments, by instrument kind and order type
func (e *Exchange) GetOpenOrdersByCurrency(ctx context.Context, arg *OpenOrdersByCurrencyRequest) ([]OrderResponse, error) {
	return e.getOpenOrdersByCurrency(ctx, arg, false)
}

func (e *Exchange) getOpenOrdersByCurrency(ctx context.Context, arg *OpenOrdersByCurrencyRequest, overWebsocket bool) ([]OrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateTradingKind(arg.Kind, false); err != nil {
		return nil, err
	}
	if err := validateOpenOrderType(arg.Type); err != nil {
		return nil, err
	}
	var resp []OrderResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_open_orders_by_currency", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOpenOrdersByInstrument returns the open orders of an instrument, by order type
func (e *Exchange) GetOpenOrdersByInstrument(ctx context.Context, arg *OpenOrdersByInstrumentRequest) ([]OrderResponse, error) {
	return e.getOpenOrdersByInstrument(ctx, arg, false)
}

func (e *Exchange) getOpenOrdersByInstrument(ctx context.Context, arg *OpenOrdersByInstrumentRequest, overWebsocket bool) ([]OrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateOpenOrderType(arg.Type); err != nil {
		return nil, err
	}
	var resp []OrderResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_open_orders_by_instrument", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOpenOrdersByLabel returns the open orders with a label in a currency's instruments
func (e *Exchange) GetOpenOrdersByLabel(ctx context.Context, arg *OrdersByLabelRequest) ([]OrderResponse, error) {
	return e.getOrdersByLabel(ctx, "private/get_open_orders_by_label", arg, false)
}

// getOrdersByLabel returns the open orders or the recent order states with a label, which take the same params
func (e *Exchange) getOrdersByLabel(ctx context.Context, method string, arg *OrdersByLabelRequest, overWebsocket bool) ([]OrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Label == "" {
		return nil, errLabelRequired
	}
	var resp []OrderResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, method, arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOrderHistoryByCurrency returns a page of the order history of a currency's instruments; recent orders are kept for
// 30 minutes, and Historical returns older ones
func (e *Exchange) GetOrderHistoryByCurrency(ctx context.Context, arg *OrderHistoryByCurrencyRequest) (*OrderHistoryResponse, error) {
	return e.getOrderHistoryByCurrency(ctx, arg, false)
}

func (e *Exchange) getOrderHistoryByCurrency(ctx context.Context, arg *OrderHistoryByCurrencyRequest, overWebsocket bool) (*OrderHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateTradingKind(arg.Kind, true); err != nil {
		return nil, err
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	var resp *OrderHistoryResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_order_history_by_currency", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderHistoryByInstrument returns a page of an instrument's order history; recent orders are kept for 30 minutes,
// and Historical returns older ones
func (e *Exchange) GetOrderHistoryByInstrument(ctx context.Context, arg *OrderHistoryByInstrumentRequest) (*OrderHistoryResponse, error) {
	return e.getOrderHistoryByInstrument(ctx, arg, false)
}

func (e *Exchange) getOrderHistoryByInstrument(ctx context.Context, arg *OrderHistoryByInstrumentRequest, overWebsocket bool) (*OrderHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	var resp *OrderHistoryResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_order_history_by_instrument", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderMarginByIDs returns the initial margin each order holds
func (e *Exchange) GetOrderMarginByIDs(ctx context.Context, orderIDs []string) ([]OrderMarginResponse, error) {
	return e.getOrderMarginByIDs(ctx, orderIDs, false)
}

func (e *Exchange) getOrderMarginByIDs(ctx context.Context, orderIDs []string, overWebsocket bool) ([]OrderMarginResponse, error) {
	if len(orderIDs) == 0 {
		return nil, errOrderIDRequired
	}
	for i := range orderIDs {
		if orderIDs[i] == "" {
			return nil, fmt.Errorf("%w: order ID %d is empty", errOrderIDRequired, i)
		}
	}
	var resp []OrderMarginResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_order_margin_by_ids", &OrderMarginByIDsRequest{IDs: orderIDs}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOrderState returns an order that is open or closed within the last 30 minutes; for a quote's order, Amount is
// the amount remaining. Older orders are in the order history
func (e *Exchange) GetOrderState(ctx context.Context, orderID string) (*OrderResponse, error) {
	return e.getOrderState(ctx, orderID, false)
}

func (e *Exchange) getOrderState(ctx context.Context, orderID string, overWebsocket bool) (*OrderResponse, error) {
	if orderID == "" {
		return nil, errOrderIDRequired
	}
	var resp *OrderResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_order_state", &OrderIDRequest{OrderID: orderID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOrderStateByLabel returns the orders with a label in a currency's instruments that are open or closed within the
// last 30 minutes, which a trigger order and the order it placed both share
func (e *Exchange) GetOrderStateByLabel(ctx context.Context, arg *OrdersByLabelRequest) ([]OrderResponse, error) {
	return e.getOrdersByLabel(ctx, "private/get_order_state_by_label", arg, false)
}

// GetSettlementHistoryByCurrency returns a page of the settlement, delivery and bankruptcy events that affected the
// account in a currency; the profit and loss fields are platform wide aggregates, not the account's
func (e *Exchange) GetSettlementHistoryByCurrency(ctx context.Context, arg *SettlementHistoryByCurrencyRequest) (*SettlementsResponse, error) {
	return e.getSettlementHistoryByCurrency(ctx, arg, false)
}

func (e *Exchange) getSettlementHistoryByCurrency(ctx context.Context, arg *SettlementHistoryByCurrencyRequest, overWebsocket bool) (*SettlementsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateSettlementHistoryType(arg.Type); err != nil {
		return nil, err
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	params := &SettlementHistoryByCurrencyParams{SettlementHistoryByCurrencyRequest: arg, SearchStartTimestamp: unixMilli(arg.SearchStartTime)}
	var resp *SettlementsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_settlement_history_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetSettlementHistoryByInstrument returns a page of the settlement, delivery and bankruptcy events that affected the
// account in an instrument; the profit and loss fields are platform wide aggregates, not the account's
func (e *Exchange) GetSettlementHistoryByInstrument(ctx context.Context, arg *SettlementHistoryByInstrumentRequest) (*SettlementsResponse, error) {
	return e.getSettlementHistoryByInstrument(ctx, arg, false)
}

func (e *Exchange) getSettlementHistoryByInstrument(ctx context.Context, arg *SettlementHistoryByInstrumentRequest, overWebsocket bool) (*SettlementsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateSettlementHistoryType(arg.Type); err != nil {
		return nil, err
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	params := &SettlementHistoryByInstrumentParams{SettlementHistoryByInstrumentRequest: arg, SearchStartTimestamp: unixMilli(arg.SearchStartTime)}
	var resp *SettlementsResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_settlement_history_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTriggerOrderHistory returns a page of the account's trigger order log in a currency, recording each trigger
// order's creation, triggering, execution and cancellation
func (e *Exchange) GetTriggerOrderHistory(ctx context.Context, arg *TriggerOrderHistoryRequest) (*TriggerOrderHistoryResponse, error) {
	return e.getTriggerOrderHistory(ctx, arg, false)
}

func (e *Exchange) getTriggerOrderHistory(ctx context.Context, arg *TriggerOrderHistoryRequest, overWebsocket bool) (*TriggerOrderHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	var resp *TriggerOrderHistoryResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_trigger_order_history", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByCurrency returns the account's latest trades in a currency's instruments; recent trades are kept for
// 24 hours, and Historical returns older ones
func (e *Exchange) GetUserTradesByCurrency(ctx context.Context, arg *UserTradesByCurrencyRequest) (*UserTradesResponse, error) {
	return e.getUserTradesByCurrency(ctx, arg, false)
}

func (e *Exchange) getUserTradesByCurrency(ctx context.Context, arg *UserTradesByCurrencyRequest, overWebsocket bool) (*UserTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateTradingKind(arg.Kind, true); err != nil {
		return nil, err
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	if err := validateUserTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := userTradesTimeRange(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	params := &UserTradesByCurrencyParams{UserTradesByCurrencyRequest: arg, StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *UserTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_user_trades_by_currency", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByCurrencyAndTime returns the account's trades in a currency's instruments within a time range, which
// needs a start or an end; recent trades are kept for 24 hours, and Historical returns older ones
func (e *Exchange) GetUserTradesByCurrencyAndTime(ctx context.Context, arg *UserTradesByCurrencyAndTimeRequest) (*UserTradesResponse, error) {
	return e.getUserTradesByCurrencyAndTime(ctx, arg, false)
}

func (e *Exchange) getUserTradesByCurrencyAndTime(ctx context.Context, arg *UserTradesByCurrencyAndTimeRequest, overWebsocket bool) (*UserTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.StartTime.IsZero() && arg.EndTime.IsZero() {
		return nil, fmt.Errorf("%w: a start or an end time is required", common.ErrDateUnset)
	}
	if err := validateTradingKind(arg.Kind, true); err != nil {
		return nil, err
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	if err := validateUserTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := userTradesTimeRange(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	params := &UserTradesByCurrencyAndTimeParams{UserTradesByCurrencyAndTimeRequest: arg, StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *UserTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_user_trades_by_currency_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByInstrument returns the account's latest trades in an instrument; recent trades are kept for 24 hours,
// and Historical returns older ones
func (e *Exchange) GetUserTradesByInstrument(ctx context.Context, arg *UserTradesByInstrumentRequest) (*UserTradesResponse, error) {
	return e.getUserTradesByInstrument(ctx, arg, false)
}

func (e *Exchange) getUserTradesByInstrument(ctx context.Context, arg *UserTradesByInstrumentRequest, overWebsocket bool) (*UserTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	if err := validateUserTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := userTradesTimeRange(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	params := &UserTradesByInstrumentParams{UserTradesByInstrumentRequest: arg, StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *UserTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_user_trades_by_instrument", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByInstrumentAndTime returns the account's trades in an instrument within a time range, which needs a
// start or an end; recent trades are kept for 24 hours, and Historical returns older ones
func (e *Exchange) GetUserTradesByInstrumentAndTime(ctx context.Context, arg *UserTradesByInstrumentAndTimeRequest) (*UserTradesResponse, error) {
	return e.getUserTradesByInstrumentAndTime(ctx, arg, false)
}

func (e *Exchange) getUserTradesByInstrumentAndTime(ctx context.Context, arg *UserTradesByInstrumentAndTimeRequest, overWebsocket bool) (*UserTradesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if arg.StartTime.IsZero() && arg.EndTime.IsZero() {
		return nil, fmt.Errorf("%w: a start or an end time is required", common.ErrDateUnset)
	}
	if err := validateTradingCount(arg.Count); err != nil {
		return nil, err
	}
	if err := validateUserTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := userTradesTimeRange(arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	params := &UserTradesByInstrumentAndTimeParams{UserTradesByInstrumentAndTimeRequest: arg, StartTimestamp: startTimestamp, EndTimestamp: endTimestamp}
	var resp *UserTradesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_user_trades_by_instrument_and_time", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserTradesByOrder returns the trades an order made; recent trades are kept for 24 hours, and Historical returns
// older ones
func (e *Exchange) GetUserTradesByOrder(ctx context.Context, arg *UserTradesByOrderRequest) ([]UserTradeResponse, error) {
	return e.getUserTradesByOrder(ctx, arg, false)
}

func (e *Exchange) getUserTradesByOrder(ctx context.Context, arg *UserTradesByOrderRequest, overWebsocket bool) ([]UserTradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" {
		return nil, errOrderIDRequired
	}
	if err := validateUserTradesSorting(arg.Sorting); err != nil {
		return nil, err
	}
	var resp []UserTradeResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_user_trades_by_order", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// MassQuote places or amends up to 100 quotes on instruments of one index in a single request, linked to an MMP group;
// it needs Deribit's approval and cancel on disconnect enabled, and Deribit rejects the whole request when either fails
func (e *Exchange) MassQuote(ctx context.Context, arg *MassQuoteRequest) (*MassQuoteResponse, error) {
	return e.massQuote(ctx, arg, false)
}

func (e *Exchange) massQuote(ctx context.Context, arg *MassQuoteRequest, overWebsocket bool) (*MassQuoteResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	params := &MassQuoteParams{MassQuoteRequest: arg, ValidUntil: unixMilli(arg.ValidUntil)}
	var resp *MassQuoteResponse
	if err := e.sendRequest(ctx, overWebsocket, massQuoteEPL, "private/mass_quote", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// MovePositions moves positions between subaccounts with trades at the given prices or the positions' average prices,
// and returns the trades; Deribit allows 6 moves a minute and 100 a week, and an internal_server_error may still have
// moved some positions
func (e *Exchange) MovePositions(ctx context.Context, arg *MovePositionsRequest) ([]PositionMoveTradeResponse, error) {
	return e.movePositions(ctx, arg, false)
}

func (e *Exchange) movePositions(ctx context.Context, arg *MovePositionsRequest, overWebsocket bool) ([]PositionMoveTradeResponse, error) {
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp positionMoveTrades
	if err := e.sendRequest(ctx, overWebsocket, movePositionsEPL, "private/move_positions", arg, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ResetMMP resets an index's triggered MMP group, so quoting resumes before its frozen time ends, or at all when the
// frozen time is zero
func (e *Exchange) ResetMMP(ctx context.Context, arg *MMPGroupRequest) error {
	return e.resetMMP(ctx, arg, false)
}

func (e *Exchange) resetMMP(ctx context.Context, arg *MMPGroupRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.IndexName == "" {
		return errMMPIndexNameRequired
	}
	if arg.IndexName == mmpIndexNameAll && !arg.BlockRFQ {
		return errMMPIndexNameAllNeedsBlockRFQ
	}
	return e.sendOKRequest(ctx, overWebsocket, "private/reset_mmp", arg)
}

// Sell places a sell order and returns it with the trades it made at once; spot orders routed to Coinbase Exchange fill
// asynchronously, so their trades arrive on the user.trades channel instead
func (e *Exchange) Sell(ctx context.Context, arg *OrderRequest) (*OrderWithTradesResponse, error) {
	return e.placeOrder(ctx, "private/sell", arg, false)
}

// SetMMPConfig sets an MMP group's configuration, or with a zero interval removes it, and returns the configurations
// Deribit reports; a non-zero MaxQuoteQuantity reserves initial margin until the configuration is removed
func (e *Exchange) SetMMPConfig(ctx context.Context, arg *SetMMPConfigRequest) ([]MMPConfigResponse, error) {
	return e.setMMPConfig(ctx, arg, false)
}

func (e *Exchange) setMMPConfig(ctx context.Context, arg *SetMMPConfigRequest, overWebsocket bool) ([]MMPConfigResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.IndexName == "" {
		return nil, errMMPIndexNameRequired
	}
	if !isValidMMPDuration(arg.Interval) {
		return nil, fmt.Errorf("%w: %s", errInvalidMMPInterval, arg.Interval)
	}
	if !isValidMMPDuration(arg.FrozenTime) {
		return nil, fmt.Errorf("%w: %s", errInvalidMMPFrozenTime, arg.FrozenTime)
	}
	if arg.TradeCountLimit > maximumMMPTradeCountLimit {
		return nil, fmt.Errorf("%w: %d", errInvalidMMPTradeCountLimit, arg.TradeCountLimit)
	}
	params := &SetMMPConfigParams{
		SetMMPConfigRequest: arg,
		Interval:            uint64(arg.Interval.Seconds()),
		FrozenTime:          uint64(arg.FrozenTime.Seconds()),
	}
	var resp mmpConfigs
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_mmp_config", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// validate checks an order request carries an instrument, a size and documented enum values, and that a trigger order
// carries its trigger
func (o *OrderRequest) validate() error {
	if o == nil {
		return common.ErrNilPointer
	}
	if o.InstrumentName == "" {
		return errInstrumentNameRequired
	}
	if err := validateOrderSize(o.Amount, o.Contracts); err != nil {
		return err
	}
	if err := validateOrderType(o.Type, o.Trigger, o.TriggerPrice); err != nil {
		return err
	}
	if err := validateOrderTimeInForce(o.TimeInForce); err != nil {
		return err
	}
	if err := validateOrderTrigger(o.Trigger); err != nil {
		return err
	}
	if err := validateAdvancedOrderType(o.Advanced); err != nil {
		return err
	}
	switch o.LinkedOrderType {
	case "", "one_triggers_other", "one_cancels_other", "one_triggers_one_cancels_other":
	default:
		return fmt.Errorf("%w: %q", errInvalidLinkedOrderType, o.LinkedOrderType)
	}
	switch o.TriggerFillCondition {
	case "", "first_hit", "complete_fill", "incremental":
	default:
		return fmt.Errorf("%w: %q", errInvalidTriggerFillCondition, o.TriggerFillCondition)
	}
	for i := range o.OTOCOConfig {
		if err := o.OTOCOConfig[i].validate(); err != nil {
			return fmt.Errorf("secondary order %d: %w", i, err)
		}
	}
	return nil
}

// validate checks a secondary order carries its required amount and direction and documented enum values
func (s *SecondaryOrder) validate() error {
	if s.Amount <= 0 {
		return fmt.Errorf("%w: %v", errInvalidAmount, s.Amount)
	}
	if s.Direction != DirectionBuy && s.Direction != DirectionSell {
		return fmt.Errorf("%w: %q", errInvalidDirection, s.Direction)
	}
	if err := validateOrderType(s.Type, s.Trigger, s.TriggerPrice); err != nil {
		return err
	}
	if err := validateOrderTimeInForce(s.TimeInForce); err != nil {
		return err
	}
	return validateOrderTrigger(s.Trigger)
}

// validate checks a cancel quotes request carries a documented cancel type and the field that type selects by
func (c *CancelQuotesRequest) validate() error {
	if c == nil {
		return common.ErrNilPointer
	}
	switch c.CancelType {
	case "delta", "all":
	case "quote_set_id":
		if c.QuoteSetID == "" {
			return errCancelQuoteSetIDRequired
		}
	case "instrument":
		if c.InstrumentName == "" {
			return errInstrumentNameRequired
		}
	case "instrument_kind":
		if c.Kind == "" {
			return fmt.Errorf("%w: kind is required to cancel by instrument kind", errInvalidKind)
		}
	case "currency":
		if c.Currency.IsEmpty() {
			return currency.ErrCurrencyCodeEmpty
		}
	case "currency_pair":
		if c.CurrencyPair == "" {
			return errCancelCurrencyPairRequired
		}
	default:
		return fmt.Errorf("%w: %q", errInvalidCancelQuotesType, c.CancelType)
	}
	return validateTradingKind(c.Kind, true)
}

// validateQuery checks an MMP configuration or status query names an index for an MMP group, as Deribit needs
func (m *MMPGroupRequest) validateQuery() error {
	if m == nil {
		return common.ErrNilPointer
	}
	if m.MMPGroup != "" && m.IndexName == "" {
		return fmt.Errorf("%w: MMP group %q needs its index", errMMPIndexNameRequired, m.MMPGroup)
	}
	return nil
}

// validate checks a mass quote request carries its quote ID, MMP group and quotes, each with an instrument and a side
func (m *MassQuoteRequest) validate() error {
	if m == nil {
		return common.ErrNilPointer
	}
	if m.QuoteID == "" {
		return errMassQuoteIDRequired
	}
	if m.MMPGroup == "" {
		return errMMPGroupRequired
	}
	if len(m.Quotes) == 0 {
		return errMassQuotesRequired
	}
	for i := range m.Quotes {
		if m.Quotes[i].InstrumentName == "" {
			return fmt.Errorf("%w: quote %d", errInstrumentNameRequired, i)
		}
		if m.Quotes[i].Ask == nil && m.Quotes[i].Bid == nil {
			return fmt.Errorf("%w: quote %d", errMassQuoteSideRequired, i)
		}
	}
	return nil
}

// validate checks a position move request carries both subaccounts and moves, each with an instrument and an amount
func (m *MovePositionsRequest) validate() error {
	if m == nil {
		return common.ErrNilPointer
	}
	if m.SourceUID == 0 {
		return errSourceUIDRequired
	}
	if m.TargetUID == 0 {
		return errTargetUIDRequired
	}
	if len(m.Trades) == 0 {
		return errPositionMovesRequired
	}
	for i := range m.Trades {
		if m.Trades[i].InstrumentName == "" {
			return fmt.Errorf("%w: position move %d", errInstrumentNameRequired, i)
		}
		if m.Trades[i].Amount <= 0 {
			return fmt.Errorf("%w: position move %d", errInvalidAmount, i)
		}
	}
	return nil
}

// validateOrderSize checks an order is sized by a positive amount, a positive number of contracts or both
func validateOrderSize(amount, contracts float64) error {
	if amount < 0 || contracts < 0 || (amount == 0 && contracts == 0) {
		return fmt.Errorf("%w: amount %v, contracts %v", errInvalidAmount, amount, contracts)
	}
	return nil
}

// validateOrderType checks an order type is empty or documented, and that a trigger order carries its trigger and, for
// a stop or take order, its trigger price
func validateOrderType(orderType, trigger string, triggerPrice float64) error {
	switch orderType {
	case "", OrderTypeLimit, OrderTypeMarket, OrderTypeMarketLimit:
		return nil
	case OrderTypeStopLimit, OrderTypeStopMarket, OrderTypeTakeLimit, OrderTypeTakeMarket, OrderTypeTrailingStop:
	default:
		return fmt.Errorf("%w: %q", errInvalidOrderType, orderType)
	}
	if trigger == "" {
		return fmt.Errorf("%w: %s", errOrderTriggerRequired, orderType)
	}
	if orderType != OrderTypeTrailingStop && triggerPrice == 0 {
		return fmt.Errorf("%w: %s", errOrderTriggerPriceRequired, orderType)
	}
	return nil
}

// validateOrderTimeInForce checks a time in force is empty or documented
func validateOrderTimeInForce(timeInForce string) error {
	switch timeInForce {
	case "", "good_til_cancelled", "good_til_day", "fill_or_kill", "immediate_or_cancel":
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidOrderTimeInForce, timeInForce)
}

// validateOrderTrigger checks a trigger is empty or documented
func validateOrderTrigger(trigger string) error {
	switch trigger {
	case "", "index_price", "mark_price", "last_price":
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidOrderTrigger, trigger)
}

// validateAdvancedOrderType checks an advanced option order type is empty, usd or implv
func validateAdvancedOrderType(advanced string) error {
	switch advanced {
	case "", "usd", "implv":
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidAdvancedOrderType, advanced)
}

// validateTradingKind checks an instrument kind is empty or one a trading method accepts; aggregates selects whether
// combo, which selects every combo, and any are accepted
func validateTradingKind(kind string, aggregates bool) error {
	switch kind {
	case "", KindFuture, KindOption, KindSpot, KindFutureCombo, KindOptionCombo:
		return nil
	case KindCombo, KindAny:
		if aggregates {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", errInvalidKind, kind)
}

// validateCancelOrderType checks the order type a cancel all method selects is empty or documented
func validateCancelOrderType(orderType string) error {
	switch orderType {
	case "", "all", OrderTypeLimit, "trigger_all", "stop", "take", OrderTypeTrailingStop:
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidCancelOrderType, orderType)
}

// validateOpenOrderType checks the order type an open orders method selects is empty or documented
func validateOpenOrderType(orderType string) error {
	switch orderType {
	case "", "all", OrderTypeLimit, "trigger_all", "stop_all", OrderTypeStopLimit, OrderTypeStopMarket, "take_all", OrderTypeTakeLimit, OrderTypeTakeMarket, "trailing_all", OrderTypeTrailingStop:
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidOpenOrderType, orderType)
}

// validateSettlementHistoryType checks a settlement event type is empty or documented
func validateSettlementHistoryType(settlementType string) error {
	switch settlementType {
	case "", "settlement", "delivery", "bankruptcy":
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidSettlementHistoryType, settlementType)
}

// validateUserTradesSorting checks a user trades sorting is empty or documented
func validateUserTradesSorting(sorting string) error {
	switch sorting {
	case "", "asc", "desc", "default":
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidUserTradesSorting, sorting)
}

// validateTradingCount checks a page size is within the 1000 items a trading history method returns at most; zero
// leaves Deribit's default
func validateTradingCount(count uint64) error {
	if count > maximumTradingCount {
		return fmt.Errorf("%w: %d exceeds %d", errInvalidCount, count, maximumTradingCount)
	}
	return nil
}

// isValidMMPDuration reports whether an MMP interval or frozen time is whole seconds up to an hour, as Deribit takes
// them in whole seconds
func isValidMMPDuration(d time.Duration) bool {
	return d >= 0 && d <= maximumMMPDuration && d%time.Second == 0
}

// userTradesTimeRange returns a trade query's bounds in Unix milliseconds, zero for an unset bound, checking the start
// does not follow the end
func userTradesTimeRange(start, end time.Time) (startTimestamp, endTimestamp uint64, err error) {
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return 0, 0, fmt.Errorf("%w: %s after %s", common.ErrStartAfterEnd, start, end)
	}
	return unixMilli(start), unixMilli(end), nil
}
