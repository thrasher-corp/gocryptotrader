package accounts

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// TestCurrencyBalancesSet exercises CurrencyBalances.Set
func TestCurrencyBalancesSet(t *testing.T) {
	t.Parallel()

	c := CurrencyBalances{}

	c.Set("BTC", Balance{Total: 4.2})
	assert.Contains(t, c, currency.BTC, "string for courrency should be converted to a currency.Code")
	assert.Equal(t, currency.BTC, c[currency.BTC].Currency, "should set the Currency")

	c.Set(currency.LTC, Balance{Currency: currency.ETH, Total: 52.4})
	assert.Contains(t, c, currency.LTC, "currency.Code for currency should just work")
	assert.Equal(t, currency.LTC, c[currency.LTC].Currency, "should overwrite Currency")

	expErr := fmt.Sprintf("%s: '<nil>'", currency.ErrCurrencyNotSupported)
	assert.PanicsWithError(t, expErr, func() { c.Set(nil, Balance{}) }, "should panic when currency is not a currency.Code or string")
}

// TestCurrencyBalancesAdd exercises CurrencyBalances.Add
func TestCurrencyBalancesAdd(t *testing.T) {
	t.Parallel()

	c := CurrencyBalances{}
	c.Add("BTC", Balance{Total: 4.2})

	assert.Contains(t, c, currency.BTC, "string for courrency should be converted to a currency.Code")
	assert.Equal(t, currency.BTC, c[currency.BTC].Currency, "should set the Currency")
	assert.Equal(t, 4.2, c[currency.BTC].Total, "should initialise the Total")

	c.Add(currency.BTC, Balance{Total: 1.3, Hold: 2.4})
	assert.Equal(t, 5.5, c[currency.BTC].Total, "should add to existing Total")
	assert.Equal(t, 2.4, c[currency.BTC].Hold, "should initialise Hold")

	c.Add(currency.LTC, Balance{Currency: currency.LTC, Total: 14.3})
	assert.Equal(t, 14.3, c[currency.LTC].Total, "should add when Balance.Currency is equal")

	expErr := fmt.Sprintf("%s: 'ETH'", errBalanceCurrencyMismatch)
	assert.PanicsWithError(t, expErr, func() { c.Add(currency.ETH, Balance{Currency: currency.LTC, Total: 14.2}) }, "should panic when currency does not match")

	assert.Panics(t, func() { c.Add(nil, Balance{}) }, "should panic when currency is not a currency.Code or string")
}

// TestCurrencyBalancesPublic exercises currencyBalances.Public
func TestCurrencyBalancesPublic(t *testing.T) {
	t.Parallel()
	b := (&currencyBalances{
		currency.BTC: &balance{internal: Balance{Total: 4.2}},
		currency.LTC: &balance{internal: Balance{Total: 1.7}},
	}).Public()
	require.Equal(t, 2, len(b), "Pulbic must return the correct number of Balances")
	require.Contains(t, b, currency.BTC)
	require.Contains(t, b, currency.LTC)
	assert.Equal(t, 4.2, b[currency.BTC].Total)
	assert.Equal(t, 1.7, b[currency.LTC].Total)
}

// TestCurrencyBalancesBalance exercises currencyBalances.balance
func TestCurrencyBalancesBalance(t *testing.T) {
	t.Parallel()
	c := currencyBalances{}
	b := c.balance(currency.BTC)
	require.NotNil(t, b)
	assert.Same(t, c[currency.BTC], b, "should make and return the same entry")
	assert.Same(t, b, c.balance(currency.BTC), "should make and return the same entry")
}

// TestBalanceBalance exercises balance.Balance
func TestBalanceBalance(t *testing.T) {
	t.Parallel()
	b := &balance{internal: Balance{Currency: currency.BTC}}
	i := b.Balance()
	assert.Equal(t, b.internal, i)
}

// TestBalanceAdd exercises Balance.Add
func TestBalanceAdd(t *testing.T) {
	t.Parallel()
	n1 := time.Now()
	n2 := n1.Add(-2 * time.Minute)
	b := new(Balance).Add(Balance{Total: 4.2, UpdatedAt: n2})
	assert.Equal(t, 4.2, b.Total, "should initialize Total")
	assert.Equal(t, n2, b.UpdatedAt, "should set UpdatedAt")
	b = b.Add(Balance{Total: 1.3, Hold: 3.0, UpdatedAt: n1})
	assert.Equal(t, 5.5, b.Total, "should add to Total")
	assert.Equal(t, 3.0, b.Hold, "should initialise Hold")
	assert.Equal(t, n1, b.UpdatedAt, "should set UpdatedAt")
	b = b.Add(Balance{Total: 2.2, Hold: 4.0, UpdatedAt: n1.Add(-time.Minute)})
	assert.Equal(t, 7.7, b.Total, "should add to Total")
	assert.Equal(t, 7.0, b.Hold, "should add to Hold")
	assert.Equal(t, n1, b.UpdatedAt, "should keep newer UpdatedAt")
}

// TestBalanceUpdate exercises balance.update
func TestBalanceUpdate(t *testing.T) {
	t.Parallel()

	_, err := (*balance)(nil).update(Balance{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	n := time.Now()
	b := &balance{internal: Balance{
		Currency:  currency.LTC,
		Total:     4.2,
		UpdatedAt: n,
	}}

	_, err = b.update(Balance{})
	require.ErrorIs(t, err, errUpdatedAtIsZero)

	_, err = b.update(Balance{UpdatedAt: n, Currency: currency.ETH})
	assert.ErrorIs(t, err, errBalanceCurrencyMismatch)

	_, err = b.update(Balance{UpdatedAt: n})
	assert.ErrorIs(t, err, errOutOfSequence, "should error when time out of sequence")

	n = time.Now()
	u, err := b.update(Balance{UpdatedAt: n, Total: 4.2})
	require.NoError(t, err, "msut not error when Currency is empty")
	assert.Equal(t, n, b.internal.UpdatedAt, "should update UpdatedAt")
	assert.False(t, u, "should not be updated when nothing really changed")

	n = time.Now()
	u, err = b.update(Balance{UpdatedAt: n, Currency: currency.LTC, Total: 4.2})
	require.NoError(t, err, "must not error when Currency matches")
	assert.Equal(t, n, b.internal.UpdatedAt, "should update UpdatedAt")
	assert.False(t, u, "should return not updated when only time changed")

	n = time.Now()
	u, err = b.update(Balance{UpdatedAt: n, Currency: currency.LTC, Total: 4.4})
	require.NoError(t, err, "must not error when Currency matches")
	assert.Equal(t, n, b.internal.UpdatedAt, "should update UpdatedAt")
	assert.Equal(t, 4.4, b.internal.Total, "should update Total")
	assert.True(t, u, "should return updated")
}

func TestCurrencyBalancesClone(t *testing.T) {
	t.Parallel()
	b := CurrencyBalances{currency.BTC: {Total: 1}, currency.LTC: {Total: 2}}
	c := b.clone()
	require.Equal(t, b, c)
	c[currency.BTC] = Balance{Total: 3}
	assert.NotEqual(t, b, c, "should not be equal after modification")
}
