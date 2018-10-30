package config

import (
	"strings"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/ntpclient"
)

const (
	// Default number of enabled exchanges. Modify this whenever an exchange is
	// added or removed
	defaultEnabledExchanges = 28
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
			if !exch.BankAccounts[0].Enabled {
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
	err = cfg.UpdateClientBankAccounts(&b)
	if err != nil {
		t.Error("Test failed. UpdateClientBankAccounts error", err)
	}

	err = cfg.UpdateClientBankAccounts(&BankAccount{})
	if err == nil {
		t.Error("Test failed. UpdateClientBankAccounts error")
	}

	var count int
	for _, bank := range cfg.BankAccounts {
		if bank.BankName == b.BankName {
			if !bank.Enabled {
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
	cfg.UpdateCommunicationsConfig(&CommunicationsConfig{SlackConfig: SlackConfig{Name: "TEST"}})
	if cfg.Communications.SlackConfig.Name != "TEST" {
		t.Error("Test failed. UpdateCommunicationsConfig LoadConfig error")
	}
}

func TestGetCryptocurrencyProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. GetCryptocurrencyProviderConfig LoadConfig error", err)
	}
	_ = cfg.GetCryptocurrencyProviderConfig()
}

func TestUpdateCryptocurrencyProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. UpdateCryptocurrencyProviderConfig LoadConfig error", err)
	}

	orig := cfg.GetCryptocurrencyProviderConfig()
	cfg.UpdateCryptocurrencyProviderConfig(CryptocurrencyProvider{Name: "SERIOUS TESTING PROCEDURE!"})
	if cfg.Currency.CryptocurrencyProvider.Name != "SERIOUS TESTING PROCEDURE!" {
		t.Error("Test failed. UpdateCurrencyProviderConfig LoadConfig error")
	}

	cfg.UpdateCryptocurrencyProviderConfig(orig)
}

func TestCheckCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Error("Test failed. CheckCommunicationsConfig LoadConfig error", err)
	}

	cfg.Communications = CommunicationsConfig{}
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Name != "Slack" ||
		cfg.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		cfg.Communications.SMTPConfig.Name != "SMTP" ||
		cfg.Communications.TelegramConfig.Name != "Telegram" {
		t.Error("Test failed. CheckCommunicationsConfig unexpected data:",
			cfg.Communications)
	}

	cfg.SMS = &SMSGlobalConfig{}
	cfg.Communications.SMSGlobalConfig.Name = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.Password != "test" {
		t.Error("Test failed. CheckCommunicationsConfig error:", err)
	}

	cfg.SMS.Contacts = append(cfg.SMS.Contacts, SMSContact{
		Name:    "Bobby",
		Number:  "4321",
		Enabled: false,
	})
	cfg.Communications.SMSGlobalConfig.Name = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.Contacts[0].Name != "Bobby" {
		t.Error("Test failed. CheckCommunicationsConfig error:", err)
	}

	cfg.SMS = &SMSGlobalConfig{}
	cfg.CheckCommunicationsConfig()
	if cfg.SMS != nil {
		t.Error("Test failed. CheckCommunicationsConfig unexpected data:",
			cfg.SMS)
	}

	cfg.Communications.SlackConfig.Name = "NOT Slack"
	cfg.CheckCommunicationsConfig()

	cfg.Communications.SlackConfig.Name = "Slack"
	cfg.Communications.SlackConfig.Enabled = true
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("Test failed. CheckCommunicationsConfig Slack is enabled when it shouldn't be.")
	}

	cfg.Communications.SlackConfig.Enabled = false
	cfg.Communications.SMSGlobalConfig.Enabled = true
	cfg.Communications.SMSGlobalConfig.Password = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("Test failed. CheckCommunicationsConfig SMSGlobal is enabled when it shouldn't be.")
	}

	cfg.Communications.SMSGlobalConfig.Enabled = false
	cfg.Communications.SMTPConfig.Enabled = true
	cfg.Communications.SMTPConfig.AccountPassword = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("Test failed. CheckCommunicationsConfig SMTPConfig is enabled when it shouldn't be.")
	}

	cfg.Communications.SMTPConfig.Enabled = false
	cfg.Communications.TelegramConfig.Enabled = true
	cfg.Communications.TelegramConfig.VerificationToken = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.TelegramConfig.Enabled {
		t.Error("Test failed. CheckCommunicationsConfig TelegramConfig is enabled when it shouldn't be.")
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

	pairsMan := currency.PairsManager{
		UseGlobalFormat: true,
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}
	pairsMan.Store(assets.AssetTypeSpot, currency.PairStore{
		Available: currency.NewPairsFromStrings([]string{"DOGE_USD,DOGE_AUD"}),
		Enabled:   currency.NewPairsFromStrings([]string{"DOGE_USD,DOGE_AUD,DOGE_BTC"}),
	})

	cfg.Exchanges = append(cfg.Exchanges, ExchangeConfig{
		Name:          "TestExchange",
		Enabled:       true,
		CurrencyPairs: &pairsMan,
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

	tec.CurrencyPairs.StorePairs(assets.AssetTypeSpot, currency.NewPairsFromStrings([]string{"DOGE_LTC,BTC_LTC"}), false)
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

	assetType := assets.AssetTypeSpot
	_, err = cfg.SupportsPair("asdf",
		currency.NewPair(currency.BTC, currency.USD), assetType)
	if err == nil {
		t.Error(
			"Test failed. TestSupportsPair. Non-existent exchange returned nil error",
		)
	}

	_, err = cfg.SupportsPair("Bitfinex",
		currency.NewPair(currency.BTC, currency.USD), assetType)
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

	assetType := assets.AssetTypeSpot
	_, err = cfg.GetAvailablePairs("asdf", assetType)
	if err == nil {
		t.Error(
			"Test failed. TestGetAvailablePairs. Non-existent exchange returned nil error")
	}

	_, err = cfg.GetAvailablePairs("Bitfinex", assetType)
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

	assetType := assets.AssetTypeSpot
	_, err = cfg.GetEnabledPairs("asdf", assetType)
	if err == nil {
		t.Error(
			"Test failed. TestGetEnabledPairs. Non-existent exchange returned nil error")
	}

	_, err = cfg.GetEnabledPairs("Bitfinex", assetType)
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
	if len(exchanges) != defaultEnabledExchanges {
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

	exchFmt, err := cfg.GetConfigCurrencyPairFormat("Yobit")
	if err != nil {
		t.Errorf("Test failed. TestGetConfigCurrencyPairFormat err: %s", err)
	}
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

	exchFmt, err := cfg.GetRequestCurrencyPairFormat("Yobit")
	if err != nil {
		t.Errorf("Test failed. TestGetRequestCurrencyPairFormat. Err: %s", err)
	}
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
	e.API.Credentials.Key = "test1234"
	err3 := UpdateExchangeConfig.UpdateExchangeConfig(e)
	if err3 != nil {
		t.Errorf(
			"Test failed. UpdateExchangeConfig.UpdateExchangeConfig: %s", err.Error(),
		)
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

	checkExchangeConfigValues.Exchanges[0].API.Credentials.Key = "Key"
	checkExchangeConfigValues.Exchanges[0].API.Credentials.Secret = "Secret"
	checkExchangeConfigValues.Exchanges[0].API.AuthenticatedSupport = true
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges[0].API.AuthenticatedSupport = true
	checkExchangeConfigValues.Exchanges[0].API.Credentials.Key = "TESTYTEST"
	checkExchangeConfigValues.Exchanges[0].API.Credentials.Secret = "TESTYTEST"
	checkExchangeConfigValues.Exchanges[0].Name = "ITBIT"
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err != nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges[0].Enabled = true
	checkExchangeConfigValues.Exchanges[0].Name = ""
	checkExchangeConfigValues.CheckExchangeConfigValues()
	if checkExchangeConfigValues.Exchanges[0].Enabled {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
		)
	}

	checkExchangeConfigValues.Exchanges = checkExchangeConfigValues.Exchanges[:0]
	checkExchangeConfigValues.Cryptocurrencies = currency.NewCurrenciesFromStringArray([]string{"TESTYTEST"})
	err = checkExchangeConfigValues.CheckExchangeConfigValues()
	if err == nil {
		t.Errorf(
			"Test failed. checkExchangeConfigValues.CheckExchangeConfigValues Error",
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
	err = c.UpdateConfig(ConfigTestFile, &newCfg)
	if err != nil {
		t.Fatalf("Test failed. %s", err)
	}

	err = c.UpdateConfig("//non-existantpath\\", &newCfg)
	if err == nil {
		t.Fatalf("Test failed. Error should of been thrown for invalid path")
	}

	newCfg.Currency.Cryptocurrencies = currency.NewCurrenciesFromStringArray([]string{""})
	err = c.UpdateConfig(ConfigTestFile, &newCfg)
	if err != nil {
		t.Errorf("Test failed. %s", err)
	}
	if c.Currency.Cryptocurrencies.Join() == "" {
		t.Fatalf("Test failed. Cryptocurrencies should have been repopulated")
	}
}

func BenchmarkUpdateConfig(b *testing.B) {
	var c Config

	err := c.LoadConfig(ConfigTestFile)
	if err != nil {
		b.Errorf("Unable to benchmark UpdateConfig(): %s", err)
	}

	newCfg := c
	for i := 0; i < b.N; i++ {
		_ = c.UpdateConfig(ConfigTestFile, &newCfg)
	}
}

func TestCheckLoggerConfig(t *testing.T) {
	c := GetConfig()
	err := c.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Fatal(err)
	}
	c.Logging = log.Logging{}
	err = c.CheckLoggerConfig()
	if err != nil {
		t.Errorf("Failed to create default logger reason: %v", err)
	}
	c.LoadConfig(ConfigTestFile)
	err = c.CheckLoggerConfig()
	if err != nil {
		t.Errorf("Failed to create logger with user settings: reason: %v", err)
	}
}

func TestDisableNTPCheck(t *testing.T) {
	c := GetConfig()
	err := c.LoadConfig(ConfigTestFile)
	if err != nil {
		t.Fatal(err)
	}

	warn, err := c.DisableNTPCheck(strings.NewReader("w\n"))
	if err != nil {
		t.Fatalf("test failed to create ntpclient failed reason: %v", err)
	}

	if warn != "Time sync has been set to warn only" {
		t.Errorf("failed expected %v got %v", "Time sync has been set to warn only", warn)
	}
	alert, _ := c.DisableNTPCheck(strings.NewReader("a\n"))
	if alert != "Time sync has been set to alert" {
		t.Errorf("failed expected %v got %v", "Time sync has been set to alert", alert)
	}

	disable, _ := c.DisableNTPCheck(strings.NewReader("d\n"))
	if disable != "Future notications for out time sync have been disabled" {
		t.Errorf("failed expected %v got %v", "Future notications for out time sync have been disabled", disable)
	}

	_, err = c.DisableNTPCheck(strings.NewReader(" "))
	if err.Error() != "EOF" {
		t.Errorf("failed expected EOF got: %v", err)
	}
}

func TestCheckNTPConfig(t *testing.T) {
	c := GetConfig()

	c.NTPClient.Level = 0
	c.NTPClient.Pool = nil
	c.NTPClient.AllowedNegativeDifference = nil
	c.NTPClient.AllowedDifference = nil

	c.CheckNTPConfig()
	_, err := ntpclient.NTPClient(c.NTPClient.Pool)
	if err != nil {
		t.Fatalf("test failed to create ntpclient failed reason: %v", err)
	}

	if c.NTPClient.Pool[0] != "pool.ntp.org:123" {
		t.Error("ntpclient with no valid pool should default to pool.ntp.org ")
	}

	if c.NTPClient.AllowedDifference == nil {
		t.Error("ntpclient with nil alloweddifference should default to sane value")
	}

	if c.NTPClient.AllowedNegativeDifference == nil {
		t.Error("ntpclient with nil allowednegativedifference should default to sane value")
	}
}
