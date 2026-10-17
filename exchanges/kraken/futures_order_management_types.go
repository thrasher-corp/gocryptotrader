package kraken

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var (
	errFuturesStopPriceRequired             = errors.New("stop price required")
	errFuturesTrailingStopDeviationRequired = errors.New("trailing stop maximum deviation and unit required")
	errFuturesInvalidTrailingStopDeviation  = errors.New("invalid trailing stop maximum deviation")
	errFuturesLimitPriceOffsetUnitRequired  = errors.New("limit price offset unit required")
	errFuturesClientOrderIDTooLong          = errors.New("client order ID too long")
	errFuturesInvalidTimeout                = errors.New("invalid timeout")
	errFuturesInvalidBatchInstruction       = errors.New("invalid batch instruction")
	errFuturesOrderTagRequired              = errors.New("order tag required")
	errFuturesTooManyEntries                = errors.New("too many entries")
)

// FuturesSendOrderRequest holds the parameters of Send order
type FuturesSendOrderRequest struct {
	Symbol currency.Pair
	// Side is buy or sell
	Side string
	// OrderType is lmt, post for a post-only limit order, ioc, mkt for an immediate-or-cancel order with 1% price
	// protection, stp, take_profit, trailing_stop or fok
	OrderType string
	// Size is a number of contracts, whose size differs between contracts
	Size float64
	// LimitPrice is also the worst price a stp or take_profit order fills at once triggered, without which it triggers a
	// market order. A trailing_stop order takes none
	LimitPrice float64
	// StopPrice is required for a stp or take_profit order; a trailing_stop order takes none
	StopPrice float64
	// ClientOrderID must be globally unique, and at most 100 characters
	ClientOrderID string
	// TriggerSignal is the price a stp, take_profit or trailing_stop order triggers on: mark, index or last
	TriggerSignal string
	ReduceOnly    bool
	// TrailingStopMaximumDeviation is how far a trailing_stop order's trigger price may trail the trigger signal, in
	// TrailingStopDeviationUnit, PERCENT or QUOTE_CURRENCY. Both are required for a trailing_stop order, and a
	// percentage must be from 0.1 to 50
	TrailingStopMaximumDeviation float64
	TrailingStopDeviationUnit    string
	// LimitPriceOffsetValue sets a trigger order's limit price relative to its stop price, in LimitPriceOffsetUnit,
	// QUOTE_CURRENCY or PERCENT. The value may be negative or zero, and both are sent once the unit is set
	LimitPriceOffsetValue float64
	LimitPriceOffsetUnit  string
	// Broker is the IIBAN of the broker the order is sent for, which only Kraken's pre-production environments take
	Broker string
	// ProcessBefore rejects the order unless Kraken processes it before then
	ProcessBefore time.Time
	// AlgorithmID identifies the algorithm sending the order
	AlgorithmID string
}

// FuturesSendOrderResponse holds the outcome of Send order
type FuturesSendOrderResponse struct {
	SendStatus FuturesSendOrderStatus `json:"sendStatus"`
	ServerTime time.Time              `json:"serverTime"`
}

// FuturesSendOrderStatus is the outcome of an order placement
type FuturesSendOrderStatus struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"cliOrdId"`
	// Status is placed, or why the order was not placed, such as postWouldExecute, iocWouldNotExecute or
	// wouldProcessAfterSpecifiedTime
	Status       string              `json:"status"`
	ReceivedTime time.Time           `json:"receivedTime"`
	OrderEvents  []FuturesOrderEvent `json:"orderEvents"`
}

// FuturesOrderEvent is an event an order request caused. Type is PLACE, CANCEL, EDIT, REJECT or EXECUTION, and an
// event about a trigger order carries OrderTrigger in place of Order:
//   - PLACE: Order and ReducedQuantity, or OrderTrigger
//   - CANCEL: OrderID, with Order or OrderTrigger when Kraken sends it
//   - EDIT: OldOrder, NewOrder and ReducedQuantity
//   - REJECT: OrderID and Reason, with Order or OrderTrigger when Kraken sends it
//   - EXECUTION: ExecutionID, ExecutionPrice, ExecutedAmount, OrderPriorExecution, OrderPriorEdit and
//     TakerReducedQuantity
type FuturesOrderEvent struct {
	Type string `json:"type"`
	// OrderID is the UID of the order or trigger order a CANCEL or REJECT event is about
	OrderID      string               `json:"uid"`
	Order        *FuturesOrder        `json:"order"`
	OrderTrigger *FuturesOrderTrigger `json:"orderTrigger"`
	OldOrder     *FuturesOrder        `json:"old"`
	NewOrder     *FuturesOrder        `json:"new"`
	// ReducedQuantity is the quantity a reduce-only order lost to fit the position when it was placed or edited
	ReducedQuantity float64 `json:"reducedQuantity"`
	// Reason is why an order was rejected, POST_WOULD_EXECUTE or IOC_WOULD_NOT_EXECUTE, or for a trigger order an
	// order error, such as INSUFFICIENT_MARGIN
	Reason         string  `json:"reason"`
	ExecutionID    string  `json:"executionId"`
	ExecutionPrice float64 `json:"price"`
	ExecutedAmount float64 `json:"amount"`
	// OrderPriorEdit is set when an edit caused the execution
	OrderPriorEdit      *FuturesOrder `json:"orderPriorEdit"`
	OrderPriorExecution *FuturesOrder `json:"orderPriorExecution"`
	// TakerReducedQuantity is the quantity a reduce-only order lost to fit the position before it executed
	TakerReducedQuantity float64 `json:"takerReducedQuantity"`
}

// FuturesOrder is an order as an order event reports it
type FuturesOrder struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"cliOrdId"`
	// OrderType is lmt, ioc, post, liquidation, assignment, stp, unwind, block or fok
	OrderType      string    `json:"type"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	Quantity       float64   `json:"quantity"`
	FilledQuantity float64   `json:"filled"`
	LimitPrice     float64   `json:"limitPrice"`
	ReduceOnly     bool      `json:"reduceOnly"`
	PlacedTime     time.Time `json:"timestamp"`
	LastUpdateTime time.Time `json:"lastUpdateTimestamp"`
	AlgorithmID    string    `json:"algoId"`
}

// FuturesOrderTrigger is a trigger order, such as a stop, as an order event reports it
type FuturesOrderTrigger struct {
	OrderID       string `json:"uid"`
	ClientOrderID string `json:"clientId"`
	// OrderType is lmt, ioc, post, liquidation, assignment, stp, unwind or fok
	OrderType    string  `json:"type"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	Quantity     float64 `json:"quantity"`
	LimitPrice   float64 `json:"limitPrice"`
	TriggerPrice float64 `json:"triggerPrice"`
	// TriggerSide is trigger_above or trigger_below
	TriggerSide string `json:"triggerSide"`
	// TriggerSignal is mark_price, last_price or spot_price
	TriggerSignal  string    `json:"triggerSignal"`
	ReduceOnly     bool      `json:"reduceOnly"`
	PlacedTime     time.Time `json:"timestamp"`
	LastUpdateTime time.Time `json:"lastUpdateTimestamp"`
	StartTime      time.Time `json:"startTime"`
}

// FuturesEditOrderRequest holds the parameters of Edit order. OrderID or ClientOrderID identifies the order, and only
// the values set are changed
type FuturesEditOrderRequest struct {
	OrderID       string
	ClientOrderID string
	Size          float64
	LimitPrice    float64
	// StopPrice is required to edit a stp order, whose LimitPrice is then required too
	StopPrice float64
	// TrailingStopMaximumDeviation is in TrailingStopDeviationUnit, PERCENT or QUOTE_CURRENCY; a percentage must be from
	// 0.1 to 50
	TrailingStopMaximumDeviation float64
	TrailingStopDeviationUnit    string
	// QuantityMode is RELATIVE, the default, which sets the open size to Size, or ABSOLUTE, which sets the order's
	// total size, past fills included
	QuantityMode  string
	ProcessBefore time.Time
	AlgorithmID   string
}

// FuturesEditOrderResponse holds the outcome of Edit order
type FuturesEditOrderResponse struct {
	EditStatus FuturesEditOrderStatus `json:"editStatus"`
	ServerTime time.Time              `json:"serverTime"`
}

// FuturesEditOrderStatus is the outcome of an order edit
type FuturesEditOrderStatus struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"cliOrdId"`
	// Status is edited, or why the order was not edited, such as orderForEditNotFound, wouldNotReducePosition or, with
	// 250 of the account's requests already held by maker protection, tooManyOrders
	Status       string              `json:"status"`
	ReceivedTime time.Time           `json:"receivedTime"`
	OrderEvents  []FuturesOrderEvent `json:"orderEvents"`
}

// FuturesCancelOrderRequest holds the parameters of Cancel order; OrderID or ClientOrderID identifies the order
type FuturesCancelOrderRequest struct {
	OrderID       string
	ClientOrderID string
	ProcessBefore time.Time
	AlgorithmID   string
}

// FuturesCancelOrderResponse holds the outcome of Cancel order
type FuturesCancelOrderResponse struct {
	CancelStatus FuturesCancelOrderStatus `json:"cancelStatus"`
	ServerTime   time.Time                `json:"serverTime"`
}

// FuturesCancelOrderStatus is the outcome of an order cancellation
type FuturesCancelOrderStatus struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"cliOrdId"`
	// Status is cancelled, which may leave part of the order filled, filled when the order had filled, or notFound
	Status       string              `json:"status"`
	ReceivedTime time.Time           `json:"receivedTime"`
	OrderEvents  []FuturesOrderEvent `json:"orderEvents"`
}

// FuturesCancelAllOrdersRequest holds the parameters of Cancel all orders
type FuturesCancelAllOrdersRequest struct {
	// Symbol limits the cancellation to a contract's orders; every open order is cancelled when it is empty
	Symbol      currency.Pair
	AlgorithmID string
}

// FuturesCancelAllOrdersResponse holds the outcome of Cancel all orders
type FuturesCancelAllOrdersResponse struct {
	CancelStatus FuturesCancelAllOrdersStatus `json:"cancelStatus"`
	ServerTime   time.Time                    `json:"serverTime"`
}

// FuturesCancelAllOrdersStatus is the outcome of cancelling every order. Orders held by maker protection are reported
// through their own placement or edit responses rather than here
type FuturesCancelAllOrdersStatus struct {
	// CancelOnly is the symbol whose orders were cancelled, or all
	CancelOnly string `json:"cancelOnly"`
	// Status is cancelled, or noOrdersToCancel
	Status          string                  `json:"status"`
	ReceivedTime    time.Time               `json:"receivedTime"`
	CancelledOrders []FuturesCancelledOrder `json:"cancelledOrders"`
	OrderEvents     []FuturesOrderEvent     `json:"orderEvents"`
}

// FuturesCancelledOrder identifies a cancelled order
type FuturesCancelledOrder struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"cliOrdId"`
}

// FuturesCancelAllOrdersAfterRequest holds the parameters of Dead man's switch
type FuturesCancelAllOrdersAfterRequest struct {
	// Timeout is the whole number of seconds, up to 4294967295, after which every order is cancelled; zero deactivates
	// the switch
	Timeout     time.Duration
	AlgorithmID string
}

// FuturesCancelAllOrdersAfterResponse holds the state of the dead man's switch
type FuturesCancelAllOrdersAfterResponse struct {
	Status     FuturesDeadMansSwitchStatus `json:"status"`
	ServerTime time.Time                   `json:"serverTime"`
}

// FuturesDeadMansSwitchStatus holds when Kraken received the request and when the switch will cancel every order
type FuturesDeadMansSwitchStatus struct {
	CurrentTime time.Time `json:"currentTime"`
	// TriggerTime is zero when the switch is deactivated
	TriggerTime time.Time `json:"triggerTime"`
}

// UnmarshalJSON decodes the switch's status, whose trigger time Kraken sends as "0" when the switch is deactivated
func (s *FuturesDeadMansSwitchStatus) UnmarshalJSON(data []byte) error {
	type alias FuturesDeadMansSwitchStatus
	status := struct {
		*alias
		TriggerTime string `json:"triggerTime"`
	}{alias: (*alias)(s)}
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	if status.TriggerTime == "0" {
		s.TriggerTime = time.Time{}
		return nil
	}
	triggerTime, err := time.Parse(time.RFC3339, status.TriggerTime)
	if err != nil {
		return fmt.Errorf("error parsing dead man's switch trigger time: %w", err)
	}
	s.TriggerTime = triggerTime
	return nil
}

// FuturesBatchOrderRequest holds the parameters of Batch order management
type FuturesBatchOrderRequest struct {
	// Instructions are applied in the order given, up to 500
	Instructions []FuturesBatchInstruction
	// Broker is the IIBAN of the broker the orders are sent for, which only Kraken's pre-production environments take
	Broker        string
	ProcessBefore time.Time
	AlgorithmID   string
}

// FuturesBatchInstruction is an instruction of a batch, setting exactly one of Send, Edit and Cancel
type FuturesBatchInstruction struct {
	Send   *FuturesBatchSendInstruction
	Edit   *FuturesBatchEditInstruction
	Cancel *FuturesBatchCancelInstruction
}

// FuturesBatchSendInstruction places an order within a batch
type FuturesBatchSendInstruction struct {
	// OrderTag is required, and is echoed by the instruction's result to map the two
	OrderTag string
	Symbol   currency.Pair
	// Side is buy or sell
	Side string
	// OrderType is lmt, post, ioc, mkt, stp, take_profit, trailing_stop or fok
	OrderType  string
	Size       float64
	LimitPrice float64
	// StopPrice is required for a stp or take_profit order
	StopPrice     float64
	ClientOrderID string
	// TriggerSignal is mark, index or last
	TriggerSignal string
	ReduceOnly    bool
	// TrailingStopMaximumDeviation and TrailingStopDeviationUnit, PERCENT or QUOTE_CURRENCY, are required for a
	// trailing_stop order
	TrailingStopMaximumDeviation float64
	TrailingStopDeviationUnit    string
}

// FuturesBatchEditInstruction edits an order within a batch. OrderID or ClientOrderID identifies the order, and only
// the values set are changed
type FuturesBatchEditInstruction struct {
	OrderID                      string
	ClientOrderID                string
	Size                         float64
	LimitPrice                   float64
	StopPrice                    float64
	TrailingStopMaximumDeviation float64
	TrailingStopDeviationUnit    string
	// QuantityMode is RELATIVE, the default, or ABSOLUTE
	QuantityMode string
}

// FuturesBatchCancelInstruction cancels an order within a batch; OrderID or ClientOrderID identifies the order
type FuturesBatchCancelInstruction struct {
	OrderID       string
	ClientOrderID string
}

// futuresBatchOrderJSON is the document Batch order management takes in its json parameter
type futuresBatchOrderJSON struct {
	BatchOrder []futuresBatchInstructionJSON `json:"batchOrder"`
}

// futuresBatchInstructionJSON is a batch instruction as sent, Order naming its kind
type futuresBatchInstructionJSON struct {
	Order                        string  `json:"order"`
	OrderTag                     string  `json:"order_tag,omitempty"`
	OrderType                    string  `json:"orderType,omitempty"`
	Symbol                       string  `json:"symbol,omitempty"`
	Side                         string  `json:"side,omitempty"`
	Size                         float64 `json:"size,omitempty"`
	LimitPrice                   float64 `json:"limitPrice,omitempty"`
	StopPrice                    float64 `json:"stopPrice,omitempty"`
	OrderID                      string  `json:"order_id,omitempty"`
	ClientOrderID                string  `json:"cliOrdId,omitempty"`
	TriggerSignal                string  `json:"triggerSignal,omitempty"`
	ReduceOnly                   bool    `json:"reduceOnly,omitempty"`
	TrailingStopMaximumDeviation float64 `json:"trailingStopMaxDeviation,omitempty"`
	TrailingStopDeviationUnit    string  `json:"trailingStopDeviationUnit,omitempty"`
	QuantityMode                 string  `json:"qtyMode,omitempty"`
}

// FuturesBatchOrderResponse holds the outcome of Batch order management
type FuturesBatchOrderResponse struct {
	BatchStatus []FuturesBatchInstructionResult `json:"batchStatus"`
	ServerTime  time.Time                       `json:"serverTime"`
}

// FuturesBatchInstructionResult is the outcome of a batch instruction
type FuturesBatchInstructionResult struct {
	// OrderTag echoes a send instruction's OrderTag
	OrderTag      string `json:"order_tag"`
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"cliOrdId"`
	// Status is placed, edited or cancelled, or why the instruction failed, such as invalidSize or tooManyOrders
	Status       string              `json:"status"`
	ReceivedTime time.Time           `json:"dateTimeReceived"`
	OrderEvents  []FuturesOrderEvent `json:"orderEvents"`
}

// FuturesOpenOrdersResponse holds the open orders, newest first
type FuturesOpenOrdersResponse struct {
	OpenOrders []FuturesOpenOrder `json:"openOrders"`
	ServerTime time.Time          `json:"serverTime"`
}

// FuturesOpenOrder is an open order
type FuturesOpenOrder struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"cliOrdId"`
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	// OrderType is lmt, stp or take_profit
	OrderType string `json:"orderType"`
	// Status is untouched or partiallyFilled
	Status     string  `json:"status"`
	LimitPrice float64 `json:"limitPrice"`
	// StopPrice is set for a stop or take profit order
	StopPrice    float64 `json:"stopPrice"`
	FilledSize   float64 `json:"filledSize"`
	UnfilledSize float64 `json:"unfilledSize"`
	ReduceOnly   bool    `json:"reduceOnly"`
	// TriggerSignal is the price a stop or take profit order triggers on: mark, last or spot
	TriggerSignal  string    `json:"triggerSignal"`
	ReceivedTime   time.Time `json:"receivedTime"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
	AlgorithmID    string    `json:"algoId"`
}

// FuturesOrdersStatusRequest holds the parameters of Get Specific Orders' Status, which takes at least one ID
type FuturesOrdersStatusRequest struct {
	OrderIDs       []string
	ClientOrderIDs []string
}

// FuturesOrdersStatusResponse holds the status of the orders requested
type FuturesOrdersStatusResponse struct {
	Orders     []FuturesOrderStatusDetails `json:"orders"`
	ServerTime time.Time                   `json:"serverTime"`
}

// FuturesOrderStatusDetails holds an order's status
type FuturesOrderStatusDetails struct {
	Order FuturesCachedOrder `json:"order"`
	// Status is ENTERED_BOOK, FULLY_EXECUTED, REJECTED, CANCELLED, TRIGGER_PLACED or TRIGGER_ACTIVATION_FAILURE
	Status string `json:"status"`
	// UpdateReason is why the order last changed, such as NEW_USER_ORDER, PARTIAL_FILL or CANCELLED_BY_USER. On a market
	// with maker protection a resting order cancelled so that a released order of the account's own could trade
	// reports CANCELLED_BY_SELF_TRADE
	UpdateReason string `json:"updateReason"`
	// Error is why the order failed, such as INSUFFICIENT_MARGIN
	Error string `json:"error"`
}

// FuturesCachedOrder is an order or trigger order, which Kraken reports while it is open and for 5 seconds after it
// closes
type FuturesCachedOrder struct {
	// Type is ORDER or TRIGGER_ORDER
	Type           string    `json:"type"`
	OrderID        string    `json:"orderId"`
	ClientOrderID  string    `json:"cliOrdId"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	Quantity       float64   `json:"quantity"`
	FilledQuantity float64   `json:"filled"`
	LimitPrice     float64   `json:"limitPrice"`
	ReduceOnly     bool      `json:"reduceOnly"`
	PlacedTime     time.Time `json:"timestamp"`
	LastUpdateTime time.Time `json:"lastUpdateTimestamp"`
	AlgorithmID    string    `json:"algoId"`
	// PriceTriggerOptions is set for a trigger order
	PriceTriggerOptions *FuturesPriceTriggerOptions `json:"priceTriggerOptions"`
	TriggerTime         time.Time                   `json:"triggerTime"`
}

// FuturesPriceTriggerOptions holds when a trigger order triggers
type FuturesPriceTriggerOptions struct {
	TriggerPrice float64 `json:"triggerPrice"`
	// TriggerSide is TRIGGER_ABOVE or TRIGGER_BELOW
	TriggerSide string `json:"triggerSide"`
	// TriggerSignal is MARK_PRICE, LAST_PRICE or SPOT_PRICE
	TriggerSignal string `json:"triggerSignal"`
	// TrailingStopOptions is set for a trailing stop
	TrailingStopOptions *FuturesTrailingStopOptions `json:"trailingStopOptions"`
}

// FuturesTrailingStopOptions holds how far a trailing stop's trigger price may trail its trigger signal
type FuturesTrailingStopOptions struct {
	MaximumDeviation float64 `json:"maxDeviation"`
	// Unit is PERCENT or QUOTE_CURRENCY
	Unit string `json:"unit"`
}
