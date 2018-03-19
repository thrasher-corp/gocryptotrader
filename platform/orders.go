package platform

const (
	limitOrder = iota
	marketOrder
)

// Orders variable holds an array of pointers to order structs
var Orders []*Order

// Order struct holds order values
type Order struct {
	OrderID  int
	Exchange string
	Type     int
	Amount   float64
	Price    float64
}

// NewOrder creates a new order and returns a an orderID
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
	orders := []*Order{}
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
