package config

import (
	"testing"
)

func TestGetConfigEnabledExchanges(t *testing.T) {
	defaultEnabledExchanges := 17
	GetConfigEnabledExchanges := GetConfig()
	err := GetConfigEnabledExchanges.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error(
			"Test failed. GetConfigEnabledExchanges load config error: " + err.Error(),
		)
	}
	enabledExch := GetConfigEnabledExchanges.GetConfigEnabledExchanges()
	if enabledExch != defaultEnabledExchanges {
		t.Error("Test failed. GetConfigEnabledExchanges is wrong")
	}
}

func TestGetCurrencyPairDisplayConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. GetCurrencyPairDisplayConfig. LoadConfig Error: %s", err.Error(),
		)
	}
	settings := cfg.GetCurrencyPairDisplayConfig()
	if settings.Delimiter != "-" || !settings.Uppercase {
		t.Errorf(
			"Test failed. GetCurrencyPairDisplayConfi. Invalid values",
		)
	}
}

func TestGetExchangeConfig(t *testing.T) {
	GetExchangeConfig := GetConfig()
	err := GetExchangeConfig.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. GetExchangeConfig.LoadConfig Error: %s", err.Error(),
		)
	}
	r, err := GetExchangeConfig.GetExchangeConfig("ANX")
	if err != nil && (ExchangeConfig{}) == r {
		t.Errorf(
			"Test failed. GetExchangeConfig.GetExchangeConfig Error: %s", err.Error(),
		)
	}
	r, err = GetExchangeConfig.GetExchangeConfig("Testy")
	if err == nil && (ExchangeConfig{}) == r {
		t.Error("Test failed. GetExchangeConfig.GetExchangeConfig Error")
	}
}

func TestUpdateExchangeConfig(t *testing.T) {
	UpdateExchangeConfig := GetConfig()
	err := UpdateExchangeConfig.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. UpdateExchangeConfig.LoadConfig Error: %s", err.Error(),
		)
	}
	e, err2 := UpdateExchangeConfig.GetExchangeConfig("ANX")
	if err2 != nil {
		t.Errorf(
			"Test failed. UpdateExchangeConfig.GetExchangeConfig: %s", err.Error(),
		)
	}
	e.APIKey = "test1234"
	err3 := UpdateExchangeConfig.UpdateExchangeConfig(e)
	if err3 != nil {
		t.Errorf(
			"Test failed. UpdateExchangeConfig.UpdateExchangeConfig: %s", err.Error(),
		)
	}
	e.Name = "testyTest"
	err = UpdateExchangeConfig.UpdateExchangeConfig(e)
	if err == nil {
		t.Error("Test failed. UpdateExchangeConfig.UpdateExchangeConfig Error")
	}
}

func TestCheckSMSGlobalConfigValues(t *testing.T) {
	t.Parallel()

	checkSMSGlobalConfigValues := GetConfig()
	err := checkSMSGlobalConfigValues.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. checkSMSGlobalConfigValues.LoadConfig: %s", err)
	}
	err = checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues()
	if err != nil {
		t.Error(
			`Test failed. checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues: Incorrect Return Value`,
		)
	}

	checkSMSGlobalConfigValues.SMS.Username = "Username"
	err = checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues()
	if err == nil {
		t.Error(
			"Test failed. checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues: Incorrect Return Value",
		)
	}

	checkSMSGlobalConfigValues.SMS.Username = "1234"
	checkSMSGlobalConfigValues.SMS.Contacts[0].Name = "Bob"
	checkSMSGlobalConfigValues.SMS.Contacts[0].Number = "12345"
	err = checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues()
	if err == nil {
		t.Error(
			"Test failed. checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues: Incorrect Return Value",
		)
	}
	checkSMSGlobalConfigValues.SMS.Contacts = checkSMSGlobalConfigValues.SMS.Contacts[:0]
	err = checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues()
	if err == nil {
		t.Error(
			"Test failed. checkSMSGlobalConfigValues.CheckSMSGlobalConfigValues: Incorrect Return Value",
		)
	}
}

func TestCheckExchangeConfigValues(t *testing.T) {
	t.Parallel()
	checkExchangeConfigValues := Config{}

	err := checkExchangeConfigValues.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.LoadConfig: %s", err.Error(),
		)
	}
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues: %s",
			err.Error(),
		)
	}

	checkExchangeConfigValues.Exchanges[0].APIKey = "Key"
	checkExchangeConfigValues.Exchanges[0].APISecret = "Secret"
	checkExchangeConfigValues.Exchanges[0].AuthenticatedAPISupport = true
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges[0].AuthenticatedAPISupport = true
	checkExchangeConfigValues.Exchanges[0].APIKey = "TESTYTEST"
	checkExchangeConfigValues.Exchanges[0].APISecret = "TESTYTEST"
	checkExchangeConfigValues.Exchanges[0].Name = "ITBIT"
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges[0].BaseCurrencies = ""
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err == nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges[0].EnabledPairs = ""
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err == nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges[0].AvailablePairs = ""
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err == nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges[0].Name = ""
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err == nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Cryptocurrencies = ""
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err == nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges = checkExchangeConfigValues.Exchanges[:0]
	checkExchangeConfigValues.Cryptocurrencies = "TESTYTEST"
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err == nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}
}

func TestCheckWebserverConfigValues(t *testing.T) {
	checkWebserverConfigValues := GetConfig()
	err := checkWebserverConfigValues.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. checkWebserverConfigValues.LoadConfig: %s", err.Error(),
		)
	}
	err = checkWebserverConfigValues.CheckWebserverConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues: %s",
			err.Error(),
		)
	}

	checkWebserverConfigValues.Webserver.ListenAddress = ":0"
	err = checkWebserverConfigValues.CheckWebserverConfigValues()
	if err == nil {
		t.Error(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues error",
		)
	}

	checkWebserverConfigValues.Webserver.ListenAddress = ":LOLOLOL"
	err = checkWebserverConfigValues.CheckWebserverConfigValues()
	if err == nil {
		t.Error(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues error",
		)
	}

	checkWebserverConfigValues.Webserver.ListenAddress = "LOLOLOL"
	err = checkWebserverConfigValues.CheckWebserverConfigValues()
	if err == nil {
		t.Error(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues error",
		)
	}

	checkWebserverConfigValues.Webserver.AdminUsername = ""
	err = checkWebserverConfigValues.CheckWebserverConfigValues()
	if err == nil {
		t.Error(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues error",
		)
	}
}

func TestRetrieveConfigCurrencyPairs(t *testing.T) {
	retrieveConfigCurrencyPairs := GetConfig()
	err := retrieveConfigCurrencyPairs.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. checkWebserverConfigValues.LoadConfig: %s", err.Error(),
		)
	}
	err = retrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs()
	if err != nil {
		t.Errorf(
			"Test failed. checkWebserverConfigValues.RetrieveConfigCurrencyPairs: %s",
			err.Error(),
		)
	}
}

func TestReadConfig(t *testing.T) {
	readConfig := GetConfig()
	err := readConfig.ReadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. TestReadConfig %s", err.Error())
	}

	err = readConfig.ReadConfig("bla")
	if err == nil {
		t.Error("Test failed. TestReadConfig " + err.Error())
	}

	err = readConfig.ReadConfig("")
	if err != nil {
		t.Error("Test failed. TestReadConfig error")
	}
}

func TestLoadConfig(t *testing.T) {
	loadConfig := GetConfig()
	err := loadConfig.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. TestLoadConfig " + err.Error())
	}

	err = loadConfig.LoadConfig("testy")
	if err == nil {
		t.Error("Test failed. TestLoadConfig ")
	}
}

func TestSaveConfig(t *testing.T) {
	saveConfig := GetConfig()
	err := saveConfig.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. TestSaveConfig.LoadConfig: %s", err.Error())
	}
	err2 := saveConfig.SaveConfig(ConfigTestFile)
	if err2 != nil {
		t.Errorf("Test failed. TestSaveConfig.SaveConfig, %s", err2.Error())
	}
}

func TestGetFilePath(t *testing.T) {
	expected := "blah.json"
	result := GetFilePath("blah.json")
	if result != "blah.json" {
		t.Errorf("Test failed. TestGetFilePath: expected %s got %s", expected, result)
	}

	expected = ConfigTestFile
	result = GetFilePath("")
	if result != expected {
		t.Errorf("Test failed. TestGetFilePath: expected %s got %s", expected, result)
	}
}
