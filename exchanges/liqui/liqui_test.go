package liqui

import (
	"log"
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var l Liqui

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
	liquiConfig, err := cfg.GetExchangeConfig("Liqui")
	if err != nil {
		t.Error("Test Failed - liqui Setup() init error")
	}

	liquiConfig.AuthenticatedAPISupport = true
	liquiConfig.APIKey = apiKey
	liquiConfig.APISecret = apiSecret

	l.Setup(liquiConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := l.GetFee("usd")
	if err == nil {
		t.Error("Test Failed - liqui GetFee() error", err)
	}
}

func TestGetAvailablePairs(t *testing.T) {
	t.Parallel()
	v := l.GetAvailablePairs(false)
	if len(v) != 0 {
		t.Error("Test Failed - liqui GetFee() error")
	}
}

func TestGetInfo(t *testing.T) {
	t.Parallel()
	_, err := l.GetInfo()
	if err != nil {
		t.Error("Test Failed - liqui GetInfo() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := l.GetTicker("eth_btc")
	if err != nil {
		t.Error("Test Failed - liqui GetTicker() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	v, err := l.GetDepth("eth_btc")
	if err != nil {
		t.Error("Test Failed - liqui GetDepth() error", err)
	}
	log.Println(v)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := l.GetTrades("eth_btc")
	if err != nil {
		t.Error("Test Failed - liqui GetTrades() error", err)
	}
}

func TestAuthRequests(t *testing.T) {
	if l.APIKey != "" && l.APISecret != "" {
		_, err := l.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - liqui GetAccountInfo() error", err)
		}

		_, err = l.Trade("", "", 0, 1)
		if err == nil {
			t.Error("Test Failed - liqui Trade() error", err)
		}

		_, err = l.GetActiveOrders("eth_btc")
		if err == nil {
			t.Error("Test Failed - liqui GetActiveOrders() error", err)
		}

		_, err = l.GetOrderInfo(1337)
		if err == nil {
			t.Error("Test Failed - liqui GetOrderInfo() error", err)
		}

		_, err = l.CancelOrder(1337)
		if err == nil {
			t.Error("Test Failed - liqui CancelOrder() error", err)
		}

		_, err = l.GetTradeHistory(url.Values{}, "")
		if err == nil {
			t.Error("Test Failed - liqui GetTradeHistory() error", err)
		}

		_, err = l.WithdrawCoins("btc", 1337, "someaddr")
		if err == nil {
			t.Error("Test Failed - liqui WithdrawCoins() error", err)
		}
	}
}
