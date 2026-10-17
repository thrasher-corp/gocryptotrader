package deribit

import (
	"bytes"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OrderRequest places a buy or a sell order; Amount or Contracts sets its size
type OrderRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount float64 `json:"amount,omitempty"`
	// Contracts is the size in contracts, which can replace Amount; Deribit rejects the order when both are set and
	// differ
	Contracts float64 `json:"contracts,omitempty"`
	// Type is limit, which Deribit defaults to, stop_limit, take_limit, market, stop_market, take_market, market_limit
	// or trailing_stop; options take no trigger order types
	Type string `json:"type,omitempty"`
	// Label may be up to 64 characters
	Label string `json:"label,omitempty"`
	// Price is the limit price, in USD for an advanced usd option order and in implied volatility percent for an implv
	// one; a zero price is left out, as market orders take none
	Price float64 `json:"price,omitempty"`
	// TimeInForce is good_til_cancelled, which Deribit defaults to, good_til_day, fill_or_kill or immediate_or_cancel
	TimeInForce string `json:"time_in_force,omitempty"`
	// DisplayAmount is the visible amount of an iceberg order
	DisplayAmount float64 `json:"display_amount,omitempty"`
	// PostOnly moves a price that would take liquidity to just inside the spread, or with RejectPostOnly rejects the
	// order; it is always sent, as Deribit documents it defaulting to true
	PostOnly       bool `json:"post_only"`
	RejectPostOnly bool `json:"reject_post_only,omitempty"`
	ReduceOnly     bool `json:"reduce_only,omitempty"`
	// TriggerPrice is required for stop and take orders, and TriggerOffset is a trailing stop's distance from the price
	// peak
	TriggerPrice  float64 `json:"trigger_price,omitempty"`
	TriggerOffset float64 `json:"trigger_offset,omitempty"`
	// Trigger is index_price, mark_price or last_price, and is required for trigger orders
	Trigger string `json:"trigger,omitempty"`
	// Advanced is usd or implv for an option order priced in USD or in implied volatility
	Advanced string `json:"advanced,omitempty"`
	// MMP subjects a limit order to market maker protection
	MMP bool `json:"mmp,omitempty"`
	// ValidUntil makes Deribit reject the order with timed_out unless its matching engine starts on it before then
	ValidUntil time.Time `json:"-"`
	// LinkedOrderType is one_triggers_other, one_cancels_other or one_triggers_one_cancels_other
	LinkedOrderType string `json:"linked_order_type,omitempty"`
	// TriggerFillCondition is first_hit, which Deribit defaults to, complete_fill or incremental, and sets which fills
	// of a linked order place or cancel its secondary orders
	TriggerFillCondition string `json:"trigger_fill_condition,omitempty"`
	// OTOCOConfig lists the secondary orders a fill of a linked order places or cancels
	OTOCOConfig []SecondaryOrder `json:"otoco_config,omitempty"`
}

// orderParams contains a buy or sell request's params, with its valid until time in Unix milliseconds
type orderParams struct {
	*OrderRequest
	ValidUntil int64 `json:"valid_until,omitempty"`
}

// SecondaryOrder is an order a linked order places or cancels once it fills; Amount and Direction are required
type SecondaryOrder struct {
	Amount float64 `json:"amount"`
	// Direction is buy or sell
	Direction string `json:"direction"`
	// Type is limit, which Deribit defaults to, stop_limit, take_limit, market, stop_market, take_market, market_limit
	// or trailing_stop
	Type  string `json:"type,omitempty"`
	Label string `json:"label,omitempty"`
	// Price is required for limit and stop_limit orders
	Price      float64 `json:"price,omitempty"`
	ReduceOnly bool    `json:"reduce_only,omitempty"`
	// TimeInForce is good_til_cancelled, which Deribit defaults to, good_til_day, fill_or_kill or immediate_or_cancel
	TimeInForce    string  `json:"time_in_force,omitempty"`
	PostOnly       bool    `json:"post_only,omitempty"`
	RejectPostOnly bool    `json:"reject_post_only,omitempty"`
	TriggerPrice   float64 `json:"trigger_price,omitempty"`
	TriggerOffset  float64 `json:"trigger_offset,omitempty"`
	// Trigger is index_price, mark_price or last_price, and is required for trigger orders
	Trigger string `json:"trigger,omitempty"`
}

// OrderResponse contains an order placed, edited or closing a position, and the trades it made at once
type OrderResponse struct {
	Order Order `json:"order"`
	// Trades can be empty for a filled spot order routed to Coinbase Exchange, whose fills arrive asynchronously on the
	// user.trades channel
	Trades []UserTrade `json:"trades"`
}

// CancelAllRequest cancels every open order and trigger order of the account
type CancelAllRequest struct {
	// Detailed returns an execution report of each internal cancellation request in place of the count
	Detailed bool `json:"detailed,omitempty"`
	// FreezeQuotes rejects incoming quotes for a second after cancelling
	FreezeQuotes bool `json:"freeze_quotes,omitempty"`
	// IncludeStarbaseOrders also cancels the subaccount's Starbase direct access orders, which are counted in the
	// result but not frozen by FreezeQuotes; Deribit rejects it with Detailed
	IncludeStarbaseOrders bool `json:"include_starbase_orders,omitempty"`
}

// CancelAllByCurrencyRequest cancels the open orders of a currency's instruments
type CancelAllByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which Deribit defaults to, limit, trigger_all, stop, take or trailing_stop
	Type         string `json:"type,omitempty"`
	Detailed     bool   `json:"detailed,omitempty"`
	FreezeQuotes bool   `json:"freeze_quotes,omitempty"`
}

// CancelAllByCurrencyPairRequest cancels the open orders of a currency pair's instruments
type CancelAllByCurrencyPairRequest struct {
	// CurrencyPair is an index name, such as btc_usd or eth_usdc
	CurrencyPair string `json:"currency_pair"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which Deribit defaults to, limit, trigger_all, stop, take or trailing_stop
	Type     string `json:"type,omitempty"`
	Detailed bool   `json:"detailed,omitempty"`
	// FreezeQuotes rejects incoming quotes in the currency pair alone for a second after cancelling
	FreezeQuotes bool `json:"freeze_quotes,omitempty"`
}

// CancelAllByInstrumentRequest cancels the open orders of an instrument
type CancelAllByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is all, which Deribit defaults to, limit, trigger_all, stop, take or trailing_stop
	Type     string `json:"type,omitempty"`
	Detailed bool   `json:"detailed,omitempty"`
	// IncludeCombos also cancels the orders of combos that would affect the instrument's position
	IncludeCombos bool `json:"include_combos,omitempty"`
	FreezeQuotes  bool `json:"freeze_quotes,omitempty"`
}

// CancelAllByKindOrTypeRequest cancels the open orders of one or more currencies, by instrument kind and order type
type CancelAllByKindOrTypeRequest struct {
	// Currencies selects the currencies, or with currency.NewCode("any") every currency; a single currency is sent on
	// its own and several as a list
	Currencies []currency.Code `json:"-"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which Deribit defaults to, limit, trigger_all, stop, take or trailing_stop
	Type         string `json:"type,omitempty"`
	Detailed     bool   `json:"detailed,omitempty"`
	FreezeQuotes bool   `json:"freeze_quotes,omitempty"`
	// IncludeStarbaseOrders also cancels Starbase direct access orders when Type is all; Deribit rejects it with Detailed
	IncludeStarbaseOrders bool `json:"include_starbase_orders,omitempty"`
}

// cancelAllByKindOrTypeParams contains a cancel by kind or type request's params, with its currencies as a single
// currency or a list
type cancelAllByKindOrTypeParams struct {
	*CancelAllByKindOrTypeRequest
	Currency any `json:"currency"`
}

// CancelByLabelRequest cancels the open orders and trigger orders with a label
type CancelByLabelRequest struct {
	Label string `json:"label"`
	// Currency limits the orders cancelled to a currency's instruments; without it the request counts against the
	// cancel all rate limit
	Currency currency.Code `json:"currency,omitzero"`
	// Detailed returns an execution report of each internal cancellation request in place of the count; Deribit's
	// JSON-RPC guide and changelog list it for this method, which its method reference leaves out
	Detailed bool `json:"detailed,omitempty"`
}

// CancelQuotesRequest cancels mass quotes selected by CancelType and the field it names
type CancelQuotesRequest struct {
	// CancelType is delta, which uses MinDelta and MaxDelta, quote_set_id, instrument, instrument_kind, which uses Kind,
	// currency, currency_pair or all
	CancelType   string `json:"cancel_type"`
	Detailed     bool   `json:"detailed,omitempty"`
	FreezeQuotes bool   `json:"freeze_quotes,omitempty"`
	// MinDelta and MaxDelta bound the option deltas a delta cancel selects; both are sent for a delta cancel, as zero
	// and negative bounds are valid
	MinDelta       float64 `json:"-"`
	MaxDelta       float64 `json:"-"`
	QuoteSetID     string  `json:"quote_set_id,omitempty"`
	InstrumentName string  `json:"instrument_name,omitempty"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any
	Kind     string        `json:"kind,omitempty"`
	Currency currency.Code `json:"currency,omitzero"`
	// CurrencyPair is an index name, such as btc_usd
	CurrencyPair string `json:"currency_pair,omitempty"`
}

// cancelQuotesParams contains a cancel quotes request's params, with the delta bounds of a delta cancel
type cancelQuotesParams struct {
	*CancelQuotesRequest
	MinDelta *float64 `json:"min_delta,omitempty"`
	MaxDelta *float64 `json:"max_delta,omitempty"`
}

// CancelAllResponse contains the number of orders a cancel method cancelled and, when detailed is requested, the
// execution report of each internal cancellation request in place of the number Deribit would send
type CancelAllResponse struct {
	// CancelledCount is counted from the execution reports of a detailed response
	CancelledCount   uint64
	ExecutionReports []CancelExecutionReport
}

// UnmarshalJSON decodes the number of orders cancelled, or the list of execution reports of a detailed response
func (c *CancelAllResponse) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		var reports []CancelExecutionReport
		if err := json.Unmarshal(trimmed, &reports); err != nil {
			return fmt.Errorf("%w: %w", errInvalidCancelAllResult, err)
		}
		var count uint64
		for i := range reports {
			count += uint64(len(reports[i].Result))
		}
		*c = CancelAllResponse{CancelledCount: count, ExecutionReports: reports}
		return nil
	}
	var count uint64
	if err := json.Unmarshal(data, &count); err != nil {
		return fmt.Errorf("%w: %w", errInvalidCancelAllResult, err)
	}
	*c = CancelAllResponse{CancelledCount: count}
	return nil
}

// CancelExecutionReport contains the orders one internal cancellation request cancelled; a cancel all request is split
// into requests by currency, order type and instrument book
type CancelExecutionReport struct {
	InstrumentName string `json:"instrument_name"`
	// Currency is the currency, or for spot the pair, such as BTC_USDC, the request cancelled orders in
	Currency string `json:"currency"`
	// Type is the order type the request cancelled
	Type   string  `json:"type"`
	Result []Order `json:"result"`
}

// ClosePositionRequest closes a position with a reduce only order
type ClosePositionRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is limit or market
	Type string `json:"type"`
	// Price is a limit order's price
	Price float64 `json:"price,omitempty"`
}

// EditOrderRequest edits an open order; Amount or Contracts sets its size
type EditOrderRequest struct {
	OrderID string `json:"order_id"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount float64 `json:"amount,omitempty"`
	// Contracts is the size in contracts, which can replace Amount; Deribit rejects the edit when both are set and
	// differ
	Contracts float64 `json:"contracts,omitempty"`
	// Price is the limit price, in USD for an advanced usd option order and in implied volatility percent for an implv
	// one
	Price float64 `json:"price,omitempty"`
	// PostOnly is always sent, as Deribit documents it defaulting to true, so it must be set again to keep an order
	// post only
	PostOnly       bool `json:"post_only"`
	ReduceOnly     bool `json:"reduce_only,omitempty"`
	RejectPostOnly bool `json:"reject_post_only,omitempty"`
	// Advanced is usd or implv, and must be set again to edit an advanced option order
	Advanced      string  `json:"advanced,omitempty"`
	TriggerPrice  float64 `json:"trigger_price,omitempty"`
	TriggerOffset float64 `json:"trigger_offset,omitempty"`
	// MMP is left out when false, as Deribit rejects a change to an order's MMP flag and keeps it when left out
	MMP bool `json:"mmp,omitempty"`
	// ValidUntil makes Deribit reject the edit with timed_out unless its matching engine starts on it before then
	ValidUntil    time.Time `json:"-"`
	DisplayAmount float64   `json:"display_amount,omitempty"`
}

// editOrderParams contains an edit request's params, with its valid until time in Unix milliseconds
type editOrderParams struct {
	*EditOrderRequest
	ValidUntil int64 `json:"valid_until,omitempty"`
}

// EditOrderByLabelRequest edits the only open order of an instrument with a label; Amount or Contracts sets its size
type EditOrderByLabelRequest struct {
	Label          string `json:"label,omitempty"`
	InstrumentName string `json:"instrument_name"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount    float64 `json:"amount,omitempty"`
	Contracts float64 `json:"contracts,omitempty"`
	Price     float64 `json:"price,omitempty"`
	// PostOnly is always sent, as Deribit documents it defaulting to true, so it must be set again to keep an order
	// post only
	PostOnly       bool `json:"post_only"`
	ReduceOnly     bool `json:"reduce_only,omitempty"`
	RejectPostOnly bool `json:"reject_post_only,omitempty"`
	// Advanced is usd or implv, and must be set again to edit an advanced option order
	Advanced     string  `json:"advanced,omitempty"`
	TriggerPrice float64 `json:"trigger_price,omitempty"`
	// MMP is left out when false, as Deribit rejects a change to an order's MMP flag and keeps it when left out
	MMP bool `json:"mmp,omitempty"`
	// ValidUntil makes Deribit reject the edit with timed_out unless its matching engine starts on it before then
	ValidUntil time.Time `json:"-"`
}

// editOrderByLabelParams contains an edit by label request's params, with its valid until time in Unix milliseconds
type editOrderByLabelParams struct {
	*EditOrderByLabelRequest
	ValidUntil int64 `json:"valid_until,omitempty"`
}

// MarginsRequest requests the margins and fees of a hypothetical order
type MarginsRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is in USD for perpetuals and inverse futures, in the base currency for options and linear futures, and for
	// spot in the unit AmountType selects
	Amount float64 `json:"amount"`
	// AmountType is base, which Deribit defaults to, for a base currency amount with fees on top, or quote for a quote
	// currency spend including fees; spot only
	AmountType string  `json:"amount_type,omitempty"`
	Price      float64 `json:"price"`
}

// MarginsResponse contains the margins and fees of a hypothetical order
type MarginsResponse struct {
	// Buy is the margin of buying; for spot with a quote amount type it is the taker notional
	Buy          float64 `json:"buy"`
	Sell         float64 `json:"sell"`
	BuyTakerFee  float64 `json:"buy_taker_fee"`
	BuyMakerFee  float64 `json:"buy_maker_fee"`
	SellTakerFee float64 `json:"sell_taker_fee"`
	SellMakerFee float64 `json:"sell_maker_fee"`
	// MinPrice and MaxPrice clamp sell orders priced lower and buy orders priced higher
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
	// MakerMarginRequired is the fee inclusive maker total of a spot buy with a quote amount type, which Deribit
	// describes alongside Buy without listing it in the result
	MakerMarginRequired float64 `json:"maker_margin_required"`
	// TakerBaseAmount and MakerBaseAmount are the base currency amounts a spot order would trade as a taker and as a
	// maker, floored to the instrument's minimum trade amount
	TakerBaseAmount *MarginBaseAmount `json:"taker_base_amount"`
	MakerBaseAmount *MarginBaseAmount `json:"maker_base_amount"`
}

// MarginBaseAmount contains the base currency amounts a spot order would buy and sell
type MarginBaseAmount struct {
	Buy  float64 `json:"buy"`
	Sell float64 `json:"sell"`
}

// MMPGroupRequest selects an index's MMP group
type MMPGroupRequest struct {
	// IndexName is an index, such as btc_usd, or all, which only block RFQ requests accept for a reset
	IndexName string `json:"index_name,omitempty"`
	// MMPGroup names a mass quote MMP group, which needs IndexName; it is left out when empty to select the orders MMP
	// group, as Deribit rejects an empty name
	MMPGroup string `json:"mmp_group,omitempty"`
	// BlockRFQ selects block RFQ MMP, which is separate from order and quote MMP and needs the block_rfq scope
	BlockRFQ bool `json:"block_rfq,omitempty"`
}

// MMPConfig contains an MMP group's configuration
type MMPConfig struct {
	IndexName string `json:"index_name"`
	// Interval is the monitoring window in seconds, and zero disables MMP
	Interval uint64 `json:"interval"`
	// FrozenTime is how long, in seconds, triggered MMP blocks orders before resetting, and zero needs a manual reset
	FrozenTime uint64 `json:"frozen_time"`
	// ID identifies the group, including the orders MMP group, which has no name
	ID uint64 `json:"id"`
	// MMPGroup is empty for the orders MMP group
	MMPGroup string `json:"mmp_group"`
	// QuantityLimit is the traded quantity, in the base currency, that triggers MMP within an interval, counting buys
	// and sells alike
	QuantityLimit float64 `json:"quantity_limit"`
	// DeltaLimit is the net transaction delta change, in the base currency, that triggers MMP within an interval
	DeltaLimit float64 `json:"delta_limit"`
	// VegaLimit is the vega exposure change, in USD, that triggers MMP within an interval
	VegaLimit float64 `json:"vega_limit"`
	// MaxQuoteQuantity caps the open size of MMP orders per side per instrument, in the base currency, and reserves
	// initial margin of 3% of it while the configuration exists
	MaxQuoteQuantity float64 `json:"max_quote_quantity"`
	BlockRFQ         bool    `json:"block_rfq"`
	// TradeCountLimit is the number of block RFQ trades that triggers block RFQ MMP within an interval
	TradeCountLimit uint64 `json:"trade_count_limit"`
}

// mmpConfigs decodes the configurations private/set_mmp_config returns, which Deribit documents as a list but shows as
// a single configuration
type mmpConfigs []MMPConfig

// UnmarshalJSON decodes a list of MMP configurations or a single one
func (m *mmpConfigs) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '{' {
		var config MMPConfig
		if err := json.Unmarshal(trimmed, &config); err != nil {
			return err
		}
		*m = mmpConfigs{config}
		return nil
	}
	var configs []MMPConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}
	*m = configs
	return nil
}

// MMPStatus contains the status of a triggered MMP group
type MMPStatus struct {
	IndexName string `json:"index_name"`
	// FrozenUntil is when MMP resets itself, and is zero when it waits for a manual reset
	FrozenUntil types.Time `json:"frozen_until"`
	// MMPGroup is set for a triggered mass quote MMP group
	MMPGroup string `json:"mmp_group"`
	BlockRFQ bool   `json:"block_rfq"`
}

// OpenOrdersRequest filters the account's open orders
type OpenOrdersRequest struct {
	// Kind is future, option, spot, future_combo or option_combo; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which Deribit defaults to, limit, trigger_all, stop_all, stop_limit, stop_market, take_all,
	// take_limit, take_market, trailing_all or trailing_stop
	Type string `json:"type,omitempty"`
}

// OpenOrdersByCurrencyRequest filters the open orders of a currency's instruments
type OpenOrdersByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
	// Kind is future, option, spot, future_combo or option_combo; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// Type is all, which Deribit defaults to, limit, trigger_all, stop_all, stop_limit, stop_market, take_all,
	// take_limit, take_market, trailing_all or trailing_stop
	Type string `json:"type,omitempty"`
}

// OpenOrdersByInstrumentRequest filters the open orders of an instrument
type OpenOrdersByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is all, which Deribit defaults to, limit, trigger_all, stop_all, stop_limit, stop_market, take_all,
	// take_limit, take_market, trailing_all or trailing_stop
	Type string `json:"type,omitempty"`
}

// OrdersByLabelRequest selects the orders with a label in a currency's instruments
type OrdersByLabelRequest struct {
	Currency currency.Code `json:"currency"`
	Label    string        `json:"label"`
}

// OrderHistoryByCurrencyRequest requests the order history of a currency's instruments
type OrderHistoryByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
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
}

// OrderHistoryByInstrumentRequest requests the order history of an instrument
type OrderHistoryByInstrumentRequest struct {
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

// OrderHistoryResponse contains a page of order history; Deribit sends the orders alone unless a continuation token is
// requested
type OrderHistoryResponse struct {
	Orders       []Order `json:"orders"`
	Continuation string  `json:"continuation"`
}

// orderHistoryPage is an order history page with a continuation token; its orders field is a pointer to tell a missing
// field from an empty list, as Deribit does not document the page's fields
type orderHistoryPage struct {
	Orders       *[]Order `json:"orders"`
	Continuation string   `json:"continuation"`
}

// UnmarshalJSON decodes a list of orders, or a page of orders with its continuation token
func (o *OrderHistoryResponse) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var orders []Order
		if err := json.Unmarshal(trimmed, &orders); err != nil {
			return err
		}
		*o = OrderHistoryResponse{Orders: orders}
		return nil
	}
	var page orderHistoryPage
	if err := json.Unmarshal(trimmed, &page); err != nil {
		return err
	}
	if page.Orders == nil {
		return fmt.Errorf("%w: %s", errOrderHistoryOrdersMissing, trimmed)
	}
	*o = OrderHistoryResponse{Orders: *page.Orders, Continuation: page.Continuation}
	return nil
}

// orderMarginByIDsParams contains the orders of a private/get_order_margin_by_ids request
type orderMarginByIDsParams struct {
	IDs []string `json:"ids"`
}

// OrderMargin contains the initial margin an open order holds
type OrderMargin struct {
	OrderID               string        `json:"order_id"`
	InitialMargin         float64       `json:"initial_margin"`
	InitialMarginCurrency currency.Code `json:"initial_margin_currency"`
}

// SettlementHistoryByCurrencyRequest requests the settlement events that affected the account in a currency
type SettlementHistoryByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
	// Type is settlement, delivery or bankruptcy; empty selects every type
	Type string `json:"type,omitempty"`
	// Count is the number of events, 20 by default and at most 1000
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
	// SearchStartTime is the latest time events are returned from
	SearchStartTime time.Time `json:"-"`
}

// settlementHistoryByCurrencyParams contains a settlement history request's params, with its search start time in Unix
// milliseconds
type settlementHistoryByCurrencyParams struct {
	*SettlementHistoryByCurrencyRequest
	SearchStartTimestamp int64 `json:"search_start_timestamp,omitempty"`
}

// SettlementHistoryByInstrumentRequest requests the settlement events that affected the account in an instrument
type SettlementHistoryByInstrumentRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Type is settlement, delivery or bankruptcy; empty selects every type
	Type string `json:"type,omitempty"`
	// Count is the number of events, 20 by default and at most 1000
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
	// SearchStartTime is the latest time events are returned from
	SearchStartTime time.Time `json:"-"`
}

// settlementHistoryByInstrumentParams contains a settlement history request's params, with its search start time in
// Unix milliseconds
type settlementHistoryByInstrumentParams struct {
	*SettlementHistoryByInstrumentRequest
	SearchStartTimestamp int64 `json:"search_start_timestamp,omitempty"`
}

// TriggerOrderHistoryRequest requests the account's trigger order log in a currency
type TriggerOrderHistoryRequest struct {
	Currency       currency.Code `json:"currency"`
	InstrumentName string        `json:"instrument_name,omitempty"`
	// Count is the number of entries, 20 by default and at most 1000
	Count        uint64 `json:"count,omitempty"`
	Continuation string `json:"continuation,omitempty"`
}

// TriggerOrderHistoryResponse contains a page of the trigger order log and the continuation token of the next page
type TriggerOrderHistoryResponse struct {
	Entries      []TriggerOrderHistoryEntry `json:"entries"`
	Continuation string                     `json:"continuation"`
}

// TriggerOrderHistoryEntry contains a trigger order's state after the last request on it
type TriggerOrderHistoryEntry struct {
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
	// Price is an OrderPrice, as a trigger market order may have no price
	Price               OrderPrice `json:"price"`
	Amount              float64    `json:"amount"`
	LastUpdateTimestamp types.Time `json:"last_update_timestamp"`
	ReduceOnly          bool       `json:"reduce_only"`
	PostOnly            bool       `json:"post_only"`
	// OrderType is the order type requested, limit or market
	OrderType      string `json:"order_type"`
	Label          string `json:"label"`
	IsSecondaryOTO bool   `json:"is_secondary_oto"`
	OCOReference   string `json:"oco_ref"`
	// Source is where the order linked to the trigger order came from, such as api
	Source string `json:"source"`
}

// UserTradesByCurrencyRequest requests the account's latest trades in a currency's instruments
type UserTradesByCurrencyRequest struct {
	Currency currency.Code `json:"currency"`
	// Kind is future, option, spot, future_combo, option_combo, combo for any combo, or any; empty selects every kind
	Kind string `json:"kind,omitempty"`
	// StartID and EndID bound the trades by trade ID, such as ETH-34066
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
	// SubaccountID selects a subaccount's trades
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// userTradesByCurrencyParams contains a user trades request's params, with its time range in Unix milliseconds
type userTradesByCurrencyParams struct {
	*UserTradesByCurrencyRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// UserTradesByCurrencyAndTimeRequest requests the account's trades in a currency's instruments within a time range,
// which needs a start or an end
type UserTradesByCurrencyAndTimeRequest struct {
	Currency currency.Code `json:"currency"`
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

// userTradesByCurrencyAndTimeParams contains a user trades request's params, with its time range in Unix milliseconds
type userTradesByCurrencyAndTimeParams struct {
	*UserTradesByCurrencyAndTimeRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// UserTradesByInstrumentRequest requests the account's latest trades in an instrument
type UserTradesByInstrumentRequest struct {
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

// userTradesByInstrumentParams contains a user trades request's params, with its time range in Unix milliseconds
type userTradesByInstrumentParams struct {
	*UserTradesByInstrumentRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// UserTradesByInstrumentAndTimeRequest requests the account's trades in an instrument within a time range, which needs
// a start or an end
type UserTradesByInstrumentAndTimeRequest struct {
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

// userTradesByInstrumentAndTimeParams contains a user trades request's params, with its time range in Unix
// milliseconds
type userTradesByInstrumentAndTimeParams struct {
	*UserTradesByInstrumentAndTimeRequest
	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	EndTimestamp   int64 `json:"end_timestamp,omitempty"`
}

// UserTradesByOrderRequest requests the trades an order made
type UserTradesByOrderRequest struct {
	OrderID string `json:"order_id"`
	// Sorting is asc, desc or default, which returns the trades in the order the database holds them
	Sorting string `json:"sorting,omitempty"`
	// Historical returns trades older than 24 hours, after a short indexing delay, in place of recent ones
	Historical bool `json:"historical,omitempty"`
	// SubaccountID selects a subaccount's trades, which needs the mainaccount scope
	SubaccountID uint64 `json:"subaccount_id,omitempty"`
}

// UserTradesResponse contains a page of the account's trades and whether more match the request
type UserTradesResponse struct {
	Trades  []UserTrade `json:"trades"`
	HasMore bool        `json:"has_more"`
}

// MassQuoteRequest places or amends up to 100 quotes on instruments of one index, all linked to an MMP group
type MassQuoteRequest struct {
	// WaitForResponse, which Deribit defaults to true, holds the response until every quote has passed the matching
	// engine; false answers once the risk check passes
	WaitForResponse *bool `json:"wait_for_response,omitempty"`
	// Detailed lists every order change and error in place of the error count
	Detailed bool `json:"detailed,omitempty"`
	// QuoteID identifies the request to match its trades to; Deribit recommends an incrementing counter
	QuoteID string `json:"quote_id"`
	// MMPGroup must be an existing MMP group, which holds one quote per instrument and side
	MMPGroup string `json:"mmp_group"`
	// ValidUntil makes Deribit reject the request with timed_out unless its matching engine starts on it before then
	ValidUntil time.Time        `json:"-"`
	Quotes     []MassQuoteEntry `json:"quotes"`
}

// massQuoteParams contains a mass quote request's params, with its valid until time in Unix milliseconds
type massQuoteParams struct {
	*MassQuoteRequest
	ValidUntil int64 `json:"valid_until,omitempty"`
}

// MassQuoteEntry quotes an instrument on one or both sides
type MassQuoteEntry struct {
	InstrumentName string `json:"instrument_name"`
	// QuoteSetID labels the quote for CancelQuotes, and changing only it keeps the quote's priority
	QuoteSetID string         `json:"quote_set_id,omitempty"`
	Ask        *MassQuoteSide `json:"ask,omitempty"`
	Bid        *MassQuoteSide `json:"bid,omitempty"`
}

// MassQuoteSide sets one side of a quote; leaving Price or Amount nil amends only the other, and a zero Amount cancels
// the side
type MassQuoteSide struct {
	Price          *float64 `json:"price,omitempty"`
	Amount         *float64 `json:"amount,omitempty"`
	PostOnly       bool     `json:"post_only,omitempty"`
	RejectPostOnly bool     `json:"reject_post_only,omitempty"`
}

// MassQuoteResponse contains a mass quote's orders and trades and, depending on Detailed and WaitForResponse, its
// errors and pending quotes or their counts
type MassQuoteResponse struct {
	Orders []Order     `json:"orders"`
	Trades []UserTrade `json:"trades"`
	// ErrorsCount is sent unless Detailed is set
	ErrorsCount uint64 `json:"errors_count"`
	// PendingRequestsCount is sent when WaitForResponse is false and Detailed is not set
	PendingRequestsCount uint64 `json:"pending_requests_count"`
	// PendingRequests is sent when WaitForResponse is false and Detailed is set
	PendingRequests []MassQuotePendingRequest `json:"pending_requests"`
	// Errors is sent when Detailed is set
	Errors []MassQuoteError `json:"errors"`
}

// MassQuotePendingRequest contains a quote side the matching engine has yet to process
type MassQuotePendingRequest struct {
	InstrumentName string `json:"instrument_name"`
	// Side is bid or ask
	Side string `json:"side"`
}

// MassQuoteError contains the error of a quote side Deribit rejected
type MassQuoteError struct {
	InstrumentName string `json:"instrument_name"`
	// Side is bid or ask
	Side  string    `json:"side"`
	Error *APIError `json:"error"`
}

// MovePositionsRequest moves positions between subaccounts; Deribit allows 100 moves a week
type MovePositionsRequest struct {
	Currency currency.Code `json:"currency,omitzero"`
	// SourceUID and TargetUID identify the subaccounts the positions move from and to
	SourceUID uint64         `json:"source_uid"`
	TargetUID uint64         `json:"target_uid"`
	Trades    []PositionMove `json:"trades"`
}

// PositionMove moves part or all of a position
type PositionMove struct {
	InstrumentName string `json:"instrument_name"`
	// Price is the trade's price; zero moves the position at its average price
	Price float64 `json:"price,omitempty"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures, and
	// cannot exceed the position
	Amount float64 `json:"amount"`
}

// PositionMoveTrade contains a trade that moved a position, from the source subaccount's perspective
type PositionMoveTrade struct {
	SourceUID      uint64  `json:"source_uid"`
	TargetUID      uint64  `json:"target_uid"`
	InstrumentName string  `json:"instrument_name"`
	Direction      string  `json:"direction"`
	Price          float64 `json:"price"`
	Amount         float64 `json:"amount"`
}

// positionMoveTrades decodes the trades private/move_positions returns, which Deribit shows as a list but documents in
// an object's trades field
type positionMoveTrades []PositionMoveTrade

// positionMoveTradesPage is the object form of a position move's trades
type positionMoveTradesPage struct {
	Trades []PositionMoveTrade `json:"trades"`
}

// UnmarshalJSON decodes a list of position move trades, or an object holding them
func (p *positionMoveTrades) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '{' {
		var page positionMoveTradesPage
		if err := json.Unmarshal(trimmed, &page); err != nil {
			return err
		}
		*p = page.Trades
		return nil
	}
	var trades []PositionMoveTrade
	if err := json.Unmarshal(data, &trades); err != nil {
		return err
	}
	*p = trades
	return nil
}

// SetMMPConfigRequest sets, or with a zero Interval removes, an MMP group's configuration; limits left at zero are
// disabled rather than kept
type SetMMPConfigRequest struct {
	// IndexName is an index, such as btc_usd, or all for a block RFQ TradeCountLimit
	IndexName string `json:"index_name"`
	// Interval is the monitoring window, in whole seconds up to an hour; zero removes the configuration and releases the
	// margin MaxQuoteQuantity reserves
	Interval time.Duration `json:"-"`
	// FrozenTime is how long triggered MMP blocks orders, in whole seconds up to an hour; zero needs a manual reset
	FrozenTime time.Duration `json:"-"`
	// MMPGroup names a mass quote MMP group, of up to 64 characters; it is left out when empty to select the orders MMP
	// group, as Deribit rejects an empty name
	MMPGroup      string  `json:"mmp_group,omitempty"`
	QuantityLimit float64 `json:"quantity_limit,omitempty"`
	DeltaLimit    float64 `json:"delta_limit,omitempty"`
	VegaLimit     float64 `json:"vega_limit,omitempty"`
	// MaxQuoteQuantity is required, and reserves initial margin of 3% of it while the configuration exists
	MaxQuoteQuantity float64 `json:"max_quote_quantity"`
	BlockRFQ         bool    `json:"block_rfq,omitempty"`
	// TradeCountLimit is the number of block RFQ trades, up to 1000, that triggers block RFQ MMP within an interval
	TradeCountLimit uint64 `json:"trade_count_limit,omitempty"`
}

// setMMPConfigParams contains a set MMP configuration request's params, with its interval and frozen time in seconds,
// which are always sent as zero is meaningful
type setMMPConfigParams struct {
	*SetMMPConfigRequest
	Interval   uint64 `json:"interval"`
	FrozenTime uint64 `json:"frozen_time"`
}
