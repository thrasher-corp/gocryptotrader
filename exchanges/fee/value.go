package fee

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var errInvalid = errors.New("invalid value")

// Value defines fee value calculation functionality
type Value interface {
	GetFee(amount float64) decimal.Decimal
	Display() (string, error)
	Validate() error
	LessThan(val Value) (bool, error)
}

// Convert returns a pointer to a float64 for use in explicit exported
// parameters to define functionality. TODO: Maybe return a *fee.Value type
// consideration
func Convert(f float64) Value {
	return Standard{Decimal: decimal.NewFromFloat(f)}
}

// ConvertWithAmount takes in two fees for when fees are based of amount
// thresholds
func ConvertWithAmount(feeWhenLower, feeWhenHigherOrEqual, amount float64) Value {
	return Switch{
		FeeWhenLower:         decimal.NewFromFloat(feeWhenLower),
		FeeWhenHigherOrEqual: decimal.NewFromFloat(feeWhenHigherOrEqual),
		Amount:               decimal.NewFromFloat(amount),
	}
}

// Standard standard float fee
type Standard struct {
	decimal.Decimal
}

// GetFee implements Value interface
func (s Standard) GetFee(amount float64) decimal.Decimal {
	return s.Decimal
}

// Display implements Value interface
func (s Standard) Display() (string, error) {
	return s.String(), nil
}

// Display implements Value interface
func (s Standard) Validate() error {
	if s.Decimal.LessThan(decimal.Zero) {
		return errInvalid
	}
	return nil
}

// Display implements Value interface
func (s Standard) LessThan(val Value) (bool, error) {
	other, ok := val.(Standard)
	if !ok {
		return false, fmt.Errorf("cannot compare a non standard value %t", val)
	}
	return s.GreaterThan(decimal.Zero) &&
		other.GreaterThan(decimal.Zero) &&
		s.Decimal.LessThan(other.Decimal), nil
}

// Switch defines a holder for upper and lower bands of fees based on an amount
type Switch struct {
	FeeWhenLower         decimal.Decimal `json:"feeWhenLower"`
	FeeWhenHigherOrEqual decimal.Decimal `json:"feeWhenHigherOrEqual"`
	Amount               decimal.Decimal `json:"amount"`
}

// GetFee implements Value interface
func (s Switch) GetFee(amount float64) decimal.Decimal {
	amt := decimal.NewFromFloat(amount)
	if amt.GreaterThanOrEqual(s.Amount) {
		return s.FeeWhenHigherOrEqual
	}
	return s.FeeWhenLower
}

// Display implements Value interface
func (s Switch) Display() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Display implements Value interface
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

// Display implements Value interface
func (s Switch) LessThan(_ Value) (bool, error) {
	return false, errors.New("cannot compare")
}
