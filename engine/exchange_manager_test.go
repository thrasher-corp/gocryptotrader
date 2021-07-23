package engine

import (
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
)

func TestSetupExchangeManager(t *testing.T) {
	t.Parallel()
	m := SetupExchangeManager()
	if m == nil {
		t.Fatalf("unexpected response")
	}
	if m.exchanges == nil {
		t.Error("unexpected response")
	}
}

func TestExchangeManagerAdd(t *testing.T) {
	t.Parallel()
	m := SetupExchangeManager()
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	m.Add(b)
	if exch := m.GetExchanges(); exch[0].GetName() != "Bitfinex" {
		t.Error("unexpected exchange name")
	}
}

func TestExchangeManagerGetExchanges(t *testing.T) {
	t.Parallel()
	m := SetupExchangeManager()
	if exchanges := m.GetExchanges(); exchanges != nil {
		t.Error("unexpected value")
	}
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	m.Add(b)
	if exch := m.GetExchanges(); exch[0].GetName() != "Bitfinex" {
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
	if err := m.RemoveExchange("Bitstamp"); err != ErrExchangeNotFound {
		t.Error("Bitstamp exchange should return an error")
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
	exchanges := []string{"binance", "bitfinex", "bitflyer", "bithumb", "bitmex", "bitstamp", "bittrex", "btc markets", "btse", "coinbene", "coinut", "exmo", "coinbasepro", "ftx", "gateio", "gemini", "hitbtc", "huobi", "itbit", "kraken", "lbank", "localbitcoins", "okcoin international", "okex", "poloniex", "yobit", "zb", "fake"}
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
