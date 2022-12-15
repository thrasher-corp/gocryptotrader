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

func TestSetupExchangeManager(t *testing.T) {
	t.Parallel()
	m := SetupExchangeManager()
	if m == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatalf("unexpected response")
	}
	if m.exchanges == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Error("unexpected response")
	}
}

func TestExchangeManagerAdd(t *testing.T) {
	t.Parallel()
	m := SetupExchangeManager()
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	m.Add(b)
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
	m := SetupExchangeManager()
	exchanges, err := m.GetExchanges()
	if err != nil {
		t.Error("no exchange manager found")
	}
	if len(exchanges) != 0 {
		t.Error("unexpected value")
	}
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	m.Add(b)
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
	m := SetupExchangeManager()
	if err := m.RemoveExchange("Bitfinex"); err != ErrNoExchangesLoaded {
		t.Error("no exchanges should be loaded")
	}
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	m.Add(b)
	err := m.RemoveExchange("Bitstamp")
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received: %v but expected: %v", err, ErrExchangeNotFound)
	}
	if err := m.RemoveExchange("BiTFiNeX"); err != nil {
		t.Error("exchange should have been removed")
	}
	if m.Len() != 0 {
		t.Error("exchange manager len should be 0")
	}
}

func TestNewExchangeByName(t *testing.T) {
	m := SetupExchangeManager()
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
	m := SetupExchangeManager()
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
