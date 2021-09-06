package settings

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

func TestGetLatestHoldings(t *testing.T) {
	t.Parallel()
	cs := Settings{}
	h := cs.GetLatestHoldings()
	if h != nil {
		t.Error("expected nil holdings")
	}
	tt := time.Now()
	cs.HoldingsSnapshots = append(cs.HoldingsSnapshots, &holdings.Holding{Timestamp: tt})

	h = cs.GetLatestHoldings()
	if !h.Timestamp.Equal(tt) {
		t.Errorf("expected %v, received %v", tt, h.Timestamp)
	}
}

func TestValue(t *testing.T) {
	t.Parallel()
	cs := Settings{}
	v := cs.Value()
	if !v.IsZero() {
		t.Error("expected 0")
	}
	cs.HoldingsSnapshots = append(cs.HoldingsSnapshots, &holdings.Holding{TotalValue: decimal.NewFromInt(1337)})

	v = cs.Value()
	if !v.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected %v, received %v", decimal.NewFromInt(1337), v)
	}
}
