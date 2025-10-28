package order

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
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
	ErrCollateralInvalid          = errors.New("collateral type is invalid")
	ErrTypeIsInvalid              = errors.New("order type is invalid")
	ErrAmountIsInvalid            = errors.New("order amount is equal or less than zero")
	ErrPriceMustBeSetIfLimitOrder = errors.New("order price must be set if limit order type is desired")
	ErrOrderIDNotSet              = errors.New("order id or client order id is not set")
	ErrSubmitLeverageNotSupported = errors.New("leverage is not supported via order submission")
	ErrClientOrderIDNotSupported  = errors.New("client order id not supported")
	ErrUnsupportedOrderType       = errors.New("unsupported order type")
	ErrUnsupportedStatusType      = errors.New("unsupported status type")
	// ErrNoRates is returned when no margin rates are returned when they are expected
	ErrNoRates         = errors.New("no rates")
	ErrCannotLiquidate = errors.New("cannot liquidate position")

	ErrUnknownTrackingMode = errors.New("unknown tracking mode")
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

	// TimeInForce holds time in force values
	TimeInForce TimeInForce

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
	TriggerPrice float64

	// added to represent a unified trigger price type information such as LastPrice, MarkPrice, and IndexPrice
	// https://bybit-exchange.github.io/docs/v5/order/create-order
	TriggerPriceType PriceType
	ClientID         string // TODO: Shift to credentials
	ClientOrderID    string

	// The system will first borrow you funds at the optimal interest rate and then place an order for you.
	// see kucoin_wrapper.go
	AutoBorrow bool

	// MarginType such as isolated or cross margin for when an exchange
	// supports margin type definition when submitting an order eg okx
	MarginType margin.Type
	// RetrieveFees use if an API submit order response does not return fees
	// enabling this will perform additional request(s) to retrieve them
	// and set it in the SubmitResponse
	RetrieveFees bool
	// RetrieveFeeDelay some exchanges take time to properly save order data
	// and cannot retrieve fees data immediately
	RetrieveFeeDelay    time.Duration
	RiskManagementModes RiskManagementModes

	// Hidden when enabled orders not displaying in order book.
	Hidden bool

	// Iceberg specifies whether or not only visible portions of orders are shown in iceberg orders
	Iceberg bool

	// EndTime is the moment which a good til date order is valid until
	EndTime time.Time

	// StopDirection is the direction from which the stop order will trigger
	StopDirection StopDirection
	// TrackingMode specifies the way trailing stop and chase orders follow the market price or ask/bid prices.
	// See: https://www.okx.com/docs-v5/en/#order-book-trading-algo-trading-post-place-algo-order
	TrackingMode  TrackingMode
	TrackingValue float64

	// RFQDisabled, when set, attempts to route the order to the exchange CLOB. Currently only supported by Coinbase
	RFQDisabled bool
}

// SubmitResponse is what is returned after submitting an order to an exchange
type SubmitResponse struct {
	Exchange             string
	Type                 Type
	Side                 Side
	Pair                 currency.Pair
	AssetType            asset.Item
	TimeInForce          TimeInForce
	ReduceOnly           bool
	Leverage             float64
	Price                float64
	Amount               float64
	QuoteAmount          float64
	RemainingAmount      float64
	TriggerPrice         float64
	ClientID             string
	ClientOrderID        string
	AverageExecutedPrice float64
	LastUpdated          time.Time
	Date                 time.Time
	Status               Status
	OrderID              string
	Trades               []TradeHistory
	Fee                  float64
	FeeAsset             currency.Code
	Cost                 float64
	Purchased            float64 // Buy in base currency, Sell in quote
	BorrowSize           float64
	LoanApplyID          string
	MarginType           margin.Type
	SubmissionError      error
}

// TrackingMode defines how the stop price follows the market price.
type TrackingMode uint8

// Defined package tracking modes
const (
	UnknownTrackingMode TrackingMode = iota
	Distance                         // Distance fixed amount away from the market price
	Percentage                       // Percentage fixed percentage away from the market price
)

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
	TimeInForce  TimeInForce
	Price        float64
	Amount       float64
	TriggerPrice float64

	// added to represent a unified trigger price type information such as LastPrice, MarkPrice, and IndexPrice
	// https://bybit-exchange.github.io/docs/v5/order/create-order
	TriggerPriceType PriceType

	RiskManagementModes RiskManagementModes
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
	TimeInForce  TimeInForce
	Price        float64
	Amount       float64
	TriggerPrice float64

	// Fields that need to be handled in scope after DeriveModifyResponse()
	// if applicable
	RemainingAmount float64
	Date            time.Time
	LastUpdated     time.Time
}

// Detail contains all properties of an order
// Each exchange has their own requirements, so not all fields are required to be populated
type Detail struct {
	HiddenOrder          bool
	TimeInForce          TimeInForce
	ReduceOnly           bool
	Leverage             float64
	Price                float64
	Amount               float64
	ContractAmount       float64
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
	Type                 Type
	Side                 Side
	Status               Status
	AssetType            asset.Item
	Date                 time.Time
	CloseTime            time.Time
	LastUpdated          time.Time
	Pair                 currency.Pair
	MarginType           margin.Type
	Trades               []TradeHistory
	SettlementCurrency   currency.Code
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
	Type          Type
	Side          Side
	AssetType     asset.Item
	Pair          currency.Pair
	MarginType    margin.Type
	TimeInForce   TimeInForce
}

// CancelAllResponse returns the status from attempting to
// cancel all orders on an exchange
type CancelAllResponse struct {
	Status map[string]string
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
	Pairs       currency.Pairs
	AssetType   asset.Item
	Type        Type
	Side        Side
	TimeInForce TimeInForce
	StartTime   time.Time
	EndTime     time.Time
	// FromOrderID for some APIs require order history searching
	// from a specific orderID rather than via timestamps
	FromOrderID string

	MarginType margin.Type
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
	PartiallyFilledCancelled
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
	STP
)

// Type enforces a standard for order types across the code base
type Type uint64

// Is checks to see if the Type contains the Type cmp
func (t Type) Is(cmp Type) bool {
	return cmp != 0 && t&cmp == cmp
}

// Defined package order types
const (
	UnknownType Type = 0
	Limit       Type = 1 << iota
	Market
	Stop
	TakeProfit
	TrailingStop
	IOS
	AnyType
	Liquidation
	Trigger
	OCO             // One-cancels-the-other order
	ConditionalStop // One-way stop order
	TWAP            // time-weighted average price
	Chase           // chase limit order
	OptimalLimit
	MarketMakerProtection

	// Hybrid order types
	StopLimit        = Stop | Limit
	StopMarket       = Stop | Market
	TakeProfitMarket = TakeProfit | Market
	Bracket          = Stop | TakeProfit
)

// order-type string representations
const (
	orderStopMarket            = "STOP MARKET"
	orderStopLimit             = "STOP LIMIT"
	orderLimit                 = "LIMIT"
	orderMarket                = "MARKET"
	orderStop                  = "STOP"
	orderConditionalStop       = "CONDITIONAL"
	orderTWAP                  = "TWAP"
	orderChase                 = "CHASE"
	orderTakeProfit            = "TAKE PROFIT"
	orderTakeProfitMarket      = "TAKE PROFIT MARKET"
	orderTrailingStop          = "TRAILING_STOP"
	orderIOS                   = "IOS"
	orderLiquidation           = "LIQUIDATION"
	orderTrigger               = "TRIGGER"
	orderOCO                   = "OCO"
	orderOptimalLimit          = "OPTIMAL_LIMIT"
	orderMarketMakerProtection = "MMP"
	orderBracket               = "BRACKET"
	orderAnyType               = "ANY"
)

// AllOrderTypes collects all order types for easy and consistent comparisons
var AllOrderTypes = Limit |
	Market |
	Stop |
	StopLimit |
	StopMarket |
	TakeProfit |
	TakeProfitMarket |
	TrailingStop |
	IOS |
	AnyType |
	Liquidation |
	Trigger |
	OCO |
	ConditionalStop |
	TWAP |
	Chase |
	OptimalLimit |
	MarketMakerProtection |
	Bracket

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

// StopDirection is the direction from which the stop order will trigger; Up will have the order trigger
// when the last trade price goes above the TriggerPrice; Down will have the order trigger when the
// last trade price goes below the TriggerPrice
type StopDirection bool

// StopDirection types
const (
	StopUp   StopDirection = true
	StopDown StopDirection = false
)

// RiskManagement represents a risk management detail information.
type RiskManagement struct {
	Enabled          bool
	TriggerPriceType PriceType
	Price            float64

	// LimitPrice limit order price when stop-loss or take-profit risk management method is triggered
	LimitPrice float64
	// OrderType order type when stop-loss or take-profit risk management method is triggered.
	OrderType Type
}

// RiskManagementModes represents take-profit and stop-loss risk management methods.
type RiskManagementModes struct {
	// Mode take-profit/stop-loss mode
	Mode       string
	TakeProfit RiskManagement
	StopLoss   RiskManagement

	// StopEntry stop: 'entry': Triggers when the last trade price changes to a value at or above the stopPrice.
	// see: https://www.kucoin.com/docs/rest/spot-trading/stop-order/introduction
	StopEntry RiskManagement
}

// PriceType enforces a standard for price types used for take-profit and stop-loss trigger types
type PriceType uint8

// price types
const (
	LastPrice  PriceType = 0
	IndexPrice PriceType = 1 << iota
	MarkPrice
	UnknownPriceType
)
