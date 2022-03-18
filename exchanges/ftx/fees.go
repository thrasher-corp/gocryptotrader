package ftx

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

var errExchangeNotSet = errors.New("exchange has not been set")

// GetLoadableTransferFees assigns connectors for fetching fees adhoc for
// withdrawal
func (f *FTX) GetLoadableTransferFees() ([]fee.Transfer, error) {
	pairs, err := f.GetAvailablePairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	load := pairs.GetCurrencies()

	var transferFees []fee.Transfer
	for x := range load {
		transferFees = append(transferFees, fee.Transfer{
			Currency:   load[x],
			Deposit:    fee.Convert(0),
			Withdrawal: GetAdhocFees(load[x], f),
		})
	}
	return transferFees, nil
}

// UpdateCommissionFees updates all the fees associated with the asset type.
func (f *FTX) UpdateCommissionFees(ctx context.Context, a asset.Item) error {
	if a != asset.Spot && a != asset.Futures {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	ai, err := f.GetAccountInfo(ctx, "")
	if err != nil {
		return err
	}
	return f.Fees.LoadDynamicFeeRate(ai.MakerFee, ai.TakerFee, a, fee.OmitPair)
}

// UpdateTransferFees updates transfer fees for cryptocurrency withdrawal and
// deposits for this exchange
func (f *FTX) UpdateTransferFees(ctx context.Context) error {
	transfers, err := f.GetLoadableTransferFees()
	if err != nil {
		return err
	}
	return f.Fees.LoadChainTransferFees(transfers)
}

// GetAdhocFees returns an Adhoc type that will utilise the REST endpoint to
// fetch fee per request.
func GetAdhocFees(c currency.Code, exch *FTX) fee.Value {
	return &Adhoc{Currency: c, Exch: exch}
}

// Adhoc holds a currency and a pointer to FTX struct to access methods to
// send requests using the fee interface fee.Value
type Adhoc struct {
	Currency currency.Code
	Exch     *FTX
}

// GetFee returns the fee, either a percentage or fixed amount. The amount
// param is only used as a switch for if fees scale with potential amounts.
func (a Adhoc) GetFee(ctx context.Context, amount float64, destinationAddress, tag string) (decimal.Decimal, error) {
	fmt.Println("wow dude")
	if a.Exch == nil {
		return decimal.Zero, errExchangeNotSet
	}
	fmt.Println("wow dude")
	withdrawalFee, err := a.Exch.GetWithdrawalFee(ctx, a.Currency, amount, destinationAddress, tag)
	if err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromFloat(withdrawalFee.Fee), nil
}

// Display displays either the float64 value or the JSON of the struct as a
// string to be unmarshalled via GRPC if needed.
func (a Adhoc) Display() (string, error) {
	if a.Exch == nil {
		return "", errExchangeNotSet
	}

	return fmt.Sprintf("Currency: %s using %s method %T",
		a.Currency,
		a.Exch.Name,
		a.Exch.GetWithdrawalFee), nil
}

// Validate checks current stored struct values for any issues.
func (a Adhoc) Validate() error {
	if a.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if a.Exch == nil {
		return errExchangeNotSet
	}
	return nil
}

// LessThan determines if the current fee is less than another. Most of the time
// this is not needed.
func (a Adhoc) LessThan(val fee.Value) (bool, error) {
	return false, fmt.Errorf("%w %t", fee.ErrCannotCompare, val)
}
