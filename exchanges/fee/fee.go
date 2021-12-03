package fee

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
)

var (
	// ErrScheduleIsNil defines if the exchange specific fee schedule has bot
	// been loaded or set up.
	ErrScheduleIsNil = errors.New("fee Schedule is nil")

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
	errNoTransferFees          = errors.New("missing transfer fees to load")

	// OmitPair is a an empty pair designation for unused pair variables
	OmitPair = currency.Pair{}

	// AllAllowed defines a potential bank transfer when all foreign exchange
	// currencies are allowed to operate.
	AllAllowed = currency.NewCode("ALLALLOWED")
)

// NewFeeSchedule generates a new fee struct for exchange usage
func NewFeeSchedule() *Schedule {
	return &Schedule{
		globalCommissions: make(map[asset.Item]*CommissionInternal),
		pairCommissions:   make(map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal),
		chainTransfer:     make(map[*currency.Item]map[string]*transfer),
		bankTransfer:      make(map[bank.Transfer]map[*currency.Item]*transfer),
	}
}

// Schedule defines the full fee schedule for different currencies
// TODO: Eventually upgrade with key manager for different fees associated
// with different accounts/keys.
type Schedule struct {
	// Commission is the holder for the up to date commission rates for the
	// assets.
	globalCommissions map[asset.Item]*CommissionInternal
	// pairCommissions is the holder for the up to date commissions rates for
	// the specific trading pairs.
	pairCommissions map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal
	// transfer defines a map of currencies with differing withdrawal and
	// deposit fee schedule. These will commonly be real values.
	chainTransfer map[*currency.Item]map[string]*transfer
	// bankTransfer defines a map of currencies with differing withdrawal and
	// deposit fee schedule for banking. These will commonly be fixed real
	// values.
	bankTransfer map[bank.Transfer]map[*currency.Item]*transfer
	mtx          sync.RWMutex
}

// LoadDynamicFeeRate loads the current dynamic account fee rate for maker and
// taker values. As a standard this is loaded as a rate e.g. 0.2% fee as a rate
// would be 0.2/100 == 0.002.
//
// The pair is an optional parameter if omitted will designate global/exchange
// maker, taker fees irrespective of individual trading operations.
func (d *Schedule) LoadDynamicFeeRate(maker, taker float64, a asset.Item, pair currency.Pair) error {
	if d == nil {
		return ErrScheduleIsNil
	}

	if err := checkCommissionRates(maker, taker); err != nil {
		return err
	}

	if !a.IsValid() {
		return fmt.Errorf("%s: %w", a, asset.ErrNotSupported)
	}

	var c *CommissionInternal

	d.mtx.Lock()
	defer d.mtx.Unlock()
	if !pair.IsEmpty() {
		// NOTE: These will create maps when needed, this system can initially
		// start out as a global commission rate and is updated ad-hoc.
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

// LoadStaticFees loads predefined custom long term fee structures for trading
// fees (which will automatically be loaded as worst case scenario fees),
// transfer fees to and from blockchains/wallets/exchanges, and bank transfer
// fees.
func (d *Schedule) LoadStaticFees(o Options) error {
	if d == nil {
		return ErrScheduleIsNil
	}

	if err := o.validate(); err != nil {
		return err
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	// Loads global commission rates based on asset item.
	for a, value := range o.GlobalCommissions {
		d.globalCommissions[a] = value.convert()
	}

	// Loads pair specific commission rates.
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

	// Loads blockchain/wallet/exchange withdrawal and deposit fees.
	for x := range o.ChainTransfer {
		chainTransfer, ok := d.chainTransfer[o.ChainTransfer[x].Currency.Item]
		if !ok {
			chainTransfer = make(map[string]*transfer)
			d.chainTransfer[o.ChainTransfer[x].Currency.Item] = chainTransfer
		}
		chainTransfer[o.ChainTransfer[x].Chain] = o.ChainTransfer[x].convert()
	}

	// Loads international banking withdrawal and deposit fees.
	for x := range o.BankTransfer {
		transferFees, ok := d.bankTransfer[o.BankTransfer[x].BankTransfer]
		if !ok {
			transferFees = make(map[*currency.Item]*transfer)
			d.bankTransfer[o.BankTransfer[x].BankTransfer] = transferFees
		}
		transferFees[o.BankTransfer[x].Currency.Item] = o.BankTransfer[x].convert()
	}
	return nil
}

// GetCommissionFee returns a pointer of the current commission rate for the
// asset type.
func (d *Schedule) GetCommissionFee(a asset.Item, pair currency.Pair) (*CommissionInternal, error) {
	if d == nil {
		return nil, ErrScheduleIsNil
	}
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getCommission(a, pair)
}

// getCommission returns the internal commission rate based on provided params
func (d *Schedule) getCommission(a asset.Item, pair currency.Pair) (*CommissionInternal, error) {
	if len(d.pairCommissions) != 0 && !pair.IsEmpty() {
		if c, ok := d.pairCommissions[a][pair.Base.Item][pair.Quote.Item]; ok {
			return c, nil
		}
		return nil, fmt.Errorf("pair %w for %s %s", errCommissionRateNotFound, a, pair)
	}
	if c, ok := d.globalCommissions[a]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("global %w for %s", errCommissionRateNotFound, a)
}

// CalculateMaker returns the fee amount derived from the price, amount and fee
// percentage.
func (d *Schedule) CalculateMaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrScheduleIsNil
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
// and fee percentage using the worst-case scenario trading fee. This is usually
// the initial loaded fee in an exchanges wrapper.go setup function.
func (d *Schedule) CalculateWorstCaseMaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrScheduleIsNil
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
// number.
func (d *Schedule) GetMaker(a asset.Item, pair currency.Pair) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrScheduleIsNil
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
func (d *Schedule) CalculateTaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrScheduleIsNil
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
// and fee percentage using the worst-case scenario trading fee. This is usually
// the initial loaded fee in an exchanges wrapper.go setup function.
func (d *Schedule) CalculateWorstCaseTaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrScheduleIsNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateWorstCaseTaker(price, amount)
}

// GetTaker returns the taker fee value and if it is a percentage or real number.
func (d *Schedule) GetTaker(a asset.Item, pair currency.Pair) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrScheduleIsNil
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

// CalculateDeposit returns calculated fee from the amount, chain can be omitted
// as a it refers to the main chain.
func (d *Schedule) CalculateDeposit(c currency.Code, chain string, amount float64) (float64, error) {
	if d == nil {
		return 0, ErrScheduleIsNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return 0, err
	}
	return t.calculate(t.Deposit, amount)
}

// GetDeposit returns the deposit fee associated with the currency, chain can be
// omitted as a it refers to the main chain.
func (d *Schedule) GetDeposit(c currency.Code, chain string) (fee Value, isPercentage bool, err error) {
	if d == nil {
		return nil, false, ErrScheduleIsNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return nil, false, err
	}
	return t.Deposit, t.Percentage, nil
}

// CalculateDeposit returns calculated fee from the amount, chain can be omitted
// as a it refers to the main chain.
func (d *Schedule) CalculateWithdrawal(c currency.Code, chain string, amount float64) (float64, error) {
	if d == nil {
		return 0, ErrScheduleIsNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return 0, err
	}
	return t.calculate(t.Withdrawal, amount)
}

// GetWithdrawal returns the withdrawal fee associated with the currency, chain
// can be omitted as a it refers to the main chain.
func (d *Schedule) GetWithdrawal(c currency.Code, chain string) (fee Value, isPercentage bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return nil, false, err
	}
	return t.Withdrawal, t.Percentage, nil
}

// get returns the fee structure by the currency and its chain type.
func (d *Schedule) get(c currency.Code, chain string) (*transfer, error) {
	if c.String() == "" {
		return nil, errCurrencyIsEmpty
	}

	s, ok := d.chainTransfer[c.Item][chain]
	if !ok {
		return nil, errTransferFeeNotFound
	}
	return s, nil
}

// GetAllFees returns a snapshot of the full fee Schedule.
func (d *Schedule) GetAllFees() (Options, error) {
	if d == nil {
		return Options{}, ErrScheduleIsNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	op := Options{
		GlobalCommissions: make(map[asset.Item]Commission),
		PairCommissions:   make(map[asset.Item]map[currency.Pair]Commission),
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

	for currencyItem, m1 := range d.chainTransfer {
		for chain, val := range m1 {
			out := val.convert()
			out.Currency = currency.Code{Item: currencyItem, UpperCase: true}
			out.Chain = chain
			op.ChainTransfer = append(op.ChainTransfer, *out)
		}
	}

	for bankProtocol, m1 := range d.bankTransfer {
		for currencyItem, val := range m1 {
			out := val.convert()
			out.Currency = currency.Code{Item: currencyItem, UpperCase: true}
			out.BankTransfer = bankProtocol
			op.BankTransfer = append(op.BankTransfer, *out)
		}
	}
	return op, nil
}

// SetCommissionFee sets new global fees and forces custom control for that
// asset. TODO: Add write control when this gets changed.
func (d *Schedule) SetCommissionFee(a asset.Item, pair currency.Pair, maker, taker float64, isFixedAmount bool) error {
	if d == nil {
		return ErrScheduleIsNil
	}

	err := checkCommissionRates(maker, taker)
	if err != nil {
		return err
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
	return c.set(maker, taker, isFixedAmount)
}

// GetTransferFee returns a snapshot of the current transfer fees for the
// asset type.
func (d *Schedule) GetTransferFee(c currency.Code, chain string) (*Transfer, error) {
	if d == nil {
		return nil, ErrScheduleIsNil
	}

	if c.String() == "" {
		return nil, errCurrencyIsEmpty
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, ok := d.chainTransfer[c.Item][chain]
	if !ok {
		return nil, errRateNotFound
	}
	return t.convert(), nil
}

// SetTransferFees sets new transfer fees.
// TODO: Need min and max settings, might deprecate due to complexity of value
// types. Or expand out the RPC to set custom values.
func (d *Schedule) SetTransferFee(c currency.Code, chain string, withdraw, deposit float64, isPercentage bool) error {
	if d == nil {
		return ErrScheduleIsNil
	}

	if withdraw < 0 {
		return errWithdrawalIsInvalid
	}

	if deposit < 0 {
		return errDepositIsInvalid
	}

	if c.String() == "" {
		return errCurrencyIsEmpty
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	t, ok := d.chainTransfer[c.Item][chain]
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

// GetBankTransferFee returns a snapshot of the current bank transfer fees for
// the asset.
func (d *Schedule) GetBankTransferFee(c currency.Code, transType bank.Transfer) (*Transfer, error) {
	if d == nil {
		return nil, ErrScheduleIsNil
	}

	if c.String() == "" {
		return nil, errCurrencyIsEmpty
	}

	if err := transType.Validate(); err != nil {
		return nil, err
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, ok := d.bankTransfer[transType][c.Item]
	if !ok {
		return nil, errRateNotFound
	}
	return t.convert(), nil
}

// SetBankTransferFee sets new bank transfer fees
// TODO: Need min and max settings, might deprecate due to complexity of value
// types. Or expand out the RPC to set custom values.
func (d *Schedule) SetBankTransferFee(c currency.Code, transType bank.Transfer, withdraw, deposit float64, isPercentage bool) error {
	if d == nil {
		return ErrScheduleIsNil
	}

	if c.String() == "" {
		return errCurrencyIsEmpty
	}

	if err := transType.Validate(); err != nil {
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
	tFee, ok := d.bankTransfer[transType][c.Item]
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

// LoadChainTransferFees allows the loading of current transfer fees for
// cryptocurrency deposit and withdrawals.
func (d *Schedule) LoadChainTransferFees(fees []Transfer) error {
	if d == nil {
		return ErrScheduleIsNil
	}

	if len(fees) == 0 {
		return errNoTransferFees
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	for x := range fees {
		err := fees[x].validate()
		if err != nil {
			return fmt.Errorf("loading crypto fees error: %w", err)
		}
		m1, ok := d.chainTransfer[fees[x].Currency.Item]
		if !ok {
			m1 = make(map[string]*transfer)
			d.chainTransfer[fees[x].Currency.Item] = m1
		}
		val, ok := m1[fees[x].Chain]
		if !ok {
			m1[fees[x].Chain] = fees[x].convert()
			continue
		}
		err = val.update(&fees[x])
		if err != nil {
			return fmt.Errorf("loading crypto fees error: %w", err)
		}
	}
	return nil
}

// checkCommissionRates checks and validates maker and taker rates
func checkCommissionRates(maker, taker float64) error {
	if maker > taker {
		return errMakerBiggerThanTaker
	}

	// Abs so we check threshold levels in positive and negative direction.
	if math.Abs(maker) >= defaultPercentageRateThreshold {
		return fmt.Errorf("%w exceeds percentage rate threshold %f",
			errMakerInvalid,
			defaultPercentageRateThreshold)
	}
	if math.Abs(taker) >= defaultPercentageRateThreshold {
		return fmt.Errorf("%w exceeds percentage rate threshold %f",
			errTakerInvalid,
			defaultPercentageRateThreshold)
	}
	return nil
}
