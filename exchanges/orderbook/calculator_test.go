package orderbook

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func testSetup() Base {
	return Base{
		Exchange: "a",
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Asks: []Item{
			{Price: 7000, Amount: 1},
			{Price: 7001, Amount: 2},
		},
		Bids: []Item{
			{Price: 6999, Amount: 1},
			{Price: 6998, Amount: 2},
		},
	}
}

func TestWhaleBomb(t *testing.T) {
	t.Parallel()
	b := testSetup()

	// invalid price amount
	_, err := b.WhaleBomb(-1, true)
	if err == nil {
		t.Error("unexpected result")
	}

	// valid
	b.WhaleBomb(7001, true)
	// invalid
	b.WhaleBomb(7002, true)

	// valid
	b.WhaleBomb(6998, false)
	// invalid
	b.WhaleBomb(6997, false)
}

func TestSimulateOrder(t *testing.T) {
	t.Parallel()
	b := testSetup()
	b.SimulateOrder(8000, true)
	b.SimulateOrder(1.5, false)
}

func TestOrderSummary(t *testing.T) {
	var o orderSummary
	if p := o.MaximumPrice(false); p != 0 {
		t.Error("unexpected result")
	}
	if p := o.MinimumPrice(false); p != 0 {
		t.Error("unexpected result")
	}

	o = orderSummary{
		{Price: 1337, Amount: 1},
		{Price: 9001, Amount: 1},
	}
	if p := o.MaximumPrice(false); p != 1337 {
		t.Error("unexpected result")
	}
	if p := o.MaximumPrice(true); p != 9001 {
		t.Error("unexpected result")
	}
	if p := o.MinimumPrice(false); p != 1337 {
		t.Error("unexpected result")
	}
	if p := o.MinimumPrice(true); p != 9001 {
		t.Error("unexpected result")
	}

	o.Print()
}
