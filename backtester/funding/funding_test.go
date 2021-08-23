package funding

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestCanPlaceOrder(t *testing.T) {
	p := Pair{
		Base:  &Item{},
		Quote: &Item{},
	}

	if p.CanPlaceOrder(gctorder.Buy) {
		t.Error("expected false")
	}
	if p.CanPlaceOrder(gctorder.Sell) {
		t.Error("expected false")
	}

	p.Quote.Available = decimal.NewFromFloat(32)
	if !p.CanPlaceOrder(gctorder.Buy) {
		t.Error("expected true")
	}
	p.Base.Available = decimal.NewFromFloat(32)
	if !p.CanPlaceOrder(gctorder.Sell) {
		t.Error("expected true")
	}
}

func TestIncreaseAvailable(t *testing.T) {
	i := Item{}
	i.IncreaseAvailable(decimal.NewFromFloat(3))
	if !i.Available.Equal(decimal.NewFromFloat(3)) {
		t.Error("expected 3")
	}
	i.IncreaseAvailable(decimal.NewFromFloat(0))
	i.IncreaseAvailable(decimal.NewFromFloat(-1))
	if !i.Available.Equal(decimal.NewFromFloat(3)) {
		t.Error("expected 3")
	}
}

func TestRelease(t *testing.T) {
	i := Item{}
	err := i.Release(decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = i.Release(decimal.NewFromFloat(1337), decimal.Zero)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}
	i.Reserved = decimal.NewFromFloat(1337)
	err = i.Release(decimal.NewFromFloat(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.Release(decimal.NewFromFloat(-1), decimal.Zero)
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
	err = i.Release(decimal.NewFromFloat(1337), decimal.NewFromFloat(-1))
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
}
