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

// InternationalBankTransactionType custom type for calculating fees based on fiat transaction types
type InternationalBankTransactionType uint8

var (
	manager Manager

	errFeeDefinitionsAlreadyLoaded = errors.New("fee definitions are already loaded for exchange")
	errFeeDefinitionsAreNil        = errors.New("fee definitions are nil")
	errExchangeNameIsEmpty         = errors.New("exchange name is empty")
	errCurrencyIsEmpty             = errors.New("currency is empty")

	// errNoRealValue  = errors.New("no real value")
	// errNoRatioValue = errors.New("no ratio value")

	errTransferFeeNotFound = errors.New("transfer fee not found")

	errPriceIsZero          = errors.New("price is zero")
	errAmountIsZero         = errors.New("amount is zero")
	errMakerAndTakerInvalid = errors.New("maker and taker fee invalid")

	errNotRatio = errors.New("loaded values are not ratios")

	identity = decimal.NewFromInt(1)
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
	r := &Definitions{}
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
		return errFeeDefinitionsAreNil
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
	// transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be real values.
	transfer map[*currency.Item]map[asset.Item]*Transfer
	mtx      sync.RWMutex
}

type Global struct {
	// ratio defines if the value is a ratio e.g. 0.8% == 0.008 or a real value
	// like 15 dollars.
	ratio bool
	// maker defines the fee when you provide liqudity for the orderbooks
	maker decimal.Decimal
	// taker defines the fee when you remove liqudity for the orderbooks
	taker decimal.Decimal
}

// LoadDynamic loads the current dynamic account fee structure for maker and
// taker values.
func (d *Definitions) LoadDynamic(maker, taker float64) error {
	if d == nil {
		return errFeeDefinitionsAreNil
	}

	if maker <= 0 && taker <= 0 {
		return errMakerAndTakerInvalid
	}
	d.mtx.Lock()
	d.online.maker = decimal.NewFromFloat(maker)
	d.online.taker = decimal.NewFromFloat(taker)
	d.mtx.Unlock()
	return nil
}

// LoadStatic loads custom long term fee structure in the event there are no
// dynamic loading options.
func (d *Definitions) LoadStatic(o Options) error {
	if d == nil {
		return errFeeDefinitionsAreNil
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.online.taker = decimal.NewFromFloat(o.Taker)
	d.online.maker = decimal.NewFromFloat(o.Maker)
	d.online.ratio = o.Ratio

	d.offline.taker = decimal.NewFromFloat(o.Taker)
	d.offline.maker = decimal.NewFromFloat(o.Maker)
	d.offline.ratio = o.Ratio

	for code, val := range o.Transfer {
		for as, trans := range val {
			m1, ok := d.transfer[code.Item]
			if !ok {
				m1 = make(map[asset.Item]*Transfer)
				d.transfer[code.Item] = m1
			}
			m1[as] = &trans
		}
	}
	return nil
}

// GetMakerTotal returns the fee amount derived from the price, amount and fee
// ratio.
func (d *Definitions) GetMakerTotal(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.online.maker, price, amount)
}

// GetMakerTotalOffline returns the fee amount derived from the price, amount
// and fee ratio using the worst case-scenario trading fee.
func (d *Definitions) GetMakerTotalOffline(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.offline.maker, price, amount)
}

// GetMaker returns the maker fee value and if it is a ratio or real number
func (d *Definitions) GetMaker() (fee float64, ratio bool) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	rVal, _ := d.online.maker.Float64()
	return rVal, d.online.ratio
}

// GetTakerTotal returns the fee amount derived from the price, amount and fee
// ratio.
func (d *Definitions) GetTakerTotal(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.online.taker, price, amount)
}

// GetMakerTotalOffline returns the fee amount derived from the price, amount
// and fee ratio using the worst case-scenario trading fee.
func (d *Definitions) GetTakerTotalOffline(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.deriveValue(d.offline.taker, price, amount)
}

// GetTaker returns the taker fee value and if it is a ratio or real number
func (d *Definitions) GetTaker() (fee float64, ratio bool) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	rVal, _ := d.online.taker.Float64()
	return rVal, d.online.ratio
}

// deriveValue returns the fee value from the price, amount and fee ratio.
func (d *Definitions) deriveValue(fee decimal.Decimal, price, amount float64) (float64, error) {
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}
	if !d.online.ratio {
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
func (d *Definitions) GetDeposit(c currency.Code, a asset.Item) (fee float64, ratio bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, false, err
	}
	rVal, _ := t.Deposit.Float64()
	return rVal, t.Ratio, nil
}

// GetWithdrawal returns the withdrawal fee associated with the currency
func (d *Definitions) GetWithdrawal(c currency.Code, a asset.Item) (fee float64, ratio bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, false, err
	}
	rVal, _ := t.Withdrawal.Float64()
	return rVal, t.Ratio, nil
}

// get returns the fee structure by the currency and its asset type
func (d *Definitions) get(c currency.Code, a asset.Item) (*Transfer, error) {
	if c.String() == "" {
		return nil, errCurrencyIsEmpty
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("%s, %w", a, asset.ErrNotSupported)
	}
	s, ok := d.transfer[c.Item][a]
	if !ok {
		return nil, errTransferFeeNotFound
	}
	return s, nil
}

// Transfer defines usually static real number values.
type Transfer struct {
	Ratio      bool // Toggle if ratio is present
	Deposit    decimal.Decimal
	Withdrawal decimal.Decimal
}

// Builder is the type which holds all parameters required to calculate a fee
// for an exchange
type Builder struct {
	Type Item
	// Used for calculating crypto trading fees, deposits & withdrawals
	Pair    currency.Pair
	IsMaker bool
	// Fiat currency used for bank deposits & withdrawals
	FiatCurrency        currency.Code
	BankTransactionType InternationalBankTransactionType
	// Used to multiply for fee calculations
	PurchasePrice float64
	Amount        float64
}

// Item defines a different fee type
type Item uint8

// Options defines fee options either, will be interchangeable for static and
// dynamic loading.
type Options struct {
	// Ratio defines if the fee is a ratio or fixed amount
	Ratio bool
	// Maker defines the fee when you provide liqudity for the orderbooks
	Maker float64
	// Taker defines the fee when you remove liqudity for the orderbooks
	Taker float64
	// Transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be fixed real values.
	Transfer map[currency.Code]map[asset.Item]Transfer
}
