package limits

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestGreatestCommonDivisor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		first    int64
		second   int64
		expected int64
	}{
		{name: "shared factors", first: 18, second: 24, expected: 6},
		{name: "coprime", first: 17, second: 13, expected: 1},
		{name: "zero operand", first: 0, second: 12, expected: 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := greatestCommonDivisor(big.NewInt(tc.first), big.NewInt(tc.second))
			assert.Equal(t, tc.expected, result.Int64(), "greatestCommonDivisor should return the expected divisor")
		})
	}
}

func TestFractionalDigits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    string
		expected int
	}{
		{name: "integer", value: "12", expected: 0},
		{name: "fraction", value: "12.34", expected: 2},
		{name: "trailing zeroes", value: "12.3400", expected: 4},
		{name: "small negative fraction", value: "-0.001", expected: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, fractionalDigits(tc.value), "fractionalDigits should count digits after the decimal point")
		})
	}
}

func TestAmountStepOrderIncrement(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		increment   decimal.Decimal
		expectedErr error
	}{
		{name: "positive", increment: decimal.MustFromString("0.25")},
		{name: "zero", increment: decimal.Zero, expectedErr: ErrAmountStepNotPositive},
		{name: "negative", increment: decimal.NewFromInt(-1), expectedErr: ErrAmountStepNotPositive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := (AmountStep{Increment: tc.increment}).orderIncrement()
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "orderIncrement must reject a non-positive increment")
				assert.True(t, result.IsZero(), "orderIncrement should return zero on validation failure")
				return
			}
			require.NoError(t, err, "orderIncrement must accept a positive increment")
			assert.True(t, result.Equal(tc.increment), "orderIncrement should preserve a valid increment")
		})
	}
}

func TestAmountStepBaseIncrement(t *testing.T) {
	t.Parallel()
	step := AmountStep{
		Increment:          decimal.MustFromString("0.25"),
		ContractMultiplier: decimal.MustFromString("0.1"),
	}
	result, err := step.BaseIncrement()
	require.NoError(t, err, "BaseIncrement must accept positive inputs")
	assert.Equal(t, "0.025", result.String(), "BaseIncrement should combine the amount step and multiplier")
}

func TestAmountStepBaseIncrementInexactProduct(t *testing.T) {
	t.Parallel()
	unit := AmountStep{Increment: decimal.NewFromInt(1), ContractMultiplier: decimal.NewFromInt(1)}
	for _, tc := range []struct {
		name       string
		increment  string
		multiplier string
		exact      string
	}{
		{name: "underflow", increment: "0.0000000001", multiplier: "0.0000000001", exact: "0.00000000000000000001"},
		{name: "truncation", increment: "0.0000000003", multiplier: "0.0000000005", exact: "0.00000000000000000015"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			step := AmountStep{
				Increment:          decimal.MustFromString(tc.increment),
				ContractMultiplier: decimal.MustFromString(tc.multiplier),
			}
			result, err := step.BaseIncrement()
			if decimal.MaxFractionalDigits == 0 {
				require.NoError(t, err, "BaseIncrement must accept an exactly representable product")
				assert.Equal(t, tc.exact, result.String(), "BaseIncrement should retain the exact product")
				return
			}
			require.ErrorIs(t, err, ErrBaseIncrementNotRepresentable, "BaseIncrement must reject an inexact product")
			assert.True(t, result.IsZero(), "BaseIncrement should return zero for an inexact product")
			_, err = step.FloorBaseAmount(decimal.NewFromInt(1))
			assert.ErrorIs(t, err, ErrBaseIncrementNotRepresentable, "FloorBaseAmount should reject an inexact increment")
			_, err = step.CeilBaseAmount(decimal.NewFromInt(1))
			assert.ErrorIs(t, err, ErrBaseIncrementNotRepresentable, "CeilBaseAmount should reject an inexact increment")
			_, err = step.CommonBaseIncrement(unit)
			assert.ErrorIs(t, err, ErrBaseIncrementNotRepresentable, "CommonBaseIncrement should reject an inexact first increment")
			_, err = unit.CommonBaseIncrement(step)
			assert.ErrorIs(t, err, ErrBaseIncrementNotRepresentable, "CommonBaseIncrement should reject an inexact second increment")
		})
	}
}

func TestAmountStepBaseIncrementHighScaleExactProduct(t *testing.T) {
	t.Parallel()
	step := AmountStep{
		Increment:          decimal.MustFromString("0.0000000002"),
		ContractMultiplier: decimal.MustFromString("0.0000000005"),
	}
	result, err := step.BaseIncrement()
	require.NoError(t, err, "BaseIncrement must accept an exact product despite the combined input scale")
	assert.Equal(t, "0.0000000000000000001", result.String(), "BaseIncrement should retain the exact high-scale product")
}

func TestAmountStepRoundBaseAmount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		amount     string
		increment  string
		multiplier string
		floor      string
		ceil       string
	}{
		{name: "exact increment", amount: "150", increment: "1", multiplier: "1", floor: "150", ceil: "150"},
		{name: "fractional spot increment", amount: "10.251", increment: "0.25", multiplier: "1", floor: "10.25", ceil: "10.5"},
		{name: "fractional contract multiplier", amount: "1.82624", increment: "1", multiplier: "0.1", floor: "1.8", ceil: "1.9"},
		{name: "large contract multiplier", amount: "2044", increment: "1", multiplier: "1000", floor: "2000", ceil: "3000"},
		{name: "amount below increment", amount: "0.5", increment: "1", multiplier: "1", floor: "0", ceil: "1"},
		{
			name: "precision beyond default division scale", amount: "0.000000000000000007",
			increment: "0.000000000000000002", multiplier: "1",
			floor: "0.000000000000000006", ceil: "0.000000000000000008",
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			t.Parallel()
			step := AmountStep{
				Increment:          decimal.MustFromString(tests[i].increment),
				ContractMultiplier: decimal.MustFromString(tests[i].multiplier),
			}
			amount := decimal.MustFromString(tests[i].amount)
			floor, err := step.FloorBaseAmount(amount)
			require.NoError(t, err, "FloorBaseAmount must accept valid inputs")
			assert.Equal(t, tests[i].floor, floor.String(), "FloorBaseAmount should return the expected increment")
			ceil, err := step.CeilBaseAmount(amount)
			require.NoError(t, err, "CeilBaseAmount must accept valid inputs")
			assert.Equal(t, tests[i].ceil, ceil.String(), "CeilBaseAmount should return the expected increment")
		})
	}
}

func TestAmountStepFloorOrderAmount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		amount     string
		increment  string
		multiplier string
		expected   string
	}{
		{name: "contract amount", amount: "2.5369", increment: "1", multiplier: "100", expected: "2"},
		{name: "spot amount", amount: "2.61469", increment: "0.01", multiplier: "1", expected: "2.61"},
		{name: "exact amount", amount: "8242", increment: "1", multiplier: "1", expected: "8242"},
		{name: "does not require multiplier", amount: "2.5369", increment: "1", multiplier: "0", expected: "2"},
		{name: "amount below increment", amount: "0.5", increment: "1", multiplier: "1", expected: "0"},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			t.Parallel()
			step := AmountStep{
				Increment:          decimal.MustFromString(tests[i].increment),
				ContractMultiplier: decimal.MustFromString(tests[i].multiplier),
			}
			result, err := step.FloorOrderAmount(decimal.MustFromString(tests[i].amount))
			require.NoError(t, err, "FloorOrderAmount must accept valid inputs")
			assert.Equal(t, tests[i].expected, result.String(), "FloorOrderAmount should return the expected increment")
		})
	}
}

func TestAmountStepValidation(t *testing.T) {
	t.Parallel()
	valid := AmountStep{Increment: decimal.NewFromInt(1), ContractMultiplier: decimal.NewFromInt(1)}
	tests := []struct {
		name     string
		step     AmountStep
		amount   decimal.Decimal
		expected error
	}{
		{name: "zero amount", step: valid, amount: decimal.Zero, expected: ErrAmountNotPositive},
		{name: "negative amount", step: valid, amount: decimal.NewFromInt(-1), expected: ErrAmountNotPositive},
		{
			name:   "zero increment",
			step:   AmountStep{ContractMultiplier: decimal.NewFromInt(1)},
			amount: decimal.NewFromInt(1), expected: ErrAmountStepNotPositive,
		},
		{
			name:   "negative increment",
			step:   AmountStep{Increment: decimal.NewFromInt(-1), ContractMultiplier: decimal.NewFromInt(1)},
			amount: decimal.NewFromInt(1), expected: ErrAmountStepNotPositive,
		},
		{
			name:   "zero multiplier",
			step:   AmountStep{Increment: decimal.NewFromInt(1)},
			amount: decimal.NewFromInt(1), expected: ErrContractMultiplierNotPositive,
		},
		{
			name:   "negative multiplier",
			step:   AmountStep{Increment: decimal.NewFromInt(1), ContractMultiplier: decimal.NewFromInt(-1)},
			amount: decimal.NewFromInt(1), expected: ErrContractMultiplierNotPositive,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			t.Parallel()
			_, err := tests[i].step.FloorBaseAmount(tests[i].amount)
			assert.ErrorIs(t, err, tests[i].expected, "FloorBaseAmount should return the expected validation error")
			_, err = tests[i].step.CeilBaseAmount(tests[i].amount)
			assert.ErrorIs(t, err, tests[i].expected, "CeilBaseAmount should return the expected validation error")
			_, err = tests[i].step.FloorOrderAmount(tests[i].amount)
			if tests[i].expected == ErrContractMultiplierNotPositive {
				assert.NoError(t, err, "FloorOrderAmount should not require a contract multiplier")
			} else {
				assert.ErrorIs(t, err, tests[i].expected, "FloorOrderAmount should return the expected validation error")
			}
		})
	}
}

func TestAmountStepCommonBaseIncrement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		first       AmountStep
		second      AmountStep
		expected    string
		expectedErr error
	}{
		{
			name:     "one increment divides the other",
			first:    AmountStep{Increment: decimal.MustFromString("0.1"), ContractMultiplier: decimal.NewFromInt(1)},
			second:   AmountStep{Increment: decimal.MustFromString("0.01"), ContractMultiplier: decimal.NewFromInt(1)},
			expected: "0.1",
		},
		{
			name:     "non-divisible decimal increments",
			first:    AmountStep{Increment: decimal.MustFromString("0.2"), ContractMultiplier: decimal.NewFromInt(1)},
			second:   AmountStep{Increment: decimal.MustFromString("0.03"), ContractMultiplier: decimal.NewFromInt(1)},
			expected: "0.6",
		},
		{
			name:     "non-coprime numerators require greatest common divisor",
			first:    AmountStep{Increment: decimal.MustFromString("0.6"), ContractMultiplier: decimal.NewFromInt(1)},
			second:   AmountStep{Increment: decimal.MustFromString("0.9"), ContractMultiplier: decimal.NewFromInt(1)},
			expected: "1.8",
		},
		{
			name:     "different contract multipliers",
			first:    AmountStep{Increment: decimal.MustFromString("0.01"), ContractMultiplier: decimal.NewFromInt(10)},
			second:   AmountStep{Increment: decimal.MustFromString("0.001"), ContractMultiplier: decimal.NewFromInt(100)},
			expected: "0.1",
		},
		{
			name: "precision beyond default division scale",
			first: AmountStep{
				Increment:          decimal.MustFromString("0.000000000000000002"),
				ContractMultiplier: decimal.NewFromInt(1),
			},
			second: AmountStep{
				Increment:          decimal.MustFromString("0.000000000000000003"),
				ContractMultiplier: decimal.NewFromInt(1),
			},
			expected: "0.000000000000000006",
		},
		{
			name:        "invalid first step",
			first:       AmountStep{ContractMultiplier: decimal.NewFromInt(1)},
			second:      AmountStep{Increment: decimal.NewFromInt(1), ContractMultiplier: decimal.NewFromInt(1)},
			expectedErr: ErrAmountStepNotPositive,
		},
		{
			name:        "invalid second multiplier",
			first:       AmountStep{Increment: decimal.NewFromInt(1), ContractMultiplier: decimal.NewFromInt(1)},
			second:      AmountStep{Increment: decimal.NewFromInt(1)},
			expectedErr: ErrContractMultiplierNotPositive,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			t.Parallel()
			result, err := tests[i].first.CommonBaseIncrement(tests[i].second)
			if tests[i].expectedErr != nil {
				require.ErrorIs(t, err, tests[i].expectedErr, "CommonBaseIncrement must return the expected validation error")
				assert.True(t, result.IsZero(), "CommonBaseIncrement should return zero on validation failure")
				return
			}
			require.NoError(t, err, "CommonBaseIncrement must accept valid inputs")
			assert.Equal(t, tests[i].expected, result.String(), "CommonBaseIncrement should return the expected increment")

			reverse, err := tests[i].second.CommonBaseIncrement(tests[i].first)
			require.NoError(t, err, "CommonBaseIncrement must be symmetric")
			assert.True(t, result.Equal(reverse), "CommonBaseIncrement should be symmetric")
		})
	}
}

func TestAmountStepCommonBaseIncrementUnparsable(t *testing.T) {
	t.Parallel()
	if decimal.MaxFractionalDigits > 0 {
		t.Skip("big.Rat rejects more than 1e6 fractional digits, which udecimal cannot represent")
	}
	parseable := AmountStep{Increment: decimal.MustFromString("0.2"), ContractMultiplier: decimal.NewFromInt(1)}
	unparsable := AmountStep{Increment: decimal.MustFromString("3e-1000001"), ContractMultiplier: decimal.NewFromInt(1)}
	_, err := unparsable.CommonBaseIncrement(parseable)
	assert.ErrorIs(t, err, errCannotParseBaseIncrement, "CommonBaseIncrement should reject a first increment big.Rat cannot parse")
	_, err = parseable.CommonBaseIncrement(unparsable)
	assert.ErrorIs(t, err, errCannotParseBaseIncrement, "CommonBaseIncrement should reject a second increment big.Rat cannot parse")
}

func BenchmarkAmountStepCommonBaseIncrement(b *testing.B) {
	first := AmountStep{Increment: decimal.MustFromString("0.2"), ContractMultiplier: decimal.NewFromInt(1)}
	second := AmountStep{Increment: decimal.MustFromString("0.03"), ContractMultiplier: decimal.NewFromInt(1)}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = first.CommonBaseIncrement(second)
	}
}
