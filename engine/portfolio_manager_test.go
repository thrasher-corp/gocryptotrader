package engine

import (
	"errors"
	"sync"
	"testing"
)

func TestSetupPortfolioManager(t *testing.T) {
	_, err := setupPortfolioManager(nil, 0, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	m, err := setupPortfolioManager(SetupExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestIsPortfolioManagerRunning(t *testing.T) {
	var m *portfolioManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	m, err := setupPortfolioManager(SetupExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
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
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	m, err = setupPortfolioManager(SetupExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start(nil)
	if !errors.Is(err, errNilWaitGroup) {
		t.Errorf("error '%v', expected '%v'", err, errNilWaitGroup)
	}

	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start(&wg)
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemAlreadyStarted)
	}
}

func TestPortfolioManagerStop(t *testing.T) {
	var m *portfolioManager
	var wg sync.WaitGroup
	err := m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	m, err = setupPortfolioManager(SetupExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestProcessPortfolio(t *testing.T) {
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err := setupPortfolioManager(em, 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.processPortfolio()
}
