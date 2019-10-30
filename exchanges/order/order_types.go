package order

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	limitOrder = iota
	marketOrder
)

// Orders variable holds an array of pointers to order structs
var Orders []*Order

// Order struct holds order values
type Order struct {
	OrderID  int
	Exchange string
	Type     int
	Amount   float64
	Price    float64
}

// vars related to orders
var (
	ErrSubmissionIsNil            = errors.New("order submission is nil")
	ErrPairIsEmpty                = errors.New("order pair is empty")
	ErrSideIsInvalid              = errors.New("order side is invalid")
	ErrTypeIsInvalid              = errors.New("order type is invalid")
	ErrAmountIsInvalid            = errors.New("order amount is invalid")
	ErrPriceMustBeSetIfLimitOrder = errors.New("order price must be set if limit order type is desired")
)

// Submit contains the order submission data
type Submit struct {
	Pair      currency.Pair
	OrderType Type
	OrderSide Side
	Price     float64
	Amount    float64
	ClientID  string
}

// SubmitResponse is what is returned after submitting an order to an exchange
type SubmitResponse struct {
	IsOrderPlaced bool
	OrderID       string
}

// Modify is an order modifyer
type Modify struct {
	OrderID string
	Type
	Side
	Price             float64
	Amount            float64
	LimitPriceUpper   float64
	LimitPriceLower   float64
	CurrencyPair      currency.Pair
	ImmediateOrCancel bool
	HiddenOrder       bool
	FillOrKill        bool
	PostOnly          bool
}

// ModifyResponse is an order modifying return type
type ModifyResponse struct {
	OrderID string
}

// CancelAllResponse returns the status from attempting to cancel all orders on
// an exchagne
type CancelAllResponse struct {
	Status map[string]string
}

// Type enforces a standard for order types across the code base
type Type string

// Defined package order types
const (
	AnyType           Type = "ANY"
	Limit             Type = "LIMIT"
	Market            Type = "MARKET"
	ImmediateOrCancel Type = "IMMEDIATE_OR_CANCEL"
	Stop              Type = "STOP"
	TrailingStop      Type = "TRAILINGSTOP"
	Unknown           Type = "UNKNOWN"
)

// Side enforces a standard for order sides across the code base
type Side string

// Order side types
const (
	AnySide Side = "ANY"
	Buy     Side = "BUY"
	Sell    Side = "SELL"
	Bid     Side = "BID"
	Ask     Side = "ASK"
)

// Detail holds order detail data
type Detail struct {
	Exchange     string
	AccountID    string
	ID           string
	CurrencyPair currency.Pair
	OrderSide    Side
	OrderType    Type
	OrderDate    time.Time
	Status
	Price           float64
	Amount          float64
	ExecutedAmount  float64
	RemainingAmount float64
	Fee             float64
	Trades          []TradeHistory
}

// TradeHistory holds exchange history data
type TradeHistory struct {
	Timestamp time.Time
	TID       int64
	Price     float64
	Amount    float64
	Exchange  string
	Type
	Side
	Fee         float64
	Description string
}

// Cancel type required when requesting to cancel an order
type Cancel struct {
	AccountID     string
	OrderID       string
	CurrencyPair  currency.Pair
	AssetType     asset.Item
	WalletAddress string
	Side
}

// GetOrdersRequest used for GetOrderHistory and GetOpenOrders wrapper functions
type GetOrdersRequest struct {
	OrderType  Type
	OrderSide  Side
	StartTicks time.Time
	EndTicks   time.Time
	// Currencies Empty array = all currencies. Some endpoints only support
	// singular currency enquiries
	Currencies []currency.Pair
}

// Status defines order status types
type Status string

// All order status types
const (
	AnyStatus       Status = "ANY"
	New             Status = "NEW"
	Active          Status = "ACTIVE"
	PartiallyFilled Status = "PARTIALLY_FILLED"
	Filled          Status = "FILLED"
	Cancelled       Status = "CANCELED"
	PendingCancel   Status = "PENDING_CANCEL"
	Rejected        Status = "REJECTED"
	Expired         Status = "EXPIRED"
	Hidden          Status = "HIDDEN"
	UnknownStatus   Status = "UNKNOWN"
)

// ByPrice used for sorting orders by price
type ByPrice []Detail

// ByOrderType used for sorting orders by order type
type ByOrderType []Detail

// ByCurrency used for sorting orders by order currency
type ByCurrency []Detail

// ByDate used for sorting orders by order date
type ByDate []Detail

// ByOrderSide used for sorting orders by order side (buy sell)
type ByOrderSide []Detail
