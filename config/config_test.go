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
		t.Error("Test failed. GetExchangeBankAccounts LoadConfig error", err)
	}
	_, err = cfg.GetExchangeBankAccounts("Bitfinex", "USD")
	if err != nil {
		t.Error("Test failed. GetExchangeBankAccounts error", err)
	}
	_, err = cfg.GetExchangeBankAccounts("Not an exchange", "Not a currency")
	if err == nil {
		t.Error("Test failed. GetExchangeBankAccounts, no error returned for invalid exchange")
	}
}

func TestUpdateExchangeBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. UpdateExchangeBankAccounts LoadConfig error", err)
	}

	b := []BankAccount{{Enabled: false}}
	err = cfg.UpdateExchangeBankAccounts("Bitfinex", b)
	if err != nil {
		t.Error("Test failed. UpdateExchangeBankAccounts error", err)
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
		t.Error("Test failed. UpdateExchangeBankAccounts error")
	}

	err = cfg.UpdateExchangeBankAccounts("Not an exchange", b)
	if err == nil {
		t.Error("Test failed. UpdateExchangeBankAccounts, no error returned for invalid exchange")
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
		t.Error("Test failed. UpdateClientBankAccounts error")
	}
}

func TestCheckClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. CheckClientBankAccounts LoadConfig error", err)
	}

	cfg.BankAccounts = nil
	err = cfg.CheckClientBankAccounts()
	if err != nil || len(cfg.BankAccounts) == 0 {
		t.Error("Test failed. CheckClientBankAccounts error:", err)
	}

	cfg.BankAccounts = nil
	cfg.BankAccounts = append(cfg.BankAccounts, BankAccount{
		Enabled:  true,
		BankName: "test",
	})
	err = cfg.CheckClientBankAccounts()
	if err.Error() != "banking details for test is enabled but variables not set correctly" {
		t.Error("Test failed. CheckClientBankAccounts unexpected error:", err)
	}

	cfg.BankAccounts[0].BankAddress = "test"
	err = cfg.CheckClientBankAccounts()
	if err.Error() != "banking account details for test variables not set correctly" {
		t.Error("Test failed. CheckClientBankAccounts unexpected error:", err)
	}

	cfg.BankAccounts[0].AccountName = "Thrasher"
	cfg.BankAccounts[0].AccountNumber = "1337"
	err = cfg.CheckClientBankAccounts()
	if err.Error() != "critical banking numbers not set for test in Thrasher account" {
		t.Error("Test failed. CheckClientBankAccounts unexpected error:", err)
	}

	cfg.BankAccounts[0].IBAN = "12345678"
	err = cfg.CheckClientBankAccounts()
	if err != nil {
		t.Error("Test failed. CheckClientBankAccounts error:", err)
	}
	if cfg.BankAccounts[0].SupportedExchanges == "" {
		t.Error("Test failed. CheckClientBankAccounts SupportedExchanges unexpectedly nil, data:",
			cfg.BankAccounts[0])
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

func TestCheckCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. CheckCommunicationsConfig LoadConfig error", err)
	}

	cfg.Communications = CommunicationsConfig{}
	err = cfg.CheckCommunicationsConfig()
	if err != nil {
		t.Error("Test failed. CheckCommunicationsConfig error:", err)
	}
	if cfg.Communications.SlackConfig.Name != "Slack" ||
		cfg.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		cfg.Communications.SMTPConfig.Name != "SMTP" ||
		cfg.Communications.TelegramConfig.Name != "Telegram" {
		t.Error("Test failed. CheckCommunicationsConfig unexpected data:",
			cfg.Communications)
	}

	cfg.SMS = &SMSGlobalConfig{}
	cfg.Communications.SMSGlobalConfig.Name = ""
	err = cfg.CheckCommunicationsConfig()
	if err != nil || cfg.Communications.SMSGlobalConfig.Password != "test" {
		t.Error("Test failed. CheckCommunicationsConfig error:", err)
	}

	cfg.SMS.Contacts = append(cfg.SMS.Contacts, SMSContact{
		Name:    "Bobby",
		Number:  "4321",
		Enabled: false,
	})
	cfg.Communications.SMSGlobalConfig.Name = ""
	err = cfg.CheckCommunicationsConfig()
	if err != nil || cfg.Communications.SMSGlobalConfig.Contacts[0].Name != "Bobby" {
		t.Error("Test failed. CheckCommunicationsConfig error:", err)
	}

	cfg.SMS = &SMSGlobalConfig{}
	err = cfg.CheckCommunicationsConfig()
	if err != nil {
		t.Error("Test failed. CheckCommunicationsConfig error:", err)
	}
	if cfg.SMS != nil {
		t.Error("Test failed. CheckCommunicationsConfig unexpected data:",
			cfg.SMS)
	}

	cfg.Communications.SlackConfig.Name = "NOT Slack"
	err = cfg.CheckCommunicationsConfig()
	if err.Error() != "Communications config name/s not set correctly" {
		t.Error("Test failed. CheckCommunicationsConfig unexpected error:", err)
	}

	cfg.Communications.SlackConfig.Name = "Slack"
	cfg.Communications.SlackConfig.Enabled = true
	err = cfg.CheckCommunicationsConfig()
	if err.Error() != "Slack enabled in config but variable data not set" {
		t.Error("Test failed. CheckCommunicationsConfig unexpected error:", err)
	}

	cfg.Communications.SlackConfig.Enabled = false
	cfg.Communications.SMSGlobalConfig.Enabled = true
	cfg.Communications.SMSGlobalConfig.Password = ""
	err = cfg.CheckCommunicationsConfig()
	if err.Error() != "SMSGlobal enabled in config but variable data not set" {
		t.Error("Test failed. CheckCommunicationsConfig unexpected error:", err)
	}

	cfg.Communications.SMSGlobalConfig.Enabled = false
	cfg.Communications.SMTPConfig.Enabled = true
	cfg.Communications.SMTPConfig.AccountPassword = ""
	err = cfg.CheckCommunicationsConfig()
	if err.Error() != "SMTP enabled in config but variable data not set" {
		t.Error("Test failed. CheckCommunicationsConfig unexpected error:", err)
	}

	cfg.Communications.SMTPConfig.Enabled = false
	cfg.Communications.TelegramConfig.Enabled = true
	cfg.Communications.TelegramConfig.VerificationToken = ""
	err = cfg.CheckCommunicationsConfig()
	if err.Error() != "Telegram enabled in config but variable data not set" {
		t.Error("Test failed. CheckCommunicationsConfig unexpected error:", err)
	}
}

func TestCheckPairConsistency(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. CheckPairConsistency LoadConfig error", err)
	}

	err = cfg.CheckPairConsistency("asdf")
	if err == nil {
		t.Error("Test failed. CheckPairConsistency. Non-existent exchange returned nil error")
	}

	cfg.Exchanges = append(cfg.Exchanges, ExchangeConfig{
		Name:           "TestExchange",
		Enabled:        true,
		AvailablePairs: "DOGE_USD,DOGE_AUD",
		EnabledPairs:   "DOGE_USD,DOGE_AUD,DOGE_BTC",
		ConfigCurrencyPairFormat: &CurrencyPairFormatConfig{
			Uppercase: true,
			Delimiter: "_",
		},
	})
	tec, err := cfg.GetExchangeConfig("TestExchange")
	if err != nil {
		t.Error("Test failed. CheckPairConsistency GetExchangeConfig error", err)
	}

	err = cfg.CheckPairConsistency("TestExchange")
	if err != nil {
		t.Error("Test failed. CheckPairConsistency error:", err)
	}
	// Calling again immediately to hit the if !update {return nil}
	err = cfg.CheckPairConsistency("TestExchange")
	if err != nil {
		t.Error("Test failed. CheckPairConsistency error:", err)
	}

	tec.EnabledPairs = "DOGE_LTC,BTC_LTC"
	err = cfg.UpdateExchangeConfig(tec)
	if err != nil {
		t.Error("Test failed. CheckPairConsistency Update config failed, error:", err)
	}

	err = cfg.CheckPairConsistency("TestExchange")
	if err != nil {
		t.Error("Test failed. CheckPairConsistency error:", err)
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
			"Test failed. TestGetAvailablePairs. LoadConfig Error: %s", err.Error())
	}

	_, err = cfg.GetAvailablePairs("asdf")
	if err == nil {
		t.Error(
			"Test failed. TestGetAvailablePairs. Non-existent exchange returned nil error")
	}

	_, err = cfg.GetAvailablePairs("Bitfinex")
	if err != nil {
		t.Errorf(
			"Test failed. TestGetAvailablePairs. Incorrect values. Err: %s", err)
	}
}

func TestGetEnabledPairs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestGetEnabledPairs. LoadConfig Error: %s", err.Error())
	}

	_, err = cfg.GetEnabledPairs("asdf")
	if err == nil {
		t.Error(
			"Test failed. TestGetEnabledPairs. Non-existent exchange returned nil error")
	}

	_, err = cfg.GetEnabledPairs("Bitfinex")
	if err != nil {
		t.Errorf(
			"Test failed. TestGetEnabledPairs. Incorrect values. Err: %s", err)
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

	_, err = cfg.GetForexProviderConfig("this is not a forex provider")
	if err == nil {
		t.Error("Test failed. GetForexProviderConfig no error for invalid provider")
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

	for i := range cfg.Currency.ForexProviders {
		cfg.Currency.ForexProviders[i].PrimaryProvider = false
	}
	primary = cfg.GetPrimaryForexProvider()
	if primary != "" {
		t.Error("Test failed. GetPrimaryForexProvider error, expected nil got:", primary)
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

func TestCheckRESTServerConfigValues(t *testing.T) {
	c := GetConfig()
	err := c.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. CheckRESTServerConfigValues: %s", err.Error(),
		)
	}

	err = c.CheckRESTServerConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. CheckRESTServerConfigValues: %s",
			err.Error(),
		)
	}

	c.RESTServer.ListenAddress = ":0"
	err = c.CheckRESTServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckRESTServerConfigValues error",
		)
	}

	c.RESTServer.ListenAddress = ":LOLOLOL"
	err = c.CheckRESTServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckRESTServerConfigValues error",
		)
	}

	c.RESTServer.ListenAddress = "LOLOLOL"
	err = c.CheckRESTServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckRESTServerConfigValues error",
		)
	}

	c.RESTServer.AdminUsername = ""
	err = c.CheckRESTServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckRESTServerConfigValues error",
		)
	}
}

func TestCheckWebsocketServerConfigValues(t *testing.T) {
	c := GetConfig()
	err := c.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test failed. TestCheckWebsocketServerConfigValues - LoadConfig: %s", err.Error(),
		)
	}

	err = c.CheckWebsocketServerConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. CheckWebsocketServerConfigValues: %s",
			err.Error(),
		)
	}

	c.WebsocketServer.WebsocketConnectionLimit = -1
	err = c.CheckWebsocketServerConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. CheckWebsocketServerConfigValues: %s",
			err.Error(),
		)
	}

	if c.WebsocketServer.WebsocketConnectionLimit != 1 {
		t.Error(
			"Test failed. CheckWebsocketServerConfigValues invalid values",
		)
	}

	c.WebsocketServer.WebsocketMaxAuthFailures = -1
	c.CheckWebsocketServerConfigValues()
	if c.WebsocketServer.WebsocketMaxAuthFailures != 3 {
		t.Error(
			"Test failed. CheckWebsocketServerConfigValues invalid values",
		)
	}

	c.WebsocketServer.ListenAddress = ":0"
	err = c.CheckWebsocketServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckWebsocketServerConfigValues should have returned an error on a bad listen address (bad port)",
		)
	}

	c.WebsocketServer.ListenAddress = ":LOLOLOL"
	err = c.CheckWebsocketServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckWebsocketServerConfigValues should have returned an error on a bad listen address (bad port)",
		)
	}

	c.WebsocketServer.ListenAddress = "LOLOLOL"
	err = c.CheckWebsocketServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckWebsocketServerConfigValues should have returned an error on a bad listen address",
		)
	}

	c.WebsocketServer.AdminUsername = ""
	err = c.CheckWebsocketServerConfigValues()
	if err == nil {
		t.Error(
			"Test failed. CheckWebsocketServerConfigValues should have returned an error on a nil AdminUsername",
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
	err = c.UpdateConfig(ConfigTestFile, newCfg)
	if err != nil {
		t.Fatalf("Test failed. %s", err)
	}

	err = c.UpdateConfig("//non-existantpath\\", newCfg)
	if err == nil {
		t.Fatalf("Test failed. Error should of been thrown for invalid path")
	}

	newCfg.Currency.Cryptocurrencies = ""
	err = c.UpdateConfig(ConfigTestFile, newCfg)
	if err != nil {
		t.Errorf("Test failed. %s", err)
	}
	if len(c.Currency.Cryptocurrencies) == 0 {
		t.Fatalf("Test failed. Cryptocurrencies should have been repopulated")
	}
}
