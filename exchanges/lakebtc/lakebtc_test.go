package lakebtc

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var l LakeBTC

// Please add your own APIkeys to do correct due diligence testing.
const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	l.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	lakebtcConfig, err := cfg.GetExchangeConfig("LakeBTC")
	if err != nil {
		t.Error("Test Failed - LakeBTC Setup() init error")
	}

	lakebtcConfig.AuthenticatedAPISupport = true
	lakebtcConfig.APIKey = apiKey
	lakebtcConfig.APISecret = apiSecret

	l.Setup(lakebtcConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if l.GetFee(false) != 0.2 {
		t.Error("Test Failed - GetFee() error")
	}
	if l.GetFee(true) != 0.15 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := l.GetTicker()
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := l.GetOrderBook("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetOrderBook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := l.GetTradeHistory("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestTrade(t *testing.T) {
	t.Parallel()
	_, err := l.Trade(0, 0, 0, "USD")
	if err == nil {
		t.Error("Test Failed - Trade() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := l.GetOpenOrders()
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := l.GetOrders([]int64{1, 2})
	if err == nil {
		t.Error("Test Failed - GetOrders() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := l.CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := l.GetTrades(1337)
	if err == nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetExternalAccounts(t *testing.T) {
	t.Parallel()
	_, err := l.GetExternalAccounts()
	if err == nil {
		t.Error("Test Failed - GetExternalAccounts() error", err)
	}
}

func TestCreateWithdraw(t *testing.T) {
	t.Parallel()
	_, err := l.CreateWithdraw(0, 1337)
	if err == nil {
		t.Error("Test Failed - CreateWithdraw() error", err)
	}
}
