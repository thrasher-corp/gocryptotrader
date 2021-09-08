package fee

import (
	"errors"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	// Bank defines a domicile bank transfer fee
	Bank Item = iota
	// InternationalBankDeposit defines an international bank deposit fee
	InternationalBankDeposit
	// InternationalBankDeposit defines an international bank withdrawal fee
	InternationalBankWithdrawal
	// Trade defines an exchange trading fee
	Trade
	// Deposit defines an exchange deposit fee
	Deposit
	// Withdrawal defines an exchange withdrawal fee
	Withdrawal
	// OfflineTrade defines a worst case scenario scenario fee
	OfflineTrade
)

// InternationalBankTransaction custom type for calculating fees based on fiat
// transaction types
type InternationalBankTransaction uint8

// TODO LIST:
// if f < 0 {
// f = 0
// }
// Find out why this was the case in any situation? neg val?

var (
	manager Manager

	errFeeDefinitionsAlreadyLoaded = errors.New("fee definitions are already loaded for exchange")
	// ErrDefinitionsAreNil defines if the exchange specific fee definitions
	// have bot been loaded or set up.
	ErrDefinitionsAreNil   = errors.New("fee definitions are nil")
	errExchangeNameIsEmpty = errors.New("exchange name is empty")
	errCurrencyIsEmpty     = errors.New("currency is empty")

	// errNoRealValue  = errors.New("no real value")
	// errNoRatioValue = errors.New("no ratio value")

	errTransferFeeNotFound = errors.New("transfer fee not found")

	errPriceIsZero          = errors.New("price is zero")
	errAmountIsZero         = errors.New("amount is zero")
	errMakerAndTakerInvalid = errors.New("maker and taker fee invalid")

	errNotRatio = errors.New("loaded values are not ratios")
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
		transfer:        make(map[asset.Item]map[*currency.Item]*transfer),
		bankingTransfer: make(map[InternationalBankTransaction]map[*currency.Item]*transfer),
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
	// online live global fees
	online Global
	// offline fees for global state
	offline Global
	// custom allows for the custom setting of the global fee state, this
	// stops dynamic updating.
	custom bool
	// transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be real values.
	transfer map[asset.Item]map[*currency.Item]*transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	bankingTransfer map[InternationalBankTransaction]map[*currency.Item]*transfer
	mtx             sync.RWMutex
}

type Global struct {
	// SetAmount defines if the value is a set amount (15 USD) rather than a
	// ratio e.g. 0.8% == 0.008 or a real value.
	SetAmount bool
	// maker defines the fee when you provide liqudity for the orderbooks
	Maker decimal.Decimal
	// taker defines the fee when you remove liqudity for the orderbooks
	Taker decimal.Decimal
}

// LoadDynamic loads the current dynamic account fee structure for maker and
// taker values.
func (d *Definitions) LoadDynamic(maker, taker float64) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if maker <= 0 && taker <= 0 {
		return errMakerAndTakerInvalid
	}
	d.mtx.Lock()
	d.online.Maker = decimal.NewFromFloat(maker)
	d.online.Taker = decimal.NewFromFloat(taker)
	d.mtx.Unlock()
	return nil
}

// LoadStatic loads custom long term fee structure in the event there are no
// dynamic loading options.
func (d *Definitions) LoadStatic(o Options) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.online.Taker = decimal.NewFromFloat(o.Taker)
	d.online.Maker = decimal.NewFromFloat(o.Maker)
	d.online.SetAmount = o.IsSetAmount

	d.offline.Taker = decimal.NewFromFloat(o.Taker)
	d.offline.Maker = decimal.NewFromFloat(o.Maker)
	d.offline.SetAmount = o.IsSetAmount

	for as, m1 := range o.Transfer {
		for code, value := range m1 {
			m1, ok := d.transfer[as]
			if !ok {
				m1 = make(map[*currency.Item]*transfer)
				d.transfer[as] = m1
			}
			m1[code.Item] = &transfer{
				Percentage: value.IsPercentage,
				Deposit:    decimal.NewFromFloat(value.Deposit),
				Withdrawal: decimal.NewFromFloat(value.Withdrawal),
			}
		}
	}

	for transactionType, m1 := range o.BankingTransfer {
		for code, value := range m1 {
			m1, ok := d.bankingTransfer[transactionType]
			if !ok {
				m1 = make(map[*currency.Item]*transfer)
				d.bankingTransfer[transactionType] = m1
			}
			m1[code.Item] = &transfer{
				Percentage: value.IsPercentage,
				Deposit:    decimal.NewFromFloat(value.Deposit),
				Withdrawal: decimal.NewFromFloat(value.Withdrawal),
			}
		}
	}
	return nil
}

// GetMakerTotal returns the fee amount derived from the price, amount and fee
// ratio.
func (d *Definitions) GetMakerTotal(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.online.Maker, price, amount)
}

// GetMakerTotalOffline returns the fee amount derived from the price, amount
// and fee ratio using the worst case-scenario trading fee.
func (d *Definitions) GetMakerTotalOffline(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.offline.Maker, price, amount)
}

// GetMaker returns the maker fee value and if it is a ratio or real number
func (d *Definitions) GetMaker() (fee float64, isSetAmount bool) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	rVal, _ := d.online.Maker.Float64()
	return rVal, d.online.SetAmount
}

// GetTakerTotal returns the fee amount derived from the price, amount and fee
// ratio.
func (d *Definitions) GetTakerTotal(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.online.Taker, price, amount)
}

// GetMakerTotalOffline returns the fee amount derived from the price, amount
// and fee ratio using the worst case-scenario trading fee.
func (d *Definitions) GetTakerTotalOffline(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.offline.Taker, price, amount)
}

// GetTaker returns the taker fee value and if it is a ratio or real number
func (d *Definitions) GetTaker() (fee float64, isSetAmount bool) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	rVal, _ := d.online.Taker.Float64()
	return rVal, d.online.SetAmount
}

// deriveValue returns the fee value from the price, amount and fee ratio.
func (d *Definitions) deriveValue(fee decimal.Decimal, price, amount float64) (float64, error) {
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}
	if !d.online.SetAmount {
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
	s, ok := d.transfer[a][c.Item]
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

	maker, _ := d.online.Maker.Float64()
	taker, _ := d.online.Taker.Float64()

	offlineMaker, _ := d.offline.Maker.Float64()
	offlineTaker, _ := d.offline.Taker.Float64()

	wcs := maker == offlineMaker && taker == offlineTaker

	op := Options{
		IsSetAmount:       d.online.SetAmount,
		Maker:             maker,
		Taker:             taker,
		WorstCaseScenario: wcs,
		Transfer:          make(map[asset.Item]map[currency.Code]Transfer),
	}

	for as, m1 := range d.transfer {
		temp := make(map[currency.Code]Transfer)
		for c, val := range m1 {
			deposit, _ := val.Deposit.Float64()
			withdraw, _ := val.Withdrawal.Float64()
			temp[currency.Code{Item: c, UpperCase: true}] = Transfer{
				Deposit:      deposit,
				Withdrawal:   withdraw,
				IsPercentage: val.Percentage,
			}
		}
		op.Transfer[as] = temp
	}
	return op, nil
}

// GetOfflineFees returns a snapshot of the offline fees
func (d *Definitions) GetOfflineFees() (Global, error) {
	if d == nil {
		return Global{}, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.offline, nil
}

var errFeeTypeMismatch = errors.New("fee type mismatch")

// SetGlobalFees sets new global fees and forces custom control
func (d *Definitions) SetGlobalFees(maker, taker float64, setAmount bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.online.SetAmount != setAmount {
		return errFeeTypeMismatch
	}

	d.online.Maker = decimal.NewFromFloat(maker)
	d.online.Taker = decimal.NewFromFloat(taker)
	d.custom = true

	return nil
}

// SetTransferFees sets new transfer fees
func (d *Definitions) SetTransferFees(c currency.Code, a asset.Item, withdraw, deposit float64, isPercentage bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	t, ok := d.transfer[a][c.Item]
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

var errSameBoolean = errors.New("same boolean value")

// SetCustom sets if the fees are in a custom state and can yield control from
// the fee manager.
func (d *Definitions) SetCustom(on bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.custom == on {
		return errSameBoolean
	}
	d.custom = on
	return nil
}

// Transfer defines usually static real number values. But has the option of
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

// // Builder is the type which holds all parameters required to calculate a fee
// // for an exchange
// type Builder struct {
// 	Type Item
// 	// Used for calculating crypto trading fees, deposits & withdrawals
// 	Pair    currency.Pair
// 	IsMaker bool
// 	// Fiat currency used for bank deposits & withdrawals
// 	FiatCurrency        currency.Code
// 	BankTransactionType InternationalBankTransactionType
// 	// Used to multiply for fee calculations
// 	PurchasePrice float64
// 	Amount        float64
// }

// Item defines a different fee type
type Item uint8

// Options defines fee loading options and is also used as a state snapshot, in
// GetAllFees method.
type Options struct {
	// IsSetAmount defines if the fee is a fixed amount rather than a percentage.
	IsSetAmount bool
	// Maker defines the fee when you provide liqudity for the orderbooks
	Maker float64
	// Taker defines the fee when you remove liqudity for the orderbooks
	Taker float64
	// Transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be fixed real values.
	Transfer map[asset.Item]map[currency.Code]Transfer
	// BankingTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	BankingTransfer map[InternationalBankTransaction]map[currency.Code]Transfer
	// WorstCaseScenario defines the worst case scenario of fees in the event
	// that either there is no authenticated connection.
	WorstCaseScenario bool
}
