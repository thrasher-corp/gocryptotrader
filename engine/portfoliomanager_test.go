package engine

import (
	"errors"
	"sync"
	"testing"
)

func TestSetupPortfolioManager(t *testing.T) {
	_, err := SetupPortfolioManager(nil, 0, nil)
	if !errors.Is(err, subsystem.errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.errNilExchangeManager)
	}

	m, err := SetupPortfolioManager(SetupExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestIsPortfolioManagerRunning(t *testing.T) {
	var m *PortfolioManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	m, err := SetupPortfolioManager(SetupExchangeManager(), 0, nil)
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
	var m *PortfolioManager
	var wg sync.WaitGroup
	err := m.Start(nil)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	m, err = SetupPortfolioManager(SetupExchangeManager(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start(nil)
	if !errors.Is(err, subsystem.errNilWaitGroup) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.errNilWaitGroup)
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
	var m *PortfolioManager
	var wg sync.WaitGroup
	err := m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}

	m, err = SetupPortfolioManager(SetupExchangeManager(), 0, nil)
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
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err := SetupPortfolioManager(em, 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.processPortfolio()
}
