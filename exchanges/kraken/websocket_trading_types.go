package kraken

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// WsOrder holds an order's parameters on add_order and batch_add
type WsOrder struct {
	// OrderType is limit, market, iceberg, stop-loss, stop-loss-limit, take-profit, take-profit-limit, trailing-stop,
	// trailing-stop-limit or settle-position
	OrderType string
	// Side is buy or sell
	Side string
	// Quantity is in the base asset
	Quantity   float64
	LimitPrice float64
	// LimitPriceType is static, the default, pct for a percentage or quote for an offset in the quote currency. On a
	// trailing-stop-limit order the limit price is an offset from the trigger price
	LimitPriceType string
	// Triggers sets a stop-loss, take-profit or trailing-stop order's trigger
	Triggers *WsOrderTriggers
	// TimeInForce is gtc, the default, gtd, ioc or fok; fok is for limit orders only
	TimeInForce string
	// Margin funds the order on margin at the pair's maximum leverage
	Margin     bool
	PostOnly   bool
	ReduceOnly bool
	// EffectiveTime and ExpireTime schedule the order to the second; a gtd order expires up to a month ahead
	EffectiveTime time.Time
	ExpireTime    time.Time
	// ClientOrderID is a UUID, with or without dashes, or up to 18 characters of text. It cannot be sent with
	// UserReference
	ClientOrderID string
	// UserReference tags a group of orders; Kraken does not enforce its uniqueness
	UserReference int32
	// Conditional creates a close order for each fill of the order
	Conditional *WsConditionalOrder
	// DisplayQuantity is an iceberg order's visible quantity, at least a fifteenth of Quantity
	DisplayQuantity float64
	// FeePreference is base or quote; quote is the default for buys and base for sells
	FeePreference string
	// SelfTradePrevention is cancel_newest, cancel_oldest or cancel_both
	SelfTradePrevention string
	// CashOrderQuantity is the order's quantity in the quote currency
	CashOrderQuantity float64
	// SenderSubID identifies a sub-account or trader for self trade prevention
	SenderSubID string
}

// WsOrderTriggers sets a triggered order's trigger
type WsOrderTriggers struct {
	// Reference is the price tracked: index or last
	Reference string `json:"reference,omitempty"`
	// Price is the trigger price, or with PriceType pct or quote a relative offset, such as -2 for 2% below the
	// reference. A trailing stop's price is its positive reversion from the peak
	Price     float64 `json:"price"`
	PriceType string  `json:"price_type,omitempty"`
}

// WsConditionalOrder sets the close order each fill of an order creates, with the fill's quantity and opposite side
type WsConditionalOrder struct {
	OrderType string `json:"order_type"`
	// LimitPrice is required on limit, stop-loss-limit and take-profit-limit close orders
	LimitPrice     float64 `json:"limit_price,omitempty"`
	LimitPriceType string  `json:"limit_price_type,omitempty"`
	// TriggerPrice and TriggerPriceType set a triggered close order's trigger
	TriggerPrice     float64 `json:"trigger_price,omitempty"`
	TriggerPriceType string  `json:"trigger_price_type,omitempty"`
}

// WsAddOrderRequest holds the parameters of add_order
type WsAddOrderRequest struct {
	Pair  currency.Pair
	Order WsOrder
	// Deadline is when the engine stops trying to match the order, from 500 milliseconds to 60 seconds ahead; Kraken
	// defaults to 5 seconds
	Deadline time.Time
	// Validate validates the order without placing it
	Validate bool
}

// WsAddOrderResponse holds a placed order's identifiers
type WsAddOrderResponse struct {
	OrderID            string    `json:"order_id"`
	ClientOrderID      string    `json:"cl_ord_id"`
	OrderUserReference int32     `json:"order_userref"`
	Warnings           []string  `json:"warnings"`
	TimeIn             time.Time `json:"-"`
	TimeOut            time.Time `json:"-"`
}

// WsEditOrderRequest holds the parameters of edit_order. Kraken cannot edit triggered stop-loss or take-profit orders,
// orders with close orders attached, or orders identified by client order ID, and rejects a quantity below the filled
// quantity
type WsEditOrderRequest struct {
	OrderID string
	// Pair is the order's pair, which cannot be changed
	Pair            currency.Pair
	Quantity        float64
	LimitPrice      float64
	DisplayQuantity float64
	FeePreference   string
	PostOnly        bool
	ReduceOnly      bool
	Triggers        *WsOrderTriggers
	// UserReference is placed on the new order; it does not identify the edited order
	UserReference int32
	Validate      bool
	Deadline      time.Time
}

// WsEditOrderResponse holds the new order replacing an edited one
type WsEditOrderResponse struct {
	OrderID         string    `json:"order_id"`
	OriginalOrderID string    `json:"original_order_id"`
	Warnings        []string  `json:"warnings"`
	TimeIn          time.Time `json:"-"`
	TimeOut         time.Time `json:"-"`
}

// WsAmendOrderRequest holds the parameters of amend_order. Either OrderID or ClientOrderID identifies the order
type WsAmendOrderRequest struct {
	OrderID       string
	ClientOrderID string
	// Pair is required for xStocks only
	Pair     currency.Pair
	Quantity float64
	// DisplayQuantity is at least a fifteenth of the remaining quantity
	DisplayQuantity float64
	LimitPrice      float64
	// LimitPriceType is static, the default except on trailing-stop-limit orders, pct or quote
	LimitPriceType string
	// PostOnly rejects a limit price change that cannot rest in the book
	PostOnly         bool
	TriggerPrice     float64
	TriggerPriceType string
	Deadline         time.Time
}

// WsAmendOrderResponse holds an amend transaction's identifier
type WsAmendOrderResponse struct {
	AmendID       string    `json:"amend_id"`
	OrderID       string    `json:"order_id"`
	ClientOrderID string    `json:"cl_ord_id"`
	Warnings      []string  `json:"warnings"`
	TimeIn        time.Time `json:"-"`
	TimeOut       time.Time `json:"-"`
}

// WsCancelOrdersRequest holds the parameters of cancel_order. Exactly one kind of identifier may be sent
type WsCancelOrdersRequest struct {
	OrderIDs       []string
	ClientOrderIDs []string
	UserReferences []int32
}

// WsCancelOrderResponse holds the outcome of cancelling one order; Error is set when it failed
type WsCancelOrderResponse struct {
	OrderID       string    `json:"order_id"`
	ClientOrderID string    `json:"cl_ord_id"`
	Warnings      []string  `json:"warnings"`
	Error         error     `json:"-"`
	TimeIn        time.Time `json:"-"`
	TimeOut       time.Time `json:"-"`
}

// WsCancelAllOrdersResponse holds the number of orders cancel_all cancelled
type WsCancelAllOrdersResponse struct {
	Count    uint64    `json:"count"`
	Warnings []string  `json:"warnings"`
	TimeIn   time.Time `json:"-"`
	TimeOut  time.Time `json:"-"`
}

// WsCancelAllOrdersAfterResponse holds the engine's time and when it will cancel every order
type WsCancelAllOrdersAfterResponse struct {
	CurrentTime time.Time `json:"currentTime"`
	TriggerTime time.Time `json:"triggerTime"`
	TimeIn      time.Time `json:"-"`
	TimeOut     time.Time `json:"-"`
}

// WsBatchAddOrdersRequest holds the parameters of batch_add
type WsBatchAddOrdersRequest struct {
	Pair     currency.Pair
	Orders   []WsOrder
	Deadline time.Time
	Validate bool
}

// WsBatchCancelOrdersRequest holds the parameters of batch_cancel
type WsBatchCancelOrdersRequest struct {
	// Orders holds order IDs or user references
	Orders         []string
	ClientOrderIDs []string
}

// WsBatchCancelOrdersResponse holds the number of orders batch_cancel cancelled
type WsBatchCancelOrdersResponse struct {
	Count    uint64    `json:"count"`
	Warnings []string  `json:"warnings"`
	TimeIn   time.Time `json:"-"`
	TimeOut  time.Time `json:"-"`
}

// wsTokenSetter is a private request's parameters, which carry the websocket token
type wsTokenSetter interface {
	setToken(string)
}

// wsTokenParams holds the websocket token of a private request
type wsTokenParams struct {
	Token string `json:"token"`
}

func (p *wsTokenParams) setToken(token string) {
	p.Token = token
}

// wsOrderParams holds an order's parameters as add_order and batch_add take them
type wsOrderParams struct {
	OrderType           string              `json:"order_type"`
	Side                string              `json:"side"`
	OrderQuantity       float64             `json:"order_qty"`
	LimitPrice          float64             `json:"limit_price,omitempty"`
	LimitPriceType      string              `json:"limit_price_type,omitempty"`
	Triggers            *WsOrderTriggers    `json:"triggers,omitempty"`
	TimeInForce         string              `json:"time_in_force,omitempty"`
	Margin              bool                `json:"margin,omitempty"`
	PostOnly            bool                `json:"post_only,omitempty"`
	ReduceOnly          bool                `json:"reduce_only,omitempty"`
	EffectiveTime       string              `json:"effective_time,omitempty"`
	ExpireTime          string              `json:"expire_time,omitempty"`
	ClientOrderID       string              `json:"cl_ord_id,omitempty"`
	OrderUserReference  int32               `json:"order_userref,omitempty"`
	Conditional         *WsConditionalOrder `json:"conditional,omitempty"`
	DisplayQuantity     float64             `json:"display_qty,omitempty"`
	FeePreference       string              `json:"fee_preference,omitempty"`
	SelfTradePrevention string              `json:"stp_type,omitempty"`
	CashOrderQuantity   float64             `json:"cash_order_qty,omitempty"`
	SenderSubID         string              `json:"sender_sub_id,omitempty"`
}

// wsAddOrderParams holds the parameters of add_order
type wsAddOrderParams struct {
	wsOrderParams
	Symbol   string `json:"symbol"`
	Deadline string `json:"deadline,omitempty"`
	Validate bool   `json:"validate,omitempty"`
	wsTokenParams
}

// wsEditOrderParams holds the parameters of edit_order
type wsEditOrderParams struct {
	OrderID            string           `json:"order_id"`
	Symbol             string           `json:"symbol"`
	OrderQuantity      float64          `json:"order_qty,omitempty"`
	LimitPrice         float64          `json:"limit_price,omitempty"`
	DisplayQuantity    float64          `json:"display_qty,omitempty"`
	FeePreference      string           `json:"fee_preference,omitempty"`
	PostOnly           bool             `json:"post_only,omitempty"`
	ReduceOnly         bool             `json:"reduce_only,omitempty"`
	Triggers           *WsOrderTriggers `json:"triggers,omitempty"`
	OrderUserReference int32            `json:"order_userref,omitempty"`
	Validate           bool             `json:"validate,omitempty"`
	Deadline           string           `json:"deadline,omitempty"`
	wsTokenParams
}

// wsAmendOrderParams holds the parameters of amend_order
type wsAmendOrderParams struct {
	OrderID          string  `json:"order_id,omitempty"`
	ClientOrderID    string  `json:"cl_ord_id,omitempty"`
	Symbol           string  `json:"symbol,omitempty"`
	OrderQuantity    float64 `json:"order_qty,omitempty"`
	DisplayQuantity  float64 `json:"display_qty,omitempty"`
	LimitPrice       float64 `json:"limit_price,omitempty"`
	LimitPriceType   string  `json:"limit_price_type,omitempty"`
	PostOnly         bool    `json:"post_only,omitempty"`
	TriggerPrice     float64 `json:"trigger_price,omitempty"`
	TriggerPriceType string  `json:"trigger_price_type,omitempty"`
	Deadline         string  `json:"deadline,omitempty"`
	wsTokenParams
}

// wsCancelOrderParams holds the parameters of cancel_order
type wsCancelOrderParams struct {
	OrderIDs            []string `json:"order_id,omitempty"`
	ClientOrderIDs      []string `json:"cl_ord_id,omitempty"`
	OrderUserReferences []int32  `json:"order_userref,omitempty"`
	wsTokenParams
}

// wsCancelAfterParams holds the parameters of cancel_all_orders_after
type wsCancelAfterParams struct {
	Timeout uint64 `json:"timeout"`
	wsTokenParams
}

// wsBatchAddParams holds the parameters of batch_add
type wsBatchAddParams struct {
	Symbol   string          `json:"symbol"`
	Orders   []wsOrderParams `json:"orders"`
	Deadline string          `json:"deadline,omitempty"`
	Validate bool            `json:"validate,omitempty"`
	wsTokenParams
}

// wsBatchCancelParams holds the parameters of batch_cancel
type wsBatchCancelParams struct {
	Orders         []string `json:"orders"`
	ClientOrderIDs []string `json:"cl_ord_id,omitempty"`
	wsTokenParams
}
