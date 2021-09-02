package state

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	manager Manager

	errEmptyCurrency         = errors.New("empty currency")
	errStatesAlreadyLoaded   = errors.New("states already loaded for exchange")
	errStatesIsNil           = errors.New("states is nil")
	errCurrencyStateNotFound = errors.New("currency state not found")
	errUpdatesAreNil         = errors.New("updates are nil")
	errExchangeNotFound      = errors.New("exchange not found")
	errExchangeNameIsEmpty   = errors.New("exchange name is empty")

	// Specific operational errors
	errDepositNotAllowed     = errors.New("depositing not allowed")
	errWithdrawalsNotAllowed = errors.New("withdrawals not allowed")
	errTradingNotAllowed     = errors.New("trading not allowed")
)

// GetManager returns the package management struct
func GetManager() *Manager {
	return &manager
}

// RegisterExchangeState generates a new states struct and registers it with the
// manager
func RegisterExchangeState(exch string) (*States, error) {
	if exch == "" {
		return nil, errExchangeNameIsEmpty
	}
	r := &States{m: make(map[asset.Item]map[*currency.Item]*Currency)}
	return r, manager.Register(exch, r)
}

// Manager attempts to govern the different states of currency defined by an
// exchange
type Manager struct {
	m   map[string]*States
	mtx sync.RWMutex
}

// Register registers a new exchange states struct
func (m *Manager) Register(exch string, s *States) error {
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
		m.m = make(map[string]*States)
	}
	m.m[exch] = s
	return nil
}

// CanTrade returns if the currency is currently tradeable on an exchange
func (m *Manager) CanTrade(exch string, c currency.Code, a asset.Item) error {
	e, err := m.getExchangeStates(exch)
	if err != nil {
		return err
	}
	return e.CanTrade(c, a)
}

// CanWithdraw returns if the currency can be withdrawn from an exchange
func (m *Manager) CanWithdraw(exch string, c currency.Code, a asset.Item) error {
	e, err := m.getExchangeStates(exch)
	if err != nil {
		return err
	}
	return e.CanWithdraw(c, a)
}

// CanDeposit returns if the currency can be deposited onto an exchange
func (m *Manager) CanDeposit(exch string, c currency.Code, a asset.Item) error {
	e, err := m.getExchangeStates(exch)
	if err != nil {
		return err
	}
	return e.CanDeposit(c, a)
}

// getExchangeStates returns the exchanges states
func (m *Manager) getExchangeStates(exch string) (*States, error) {
	if exch == "" {
		return nil, errExchangeNameIsEmpty
	}
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	e, ok := m.m[exch]
	if !ok {
		return nil, errExchangeNotFound
	}
	return e, nil
}

// States defines all currency states for an exchange
type States struct {
	m   map[asset.Item]map[*currency.Item]*Currency
	mtx sync.RWMutex
}

// CanTrade returns if the currency is currently tradeable for this exchange
func (s *States) CanTrade(c currency.Code, a asset.Item) error {
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
		return nil, errCurrencyStateNotFound
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
	if c.withdrawals != o.Withdraw {
		c.withdrawals = o.Withdraw
		c.withdrawAlerts.Alert()
	}
	if c.deposits != o.Deposit {
		c.deposits = o.Deposit
		c.depositAlerts.Alert()
	}
	if c.trading != o.Trade {
		c.trading = o.Trade
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

// Options defines the current allowable options for a currency
type Options struct {
	Withdraw bool
	Deposit  bool
	Trade    bool
}
