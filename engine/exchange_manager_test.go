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
	m := NewExchangeManager()
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	err := m.Add(b)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
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
	m := NewExchangeManager()
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
	m := NewExchangeManager()
	err := m.RemoveExchange("Bitfinex")
	if !errors.Is(err, ErrNoExchangesLoaded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrNoExchangesLoaded)
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

	if m.Len() != 0 {
		t.Error("exchange manager len should be 0")
	}
}

func TestNewExchangeByName(t *testing.T) {
	m := NewExchangeManager()
	exchanges := []string{"binanceus", "binance", "bitfinex", "bitflyer", "bithumb", "bitmex", "bitstamp", "bittrex", "btc markets", "btse", "bybit", "coinut", "exmo", "coinbasepro", "ftx", "gateio", "gemini", "hitbtc", "huobi", "itbit", "kraken", "lbank", "localbitcoins", "okcoin international", "okx", "poloniex", "yobit", "zb", "fake"}
	for i := range exchanges {
		exch, err := m.NewExchangeByName(exchanges[i])
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
