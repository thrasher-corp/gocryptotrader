package currencystate

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errEmptyCurrency = errors.New("empty currency")
	errUpdatesAreNil = errors.New("updates are nil")
	errNilStates     = errors.New("states is not started or set up")

	// Specific operational errors
	errDepositNotAllowed     = errors.New("depositing not allowed")
	errWithdrawalsNotAllowed = errors.New("withdrawals not allowed")
	errTradingNotAllowed     = errors.New("trading not allowed")

	// ErrCurrencyStateNotFound is an error when the currency state has not been
	// found
	// TODO: distinguish between unsupported and not found
	ErrCurrencyStateNotFound = errors.New("currency state not found")
)

// NewCurrencyStates gets a new type for tracking exchange currency states
func NewCurrencyStates() *States {
	return &States{m: make(map[asset.Item]map[*currency.Item]*Currency)}
}

// States defines all currency states for an exchange
type States struct {
	m   map[asset.Item]map[*currency.Item]*Currency
	mtx sync.RWMutex
}

// GetCurrencyStateSnapshot returns the exchange currency state snapshot
func (s *States) GetCurrencyStateSnapshot() ([]Snapshot, error) {
	if s == nil {
		return nil, errNilStates
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()
	var sh []Snapshot
	for a, m1 := range s.m {
		for c, val := range m1 {
			sh = append(sh, Snapshot{
				Code:    currency.Code{Item: c},
				Asset:   a,
				Options: val.GetState(),
			})
		}
	}
	return sh, nil
}

// CanTradePair returns if the currency pair is currently tradeable for this
// exchange. If there are no states loaded for a specific currency, this will
// assume the currency pair is operational. NOTE: Future exchanges will have
// functionality specific to a currency.Pair, can upgrade this when needed.
func (s *States) CanTradePair(pair currency.Pair, a asset.Item) error {
	err := s.CanTrade(pair.Base, a)
	if err != nil && err != ErrCurrencyStateNotFound {
		return fmt.Errorf("cannot trade base currency %s %s: %w",
			pair.Base, a, err)
	}
	err = s.CanTrade(pair.Quote, a)
	if err != nil && err != ErrCurrencyStateNotFound {
		return fmt.Errorf("cannot trade quote currency %s %s: %w",
			pair.Base, a, err)
	}
	return nil
}

// CanTrade returns if the currency is currently tradeable for this exchange
func (s *States) CanTrade(c currency.Code, a asset.Item) error {
	if s == nil {
		return errNilStates
	}

	p, err := s.Get(c, a)
	if err != nil {
		return err
	}
	if !p.CanTrade() {
		return errTradingNotAllowed
	}
	return nil
}

// CanWithdraw returns if the currency can be withdrawn from this exchange
func (s *States) CanWithdraw(c currency.Code, a asset.Item) error {
	if s == nil {
		return errNilStates
	}

	p, err := s.Get(c, a)
	if err != nil {
		return err
	}
	if !p.CanWithdraw() {
		return errWithdrawalsNotAllowed
	}
	return nil
}

// CanDeposit returns if the currency can be deposited onto this exchange
func (s *States) CanDeposit(c currency.Code, a asset.Item) error {
	if s == nil {
		return errNilStates
	}

	p, err := s.Get(c, a)
	if err != nil {
		return err
	}
	if !p.CanDeposit() {
		return errDepositNotAllowed
	}
	return nil
}

// UpdateAll updates the full currency state, used for REST calls
func (s *States) UpdateAll(a asset.Item, updates map[currency.Code]Options) error {
	if s == nil {
		return errNilStates
	}

	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	if updates == nil {
		return errUpdatesAreNil
	}
	s.mtx.Lock()
	for code, option := range updates {
		s.update(code, a, option)
	}
	s.mtx.Unlock()
	return nil
}

// Update updates a singular currency state, primarily used for singular
// websocket updates or alerts.
func (s *States) Update(c currency.Code, a asset.Item, o Options) error {
	if s == nil {
		return errNilStates
	}

	if c.String() == "" {
		return errEmptyCurrency
	}
	if !a.IsValid() {
		return fmt.Errorf("%s, %w", a, asset.ErrNotSupported)
	}
	s.mtx.Lock()
	s.update(c, a, o)
	s.mtx.Unlock()
	return nil
}

// update updates a singular currency state without protection
func (s *States) update(c currency.Code, a asset.Item, o Options) {
	m1, ok := s.m[a]
	if !ok {
		m1 = make(map[*currency.Item]*Currency)
		s.m[a] = m1
	}
	p, ok := m1[c.Item]
	if !ok {
		p = &Currency{}
		m1[c.Item] = p
	}
	p.update(o)
}

// Get returns the currency state by currency code
func (s *States) Get(c currency.Code, a asset.Item) (*Currency, error) {
	if s == nil {
		return nil, errNilStates
	}

	if c.String() == "" {
		return nil, errEmptyCurrency
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()
	cs, ok := s.m[a][c.Item]
	if !ok {
		return nil, ErrCurrencyStateNotFound
	}
	return cs, nil
}

// Currency defines the state of currency operations
type Currency struct {
	withdrawals    bool
	withdrawAlerts alert.Notice
	deposits       bool
	depositAlerts  alert.Notice
	trading        bool
	tradingAlerts  alert.Notice
	mtx            sync.RWMutex
}

// update updates the underlying values
func (c *Currency) update(o Options) {
	c.mtx.Lock()
	if o.Withdraw == nil {
		c.withdrawals = true
		c.withdrawAlerts.Alert()
	} else if c.withdrawals != *o.Withdraw {
		c.withdrawals = *o.Withdraw
		c.withdrawAlerts.Alert()
	}

	if o.Deposit == nil {
		c.deposits = true
		c.depositAlerts.Alert()
	} else if c.deposits != *o.Deposit {
		c.deposits = *o.Deposit
		c.depositAlerts.Alert()
	}

	if o.Trade == nil {
		c.trading = true
		c.tradingAlerts.Alert()
	} else if c.trading != *o.Trade {
		c.trading = *o.Trade
		c.tradingAlerts.Alert()
	}
	c.mtx.Unlock()
}

// CanTrade returns if the currency is currently tradeable
func (c *Currency) CanTrade() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.trading
}

// CanWithdraw returns if the currency can be withdrawn from the exchange
func (c *Currency) CanWithdraw() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.withdrawals
}

// CanDeposit returns if the currency can be deposited onto an exchange
func (c *Currency) CanDeposit() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.deposits
}

// WaitTrading allows a routine to wait until a trading change of state occurs
func (c *Currency) WaitTrading(kick <-chan struct{}) <-chan bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.tradingAlerts.Wait(kick)
}

// WaitDeposit allows a routine to wait until a deposit change of state occurs
func (c *Currency) WaitDeposit(kick <-chan struct{}) <-chan bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.depositAlerts.Wait(kick)
}

// WaitWithdraw allows a routine to wait until a withdraw change of state occurs
func (c *Currency) WaitWithdraw(kick <-chan struct{}) <-chan bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.withdrawAlerts.Wait(kick)
}

// GetState returns the internal state of the currency
func (c *Currency) GetState() Options {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return Options{
		Withdraw: convert.BoolPtr(c.withdrawals),
		Deposit:  convert.BoolPtr(c.deposits),
		Trade:    convert.BoolPtr(c.trading),
	}
}

// Options defines the current allowable options for a currency, using a bool
// pointer for optional setting for incomplete data, so we can default to true
// on nil values.
type Options struct {
	Withdraw *bool
	Deposit  *bool
	Trade    *bool
}

// Snapshot defines a snapshot of the internal asset for exportation
type Snapshot struct {
	Code  currency.Code
	Asset asset.Item
	Options
}
