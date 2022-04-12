package account

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func init() {
	service.exchangeAccounts = make(map[string]*Accounts)
	service.mux = dispatch.GetNewMux()
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
)

// CollectBalances converts a map of sub-account balances into a slice
func CollectBalances(accountBalances map[string][]Balance, assetType asset.Item) (accounts []SubAccount, err error) {
	if accountBalances == nil {
		return nil, errAccountBalancesIsNil
	}

	if !assetType.IsValid() {
		return nil, fmt.Errorf("%s, %w", assetType, asset.ErrNotSupported)
	}

	accounts = make([]SubAccount, len(accountBalances))
	i := 0
	for accountID, balances := range accountBalances {
		accounts[i] = SubAccount{
			ID:         accountID,
			AssetType:  assetType,
			Currencies: balances,
		}
		i++
	}
	return
}

// SubscribeToExchangeAccount subscribes to your exchange account
func SubscribeToExchangeAccount(exchange string) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.m.Lock()
	defer service.m.Unlock()
	accounts, ok := service.exchangeAccounts[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("cannot subscribe %s %w",
			exchange,
			errExchangeAccountsNotFound)
	}
	return service.mux.Subscribe(accounts.ID)
}

// Process processes new account holdings updates
func Process(h *Holdings) error {
	if h == nil {
		return errHoldingsIsNil
	}

	if h.Exchange == "" {
		return errExchangeNameUnset
	}

	return service.Update(h)
}

// GetHoldings returns full holdings for an exchange
func GetHoldings(exch string, assetType asset.Item) (Holdings, error) {
	if exch == "" {
		return Holdings{}, errExchangeNameUnset
	}

	if !assetType.IsValid() {
		return Holdings{}, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}

	exch = strings.ToLower(exch)

	service.m.Lock()
	defer service.m.Unlock()
	accounts, ok := service.exchangeAccounts[exch]
	if !ok {
		return Holdings{}, errExchangeHoldingsNotFound
	}

	var accountsHoldings []SubAccount
	for subAccount, assetHoldings := range accounts.SubAccounts {
		for ai, currencyHoldings := range assetHoldings {
			if ai != assetType {
				continue
			}
			var currencyBalances = make([]Balance, len(currencyHoldings))
			target := 0
			for item, balance := range currencyHoldings {
				balance.m.Lock()
				currencyBalances[target] = Balance{
					CurrencyName:           currency.Code{Item: item, UpperCase: true},
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

			accountsHoldings = append(accountsHoldings, SubAccount{
				ID:         subAccount,
				AssetType:  ai,
				Currencies: currencyBalances,
			})
			break // Don't continue to iterate through the other assets.
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
func GetBalance(exch, subAccount string, ai asset.Item, c currency.Code) (*BalanceInternal, error) {
	if exch == "" {
		return nil, errExchangeNameUnset
	}

	if !ai.IsValid() {
		return nil, fmt.Errorf("%s %w", ai, asset.ErrNotSupported)
	}

	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}

	exch = strings.ToLower(exch)
	subAccount = strings.ToLower(subAccount)

	service.m.Lock()
	defer service.m.Unlock()

	accounts, ok := service.exchangeAccounts[exch]
	if !ok {
		return nil, errExchangeHoldingsNotFound
	}

	assetBalances, ok := accounts.SubAccounts[subAccount]
	if !ok {
		return nil, errNoExchangeSubAccountBalances
	}

	currencyBalances, ok := assetBalances[ai]
	if !ok {
		return nil, errAssetHoldingsNotFound
	}

	bal, ok := currencyBalances[c.Item]
	if !ok {
		return nil, errNoBalanceFound
	}
	return bal, nil
}

// Update updates holdings with new account info
func (s *Service) Update(a *Holdings) error {
	exch := strings.ToLower(a.Exchange)
	s.m.Lock()
	defer s.m.Unlock()
	accounts, ok := s.exchangeAccounts[exch]
	if !ok {
		id, err := s.mux.GetID()
		if err != nil {
			return err
		}
		accounts = &Accounts{
			ID:          id,
			SubAccounts: make(map[string]map[asset.Item]map[*currency.Item]*BalanceInternal),
		}
		s.exchangeAccounts[exch] = accounts
	}

	for x := range a.Accounts {
		lowerSA := strings.ToLower(a.Accounts[x].ID)
		var accountAssets map[asset.Item]map[*currency.Item]*BalanceInternal
		accountAssets, ok = accounts.SubAccounts[lowerSA]
		if !ok {
			accountAssets = make(map[asset.Item]map[*currency.Item]*BalanceInternal)
			accounts.SubAccounts[lowerSA] = accountAssets
		}

		if !a.Accounts[x].AssetType.IsValid() {
			return fmt.Errorf("cannot load sub account holdings for %s [%s] %w",
				a.Accounts[x].ID,
				a.Accounts[x].AssetType,
				asset.ErrNotSupported)
		}

		var currencyBalances map[*currency.Item]*BalanceInternal
		currencyBalances, ok = accountAssets[a.Accounts[x].AssetType]
		if !ok {
			currencyBalances = make(map[*currency.Item]*BalanceInternal)
			accountAssets[a.Accounts[x].AssetType] = currencyBalances
		}

		for y := range a.Accounts[x].Currencies {
			bal := currencyBalances[a.Accounts[x].Currencies[y].CurrencyName.Item]
			if bal == nil {
				bal = &BalanceInternal{}
				currencyBalances[a.Accounts[x].Currencies[y].CurrencyName.Item] = bal
			}
			bal.load(a.Accounts[x].Currencies[y])
		}
	}
	return s.mux.Publish([]uuid.UUID{accounts.ID}, a)
}

// load checks to see if there is a change from incoming balance, if there is a
// change it will change then alert external routines.
func (b *BalanceInternal) load(change Balance) {
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
// to achieve a state change in the time specified.
func (b *BalanceInternal) Wait(maxWait time.Duration) (wait <-chan bool, cancel chan<- struct{}, err error) {
	if b == nil {
		return nil, nil, errBalanceIsNil
	}

	ch := make(chan struct{})

	if maxWait > 0 {
		go func(ch chan<- struct{}, until time.Duration) {
			time.Sleep(until)
			select {
			case ch <- struct{}{}:
			default:
			}
		}(ch, maxWait)
	}

	return b.notice.Wait(ch), ch, nil
}

// GetFree returns the current free balance for the exchange
func (b *BalanceInternal) GetFree() float64 {
	if b == nil {
		return 0
	}
	b.m.Lock()
	defer b.m.Unlock()
	return b.free
}
