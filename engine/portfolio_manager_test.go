package engine

import (
	"errors"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
)

func TestSetupPortfolioManager(t *testing.T) {
	_, err := setupPortfolioManager(nil, 0, nil)
	if !errors.Is(err, subsystem.ErrNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNilExchangeManager)
	}

	m, err := setupPortfolioManager(NewExchangeManager(), 0, nil)
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

	m, err := setupPortfolioManager(NewExchangeManager(), 0, nil)
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
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}

	m, err = setupPortfolioManager(NewExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start(nil)
	if !errors.Is(err, subsystem.ErrNilWaitGroup) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNilWaitGroup)
	}

	err = m.Start(&wg)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start(&wg)
	if !errors.Is(err, subsystem.ErrAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrAlreadyStarted)
	}
}

func TestPortfolioManagerStop(t *testing.T) {
	var m *portfolioManager
	var wg sync.WaitGroup
	err := m.Stop()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}

	m, err = setupPortfolioManager(NewExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, subsystem.ErrNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNotStarted)
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
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	m, err := setupPortfolioManager(em, 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.processPortfolio()
}
