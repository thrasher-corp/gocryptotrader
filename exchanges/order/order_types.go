package order

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

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

// Validate checks the supplied data and returns whether or not it's valid
func (s *Submit) Validate() error {
	if s == nil {
		return ErrSubmissionIsNil
	}

	if s.Pair.IsEmpty() {
		return ErrPairIsEmpty
	}

	if s.OrderSide != Buy &&
		s.OrderSide != Sell &&
		s.OrderSide != Bid &&
		s.OrderSide != Ask {
		return ErrSideIsInvalid
	}

	if s.OrderType != Market && s.OrderType != Limit {
		return ErrTypeIsInvalid
	}

	if s.Amount <= 0 {
		return ErrAmountIsInvalid
	}

	if s.OrderType == Limit && s.Price <= 0 {
		return ErrPriceMustBeSetIfLimitOrder
	}

	return nil
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

// String implements the stringer interface
func (t Type) String() string {
	return string(t)
}

// Lower returns the type lower case string
func (t Type) Lower() string {
	return strings.ToLower(string(t))
}

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

// String implements the stringer interface
func (s Side) String() string {
	return string(s)
}

// Lower returns the side lower case string
func (s Side) Lower() string {
	return strings.ToLower(string(s))
}

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

// Cancellation type required when requesting to cancel an order
type Cancellation struct {
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

// String implements the stringer interface
func (s Status) String() string {
	return string(s)
}

// FilterOrdersBySide removes any order details that don't match the
// order status provided
func FilterOrdersBySide(orders *[]Detail, side Side) {
	if side == "" || side == AnySide {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].OrderSide), string(side)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByType removes any order details that don't match the order type
// provided
func FilterOrdersByType(orders *[]Detail, orderType Type) {
	if orderType == "" || orderType == AnyType {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].OrderType), string(orderType)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByTickRange removes any OrderDetails outside of the tick range
func FilterOrdersByTickRange(orders *[]Detail, startTicks, endTicks time.Time) {
	if startTicks.IsZero() ||
		endTicks.IsZero() ||
		startTicks.Unix() == 0 ||
		endTicks.Unix() == 0 ||
		endTicks.Before(startTicks) {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		if (*orders)[i].OrderDate.Unix() >= startTicks.Unix() &&
			(*orders)[i].OrderDate.Unix() <= endTicks.Unix() {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByCurrencies removes any order details that do not match the
// provided currency list. It is forgiving in that the provided currencies can
// match quote or base currencies
func FilterOrdersByCurrencies(orders *[]Detail, currencies []currency.Pair) {
	if len(currencies) == 0 {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		matchFound := false
		for _, c := range currencies {
			if !matchFound && (*orders)[i].CurrencyPair.EqualIncludeReciprocal(c) {
				matchFound = true
			}
		}

		if matchFound {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// ByPrice used for sorting orders by price
type ByPrice []Detail

func (b ByPrice) Len() int {
	return len(b)
}

func (b ByPrice) Less(i, j int) bool {
	return b[i].Price < b[j].Price
}

func (b ByPrice) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByPrice the caller function to sort orders
func SortOrdersByPrice(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByPrice(*orders)))
	} else {
		sort.Sort(ByPrice(*orders))
	}
}

// ByOrderType used for sorting orders by order type
type ByOrderType []Detail

func (b ByOrderType) Len() int {
	return len(b)
}

func (b ByOrderType) Less(i, j int) bool {
	return b[i].OrderType.String() < b[j].OrderType.String()
}

func (b ByOrderType) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByType the caller function to sort orders
func SortOrdersByType(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderType(*orders)))
	} else {
		sort.Sort(ByOrderType(*orders))
	}
}

// ByCurrency used for sorting orders by order currency
type ByCurrency []Detail

func (b ByCurrency) Len() int {
	return len(b)
}

func (b ByCurrency) Less(i, j int) bool {
	return b[i].CurrencyPair.String() < b[j].CurrencyPair.String()
}

func (b ByCurrency) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByCurrency the caller function to sort orders
func SortOrdersByCurrency(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByCurrency(*orders)))
	} else {
		sort.Sort(ByCurrency(*orders))
	}
}

// ByDate used for sorting orders by order date
type ByDate []Detail

func (b ByDate) Len() int {
	return len(b)
}

func (b ByDate) Less(i, j int) bool {
	return b[i].OrderDate.Unix() < b[j].OrderDate.Unix()
}

func (b ByDate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByDate the caller function to sort orders
func SortOrdersByDate(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByDate(*orders)))
	} else {
		sort.Sort(ByDate(*orders))
	}
}

// ByOrderSide used for sorting orders by order side (buy sell)
type ByOrderSide []Detail

func (b ByOrderSide) Len() int {
	return len(b)
}

func (b ByOrderSide) Less(i, j int) bool {
	return b[i].OrderSide.String() < b[j].OrderSide.String()
}

func (b ByOrderSide) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersBySide the caller function to sort orders
func SortOrdersBySide(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderSide(*orders)))
	} else {
		sort.Sort(ByOrderSide(*orders))
	}
}
