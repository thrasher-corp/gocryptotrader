package fee

import (
	"errors"

	"github.com/shopspring/decimal"
)

// Transfer defines usually static whole number values. But has the option of
// being percentage value.
type Transfer struct {
	// IsPercentage defines if the transfer fee is a percentage rather than a set
	// amount.
	IsPercentage bool
	// Deposit defines a deposit fee
	Deposit float64
	// MinimumDeposit defines the minimal allowable deposit amount
	MinimumDeposit float64
	// MaximumDeposit defines the maximum allowable deposit amount
	MaximumDeposit float64
	// Withdrawal defines a withdrawal fee
	Withdrawal float64
	// MinimumWithdrawal defines the minimal allowable withdrawal amount
	MinimumWithdrawal float64
	// MaximumWithdrawal defines the maximum allowable withdrawal amount
	MaximumWithdrawal float64
}

// convert returns an internal transfer struct
func (t Transfer) convert() *transfer {
	return &transfer{
		Percentage:        t.IsPercentage,
		Deposit:           decimal.NewFromFloat(t.Deposit),
		MinimumDeposit:    decimal.NewFromFloat(t.MinimumDeposit),
		MaximumDeposit:    decimal.NewFromFloat(t.MaximumDeposit),
		Withdrawal:        decimal.NewFromFloat(t.Withdrawal),
		MinimumWithdrawal: decimal.NewFromFloat(t.MinimumWithdrawal),
		MaximumWithdrawal: decimal.NewFromFloat(t.MaximumWithdrawal),
	}
}

// transfer defines an internal fee structure
type transfer struct {
	// Percentage defines if the transfer fee is a percentage rather than a set
	// amount.
	Percentage bool
	// Deposit defines a deposit fee as a decimal value
	Deposit decimal.Decimal
	// MinimumDeposit defines the minimal allowable deposit amount
	MinimumDeposit decimal.Decimal
	// MaximumDeposit defines the maximum allowable deposit amount
	MaximumDeposit decimal.Decimal
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
	deposit, _ := t.Deposit.Float64()
	withdrawal, _ := t.Withdrawal.Float64()
	return Transfer{
		IsPercentage: t.Percentage,
		Deposit:      deposit,
		Withdrawal:   withdrawal,
	}
}

var errTransferIsNil = errors.New("transfer is nil")

// update updates using incoming transfer information
func (t *transfer) update(incoming Transfer) error {
	if t == nil {
		return errTransferIsNil
	}

	if t.Percentage != incoming.IsPercentage {
		return errFeeTypeMismatch
	}

	if incoming.Deposit > 0 {
		t.Deposit = decimal.NewFromFloat(incoming.Deposit)
	}

	if incoming.MinimumDeposit > 0 {
		t.MinimumDeposit = decimal.NewFromFloat(incoming.MinimumDeposit)
	}

	if incoming.MaximumDeposit > 0 {
		t.MaximumDeposit = decimal.NewFromFloat(incoming.MaximumDeposit)
	}

	if incoming.Withdrawal > 0 {
		t.Withdrawal = decimal.NewFromFloat(incoming.Withdrawal)
	}

	if incoming.MinimumWithdrawal > 0 {
		t.MinimumWithdrawal = decimal.NewFromFloat(incoming.MinimumWithdrawal)
	}

	if incoming.MaximumWithdrawal > 0 {
		t.MaximumWithdrawal = decimal.NewFromFloat(incoming.MaximumWithdrawal)
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
