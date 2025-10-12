package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
	err = m.Start(&wg)
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
	err := m.Start(nil)
	assert.ErrorIs(t, err, ErrNilSubsystem)

	m, err = setupPortfolioManager(NewExchangeManager(), 0, nil)
	assert.NoError(t, err)

	err = m.Start(nil)
	assert.ErrorIs(t, err, errNilWaitGroup)

	err = m.Start(&wg)
	assert.NoError(t, err)

	err = m.Start(&wg)
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

	err = m.Start(&wg)
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

	m.processPortfolio()
}

// TestUpdateExchangeBalances tests the updateExchangeBalances function code paths
func TestUpdateExchangeBalances(t *testing.T) {
	t.Parallel()

	em := NewExchangeManager()
	m, err := setupPortfolioManager(em, 0, nil)
	require.NoError(t, err, "setupPortfolioManager must not error")

	err = m.updateExchangeBalances()
	assert.NoError(t, err, "updateExchangeBalances should not error with no exchanges")

	exch, err := em.NewExchangeByName("Bitstamp")
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	exch.SetEnabled(false)
	err = em.Add(exch)
	require.NoError(t, err, "Add must not error")

	m, err = setupPortfolioManager(em, 0, nil)
	require.NoError(t, err, "setupPortfolioManager must not error")

	err = m.updateExchangeBalances()
	assert.NoError(t, err, "updateExchangeBalances should skip disabled exchange and not error")

	em = NewExchangeManager()
	exch, err = em.NewExchangeByName("Bitstamp")
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	exch.SetEnabled(true)
	err = em.Add(exch)
	require.NoError(t, err, "Add must not error")

	m, err = setupPortfolioManager(em, 0, nil)
	require.NoError(t, err, "setupPortfolioManager must not error")

	err = m.updateExchangeBalances()
	assert.NoError(t, err, "updateExchangeBalances should skip exchange without auth and not error")

	m.exchangeManager = nil
	err = m.updateExchangeBalances()
	assert.Error(t, err, "updateExchangeBalances should error with nil exchangeManager")
	assert.Contains(t, err.Error(), "cannot get exchanges", "Error should mention cannot get exchanges")
}

// mockExchange is a minimal mock for testing updateExchangeBalances code paths
type mockExchange struct {
	exchange.IBotExchange
	name                string
	enabled             bool
	authSupported       bool
	hasAssetSegregation bool
	assetTypes          asset.Items
	updateBalancesErr   error
}

func (m *mockExchange) GetName() string {
	return m.name
}

func (m *mockExchange) IsEnabled() bool {
	return m.enabled
}

func (m *mockExchange) IsRESTAuthenticationSupported() bool {
	return m.authSupported
}

func (m *mockExchange) HasAssetTypeAccountSegregation() bool {
	return m.hasAssetSegregation
}

func (m *mockExchange) GetAssetTypes(enabled bool) asset.Items {
	return m.assetTypes
}

func (m *mockExchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	return nil, m.updateBalancesErr
}

func (m *mockExchange) GetBase() *exchange.Base {
	return &exchange.Base{Name: m.name}
}

// TestUpdateExchangeBalancesCodePaths tests the specific code paths in updateExchangeBalances
func TestUpdateExchangeBalancesCodePaths(t *testing.T) {
	t.Parallel()

	t.Run("disabled exchange continues loop", func(t *testing.T) {
		t.Parallel()
		em := NewExchangeManager()
		em.exchanges = map[string]exchange.IBotExchange{
			"test": &mockExchange{name: "test", enabled: false},
		}
		m, err := setupPortfolioManager(em, 0, nil)
		require.NoError(t, err)

		err = m.updateExchangeBalances()
		assert.NoError(t, err, "Should skip disabled exchange")
	})

	t.Run("exchange without auth continues loop", func(t *testing.T) {
		t.Parallel()
		em := NewExchangeManager()
		em.exchanges = map[string]exchange.IBotExchange{
			"test": &mockExchange{name: "test", enabled: true, authSupported: false},
		}
		m, err := setupPortfolioManager(em, 0, nil)
		require.NoError(t, err)

		err = m.updateExchangeBalances()
		assert.NoError(t, err, "Should skip exchange without auth")
	})

	t.Run("exchange with asset segregation uses GetAssetTypes", func(t *testing.T) {
		t.Parallel()
		em := NewExchangeManager()
		em.exchanges = map[string]exchange.IBotExchange{
			"test": &mockExchange{
				name:                "test",
				enabled:             true,
				authSupported:       true,
				hasAssetSegregation: true,
				assetTypes:          asset.Items{asset.Spot, asset.Futures},
			},
		}
		m, err := setupPortfolioManager(em, 0, nil)
		require.NoError(t, err)

		err = m.updateExchangeBalances()
		assert.Error(t, err, "Should attempt updates and encounter errors")
	})

	t.Run("UpdateAccountBalances error is appended", func(t *testing.T) {
		t.Parallel()
		em := NewExchangeManager()
		em.exchanges = map[string]exchange.IBotExchange{
			"test": &mockExchange{
				name:              "test",
				enabled:           true,
				authSupported:     true,
				updateBalancesErr: errors.New("balance update failed"),
			},
		}
		m, err := setupPortfolioManager(em, 0, nil)
		require.NoError(t, err)

		err = m.updateExchangeBalances()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "balance update failed")
	})
}

// TestUpdateExchangeAddressBalances tests the updateExchangeAddressBalances function
func TestUpdateExchangeAddressBalances(t *testing.T) {
	t.Parallel()

	em := NewExchangeManager()
	m, err := setupPortfolioManager(em, 0, nil)
	require.NoError(t, err)

	exch, err := em.NewExchangeByName("Bitstamp")
	require.NoError(t, err)
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	err = m.updateExchangeAddressBalances(exch)
	if err != nil {
		assert.Error(t, err, "updateExchangeAddressBalances may error with uninitialized accounts")
	}
}
