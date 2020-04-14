package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

// GetCurrencyConfig returns currency configurations
func (c *Config) GetCurrencyConfig() CurrencyConfig {
	return c.Currency
}

// GetExchangeBankAccounts returns banking details associated with an exchange
// for depositing funds
func (c *Config) GetExchangeBankAccounts(exchangeName, id, depositingCurrency string) (*banking.Account, error) {
	m.Lock()
	defer m.Unlock()

	for x := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[x].Name, exchangeName) {
			for y := range c.Exchanges[x].BankAccounts {
				if strings.EqualFold(c.Exchanges[x].BankAccounts[y].ID, id) {
					if common.StringDataCompareInsensitive(
						strings.Split(c.Exchanges[x].BankAccounts[y].SupportedCurrencies, ","),
						depositingCurrency) {
						return &c.Exchanges[x].BankAccounts[y], nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("exchange %s bank details not found for %s",
		exchangeName,
		depositingCurrency)
}

// UpdateExchangeBankAccounts updates the configuration for the associated
// exchange bank
func (c *Config) UpdateExchangeBankAccounts(exchangeName string, bankCfg []banking.Account) error {
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
func (c *Config) GetClientBankAccounts(exchangeName, targetCurrency string) (*banking.Account, error) {
	m.Lock()
	defer m.Unlock()

	for x := range c.BankAccounts {
		if (strings.Contains(c.BankAccounts[x].SupportedExchanges, exchangeName) ||
			c.BankAccounts[x].SupportedExchanges == "ALL") &&
			strings.Contains(c.BankAccounts[x].SupportedCurrencies, targetCurrency) {
			return &c.BankAccounts[x], nil
		}
	}
	return nil, fmt.Errorf("client banking details not found for %s and currency %s",
		exchangeName,
		targetCurrency)
}

// UpdateClientBankAccounts updates the configuration for a bank
func (c *Config) UpdateClientBankAccounts(bankCfg *banking.Account) error {
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
			banking.Account{
				ID:                  "test-bank-01",
				BankName:            "Test Bank",
				BankAddress:         "42 Bank Street",
				BankPostalCode:      "13337",
				BankPostalCity:      "Satoshiville",
				BankCountry:         "Japan",
				AccountName:         "Satoshi Nakamoto",
				AccountNumber:       "0234",
				SWIFTCode:           "91272837",
				IBAN:                "98218738671897",
				SupportedCurrencies: "USD",
				SupportedExchanges:  "Kraken,Bitstamp",
			},
		)
		return
	}

	for i := range c.BankAccounts {
		if c.BankAccounts[i].Enabled {
			err := c.BankAccounts[i].Validate()
			if err != nil {
				c.BankAccounts[i].Enabled = false
				log.Warn(log.ConfigMgr, err.Error())
			}
		}
	}
}

// PurgeExchangeAPICredentials purges the stored API credentials
func (c *Config) PurgeExchangeAPICredentials() {
	m.Lock()
	defer m.Unlock()
	for x := range c.Exchanges {
		if !c.Exchanges[x].API.AuthenticatedSupport && !c.Exchanges[x].API.AuthenticatedWebsocketSupport {
			continue
		}
		c.Exchanges[x].API.AuthenticatedSupport = false
		c.Exchanges[x].API.AuthenticatedWebsocketSupport = false

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
	comms := c.Communications
	m.Unlock()
	return comms
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
	provider := c.Currency.CryptocurrencyProvider
	m.Unlock()
	return provider
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
			log.Warnf(log.ConfigMgr, "SMSGlobal config supplied from name exceeds 11 characters, trimming.\n")
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
		log.Warnln(log.ConfigMgr, "Communications config name/s not set correctly")
	}
	if c.Communications.SlackConfig.Enabled {
		if c.Communications.SlackConfig.TargetChannel == "" ||
			c.Communications.SlackConfig.VerificationToken == "" ||
			c.Communications.SlackConfig.VerificationToken == "testtest" {
			c.Communications.SlackConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "Slack enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMSGlobalConfig.Enabled {
		if c.Communications.SMSGlobalConfig.Username == "" ||
			c.Communications.SMSGlobalConfig.Password == "" ||
			len(c.Communications.SMSGlobalConfig.Contacts) == 0 {
			c.Communications.SMSGlobalConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "SMSGlobal enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMTPConfig.Enabled {
		if c.Communications.SMTPConfig.Host == "" ||
			c.Communications.SMTPConfig.Port == "" ||
			c.Communications.SMTPConfig.AccountName == "" ||
			c.Communications.SMTPConfig.AccountPassword == "" {
			c.Communications.SMTPConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "SMTP enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.TelegramConfig.Enabled {
		if c.Communications.TelegramConfig.VerificationToken == "" {
			c.Communications.TelegramConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "Telegram enabled in config but variable data not set, disabling.")
		}
	}
}

// GetExchangeAssetTypes returns the exchanges supported asset types
func (c *Config) GetExchangeAssetTypes(exchName string) (asset.Items, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	if exchCfg.CurrencyPairs == nil {
		return nil, fmt.Errorf("exchange %s currency pairs is nil", exchName)
	}

	return exchCfg.CurrencyPairs.AssetTypes, nil
}

// SupportsExchangeAssetType returns whether or not the exchange supports the supplied asset type
func (c *Config) SupportsExchangeAssetType(exchName string, assetType asset.Item) (bool, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return false, err
	}

	if exchCfg.CurrencyPairs == nil {
		return false, fmt.Errorf("exchange %s currency pairs is nil", exchName)
	}

	if !asset.IsValid(assetType) {
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

	exchangeAssetTypes, err := c.GetExchangeAssetTypes(exchName)
	if err != nil {
		return
	}

	storedAssetTypes := exchCfg.CurrencyPairs.GetAssetTypes()
	for x := range storedAssetTypes {
		if !exchangeAssetTypes.Contains(storedAssetTypes[x]) {
			log.Warnf(log.ConfigMgr,
				"%s has non-needed stored asset type %v. Removing..\n",
				exchName, storedAssetTypes[x])
			exchCfg.CurrencyPairs.Delete(storedAssetTypes[x])
		}
	}
}

// SetPairs sets the exchanges currency pairs
func (c *Config) SetPairs(exchName string, assetType asset.Item, enabled bool, pairs currency.Pairs) error {
	if len(pairs) == 0 {
		return fmt.Errorf("pairs is nil")
	}

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
func (c *Config) GetCurrencyPairConfig(exchName string, assetType asset.Item) (*currency.PairStore, error) {
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

		// No err checking is required as the above checks the same
		// conditions
		pairs, _ := c.GetCurrencyPairConfig(exchName, assetType)

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
				if pairFmt.Delimiter != "" && pairFmt.Index != "" {
					return fmt.Errorf(
						"exchange %s %s %s cannot have an index and delimiter set at the same time",
						exchName, pairsType, assetType)
				}
				if pairFmt.Delimiter != "" {
					if !strings.Contains(loadedPairs[y].String(), pairFmt.Delimiter) {
						return fmt.Errorf(
							"exchange %s %s %s pairs does not contain delimiter",
							exchName, pairsType, assetType)
					}
				}
				if pairFmt.Index != "" {
					if !strings.Contains(loadedPairs[y].String(), pairFmt.Index) {
						return fmt.Errorf("exchange %s %s %s pairs does not contain an index",
							exchName, pairsType, assetType)
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

	for x := range assetTypes {
		enabledPairs, err := c.GetEnabledPairs(exchName, assetTypes[x])
		if err != nil {
			return err
		}

		availPairs, _ := c.GetAvailablePairs(exchName, assetTypes[x])
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
			c.SetPairs(exchName, assetTypes[x], true, currency.Pairs{newPair})
			log.Warnf(log.ExchangeSys, "Exchange %s: [%v] No enabled pairs found in available pairs, randomly added %v pair.\n",
				exchName, assetTypes[x], newPair)
			continue
		} else {
			c.SetPairs(exchName, assetTypes[x], true, pairs)
		}
		log.Warnf(log.ExchangeSys, "Exchange %s: [%v] Removing enabled pair(s) %v from enabled pairs as it isn't an available pair.\n",
			exchName, assetTypes[x], pairsRemoved.Strings())
	}
	return nil
}

// SupportsPair returns true or not whether the exchange supports the supplied
// pair
func (c *Config) SupportsPair(exchName string, p currency.Pair, assetType asset.Item) (bool, error) {
	pairs, err := c.GetAvailablePairs(exchName, assetType)
	if err != nil {
		return false, err
	}
	return pairs.Contains(p, false), nil
}

// GetPairFormat returns the exchanges pair config storage format
func (c *Config) GetPairFormat(exchName string, assetType asset.Item) (currency.PairFormat, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return currency.PairFormat{}, err
	}

	supports, err := c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return currency.PairFormat{}, err
	}

	if !supports {
		return currency.PairFormat{},
			fmt.Errorf("exchange %s does not support asset type %s", exchName,
				assetType)
	}

	if exchCfg.CurrencyPairs.UseGlobalFormat {
		return *exchCfg.CurrencyPairs.ConfigFormat, nil
	}

	p := exchCfg.CurrencyPairs.Get(assetType)
	if p == nil {
		return currency.PairFormat{},
			fmt.Errorf("exchange %s pair store for asset type %s is nil", exchName,
				assetType)
	}

	return *p.ConfigFormat, nil
}

// GetAvailablePairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetAvailablePairs(exchName string, assetType asset.Item) (currency.Pairs, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairFormat, err := c.GetPairFormat(exchName, assetType)
	if err != nil {
		return nil, err
	}

	pairs := exchCfg.CurrencyPairs.GetPairs(assetType, false)
	if pairs == nil {
		return nil, nil
	}

	return pairs.Format(pairFormat.Delimiter, pairFormat.Index,
		pairFormat.Uppercase), nil
}

// GetEnabledPairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetEnabledPairs(exchName string, assetType asset.Item) ([]currency.Pair, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairFormat, err := c.GetPairFormat(exchName, assetType)
	if err != nil {
		return nil, err
	}

	pairs := exchCfg.CurrencyPairs.GetPairs(assetType, true)
	if pairs == nil {
		return nil, nil
	}

	return pairs.Format(pairFormat.Delimiter, pairFormat.Index,
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

// GetCurrencyPairDisplayConfig retrieves the currency pair display preference
func (c *Config) GetCurrencyPairDisplayConfig() *CurrencyPairFormatConfig {
	return c.Currency.CurrencyPairFormat
}

// GetAllExchangeConfigs returns all exchange configurations
func (c *Config) GetAllExchangeConfigs() []ExchangeConfig {
	m.Lock()
	configs := c.Exchanges
	m.Unlock()
	return configs
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

// GetForexProvider returns a forex provider configuration by its name
func (c *Config) GetForexProvider(name string) (currency.FXSettings, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Currency.ForexProviders {
		if strings.EqualFold(c.Currency.ForexProviders[i].Name, name) {
			return c.Currency.ForexProviders[i], nil
		}
	}
	return currency.FXSettings{}, errors.New("provider not found")
}

// GetForexProviders returns a list of available forex providers
func (c *Config) GetForexProviders() []currency.FXSettings {
	m.Lock()
	fxProviders := c.Currency.ForexProviders
	m.Unlock()
	return fxProviders
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
	if len(c.Exchanges) == 0 {
		return errors.New("no exchange configs found")
	}

	exchanges := 0
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, "GDAX") {
			c.Exchanges[i].Name = "CoinbasePro"
		}

		// Check to see if the old API storage format is used
		if c.Exchanges[i].APIKey != nil {
			// It is, migrate settings to new format
			c.Exchanges[i].API.AuthenticatedSupport = *c.Exchanges[i].AuthenticatedAPISupport
			if c.Exchanges[i].AuthenticatedWebsocketAPISupport != nil {
				c.Exchanges[i].API.AuthenticatedWebsocketSupport = *c.Exchanges[i].AuthenticatedWebsocketAPISupport
			}
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
			c.Exchanges[i].AuthenticatedWebsocketAPISupport = nil
			c.Exchanges[i].APIKey = nil
			c.Exchanges[i].APISecret = nil
			c.Exchanges[i].ClientID = nil
			c.Exchanges[i].APIAuthPEMKeySupport = nil
			c.Exchanges[i].APIAuthPEMKey = nil
			c.Exchanges[i].APIURL = nil
			c.Exchanges[i].APIURLSecondary = nil
			c.Exchanges[i].WebsocketURL = nil
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
			c.Exchanges[i].CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)

			if c.Exchanges[i].PairsLastUpdated != nil {
				c.Exchanges[i].CurrencyPairs.LastUpdated = *c.Exchanges[i].PairsLastUpdated
			}

			c.Exchanges[i].CurrencyPairs.ConfigFormat = c.Exchanges[i].ConfigCurrencyPairFormat
			c.Exchanges[i].CurrencyPairs.RequestFormat = c.Exchanges[i].RequestCurrencyPairFormat

			if c.Exchanges[i].AssetTypes == nil {
				c.Exchanges[i].CurrencyPairs.AssetTypes = asset.Items{
					asset.Spot,
				}
			} else {
				c.Exchanges[i].CurrencyPairs.AssetTypes = asset.New(
					strings.ToLower(*c.Exchanges[i].AssetTypes),
				)
			}

			var availPairs, enabledPairs currency.Pairs
			if c.Exchanges[i].AvailablePairs != nil {
				availPairs = *c.Exchanges[i].AvailablePairs
			}

			if c.Exchanges[i].EnabledPairs != nil {
				enabledPairs = *c.Exchanges[i].EnabledPairs
			}

			c.Exchanges[i].CurrencyPairs.UseGlobalFormat = true
			c.Exchanges[i].CurrencyPairs.Store(asset.Spot,
				currency.PairStore{
					Available: availPairs,
					Enabled:   enabledPairs,
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
				log.Errorf(log.ConfigMgr, ErrExchangeNameEmpty, i)
				c.Exchanges[i].Enabled = false
				continue
			}
			if (c.Exchanges[i].API.AuthenticatedSupport || c.Exchanges[i].API.AuthenticatedWebsocketSupport) && c.Exchanges[i].API.CredentialsValidator != nil {
				var failed bool
				if c.Exchanges[i].API.CredentialsValidator.RequiresKey && (c.Exchanges[i].API.Credentials.Key == "" || c.Exchanges[i].API.Credentials.Key == DefaultAPIKey) {
					failed = true
				}

				if c.Exchanges[i].API.CredentialsValidator.RequiresSecret && (c.Exchanges[i].API.Credentials.Secret == "" || c.Exchanges[i].API.Credentials.Secret == DefaultAPISecret) {
					failed = true
				}

				if c.Exchanges[i].API.CredentialsValidator.RequiresClientID && (c.Exchanges[i].API.Credentials.ClientID == DefaultAPIClientID || c.Exchanges[i].API.Credentials.ClientID == "") {
					failed = true
				}

				if failed {
					c.Exchanges[i].API.AuthenticatedSupport = false
					c.Exchanges[i].API.AuthenticatedWebsocketSupport = false
					log.Warnf(log.ExchangeSys, WarningExchangeAuthAPIDefaultOrEmptyValues, c.Exchanges[i].Name)
				}
			}
			if !c.Exchanges[i].Features.Supports.RESTCapabilities.AutoPairUpdates && !c.Exchanges[i].Features.Supports.WebsocketCapabilities.AutoPairUpdates {
				lastUpdated := convert.UnixTimestampToTime(c.Exchanges[i].CurrencyPairs.LastUpdated)
				lastUpdated = lastUpdated.AddDate(0, 0, pairsLastUpdatedWarningThreshold)
				if lastUpdated.Unix() <= time.Now().Unix() {
					log.Warnf(log.ExchangeSys, WarningPairsLastUpdatedThresholdExceeded, c.Exchanges[i].Name, pairsLastUpdatedWarningThreshold)
				}
			}
			if c.Exchanges[i].HTTPTimeout <= 0 {
				log.Warnf(log.ExchangeSys, "Exchange %s HTTP Timeout value not set, defaulting to %v.\n", c.Exchanges[i].Name, defaultHTTPTimeout)
				c.Exchanges[i].HTTPTimeout = defaultHTTPTimeout
			}

			if c.Exchanges[i].WebsocketResponseCheckTimeout <= 0 {
				log.Warnf(log.ExchangeSys, "Exchange %s Websocket response check timeout value not set, defaulting to %v.",
					c.Exchanges[i].Name, defaultWebsocketResponseCheckTimeout)
				c.Exchanges[i].WebsocketResponseCheckTimeout = defaultWebsocketResponseCheckTimeout
			}

			if c.Exchanges[i].WebsocketResponseMaxLimit <= 0 {
				log.Warnf(log.ExchangeSys, "Exchange %s Websocket response max limit value not set, defaulting to %v.",
					c.Exchanges[i].Name, defaultWebsocketResponseMaxLimit)
				c.Exchanges[i].WebsocketResponseMaxLimit = defaultWebsocketResponseMaxLimit
			}
			if c.Exchanges[i].WebsocketTrafficTimeout <= 0 {
				log.Warnf(log.ExchangeSys, "Exchange %s Websocket response traffic timeout value not set, defaulting to %v.",
					c.Exchanges[i].Name, defaultWebsocketTrafficTimeout)
				c.Exchanges[i].WebsocketTrafficTimeout = defaultWebsocketTrafficTimeout
			}
			if c.Exchanges[i].WebsocketOrderbookBufferLimit <= 0 {
				log.Warnf(log.ExchangeSys, "Exchange %s Websocket orderbook buffer limit value not set, defaulting to %v.",
					c.Exchanges[i].Name, defaultWebsocketOrderbookBufferLimit)
				c.Exchanges[i].WebsocketOrderbookBufferLimit = defaultWebsocketOrderbookBufferLimit
			}
			err := c.CheckPairConsistency(c.Exchanges[i].Name)
			if err != nil {
				log.Errorf(log.ExchangeSys, "Exchange %s: CheckPairConsistency error: %s\n", c.Exchanges[i].Name, err)
				c.Exchanges[i].Enabled = false
				continue
			}

			c.CheckExchangeAssetsConsistency(c.Exchanges[i].Name)

			exchanges++
		}
	}
	if exchanges == 0 {
		return errors.New(ErrNoEnabledExchanges)
	}
	return nil
}

// CheckBankAccountConfig checks all bank accounts to see if they are valid
func (c *Config) CheckBankAccountConfig() {
	for x := range c.BankAccounts {
		if c.BankAccounts[x].Enabled {
			err := c.BankAccounts[x].Validate()
			if err != nil {
				c.BankAccounts[x].Enabled = false
				log.Warn(log.ConfigMgr, err.Error())
			}
		}
	}
	banking.Accounts = c.BankAccounts
}

// CheckCurrencyConfigValues checks to see if the currency config values are correct or not
func (c *Config) CheckCurrencyConfigValues() error {
	fxProviders := forexprovider.GetSupportedForexProviders()

	if len(fxProviders) != len(c.Currency.ForexProviders) {
		for x := range fxProviders {
			_, err := c.GetForexProvider(fxProviders[x])
			if err != nil {
				log.Warnf(log.Global, "%s forex provider not found, adding to config..\n", fxProviders[x])
				c.Currency.ForexProviders = append(c.Currency.ForexProviders, currency.FXSettings{
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
			if c.Currency.ForexProviders[i].Name == "CurrencyConverter" &&
				c.Currency.ForexProviders[i].PrimaryProvider &&
				(c.Currency.ForexProviders[i].APIKey == "" ||
					c.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey) {
				log.Warnln(log.Global, "CurrencyConverter forex provider no longer supports unset API key requests. Switching to ExchangeRates FX provider..")
				c.Currency.ForexProviders[i].Enabled = false
				c.Currency.ForexProviders[i].PrimaryProvider = false
				c.Currency.ForexProviders[i].APIKey = DefaultUnsetAPIKey
				c.Currency.ForexProviders[i].APIKeyLvl = -1
				continue
			}
			if c.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey &&
				c.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf(log.Global, "%s enabled forex provider API key not set. Please set this in your config.json file\n", c.Currency.ForexProviders[i].Name)
				c.Currency.ForexProviders[i].Enabled = false
				c.Currency.ForexProviders[i].PrimaryProvider = false
				continue
			}

			if c.Currency.ForexProviders[i].APIKeyLvl == -1 && c.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf(log.Global, "%s APIKey Level not set, functions limited. Please set this in your config.json file\n",
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
				log.Warnln(log.ConfigMgr, "Using ExchangeRatesAPI for default forex provider.")
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
			log.Warnln(log.ConfigMgr, "CryptocurrencyProvider enabled but api key is unset please set this in your config.json file")
		}
		if c.Currency.CryptocurrencyProvider.AccountPlan == "" ||
			c.Currency.CryptocurrencyProvider.AccountPlan == DefaultUnsetAccountPlan {
			log.Warnln(log.ConfigMgr, "CryptocurrencyProvider enabled but account plan is unset please set this in your config.json file")
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
func (c *Config) RetrieveConfigCurrencyPairs(enabledOnly bool, assetType asset.Item) error {
	cryptoCurrencies := c.Currency.Cryptocurrencies
	fiatCurrencies := currency.GetFiatCurrencies()

	for x := range c.Exchanges {
		if !c.Exchanges[x].Enabled && enabledOnly {
			continue
		}

		supports, _ := c.SupportsExchangeAssetType(c.Exchanges[x].Name, assetType)
		if !supports {
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
		supports, _ := c.SupportsExchangeAssetType(c.Exchanges[x].Name, assetType)
		if !supports {
			continue
		}

		var pairs []currency.Pair
		var err error
		if !c.Exchanges[x].Enabled && enabledOnly {
			pairs, err = c.GetEnabledPairs(c.Exchanges[x].Name, assetType)
		} else {
			pairs, err = c.GetAvailablePairs(c.Exchanges[x].Name, assetType)
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

// CheckLoggerConfig checks to see logger values are present and valid in config
// if not creates a default instance of the logger
func (c *Config) CheckLoggerConfig() error {
	m.Lock()
	defer m.Unlock()

	if c.Logging.Enabled == nil || c.Logging.Output == "" {
		c.Logging = log.GenDefaultSettings()
	}

	if c.Logging.AdvancedSettings.ShowLogSystemName == nil {
		c.Logging.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(false)
	}

	if c.Logging.LoggerFileConfig != nil {
		if c.Logging.LoggerFileConfig.FileName == "" {
			c.Logging.LoggerFileConfig.FileName = "log.txt"
		}
		if c.Logging.LoggerFileConfig.Rotate == nil {
			c.Logging.LoggerFileConfig.Rotate = convert.BoolPtr(false)
		}
		if c.Logging.LoggerFileConfig.MaxSize <= 0 {
			log.Warnf(log.Global, "Logger rotation size invalid, defaulting to %v", log.DefaultMaxFileSize)
			c.Logging.LoggerFileConfig.MaxSize = log.DefaultMaxFileSize
		}
		log.FileLoggingConfiguredCorrectly = true
	}
	log.RWM.Lock()
	log.GlobalLogConfig = &c.Logging
	log.RWM.Unlock()

	logPath := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "logs")
	err := common.CreateDir(logPath)
	if err != nil {
		return err
	}
	log.LogPath = logPath

	return nil
}

func (c *Config) checkGCTScriptConfig() error {
	m.Lock()
	defer m.Unlock()

	if c.GCTScript.ScriptTimeout <= 0 {
		c.GCTScript.ScriptTimeout = gctscript.DefaultTimeoutValue
	}

	if c.GCTScript.MaxVirtualMachines == 0 {
		c.GCTScript.MaxVirtualMachines = gctscript.DefaultMaxVirtualMachines
	}

	scriptPath := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "scripts")
	err := common.CreateDir(scriptPath)
	if err != nil {
		return err
	}

	gctscript.ScriptPath = scriptPath
	gctscript.GCTScriptConfig = &c.GCTScript

	return nil
}

func (c *Config) checkDatabaseConfig() error {
	m.Lock()
	defer m.Unlock()

	if (c.Database == database.Config{}) {
		c.Database.Driver = database.DBSQLite3
		c.Database.Database = database.DefaultSQLiteDatabase
	}

	if !c.Database.Enabled {
		return nil
	}

	if !common.StringDataCompare(database.SupportedDrivers, c.Database.Driver) {
		c.Database.Enabled = false
		return fmt.Errorf("unsupported database driver %v, database disabled", c.Database.Driver)
	}

	if c.Database.Driver == database.DBSQLite || c.Database.Driver == database.DBSQLite3 {
		databaseDir := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "/database")
		err := common.CreateDir(databaseDir)
		if err != nil {
			return err
		}
		database.DB.DataPath = databaseDir
	}

	database.DB.Config = &c.Database

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
		log.Warnln(log.ConfigMgr, "NTPClient enabled with no servers configured, enabling default pool.")
		c.NTPClient.Pool = []string{"pool.ntp.org:123"}
	}
}

// DisableNTPCheck allows the user to change how they are prompted for timesync alerts
func (c *Config) DisableNTPCheck(input io.Reader) (string, error) {
	m.Lock()
	defer m.Unlock()

	reader := bufio.NewReader(input)
	log.Warnln(log.ConfigMgr, "Your system time is out of sync, this may cause issues with trading")
	log.Warnln(log.ConfigMgr, "How would you like to show future notifications? (a)lert / (w)arn / (d)isable")

	var resp string
	answered := false
	for !answered {
		answer, err := reader.ReadString('\n')
		if err != nil {
			return resp, err
		}

		answer = strings.TrimRight(answer, "\r\n")
		switch answer {
		case "a":
			c.NTPClient.Level = 0
			resp = "Time sync has been set to alert"
			answered = true
		case "w":
			c.NTPClient.Level = 1
			resp = "Time sync has been set to warn only"
			answered = true
		case "d":
			c.NTPClient.Level = -1
			resp = "Future notifications for out of time sync has been disabled"
			answered = true
		default:
			log.Warnln(log.ConfigMgr,
				"Invalid option selected, please try again (a)lert / (w)arn / (d)isable")
		}
	}
	return resp, nil
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

// DefaultFilePath returns the default config file path
// MacOS/Linux: $HOME/.gocryptotrader/config.json or config.dat
// Windows: %APPDATA%\GoCryptoTrader\config.json or config.dat
// Helpful for printing application usage
func DefaultFilePath() string {
	f := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), File)
	if !file.Exists(f) {
		encFile := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), EncryptedFile)
		if file.Exists(encFile) {
			return encFile
		}
	}
	return f
}

// GetFilePath returns the desired config file or the default config file name
// based on if the application is being run under test or normal mode. It will
// also move/rename the config file under the following conditions:
// 1) If a config file is found in the executable path directory and no explicit
//    config path is set, plus no config is found in the GCT data dir, it will
//    move it to the GCT data dir. If a config already exists in the GCT data
//    dir, it will warn the user and load the config found in the GCT data dir
// 2) If a config file in the GCT data dir has the file extension .dat but
//    contains json data, it will rename to the file to config.json
// 3) If a config file in the GCT data dir has the file extension .json but
//    contains encrypted data, it will rename the file to config.dat
func GetFilePath(configfile string) (string, error) {
	if configfile != "" {
		return configfile, nil
	}

	if flag.Lookup("test.v") != nil && !testBypass {
		return TestFile, nil
	}

	exePath, err := common.GetExecutablePath()
	if err != nil {
		return "", err
	}

	oldDirs := []string{
		filepath.Join(exePath, File),
		filepath.Join(exePath, EncryptedFile),
	}

	newDir := common.GetDefaultDataDir(runtime.GOOS)
	err = common.CreateDir(newDir)
	if err != nil {
		return "", err
	}
	newDirs := []string{
		filepath.Join(newDir, File),
		filepath.Join(newDir, EncryptedFile),
	}

	// First upgrade the old dir config file if it exists to the corresponding
	// new one
	for x := range oldDirs {
		if !file.Exists(oldDirs[x]) {
			continue
		}
		if file.Exists(newDirs[x]) {
			log.Warnf(log.ConfigMgr,
				"config.json file found in root dir and gct dir; cannot overwrite, defaulting to gct dir config.json at %s",
				newDirs[x])
			return newDirs[x], nil
		}
		if filepath.Ext(oldDirs[x]) == ".json" {
			err = file.Move(oldDirs[x], newDirs[0])
			if err != nil {
				return "", err
			}
			log.Debugf(log.ConfigMgr,
				"Renamed old config file %s to %s\n",
				oldDirs[x],
				newDirs[0])
		} else {
			err = file.Move(oldDirs[x], newDirs[1])
			if err != nil {
				return "", err
			}
			log.Debugf(log.ConfigMgr,
				"Renamed old config file %s to %s\n",
				oldDirs[x],
				newDirs[1])
		}
	}

	// Secondly check to see if the new config file extension is correct or not
	for x := range newDirs {
		if !file.Exists(newDirs[x]) {
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

			err = file.Move(newDirs[x], newDirs[1])
			if err != nil {
				return "", err
			}
			return newDirs[1], nil
		}

		if filepath.Ext(newDirs[x]) == ".json" {
			return newDirs[x], nil
		}

		err = file.Move(newDirs[x], newDirs[0])
		if err != nil {
			return "", err
		}

		return newDirs[0], nil
	}

	return "", fmt.Errorf("config.json file not found in %s, please follow README.md in root dir for config generation",
		newDir)
}

// ReadConfig verifies and checks for encryption and verifies the unencrypted
// file contains JSON.
func (c *Config) ReadConfig(configPath string, dryrun bool) error {
	defaultPath, err := GetFilePath(configPath)
	if err != nil {
		return err
	}

	fileData, err := ioutil.ReadFile(defaultPath)
	if err != nil {
		return err
	}

	if !ConfirmECS(fileData) {
		err = ConfirmConfigJSON(fileData, &c)
		if err != nil {
			return err
		}

		if c.EncryptConfig == fileEncryptionDisabled {
			return nil
		}

		if c.EncryptConfig == fileEncryptionPrompt {
			m.Lock()
			IsInitialSetup = true
			m.Unlock()
			if c.PromptForConfigEncryption(configPath, dryrun) {
				c.EncryptConfig = fileEncryptionEnabled
				return c.SaveConfig(defaultPath, dryrun)
			}
		}
	} else {
		errCounter := 0
		for {
			if errCounter >= maxAuthFailures {
				return errors.New("failed to decrypt config after 3 attempts")
			}
			key, err := PromptForConfigKey(IsInitialSetup)
			if err != nil {
				log.Errorf(log.ConfigMgr, "PromptForConfigKey err: %s", err)
				errCounter++
				continue
			}

			var f []byte
			f = append(f, fileData...)
			data, err := DecryptConfigFile(f, key)
			if err != nil {
				log.Errorf(log.ConfigMgr, "DecryptConfigFile err: %s", err)
				errCounter++
				continue
			}

			err = ConfirmConfigJSON(data, &c)
			if err != nil {
				if errCounter < maxAuthFailures {
					log.Error(log.ConfigMgr, "Invalid password.")
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
func (c *Config) SaveConfig(configPath string, dryrun bool) error {
	if dryrun {
		return nil
	}

	defaultPath, err := GetFilePath(configPath)
	if err != nil {
		return err
	}

	payload, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	if c.EncryptConfig == fileEncryptionEnabled {
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
	return file.Write(defaultPath, payload)
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
		log.Errorf(log.ConfigMgr, "Failed to configure logger, some logging features unavailable: %s\n", err)
	}

	err = c.checkDatabaseConfig()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to configure database: %v", err)
	}

	err = c.CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	err = c.checkGCTScriptConfig()
	if err != nil {
		log.Errorf(log.Global, "Failed to configure gctscript, feature has been disabled: %s\n", err)
	}

	c.CheckConnectionMonitorConfig()
	c.CheckCommunicationsConfig()
	c.CheckClientBankAccounts()
	c.CheckBankAccountConfig()
	c.CheckRemoteControlConfig()

	err = c.CheckCurrencyConfigValues()
	if err != nil {
		return err
	}

	if c.GlobalHTTPTimeout <= 0 {
		log.Warnf(log.ConfigMgr, "Global HTTP Timeout value not set, defaulting to %v.\n", defaultHTTPTimeout)
		c.GlobalHTTPTimeout = defaultHTTPTimeout
	}

	if c.NTPClient.Level != 0 {
		c.CheckNTPConfig()
	}

	return nil
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig(configPath string, dryrun bool) error {
	err := c.ReadConfig(configPath, dryrun)
	if err != nil {
		return fmt.Errorf(ErrFailureOpeningConfig, configPath, err)
	}

	return c.CheckConfig()
}

// UpdateConfig updates the config with a supplied config file
func (c *Config) UpdateConfig(configPath string, newCfg *Config, dryrun bool) error {
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

	err = c.SaveConfig(configPath, dryrun)
	if err != nil {
		return err
	}

	return c.LoadConfig(configPath, dryrun)
}

// GetConfig returns a pointer to a configuration object
func GetConfig() *Config {
	return &Cfg
}
