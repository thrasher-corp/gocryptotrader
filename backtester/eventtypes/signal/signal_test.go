package signal

import (
	"testing"

	"github.com/shopspring/decimal"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestIsSignal(t *testing.T) {
	t.Parallel()
	s := Signal{}
	if !s.IsSignal() {
		t.Error("expected true")
	}
}

func TestSetDirection(t *testing.T) {
	t.Parallel()
	s := Signal{Direction: gctorder.Sell}
	s.SetDirection(gctorder.Buy)
	if s.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetPrice(t *testing.T) {
	t.Parallel()
	s := Signal{
		ClosePrice: decimal.NewFromInt(1),
	}
	s.SetPrice(decimal.NewFromInt(1337))
	if !s.GetClosePrice().Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
}

func TestSetBuyLimit(t *testing.T) {
	t.Parallel()
	s := Signal{
		BuyLimit: decimal.NewFromInt(10),
	}
	s.SetBuyLimit(decimal.NewFromInt(20))
	if !s.GetBuyLimit().Equal(decimal.NewFromInt(20)) {
		t.Errorf("expected 20, received %v", s.GetBuyLimit())
	}
}

func TestSetSellLimit(t *testing.T) {
	t.Parallel()
	s := Signal{
		SellLimit: decimal.NewFromInt(10),
	}
	s.SetSellLimit(decimal.NewFromInt(20))
	if !s.GetSellLimit().Equal(decimal.NewFromInt(20)) {
		t.Errorf("expected 20, received %v", s.GetSellLimit())
	}
}
