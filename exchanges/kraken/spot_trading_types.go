package kraken

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// The times in force spot orders take
const (
	timeInForceGTC = "GTC"
	timeInForceIOC = "IOC"
	timeInForceGTD = "GTD"
	timeInForceFOK = "FOK"
)

var (
	errMultipleOrderIdentifiers = errors.New("more than one order identifier set")
	errConflictingOrderFlags    = errors.New("conflicting order flags")
	errInvalidOrderPrice        = errors.New("invalid order price")
	errInvalidOrderTime         = errors.New("invalid order start or expiry time")
	errInvalidCancelTimeout     = errors.New("invalid cancel timeout")
)

// OrderPrice is an order price as Kraken reads it, which a float64 alone cannot express: an absolute price, such as
// 40000.5, or an offset from a reference price. Offset is the prefix of a relative price, + to add Value to the
// reference price, - to subtract it or # to add or subtract it as the order's side and type require, and Percent makes
// Value a percentage of the reference price. The reference price is the last traded price, and the trigger price for
// a trailing stop limit order's limit price. A relative zero, such as +0, differs from no price, which is the zero
// OrderPrice and is not sent
type OrderPrice struct {
	Value   float64
	Offset  string
	Percent bool
}

// String returns the price as Kraken takes it, such as 40000.5, +50 or -2.5%
func (p OrderPrice) String() string {
	s := p.Offset + strconv.FormatFloat(p.Value, 'f', -1, 64)
	if p.Percent {
		s += "%"
	}
	return s
}

// absolute returns an absolute price, and zero for an offset from another price, such as +50 or -2.5%
func (p OrderPrice) absolute() float64 {
	if p.Offset != "" || p.Percent {
		return 0
	}
	return p.Value
}

// UnmarshalJSON decodes a price sent as a number or a string, keeping the offset and percent sign of a relative one
func (p *OrderPrice) UnmarshalJSON(data []byte) error {
	s := string(data)
	if len(data) != 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	}
	*p = OrderPrice{}
	if s == "" || s == "null" {
		return nil
	}
	if strings.ContainsAny(s[:1], "+-#") {
		p.Offset, s = s[:1], s[1:]
	}
	if trimmed, ok := strings.CutSuffix(s, "%"); ok {
		p.Percent, s = true, trimmed
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidOrderPrice, data)
	}
	p.Value = v
	return nil
}

// OrderParameters holds the parameters of an order, which Add Order takes for one order and Add Order Batch for each
// of its orders
type OrderParameters struct {
	// UserReference is a number grouping orders for queries and cancellation, unique or not; 0 is not sent. It excludes
	// ClientOrderID
	UserReference int32
	// ClientOrderID uniquely identifies an open order: a UUID, with or without its dashes, or up to 18 characters of
	// free text
	ClientOrderID string
	// OrderType is market, limit, iceberg, stop-loss, take-profit, stop-loss-limit, take-profit-limit, trailing-stop,
	// trailing-stop-limit or settle-position
	OrderType string
	// Side is buy or sell
	Side string
	// Volume is the quantity in the base currency, or in the quote currency with VolumeInQuote. A volume of 0 closes a
	// margin position in full, so only margin orders, which set Leverage, and settle-position orders may leave it 0
	Volume float64
	// DisplayVolume is the quantity an iceberg order shows in the book, at least a fifteenth of Volume
	DisplayVolume float64
	// Price is the limit price of limit and iceberg orders, and the trigger price of stop-loss, take-profit and trailing
	// stop orders and their limit variants. A trailing stop order's is relative, with a + offset
	Price OrderPrice
	// SecondaryPrice is the limit price of stop-loss-limit, take-profit-limit and trailing-stop-limit orders. A
	// trailing stop limit order's is an offset from its trigger price, with a + or - prefix
	SecondaryPrice OrderPrice
	// TriggerSignal is the price that triggers the order and its conditional close order: last, the default, or index.
	// The last price stands in for the index while index feeds are unavailable
	TriggerSignal string
	// Leverage makes the order a margin order with this leverage, such as 2 for 2:1
	Leverage uint64
	// ReduceOnly only lets the order reduce an open margin position
	ReduceOnly bool
	// SelfTradePrevention is which order a match against another of the account's orders cancels: cancel-newest, the
	// default, cancel-oldest or cancel-both
	SelfTradePrevention string
	// PostOnly only lets a limit order rest in the book
	PostOnly bool
	// FeeInBase prefers the fee in the base currency, the default when selling. It excludes FeeInQuote, the default when
	// buying
	FeeInBase  bool
	FeeInQuote bool
	// VolumeInQuote expresses Volume in the quote currency, which only buy market orders without leverage take
	VolumeInQuote bool
	// TimeInForce is GTC, the default, IOC, GTD, which needs an expiry, or FOK
	TimeInForce string
	// StartTime schedules the order, and StartDelay schedules it in whole seconds after Kraken receives it instead
	StartTime  time.Time
	StartDelay time.Duration
	// ExpireTime expires a GTD order, up to a month ahead, and ExpireDelay expires it in whole seconds, at least 5, after
	// Kraken receives it instead
	ExpireTime  time.Time
	ExpireDelay time.Duration
	// Close is an order Kraken places once this order fills
	Close *ConditionalCloseOrder
}

// ConditionalCloseOrder is an order Kraken places once its primary order fills, for the same volume in the opposite
// direction. Once placed it is independent, so it may increase a position as well as reduce it
type ConditionalCloseOrder struct {
	// OrderType is limit, iceberg, stop-loss, take-profit, stop-loss-limit, take-profit-limit, trailing-stop or
	// trailing-stop-limit
	OrderType      string
	Price          OrderPrice
	SecondaryPrice OrderPrice
}

// AddOrderRequest holds the parameters of Add Order
type AddOrderRequest struct {
	Pair currency.Pair
	// AssetClass is tokenized_asset, which xStocks pairs require
	AssetClass string
	Order      OrderParameters
	// Deadline rejects the order if the engine has not processed it by then, 2 to 60 seconds ahead, and is sent to the
	// second
	Deadline time.Time
	// Validate checks the order without placing it
	Validate bool
	// BrokerIIBAN is the IIBAN of a Kraken API partner
	BrokerIIBAN string
}

// AddOrderResponse holds a placed order's description and transaction ID, which an order that was only validated lacks
type AddOrderResponse struct {
	Description    AddOrderDescription `json:"descr"`
	TransactionIDs []string            `json:"txid"`
}

// AddOrderDescription describes a placed order and its conditional close order
type AddOrderDescription struct {
	Order string `json:"order"`
	Close string `json:"close"`
}

// AmendOrderRequest holds the parameters of Amend Order. Quantities and prices left unset keep their values
type AmendOrderRequest struct {
	// TransactionID identifies the order, and ClientOrderID identifies it by its client order ID instead
	TransactionID string
	ClientOrderID string
	OrderQuantity float64
	// DisplayQuantity is an iceberg order's visible quantity, at least a fifteenth of the remaining quantity
	DisplayQuantity float64
	// LimitPrice and TriggerPrice take + and - offsets from the reference price, but not #
	LimitPrice   OrderPrice
	TriggerPrice OrderPrice
	// Pair is required to amend an xStocks order
	Pair currency.Pair
	// PostOnly rejects a limit price amend that would not rest in the book
	PostOnly bool
	// Deadline rejects the amend if the engine has not processed it by then, 2 to 60 seconds ahead, and is sent to the
	// second
	Deadline time.Time
}

// AmendOrderResponse holds the ID of an amend transaction
type AmendOrderResponse struct {
	AmendID string `json:"amend_id"`
}

// EditOrderRequest holds the parameters of Edit Order. Volumes and prices left unset keep their values
type EditOrderRequest struct {
	// TransactionID identifies the order, and OrderUserReference identifies it by its user reference instead, which
	// Kraken refuses when several orders share it
	TransactionID      string
	OrderUserReference int32
	// NewUserReference is the replacement order's user reference, as the original order's is not kept
	NewUserReference int32
	Pair             currency.Pair
	// AssetClass is tokenized_asset, which xStocks pairs require
	AssetClass     string
	Volume         float64
	DisplayVolume  float64
	Price          OrderPrice
	SecondaryPrice OrderPrice
	// PostOnly makes the replacement post-only, which it is not otherwise even if the original order was; every other
	// flag of the original order is kept
	PostOnly bool
	// Deadline rejects the edit if the engine has not processed it by then, 2 to 60 seconds ahead, and is sent to the
	// second
	Deadline time.Time
	// CancelResponse asks for the pending replacement before the original order is replaced
	CancelResponse bool
	// Validate checks the edit without making it
	Validate bool
}

// EditOrderResponse holds the result of an order edit
type EditOrderResponse struct {
	Description EditOrderDescription `json:"descr"`
	// TransactionID is the replacement order's
	TransactionID string `json:"txid"`
	// NewUserReference and OldUserReference are documented as strings holding integer user references
	NewUserReference types.Number `json:"newuserref"`
	OldUserReference types.Number `json:"olduserref"`
	// OrdersCancelled is 1 once the original order is cancelled, and 0 otherwise
	OrdersCancelled       uint64       `json:"orders_cancelled"`
	OriginalTransactionID string       `json:"originaltxid"`
	Status                string       `json:"status"`
	Volume                types.Number `json:"volume"`
	// Price and SecondaryPrice keep the offset and percent sign of a relative price
	Price          OrderPrice `json:"price"`
	SecondaryPrice OrderPrice `json:"price2"`
	ErrorMessage   string     `json:"error_message"`
}

// EditOrderDescription describes a replacement order
type EditOrderDescription struct {
	Order string `json:"order"`
}

// CancelExistingOrderRequest holds the parameters of Cancel Order, which takes exactly one identifier
type CancelExistingOrderRequest struct {
	TransactionID string
	// UserReference cancels every open order with this user reference
	UserReference int32
	ClientOrderID string
}

// CancelOrderResponse holds the result of Cancel Order and Cancel All Orders
type CancelOrderResponse struct {
	Count uint64 `json:"count"`
	// Pending is true while the orders' cancellation is pending
	Pending bool `json:"pending"`
}

// CancelAllOrdersAfterResponse holds the dead man's switch timer
type CancelAllOrdersAfterResponse struct {
	CurrentTime time.Time
	// TriggerTime is when every order is cancelled unless the timer is extended or disabled, and zero once it is
	// disabled
	TriggerTime time.Time
}

// UnmarshalJSON decodes the timer's times, reading the trigger time Kraken sends as "0" once the timer is disabled as a
// zero time
func (c *CancelAllOrdersAfterResponse) UnmarshalJSON(data []byte) error {
	var resp struct {
		CurrentTime time.Time `json:"currentTime"`
		TriggerTime string    `json:"triggerTime"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	c.CurrentTime, c.TriggerTime = resp.CurrentTime, time.Time{}
	if resp.TriggerTime == "" || resp.TriggerTime == "0" {
		return nil
	}
	var err error
	c.TriggerTime, err = time.Parse(time.RFC3339, resp.TriggerTime)
	return err
}

// AddOrderBatchRequest holds the parameters of Add Order Batch
type AddOrderBatchRequest struct {
	Pair currency.Pair
	// AssetClass is tokenized_asset, which xStocks pairs require
	AssetClass string
	// Orders holds 2 to 15 orders
	Orders []OrderParameters
	// Deadline rejects the batch if the engine has not processed it by then, 2 to 60 seconds ahead, and is sent to the
	// second
	Deadline time.Time
	// Validate checks the orders without placing them
	Validate bool
	// BrokerIIBAN is the IIBAN of a Kraken API partner
	BrokerIIBAN string
}

// AddOrderBatchResponse holds the result of each order of a batch, in the order they were sent
type AddOrderBatchResponse struct {
	Orders []BatchOrderResult `json:"orders"`
}

// BatchOrderResult is the result of an order of a batch: its description and transaction ID once placed, or the error
// that rejected it
type BatchOrderResult struct {
	TransactionID string              `json:"txid"`
	Description   AddOrderDescription `json:"descr"`
	Error         string              `json:"error"`
}

// CancelOrderBatchRequest holds the parameters of Cancel Order Batch, which takes up to 50 identifiers in all
type CancelOrderBatchRequest struct {
	TransactionIDs []string
	// UserReferences cancel every open order with each user reference
	UserReferences []int32
	ClientOrderIDs []string
}

// CancelOrderBatchResponse holds the number of orders a batch cancelled
type CancelOrderBatchResponse struct {
	Count uint64 `json:"count"`
}

// WebsocketTokenResponse holds a websocket authentication token
type WebsocketTokenResponse struct {
	Token string `json:"token"`
	// ExpiresInSeconds is how long the token stays valid for connecting
	ExpiresInSeconds uint64 `json:"expires"`
}
