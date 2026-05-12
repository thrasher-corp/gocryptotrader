package engine

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

type delayedShutdownExchange struct {
	bitfinex.Exchange
	name        string
	delay       time.Duration
	shutdownErr error
}

func (d *delayedShutdownExchange) GetName() string {
	if d.name != "" {
		return d.name
	}
	return d.Exchange.GetName()
}

func (d *delayedShutdownExchange) Shutdown() error {
	if d.delay > 0 {
		time.Sleep(d.delay)
	}
	return d.shutdownErr
}

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
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	assert.Equal(t, name, exch.GetName(), "GetName should return the same name")
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

func TestExchangeManagerShutdownTimeoutKeepsUnfinishedExchange(t *testing.T) {
	m := NewExchangeManager()
	slow := &delayedShutdownExchange{name: "slowex", delay: 500 * time.Millisecond}
	slow.SetDefaults()
	require.NoError(t, m.Add(slow))

	start := time.Now()
	require.NoError(t, m.Shutdown(50*time.Millisecond))
	assert.Less(t, time.Since(start), 400*time.Millisecond)

	_, err := m.GetExchangeByName("slowex")
	require.NoError(t, err)
}

func TestExchangeManagerShutdownRemovesSuccessfulExchangeAndKeepsFailures(t *testing.T) {
	t.Parallel()
	m := NewExchangeManager()

	success := &delayedShutdownExchange{name: "successex", delay: time.Millisecond}
	success.SetDefaults()
	require.NoError(t, m.Add(success))

	failed := &delayedShutdownExchange{name: "errorex", delay: time.Millisecond, shutdownErr: errExpectedTestError}
	failed.SetDefaults()
	require.NoError(t, m.Add(failed))

	require.NoError(t, m.Shutdown(time.Second))

	_, err := m.GetExchangeByName("successex")
	require.ErrorIs(t, err, ErrExchangeNotFound)

	_, err = m.GetExchangeByName("errorex")
	require.NoError(t, err)
}
