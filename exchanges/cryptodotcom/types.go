package cryptodotcom

import (
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Instrument types
const (
	InstrumentTypeCurrencyPair  = "CCY_PAIR"
	InstrumentTypePerpetualSwap = "PERPETUAL_SWAP"
	InstrumentTypeFuture        = "FUTURE"
	// InstrumentTypeCurrency is a deposit instrument, a currency rather than a pair
	InstrumentTypeCurrency = "CCY"
)

// Order sides
const (
	SideBuy  = "BUY"
	SideSell = "SELL"
)

// Order types; the trigger types are placed with the advanced order methods
const (
	OrderTypeLimit           = "LIMIT"
	OrderTypeMarket          = "MARKET"
	OrderTypeStopLoss        = "STOP_LOSS"
	OrderTypeStopLimit       = "STOP_LIMIT"
	OrderTypeTakeProfit      = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit = "TAKE_PROFIT_LIMIT"
)

// Times in force
const (
	TimeInForceGoodTillCancel    = "GOOD_TILL_CANCEL"
	TimeInForceImmediateOrCancel = "IMMEDIATE_OR_CANCEL"
	TimeInForceFillOrKill        = "FILL_OR_KILL"
)

// Execution instructions; ExecutionInstructionLiquidation is only reported, on orders the exchange places to liquidate
// a position
const (
	ExecutionInstructionPostOnly       = "POST_ONLY"
	ExecutionInstructionSmartPostOnly  = "SMART_POST_ONLY"
	ExecutionInstructionReduceOnly     = "REDUCE_ONLY"
	ExecutionInstructionIsolatedMargin = "ISOLATED_MARGIN"
	ExecutionInstructionMarginOrder    = "MARGIN_ORDER"
	ExecutionInstructionLiquidation    = "LIQUIDATION"
)

// Order statuses
const (
	OrderStatusPending  = "PENDING"
	OrderStatusNew      = "NEW"
	OrderStatusActive   = "ACTIVE"
	OrderStatusFilled   = "FILLED"
	OrderStatusCanceled = "CANCELED"
	OrderStatusRejected = "REJECTED"
	OrderStatusExpired  = "EXPIRED"
)

// Reference price types, the price a trigger order's reference price is compared with
const (
	ReferencePriceTypeMarkPrice  = "MARK_PRICE"
	ReferencePriceTypeIndexPrice = "INDEX_PRICE"
	ReferencePriceTypeLastPrice  = "LAST_PRICE"
)

// Spot margin modes, whether a spot order trades on margin
const (
	SpotMarginSpot   = "SPOT"
	SpotMarginMargin = "MARGIN"
)

// Self-trade prevention instructions: cancel the maker, the taker or both
const (
	SelfTradePreventionCancelMaker = "M"
	SelfTradePreventionCancelTaker = "T"
	SelfTradePreventionCancelBoth  = "B"
)

// Self-trade prevention scopes: match orders of the master and sub accounts, or of the sub account only;
// SelfTradePreventionScopeReset resets every self-trade prevention setting to its default
const (
	SelfTradePreventionScopeMasterOrSubAccount = "M"
	SelfTradePreventionScopeSubAccount         = "S"
	SelfTradePreventionScopeReset              = "D"
)

// Contingency types; ContingencyTypeList is the order list methods' only type
const (
	ContingencyTypeList        = "LIST"
	ContingencyTypeOCO         = "OCO"
	ContingencyTypeOTO         = "OTO"
	ContingencyTypeOTOCO       = "OTOCO"
	ContingencyTypeSpotAttach  = "SPOT_ATTACH"
	ContingencyTypeDerivAttach = "DERIV_ATTACH"
)

// Isolation types
const (
	IsolationTypeIsolatedMargin = "ISOLATED_MARGIN"
)

// Cancel order types, the orders a cancel-all request cancels; private/cancel-all-orders takes LIMIT and ALL, and
// private/advanced/cancel-all-orders all three, where LIMIT cancels the limit orders of advanced orders only
const (
	CancelOrderTypeLimit   = "LIMIT"
	CancelOrderTypeTrigger = "TRIGGER"
	CancelOrderTypeAll     = "ALL"
)

// Maker or taker sides of a trade
const (
	TakerSideMaker = "MAKER"
	TakerSideTaker = "TAKER"
)

// OrderResponse is an order as the order queries, the advanced order queries and the user order channels report it;
// the list, leg, reference price, attachment, exchange order ID and reject reason fields are only set for advanced
// orders, and the transaction times only by the user order channels
type OrderResponse struct {
	AccountID             string              `json:"account_id"`
	OrderID               string              `json:"order_id"`
	ClientOrderID         string              `json:"client_oid"`
	OrderType             string              `json:"order_type"`
	TimeInForce           string              `json:"time_in_force"`
	Side                  string              `json:"side"`
	ExecutionInstructions []string            `json:"exec_inst"`
	Quantity              types.PreciseNumber `json:"quantity"`
	LimitPrice            types.PreciseNumber `json:"limit_price"`
	OrderValue            types.PreciseNumber `json:"order_value"`
	MakerFeeRate          types.Number        `json:"maker_fee_rate"`
	TakerFeeRate          types.Number        `json:"taker_fee_rate"`
	AveragePrice          types.PreciseNumber `json:"avg_price"`
	CumulativeQuantity    types.PreciseNumber `json:"cumulative_quantity"`
	CumulativeValue       types.PreciseNumber `json:"cumulative_value"`
	CumulativeFee         types.PreciseNumber `json:"cumulative_fee"`
	Status                string              `json:"status"`
	UpdateUserID          string              `json:"update_user_id"`
	// OrderDate is the day the order was created as YYYY-MM-DD, which CreateTime times
	OrderDate            string              `json:"order_date"`
	InstrumentName       string              `json:"instrument_name"`
	FeeInstrumentName    currency.Code       `json:"fee_instrument_name"`
	CreateTime           types.Time          `json:"create_time"`
	CreateTimeNano       types.Time          `json:"create_time_ns"`
	UpdateTime           types.Time          `json:"update_time"`
	TransactionTime      types.Time          `json:"transaction_time"`
	TransactionTimeNano  types.Time          `json:"transaction_time_ns"`
	Reason               NumericID           `json:"reason"`
	IsolationID          string              `json:"isolation_id"`
	IsolationType        string              `json:"isolation_type"`
	ListID               NumericID           `json:"list_id"`
	ContingencyType      string              `json:"contingency_type"`
	LegID                NumericID           `json:"leg_id"`
	ReferencePrice       types.PreciseNumber `json:"ref_price"`
	ReferencePriceType   string              `json:"ref_price_type"`
	AttachOrderID        string              `json:"attach_order_id"`
	AttachIsolationID    string              `json:"attach_isolation_id"`
	ExchangeOrderID      string              `json:"exchange_order_id"`
	RejectReason         string              `json:"reject_reason"`
	Leverage             types.Number        `json:"leverage"`
	IsolatedMarginAmount types.PreciseNumber `json:"isolated_margin_amount"`
}

// OrderIDResponse identifies the order a create, amend, cancel or close request acted on
type OrderIDResponse struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"client_oid"`
}

// OrderListEntryResponse is the outcome of one order of an order list request, by its index in the request; Code is 0
// when the order was accepted, and the order IDs are only set for a created order
type OrderListEntryResponse struct {
	Index         types.Number `json:"index"`
	Code          types.Number `json:"code"`
	OrderID       string       `json:"order_id"`
	ClientOrderID string       `json:"client_oid"`
	Message       string       `json:"message"`
}

// NumericID is an identifier Crypto.com's docs give both as a JSON string and as a bare JSON integer, such as an
// advanced order's list and leg IDs and a transaction's journal ID; it is kept digit for digit, as a list ID exceeds
// what a float holds exactly
type NumericID string

// UnmarshalJSON decodes an identifier from a JSON string or a bare JSON integer, leaving it unchanged for null
func (n *NumericID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) != 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("%w: %w", errInvalidInteger, err)
		}
		*n = NumericID(s)
		return nil
	}
	if _, err := strconv.ParseInt(string(data), 10, 64); err != nil {
		return fmt.Errorf("%w: %w", errInvalidInteger, err)
	}
	*n = NumericID(data)
	return nil
}
