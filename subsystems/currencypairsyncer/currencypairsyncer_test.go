package currencypairsyncer

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

func TestSetup(t *testing.T) {
	_, err := Setup(&Config{}, nil, nil, nil)
	if !errors.Is(err, errNoSyncItemsEnabled) {
		t.Errorf("error '%v', expected '%v'", err, errNoSyncItemsEnabled)
	}

	_, err = Setup(&Config{SyncTrades: true}, nil, nil, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = Setup(&Config{SyncTrades: true}, &exchangemanager.Manager{}, nil, nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	m, err := Setup(&Config{SyncTrades: true}, &exchangemanager.Manager{}, nil, &config.RemoteControlConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestStart(t *testing.T) {
	m, err := Setup(&Config{SyncTrades: true}, &exchangemanager.Manager{}, nil, &config.RemoteControlConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	em := exchangemanager.Setup()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	m.exchangeManager = em
	m.config.SyncContinuously = true
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, subsystems.ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemAlreadyStarted)
	}

	m = nil
	err = m.Start()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
}
