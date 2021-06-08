package account

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Main defines a default string for the main account name, used if there is no
// need for differentiation.
const Main = "main"

var (
	errAccountNameUnset                = errors.New("account name unset")
	errCurrencyIsEmpty                 = errors.New("currency is empty")
	errAccountsNotLoaded               = errors.New("accounts not loaded")
	errAccountNotFound                 = errors.New("account not found in holdings")
	errSnapshotIsNil                   = errors.New("holdings snapshot is nil")
	errAccountBalancesNotLoaded        = errors.New("account balances not loaded")
	errCurrencyCodeEmpty               = errors.New("currrency code cannot be empty")
	errAmountCannotBeZero              = errors.New("amount cannot be zero")
	errAssetTypeNotFound               = errors.New("asset type not found in holdings")
	errAmountCannotBeLessOrEqualToZero = errors.New("amount cannot be less or equal to zero")
	errCurrencyItemNotFound            = errors.New("currency not found in holdings")
)

// Holdings defines exchange account holdings
type Holdings struct {
	Exchange string
	Verbose  bool
	// Asset type is added because of potential unknown unknowns; we can reduce
	// this map in the future when we can ensure the ability to differentiate
	// between asset types is not needed.
	funds map[string]map[asset.Item]map[*currency.Item]*Holding

	// TODO: Link up with RPC stream
	// TODO: Update dispatch.Mux type with core uuid or switch over to
	// orderbook wait and alert system for more efficient processing.
	mux *dispatch.Mux
	id  uuid.UUID

	m sync.Mutex

	// availableAccounts is segregated so we can attach a RW mutex and it
	// doesn't interfere with other systems when checking availability.
	// TODO: Deprecate and add in key manager with associated account
	// jurisdiction.
	availableAccounts []string
	accMtx            sync.RWMutex
}

// LoadAccount loads an account for future checking
func (h *Holdings) LoadAccount(account string) error {
	if account == "" {
		return errAccountNameUnset
	}

	account = strings.ToLower(account)

	h.accMtx.Lock()
	defer h.accMtx.Unlock()

	for x := range h.availableAccounts {
		if h.availableAccounts[x] == account {
			return nil
		}
	}
	h.availableAccounts = append(h.availableAccounts, account)
	return nil
}

// GetAccounts returns the loaded accounts in usage associated with the current
// global API credentials
func (h *Holdings) GetAccounts() ([]string, error) {
	h.accMtx.RLock()
	defer h.accMtx.RUnlock()

	amount := len(h.availableAccounts)
	if amount == 0 {
		return nil, errAccountsNotLoaded
	}

	acc := make([]string, amount)
	copy(acc, h.availableAccounts)
	return acc, nil
}

// AccountValid cross references account with available accounts list. Used by
// external calls GRPC and/or strategies to ensure availability before locking
// core systems.
func (h *Holdings) AccountValid(account string) error {
	if account == "" {
		return errAccountNameUnset
	}

	account = strings.ToLower(account)

	h.accMtx.RLock()
	defer h.accMtx.RUnlock()

	for x := range h.availableAccounts {
		if h.availableAccounts[x] == account {
			return nil
		}
	}
	return fmt.Errorf("%s %w: available accounts [%s]",
		account,
		errAccountNotFound,
		h.availableAccounts)
}

// GetHolding returns the holding for a specific currency tied to an account
func (h *Holdings) GetHolding(account string, a asset.Item, c currency.Code) (*Holding, error) {
	if account == "" {
		return nil, fmt.Errorf("cannot get holding for %s %s %s %s: %w",
			h.Exchange,
			account,
			a,
			c,
			errAccountNameUnset)
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("cannot get holding for %s %s %s %s: %w",
			h.Exchange,
			account,
			a,
			c,
			asset.ErrNotSupported)
	}

	if c.IsEmpty() {
		return nil, fmt.Errorf("cannot get holding for %s %s %s %s: %w",
			h.Exchange,
			account,
			a,
			c,
			errCurrencyIsEmpty)
	}

	account = strings.ToLower(account)

	h.m.Lock()
	defer h.m.Unlock()
	// Below we create the map contents if not found because if we have a
	// strategy waiting for funds on an exchange or even transfer between
	// accounts, this will set up the requirements in memory to be updated when
	// the funds come in.
	m1, ok := h.funds[account]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]*Holding)
		h.funds[account] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]*Holding)
		m1[a] = m2
	}

	holding, ok := m2[c.Item]
	if !ok {
		holding = &Holding{}
		m2[c.Item] = holding
	}

	return holding, nil
}

// LoadHoldings flushes the entire amounts with the supplied account values,
// this acts as a complete snapshot, anything held in the current holdings that
// is not part of the supplied values list will be readjusted to zero value.
func (h *Holdings) LoadHoldings(account string, a asset.Item, snapshot HoldingsSnapshot) error {
	if account == "" {
		return fmt.Errorf("cannot load holdings for %s %s %s: %w",
			h.Exchange,
			account,
			a,
			errAccountNameUnset)
	}

	if !a.IsValid() {
		return fmt.Errorf("cannot load holdings for %s %s %s: %w",
			h.Exchange,
			account,
			a,
			asset.ErrNotSupported)
	}

	// Can be of zero len as that means there is no balance associated with that
	// account. So a nil check suffices.
	if snapshot == nil {
		return fmt.Errorf("cannot load holdings for %s %s %s: %w",
			h.Exchange,
			account,
			a,
			errSnapshotIsNil)
	}

	account = strings.ToLower(account)

	h.m.Lock()
	if h.funds == nil {
		h.funds = make(map[string]map[asset.Item]map[*currency.Item]*Holding)
	}

	m1, ok := h.funds[account]
	if !ok {
		// Loads instance of account name for other sub-system interactions
		err := h.LoadAccount(account)
		if err != nil {
			h.m.Unlock()
			return fmt.Errorf("cannot load holdings for %s %s %s: %w",
				h.Exchange,
				account,
				a,
				err)
		}
		m1 = make(map[asset.Item]map[*currency.Item]*Holding)
		h.funds[account] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]*Holding)
		m1[a] = m2
	}

	// Add/Change
	for code, val := range snapshot {
		total := decimal.NewFromFloat(val.Total)
		locked := decimal.NewFromFloat(val.Locked)
		free := total.Sub(locked)
		holding, ok := m2[code.Item]
		if !ok {
			m2[code.Item] = &Holding{
				total:  total,
				locked: locked,
				free:   free,
			}
			continue
		}
		holding.setAmounts(total, locked)
	}

	// Set dangling values to zero
holdings:
	for code, holding := range m2 {
		for c := range snapshot {
			if code == c.Item {
				continue holdings
			}
		}
		holding.setAmounts(decimal.Zero, decimal.Zero)
	}

	if h.Verbose {
		m := make(HoldingsSnapshot)
		for k, v := range m2 {
			balance, err := v.GetBalance(false)
			if err != nil {
				continue
			}
			m[currency.Code{Item: k}] = balance
		}
		log.Debugf(log.Accounts,
			"Exchange:%s Account:%s Asset:%s Holdings Loaded %+v",
			h.Exchange,
			account,
			a,
			m)
	}

	h.m.Unlock()
	// publish change to dispatch system and TODO: portfolio
	h.publish()
	return nil
}

// GetHoldingsSnapshot returns holdings for an account asset
func (h *Holdings) GetHoldingsSnapshot(account string, ai asset.Item) (HoldingsSnapshot, error) {
	if account == "" {
		return nil, fmt.Errorf("cannot load holdings for %s %s %s: %w",
			h.Exchange,
			account,
			ai,
			errAccountNameUnset)
	}

	if !ai.IsValid() {
		return nil, fmt.Errorf("cannot load holdings for %s %s %s: %w",
			h.Exchange,
			account,
			ai,
			asset.ErrNotSupported)
	}

	h.m.Lock()
	defer h.m.Unlock()

	if h.funds == nil {
		return nil, errAccountBalancesNotLoaded
	}

	m1, ok := h.funds[account]
	if !ok {
		return nil, errAccountNotFound
	}

	holdings, ok := m1[ai]
	if !ok {
		return nil, errAssetTypeNotFound
	}

	m := make(HoldingsSnapshot)
	for c, bal := range holdings {
		total := bal.GetTotal()
		if total > 0 {
			m[currency.Code{Item: c}] = Balance{Total: total, Locked: bal.GetLocked()}
		}
	}
	return m, nil
}

// GetFullSnapshot returns a full snapshot of the current exchange' account
// balances
func (h *Holdings) GetFullSnapshot() (FullSnapshot, error) {
	if h.funds == nil {
		return nil, errAccountBalancesNotLoaded
	}

	m := make(FullSnapshot)

	h.m.Lock()
	defer h.m.Unlock()
	for account, m1 := range h.funds {
		for ai, m2 := range m1 {
			for c, holding := range m2 {
				shm1, ok := m[account]
				if !ok {
					shm1 = make(map[asset.Item]HoldingsSnapshot)
					m[account] = shm1
				}

				shm2, ok := shm1[ai]
				if !ok {
					shm2 = make(HoldingsSnapshot)
					shm1[ai] = shm2
				}

				bal, err := holding.GetBalance(false)
				if err != nil {
					continue
				}
				shm2[currency.Code{Item: c}] = bal
			}
		}
	}
	return m, nil
}

// publish publishes update to the dispatch mux to be called in a go routine
func (h *Holdings) publish() {
	ss, err := h.GetFullSnapshot()
	if err != nil {
		log.Errorf(log.Accounts, "cannot publish to dispatch mux %v", err)
	}
	err = h.mux.Publish(&ss, h.id)
	if err != nil {
		log.Errorf(log.Accounts, "cannot publish to dispatch mux %v", err)
	}
}

// AdjustByBalance matches with currency currency holding and decreases or
// increases on value change. i.e. if negative will decrease current holdings
// if positive will increase current holdings
func (h *Holdings) AdjustByBalance(account string, ai asset.Item, c currency.Code, amount float64) error {
	err := h.validate(account, ai, c, amount, true)
	if err != nil {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			ai,
			c,
			amount,
			err)
	}

	account = strings.ToLower(account)

	holding, err := h.getHoldingInternal(account, ai, c.Item)
	if err != nil {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			ai,
			c,
			amount,
			err)
	}

	err = holding.adjustByBalance(amount)
	if err != nil {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			ai,
			c,
			amount,
			err)
	}

	if h.Verbose {
		bal, _ := holding.GetBalance(false)
		log.Debugf(log.Accounts,
			"Exchange:%s Account:%s Asset:%s Currency:%s Balance Adjusted by %f Current Free Holdings:%f Current Total Holdings:%f",
			h.Exchange,
			account,
			ai,
			c,
			amount,
			bal.Total-bal.Locked,
			bal.Total)
	}
	return nil
}

func (h *Holdings) Claim(account string, ai asset.Item, c currency.Code, amount float64, totalRequired bool) (*Claim, error) {
	err := h.validate(account, ai, c, amount, false)
	if err != nil {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			ai,
			c,
			amount,
			err)
	}

	account = strings.ToLower(account)
	holding, err := h.getHoldingInternal(account, ai, c.Item)
	if err != nil {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			ai,
			c,
			amount,
			err)
	}

	claim, err := holding.Claim(amount, totalRequired)
	if err != nil {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			ai,
			c,
			amount,
			err)
	}

	if h.Verbose {
		bal, _ := holding.GetBalance(false)
		log.Debugf(log.Accounts,
			"Exchange:%s Account:%s Asset:%s Currency:%s total required: %v, amount %f claimed on holdings with amount requested %f Free Holdings:%f Total Holdings:%f",
			h.Exchange,
			account,
			ai,
			c,
			totalRequired,
			claim.GetAmount(),
			amount,
			bal.Total-bal.Locked,
			bal.Total)
	}

	return claim, err
}

// getHoldingInternal returns the individual account holding but does not create
// an instance like the function above
func (h *Holdings) getHoldingInternal(account string, ai asset.Item, ci *currency.Item) (*Holding, error) {
	// lock and unlock here so we can release this 'global' lock as fast as
	// possible and only work on the individual holding locks if needed.
	h.m.Lock()
	defer h.m.Unlock()
	m1, ok := h.funds[account]
	if !ok {
		return nil, errAccountNotFound
	}
	m2, ok := m1[ai]
	if !ok {
		return nil, errAssetTypeNotFound
	}
	holding, ok := m2[ci]
	if !ok {
		return nil, errCurrencyItemNotFound
	}
	return holding, nil
}

// validate checks if request values are correct before locking down holdings
func (h *Holdings) validate(account string, ai asset.Item, c currency.Code, amount float64, lessThanZero bool) error {
	if account == "" {
		return errAccountNameUnset
	}
	if !ai.IsValid() {
		return asset.ErrNotSupported
	}
	if c.IsEmpty() {
		return errCurrencyCodeEmpty
	}
	if amount == 0 || (!lessThanZero && amount < 0) {
		return errAmountCannotBeLessOrEqualToZero
	}
	return h.AccountValid(account)
}
