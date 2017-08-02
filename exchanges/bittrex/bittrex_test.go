package bittrex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply you own test keys here to run better tests.
const (
	apiKey    = "TestKey"
	apiSecret = "TestKey"
)

func TestSetDefaults(t *testing.T) {
	b := Bittrex{}
	b.SetDefaults()
	if b.GetName() != "Bittrex" {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	exch := config.ExchangeConfig{
		Name:   "Bittrex",
		APIKey: apiKey,
	}
	exch.Enabled = true
	b := Bittrex{}
	b.Setup(exch)
	if b.APIKey != apiKey {
		t.Error("Test Failed - Bittrex - Setup() error")
	}
	exch.Enabled = false
	b.Setup(exch)
	if b.IsEnabled() {
		t.Error("Test Failed - Bittrex - Setup() error")
	}
}

func TestGetMarkets(t *testing.T) {
	obj := Bittrex{}
	_, err := obj.GetMarkets()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarkets() error: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	obj := Bittrex{}
	_, err := obj.GetCurrencies()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetCurrencies() error: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	invalid := ""
	btc := "btc-ltc"
	doge := "btc-DOGE"

	obj := Bittrex{}
	_, err := obj.GetTicker(invalid)
	if err == nil {
		t.Error("Test Failed - Bittrex - GetTicker() error")
	}
	_, err = obj.GetTicker(btc)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetTicker() error: %s", err)
	}
	_, err = obj.GetTicker(doge)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetTicker() error: %s", err)
	}
}

func TestGetMarketSummaries(t *testing.T) {
	obj := Bittrex{}
	_, err := obj.GetMarketSummaries()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketSummaries() error: %s", err)
	}
}

func TestGetMarketSummary(t *testing.T) {
	pairOne := "BTC-LTC"
	invalid := "WigWham"

	obj := Bittrex{}
	_, err := obj.GetMarketSummary(pairOne)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketSummary() error: %s", err)
	}
	_, err = obj.GetMarketSummary(invalid)
	if err == nil {
		t.Error("Test Failed - Bittrex - GetMarketSummary() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	obj := Bittrex{}
	value, err := obj.GetOrderbook("btc-ltc", "buy", 1)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetOrderbook() error: %s", err)
	}
	if len(value.Sell) > 0 {
		t.Error("Test Failed - Bittrex - GetOrderbook() error")
	}
	value, err = obj.GetOrderbook("btc-ltc", "sell", 1)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetOrderbook() error: %s", err)
	}
	if len(value.Buy) > 0 {
		t.Error("Test Failed - Bittrex - GetOrderbook() error")
	}
	_, err = obj.GetOrderbook("btc-ltc", "both", 1)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetOrderbook() error: %s", err)
	}
	_, err = obj.GetOrderbook("btc-ltc", "Whigwham", 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrderbook() error")
	}
	_, err = obj.GetOrderbook("btc-ltc", "Whigwham", 51)
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrderbook() error")
	}
	_, err = obj.GetOrderbook("wiggy", "both", 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrderbook() error")
	}
}

func TestGetMarketHistory(t *testing.T) {
	obj := Bittrex{}
	_, err := obj.GetMarketHistory("btc-ltc")
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketHistory() error: %s", err)
	}
	_, err = obj.GetMarketHistory("malum")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetMarketHistory() error")
	}
}

func TestPlaceBuyLimit(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.PlaceBuyLimit("btc-ltc", 1, 1)
	if err == nil {
		t.Errorf("Test Failed - Bittrex - PlaceBuyLimit() error")
	}
}

func TestPlaceSellLimit(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.PlaceSellLimit("btc-ltc", 1, 1)
	if err == nil {
		t.Errorf("Test Failed - Bittrex - PlaceSellLimit() error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetOpenOrders("")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetOrder() error")
	}
}

func TestCancelOrder(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.CancelOrder("blaaaaaaa")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - CancelOrder() error")
	}
}

func TestGetAccountBalances(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetAccountBalances()
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetAccountBalances() error")
	}
}

func TestGetAccountBalanceByCurrency(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetAccountBalanceByCurrency("btc")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetAccountBalanceByCurrency() error")
	}
}

func TestGetDepositAddress(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetDepositAddress("btc")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetDepositAddress() error")
	}
}

func TestWithdraw(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.Withdraw("btc", "something", "someplace", 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - Withdraw() error")
	}
}

func TestGetOrder(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetOrder("0cb4c4e4-bdc7-4e13-8c13-430e587d2cc1")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetOrder() error")
	}
	_, err = obj.GetOrder("")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetOrder() error")
	}
}

func TestGetOrderHistory(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetOrderHistory("")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetOrderHistory() error")
	}
	_, err = obj.GetOrderHistory("btc-ltc")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetOrderHistory() error")
	}
}

func TestGetWithdrawelHistory(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetWithdrawelHistory("")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetWithdrawelHistory() error")
	}
	_, err = obj.GetWithdrawelHistory("btc-ltc")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetWithdrawelHistory() error")
	}
}

func TestGetDepositHistory(t *testing.T) {
	obj := Bittrex{}
	obj.APIKey = apiKey
	obj.APISecret = apiSecret
	_, err := obj.GetDepositHistory("")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetDepositHistory() error")
	}
	_, err = obj.GetDepositHistory("btc-ltc")
	if err == nil {
		t.Errorf("Test Failed - Bittrex - GetDepositHistory() error")
	}
}
