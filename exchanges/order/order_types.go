package order

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// var error definitions
var (
	ErrSubmissionIsNil            = errors.New("order submission is nil")
	ErrCancelOrderIsNil           = errors.New("cancel order is nil")
	ErrOrderDetailIsNil           = errors.New("order detail is nil")
	ErrGetOrdersRequestIsNil      = errors.New("get order request is nil")
	ErrModifyOrderIsNil           = errors.New("modify order request is nil")
	ErrPairIsEmpty                = errors.New("order pair is empty")
	ErrAssetNotSet                = errors.New("order asset type is not set")
	ErrSideIsInvalid              = errors.New("order side is invalid")
	ErrTypeIsInvalid              = errors.New("order type is invalid")
	ErrAmountIsInvalid            = errors.New("order amount is equal or less than zero")
	ErrPriceMustBeSetIfLimitOrder = errors.New("order price must be set if limit order type is desired")
	ErrOrderIDNotSet              = errors.New("order id or client order id is not set")
	ErrClientOrderIDNotSupported  = errors.New("client order id not supported")
	ErrUnsupportedOrderType       = errors.New("unsupported order type")
	// ErrNoRates is returned when no margin rates are returned when they are expected
	ErrNoRates = errors.New("no rates")

	errCannotLiquidate = errors.New("cannot liquidate position")
)

// Submit contains all properties of an order that may be required
// for an order to be created on an exchange
// Each exchange has their own requirements, so not all fields
// need to be populated
type Submit struct {
	Exchange  string
	Type      Type
	Side      Side
	Pair      currency.Pair
	AssetType asset.Item

	// Time in force values ------ TODO: Time In Force uint8
	ImmediateOrCancel bool
	FillOrKill        bool

	PostOnly bool
	// ReduceOnly reduces a position instead of opening an opposing
	// position; this also equates to closing the position in huobi_wrapper.go
	// swaps.
	ReduceOnly bool
	// Leverage is the amount of leverage that will be used: see huobi_wrapper.go
	Leverage float64
	Price    float64
	// Amount in base terms
	Amount float64
	// QuoteAmount is the max amount in quote currency when purchasing base.
	// This is only used in Market orders.
	QuoteAmount float64
	// TriggerPrice is mandatory if order type `Stop, Stop Limit or Take Profit`
	// See btcmarkets_wrapper.go.
	TriggerPrice  float64
	ClientID      string // TODO: Shift to credentials
	ClientOrderID string
	// RetrieveFees use if an API submit order response does not return fees
	// enabling this will perform additional request(s) to retrieve them
	// and set it in the SubmitResponse
	RetrieveFees bool
	// RetrieveFeeDelay some exchanges take time to properly save order data
	// and cannot retrieve fees data immediately
	RetrieveFeeDelay time.Duration
	// TradeMode specifies the trading mode for margin and non-margin orders: see okcoin_wrapper.go
	TradeMode string
}

// SubmitResponse is what is returned after submitting an order to an exchange
type SubmitResponse struct {
	Exchange  string
	Type      Type
	Side      Side
	Pair      currency.Pair
	AssetType asset.Item

	ImmediateOrCancel bool
	FillOrKill        bool
	PostOnly          bool
	ReduceOnly        bool
	Leverage          float64
	Price             float64
	Amount            float64
	QuoteAmount       float64
	TriggerPrice      float64
	ClientID          string
	ClientOrderID     string

	LastUpdated time.Time
	Date        time.Time
	Status      Status
	OrderID     string
	Trades      []TradeHistory
	Fee         float64
	FeeAsset    currency.Code
	Cost        float64
}

// Modify contains all properties of an order
// that may be updated after it has been created
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Modify struct {
	// Order Identifiers
	Exchange      string
	OrderID       string
	ClientOrderID string
	Type          Type
	Side          Side
	AssetType     asset.Item
	Pair          currency.Pair

	// Change fields
	ImmediateOrCancel bool
	PostOnly          bool
	Price             float64
	Amount            float64
	TriggerPrice      float64
}

// ModifyResponse is an order modifying return type
type ModifyResponse struct {
	// Order Identifiers
	Exchange      string
	OrderID       string
	ClientOrderID string
	Pair          currency.Pair
	Type          Type
	Side          Side
	Status        Status
	AssetType     asset.Item

	// Fields that will be copied over from Modify
	ImmediateOrCancel bool
	PostOnly          bool
	Price             float64
	Amount            float64
	TriggerPrice      float64

	// Fields that need to be handled in scope after DeriveModifyResponse()
	// if applicable
	RemainingAmount float64
	Date            time.Time
	LastUpdated     time.Time
}

// Detail contains all properties of an order
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Detail struct {
	ImmediateOrCancel    bool
	HiddenOrder          bool
	FillOrKill           bool
	PostOnly             bool
	ReduceOnly           bool
	Leverage             float64
	Price                float64
	Amount               float64
	LimitPriceUpper      float64
	LimitPriceLower      float64
	TriggerPrice         float64
	AverageExecutedPrice float64
	QuoteAmount          float64
	ExecutedAmount       float64
	RemainingAmount      float64
	Cost                 float64
	CostAsset            currency.Code
	Fee                  float64
	FeeAsset             currency.Code
	Exchange             string
	InternalOrderID      uuid.UUID
	OrderID              string
	ClientOrderID        string
	AccountID            string
	ClientID             string
	WalletAddress        string
	Type                 Type
	Side                 Side
	Status               Status
	AssetType            asset.Item
	Date                 time.Time
	CloseTime            time.Time
	LastUpdated          time.Time
	Pair                 currency.Pair
	Trades               []TradeHistory
}

// Filter contains all properties an order can be filtered for
// empty strings indicate to ignore the property otherwise all need to match
type Filter struct {
	Exchange        string
	InternalOrderID uuid.UUID
	OrderID         string
	ClientOrderID   string
	AccountID       string
	ClientID        string
	WalletAddress   string
	Type            Type
	Side            Side
	Status          Status
	AssetType       asset.Item
	Pair            currency.Pair
}

// Cancel contains all properties that may be required
// to cancel an order on an exchange
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Cancel struct {
	Exchange      string
	OrderID       string
	ClientOrderID string
	AccountID     string
	ClientID      string
	WalletAddress string
	Type          Type
	Side          Side
	AssetType     asset.Item
	Pair          currency.Pair
}

// CancelAllResponse returns the status from attempting to
// cancel all orders on an exchange
type CancelAllResponse struct {
	Status map[string]string
	Count  int64
}

// CancelBatchResponse returns the status of orders
// that have been requested for cancellation
type CancelBatchResponse struct {
	Status map[string]string
}

// TradeHistory holds exchange history data
type TradeHistory struct {
	Price       float64
	Amount      float64
	Fee         float64
	Exchange    string
	TID         string
	Description string
	Type        Type
	Side        Side
	Timestamp   time.Time
	IsMaker     bool
	FeeAsset    string
	Total       float64
}

// MultiOrderRequest used for GetOrderHistory and GetOpenOrders wrapper functions
type MultiOrderRequest struct {
	// Currencies Empty array = all currencies. Some endpoints only support
	// singular currency enquiries
	Pairs     currency.Pairs
	AssetType asset.Item
	Type      Type
	Side      Side
	StartTime time.Time
	EndTime   time.Time
	// FromOrderID for some APIs require order history searching
	// from a specific orderID rather than via timestamps
	FromOrderID string
}

// Status defines order status types
type Status uint32

// All order status types
const (
	UnknownStatus Status = 0
	AnyStatus     Status = 1 << iota
	New
	Active
	PartiallyCancelled
	PartiallyFilled
	Filled
	Cancelled
	PendingCancel
	InsufficientBalance
	MarketUnavailable
	Rejected
	Expired
	Hidden
	Open
	AutoDeleverage
	Closed
	Pending
	Cancelling
	Liquidated
)

// Type enforces a standard for order types across the code base
type Type uint32

// Defined package order types
const (
	UnknownType Type = 0
	Limit       Type = 1 << iota
	Market
	PostOnly
	ImmediateOrCancel
	Stop
	StopLimit
	StopMarket
	TakeProfit
	TakeProfitMarket
	TrailingStop
	FillOrKill
	IOS
	AnyType
	Liquidation
	Trigger
	OptimalLimitIOC
	OCO             // One-cancels-the-other order
	ConditionalStop // One-way stop order
)

// Side enforces a standard for order sides across the code base
type Side uint32

// Order side types
const (
	UnknownSide Side = 0
	Buy         Side = 1 << iota
	Sell
	Bid
	Ask
	AnySide
	Long
	Short
	ClosePosition
	// Backtester signal types
	DoNothing
	TransferredFunds
	CouldNotBuy
	CouldNotSell
	CouldNotShort
	CouldNotLong
	CouldNotCloseShort
	CouldNotCloseLong
	MissingData
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

// ClassificationError returned when an order status
// side or type cannot be recognised
type ClassificationError struct {
	Exchange string
	OrderID  string
	Err      error
}

// FilteredOrders defines orders that have been filtered at the wrapper level
// forcing required filter operations when calling method Filter() on
// MultiOrderRequest.
type FilteredOrders []Detail
