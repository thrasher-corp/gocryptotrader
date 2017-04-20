package bitfinex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
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
	_, err := getTickerPrice.GetTickerPrice(pair.NewCurrencyPair("BTC", "USD"))
	if err != nil {
		t.Errorf("Test Failed - Bitfinex GetTickerPrice() error: %s", err)
	}
}

func TestGetOrderbookEx(t *testing.T) {
	getOrderBookEx := Bitfinex{}
	_, err := getOrderBookEx.GetOrderbookEx(pair.NewCurrencyPair("BTC", "USD"))
	if err != nil {
		t.Errorf("Test Failed - Bitfinex GetOrderbookEx() error: %s", err)
	}
}

func TestGetExchangeAccountInfo(t *testing.T) {
	getExchangeAccountInfo := Bitfinex{}
	newConfig := config.GetConfig()
	newConfig.LoadConfig("../../testdata/configtest.dat")
	exchConf, err := newConfig.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Errorf("Test Failed - Bitfinex getExchangeConfig(): %s", err)
	}
	getExchangeAccountInfo.Setup(exchConf)
	_, err = getExchangeAccountInfo.GetExchangeAccountInfo()
	if err != nil {
		t.Errorf("Test Failed - Bitfinex GetExchangeAccountInfo() error: %s", err)
	}
}
