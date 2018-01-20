package btcc

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply your own APIkeys here to do better tests
const (
	apiKey    = ""
	apiSecret = ""
)

var b BTCC

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	t.Parallel()
	b.Name = "BTCC"
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("BTCC")
	if err != nil {
		t.Error("Test Failed - BTCC Setup() init error")
	}

	b.SetDefaults()
	b.Setup(bConfig)

	if !b.IsEnabled() || b.AuthenticatedAPISupport || b.RESTPollingDelay != time.Duration(10) ||
		b.Verbose || b.Websocket || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - BTCC Setup values not set correctly")
	}

	bConfig.Enabled = false
	b.Setup(bConfig)

	if b.IsEnabled() {
		t.Error("Test failed - BTCC TestSetup incorrect value")
	}
}

func TestGetFee(t *testing.T) {
	if b.GetFee() != 0 {
		t.Error("Test failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	_, err := b.GetTicker("BTCUSD")
	if err != nil {
		t.Error("Test failed - GetTicker() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	_, err := b.GetTradeHistory("BTCUSD", 0, 0, time.Time{})
	if err != nil {
		t.Error("Test failed - GetTradeHistory() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	b.Verbose = true
	_, err := b.GetOrderBook("BTCUSD", 100)
	if err != nil {
		t.Error("Test failed - GetOrderBook() error", err)
	}
	_, err = b.GetOrderBook("BTCUSD", 0)
	if err != nil {
		t.Error("Test failed - GetOrderBook() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	err := b.GetAccountInfo("")
	if err == nil {
		t.Error("Test failed - GetAccountInfo() error", err)
	}
}
