package accounts

import (
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

	c.Set(currency.BTC, Balance{Total: 4.2})
	require.Contains(t, c, currency.BTC, "must add an entry to an uninitialised CurrencyBalances")
	assert.Equal(t, currency.BTC, c[currency.BTC].Currency, "should set the Currency")

	c.Set(currency.LTC, Balance{Currency: currency.ETH, Total: 52.4})
	require.Contains(t, c, currency.LTC, "must add an entry to an existing CurrencyBalances")
	assert.Equal(t, currency.LTC, c[currency.LTC].Currency, "should overwrite Currency")
}

// TestCurrencyBalancesAdd exercises CurrencyBalances.Add
func TestCurrencyBalancesAdd(t *testing.T) {
	t.Parallel()

	c := CurrencyBalances{}
	assert.ErrorIs(t, c.Add(currency.EMPTYCODE, Balance{}), currency.ErrCurrencyCodeEmpty)

	err := c.Add(currency.BTC, Balance{Total: 4.2})
	require.NoError(t, err)

	require.Contains(t, c, currency.BTC, "must add an entry to an uninitialised CurrencyBalances")
	assert.Equal(t, currency.BTC, c[currency.BTC].Currency, "should set the Currency")
	assert.Equal(t, 4.2, c[currency.BTC].Total, "should initialise the Total")

	err = c.Add(currency.BTC, Balance{Total: 1.3, Hold: 2.4})
	require.NoError(t, err)
	assert.Equal(t, 5.5, c[currency.BTC].Total, "should add to existing Total")
	assert.Equal(t, 2.4, c[currency.BTC].Hold, "should initialise Hold")

	err = c.Add(currency.LTC, Balance{Currency: currency.LTC, Total: 14.3})
	require.NoError(t, err)
	require.Contains(t, c, currency.LTC, "must add an entry to an existing CurrencyBalances")
	assert.Equal(t, 14.3, c[currency.LTC].Total, "should add when Balance.Currency is equal")

	err = c.Add(currency.ETH, Balance{Currency: currency.LTC, Total: 14.2})
	assert.ErrorIs(t, err, errBalanceCurrencyMismatch)
}

// TestCurrencyBalancesPublic exercises currencyBalances.Public
func TestCurrencyBalancesPublic(t *testing.T) {
	t.Parallel()
	b := (&currencyBalances{
		currency.BTC.Item: &balance{internal: Balance{Total: 4.2}},
		currency.LTC.Item: &balance{internal: Balance{Total: 1.7}},
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
	b := c.balance(currency.BTC.Item)
	require.NotNil(t, b)
	assert.Same(t, c[currency.BTC.Item], b, "should make and return the same entry")
	assert.Same(t, b, c.balance(currency.BTC.Item), "should make and return the same entry")
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
	assert.Equal(t, 4.2, b.Total, "should initialise Total")
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

	_, err = b.update(Balance{UpdatedAt: n.Add(-time.Millisecond)})
	assert.ErrorIs(t, err, errOutOfSequence, "should error when time out of sequence")

	u, err := b.update(Balance{UpdatedAt: n, Total: 5.1})
	require.NoError(t, err, "must not error when time is the same instant and currency is empty")
	assert.Equal(t, 5.1, b.internal.Total, "Total should be correct")
	assert.True(t, u, "should return updated")

	n = time.Now()
	u, err = b.update(Balance{UpdatedAt: n, Total: 5.1})
	require.NoError(t, err)
	assert.Equal(t, n, b.internal.UpdatedAt, "should update UpdatedAt")
	assert.False(t, u, "should not return updated when nothing really changed")

	n = time.Now()
	u, err = b.update(Balance{UpdatedAt: n, Currency: currency.LTC, Total: 5.1})
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
