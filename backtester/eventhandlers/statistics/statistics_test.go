package statistics

import "testing"

func TestReset(t *testing.T) {
	s := Statistic{
		TotalOrders: 1,
	}
	s.Reset()
	if s.TotalOrders != 0 {
		t.Error("expected 0")
	}
}
