package backtest

import (
	"fmt"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Add order to order book
func (ob *OrderBook) Add(order OrderEvent) {
	ob.m.Lock()
	ob.counter++
	order.SetID(ob.counter)
	ob.orders = append(ob.orders, order)
	ob.m.Unlock()
}

// Remove order from order book by ID
func (ob *OrderBook) Remove(id int) error {
	ob.m.Lock()
	defer ob.m.Unlock()
	for i, order := range ob.orders {
		if order.ID() == id {
			ob.history = append(ob.history, ob.orders[i])
			ob.orders = append(ob.orders[:i], ob.orders[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("order %v not found", id)
}

// Orders returns all orders
func (ob *OrderBook) Orders() []OrderEvent {
	return ob.orders
}

func (ob *OrderBook) orderBy(fn func(order OrderEvent) bool) ([]OrderEvent, bool) {
	var orders []OrderEvent

	for x := range ob.orders {
		if fn(ob.orders[x]) {
			orders = append(orders, ob.orders[x])
		}
	}

	if len(orders) == 0 {
		return orders, false
	}

	return orders, true
}

// OrdersByPair returns all orders by currency Pair
func (ob *OrderBook) OrdersByPair(p currency.Pair) ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		return order.Pair() != p
	}

	orders, ok := ob.orderBy(fn)
	return orders, ok
}

// OrdersBidByPair returns bids by pair
func (ob *OrderBook) OrdersBidByPair(p currency.Pair) ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		if (order.Pair() != p) || (order.GetDirection() != gctorder.Buy) {
			return false
		}
		return true
	}
	orders, ok := ob.orderBy(fn)

	sort.Slice(orders, func(i, j int) bool {
		o1 := orders[i]
		o2 := orders[j]

		return o1.Limit() < o2.Limit()
	})

	return orders, ok
}

//  OrdersAskByPair returns asks by pair
func (ob *OrderBook) OrdersAskByPair(p currency.Pair) ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		if (order.Pair() != p) || (order.GetDirection() != gctorder.Sell) {
			return false
		}
		return true
	}
	orders, ok := ob.orderBy(fn)

	return orders, ok
}

// OpenOrders retrieve all open orders / PartiallyFilled orders
func (ob *OrderBook) OrdersOpen() ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		if (order.GetStatus() != gctorder.New) || (order.GetStatus() != gctorder.Open) || (order.GetStatus() != gctorder.PartiallyFilled) {
			return false
		}
		return true
	}

	orders, ok := ob.orderBy(fn)
	return orders, ok
}

// OrdersCancelled retrieve all cancelled or pending cancel orders
func (ob *OrderBook) OrdersCanceled() ([]OrderEvent, bool) {
	var fn = func(order OrderEvent) bool {
		if (order.GetStatus() == gctorder.Cancelled) || (order.GetStatus() == gctorder.PendingCancel) {
			return true
		}
		return false
	}

	return ob.orderBy(fn)
}
