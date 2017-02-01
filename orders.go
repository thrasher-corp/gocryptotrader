package main

const (
	LIMIT_ORDER = iota
	MARKET_ORDER
)

var Orders []*Order

type Order struct {
	OrderID  int
	Exchange string
	Type     int
	Amount   float64
	Price    float64
}

func NewOrder(Exchange string, amount, price float64) int {
	order := &Order{}
	if len(Orders) == 0 {
		order.OrderID = 0
	} else {
		order.OrderID = len(Orders)
	}

	order.Exchange = Exchange
	order.Amount = amount
	order.Price = price
	Orders = append(Orders, order)
	return order.OrderID
}

func DeleteOrder(orderID int) bool {
	for i := range Orders {
		if Orders[i].OrderID == orderID {
			Orders = append(Orders[:i], Orders[i+1:]...)
			return true
		}
	}
	return false
}

func GetOrdersByExchange(exchange string) ([]*Order, bool) {
	orders := []*Order{}
	for i := range Orders {
		if Orders[i].Exchange == exchange {
			orders = append(orders, Orders[i])
		}
	}
	if len(orders) > 0 {
		return orders, true
	}
	return nil, false
}

func GetOrderByOrderID(orderID int) (*Order, bool) {
	for i := range Orders {
		if Orders[i].OrderID == orderID {
			return Orders[i], true
		}
	}
	return nil, false
}
