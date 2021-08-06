package orderbook

import (
	"errors"
	"math"
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
	_, err = b.WhaleBomb(7001, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	// invalid
	_, err = b.WhaleBomb(7002, true)
	if err == nil {
		t.Error("unexpected result")
	}

	// valid
	_, err = b.WhaleBomb(6998, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	// invalid
	_, err = b.WhaleBomb(6997, false)
	if err == nil {
		t.Error("unexpected result")
	}
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

func TestGetAveragePrice(t *testing.T) {
	var b Base
	b.Exchange = "Binance"
	cp, err := currency.NewPairFromString("ETH-USDT")
	if err != nil {
		t.Error(err)
	}
	b.Pair = cp
	b.Bids = []Item{}
	_, err = b.GetAveragePrice(false, 5)
	if errors.Is(errNotEnoughLiquidity, err) {
		t.Error("expected: %w, received %w", errNotEnoughLiquidity, err)
	}
	b = Base{}
	b.Pair = cp
	b.Asks = []Item{
		{Amount: 5, Price: 1},
		{Amount: 5, Price: 2},
		{Amount: 5, Price: 3},
		{Amount: 5, Price: 4},
	}
	_, err = b.GetAveragePrice(true, -2)
	if !errors.Is(err, errAmountInvalid) {
		t.Errorf("expected: %v, received %v", errAmountInvalid, err)
	}
	avgPrice, err := b.GetAveragePrice(true, 15)
	if err != nil {
		t.Error(err)
	}
	if avgPrice != 2 {
		t.Errorf("avg price calculation failed: expected 2, received %f", avgPrice)
	}
	avgPrice, err = b.GetAveragePrice(true, 18)
	if err != nil {
		t.Error(err)
	}
	if math.Round(avgPrice*1000)/1000 != 2.333 {
		t.Errorf("avg price calculation failed: expected 2.333, received %f", math.Round(avgPrice*1000)/1000)
	}
	_, err = b.GetAveragePrice(true, 25)
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Errorf("expected: %v, received %v", errNotEnoughLiquidity, err)
	}
}

func TestFindNominalAmount(t *testing.T) {
	b := Items{
		{Amount: 5, Price: 1},
		{Amount: 5, Price: 2},
		{Amount: 5, Price: 3},
		{Amount: 5, Price: 4},
	}
	nomAmt, remainingAmt := b.FindNominalAmount(15)
	if nomAmt != 30 && remainingAmt != 0 {
		t.Errorf("invalid return")
	}
	b = Items{}
	nomAmt, remainingAmt = b.FindNominalAmount(15)
	if nomAmt != 0 && remainingAmt != 30 {
		t.Errorf("invalid return")
	}
}
