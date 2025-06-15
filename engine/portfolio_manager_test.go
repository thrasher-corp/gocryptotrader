package engine

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
