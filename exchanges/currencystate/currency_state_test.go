package currencystate

import (
	"errors"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestNewCurrencyStates(t *testing.T) {
	if NewCurrencyStates() == nil {
		t.Fatal("unexpected value")
	}
}

func TestGetSnapshot(t *testing.T) {
	t.Parallel()
	_, err := (*States)(nil).GetCurrencyStateSnapshot()
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
	}).GetCurrencyStateSnapshot()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	if o == nil {
		t.Fatal("unexpected value")
	}
}

func TestCanTradePair(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanTradePair(currency.EMPTYPAIR, asset.Empty)
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	err = (&States{}).CanTradePair(currency.EMPTYPAIR, asset.Empty)
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	cp := currency.NewPair(currency.BTC, currency.USD)
	err = (&States{}).CanTradePair(cp, asset.Empty)
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
	err := (*States)(nil).CanTrade(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}
	err = (&States{}).CanTrade(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}
}

func TestStatesCanWithdraw(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanWithdraw(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}
	err = (&States{}).CanWithdraw(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {withdrawals: true},
			},
		},
	}).CanWithdraw(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {},
			},
		},
	}).CanWithdraw(currency.BTC, asset.Spot)
	if !errors.Is(err, errWithdrawalsNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errWithdrawalsNotAllowed)
	}
}

func TestStatesCanDeposit(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).CanDeposit(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}
	err = (&States{}).CanDeposit(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {deposits: true},
			},
		},
	}).CanDeposit(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}

	err = (&States{
		m: map[asset.Item]map[*currency.Item]*Currency{
			asset.Spot: {
				currency.BTC.Item: {},
			},
		},
	}).CanDeposit(currency.BTC, asset.Spot)
	if !errors.Is(err, errDepositNotAllowed) {
		t.Fatalf("received: %v, but expected: %v", err, errDepositNotAllowed)
	}
}

func TestStatesUpdateAll(t *testing.T) {
	t.Parallel()
	err := (*States)(nil).UpdateAll(asset.Empty, nil)
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	err = (&States{}).UpdateAll(asset.Empty, nil)
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
	err := (*States)(nil).Update(currency.EMPTYCODE, asset.Empty, Options{})
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	err = (&States{}).Update(currency.EMPTYCODE, asset.Empty, Options{})
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	err = (&States{}).Update(currency.BTC, asset.Empty, Options{})
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
	_, err := (*States)(nil).Get(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errNilStates) {
		t.Fatalf("received: %v, but expected: %v", err, errNilStates)
	}

	_, err = (&States{}).Get(currency.EMPTYCODE, asset.Empty)
	if !errors.Is(err, errEmptyCurrency) {
		t.Fatalf("received: %v, but expected: %v", err, errEmptyCurrency)
	}

	_, err = (&States{}).Get(currency.BTC, asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, but expected: %v", err, asset.ErrNotSupported)
	}

	_, err = (&States{}).Get(currency.BTC, asset.Spot)
	if !errors.Is(err, ErrCurrencyStateNotFound) {
		t.Fatalf("received: %v, but expected: %v", err, ErrCurrencyStateNotFound)
	}
}

func TestCurrencyGetState(t *testing.T) {
	o := (&Currency{}).GetState()
	if *o.Deposit || *o.Trade || *o.Withdraw {
		t.Fatal("unexpected values")
	}
}

func TestAlerting(_ *testing.T) {
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
