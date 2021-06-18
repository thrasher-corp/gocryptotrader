package engine

import (
	"errors"
	"testing"
	"time"
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

// type badAccountInfo struct {
// 	// FakePassingExchange
// }

// func (b *badAccountInfo) UpdateAccountInfo() (account.FullSnapshot, error) {
// 	return nil, errors.New("this is intentionally evil")
// }

// func TestUpdateAccountForExchange(t *testing.T) {
// 	a := AccountManager{
// 		accounts: make(map[exchange.IBotExchange]int),
// 	}
// 	fakeExchange := &FakePassingExchange{
// 		Base: exchange.Base{
// 			Config: &config.ExchangeConfig{},
// 		},
// 	}
// 	a.updateAccountForExchange(fakeExchange)
// 	fakeExchange.Config.API.AuthenticatedSupport = true
// 	a.updateAccountForExchange(fakeExchange)
// 	fakeExchange.Config.API.AuthenticatedWebsocketSupport = true
// 	a.updateAccountForExchange(fakeExchange)
// 	a.updateAccountForExchange(fakeExchange)
// 	a.updateAccountForExchange(fakeExchange)
// 	a.updateAccountForExchange(fakeExchange)
// 	a.updateAccountForExchange(fakeExchange)
// 	a.updateAccountForExchange(fakeExchange)

// 	bad := &badAccountInfo{
// 		FakePassingExchange: FakePassingExchange{
// 			Base: exchange.Base{
// 				Config: &config.ExchangeConfig{
// 					API: config.APIConfig{
// 						AuthenticatedSupport: true,
// 					},
// 				},
// 			},
// 		},
// 	}

// 	a.updateAccountForExchange(bad)
// }
