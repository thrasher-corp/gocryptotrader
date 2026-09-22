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
)

// AmountStep describes the executable order increment and the base amount
// represented by each order unit. Spot markets conventionally use a contract
// multiplier of one.
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
	return increment.Mul(a.ContractMultiplier), nil
}

// FloorBaseAmount rounds a base-asset amount down to an executable increment.
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
// the amount and increment are already expressed in order units.
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

	firstRat, ok := new(big.Rat).SetString(first.String())
	if !ok {
		return decimal.Zero, fmt.Errorf("cannot parse first base increment %q", first)
	}
	secondRat, ok := new(big.Rat).SetString(second.String())
	if !ok {
		return decimal.Zero, fmt.Errorf("cannot parse second base increment %q", second)
	}

	var numeratorProduct, numeratorLCM, denominatorGCD big.Int
	numeratorProduct.Mul(firstRat.Num(), secondRat.Num())
	numeratorLCM.Div(&numeratorProduct, greatestCommonDivisor(firstRat.Num(), secondRat.Num()))
	denominatorGCD.GCD(nil, nil, firstRat.Denom(), secondRat.Denom())

	resultRat := new(big.Rat).SetFrac(&numeratorLCM, &denominatorGCD)
	scale := max(decimalScale(first.String()), decimalScale(second.String()))
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

func decimalScale(value string) int {
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
