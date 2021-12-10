package orderbook

import (
	"errors"
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func testSetup() Base {
	return Base{
		Exchange: "a",
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Asks: []Item{
			{Price: decimal.NewFromInt(7000), Amount: decimal.NewFromInt(1)},
			{Price: decimal.NewFromInt(7001), Amount: decimal.NewFromInt(2)},
		},
		Bids: []Item{
			{Price: decimal.NewFromInt(6999), Amount: decimal.NewFromInt(1)},
			{Price: decimal.NewFromInt(6998), Amount: decimal.NewFromInt(2)},
		},
	}
}

func TestWhaleBomb(t *testing.T) {
	t.Parallel()
	b := testSetup()

	// invalid price amount
	_, err := b.WhaleBomb(decimal.NewFromInt(-1), true)
	if err == nil {
		t.Error("unexpected result")
	}

	// valid
	_, err = b.WhaleBomb(decimal.NewFromInt(7001), true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	// invalid
	_, err = b.WhaleBomb(decimal.NewFromInt(7002), true)
	if err == nil {
		t.Error("unexpected result")
	}

	// valid
	_, err = b.WhaleBomb(decimal.NewFromInt(6998), false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	// invalid
	_, err = b.WhaleBomb(decimal.NewFromInt(6997), false)
	if err == nil {
		t.Error("unexpected result")
	}
}

func TestSimulateOrder(t *testing.T) {
	t.Parallel()
	b := testSetup()
	b.SimulateOrder(decimal.NewFromInt(8000), true)
	b.SimulateOrder(decimal.NewFromFloat(1.5), false)
}

func TestOrderSummary(t *testing.T) {
	var o orderSummary
	if p := o.MaximumPrice(false); !p.Equal(decimal.Zero) {
		t.Error("unexpected result")
	}
	if p := o.MinimumPrice(false); !p.Equal(decimal.Zero) {
		t.Error("unexpected result")
	}

	o = orderSummary{
		{Price: decimal.NewFromInt(1337), Amount: decimal.NewFromInt(1)},
		{Price: decimal.NewFromInt(9001), Amount: decimal.NewFromInt(1)},
	}
	if p := o.MaximumPrice(false); !p.Equal(decimal.NewFromInt(1337)) {
		t.Error("unexpected result")
	}
	if p := o.MaximumPrice(true); !p.Equal(decimal.NewFromInt(9001)) {
		t.Error("unexpected result")
	}
	if p := o.MinimumPrice(false); !p.Equal(decimal.NewFromInt(1337)) {
		t.Error("unexpected result")
	}
	if p := o.MinimumPrice(true); !p.Equal(decimal.NewFromInt(9001)) {
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
	_, err = b.GetAveragePrice(false, decimal.NewFromInt(5))
	if errors.Is(errNotEnoughLiquidity, err) {
		t.Error("expected: %w, received %w", errNotEnoughLiquidity, err)
	}
	b = Base{}
	b.Pair = cp
	b.Asks = []Item{
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(1)},
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(2)},
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(3)},
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(4)},
	}
	_, err = b.GetAveragePrice(true, decimal.NewFromInt(-2))
	if !errors.Is(err, errAmountInvalid) {
		t.Errorf("expected: %v, received %v", errAmountInvalid, err)
	}
	avgPrice, err := b.GetAveragePrice(true, decimal.NewFromInt(15))
	if err != nil {
		t.Error(err)
	}
	if !avgPrice.Equal(decimal.NewFromInt(2)) {
		t.Errorf("avg price calculation failed: expected 2, received %s", avgPrice)
	}
	avgPrice, err = b.GetAveragePrice(true, decimal.NewFromInt(18))
	if err != nil {
		t.Error(err)
	}
	if math.Round(avgPrice.InexactFloat64()*1000)/1000 != 2.333 {
		t.Errorf("avg price calculation failed: expected 2.333, received %f", math.Round(avgPrice.InexactFloat64()*1000)/1000)
	}
	_, err = b.GetAveragePrice(true, decimal.NewFromInt(25))
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Errorf("expected: %v, received %v", errNotEnoughLiquidity, err)
	}
}

func TestFindNominalAmount(t *testing.T) {
	b := Items{
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(1)},
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(2)},
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(3)},
		{Amount: decimal.NewFromInt(5), Price: decimal.NewFromInt(4)},
	}
	nomAmt, remainingAmt := b.FindNominalAmount(decimal.NewFromInt(15))
	if !nomAmt.Equal(decimal.NewFromInt(30)) && !remainingAmt.Equal(decimal.Zero) {
		t.Errorf("invalid return")
	}
	b = Items{}
	nomAmt, remainingAmt = b.FindNominalAmount(decimal.NewFromInt(15))
	if !nomAmt.Equal(decimal.Zero) && !remainingAmt.Equal(decimal.NewFromInt(30)) {
		t.Errorf("invalid return")
	}
}
