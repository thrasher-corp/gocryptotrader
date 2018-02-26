package bitfinex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func TestStart(t *testing.T) {
	start := Bitfinex{}
	start.Start()
}

func TestRun(t *testing.T) {
	run := Bitfinex{}
	run.Run()
}

func TestGetTickerPrice(t *testing.T) {
	getTickerPrice := Bitfinex{}
	getTickerPrice.EnabledPairs = []string{"BTCUSD", "LTCUSD"}
	_, err := getTickerPrice.GetTickerPrice(pair.NewCurrencyPair("BTC", "USD"),
		ticker.Spot)
	if err != nil {
		t.Errorf("Test Failed - Bitfinex GetTickerPrice() error: %s", err)
	}
}

func TestGetOrderbookEx(t *testing.T) {
	getOrderBookEx := Bitfinex{}
	_, err := getOrderBookEx.GetOrderbookEx(pair.NewCurrencyPair("BTC", "USD"),
		ticker.Spot)
	if err != nil {
		t.Errorf("Test Failed - Bitfinex GetOrderbookEx() error: %s", err)
	}
}
