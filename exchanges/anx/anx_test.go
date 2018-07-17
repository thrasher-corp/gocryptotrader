package anx

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var anx ANX

func TestSetDefaults(t *testing.T) {
	anx.SetDefaults()

	if anx.Name != "ANX" {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if anx.Enabled != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if anx.TakerFee != 0.6 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if anx.MakerFee != 0.3 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if anx.Verbose != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if anx.Websocket != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if anx.RESTPollingDelay != 10 {
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
	anx.Setup(anxConfig)

	if anx.Enabled != true {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if anx.AuthenticatedAPISupport != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(anx.APIKey) != 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(anx.APISecret) != 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if anx.RESTPollingDelay != 10 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if anx.Verbose != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if anx.Websocket != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(anx.BaseCurrencies) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(anx.AvailablePairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(anx.EnabledPairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := anx.GetCurrencies()
	if err != nil {
		t.Fatalf("Test failed. TestGetCurrencies failed. Err: %s", err)
	}
}

func TestGetTradablePairs(t *testing.T) {
	_, err := anx.GetTradablePairs()
	if err != nil {
		t.Fatalf("Test failed. TestGetTradablePairs failed. Err: %s", err)
	}
}

func TestGetFee(t *testing.T) {
	makerFeeExpected, takerFeeExpected := 0.3, 0.6

	if anx.GetFee(true) != makerFeeExpected {
		t.Error("Test Failed - ANX GetFee() incorrect return value")
	}
	if anx.GetFee(false) != takerFeeExpected {
		t.Error("Test Failed - ANX GetFee() incorrect return value")
	}
}

func TestGetTicker(t *testing.T) {
	ticker, err := anx.GetTicker("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetTicker() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetTicker() unsuccessful")
	}
}

func TestGetDepth(t *testing.T) {
	ticker, err := anx.GetDepth("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetDepth() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetDepth() unsuccessful")
	}
}

func TestGetAPIKey(t *testing.T) {
	apiKey, apiSecret, err := anx.GetAPIKey("userName", "passWord", "", "1337")
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
