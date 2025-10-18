package accounts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	// Initialise global in case of -count=N+; No other tests should be relying on it
	global.Store(nil)
	s := GetStore()
	require.NotNil(t, s, "GetStore must return a Store")
	require.Same(t, global.Load(), s, "GetStore must set the global store")
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
	assert.NotNil(t, got)

	_, err = s.GetExchangeAccounts(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error correctly on nil exchange")
}

type mockEx struct {
	name string
}

func (m *mockEx) GetName() string {
	return "mocky"
}

func (m *mockEx) GetCredentials(ctx context.Context) (*Credentials, error) {
	if value := ctx.Value(ContextCredentialsFlag); value != nil {
		if s, ok := value.(*ContextCredentialsStore); ok {
			return s.Get(), nil
		}
		return nil, common.GetTypeAssertError("*accounts.ContextCredentialsStore", value)
	}
	return nil, nil
}

type mockExBase struct {
	base exchange
}

func (m *mockExBase) GetBase() exchange {
	return m.base
}

func (m *mockExBase) GetCredentials(ctx context.Context) (*Credentials, error) {
	return m.base.GetCredentials(ctx)
}

func (m *mockExBase) GetName() string {
	return m.base.GetName()
}
