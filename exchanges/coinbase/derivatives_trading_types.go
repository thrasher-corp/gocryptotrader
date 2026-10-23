package coinbase

import (
	"bytes"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Gateway linked order types, which link a primary order to the secondary orders its fills place or cancel
const (
	// DerivativesLinkedOrderTypeOneTriggersOther places one or more secondary orders once the primary order fills
	DerivativesLinkedOrderTypeOneTriggersOther = "one_triggers_other"
	// DerivativesLinkedOrderTypeOneCancelsOther pairs the primary order with one secondary order, and a fill of either
	// cancels the other
	DerivativesLinkedOrderTypeOneCancelsOther = "one_cancels_other"
	// DerivativesLinkedOrderTypeOneTriggersOneCancelsOther places a pair of secondary orders, typically a take profit and
	// a stop loss, once the primary order fills, and a fill of either cancels the other
	DerivativesLinkedOrderTypeOneTriggersOneCancelsOther = "one_triggers_one_cancels_other"
)

// Gateway trigger fill conditions, which set the fills of a linked order that place or cancel its secondary orders
const (
	// DerivativesTriggerFillConditionFirstHit acts on the secondary orders in full at the primary order's first fill
	DerivativesTriggerFillConditionFirstHit = "first_hit"
	// DerivativesTriggerFillConditionCompleteFill acts on the secondary orders once the primary order has filled in full
	DerivativesTriggerFillConditionCompleteFill = "complete_fill"
	// DerivativesTriggerFillConditionIncremental acts on the secondary orders in proportion to each fill, rounded down
	// to the contract size
	DerivativesTriggerFillConditionIncremental = "incremental"
)

// Gateway advanced option order types, which price an option order in USD or in implied volatility
const (
	DerivativesAdvancedUSD               = "usd"
	DerivativesAdvancedImpliedVolatility = "implv"
)

// Gateway cancel order types, which select the orders the cancel all methods cancel
const (
	DerivativesCancelOrderTypeAll          = "all"
	DerivativesCancelOrderTypeLimit        = "limit"
	DerivativesCancelOrderTypeTriggerAll   = "trigger_all"
	DerivativesCancelOrderTypeStop         = "stop"
	DerivativesCancelOrderTypeTake         = "take"
	DerivativesCancelOrderTypeTrailingStop = "trailing_stop"
)

// Gateway open order types, which select the orders the open orders methods return
const (
	DerivativesOpenOrderTypeAll          = "all"
	DerivativesOpenOrderTypeLimit        = "limit"
	DerivativesOpenOrderTypeTriggerAll   = "trigger_all"
	DerivativesOpenOrderTypeStopAll      = "stop_all"
	DerivativesOpenOrderTypeStopLimit    = "stop_limit"
	DerivativesOpenOrderTypeStopMarket   = "stop_market"
	DerivativesOpenOrderTypeTakeAll      = "take_all"
	DerivativesOpenOrderTypeTakeLimit    = "take_limit"
	DerivativesOpenOrderTypeTakeMarket   = "take_market"
	DerivativesOpenOrderTypeTrailingAll  = "trailing_all"
	DerivativesOpenOrderTypeTrailingStop = "trailing_stop"
)

// DerivativesOrderRequest places a buy or sell order; Amount or Contracts sizes it
type DerivativesOrderRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is in the base currency for spot, options and linear instruments such as the USDC perpetuals, and in USD
	// for inverse futures and perpetuals
	Amount float64 `json:"amount,omitempty"`
	// Contracts sizes the order in contracts in place of Amount; the gateway rejects an order whose Amount and Contracts
	// differ
	Contracts float64 `json:"contracts,omitempty"`
	// Type is limit, which the gateway defaults to, market, market_limit, stop_limit, stop_market, take_limit,
	// take_market or trailing_stop; options take no trigger order types, and spot orders routed to Coinbase Exchange
	// only market, limit and stop_limit
	Type string `json:"type,omitempty"`
	// Label is the order's client order ID, up to 64 characters, which the gateway does not require to be unique, so it
	// cannot serve as an idempotency key
	Label string `json:"label,omitempty"`
	// Price is the limit price, in USD for an advanced usd option order and in implied volatility percent for an implv
	// one; a zero price is left out, as market orders take none
	Price float64 `json:"price,omitempty"`
	// TimeInForce is good_til_cancelled, which the gateway defaults to, good_til_day, fill_or_kill or
	// immediate_or_cancel; the gateway has no good til time, and spot orders take no good_til_day
	TimeInForce string `json:"time_in_force,omitempty"`
	// DisplayAmount is the visible amount of an iceberg order, at least 100 times the instrument's minimum amount
	DisplayAmount float64 `json:"display_amount,omitempty"`
	// PostOnly moves a price that would take liquidity to just inside the spread, or with RejectPostOnly rejects the
	// order, and needs a good_til_cancelled order; it is always sent, as the gateway defaults it to true. Spot orders
	// routed to Coinbase Exchange need RejectPostOnly with it
	PostOnly       bool `json:"post_only"`
	RejectPostOnly bool `json:"reject_post_only,omitempty"`
	ReduceOnly     bool `json:"reduce_only,omitempty"`
	// TriggerPrice is required for stop and take orders, and TriggerOffset is a trailing stop's distance from the price
	// peak
	TriggerPrice  float64 `json:"trigger_price,omitempty"`
	TriggerOffset float64 `json:"trigger_offset,omitempty"`
	// Trigger is index_price, mark_price or last_price, and is required for trigger orders; stop limit orders routed to
	// Coinbase Exchange take only last_price
	Trigger string `json:"trigger,omitempty"`
	// Advanced is usd or implv for an option order priced in USD or in implied volatility; linear options take no usd
	Advanced string `json:"advanced,omitempty"`
	// MMP subjects a limit order to market maker protection, which the gateway exposes no configuration methods for
	MMP bool `json:"mmp,omitempty"`
	// ValidUntil makes the gateway reject the order with timed_out unless its matching engine starts on it before then
	ValidUntil time.Time `json:"-"`
	// LinkedOrderType links the order to the secondary orders in OTOCOConfig: one_triggers_other takes one or more,
	// one_cancels_other one and one_triggers_one_cancels_other two
	LinkedOrderType string `json:"linked_order_type,omitempty"`
	// TriggerFillCondition is first_hit, which the gateway defaults to, complete_fill or incremental, and sets which fills
	// of a linked order place or cancel its secondary orders
	TriggerFillCondition string `json:"trigger_fill_condition,omitempty"`
	// OTOCOConfig lists the secondary orders a fill of a linked order places or cancels
	OTOCOConfig []DerivativesSecondaryOrder `json:"otoco_config,omitempty"`
	// Isolated places a futures order under isolated margin, in an isolated subaccount the gateway claims or provisions
	// and binds to the instrument; only main accounts can, and not for trigger, advanced or linked orders
	Isolated bool `json:"isolated,omitempty"`
	// AllocatedMargin is the initial margin moved from the main account to the isolated subaccount; the gateway computes
	// it when unset, and refuses a risk increasing order whose margin falls short of its minimum with error 14033
	AllocatedMargin float64 `json:"allocated_margin,omitempty"`
}

// DerivativesOrderWireRequest holds a buy or sell request's params, with its valid until time in Unix milliseconds
type DerivativesOrderWireRequest struct {
	*DerivativesOrderRequest
	ValidUntil uint64 `json:"valid_until,omitempty"`
}

// DerivativesSecondaryOrder is an order a linked order places or cancels once it fills; Amount and Direction are
// required
type DerivativesSecondaryOrder struct {
	// Amount is in the base currency for linear instruments such as the USDC perpetuals, and in USD for inverse ones
	Amount float64 `json:"amount"`
	// Direction is buy or sell
	Direction string `json:"direction"`
	// Type is limit, which the gateway defaults to, market, market_limit, stop_limit, stop_market, take_limit,
	// take_market or trailing_stop
	Type  string `json:"type,omitempty"`
	Label string `json:"label,omitempty"`
	// Price is required for limit and stop limit orders
	Price      float64 `json:"price,omitempty"`
	ReduceOnly bool    `json:"reduce_only,omitempty"`
	// TimeInForce is good_til_cancelled, which the gateway defaults to, good_til_day, fill_or_kill or
	// immediate_or_cancel
	TimeInForce    string  `json:"time_in_force,omitempty"`
	PostOnly       bool    `json:"post_only,omitempty"`
	RejectPostOnly bool    `json:"reject_post_only,omitempty"`
	TriggerPrice   float64 `json:"trigger_price,omitempty"`
	TriggerOffset  float64 `json:"trigger_offset,omitempty"`
	// Trigger is index_price, mark_price or last_price, and is required for trigger orders
	Trigger string `json:"trigger,omitempty"`
}

// DerivativesOrderTradesResponse is an order placed, edited or closing a position, and the trades it made at once
type DerivativesOrderTradesResponse struct {
	Order DerivativesOrderResponse `json:"order"`
	// Trades can be empty for a filled spot order routed to Coinbase Exchange, whose fills arrive asynchronously on the
	// user.trades channel
	Trades []DerivativesUserTradeResponse `json:"trades"`
}

// DerivativesUserTradeResponse is one of the account's trades, as the trading methods and the user.trades notifications
// report it
type DerivativesUserTradeResponse struct {
	// TradeID is unique within the trade's currency
	TradeID string `json:"trade_id"`
	// TradeSequence is the trade's sequence number within its instrument
	TradeSequence  uint64 `json:"trade_seq"`
	InstrumentName string `json:"instrument_name"`
	// Timestamp is in milliseconds, and StarbaseTimestamp, the Starbase causal time of a trade matched in Starbase, in
	// nanoseconds
	Timestamp         types.Time `json:"timestamp"`
	StarbaseTimestamp types.Time `json:"starbase_timestamp"`
	// StarbaseOrderID is the order's raw Starbase ID, without the gateway's currency prefix, and StarbaseClientOrderID
	// the client order ID of an order submitted to Starbase directly
	StarbaseOrderID       uint64 `json:"starbase_order_id"`
	StarbaseClientOrderID string `json:"starbase_client_order_id"`
	StarbaseMatchID       uint64 `json:"starbase_match_id"`
	// OrderType is limit, market or liquidation; OriginalOrderType is the order type placed, market or market_limit, when
	// the gateway holds the order as a limit order
	OrderType         string `json:"order_type"`
	OriginalOrderType string `json:"original_order_type"`
	// Advanced is usd or implv for an option order priced in USD or in implied volatility
	Advanced string `json:"advanced"`
	// OrderID is the account's order, maker or taker, that made the trade
	OrderID   string `json:"order_id"`
	Direction string `json:"direction"`
	// TickDirection is 0 for a plus tick, 1 for a zero plus tick, 2 for a minus tick and 3 for a zero minus tick
	TickDirection uint64  `json:"tick_direction"`
	IndexPrice    float64 `json:"index_price"`
	Price         float64 `json:"price"`
	MarkPrice     float64 `json:"mark_price"`
	// Amount is in the base currency for spot, options and linear instruments such as the USDC perpetuals, and in USD
	// for inverse futures and perpetuals
	Amount    float64 `json:"amount"`
	Contracts float64 `json:"contracts"`
	// ImpliedVolatility and UnderlyingPrice are set for option trades
	ImpliedVolatility float64 `json:"iv"`
	UnderlyingPrice   float64 `json:"underlying_price"`
	// Liquidation is M, T or MT when the maker, the taker or both were being liquidated
	Liquidation string `json:"liquidation"`
	// Liquidity is M when the account's order was the maker and T when it was the taker
	Liquidity   string        `json:"liquidity"`
	Fee         float64       `json:"fee"`
	FeeCurrency currency.Code `json:"fee_currency"`
	// ProfitLoss is in the base currency
	ProfitLoss float64 `json:"profit_loss"`
	// Label is the order's client order ID, sent only when the order has one
	Label string `json:"label"`
	// State is the order's state: open, filled, rejected, cancelled, untriggered or archive
	State              string `json:"state"`
	BlockTradeID       string `json:"block_trade_id"`
	BlockTradeLegCount uint64 `json:"block_trade_leg_count"`
	BlockRFQID         uint64 `json:"block_rfq_id"`
	BlockRFQQuoteID    uint64 `json:"block_rfq_quote_id"`
	// ReduceOnly and PostOnly are documented as strings and sent as booleans, so either form is accepted
	ReduceOnly   types.Boolean `json:"reduce_only"`
	PostOnly     types.Boolean `json:"post_only"`
	MMP          bool          `json:"mmp"`
	RiskReducing bool          `json:"risk_reducing"`
	API          bool          `json:"api"`
	// Legs holds a combo trade's leg trades, which the gateway includes only when combo trades alone are requested
	Legs             []DerivativesUserTradeResponse `json:"legs"`
	ComboID          string                         `json:"combo_id"`
	ComboTradeID     string                         `json:"combo_trade_id"`
	QuoteSetID       string                         `json:"quote_set_id"`
	QuoteID          string                         `json:"quote_id"`
	TradeAllocations []DerivativesTradeAllocation   `json:"trade_allocations"`
	// Isolated, MainUserID and UserID are set for a trade of an isolated margin subaccount, whose user ID is the
	// subaccount's
	Isolated   bool   `json:"isolated"`
	MainUserID uint64 `json:"main_uid"`
	UserID     uint64 `json:"user_id"`
}

// DerivativesTradeAllocation is the part of a Block RFQ trade pre-allocated to one user
type DerivativesTradeAllocation struct {
	// UserID is the user the part is allocated to, which the gateway obscures for brokers
	UserID     uint64                                `json:"user_id"`
	Amount     float64                               `json:"amount"`
	Fee        float64                               `json:"fee"`
	ClientInfo *DerivativesTradeAllocationClientInfo `json:"client_info"`
}

// DerivativesTradeAllocationClientInfo identifies the broker client a trade allocation belongs to
type DerivativesTradeAllocationClientInfo struct {
	// ClientID identifies a group of users under one name, and ClientLinkID one user of the client
	ClientID     uint64 `json:"client_id"`
	ClientLinkID uint64 `json:"client_link_id"`
	Name         string `json:"name"`
}

// DerivativesCancelOrderByIDRequest cancels an open order or trigger order
type DerivativesCancelOrderByIDRequest struct {
	OrderID string `json:"order_id"`
	// Isolated looks the order up in the isolated margin subaccount that owns it; without it the gateway searches the main
	// account alone and answers order_not_found or not_owner_of_order for an isolated order
	Isolated bool `json:"isolated,omitempty"`
}

// DerivativesCancelAllRequest cancels every open order and trigger order of the account
type DerivativesCancelAllRequest struct {
	// Detailed returns an execution report of each internal cancellation request in place of the count
	Detailed bool `json:"detailed,omitempty"`
	// FreezeQuotes rejects incoming quotes for a second after cancelling
	FreezeQuotes bool `json:"freeze_quotes,omitempty"`
	// IncludeIsolated also cancels the open orders of the active isolated margin subaccounts, which are kept otherwise
	IncludeIsolated bool `json:"include_isolated,omitempty"`
}

// DerivativesCancelAllByCurrencyRequest cancels the open orders of a currency's instruments
type DerivativesCancelAllByCurrencyRequest struct {
	Currency currency.Code `json:"-"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which the gateway defaults to, limit, trigger_all, stop, take or trailing_stop
	Type         string `json:"type,omitempty"`
	Detailed     bool   `json:"detailed,omitempty"`
	FreezeQuotes bool   `json:"freeze_quotes,omitempty"`
	// IncludeIsolated also cancels the matching open orders of the active isolated margin subaccounts
	IncludeIsolated bool `json:"include_isolated,omitempty"`
}

// DerivativesCancelAllByCurrencyWireRequest holds a cancel by currency request's params, with its currency as the
// gateway takes it
type DerivativesCancelAllByCurrencyWireRequest struct {
	*DerivativesCancelAllByCurrencyRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency"`
}

// DerivativesCancelAllByCurrencyPairRequest cancels the open orders of a currency pair's instruments
type DerivativesCancelAllByCurrencyPairRequest struct {
	// CurrencyPair is an index name, such as btc_usdc
	CurrencyPair string `json:"currency_pair"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which the gateway defaults to, limit, trigger_all, stop, take or trailing_stop
	Type     string `json:"type,omitempty"`
	Detailed bool   `json:"detailed,omitempty"`
	// FreezeQuotes rejects incoming quotes in the currency pair alone for a second after cancelling
	FreezeQuotes bool `json:"freeze_quotes,omitempty"`
}

// DerivativesCancelAllByInstrumentRequest cancels the open orders of an instrument
type DerivativesCancelAllByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is all, which the gateway defaults to, limit, trigger_all, stop, take or trailing_stop
	Type     string `json:"type,omitempty"`
	Detailed bool   `json:"detailed,omitempty"`
	// IncludeCombos also cancels the orders of combos that would affect the instrument's position
	IncludeCombos bool `json:"include_combos,omitempty"`
	FreezeQuotes  bool `json:"freeze_quotes,omitempty"`
	// IncludeIsolated also cancels the instrument's open orders in the active isolated margin subaccounts
	IncludeIsolated bool `json:"include_isolated,omitempty"`
}

// DerivativesCancelAllByKindOrTypeRequest cancels the open orders of one or more currencies, by instrument kind and
// order type
type DerivativesCancelAllByKindOrTypeRequest struct {
	// Currencies selects the currencies, or with currency.NewCode(DerivativesCurrencyAny) every currency; a single
	// currency is sent on its own and several as a list
	Currencies []currency.Code `json:"-"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which the gateway defaults to, limit, trigger_all, stop, take or trailing_stop
	Type         string `json:"type,omitempty"`
	Detailed     bool   `json:"detailed,omitempty"`
	FreezeQuotes bool   `json:"freeze_quotes,omitempty"`
}

// DerivativesCancelAllByKindOrTypeWireRequest holds a cancel by kind or type request's params, with its currencies as a
// single currency or a list
type DerivativesCancelAllByKindOrTypeWireRequest struct {
	*DerivativesCancelAllByKindOrTypeRequest
	Currency any `json:"currency"`
}

// DerivativesCancelByLabelRequest cancels the open orders and trigger orders with a label
type DerivativesCancelByLabelRequest struct {
	Label string `json:"label"`
	// Currency limits the orders cancelled to a currency's instruments; without it the request counts against the
	// cancel all rate limit
	Currency currency.Code `json:"-"`
	// IncludeIsolated also cancels the open orders with the label in the active isolated margin subaccounts
	IncludeIsolated bool `json:"include_isolated,omitempty"`
}

// DerivativesCancelByLabelWireRequest holds a cancel by label request's params, with its currency as the gateway takes
// it
type DerivativesCancelByLabelWireRequest struct {
	*DerivativesCancelByLabelRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency,omitempty"`
}

// DerivativesCancelAllResponse is the number of orders a cancel method cancelled and, when detailed is requested, the
// execution report of each internal cancellation request, which the gateway sends in place of the number
type DerivativesCancelAllResponse struct {
	// CancelledCount is counted from the execution reports of a detailed response
	CancelledCount   uint64
	ExecutionReports []DerivativesCancelExecutionReport
}

// UnmarshalJSON decodes the number of orders cancelled, or the list of execution reports of a detailed response
func (d *DerivativesCancelAllResponse) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		var reports []DerivativesCancelExecutionReport
		if err := json.Unmarshal(trimmed, &reports); err != nil {
			return fmt.Errorf("%w: %w", errDerivativesInvalidCancelAllResult, err)
		}
		var count uint64
		for i := range reports {
			count += uint64(len(reports[i].Orders()))
		}
		*d = DerivativesCancelAllResponse{CancelledCount: count, ExecutionReports: reports}
		return nil
	}
	var count uint64
	if err := json.Unmarshal(data, &count); err != nil {
		return fmt.Errorf("%w: %w", errDerivativesInvalidCancelAllResult, err)
	}
	*d = DerivativesCancelAllResponse{CancelledCount: count}
	return nil
}

// DerivativesCancelExecutionReport is the orders one internal cancellation request cancelled; a cancel all request is
// split into requests by currency, order type and instrument book. The gateway documents the report by an example
// alone, which holds a cancelled order in Order, while its prose describes reports by request, holding the orders in
// Result, so both are read
type DerivativesCancelExecutionReport struct {
	InstrumentName string `json:"instrument_name"`
	// Currency is the currency, or for spot the pair, such as BTC_USDC, the request cancelled orders in
	Currency string `json:"currency"`
	// Type is the order type the request cancelled
	Type   string                     `json:"type"`
	Result []DerivativesOrderResponse `json:"result"`
	Order  *DerivativesOrderResponse  `json:"order"`
}

// Orders returns the orders a report holds
func (r *DerivativesCancelExecutionReport) Orders() []*DerivativesOrderResponse {
	orders := make([]*DerivativesOrderResponse, 0, len(r.Result)+1)
	for i := range r.Result {
		orders = append(orders, &r.Result[i])
	}
	if r.Order != nil {
		orders = append(orders, r.Order)
	}
	return orders
}

// DerivativesClosePositionRequest closes a position with a reduce only order
type DerivativesClosePositionRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is limit or market
	Type string `json:"type"`
	// Price is a limit order's price
	Price float64 `json:"price,omitempty"`
	// Isolated closes the position of the isolated margin subaccount bound to the instrument, in full
	Isolated bool `json:"isolated,omitempty"`
}

// DerivativesEditOrderRequest edits an open order; Amount or Contracts sizes it
type DerivativesEditOrderRequest struct {
	OrderID string `json:"order_id"`
	// Amount is in the base currency for spot, options and linear instruments such as the USDC perpetuals, and in USD
	// for inverse futures and perpetuals
	Amount float64 `json:"amount,omitempty"`
	// Contracts sizes the order in contracts in place of Amount; the gateway rejects an edit whose Amount and Contracts
	// differ
	Contracts float64 `json:"contracts,omitempty"`
	// Price is the limit price, in USD for an advanced usd option order and in implied volatility percent for an implv
	// one
	Price float64 `json:"price,omitempty"`
	// PostOnly is always sent, as the gateway takes a missing one as true: an edit sets it afresh, true keeping or making
	// the order post only and false clearing it
	PostOnly       bool `json:"post_only"`
	ReduceOnly     bool `json:"reduce_only,omitempty"`
	RejectPostOnly bool `json:"reject_post_only,omitempty"`
	// Advanced is usd or implv, and must be set again to edit an advanced option order
	Advanced      string  `json:"advanced,omitempty"`
	TriggerPrice  float64 `json:"trigger_price,omitempty"`
	TriggerOffset float64 `json:"trigger_offset,omitempty"`
	// MMP is left out when false, as the gateway rejects a change to an order's MMP flag and keeps it when left out
	MMP bool `json:"mmp,omitempty"`
	// ValidUntil makes the gateway reject the edit with timed_out unless its matching engine starts on it before then
	ValidUntil    time.Time `json:"-"`
	DisplayAmount float64   `json:"display_amount,omitempty"`
	// Isolated edits an order of the isolated margin subaccount that owns it, from the main account
	Isolated bool `json:"isolated,omitempty"`
	// AllocatedMargin is margin moved from the main account to the order's isolated margin subaccount by an edit that
	// increases risk
	AllocatedMargin float64 `json:"allocated_margin,omitempty"`
}

// DerivativesEditOrderWireRequest holds an edit request's params, with its valid until time in Unix milliseconds
type DerivativesEditOrderWireRequest struct {
	*DerivativesEditOrderRequest
	ValidUntil uint64 `json:"valid_until,omitempty"`
}

// DerivativesEditOrderByLabelRequest edits the only open order of an instrument with a label; Amount or Contracts
// sizes it
type DerivativesEditOrderByLabelRequest struct {
	Label          string `json:"label,omitempty"`
	InstrumentName string `json:"instrument_name"`
	// Amount is in the base currency for spot, options and linear instruments such as the USDC perpetuals, and in USD
	// for inverse futures and perpetuals
	Amount    float64 `json:"amount,omitempty"`
	Contracts float64 `json:"contracts,omitempty"`
	Price     float64 `json:"price,omitempty"`
	// PostOnly is always sent, as the gateway takes a missing one as true: an edit sets it afresh, true keeping or making
	// the order post only and false clearing it
	PostOnly       bool `json:"post_only"`
	ReduceOnly     bool `json:"reduce_only,omitempty"`
	RejectPostOnly bool `json:"reject_post_only,omitempty"`
	// Advanced is usd or implv, and must be set again to edit an advanced option order
	Advanced     string  `json:"advanced,omitempty"`
	TriggerPrice float64 `json:"trigger_price,omitempty"`
	// MMP is left out when false, as the gateway rejects a change to an order's MMP flag and keeps it when left out
	MMP bool `json:"mmp,omitempty"`
	// ValidUntil makes the gateway reject the edit with timed_out unless its matching engine starts on it before then
	ValidUntil time.Time `json:"-"`
	// Isolated edits an order of the isolated margin subaccount that owns it, from the main account
	Isolated bool `json:"isolated,omitempty"`
	// AllocatedMargin is margin moved from the main account to the order's isolated margin subaccount by an edit that
	// increases risk
	AllocatedMargin float64 `json:"allocated_margin,omitempty"`
}

// DerivativesEditOrderByLabelWireRequest holds an edit by label request's params, with its valid until time in Unix
// milliseconds
type DerivativesEditOrderByLabelWireRequest struct {
	*DerivativesEditOrderByLabelRequest
	ValidUntil uint64 `json:"valid_until,omitempty"`
}

// DerivativesMarginsRequest asks for the margins and fees of a hypothetical order
type DerivativesMarginsRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is in the base currency for options and linear instruments such as the USDC perpetuals, and in USD for
	// inverse futures and perpetuals
	Amount float64 `json:"amount"`
	// Price is always sent, as the gateway requires it
	Price float64 `json:"price"`
	// Isolated computes the margins and fees under isolated margin in place of cross margin
	Isolated bool `json:"isolated,omitempty"`
}

// DerivativesMarginsResponse is the margins and fees of a hypothetical order, and the prices orders are clamped to
type DerivativesMarginsResponse struct {
	Buy          float64 `json:"buy"`
	Sell         float64 `json:"sell"`
	BuyMakerFee  float64 `json:"buy_maker_fee"`
	BuyTakerFee  float64 `json:"buy_taker_fee"`
	SellMakerFee float64 `json:"sell_maker_fee"`
	SellTakerFee float64 `json:"sell_taker_fee"`
	// MinimumPrice and MaximumPrice clamp sell orders priced lower and buy orders priced higher
	MinimumPrice float64 `json:"min_price"`
	MaximumPrice float64 `json:"max_price"`
}

// DerivativesMaxOrderSizeRequest asks for the largest orders a future can take, optionally under hypothetical
// conditions
type DerivativesMaxOrderSizeRequest struct {
	InstrumentName string `json:"instrument_name"`
	// SubaccountID selects a subaccount, for retail broker callers; the authenticated account is used when unset
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
	// Isolated computes the sizes under isolated margin
	Isolated bool `json:"isolated,omitempty"`
	// Leverage is a hypothetical leverage, from 1 to the account's effective maximum
	Leverage float64 `json:"leverage,omitempty"`
	// TakerFeeRate is a hypothetical taker fee rate, which may be zero, so it is sent whenever it is set; overriding the
	// rate needs the gateway's authorisation
	TakerFeeRate *float64 `json:"taker_fee_rate,omitempty"`
	// Price is a hypothetical order price; the book and mark prices are used when unset
	Price float64 `json:"price,omitempty"`
}

// DerivativesMaxOrderSizeResponse is the largest orders a future can take given the account's funds, position and
// open orders
type DerivativesMaxOrderSizeResponse struct {
	// BuyPrice and SellPrice are the reference prices the sizes are computed at
	BuyPrice  float64 `json:"buy_price"`
	SellPrice float64 `json:"sell_price"`
	// MaximumTradableAmount and MaximumTradableAmountReduceOnly are the largest orders and reduce only orders, as
	// notional amounts in the quote currency
	MaximumTradableAmount           DerivativesTradableNotional `json:"max_tradable_amount"`
	MaximumTradableAmountReduceOnly DerivativesTradableNotional `json:"max_tradable_amount_reduce_only"`
}

// DerivativesTradableNotional is the largest buy and sell notionals, in the quote currency
type DerivativesTradableNotional struct {
	BuyNotional  float64 `json:"buy_notional"`
	SellNotional float64 `json:"sell_notional"`
}

// DerivativesOpenOrdersRequest filters the account's open orders
type DerivativesOpenOrdersRequest struct {
	// Kind is future, option, spot, future_combo or option_combo; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which the gateway defaults to, limit, trigger_all, stop_all, stop_limit, stop_market, take_all,
	// take_limit, take_market, trailing_all or trailing_stop
	Type string `json:"type,omitempty"`
	// IncludeIsolated also returns the open orders of the hidden isolated margin subaccounts
	IncludeIsolated bool `json:"include_isolated,omitempty"`
	// SubaccountID selects a subaccount's open orders, such as an isolated margin subaccount of the main account
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// DerivativesOpenOrdersByCurrencyRequest filters the open orders of a currency's instruments
type DerivativesOpenOrdersByCurrencyRequest struct {
	Currency currency.Code `json:"-"`
	// Kind is future, option, spot, future_combo or option_combo; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which the gateway defaults to, limit, trigger_all, stop_all, stop_limit, stop_market, take_all,
	// take_limit, take_market, trailing_all or trailing_stop
	Type string `json:"type,omitempty"`
}

// DerivativesOpenOrdersByCurrencyWireRequest holds an open orders by currency request's params, with its currency as
// the gateway takes it
type DerivativesOpenOrdersByCurrencyWireRequest struct {
	*DerivativesOpenOrdersByCurrencyRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency"`
}

// DerivativesOpenOrdersByInstrumentRequest filters the open orders of an instrument
type DerivativesOpenOrdersByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is all, which the gateway defaults to, limit, trigger_all, stop_all, stop_limit, stop_market, take_all,
	// take_limit, take_market, trailing_all or trailing_stop
	Type string `json:"type,omitempty"`
}

// DerivativesOpenOrdersByLabelRequest selects the open orders with a label in a currency's instruments
type DerivativesOpenOrdersByLabelRequest struct {
	Currency currency.Code `json:"-"`
	Label    string        `json:"label"`
}

// DerivativesOpenOrdersByLabelWireRequest holds an open orders by label request's params, with its currency as the
// gateway takes it
type DerivativesOpenOrdersByLabelWireRequest struct {
	*DerivativesOpenOrdersByLabelRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency"`
}

// DerivativesOrderHistoryByCurrencyRequest asks for the order history of a currency's instruments
type DerivativesOrderHistoryByCurrencyRequest struct {
	Currency currency.Code `json:"-"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Count is the number of orders, 20 by default and at most 1000
	Count  uint64 `json:"count,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	// IncludeOld includes orders older than two days, and IncludeUnfilled closed orders that never filled
	IncludeOld      bool `json:"include_old,omitempty"`
	IncludeUnfilled bool `json:"include_unfilled,omitempty"`
	// WithContinuation returns a continuation token with the orders, which Continuation then pages from
	WithContinuation bool   `json:"with_continuation,omitempty"`
	Continuation     string `json:"continuation,omitempty"`
	// Historical returns orders older than 30 minutes, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
	// SubaccountID selects a subaccount's history; an isolated margin subaccount's history is only returned for its
	// subaccount ID
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// DerivativesOrderHistoryByCurrencyWireRequest holds an order history by currency request's params, with its currency
// as the gateway takes it
type DerivativesOrderHistoryByCurrencyWireRequest struct {
	*DerivativesOrderHistoryByCurrencyRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency"`
}

// DerivativesOrderHistoryByInstrumentRequest asks for the order history of an instrument
type DerivativesOrderHistoryByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Count is the number of orders, 20 by default and at most 1000
	Count  uint64 `json:"count,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	// IncludeOld includes orders older than two days, and IncludeUnfilled closed orders that never filled
	IncludeOld      bool `json:"include_old,omitempty"`
	IncludeUnfilled bool `json:"include_unfilled,omitempty"`
	// WithContinuation returns a continuation token with the orders, which Continuation then pages from
	WithContinuation bool   `json:"with_continuation,omitempty"`
	Continuation     string `json:"continuation,omitempty"`
	// Historical returns orders older than 30 minutes, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
}

// DerivativesOrderHistoryResponse is a page of order history; the gateway sends the orders alone unless a
// continuation token is requested
type DerivativesOrderHistoryResponse struct {
	Orders       []DerivativesOrderResponse `json:"orders"`
	Continuation string                     `json:"continuation"`
}

// DerivativesOrderHistoryPage is an order history page with a continuation token; its orders field is a pointer to
// tell a missing field from an empty list, as the gateway does not document the page's fields
type DerivativesOrderHistoryPage struct {
	Orders       *[]DerivativesOrderResponse `json:"orders"`
	Continuation string                      `json:"continuation"`
}

// UnmarshalJSON decodes a list of orders, or a page of orders with its continuation token
func (d *DerivativesOrderHistoryResponse) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var orders []DerivativesOrderResponse
		if err := json.Unmarshal(trimmed, &orders); err != nil {
			return err
		}
		*d = DerivativesOrderHistoryResponse{Orders: orders}
		return nil
	}
	var page DerivativesOrderHistoryPage
	if err := json.Unmarshal(trimmed, &page); err != nil {
		return err
	}
	if page.Orders == nil {
		return fmt.Errorf("%w: %s", errDerivativesOrderHistoryOrdersMissing, trimmed)
	}
	*d = DerivativesOrderHistoryResponse{Orders: *page.Orders, Continuation: page.Continuation}
	return nil
}

// DerivativesOrderMarginByIDsRequest asks for the initial margin of orders
type DerivativesOrderMarginByIDsRequest struct {
	OrderIDs []string `json:"ids"`
	// Isolated computes the margins under isolated margin
	Isolated bool `json:"isolated,omitempty"`
}

// DerivativesOrderMarginResponse is the initial margin an open order holds
type DerivativesOrderMarginResponse struct {
	OrderID               string        `json:"order_id"`
	InitialMargin         float64       `json:"initial_margin"`
	InitialMarginCurrency currency.Code `json:"initial_margin_currency"`
}

// DerivativesOrderStateRequest asks for an order that is open or closed within the last 30 minutes
type DerivativesOrderStateRequest struct {
	OrderID string `json:"order_id"`
	// Isolated looks the order up in the isolated margin subaccount that owns it
	Isolated bool `json:"isolated,omitempty"`
}

// DerivativesOrderStateByLabelRequest asks for the recent orders with a label in a currency's instruments
type DerivativesOrderStateByLabelRequest struct {
	Currency currency.Code `json:"-"`
	Label    string        `json:"label"`
	// IncludeIsolated also searches the hidden isolated margin subaccounts
	IncludeIsolated bool `json:"include_isolated,omitempty"`
}

// DerivativesOrderStateByLabelWireRequest holds an order state by label request's params, with its currency as the
// gateway takes it
type DerivativesOrderStateByLabelWireRequest struct {
	*DerivativesOrderStateByLabelRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency"`
}

// DerivativesSettlementHistoryByCurrencyRequest asks for the settlement events that affected the account in a currency
type DerivativesSettlementHistoryByCurrencyRequest struct {
	Currency currency.Code `json:"-"`
	// Type is settlement, delivery or bankruptcy; empty selects every type
	Type string `json:"type,omitempty"`
	// Count is the number of events, 20 by default and at most 1000
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
	// SearchStartTime is the latest time events are returned from
	SearchStartTime time.Time `json:"-"`
	// SubaccountID selects a subaccount's events; an isolated margin subaccount's perpetual settlements are only returned
	// for its subaccount ID
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// DerivativesSettlementHistoryByCurrencyWireRequest holds a settlement history request's params, with its currency as
// the gateway takes it and its search start time in Unix milliseconds
type DerivativesSettlementHistoryByCurrencyWireRequest struct {
	*DerivativesSettlementHistoryByCurrencyRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency             string `json:"currency"`
	SearchStartTimestamp uint64 `json:"search_start_timestamp,omitempty"`
}

// DerivativesSettlementHistoryByInstrumentRequest asks for the settlement events that affected the account in an
// instrument
type DerivativesSettlementHistoryByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is settlement, delivery or bankruptcy; empty selects every type
	Type string `json:"type,omitempty"`
	// Count is the number of events, 20 by default and at most 1000
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
	// SearchStartTime is the latest time events are returned from
	SearchStartTime time.Time `json:"-"`
}

// DerivativesSettlementHistoryByInstrumentWireRequest holds a settlement history request's params, with its search
// start time in Unix milliseconds
type DerivativesSettlementHistoryByInstrumentWireRequest struct {
	*DerivativesSettlementHistoryByInstrumentRequest
	SearchStartTimestamp uint64 `json:"search_start_timestamp,omitempty"`
}

// DerivativesTriggerOrderHistoryRequest asks for the account's trigger order log in a currency
type DerivativesTriggerOrderHistoryRequest struct {
	Currency       currency.Code `json:"-"`
	InstrumentName string        `json:"instrument_name,omitempty"`
	// Count is the number of entries, 20 by default and at most 1000
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
}

// DerivativesTriggerOrderHistoryWireRequest holds a trigger order history request's params, with its currency as the
// gateway takes it
type DerivativesTriggerOrderHistoryWireRequest struct {
	*DerivativesTriggerOrderHistoryRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency"`
}

// DerivativesTriggerOrderHistoryResponse is a page of the trigger order log and the continuation token of the next
// page
type DerivativesTriggerOrderHistoryResponse struct {
	Entries      []DerivativesTriggerOrderHistoryEntry `json:"entries"`
	Continuation string                                `json:"continuation"`
}

// DerivativesTriggerOrderHistoryEntry is a trigger order's state after the last request on it
type DerivativesTriggerOrderHistoryEntry struct {
	Timestamp types.Time `json:"timestamp"`
	// Trigger is index_price, mark_price or last_price
	Trigger      string  `json:"trigger"`
	TriggerPrice float64 `json:"trigger_price"`
	// TriggerOffset is a trailing trigger order's distance from the price peak
	TriggerOffset float64 `json:"trigger_offset"`
	// TriggerOrderID identifies the trigger order before it triggered, and OrderID the order it placed
	TriggerOrderID string `json:"trigger_order_id"`
	OrderID        string `json:"order_id"`
	// OrderState is triggered, cancelled, or rejected with its reason, such as rejected:reduce_direction
	OrderState     string `json:"order_state"`
	InstrumentName string `json:"instrument_name"`
	// Request is the last request on the trigger order, such as cancel or trigger:order
	Request   string `json:"request"`
	Direction string `json:"direction"`
	// Price is a DerivativesOrderPrice, as a trigger market order may have no price
	Price               DerivativesOrderPrice `json:"price"`
	Amount              float64               `json:"amount"`
	LastUpdateTimestamp types.Time            `json:"last_update_timestamp"`
	ReduceOnly          bool                  `json:"reduce_only"`
	PostOnly            bool                  `json:"post_only"`
	// OrderType is the order type requested, limit or market
	OrderType      string `json:"order_type"`
	Label          string `json:"label"`
	IsSecondaryOTO bool   `json:"is_secondary_oto"`
	OCOReference   string `json:"oco_ref"`
	// Source is where the order linked to the trigger order came from, such as api
	Source string `json:"source"`
}

// DerivativesUserTradesByCurrencyRequest asks for the account's latest trades in a currency's instruments
type DerivativesUserTradesByCurrencyRequest struct {
	Currency currency.Code `json:"-"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// StartID and EndID bound the trades by trade ID, such as ETH-15 or ETH_USDC-16
	StartID string `json:"start_id,omitempty"`
	EndID   string `json:"end_id,omitempty"`
	// Count is the number of trades, 10 by default and at most 1000
	Count     uint64    `json:"count,omitempty"`
	StartTime time.Time `json:"-"`
	EndTime   time.Time `json:"-"`
	// Sorting is asc, desc or default, which returns the trades in the order the database holds them
	Sorting string `json:"sorting,omitempty"`
	// Historical returns trades older than 24 hours, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
	// SubaccountID selects a subaccount's trades, such as an isolated margin subaccount's
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// DerivativesUserTradesByCurrencyWireRequest holds a user trades request's params, with its currency as the gateway
// takes it and its time range in Unix milliseconds
type DerivativesUserTradesByCurrencyWireRequest struct {
	*DerivativesUserTradesByCurrencyRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency       string `json:"currency"`
	StartTimestamp uint64 `json:"start_timestamp,omitempty"`
	EndTimestamp   uint64 `json:"end_timestamp,omitempty"`
}

// DerivativesUserTradesByCurrencyAndTimeRequest asks for the account's trades in a currency's instruments within a time
// range, which needs a start or an end
type DerivativesUserTradesByCurrencyAndTimeRequest struct {
	Currency currency.Code `json:"-"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind      string    `json:"kind,omitempty"`
	StartTime time.Time `json:"-"`
	EndTime   time.Time `json:"-"`
	// Count is the number of trades, 10 by default and at most 1000
	Count uint64 `json:"count,omitempty"`
	// Sorting is asc, desc or default, which returns the trades in the order the database holds them
	Sorting string `json:"sorting,omitempty"`
	// Historical returns trades older than 24 hours, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
	// SubaccountID selects a subaccount's trades, which needs the mainaccount scope
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// DerivativesUserTradesByCurrencyAndTimeWireRequest holds a user trades request's params, with its currency as the
// gateway takes it and its time range in Unix milliseconds
type DerivativesUserTradesByCurrencyAndTimeWireRequest struct {
	*DerivativesUserTradesByCurrencyAndTimeRequest
	// Currency stays a string, the code in the case the gateway takes
	Currency       string `json:"currency"`
	StartTimestamp uint64 `json:"start_timestamp,omitempty"`
	EndTimestamp   uint64 `json:"end_timestamp,omitempty"`
}

// DerivativesUserTradesByInstrumentRequest asks for the account's latest trades in an instrument
type DerivativesUserTradesByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// StartSequence and EndSequence bound the trades by their sequence number within the instrument
	StartSequence uint64 `json:"start_seq,omitempty"`
	EndSequence   uint64 `json:"end_seq,omitempty"`
	// Count is the number of trades, 10 by default and at most 1000
	Count     uint64    `json:"count,omitempty"`
	StartTime time.Time `json:"-"`
	EndTime   time.Time `json:"-"`
	// Historical returns trades older than 24 hours, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
	// Sorting is asc, desc or default, which returns the trades in the order the database holds them
	Sorting string `json:"sorting,omitempty"`
	// SubaccountID selects a subaccount's trades, which needs the mainaccount scope
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// DerivativesUserTradesByInstrumentWireRequest holds a user trades request's params, with its time range in Unix
// milliseconds
type DerivativesUserTradesByInstrumentWireRequest struct {
	*DerivativesUserTradesByInstrumentRequest
	StartTimestamp uint64 `json:"start_timestamp,omitempty"`
	EndTimestamp   uint64 `json:"end_timestamp,omitempty"`
}

// DerivativesUserTradesByInstrumentAndTimeRequest asks for the account's trades in an instrument within a time range,
// which needs a start or an end
type DerivativesUserTradesByInstrumentAndTimeRequest struct {
	InstrumentName string    `json:"instrument_name"`
	StartTime      time.Time `json:"-"`
	EndTime        time.Time `json:"-"`
	// Count is the number of trades, 10 by default and at most 1000
	Count uint64 `json:"count,omitempty"`
	// Sorting is asc, desc or default, which returns the trades in the order the database holds them
	Sorting string `json:"sorting,omitempty"`
	// Historical returns trades older than 24 hours, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
	// SubaccountID selects a subaccount's trades, which needs the mainaccount scope
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// DerivativesUserTradesByInstrumentAndTimeWireRequest holds a user trades request's params, with its time range in Unix
// milliseconds
type DerivativesUserTradesByInstrumentAndTimeWireRequest struct {
	*DerivativesUserTradesByInstrumentAndTimeRequest
	StartTimestamp uint64 `json:"start_timestamp,omitempty"`
	EndTimestamp   uint64 `json:"end_timestamp,omitempty"`
}

// DerivativesUserTradesByOrderRequest asks for the trades an order made
type DerivativesUserTradesByOrderRequest struct {
	OrderID string `json:"order_id"`
	// Sorting is asc, desc or default, which returns the trades in the order the database holds them
	Sorting string `json:"sorting,omitempty"`
	// Historical returns trades older than 24 hours, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
	// SubaccountID selects a subaccount's trades, which needs the mainaccount scope
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
	// Isolated looks the order up in the isolated margin subaccount that owns it; without it an isolated order's trades
	// are not found
	Isolated bool `json:"isolated,omitempty"`
}

// DerivativesUserTradesResponse is a page of the account's trades and whether more match the request
type DerivativesUserTradesResponse struct {
	Trades  []DerivativesUserTradeResponse `json:"trades"`
	HasMore bool                           `json:"has_more"`
}

// DerivativesSetLeverageRequest sets or removes a future's custom leverage
type DerivativesSetLeverageRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Leverage is the custom leverage, from 1 to the instrument's effective maximum; nil removes the custom leverage and
	// restores the risk profile's default, so it is always sent
	Leverage *float64 `json:"leverage"`
	// SubaccountID selects a subaccount, for retail broker callers alone; the gateway answers anyone else with
	// only_for_retail_brokers, and an isolated margin subaccount's leverage is set with Isolated instead
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
	// Isolated sets the leverage of the instrument's isolated margin scope, which only main accounts can, before or after
	// an isolated margin subaccount is bound to it
	Isolated bool `json:"isolated,omitempty"`
}
