package state

import (
	"errors"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestGetManager(t *testing.T) {
	t.Parallel()
	if GetManager() == nil {
		t.Fatal("unexpected return")
	}
}

func TestRegisterExchangeState(t *testing.T) {
	t.Parallel()
	_, err := RegisterExchangeState("")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v, but expected: %v", err, errExchangeNameIsEmpty)
	}

	s, err := RegisterExchangeState("boo")
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	if s == nil {
		t.Fatal("unexpected value")
	}
}

func TestManagerRegister(t *testing.T) {
	t.Parallel()
	err := (&Manager{}).Register("", nil)
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v, but expected: %v", err, errExchangeNameIsEmpty)
	}

	err = (&Manager{}).Register("boo", nil)
	if !errors.Is(err, errStatesIsNil) {
		t.Fatalf("received: %v, but expected: %v", err, errStatesIsNil)
	}

	man := &Manager{}
	s := &States{m: make(map[asset.Item]map[*currency.Item]*Currency)}
	err = man.Register("boo", s)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	err = man.Register("boo", s)
	if !errors.Is(err, errStatesAlreadyLoaded) {
		t.Fatalf("received: %v, but expected: %v", err, errStatesAlreadyLoaded)
	}
}

func TestManagerCanTrade(t *testing.T) {
	t.Parallel()
	err := (&Manager{}).CanTrade("", currency.Code{}, "")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v, but expected: %v", err, errExchangeNameIsEmpty)
	}

	man := &Manager{m: map[string]*States{"boo": {
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: &Currency{},
			},
		},
	}}}

	err = man.CanTrade("boo", currency.BTC, asset.Spot)
	if !errors.Is(err, errTradingNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errTradingNotAllowed)
	}

	man = &Manager{m: map[string]*States{"boo": {
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: &Currency{trading: true},
			},
		},
	}}}

	err = man.CanTrade("boo", currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
}

func TestManagerCanWithdraw(t *testing.T) {
	t.Parallel()
	err := (&Manager{}).CanWithdraw("", currency.Code{}, "")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v, but expected: %v", err, errExchangeNameIsEmpty)
	}

	man := &Manager{m: map[string]*States{"boo": {
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: &Currency{},
			},
		},
	}}}

	err = man.CanWithdraw("boo", currency.BTC, asset.Spot)
	if !errors.Is(err, errWithdrawalsNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errWithdrawalsNotAllowed)
	}

	man = &Manager{m: map[string]*States{"boo": {
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: &Currency{withdrawals: true},
			},
		},
	}}}

	err = man.CanWithdraw("boo", currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
}

func TestManagerCanDeposit(t *testing.T) {
	t.Parallel()
	err := (&Manager{}).CanDeposit("", currency.Code{}, "")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v, but expected: %v", err, errExchangeNameIsEmpty)
	}

	man := &Manager{m: map[string]*States{"boo": {
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: &Currency{},
			},
		},
	}}}

	err = man.CanDeposit("boo", currency.BTC, asset.Spot)
	if !errors.Is(err, errDepositNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errDepositNotAllowed)
	}

	man = &Manager{m: map[string]*States{"boo": {
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: &Currency{deposits: true},
			},
		},
	}}}

	err = man.CanDeposit("boo", currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
}

func TestManagerGetExchangeStates(t *testing.T) {
	t.Parallel()
	_, err := (&Manager{}).getExchangeStates("")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v, but expected: %v", err, errExchangeNameIsEmpty)
	}

	_, err = (&Manager{}).getExchangeStates("boo")
	if !errors.Is(err, errExchangeNotFound) {
		t.Fatalf("received: %v, but expected: %v", err, errExchangeNotFound)
	}

	man := &Manager{m: map[string]*States{"boo": {}}}
	_, err = man.getExchangeStates("boo")
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
}

func TestGetSnapshot(t *testing.T) {
	t.Parallel()
	_, err := (*States)(nil).GetSnapshot()
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	o, err := (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {currency.BTC.Item: {
				withdrawals: true,
				deposits:    true,
				trading:     true,
			}},
		},
	}).GetSnapshot()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	if o == nil {
		t.Fatal("unexpected value")
	}
}

func TestCanTradePair(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanTradePair(currency.Pair{}, "")
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	err = (&States{}).CanTradePair(currency.Pair{}, "")
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	cp := currency.NewPair(currency.BTC, currency.USD)
	err = (&States{}).CanTradePair(cp, "")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&States{}).CanTradePair(cp, asset.Spot)
	if !errors.Is(err, nil) { // not found but default to operational
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: true},
				currency.USD.Item: {trading: true},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: false},
				currency.USD.Item: {trading: true},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	if !errors.Is(err, errTradingNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errTradingNotAllowed)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: true},
				currency.USD.Item: {trading: false},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	if !errors.Is(err, errTradingNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errTradingNotAllowed)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {trading: false},
				currency.USD.Item: {trading: false},
			},
		},
	}).CanTradePair(cp, asset.Spot)
	if !errors.Is(err, errTradingNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errTradingNotAllowed)
	}
}

func TestStatesCanTrade(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanTrade(currency.Code{}, "")
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}
	err = (&States{}).CanTrade(currency.Code{}, "")
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}
}

func TestStatesCanWithdraw(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanWithdraw(currency.Code{}, "")
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}
	err = (&States{}).CanWithdraw(currency.Code{}, "")
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}
}

func TestStatesCanDeposit(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanDeposit(currency.Code{}, "")
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}
	err = (&States{}).CanDeposit(currency.Code{}, "")
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}
}

func TestStatesUpdateAll(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).UpdateAll("", nil)
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	err = (&States{}).UpdateAll("", nil)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&States{}).UpdateAll(asset.Spot, nil)
	if !errors.Is(err, errUpdatesAreNil) {
		t.Fatalf("received: %v, but expected: %v", err, errUpdatesAreNil)
	}

	s := &States{
		m: map[asset.Item]map[*currency.Item]*Currency{},
	}

	err = s.UpdateAll(asset.Spot, map[currency.Code]Options{
		currency.BTC: {
			Withdraw: convert.BoolPtr(true),
			Trade:    convert.BoolPtr(true),
			Deposit:  convert.BoolPtr(true)},
	})

	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	err = s.UpdateAll(asset.Spot, map[currency.Code]Options{currency.BTC: {
		Withdraw: convert.BoolPtr(false),
		Deposit:  convert.BoolPtr(false),
		Trade:    convert.BoolPtr(false),
	}})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	c, err := s.Get(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	if c.CanDeposit() || c.CanTrade() || c.CanWithdraw() {
		t.Fatal()
	}
}

func TestStatesUpdate(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).Update(currency.Code{}, "", Options{})
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	err = (&States{}).Update(currency.Code{}, "", Options{})
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	err = (&States{}).Update(currency.BTC, "", Options{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {currency.BTC.Item: &Currency{}},
		},
	}).Update(currency.BTC, asset.Spot, Options{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
}

func TestStatesGet(t *testing.T) {
	t.Parallel()
	_, err := (*States)(nil).Get(currency.Code{}, "")
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	_, err = (&States{}).Get(currency.Code{}, "")
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	_, err = (&States{}).Get(currency.BTC, "")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	_, err = (&States{}).Get(currency.BTC, asset.Spot)
	if !errors.Is(err, errCurrencyStateNotFound) {
		t.Fatalf("received: %v, but expected: %v", err, errCurrencyStateNotFound)
	}
}

func TestCurrencyGetState(t *testing.T) {
	o := (&Currency{}).GetState()
	if *o.Deposit || *o.Trade || *o.Withdraw {
		t.Fatal("unexpected values")
	}
}

func TestAlerting(t *testing.T) {
	c := Currency{}
	var start, finish sync.WaitGroup
	start.Add(3)
	finish.Add(3)
	go waitForAlert(c.WaitTrading(nil), &start, &finish)
	go waitForAlert(c.WaitDeposit(nil), &start, &finish)
	go waitForAlert(c.WaitWithdraw(nil), &start, &finish)
	start.Wait()
	c.update(Options{
		Trade:    convert.BoolPtr(true),
		Withdraw: convert.BoolPtr(true),
		Deposit:  convert.BoolPtr(true)})
	finish.Wait()
}

func waitForAlert(ch <-chan bool, start, finish *sync.WaitGroup) {
	defer finish.Done()
	start.Done()
	<-ch
}
