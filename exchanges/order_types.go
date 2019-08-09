package exchange

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// vars related to orders
var (
	ErrOrderSubmissionIsNil            = errors.New("order submission is nil")
	ErrOrderPairIsEmpty                = errors.New("order pair is empty")
	ErrOrderSideIsInvalid              = errors.New("order side is invalid")
	ErrOrderTypeIsInvalid              = errors.New("order type is invalid")
	ErrOrderAmountIsInvalid            = errors.New("order amount is invalid")
	ErrOrderPriceMustBeSetIfLimitOrder = errors.New("order price must be set if limit order type is desired")
)

// OrderSubmission contains the order submission data
type OrderSubmission struct {
	Pair      currency.Pair
	OrderSide OrderSide
	OrderType OrderType
	Price     float64
	Amount    float64
	ClientID  string
}

// Validate checks the supplied data and returns whether or not its valid
func (o *OrderSubmission) Validate() error {
	if o.Pair.IsEmpty() {
		return ErrOrderPairIsEmpty
	}

	o.OrderSide = OrderSide(strings.ToUpper(o.OrderSide.ToString()))
	if o.OrderSide != BuyOrderSide && o.OrderSide != SellOrderSide &&
		o.OrderSide != BidOrderSide && o.OrderSide != AskOrderSide {
		return ErrOrderSideIsInvalid
	}

	o.OrderType = OrderType(strings.ToUpper(o.OrderType.ToString()))
	if o.OrderType != MarketOrderType && o.OrderType != LimitOrderType {
		return ErrOrderTypeIsInvalid
	}

	if o.Amount <= 0 {
		return ErrOrderAmountIsInvalid
	}

	if o.OrderType == LimitOrderType && o.Price <= 0 {
		return ErrOrderPriceMustBeSetIfLimitOrder
	}

	return nil
}

// SubmitOrderResponse is what is returned after submitting an order to an exchange
type SubmitOrderResponse struct {
	IsOrderPlaced bool
	OrderID       string
}

// ModifyOrder is a an order modifyer
type ModifyOrder struct {
	OrderID string
	OrderType
	OrderSide
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

// ModifyOrderResponse is an order modifying return type
type ModifyOrderResponse struct {
	OrderID string
}

// CancelAllOrdersResponse returns the status from attempting to cancel all orders on an exchagne
type CancelAllOrdersResponse struct {
	OrderStatus map[string]string
}

// OrderType enforces a standard for Ordertypes across the code base
type OrderType string

// OrderType ...types
const (
	AnyOrderType               OrderType = "ANY"
	LimitOrderType             OrderType = "LIMIT"
	MarketOrderType            OrderType = "MARKET"
	ImmediateOrCancelOrderType OrderType = "IMMEDIATE_OR_CANCEL"
	StopOrderType              OrderType = "STOP"
	TrailingStopOrderType      OrderType = "TRAILINGSTOP"
	UnknownOrderType           OrderType = "UNKNOWN"
)

// ToLower changes the ordertype to lower case
func (o OrderType) ToLower() OrderType {
	return OrderType(strings.ToLower(string(o)))
}

// ToString changes the ordertype to the exchange standard and returns a string
func (o OrderType) ToString() string {
	return fmt.Sprintf("%v", o)
}

// OrderSide enforces a standard for OrderSides across the code base
type OrderSide string

// OrderSide types
const (
	AnyOrderSide  OrderSide = "ANY"
	BuyOrderSide  OrderSide = "BUY"
	SellOrderSide OrderSide = "SELL"
	BidOrderSide  OrderSide = "BID"
	AskOrderSide  OrderSide = "ASK"
)

// ToLower changes the ordertype to lower case
func (o OrderSide) ToLower() OrderSide {
	return OrderSide(strings.ToLower(string(o)))
}

// ToString changes the ordertype to the exchange standard and returns a string
func (o OrderSide) ToString() string {
	return fmt.Sprintf("%v", o)
}

// OrderDetail holds order detail data
type OrderDetail struct {
	Exchange        string
	AccountID       string
	ID              string
	CurrencyPair    currency.Pair
	OrderSide       OrderSide
	OrderType       OrderType
	OrderDate       time.Time
	Status          string
	Price           float64
	Amount          float64
	ExecutedAmount  float64
	RemainingAmount float64
	Fee             float64
	Trades          []TradeHistory
}

// OrderCancellation type required when requesting to cancel an order
type OrderCancellation struct {
	AccountID     string
	OrderID       string
	CurrencyPair  currency.Pair
	AssetType     asset.Item
	WalletAddress string
	Side          OrderSide
}

// GetOrdersRequest used for GetOrderHistory and GetOpenOrders wrapper functions
type GetOrdersRequest struct {
	OrderType  OrderType
	OrderSide  OrderSide
	StartTicks time.Time
	EndTicks   time.Time
	// Currencies Empty array = all currencies. Some endpoints only support singular currency enquiries
	Currencies []currency.Pair
}

// OrderStatus defines order status types
type OrderStatus string

// All OrderStatus types
const (
	AnyOrderStatus             OrderStatus = "ANY"
	NewOrderStatus             OrderStatus = "NEW"
	ActiveOrderStatus          OrderStatus = "ACTIVE"
	PartiallyFilledOrderStatus OrderStatus = "PARTIALLY_FILLED"
	FilledOrderStatus          OrderStatus = "FILLED"
	CancelledOrderStatus       OrderStatus = "CANCELED"
	PendingCancelOrderStatus   OrderStatus = "PENDING_CANCEL"
	RejectedOrderStatus        OrderStatus = "REJECTED"
	ExpiredOrderStatus         OrderStatus = "EXPIRED"
	HiddenOrderStatus          OrderStatus = "HIDDEN"
	UnknownOrderStatus         OrderStatus = "UNKNOWN"
)

// FilterOrdersBySide removes any OrderDetails that don't match the orderStatus provided
func FilterOrdersBySide(orders *[]OrderDetail, orderSide OrderSide) {
	if orderSide == "" || orderSide == AnyOrderSide {
		return
	}

	var filteredOrders []OrderDetail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].OrderSide), string(orderSide)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByType removes any OrderDetails that don't match the orderType provided
func FilterOrdersByType(orders *[]OrderDetail, orderType OrderType) {
	if orderType == "" || orderType == AnyOrderType {
		return
	}

	var filteredOrders []OrderDetail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].OrderType), string(orderType)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByTickRange removes any OrderDetails outside of the tick range
func FilterOrdersByTickRange(orders *[]OrderDetail, startTicks, endTicks time.Time) {
	if startTicks.IsZero() || endTicks.IsZero() ||
		startTicks.Unix() == 0 || endTicks.Unix() == 0 || endTicks.Before(startTicks) {
		return
	}

	var filteredOrders []OrderDetail
	for i := range *orders {
		if (*orders)[i].OrderDate.Unix() >= startTicks.Unix() && (*orders)[i].OrderDate.Unix() <= endTicks.Unix() {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByCurrencies removes any OrderDetails that do not match the provided currency list
// It is forgiving in that the provided currencies can match quote or base currencies
func FilterOrdersByCurrencies(orders *[]OrderDetail, currencies []currency.Pair) {
	if len(currencies) == 0 {
		return
	}

	var filteredOrders []OrderDetail
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
type ByPrice []OrderDetail

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
func SortOrdersByPrice(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByPrice(*orders)))
	} else {
		sort.Sort(ByPrice(*orders))
	}
}

// ByOrderType used for sorting orders by order type
type ByOrderType []OrderDetail

func (b ByOrderType) Len() int {
	return len(b)
}

func (b ByOrderType) Less(i, j int) bool {
	return b[i].OrderType.ToString() < b[j].OrderType.ToString()
}

func (b ByOrderType) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByType the caller function to sort orders
func SortOrdersByType(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderType(*orders)))
	} else {
		sort.Sort(ByOrderType(*orders))
	}
}

// ByCurrency used for sorting orders by order currency
type ByCurrency []OrderDetail

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
func SortOrdersByCurrency(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByCurrency(*orders)))
	} else {
		sort.Sort(ByCurrency(*orders))
	}
}

// ByDate used for sorting orders by order date
type ByDate []OrderDetail

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
func SortOrdersByDate(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByDate(*orders)))
	} else {
		sort.Sort(ByDate(*orders))
	}
}

// ByOrderSide used for sorting orders by order side (buy sell)
type ByOrderSide []OrderDetail

func (b ByOrderSide) Len() int {
	return len(b)
}

func (b ByOrderSide) Less(i, j int) bool {
	return b[i].OrderSide.ToString() < b[j].OrderSide.ToString()
}

func (b ByOrderSide) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersBySide the caller function to sort orders
func SortOrdersBySide(orders *[]OrderDetail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderSide(*orders)))
	} else {
		sort.Sort(ByOrderSide(*orders))
	}
}
