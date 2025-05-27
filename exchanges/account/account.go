package account

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Public errors
var (
	ErrExchangeHoldingsNotFound = errors.New("exchange holdings not found")
)

var (
	errExchangeNameUnset            = errors.New("exchange name unset")
	errNoExchangeSubAccountBalances = errors.New("no exchange sub account balances")
	errNoCredentialBalances         = errors.New("no balances associated with credentials")
	errCredentialsAreNil            = errors.New("credentials are nil")
	errOutOfSequence                = errors.New("out of sequence")
	errUpdatedAtIsZero              = errors.New("updatedAt may not be zero")
	errLoadingBalance               = errors.New("error loading balance")
	errExchangeAlreadyExists        = errors.New("exchange already exists")
	errCannotUpdateBalance          = errors.New("cannot update balance")
)

var global atomic.Pointer[store]

// NewStore returns a new store with the default global dispatcher mux
func NewStore() *store {
	return &store{
		exchangeAccounts: make(map[string]*Accounts),
		mux:              dispatch.GetNewMux(nil),
	}
}

// GetStore returns the singleton accounts store for global use; Initialising if necessary
func GetStore() *store {
	if s := global.Load(); s != nil {
		return s
	}
	_ = global.CompareAndSwap(nil, NewStore())
	return global.Load()
}

// MustNewAccounts returns an initialized Accounts store for use in isolation from a global exchange accounts store
// Any errors in mux ID generation will panic, so users should balance risk vs utility accordingly depending on use-case
func MustNewAccounts(eName string, mux *dispatch.Mux) *Accounts {
	a, err := NewAccounts(eName, mux)
	if err != nil {
		panic(err)
	}
	return a
}

// NewAccounts returns an initialized Accounts store for use in isolation from a global exchange accounts store
func NewAccounts(exchange string, mux *dispatch.Mux) (*Accounts, error) {
	id, err := mux.GetID()
	if err != nil {
		return nil, err
	}
	return &Accounts{
		Exchange:    strings.ToLower(exchange),
		subAccounts: make(map[Credentials]map[key.SubAccountAsset]currencyBalances),
		ID:          id,
		mux:         mux,
	}, nil
}

// registerExchange adds a new empty shared account accounts entry for an exchange
// must be called with s.mu locked
func (s *store) registerExchange(exch string) (*Accounts, error) {
	exch = strings.ToLower(exch)
	if _, ok := s.exchangeAccounts[exch]; ok {
		return nil, errExchangeAlreadyExists
	}
	a, err := NewAccounts(exch, s.mux)
	s.exchangeAccounts[exch] = a
	return a, err
}

// CollectBalances converts a map of sub-account balances into a slice
func CollectBalances(accountBalances map[string][]Balance, assetType asset.Item) (accounts []SubAccount, err error) {
	if err := common.NilGuard(accountBalances); err != nil {
		return nil, err
	}

	if !assetType.IsValid() {
		return nil, fmt.Errorf("%s, %w", assetType, asset.ErrNotSupported)
	}

	accounts = make([]SubAccount, 0, len(accountBalances))
	for accountID, balances := range accountBalances {
		accounts = append(accounts, SubAccount{
			ID:         accountID,
			AssetType:  assetType,
			Currencies: balances,
		})
	}
	return
}

// GetExchangeAccounts returns accounts for a specific exchange
func (s *store) GetExchangeAccounts(e string) (a *Accounts, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e == "" {
		return nil, errExchangeNameUnset
	}
	a, ok := s.exchangeAccounts[strings.ToLower(e)]
	if !ok {
		if a, err = s.registerExchange(e); err != nil {
			return nil, fmt.Errorf("error subscribing to `%s` exchange account: %w", e, err)
		}
	}
	return a, nil
}

// Subscribe subscribes to your exchange accounts
func (a *Accounts) Subscribe() (dispatch.Pipe, error) {
	return a.mux.Subscribe(a.ID)
}

// GetHoldings returns full holdings for an exchange.
// NOTE: Due to credentials these amounts could be N*APIKEY actual holdings.
// TODO: Add jurisdiction and differentiation between APIKEY holdings.
func (a *Accounts) GetHoldings(creds *Credentials, assetType asset.Item) (Holdings, error) {
	if err := common.NilGuard(a); err != nil {
		return Holdings{}, err
	}

	if creds.IsEmpty() {
		return Holdings{}, fmt.Errorf("%s %s %w", a.Exchange, assetType, errCredentialsAreNil)
	}

	if !assetType.IsValid() {
		return Holdings{}, fmt.Errorf("%s %s %w", a.Exchange, assetType, asset.ErrNotSupported)
	}

	subAccountHoldings, ok := a.subAccounts[*creds]
	if !ok {
		return Holdings{}, fmt.Errorf("%s %s %s %w %w", a.Exchange, creds, assetType, errNoCredentialBalances, ErrExchangeHoldingsNotFound)
	}

	currencyBalances := make([]Balance, 0, len(subAccountHoldings))
	cpy := *creds
	for mapKey, assets := range subAccountHoldings {
		if mapKey.Asset != assetType {
			continue
		}
		for currItem, bal := range assets {
			bal.m.Lock()
			currencyBalances = append(currencyBalances, Balance{
				Currency:               currItem.Currency().Upper(),
				Total:                  bal.total,
				Hold:                   bal.hold,
				Free:                   bal.free,
				AvailableWithoutBorrow: bal.availableWithoutBorrow,
				Borrowed:               bal.borrowed,
				UpdatedAt:              bal.updatedAt,
			})
			bal.m.Unlock()
		}
		if cpy.SubAccount == "" && mapKey.SubAccount != "" {
			// TODO: fix this backwards population
			// the subAccount here may not be associated with the balance across all subAccountHoldings
			cpy.SubAccount = mapKey.SubAccount
		}
	}
	if len(currencyBalances) == 0 {
		return Holdings{}, fmt.Errorf("%s %s %w", a.Exchange, assetType, ErrExchangeHoldingsNotFound)
	}
	return Holdings{
		Exchange: a.Exchange,
		Accounts: []SubAccount{{
			Credentials: Protected{creds: cpy},
			ID:          cpy.SubAccount,
			AssetType:   assetType,
			Currencies:  currencyBalances,
		}},
	}, nil
}

// GetBalance returns the internal balance for that asset item.
func (s *store) GetBalance(exch, subAccount string, creds *Credentials, a asset.Item, c currency.Code) (*ProtectedBalance, error) {
	if exch == "" {
		return nil, fmt.Errorf("cannot get balance: %w", errExchangeNameUnset)
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("cannot get balance: %s %w", a, asset.ErrNotSupported)
	}

	if creds.IsEmpty() {
		return nil, fmt.Errorf("cannot get balance: %w", errCredentialsAreNil)
	}

	if c.IsEmpty() {
		return nil, fmt.Errorf("cannot get balance: %w", currency.ErrCurrencyCodeEmpty)
	}

	exch = strings.ToLower(exch)
	s.mu.Lock()
	defer s.mu.Unlock()

	accounts, ok := s.exchangeAccounts[exch]
	if !ok {
		return nil, fmt.Errorf("%w for %s", ErrExchangeHoldingsNotFound, exch)
	}

	subAccounts, ok := accounts.subAccounts[*creds]
	if !ok {
		return nil, fmt.Errorf("%w for %s %s", errNoCredentialBalances, exch, creds)
	}

	assets, ok := subAccounts[key.SubAccountAsset{
		SubAccount: subAccount,
		Asset:      a,
	}]
	if !ok {
		return nil, fmt.Errorf("%w for %s SubAccount %q %s %s", errNoExchangeSubAccountBalances, exch, subAccount, a, c)
	}
	bal, ok := assets[c.Item]
	if !ok {
		return nil, fmt.Errorf("%w for %s SubAccount %q %s %s", errNoExchangeSubAccountBalances, exch, subAccount, a, c)
	}
	return bal, nil
}

// Save saves the holdings with new account info
// h should be a full update, and any missing currencies will be zeroed
// h.Exchange is ignored
func (a *Accounts) Save(h *Holdings, creds *Credentials) error {
	if err := common.NilGuard(a, h); err != nil {
		return fmt.Errorf("cannot save holdings: %w", err)
	}
	if err := common.NilGuard(a.subAccounts); err != nil {
		return fmt.Errorf("cannot save holdings: %w", err)
	}

	if creds.IsEmpty() {
		return fmt.Errorf("cannot save holdings: %w", errCredentialsAreNil)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	subAccounts, ok := a.subAccounts[*creds]
	if !ok {
		subAccounts = make(map[key.SubAccountAsset]currencyBalances)
		a.subAccounts[*creds] = subAccounts
	}

	var errs error
	for i := range h.Accounts {
		subAccount := h.Accounts[i]
		if !subAccount.AssetType.IsValid() {
			errs = common.AppendError(errs, fmt.Errorf("cannot load sub account holdings for %s [%s] %w",
				subAccount.ID,
				subAccount.AssetType,
				asset.ErrNotSupported))
			continue
		}

		// This assignment outside of scope is designed to have minimal impact
		// on the exchange implementation UpdateAccountInfo() and portfoio
		// management.
		// TODO: Update incoming Holdings type to already be populated. (Suggestion)
		cpy := *creds
		if cpy.SubAccount == "" {
			cpy.SubAccount = subAccount.ID
		}

		accAsset := key.SubAccountAsset{
			SubAccount: subAccount.ID,
			Asset:      subAccount.AssetType,
		}
		assets, ok := subAccounts[accAsset]
		if !ok {
			assets = make(map[*currency.Item]*ProtectedBalance)
			a.subAccounts[*creds][accAsset] = assets
		}

		updated := make(map[*currency.Item]bool)
		for y := range subAccount.Currencies {
			accBal := &subAccount.Currencies[y]
			if accBal.UpdatedAt.IsZero() {
				accBal.UpdatedAt = time.Now()
			}
			bal, ok := assets[accBal.Currency.Item]
			if !ok || bal == nil {
				bal = &ProtectedBalance{}
			}
			if err := bal.load(accBal); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w for account ID `%s` [%s %s]: %w",
					errLoadingBalance,
					subAccount.ID,
					subAccount.AssetType,
					subAccount.Currencies[y].Currency,
					err))
				continue
			}
			assets[accBal.Currency.Item] = bal
			updated[accBal.Currency.Item] = true
		}
		for cur, bal := range assets {
			if !updated[cur] {
				bal.reset()
			}
		}

		if err := a.mux.Publish(subAccount, a.ID); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("cannot publish load for %s %w", a.Exchange, err))
		}
	}

	return errs
}

// Update updates the balance for a specific exchange and credentials
func (a *Accounts) Update(changes []Change, creds *Credentials) error {
	if err := common.NilGuard(a); err != nil {
		return fmt.Errorf("cannot save holdings: %w", err)
	}
	if err := common.NilGuard(a.subAccounts); err != nil {
		return fmt.Errorf("cannot save holdings: %w", err)
	}
	if creds.IsEmpty() {
		return fmt.Errorf("%w: %w", errCannotUpdateBalance, errCredentialsAreNil)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	subAccounts, ok := a.subAccounts[*creds]
	if !ok {
		subAccounts = make(map[key.SubAccountAsset]currencyBalances)
		a.subAccounts[*creds] = subAccounts
	}

	var errs error
	for _, change := range changes {
		if !change.AssetType.IsValid() {
			errs = common.AppendError(errs, fmt.Errorf("%w for %s.%s %w",
				errCannotUpdateBalance, change.Account, change.AssetType, asset.ErrNotSupported))
			continue
		}
		if err := common.NilGuard(change.Balance); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w for %s.%s %w",
				errCannotUpdateBalance, change.Account, change.AssetType, err))
			continue
		}

		accAsset := key.SubAccountAsset{
			SubAccount: change.Account,
			Asset:      change.AssetType,
		}
		assets, ok := subAccounts[accAsset]
		if !ok {
			assets = make(map[*currency.Item]*ProtectedBalance)
			a.subAccounts[*creds][accAsset] = assets
		}
		bal, ok := assets[change.Balance.Currency.Item]
		if !ok || bal == nil {
			bal = &ProtectedBalance{}
			assets[change.Balance.Currency.Item] = bal
		}

		if err := bal.load(change.Balance); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w for %s.%s.%s %w",
				errCannotUpdateBalance,
				change.Account,
				change.AssetType,
				change.Balance.Currency,
				err))
			continue
		}
		if err := a.mux.Publish(change, a.ID); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("cannot publish update balance for %s: %w", a.Exchange, err))
		}
	}
	return errs
}

// load checks to see if there is a change from incoming balance, if there is a
// change it will change then alert external routines.
func (b *ProtectedBalance) load(change *Balance) error {
	if change == nil {
		return fmt.Errorf("%w for '%T'", common.ErrNilPointer, change)
	}
	if change.UpdatedAt.IsZero() {
		return errUpdatedAtIsZero
	}
	b.m.Lock()
	defer b.m.Unlock()
	if !b.updatedAt.IsZero() && !b.updatedAt.Before(change.UpdatedAt) {
		return errOutOfSequence
	}
	if b.total == change.Total &&
		b.hold == change.Hold &&
		b.free == change.Free &&
		b.availableWithoutBorrow == change.AvailableWithoutBorrow &&
		b.borrowed == change.Borrowed &&
		b.updatedAt.Equal(change.UpdatedAt) {
		return nil
	}
	b.total = change.Total
	b.hold = change.Hold
	b.free = change.Free
	b.availableWithoutBorrow = change.AvailableWithoutBorrow
	b.borrowed = change.Borrowed
	b.updatedAt = change.UpdatedAt
	b.notice.Alert()
	return nil
}

// Wait waits for a change in amounts for an asset type. This will pause
// indefinitely if no change ever occurs. Max wait will return true if it failed
// to achieve a state change in the time specified. If Max wait is not specified
// it will default to a minute wait time.
func (b *ProtectedBalance) Wait(maxWait time.Duration) (wait <-chan bool, cancel chan<- struct{}, err error) {
	if err := common.NilGuard(b); err != nil {
		return nil, nil, err
	}

	if maxWait <= 0 {
		maxWait = time.Minute
	}
	ch := make(chan struct{})
	go func(ch chan<- struct{}, until time.Duration) {
		time.Sleep(until)
		close(ch)
	}(ch, maxWait)

	return b.notice.Wait(ch), ch, nil
}

// GetFree returns the current free balance for the exchange
func (b *ProtectedBalance) GetFree() float64 {
	if b == nil {
		return 0
	}
	b.m.Lock()
	defer b.m.Unlock()
	return b.free
}

func (b *ProtectedBalance) reset() {
	b.m.Lock()
	defer b.m.Unlock()

	b.total = 0
	b.hold = 0
	b.free = 0
	b.availableWithoutBorrow = 0
	b.borrowed = 0
	b.updatedAt = time.Now()
	b.notice.Alert()
}
