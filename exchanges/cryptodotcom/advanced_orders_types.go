package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Position types, the derivative positions whose advanced orders an advanced cancel-all request can be limited to
const (
	PositionTypeCrossMargin    = "CROSS_MARGIN"
	PositionTypeIsolatedMargin = "ISOLATED_MARGIN"
)

// CreateAdvancedOrderRequest is a trigger order, which Crypto.com places once the reference price type's price reaches
// ReferencePrice: STOP_LOSS and TAKE_PROFIT orders as market orders, and STOP_LIMIT and TAKE_PROFIT_LIMIT orders as limit
// orders at Price. It is a SPOT_ATTACH order on a spot instrument and a DERIV_ATTACH order on a derivative
type CreateAdvancedOrderRequest struct {
	InstrumentName string       `json:"instrument_name"`
	Side           string       `json:"side"`
	OrderType      string       `json:"type"`
	Quantity       types.Number `json:"quantity"`
	// Price is the limit price, which STOP_LIMIT and TAKE_PROFIT_LIMIT orders need
	Price          types.Number `json:"price,omitempty"`
	ReferencePrice types.Number `json:"ref_price"`
	// ReferencePriceType is the price compared with ReferencePrice, by default the mark price
	ReferencePriceType             string        `json:"ref_price_type,omitempty"`
	ClientOrderID                  string        `json:"client_oid,omitempty"`
	TimeInForce                    string        `json:"time_in_force,omitempty"`
	ExecutionInstructions          []string      `json:"exec_inst,omitempty"`
	FeeInstrumentName              currency.Code `json:"fee_instrument_name,omitzero"`
	SelfTradePreventionID          uint64        `json:"stp_id,string,omitempty"`
	SelfTradePreventionInstruction string        `json:"stp_inst,omitempty"`
	SelfTradePreventionScope       string        `json:"stp_scope,omitempty"`
	// IsolationID places a derivative order that attaches to neither an order nor a position on the isolated position it
	// names; without it the order goes on the cross margin position
	IsolationID string `json:"isolation_id,omitempty"`
	// AttachOrderID attaches the order to an existing limit order, as its take profit or stop loss
	AttachOrderID string `json:"attach_order_id,omitempty"`
	// AttachIsolationID attaches the order to a derivative position: an isolated position by its isolation ID, or the
	// cross margin position by "0". The order must then be REDUCE_ONLY, and set neither AttachOrderID nor IsolationID
	AttachIsolationID string `json:"attach_isolation_id,omitempty"`
}

// AmendAdvancedOrderRequest amends a trigger order, identified by its order ID or client order ID. Crypto.com requires
// the quantity and the limit price, so unchanged values are restated
type AmendAdvancedOrderRequest struct {
	OrderID       string
	ClientOrderID string
	NewQuantity   types.Number
	// NewPrice is the order's limit price after the amendment. Zero makes the order, or keeps it, a trigger market order
	// (STOP_LOSS or TAKE_PROFIT), and a positive price a trigger limit order (STOP_LIMIT or TAKE_PROFIT_LIMIT); it is a
	// pointer as zero is a price to send, not an absent one
	NewPrice *types.Number
	// NewReferencePrice is the order's reference price after the amendment; zero keeps the reference price
	NewReferencePrice types.Number
}

// amendAdvancedOrderParams is the params of an amend-order request; the new price is sent formatted, as zero is a price
// Crypto.com takes and types.Number sends zero as an empty string
type amendAdvancedOrderParams struct {
	OrderID           string       `json:"order_id,omitempty"`
	ClientOrderID     string       `json:"client_oid,omitempty"`
	NewQuantity       types.Number `json:"new_quantity"`
	NewPrice          string       `json:"new_price"`
	NewReferencePrice types.Number `json:"new_ref_price,omitempty"`
}

// AdvancedOrderLegRequest is an order of an OCO, OTO or OTOCO: a LIMIT or MARKET order, or a trigger order, which
// Crypto.com places once the reference price type's price reaches ReferencePrice, as a market order for STOP_LOSS and
// TAKE_PROFIT and a limit order at Price for STOP_LIMIT and TAKE_PROFIT_LIMIT
type AdvancedOrderLegRequest struct {
	InstrumentName string `json:"instrument_name"`
	Side           string `json:"side"`
	OrderType      string `json:"type"`
	// Quantity may be left out of a MARKET, STOP_LOSS or TAKE_PROFIT buy that spends Notional instead
	Quantity types.Number `json:"quantity,omitempty"`
	// Price is the limit price, which LIMIT, STOP_LIMIT and TAKE_PROFIT_LIMIT orders need
	Price types.Number `json:"price,omitempty"`
	// Notional is the amount a MARKET, STOP_LOSS or TAKE_PROFIT buy spends
	Notional types.Number `json:"notional,omitempty"`
	// ReferencePrice is the trigger price, which trigger orders need
	ReferencePrice types.Number `json:"ref_price,omitempty"`
	// ReferencePriceType is the price compared with ReferencePrice, by default the mark price; an OCO's trigger order
	// takes the mark price only
	ReferencePriceType             string        `json:"ref_price_type,omitempty"`
	ClientOrderID                  string        `json:"client_oid,omitempty"`
	TimeInForce                    string        `json:"time_in_force,omitempty"`
	ExecutionInstructions          []string      `json:"exec_inst,omitempty"`
	FeeInstrumentName              currency.Code `json:"fee_instrument_name,omitzero"`
	SelfTradePreventionID          uint64        `json:"stp_id,string,omitempty"`
	SelfTradePreventionInstruction string        `json:"stp_inst,omitempty"`
	SelfTradePreventionScope       string        `json:"stp_scope,omitempty"`
	SpotMargin                     string        `json:"spot_margin,omitempty"`
	// IsolationID, Leverage and IsolatedMarginAmount set a derivative OTO or OTOCO's primary order on an isolated
	// position, whose isolation its trigger orders inherit
	IsolationID          string       `json:"isolation_id,omitempty"`
	Leverage             types.Number `json:"leverage,omitempty"`
	IsolatedMarginAmount types.Number `json:"isolated_margin_amount,omitempty"`
	AttachOrderID        string       `json:"attach_order_id,omitempty"`
}

// advancedOrderListParams is the params of a request carrying a list of orders
type advancedOrderListParams[T any] struct {
	OrderList []T `json:"order_list"`
}

// AdvancedOrderListIDResponse identifies an OCO, OTO or OTOCO by the list ID its orders share
type AdvancedOrderListIDResponse struct {
	ListID NumericID `json:"list_id"`
}

// advancedListIDParams is the params of a request naming an OCO, OTO or OTOCO by its list ID
type advancedListIDParams struct {
	ListID string `json:"list_id"`
}

// AdvancedOrderIDRequest identifies an advanced order by its order ID or its client order ID
type AdvancedOrderIDRequest struct {
	OrderID       string `json:"order_id,omitempty"`
	ClientOrderID string `json:"client_oid,omitempty"`
}

// CancelAllAdvancedOrdersRequest selects the advanced orders to cancel: every instrument's, or one instrument's.
// OrderType limits the cancel to the limit or the trigger orders. PositionType limits it to the orders tied to a
// derivative's cross margin or isolated positions; it needs InstrumentName, and takes no OrderType but ALL
type CancelAllAdvancedOrdersRequest struct {
	InstrumentName string `json:"instrument_name,omitempty"`
	OrderType      string `json:"type,omitempty"`
	PositionType   string `json:"position_type,omitempty"`
}

// GetAdvancedOpenOrdersRequest selects the open advanced orders to return: one instrument's, or every instrument's
// when InstrumentName is empty
type GetAdvancedOpenOrdersRequest struct {
	InstrumentName string `json:"instrument_name,omitempty"`
}

// GetAdvancedOrderHistoryRequest selects the advanced orders to return from the order history: one instrument's, or
// every instrument's when InstrumentName is empty. Crypto.com defaults EndTime, which is exclusive, to now, StartTime to
// a day before EndTime, and Limit, the most orders returned, to its maximum of 100
type GetAdvancedOrderHistoryRequest struct {
	InstrumentName string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
}

// advancedOrderHistoryParams is the params of an advanced get-order-history request, its times in Unix nanoseconds
type advancedOrderHistoryParams struct {
	InstrumentName string `json:"instrument_name,omitempty"`
	StartTime      uint64 `json:"start_time,omitempty,string"`
	EndTime        uint64 `json:"end_time,omitempty,string"`
	Limit          uint64 `json:"limit,omitempty,string"`
}

// advancedOrderListDetailParams is the params of a get-order-list-detail request
type advancedOrderListDetailParams struct {
	ListIDs []string `json:"list_id_list"`
}
