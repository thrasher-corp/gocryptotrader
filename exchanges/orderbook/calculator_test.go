package orderbook

import (
	"errors"
	"fmt"
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

	_, err := b.WhaleBomb(-1, true)
	if !errors.Is(err, errPriceTargetInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errPriceTargetInvalid)
	}

	result, err := b.WhaleBomb(7001, true) // <- This price should not be wiped out on the book.
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if result.Amount != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 7000)
	}

	if result.MaximumPrice != 7001 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7001)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.PercentageGainOrLoss != 0.014285714285714287 {
		t.Fatalf("received: '%v' but expected: '%v'", result.PercentageGainOrLoss, 0.014285714285714287)
	}

	fmt.Println(result.Status)

	result, err = b.WhaleBomb(7000.5, true) // <- Slot between prices will lift to next ask tranche
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}

	if result.Amount != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 7000)
	}

	if result.MaximumPrice != 7001 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7001)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.PercentageGainOrLoss != 0.014285714285714287 {
		t.Fatalf("received: '%v' but expected: '%v'", result.PercentageGainOrLoss, 0.014285714285714287)
	}

	fmt.Println(result.Status)

	_, err = b.WhaleBomb(7002, true) // <- exceed available quotations
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	result, err = b.WhaleBomb(7000, true) // <- Book should not move
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if result.Amount != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 0)
	}

	if result.MaximumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7000)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.PercentageGainOrLoss != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", result.PercentageGainOrLoss, 0)
	}

	fmt.Println(result.Status)

	_, err = b.WhaleBomb(6000, true)
	if !errors.Is(err, errUnableToHitPriceTarget) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUnableToHitPriceTarget)
	}

	_, err = b.WhaleBomb(-1, false)
	if !errors.Is(err, errPriceTargetInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errPriceTargetInvalid)
	}

	result, err = b.WhaleBomb(6998, false) // <- This price should not be wiped out on the book.
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if result.Amount != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 1)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if result.MinimumPrice != 6998 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6998)
	}

	if result.PercentageGainOrLoss != -0.014287755393627661 {
		t.Fatalf("received: '%v' but expected: '%v'", result.PercentageGainOrLoss, -0.014287755393627661)
	}

	fmt.Println(result.Status)

	result, err = b.WhaleBomb(6998.5, false) // <- Slot between prices will drop to next bid tranche
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}

	if result.Amount != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 1)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if result.MinimumPrice != 6998 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6998)
	}

	if result.PercentageGainOrLoss != -0.014287755393627661 {
		t.Fatalf("received: '%v' but expected: '%v'", result.PercentageGainOrLoss, -0.014287755393627661)
	}

	fmt.Println(result.Status)

	_, err = b.WhaleBomb(6997, false) // <- exceed available quotations
	if !errors.Is(err, errNotEnoughLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughLiquidity)
	}

	result, err = b.WhaleBomb(6999, false) // <- Book should not move
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if result.Amount != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 0)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if result.MinimumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6999)
	}

	if result.PercentageGainOrLoss != 0 {
		t.Fatalf("received: '%v' but expected: '%v'", result.PercentageGainOrLoss, 0)
	}

	fmt.Println(result.Status)

	_, err = b.WhaleBomb(7500, false)
	if !errors.Is(err, errUnableToHitPriceTarget) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUnableToHitPriceTarget)
	}
}

func TestSimulateOrder(t *testing.T) {
	t.Parallel()
	b := testSetup()
	b.SimulateOrder(8000, true)
	b.SimulateOrder(1.5, false)
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
