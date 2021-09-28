package fee

import (
	"errors"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrDefinitionsAreNil defines if the exchange specific fee definitions
	// have bot been loaded or set up.
	ErrDefinitionsAreNil = errors.New("fee definitions are nil")

	errCurrencyIsEmpty         = errors.New("currency is empty")
	errTransferFeeNotFound     = errors.New("transfer fee not found")
	errBankTransferFeeNotFound = errors.New("bank transfer fee not found")
	errPriceIsZero             = errors.New("price is zero")
	errAmountIsZero            = errors.New("amount is zero")
	errFeeTypeMismatch         = errors.New("fee type mismatch")
	errRateNotFound            = errors.New("rate not found")
	errCommissionRateNotFound  = errors.New("Commission rate not found")
	errTakerInvalid            = errors.New("taker is invalid")
	errMakerInvalid            = errors.New("maker is invalid")
	errMakerBiggerThanTaker    = errors.New("maker cannot be bigger than taker")
)

// NewFeeDefinitions generates a new fee struct for exchange usage
func NewFeeDefinitions() *Definitions {
	return &Definitions{
		commissions:      make(map[asset.Item]*CommissionInternal),
		transfers:        make(map[asset.Item]map[*currency.Item]*transfer),
		bankingTransfers: make(map[BankTransaction]map[*currency.Item]*transfer),
	}
}

// Convert returns a pointer to a float64 for use in explicit exported
// parameters to define functionality. TODO: Maybe return a *fee.Value type
// consideration
func Convert(f float64) *float64 { return &f }

// Definitions defines the full fee definitions for different currencies
// TODO: Eventually upgrade with key manager for different fees associated
// with different accounts/keys.
type Definitions struct {
	// Commission is the holder for the up to date comission rates for the assets.
	commissions map[asset.Item]*CommissionInternal
	// transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be real values.
	transfers map[asset.Item]map[*currency.Item]*transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	bankingTransfers map[BankTransaction]map[*currency.Item]*transfer
	mtx              sync.RWMutex
}

// LoadDynamic loads the current dynamic account fee structure for maker and
// taker values.
func (d *Definitions) LoadDynamic(maker, taker float64, a asset.Item) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}
	if maker < 0 {
		return errMakerInvalid
	}
	if taker < 0 {
		return errTakerInvalid
	}
	if maker > taker {
		return errMakerBiggerThanTaker
	}
	if !a.IsValid() {
		return fmt.Errorf("%s: %w", a, asset.ErrNotSupported)
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	c, ok := d.commissions[a]
	if !ok {
		return errCommissionRateNotFound
	}
	c.load(maker, taker)
	return nil
}

// LoadStatic loads predefined custom long term fee structures for items like
// worst case scenario values, transfer fees to and from exchanges, and
// international bank transfer rates.
func (d *Definitions) LoadStatic(o Options) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if err := o.validate(); err != nil {
		return err
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	// Loads standard commission rates based on asset item
	for a, value := range o.Commission {
		d.commissions[a] = value.convert()
	}

	// Loads exchange withdrawal and deposit fees
	for as, m1 := range o.Transfer {
		for code, value := range m1 {
			cTrans, ok := d.transfers[as]
			if !ok {
				cTrans = make(map[*currency.Item]*transfer)
				d.transfers[as] = cTrans
			}
			cTrans[code.Item] = value.convert()
		}
	}

	// Loads international banking withdrawal and deposit fees
	for transactionType, m1 := range o.BankingTransfer {
		for code, value := range m1 {
			bTrans, ok := d.bankingTransfers[transactionType]
			if !ok {
				bTrans = make(map[*currency.Item]*transfer)
				d.bankingTransfers[transactionType] = bTrans
			}
			bTrans[code.Item] = value.convert()
		}
	}
	return nil
}

// CalculateMaker returns the fee amount derived from the price, amount and fee
// percentage.
func (d *Definitions) CalculateMaker(price, amount float64, a asset.Item) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commissions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return c.CalculateMaker(price, amount)
}

// CalculateWorstCaseMaker returns the fee amount derived from the price, amount
// and fee percentage using the worst-case scenario trading fee.
func (d *Definitions) CalculateWorstCaseMaker(price, amount float64, a asset.Item) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commissions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return c.CalculateWorstCaseMaker(price, amount)
}

// GetMaker returns the maker fee value and if it is a percentage or whole
// number
func (d *Definitions) GetMaker(a asset.Item) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commissions[a]
	if !ok {
		return 0, false, errRateNotFound
	}
	fee, isSetAmount = c.GetMaker()
	return
}

// CalculateTaker returns the fee amount derived from the price, amount and fee
// percentage.
func (d *Definitions) CalculateTaker(price, amount float64, a asset.Item) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commissions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return c.CalculateTaker(price, amount)
}

// CalculateWorstCaseTaker returns the fee amount derived from the price, amount
// and fee percentage using the worst-case scenario trading fee.
func (d *Definitions) CalculateWorstCaseTaker(price, amount float64, a asset.Item) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commissions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return c.CalculateWorstCaseTaker(price, amount)
}

// GetTaker returns the taker fee value and if it is a percentage or real number
func (d *Definitions) GetTaker(a asset.Item) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commissions[a]
	if !ok {
		return 0, false, errRateNotFound
	}
	fee, isSetAmount = c.GetTaker()
	return
}

// CalculateDeposit returns calculated fee from the amount
func (d *Definitions) CalculateDeposit(c currency.Code, a asset.Item, amount float64) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, err
	}
	return t.calculate(t.Deposit, amount)
}

// GetDeposit returns the deposit fee associated with the currency
func (d *Definitions) GetDeposit(c currency.Code, a asset.Item) (fee float64, isPercentage bool, err error) {
	if d == nil {
		return 0, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, false, err
	}
	rVal, _ := t.Deposit.Float64()
	return rVal, t.Percentage, nil
}

// CalculateDeposit returns calculated fee from the amount
func (d *Definitions) CalculateWithdrawal(c currency.Code, a asset.Item, amount float64) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, err
	}
	return t.calculate(t.Withdrawal, amount)
}

// GetWithdrawal returns the withdrawal fee associated with the currency
func (d *Definitions) GetWithdrawal(c currency.Code, a asset.Item) (fee float64, isPercentage bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, false, err
	}
	rVal, _ := t.Withdrawal.Float64()
	return rVal, t.Percentage, nil
}

// get returns the fee structure by the currency and its asset type
func (d *Definitions) get(c currency.Code, a asset.Item) (*transfer, error) {
	if c.String() == "" {
		return nil, errCurrencyIsEmpty
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("%s, %w", a, asset.ErrNotSupported)
	}
	s, ok := d.transfers[a][c.Item]
	if !ok {
		return nil, errTransferFeeNotFound
	}
	return s, nil
}

// GetAllFees returns a snapshot of the full fee definitions, super cool.
func (d *Definitions) GetAllFees() (Options, error) {
	if d == nil {
		return Options{}, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	op := Options{
		Commission:      make(map[asset.Item]Commission),
		Transfer:        make(map[asset.Item]map[currency.Code]Transfer),
		BankingTransfer: make(map[BankTransaction]map[currency.Code]Transfer),
	}

	for a, value := range d.commissions {
		op.Commission[a] = value.convert()
	}

	for as, m1 := range d.transfers {
		temp := make(map[currency.Code]Transfer)
		for c, val := range m1 {
			temp[currency.Code{Item: c, UpperCase: true}] = val.convert()
		}
		op.Transfer[as] = temp
	}

	for bankingID, m1 := range d.bankingTransfers {
		temp := make(map[currency.Code]Transfer)
		for c, val := range m1 {
			temp[currency.Code{Item: c, UpperCase: true}] = val.convert()
		}
		op.BankingTransfer[bankingID] = temp
	}
	return op, nil
}

// GetCommissionFee returns a pointer of the current commission rate for the
// asset type.
func (d *Definitions) GetCommissionFee(a asset.Item) (*CommissionInternal, error) {
	if d == nil {
		return nil, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commissions[a]
	if !ok {
		return nil, errRateNotFound
	}
	return c, nil
}

// SetCommissionFee sets new global fees and forces custom control for that
// asset
func (d *Definitions) SetCommissionFee(a asset.Item, maker, taker float64, setAmount bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if maker < 0 { // TODO: rebates will be negative so we can deprecate this
		return errMakerInvalid
	}

	if taker < 0 {
		return errTakerInvalid
	}

	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	c, ok := d.commissions[a]
	if !ok {
		return errRateNotFound
	}
	return c.set(maker, taker, setAmount)
}

// GetTransferFee returns a snapshot of the current Commission rate for the
// asset type.
func (d *Definitions) GetTransferFee(c currency.Code, a asset.Item) (Transfer, error) {
	if d == nil {
		return Transfer{}, ErrDefinitionsAreNil
	}

	if c.String() == "" {
		return Transfer{}, errCurrencyIsEmpty
	}

	if !a.IsValid() {
		return Transfer{}, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, ok := d.transfers[a][c.Item]
	if !ok {
		return Transfer{}, errRateNotFound
	}
	return t.convert(), nil
}

// SetTransferFees sets new transfer fees
func (d *Definitions) SetTransferFee(c currency.Code, a asset.Item, withdraw, deposit float64, isPercentage bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if withdraw < 0 {
		return errWithdrawalIsInvalid
	}

	if deposit < 0 {
		return errDepositIsInvalid
	}

	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	if c.String() == "" {
		return errCurrencyIsEmpty
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	t, ok := d.transfers[a][c.Item]
	if !ok {
		return errTransferFeeNotFound
	}

	// These should not change, and a package update might need to occur.
	if t.Percentage != isPercentage {
		return errFeeTypeMismatch
	}

	t.Withdrawal = decimal.NewFromFloat(withdraw)
	t.Deposit = decimal.NewFromFloat(deposit)
	return nil
}

// GetBankTransferFee returns a snapshot of the current bank transfer rate for the
// asset.
func (d *Definitions) GetBankTransferFee(c currency.Code, transType BankTransaction) (Transfer, error) {
	if d == nil {
		return Transfer{}, ErrDefinitionsAreNil
	}

	if c.String() == "" {
		return Transfer{}, errCurrencyIsEmpty
	}

	err := transType.Validate()
	if err != nil {
		return Transfer{}, err
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, ok := d.bankingTransfers[transType][c.Item]
	if !ok {
		return Transfer{}, errRateNotFound
	}
	return t.convert(), nil
}

// SetTransferFees sets new transfer fees
func (d *Definitions) SetBankTransferFee(c currency.Code, transType BankTransaction, withdraw, deposit float64, isPercentage bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if c.String() == "" {
		return errCurrencyIsEmpty
	}

	err := transType.Validate()
	if err != nil {
		return err
	}

	if withdraw < 0 {
		return errWithdrawalIsInvalid
	}

	if deposit < 0 {
		return errDepositIsInvalid
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	tFee, ok := d.bankingTransfers[transType][c.Item]
	if !ok {
		return errBankTransferFeeNotFound
	}

	if tFee.Percentage != isPercentage {
		return errFeeTypeMismatch
	}

	tFee.Withdrawal = decimal.NewFromFloat(withdraw)
	tFee.Deposit = decimal.NewFromFloat(deposit)
	return nil
}

var errNoTransferFees = errors.New("missing transfer fees to load")

// LoadTransferFees allows the loading of current transfer fees for
// cryptocurrency deposit and withdrawals
func (d *Definitions) LoadTransferFees(fees map[asset.Item]map[currency.Code]Transfer) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if len(fees) == 0 {
		return errNoTransferFees
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	for assetItem, m1 := range fees {
		for code, incomingVal := range m1 {
			trAssets, ok := d.transfers[assetItem]
			if !ok {
				trAssets = make(map[*currency.Item]*transfer)
				d.transfers[assetItem] = trAssets
			}
			trVal, ok := trAssets[code.Item]
			if !ok {
				trAssets[code.Item] = incomingVal.convert()
				continue
			}
			err := trVal.update(incomingVal)
			if err != nil {
				return fmt.Errorf("loading crypto fees error: %w", err)
			}
		}
	}
	return nil
}

// LoadBankTransferFees allows the loading of current banking transfer fees for
// banking deposit and withdrawals
func (d *Definitions) LoadBankTransferFees(fees map[BankTransaction]map[currency.Code]Transfer) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if len(fees) == 0 {
		return errNoTransferFees
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	for bankType, m1 := range fees {
		for code, incomingVal := range m1 {
			trAssets, ok := d.bankingTransfers[bankType]
			if !ok {
				trAssets = make(map[*currency.Item]*transfer)
				d.bankingTransfers[bankType] = trAssets
			}
			trVal, ok := trAssets[code.Item]
			if !ok {
				trAssets[code.Item] = incomingVal.convert()
				continue
			}
			err := trVal.update(incomingVal)
			if err != nil {
				return fmt.Errorf("loading banking fees error: %w", err)
			}
		}
	}
	return nil
}
