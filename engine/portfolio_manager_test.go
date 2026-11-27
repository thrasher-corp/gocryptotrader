package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

func TestSetupPortfolioManager(t *testing.T) {
	_, err := setupPortfolioManager(nil, 0, nil)
	assert.ErrorIs(t, err, errNilExchangeManager)

	m, err := setupPortfolioManager(NewExchangeManager(), 0, nil)
	assert.NoError(t, err)

	if m == nil {
		t.Error("expected manager")
	}
}

func TestIsPortfolioManagerRunning(t *testing.T) {
	var m *portfolioManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	m, err := setupPortfolioManager(NewExchangeManager(), 0, nil)
	assert.NoError(t, err)

	if m.IsRunning() {
		t.Error("expected false")
	}
	var wg sync.WaitGroup
	err = m.Start(t.Context(), &wg)
	if err != nil {
		t.Error(err)
	}
	if !m.IsRunning() {
		t.Error("expected true")
	}
}

func TestPortfolioManagerStart(t *testing.T) {
	var m *portfolioManager
	var wg sync.WaitGroup
	err := m.Start(t.Context(), nil)
	assert.ErrorIs(t, err, ErrNilSubsystem)

	m, err = setupPortfolioManager(NewExchangeManager(), 0, nil)
	assert.NoError(t, err)

	err = m.Start(t.Context(), nil)
	assert.ErrorIs(t, err, errNilWaitGroup)

	err = m.Start(t.Context(), &wg)
	assert.NoError(t, err)

	err = m.Start(t.Context(), &wg)
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)
}

func TestPortfolioManagerStop(t *testing.T) {
	var m *portfolioManager
	var wg sync.WaitGroup
	err := m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	m, err = setupPortfolioManager(NewExchangeManager(), 0, nil)
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start(t.Context(), &wg)
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)
}

func TestProcessPortfolio(t *testing.T) {
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	require.NoError(t, err)

	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m, err := setupPortfolioManager(em, 0, nil)
	assert.NoError(t, err)

	m.processPortfolio(t.Context())
}

func TestUpdateExchangeBalances(t *testing.T) {
	t.Parallel()

	assert.ErrorContains(t, (*portfolioManager)(nil).updateExchangeBalances(t.Context()), "nil pointer: *engine.portfolioManager")
	assert.ErrorIs(t, new(portfolioManager).updateExchangeBalances(t.Context()), ErrNilSubsystem)

	m, err := setupPortfolioManager(NewExchangeManager(), 0, &portfolio.Base{Verbose: true})
	require.NoError(t, err, "setupPortfolioManager must not error")
	assert.NoError(t, m.updateExchangeBalances(t.Context()), "updateExchangeBalances should not error with an empty exchange list")

	e := &mockExchange{err: errors.New("Mock UpdateBalanceError")}
	m.exchangeManager.exchanges = map[string]exchange.IBotExchange{"mock": e}
	assert.NoError(t, m.updateExchangeBalances(t.Context()), "updateExchangeBalances should not error on disabled exchanges")

	e.enabled = true
	assert.NoError(t, m.updateExchangeBalances(t.Context()), "updateExchangeBalances should skip exchange without auth support")

	e.authSupported = true
	assert.ErrorIs(t, m.updateExchangeBalances(t.Context()), e.err, "error should contain the UpdateAccountBalances error message")
}

func TestUpdateExchangeAddressBalances(t *testing.T) {
	t.Parallel()

	assert.ErrorContains(t, (*portfolioManager)(nil).updateExchangeAddressBalances(nil), "nil pointer: *engine.portfolioManager")
	assert.ErrorContains(t, new(portfolioManager).updateExchangeAddressBalances(nil), "nil pointer: <nil>")

	e := &mockExchange{enabled: false, err: errors.New("Mock UpdateBalanceError")}
	m, err := setupPortfolioManager(NewExchangeManager(), 0, nil)
	require.NoError(t, err, "setupPortfolioManager must not error")
	assert.ErrorContains(t, m.updateExchangeAddressBalances(e), "nil pointer: *accounts.Accounts", "updateExchangeAddressBalances should propagate CurrencyBalances errors")

	a := accounts.MustNewAccounts(e)
	e.accounts = a
	subAcct := accounts.NewSubAccount(asset.Spot, "")
	subAcct.Balances.Set(currency.BTC, accounts.Balance{Total: 1.5})
	subAcct.Balances.Set(currency.ETH, accounts.Balance{Total: 0})
	require.NoError(t, a.Save(t.Context(), accounts.SubAccounts{subAcct}, false), "accounts.Save must not error")
	require.NoError(t, m.updateExchangeAddressBalances(e))
	require.Len(t, m.base.Addresses, 1, "must have one address for the positive balance")
	assert.Equal(t, 1.5, m.base.Addresses[0].Balance, "balance should match on a new address")

	subAcct.Balances.Set(currency.BTC, accounts.Balance{Total: 2})
	require.NoError(t, a.Save(t.Context(), accounts.SubAccounts{subAcct}, true), "accounts.Save must not error")
	require.NoError(t, m.updateExchangeAddressBalances(e))
	require.Len(t, m.base.Addresses, 1, "must have one address for the positive balance")
	assert.Equal(t, 2.0, m.base.Addresses[0].Balance, "balance should match after update existing address")

	subAcct.Balances.Set(currency.BTC, accounts.Balance{Total: 0})
	require.NoError(t, a.Save(t.Context(), accounts.SubAccounts{subAcct}, true), "accounts.Save must not error")
	require.NoError(t, m.updateExchangeAddressBalances(e))
	assert.Empty(t, m.base.Addresses, "should have removed address with no balance")
}

// mockExchange is a minimal mock for testing
type mockExchange struct {
	exchange.IBotExchange
	enabled       bool
	authSupported bool
	err           error
	accounts      *accounts.Accounts
}

func (m *mockExchange) GetName() string {
	return "mocky"
}

func (m *mockExchange) IsEnabled() bool {
	return m.enabled
}

func (m *mockExchange) IsRESTAuthenticationSupported() bool {
	return m.authSupported
}

func (m *mockExchange) HasAssetTypeAccountSegregation() bool {
	return true
}

func (m *mockExchange) GetAssetTypes(bool) asset.Items {
	return asset.Items{asset.Spot, asset.Futures}
}

func (m *mockExchange) UpdateAccountBalances(context.Context, asset.Item) (accounts.SubAccounts, error) {
	return nil, m.err
}

func (m *mockExchange) GetBase() *exchange.Base {
	return &exchange.Base{Name: "mocky", Accounts: m.accounts}
}

func (m *mockExchange) GetCredentials(context.Context) (*accounts.Credentials, error) {
	return &accounts.Credentials{Key: m.GetName()}, nil
}

func TestPortfolioManagerGetAddresses(t *testing.T) {
	t.Parallel()
	var nilManager *portfolioManager
	assert.Nil(t, nilManager.GetAddresses(), "GetAddresses should return nil for a nil manager")

	base := &portfolio.Base{Addresses: []portfolio.Address{{Address: "one"}}}
	m, err := setupPortfolioManager(NewExchangeManager(), 0, base)
	require.NoError(t, err, "setupPortfolioManager must not error")
	assert.Nil(t, m.GetAddresses(), "GetAddresses should return nil until the manager runs")

	m.started.Store(true)
	got := m.GetAddresses()
	require.Len(t, got, 1, "GetAddresses must return the address")
	got[0].Address = "changed"
	assert.Equal(t, "one", base.Addresses[0].Address, "GetAddresses should return a copy")
}

func TestInitPortfolioManager(t *testing.T) {
	t.Parallel()
	const address = "test-wallet-address"
	bot := &Engine{
		ExchangeManager: NewExchangeManager(),
		Config:          &config.Config{Portfolio: &portfolio.Base{Addresses: []portfolio.Address{{Address: address, WhiteListed: true}}}},
	}
	bot.initPortfolioManager(t.Context())
	require.NotNil(t, bot.portfolioManager, "initPortfolioManager must set up the manager while it is disabled")
	assert.False(t, bot.portfolioManager.IsRunning(), "a disabled portfolio manager should not run")

	withdrawManager, err := SetupWithdrawManager(bot.ExchangeManager, bot.portfolioManager, false)
	require.NoError(t, err, "SetupWithdrawManager must not error")
	assert.False(t, withdrawManager.portfolioManager.IsWhiteListed(address), "a stopped portfolio manager should whitelist nothing")

	require.NoError(t, bot.SetSubsystem(PortfolioManagerName, true), "SetSubsystem must not error")
	t.Cleanup(func() {
		assert.NoError(t, bot.SetSubsystem(PortfolioManagerName, false), "SetSubsystem should not error")
	})
	assert.True(t, withdrawManager.portfolioManager.IsWhiteListed(address), "the withdraw manager should see the portfolio manager enabled at runtime")

	existing := bot.portfolioManager
	bot.initPortfolioManager(t.Context())
	assert.Same(t, existing, bot.portfolioManager, "initPortfolioManager should keep an existing manager")
}
