package yobit

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var y Yobit

// Please supply your own keys for better unit testing
const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	y.SetDefaults()
}

func TestSetup(t *testing.T) {
	yobitConfig := config.GetConfig()
	yobitConfig.LoadConfig("../../testdata/configtest.json")
	conf, err := yobitConfig.GetExchangeConfig("Yobit")
	if err != nil {
		t.Error("Test Failed - Yobit init error")
	}
	conf.APIKey = apiKey
	conf.APISecret = apiSecret
	conf.AuthenticatedAPISupport = true

	y.Setup(conf)
}

func TestGetFee(t *testing.T) {
	if y.GetFee() != 0.2 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetInfo(t *testing.T) {
	_, err := y.GetInfo()
	if err != nil {
		t.Error("Test Failed - GetInfo() error")
	}
}

func TestGetTicker(t *testing.T) {
	_, err := y.GetTicker("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	_, err := y.GetDepth("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetDepth() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := y.GetTrades("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	_, err := y.GetAccountInfo()
	if err == nil {
		t.Error("Test Failed - GetAccountInfo() error", err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	_, err := y.GetActiveOrders("")
	if err == nil {
		t.Error("Test Failed - GetActiveOrders() error", err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	_, err := y.GetOrderInfo(6196974)
	if err == nil {
		t.Error("Test Failed - GetOrderInfo() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	_, err := y.CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestTrade(t *testing.T) {
	_, err := y.Trade("", "buy", 0, 0)
	if err == nil {
		t.Error("Test Failed - Trade() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	_, err := y.GetTradeHistory(0, 0, 0, "", "", "", "")
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestWithdrawCoinsToAddress(t *testing.T) {
	_, err := y.WithdrawCoinsToAddress("", 0, "")
	if err == nil {
		t.Error("Test Failed - WithdrawCoinsToAddress() error", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := y.GetDepositAddress("btc")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress() error", err)
	}
}

func TestCreateYobicode(t *testing.T) {
	_, err := y.CreateCoupon("bla", 0)
	if err == nil {
		t.Error("Test Failed - CreateYobicode() error", err)
	}
}

func TestRedeemYobicode(t *testing.T) {
	_, err := y.RedeemCoupon("bla")
	if err == nil {
		t.Error("Test Failed - RedeemYobicode() error", err)
	}
}
