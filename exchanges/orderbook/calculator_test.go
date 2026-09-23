package orderbook

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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

func TestLevelsCalculateExecution(t *testing.T) {
	t.Parallel()
	levels := Levels{
		{Price: 100, Amount: 2},
		{Price: 110, Amount: 3},
		{Price: 120, Amount: 5},
	}

	result, err := levels.CalculateExecution(decimal.NewFromInt(4), decimal.MustFromFloat(0.5))
	require.NoError(t, err, "CalculateExecution must not error for sufficient liquidity")
	assert.True(t, result.RequestedAmount.Equal(decimal.NewFromInt(4)), "requested amount should match")
	assert.True(t, result.ExecutedAmount.Equal(decimal.NewFromInt(4)), "executed amount should match")
	assert.True(t, result.RemainingAmount.IsZero(), "remaining amount should be zero")
	assert.True(t, result.BaseAmount.Equal(decimal.NewFromInt(2)), "base amount should include the multiplier")
	assert.True(t, result.QuoteAmount.Equal(decimal.NewFromInt(210)), "quote amount should match the book walk")
	assert.True(t, result.AveragePrice.Equal(decimal.NewFromInt(105)), "average price should be the VWAP")
	assert.True(t, result.MarginalPrice.Equal(decimal.NewFromInt(110)), "marginal price should be the final consumed level")
	assert.Equal(t, uint64(2), result.LevelsUsed, "levels used should match")
	assert.False(t, result.FullLiquidityUsed, "execution should leave liquidity on the book")

	result, err = levels.CalculateExecution(decimal.NewFromInt(2), decimal.NewFromInt(1))
	require.NoError(t, err, "CalculateExecution must not error at an exact level boundary")
	assert.True(t, result.MarginalPrice.Equal(decimal.NewFromInt(100)), "marginal price should identify the final consumed level")
	assert.False(t, result.FullLiquidityUsed, "an exact internal boundary should leave later levels available")

	result, err = levels.CalculateExecution(decimal.NewFromInt(10), decimal.NewFromInt(1))
	require.NoError(t, err, "CalculateExecution must not error when exactly consuming the book")
	assert.True(t, result.FullLiquidityUsed, "an execution consuming the final level should report full liquidity use")

	result, err = levels.CalculateExecution(decimal.NewFromInt(6), decimal.NewFromInt(1))
	require.NoError(t, err, "CalculateExecution must not error when partially consuming the final level")
	assert.Equal(t, uint64(3), result.LevelsUsed, "execution should reach the final level")
	assert.True(t, result.RemainingAmount.IsZero(), "requested amount should be filled")
	assert.False(t, result.FullLiquidityUsed, "a partially consumed final level should leave liquidity available")
}

func TestLevelsCalculateExecutionAggregatesBeforeScaling(t *testing.T) {
	t.Parallel()
	levels := make(Levels, 10)
	for i := range levels {
		levels[i] = Level{StrAmount: "0.0000000001", Price: float64((i + 1) * 100)}
	}
	result, err := levels.CalculateExecution(decimal.MustFromString("0.000000001"), decimal.MustFromString("0.0000000001"))
	require.NoError(t, err, "CalculateExecution must not error when the aggregate is representable")
	assert.True(t, result.BaseAmount.Equal(decimal.MustFromString("0.0000000000000000001")), "base amount should retain the aggregate before scaling")
	assert.True(t, result.QuoteAmount.Equal(decimal.MustFromString("0.000000000000000055")), "quote amount should retain the aggregate before scaling")
	assert.True(t, result.AveragePrice.Equal(decimal.NewFromInt(550)), "average price should reflect all levels")
}

func TestLevelsCalculateExecutionExactValues(t *testing.T) {
	t.Parallel()
	levels := Levels{{
		Amount:    0.1,
		StrAmount: "0.100000000000000001",
		Price:     0.2,
		StrPrice:  "0.200000000000000003",
	}}
	amount := decimal.MustFromString("0.100000000000000001")

	result, err := levels.CalculateExecution(amount, decimal.NewFromInt(1))
	require.NoError(t, err, "CalculateExecution must use valid exact level values")
	assert.Equal(t, expectedExactExecutionQuoteAmount, result.QuoteAmount.String(),
		"quote amount should retain exact level precision")
	assert.True(t, result.AveragePrice.Equal(decimal.MustFromString("0.200000000000000003")),
		"average price should retain exact level precision")
	assert.True(t, result.FullLiquidityUsed, "single-level execution should report full liquidity use")

	levels = Levels{{Amount: 1, Price: 1}, {Amount: 2, Price: 2}}
	result, err = levels.CalculateExecution(decimal.NewFromInt(3), decimal.NewFromInt(1))
	require.NoError(t, err, "CalculateExecution must use valid multi-level values")
	assert.Equal(t, expectedMultiLevelAveragePrice, result.AveragePrice.String(),
		"average price should use the selected implementation's division precision")
}

func TestLevelsCalculateExecutionInsufficientLiquidity(t *testing.T) {
	t.Parallel()
	levels := Levels{{Price: 100, Amount: 2}, {Price: 110, Amount: 3}}

	result, err := levels.CalculateExecution(decimal.NewFromInt(6), decimal.NewFromInt(1))
	require.ErrorIs(t, err, ErrNotEnoughLiquidity, "CalculateExecution must report insufficient liquidity")
	assert.True(t, result.ExecutedAmount.Equal(decimal.NewFromInt(5)), "executed amount should describe the partial fill")
	assert.True(t, result.RemainingAmount.Equal(decimal.NewFromInt(1)), "remaining amount should describe the shortfall")
	assert.True(t, result.QuoteAmount.Equal(decimal.NewFromInt(530)), "quote amount should describe consumed liquidity")
	assert.True(t, result.AveragePrice.Equal(decimal.NewFromInt(106)), "average price should describe consumed liquidity")
	assert.True(t, result.MarginalPrice.Equal(decimal.NewFromInt(110)), "marginal price should describe the final consumed level")
	assert.Equal(t, uint64(2), result.LevelsUsed, "levels used should describe consumed liquidity")
	assert.True(t, result.FullLiquidityUsed, "insufficient execution should consume all liquidity")
}

func TestLevelsCalculateExecutionValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		levels     Levels
		amount     decimal.Decimal
		multiplier decimal.Decimal
		expected   error
		contains   string
	}{
		{
			name:       "zero amount",
			amount:     decimal.Zero,
			multiplier: decimal.NewFromInt(1),
			expected:   errAmountInvalid,
		},
		{
			name:       "negative amount",
			amount:     decimal.NewFromInt(-1),
			multiplier: decimal.NewFromInt(1),
			expected:   errAmountInvalid,
		},
		{
			name:       "invalid multiplier",
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.Zero,
			expected:   ErrInvalidContractMultiplier,
		},
		{
			name:       "negative multiplier",
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(-1),
			expected:   ErrInvalidContractMultiplier,
		},
		{
			name:       "empty levels",
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   errNoLiquidity,
		},
		{
			name:       "zero level amount",
			levels:     Levels{{Price: 1}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   `amount "0"`,
		},
		{
			name:       "negative level amount",
			levels:     Levels{{Amount: -1, Price: 1}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   "-1",
		},
		{
			name:       "NaN level amount",
			levels:     Levels{{Amount: math.NaN(), Price: 1}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   "NaN",
		},
		{
			name:       "infinite level amount",
			levels:     Levels{{Amount: math.Inf(1), Price: 1}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   "+Inf",
		},
		{
			name:       "zero level price",
			levels:     Levels{{Amount: 1}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   `price "0"`,
		},
		{
			name:       "negative level price",
			levels:     Levels{{Amount: 1, Price: -1}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   "-1",
		},
		{
			name:       "NaN level price",
			levels:     Levels{{Amount: 1, Price: math.NaN()}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   "NaN",
		},
		{
			name:       "invalid exact price",
			levels:     Levels{{Amount: 1, Price: 1, StrPrice: "invalid"}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   `price "invalid":`,
		},
		{
			name:       "infinite level price",
			levels:     Levels{{Amount: 1, Price: math.Inf(1)}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   "+Inf",
		},
		{
			name:       "invalid exact amount",
			levels:     Levels{{Amount: 1, StrAmount: "invalid", Price: 1}},
			amount:     decimal.NewFromInt(1),
			multiplier: decimal.NewFromInt(1),
			expected:   ErrOrderbookInvalid,
			contains:   `amount "invalid":`,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			t.Parallel()
			_, err := tests[i].levels.CalculateExecution(tests[i].amount, tests[i].multiplier)
			assert.ErrorIs(t, err, tests[i].expected, "CalculateExecution should return the expected validation error")
			if tests[i].contains != "" {
				assert.ErrorContains(t, err, tests[i].contains, "CalculateExecution should identify the invalid input")
			}
		})
	}
}

func BenchmarkLevelsCalculateExecution(b *testing.B) {
	levels := Levels{
		{Price: 100, Amount: 2},
		{Price: 110, Amount: 3},
		{Price: 120, Amount: 5},
	}
	amount := decimal.NewFromInt(4)
	multiplier := decimal.MustFromFloat(0.5)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = levels.CalculateExecution(amount, multiplier)
	}
}
