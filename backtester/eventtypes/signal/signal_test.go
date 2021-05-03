package signal

import (
	"testing"

	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestIsSignal(t *testing.T) {
	s := Signal{}
	if !s.IsSignal() {
		t.Error("expected true")
	}
}

func TestSetDirection(t *testing.T) {
	s := Signal{Direction: gctorder.Sell}
	s.SetDirection(gctorder.Buy)
	if s.GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetPrice(t *testing.T) {
	s := Signal{
		ClosePrice: 1,
	}
	s.SetPrice(1337)
	if s.GetPrice() != 1337 {
		t.Error("expected 1337")
	}
}

func TestSetBuyLimit(t *testing.T) {
	s := Signal{
		BuyLimit: 10,
	}
	s.SetBuyLimit(20)
	if s.GetBuyLimit() != 20 {
		t.Errorf("expected 20, received %v", s.GetBuyLimit())
	}
}

func TestSetSellLimit(t *testing.T) {
	s := Signal{
		SellLimit: 10,
	}
	s.SetSellLimit(20)
	if s.GetSellLimit() != 20 {
		t.Errorf("expected 20, received %v", s.GetSellLimit())
	}
}
