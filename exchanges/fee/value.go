package fee

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var (
	// ErrCannotCompare defines an error when you cannot compare between
	// interfaces
	ErrCannotCompare                   = errors.New("cannot compare")
	errInvalid                         = errors.New("invalid value")
	errBlockchainEmpty                 = errors.New("blockchain string is empty")
	errAmountIsLessThanMinimumRequired = errors.New("amount is less than minimum required")
)

// Value defines custom fee value calculation functionality
type Value interface {
	// GetFee returns the fee, either a percentage or fixed amount. The amount
	// param is only used as a switch for if fees scale with potential amounts.
	GetFee(amount float64) (decimal.Decimal, error)
	// Display displays either the float64 value or the JSON of the struct as a
	// string to be unmarshalled via GRPC if needed.
	Display() (string, error)
	// Validate checks current stored struct values for any issues.
	Validate() error
	// LessThan determines if the current fee is less than another. Most of
	// the time this is not needed.
	LessThan(val Value) (bool, error)
}

// Convert returns a "Standard" struct depicting a single float value that
// implements the value interface.
func Convert(f float64) Value {
	return Standard{Decimal: decimal.NewFromFloat(f)}
}

// Standard standard float fee
type Standard struct {
	decimal.Decimal
}

// GetFee returns the fee, either a percentage or fixed amount. The amount
// param is only used as a switch for if fees scale with potential amounts.
func (s Standard) GetFee(amount float64) (decimal.Decimal, error) {
	return s.Decimal, nil
}

// Display displays either the float64 value or the JSON of the struct as a
// string to be unmarshalled via GRPC if needed.
func (s Standard) Display() (string, error) {
	return s.String(), nil
}

// Validate checks current stored struct values for any issues.
func (s Standard) Validate() error {
	if s.Decimal.LessThan(decimal.Zero) {
		return errInvalid
	}
	return nil
}

// LessThan determines if the current fee is less than another. Most of the time
// this is not needed.
func (s Standard) LessThan(val Value) (bool, error) {
	other, ok := val.(Standard)
	if !ok {
		return false, fmt.Errorf("%w a non standard value %t", ErrCannotCompare, val)
	}
	return s.GreaterThan(decimal.Zero) &&
		other.GreaterThan(decimal.Zero) &&
		s.Decimal.LessThan(other.Decimal), nil
}

// ConvertWithAmount takes in two fees for when fees are based on amount
// thresholds
func ConvertWithAmount(feeWhenLower, feeWhenHigherOrEqual, amount float64) Value {
	return Switch{
		FeeWhenLower:         decimal.NewFromFloat(feeWhenLower),
		FeeWhenHigherOrEqual: decimal.NewFromFloat(feeWhenHigherOrEqual),
		Amount:               decimal.NewFromFloat(amount),
	}
}

// Switch defines a holder for upper and lower bands of fees based on an amount
type Switch struct {
	FeeWhenLower         decimal.Decimal `json:"feeWhenLower"`
	FeeWhenHigherOrEqual decimal.Decimal `json:"feeWhenHigherOrEqual"`
	Amount               decimal.Decimal `json:"amount"`
}

// GetFee returns the fee, either a percentage or fixed amount. The amount
// param is only used as a switch for if fees scale with potential amounts.
func (s Switch) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	if amt.GreaterThanOrEqual(s.Amount) {
		return s.FeeWhenHigherOrEqual, nil
	}
	return s.FeeWhenLower, nil
}

// Display displays either the float64 value or the JSON of the struct as a
// string to be unmarshalled via GRPC if needed.
func (s Switch) Display() (string, error) {
	data, err := json.Marshal(s)
	return string(data), err
}

// Validate checks current stored struct values for any issues.
func (s Switch) Validate() error {
	if s.FeeWhenLower.LessThan(decimal.Zero) {
		return fmt.Errorf("fee when lower %w", errInvalid)
	}
	if s.FeeWhenHigherOrEqual.LessThan(decimal.Zero) {
		return fmt.Errorf("fee when higher or equal %w", errInvalid)
	}
	if s.Amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("fee amount %w", errInvalid)
	}
	return nil
}

// LessThan determines if the current fee is less than another. Most of the time
// this is not needed.
func (s Switch) LessThan(_ Value) (bool, error) {
	return false, ErrCannotCompare
}

// ConvertBlockchain is a placeholder for blockchain specific fees
func ConvertBlockchain(blockchain string) Value {
	return Blockchain(blockchain)
}

// Blockchain is a subtype implementing the value interface to designate
// certain fee options as a blockchain component. This will be changed in
// the future when another PR can help resolve this.
type Blockchain string

// GetFee returns the fee, either a percentage or fixed amount. The amount
// param is only used as a switch for if fees scale with potential amounts.
func (b Blockchain) GetFee(amount float64) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

// Display displays either the float64 value or the JSON of the struct as a
// string to be unmarshalled via GRPC if needed.
func (b Blockchain) Display() (string, error) {
	return fmt.Sprintf("current fees are %s blockchain transaction fees", b), nil
}

// Validate checks current stored struct values for any issues.
func (b Blockchain) Validate() error {
	if b == "" {
		return errBlockchainEmpty
	}
	return nil
}

// LessThan determines if the current fee is less than another. Most of the time
// this is not needed.
func (b Blockchain) LessThan(_ Value) (bool, error) {
	return false, ErrCannotCompare
}

// ConvertWithMaxAndMin returns a fee value with maximum and minimum fees
func ConvertWithMaxAndMin(fee, maximum, minimum float64) Value {
	return MinMax{
		Fee:     decimal.NewFromFloat(fee),
		Maximum: decimal.NewFromFloat(maximum),
		Minimum: decimal.NewFromFloat(minimum),
	}
}

// MinMax implements the value interface for when there are min and max fees
type MinMax struct {
	Minimum decimal.Decimal `json:"minimumFee"`
	Maximum decimal.Decimal `json:"maximumFee"`
	Fee     decimal.Decimal `json:"fee"`
}

// GetFee returns the fee, either a percentage or fixed amount. The amount
// param is only used as a switch for if fees scale with potential amounts.
func (m MinMax) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	potential := amt.Mul(m.Fee)
	if m.Maximum.GreaterThan(decimal.Zero) && potential.GreaterThan(m.Maximum) {
		return m.Maximum, nil
	}
	if m.Minimum.GreaterThan(decimal.Zero) && potential.LessThan(m.Minimum) {
		return m.Minimum, nil
	}
	return potential, nil
}

// Display displays either the float64 value or the JSON of the struct as a
// string to be unmarshalled via GRPC if needed.
func (m MinMax) Display() (string, error) {
	data, err := json.Marshal(m)
	return string(data), err
}

// Validate checks current stored struct values for any issues.
func (m MinMax) Validate() error {
	if m.Fee.LessThan(decimal.Zero) {
		return fmt.Errorf("%w fee", errInvalid)
	}
	if m.Maximum.LessThan(decimal.Zero) {
		return fmt.Errorf("%w maximum fee", errInvalid)
	}
	if m.Minimum.LessThan(decimal.Zero) {
		return fmt.Errorf("%w minimum fee", errInvalid)
	}
	return nil
}

// LessThan determines if the current fee is less than another. Most of the time
// this is not needed.
func (m MinMax) LessThan(_ Value) (bool, error) {
	return false, ErrCannotCompare
}

// ConvertWithMinimumAmount returns a value with a minimum amount required
func ConvertWithMinimumAmount(fee, minAmount float64) Value {
	return WithMinimumAmount{
		Fee:           decimal.NewFromFloat(fee),
		MinimumAmount: decimal.NewFromFloat(minAmount),
	}
}

// WithMinimumAmount defines a fee that requires atleast a minimum amount to
// be allowed for the transfer.
type WithMinimumAmount struct {
	MinimumAmount decimal.Decimal `json:"withMinimumAmount"`
	Fee           decimal.Decimal `json:"fee"`
}

// GetFee returns the fee, either a percentage or fixed amount. The amount
// param is only used as a switch for if fees scale with potential amounts.
func (m WithMinimumAmount) GetFee(amount float64) (decimal.Decimal, error) {
	amt := decimal.NewFromFloat(amount)
	if amt.LessThan(m.MinimumAmount) {
		return decimal.Zero, errAmountIsLessThanMinimumRequired
	}
	return m.Fee, nil
}

// Display displays either the float64 value or the JSON of the struct as a
// string to be unmarshalled via GRPC if needed.
func (m WithMinimumAmount) Display() (string, error) {
	data, err := json.Marshal(m)
	return string(data), err
}

// Validate checks current stored struct values for any issues.
func (m WithMinimumAmount) Validate() error {
	if m.Fee.LessThan(decimal.Zero) {
		return fmt.Errorf("%w fee %s", errInvalid, m.Fee)
	}
	if m.MinimumAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%w minimum amount %s", errInvalid, m.MinimumAmount)
	}
	return nil
}

// LessThan determines if the current fee is less than another. Most of the time
// this is not needed.
func (m WithMinimumAmount) LessThan(_ Value) (bool, error) {
	return false, ErrCannotCompare
}
