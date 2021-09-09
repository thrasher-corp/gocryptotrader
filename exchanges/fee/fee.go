package fee

import (
	"errors"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// BankTransaction defines the different fee types associated with bank
// transactions to and from an exchange.
type BankTransaction uint8

// TODO LIST:
// if f < 0 {
// f = 0
// }
// Find out why this was the case in any situation? neg val?

var (
	manager Manager

	// ErrDefinitionsAreNil defines if the exchange specific fee definitions
	// have bot been loaded or set up.
	ErrDefinitionsAreNil = errors.New("fee definitions are nil")

	errFeeDefinitionsAlreadyLoaded = errors.New("fee definitions are already loaded for exchange")
	errExchangeNameIsEmpty         = errors.New("exchange name is empty")
	errCurrencyIsEmpty             = errors.New("currency is empty")
	errTransferFeeNotFound         = errors.New("transfer fee not found")
	errBankTransferFeeNotFound     = errors.New("bank transfer fee not found")
	errPriceIsZero                 = errors.New("price is zero")
	errAmountIsZero                = errors.New("amount is zero")
	errFeeTypeMismatch             = errors.New("fee type mismatch")
	errRateNotFound                = errors.New("rate not found")
	errNotRatio                    = errors.New("loaded values are not ratios")
	errCommisionRateNotFound       = errors.New("commision rate not found")
	errTakerInvalid                = errors.New("taker is invalid")
	errMakerInvalid                = errors.New("maker is invalid")
	errDepositIsInvalid            = errors.New("deposit is invalid")
	errWithdrawalIsInvalid         = errors.New("withdrawal is invalid")
	errTakerBiggerThanMaker        = errors.New("taker cannot be bigger than maker")
)

// GetManager returns the package management struct
func GetManager() *Manager {
	return &manager
}

// RegisterExchangeState generates a new fee struct and registers it with the
// manager
func RegisterFeeDefinitions(exch string) (*Definitions, error) {
	if exch == "" {
		return nil, errExchangeNameIsEmpty
	}
	r := &Definitions{
		commisions:       make(map[asset.Item]*commision),
		transfers:        make(map[asset.Item]map[*currency.Item]*transfer),
		bankingTransfers: make(map[BankTransaction]map[*currency.Item]*transfer),
	}
	return r, manager.Register(exch, r)
}

// Manager defines operating fee structures across all enabled exchanges
type Manager struct {
	m   map[string]*Definitions
	mtx sync.RWMutex
}

// Register registers new exchange fee definitions
func (m *Manager) Register(exch string, s *Definitions) error {
	if exch == "" {
		return errExchangeNameIsEmpty
	}
	if s == nil {
		return ErrDefinitionsAreNil
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	_, ok := m.m[exch]
	if ok {
		return fmt.Errorf("%w for %s", errFeeDefinitionsAlreadyLoaded, exch)
	}
	if m.m == nil {
		m.m = make(map[string]*Definitions)
	}
	m.m[exch] = s
	return nil
}

// Definitions defines the full fee definitions for different currencies
// TODO: Eventually upgrade with key manager for different fees associated
// with different accounts/keys.
type Definitions struct {
	// commision is the holder for the up to date comission rates for the assets.
	commisions map[asset.Item]*commision
	// transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be real values.
	transfers map[asset.Item]map[*currency.Item]*transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	bankingTransfers map[BankTransaction]map[*currency.Item]*transfer
	mtx              sync.RWMutex
}

// commision defines a trading fee structure for internal tracking
type commision struct {
	// SetAmount defines if the value is a set amount (15 USD) rather than a
	// percentage e.g. 0.8% == 0.008.
	SetAmount bool
	// Maker defines the fee when you provide liqudity for the orderbooks
	Maker decimal.Decimal
	// Taker defines the fee when you remove liqudity for the orderbooks
	Taker decimal.Decimal
	// WorstCaseMaker defines the worst case fee when you provide liqudity for
	// the orderbooks
	WorstCaseMaker decimal.Decimal
	// WorstCaseTaker defines the worst case fee when you remove liqudity for
	//the orderbooks
	WorstCaseTaker decimal.Decimal
}

// convert returns a friendly package exportedable type
func (c commision) convert() Commision {
	maker, _ := c.Maker.Float64()
	taker, _ := c.Taker.Float64()
	worstCaseMaker, _ := c.WorstCaseMaker.Float64()
	worstCaseTaker, _ := c.WorstCaseTaker.Float64()
	return Commision{
		IsSetAmount:    c.SetAmount,
		Maker:          maker,
		Taker:          taker,
		WorstCaseMaker: worstCaseMaker,
		WorstCaseTaker: worstCaseTaker,
	}
}

// Commision defines a trading fee structure
type Commision struct {
	// IsSetAmount defines if the value is a set amount (15 USD) rather than a
	// percentage e.g. 0.8% == 0.008.
	IsSetAmount bool
	// Maker defines the fee when you provide liqudity for the orderbooks
	Maker float64
	// Taker defines the fee when you remove liqudity for the orderbooks
	Taker float64
	// WorstCaseMaker defines the worst case fee when you provide liqudity for
	// the orderbooks
	WorstCaseMaker float64
	// WorstCaseTaker defines the worst case fee when you remove liqudity for
	//the orderbooks
	WorstCaseTaker float64
}

// convert returns a internal commission rate type
func (c Commision) convert() *commision {
	return &commision{
		SetAmount:      c.IsSetAmount,
		Maker:          decimal.NewFromFloat(c.Maker),
		Taker:          decimal.NewFromFloat(c.Taker),
		WorstCaseMaker: decimal.NewFromFloat(c.WorstCaseMaker),
		WorstCaseTaker: decimal.NewFromFloat(c.WorstCaseTaker),
	}
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
	if !a.IsValid() {
		return fmt.Errorf("%s: %w", a, asset.ErrNotSupported)
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	c, ok := d.commisions[a]
	if !ok {
		return errCommisionRateNotFound
	}
	c.Maker = decimal.NewFromFloat(maker)
	c.Taker = decimal.NewFromFloat(taker)
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
	// Loads standard commision rates based on asset item
	for a, value := range o.Commission {
		var wcm = decimal.NewFromFloat(value.WorstCaseMaker)
		if wcm.IsZero() {
			decimal.NewFromFloat(value.Maker)
		}
		var wct = decimal.NewFromFloat(value.WorstCaseTaker)
		if wct.IsZero() {
			decimal.NewFromFloat(value.Taker)
		}
		d.commisions[a] = value.convert()
	}

	// Loads exchange withdrawal and deposit fees
	for as, m1 := range o.Transfer {
		for code, value := range m1 {
			m1, ok := d.transfers[as]
			if !ok {
				m1 = make(map[*currency.Item]*transfer)
				d.transfers[as] = m1
			}
			m1[code.Item] = value.convert()
		}
	}

	// Loads international banking withdrawal and deposit fees
	for transactionType, m1 := range o.BankingTransfer {
		for code, value := range m1 {
			m1, ok := d.bankingTransfers[transactionType]
			if !ok {
				m1 = make(map[*currency.Item]*transfer)
				d.bankingTransfers[transactionType] = m1
			}
			m1[code.Item] = value.convert()
		}
	}
	return nil
}

// GetMakerTotal returns the fee amount derived from the price, amount and fee
// ratio.
func (d *Definitions) GetMakerTotal(price, amount float64, a asset.Item) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commisions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return d.deriveValue(c.Maker, c.SetAmount, price, amount)
}

// GetMakerTotalOffline returns the fee amount derived from the price, amount
// and fee ratio using the worst case-scenario trading fee.
func (d *Definitions) GetMakerTotalOffline(price, amount float64, a asset.Item) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commisions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return d.deriveValue(c.WorstCaseMaker, c.SetAmount, price, amount)
}

// GetMaker returns the maker fee value and if it is a percentage or whole
// number
func (d *Definitions) GetMaker(a asset.Item) (fee float64, isSetAmount bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commisions[a]
	if !ok {
		return 0, false, errRateNotFound
	}
	rVal, _ := c.Maker.Float64()
	return rVal, c.SetAmount, nil
}

// GetTakerTotal returns the fee amount derived from the price, amount and fee
// ratio.
func (d *Definitions) GetTakerTotal(price, amount float64, a asset.Item) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commisions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return d.deriveValue(c.Taker, c.SetAmount, price, amount)
}

// GetMakerTotalOffline returns the fee amount derived from the price, amount
// and fee ratio using the worst case-scenario trading fee.
func (d *Definitions) GetTakerTotalOffline(price, amount float64, a asset.Item) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commisions[a]
	if !ok {
		return 0, errRateNotFound
	}
	return d.deriveValue(c.WorstCaseTaker, c.SetAmount, price, amount)
}

// GetTaker returns the taker fee value and if it is a ratio or real number
func (d *Definitions) GetTaker(a asset.Item) (fee float64, isSetAmount bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commisions[a]
	if !ok {
		return 0, false, errRateNotFound
	}
	rVal, _ := c.Taker.Float64()
	return rVal, c.SetAmount, nil
}

// deriveValue returns the fee value from the price, amount and fee ratio.
func (d *Definitions) deriveValue(fee decimal.Decimal, setAmount bool, price, amount float64) (float64, error) {
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}
	if !setAmount {
		return 0, errNotRatio
	}
	// let currency = BTC/USD
	// price (Quotation) * amount (Base) * fee (ratio)
	// :. 50000 * 1 * 0.01 = 500 USD
	var val = decimal.NewFromFloat(price).Mul(decimal.NewFromFloat(amount)).Mul(fee)
	rVal, _ := val.Float64()
	return rVal, nil
}

// GetDeposit returns the deposit fee associated with the currency
func (d *Definitions) GetDeposit(c currency.Code, a asset.Item) (fee float64, isPercentage bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, false, err
	}
	rVal, _ := t.Deposit.Float64()
	return rVal, t.Percentage, nil
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
		Commission:      make(map[asset.Item]Commision),
		Transfer:        make(map[asset.Item]map[currency.Code]Transfer),
		BankingTransfer: make(map[BankTransaction]map[currency.Code]Transfer),
	}

	for a, value := range d.commisions {
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

// GetCommisionFee returns a snapshot of the current commision rate for the
// asset type.
func (d *Definitions) GetCommisionFee(a asset.Item) (Commision, error) {
	if d == nil {
		return Commision{}, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, ok := d.commisions[a]
	if !ok {
		return Commision{}, errRateNotFound
	}
	return c.convert(), nil
}

// SetCommissionFee sets new global fees and forces custom control for that
// asset
func (d *Definitions) SetCommissionFee(a asset.Item, maker, taker float64, setAmount bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if maker < 0 {
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
	c, ok := d.commisions[a]
	if !ok {
		return errRateNotFound
	}

	// These should not change, and a package update might need to occur.
	if c.SetAmount != setAmount {
		return errFeeTypeMismatch
	}

	c.Maker = decimal.NewFromFloat(maker)
	c.Taker = decimal.NewFromFloat(taker)
	return nil
}

// GetTransferFee returns a snapshot of the current commision rate for the
// asset type.
func (d *Definitions) GetTransferFee(c currency.Code, a asset.Item) (Transfer, error) {
	if d == nil {
		return Transfer{}, ErrDefinitionsAreNil
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

	d.mtx.Lock()
	defer d.mtx.Unlock()
	t, ok := d.transfers[a][c.Item]
	if !ok {
		return errTransferFeeNotFound
	}

	if t.Percentage != isPercentage {
		return errFeeTypeMismatch
	}

	t.Withdrawal = decimal.NewFromFloat(withdraw)
	t.Deposit = decimal.NewFromFloat(deposit)
	return nil
}

// GetBankTransferFee returns a snapshot of the current bank transfer rate for the
// asset.
func (d *Definitions) GetBankTransferFee(c currency.Code, b BankTransaction) (Transfer, error) {
	if d == nil {
		return Transfer{}, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, ok := d.bankingTransfers[b][c.Item]
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

// Options defines fee loading options and is also used as a state snapshot, in
// GetAllFees method.
type Options struct {
	// Commission defines the maker and taker rates for the indv. asset item.
	Commission map[asset.Item]Commision
	// Transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be fixed real values.
	Transfer map[asset.Item]map[currency.Code]Transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	BankingTransfer map[BankTransaction]map[currency.Code]Transfer
}

// validate checks for invalid values on struct, should be used prior to lock
func (o Options) validate() error {
	for _, v := range o.Commission {
		if v.Maker < 0 {
			return errMakerInvalid
		}
		if v.Taker < 0 {
			return errTakerInvalid
		}

		if v.Taker > v.Maker {
			return errTakerBiggerThanMaker
		}
	}

	for _, m1 := range o.Transfer {
		for _, v := range m1 {
			if v.Deposit < 0 {
				return errDepositIsInvalid
			}
			if v.Withdrawal < 0 {
				return errWithdrawalIsInvalid
			}
		}
	}

	for _, m1 := range o.BankingTransfer {
		for _, v := range m1 {
			if v.Deposit < 0 {
				return errDepositIsInvalid
			}
			if v.Withdrawal < 0 {
				return errWithdrawalIsInvalid
			}
		}
	}
	return nil
}
