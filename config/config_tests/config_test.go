package test

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

func TestGetConfigEnabledExchanges(t *testing.T) {
	t.Parallel()

	defaultEnabledExchanges := 17
	GetConfigEnabledExchanges := config.GetConfig()
	err := GetConfigEnabledExchanges.LoadConfig()
	if err != nil {
		t.Error("Test failed. GetConfigEnabledExchanges load config error: " + err.Error())
	}
	enabledExch := GetConfigEnabledExchanges.GetConfigEnabledExchanges()
	if enabledExch != defaultEnabledExchanges {
		t.Error("Test failed. GetConfigEnabledExchanges is wrong")
	}
}

func TestGetExchangeConfig(t *testing.T) {
	t.Parallel()

	GetExchangeConfig := config.GetConfig()
	err := GetExchangeConfig.LoadConfig()
	if err != nil {
		t.Errorf("Test failed. GetExchangeConfig.LoadConfig Error: %s", err.Error())
	}
	r, err := GetExchangeConfig.GetExchangeConfig("ANX")
	if err != nil && (config.ExchangeConfig{}) == r {
		t.Errorf("Test failed. GetExchangeConfig.GetExchangeConfig Error: %s", err.Error())
	}
}

func TestUpdateExchangeConfig(t *testing.T) {
	t.Parallel()

	UpdateExchangeConfig := config.GetConfig()
	err := UpdateExchangeConfig.LoadConfig()
	if err != nil {
		t.Errorf("Test failed. UpdateExchangeConfig.LoadConfig Error: %s", err.Error())
	}
	e, err2 := UpdateExchangeConfig.GetExchangeConfig("ANX")
	if err2 != nil {
		t.Errorf("Test failed. UpdateExchangeConfig.GetExchangeConfig: %s", err.Error())
	}
	e.APIKey = "test1234"
	err3 := UpdateExchangeConfig.UpdateExchangeConfig(e)
	if err3 != nil {
		t.Errorf("Test failed. UpdateExchangeConfig.UpdateExchangeConfig: %s", err.Error())
	}
}

func TestCheckSMSGlobalConfigValues(t *testing.T) {
	t.Parallel()

	checkSMSGlobalConfigValues := config.GetConfig()
	err := checkSMSGlobalConfigValues.LoadConfig()
	if err != nil {
		t.Errorf("Test failed. checkSMSGlobalConfigValues.LoadConfig: %s", err)
	}
	err2 := checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues()
	if err2 == nil {
		t.Error("Test failed. checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues: Incorrect Return Value")
	}
}

func TestCheckExchangeConfigValues(t *testing.T) {
	t.Parallel()

	checkExchangeConfigValues := config.Config{}
	err := checkExchangeConfigValues.LoadConfig()
	if err != nil {
		t.Errorf("Test failed. checkExchangeConfigValues.LoadConfig: %s", err.Error())
	}

	err3 := checkExchangeConfigValues.CheckExchangeConfigValues()
	if err3 != nil {
		t.Errorf("Test failed. checkExchangeConfigValues.CheckExchangeConfigValues: %s", err.Error())
	}
}

func TestCheckWebserverConfigValues(t *testing.T) {
	t.Parallel()

	checkWebserverConfigValues := config.GetConfig()
	err := checkWebserverConfigValues.LoadConfig()
	if err != nil {
		t.Errorf("Test failed. checkWebserverConfigValues.LoadConfig: %s", err.Error())
	}
	err2 := checkWebserverConfigValues.CheckWebserverConfigValues()
	if err2 != nil {
		t.Errorf("Test failed. checkWebserverConfigValues.CheckWebserverConfigValues: %s", err2.Error())
	}
}

func TestRetrieveConfigCurrencyPairs(t *testing.T) {
	t.Parallel()

	retrieveConfigCurrencyPairs := config.GetConfig()
	err := retrieveConfigCurrencyPairs.LoadConfig()
	if err != nil {
		t.Errorf("Test failed. checkWebserverConfigValues.LoadConfig: %s", err.Error())
	}
	err2 := retrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs()
	if err2 != nil {
		t.Errorf("Test failed. checkWebserverConfigValues.RetrieveConfigCurrencyPairs: %s", err2.Error())
	}
}

func TestReadConfig(t *testing.T) {
	t.Parallel()

	readConfig := config.GetConfig()
	err := readConfig.ReadConfig()
	if err != nil {
		t.Error("Test failed. TestReadConfig " + err.Error())
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	loadConfig := config.GetConfig()
	err := loadConfig.LoadConfig()
	if err != nil {
		t.Error("Test failed. TestLoadConfig " + err.Error())
	}
}

func TestSaveConfig(t *testing.T) {
	t.Parallel()

	saveConfig := config.GetConfig()
	err := saveConfig.LoadConfig()
	if err != nil {
		t.Errorf("Test failed. TestSaveConfig.LoadConfig: %s", err.Error())
	}
	err2 := saveConfig.SaveConfig()
	if err2 != nil {
		t.Error("Test failed. TestSaveConfig.SaveConfig, %s", err2.Error())
	}
}
