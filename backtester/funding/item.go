package funding

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Reserve allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
func (i *Item) Reserve(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errZeroAmountReceived
	}
	if amount.GreaterThan(i.available) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Available: %v",
			errCannotAllocate,
			i.exchange,
			i.asset,
			i.currency,
			amount,
			i.available)
	}
	i.available = i.available.Sub(amount)
	i.reserved = i.reserved.Add(amount)
	return nil
}

// Release reduces the amount of funding reserved and adds any difference
// back to the available amount
func (i *Item) Release(amount, diff decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errZeroAmountReceived
	}
	if diff.IsNegative() && !i.asset.IsFutures() {
		return fmt.Errorf("%w diff %v", errNegativeAmountReceived, diff)
	}
	if amount.GreaterThan(i.reserved) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Reserved: %v",
			errCannotAllocate,
			i.exchange,
			i.asset,
			i.currency,
			amount,
			i.reserved)
	}
	i.reserved = i.reserved.Sub(amount)
	i.available = i.available.Add(diff)
	return nil
}

// IncreaseAvailable adds funding to the available amount
func (i *Item) IncreaseAvailable(amount decimal.Decimal) error {
	if amount.IsNegative() || amount.IsZero() {
		return fmt.Errorf("%w amount <= zero", errZeroAmountReceived)
	}
	i.available = i.available.Add(amount)
	return nil
}

// CanPlaceOrder checks if the item has any funds available
func (i *Item) CanPlaceOrder() bool {
	return i.available.GreaterThan(decimal.Zero)
}

// Equal checks for equality via an Item to compare to
func (i *Item) Equal(item *Item) bool {
	if i == nil && item == nil {
		return true
	}
	if item == nil || i == nil {
		return false
	}
	if i.currency == item.currency &&
		i.asset == item.asset &&
		i.exchange == item.exchange {
		if i.pairedWith == nil && item.pairedWith == nil {
			return true
		}
		if i.pairedWith == nil || item.pairedWith == nil {
			return false
		}
		if i.pairedWith.currency == item.pairedWith.currency &&
			i.pairedWith.asset == item.pairedWith.asset &&
			i.pairedWith.exchange == item.pairedWith.exchange {
			return true
		}
	}
	return false
}

// BasicEqual checks for equality via passed in values
func (i *Item) BasicEqual(exch string, a asset.Item, ccy, pairedCurrency currency.Code) bool {
	return i != nil &&
		i.exchange == exch &&
		i.asset == a &&
		i.currency.Equal(ccy) &&
		(i.pairedWith == nil ||
			(i.pairedWith != nil && i.pairedWith.currency.Equal(pairedCurrency)))
}

// MatchesCurrency checks that an item's currency is equal
func (i *Item) MatchesCurrency(c currency.Code) bool {
	return i != nil && i.currency.Equal(c)
}

// MatchesItemCurrency checks that an item's currency is equal
func (i *Item) MatchesItemCurrency(item *Item) bool {
	return i != nil && item != nil && i.currency.Equal(item.currency)
}

// MatchesExchange checks that an item's exchange is equal
func (i *Item) MatchesExchange(item *Item) bool {
	return i != nil && item != nil && i.exchange == item.exchange
}

// TakeProfit increases/decreases available funds for a futures collateral item
func (i *Item) TakeProfit(amount decimal.Decimal) error {
	if i.asset.IsFutures() && !i.isCollateral {
		return fmt.Errorf("%v %v %v %w cannot add profit to contracts", i.exchange, i.asset, i.currency, ErrNotCollateral)
	}
	i.available = i.available.Add(amount)
	return nil
}

// AddContracts allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
func (i *Item) AddContracts(amount decimal.Decimal) error {
	if !i.asset.IsFutures() {
		return fmt.Errorf("%v %v %v %w", i.exchange, i.asset, i.currency, errNotFutures)
	}
	if i.isCollateral {
		return fmt.Errorf("%v %v %v %w cannot add contracts to collateral", i.exchange, i.asset, i.currency, ErrIsCollateral)
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return errZeroAmountReceived
	}
	i.available = i.available.Add(amount)
	return nil
}

// ReduceContracts allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
func (i *Item) ReduceContracts(amount decimal.Decimal) error {
	if !i.asset.IsFutures() {
		return fmt.Errorf("%v %v %v %w", i.exchange, i.asset, i.currency, errNotFutures)
	}
	if i.isCollateral {
		return fmt.Errorf("%v %v %v %w cannot add contracts to collateral", i.exchange, i.asset, i.currency, ErrIsCollateral)
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return errZeroAmountReceived
	}
	if amount.GreaterThan(i.available) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Reserved: %v",
			errCannotAllocate,
			i.exchange,
			i.asset,
			i.currency,
			amount,
			i.reserved)
	}
	i.available = i.available.Sub(amount)
	return nil
}
