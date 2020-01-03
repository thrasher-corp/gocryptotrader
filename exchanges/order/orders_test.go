package order

import (
	"testing"
)

func TestNewOrder(t *testing.T) {
	ID := NewOrder("OKEX", 2000, 20.00)
	if ID != 0 {
		t.Error("Orders_test.go NewOrder() - Error")
	}
	ID = NewOrder("BATMAN", 400, 25.00)
	if ID != 1 {
		t.Error("Orders_test.go NewOrder() - Error")
	}
}

func TestDeleteOrder(t *testing.T) {
	if value := DeleteOrder(0); !value {
		t.Error("Orders_test.go DeleteOrder() - Error")
	}
	if value := DeleteOrder(100); value {
		t.Error("Orders_test.go DeleteOrder() - Error")
	}
}

func TestGetOrdersByExchange(t *testing.T) {
	if value := GetOrdersByExchange("OKEX"); len(value) != 0 {
		t.Error("Orders_test.go GetOrdersByExchange() - Error")
	}
}

func TestGetOrderByOrderID(t *testing.T) {
	if value := GetOrderByOrderID(69); value != nil {
		t.Error("Orders_test.go GetOrdersByExchange() - Error")
	}
}
