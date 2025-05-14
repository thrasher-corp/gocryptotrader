package accounts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	t.Parallel()
	s := NewStore()
	require.NotNil(t, s, "NewStore must return a store")
	require.NotNil(t, s.mux, "NewStore must set mux")
	require.NotNil(t, s.exchangeAccounts, "NewStore must set exchangeAccounts")
}

func TestGetStore(t *testing.T) {
	t.Parallel()
	// Initialize global in case of -count=N+; No other tests should be relying on it
	global.Store(nil)
	s := GetStore()
	require.NotNil(t, s)
	require.Same(t, global.Load(), s, "GetStore must initialize the store")
	require.Same(t, s, GetStore(), "GetStore must return the global store on second call")
}

func TestGetExchangeAccounts(t *testing.T) {
	t.Parallel()
	s := NewStore()
	m := &mockEx{"mocky"}
	a := &Accounts{}
	s.exchangeAccounts[m] = a
	got, err := s.GetExchangeAccounts(m)
	require.NoError(t, err)
	assert.Same(t, a, got, "Should retrieve same existing Accounts")

	m = &mockEx{"new"}
	got, err = s.GetExchangeAccounts(m)
	require.NoError(t, err)
	assert.Same(t, s.exchangeAccounts[m], got, "Should retrieve the new exchange")

	w := &mockExBase{m}
	got, err = s.GetExchangeAccounts(w)
	require.NoError(t, err)
	require.NotNil(t, got)
}
