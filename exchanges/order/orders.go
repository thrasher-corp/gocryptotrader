package order

import (
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// NewOrder creates a new order and returns a an orderID
func NewOrder(exchangeName string, amount, price float64) int {
	order := &Order{}
	if len(Orders) == 0 {
		order.OrderID = 0
	} else {
		order.OrderID = len(Orders)
	}

	order.Exchange = exchangeName
	order.Amount = amount
	order.Price = price
	Orders = append(Orders, order)
	return order.OrderID
}

// DeleteOrder deletes orders by ID and returns state
func DeleteOrder(orderID int) bool {
	for i := range Orders {
		if Orders[i].OrderID == orderID {
			Orders = append(Orders[:i], Orders[i+1:]...)
			return true
		}
	}
	return false
}

// GetOrdersByExchange returns order pointer grouped by exchange
func GetOrdersByExchange(exchange string) []*Order {
	var orders []*Order
	for i := range Orders {
		if Orders[i].Exchange == exchange {
			orders = append(orders, Orders[i])
		}
	}
	if len(orders) > 0 {
		return orders
	}
	return nil
}

// GetOrderByOrderID returns order pointer by ID
func GetOrderByOrderID(orderID int) *Order {
	for i := range Orders {
		if Orders[i].OrderID == orderID {
			return Orders[i]
		}
	}
	return nil
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

// String implements the stringer interface
func (t Type) String() string {
	return string(t)
}

// Lower returns the type lower case string
func (t Type) Lower() string {
	return strings.ToLower(string(t))
}

// String implements the stringer interface
func (s Side) String() string {
	return string(s)
}

// Lower returns the side lower case string
func (s Side) Lower() string {
	return strings.ToLower(string(s))
}

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
