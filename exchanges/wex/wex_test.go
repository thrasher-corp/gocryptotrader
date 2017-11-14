package wex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var w WEX

// Please supply your own keys for better unit testing
const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	w.SetDefaults()
}

func TestSetup(t *testing.T) {
	wexConfig := config.GetConfig()
	wexConfig.LoadConfig("../../testdata/configtest.json")
	conf, err := wexConfig.GetExchangeConfig("WEX")
	if err != nil {
		t.Error("Test Failed - WEX init error")
	}
	conf.APIKey = apiKey
	conf.APISecret = apiSecret
	conf.AuthenticatedAPISupport = true

	w.Setup(conf)
}

func TestGetFee(t *testing.T) {
	if w.GetFee() != 0.2 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetInfo(t *testing.T) {
	_, err := w.GetInfo()
	if err != nil {
		t.Error("Test Failed - GetInfo() error")
	}
}

func TestGetTicker(t *testing.T) {
	_, err := w.GetTicker("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	_, err := w.GetDepth("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetDepth() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := w.GetTrades("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	_, err := w.GetAccountInfo()
	if err == nil {
		t.Error("Test Failed - GetAccountInfo() error", err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	_, err := w.GetActiveOrders("")
	if err == nil {
		t.Error("Test Failed - GetActiveOrders() error", err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	_, err := w.GetOrderInfo(6196974)
	if err == nil {
		t.Error("Test Failed - GetOrderInfo() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	_, err := w.CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestTrade(t *testing.T) {
	_, err := w.Trade("", "buy", 0, 0)
	if err == nil {
		t.Error("Test Failed - Trade() error", err)
	}
}

func TestGetTransactionHistory(t *testing.T) {
	_, err := w.GetTransactionHistory(0, 0, 0, "", "", "")
	if err == nil {
		t.Error("Test Failed - GetTransactionHistory() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	_, err := w.GetTradeHistory(0, 0, 0, "", "", "", "")
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestWithdrawCoins(t *testing.T) {
	_, err := w.WithdrawCoins("", 0, "")
	if err == nil {
		t.Error("Test Failed - WithdrawCoins() error", err)
	}
}

func TestCoinDepositAddress(t *testing.T) {
	_, err := w.CoinDepositAddress("btc")
	if err == nil {
		t.Error("Test Failed - WithdrawCoins() error", err)
	}
}

func TestCreateCoupon(t *testing.T) {
	_, err := w.CreateCoupon("bla", 0)
	if err == nil {
		t.Error("Test Failed - CreateCoupon() error", err)
	}
}

func TestRedeemCoupon(t *testing.T) {
	_, err := w.RedeemCoupon("bla")
	if err == nil {
		t.Error("Test Failed - RedeemCoupon() error", err)
	}
}
