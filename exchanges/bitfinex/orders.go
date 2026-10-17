package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// maxOrderOperations is the most operations an OrderMultiOp request may send
const maxOrderOperations = 75

var (
	errInvalidOrderType        = errors.New("invalid order type")
	errOrdersToCancelRequired  = errors.New("orders to cancel required: order IDs, group IDs, client order IDs or all")
	errOrderOperationsRequired = errors.New("order operations required")
	errTooManyOrderOperations  = errors.New("too many order operations")
	errInvalidLedgerCategory   = errors.New("invalid ledger category")
)

// GetOpenOrders returns the account's active orders, or those the request filters by ID, group ID or client order ID
func (e *Exchange) GetOpenOrders(ctx context.Context, req *OpenOrdersRequest) ([]Order, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := newOpenOrdersBody(req.IDs, req.GroupID, req.ClientOrderID, req.ClientOrderIDDate)
	if err != nil {
		return nil, err
	}
	var resp []Order
	return resp, e.SendAuthenticatedHTTPRequest(ctx, retrieveOrdersEPL, "auth/r/orders", body, &resp)
}

// GetOpenOrdersBySymbol returns the account's active orders on a trading pair, or those the request filters by ID,
// group ID or client order ID
func (e *Exchange) GetOpenOrdersBySymbol(ctx context.Context, req *OpenOrdersBySymbolRequest) ([]Order, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, fmt.Errorf("%w: %q", errTradingSymbolRequired, req.Symbol)
	}
	body, err := newOpenOrdersBody(req.IDs, req.GroupID, req.ClientOrderID, req.ClientOrderIDDate)
	if err != nil {
		return nil, err
	}
	var resp []Order
	return resp, e.SendAuthenticatedHTTPRequest(ctx, retrieveOrdersBySymbolEPL, "auth/r/orders/"+req.Symbol, body, &resp)
}

// PlaceOrder submits an order on a trading pair, returning a notification whose Data holds the new order
func (e *Exchange) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*Notification[[]Order], error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	return sendNotificationRequest[[]Order](ctx, e, submitOrderEPL, "auth/w/order/submit", body)
}

// UpdateOrder updates a margin, exchange or derivative order, returning a notification whose Data holds the order
func (e *Exchange) UpdateOrder(ctx context.Context, req *UpdateOrderRequest) (*Notification[Order], error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	return sendNotificationRequest[Order](ctx, e, updateOrderEPL, "auth/w/order/update", body)
}

// CancelExistingOrder submits a margin, exchange or derivative order for cancellation, which Bitfinex confirms
// afterwards, returning a notification whose Data holds the order
func (e *Exchange) CancelExistingOrder(ctx context.Context, req *CancelExistingOrderRequest) (*Notification[Order], error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	return sendNotificationRequest[Order](ctx, e, cancelOrderEPL, "auth/w/order/cancel", body)
}

// CancelOrders submits orders for cancellation, by their IDs, group IDs or client order IDs, or every open order,
// trading and derivative alike, returning a notification whose Data holds the orders
func (e *Exchange) CancelOrders(ctx context.Context, req *CancelOrdersRequest) (*Notification[[]Order], error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	return sendNotificationRequest[[]Order](ctx, e, cancelOrdersEPL, "auth/w/order/cancel/multi", body)
}

// OrderMultiOp sends up to 75 order operations in one request, returning a notification whose Data holds each
// operation's notification in order; each reports its operation's own outcome, which its Err checks
func (e *Exchange) OrderMultiOp(ctx context.Context, ops []OrderOperation) (*Notification[[]Notification[OrderOperationOrders]], error) {
	if len(ops) == 0 {
		return nil, errOrderOperationsRequired
	}
	if len(ops) > maxOrderOperations {
		return nil, fmt.Errorf("%w: %d, at most %d", errTooManyOrderOperations, len(ops), maxOrderOperations)
	}
	body := &orderMultiOpBody{Ops: make([][2]any, len(ops))}
	for i, op := range ops {
		// A nil operation, typed or not, has no body to send
		if err := common.NilGuard(op); err != nil {
			return nil, fmt.Errorf("operation %d: %w", i, err)
		}
		abbreviation, opBody, err := op.operation()
		if err != nil {
			return nil, fmt.Errorf("operation %d: %w", i, err)
		}
		body.Ops[i] = [2]any{abbreviation, opBody}
	}
	return sendNotificationRequest[[]Notification[OrderOperationOrders]](ctx, e, orderMultiOpEPL, "auth/w/order/multi", body)
}

// GetOrdersHistory returns the account's closed and cancelled orders from up to two weeks ago
func (e *Exchange) GetOrdersHistory(ctx context.Context, req *OrdersHistoryRequest) ([]Order, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []Order
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ordersHistoryEPL, "auth/r/orders/hist", &ordersHistoryBody{historyBody: history, IDs: req.IDs}, &resp)
}

// GetOrdersHistoryBySymbol returns the account's closed and cancelled orders on a trading pair from up to two weeks
// ago
func (e *Exchange) GetOrdersHistoryBySymbol(ctx context.Context, req *OrdersHistoryBySymbolRequest) ([]Order, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, fmt.Errorf("%w: %q", errTradingSymbolRequired, req.Symbol)
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []Order
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ordersHistoryBySymbolEPL, "auth/r/orders/"+req.Symbol+"/hist", &ordersHistoryBody{historyBody: history, IDs: req.IDs}, &resp)
}

// GetOrderTrades returns the trades an order made, up to 10 days old; the trades leave OrderPrice unset
func (e *Exchange) GetOrderTrades(ctx context.Context, req *OrderTradesRequest) ([]AccountTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, fmt.Errorf("%w: %q", errTradingSymbolRequired, req.Symbol)
	}
	if req.OrderID == 0 {
		return nil, errOrderIDRequired
	}
	var resp []AccountTrade
	return resp, e.SendAuthenticatedHTTPRequest(ctx, orderTradesEPL, "auth/r/order/"+req.Symbol+":"+strconv.FormatUint(req.OrderID, 10)+"/trades", nil, &resp)
}

// GetAccountTrades returns the account's trades, the most recent by default; trades from before March 2020 leave
// OrderType unset
func (e *Exchange) GetAccountTrades(ctx context.Context, req *AccountTradesRequest) ([]AccountTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, req.Sort)
	if err != nil {
		return nil, err
	}
	var resp []AccountTrade
	return resp, e.SendAuthenticatedHTTPRequest(ctx, tradesEPL, "auth/r/trades/hist", &body, &resp)
}

// GetAccountTradesBySymbol returns the account's trades on a trading pair, the most recent by default; trades from
// before March 2020 leave OrderType unset
func (e *Exchange) GetAccountTradesBySymbol(ctx context.Context, req *AccountTradesBySymbolRequest) ([]AccountTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, fmt.Errorf("%w: %q", errTradingSymbolRequired, req.Symbol)
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, req.Sort)
	if err != nil {
		return nil, err
	}
	var resp []AccountTrade
	return resp, e.SendAuthenticatedHTTPRequest(ctx, tradesBySymbolEPL, "auth/r/trades/"+req.Symbol+"/hist", &body, &resp)
}

// GetOTCOrdersHistory returns the account's over the counter orders, on a trading pair or on every pair
func (e *Exchange) GetOTCOrdersHistory(ctx context.Context, req *OTCOrdersHistoryRequest) ([]OTCOrder, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	path := "auth/r/orders/otc/hist"
	if req.Symbol != "" {
		if !isTradingSymbol(req.Symbol) {
			return nil, fmt.Errorf("%w: %q", errTradingSymbolRequired, req.Symbol)
		}
		path = "auth/r/orders/otc/" + req.Symbol + "/hist"
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []OTCOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, otcOrdersHistoryEPL, path, &ordersHistoryBody{historyBody: history, IDs: req.IDs}, &resp)
}

// GetLedgers returns the account's ledger entries, of a currency or of every currency, the most recent by default and
// up to six years old
func (e *Exchange) GetLedgers(ctx context.Context, req *LedgersRequest) ([]LedgerEntry, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category != 0 && !isLedgerCategory(req.Category) {
		return nil, fmt.Errorf("%w: %d", errInvalidLedgerCategory, req.Category)
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	path := "auth/r/ledgers/hist"
	if !req.Currency.IsEmpty() {
		path = "auth/r/ledgers/" + req.Currency.Upper().String() + "/hist"
	}
	var resp []LedgerEntry
	return resp, e.SendAuthenticatedHTTPRequest(ctx, ledgersEPL, path, &ledgersBody{historyBody: history, Category: req.Category, Wallet: req.Wallet}, &resp)
}

// newOpenOrdersBody returns the filters of an active orders request, erroring for a client order ID without its date
func newOpenOrdersBody(ids []uint64, groupID, clientOrderID uint64, clientOrderIDDate time.Time) (*openOrdersBody, error) {
	if clientOrderID != 0 && clientOrderIDDate.IsZero() {
		return nil, errClientOrderIDDateUnset
	}
	return &openOrdersBody{
		IDs:               ids,
		GroupID:           groupID,
		ClientOrderID:     clientOrderID,
		ClientOrderIDDate: orderTimeParam(clientOrderIDDate, cidDateLayout),
	}, nil
}

// body validates the order and returns the body its submission sends
func (r *PlaceOrderRequest) body() (*placeOrderBody, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !isOrderType(r.Type) {
		return nil, fmt.Errorf("%w: %q", errInvalidOrderType, r.Type)
	}
	if !isTradingSymbol(r.Symbol) {
		return nil, fmt.Errorf("%w: %q", errTradingSymbolRequired, r.Symbol)
	}
	if r.Amount == 0 {
		return nil, fmt.Errorf("%w: an order's amount must not be zero", errInvalidAmount)
	}
	if r.Price < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPrice, r.Price)
	}
	return &placeOrderBody{
		Type:                r.Type,
		Symbol:              r.Symbol,
		Amount:              strconv.FormatFloat(r.Amount, 'f', -1, 64),
		Price:               strconv.FormatFloat(r.Price, 'f', -1, 64),
		Leverage:            r.Leverage,
		TrailingPrice:       orderDecimalParam(r.TrailingPrice),
		AuxiliaryLimitPrice: orderDecimalParam(r.AuxiliaryLimitPrice),
		OCOStopPrice:        orderDecimalParam(r.OCOStopPrice),
		GroupID:             r.GroupID,
		ClientOrderID:       r.ClientOrderID,
		Flags:               r.Flags,
		TimeInForce:         orderTimeParam(r.TimeInForce, tifLayout),
		Meta:                r.Meta.body(),
	}, nil
}

// operation returns the order submission OrderMultiOp sends
func (r *PlaceOrderRequest) operation() (abbreviation string, body any, err error) {
	body, err = r.body()
	return "on", body, err
}

// body validates the update and returns the body it sends
func (r *UpdateOrderRequest) body() (*updateOrderBody, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.ID == 0 {
		return nil, errOrderIDRequired
	}
	if r.Price < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPrice, r.Price)
	}
	return &updateOrderBody{
		ID:                  r.ID,
		Amount:              orderDecimalParam(r.Amount),
		Price:               orderDecimalParam(r.Price),
		Delta:               orderDecimalParam(r.Delta),
		ClientOrderID:       r.ClientOrderID,
		ClientOrderIDDate:   orderTimeParam(r.ClientOrderIDDate, cidDateLayout),
		GroupID:             r.GroupID,
		Flags:               r.Flags,
		Leverage:            r.Leverage,
		TrailingPrice:       orderDecimalParam(r.TrailingPrice),
		AuxiliaryLimitPrice: orderDecimalParam(r.AuxiliaryLimitPrice),
		TimeInForce:         orderTimeParam(r.TimeInForce, tifLayout),
		Meta:                r.Meta.body(),
	}, nil
}

// operation returns the order update OrderMultiOp sends
func (r *UpdateOrderRequest) operation() (abbreviation string, body any, err error) {
	body, err = r.body()
	return "ou", body, err
}

// body validates the cancellation and returns the body it sends
func (r *CancelExistingOrderRequest) body() (*cancelOrderBody, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.ID == 0 && r.ClientOrderID == 0 {
		return nil, fmt.Errorf("%w: set an order ID, or a client order ID with its date", errOrderIDRequired)
	}
	if r.ClientOrderID != 0 && r.ClientOrderIDDate.IsZero() {
		return nil, errClientOrderIDDateUnset
	}
	return &cancelOrderBody{
		ID:                r.ID,
		ClientOrderID:     r.ClientOrderID,
		ClientOrderIDDate: orderTimeParam(r.ClientOrderIDDate, cidDateLayout),
	}, nil
}

// operation returns the order cancellation OrderMultiOp sends
func (r *CancelExistingOrderRequest) operation() (abbreviation string, body any, err error) {
	body, err = r.body()
	return "oc", body, err
}

// body validates the cancellations and returns the body they send
func (r *CancelOrdersRequest) body() (*cancelOrdersBody, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if len(r.IDs) == 0 && len(r.GroupIDs) == 0 && len(r.ClientOrderIDs) == 0 && !r.All {
		return nil, errOrdersToCancelRequired
	}
	body := &cancelOrdersBody{IDs: r.IDs, GroupIDs: r.GroupIDs, ClientOrderIDs: make([][2]any, len(r.ClientOrderIDs))}
	for i, c := range r.ClientOrderIDs {
		if c.Date.IsZero() {
			return nil, fmt.Errorf("%w: client order ID %d", errClientOrderIDDateUnset, c.ID)
		}
		body.ClientOrderIDs[i] = [2]any{c.ID, c.Date.UTC().Format(cidDateLayout)}
	}
	if r.All {
		body.All = 1
	}
	return body, nil
}

// operation returns the multiple order cancellation OrderMultiOp sends
func (r *CancelOrdersRequest) operation() (abbreviation string, body any, err error) {
	body, err = r.body()
	return "oc_multi", body, err
}

// body returns the meta object an order placement or update sends, or nil without meta data
func (m *OrderMeta) body() *orderMetaBody {
	if m == nil {
		return nil
	}
	body := &orderMetaBody{AffiliateCode: m.AffiliateCode}
	if m.MakeVisible {
		body.MakeVisible = 1
	}
	if m.ProtectSelfMatch {
		body.ProtectSelfMatch = 1
	}
	return body
}

// isOrderType reports whether orderType is one of the order types, with or without ExchangeOrderTypePrefix
func isOrderType(orderType string) bool {
	switch strings.TrimPrefix(orderType, ExchangeOrderTypePrefix) {
	case OrderTypeLimit, OrderTypeMarket, OrderTypeStop, OrderTypeStopLimit, OrderTypeTrailingStop, OrderTypeFillOrKill, OrderTypeImmediateOrCancel:
		return true
	}
	return false
}

// isLedgerCategory reports whether category is one of the ledger categories GetLedgers filters by
func isLedgerCategory(category uint64) bool {
	switch category {
	case LedgerCategoryExchange, LedgerCategoryPositionChange, LedgerCategoryPositionClaim, LedgerCategoryPositionTransfer,
		LedgerCategoryPositionSwap, LedgerCategoryPositionFundingCost, LedgerCategoryMarginPayment,
		LedgerCategoryDerivativesFunding, LedgerCategorySettlement, LedgerCategoryTransfer, LedgerCategoryDeposit,
		LedgerCategoryWithdrawal, LedgerCategoryCancelledWithdrawal, LedgerCategoryTradingFee, LedgerCategoryTradingRebate,
		LedgerCategoryHiddenOrderFee, LedgerCategoryOTCTradeFee, LedgerCategorySwapFee, LedgerCategoryClaimingFee,
		LedgerCategoryUsedMarginFundingCharge, LedgerCategoryUnusedMarginFundingFee, LedgerCategoryEarnedFee,
		LedgerCategoryETHFXLoyaltyFee, LedgerCategoryDepositFee, LedgerCategoryWithdrawalFee,
		LedgerCategoryWithdrawalExpressFee, LedgerCategoryMinerFee, LedgerCategoryStakingPayment, LedgerCategoryAdjustment,
		LedgerCategoryExpense, LedgerCategoryCurrencyConversionFee, LedgerCategoryMonthlyProfitPayment,
		LedgerCategoryLosses:
		return true
	}
	return false
}

// orderDecimalParam formats an optional amount or price as Bitfinex's decimal strings, never in exponent notation;
// zero formats as empty, which omitempty leaves unsent
func orderDecimalParam(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// orderTimeParam formats an optional time with layout in UTC, which Bitfinex reads the order times in; the zero time
// formats as empty, which omitempty leaves unsent
func orderTimeParam(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(layout)
}
