package account

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func init() {
	service.exchangeAccounts = make(map[string]*Accounts)
	service.mux = dispatch.GetNewMux(nil)
}

var (
	errHoldingsIsNil                = errors.New("holdings cannot be nil")
	errExchangeNameUnset            = errors.New("exchange name unset")
	errExchangeHoldingsNotFound     = errors.New("exchange holdings not found")
	errAssetHoldingsNotFound        = errors.New("asset holdings not found")
	errExchangeAccountsNotFound     = errors.New("exchange accounts not found")
	errNoExchangeSubAccountBalances = errors.New("no exchange sub account balances")
	errNoBalanceFound               = errors.New("no balance found")
	errBalanceIsNil                 = errors.New("balance is nil")
	errNoCredentialBalances         = errors.New("no balances associated with credentials")
	errCredentialsAreNil            = errors.New("credentials are nil")
)

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
		return dispatch.Pipe{}, fmt.Errorf("cannot subscribe %s %w",
			exchange,
			errExchangeAccountsNotFound)
	}
	return service.mux.Subscribe(accounts.ID)
}

// Process processes new account holdings updates
func Process(h *Holdings, c *Credentials) error {
	return service.Update(h, c)
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
		return Holdings{}, fmt.Errorf("%s %s %w", exch, assetType, errExchangeHoldingsNotFound)
	}

	var accountsHoldings []SubAccount
	subAccountHoldings, ok := accounts.SubAccounts[*creds]
	if !ok {
		return Holdings{}, fmt.Errorf("%s %s %s %w",
			exch,
			creds,
			assetType,
			errNoCredentialBalances)
	}

	for subAccount, assetHoldings := range subAccountHoldings {
		for ai, currencyHoldings := range assetHoldings {
			if ai != assetType {
				continue
			}
			var currencyBalances = make([]Balance, len(currencyHoldings))
			target := 0
			for item, balance := range currencyHoldings {
				balance.m.Lock()
				currencyBalances[target] = Balance{
					Currency:               currency.Code{Item: item, UpperCase: true},
					Total:                  balance.total,
					Hold:                   balance.hold,
					Free:                   balance.free,
					AvailableWithoutBorrow: balance.availableWithoutBorrow,
					Borrowed:               balance.borrowed,
				}
				balance.m.Unlock()
				target++
			}

			if len(currencyBalances) == 0 {
				continue
			}

			cpy := *creds
			if cpy.SubAccount == "" {
				cpy.SubAccount = subAccount
			}

			accountsHoldings = append(accountsHoldings, SubAccount{
				Credentials: Protected{creds: cpy},
				ID:          subAccount,
				AssetType:   ai,
				Currencies:  currencyBalances,
			})
			break
		}
	}

	if len(accountsHoldings) == 0 {
		return Holdings{}, fmt.Errorf("%s %s %w",
			exch,
			assetType,
			errAssetHoldingsNotFound)
	}
	return Holdings{Exchange: exch, Accounts: accountsHoldings}, nil
}

// GetBalance returns the internal balance for that asset item.
func GetBalance(exch, subAccount string, creds *Credentials, ai asset.Item, c currency.Code) (*ProtectedBalance, error) {
	if exch == "" {
		return nil, fmt.Errorf("cannot get balance: %w", errExchangeNameUnset)
	}

	if !ai.IsValid() {
		return nil, fmt.Errorf("cannot get balance: %s %w", ai, asset.ErrNotSupported)
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
		return nil, fmt.Errorf("%s %w", exch, errExchangeHoldingsNotFound)
	}

	subAccounts, ok := accounts.SubAccounts[*creds]
	if !ok {
		return nil, fmt.Errorf("%s %s %w",
			exch, creds, errNoCredentialBalances)
	}

	assetBalances, ok := subAccounts[subAccount]
	if !ok {
		return nil, fmt.Errorf("%s %s %w",
			exch, subAccount, errNoExchangeSubAccountBalances)
	}

	currencyBalances, ok := assetBalances[ai]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w",
			exch, subAccount, ai, errAssetHoldingsNotFound)
	}

	bal, ok := currencyBalances[c.Item]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %s %w",
			exch, subAccount, ai, c, errNoBalanceFound)
	}
	return bal, nil
}

// Update updates holdings with new account info
func (s *Service) Update(incoming *Holdings, creds *Credentials) error {
	if incoming == nil {
		return fmt.Errorf("cannot update holdings: %w", errHoldingsIsNil)
	}

	if incoming.Exchange == "" {
		return fmt.Errorf("cannot update holdings: %w", errExchangeNameUnset)
	}

	if creds.IsEmpty() {
		return fmt.Errorf("cannot update holdings: %w", errCredentialsAreNil)
	}

	exch := strings.ToLower(incoming.Exchange)
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, ok := s.exchangeAccounts[exch]
	if !ok {
		id, err := s.mux.GetID()
		if err != nil {
			return err
		}
		accounts = &Accounts{
			ID:          id,
			SubAccounts: make(map[Credentials]map[string]map[asset.Item]map[*currency.Item]*ProtectedBalance),
		}
		s.exchangeAccounts[exch] = accounts
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

		var subAccounts map[string]map[asset.Item]map[*currency.Item]*ProtectedBalance
		subAccounts, ok = accounts.SubAccounts[*creds]
		if !ok {
			subAccounts = make(map[string]map[asset.Item]map[*currency.Item]*ProtectedBalance)
			accounts.SubAccounts[*creds] = subAccounts
		}

		var accountAssets map[asset.Item]map[*currency.Item]*ProtectedBalance
		accountAssets, ok = subAccounts[incoming.Accounts[x].ID]
		if !ok {
			accountAssets = make(map[asset.Item]map[*currency.Item]*ProtectedBalance)
			// Note: Sub accounts are case sensitive and an account "name" is
			// different to account "naMe".
			subAccounts[incoming.Accounts[x].ID] = accountAssets
		}

		var currencyBalances map[*currency.Item]*ProtectedBalance
		currencyBalances, ok = accountAssets[incoming.Accounts[x].AssetType]
		if !ok {
			currencyBalances = make(map[*currency.Item]*ProtectedBalance)
			accountAssets[incoming.Accounts[x].AssetType] = currencyBalances
		}

		for y := range incoming.Accounts[x].Currencies {
			bal := currencyBalances[incoming.Accounts[x].Currencies[y].Currency.Item]
			if bal == nil {
				bal = &ProtectedBalance{}
				currencyBalances[incoming.Accounts[x].Currencies[y].Currency.Item] = bal
			}
			bal.load(incoming.Accounts[x].Currencies[y])
		}
	}

	err := s.mux.Publish(incoming, accounts.ID)
	if err != nil {
		return err
	}

	return errs
}

// load checks to see if there is a change from incoming balance, if there is a
// change it will change then alert external routines.
func (b *ProtectedBalance) load(change Balance) {
	b.m.Lock()
	defer b.m.Unlock()
	if b.total == change.Total &&
		b.hold == change.Hold &&
		b.free == change.Free &&
		b.availableWithoutBorrow == change.AvailableWithoutBorrow &&
		b.borrowed == change.Borrowed {
		return
	}
	b.total = change.Total
	b.hold = change.Hold
	b.free = change.Free
	b.availableWithoutBorrow = change.AvailableWithoutBorrow
	b.borrowed = change.Borrowed
	b.notice.Alert()
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
