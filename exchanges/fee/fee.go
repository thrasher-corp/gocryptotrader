package fee

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
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
	errCommissionRateNotFound  = errors.New("commission rate not found")
	errTakerInvalid            = errors.New("taker is invalid")
	errMakerInvalid            = errors.New("maker is invalid")
	errMakerBiggerThanTaker    = errors.New("maker cannot be bigger than taker")

	// OmitPair is a an empty pair designation for unused pair variables
	OmitPair = currency.Pair{}
)

// NewFeeDefinitions generates a new fee struct for exchange usage
func NewFeeDefinitions() *Definitions {
	return &Definitions{
		globalCommissions: make(map[asset.Item]*CommissionInternal),
		pairCommissions:   make(map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal),
		transfers:         make(map[asset.Item]map[*currency.Item]*transfer),
		bankingTransfers:  make(map[bank.Transfer]map[*currency.Item]*transfer),
	}
}

// Definitions defines the full fee definitions for different currencies
// TODO: Eventually upgrade with key manager for different fees associated
// with different accounts/keys.
type Definitions struct {
	// Commission is the holder for the up to date comission rates for the assets.
	globalCommissions map[asset.Item]*CommissionInternal
	// pairCommissions is the holder for the up to date commissions rates for
	// the specific trading pairs.
	pairCommissions map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal
	// transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be real values.
	transfers map[asset.Item]map[*currency.Item]*transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	bankingTransfers map[bank.Transfer]map[*currency.Item]*transfer
	mtx              sync.RWMutex
}

// LoadDynamic loads the current dynamic account fee structure for maker and
// taker values. The pair is an optional paramater if ommited will designate
// global/exchange maker, taker fees irrespective of individual trading
// operations.
func (d *Definitions) LoadDynamic(maker, taker float64, a asset.Item, pair currency.Pair) error {
	if d == nil {
		return ErrDefinitionsAreNil
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
	var c *CommissionInternal
	if !pair.IsEmpty() {
		// NOTE: These will create maps, as we can initially start out as global
		// commission rates and update ad-hoc.
		m1, ok := d.pairCommissions[a]
		if !ok {
			m1 = make(map[*currency.Item]map[*currency.Item]*CommissionInternal)
			d.pairCommissions[a] = m1
		}
		m2, ok := m1[pair.Base.Item]
		if !ok {
			m2 = make(map[*currency.Item]*CommissionInternal)
			m1[pair.Base.Item] = m2
		}
		c, ok = m2[pair.Quote.Item]
		if !ok {
			c = new(CommissionInternal)
			m2[pair.Quote.Item] = c
		}
	} else {
		var ok bool
		c, ok = d.globalCommissions[a]
		if !ok {
			return fmt.Errorf("global %w", errCommissionRateNotFound)
		}
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
	// Loads global commission rates based on asset item
	for a, value := range o.GlobalCommissions {
		d.globalCommissions[a] = value.convert()
	}

	// Loads pair specific commission rates
	for a, incoming := range o.PairCommissions {
		for pair, value := range incoming {
			m1, ok := d.pairCommissions[a]
			if !ok {
				m1 = make(map[*currency.Item]map[*currency.Item]*CommissionInternal)
				d.pairCommissions[a] = m1
			}
			m2, ok := m1[pair.Base.Item]
			if !ok {
				m2 = make(map[*currency.Item]*CommissionInternal)
			}
			m2[pair.Quote.Item] = value.convert()
		}
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

// GetCommissionFee returns a pointer of the current commission rate for the
// asset type.
func (d *Definitions) GetCommissionFee(a asset.Item, pair currency.Pair) (*CommissionInternal, error) {
	if d == nil {
		return nil, ErrDefinitionsAreNil
	}
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getCommission(a, pair)
}

// getCommission returns the internal commission rate based on provided params
func (d *Definitions) getCommission(a asset.Item, pair currency.Pair) (*CommissionInternal, error) {
	if len(d.pairCommissions) != 0 && !pair.IsEmpty() {
		m1, ok := d.pairCommissions[a]
		if !ok {
			return nil, fmt.Errorf("pair %w", errCommissionRateNotFound)
		}

		m2, ok := m1[pair.Base.Item]
		if !ok {
			return nil, fmt.Errorf("pair %w", errCommissionRateNotFound)
		}

		c, ok := m2[pair.Quote.Item]
		if !ok {
			return nil, fmt.Errorf("pair %w", errCommissionRateNotFound)
		}
		return c, nil
	} else {
		c, ok := d.globalCommissions[a]
		if !ok {
			return nil, fmt.Errorf("global %w", errCommissionRateNotFound)
		}
		return c, nil
	}
}

// CalculateMaker returns the fee amount derived from the price, amount and fee
// percentage.
func (d *Definitions) CalculateMaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateMaker(price, amount)
}

// CalculateWorstCaseMaker returns the fee amount derived from the price, amount
// and fee percentage using the worst-case scenario trading fee.
func (d *Definitions) CalculateWorstCaseMaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateWorstCaseMaker(price, amount)
}

// GetMaker returns the maker fee value and if it is a percentage or whole
// number
func (d *Definitions) GetMaker(a asset.Item, pair currency.Pair) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, false, err
	}
	fee, isSetAmount = c.GetMaker()
	return
}

// CalculateTaker returns the fee amount derived from the price, amount and fee
// percentage.
func (d *Definitions) CalculateTaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateTaker(price, amount)
}

// CalculateWorstCaseTaker returns the fee amount derived from the price, amount
// and fee percentage using the worst-case scenario trading fee.
func (d *Definitions) CalculateWorstCaseTaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateWorstCaseTaker(price, amount)
}

// GetTaker returns the taker fee value and if it is a percentage or real number
func (d *Definitions) GetTaker(a asset.Item, pair currency.Pair) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, false, err
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
func (d *Definitions) GetDeposit(c currency.Code, a asset.Item) (fee Value, isPercentage bool, err error) {
	if d == nil {
		return nil, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return nil, false, err
	}
	return t.Deposit, t.Percentage, nil
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
func (d *Definitions) GetWithdrawal(c currency.Code, a asset.Item) (fee Value, isPercentage bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return nil, false, err
	}
	return t.Withdrawal, t.Percentage, nil
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
		GlobalCommissions: make(map[asset.Item]Commission),
		PairCommissions:   make(map[asset.Item]map[currency.Pair]Commission),
		Transfer:          make(map[asset.Item]map[currency.Code]Transfer),
		BankingTransfer:   make(map[bank.Transfer]map[currency.Code]Transfer),
	}

	for a, value := range d.globalCommissions {
		op.GlobalCommissions[a] = value.convert()
	}

	for a, mInternal := range d.pairCommissions {
		for c1, mInternal2 := range mInternal {
			for c2, value := range mInternal2 {
				mOutgoing, ok := op.PairCommissions[a]
				if !ok {
					mOutgoing = make(map[currency.Pair]Commission)
					op.PairCommissions[a] = mOutgoing
				}
				p := currency.NewPair(currency.Code{Item: c1}, currency.Code{Item: c2})
				mOutgoing[p] = value.convert()
			}
		}
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

// SetCommissionFee sets new global fees and forces custom control for that
// asset
func (d *Definitions) SetCommissionFee(a asset.Item, pair currency.Pair, maker, taker float64, setAmount bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if taker < 0 {
		return errTakerInvalid
	}

	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return err
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
// TODO: need min and max settings might deprecate due to complexity of value
// types
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

	t.Withdrawal = Convert(withdraw) // TODO: need min and max settings
	t.Deposit = Convert(deposit)     // TODO: need min and max settings
	return nil
}

// GetBankTransferFee returns a snapshot of the current bank transfer rate for the
// asset.
func (d *Definitions) GetBankTransferFee(c currency.Code, transType bank.Transfer) (Transfer, error) {
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

// SetBankTransferFee sets new bank transfer fees
// TODO: need min and max settings might deprecate due to complexity of value
// types
func (d *Definitions) SetBankTransferFee(c currency.Code, transType bank.Transfer, withdraw, deposit float64, isPercentage bool) error {
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

	tFee.Withdrawal = Convert(withdraw) // TODO: need min and max settings
	tFee.Deposit = Convert(deposit)     // TODO: need min and max settings
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
func (d *Definitions) LoadBankTransferFees(fees map[bank.Transfer]map[currency.Code]Transfer) error {
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
