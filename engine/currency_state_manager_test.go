package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestSetupCurrencyStateManager(t *testing.T) {
	t.Parallel()
	_, err := SetupCurrencyStateManager(0, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilExchangeManager)
	}

	cm, err := SetupCurrencyStateManager(0, &ExchangeManager{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if cm.sleep != DefaultStateManagerDelay {
		t.Fatal("unexpected value")
	}
}

type fakeExchangeManagerino struct {
	ErrorMeOne bool
	ErrorMeTwo bool
}

func (f *fakeExchangeManagerino) GetExchanges() ([]exchange.IBotExchange, error) {
	if f.ErrorMeOne {
		return nil, errors.New("woah nelly ;)")
	}
	return []exchange.IBotExchange{&fakerino{errorMe: f.ErrorMeTwo}}, nil
}

func (f *fakeExchangeManagerino) GetExchangeByName(_ string) (exchange.IBotExchange, error) {
	return nil, nil
}

type fakerino struct {
	exchange.IBotExchange
	errorMe bool
}

func (f *fakerino) UpdateCurrencyStates(_ context.Context, _ asset.Item) error {
	if f.errorMe {
		return errors.New("norty")
	}
	return nil
}

func (f *fakerino) GetAssetTypes(_ bool) asset.Items {
	return asset.Items{asset.Spot}
}

func (f *fakerino) GetName() string {
	return "testssssssssssssss"
}

func TestCurrencyStateManagerCoolRunnings(t *testing.T) {
	t.Parallel()
	err := (*CurrencyStateManager)(nil).Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	err = (&CurrencyStateManager{}).Stop()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrSubSystemNotStarted)
	}

	err = (&CurrencyStateManager{started: 1, shutdown: make(chan struct{})}).Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = (*CurrencyStateManager)(nil).Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	err = (&CurrencyStateManager{started: 1}).Start()
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrSubSystemAlreadyStarted)
	}

	man := &CurrencyStateManager{
		shutdown:         make(chan struct{}),
		iExchangeManager: &fakeExchangeManagerino{},
		sleep:            time.Minute}
	err = man.Start()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	time.Sleep(time.Millisecond)

	err = man.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	man.iExchangeManager = &fakeExchangeManagerino{ErrorMeOne: true}
	err = man.Start()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	time.Sleep(time.Millisecond)

	err = man.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	man.iExchangeManager = &fakeExchangeManagerino{ErrorMeTwo: true}
	err = man.Start()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	time.Sleep(time.Millisecond)

	if !man.IsRunning() {
		t.Fatal("this should be running")
	}

	err = man.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if man.IsRunning() {
		t.Fatal("this should be stopped")
	}
}
