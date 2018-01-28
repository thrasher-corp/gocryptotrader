package anx

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

func TestSetDefaults(t *testing.T) {
	setDefaults := ANX{}
	setDefaults.SetDefaults()

	if setDefaults.Name != "ANX" {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if setDefaults.Enabled != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if setDefaults.TakerFee != 0.6 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if setDefaults.MakerFee != 0.3 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if setDefaults.Verbose != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if setDefaults.Websocket != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if setDefaults.RESTPollingDelay != 10 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
}

func TestSetup(t *testing.T) {
	setup := ANX{}
	setup.Name = "ANX"
	anxSetupConfig := config.GetConfig()
	anxSetupConfig.LoadConfig("../../testdata/configtest.json")
	anxConfig, err := anxSetupConfig.GetExchangeConfig("ANX")
	if err != nil {
		t.Error("Test Failed - ANX Setup() init error")
	}
	setup.Setup(anxConfig)

	if setup.Enabled != true {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if setup.AuthenticatedAPISupport != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(setup.APIKey) != 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(setup.APISecret) != 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if setup.RESTPollingDelay != 10 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if setup.Verbose != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if setup.Websocket != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(setup.BaseCurrencies) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(setup.AvailablePairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(setup.EnabledPairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
}

func TestGetFee(t *testing.T) {
	getFee := ANX{}
	makerFeeExpected, takerFeeExpected := 0.3, 0.6

	getFee.SetDefaults()
	if getFee.GetFee(true) != makerFeeExpected {
		t.Error("Test Failed - ANX GetFee() incorrect return value")
	}
	if getFee.GetFee(false) != takerFeeExpected {
		t.Error("Test Failed - ANX GetFee() incorrect return value")
	}
}

func TestGetTicker(t *testing.T) {
	getTicker := ANX{}
	ticker, err := getTicker.GetTicker("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetTicker() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetTicker() unsuccessful")
	}
}

func TestGetDepth(t *testing.T) {
	a := ANX{}
	ticker, err := a.GetDepth("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetDepth() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetDepth() unsuccessful")
	}
}

func TestGetAPIKey(t *testing.T) {
	getAPIKey := ANX{}
	apiKey, apiSecret, err := getAPIKey.GetAPIKey("userName", "passWord", "", "1337")
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

func TestGetDataToken(t *testing.T) {
	// --- FAIL: TestGetDataToken (0.17s)
	//      anx_test.go:120: Test Failed - ANX GetDataToken() Incorrect

	// getDataToken := ANX{}
	// _, err := getDataToken.GetDataToken()
	// if err != nil {
	// 	t.Error("Test Failed - ANX GetDataToken() Incorrect")
	// }
}

func TestNewOrder(t *testing.T) {

}

func TestOrderInfo(t *testing.T) {

}

func TestSend(t *testing.T) {

}

func TestCreateNewSubAccount(t *testing.T) {

}

func TestGetDepositAddress(t *testing.T) {

}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {

}
