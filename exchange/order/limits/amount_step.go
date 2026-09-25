package limits

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	// ErrAmountNotPositive is returned when an amount cannot represent an order quantity.
	ErrAmountNotPositive = errors.New("amount must be greater than zero")
	// ErrAmountStepNotPositive is returned when an order amount increment is invalid.
	ErrAmountStepNotPositive = errors.New("amount step must be greater than zero")
	// ErrContractMultiplierNotPositive is returned when a contract multiplier is invalid.
	ErrContractMultiplierNotPositive = errors.New("contract multiplier must be greater than zero")
	// ErrBaseIncrementNotRepresentable is returned when the decimal backend truncates a positive product, including to zero.
	ErrBaseIncrementNotRepresentable = errors.New("base increment cannot be represented exactly")
	errCannotParseBaseIncrement      = errors.New("cannot parse base increment")
)

// AmountStep describes the executable order increment and the base amount
// represented by each order unit. Spot markets conventionally use a contract
// multiplier of one. The base-amount methods require ContractMultiplier to be
// a fixed base-asset amount per order unit; a quote-denominated contract value,
// such as an inverse contract's, cannot be used directly. FloorOrderAmount
// needs order units only.
type AmountStep struct {
	Increment          decimal.Decimal
	ContractMultiplier decimal.Decimal
}

// BaseIncrement returns the smallest base-asset quantity executable under the
// amount step.
func (a AmountStep) BaseIncrement() (decimal.Decimal, error) {
	increment, err := a.orderIncrement()
	if err != nil {
		return decimal.Zero, err
	}
	if !a.ContractMultiplier.IsPositive() {
		return decimal.Zero, fmt.Errorf("%w: %s", ErrContractMultiplierNotPositive, a.ContractMultiplier)
	}
	baseIncrement := increment.Mul(a.ContractMultiplier)
	// udecimal's Div truncates like its Mul, so a truncated product, including
	// one that underflows to zero, cannot divide back to the increment.
	if decimal.MaxFractionalDigits > 0 && !baseIncrement.Div(a.ContractMultiplier).Equal(increment) {
		return decimal.Zero, fmt.Errorf("%w: increment %s with multiplier %s truncates to %s",
			ErrBaseIncrementNotRepresentable, increment, a.ContractMultiplier, baseIncrement)
	}
	return baseIncrement, nil
}

// FloorBaseAmount rounds a base-asset amount down to an executable increment.
// A positive amount smaller than the increment rounds to zero without error.
func (a AmountStep) FloorBaseAmount(amount decimal.Decimal) (decimal.Decimal, error) {
	if !amount.IsPositive() {
		return decimal.Zero, fmt.Errorf("%w: %s", ErrAmountNotPositive, amount)
	}
	increment, err := a.BaseIncrement()
	if err != nil {
		return decimal.Zero, err
	}
	return amount.Sub(amount.Mod(increment)), nil
}

// CeilBaseAmount rounds a base-asset amount up to an executable increment.
func (a AmountStep) CeilBaseAmount(amount decimal.Decimal) (decimal.Decimal, error) {
	if !amount.IsPositive() {
		return decimal.Zero, fmt.Errorf("%w: %s", ErrAmountNotPositive, amount)
	}
	increment, err := a.BaseIncrement()
	if err != nil {
		return decimal.Zero, err
	}
	result := amount.Sub(amount.Mod(increment))
	if result.LessThan(amount) {
		result = result.Add(increment)
	}
	return result, nil
}

// FloorOrderAmount rounds an amount in exchange order units down to its
// executable increment. A contract multiplier is not required because both
// the amount and increment are already expressed in order units. A positive
// amount smaller than the increment rounds to zero without error.
func (a AmountStep) FloorOrderAmount(amount decimal.Decimal) (decimal.Decimal, error) {
	if !amount.IsPositive() {
		return decimal.Zero, fmt.Errorf("%w: %s", ErrAmountNotPositive, amount)
	}
	increment, err := a.orderIncrement()
	if err != nil {
		return decimal.Zero, err
	}
	return amount.Sub(amount.Mod(increment)), nil
}

// CommonBaseIncrement returns the smallest base-asset amount executable by
// both amount steps.
func (a AmountStep) CommonBaseIncrement(other AmountStep) (decimal.Decimal, error) {
	first, err := a.BaseIncrement()
	if err != nil {
		return decimal.Zero, err
	}
	second, err := other.BaseIncrement()
	if err != nil {
		return decimal.Zero, err
	}
	if first.Mod(second).IsZero() {
		return first, nil
	}
	if second.Mod(first).IsZero() {
		return second, nil
	}

	firstString := first.String()
	secondString := second.String()
	firstRat, ok := new(big.Rat).SetString(firstString)
	if !ok {
		return decimal.Zero, fmt.Errorf("%w: first value %q", errCannotParseBaseIncrement, firstString)
	}
	secondRat, ok := new(big.Rat).SetString(secondString)
	if !ok {
		return decimal.Zero, fmt.Errorf("%w: second value %q", errCannotParseBaseIncrement, secondString)
	}

	var numeratorProduct, numeratorLCM, denominatorGCD big.Int
	numeratorProduct.Mul(firstRat.Num(), secondRat.Num())
	numeratorLCM.Div(&numeratorProduct, greatestCommonDivisor(firstRat.Num(), secondRat.Num()))
	denominatorGCD.GCD(nil, nil, firstRat.Denom(), secondRat.Denom())

	resultRat := new(big.Rat).SetFrac(&numeratorLCM, &denominatorGCD)
	scale := max(fractionalDigits(firstString), fractionalDigits(secondString))
	result, err := decimal.NewFromString(resultRat.FloatString(scale))
	if err != nil {
		return decimal.Zero, fmt.Errorf("cannot convert common base increment: %w", err)
	}
	return result, nil
}

func greatestCommonDivisor(first, second *big.Int) *big.Int {
	var result big.Int
	result.GCD(nil, nil, first, second)
	return &result
}

func fractionalDigits(value string) int {
	point := strings.IndexByte(value, '.')
	if point == -1 {
		return 0
	}
	return len(value) - point - 1
}

func (a AmountStep) orderIncrement() (decimal.Decimal, error) {
	if !a.Increment.IsPositive() {
		return decimal.Zero, fmt.Errorf("%w: %s", ErrAmountStepNotPositive, a.Increment)
	}
	return a.Increment, nil
}
