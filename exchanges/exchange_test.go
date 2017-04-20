package exchange

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

func TestGetName(t *testing.T) {
	GetName := ExchangeBase{
		Name: "TESTNAME",
	}

	name := GetName.GetName()
	if name != "TESTNAME" {
		t.Error("Test Failed - Exchange getName() returned incorrect name")
	}
}

func TestGetEnabledCurrencies(t *testing.T) {
	enabledPairs := []string{"BTCUSD", "BTCAUD", "LTCUSD", "LTCAUD"}
	GetEnabledCurrencies := ExchangeBase{
		Name:         "TESTNAME",
		EnabledPairs: enabledPairs,
	}

	enCurr := GetEnabledCurrencies.GetEnabledCurrencies()
	if enCurr[0] != "BTCUSD" {
		t.Error("Test Failed - Exchange GetEnabledCurrencies() incorrect string")
	}
}

func TestSetEnabled(t *testing.T) {
	SetEnabled := ExchangeBase{
		Name:    "TESTNAME",
		Enabled: false,
	}

	SetEnabled.SetEnabled(true)
	if !SetEnabled.Enabled {
		t.Error("Test Failed - Exchange SetEnabled(true) did not set boolean")
	}
}

func TestIsEnabled(t *testing.T) {
	IsEnabled := ExchangeBase{
		Name:    "TESTNAME",
		Enabled: false,
	}

	if IsEnabled.IsEnabled() {
		t.Error("Test Failed - Exchange IsEnabled() did not return correct boolean")
	}
}

func TestSetAPIKeys(t *testing.T) {
	SetAPIKeys := ExchangeBase{
		Name:    "TESTNAME",
		Enabled: false,
	}

	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", false)

	if SetAPIKeys.APIKey != "RocketMan" && SetAPIKeys.APISecret != "Digereedoo" && SetAPIKeys.ClientID != "007" {
		t.Error("Test Failed - Exchange SetAPIKeys() did not set correct values")
	}

}

func TestUpdateAvailableCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.CONFIG_TEST_FILE)
	if err != nil {
		t.Log("SOMETHING DONE HAPPENED!")
	}

	UAC := ExchangeBase{
		Name: "ANX",
	}
	exchangeProducts := []string{"ltc", "btc", "usd", "aud"}

	err2 := UAC.UpdateAvailableCurrencies(exchangeProducts)
	if err2 != nil {
		t.Errorf("Test Failed - Exchange UpdateAvailableCurrencies() error: %s", err2)
	}
}
