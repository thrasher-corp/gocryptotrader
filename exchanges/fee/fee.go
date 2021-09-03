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

	errStatesAlreadyLoaded = errors.New("states already loaded for exchange")
	errStatesIsNil         = errors.New("states is nil")
	errExchangeNameIsEmpty = errors.New("exchange name is empty")
	errCurrencyIsEmpty     = errors.New("currency is empty")

	// errNoRealValue  = errors.New("no real value")
	// errNoRatioValue = errors.New("no ratio value")

	errTransferFeeNotFound = errors.New("transfer fee not found")

	errFeeIsZero    = errors.New("fee is zero")
	errPriceIsZero  = errors.New("price is zero")
	errAmountIsZero = errors.New("amount is zero")

	identity = decimal.NewFromInt(1)
)

type Functionality struct {
}

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

// Manager
type Manager struct {
	m   map[string]*Definitions
	mtx sync.RWMutex
}

// Register registers a new exchange states struct
func (m *Manager) Register(exch string, s *Definitions) error {
	if exch == "" {
		return errExchangeNameIsEmpty
	}
	if s == nil {
		return errStatesIsNil
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	_, ok := m.m[exch]
	if ok {
		return fmt.Errorf("%w %s", errStatesAlreadyLoaded, exch)
	}
	if m.m == nil {
		m.m = make(map[string]*Definitions)
	}
	m.m[exch] = s
	return nil
}

// Definitions defines the full fee definitions for different currencies
// TODO: Upgrade with key manager for different fees.
type Definitions struct {
	// ratio defines if the value is a ratio e.g. 0.8% == 0.008 or a real value
	// for addition.
	ratio bool
	// maker defines the fee when you provide liqudity for the orderbooks
	maker decimal.Decimal
	// taker defines the fee when you remove liqudity for the orderbooks
	taker decimal.Decimal
	// transfer defines a map of currencies with differing withdrawal and
	// deposit situations
	transfer map[*currency.Item]map[asset.Item]*Transfer
	mtx      sync.RWMutex
}

var (
	errTakerFeeIsZero = errors.New("taker fee is zero")
	errMakerFeeIsZero = errors.New("maker fee is zero")
)

// LoadDynamic loads the current dynamic fee structure, if ratio this will
// default to an identity value.
func (d *Definitions) LoadDynamic(takerFee, makerFee float64, ratio bool) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	taker := decimal.NewFromFloat(takerFee)
	if taker.IsZero() {
		if !ratio {
			return errTakerFeeIsZero
		}
		taker = identity
	}
	maker := decimal.NewFromFloat(makerFee)
	if maker.IsZero() {
		if !ratio {
			return errMakerFeeIsZero
		}
		maker = identity
	}
	d.ratio = ratio
	return nil
}

func (d *Definitions) LoadStatic(i interface{}) error {
	return nil
}

// GetMakerValue returns the fee value derived from the price, amount and fee
// ratio
func (d *Definitions) GetMakerValue(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getValue(d.maker, price, amount)
}

// GetMaker returns the maker fee value and if it is a ratio or real number
func (d *Definitions) GetMaker() (fee float64, ratio bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	if d.maker.IsZero() {
		return 0, false, errFeeIsZero
	}
	rVal, _ := d.maker.Float64()
	return rVal, d.ratio, nil
}

// GetTakerValue returns the fee value derived from the price, amount and fee
// ratio
func (d *Definitions) GetTakerValue(price, amount float64) (float64, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getValue(d.taker, price, amount)
}

// GetTaker returns the taker fee value and if it is a ratio or real number
func (d *Definitions) GetTaker() (fee float64, ratio bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	if d.maker.IsZero() {
		return 0, false, errFeeIsZero
	}
	rVal, _ := d.maker.Float64()
	return rVal, d.ratio, nil
}

// getValue returns the value of the from the fee depending on its type.
func (d *Definitions) getValue(fee decimal.Decimal, price, amount float64) (float64, error) {
	if fee.IsZero() {
		return 0, errFeeIsZero
	}
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}
	var val = decimal.NewFromFloat(price).Mul(decimal.NewFromFloat(amount))
	if d.ratio {
		val = val.Mul(fee)
	} else {
		val = val.Add(fee)
	}
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
	rVal, _ := t.deposit.Float64()
	return rVal, t.ratio, nil
}

// GetWithdrawal returns the withdrawal fee associated with the currency
func (d *Definitions) GetWithdrawal(c currency.Code, a asset.Item) (fee float64, ratio bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, a)
	if err != nil {
		return 0, false, err
	}
	rVal, _ := t.withdrawal.Float64()
	return rVal, t.ratio, nil
}

// get returns the fee structure by the currency and its asset type
func (d *Definitions) get(c currency.Code, a asset.Item) (*Transfer, error) {
	if c.String() == "" {
		return nil, errCurrencyIsEmpty
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("%s, %w", c, asset.ErrNotSupported)
	}
	s, ok := d.transfer[c.Item][a]
	if !ok {
		return nil, errTransferFeeNotFound
	}
	return s, nil
}

// Transfer defines usually static real number values.
type Transfer struct {
	ratio      bool // Toggle if ratio is present
	deposit    decimal.Decimal
	withdrawal decimal.Decimal
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
