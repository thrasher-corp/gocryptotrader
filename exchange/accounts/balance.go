package accounts

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errBalanceCurrencyMismatch = errors.New("balance currency does not match update currency")
	errOutOfSequence           = errors.New("out of sequence")
	errUpdatedAtIsZero         = errors.New("updatedAt may not be zero")
)

// Balance contains an exchange currency balance
type Balance struct {
	Currency               currency.Code
	Total                  float64
	Hold                   float64
	Free                   float64
	AvailableWithoutBorrow float64
	Borrowed               float64
	UpdatedAt              time.Time
}

// Change defines incoming balance change on currency holdings
type Change struct {
	Account   string
	AssetType asset.Item
	Balance   Balance
}

// balance contains a balance with live updates
type balance struct {
	internal Balance
	m        sync.RWMutex
}

// CurrencyBalances provides a map of currencies to balances
type CurrencyBalances map[currency.Code]Balance

// currencyBalances provides a map of currencies to balances
type currencyBalances map[*currency.Item]*balance

// Set will set a currency balance, overwriting any previous Balance
func (c *CurrencyBalances) Set(curr currency.Code, b Balance) { //nolint:gocritic // hugeparam not relevant; we want to store a value so we'd deref anyway
	b.Currency = curr
	(*c)[curr] = b
}

// Add will add to a currency balance
func (c *CurrencyBalances) Add(curr currency.Code, b Balance) error { //nolint:gocritic // hugeparam not relevant; we want to store a value so we'd deref anyway
	if b.Currency != currency.EMPTYCODE && b.Currency != curr {
		return fmt.Errorf("%w: '%v'", errBalanceCurrencyMismatch, curr)
	}
	if e, ok := (*c)[curr]; !ok {
		b.Currency = curr
		(*c)[curr] = b
	} else {
		(*c)[curr] = e.Add(b)
	}
	return nil
}

// Balance returns a snapshot copy of the Balance
func (b *balance) Balance() Balance {
	b.m.RLock()
	defer b.m.RUnlock()
	return b.internal
}

// Add returns a new Balance adding together a and b
// UpdatedAt is the later of the two Balances
func (b *Balance) Add(a Balance) Balance { //nolint:gocritic // hugeparam not relevant; We'd need to copy it in map iterations anyway
	var u time.Time
	if a.UpdatedAt.After(b.UpdatedAt) {
		u = a.UpdatedAt
	} else {
		u = b.UpdatedAt
	}
	return Balance{
		Total:                  b.Total + a.Total,
		Hold:                   b.Hold + a.Hold,
		Free:                   b.Free + a.Free,
		AvailableWithoutBorrow: b.AvailableWithoutBorrow + a.AvailableWithoutBorrow,
		Borrowed:               b.Borrowed + a.Borrowed,
		UpdatedAt:              u,
	}
}

func (c *currencyBalances) Public() CurrencyBalances {
	n := make(CurrencyBalances, len(*c))
	for curr, bal := range *c {
		n[curr.Currency()] = bal.Balance()
	}
	return n
}

// update checks that an incoming change has a valid change, and returns if the balances were changed
// If change does not have a Currency set, the existing Currency is preserved
func (b *balance) update(change Balance) (bool, error) { //nolint:gocritic // hugeparam not relevant; We'd need to copy it later anyway
	if err := common.NilGuard(b); err != nil {
		return false, err
	}
	if change.UpdatedAt.IsZero() {
		return false, errUpdatedAtIsZero
	}
	b.m.Lock()
	defer b.m.Unlock()
	if b.internal.Currency != currency.EMPTYCODE {
		switch change.Currency {
		case b.internal.Currency:
			// All good
		case currency.EMPTYCODE:
			change.Currency = b.internal.Currency
		default:
			return false, errBalanceCurrencyMismatch
		}
	}
	if !b.internal.UpdatedAt.Before(change.UpdatedAt) {
		return false, errOutOfSequence
	}
	b.internal.UpdatedAt = change.UpdatedAt // Set just the time, and then can compare easily
	if b.internal == change {
		return false, nil
	}
	b.internal = change
	return true, nil
}

// balance rutens a balance for a currency
func (c currencyBalances) balance(curr *currency.Item) *balance {
	if _, ok := c[curr]; !ok {
		c[curr] = &balance{internal: Balance{Currency: curr.Currency()}}
	}
	return c[curr]
}
