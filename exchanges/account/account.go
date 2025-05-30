package account

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func init() {
	service.exchangeAccounts = make(map[string]*Accounts)
	service.mux = dispatch.GetNewMux(nil)
}

// Public errors
var (
	ErrExchangeHoldingsNotFound = errors.New("exchange holdings not found")
)

var (
	errHoldingsIsNil                = errors.New("holdings cannot be nil")
	errExchangeNameUnset            = errors.New("exchange name unset")
	errNoExchangeSubAccountBalances = errors.New("no exchange sub account balances")
	errBalanceIsNil                 = errors.New("balance is nil")
	errNoCredentialBalances         = errors.New("no balances associated with credentials")
	errCredentialsAreNil            = errors.New("credentials are nil")
	errOutOfSequence                = errors.New("out of sequence")
	errUpdatedAtIsZero              = errors.New("updatedAt may not be zero")
	errLoadingBalance               = errors.New("error loading balance")
	errExchangeAlreadyExists        = errors.New("exchange already exists")
	errCannotUpdateBalance          = errors.New("cannot update balance")
)

// initAccounts adds a new empty shared account accounts entry for an exchange
// must be called with s.mu locked
func (s *Service) initAccounts(exch string) (*Accounts, error) {
	id, err := s.mux.GetID()
	if err != nil {
		return nil, err
	}
	_, ok := s.exchangeAccounts[exch]
	if ok {
		return nil, errExchangeAlreadyExists
	}
	accounts := &Accounts{
		ID:          id,
		subAccounts: make(map[Credentials]map[key.SubAccountAsset]currencyBalances),
	}
	s.exchangeAccounts[exch] = accounts
	return accounts, nil
}

// CollectBalances converts a map of sub-account balances into a slice
func CollectBalances(accountBalances map[string][]Balance, assetType asset.Item) (accounts []SubAccount, err error) {
	if accountBalances == nil {
		return nil, errAccountBalancesIsNil
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

// SubscribeToExchangeAccount subscribes to your exchange account
func SubscribeToExchangeAccount(exchange string) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.mu.Lock()
	defer service.mu.Unlock()
	accounts, ok := service.exchangeAccounts[exchange]
	if !ok {
		var err error
		if accounts, err = service.initAccounts(exchange); err != nil {
			return dispatch.Pipe{}, fmt.Errorf("cannot subscribe to exchange account %w", err)
		}
	}
	return service.mux.Subscribe(accounts.ID)
}

// Process processes new account holdings updates
func Process(h *Holdings, c *Credentials) error {
	return service.Save(h, c)
}

// ProcessChange updates the changes to the exchange account
func ProcessChange(exch string, changes []Change, c *Credentials) error {
	return service.Update(exch, changes, c)
}

// GetHoldings returns full holdings for an exchange.
// NOTE: Due to credentials these amounts could be N*APIKEY actual holdings.
// TODO: Add jurisdiction and differentiation between APIKEY holdings.
func GetHoldings(exch string, creds *Credentials, assetType asset.Item) (Holdings, error) {
	if exch == "" {
		return Holdings{}, errExchangeNameUnset
	}

	if creds.IsEmpty() {
		return Holdings{}, fmt.Errorf("%s %s %w", exch, assetType, errCredentialsAreNil)
	}

	if !assetType.IsValid() {
		return Holdings{}, fmt.Errorf("%s %s %w", exch, assetType, asset.ErrNotSupported)
	}

	exch = strings.ToLower(exch)

	service.mu.Lock()
	defer service.mu.Unlock()
	accounts, ok := service.exchangeAccounts[exch]
	if !ok {
		return Holdings{}, fmt.Errorf("%s %w: %q", exch, ErrExchangeHoldingsNotFound, assetType)
	}

	subAccountHoldings, ok := accounts.subAccounts[*creds]
	if !ok {
		return Holdings{}, fmt.Errorf("%s %s %s %w %w", exch, creds, assetType, errNoCredentialBalances, ErrExchangeHoldingsNotFound)
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
		return Holdings{}, fmt.Errorf("%s %s %w", exch, assetType, ErrExchangeHoldingsNotFound)
	}
	return Holdings{Exchange: exch, Accounts: []SubAccount{{
		Credentials: Protected{creds: cpy},
		ID:          cpy.SubAccount,
		AssetType:   assetType,
		Currencies:  currencyBalances,
	}}}, nil
}

// GetBalance returns the internal balance for that asset item.
func GetBalance(exch, subAccount string, creds *Credentials, a asset.Item, c currency.Code) (*ProtectedBalance, error) {
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
	service.mu.Lock()
	defer service.mu.Unlock()

	accounts, ok := service.exchangeAccounts[exch]
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
// incoming should be a full update, and any missing currencies will be zeroed
func (s *Service) Save(incoming *Holdings, creds *Credentials) error {
	if incoming == nil {
		return fmt.Errorf("cannot save holdings: %w", errHoldingsIsNil)
	}

	if incoming.Exchange == "" {
		return fmt.Errorf("cannot save holdings: %w", errExchangeNameUnset)
	}

	if creds.IsEmpty() {
		return fmt.Errorf("cannot save holdings: %w", errCredentialsAreNil)
	}

	exch := strings.ToLower(incoming.Exchange)
	s.mu.Lock()
	defer s.mu.Unlock()

	accounts, ok := s.exchangeAccounts[exch]
	if !ok {
		var err error
		if accounts, err = s.initAccounts(exch); err != nil {
			return fmt.Errorf("cannot save holdings for %s %w", exch, err)
		}
	}

	subAccounts, ok := accounts.subAccounts[*creds]
	if !ok {
		subAccounts = make(map[key.SubAccountAsset]currencyBalances)
		accounts.subAccounts[*creds] = subAccounts
	}

	var errs error
	for x := range incoming.Accounts {
		if !incoming.Accounts[x].AssetType.IsValid() {
			errs = common.AppendError(errs, fmt.Errorf("cannot load sub account holdings for %s [%s] %w",
				incoming.Accounts[x].ID,
				incoming.Accounts[x].AssetType,
				asset.ErrNotSupported))
			continue
		}

		// This assignment outside of scope is designed to have minimal impact
		// on the exchange implementation UpdateAccountInfo() and portfoio
		// management.
		// TODO: Update incoming Holdings type to already be populated. (Suggestion)
		cpy := *creds
		if cpy.SubAccount == "" {
			cpy.SubAccount = incoming.Accounts[x].ID
		}
		incoming.Accounts[x].Credentials.creds = cpy

		accAsset := key.SubAccountAsset{
			SubAccount: incoming.Accounts[x].ID,
			Asset:      incoming.Accounts[x].AssetType,
		}
		assets, ok := subAccounts[accAsset]
		if !ok {
			assets = make(map[*currency.Item]*ProtectedBalance)
			accounts.subAccounts[*creds][accAsset] = assets
		}

		updated := make(map[*currency.Item]bool)
		for y := range incoming.Accounts[x].Currencies {
			accBal := &incoming.Accounts[x].Currencies[y]
			if accBal.UpdatedAt.IsZero() {
				accBal.UpdatedAt = time.Now()
			}
			bal, ok := assets[accBal.Currency.Item]
			if !ok || bal == nil {
				bal = &ProtectedBalance{}
			}
			if err := bal.load(accBal); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w for account ID %q [%s %s]: %w",
					errLoadingBalance,
					incoming.Accounts[x].ID,
					incoming.Accounts[x].AssetType,
					incoming.Accounts[x].Currencies[y].Currency,
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

		if err := s.mux.Publish(incoming.Accounts[x], accounts.ID); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("cannot publish load for %s %w", exch, err))
		}
	}

	return errs
}

// Update updates the balance for a specific exchange and credentials
func (s *Service) Update(exch string, changes []Change, creds *Credentials) error {
	if exch == "" {
		return fmt.Errorf("%w: %w", errCannotUpdateBalance, errExchangeNameUnset)
	}

	if creds.IsEmpty() {
		return fmt.Errorf("%w: %w", errCannotUpdateBalance, errCredentialsAreNil)
	}

	exch = strings.ToLower(exch)
	s.mu.Lock()
	defer s.mu.Unlock()

	accounts, ok := s.exchangeAccounts[exch]
	if !ok {
		var err error
		if accounts, err = s.initAccounts(exch); err != nil {
			return fmt.Errorf("%w for %s %w", errCannotUpdateBalance, exch, err)
		}
	}

	subAccounts, ok := accounts.subAccounts[*creds]
	if !ok {
		subAccounts = make(map[key.SubAccountAsset]currencyBalances)
		accounts.subAccounts[*creds] = subAccounts
	}

	var errs error
	for _, change := range changes {
		if !change.AssetType.IsValid() {
			errs = common.AppendError(errs, fmt.Errorf("%w for %s.%s %w",
				errCannotUpdateBalance, change.Account, change.AssetType, asset.ErrNotSupported))
			continue
		}
		if change.Balance == nil {
			errs = common.AppendError(errs, fmt.Errorf("%w for %s.%s %w",
				errCannotUpdateBalance, change.Account, change.AssetType, errBalanceIsNil))
			continue
		}

		accAsset := key.SubAccountAsset{
			SubAccount: change.Account,
			Asset:      change.AssetType,
		}
		assets, ok := subAccounts[accAsset]
		if !ok {
			assets = make(map[*currency.Item]*ProtectedBalance)
			accounts.subAccounts[*creds][accAsset] = assets
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
		if err := s.mux.Publish(change, accounts.ID); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("cannot publish update balance for %s: %w", exch, err))
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
	if b == nil {
		return nil, nil, errBalanceIsNil
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
