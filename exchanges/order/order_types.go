package order

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// var error definitions
var (
	ErrSubmissionIsNil            = errors.New("order submission is nil")
	ErrCancelOrderIsNil           = errors.New("cancel order is nil")
	ErrGetOrdersRequestIsNil      = errors.New("get order request is nil")
	ErrModifyOrderIsNil           = errors.New("modify order request is nil")
	ErrPairIsEmpty                = errors.New("order pair is empty")
	ErrAssetNotSet                = errors.New("order asset type is not set")
	ErrSideIsInvalid              = errors.New("order side is invalid")
	ErrTypeIsInvalid              = errors.New("order type is invalid")
	ErrAmountIsInvalid            = errors.New("order amount is equal or less than zero")
	ErrPriceMustBeSetIfLimitOrder = errors.New("order price must be set if limit order type is desired")
	ErrOrderIDNotSet              = errors.New("order id or client order id is not set")
)

// Submit contains all properties of an order that may be required
// for an order to be created on an exchange
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Submit struct {
	ImmediateOrCancel bool
	HiddenOrder       bool
	FillOrKill        bool
	PostOnly          bool
	ReduceOnly        bool
	Leverage          float64
	Price             float64
	Amount            float64
	StopPrice         float64
	LimitPriceUpper   float64
	LimitPriceLower   float64
	TriggerPrice      float64
	TargetAmount      float64
	ExecutedAmount    float64
	RemainingAmount   float64
	Fee               float64
	Exchange          string
	InternalOrderID   string
	ID                string
	AccountID         string
	ClientID          string
	ClientOrderID     string
	WalletAddress     string
	Offset            string
	Type              Type
	Side              Side
	Status            Status
	AssetType         asset.Item
	Date              time.Time
	LastUpdated       time.Time
	Pair              currency.Pair
	Trades            []TradeHistory
}

// SubmitResponse is what is returned after submitting an order to an exchange
type SubmitResponse struct {
	IsOrderPlaced bool
	FullyMatched  bool
	OrderID       string
	Rate          float64
	Fee           float64
	Cost          float64
	Trades        []TradeHistory
}

// Modify contains all properties of an order
// that may be updated after it has been created
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Modify struct {
	ImmediateOrCancel bool
	HiddenOrder       bool
	FillOrKill        bool
	PostOnly          bool
	Leverage          float64
	Price             float64
	Amount            float64
	LimitPriceUpper   float64
	LimitPriceLower   float64
	TriggerPrice      float64
	TargetAmount      float64
	ExecutedAmount    float64
	RemainingAmount   float64
	Fee               float64
	Exchange          string
	InternalOrderID   string
	ID                string
	ClientOrderID     string
	AccountID         string
	ClientID          string
	WalletAddress     string
	Type              Type
	Side              Side
	Status            Status
	AssetType         asset.Item
	Date              time.Time
	LastUpdated       time.Time
	Pair              currency.Pair
	Trades            []TradeHistory
}

// ModifyResponse is an order modifying return type
type ModifyResponse struct {
	OrderID string
}

// Detail contains all properties of an order
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Detail struct {
	ImmediateOrCancel bool
	HiddenOrder       bool
	FillOrKill        bool
	PostOnly          bool
	Leverage          float64
	Price             float64
	Amount            float64
	LimitPriceUpper   float64
	LimitPriceLower   float64
	TriggerPrice      float64
	TargetAmount      float64
	ExecutedAmount    float64
	RemainingAmount   float64
	Cost              float64
	Fee               float64
	Exchange          string
	InternalOrderID   string
	ID                string
	ClientOrderID     string
	AccountID         string
	ClientID          string
	WalletAddress     string
	Type              Type
	Side              Side
	Status            Status
	AssetType         asset.Item
	Date              time.Time
	CloseTime         time.Time
	LastUpdated       time.Time
	Pair              currency.Pair
	Trades            []TradeHistory
}

// Filter contains all properties an order can be filtered for
// empty strings indicate to ignore the property otherwise all need to match
type Filter struct {
	Exchange        string
	InternalOrderID string
	ID              string
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
	Price         float64
	Amount        float64
	Exchange      string
	ID            string
	ClientOrderID string
	AccountID     string
	ClientID      string
	WalletAddress string
	Type          Type
	Side          Side
	Status        Status
	AssetType     asset.Item
	Date          time.Time
	Pair          currency.Pair
	Symbol        string
	Trades        []TradeHistory
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

// GetOrdersRequest used for GetOrderHistory and GetOpenOrders wrapper functions
type GetOrdersRequest struct {
	Type      Type
	Side      Side
	StartTime time.Time
	EndTime   time.Time
	OrderID   string
	// Currencies Empty array = all currencies. Some endpoints only support
	// singular currency enquiries
	Pairs     currency.Pairs
	AssetType asset.Item
}

// Status defines order status types
type Status string

// All order status types
const (
	AnyStatus           Status = "ANY"
	New                 Status = "NEW"
	Active              Status = "ACTIVE"
	PartiallyCancelled  Status = "PARTIALLY_CANCELLED"
	PartiallyFilled     Status = "PARTIALLY_FILLED"
	Filled              Status = "FILLED"
	Cancelled           Status = "CANCELLED"
	PendingCancel       Status = "PENDING_CANCEL"
	InsufficientBalance Status = "INSUFFICIENT_BALANCE"
	MarketUnavailable   Status = "MARKET_UNAVAILABLE"
	Rejected            Status = "REJECTED"
	Expired             Status = "EXPIRED"
	Hidden              Status = "HIDDEN"
	UnknownStatus       Status = "UNKNOWN"
	Open                Status = "OPEN"
	AutoDeleverage      Status = "ADL"
	Closed              Status = "CLOSED"
	Pending             Status = "PENDING"
)

// Type enforces a standard for order types across the code base
type Type string

// Defined package order types
const (
	AnyType           Type = "ANY"
	Limit             Type = "LIMIT"
	Market            Type = "MARKET"
	PostOnly          Type = "POST_ONLY"
	ImmediateOrCancel Type = "IMMEDIATE_OR_CANCEL"
	Stop              Type = "STOP"
	StopLimit         Type = "STOP LIMIT"
	StopMarket        Type = "STOP MARKET"
	TakeProfit        Type = "TAKE PROFIT"
	TakeProfitMarket  Type = "TAKE PROFIT MARKET"
	TrailingStop      Type = "TRAILING_STOP"
	FillOrKill        Type = "FOK"
	IOS               Type = "IOS"
	UnknownType       Type = "UNKNOWN"
	Liquidation       Type = "LIQUIDATION"
	Trigger           Type = "TRIGGER"
)

// Side enforces a standard for order sides across the code base
type Side string

// Order side types
const (
	AnySide     Side = "ANY"
	Buy         Side = "BUY"
	Sell        Side = "SELL"
	Bid         Side = "BID"
	Ask         Side = "ASK"
	UnknownSide Side = "UNKNOWN"
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
