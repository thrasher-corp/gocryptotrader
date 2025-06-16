package orderbook

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func testSetup() Book {
	return Book{
		Exchange: "a",
		Pair:     currency.NewBTCUSD(),
		Asks:     []Level{{Price: 7000, Amount: 1}, {Price: 7001, Amount: 2}},
		Bids:     []Level{{Price: 6999, Amount: 1}, {Price: 6998, Amount: 2}},
	}
}

func TestWhaleBomb(t *testing.T) {
	t.Parallel()
	b := testSetup()

	_, err := b.WhaleBomb(-1, true)
	require.ErrorIs(t, err, errPriceTargetInvalid)

	result, err := b.WhaleBomb(7001, true) // <- This price should not be wiped out on the book.
	require.NoError(t, err)

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

	result, err = b.WhaleBomb(7000.5, true) // <- Slot between prices will lift to next ask level
	assert.NoError(t, err)

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

	result, err = b.WhaleBomb(7002, true) // <- exceed available quotations
	require.NoError(t, err)

	if !strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatal("expected status to contain liquidity warning")
	}

	result, err = b.WhaleBomb(7000, true) // <- Book should not move
	require.NoError(t, err)

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

	_, err = b.WhaleBomb(6000, true)
	require.ErrorIs(t, err, errCannotShiftPrice)

	_, err = b.WhaleBomb(-1, false)
	require.ErrorIs(t, err, errPriceTargetInvalid)

	result, err = b.WhaleBomb(6998, false) // <- This price should not be wiped out on the book.
	require.NoError(t, err)

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

	result, err = b.WhaleBomb(6998.5, false) // <- Slot between prices will drop to next bid level
	assert.NoError(t, err)

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

	result, err = b.WhaleBomb(6997, false) // <- exceed available quotations
	require.NoError(t, err)

	if !strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatal("expected status to contain liquidity warning")
	}

	result, err = b.WhaleBomb(6999, false) // <- Book should not move
	require.NoError(t, err)

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

	_, err = b.WhaleBomb(7500, false)
	require.ErrorIs(t, err, errCannotShiftPrice)
}

func TestSimulateOrder(t *testing.T) {
	t.Parallel()
	b := testSetup()

	// Invalid
	_, err := b.SimulateOrder(-8000, true)
	require.ErrorIs(t, err, errQuoteAmountInvalid)

	_, err = (&Book{}).SimulateOrder(1337, true)
	require.ErrorIs(t, err, errNoLiquidity)

	// Full liquidity used
	result, err := b.SimulateOrder(21002, true)
	require.NoError(t, err)

	if result.Amount != 3 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 3)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.MaximumPrice != 7001 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7001)
	}

	if !strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, fullLiquidityUsageWarning)
	}

	if len(result.Orders) != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 2)
	}

	// Exceed full liquidity used
	result, err = b.SimulateOrder(21003, true)
	require.NoError(t, err)

	if result.Amount != 3 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 3)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.MaximumPrice != 7001 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7001)
	}

	if !strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, fullLiquidityUsageWarning)
	}

	if len(result.Orders) != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 2)
	}

	// First level
	result, err = b.SimulateOrder(7000, true)
	require.NoError(t, err)

	if result.Amount != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 1)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.MaximumPrice != 7001 { // A full level is wiped out and this one should be preserved.
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7001)
	}

	if strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, "NO WARNING")
	}

	if len(result.Orders) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 1)
	}

	// Half of first tranch
	result, err = b.SimulateOrder(3500, true)
	require.NoError(t, err)

	if result.Amount != .5 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, .5)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.MaximumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7000)
	}

	if strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, "NO WARNING")
	}

	if len(result.Orders) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 1)
	}

	if result.Orders[0].Amount != 0.5 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Orders[0].Amount, 0.5)
	}

	// Half of second level
	result, err = b.SimulateOrder(14001, true)
	require.NoError(t, err)

	if result.Amount != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 2)
	}

	if result.MinimumPrice != 7000 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 7000)
	}

	if result.MaximumPrice != 7001 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 7001)
	}

	if strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, "NO WARNING")
	}

	if len(result.Orders) != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 1)
	}

	if result.Orders[1].Amount != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Orders[1].Amount, 1)
	}

	// Hitting bids

	// Invalid

	_, err = (&Book{}).SimulateOrder(-1, false)
	require.ErrorIs(t, err, errBaseAmountInvalid)

	_, err = (&Book{}).SimulateOrder(2, false)
	require.ErrorIs(t, err, errNoLiquidity)

	// Full liquidity used
	result, err = b.SimulateOrder(3, false)
	require.NoError(t, err)

	if result.Amount != 20995 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 20995)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if result.MinimumPrice != 6998 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6998)
	}

	if !strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, fullLiquidityUsageWarning)
	}

	if len(result.Orders) != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 2)
	}

	// Exceed full liquidity used
	result, err = b.SimulateOrder(3.1, false)
	require.NoError(t, err)

	if result.Amount != 20995 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 20995)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if result.MinimumPrice != 6998 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6998)
	}

	if !strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, fullLiquidityUsageWarning)
	}

	if len(result.Orders) != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 2)
	}

	// First level
	result, err = b.SimulateOrder(1, false)
	require.NoError(t, err)

	if result.Amount != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 6999)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if result.MinimumPrice != 6998 { // A full level is wiped out and this one should be preserved.
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6998)
	}

	if strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, "NO WARNING")
	}

	if len(result.Orders) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 1)
	}

	// Half of first tranch
	result, err = b.SimulateOrder(.5, false)
	require.NoError(t, err)

	if result.Amount != 3499.5 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 3499.5)
	}

	if result.MinimumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6999)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, "NO WARNING")
	}

	if len(result.Orders) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 1)
	}

	if result.Orders[0].Amount != 0.5 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Orders[0].Amount, 0.5)
	}

	// Half of second level
	result, err = b.SimulateOrder(2, false)
	require.NoError(t, err)

	if result.Amount != 13997 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Amount, 13997)
	}

	if result.MaximumPrice != 6999 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MaximumPrice, 6999)
	}

	if result.MinimumPrice != 6998 {
		t.Fatalf("received: '%v' but expected: '%v'", result.MinimumPrice, 6998)
	}

	if strings.Contains(result.Status, fullLiquidityUsageWarning) {
		t.Fatalf("received: '%v' but expected string to contain: '%v'", result.Status, "NO WARNING")
	}

	if len(result.Orders) != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", len(result.Orders), 2)
	}

	if result.Orders[1].Amount != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", result.Orders[1].Amount, 1)
	}
}

func TestGetAveragePrice(t *testing.T) {
	b := Book{
		Exchange: "Binance",
		Pair:     currency.NewBTCUSD(),
	}
	_, err := b.GetAveragePrice(false, 5)
	assert.ErrorIs(t, err, errNotEnoughLiquidity)

	b = Book{
		Asks: []Level{
			{Amount: 5, Price: 1},
			{Amount: 5, Price: 2},
			{Amount: 5, Price: 3},
			{Amount: 5, Price: 4},
		},
	}
	_, err = b.GetAveragePrice(true, -2)
	assert.ErrorIs(t, err, errAmountInvalid)

	avgPrice, err := b.GetAveragePrice(true, 15)
	require.NoError(t, err)
	assert.Equal(t, 2.0, avgPrice)

	avgPrice, err = b.GetAveragePrice(true, 18)
	require.NoError(t, err)
	assert.Equal(t, 2.333, math.Round(avgPrice*1000)/1000)

	_, err = b.GetAveragePrice(true, 25)
	assert.ErrorIs(t, err, errNotEnoughLiquidity)
}

func TestFindNominalAmount(t *testing.T) {
	b := Levels{
		{Amount: 5, Price: 1},
		{Amount: 5, Price: 2},
		{Amount: 5, Price: 3},
		{Amount: 5, Price: 4},
	}
	nomAmt, remainingAmt := b.FindNominalAmount(15)
	if nomAmt != 30 && remainingAmt != 0 {
		t.Errorf("invalid return")
	}
	b = Levels{}
	nomAmt, remainingAmt = b.FindNominalAmount(15)
	if nomAmt != 0 && remainingAmt != 30 {
		t.Errorf("invalid return")
	}
}
