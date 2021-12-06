package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var errTestError = errors.New("test error")

type feeExchangeManager struct {
	iExchangeManager
}

func (f *feeExchangeManager) GetExchanges() ([]exchange.IBotExchange, error) {
	return []exchange.IBotExchange{&feeExchange{}}, errTestError
}

type feeExchange struct {
	exchange.IBotExchange
	ErrorUpdateCommissionFees   error
	ErrorUpdateTransferFees     error
	ErrorUpdateBankTransferFees error
}

func (f *feeExchange) UpdateCommissionFees(_ context.Context, _ asset.Item) error {
	return f.ErrorUpdateCommissionFees
}

func (f *feeExchange) UpdateTransferFees(_ context.Context) error {
	return f.ErrorUpdateTransferFees
}

func (f *feeExchange) UpdateBankTransferFees(_ context.Context) error {
	return f.ErrorUpdateBankTransferFees
}

func (f *feeExchange) GetName() string {
	return "test fee exchange"
}

func (f *feeExchange) GetAuthenticatedAPISupport(_ uint8) bool {
	return true
}

func TestSetupFeeManager(t *testing.T) {
	t.Parallel()
	_, err := SetupFeeManager(0, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilExchangeManager)
	}

	fm, err := SetupFeeManager(0, &fakeExchangeManagerino{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if fm.sleep != DefaultFeeManagerDelay {
		t.Fatal("unexpected delay")
	}
}

func TestFeeManagerStartStop(t *testing.T) {
	t.Parallel()
	var fm *FeeManager
	err := fm.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	fm = new(FeeManager)
	fm.iExchangeManager = &feeExchangeManager{}
	err = fm.Start()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = fm.Start()
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrSubSystemAlreadyStarted)
	}

	err = fm.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = fm.Stop()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrSubSystemNotStarted)
	}

	fm = nil
	err = fm.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	// fm = new(FeeManager)
	// fm.iExchangeManager = &feeExchangeManager{}

	// err = fm.Stop()
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	// }
}

func TestFeeManagerIsRunning(t *testing.T) {
	t.Parallel()
	fm := new(FeeManager)
	if fm.IsRunning() {
		t.Fatal("unexpected result")
	}

	err := fm.Start()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !fm.IsRunning() {
		t.Fatal("unexpected result")
	}

	err = fm.Stop()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if fm.IsRunning() {
		t.Fatal("unexpected result")
	}
}

func TestFeeManagerUpdate(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	wg.Add(1)
	update(&feeExchange{
		ErrorUpdateCommissionFees:   errTestError,
		ErrorUpdateTransferFees:     errTestError,
		ErrorUpdateBankTransferFees: errTestError,
	}, &wg, asset.Items{asset.Spot})
	wg.Wait()
}
