package config

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestGetCurrencyConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetCurrencyConfig LoadConfig error", err)
	}
	_ = cfg.GetCurrencyConfig()
}

func TestGetExchangeBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetDepositBankAccounts LoadConfig error", err)
	}
	_, err = cfg.GetExchangeBankAccounts("Bitfinex", "USD")
	if err != nil {
		t.Error("Test failed. GetDepositBankAccounts error", err)
	}
}

func TestUpdateExchangeBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. UpdateDepositBankAccounts LoadConfig error", err)
	}

	b := []BankAccount{{Enabled: false}}
	err = cfg.UpdateExchangeBankAccounts("Bitfinex", b)
	if err != nil {
		t.Error("Test failed. UpdateDepositBankAccounts error", err)
	}
	var count int
	for _, exch := range cfg.Exchanges {
		if exch.Name == "Bitfinex" {
			if exch.BankAccounts[0].Enabled == false {
				count++
			}
		}
	}
	if count != 1 {
		t.Error("Test failed. UpdateDepositBankAccounts error")
	}
}

func TestGetClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetClientBankAccounts LoadConfig error", err)
	}
	_, err = cfg.GetClientBankAccounts("Kraken", "USD")
	if err != nil {
		t.Error("Test failed. GetClientBankAccounts error", err)
	}
	_, err = cfg.GetClientBankAccounts("Bla", "USD")
	if err == nil {
		t.Error("Test failed. GetClientBankAccounts error")
	}
	_, err = cfg.GetClientBankAccounts("Kraken", "JPY")
	if err == nil {
		t.Error("Test failed. GetClientBankAccounts error", err)
	}
}

func TestUpdateClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. UpdateClientBankAccounts LoadConfig error", err)
	}
	b := BankAccount{Enabled: false, BankName: "test", AccountNumber: "0234"}
	err = cfg.UpdateClientBankAccounts(b)
	if err != nil {
		t.Error("Test failed. UpdateClientBankAccounts error", err)
	}

	err = cfg.UpdateClientBankAccounts(BankAccount{})
	if err == nil {
		t.Error("Test failed. UpdateClientBankAccounts error")
	}

	var count int
	for _, bank := range cfg.BankAccounts {
		if bank.BankName == b.BankName {
			if bank.Enabled == false {
				count++
			}
		}
	}
	if count != 1 {
		t.Error("Test failed. UpdateDepositBankAccounts error")
	}
}

func TestGetCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetCommunicationsConfig LoadConfig error", err)
	}
	_ = cfg.GetCommunicationsConfig()
}

func TestUpdateCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. UpdateCommunicationsConfig LoadConfig error", err)
	}
	cfg.UpdateCommunicationsConfig(CommunicationsConfig{SlackConfig: SlackConfig{Name: "TEST"}})
	if cfg.Communications.SlackConfig.Name != "TEST" {
		t.Error("Test failed. UpdateCommunicationsConfig LoadConfig error")
	}
}

func TestSupportsPair(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestSupportsPair. LoadConfig Error: %s", err.Error(),
		)
	}

	_, err = cfg.SupportsPair("asdf", pair.NewCurrencyPair("BTC", "USD"))
	if err == nil {
		t.Error(
			"Test failed. TestSupportsPair. Non-existent exchange returned nil error",
		)
	}

	_, err = cfg.SupportsPair("Bitfinex", pair.NewCurrencyPair("BTC", "USD"))
	if err != nil {
		t.Errorf(
			"Test failed. TestSupportsPair. Incorrect values. Err: %s", err,
		)
	}
}

func TestGetAvailablePairs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetAvailablePairs. LoadConfig Error: %s", err.Error(),
		)
	}

	_, err = cfg.GetAvailablePairs("asdf")
	if err == nil {
		t.Error(
			"Test failed. TestGetAvailablePairs. Non-existent exchange returned nil error",
		)
	}

	_, err = cfg.GetAvailablePairs("Bitfinex")
	if err != nil {
		t.Errorf(
			"Test failed. TestGetAvailablePairs. Incorrect values. Err: %s", err,
		)
	}
}

func TestGetEnabledPairs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetEnabledPairs. LoadConfig Error: %s", err.Error(),
		)
	}

	_, err = cfg.GetEnabledPairs("asdf")
	if err == nil {
		t.Error(
			"Test failed. TestGetEnabledPairs. Non-existent exchange returned nil error",
		)
	}

	_, err = cfg.GetEnabledPairs("Bitfinex")
	if err != nil {
		t.Errorf(
			"Test failed. TestGetEnabledPairs. Incorrect values. Err: %s", err,
		)
	}
}

func TestGetEnabledExchanges(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetEnabledExchanges. LoadConfig Error: %s", err.Error(),
		)
	}

	exchanges := cfg.GetEnabledExchanges()
	if len(exchanges) != 30 {
		t.Error(
			"Test failed. TestGetEnabledExchanges. Enabled exchanges value mismatch",
		)
	}

	if !common.StringDataCompare(exchanges, "Bitfinex") {
		t.Error(
			"Test failed. TestGetEnabledExchanges. Expected exchange Bitfinex not found",
		)
	}
}

func TestGetDisabledExchanges(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetDisabledExchanges. LoadConfig Error: %s", err.Error(),
		)
	}

	exchanges := cfg.GetDisabledExchanges()
	if len(exchanges) != 0 {
		t.Error(
			"Test failed. TestGetDisabledExchanges. Enabled exchanges value mismatch",
		)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Errorf(
			"Test failed. TestGetDisabledExchanges. GetExchangeConfig Error: %s", err.Error(),
		)
	}

	exchCfg.Enabled = false
	err = cfg.UpdateExchangeConfig(exchCfg)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetDisabledExchanges. UpdateExchangeConfig Error: %s", err.Error(),
		)
	}

	if len(cfg.GetDisabledExchanges()) != 1 {
		t.Error(
			"Test failed. TestGetDisabledExchanges. Enabled exchanges value mismatch",
		)
	}
}

func TestCountEnabledExchanges(t *testing.T) {
	defaultEnabledExchanges := 30
	GetConfigEnabledExchanges := GetConfig()
	err := GetConfigEnabledExchanges.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error(
			"Test failed. GetConfigEnabledExchanges load config error: " + err.Error(),
		)
	}
	enabledExch := GetConfigEnabledExchanges.CountEnabledExchanges()
	if enabledExch != defaultEnabledExchanges {
		t.Error("Test failed. GetConfigEnabledExchanges is wrong")
	}
}

func TestGetConfigCurrencyPairFormat(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetConfigCurrencyPairFormat. LoadConfig Error: %s", err.Error(),
		)
	}
	_, err = cfg.GetConfigCurrencyPairFormat("asdasdasd")
	if err == nil {
		t.Errorf(
			"Test failed. TestGetRequestCurrencyPairFormat. Non-existent exchange returned nil error",
		)
	}

	exchFmt, err := cfg.GetConfigCurrencyPairFormat("Liqui")
	if !exchFmt.Uppercase || exchFmt.Delimiter != "_" {
		t.Errorf(
			"Test failed. TestGetConfigCurrencyPairFormat. Invalid values",
		)
	}
}

func TestGetRequestCurrencyPairFormat(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetRequestCurrencyPairFormat. LoadConfig Error: %s", err.Error(),
		)
	}

	_, err = cfg.GetRequestCurrencyPairFormat("asdasdasd")
	if err == nil {
		t.Errorf(
			"Test failed. TestGetRequestCurrencyPairFormat. Non-existent exchange returned nil error",
		)
	}

	exchFmt, err := cfg.GetRequestCurrencyPairFormat("Liqui")
	if exchFmt.Uppercase || exchFmt.Delimiter != "_" || exchFmt.Separator != "-" {
		t.Errorf(
			"Test failed. TestGetRequestCurrencyPairFormat. Invalid values",
		)
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

func TestGetAllExchangeConfigs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetAllExchangeConfigs. LoadConfig error", err)
	}
	if len(cfg.GetAllExchangeConfigs()) < 26 {
		t.Error("Test failed. GetAllExchangeConfigs error")
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
	_, err = GetExchangeConfig.GetExchangeConfig("ANX")
	if err != nil {
		t.Errorf("Test failed. GetExchangeConfig.GetExchangeConfig Error: %s",
			err.Error())
	}
	_, err = GetExchangeConfig.GetExchangeConfig("Testy")
	if err == nil {
		t.Error("Test failed. GetExchangeConfig.GetExchangeConfig Error")
	}
}

func TestGetForexProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetForexProviderConfig. LoadConfig error", err)
	}
	_, err = cfg.GetForexProviderConfig("Fixer")
	if err != nil {
		t.Error("Test failed. GetForexProviderConfig error", err)
	}
}

func TestGetPrimaryForexProvider(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetPrimaryForexProvider. LoadConfig error", err)
	}
	primary := cfg.GetPrimaryForexProvider()
	if primary == "" {
		t.Error("Test failed. GetPrimaryForexProvider error")
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

	checkExchangeConfigValues.Exchanges[0].HTTPTimeout = 0
	checkExchangeConfigValues.CheckExchangeConfigValues()
	if checkExchangeConfigValues.Exchanges[0].HTTPTimeout == 0 {
		t.Fatalf("Test failed. Expected exchange %s to have updated HTTPTimeout value", checkExchangeConfigValues.Exchanges[0].Name)
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

	checkWebserverConfigValues.Webserver.WebsocketConnectionLimit = -1
	err = checkWebserverConfigValues.CheckWebserverConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues: %s",
			err.Error(),
		)
	}

	if checkWebserverConfigValues.Webserver.WebsocketConnectionLimit != 1 {
		t.Error(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues error",
		)
	}

	checkWebserverConfigValues.Webserver.WebsocketMaxAuthFailures = -1
	checkWebserverConfigValues.CheckWebserverConfigValues()
	if checkWebserverConfigValues.Webserver.WebsocketMaxAuthFailures != 3 {
		t.Error(
			"Test failed. checkWebserverConfigValues.CheckWebserverConfigValues error",
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
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestRetrieveConfigCurrencyPairs.LoadConfig: %s", err.Error(),
		)
	}
	err = cfg.RetrieveConfigCurrencyPairs(true)
	if err != nil {
		t.Errorf(
			"Test failed. TestRetrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs: %s",
			err.Error(),
		)
	}

	err = cfg.RetrieveConfigCurrencyPairs(false)
	if err != nil {
		t.Errorf(
			"Test failed. TestRetrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs: %s",
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
	result, _ := GetFilePath("blah.json")
	if result != "blah.json" {
		t.Errorf("Test failed. TestGetFilePath: expected %s got %s", expected, result)
	}

	expected = ConfigTestFile
	result, _ = GetFilePath("")
	if result != expected {
		t.Errorf("Test failed. TestGetFilePath: expected %s got %s", expected, result)
	}
	testBypass = true
}

func TestCheckConfig(t *testing.T) {
	var c Config
	err := c.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. %s", err)
	}

	err = c.CheckConfig()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateConfig(t *testing.T) {
	var c Config
	err := c.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf("Test failed. %s", err)
	}

	newCfg := c
	err = c.UpdateConfig("", newCfg)
	if err != nil {
		t.Fatalf("Test failed. %s", err)
	}

	err = c.UpdateConfig("//non-existantpath\\", newCfg)
	if err == nil {
		t.Fatalf("Test failed. Error should of been thrown for invalid path")
	}

	newCfg.Currency.Cryptocurrencies = ""
	err = c.UpdateConfig("", newCfg)
	if err != nil {
		t.Errorf("Test failed. %s", err)
	}
	if len(c.Currency.Cryptocurrencies) == 0 {
		t.Fatalf("Test failed. Cryptocurrencies should have been repopulated")
	}
}
