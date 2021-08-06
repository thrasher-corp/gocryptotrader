package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
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
func (c *Config) GetCommunicationsConfig() base.CommunicationsConfig {
	m.Lock()
	comms := c.Communications
	m.Unlock()
	return comms
}

// UpdateCommunicationsConfig sets a new updated version of a Communications
// configuration
func (c *Config) UpdateCommunicationsConfig(config *base.CommunicationsConfig) {
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
		c.Communications.SlackConfig = base.SlackConfig{
			Name:              "Slack",
			TargetChannel:     "general",
			VerificationToken: "testtest",
		}
	}

	if c.Communications.SMSGlobalConfig.Name == "" {
		if c.SMS != nil {
			if c.SMS.Contacts != nil {
				c.Communications.SMSGlobalConfig = base.SMSGlobalConfig{
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
				c.Communications.SMSGlobalConfig = base.SMSGlobalConfig{
					Name:     "SMSGlobal",
					From:     c.Name,
					Username: "main",
					Password: "test",

					Contacts: []base.SMSContact{
						{
							Name:    "bob",
							Number:  "1234",
							Enabled: false,
						},
					},
				}
			}
		} else {
			c.Communications.SMSGlobalConfig = base.SMSGlobalConfig{
				Name:     "SMSGlobal",
				Username: "main",
				Password: "test",

				Contacts: []base.SMSContact{
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
		c.Communications.SMTPConfig = base.SMTPConfig{
			Name:            "SMTP",
			Host:            "smtp.google.com",
			Port:            "537",
			AccountName:     "some",
			AccountPassword: "password",
			RecipientList:   "lol123@gmail.com",
		}
	}

	if c.Communications.TelegramConfig.Name == "" {
		c.Communications.TelegramConfig = base.TelegramConfig{
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

	return exchCfg.CurrencyPairs.GetAssetTypes(false), nil
}

// SupportsExchangeAssetType returns whether or not the exchange supports the supplied asset type
func (c *Config) SupportsExchangeAssetType(exchName string, assetType asset.Item) error {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	if exchCfg.CurrencyPairs == nil {
		return fmt.Errorf("exchange %s currency pairs is nil", exchName)
	}

	if !assetType.IsValid() {
		return fmt.Errorf("exchange %s invalid asset type %s",
			exchName,
			assetType)
	}

	if !exchCfg.CurrencyPairs.GetAssetTypes(false).Contains(assetType) {
		return fmt.Errorf("exchange %s unsupported asset type %s",
			exchName,
			assetType)
	}
	return nil
}

// SetPairs sets the exchanges currency pairs
func (c *Config) SetPairs(exchName string, assetType asset.Item, enabled bool, pairs currency.Pairs) error {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	err = c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return err
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

	err = c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return nil, err
	}

	return exchCfg.CurrencyPairs.Get(assetType)
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

	var atLeastOneEnabled bool
	for x := range assetTypes {
		enabledPairs, err := c.GetEnabledPairs(exchName, assetTypes[x])
		if err == nil {
			if len(enabledPairs) != 0 {
				atLeastOneEnabled = true
				continue
			}
			var enabled bool
			enabled, err = c.AssetTypeEnabled(assetTypes[x], exchName)
			if err != nil {
				return err
			}

			if !enabled {
				continue
			}

			var availPairs currency.Pairs
			availPairs, err = c.GetAvailablePairs(exchName, assetTypes[x])
			if err != nil {
				return err
			}

			err = c.SetPairs(exchName,
				assetTypes[x],
				true,
				currency.Pairs{availPairs.GetRandomPair()})
			if err != nil {
				return err
			}
			atLeastOneEnabled = true
			continue
		}

		// On error an enabled pair is not found in the available pairs list
		// so remove and report
		availPairs, err := c.GetAvailablePairs(exchName, assetTypes[x])
		if err != nil {
			return err
		}

		var pairs, pairsRemoved currency.Pairs
		for x := range enabledPairs {
			if !availPairs.Contains(enabledPairs[x], true) {
				pairsRemoved = append(pairsRemoved, enabledPairs[x])
				continue
			}
			pairs = append(pairs, enabledPairs[x])
		}

		if len(pairsRemoved) == 0 {
			return fmt.Errorf("check pair consistency fault for asset %s, conflict found but no pairs removed",
				assetTypes[x])
		}

		// Flush corrupted/misspelled enabled pairs in config
		err = c.SetPairs(exchName, assetTypes[x], true, pairs)
		if err != nil {
			return err
		}

		log.Warnf(log.ConfigMgr,
			"Exchange %s: [%v] Removing enabled pair(s) %v from enabled pairs list, as it isn't located in the available pairs list.\n",
			exchName,
			assetTypes[x],
			pairsRemoved.Strings())

		if len(pairs) != 0 {
			atLeastOneEnabled = true
			continue
		}

		enabled, err := c.AssetTypeEnabled(assetTypes[x], exchName)
		if err != nil {
			return err
		}

		if !enabled {
			continue
		}

		err = c.SetPairs(exchName,
			assetTypes[x],
			true,
			currency.Pairs{availPairs.GetRandomPair()})
		if err != nil {
			return err
		}
		atLeastOneEnabled = true
	}

	// If no pair is enabled across the entire range of assets, then atleast
	// enable one and turn on the asset type
	if !atLeastOneEnabled {
		avail, err := c.GetAvailablePairs(exchName, assetTypes[0])
		if err != nil {
			return err
		}

		newPair := avail.GetRandomPair()
		err = c.SetPairs(exchName, assetTypes[0], true, currency.Pairs{newPair})
		if err != nil {
			return err
		}
		log.Warnf(log.ConfigMgr,
			"Exchange %s: [%v] No enabled pairs found in available pairs list, randomly added %v pair.\n",
			exchName,
			assetTypes[0],
			newPair)
	}
	return nil
}

// SupportsPair returns true or not whether the exchange supports the supplied
// pair
func (c *Config) SupportsPair(exchName string, p currency.Pair, assetType asset.Item) bool {
	pairs, err := c.GetAvailablePairs(exchName, assetType)
	if err != nil {
		return false
	}
	return pairs.Contains(p, false)
}

// GetPairFormat returns the exchanges pair config storage format
func (c *Config) GetPairFormat(exchName string, assetType asset.Item) (currency.PairFormat, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return currency.PairFormat{}, err
	}

	err = c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return currency.PairFormat{}, err
	}

	if exchCfg.CurrencyPairs.UseGlobalFormat {
		return *exchCfg.CurrencyPairs.ConfigFormat, nil
	}

	p, err := exchCfg.CurrencyPairs.Get(assetType)
	if err != nil {
		return currency.PairFormat{}, err
	}

	if p == nil {
		return currency.PairFormat{},
			fmt.Errorf("exchange %s pair store for asset type %s is nil",
				exchName,
				assetType)
	}

	if p.ConfigFormat == nil {
		return currency.PairFormat{},
			fmt.Errorf("exchange %s pair config format for asset type %s is nil",
				exchName,
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

	pairs, err := exchCfg.CurrencyPairs.GetPairs(assetType, false)
	if err != nil {
		return nil, err
	}

	if pairs == nil {
		return nil, nil
	}

	return pairs.Format(pairFormat.Delimiter, pairFormat.Index,
		pairFormat.Uppercase), nil
}

// GetEnabledPairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetEnabledPairs(exchName string, assetType asset.Item) (currency.Pairs, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairFormat, err := c.GetPairFormat(exchName, assetType)
	if err != nil {
		return nil, err
	}

	pairs, err := exchCfg.CurrencyPairs.GetPairs(assetType, true)
	if err != nil {
		return pairs, err
	}

	if pairs == nil {
		return nil, nil
	}

	return pairs.Format(pairFormat.Delimiter,
			pairFormat.Index,
			pairFormat.Uppercase),
		nil
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
	return nil, fmt.Errorf("%s %w", name, ErrExchangeNotFound)
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
	return fmt.Errorf("%s %w", e.Name, ErrExchangeNotFound)
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

		// Check if see if the new currency pairs format is empty and flesh it out if so
		if c.Exchanges[i].CurrencyPairs == nil {
			c.Exchanges[i].CurrencyPairs = new(currency.PairsManager)
			c.Exchanges[i].CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)

			if c.Exchanges[i].PairsLastUpdated != nil {
				c.Exchanges[i].CurrencyPairs.LastUpdated = *c.Exchanges[i].PairsLastUpdated
			}

			c.Exchanges[i].CurrencyPairs.ConfigFormat = c.Exchanges[i].ConfigCurrencyPairFormat
			c.Exchanges[i].CurrencyPairs.RequestFormat = c.Exchanges[i].RequestCurrencyPairFormat

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
					AssetEnabled: convert.BoolPtr(true),
					Available:    availPairs,
					Enabled:      enabledPairs,
				},
			)

			// flush old values
			c.Exchanges[i].PairsLastUpdated = nil
			c.Exchanges[i].ConfigCurrencyPairFormat = nil
			c.Exchanges[i].RequestCurrencyPairFormat = nil
			c.Exchanges[i].AssetTypes = nil
			c.Exchanges[i].AvailablePairs = nil
			c.Exchanges[i].EnabledPairs = nil
		} else {
			assets := c.Exchanges[i].CurrencyPairs.GetAssetTypes(false)
			var atLeastOne bool
			for index := range assets {
				err := c.Exchanges[i].CurrencyPairs.IsAssetEnabled(assets[index])
				if err != nil {
					// Checks if we have an old config without the ability to
					// enable disable the entire asset
					if err.Error() == "cannot ascertain if asset is enabled, variable is nil" {
						log.Warnf(log.ConfigMgr,
							"Exchange %s: upgrading config for asset type %s and setting enabled.\n",
							c.Exchanges[i].Name,
							assets[index])
						err = c.Exchanges[i].CurrencyPairs.SetAssetEnabled(assets[index], true)
						if err != nil {
							return err
						}
						atLeastOne = true
					}
					continue
				}
				atLeastOne = true
			}

			if !atLeastOne {
				if len(assets) == 0 {
					c.Exchanges[i].Enabled = false
					log.Warnf(log.ConfigMgr,
						"%s no assets found, disabling...",
						c.Exchanges[i].Name)
					continue
				}

				// turn on an asset if all disabled
				log.Warnf(log.ConfigMgr,
					"%s assets disabled, turning on asset %s",
					c.Exchanges[i].Name,
					assets[0])

				err := c.Exchanges[i].CurrencyPairs.SetAssetEnabled(assets[0], true)
				if err != nil {
					return err
				}
			}
		}

		if c.Exchanges[i].Enabled {
			if c.Exchanges[i].Name == "" {
				log.Errorf(log.ConfigMgr, ErrExchangeNameEmpty, i)
				c.Exchanges[i].Enabled = false
				continue
			}
			if (c.Exchanges[i].API.AuthenticatedSupport || c.Exchanges[i].API.AuthenticatedWebsocketSupport) &&
				c.Exchanges[i].API.CredentialsValidator != nil {
				var failed bool
				if c.Exchanges[i].API.CredentialsValidator.RequiresKey &&
					(c.Exchanges[i].API.Credentials.Key == "" || c.Exchanges[i].API.Credentials.Key == DefaultAPIKey) {
					failed = true
				}

				if c.Exchanges[i].API.CredentialsValidator.RequiresSecret &&
					(c.Exchanges[i].API.Credentials.Secret == "" || c.Exchanges[i].API.Credentials.Secret == DefaultAPISecret) {
					failed = true
				}

				if c.Exchanges[i].API.CredentialsValidator.RequiresClientID &&
					(c.Exchanges[i].API.Credentials.ClientID == DefaultAPIClientID || c.Exchanges[i].API.Credentials.ClientID == "") {
					failed = true
				}

				if failed {
					c.Exchanges[i].API.AuthenticatedSupport = false
					c.Exchanges[i].API.AuthenticatedWebsocketSupport = false
					log.Warnf(log.ConfigMgr, WarningExchangeAuthAPIDefaultOrEmptyValues, c.Exchanges[i].Name)
				}
			}
			if !c.Exchanges[i].Features.Supports.RESTCapabilities.AutoPairUpdates &&
				!c.Exchanges[i].Features.Supports.WebsocketCapabilities.AutoPairUpdates {
				lastUpdated := convert.UnixTimestampToTime(c.Exchanges[i].CurrencyPairs.LastUpdated)
				lastUpdated = lastUpdated.AddDate(0, 0, pairsLastUpdatedWarningThreshold)
				if lastUpdated.Unix() <= time.Now().Unix() {
					log.Warnf(log.ConfigMgr,
						WarningPairsLastUpdatedThresholdExceeded,
						c.Exchanges[i].Name,
						pairsLastUpdatedWarningThreshold)
				}
			}
			if c.Exchanges[i].HTTPTimeout <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s HTTP Timeout value not set, defaulting to %v.\n",
					c.Exchanges[i].Name,
					defaultHTTPTimeout)
				c.Exchanges[i].HTTPTimeout = defaultHTTPTimeout
			}

			if c.Exchanges[i].WebsocketResponseCheckTimeout <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket response check timeout value not set, defaulting to %v.",
					c.Exchanges[i].Name,
					defaultWebsocketResponseCheckTimeout)
				c.Exchanges[i].WebsocketResponseCheckTimeout = defaultWebsocketResponseCheckTimeout
			}

			if c.Exchanges[i].WebsocketResponseMaxLimit <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket response max limit value not set, defaulting to %v.",
					c.Exchanges[i].Name,
					defaultWebsocketResponseMaxLimit)
				c.Exchanges[i].WebsocketResponseMaxLimit = defaultWebsocketResponseMaxLimit
			}
			if c.Exchanges[i].WebsocketTrafficTimeout <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket response traffic timeout value not set, defaulting to %v.",
					c.Exchanges[i].Name,
					defaultWebsocketTrafficTimeout)
				c.Exchanges[i].WebsocketTrafficTimeout = defaultWebsocketTrafficTimeout
			}
			if c.Exchanges[i].OrderbookConfig.WebsocketBufferLimit <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket orderbook buffer limit value not set, defaulting to %v.",
					c.Exchanges[i].Name,
					defaultWebsocketOrderbookBufferLimit)
				c.Exchanges[i].OrderbookConfig.WebsocketBufferLimit = defaultWebsocketOrderbookBufferLimit
			}
			err := c.CheckPairConsistency(c.Exchanges[i].Name)
			if err != nil {
				log.Errorf(log.ConfigMgr,
					"Exchange %s: CheckPairConsistency error: %s\n",
					c.Exchanges[i].Name,
					err)
				c.Exchanges[i].Enabled = false
				continue
			}
			for x := range c.Exchanges[i].BankAccounts {
				if !c.Exchanges[i].BankAccounts[x].Enabled {
					continue
				}
				err := c.Exchanges[i].BankAccounts[x].Validate()
				if err != nil {
					c.Exchanges[i].BankAccounts[x].Enabled = false
					log.Warnln(log.ConfigMgr, err.Error())
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
	banking.SetAccounts(c.BankAccounts...)
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
			if (c.Currency.ForexProviders[i].Name == "CurrencyConverter" || c.Currency.ForexProviders[i].Name == "ExchangeRates") &&
				c.Currency.ForexProviders[i].PrimaryProvider &&
				(c.Currency.ForexProviders[i].APIKey == "" ||
					c.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey) {
				log.Warnf(log.Global, "%s forex provider no longer supports unset API key requests. Switching to %s FX provider..",
					c.Currency.ForexProviders[i].Name, DefaultForexProviderExchangeRatesAPI)
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
				log.Warnf(log.ConfigMgr, "No valid forex providers configured. Defaulting to %s.",
					DefaultForexProviderExchangeRatesAPI)
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

		err := c.SupportsExchangeAssetType(c.Exchanges[x].Name, assetType)
		if err != nil {
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
		err := c.SupportsExchangeAssetType(c.Exchanges[x].Name, assetType)
		if err != nil {
			continue
		}

		var pairs []currency.Pair
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

	logPath := c.GetDataPath("logs")
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

	scriptPath := c.GetDataPath("scripts")
	err := common.CreateDir(scriptPath)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(scriptPath, "output")
	err = common.CreateDir(outputPath)
	if err != nil {
		return err
	}

	gctscript.ScriptPath = scriptPath

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
		databaseDir := c.GetDataPath("database")
		err := common.CreateDir(databaseDir)
		if err != nil {
			return err
		}
		database.DB.DataPath = databaseDir
	}

	return database.DB.SetConfig(&c.Database)
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

// SetNTPCheck allows the user to change how they are prompted for timesync alerts
func (c *Config) SetNTPCheck(input io.Reader) (string, error) {
	m.Lock()
	defer m.Unlock()

	reader := bufio.NewReader(input)
	log.Warnln(log.ConfigMgr, "Your system time is out of sync, this may cause issues with trading")
	log.Warnln(log.ConfigMgr, "How would you like to show future notifications? (a)lert at startup / (w)arn periodically / (d)isable")

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

// CheckDataHistoryMonitorConfig ensures the data history config is
// valid, or sets default values
func (c *Config) CheckDataHistoryMonitorConfig() {
	m.Lock()
	defer m.Unlock()
	if c.DataHistoryManager.CheckInterval <= 0 {
		c.DataHistoryManager.CheckInterval = defaultDataHistoryMonitorCheckTimer
	}
	if c.DataHistoryManager.MaxJobsPerCycle == 0 {
		c.DataHistoryManager.MaxJobsPerCycle = defaultMaxJobsPerCycle
	}
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
	foundConfig, _, err := GetFilePath("")
	if err != nil {
		// If there was no config file, show default location for .json
		return filepath.Join(common.GetDefaultDataDir(runtime.GOOS), File)
	}
	return foundConfig
}

// GetAndMigrateDefaultPath returns the target config file
// migrating it from the old default location to new one,
// if it was implicitly loaded from a default location and
// wasn't already in the correct 'new' default location
func GetAndMigrateDefaultPath(configFile string) (string, error) {
	filePath, wasDefault, err := GetFilePath(configFile)
	if err != nil {
		return "", err
	}
	if wasDefault {
		return migrateConfig(filePath, common.GetDefaultDataDir(runtime.GOOS))
	}
	return filePath, nil
}

// GetFilePath returns the desired config file or the default config file name
// and whether it was loaded from a default location (rather than explicitly specified)
func GetFilePath(configFile string) (configPath string, isImplicitDefaultPath bool, err error) {
	if configFile != "" {
		return configFile, false, nil
	}

	exePath, err := common.GetExecutablePath()
	if err != nil {
		return "", false, err
	}
	newDir := common.GetDefaultDataDir(runtime.GOOS)
	defaultPaths := []string{
		filepath.Join(exePath, File),
		filepath.Join(exePath, EncryptedFile),
		filepath.Join(newDir, File),
		filepath.Join(newDir, EncryptedFile),
	}

	for _, p := range defaultPaths {
		if file.Exists(p) {
			configFile = p
			break
		}
	}
	if configFile == "" {
		return "", false, fmt.Errorf("config.json file not found in %s, please follow README.md in root dir for config generation",
			newDir)
	}

	return configFile, true, nil
}

// migrateConfig will move the config file to the target
// config directory as `File` or `EncryptedFile` depending on whether the config
// is encrypted
func migrateConfig(configFile, targetDir string) (string, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return "", err
	}

	var target string
	if ConfirmECS(data) {
		target = EncryptedFile
	} else {
		target = File
	}
	target = filepath.Join(targetDir, target)
	if configFile == target {
		return configFile, nil
	}
	if file.Exists(target) {
		log.Warnf(log.ConfigMgr, "config file already found in '%s'; not overwriting, defaulting to %s", target, configFile)
		return configFile, nil
	}

	err = file.Move(configFile, target)
	if err != nil {
		return "", err
	}

	return target, nil
}

// ReadConfigFromFile reads the configuration from the given file
// if target file is encrypted, prompts for encryption key
// Also - if not in dryrun mode - it checks if the configuration needs to be encrypted
// and stores the file as encrypted, if necessary (prompting for enryption key)
func (c *Config) ReadConfigFromFile(configPath string, dryrun bool) error {
	defaultPath, _, err := GetFilePath(configPath)
	if err != nil {
		return err
	}
	confFile, err := os.Open(defaultPath)
	if err != nil {
		return err
	}
	defer confFile.Close()
	result, wasEncrypted, err := ReadConfig(confFile, func() ([]byte, error) { return PromptForConfigKey(false) })
	if err != nil {
		return fmt.Errorf("error reading config %w", err)
	}
	// Override values in the current config
	*c = *result

	if dryrun || wasEncrypted || c.EncryptConfig == fileEncryptionDisabled {
		return nil
	}

	if c.EncryptConfig == fileEncryptionPrompt {
		confirm, err := promptForConfigEncryption()
		if err != nil {
			log.Errorf(log.ConfigMgr, "The encryption prompt failed, ignoring for now, next time we will prompt again. Error: %s\n", err)
			return nil
		}
		if confirm {
			c.EncryptConfig = fileEncryptionEnabled
			return c.SaveConfigToFile(defaultPath)
		}

		c.EncryptConfig = fileEncryptionDisabled
		err = c.SaveConfigToFile(defaultPath)
		if err != nil {
			log.Errorf(log.ConfigMgr, "Cannot save config. Error: %s\n", err)
		}
	}
	return nil
}

// ReadConfig verifies and checks for encryption and loads the config from a JSON object.
// Prompts for decryption key, if target data is encrypted.
// Returns the loaded configuration and whether it was encrypted.
func ReadConfig(configReader io.Reader, keyProvider func() ([]byte, error)) (*Config, bool, error) {
	reader := bufio.NewReader(configReader)

	pref, err := reader.Peek(len(EncryptConfirmString))
	if err != nil {
		return nil, false, err
	}

	if !ConfirmECS(pref) {
		// Read unencrypted configuration
		decoder := json.NewDecoder(reader)
		c := &Config{}
		err = decoder.Decode(c)
		return c, false, err
	}

	conf, err := readEncryptedConfWithKey(reader, keyProvider)
	return conf, true, err
}

// readEncryptedConf reads encrypted configuration and requests key from provider
func readEncryptedConfWithKey(reader *bufio.Reader, keyProvider func() ([]byte, error)) (*Config, error) {
	fileData, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	for errCounter := 0; errCounter < maxAuthFailures; errCounter++ {
		key, err := keyProvider()
		if err != nil {
			log.Errorf(log.ConfigMgr, "PromptForConfigKey err: %s", err)
			continue
		}

		var c *Config
		c, err = readEncryptedConf(bytes.NewReader(fileData), key)
		if err != nil {
			log.Error(log.ConfigMgr, "Could not decrypt and deserialise data with given key. Invalid password?", err)
			continue
		}
		return c, nil
	}
	return nil, errors.New("failed to decrypt config after 3 attempts")
}

func readEncryptedConf(reader io.Reader, key []byte) (*Config, error) {
	c := &Config{}
	data, err := c.decryptConfigData(reader, key)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, c)
	return c, err
}

// SaveConfigToFile saves your configuration to your desired path as a JSON object.
// The function encrypts the data and prompts for encryption key, if necessary
func (c *Config) SaveConfigToFile(configPath string) error {
	defaultPath, _, err := GetFilePath(configPath)
	if err != nil {
		return err
	}
	var writer *os.File
	provider := func() (io.Writer, error) {
		writer, err = file.Writer(defaultPath)
		return writer, err
	}
	defer func() {
		if writer != nil {
			err = writer.Close()
			if err != nil {
				log.Error(log.Global, err)
			}
		}
	}()
	return c.Save(provider, func() ([]byte, error) { return PromptForConfigKey(true) })
}

// Save saves your configuration to the writer as a JSON object
// with encryption, if configured
// If there is an error when preparing the data to store, the writer is never requested
func (c *Config) Save(writerProvider func() (io.Writer, error), keyProvider func() ([]byte, error)) error {
	payload, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	if c.EncryptConfig == fileEncryptionEnabled {
		// Ensure we have the key from session or from user
		if len(c.sessionDK) == 0 {
			var key []byte
			key, err = keyProvider()
			if err != nil {
				return err
			}
			var sessionDK, storedSalt []byte
			sessionDK, storedSalt, err = makeNewSessionDK(key)
			if err != nil {
				return err
			}
			c.sessionDK, c.storedSalt = sessionDK, storedSalt
		}
		payload, err = c.encryptConfigFile(payload)
		if err != nil {
			return err
		}
	}
	configWriter, err := writerProvider()
	if err != nil {
		return err
	}
	_, err = io.Copy(configWriter, bytes.NewReader(payload))
	return err
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
		log.Errorf(log.ConfigMgr,
			"Failed to configure logger, some logging features unavailable: %s\n",
			err)
	}

	err = c.checkDatabaseConfig()
	if err != nil {
		log.Errorf(log.DatabaseMgr,
			"Failed to configure database: %v",
			err)
	}

	err = c.CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	err = c.checkGCTScriptConfig()
	if err != nil {
		log.Errorf(log.Global,
			"Failed to configure gctscript, feature has been disabled: %s\n",
			err)
	}

	c.CheckConnectionMonitorConfig()
	c.CheckDataHistoryMonitorConfig()
	c.CheckCommunicationsConfig()
	c.CheckClientBankAccounts()
	c.CheckBankAccountConfig()
	c.CheckRemoteControlConfig()

	err = c.CheckCurrencyConfigValues()
	if err != nil {
		return err
	}

	if c.GlobalHTTPTimeout <= 0 {
		log.Warnf(log.ConfigMgr,
			"Global HTTP Timeout value not set, defaulting to %v.\n",
			defaultHTTPTimeout)
		c.GlobalHTTPTimeout = defaultHTTPTimeout
	}

	if c.NTPClient.Level != 0 {
		c.CheckNTPConfig()
	}

	return nil
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig(configPath string, dryrun bool) error {
	err := c.ReadConfigFromFile(configPath, dryrun)
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

	if !dryrun {
		err = c.SaveConfigToFile(configPath)
		if err != nil {
			return err
		}
	}

	return c.LoadConfig(configPath, dryrun)
}

// GetConfig returns a pointer to a configuration object
func GetConfig() *Config {
	return &Cfg
}

// RemoveExchange removes an exchange config
func (c *Config) RemoveExchange(exchName string) bool {
	m.Lock()
	defer m.Unlock()
	for x := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[x].Name, exchName) {
			c.Exchanges = append(c.Exchanges[:x], c.Exchanges[x+1:]...)
			return true
		}
	}
	return false
}

// AssetTypeEnabled checks to see if the asset type is enabled in configuration
func (c *Config) AssetTypeEnabled(a asset.Item, exch string) (bool, error) {
	cfg, err := c.GetExchangeConfig(exch)
	if err != nil {
		return false, err
	}

	err = cfg.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetDataPath gets the data path for the given subpath
func (c *Config) GetDataPath(elem ...string) string {
	var baseDir string
	if c.DataDirectory != "" {
		baseDir = c.DataDirectory
	} else {
		baseDir = common.GetDefaultDataDir(runtime.GOOS)
	}
	return filepath.Join(append([]string{baseDir}, elem...)...)
}
