package bittrex

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
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

	if !b.IsEnabled() || b.AuthenticatedAPISupport ||
		b.RESTPollingDelay != time.Duration(10) || b.Verbose ||
		b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 ||
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

func TestGetFee(t *testing.T) {
	t.Parallel()
	b.SetDefaults()
	TestSetup(t)

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, "BTC", 1, 1); resp != float64(0.002500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002500), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, "BTC", 10000000000, -1000000000); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}
	// This is an integration test, this value could change for any reason. So we only check for > 0
	if resp, err := b.GetFee(exchange.CryptocurrencyWithdrawalFee, "BTC", 1, 1); resp <= float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %s, Recieved: %f", "A value above 0", resp)
	}

	if resp, err := b.GetFee(exchange.CyptocurrencyDepositFee, "BTCUSD", 1, 1); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankDepositFee, "BTCUSD", 1, 1); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankDepositFee, "BTCUSD", 10000000, 100000); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankDepositFee, "BTCUSD", 10000000000, 1000000000); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankWithdrawalFee, "BTCUSD", 1, 1); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(15), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankWithdrawalFee, "BTCUSD", 10000000000, 1000000000); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}
}
