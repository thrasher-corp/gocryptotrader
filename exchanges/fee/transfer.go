package fee

import "github.com/shopspring/decimal"

// Transfer defines usually static whole number values. But has the option of
// being percentage value.
type Transfer struct {
	// IsPercentage defines if the transfer fee is a percentage rather than a set
	// amount.
	IsPercentage bool
	// Deposit defines a deposit fee
	Deposit float64
	// Withdrawal defines a withdrawal fee
	Withdrawal float64
}

// convert returns an internal transfer struct
func (t Transfer) convert() *transfer {
	return &transfer{
		Percentage: t.IsPercentage,
		Deposit:    decimal.NewFromFloat(t.Deposit),
		Withdrawal: decimal.NewFromFloat(t.Withdrawal),
	}
}

// transfer defines an internal fee structure
type transfer struct {
	// Percentage defines if the transfer fee is a percentage rather than a set
	// amount.
	Percentage bool
	// Deposit defines a deposit fee as a decimal value
	Deposit decimal.Decimal
	// Withdrawal defines a withdrawal fee as a decimal value
	Withdrawal decimal.Decimal
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
