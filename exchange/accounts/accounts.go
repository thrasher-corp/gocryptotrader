package accounts

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Public errors.
var (
	ErrNoBalances    = errors.New("no balances found")
	ErrNoSubAccounts = errors.New("no subAccounts found")
)

var (
	errCredentialsEmpty = errors.New("no credentials provided")
	errUpdatingBalance  = errors.New("error updating balance")
	errPublish          = errors.New("error publishing account changes")
)

// Accounts holds a stream ID and a map to the exchange holdings.
type Accounts struct {
	Exchange    exchange
	routingID   uuid.UUID // GCT internal routing mux id
	subAccounts credSubAccounts
	mu          sync.RWMutex
	mux         *dispatch.Mux
}

type (
	credSubAccounts map[Credentials]subAccounts
	subAccounts     map[key.SubAccountAsset]currencyBalances
)

// SubAccount contains an account for an asset type and its balances.
// The SubAccount may be the main account depending on exchange structure.
type SubAccount struct {
	ID        string
	AssetType asset.Item
	Balances  CurrencyBalances
}

// SubAccounts contains a list of public SubAccounts.
type SubAccounts []*SubAccount

// MustNewAccounts returns an initialised Accounts store for use in isolation from a global exchange accounts store.
// mux is set to the global dispatch.Dispatcher.
// Any errors in mux ID generation will panic, so users should balance risk vs utility accordingly depending on use-case.
func MustNewAccounts(e exchange) *Accounts {
	a, err := NewAccounts(e, dispatch.GetNewMux(nil))
	if err != nil {
		panic(err)
	}
	return a
}

// NewAccounts returns an initialised Accounts store for use in isolation from a global exchange accounts store.
func NewAccounts(e exchange, mux *dispatch.Mux) (*Accounts, error) {
	if err := common.NilGuard(e); err != nil {
		return nil, err
	}
	id, err := mux.GetID()
	if err != nil {
		return nil, err
	}
	return &Accounts{
		Exchange:    e,
		subAccounts: make(credSubAccounts),
		routingID:   id,
		mux:         mux,
	}, nil
}

// NewSubAccount returns a new SubAccount.
// id may be empty.
func NewSubAccount(a asset.Item, id string) *SubAccount {
	return &SubAccount{
		AssetType: a,
		ID:        id,
		Balances:  CurrencyBalances{},
	}
}

// Subscribe subscribes to your exchange accounts.
func (a *Accounts) Subscribe() (dispatch.Pipe, error) {
	if err := common.NilGuard(a); err != nil {
		return dispatch.Pipe{}, err
	}
	return a.mux.Subscribe(a.routingID)
}

// CurrencyBalances returns the balances for the Accounts grouped by currency.
// If creds is nil, all credential SubAccounts will be collated.
// If assetType is asset.All, all assets will be collated.
func (a *Accounts) CurrencyBalances(creds *Credentials, assetType asset.Item) (CurrencyBalances, error) {
	if err := common.NilGuard(a); err != nil {
		return nil, err
	}
	if !assetType.IsValid() && assetType != asset.All {
		return nil, fmt.Errorf("%s %s %w", a.Exchange.GetName(), assetType, asset.ErrNotSupported)
	}

	currs := CurrencyBalances{}

	a.mu.RLock()
	defer a.mu.RUnlock()

	for credsKey, subAccountsForCreds := range a.subAccounts {
		if !creds.IsEmpty() && *creds != credsKey {
			continue
		}
		for subAcctKey, balances := range subAccountsForCreds {
			if assetType != asset.All && assetType != subAcctKey.Asset {
				continue
			}
			for curr, bal := range balances {
				if err := currs.Add(curr.Currency(), bal.Balance()); err != nil {
					return nil, err // Should be impossible, so return immediately
				}
			}
		}
	}
	if len(currs) == 0 {
		return nil, fmt.Errorf("%w for %s credentials %s asset %s", ErrNoBalances, a.Exchange.GetName(), creds, assetType)
	}
	return currs, nil
}

// SubAccounts returns the public SubAccounts and their balances.
// If creds is nil, all credential SubAccounts will be returned.
// If assetType is asset.All, all assets will be returned.
func (a *Accounts) SubAccounts(creds *Credentials, assetType asset.Item) (SubAccounts, error) {
	if err := common.NilGuard(a); err != nil {
		return nil, err
	}

	if !assetType.IsValid() && assetType != asset.All {
		return nil, fmt.Errorf("%s %s %w", a.Exchange.GetName(), assetType, asset.ErrNotSupported)
	}

	var subAccts SubAccounts

	a.mu.RLock()
	defer a.mu.RUnlock()

	for credsKey, subAccountsForCreds := range a.subAccounts {
		if !creds.IsEmpty() && *creds != credsKey {
			continue
		}
		for subAcctKey, balances := range subAccountsForCreds {
			if assetType != asset.All && assetType != subAcctKey.Asset {
				continue
			}
			subAccts = append(subAccts, &SubAccount{
				ID:        subAcctKey.SubAccount,
				AssetType: subAcctKey.Asset,
				Balances:  balances.Public(),
			})
		}
	}

	if len(subAccts) == 0 {
		return nil, fmt.Errorf("%w for %s credentials %s asset %s", ErrNoSubAccounts, a.Exchange.GetName(), creds, assetType)
	}
	return subAccts, nil
}

// GetBalance returns a copy of the balance for that asset item.
func (a *Accounts) GetBalance(subAccount string, creds *Credentials, aType asset.Item, c currency.Code) (Balance, error) {
	if err := common.NilGuard(a); err != nil {
		return Balance{}, err
	}
	if !aType.IsValid() {
		return Balance{}, fmt.Errorf("cannot get balance: %w: %q", asset.ErrNotSupported, aType)
	}

	if creds.IsEmpty() {
		return Balance{}, fmt.Errorf("cannot get balance: %w", errCredentialsEmpty)
	}

	if c.IsEmpty() {
		return Balance{}, fmt.Errorf("cannot get balance: %w", currency.ErrCurrencyCodeEmpty)
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	subAccts, ok := a.subAccounts[*creds]
	if !ok {
		return Balance{}, fmt.Errorf("%w for %s", ErrNoBalances, creds)
	}

	assets, ok := subAccts[key.SubAccountAsset{
		SubAccount: subAccount,
		Asset:      aType,
	}]
	if !ok {
		return Balance{}, fmt.Errorf("%w for %s SubAccount %q %s", ErrNoBalances, a.Exchange.GetName(), subAccount, aType)
	}
	b, ok := assets[c.Item]
	if !ok {
		return Balance{}, fmt.Errorf("%w for %s SubAccount %q %s %s", ErrNoBalances, a.Exchange.GetName(), subAccount, aType, c)
	}
	return b.Balance(), nil
}

// Save updates the account balances.
// If isSnapshot is true any missing currencies will be removed.
// Credentials will be retrieved from ctx, Use DeployCredentialsToContext.
// Changes to balances are published individually.
func (a *Accounts) Save(ctx context.Context, subAccts SubAccounts, isSnapshot bool) error {
	if err := common.NilGuard(a); err != nil {
		return fmt.Errorf("cannot save holdings: %w", err)
	}
	if err := common.NilGuard(a.subAccounts); err != nil {
		return fmt.Errorf("cannot save holdings: %w", err)
	}

	creds, err := a.Exchange.GetCredentials(ctx)
	if err != nil {
		return err
	}
	if creds.IsEmpty() {
		return fmt.Errorf("%w: %w", errUpdatingBalance, errCredentialsEmpty)
	}

	var errs error

	a.mu.Lock()
	defer a.mu.Unlock()

	for _, s := range subAccts {
		if !s.AssetType.IsValid() {
			errs = common.AppendError(errs, fmt.Errorf("error loading %s[%s] SubAccount holdings: %w", s.ID, s.AssetType, asset.ErrNotSupported))
			continue
		}

		accBalances := a.currencyBalances(creds, s.ID, s.AssetType)

		updated := false
		missing := maps.Clone(accBalances)
		for curr, newBal := range s.Balances {
			delete(missing, curr.Item)
			if newBal.UpdatedAt.IsZero() {
				newBal.UpdatedAt = time.Now()
			}
			if newBal.Currency.IsEmpty() {
				newBal.Currency = curr
			}
			s.Balances[curr] = newBal
			if u, err := accBalances.balance(curr.Item).update(newBal); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w for account ID %q [%s %s]: %w", errUpdatingBalance, s.ID, s.AssetType, curr, err))
			} else if u {
				updated = true
			}
		}
		if isSnapshot {
			for cur := range missing {
				delete(accBalances, cur)
				updated = true
			}
		}
		if updated {
			if err := a.mux.Publish(s, a.routingID); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w for %s %w", errPublish, a.Exchange, err))
			}
		}
	}

	return errs
}

// Merge adds CurrencyBalances in s to the SubAccount in l with a matching AssetType and ID.
// If no SubAccount matches, s is appended.
// Duplicate Currency Balances are added together.
func (l SubAccounts) Merge(s *SubAccount) SubAccounts {
	if err := common.NilGuard(s); err != nil {
		return nil
	}
	i := slices.IndexFunc(l, func(b *SubAccount) bool { return s.AssetType == b.AssetType && s.ID == b.ID })
	if i == -1 {
		return append(l, s)
	}
	for curr, newBal := range s.Balances {
		l[i].Balances[curr] = newBal.Add(l[i].Balances[curr])
	}
	return l
}

// currencyBalances returns a currencyBalances entry for Credentials, SubAccount and asset.
// No nilguard protection provided, since this is a private function.
func (a *Accounts) currencyBalances(c *Credentials, subAcct string, aType asset.Item) currencyBalances {
	k := key.SubAccountAsset{SubAccount: subAcct, Asset: aType}
	if _, ok := a.subAccounts[*c]; !ok {
		a.subAccounts[*c] = make(subAccounts)
	}
	if _, ok := a.subAccounts[*c][k]; !ok {
		a.subAccounts[*c][k] = make(currencyBalances)
	}
	return a.subAccounts[*c][k]
}
