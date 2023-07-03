package engine

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

type broken struct {
	bitfinex.Bitfinex
}

func (b *broken) Shutdown() error { return errExpectedTestError }

func TestNewExchangeManager(t *testing.T) {
	t.Parallel()
	m := NewExchangeManager()
	if m == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatalf("unexpected response")
	}
	if m.exchanges == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Error("unexpected response")
	}
}

func TestExchangeManagerAdd(t *testing.T) {
	t.Parallel()
	var m *ExchangeManager
	err := m.Add(nil)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	m = NewExchangeManager()
	err = m.Add(nil)
	if !errors.Is(err, errExchangeIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeIsNil)
	}
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	err = m.Add(b)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	err = m.Add(b)
	if !errors.Is(err, errExchangeAlreadyLoaded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeAlreadyLoaded)
	}
	exchanges, err := m.GetExchanges()
	if err != nil {
		t.Error("no exchange manager found")
	}
	if exchanges[0].GetName() != "Bitfinex" {
		t.Error("unexpected exchange name")
	}
}

func TestExchangeManagerGetExchanges(t *testing.T) {
	t.Parallel()
	var m *ExchangeManager
	_, err := m.GetExchanges()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	m = NewExchangeManager()
	exchanges, err := m.GetExchanges()
	if err != nil {
		t.Error("no exchange manager found")
	}
	if len(exchanges) != 0 {
		t.Error("unexpected value")
	}
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	err = m.Add(b)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	exchanges, err = m.GetExchanges()
	if err != nil {
		t.Error("no exchange manager found")
	}
	if exchanges[0].GetName() != "Bitfinex" {
		t.Error("unexpected exchange name")
	}
}

func TestExchangeManagerRemoveExchange(t *testing.T) {
	t.Parallel()
	var m *ExchangeManager
	err := m.RemoveExchange("")
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	m = NewExchangeManager()

	err = m.RemoveExchange("")
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}

	err = m.RemoveExchange("Bitfinex")
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrExchangeNotFound)
	}

	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	err = m.Add(b)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = m.RemoveExchange("Bitstamp")
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received: %v but expected: %v", err, ErrExchangeNotFound)
	}

	err = m.RemoveExchange("BiTFiNeX")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(m.exchanges) != 0 {
		t.Error("exchange manager len should be 0")
	}

	brokenExch := &broken{}
	brokenExch.SetDefaults()

	err = m.Add(brokenExch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = m.RemoveExchange("BiTFiNeX")
	if !errors.Is(err, errExpectedTestError) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExpectedTestError)
	}
}

func TestNewExchangeByName(t *testing.T) {
	var m *ExchangeManager
	_, err := m.NewExchangeByName("")
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	m = NewExchangeManager()
	_, err = m.NewExchangeByName("")
	if !errors.Is(err, ErrExchangeNameIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrExchangeNameIsEmpty)
	}

	exchanges := exchange.Exchanges
	exchanges = append(exchanges, "fake")
	for i := range exchanges {
		var exch exchange.IBotExchange
		exch, err = m.NewExchangeByName(exchanges[i])
		if err != nil && exchanges[i] != "fake" {
			t.Fatal(err)
		}
		if err == nil {
			exch.SetDefaults()
			if !strings.EqualFold(exch.GetName(), exchanges[i]) {
				t.Error("did not load expected exchange")
			}
		}
	}

	load := &bitfinex.Bitfinex{}
	load.SetDefaults()

	err = m.Add(load)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = m.NewExchangeByName("bitfinex")
	if !errors.Is(err, ErrExchangeAlreadyLoaded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrExchangeAlreadyLoaded)
	}
}

type ExchangeBuilder struct{}

func (n ExchangeBuilder) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	var exch exchange.IBotExchange

	switch name {
	case "customex":
		exch = new(sharedtestvalues.CustomEx)
	default:
		return nil, fmt.Errorf("%s, %w", name, ErrExchangeNotFound)
	}

	return exch, nil
}

func TestNewCustomExchangeByName(t *testing.T) {
	m := NewExchangeManager()
	m.Builder = ExchangeBuilder{}
	name := "customex"
	exch, err := m.NewExchangeByName(name)
	if err != nil {
		t.Fatal(err)
	}
	if err == nil {
		exch.SetDefaults()
		if !strings.EqualFold(exch.GetName(), name) {
			t.Error("did not load expected exchange")
		}
	}
}

func TestExchangeManagerShutdown(t *testing.T) {
	t.Parallel()
	var m *ExchangeManager
	err := m.Shutdown(-1)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNilSubsystem)
	}

	m = NewExchangeManager()
	err = m.Shutdown(-1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	brokenExch := &broken{}
	brokenExch.SetDefaults()

	err = m.Add(brokenExch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = m.Shutdown(-1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}
