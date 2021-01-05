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

func TestSetAmount(t *testing.T) {
	s := Signal{
		Amount: 1,
	}
	s.SetAmount(1337)
	if s.GetAmount() != 1337 {
		t.Error("expected 1337")
	}
}

func TestSetPrice(t *testing.T) {
	s := Signal{
		Price: 1,
	}
	s.SetPrice(1337)
	if s.GetPrice() != 1337 {
		t.Error("expected 1337")
	}
}
