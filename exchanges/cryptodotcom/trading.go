package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/thrasher-corp/gocryptotrader/common"
)

const (
	maxClientOrderIDLength = 36
	maxOrderListLength     = 10
	// maxBatchCreates is the most orders a batch creates; the batch's docs page gives 10 in its limits and 1000 in its
	// params, and the stricter is kept
	maxBatchCreates          = 10
	maxTradingHistoryLimit   = 100
	maxSelfTradePreventionID = 32767
	maxAccountLeverage       = 100
)

var (
	errInvalidNotional                        = errors.New("invalid notional")
	errClientOrderIDTooLong                   = errors.New("client order ID longer than 36 characters")
	errInvalidExecutionInstruction            = errors.New("invalid execution instruction")
	errInvalidSpotMargin                      = errors.New("invalid spot margin mode")
	errInvalidSelfTradePreventionScope        = errors.New("invalid self-trade prevention scope")
	errInvalidSelfTradePreventionInstruction  = errors.New("invalid self-trade prevention instruction")
	errInvalidSelfTradePreventionID           = errors.New("invalid self-trade prevention ID")
	errInvalidIsolatedMarginTransferDirection = errors.New("invalid isolated margin transfer direction")
	errAccountSettingsRequired                = errors.New("account setting to change required")
	errBatchInstrumentMismatch                = errors.New("order instrument differs from the batch's")
)

// CreateOrder creates a LIMIT or MARKET order. Crypto.com confirms the request before its matching engine accepts the
// order, which the user.order channel reports; trigger orders are created with CreateAdvancedOrder, as
// private/create-order stopped taking them on 2025-12-17
func (e *Exchange) CreateOrder(ctx context.Context, r *CreateOrderRequest) (*OrderIDResponse, error) {
	params, err := newCreateOrderParams(r)
	if err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, createOrderEPL, "private/create-order", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// AmendOrder amends an order's price and quantity. Crypto.com cancels the order and creates its replacement, which
// loses the order's queue priority unless only the quantity is reduced, and confirms the request before the
// replacement is placed
func (e *Exchange) AmendOrder(ctx context.Context, r *AmendOrderRequest) (*OrderIDResponse, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/amend-order", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelExistingOrder cancels an order; Crypto.com confirms the request before the order is cancelled, which the
// user.order channel reports
func (e *Exchange) CancelExistingOrder(ctx context.Context, r *CancelOrderRequest) (*OrderIDResponse, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, cancelOrderEPL, "private/cancel-order", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelAllExistingOrders cancels the orders of an instrument, or of every instrument; Crypto.com confirms the request
// before the orders are cancelled
func (e *Exchange) CancelAllExistingOrders(ctx context.Context, r *CancelAllOrdersRequest) error {
	if err := r.validate(); err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, cancelAllOrdersEPL, "private/cancel-all-orders", r, nil)
}

// ClosePosition closes a position, wholly or partly, with a LIMIT or MARKET order; Crypto.com confirms the request
// before the order is placed
func (e *Exchange) ClosePosition(ctx context.Context, r *ClosePositionRequest) (*OrderIDResponse, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/close-position", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOpenOrders returns the open orders of an instrument, or of every instrument
func (e *Exchange) GetOpenOrders(ctx context.Context, r *OpenOrdersRequest) ([]OrderResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	var resp *dataResponse[OrderResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-open-orders", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetOrderDetail returns an order
func (e *Exchange) GetOrderDetail(ctx context.Context, r *OrderDetailRequest) (*OrderResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if r.OrderID == "" && r.ClientOrderID == "" {
		return nil, errOrderIDRequired
	}
	var resp *OrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, getOrderDetailEPL, "private/get-order-detail", r, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateOrderList creates a list of 1 to 10 LIMIT or MARKET orders and returns each order's outcome by its index in
// the list; an order Crypto.com refuses is reported in its entry's code and message rather than as an error
func (e *Exchange) CreateOrderList(ctx context.Context, orders []CreateOrderRequest) ([]OrderListEntryResponse, error) {
	params, err := newCreateOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/create-order-list", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelOrderList cancels a list of 1 to 10 orders and returns each cancel's outcome by its index in the list; a cancel
// Crypto.com refuses is reported in its entry's code and message rather than as an error
func (e *Exchange) CancelOrderList(ctx context.Context, orders []CancelOrderListEntryRequest) ([]OrderListEntryResponse, error) {
	params, err := newCancelOrderListParams(orders)
	if err != nil {
		return nil, err
	}
	var resp []OrderListEntryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/cancel-order-list", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ChangeAccountLeverage sets the account's maximum leverage
func (e *Exchange) ChangeAccountLeverage(ctx context.Context, r *ChangeAccountLeverageRequest) error {
	if r == nil {
		return common.ErrNilPointer
	}
	if r.AccountID == "" {
		return errAccountIDRequired
	}
	if r.Leverage == 0 || r.Leverage > maxAccountLeverage {
		return fmt.Errorf("%w: %d not within 1 to %d", errInvalidLeverage, r.Leverage, maxAccountLeverage)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/change-account-leverage", r, nil)
}

// ChangeAccountSettings changes the account's self-trade prevention settings or maximum leverage
func (e *Exchange) ChangeAccountSettings(ctx context.Context, r *ChangeAccountSettingsRequest) error {
	if r == nil {
		return common.ErrNilPointer
	}
	if *r == (ChangeAccountSettingsRequest{}) {
		return errAccountSettingsRequired
	}
	// The maximum leverage is the setting private/change-account-leverage changes, whose docs give its range
	if r.Leverage > maxAccountLeverage {
		return fmt.Errorf("%w: %d above %d", errInvalidLeverage, r.Leverage, maxAccountLeverage)
	}
	if err := validateSelfTradePrevention(r.SelfTradePreventionScope, r.SelfTradePreventionInstruction, r.SelfTradePreventionID); err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/change-account-settings", r, nil)
}

// GetAccountSettings returns the account's maximum leverage and self-trade prevention settings
func (e *Exchange) GetAccountSettings(ctx context.Context) ([]AccountSettingsResponse, error) {
	var resp []AccountSettingsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-account-settings", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CreateIsolatedMarginTransfer moves USD margin between the account and an isolated position
func (e *Exchange) CreateIsolatedMarginTransfer(ctx context.Context, r *CreateIsolatedMarginTransferRequest) error {
	if r == nil {
		return common.ErrNilPointer
	}
	if r.IsolationID == "" {
		return errIsolationIDRequired
	}
	switch r.Direction {
	case IsolatedMarginTransferDirectionCredit, IsolatedMarginTransferDirectionDebit:
	default:
		return fmt.Errorf("%w: %q", errInvalidIsolatedMarginTransferDirection, r.Direction)
	}
	if r.Amount <= 0 {
		return errInvalidAmount
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/create-isolated-margin-transfer", r, nil)
}

// ChangeIsolatedMarginLeverage sets an isolated position's maximum leverage
func (e *Exchange) ChangeIsolatedMarginLeverage(ctx context.Context, r *ChangeIsolatedMarginLeverageRequest) error {
	if r == nil {
		return common.ErrNilPointer
	}
	if r.IsolationID == "" {
		return errIsolationIDRequired
	}
	if r.Leverage <= 0 {
		return fmt.Errorf("%w: %v", errInvalidLeverage, r.Leverage)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/change-isolated-margin-leverage", r, nil)
}

// GetFeeRate returns the account's 30-day trading volume tiers and the spot and derivatives fee rates they give
func (e *Exchange) GetFeeRate(ctx context.Context) (*FeeRateResponse, error) {
	var resp *FeeRateResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-fee-rate", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetInstrumentFeeRate returns the account's fee rates for an instrument
func (e *Exchange) GetInstrumentFeeRate(ctx context.Context, instrumentName string) (*InstrumentFeeRateResponse, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *InstrumentFeeRateResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-instrument-fee-rate", &instrumentFeeRateParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserOrderHistory returns the account's past orders, for recovering what the user.order channel missed; Crypto.com
// keeps six months of history
func (e *Exchange) GetUserOrderHistory(ctx context.Context, r *UserOrderHistoryRequest) ([]OrderResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	params, err := newTradingHistoryParams(r.InstrumentName, "", r.StartTime, r.EndTime, r.Limit, r.IsolationID)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[OrderResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, getOrderHistoryEPL, "private/get-order-history", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetUserTrades returns the account's trades, for recovering what the user.trade channel missed; Crypto.com keeps six
// months of history
func (e *Exchange) GetUserTrades(ctx context.Context, r *UserTradesRequest) ([]UserTradeResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	params, err := newTradingHistoryParams(r.InstrumentName, "", r.StartTime, r.EndTime, r.Limit, r.IsolationID)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[UserTradeResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, getTradesEPL, "private/get-trades", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetTransactions returns the account's transactions, such as trades, fees, funding and settlements; Crypto.com keeps
// six months of history
func (e *Exchange) GetTransactions(ctx context.Context, r *TransactionsRequest) ([]TransactionResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	params, err := newTradingHistoryParams(r.InstrumentName, r.JournalType, r.StartTime, r.EndTime, r.Limit, r.IsolationID)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[TransactionResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/get-transactions", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// newCreateOrderParams validates an order and returns the params creating it
func newCreateOrderParams(r *CreateOrderRequest) (*createOrderParams, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if r.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	switch r.Side {
	case SideBuy, SideSell:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidSide, r.Side)
	}
	switch r.OrderType {
	case OrderTypeLimit:
		if r.Price <= 0 {
			return nil, fmt.Errorf("%w: a limit order needs one", errInvalidPrice)
		}
	case OrderTypeMarket:
		if r.Price != 0 {
			return nil, fmt.Errorf("%w: a market order takes none", errInvalidPrice)
		}
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidOrderType, r.OrderType)
	}
	switch {
	case r.Notional == 0:
		if r.Quantity <= 0 {
			return nil, errInvalidQuantity
		}
	case r.OrderType != OrderTypeMarket || r.Side != SideBuy:
		return nil, fmt.Errorf("%w: only a market buy spends a notional", errInvalidNotional)
	case r.Notional < 0:
		return nil, fmt.Errorf("%w: %v not greater than zero", errInvalidNotional, r.Notional)
	case r.Quantity != 0:
		return nil, fmt.Errorf("%w: a notional replaces the quantity", errInvalidNotional)
	}
	if err := validateOrderInstructions(r.ClientOrderID, r.TimeInForce, r.ExecutionInstructions, r.SelfTradePreventionScope, r.SelfTradePreventionInstruction, r.SelfTradePreventionID); err != nil {
		return nil, err
	}
	switch r.SpotMargin {
	case "", SpotMarginSpot, SpotMarginMargin:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidSpotMargin, r.SpotMargin)
	}
	if r.IsolatedMarginAmount < 0 {
		return nil, fmt.Errorf("%w: isolated margin amount %v", errInvalidAmount, r.IsolatedMarginAmount)
	}
	if r.Leverage < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidLeverage, r.Leverage)
	}
	return &createOrderParams{
		InstrumentName:                 r.InstrumentName,
		Side:                           r.Side,
		OrderType:                      r.OrderType,
		Price:                          r.Price,
		Quantity:                       r.Quantity,
		Notional:                       r.Notional,
		ClientOrderID:                  r.ClientOrderID,
		TimeInForce:                    r.TimeInForce,
		ExecutionInstructions:          r.ExecutionInstructions,
		SpotMargin:                     r.SpotMargin,
		FeeInstrumentName:              r.FeeInstrumentName.Upper().String(),
		SelfTradePreventionID:          r.SelfTradePreventionID,
		SelfTradePreventionInstruction: r.SelfTradePreventionInstruction,
		SelfTradePreventionScope:       r.SelfTradePreventionScope,
		IsolatedMarginAmount:           r.IsolatedMarginAmount,
		IsolationID:                    r.IsolationID,
		Leverage:                       r.Leverage,
	}, nil
}

// validateOrderInstructions checks what every new order takes alike, a regular or an advanced one: a client order ID of
// at most 36 characters, a known time in force, its execution instructions and its self-trade prevention settings
func validateOrderInstructions(clientOrderID, timeInForce string, instructions []string, stpScope, stpInstruction string, stpID uint64) error {
	if utf8.RuneCountInString(clientOrderID) > maxClientOrderIDLength {
		return fmt.Errorf("%w: %q", errClientOrderIDTooLong, clientOrderID)
	}
	switch timeInForce {
	case "", TimeInForceGoodTillCancel, TimeInForceImmediateOrCancel, TimeInForceFillOrKill:
	default:
		return fmt.Errorf("%w: %q", errInvalidTimeInForce, timeInForce)
	}
	if err := validateCreateExecutionInstructions(instructions, timeInForce); err != nil {
		return err
	}
	return validateSelfTradePrevention(stpScope, stpInstruction, stpID)
}

// validateCreateExecutionInstructions checks a new order's execution instructions, which cannot hold both post-only
// instructions, and a post-only order rests GOOD_TILL_CANCEL or Crypto.com rejects it
func validateCreateExecutionInstructions(instructions []string, timeInForce string) error {
	var postOnly, smartPostOnly bool
	for _, instruction := range instructions {
		switch instruction {
		case ExecutionInstructionPostOnly:
			postOnly = true
		case ExecutionInstructionSmartPostOnly:
			smartPostOnly = true
		case ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin, ExecutionInstructionMarginOrder:
		default:
			return fmt.Errorf("%w: %q", errInvalidExecutionInstruction, instruction)
		}
	}
	if postOnly && smartPostOnly {
		return fmt.Errorf("%w: %s and %s exclude each other", errInvalidExecutionInstruction, ExecutionInstructionPostOnly, ExecutionInstructionSmartPostOnly)
	}
	if postOnly && timeInForce != "" && timeInForce != TimeInForceGoodTillCancel {
		return fmt.Errorf("%w: a %s order rests %s, not %s", errInvalidTimeInForce, ExecutionInstructionPostOnly, TimeInForceGoodTillCancel, timeInForce)
	}
	return nil
}

// validateSelfTradePrevention checks an order's or the account's self-trade prevention settings: a scope that matches
// orders needs an instruction, the reset scope ignores the instruction and ID, and an ID only applies with a scope
func validateSelfTradePrevention(scope, instruction string, id uint64) error {
	switch scope {
	case "", SelfTradePreventionScopeMasterOrSubAccount, SelfTradePreventionScopeSubAccount, SelfTradePreventionScopeReset:
	default:
		return fmt.Errorf("%w: %q", errInvalidSelfTradePreventionScope, scope)
	}
	switch instruction {
	case "":
		if scope == SelfTradePreventionScopeMasterOrSubAccount || scope == SelfTradePreventionScopeSubAccount {
			return fmt.Errorf("%w: scope %s needs one", errInvalidSelfTradePreventionInstruction, scope)
		}
	case SelfTradePreventionCancelMaker, SelfTradePreventionCancelTaker, SelfTradePreventionCancelBoth:
	default:
		return fmt.Errorf("%w: %q", errInvalidSelfTradePreventionInstruction, instruction)
	}
	if id > maxSelfTradePreventionID {
		return fmt.Errorf("%w: %d above %d", errInvalidSelfTradePreventionID, id, maxSelfTradePreventionID)
	}
	if id != 0 && scope == "" {
		return fmt.Errorf("%w: %d needs a scope", errInvalidSelfTradePreventionID, id)
	}
	return nil
}

// validate checks an amend request
func (r *AmendOrderRequest) validate() error {
	if r == nil {
		return common.ErrNilPointer
	}
	if r.OrderID == "" && r.OriginalClientOrderID == "" {
		return errOrderIDRequired
	}
	if utf8.RuneCountInString(r.ClientOrderID) > maxClientOrderIDLength {
		return fmt.Errorf("%w: %q", errClientOrderIDTooLong, r.ClientOrderID)
	}
	if r.NewPrice <= 0 {
		return errInvalidPrice
	}
	if r.NewQuantity <= 0 {
		return errInvalidQuantity
	}
	return nil
}

// validate checks a cancel request, alone or in a batch
func (r *CancelOrderRequest) validate() error {
	if r == nil {
		return common.ErrNilPointer
	}
	if r.OrderID == "" && r.ClientOrderID == "" {
		return errOrderIDRequired
	}
	return nil
}

// validate checks a cancel-all request, whose type cannot be TRIGGER, which only the advanced order methods cancel
func (r *CancelAllOrdersRequest) validate() error {
	if r == nil {
		return common.ErrNilPointer
	}
	switch r.OrderType {
	case "", CancelOrderTypeLimit, CancelOrderTypeAll:
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidOrderType, r.OrderType)
	}
}

// validate checks a close request; a limit price is only taken by a LIMIT order, which needs one
func (r *ClosePositionRequest) validate() error {
	if r == nil {
		return common.ErrNilPointer
	}
	if r.InstrumentName == "" {
		return errInstrumentNameRequired
	}
	switch r.OrderType {
	case OrderTypeLimit:
		if r.Price <= 0 {
			return fmt.Errorf("%w: a limit order needs one", errInvalidPrice)
		}
	case OrderTypeMarket:
		if r.Price != 0 {
			return fmt.Errorf("%w: a market order takes none", errInvalidPrice)
		}
	default:
		return fmt.Errorf("%w: %q", errInvalidOrderType, r.OrderType)
	}
	if r.Quantity < 0 {
		return errInvalidQuantity
	}
	return nil
}

// newCreateOrderListParams validates an order list and returns the params creating it
func newCreateOrderListParams(orders []CreateOrderRequest) (*createOrderListParams, error) {
	if err := checkOrderListLength(len(orders), maxOrderListLength); err != nil {
		return nil, err
	}
	params := &createOrderListParams{ContingencyType: ContingencyTypeList, OrderList: make([]createOrderParams, len(orders))}
	for i := range orders {
		p, err := newCreateOrderParams(&orders[i])
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		params.OrderList[i] = *p
	}
	return params, nil
}

// newCancelOrderListParams validates a list of orders to cancel and returns the params cancelling them
func newCancelOrderListParams(orders []CancelOrderListEntryRequest) (*cancelOrderListParams, error) {
	if err := checkOrderListLength(len(orders), maxOrderListLength); err != nil {
		return nil, err
	}
	for i := range orders {
		if orders[i].OrderID == "" && orders[i].ClientOrderID == "" {
			return nil, fmt.Errorf("order %d: %w", i, errOrderIDRequired)
		}
	}
	return &cancelOrderListParams{ContingencyType: ContingencyTypeList, OrderList: orders}, nil
}

// checkOrderListLength checks a list holds at least one order and at most limit
func checkOrderListLength(length, limit int) error {
	if length == 0 {
		return errOrderListEmpty
	}
	if length > limit {
		return fmt.Errorf("%w: %d orders, above %d", errOrderListTooLong, length, limit)
	}
	return nil
}

// newTradingHistoryParams validates a history query's window and limit and returns its params, with the window in
// nanoseconds, which Crypto.com recommends for paging accurately
func newTradingHistoryParams(instrumentName, journalType string, start, end time.Time, limit uint64, isolationID string) (*tradingHistoryParams, error) {
	if !start.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(start, end); err != nil {
			return nil, err
		}
	}
	if limit > maxTradingHistoryLimit {
		return nil, fmt.Errorf("%w: limit %d above %d", errInvalidCount, limit, maxTradingHistoryLimit)
	}
	startTime, err := unixNanoParam(start)
	if err != nil {
		return nil, err
	}
	endTime, err := unixNanoParam(end)
	if err != nil {
		return nil, err
	}
	return &tradingHistoryParams{
		InstrumentName: instrumentName,
		JournalType:    journalType,
		StartTime:      startTime,
		EndTime:        endTime,
		Limit:          limit,
		IsolationID:    isolationID,
	}, nil
}

// newBatchOrderParams validates a batch and returns its params; its creates inherit its instrument, so each leaves its
// own out, and one naming another instrument is refused rather than placed on the batch's
func newBatchOrderParams(r *BatchOrderRequest) (*batchOrderParams, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if r.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	if len(r.Cancels) == 0 && len(r.Creates) == 0 {
		return nil, errOrderListEmpty
	}
	if len(r.Creates) > maxBatchCreates {
		return nil, fmt.Errorf("%w: %d creates, above %d", errOrderListTooLong, len(r.Creates), maxBatchCreates)
	}
	for i := range r.Cancels {
		if err := r.Cancels[i].validate(); err != nil {
			return nil, fmt.Errorf("cancel %d: %w", i, err)
		}
	}
	params := &batchOrderParams{
		InstrumentName:                        r.InstrumentName,
		Cancels:                               r.Cancels,
		SelfTradePreventionCancelType:         r.SelfTradePreventionCancelType,
		SelfTradePreventionAccountEnforcement: r.SelfTradePreventionAccountEnforcement,
		SelfTradePreventionGroup:              r.SelfTradePreventionGroup,
	}
	if len(r.Creates) != 0 {
		params.Creates = make([]createOrderParams, len(r.Creates))
	}
	for i := range r.Creates {
		create := r.Creates[i]
		if create.InstrumentName != "" && create.InstrumentName != r.InstrumentName {
			return nil, fmt.Errorf("create %d: %w: %s, not %s", i, errBatchInstrumentMismatch, create.InstrumentName, r.InstrumentName)
		}
		create.InstrumentName = r.InstrumentName
		p, err := newCreateOrderParams(&create)
		if err != nil {
			return nil, fmt.Errorf("create %d: %w", i, err)
		}
		p.InstrumentName = ""
		params.Creates[i] = *p
	}
	return params, nil
}
