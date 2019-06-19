package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/common/convert"
	"github.com/thrasher-/gocryptotrader/connchecker"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	log "github.com/thrasher-/gocryptotrader/logger"
	logv2 "github.com/thrasher-/gocryptotrader/loggerv2"
)

// Constants declared here are filename strings and test strings
const (
	FXProviderFixer                        = "fixer"
	EncryptedConfigFile                    = "config.dat"
	ConfigFile                             = "config.json"
	ConfigTestFile                         = "../testdata/configtest.json"
	configFileEncryptionPrompt             = 0
	configFileEncryptionEnabled            = 1
	configFileEncryptionDisabled           = -1
	configPairsLastUpdatedWarningThreshold = 30 // 30 days
	configDefaultHTTPTimeout               = time.Second * 15
	defaultNTPAllowedDifference            = 50000000
	defaultNTPAllowedNegativeDifference    = 50000000
	configMaxAuthFailures                  = 3

	DefaultAPIKey      = "Key"
	DefaultAPISecret   = "Secret"
	DefaultAPIClientID = "ClientID"
)

// Constants here hold some messages
const (
	ErrExchangeNameEmpty                       = "exchange #%d name is empty"
	ErrExchangeAvailablePairsEmpty             = "exchange %s available pairs is empty"
	ErrExchangeEnabledPairsEmpty               = "exchange %s enabled pairs is empty"
	ErrExchangeBaseCurrenciesEmpty             = "exchange %s base currencies is empty"
	ErrExchangeNotFound                        = "exchange %s not found"
	ErrNoEnabledExchanges                      = "no exchanges enabled"
	ErrCryptocurrenciesEmpty                   = "cryptocurrencies variable is empty"
	ErrFailureOpeningConfig                    = "fatal error opening %s file. Error: %s"
	ErrCheckingConfigValues                    = "fatal error checking config values. Error: %s"
	ErrSavingConfigBytesMismatch               = "config file %q bytes comparison doesn't match, read %s expected %s"
	WarningWebserverCredentialValuesEmpty      = "webserver support disabled due to empty Username/Password values"
	WarningWebserverListenAddressInvalid       = "webserver support disabled due to invalid listen address"
	WarningExchangeAuthAPIDefaultOrEmptyValues = "exchange %s authenticated API support disabled due to default/empty APIKey/Secret/ClientID values"
	WarningPairsLastUpdatedThresholdExceeded   = "exchange %s last manual update of available currency pairs has exceeded %d days. Manual update required!"
)

// Constants here define unset default values displayed in the config.json
// file
const (
	APIURLNonDefaultMessage              = "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"
	WebsocketURLNonDefaultMessage        = "NON_DEFAULT_HTTP_LINK_TO_WEBSOCKET_EXCHANGE_API"
	DefaultUnsetAPIKey                   = "Key"
	DefaultUnsetAPISecret                = "Secret"
	DefaultUnsetAccountPlan              = "accountPlan"
	DefaultForexProviderExchangeRatesAPI = "ExchangeRates"
)

// Variables here are used for configuration
var (
	Cfg            Config
	IsInitialSetup bool
	testBypass     bool
	m              sync.Mutex
)

// GetCurrencyConfig returns currency configurations
func (c *Config) GetCurrencyConfig() CurrencyConfig {
	return c.Currency
}

// GetExchangeBankAccounts returns banking details associated with an exchange
// for depositing funds
func (c *Config) GetExchangeBankAccounts(exchangeName, depositingCurrency string) (BankAccount, error) {
	m.Lock()
	defer m.Unlock()

	for x := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[x].Name, exchangeName) {
			for y := range c.Exchanges[x].BankAccounts {
				if strings.Contains(c.Exchanges[x].BankAccounts[y].SupportedCurrencies,
					depositingCurrency) {
					return c.Exchanges[x].BankAccounts[y], nil
				}
			}
		}
	}
	return BankAccount{}, fmt.Errorf("exchange %s bank details not found for %s",
		exchangeName,
		depositingCurrency)
}

// UpdateExchangeBankAccounts updates the configuration for the associated
// exchange bank
func (c *Config) UpdateExchangeBankAccounts(exchangeName string, bankCfg []BankAccount) error {
	m.Lock()
	defer m.Unlock()

	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, exchangeName) {
			c.Exchanges[i].BankAccounts = bankCfg
			return nil
		}
	}
	return fmt.Errorf("exchange %s not found",
		exchangeName)
}

// GetClientBankAccounts returns banking details used for a given exchange
// and currency
func (c *Config) GetClientBankAccounts(exchangeName, targetCurrency string) (BankAccount, error) {
	m.Lock()
	defer m.Unlock()

	for x := range c.BankAccounts {
		if (strings.Contains(c.BankAccounts[x].SupportedExchanges, exchangeName) ||
			c.BankAccounts[x].SupportedExchanges == "ALL") &&
			strings.Contains(c.BankAccounts[x].SupportedCurrencies, targetCurrency) {
			return c.BankAccounts[x], nil

		}
	}
	return BankAccount{}, fmt.Errorf("client banking details not found for %s and currency %s",
		exchangeName,
		targetCurrency)
}

// UpdateClientBankAccounts updates the configuration for a bank
func (c *Config) UpdateClientBankAccounts(bankCfg *BankAccount) error {
	m.Lock()
	defer m.Unlock()

	for i := range c.BankAccounts {
		if c.BankAccounts[i].BankName == bankCfg.BankName && c.BankAccounts[i].AccountNumber == bankCfg.AccountNumber {
			c.BankAccounts[i] = *bankCfg
			return nil
		}
	}
	return fmt.Errorf("client banking details for %s not found, update not applied",
		bankCfg.BankName)
}

// CheckClientBankAccounts checks client bank details
func (c *Config) CheckClientBankAccounts() {
	m.Lock()
	defer m.Unlock()

	if len(c.BankAccounts) == 0 {
		c.BankAccounts = append(c.BankAccounts,
			BankAccount{
				BankName:            "test",
				BankAddress:         "test",
				AccountName:         "TestAccount",
				AccountNumber:       "0234",
				SWIFTCode:           "91272837",
				IBAN:                "98218738671897",
				SupportedCurrencies: "USD",
				SupportedExchanges:  "ANX,Kraken",
			},
		)
		return
	}

	for i := range c.BankAccounts {
		if c.BankAccounts[i].Enabled {
			if c.BankAccounts[i].BankName == "" || c.BankAccounts[i].BankAddress == "" {
				c.BankAccounts[i].Enabled = false
				log.Warnf("banking details for %s is enabled but variables not set correctly",
					c.BankAccounts[i].BankName)
				continue
			}

			if c.BankAccounts[i].AccountName == "" || c.BankAccounts[i].AccountNumber == "" {
				c.BankAccounts[i].Enabled = false
				log.Warnf("banking account details for %s variables not set correctly",
					c.BankAccounts[i].BankName)
				continue
			}
			if c.BankAccounts[i].IBAN == "" && c.BankAccounts[i].SWIFTCode == "" && c.BankAccounts[i].BSBNumber == "" {
				c.BankAccounts[i].Enabled = false
				log.Warnf("critical banking numbers not set for %s in %s account",
					c.BankAccounts[i].BankName,
					c.BankAccounts[i].AccountName)
				continue
			}

			if c.BankAccounts[i].SupportedExchanges == "" {
				c.BankAccounts[i].SupportedExchanges = "ALL"
			}
		}
	}
}

// PurgeExchangeAPICredentials purges the stored API credentials
func (c *Config) PurgeExchangeAPICredentials() {
	m.Lock()
	defer m.Unlock()
	for x := range c.Exchanges {
		if !c.Exchanges[x].API.AuthenticatedSupport {
			continue
		}
		c.Exchanges[x].API.AuthenticatedSupport = false

		if c.Exchanges[x].API.CredentialsValidator.RequiresKey {
			c.Exchanges[x].API.Credentials.Key = DefaultAPIKey
		}

		if c.Exchanges[x].API.CredentialsValidator.RequiresSecret {
			c.Exchanges[x].API.Credentials.Secret = DefaultAPISecret
		}

		if c.Exchanges[x].API.CredentialsValidator.RequiresClientID {
			c.Exchanges[x].API.Credentials.ClientID = DefaultAPIClientID
		}

		c.Exchanges[x].API.Credentials.PEMKey = ""
		c.Exchanges[x].API.Credentials.OTPSecret = ""
	}
}

// GetCommunicationsConfig returns the communications configuration
func (c *Config) GetCommunicationsConfig() CommunicationsConfig {
	m.Lock()
	defer m.Unlock()
	return c.Communications
}

// UpdateCommunicationsConfig sets a new updated version of a Communications
// configuration
func (c *Config) UpdateCommunicationsConfig(config *CommunicationsConfig) {
	m.Lock()
	c.Communications = *config
	m.Unlock()
}

// GetCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) GetCryptocurrencyProviderConfig() CryptocurrencyProvider {
	m.Lock()
	defer m.Unlock()
	return c.Currency.CryptocurrencyProvider
}

// UpdateCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) UpdateCryptocurrencyProviderConfig(config CryptocurrencyProvider) {
	m.Lock()
	c.Currency.CryptocurrencyProvider = config
	m.Unlock()
}

// CheckCommunicationsConfig checks to see if the variables are set correctly
// from config.json
func (c *Config) CheckCommunicationsConfig() {
	m.Lock()
	defer m.Unlock()

	// If the communications config hasn't been populated, populate
	// with example settings

	if c.Communications.SlackConfig.Name == "" {
		c.Communications.SlackConfig = SlackConfig{
			Name:              "Slack",
			TargetChannel:     "general",
			VerificationToken: "testtest",
		}
	}

	if c.Communications.SMSGlobalConfig.Name == "" {
		if c.SMS != nil {
			if c.SMS.Contacts != nil {
				c.Communications.SMSGlobalConfig = SMSGlobalConfig{
					Name:     "SMSGlobal",
					Enabled:  c.SMS.Enabled,
					Verbose:  c.SMS.Verbose,
					Username: c.SMS.Username,
					Password: c.SMS.Password,
					Contacts: c.SMS.Contacts,
				}
				// flush old SMS config
				c.SMS = nil
			} else {
				c.Communications.SMSGlobalConfig = SMSGlobalConfig{
					Name:     "SMSGlobal",
					From:     c.Name,
					Username: "main",
					Password: "test",

					Contacts: []SMSContact{
						{
							Name:    "bob",
							Number:  "1234",
							Enabled: false,
						},
					},
				}
			}
		} else {
			c.Communications.SMSGlobalConfig = SMSGlobalConfig{
				Name:     "SMSGlobal",
				Username: "main",
				Password: "test",

				Contacts: []SMSContact{
					{
						Name:    "bob",
						Number:  "1234",
						Enabled: false,
					},
				},
			}
		}

	} else {
		if c.Communications.SMSGlobalConfig.From == "" {
			c.Communications.SMSGlobalConfig.From = c.Name
		}

		if len(c.Communications.SMSGlobalConfig.From) > 11 {
			log.Warnf("SMSGlobal config supplied from name exceeds 11 characters, trimming.")
			c.Communications.SMSGlobalConfig.From = c.Communications.SMSGlobalConfig.From[:11]
		}

		if c.SMS != nil {
			// flush old SMS config
			c.SMS = nil
		}
	}

	if c.Communications.SMTPConfig.Name == "" {
		c.Communications.SMTPConfig = SMTPConfig{
			Name:            "SMTP",
			Host:            "smtp.google.com",
			Port:            "537",
			AccountName:     "some",
			AccountPassword: "password",
			RecipientList:   "lol123@gmail.com",
		}
	}

	if c.Communications.TelegramConfig.Name == "" {
		c.Communications.TelegramConfig = TelegramConfig{
			Name:              "Telegram",
			VerificationToken: "testest",
		}
	}

	if c.Communications.SlackConfig.Name != "Slack" ||
		c.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		c.Communications.SMTPConfig.Name != "SMTP" ||
		c.Communications.TelegramConfig.Name != "Telegram" {
		log.Warn("Communications config name/s not set correctly")
	}
	if c.Communications.SlackConfig.Enabled {
		if c.Communications.SlackConfig.TargetChannel == "" ||
			c.Communications.SlackConfig.VerificationToken == "" ||
			c.Communications.SlackConfig.VerificationToken == "testtest" {
			c.Communications.SlackConfig.Enabled = false
			log.Warn("Slack enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMSGlobalConfig.Enabled {
		if c.Communications.SMSGlobalConfig.Username == "" ||
			c.Communications.SMSGlobalConfig.Password == "" ||
			len(c.Communications.SMSGlobalConfig.Contacts) == 0 {
			c.Communications.SMSGlobalConfig.Enabled = false
			log.Warn("SMSGlobal enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMTPConfig.Enabled {
		if c.Communications.SMTPConfig.Host == "" ||
			c.Communications.SMTPConfig.Port == "" ||
			c.Communications.SMTPConfig.AccountName == "" ||
			c.Communications.SMTPConfig.AccountPassword == "" {
			c.Communications.SMTPConfig.Enabled = false
			log.Warn("SMTP enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.TelegramConfig.Enabled {
		if c.Communications.TelegramConfig.VerificationToken == "" {
			c.Communications.TelegramConfig.Enabled = false
			log.Warn("Telegram enabled in config but variable data not set, disabling.")
		}
	}
}

// GetExchangeAssetTypes returns the exchanges supported asset types
func (c *Config) GetExchangeAssetTypes(exchName string) (assets.AssetTypes, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return assets.AssetTypes{}, err
	}

	if exchCfg.CurrencyPairs == nil {
		return assets.AssetTypes{}, fmt.Errorf("exchange %s currency pairs is nil", exchName)
	}

	return exchCfg.CurrencyPairs.AssetTypes, nil
}

// SupportsExchangeAssetType returns whether or not the exchange supports the supplied asset type
func (c *Config) SupportsExchangeAssetType(exchName string, assetType assets.AssetType) (bool, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return false, err
	}

	if exchCfg.CurrencyPairs == nil {
		return false, fmt.Errorf("exchange %s currency pairs is nil", exchName)
	}

	if !assets.IsValid(assetType) {
		return false, fmt.Errorf("exchange %s invalid asset types", exchName)
	}

	return exchCfg.CurrencyPairs.AssetTypes.Contains(assetType), nil
}

// CheckExchangeAssetsConsistency checks the exchanges supported assets compared to the stored
// entries and removes any non supported
func (c *Config) CheckExchangeAssetsConsistency(exchName string) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return
	}

	if exchCfg.CurrencyPairs == nil {
		return
	}

	exchangeAssetTypes, err := c.GetExchangeAssetTypes(exchName)
	if err != nil {
		return
	}

	storedAssetTypes := exchCfg.CurrencyPairs.GetAssetTypes()
	for x := range storedAssetTypes {
		if !exchangeAssetTypes.Contains(storedAssetTypes[x]) {
			log.Warnf("%s has non-needed stored asset type %v. Removing..", exchName, storedAssetTypes[x])
			exchCfg.CurrencyPairs.Delete(storedAssetTypes[x])
		}
	}
}

// SetPairs sets the exchanges currency pairs
func (c *Config) SetPairs(exchName string, assetType assets.AssetType, enabled bool, pairs currency.Pairs) error {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	supports, err := c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return err
	}

	if !supports {
		return fmt.Errorf("exchange %s does not support asset type %v", exchName, assetType)
	}

	exchCfg.CurrencyPairs.StorePairs(assetType, pairs, enabled)
	return nil
}

// GetCurrencyPairConfig returns currency pair config for the desired exchange and asset type
func (c *Config) GetCurrencyPairConfig(exchName string, assetType assets.AssetType) (*currency.PairStore, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	supports, err := c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return nil, err
	}

	if !supports {
		return nil, fmt.Errorf("exchange %s does not support asset type %v", exchName, assetType)
	}

	return exchCfg.CurrencyPairs.Get(assetType), nil
}

// CheckPairConfigFormats checks to see if the pair config format is valid
func (c *Config) CheckPairConfigFormats(exchName string) error {
	assetTypes, err := c.GetExchangeAssetTypes(exchName)
	if err != nil {
		return err
	}

	for x := range assetTypes {
		assetType := assetTypes[x]
		pairFmt, err := c.GetPairFormat(exchName, assetType)
		if err != nil {
			return err
		}

		pairs, err := c.GetCurrencyPairConfig(exchName, assetType)
		if err != nil {
			return err
		}

		if pairs == nil {
			continue
		}

		if len(pairs.Available) == 0 || len(pairs.Enabled) == 0 {
			continue
		}

		checker := func(enabled bool) error {
			pairsType := "enabled"
			loadedPairs := pairs.Enabled
			if !enabled {
				pairsType = "available"
				loadedPairs = pairs.Available
			}

			for y := range loadedPairs {
				if pairFmt.Delimiter != "" {
					if !strings.Contains(loadedPairs[y].String(), pairFmt.Delimiter) {
						return fmt.Errorf("exchange %s %s %v pairs does not contain delimiter", exchName, pairsType, assetType)
					}
				}

				if pairFmt.Index != "" {
					if !strings.Contains(loadedPairs[y].String(), pairFmt.Index) {
						return fmt.Errorf("exchange %s %s %v pairs does not contain an index", exchName, pairsType, assetType)
					}
				}
			}
			return nil
		}

		err = checker(true)
		if err != nil {
			return err
		}

		err = checker(false)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckPairConsistency checks to see if the enabled pair exists in the
// available pairs list
func (c *Config) CheckPairConsistency(exchName string) error {
	assetTypes, err := c.GetExchangeAssetTypes(exchName)
	if err != nil {
		return err
	}

	err = c.CheckPairConfigFormats(exchName)
	if err != nil {
		return err
	}

	for x := range assetTypes {
		enabledPairs, err := c.GetEnabledPairs(exchName, assetTypes[x])
		if err != nil {
			return err
		}

		availPairs, err := c.GetAvailablePairs(exchName, assetTypes[x])
		if err != nil {
			return err
		}

		if len(availPairs) == 0 {
			continue
		}

		var pairs, pairsRemoved currency.Pairs
		update := false

		if len(enabledPairs) > 0 {
			for x := range enabledPairs {
				if !availPairs.Contains(enabledPairs[x], true) {
					update = true
					pairsRemoved = append(pairsRemoved, enabledPairs[x])
					continue
				}
				pairs = append(pairs, enabledPairs[x])
			}
		} else {
			update = true
		}

		if !update {
			continue
		}

		if len(pairs) == 0 || len(enabledPairs) == 0 {
			newPair := availPairs.GetRandomPair()
			err = c.SetPairs(exchName, assetTypes[x], true,
				currency.Pairs{newPair},
			)
			if err != nil {
				return fmt.Errorf("exchange %s failed to set pairs: %v", exchName, err)
			}
			log.Warnf("Exchange %s: [%v] No enabled pairs found in available pairs, randomly added %v pair.\n",
				exchName, assetTypes[x], newPair)
			continue
		} else {
			err = c.SetPairs(exchName, assetTypes[x], true, pairs)
			if err != nil {
				return fmt.Errorf("exchange %s failed to set pairs: %v", exchName, err)
			}
		}
		log.Warnf("Exchange %s: [%v] Removing enabled pair(s) %v from enabled pairs as it isn't an available pair.",
			exchName, assetTypes[x], pairsRemoved.Strings())
	}
	return nil
}

// SupportsPair returns true or not whether the exchange supports the supplied
// pair
func (c *Config) SupportsPair(exchName string, p currency.Pair, assetType assets.AssetType) (bool, error) {
	pairs, err := c.GetAvailablePairs(exchName, assetType)
	if err != nil {
		return false, err
	}
	return pairs.Contains(p, false), nil
}

// GetPairFormat returns the exchanges pair config storage format
func (c *Config) GetPairFormat(exchName string, assetType assets.AssetType) (currency.PairFormat, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return currency.PairFormat{}, err
	}

	if exchCfg.CurrencyPairs == nil {
		return currency.PairFormat{}, errors.New("exchange currency pairs type is nil")
	}

	if exchCfg.CurrencyPairs.UseGlobalFormat {
		return *exchCfg.CurrencyPairs.ConfigFormat, nil
	}

	return *exchCfg.CurrencyPairs.Get(assetType).ConfigFormat, nil
}

// GetAvailablePairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetAvailablePairs(exchName string, assetType assets.AssetType) (currency.Pairs, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairFormat, err := c.GetPairFormat(exchName, assetType)
	if err != nil {
		return nil, err
	}

	pairs := exchCfg.CurrencyPairs.Get(assetType)
	if pairs == nil {
		return nil, nil
	}

	return pairs.Available.Format(pairFormat.Delimiter, pairFormat.Index,
		pairFormat.Uppercase), nil
}

// GetEnabledPairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetEnabledPairs(exchName string, assetType assets.AssetType) ([]currency.Pair, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairFormat, err := c.GetPairFormat(exchName, assetType)
	if err != nil {
		return nil, err
	}

	pairs := exchCfg.CurrencyPairs.Get(assetType)
	if pairs == nil {
		return nil, nil
	}

	return pairs.Enabled.Format(pairFormat.Delimiter, pairFormat.Index,
		pairFormat.Uppercase), nil
}

// GetEnabledExchanges returns a list of enabled exchanges
func (c *Config) GetEnabledExchanges() []string {
	var enabledExchs []string
	for i := range c.Exchanges {
		if c.Exchanges[i].Enabled {
			enabledExchs = append(enabledExchs, c.Exchanges[i].Name)
		}
	}
	return enabledExchs
}

// GetDisabledExchanges returns a list of disabled exchanges
func (c *Config) GetDisabledExchanges() []string {
	var disabledExchs []string
	for i := range c.Exchanges {
		if !c.Exchanges[i].Enabled {
			disabledExchs = append(disabledExchs, c.Exchanges[i].Name)
		}
	}
	return disabledExchs
}

// CountEnabledExchanges returns the number of exchanges that are enabled.
func (c *Config) CountEnabledExchanges() int {
	counter := 0
	for i := range c.Exchanges {
		if c.Exchanges[i].Enabled {
			counter++
		}
	}
	return counter
}

// GetConfigCurrencyPairFormat returns the config currency pair format
// for a specific exchange
func (c *Config) GetConfigCurrencyPairFormat(exchName string) (*currency.PairFormat, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}
	return exchCfg.ConfigCurrencyPairFormat, nil
}

// GetRequestCurrencyPairFormat returns the request currency pair format
// for a specific exchange
func (c *Config) GetRequestCurrencyPairFormat(exchName string) (*currency.PairFormat, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}
	return exchCfg.RequestCurrencyPairFormat, nil
}

// GetCurrencyPairDisplayConfig retrieves the currency pair display preference
func (c *Config) GetCurrencyPairDisplayConfig() *CurrencyPairFormatConfig {
	return c.Currency.CurrencyPairFormat
}

// GetAllExchangeConfigs returns all exchange configurations
func (c *Config) GetAllExchangeConfigs() []ExchangeConfig {
	m.Lock()
	defer m.Unlock()
	return c.Exchanges
}

// GetExchangeConfig returns exchange configurations by its indivdual name
func (c *Config) GetExchangeConfig(name string) (*ExchangeConfig, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, name) {
			return &c.Exchanges[i], nil
		}
	}
	return nil, fmt.Errorf(ErrExchangeNotFound, name)
}

// GetForexProviderConfig returns a forex provider configuration by its name
func (c *Config) GetForexProviderConfig(name string) (base.Settings, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Currency.ForexProviders {
		if strings.EqualFold(c.Currency.ForexProviders[i].Name, name) {
			return c.Currency.ForexProviders[i], nil
		}
	}
	return base.Settings{}, errors.New("provider not found")
}

// GetForexProvidersConfig returns a list of available forex providers
func (c *Config) GetForexProvidersConfig() []base.Settings {
	m.Lock()
	defer m.Unlock()
	return c.Currency.ForexProviders
}

// GetPrimaryForexProvider returns the primary forex provider
func (c *Config) GetPrimaryForexProvider() string {
	m.Lock()
	defer m.Unlock()
	for i := range c.Currency.ForexProviders {
		if c.Currency.ForexProviders[i].PrimaryProvider {
			return c.Currency.ForexProviders[i].Name
		}
	}
	return ""
}

// UpdateExchangeConfig updates exchange configurations
func (c *Config) UpdateExchangeConfig(e *ExchangeConfig) error {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, e.Name) {
			c.Exchanges[i] = *e
			return nil
		}
	}
	return fmt.Errorf(ErrExchangeNotFound, e.Name)
}

// CheckExchangeConfigValues returns configuation values for all enabled
// exchanges
func (c *Config) CheckExchangeConfigValues() error {
	exchanges := 0
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, "GDAX") {
			c.Exchanges[i].Name = "CoinbasePro"
		}

		// Check to see if the old API storage format is used
		if c.Exchanges[i].APIKey != nil {
			// It is, migrate settings to new format
			c.Exchanges[i].API.AuthenticatedSupport = *c.Exchanges[i].AuthenticatedAPISupport
			c.Exchanges[i].API.Credentials.Key = *c.Exchanges[i].APIKey
			c.Exchanges[i].API.Credentials.Secret = *c.Exchanges[i].APISecret

			if c.Exchanges[i].APIAuthPEMKey != nil {
				c.Exchanges[i].API.Credentials.PEMKey = *c.Exchanges[i].APIAuthPEMKey
			}

			if c.Exchanges[i].APIAuthPEMKeySupport != nil {
				c.Exchanges[i].API.PEMKeySupport = *c.Exchanges[i].APIAuthPEMKeySupport
			}

			if c.Exchanges[i].ClientID != nil {
				c.Exchanges[i].API.Credentials.ClientID = *c.Exchanges[i].ClientID
			}

			if c.Exchanges[i].WebsocketURL != nil {
				c.Exchanges[i].API.Endpoints.WebsocketURL = *c.Exchanges[i].WebsocketURL
			}

			c.Exchanges[i].API.Endpoints.URL = *c.Exchanges[i].APIURL
			c.Exchanges[i].API.Endpoints.URLSecondary = *c.Exchanges[i].APIURLSecondary

			// Flush settings
			c.Exchanges[i].AuthenticatedAPISupport = nil
			c.Exchanges[i].APIKey = nil
			c.Exchanges[i].APIAuthPEMKey = nil
			c.Exchanges[i].APISecret = nil
			c.Exchanges[i].APIURL = nil
			c.Exchanges[i].APIURLSecondary = nil
			c.Exchanges[i].WebsocketURL = nil
			c.Exchanges[i].ClientID = nil
		}

		if c.Exchanges[i].Features == nil {
			c.Exchanges[i].Features = &FeaturesConfig{}
		}

		if c.Exchanges[i].SupportsAutoPairUpdates != nil {
			c.Exchanges[i].Features.Supports.RESTCapabilities.AutoPairUpdates = *c.Exchanges[i].SupportsAutoPairUpdates
			c.Exchanges[i].Features.Enabled.AutoPairUpdates = *c.Exchanges[i].SupportsAutoPairUpdates
			c.Exchanges[i].SupportsAutoPairUpdates = nil
		}

		if c.Exchanges[i].Websocket != nil {
			c.Exchanges[i].Features.Enabled.Websocket = *c.Exchanges[i].Websocket
			c.Exchanges[i].Websocket = nil
		}

		if c.Exchanges[i].API.Endpoints.URL != APIURLNonDefaultMessage {
			if c.Exchanges[i].API.Endpoints.URL == "" {
				// Set default if nothing set
				c.Exchanges[i].API.Endpoints.URL = APIURLNonDefaultMessage
			}
		}

		if c.Exchanges[i].API.Endpoints.URLSecondary != APIURLNonDefaultMessage {
			if c.Exchanges[i].API.Endpoints.URLSecondary == "" {
				// Set default if nothing set
				c.Exchanges[i].API.Endpoints.URLSecondary = APIURLNonDefaultMessage
			}
		}

		if c.Exchanges[i].API.Endpoints.WebsocketURL != WebsocketURLNonDefaultMessage {
			if c.Exchanges[i].API.Endpoints.WebsocketURL == "" {
				c.Exchanges[i].API.Endpoints.WebsocketURL = WebsocketURLNonDefaultMessage
			}
		}

		// Check if see if the new currency pairs format is empty and flesh it out if so
		if c.Exchanges[i].CurrencyPairs == nil {
			c.Exchanges[i].CurrencyPairs = new(currency.PairsManager)
			c.Exchanges[i].CurrencyPairs.Pairs = make(map[assets.AssetType]*currency.PairStore)

			if c.Exchanges[i].PairsLastUpdated != nil {
				c.Exchanges[i].CurrencyPairs.LastUpdated = *c.Exchanges[i].PairsLastUpdated
			}

			c.Exchanges[i].CurrencyPairs.ConfigFormat = c.Exchanges[i].ConfigCurrencyPairFormat
			c.Exchanges[i].CurrencyPairs.RequestFormat = c.Exchanges[i].RequestCurrencyPairFormat
			c.Exchanges[i].CurrencyPairs.AssetTypes = assets.New(strings.ToLower(*c.Exchanges[i].AssetTypes))
			c.Exchanges[i].CurrencyPairs.UseGlobalFormat = true
			c.Exchanges[i].CurrencyPairs.Store(assets.AssetTypeSpot,
				currency.PairStore{
					Available: *c.Exchanges[i].AvailablePairs,
					Enabled:   *c.Exchanges[i].EnabledPairs,
				},
			)

			// flush old values
			c.Exchanges[i].PairsLastUpdated = nil
			c.Exchanges[i].ConfigCurrencyPairFormat = nil
			c.Exchanges[i].RequestCurrencyPairFormat = nil
			c.Exchanges[i].AssetTypes = nil
			c.Exchanges[i].AvailablePairs = nil
			c.Exchanges[i].EnabledPairs = nil
		}

		if c.Exchanges[i].Enabled {
			if c.Exchanges[i].Name == "" {
				log.Error(ErrExchangeNameEmpty, i)
				c.Exchanges[i].Enabled = false
				continue
			}
			if c.Exchanges[i].API.AuthenticatedSupport && c.Exchanges[i].API.CredentialsValidator != nil {
				if c.Exchanges[i].API.CredentialsValidator.RequiresKey && (c.Exchanges[i].API.Credentials.Key == "" || c.Exchanges[i].API.Credentials.Key == DefaultAPIKey) {
					c.Exchanges[i].API.AuthenticatedSupport = false
				}

				if c.Exchanges[i].API.CredentialsValidator.RequiresSecret && (c.Exchanges[i].API.Credentials.Secret == "" || c.Exchanges[i].API.Credentials.Secret == DefaultAPISecret) {
					c.Exchanges[i].API.AuthenticatedSupport = false
				}

				if c.Exchanges[i].API.CredentialsValidator.RequiresClientID && (c.Exchanges[i].API.Credentials.ClientID == DefaultAPIClientID || c.Exchanges[i].API.Credentials.ClientID == "") {
					c.Exchanges[i].API.AuthenticatedSupport = false
				}

				if !c.Exchanges[i].API.AuthenticatedSupport {
					log.Warnf(WarningExchangeAuthAPIDefaultOrEmptyValues, c.Exchanges[i].Name)
				}
			}
			if !c.Exchanges[i].Features.Supports.RESTCapabilities.AutoPairUpdates && !c.Exchanges[i].Features.Supports.WebsocketCapabilities.AutoPairUpdates {
				lastUpdated := convert.UnixTimestampToTime(c.Exchanges[i].CurrencyPairs.LastUpdated)
				lastUpdated = lastUpdated.AddDate(0, 0, configPairsLastUpdatedWarningThreshold)
				if lastUpdated.Unix() <= time.Now().Unix() {
					log.Warnf(WarningPairsLastUpdatedThresholdExceeded, c.Exchanges[i].Name, configPairsLastUpdatedWarningThreshold)
				}
			}
			if c.Exchanges[i].HTTPTimeout <= 0 {
				log.Warnf("Exchange %s HTTP Timeout value not set, defaulting to %v.", c.Exchanges[i].Name, configDefaultHTTPTimeout)
				c.Exchanges[i].HTTPTimeout = configDefaultHTTPTimeout
			}

			if c.Exchanges[i].HTTPRateLimiter != nil {
				if c.Exchanges[i].HTTPRateLimiter.Authenticated.Duration < 0 {
					log.Warnf("Exchange %s HTTP Rate Limiter authenticated duration set to negative value, defaulting to 0", c.Exchanges[i].Name)
					c.Exchanges[i].HTTPRateLimiter.Authenticated.Duration = 0
				}

				if c.Exchanges[i].HTTPRateLimiter.Authenticated.Rate < 0 {
					log.Warnf("Exchange %s HTTP Rate Limiter authenticated rate set to negative value, defaulting to 0", c.Exchanges[i].Name)
					c.Exchanges[i].HTTPRateLimiter.Authenticated.Rate = 0
				}

				if c.Exchanges[i].HTTPRateLimiter.Unauthenticated.Duration < 0 {
					log.Warnf("Exchange %s HTTP Rate Limiter unauthenticated duration set to negative value, defaulting to 0", c.Exchanges[i].Name)
					c.Exchanges[i].HTTPRateLimiter.Unauthenticated.Duration = 0
				}

				if c.Exchanges[i].HTTPRateLimiter.Unauthenticated.Rate < 0 {
					log.Warnf("Exchange %s HTTP Rate Limiter unauthenticated rate set to negative value, defaulting to 0", c.Exchanges[i].Name)
					c.Exchanges[i].HTTPRateLimiter.Unauthenticated.Rate = 0
				}
			}

			err := c.CheckPairConsistency(c.Exchanges[i].Name)
			if err != nil {
				log.Errorf("Exchange %s: CheckPairConsistency error: %s", c.Exchanges[i].Name, err)
				c.Exchanges[i].Enabled = false
				continue
			}

			c.CheckExchangeAssetsConsistency(c.Exchanges[i].Name)

			if len(c.Exchanges[i].BankAccounts) > 0 {
				for x := range c.Exchanges[i].BankAccounts {
					if !c.Exchanges[i].BankAccounts[x].Enabled {
						continue
					}
					bankError := false
					if c.Exchanges[i].BankAccounts[x].BankName == "" || c.Exchanges[i].BankAccounts[x].BankAddress == "" {
						log.Warnf("banking details for %s is enabled but variables not set",
							c.Exchanges[i].Name)
						bankError = true
					}

					if c.Exchanges[i].BankAccounts[x].AccountName == "" || c.Exchanges[i].BankAccounts[x].AccountNumber == "" {
						log.Warnf("banking account details for %s variables not set",
							c.Exchanges[i].Name)
						bankError = true
					}

					if c.Exchanges[i].BankAccounts[x].SupportedCurrencies == "" {
						log.Warnf("banking account details for %s acceptable funding currencies not set",
							c.Exchanges[i].Name)
						bankError = true
					}

					if c.Exchanges[i].BankAccounts[x].BSBNumber == "" && c.Exchanges[i].BankAccounts[x].IBAN == "" &&
						c.Exchanges[i].BankAccounts[x].SWIFTCode == "" {
						log.Warnf("banking account details for %s critical banking numbers not set",
							c.Exchanges[i].Name)
						bankError = true
					}

					if bankError {
						c.Exchanges[i].BankAccounts[x].Enabled = false
					}
				}
			}
			exchanges++
		}
	}
	if exchanges == 0 {
		return errors.New(ErrNoEnabledExchanges)
	}
	return nil
}

// CheckCurrencyConfigValues checks to see if the currency config values are correct or not
func (c *Config) CheckCurrencyConfigValues() error {
	fxProviders := forexprovider.GetAvailableForexProviders()
	if len(fxProviders) == 0 {
		return errors.New("no forex providers available")
	}

	if len(fxProviders) != len(c.Currency.ForexProviders) {
		for x := range fxProviders {
			_, err := c.GetForexProviderConfig(fxProviders[x])
			if err != nil {
				log.Warnf("%s forex provider not found, adding to config..", fxProviders[x])
				c.Currency.ForexProviders = append(c.Currency.ForexProviders, base.Settings{
					Name:             fxProviders[x],
					RESTPollingDelay: 600,
					APIKey:           DefaultUnsetAPIKey,
					APIKeyLvl:        -1,
				})
			}
		}
	}

	count := 0
	for i := range c.Currency.ForexProviders {
		if c.Currency.ForexProviders[i].Enabled {
			if c.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey && c.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf("%s enabled forex provider API key not set. Please set this in your config.json file", c.Currency.ForexProviders[i].Name)
				c.Currency.ForexProviders[i].Enabled = false
				c.Currency.ForexProviders[i].PrimaryProvider = false
				continue
			}

			if c.Currency.ForexProviders[i].Name == "CurrencyConverter" {
				if c.Currency.ForexProviders[i].Enabled &&
					c.Currency.ForexProviders[i].PrimaryProvider &&
					(c.Currency.ForexProviders[i].APIKey == "" ||
						c.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey) {
					log.Warnf("CurrencyConverter forex provider no longer supports unset API key requests. Switching to ExchangeRates FX provider..")
					c.Currency.ForexProviders[i].Enabled = false
					c.Currency.ForexProviders[i].PrimaryProvider = false
					c.Currency.ForexProviders[i].APIKey = DefaultUnsetAPIKey
					c.Currency.ForexProviders[i].APIKeyLvl = -1
					continue
				}
			}

			if c.Currency.ForexProviders[i].APIKeyLvl == -1 && c.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf("%s APIKey Level not set, functions limited. Please set this in your config.json file",
					c.Currency.ForexProviders[i].Name)
			}
			count++
		}
	}

	if count == 0 {
		for x := range c.Currency.ForexProviders {
			if c.Currency.ForexProviders[x].Name == DefaultForexProviderExchangeRatesAPI {
				c.Currency.ForexProviders[x].Enabled = true
				c.Currency.ForexProviders[x].PrimaryProvider = true
				log.Warn("Using ExchangeRatesAPI for default forex provider.")
			}
		}
	}

	if c.Currency.CryptocurrencyProvider == (CryptocurrencyProvider{}) {
		c.Currency.CryptocurrencyProvider.Name = "CoinMarketCap"
		c.Currency.CryptocurrencyProvider.Enabled = false
		c.Currency.CryptocurrencyProvider.Verbose = false
		c.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
		c.Currency.CryptocurrencyProvider.APIkey = DefaultUnsetAPIKey
	}

	if c.Currency.CryptocurrencyProvider.Enabled {
		if c.Currency.CryptocurrencyProvider.APIkey == "" ||
			c.Currency.CryptocurrencyProvider.APIkey == DefaultUnsetAPIKey {
			log.Warnf("CryptocurrencyProvider enabled but api key is unset please set this in your config.json file")
		}
		if c.Currency.CryptocurrencyProvider.AccountPlan == "" ||
			c.Currency.CryptocurrencyProvider.AccountPlan == DefaultUnsetAccountPlan {
			log.Warnf("CryptocurrencyProvider enabled but account plan is unset please set this in your config.json file")
		}
	} else {
		if c.Currency.CryptocurrencyProvider.APIkey == "" {
			c.Currency.CryptocurrencyProvider.APIkey = DefaultUnsetAPIKey
		}
		if c.Currency.CryptocurrencyProvider.AccountPlan == "" {
			c.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
		}
	}

	if c.Currency.Cryptocurrencies.Join() == "" {
		if c.Cryptocurrencies != nil {
			c.Currency.Cryptocurrencies = *c.Cryptocurrencies
			c.Cryptocurrencies = nil
		} else {
			c.Currency.Cryptocurrencies = currency.GetDefaultCryptocurrencies()
		}
	}

	if c.Currency.CurrencyPairFormat == nil {
		if c.CurrencyPairFormat != nil {
			c.Currency.CurrencyPairFormat = c.CurrencyPairFormat
			c.CurrencyPairFormat = nil
		} else {
			c.Currency.CurrencyPairFormat = &CurrencyPairFormatConfig{
				Delimiter: "-",
				Uppercase: true,
			}
		}
	}

	if c.Currency.FiatDisplayCurrency.IsEmpty() {
		if c.FiatDisplayCurrency != nil {
			c.Currency.FiatDisplayCurrency = *c.FiatDisplayCurrency
			c.FiatDisplayCurrency = nil
		} else {
			c.Currency.FiatDisplayCurrency = currency.USD
		}
	}

	// Flush old setting which still exists
	if c.FiatDisplayCurrency != nil {
		c.FiatDisplayCurrency = nil
	}

	return nil
}

// RetrieveConfigCurrencyPairs splits, assigns and verifies enabled currency
// pairs either cryptoCurrencies or fiatCurrencies
func (c *Config) RetrieveConfigCurrencyPairs(enabledOnly bool) error {
	cryptoCurrencies := c.Currency.Cryptocurrencies
	fiatCurrencies := currency.GetFiatCurrencies()

	for x := range c.Exchanges {
		if !c.Exchanges[x].Enabled && enabledOnly {
			continue
		}

		baseCurrencies := c.Exchanges[x].BaseCurrencies
		for y := range baseCurrencies {
			if !fiatCurrencies.Contains(baseCurrencies[y]) {
				fiatCurrencies = append(fiatCurrencies, baseCurrencies[y])
			}
		}
	}

	for x := range c.Exchanges {
		var pairs []currency.Pair
		var err error
		if !c.Exchanges[x].Enabled && enabledOnly {
			pairs, err = c.GetEnabledPairs(c.Exchanges[x].Name, assets.AssetTypeSpot)
		} else {
			pairs, err = c.GetAvailablePairs(c.Exchanges[x].Name, assets.AssetTypeSpot)
		}

		if err != nil {
			return err
		}

		for y := range pairs {
			if !fiatCurrencies.Contains(pairs[y].Base) &&
				!cryptoCurrencies.Contains(pairs[y].Base) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].Base)
			}

			if !fiatCurrencies.Contains(pairs[y].Quote) &&
				!cryptoCurrencies.Contains(pairs[y].Quote) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].Quote)
			}
		}
	}

	currency.UpdateCurrencies(fiatCurrencies, false)
	currency.UpdateCurrencies(cryptoCurrencies, true)
	return nil
}

func (c *Config) CheckLoggerConfigV2() error {
	m.Lock()
	logv2.GlobalLogConfig = &c.Loggingv2
	m.Unlock()
	return nil
}

// CheckLoggerConfig checks to see logger values are present and valid in config
// if not creates a default instance of the logger
func (c *Config) CheckLoggerConfig() error {
	m.Lock()
	defer m.Unlock()

	// check if enabled is nil or level is a blank string
	if c.Logging.Enabled == nil || c.Logging.Level == "" {
		// Creates a new pointer to bool and sets it as true
		t := func(t bool) *bool { return &t }(true)

		log.Warn("Missing or invalid config settings using safe defaults")

		// Set logger to safe defaults
		c.Logging = log.Logging{
			Enabled:      t,
			Level:        "DEBUG|INFO|WARN|ERROR|FATAL",
			ColourOutput: false,
			File:         "debug.txt",
			Rotate:       false,
		}
		log.Logger = &c.Logging
	} else {
		log.Logger = &c.Logging
	}

	if len(c.Logging.File) > 0 {
		logPath := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "logs")
		err := common.CreateDir(logPath)
		if err != nil {
			return err
		}
		log.LogPath = logPath
	}
	return nil
}

// CheckNTPConfig checks for missing or incorrectly configured NTPClient and recreates with known safe defaults
func (c *Config) CheckNTPConfig() {
	m.Lock()
	defer m.Unlock()

	if c.NTPClient.AllowedDifference == nil || *c.NTPClient.AllowedDifference == 0 {
		c.NTPClient.AllowedDifference = new(time.Duration)
		*c.NTPClient.AllowedDifference = defaultNTPAllowedDifference
	}

	if c.NTPClient.AllowedNegativeDifference == nil || *c.NTPClient.AllowedNegativeDifference <= 0 {
		c.NTPClient.AllowedNegativeDifference = new(time.Duration)
		*c.NTPClient.AllowedNegativeDifference = defaultNTPAllowedNegativeDifference
	}

	if len(c.NTPClient.Pool) < 1 {
		log.Warn("NTPClient enabled with no servers configured, enabling default pool.")
		c.NTPClient.Pool = []string{"pool.ntp.org:123"}
	}
}

// DisableNTPCheck allows the user to change how they are prompted for timesync alerts
func (c *Config) DisableNTPCheck(input io.Reader) (string, error) {
	m.Lock()
	defer m.Unlock()

	reader := bufio.NewReader(input)
	log.Warn("Your system time is out of sync, this may cause issues with trading.")
	log.Warn("How would you like to show future notifications? (a)lert / (w)arn / (d)isable. \n")

	var answered = false
	for ok := true; ok; ok = (!answered) {
		answer, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		answer = strings.TrimRight(answer, "\r\n")
		switch answer {
		case "a":
			c.NTPClient.Level = 0
			answered = true
			return "Time sync has been set to alert", nil
		case "w":
			c.NTPClient.Level = 1
			answered = true
			return "Time sync has been set to warn only", nil
		case "d":
			c.NTPClient.Level = -1
			answered = true
			return "Future notications for out time sync have been disabled", nil
		}
	}
	return "", errors.New("something went wrong, NTPCheck should never make it this far")
}

// CheckConnectionMonitorConfig checks and if zero value assigns default values
func (c *Config) CheckConnectionMonitorConfig() {
	m.Lock()
	defer m.Unlock()

	if c.ConnectionMonitor.CheckInterval == 0 {
		c.ConnectionMonitor.CheckInterval = connchecker.DefaultCheckInterval
	}

	if len(c.ConnectionMonitor.DNSList) == 0 {
		c.ConnectionMonitor.DNSList = connchecker.DefaultDNSList
	}

	if len(c.ConnectionMonitor.PublicDomainList) == 0 {
		c.ConnectionMonitor.PublicDomainList = connchecker.DefaultDomainList
	}
}

// GetFilePath returns the desired config file or the default config file name
// based on if the application is being run under test or normal mode.
func GetFilePath(file string) (string, error) {
	if file != "" {
		return file, nil
	}

	if flag.Lookup("test.v") != nil && !testBypass {
		return ConfigTestFile, nil
	}

	exePath, err := common.GetExecutablePath()
	if err != nil {
		return "", err
	}

	oldDirs := []string{
		filepath.Join(exePath, ConfigFile),
		filepath.Join(exePath, EncryptedConfigFile),
	}

	newDir := common.GetDefaultDataDir(runtime.GOOS)
	err = common.CreateDir(newDir)
	if err != nil {
		return "", err
	}
	newDirs := []string{
		filepath.Join(newDir, ConfigFile),
		filepath.Join(newDir, EncryptedConfigFile),
	}

	// First upgrade the old dir config file if it exists to the corresponding new one
	for x := range oldDirs {
		_, err := os.Stat(oldDirs[x])
		if os.IsNotExist(err) {
			continue
		}
		if filepath.Ext(oldDirs[x]) == ".json" {
			err = os.Rename(oldDirs[x], newDirs[0])
			if err != nil {
				return "", err
			}
			log.Debugf("Renamed old config file %s to %s", oldDirs[x], newDirs[0])
		} else {
			err = os.Rename(oldDirs[x], newDirs[1])
			if err != nil {
				return "", err
			}
			log.Debugf("Renamed old config file %s to %s", oldDirs[x], newDirs[1])
		}
	}

	// Secondly check to see if the new config file extension is correct or not
	for x := range newDirs {
		_, err := os.Stat(newDirs[x])
		if os.IsNotExist(err) {
			continue
		}

		data, err := ioutil.ReadFile(newDirs[x])
		if err != nil {
			return "", err
		}

		if ConfirmECS(data) {
			if filepath.Ext(newDirs[x]) == ".dat" {
				return newDirs[x], nil
			}

			err = os.Rename(newDirs[x], newDirs[1])
			if err != nil {
				return "", err
			}
			return newDirs[1], nil
		}

		if filepath.Ext(newDirs[x]) == ".json" {
			return newDirs[x], nil
		}

		err = os.Rename(newDirs[x], newDirs[0])
		if err != nil {
			return "", err
		}

		return newDirs[0], nil
	}

	return "", errors.New("config default file path error")
}

// ReadConfig verifies and checks for encryption and verifies the unencrypted
// file contains JSON.
func (c *Config) ReadConfig(configPath string) error {
	defaultPath, err := GetFilePath(configPath)
	if err != nil {
		return err
	}

	file, err := ioutil.ReadFile(defaultPath)
	if err != nil {
		return err
	}

	if !ConfirmECS(file) {
		err = ConfirmConfigJSON(file, &c)
		if err != nil {
			return err
		}

		if c.EncryptConfig == configFileEncryptionDisabled {
			return nil
		}

		if c.EncryptConfig == configFileEncryptionPrompt {
			m.Lock()
			IsInitialSetup = true
			m.Unlock()
			if c.PromptForConfigEncryption() {
				c.EncryptConfig = configFileEncryptionEnabled
				return c.SaveConfig(defaultPath)
			}
		}
	} else {
		errCounter := 0
		for {
			if errCounter >= configMaxAuthFailures {
				return errors.New("failed to decrypt config after 3 attempts")
			}
			key, err := PromptForConfigKey(IsInitialSetup)
			if err != nil {
				log.Errorf("PromptForConfigKey err: %s", err)
				errCounter++
				continue
			}

			var f []byte
			f = append(f, file...)
			data, err := DecryptConfigFile(f, key)
			if err != nil {
				log.Errorf("DecryptConfigFile err: %s", err)
				errCounter++
				continue
			}

			err = ConfirmConfigJSON(data, &c)
			if err != nil {
				if errCounter < configMaxAuthFailures {
					log.Errorf("Invalid password.")
				}
				errCounter++
				continue
			}
			break
		}
	}
	return nil
}

// SaveConfig saves your configuration to your desired path
func (c *Config) SaveConfig(configPath string) error {
	defaultPath, err := GetFilePath(configPath)
	if err != nil {
		return err
	}

	payload, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	if c.EncryptConfig == configFileEncryptionEnabled {
		var key []byte

		if IsInitialSetup {
			key, err = PromptForConfigKey(true)
			if err != nil {
				return err
			}
			IsInitialSetup = false
		}

		payload, err = EncryptConfigFile(payload, key)
		if err != nil {
			return err
		}
	}
	return common.WriteFile(defaultPath, payload)
}

// CheckRemoteControlConfig checks to see if the old c.Webserver field is used
// and migrates the existing settings to the new RemoteControl struct
func (c *Config) CheckRemoteControlConfig() {
	m.Lock()
	defer m.Unlock()

	if c.Webserver != nil {
		port := common.ExtractPort(c.Webserver.ListenAddress)
		host := common.ExtractHost(c.Webserver.ListenAddress)

		c.RemoteControl = RemoteControlConfig{
			Username: c.Webserver.AdminUsername,
			Password: c.Webserver.AdminPassword,

			DeprecatedRPC: DepcrecatedRPCConfig{
				Enabled:       c.Webserver.Enabled,
				ListenAddress: host + ":" + strconv.Itoa(port),
			},
		}

		port++
		c.RemoteControl.WebsocketRPC = WebsocketRPCConfig{
			Enabled:             c.Webserver.Enabled,
			ListenAddress:       host + ":" + strconv.Itoa(port),
			ConnectionLimit:     c.Webserver.WebsocketConnectionLimit,
			MaxAuthFailures:     c.Webserver.WebsocketMaxAuthFailures,
			AllowInsecureOrigin: c.Webserver.WebsocketAllowInsecureOrigin,
		}

		port++
		gRPCProxyPort := port + 1
		c.RemoteControl.GRPC = GRPCConfig{
			Enabled:                c.Webserver.Enabled,
			ListenAddress:          host + ":" + strconv.Itoa(port),
			GRPCProxyEnabled:       c.Webserver.Enabled,
			GRPCProxyListenAddress: host + ":" + strconv.Itoa(gRPCProxyPort),
		}

		// Then flush the old webserver settings
		c.Webserver = nil
	}

}

// CheckConfig checks all config settings
func (c *Config) CheckConfig() error {
	err := c.CheckLoggerConfig()
	if err != nil {
		log.Errorf("Failed to configure logger. Err: %s", err)
	}

	c.CheckLoggerConfigV2()

	err = c.CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	c.CheckConnectionMonitorConfig()
	c.CheckCommunicationsConfig()
	c.CheckClientBankAccounts()
	c.CheckRemoteControlConfig()

	err = c.CheckCurrencyConfigValues()
	if err != nil {
		return err
	}

	if c.GlobalHTTPTimeout <= 0 {
		log.Warnf("Global HTTP Timeout value not set, defaulting to %v.", configDefaultHTTPTimeout)
		c.GlobalHTTPTimeout = configDefaultHTTPTimeout
	}

	if c.NTPClient.Level != 0 {
		c.CheckNTPConfig()
	}

	return nil
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig(configPath string) error {
	err := c.ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf(ErrFailureOpeningConfig, configPath, err)
	}

	return c.CheckConfig()
}

// UpdateConfig updates the config with a supplied config file
func (c *Config) UpdateConfig(configPath string, newCfg *Config) error {
	err := newCfg.CheckConfig()
	if err != nil {
		return err
	}

	c.Name = newCfg.Name
	c.EncryptConfig = newCfg.EncryptConfig
	c.Currency = newCfg.Currency
	c.GlobalHTTPTimeout = newCfg.GlobalHTTPTimeout
	c.Portfolio = newCfg.Portfolio
	c.Communications = newCfg.Communications
	c.Webserver = newCfg.Webserver
	c.Exchanges = newCfg.Exchanges

	err = c.SaveConfig(configPath)
	if err != nil {
		return err
	}

	return c.LoadConfig(configPath)
}

// GetConfig returns a pointer to a configuration object
func GetConfig() *Config {
	return &Cfg
}
