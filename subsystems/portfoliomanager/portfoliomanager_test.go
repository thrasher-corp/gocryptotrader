package portfoliomanager

import (
	"errors"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

func TestSetup(t *testing.T) {
	_, err := Setup(nil, 0, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	m, err := Setup(exchangemanager.Setup(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestIsRunning(t *testing.T) {
	var m *Manager
	if m.IsRunning() {
		t.Error("expected false")
	}

	m, err := Setup(exchangemanager.Setup(), 0, nil)
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

func TestStart(t *testing.T) {
	var m *Manager
	var wg sync.WaitGroup
	err := m.Start(nil)
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}

	m, err = Setup(exchangemanager.Setup(), 0, nil)
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
	if !errors.Is(err, subsystems.ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemAlreadyStarted)
	}
}

func TestStop(t *testing.T) {
	var m *Manager
	var wg sync.WaitGroup
	err := m.Stop()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}

	m, err = Setup(exchangemanager.Setup(), 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
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
	em := exchangemanager.Setup()
	exch, err := em.NewExchangeByName("Bitstamp")
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	em.Add(exch)
	m, err := Setup(em, 0, nil)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.processPortfolio()
}
