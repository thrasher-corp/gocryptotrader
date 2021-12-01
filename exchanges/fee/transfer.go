package fee

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
)

var (
	errDepositIsInvalid    = errors.New("deposit is invalid")
	errWithdrawalIsInvalid = errors.New("withdrawal is invalid")
	errMaxLessThanMin      = errors.New("maximum value is less than minimum")
	errTransferIsNil       = errors.New("transfer is nil")
)

// Transfer defines usually static whole number values. But has the option of
// being percentage value. NOTE: Please see value.go for Value interface
// functionality and general implementations.
type Transfer struct {
	// IsPercentage defines if the transfer fee is a percentage rather than a
	// fixed amount.
	IsPercentage bool
	// Deposit defines a deposit fee
	Deposit Value
	// MinimumDeposit defines the minimal allowable deposit amount
	MinimumDeposit Value
	// MaximumDeposit defines the maximum allowable deposit amount
	MaximumDeposit Value
	// Withdrawal defines a withdrawal fee
	Withdrawal Value
	// MinimumWithdrawal defines the minimal allowable withdrawal amount
	MinimumWithdrawal Value
	// MaximumWithdrawal defines the maximum allowable withdrawal amount
	MaximumWithdrawal Value
	// Currency defines a currency identifier
	Currency currency.Code
	// Defines the chain that it can be withdrawn or deposited with. e.g. BEP20
	// for BNB or other wrapped tokens on the same protocol.
	Chain string
	// BankTransfer defines the bank transfer protocol for delivering or
	// receiving fiat currency from an exchange.
	BankTransfer bank.Transfer
}

// convert returns an internal transfer struct
func (t Transfer) convert() *transfer {
	c := transfer{Percentage: t.IsPercentage}
	if t.Deposit != nil {
		c.DepositEnabled = true
		c.Deposit = t.Deposit
		if t.MinimumDeposit != nil {
			c.MinimumDeposit = t.MinimumDeposit
		}
		if t.MaximumDeposit != nil {
			c.MaximumDeposit = t.MaximumDeposit
		}
	}

	if t.Withdrawal != nil {
		c.WithdrawalEnabled = true
		c.Withdrawal = t.Withdrawal
		if t.MinimumWithdrawal != nil {
			c.MinimumWithdrawal = t.MinimumWithdrawal
		}
		if t.MaximumWithdrawal != nil {
			c.MaximumWithdrawal = t.MaximumWithdrawal
		}
	}
	return &c
}

// validate validates transfer values
func (t Transfer) validate() error {
	if t.Currency.IsEmpty() {
		return errCurrencyIsEmpty
	}
	if t.Deposit != nil {
		err := t.Deposit.Validate()
		if err != nil {
			return fmt.Errorf("deposit %w", err)
		}

		if t.MaximumDeposit != nil {
			err := t.MaximumDeposit.Validate()
			if err != nil {
				return fmt.Errorf("maximum deposit %w", err)
			}
		}
		if t.MinimumDeposit != nil {
			err := t.MinimumDeposit.Validate()
			if err != nil {
				return fmt.Errorf("minimum deposit %w", err)
			}
		}

		if t.MaximumDeposit != nil &&
			t.MinimumDeposit != nil {
			b, err := t.MaximumDeposit.LessThan(t.MinimumDeposit)
			if err != nil {
				return err
			}
			if b {
				return fmt.Errorf("deposit %w", errMaxLessThanMin)
			}
		}
	}

	if t.Withdrawal != nil {
		err := t.Withdrawal.Validate()
		if err != nil {
			return fmt.Errorf("%s withdrawal %w", t.Currency, err)
		}

		if t.MaximumWithdrawal != nil {
			err := t.MaximumWithdrawal.Validate()
			if err != nil {
				return fmt.Errorf("maximum withdrawal %w", err)
			}
		}
		if t.MinimumWithdrawal != nil {
			err := t.MinimumWithdrawal.Validate()
			if err != nil {
				return fmt.Errorf("minimum withdrawal %w", err)
			}
		}

		if t.MaximumWithdrawal != nil &&
			t.MinimumWithdrawal != nil {
			b, err := t.MaximumWithdrawal.LessThan(t.MinimumWithdrawal)
			if err != nil {
				return err
			}
			if b {
				return fmt.Errorf("withdrawal %w", errMaxLessThanMin)
			}
		}
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
	Deposit Value
	// MinimumDeposit defines the minimal allowable deposit amount
	MinimumDeposit Value
	// MaximumDeposit defines the maximum allowable deposit amount
	MaximumDeposit    Value
	WithdrawalEnabled bool
	// Withdrawal defines a withdrawal fee as a decimal value
	Withdrawal Value
	// MinimumWithdrawal defines the minimal allowable withdrawal amount
	MinimumWithdrawal Value
	// MaximumWithdrawal defines the maximum allowable withdrawal amount
	MaximumWithdrawal Value
}

// convert returns an package exportable type snapshot of current internal
// transfer details
func (t transfer) convert() Transfer {
	return Transfer{
		IsPercentage:      t.Percentage,
		Deposit:           t.Deposit,
		MaximumDeposit:    t.MaximumDeposit,
		MinimumDeposit:    t.MinimumDeposit,
		Withdrawal:        t.Withdrawal,
		MaximumWithdrawal: t.MaximumWithdrawal,
		MinimumWithdrawal: t.MinimumWithdrawal,
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

	if incoming.Deposit != nil {
		t.DepositEnabled = true
		t.Deposit = incoming.Deposit
		if incoming.MinimumDeposit != nil {
			t.MinimumDeposit = incoming.MinimumDeposit
		}
		if incoming.MaximumDeposit != nil {
			t.MaximumDeposit = incoming.MaximumDeposit
		}
	} else {
		t.DepositEnabled = false
		t.Deposit = nil
		t.MaximumDeposit = nil
		t.MinimumDeposit = nil
	}

	if incoming.Withdrawal != nil {
		t.WithdrawalEnabled = true
		t.Withdrawal = incoming.Withdrawal
		if incoming.MinimumWithdrawal != nil {
			t.MinimumWithdrawal = incoming.MinimumWithdrawal
		}
		if incoming.MaximumWithdrawal != nil {
			t.MaximumWithdrawal = incoming.MaximumWithdrawal
		}
	} else {
		t.Withdrawal = nil
		t.WithdrawalEnabled = false
		t.MaximumWithdrawal = nil
		t.MinimumWithdrawal = nil
	}

	return nil
}

// calculate returns the transfer fee total based on internal loaded values
func (t transfer) calculate(val Value, amount float64) (float64, error) {
	if amount == 0 {
		return 0, errAmountIsZero
	}
	// When getting fee it is highly dependant on underlying interface value
	// see value.go for different amount tier systems definitions.
	fee, err := val.GetFee(amount)
	if err != nil {
		return 0, err
	}
	if !t.Percentage { // TODO: Needs to be checked, might need to have the
		// p value at the interface layer

		// Returns the whole number
		feeFloat, _ := fee.Float64()
		return feeFloat, nil
	}
	// Return fee derived from percentage and amount values
	rVal, _ := decimal.NewFromFloat(amount).Mul(fee).Float64()
	return rVal, nil
}
