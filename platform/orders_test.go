package platform

import "testing"

func TestNewOrder(t *testing.T) {
	ID := NewOrder("ANX", 2000, 20.00)
	if ID != 0 {
		t.Error("Test Failed - Orders_test.go NewOrder() - Error")
	}
	ID = NewOrder("BATMAN", 400, 25.00)
	if ID != 1 {
		t.Error("Test Failed - Orders_test.go NewOrder() - Error")
	}
}

func TestDeleteOrder(t *testing.T) {
	if value := DeleteOrder(0); !value {
		t.Error("Test Failed - Orders_test.go DeleteOrder() - Error")
	}
	if value := DeleteOrder(100); value {
		t.Error("Test Failed - Orders_test.go DeleteOrder() - Error")
	}
}

func TestGetOrdersByExchange(t *testing.T) {
	if value := GetOrdersByExchange("ANX"); len(value) != 0 {
		t.Error("Test Failed - Orders_test.go GetOrdersByExchange() - Error")
	}
}

func TestGetOrderByOrderID(t *testing.T) {
	if value := GetOrderByOrderID(69); value != nil {
		t.Error("Test Failed - Orders_test.go GetOrdersByExchange() - Error")
	}
}
