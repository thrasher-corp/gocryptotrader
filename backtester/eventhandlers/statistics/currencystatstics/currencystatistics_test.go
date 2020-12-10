package currencystatstics

import (
	"testing"

	"gonum.org/v1/gonum/stat"
)

func TestCalculateSharpeRatio(t *testing.T) {
	//	var c CurrencyStatistic
	//	c.Events = append(c.Events, EventStore{
	//		Holdings: holdings.Holding{EquityReturn: 1},
	//	})
	//	c.Events = append(c.Events, EventStore{
	//		Holdings: holdings.Holding{EquityReturn: 2},
	//	})
	//	c.Events = append(c.Events, EventStore{
	//		Holdings: holdings.Holding{EquityReturn: 3},
	//	})
	//	c.calculateSharpeRatio(0)
	//	if c.SharpeRatio != 2 {
	//		t.Errorf("expected %v received %v", 2, c.SharpeRatio)
	//	}
	//
	//	c.calculateSharpeRatio(1.5)
	//	if c.SharpeRatio != 0.5 {
	//		t.Errorf("expected %v received %v", 0.5, c.SharpeRatio)
	//	}
}

func TestStandardDeviation(t *testing.T) {
	dec := stat.StdDev([]float64{4, 4, 4, 4, 4, 6, 6, 6, 6, 6}, nil)
	t.Log(dec)
}

func TestStandardDeviation2(t *testing.T) {
	result := calculateStandardDeviation([]float64{4, 4, 4, 4, 4, 6, 6, 6, 6, 6})
	t.Log(result)
}
