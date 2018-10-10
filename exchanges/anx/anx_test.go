package anx

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var a ANX

func TestSetDefaults(t *testing.T) {
	a.SetDefaults()

	if a.Name != "ANX" {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.Enabled != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.TakerFee != 0.6 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.MakerFee != 0.3 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.Verbose != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if anx.Websocket.IsEnabled() != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.RESTPollingDelay != 10 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
}

func TestSetup(t *testing.T) {
	anxSetupConfig := config.GetConfig()
	anxSetupConfig.LoadConfig("../../testdata/configtest.json")
	anxConfig, err := anxSetupConfig.GetExchangeConfig("ANX")
	if err != nil {
		t.Error("Test Failed - ANX Setup() init error")
	}
	a.Setup(anxConfig)

	if a.Enabled != true {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if a.AuthenticatedAPISupport != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.APIKey) != 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.APISecret) != 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if a.RESTPollingDelay != 10 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if a.Verbose != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if anx.Websocket.IsEnabled() != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.BaseCurrencies) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.AvailablePairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.EnabledPairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := a.GetCurrencies()
	if err != nil {
		t.Fatalf("Test failed. TestGetCurrencies failed. Err: %s", err)
	}
}

func TestGetTradablePairs(t *testing.T) {
	_, err := a.GetTradablePairs()
	if err != nil {
		t.Fatalf("Test failed. TestGetTradablePairs failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	ticker, err := a.GetTicker("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetTicker() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetTicker() unsuccessful")
	}
}

func TestGetDepth(t *testing.T) {
	ticker, err := a.GetDepth("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetDepth() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetDepth() unsuccessful")
	}
}

func TestGetAPIKey(t *testing.T) {
	apiKey, apiSecret, err := a.GetAPIKey("userName", "passWord", "", "1337")
	if err == nil {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
	if apiKey != "" {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
	if apiSecret != "" {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	a.SetDefaults()
	TestSetup(t)

	if resp, err := a.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := a.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC, 100, 100, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := a.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 10000000000, -1000000000, true, true); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := a.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, true, false); resp != float64(0.020000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.100000), resp)
	}

	if resp, err := a.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, false, true); resp != float64(0.01000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.100000), resp)
	}

	if resp, err := a.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 10000000000, -1000000000, false, true); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := a.GetFee(exchange.CryptocurrencyWithdrawalFee, symbol.BTC, 1, 5, false, false); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
	}

	if resp, err := a.GetFee(exchange.CyptocurrencyDepositFee, symbol.BTC, 1, 0.001, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := a.GetFee(exchange.CyptocurrencyDepositFee, symbol.BTC, 1, 555, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := a.GetFee(exchange.InternationalBankDepositFee, symbol.BTC, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := a.GetFee(exchange.InternationalBankWithdrawalFee, symbol.HKD, 1, 1, false, false); resp != float64(250.01) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(250.01), resp)
	}

}
