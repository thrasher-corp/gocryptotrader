package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
)

func TestNewAccountManager(t *testing.T) {
	_, err := NewAccountManager(nil, true)
	if !errors.Is(err, errEngineIsNil) {
		t.Fatalf("expected %v but received %v", errEngineIsNil, err)
	}
	am, err := NewAccountManager(&Engine{}, true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected %v but received %v", nil, err)
	}
	if am == nil {
		t.Fatal("oh no")
	}
}

func TestShutdown(t *testing.T) {
	am, err := NewAccountManager(&Engine{}, true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected %v but received %v", nil, err)
	}

	err = am.Shutdown()
	if !errors.Is(err, errAccountManagerNotStarted) {
		t.Fatalf("expected %v but received %v", errAccountManagerNotStarted, err)
	}

	if am.IsRunning() {
		t.Fatal("should not be running")
	}

	err = am.RunUpdater(time.Second * 8)
	if !errors.Is(err, errUnrealisticUpdateInterval) {
		t.Fatalf("expected %v but received %v", errUnrealisticUpdateInterval, err)
	}

	err = am.RunUpdater(time.Second * 10)
	if !errors.Is(err, nil) {
		t.Fatalf("expected %v but received %v", nil, err)
	}

	if !am.IsRunning() {
		t.Fatal("should be running")
	}

	err = am.RunUpdater(time.Second * 10)
	if !errors.Is(err, errAccountManagerAlreadyStarted) {
		t.Fatalf("expected %v but received %v", errAccountManagerAlreadyStarted, err)
	}

	err = am.Shutdown()
	if !errors.Is(err, nil) {
		t.Fatalf("expected %v but received %v", nil, err)
	}

	if !am.IsRunning() {
		t.Fatal("should not be running")
	}
}

func TestUpdateAccountForExchange(t *testing.T) {
	am := AccountManager{}
	burnance := &binance.Binance{}

	am.updateAccountForExchange(burnance)
	burnance.SetDefaults()
	dConf, err := burnance.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	dConf.API.AuthenticatedSupport = true
	err = burnance.Setup(dConf)
	if err != nil {
		t.Fatal(err)
	}
	am.updateAccountForExchange(burnance)
}
