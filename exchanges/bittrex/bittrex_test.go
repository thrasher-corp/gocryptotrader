package bittrex

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply you own test keys here to run better tests.
const (
	apiKey    = "Testy"
	apiSecret = "TestyTesty"
)

var b Bittrex

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
	if b.GetName() != "Bittrex" {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("Bittrex")
	if err != nil {
		t.Error("Test Failed - Bittrex Setup() init error")
	}

	b.Setup(bConfig)

	if !b.IsEnabled() || b.AuthenticatedAPISupport || b.RESTPollingDelay != time.Duration(10) ||
		b.Verbose || b.Websocket || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - Bittrex Setup values not set correctly")
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarkets() error: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := b.GetCurrencies()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetCurrencies() error: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	btc := "btc-ltc"

	_, err := b.GetTicker(btc)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetTicker() error: %s", err)
	}
}

func TestGetMarketSummaries(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketSummaries()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketSummaries() error: %s", err)
	}
}

func TestGetMarketSummary(t *testing.T) {
	t.Parallel()
	pairOne := "BTC-LTC"

	_, err := b.GetMarketSummary(pairOne)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketSummary() error: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook("btc-ltc")
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetOrderbook() error: %s", err)
	}
}

func TestGetMarketHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetMarketHistory("btc-ltc")
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketHistory() error: %s", err)
	}
}

func TestPlaceBuyLimit(t *testing.T) {
	t.Parallel()

	_, err := b.PlaceBuyLimit("btc-ltc", 1, 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - PlaceBuyLimit() error")
	}
}

func TestPlaceSellLimit(t *testing.T) {
	t.Parallel()

	_, err := b.PlaceSellLimit("btc-ltc", 1, 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - PlaceSellLimit() error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
	_, err = b.GetOpenOrders("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := b.CancelOrder("blaaaaaaa")
	if err == nil {
		t.Error("Test Failed - Bittrex - CancelOrder() error")
	}
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalances()
	if err == nil {
		t.Error("Test Failed - Bittrex - GetAccountBalances() error")
	}
}

func TestGetAccountBalanceByCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalanceByCurrency("btc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetAccountBalanceByCurrency() error")
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositAddress("btc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetDepositAddress() error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	_, err := b.Withdraw("btc", "something", "someplace", 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - Withdraw() error")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrder("0cb4c4e4-bdc7-4e13-8c13-430e587d2cc1")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
	_, err = b.GetOrder("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderHistory("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrderHistory() error")
	}
	_, err = b.GetOrderHistory("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrderHistory() error")
	}
}

func TestGetwithdrawalHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetWithdrawalHistory("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetWithdrawalHistory() error")
	}
	_, err = b.GetWithdrawalHistory("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetWithdrawalHistory() error")
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositHistory("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetDepositHistory() error")
	}
	_, err = b.GetDepositHistory("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetDepositHistory() error")
	}
}
