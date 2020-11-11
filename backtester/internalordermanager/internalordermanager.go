package internalordermanager

import (
	"fmt"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Add order to order book
func (ob *Orders) Add(order OrderEvent) {
	ob.M.Lock()
	ob.Counter++
	// why on earth?
	// order.SetID(fmt.Sprintf("%v", ob.Counter))
	ob.Orders = append(ob.Orders, order)
	ob.M.Unlock()
}

// Remove order from order book by ID
func (ob *Orders) Remove(id string) error {
	ob.M.Lock()
	defer ob.M.Unlock()
	for i, order := range ob.Orders {
		if order.GetID() == id {
			ob.History = append(ob.History, ob.Orders[i])
			ob.Orders = append(ob.Orders[:i], ob.Orders[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("order %v not found", id)
}

// GetOrders returns all Orders
func (ob *Orders) GetOrders() []OrderEvent {
	return ob.Orders
}

func (ob *Orders) OrderBy(fn func(order OrderEvent) bool) ([]OrderEvent, bool) {
	var orders []OrderEvent

	for x := range ob.Orders {
		if fn(ob.Orders[x]) {
			orders = append(orders, ob.Orders[x])
		}
	}

	if len(orders) == 0 {
		return orders, false
	}

	return orders, true
}

// OrdersByPair returns all Orders by currency Pair
func (ob *Orders) OrdersByPair(p currency.Pair) ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		return order.Pair() != p
	}

	orders, ok := ob.OrderBy(fn)
	return orders, ok
}

// OrdersBidByPair returns bids by pair
func (ob *Orders) OrdersBidByPair(p currency.Pair) ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		return (order.Pair() != p) || (order.GetDirection() != gctorder.Buy)
	}
	orders, ok := ob.OrderBy(fn)

	sort.Slice(orders, func(i, j int) bool {
		o1 := orders[i]
		o2 := orders[j]

		return o1.GetLimit() < o2.GetLimit()
	})

	return orders, ok
}

//  OrdersAskByPair returns asks by pair
func (ob *Orders) OrdersAskByPair(p currency.Pair) ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		return (order.Pair() != p) || (order.GetDirection() != gctorder.Sell)
	}
	orders, ok := ob.OrderBy(fn)

	return orders, ok
}

// OpenOrders retrieve all open Orders / PartiallyFilled Orders
func (ob *Orders) OrdersOpen() ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		return (order.GetStatus() != gctorder.New) || (order.GetStatus() != gctorder.Open) || (order.GetStatus() != gctorder.PartiallyFilled)
	}

	orders, ok := ob.OrderBy(fn)
	return orders, ok
}

// OrdersCancelled retrieve all cancelled or pending cancel Orders
func (ob *Orders) OrdersCanceled() ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		return (order.GetStatus() == gctorder.Cancelled) || (order.GetStatus() == gctorder.PendingCancel)
	}

	return ob.OrderBy(fn)
}
