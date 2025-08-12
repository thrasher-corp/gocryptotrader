package engine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

type broken struct {
	bitfinex.Exchange
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
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = NewExchangeManager()
	err = m.Add(nil)
	require.ErrorIs(t, err, errExchangeIsNil)

	b := new(bitfinex.Exchange)
	b.SetDefaults()
	err = m.Add(b)
	require.NoError(t, err)

	err = m.Add(b)
	require.ErrorIs(t, err, ErrExchangeAlreadyLoaded)

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
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = NewExchangeManager()
	exchanges, err := m.GetExchanges()
	if err != nil {
		t.Error("no exchange manager found")
	}
	if len(exchanges) != 0 {
		t.Error("unexpected value")
	}
	b := new(bitfinex.Exchange)
	b.SetDefaults()
	err = m.Add(b)
	require.NoError(t, err)

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
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = NewExchangeManager()

	err = m.RemoveExchange("")
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	err = m.RemoveExchange("Bitfinex")
	require.ErrorIs(t, err, ErrExchangeNotFound)

	b := new(bitfinex.Exchange)
	b.SetDefaults()
	err = m.Add(b)
	require.NoError(t, err)

	err = m.RemoveExchange("Bitstamp")
	assert.ErrorIs(t, err, ErrExchangeNotFound)

	err = m.RemoveExchange("BiTFiNeX")
	require.NoError(t, err)

	if len(m.exchanges) != 0 {
		t.Error("exchange manager len should be 0")
	}

	brokenExch := &broken{}
	brokenExch.SetDefaults()

	err = m.Add(brokenExch)
	require.NoError(t, err)

	err = m.RemoveExchange("BiTFiNeX")
	require.ErrorIs(t, err, errExpectedTestError)
}

func TestNewExchangeByName(t *testing.T) {
	var m *ExchangeManager
	_, err := m.NewExchangeByName("")
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = NewExchangeManager()
	_, err = m.NewExchangeByName("")
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

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

	load := &bitfinex.Exchange{}
	load.SetDefaults()

	err = m.Add(load)
	require.NoError(t, err)

	_, err = m.NewExchangeByName("bitfinex")
	require.ErrorIs(t, err, ErrExchangeAlreadyLoaded)
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
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = NewExchangeManager()
	err = m.Shutdown(-1)
	require.NoError(t, err)

	brokenExch := &broken{}
	brokenExch.SetDefaults()

	err = m.Add(brokenExch)
	require.NoError(t, err)

	err = m.Shutdown(-1)
	require.NoError(t, err)
}
