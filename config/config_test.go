package config

import (
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/ntpclient"
)

const (
	// Default number of enabled exchanges. Modify this whenever an exchange is
	// added or removed
	defaultEnabledExchanges = 27
	testFakeExchangeName    = "Stampbit"
	testPair                = "BTC-USD"
)

func TestGetCurrencyConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetCurrencyConfig LoadConfig error", err)
	}
	_ = cfg.GetCurrencyConfig()
}

func TestGetExchangeBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetExchangeBankAccounts LoadConfig error", err)
	}
	_, err = cfg.GetExchangeBankAccounts("Bitfinex", "USD")
	if err != nil {
		t.Error("GetExchangeBankAccounts error", err)
	}
	_, err = cfg.GetExchangeBankAccounts("Not an exchange", "Not a currency")
	if err == nil {
		t.Error("GetExchangeBankAccounts, no error returned for invalid exchange")
	}
}

func TestUpdateExchangeBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateExchangeBankAccounts LoadConfig error", err)
	}

	b := []BankAccount{{Enabled: false}}
	err = cfg.UpdateExchangeBankAccounts("Bitfinex", b)
	if err != nil {
		t.Error("UpdateExchangeBankAccounts error", err)
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
		t.Error("UpdateExchangeBankAccounts error")
	}

	err = cfg.UpdateExchangeBankAccounts("Not an exchange", b)
	if err == nil {
		t.Error("UpdateExchangeBankAccounts, no error returned for invalid exchange")
	}
}

func TestGetClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetClientBankAccounts LoadConfig error", err)
	}
	_, err = cfg.GetClientBankAccounts("Kraken", "USD")
	if err != nil {
		t.Error("GetClientBankAccounts error", err)
	}
	_, err = cfg.GetClientBankAccounts("Bla", "USD")
	if err == nil {
		t.Error("GetClientBankAccounts error")
	}
	_, err = cfg.GetClientBankAccounts("Kraken", "JPY")
	if err == nil {
		t.Error("GetClientBankAccounts Expected error")
	}
}

func TestUpdateClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateClientBankAccounts LoadConfig error", err)
	}
	b := BankAccount{Enabled: false, BankName: "test", AccountNumber: "0234"}
	err = cfg.UpdateClientBankAccounts(&b)
	if err != nil {
		t.Error("UpdateClientBankAccounts error", err)
	}

	err = cfg.UpdateClientBankAccounts(&BankAccount{})
	if err == nil {
		t.Error("UpdateClientBankAccounts error")
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
		t.Error("UpdateClientBankAccounts error")
	}
}

func TestCheckClientBankAccounts(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("CheckClientBankAccounts LoadConfig error", err)
	}

	cfg.BankAccounts = nil
	cfg.CheckClientBankAccounts()
	if len(cfg.BankAccounts) == 0 {
		t.Error("CheckClientBankAccounts error:", err)
	}

	cfg.BankAccounts = nil
	cfg.BankAccounts = []BankAccount{
		{
			Enabled: true,
		},
	}

	cfg.CheckClientBankAccounts()
	if cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}

	b := BankAccount{
		Enabled:             true,
		BankName:            "Commonwealth Bank of Awesome",
		BankAddress:         "123 Fake Street",
		BankPostalCode:      "1337",
		BankPostalCity:      "Satoshiville",
		BankCountry:         "Genesis",
		AccountName:         "Satoshi Nakamoto",
		AccountNumber:       "1231006505",
		SupportedCurrencies: "USD",
	}
	cfg.BankAccounts = []BankAccount{b}
	cfg.CheckClientBankAccounts()
	if cfg.BankAccounts[0].Enabled ||
		cfg.BankAccounts[0].SupportedExchanges != "ALL" {
		t.Error("unexpected result")
	}

	// AU based bank, with no BSB number (required for domestic and international
	// transfers)
	b.SupportedCurrencies = "AUD"
	b.SWIFTCode = "BACXSI22"
	cfg.BankAccounts = []BankAccount{b}
	cfg.CheckClientBankAccounts()
	if cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}

	// Valid AU bank
	b.BSBNumber = "061337"
	cfg.BankAccounts = []BankAccount{b}
	cfg.CheckClientBankAccounts()
	if !cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}

	// Valid SWIFT/IBAN compliant bank
	b.Enabled = true
	b.IBAN = "SI56290000170073837"
	b.SWIFTCode = "BACXSI22"
	cfg.BankAccounts = []BankAccount{b}
	cfg.CheckClientBankAccounts()
	if !cfg.BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}
}

func TestGetBankAccountByID(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("CheckClientBankAccounts LoadConfig error", err)
	}

	cfg.BankAccounts = nil
	cfg.CheckClientBankAccounts()
	if len(cfg.BankAccounts) == 0 {
		t.Error("CheckClientBankAccounts error:", err)
	}

	_, err = cfg.GetBankAccountByID("test-bank-01")
	if err != nil {
		t.Error(err)
	}

	_, err = cfg.GetBankAccountByID("invalid-test-bank-01")
	if err == nil {
		t.Error("error expected for invalid account received nil")
	}
}

func TestPurgeExchangeCredentials(t *testing.T) {
	t.Parallel()
	var c Config
	c.Exchanges = []ExchangeConfig{
		{
			Name: "test",
			API: APIConfig{
				AuthenticatedSupport:          true,
				AuthenticatedWebsocketSupport: true,
				CredentialsValidator: &APICredentialsValidatorConfig{
					RequiresKey:      true,
					RequiresSecret:   true,
					RequiresClientID: true,
				},
				Credentials: APICredentialsConfig{
					Key:       "asdf123",
					Secret:    "secretp4ssw0rd",
					ClientID:  "1337",
					OTPSecret: "otp",
					PEMKey:    "aaa",
				},
			},
		},
		{
			Name: "test123",
			API: APIConfig{
				CredentialsValidator: &APICredentialsValidatorConfig{
					RequiresKey: true,
				},
				Credentials: APICredentialsConfig{
					Key:    "asdf",
					Secret: DefaultAPISecret,
				},
			},
		},
	}

	c.PurgeExchangeAPICredentials()

	exchCfg, err := c.GetExchangeConfig("test")
	if err != nil {
		t.Error(err)
	}

	if exchCfg.API.Credentials.Key != DefaultAPIKey &&
		exchCfg.API.Credentials.ClientID != DefaultAPIClientID &&
		exchCfg.API.Credentials.Secret != DefaultAPISecret &&
		exchCfg.API.Credentials.OTPSecret != "" &&
		exchCfg.API.Credentials.PEMKey != "" {
		t.Error("unexpected values")
	}

	exchCfg, err = c.GetExchangeConfig("test123")
	if err != nil {
		t.Error(err)
	}

	if exchCfg.API.Credentials.Key != "asdf" {
		t.Error("unexpected values")
	}
}

func TestGetCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetCommunicationsConfig LoadConfig error", err)
	}
	_ = cfg.GetCommunicationsConfig()
}

func TestUpdateCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateCommunicationsConfig LoadConfig error", err)
	}
	cfg.UpdateCommunicationsConfig(&CommunicationsConfig{SlackConfig: SlackConfig{Name: "TEST"}})
	if cfg.Communications.SlackConfig.Name != "TEST" {
		t.Error("UpdateCommunicationsConfig LoadConfig error")
	}
}

func TestGetCryptocurrencyProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetCryptocurrencyProviderConfig LoadConfig error", err)
	}
	_ = cfg.GetCryptocurrencyProviderConfig()
}

func TestUpdateCryptocurrencyProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("UpdateCryptocurrencyProviderConfig LoadConfig error", err)
	}

	orig := cfg.GetCryptocurrencyProviderConfig()
	cfg.UpdateCryptocurrencyProviderConfig(CryptocurrencyProvider{Name: "SERIOUS TESTING PROCEDURE!"})
	if cfg.Currency.CryptocurrencyProvider.Name != "SERIOUS TESTING PROCEDURE!" {
		t.Error("UpdateCurrencyProviderConfig LoadConfig error")
	}

	cfg.UpdateCryptocurrencyProviderConfig(orig)
}

func TestCheckCommunicationsConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("CheckCommunicationsConfig LoadConfig error", err)
	}

	cfg.Communications = CommunicationsConfig{}
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Name != "Slack" ||
		cfg.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		cfg.Communications.SMTPConfig.Name != "SMTP" ||
		cfg.Communications.TelegramConfig.Name != "Telegram" {
		t.Error("CheckCommunicationsConfig unexpected data:",
			cfg.Communications)
	}

	cfg.SMS = &SMSGlobalConfig{}
	cfg.Communications.SMSGlobalConfig.Name = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.Password != "test" {
		t.Error("CheckCommunicationsConfig error:", err)
	}

	cfg.SMS.Contacts = append(cfg.SMS.Contacts, SMSContact{
		Name:    "Bobby",
		Number:  "4321",
		Enabled: false,
	})
	cfg.Communications.SMSGlobalConfig.Name = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.Contacts[0].Name != "Bobby" {
		t.Error("CheckCommunicationsConfig error:", err)
	}

	cfg.Communications.SMSGlobalConfig.From = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.From != cfg.Name {
		t.Error("CheckCommunicationsConfig From value should have been set to the config name")
	}

	cfg.Communications.SMSGlobalConfig.From = "aaaaaaaaaaaaaaaaaaa"
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SMSGlobalConfig.From != "aaaaaaaaaaa" {
		t.Error("CheckCommunicationsConfig From value should have been trimmed to 11 characters")
	}

	cfg.SMS = &SMSGlobalConfig{}
	cfg.CheckCommunicationsConfig()
	if cfg.SMS != nil {
		t.Error("CheckCommunicationsConfig unexpected data:",
			cfg.SMS)
	}

	cfg.Communications.SlackConfig.Name = "NOT Slack"
	cfg.CheckCommunicationsConfig()

	cfg.Communications.SlackConfig.Name = "Slack"
	cfg.Communications.SlackConfig.Enabled = true
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("CheckCommunicationsConfig Slack is enabled when it shouldn't be.")
	}

	cfg.Communications.SlackConfig.Enabled = false
	cfg.Communications.SMSGlobalConfig.Enabled = true
	cfg.Communications.SMSGlobalConfig.Password = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("CheckCommunicationsConfig SMSGlobal is enabled when it shouldn't be.")
	}

	cfg.Communications.SMSGlobalConfig.Enabled = false
	cfg.Communications.SMTPConfig.Enabled = true
	cfg.Communications.SMTPConfig.AccountPassword = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.SlackConfig.Enabled {
		t.Error("CheckCommunicationsConfig SMTPConfig is enabled when it shouldn't be.")
	}

	cfg.Communications.SMTPConfig.Enabled = false
	cfg.Communications.TelegramConfig.Enabled = true
	cfg.Communications.TelegramConfig.VerificationToken = ""
	cfg.CheckCommunicationsConfig()
	if cfg.Communications.TelegramConfig.Enabled {
		t.Error("CheckCommunicationsConfig TelegramConfig is enabled when it shouldn't be.")
	}
}

func TestGetExchangeAssetTypes(t *testing.T) {
	t.Parallel()
	var c Config
	_, err := c.GetExchangeAssetTypes("void")
	if err == nil {
		t.Error("err should have been thrown on a non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				AssetTypes: asset.Items{
					asset.Spot,
					asset.Futures,
				},
			},
		},
	)

	var assets asset.Items
	assets, err = c.GetExchangeAssetTypes(testFakeExchangeName)
	if err != nil {
		t.Error(err)
	}

	if assets.JoinToString(",") != "spot,futures" {
		t.Error("unexpected results")
	}

	c.Exchanges[0].CurrencyPairs = nil
	_, err = c.GetExchangeAssetTypes(testFakeExchangeName)
	if err == nil {
		t.Error("Expected error from nil currency pair")
	}
}

func TestSupportsExchangeAssetType(t *testing.T) {
	t.Parallel()
	var c Config
	_, err := c.SupportsExchangeAssetType("void", asset.Spot)
	if err == nil {
		t.Error("Expected error for non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				AssetTypes: asset.Items{
					asset.Spot,
					asset.Futures,
				},
			},
		},
	)

	supports, err := c.SupportsExchangeAssetType(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if !supports {
		t.Error("exchange should support spot asset item")
	}

	_, err = c.SupportsExchangeAssetType(testFakeExchangeName, "asdf")
	if err == nil {
		t.Error("Expected error from invalid asset item")
	}

	c.Exchanges[0].CurrencyPairs = nil
	_, err = c.SupportsExchangeAssetType(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}
}

func TestCheckExchangeAssetsConsistency(t *testing.T) {
	t.Parallel()
	var c Config
	// Test for non-existent exchange
	c.CheckExchangeAssetsConsistency("void")

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)

	// Tests for nil currency pairs store but valid exchange name
	c.CheckExchangeAssetsConsistency(testFakeExchangeName)

	// Simulate testing a diff between stored asset types (config loading)
	// and pair store
	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
			asset.Futures,
			asset.Index,
		},
	}
	c.Exchanges[0].CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	c.Exchanges[0].CurrencyPairs.Pairs[asset.PerpetualContract] = &currency.PairStore{}
	c.CheckExchangeAssetsConsistency(testFakeExchangeName)

	supports, err := c.SupportsExchangeAssetType(testFakeExchangeName, asset.PerpetualContract)
	if err != nil {
		t.Error(err)
	}

	if supports {
		t.Error("perpetual contract should have been removed from the pair manager")
	}
}

func TestSetPairs(t *testing.T) {
	t.Parallel()

	var c Config
	pairs := currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.BTC, currency.EUR),
	}

	err := c.SetPairs("asdf", asset.Spot, true, nil)
	if err == nil {
		t.Error("Expected error from nil pairs")
	}

	err = c.SetPairs("asdf", asset.Spot, true, pairs)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)

	err = c.SetPairs(testFakeExchangeName, asset.Index, true, pairs)
	if err == nil {
		t.Error("Expected error from non initialised pair manager")
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
			asset.Futures,
		},
	}

	err = c.SetPairs(testFakeExchangeName, asset.Index, true, pairs)
	if err == nil {
		t.Error("Expected error from non supported asset type")
	}

	err = c.SetPairs(testFakeExchangeName, asset.Spot, true, pairs)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyPairConfig(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetCurrencyPairConfig("asdfg", asset.Spot)
	if err == nil {
		t.Error("Expected error with non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)

	_, err = c.GetCurrencyPairConfig(testFakeExchangeName, asset.Index)
	if err == nil {
		t.Error("Expected error with nil currency pair store")
	}

	pm := &currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
			asset.Futures,
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				RequestFormat: &currency.PairFormat{
					Uppercase: false,
					Delimiter: "_",
				},
				ConfigFormat: &currency.PairFormat{
					Uppercase: true,
					Delimiter: "~",
				},
			},
		},
	}

	c.Exchanges[0].CurrencyPairs = pm
	_, err = c.GetCurrencyPairConfig(testFakeExchangeName, asset.Index)
	if err == nil {
		t.Error("Expected error with unsupported asset")
	}

	var p *currency.PairStore
	p, err = c.GetCurrencyPairConfig(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if p.RequestFormat.Delimiter != "_" ||
		p.RequestFormat.Uppercase ||
		!p.ConfigFormat.Uppercase ||
		p.ConfigFormat.Delimiter != "~" {
		t.Error("unexpected values")
	}
}

func TestCheckPairConfigFormats(t *testing.T) {
	var c Config
	if err := c.CheckPairConfigFormats("non-existent"); err == nil {
		t.Error("non-existent exchange should throw an error")
	}

	// Test nil pair store
	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				AssetTypes: asset.Items{
					asset.Item("wrong"),
				},
			},
		},
	)

	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("nil pair store should return an error")
	}

	c.Exchanges[0].CurrencyPairs.AssetTypes = asset.Items{asset.Spot}
	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			RequestFormat: &currency.PairFormat{},
			ConfigFormat:  &currency.PairFormat{},
		},
		asset.Futures: {
			RequestFormat: &currency.PairFormat{},
			ConfigFormat:  &currency.PairFormat{},
		},
	}
	if err := c.CheckPairConfigFormats(testFakeExchangeName); err != nil {
		t.Error("nil pairs should be okay to continue")
	}

	// Test having a pair index and delimiter set at the same time throws an error
	c.Exchanges[0].CurrencyPairs.AssetTypes = asset.Items{asset.Spot}
	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			RequestFormat: &currency.PairFormat{
				Uppercase: false,
				Delimiter: "_",
			},
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "~",
				Index:     "USD",
			},
			Available: currency.Pairs{
				currency.NewPairDelimiter(testPair, "-"),
			},
			Enabled: currency.Pairs{
				currency.NewPairDelimiter("BTC~USD", "~"),
			},
		},
	}

	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("invalid pair delimiter and index should throw an error")
	}

	// Test wrong pair delimiter throws an error
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].ConfigFormat.Index = ""
	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("invalid pair delimiter should throw an error")
	}

	// Test wrong pair index in the enabled pairs throw an error
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		ConfigFormat: &currency.PairFormat{
			Index: currency.AUD.String(),
		},
	}
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		currency.NewPair(currency.BTC, currency.AUD),
	}
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{
		currency.NewPair(currency.BTC, currency.KRW),
	}

	if err := c.CheckPairConfigFormats(testFakeExchangeName); err == nil {
		t.Error("invalid pair index should throw an error")
	}
}

func TestCheckPairConsistency(t *testing.T) {
	t.Parallel()

	var c Config
	if err := c.CheckPairConsistency("asdf"); err == nil {
		t.Error("non-existent exchange should return an error")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				AssetTypes: asset.Items{
					asset.Spot,
				},
			},
		},
	)

	// Test nil pair store
	if err := c.CheckPairConsistency(testFakeExchangeName); err == nil {
		t.Error("nil pair store should return an error")
	}

	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			RequestFormat: &currency.PairFormat{
				Uppercase: false,
				Delimiter: "_",
			},
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "_",
			},
			Enabled: currency.Pairs{
				currency.NewPairDelimiter("BTC_USD", "_"),
			},
		},
	}

	// Test for nil avail pairs
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("nil available pairs should continue")
	}

	// Test that enabled pair is not found in the available pairs
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		currency.NewPairDelimiter("LTC_USD", "_"),
	}
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	// Test that an empty enabled pair is populated with an available pair
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = nil
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	// Test that an invalid enabled pair is removed from the list
	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{
		currency.NewPairDelimiter("LTC_USD", "_"),
		currency.NewPairDelimiter("BTC_USD", "_"),
	}
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}

	// Test when no update is required as the available pairs and enabled pairs
	// are consistent
	if err := c.CheckPairConsistency(testFakeExchangeName); err != nil {
		t.Error("unexpected result")
	}
}

func TestSupportsPair(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestSupportsPair. LoadConfig Error: %s", err.Error(),
		)
	}

	assetType := asset.Spot
	_, err = cfg.SupportsPair("asdf",
		currency.NewPair(currency.BTC, currency.USD), assetType)
	if err == nil {
		t.Error(
			"TestSupportsPair. Expected error from Non-existent exchange",
		)
	}

	_, err = cfg.SupportsPair("Bitfinex",
		currency.NewPair(currency.BTC, currency.USD), assetType)
	if err != nil {
		t.Errorf(
			"TestSupportsPair. Incorrect values. Err: %s", err,
		)
	}
}

func TestGetPairFormat(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetPairFormat("meow", asset.Spot)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
		},
	)
	_, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}

	c.Exchanges[0].CurrencyPairs = &currency.PairsManager{
		AssetTypes:      asset.Items{asset.Spot},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: false,
			Delimiter: "_",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "_",
		},
	}
	_, err = c.GetPairFormat(testFakeExchangeName, asset.Item("invalid"))
	if err == nil {
		t.Error("Expected error from non-existent asset item")
	}

	_, err = c.GetPairFormat(testFakeExchangeName, asset.Futures)
	if err == nil {
		t.Error("Expected error from valid but non supported asset type")
	}

	var p currency.PairFormat
	p, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if !p.Uppercase && p.Delimiter != "_" {
		t.Error("unexpected results")
	}

	// Test nil pair store
	c.Exchanges[0].CurrencyPairs.UseGlobalFormat = false
	_, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error")
	}

	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "~",
			},
		},
	}
	p, err = c.GetPairFormat(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if p.Delimiter != "~" && !p.Uppercase {
		t.Error("unexpected results")
	}
}

func TestGetAvailablePairs(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetAvailablePairs("asdf", asset.Spot)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				AssetTypes: asset.Items{
					asset.Spot,
				},
			},
		},
	)

	_, err = c.GetAvailablePairs(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}

	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			ConfigFormat: &currency.PairFormat{
				Delimiter: "-",
				Uppercase: true,
			},
		},
	}
	_, err = c.GetAvailablePairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error("Expected error from nil pairs")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Available = currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
	}
	_, err = c.GetAvailablePairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetEnabledPairs(t *testing.T) {
	t.Parallel()

	var c Config
	_, err := c.GetEnabledPairs("asdf", asset.Spot)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	c.Exchanges = append(c.Exchanges,
		ExchangeConfig{
			Name: testFakeExchangeName,
			CurrencyPairs: &currency.PairsManager{
				AssetTypes: asset.Items{
					asset.Spot,
				},
			},
		},
	)

	_, err = c.GetEnabledPairs(testFakeExchangeName, asset.Spot)
	if err == nil {
		t.Error("Expected error from nil pair manager")
	}

	c.Exchanges[0].CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			ConfigFormat: &currency.PairFormat{
				Delimiter: "-",
				Uppercase: true,
			},
		},
	}
	_, err = c.GetEnabledPairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error("nil pairs should return a nil error")
	}

	c.Exchanges[0].CurrencyPairs.Pairs[asset.Spot].Enabled = currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
	}
	_, err = c.GetEnabledPairs(testFakeExchangeName, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetEnabledExchanges(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestGetEnabledExchanges. LoadConfig Error: %s", err.Error(),
		)
	}

	exchanges := cfg.GetEnabledExchanges()
	if len(exchanges) != defaultEnabledExchanges {
		t.Error(
			"TestGetEnabledExchanges. Enabled exchanges value mismatch",
		)
	}

	if !common.StringDataCompare(exchanges, "Bitfinex") {
		t.Error(
			"TestGetEnabledExchanges. Expected exchange Bitfinex not found",
		)
	}
}

func TestGetDisabledExchanges(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestGetDisabledExchanges. LoadConfig Error: %s", err.Error(),
		)
	}

	exchanges := cfg.GetDisabledExchanges()
	if len(exchanges) != 0 {
		t.Error(
			"TestGetDisabledExchanges. Enabled exchanges value mismatch",
		)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Errorf(
			"TestGetDisabledExchanges. GetExchangeConfig Error: %s", err.Error(),
		)
	}

	exchCfg.Enabled = false
	err = cfg.UpdateExchangeConfig(exchCfg)
	if err != nil {
		t.Errorf(
			"TestGetDisabledExchanges. UpdateExchangeConfig Error: %s", err.Error(),
		)
	}

	if len(cfg.GetDisabledExchanges()) != 1 {
		t.Error(
			"TestGetDisabledExchanges. Enabled exchanges value mismatch",
		)
	}
}

func TestCountEnabledExchanges(t *testing.T) {
	GetConfigEnabledExchanges := GetConfig()
	err := GetConfigEnabledExchanges.LoadConfig(TestFile, true)
	if err != nil {
		t.Error(
			"GetConfigEnabledExchanges load config error: " + err.Error(),
		)
	}
	enabledExch := GetConfigEnabledExchanges.CountEnabledExchanges()
	if enabledExch != defaultEnabledExchanges {
		t.Errorf("Expected %v, Received %v", defaultEnabledExchanges, enabledExch)
	}
}

func TestGetCurrencyPairDisplayConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"GetCurrencyPairDisplayConfig. LoadConfig Error: %s", err.Error(),
		)
	}
	settings := cfg.GetCurrencyPairDisplayConfig()
	if settings.Delimiter != "-" || !settings.Uppercase {
		t.Errorf(
			"GetCurrencyPairDisplayConfi. Invalid values",
		)
	}
}

func TestGetAllExchangeConfigs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetAllExchangeConfigs. LoadConfig error", err)
	}
	if len(cfg.GetAllExchangeConfigs()) < 26 {
		t.Error("GetAllExchangeConfigs error")
	}
}

func TestGetExchangeConfig(t *testing.T) {
	GetExchangeConfig := GetConfig()
	err := GetExchangeConfig.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"GetExchangeConfig.LoadConfig Error: %s", err.Error(),
		)
	}
	_, err = GetExchangeConfig.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Errorf("GetExchangeConfig.GetExchangeConfig Error: %s",
			err.Error())
	}
	_, err = GetExchangeConfig.GetExchangeConfig("Testy")
	if err == nil {
		t.Error("GetExchangeConfig.GetExchangeConfig Expected error")
	}
}

func TestGetForexProviderConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetForexProviderConfig. LoadConfig error", err)
	}
	_, err = cfg.GetForexProviderConfig("Fixer")
	if err != nil {
		t.Error("GetForexProviderConfig error", err)
	}

	_, err = cfg.GetForexProviderConfig("this is not a forex provider")
	if err == nil {
		t.Error("GetForexProviderConfig no error for invalid provider")
	}
}

func TestGetForexProvidersConfig(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error(err)
	}

	if r := cfg.GetForexProvidersConfig(); len(r) != 5 {
		t.Error("unexpected length of forex providers")
	}
}

func TestGetPrimaryForexProvider(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("GetPrimaryForexProvider. LoadConfig error", err)
	}
	primary := cfg.GetPrimaryForexProvider()
	if primary == "" {
		t.Error("GetPrimaryForexProvider error")
	}

	for i := range cfg.Currency.ForexProviders {
		cfg.Currency.ForexProviders[i].PrimaryProvider = false
	}
	primary = cfg.GetPrimaryForexProvider()
	if primary != "" {
		t.Error("GetPrimaryForexProvider error, expected nil got:", primary)
	}
}

func TestUpdateExchangeConfig(t *testing.T) {
	c := GetConfig()
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		t.Error(err)
	}

	e := &ExchangeConfig{}
	err = c.UpdateExchangeConfig(e)
	if err == nil {
		t.Error("Expected error from non-existent exchange")
	}

	e, err = c.GetExchangeConfig("OKEX")
	if err != nil {
		t.Error(err)
	}

	e.API.Credentials.Key = "test1234"
	err = c.UpdateExchangeConfig(e)
	if err != nil {
		t.Error(err)
	}
}

// TestCheckExchangeConfigValues logic test
func TestCheckExchangeConfigValues(t *testing.T) {
	var cfg Config
	if err := cfg.CheckExchangeConfigValues(); err == nil {
		t.Error("nil exchanges should throw an err")
	}

	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Fatal(err)
	}

	// Test our default test config and report any errors
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Fatal(err)
	}

	cfg.Exchanges[0].Name = "GDAX"
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}
	if cfg.Exchanges[0].Name != "CoinbasePro" {
		t.Error("exchange name should have been updated from GDAX to CoinbasePRo")
	}

	// Test API settings migration
	sptr := func(s string) *string { return &s }
	bptr := func(b bool) *bool { return &b }
	int64ptr := func(i int64) *int64 { return &i }

	cfg.Exchanges[0].APIKey = sptr("awesomeKey")
	cfg.Exchanges[0].APISecret = sptr("meowSecret")
	cfg.Exchanges[0].ClientID = sptr("clientIDerino")
	cfg.Exchanges[0].APIAuthPEMKey = sptr("-----BEGIN EC PRIVATE KEY-----\nASDF\n-----END EC PRIVATE KEY-----\n")
	cfg.Exchanges[0].APIAuthPEMKeySupport = bptr(true)
	cfg.Exchanges[0].AuthenticatedAPISupport = bptr(true)
	cfg.Exchanges[0].AuthenticatedWebsocketAPISupport = bptr(true)
	cfg.Exchanges[0].WebsocketURL = sptr("wss://1337")
	cfg.Exchanges[0].APIURL = sptr(APIURLNonDefaultMessage)
	cfg.Exchanges[0].APIURLSecondary = sptr(APIURLNonDefaultMessage)
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	// Ensure that all of our previous settings are migrated
	if cfg.Exchanges[0].API.Credentials.Key != "awesomeKey" ||
		cfg.Exchanges[0].API.Credentials.Secret != "meowSecret" ||
		cfg.Exchanges[0].API.Credentials.ClientID != "clientIDerino" ||
		!strings.Contains(cfg.Exchanges[0].API.Credentials.PEMKey, "ASDF") ||
		!cfg.Exchanges[0].API.PEMKeySupport ||
		!cfg.Exchanges[0].API.AuthenticatedSupport ||
		!cfg.Exchanges[0].API.AuthenticatedWebsocketSupport ||
		cfg.Exchanges[0].API.Endpoints.WebsocketURL != "wss://1337" ||
		cfg.Exchanges[0].API.Endpoints.URL != APIURLNonDefaultMessage ||
		cfg.Exchanges[0].API.Endpoints.URLSecondary != APIURLNonDefaultMessage {
		t.Error("unexpected values")
	}

	if cfg.Exchanges[0].APIKey != nil ||
		cfg.Exchanges[0].APISecret != nil ||
		cfg.Exchanges[0].ClientID != nil ||
		cfg.Exchanges[0].APIAuthPEMKey != nil ||
		cfg.Exchanges[0].APIAuthPEMKeySupport != nil ||
		cfg.Exchanges[0].AuthenticatedAPISupport != nil ||
		cfg.Exchanges[0].AuthenticatedWebsocketAPISupport != nil ||
		cfg.Exchanges[0].WebsocketURL != nil ||
		cfg.Exchanges[0].APIURL != nil ||
		cfg.Exchanges[0].APIURLSecondary != nil {
		t.Error("unexpected values")
	}

	// Test feature and endpoint migrations migrations
	cfg.Exchanges[0].Features = nil
	cfg.Exchanges[0].SupportsAutoPairUpdates = bptr(true)
	cfg.Exchanges[0].Websocket = bptr(true)
	cfg.Exchanges[0].API.Endpoints.URL = ""
	cfg.Exchanges[0].API.Endpoints.URLSecondary = ""
	cfg.Exchanges[0].API.Endpoints.WebsocketURL = ""

	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if !cfg.Exchanges[0].Features.Enabled.AutoPairUpdates ||
		!cfg.Exchanges[0].Features.Enabled.Websocket ||
		!cfg.Exchanges[0].Features.Supports.RESTCapabilities.AutoPairUpdates {
		t.Error("unexpected values")
	}

	if cfg.Exchanges[0].API.Endpoints.URL != APIURLNonDefaultMessage ||
		cfg.Exchanges[0].API.Endpoints.URLSecondary != APIURLNonDefaultMessage ||
		cfg.Exchanges[0].API.Endpoints.WebsocketURL != WebsocketURLNonDefaultMessage {
		t.Error("unexpected values")
	}

	// Test currency pair migration
	setupPairs := func(emptyAssets bool) {
		cfg.Exchanges[0].CurrencyPairs = nil
		p := currency.Pairs{
			currency.NewPairDelimiter(testPair, "-"),
		}
		cfg.Exchanges[0].PairsLastUpdated = int64ptr(1234567)

		if !emptyAssets {
			cfg.Exchanges[0].AssetTypes = sptr("spot")
		}

		cfg.Exchanges[0].AvailablePairs = &p
		cfg.Exchanges[0].EnabledPairs = &p
		cfg.Exchanges[0].ConfigCurrencyPairFormat = &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		}
		cfg.Exchanges[0].RequestCurrencyPairFormat = &currency.PairFormat{
			Uppercase: false,
			Delimiter: "~",
		}
	}

	setupPairs(false)
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	setupPairs(true)
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if cfg.Exchanges[0].CurrencyPairs.LastUpdated != 1234567 {
		t.Error("last updated has wrong value")
	}

	pFmt := cfg.Exchanges[0].CurrencyPairs.ConfigFormat
	if pFmt.Delimiter != "-" ||
		!pFmt.Uppercase {
		t.Error("unexpected config format values")
	}

	pFmt = cfg.Exchanges[0].CurrencyPairs.RequestFormat
	if pFmt.Delimiter != "~" ||
		pFmt.Uppercase {
		t.Error("unexpected request format values")
	}

	if cfg.Exchanges[0].CurrencyPairs.AssetTypes.JoinToString(",") != "spot" ||
		!cfg.Exchanges[0].CurrencyPairs.UseGlobalFormat {
		t.Error("unexpected results")
	}

	pairs := cfg.Exchanges[0].CurrencyPairs.GetPairs(asset.Spot, true)
	if len(pairs) == 0 || pairs.Join() != testPair {
		t.Error("pairs not set properly")
	}

	pairs = cfg.Exchanges[0].CurrencyPairs.GetPairs(asset.Spot, false)
	if len(pairs) == 0 || pairs.Join() != testPair {
		t.Error("pairs not set properly")
	}

	// Ensure that all old settings are flushed
	if cfg.Exchanges[0].PairsLastUpdated != nil ||
		cfg.Exchanges[0].ConfigCurrencyPairFormat != nil ||
		cfg.Exchanges[0].RequestCurrencyPairFormat != nil ||
		cfg.Exchanges[0].AssetTypes != nil ||
		cfg.Exchanges[0].AvailablePairs != nil ||
		cfg.Exchanges[0].EnabledPairs != nil {
		t.Error("unexpected results")
	}

	// Test AutoPairUpdates
	cfg.Exchanges[0].Features.Supports.RESTCapabilities.AutoPairUpdates = false
	cfg.Exchanges[0].Features.Supports.WebsocketCapabilities.AutoPairUpdates = false
	cfg.Exchanges[0].CurrencyPairs.LastUpdated = 0
	cfg.CheckExchangeConfigValues()

	// Test exchange pair consistency error
	cfg.Exchanges[0].CurrencyPairs.UseGlobalFormat = false
	backup := cfg.Exchanges[0].CurrencyPairs.Pairs[asset.Spot]
	cfg.Exchanges[0].CurrencyPairs.Pairs[asset.Spot] = nil
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}
	if cfg.Exchanges[0].Enabled {
		t.Error("exchange should have been disabled")
	}

	// Restore to previous state
	cfg.Exchanges[0].Enabled = true
	cfg.Exchanges[0].CurrencyPairs.UseGlobalFormat = true
	cfg.Exchanges[0].CurrencyPairs.Pairs[asset.Spot] = backup

	// Test websocket and HTTP timeout values
	cfg.Exchanges[0].WebsocketResponseMaxLimit = 0
	cfg.Exchanges[0].WebsocketResponseCheckTimeout = 0
	cfg.Exchanges[0].WebsocketOrderbookBufferLimit = 0
	cfg.Exchanges[0].WebsocketTrafficTimeout = 0
	cfg.Exchanges[0].HTTPTimeout = 0
	err = cfg.CheckExchangeConfigValues()
	if err != nil {
		t.Error(err)
	}

	if cfg.Exchanges[0].WebsocketResponseMaxLimit == 0 {
		t.Errorf("expected exchange %s to have updated WebsocketResponseMaxLimit value",
			cfg.Exchanges[0].Name)
	}
	if cfg.Exchanges[0].WebsocketOrderbookBufferLimit == 0 {
		t.Errorf("expected exchange %s to have updated WebsocketOrderbookBufferLimit value",
			cfg.Exchanges[0].Name)
	}
	if cfg.Exchanges[0].WebsocketTrafficTimeout == 0 {
		t.Errorf("expected exchange %s to have updated WebsocketTrafficTimeout value",
			cfg.Exchanges[0].Name)
	}
	if cfg.Exchanges[0].HTTPTimeout == 0 {
		t.Errorf("expected exchange %s to have updated HTTPTimeout value",
			cfg.Exchanges[0].Name)
	}

	v := &APICredentialsValidatorConfig{
		RequiresKey:    true,
		RequiresSecret: true,
	}
	cfg.Exchanges[0].API.CredentialsValidator = v
	cfg.Exchanges[0].API.Credentials.Key = "Key"
	cfg.Exchanges[0].API.Credentials.Secret = "Secret"
	cfg.Exchanges[0].API.AuthenticatedSupport = true
	cfg.Exchanges[0].API.AuthenticatedWebsocketSupport = true
	cfg.CheckExchangeConfigValues()
	if cfg.Exchanges[0].API.AuthenticatedSupport ||
		cfg.Exchanges[0].API.AuthenticatedWebsocketSupport {
		t.Error("Expected authenticated endpoints to be false from invalid API keys")
	}

	v.RequiresKey = false
	v.RequiresClientID = true
	cfg.Exchanges[0].API.AuthenticatedSupport = true
	cfg.Exchanges[0].API.AuthenticatedWebsocketSupport = true
	cfg.Exchanges[0].API.Credentials.ClientID = DefaultAPIClientID
	cfg.Exchanges[0].API.Credentials.Secret = "TESTYTEST"
	cfg.CheckExchangeConfigValues()
	if cfg.Exchanges[0].API.AuthenticatedSupport ||
		cfg.Exchanges[0].API.AuthenticatedWebsocketSupport {
		t.Error("Expected AuthenticatedAPISupport to be false from invalid API keys")
	}

	v.RequiresKey = true
	cfg.Exchanges[0].API.AuthenticatedSupport = true
	cfg.Exchanges[0].API.AuthenticatedWebsocketSupport = true
	cfg.Exchanges[0].API.Credentials.Key = "meow"
	cfg.Exchanges[0].API.Credentials.Secret = "test123"
	cfg.Exchanges[0].API.Credentials.ClientID = "clientIDerino"
	cfg.CheckExchangeConfigValues()
	if !cfg.Exchanges[0].API.AuthenticatedSupport ||
		!cfg.Exchanges[0].API.AuthenticatedWebsocketSupport {
		t.Error("Expected AuthenticatedAPISupport and AuthenticatedWebsocketAPISupport to be false from invalid API keys")
	}

	// Test exchage bank accounts
	b := BankAccount{
		Enabled:             true,
		BankName:            "Commonwealth Bank of Awesome",
		BankAddress:         "123 Fake Street",
		BankPostalCode:      "1337",
		BankPostalCity:      "Satoshiville",
		BankCountry:         "Genesis",
		AccountName:         "Satoshi Nakamoto",
		AccountNumber:       "1231006505",
		SupportedCurrencies: "USD",
	}
	cfg.Exchanges[0].BankAccounts = []BankAccount{b}
	cfg.CheckExchangeConfigValues()
	if cfg.Exchanges[0].BankAccounts[0].Enabled {
		t.Error("unexpected result")
	}

	// Test empty exchange name for an enabled exchange
	cfg.Exchanges[0].Enabled = true
	cfg.Exchanges[0].Name = ""
	cfg.CheckExchangeConfigValues()
	if cfg.Exchanges[0].Enabled {
		t.Errorf(
			"Exchange with no name should be empty",
		)
	}

	// Test no enabled exchanges
	cfg.Exchanges = cfg.Exchanges[:1]
	cfg.Exchanges[0].Enabled = false
	err = cfg.CheckExchangeConfigValues()
	if err == nil {
		t.Error("Expected error from no enabled exchanges")
	}
}

func TestRetrieveConfigCurrencyPairs(t *testing.T) {
	cfg := GetConfig()
	err := cfg.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf(
			"TestRetrieveConfigCurrencyPairs.LoadConfig: %s", err.Error(),
		)
	}
	err = cfg.RetrieveConfigCurrencyPairs(true, asset.Spot)
	if err != nil {
		t.Errorf(
			"TestRetrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs: %s",
			err.Error(),
		)
	}

	err = cfg.RetrieveConfigCurrencyPairs(false, asset.Spot)
	if err != nil {
		t.Errorf(
			"TestRetrieveConfigCurrencyPairs.RetrieveConfigCurrencyPairs: %s",
			err.Error(),
		)
	}
}

func TestReadConfig(t *testing.T) {
	readConfig := GetConfig()
	err := readConfig.ReadConfig(TestFile, true)
	if err != nil {
		t.Errorf("TestReadConfig %s", err.Error())
	}

	err = readConfig.ReadConfig("bla", true)
	if err == nil {
		t.Error("TestReadConfig error cannot be nil")
	}

	err = readConfig.ReadConfig("", true)
	if err != nil {
		t.Error("TestReadConfig error")
	}
}

func TestLoadConfig(t *testing.T) {
	loadConfig := GetConfig()
	err := loadConfig.LoadConfig(TestFile, true)
	if err != nil {
		t.Error("TestLoadConfig " + err.Error())
	}

	err = loadConfig.LoadConfig("testy", true)
	if err == nil {
		t.Error("TestLoadConfig Expected error")
	}
}

func TestSaveConfig(t *testing.T) {
	saveConfig := GetConfig()
	err := saveConfig.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf("TestSaveConfig.LoadConfig: %s", err.Error())
	}
	err2 := saveConfig.SaveConfig(TestFile, true)
	if err2 != nil {
		t.Errorf("TestSaveConfig.SaveConfig, %s", err2.Error())
	}
}

func TestCheckConnectionMonitorConfig(t *testing.T) {
	t.Parallel()

	var c Config
	c.ConnectionMonitor.CheckInterval = 0
	c.ConnectionMonitor.DNSList = nil
	c.ConnectionMonitor.PublicDomainList = nil
	c.CheckConnectionMonitorConfig()

	if c.ConnectionMonitor.CheckInterval != connchecker.DefaultCheckInterval ||
		len(common.StringSliceDifference(
			c.ConnectionMonitor.DNSList, connchecker.DefaultDNSList)) != 0 ||
		len(common.StringSliceDifference(
			c.ConnectionMonitor.PublicDomainList, connchecker.DefaultDomainList)) != 0 {
		t.Error("unexpected values")
	}
}

func TestDefaultFilePath(t *testing.T) {
	// This is tricky to test because we're dealing with a config file stored
	// in a persons default directory and to properly test it, it would
	// require causing os.Stat to return !os.IsNotExist and os.IsNotExist (which
	// means moving a users config file around), a way of getting around this is
	// to pass the datadir as a param line but adds a burden to everyone who
	// uses it
	result := DefaultFilePath()
	if !strings.Contains(result, File) &&
		!strings.Contains(result, EncryptedFile) {
		t.Error("result should have contained config.json or config.dat")
	}
}

func TestGetFilePath(t *testing.T) {
	expected := "blah.json"
	result, _ := GetFilePath("blah.json")
	if result != "blah.json" {
		t.Errorf("TestGetFilePath: expected %s got %s", expected, result)
	}

	expected = TestFile
	result, _ = GetFilePath("")
	if result != expected {
		t.Errorf("TestGetFilePath: expected %s got %s", expected, result)
	}
	testBypass = true
}

func TestCheckRemoteControlConfig(t *testing.T) {
	t.Parallel()

	var c Config
	c.Webserver = &WebserverConfig{
		Enabled:                      true,
		AdminUsername:                "satoshi",
		AdminPassword:                "ultrasecurepassword",
		ListenAddress:                ":9050",
		WebsocketConnectionLimit:     5,
		WebsocketMaxAuthFailures:     10,
		WebsocketAllowInsecureOrigin: true,
	}

	c.CheckRemoteControlConfig()

	if c.RemoteControl.Username != "satoshi" ||
		c.RemoteControl.Password != "ultrasecurepassword" ||
		!c.RemoteControl.GRPC.Enabled ||
		c.RemoteControl.GRPC.ListenAddress != "localhost:9052" ||
		!c.RemoteControl.GRPC.GRPCProxyEnabled ||
		c.RemoteControl.GRPC.GRPCProxyListenAddress != "localhost:9053" ||
		!c.RemoteControl.DeprecatedRPC.Enabled ||
		c.RemoteControl.DeprecatedRPC.ListenAddress != "localhost:9050" ||
		!c.RemoteControl.WebsocketRPC.Enabled ||
		c.RemoteControl.WebsocketRPC.ListenAddress != "localhost:9051" ||
		!c.RemoteControl.WebsocketRPC.AllowInsecureOrigin ||
		c.RemoteControl.WebsocketRPC.ConnectionLimit != 5 ||
		c.RemoteControl.WebsocketRPC.MaxAuthFailures != 10 {
		t.Error("unexpected results")
	}

	// Now test to ensure the previous settings are flushed
	if c.Webserver != nil {
		t.Error("old webserver settings should be nil")
	}
}

func TestCheckConfig(t *testing.T) {
	var c Config
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = c.CheckConfig()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateConfig(t *testing.T) {
	var c Config
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		t.Errorf("%s", err)
	}

	newCfg := c
	err = c.UpdateConfig(TestFile, &newCfg, true)
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = c.UpdateConfig("//non-existantpath\\", &newCfg, true)
	if err == nil {
		t.Fatalf("Error should have been thrown for invalid path")
	}

	newCfg.Currency.Cryptocurrencies = currency.NewCurrenciesFromStringArray([]string{""})
	err = c.UpdateConfig(TestFile, &newCfg, true)
	if err != nil {
		t.Errorf("%s", err)
	}
	if c.Currency.Cryptocurrencies.Join() == "" {
		t.Fatalf("Cryptocurrencies should have been repopulated")
	}
}

func BenchmarkUpdateConfig(b *testing.B) {
	var c Config
	err := c.LoadConfig(TestFile, true)
	if err != nil {
		b.Errorf("Unable to benchmark UpdateConfig(): %s", err)
	}

	newCfg := c
	for i := 0; i < b.N; i++ {
		_ = c.UpdateConfig(TestFile, &newCfg, true)
	}
}

func TestCheckLoggerConfig(t *testing.T) {
	t.Parallel()

	var c Config
	c.Logging = log.Config{}
	err := c.CheckLoggerConfig()
	if err != nil {
		t.Errorf("Failed to create default logger. Error: %s", err)
	}

	if !*c.Logging.Enabled {
		t.Error("unexpected result")
	}

	c.Logging.LoggerFileConfig.FileName = ""
	c.Logging.LoggerFileConfig.Rotate = nil
	c.Logging.LoggerFileConfig.MaxSize = -1
	c.Logging.AdvancedSettings.ShowLogSystemName = nil

	err = c.CheckLoggerConfig()
	if err != nil {
		t.Error(err)
	}

	if c.Logging.LoggerFileConfig.FileName != "log.txt" ||
		c.Logging.LoggerFileConfig.Rotate == nil ||
		c.Logging.LoggerFileConfig.MaxSize != 100 ||
		c.Logging.AdvancedSettings.ShowLogSystemName == nil ||
		*c.Logging.AdvancedSettings.ShowLogSystemName {
		t.Error("unexpected result")
	}
}

func TestDisableNTPCheck(t *testing.T) {
	t.Parallel()

	var c Config

	warn, err := c.DisableNTPCheck(strings.NewReader("w\n"))
	if err != nil {
		t.Fatalf("to create ntpclient failed reason: %v", err)
	}

	if warn != "Time sync has been set to warn only" {
		t.Errorf("failed expected %v got %v", "Time sync has been set to warn only", warn)
	}
	alert, _ := c.DisableNTPCheck(strings.NewReader("a\n"))
	if alert != "Time sync has been set to alert" {
		t.Errorf("failed expected %v got %v", "Time sync has been set to alert", alert)
	}

	disable, _ := c.DisableNTPCheck(strings.NewReader("d\n"))
	if disable != "Future notifications for out of time sync has been disabled" {
		t.Errorf("failed expected %v got %v", "Future notifications for out of time sync has been disabled", disable)
	}

	_, err = c.DisableNTPCheck(strings.NewReader(" "))
	if err.Error() != "EOF" {
		t.Errorf("failed expected EOF got: %v", err)
	}
}

func TestCheckGCTScriptConfig(t *testing.T) {
	t.Parallel()

	var c Config
	if err := c.checkGCTScriptConfig(); err != nil {
		t.Error(err)
	}

	if c.GCTScript.ScriptTimeout != gctscript.DefaultTimeoutValue {
		t.Fatal("unexpected value return")
	}

	if c.GCTScript.MaxVirtualMachines != gctscript.DefaultMaxVirtualMachines {
		t.Fatal("unexpected value return")
	}
}

func TestCheckDatabaseConfig(t *testing.T) {
	t.Parallel()

	var c Config
	if err := c.checkDatabaseConfig(); err != nil {
		t.Error(err)
	}

	if c.Database.Driver != database.DBSQLite3 ||
		c.Database.Database != database.DefaultSQLiteDatabase ||
		c.Database.Enabled {
		t.Error("unexpected results")
	}

	c.Database.Enabled = true
	c.Database.Driver = "mssqlisthebest"
	if err := c.checkDatabaseConfig(); err == nil {
		t.Error("unexpected result")
	}

	c.Database.Driver = database.DBSQLite3
	c.Database.Enabled = true
	if err := c.checkDatabaseConfig(); err != nil {
		t.Error(err)
	}
}

func TestCheckNTPConfig(t *testing.T) {
	c := GetConfig()

	c.NTPClient.Level = 0
	c.NTPClient.Pool = nil
	c.NTPClient.AllowedNegativeDifference = nil
	c.NTPClient.AllowedDifference = nil

	c.CheckNTPConfig()
	_ = ntpclient.NTPClient(c.NTPClient.Pool)

	if c.NTPClient.Pool[0] != "pool.ntp.org:123" {
		t.Error("ntpclient with no valid pool should default to pool.ntp.org")
	}

	if c.NTPClient.AllowedDifference == nil {
		t.Error("ntpclient with nil alloweddifference should default to sane value")
	}

	if c.NTPClient.AllowedNegativeDifference == nil {
		t.Error("ntpclient with nil allowednegativedifference should default to sane value")
	}
}

func TestCheckCurrencyConfigValues(t *testing.T) {
	c := GetConfig()
	c.Currency.ForexProviders = nil
	c.Currency.CryptocurrencyProvider = CryptocurrencyProvider{}
	err := c.CheckCurrencyConfigValues()
	if err != nil {
		t.Error(err)
	}
	if c.Currency.ForexProviders == nil {
		t.Error("Failed to populate c.Currency.ForexProviders")
	}
	if c.Currency.CryptocurrencyProvider.APIkey != DefaultUnsetAPIKey {
		t.Error("Failed to set the api key to the default key")
	}
	if c.Currency.CryptocurrencyProvider.Name != "CoinMarketCap" {
		t.Error("Failed to set the  c.Currency.CryptocurrencyProvider.Name")
	}

	c.Currency.ForexProviders[0].Enabled = true
	c.Currency.ForexProviders[0].Name = "CurrencyConverter"
	c.Currency.ForexProviders[0].PrimaryProvider = true
	c.Currency.Cryptocurrencies = nil
	c.Cryptocurrencies = nil
	c.Currency.CurrencyPairFormat = nil
	c.CurrencyPairFormat = &CurrencyPairFormatConfig{
		Uppercase: true,
	}
	c.Currency.FiatDisplayCurrency = currency.Code{}
	c.FiatDisplayCurrency = &currency.BTC
	c.Currency.CryptocurrencyProvider.Enabled = true
	err = c.CheckCurrencyConfigValues()
	if err != nil {
		t.Error(err)
	}
	if c.Currency.ForexProviders[0].Enabled {
		t.Error("Failed to disable invalid forex provider")
	}
	if !c.Currency.CurrencyPairFormat.Uppercase {
		t.Error("Failed to apply c.CurrencyPairFormat format to c.Currency.CurrencyPairFormat")
	}

	c.Currency.CryptocurrencyProvider.Enabled = false
	c.Currency.CryptocurrencyProvider.APIkey = ""
	c.Currency.CryptocurrencyProvider.AccountPlan = ""
	c.FiatDisplayCurrency = &currency.BTC
	c.Currency.ForexProviders[0].Enabled = true
	c.Currency.ForexProviders[0].Name = "Name"
	c.Currency.ForexProviders[0].PrimaryProvider = true
	c.Currency.Cryptocurrencies = currency.Currencies{}
	c.Cryptocurrencies = &currency.Currencies{}
	err = c.CheckCurrencyConfigValues()
	if err != nil {
		t.Error(err)
	}
	if c.FiatDisplayCurrency != nil {
		t.Error("Failed to clear c.FiatDisplayCurrency")
	}
	if c.Currency.CryptocurrencyProvider.APIkey != DefaultUnsetAPIKey ||
		c.Currency.CryptocurrencyProvider.AccountPlan != DefaultUnsetAccountPlan {
		t.Error("Failed to set CryptocurrencyProvider.APIkey and AccountPlan")
	}
}

func TestPreengineConfigUpgrade(t *testing.T) {
	var c Config
	if err := c.LoadConfig("../testdata/preengine_config.json", false); err != nil {
		t.Fatal(err)
	}
}
