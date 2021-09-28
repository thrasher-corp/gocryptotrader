package fee

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var (
	errDepositIsInvalid    = errors.New("deposit is invalid")
	errWithdrawalIsInvalid = errors.New("withdrawal is invalid")
	errMaxLessThanMin      = errors.New("maximum value is less than minimum")
	errTransferIsNil       = errors.New("transfer is nil")
)

// Transfer defines usually static whole number values. But has the option of
// being percentage value. Pointer values to define functionality. NOTE: Please
// use fee package Convert function to define pointer type.
type Transfer struct {
	// IsPercentage defines if the transfer fee is a percentage rather than a set
	// amount.
	IsPercentage bool
	// Deposit defines a deposit fee
	Deposit *float64
	// MinimumDeposit defines the minimal allowable deposit amount
	MinimumDeposit *float64
	// MaximumDeposit defines the maximum allowable deposit amount
	MaximumDeposit *float64
	// Withdrawal defines a withdrawal fee
	Withdrawal *float64
	// MinimumWithdrawal defines the minimal allowable withdrawal amount
	MinimumWithdrawal *float64
	// MaximumWithdrawal defines the maximum allowable withdrawal amount
	MaximumWithdrawal *float64
}

// convert returns an internal transfer struct
func (t Transfer) convert() *transfer {
	c := transfer{Percentage: t.IsPercentage}
	if t.Deposit != nil {
		c.DepositEnabled = true
		c.Deposit = decimal.NewFromFloat(*t.Deposit)
	}
	if t.MinimumDeposit != nil {
		c.MinimumDeposit = decimal.NewFromFloat(*t.MinimumDeposit)
	}
	if t.MaximumDeposit != nil {
		c.MaximumDeposit = decimal.NewFromFloat(*t.MaximumDeposit)
	}

	if t.Withdrawal != nil {
		c.WithdrawalEnabled = true
		c.Withdrawal = decimal.NewFromFloat(*t.Withdrawal)
	}
	if t.MinimumWithdrawal != nil {
		c.MinimumWithdrawal = decimal.NewFromFloat(*t.MinimumWithdrawal)
	}
	if t.MaximumWithdrawal != nil {
		c.MaximumWithdrawal = decimal.NewFromFloat(*t.MaximumWithdrawal)
	}
	return &c
}

// validate validates transfer values
func (t Transfer) validate() error {
	if t.Deposit != nil && *t.Deposit < 0 {
		return errDepositIsInvalid
	}
	if t.MaximumDeposit != nil && *t.MaximumDeposit < 0 {
		return fmt.Errorf("maximum %w", errDepositIsInvalid)
	}
	if t.MinimumDeposit != nil && *t.MinimumDeposit < 0 {
		return fmt.Errorf("minimum %w", errDepositIsInvalid)
	}
	if t.MaximumDeposit != nil &&
		t.MinimumDeposit != nil &&
		*t.MaximumDeposit != 0 &&
		*t.MinimumDeposit != 0 &&
		*t.MaximumDeposit < *t.MinimumDeposit {
		return fmt.Errorf("deposit %w", errMaxLessThanMin)
	}

	if t.Withdrawal != nil && *t.Withdrawal < 0 {
		return errWithdrawalIsInvalid
	}
	if t.MaximumWithdrawal != nil && *t.MaximumWithdrawal < 0 {
		return fmt.Errorf("maximum %w", errWithdrawalIsInvalid)
	}
	if t.MinimumWithdrawal != nil && *t.MinimumWithdrawal < 0 {
		return fmt.Errorf("minimum %w", errWithdrawalIsInvalid)
	}
	if t.MaximumWithdrawal != nil &&
		t.MinimumWithdrawal != nil &&
		*t.MaximumWithdrawal != 0 &&
		*t.MinimumWithdrawal != 0 &&
		*t.MaximumWithdrawal < *t.MinimumWithdrawal {
		return fmt.Errorf("withdrawal %w", errMaxLessThanMin)
	}
	return nil
}

// transfer defines an internal fee structure
type transfer struct {
	// Percentage defines if the transfer fee is a percentage rather than a set
	// amount.
	Percentage     bool
	DepositEnabled bool
	// Deposit defines a deposit fee as a decimal value
	Deposit decimal.Decimal
	// MinimumDeposit defines the minimal allowable deposit amount
	MinimumDeposit decimal.Decimal
	// MaximumDeposit defines the maximum allowable deposit amount
	MaximumDeposit    decimal.Decimal
	WithdrawalEnabled bool
	// Withdrawal defines a withdrawal fee as a decimal value
	Withdrawal decimal.Decimal
	// MinimumWithdrawal defines the minimal allowable withdrawal amount
	MinimumWithdrawal decimal.Decimal
	// MaximumWithdrawal defines the maximum allowable withdrawal amount
	MaximumWithdrawal decimal.Decimal
}

// convert returns an package exportable type snapshot of current internal
// transfer details
func (t transfer) convert() Transfer {
	var deposit, maxDeposit, minDeposit, withdrawal, maxWithdrawal, minWithdrawal *float64

	// In the case of deposit or withdrawal being disabled; skip max and min settings.
	if t.DepositEnabled {
		d, _ := t.Deposit.Float64()
		maxD, _ := t.MaximumDeposit.Float64()
		minD, _ := t.MinimumDeposit.Float64()
		deposit = &d
		maxDeposit = &maxD
		minDeposit = &minD
	}

	if t.WithdrawalEnabled {
		w, _ := t.Withdrawal.Float64()
		maxW, _ := t.MaximumWithdrawal.Float64()
		minW, _ := t.MinimumWithdrawal.Float64()
		withdrawal = &w
		maxWithdrawal = &maxW
		minWithdrawal = &minW
	}

	return Transfer{
		IsPercentage:      t.Percentage,
		Deposit:           deposit,
		MaximumDeposit:    maxDeposit,
		MinimumDeposit:    minDeposit,
		Withdrawal:        withdrawal,
		MaximumWithdrawal: maxWithdrawal,
		MinimumWithdrawal: minWithdrawal,
	}
}

// update updates using incoming transfer information
func (t *transfer) update(incoming Transfer) error {
	if t == nil {
		return errTransferIsNil
	}

	if t.Percentage != incoming.IsPercentage {
		return errFeeTypeMismatch
	}

	if incoming.Deposit != nil && *incoming.Deposit > 0 {
		t.Deposit = decimal.NewFromFloat(*incoming.Deposit)
		if incoming.MinimumDeposit != nil && *incoming.MinimumDeposit > 0 {
			t.MinimumDeposit = decimal.NewFromFloat(*incoming.MinimumDeposit)
		}
		if incoming.MaximumDeposit != nil && *incoming.MaximumDeposit > 0 {
			t.MaximumDeposit = decimal.NewFromFloat(*incoming.MaximumDeposit)
		}
	}

	if incoming.Withdrawal != nil && *incoming.Withdrawal > 0 {
		t.Withdrawal = decimal.NewFromFloat(*incoming.Withdrawal)
		if incoming.MinimumWithdrawal != nil && *incoming.MinimumWithdrawal > 0 {
			t.MinimumWithdrawal = decimal.NewFromFloat(*incoming.MinimumWithdrawal)
		}
		if incoming.MaximumWithdrawal != nil && *incoming.MaximumWithdrawal > 0 {
			t.MaximumWithdrawal = decimal.NewFromFloat(*incoming.MaximumWithdrawal)
		}
	}

	return nil
}

// calculate returns the transfer fee total based on internal loaded values
func (t transfer) calculate(fee decimal.Decimal, amount float64) (float64, error) {
	if amount == 0 {
		return 0, errAmountIsZero
	}
	// TODO: Add fees based on trade volume of this asset.
	// TODO: Add fees when the amount is less than required.
	if !t.Percentage {
		// Returns the whole number
		setValue, _ := fee.Float64()
		return setValue, nil
	}
	// Return fee derived from percentage and amount values
	var val = decimal.NewFromFloat(amount).Mul(fee)
	rVal, _ := val.Float64()
	return rVal, nil
}
