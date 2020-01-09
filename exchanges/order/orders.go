package order

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Validate checks the supplied data and returns whether or not it's valid
func (s *Submit) Validate() error {
	if s == nil {
		return ErrSubmissionIsNil
	}

	if s.Pair.IsEmpty() {
		return ErrPairIsEmpty
	}

	if s.Side != Buy &&
		s.Side != Sell &&
		s.Side != Bid &&
		s.Side != Ask {
		return ErrSideIsInvalid
	}

	if s.Type != Market && s.Type != Limit {
		return ErrTypeIsInvalid
	}

	if s.Amount <= 0 {
		return ErrAmountIsInvalid
	}

	if s.Type == Limit && s.Price <= 0 {
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
		if strings.EqualFold(string((*orders)[i].Side), string(side)) {
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
		if strings.EqualFold(string((*orders)[i].Type), string(orderType)) {
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
		if (*orders)[i].Date.Unix() >= startTicks.Unix() &&
			(*orders)[i].Date.Unix() <= endTicks.Unix() {
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
			if !matchFound && (*orders)[i].Pair.EqualIncludeReciprocal(c) {
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
	return b[i].Type.String() < b[j].Type.String()
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
	return b[i].Pair.String() < b[j].Pair.String()
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
	return b[i].Date.Unix() < b[j].Date.Unix()
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
	return b[i].Side.String() < b[j].Side.String()
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

// StringToOrderSide for converting case insensitive order side
// and returning a real Side
func StringToOrderSide(side string) (Side, error) {
	switch {
	case strings.EqualFold(side, Buy.String()):
		return Buy, nil
	case strings.EqualFold(side, Sell.String()):
		return Sell, nil
	case strings.EqualFold(side, Bid.String()):
		return Bid, nil
	case strings.EqualFold(side, Ask.String()):
		return Ask, nil
	case strings.EqualFold(side, AnySide.String()):
		return AnySide, nil
	default:
		return Side(""), fmt.Errorf("%s not recognised as side type", side)
	}
}

// StringToOrderType for converting case insensitive order type
// and returning a real Type
func StringToOrderType(oType string) (Type, error) {
	switch {
	case strings.EqualFold(oType, Limit.String()):
		return Limit, nil
	case strings.EqualFold(oType, Market.String()):
		return Market, nil
	case strings.EqualFold(oType, ImmediateOrCancel.String()):
		return ImmediateOrCancel, nil
	case strings.EqualFold(oType, Stop.String()):
		return Stop, nil
	case strings.EqualFold(oType, TrailingStop.String()):
		return TrailingStop, nil
	case strings.EqualFold(oType, AnyType.String()):
		return AnyType, nil
	default:
		return UnknownType, fmt.Errorf("%s not recognised as order type", oType)
	}
}

// StringToOrderStatus for converting case insensitive order status
// and returning a real Status
func StringToOrderStatus(status string) (Status, error) {
	switch {
	case strings.EqualFold(status, AnyStatus.String()):
		return AnyStatus, nil
	case strings.EqualFold(status, New.String()):
		return New, nil
	case strings.EqualFold(status, Active.String()):
		return Active, nil
	case strings.EqualFold(status, PartiallyFilled.String()):
		return PartiallyFilled, nil
	case strings.EqualFold(status, Filled.String()):
		return Filled, nil
	case strings.EqualFold(status, Cancelled.String()):
		return Cancelled, nil
	case strings.EqualFold(status, PendingCancel.String()):
		return PendingCancel, nil
	case strings.EqualFold(status, Rejected.String()):
		return Rejected, nil
	case strings.EqualFold(status, Expired.String()):
		return Expired, nil
	case strings.EqualFold(status, Hidden.String()):
		return Hidden, nil
	default:
		return UnknownStatus, fmt.Errorf("%s not recognised as order STATUS", status)
	}
}
