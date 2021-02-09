package settings

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

func TestGetLatestHoldings(t *testing.T) {
	t.Parallel()
	cs := Settings{}
	h := cs.GetLatestHoldings()
	if !h.Timestamp.IsZero() {
		t.Error("expected zero time")
	}
	tt := time.Now()
	cs.HoldingsSnapshots = append(cs.HoldingsSnapshots, holdings.Holding{Timestamp: tt})

	h = cs.GetLatestHoldings()
	if !h.Timestamp.Equal(tt) {
		t.Errorf("expected %v, received %v", tt, h.Timestamp)
	}
}

func TestValue(t *testing.T) {
	t.Parallel()
	cs := Settings{}
	v := cs.Value()
	if v != 0 {
		t.Error("expected 0")
	}
	cs.HoldingsSnapshots = append(cs.HoldingsSnapshots, holdings.Holding{TotalValue: 1337})

	v = cs.Value()
	if v != 1337 {
		t.Errorf("expected %v, received %v", 1337, v)
	}
}
