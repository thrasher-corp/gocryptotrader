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
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

// GetCurrencyConfig returns currency configurations
func (c *Config) GetCurrencyConfig() CurrencyConfig {
	return c.d.Currency
}

// GetExchangeBankAccounts returns banking details associated with an exchange
// for depositing funds
func (c *Config) GetExchangeBankAccounts(exchangeName, id, depositingCurrency string) (*banking.Account, error) {
	c.Lock()
	defer c.Unlock()

	for x := range c.d.Exchanges {
		if strings.EqualFold(c.d.Exchanges[x].Name, exchangeName) {
			for y := range c.d.Exchanges[x].BankAccounts {
				if strings.EqualFold(c.d.Exchanges[x].BankAccounts[y].ID, id) {
					if common.StringDataCompareInsensitive(
						strings.Split(c.d.Exchanges[x].BankAccounts[y].SupportedCurrencies, ","),
						depositingCurrency) {
						return &c.d.Exchanges[x].BankAccounts[y], nil
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
	c.Lock()
	defer c.Unlock()

	for i := range c.d.Exchanges {
		if strings.EqualFold(c.d.Exchanges[i].Name, exchangeName) {
			c.d.Exchanges[i].BankAccounts = bankCfg
			return nil
		}
	}
	return fmt.Errorf("exchange %s not found",
		exchangeName)
}

// GetClientBankAccounts returns banking details used for a given exchange
// and currency
func (c *Config) GetClientBankAccounts(exchangeName, targetCurrency string) (*banking.Account, error) {
	c.Lock()
	defer c.Unlock()

	for x := range c.d.BankAccounts {
		if (strings.Contains(c.d.BankAccounts[x].SupportedExchanges, exchangeName) ||
			c.d.BankAccounts[x].SupportedExchanges == "ALL") &&
			strings.Contains(c.d.BankAccounts[x].SupportedCurrencies, targetCurrency) {
			return &c.d.BankAccounts[x], nil
		}
	}
	return nil, fmt.Errorf("client banking details not found for %s and currency %s",
		exchangeName,
		targetCurrency)
}

// UpdateClientBankAccounts updates the configuration for a bank
func (c *Config) UpdateClientBankAccounts(bankCfg *banking.Account) error {
	c.Lock()
	defer c.Unlock()

	for i := range c.d.BankAccounts {
		if c.d.BankAccounts[i].BankName == bankCfg.BankName && c.d.BankAccounts[i].AccountNumber == bankCfg.AccountNumber {
			c.d.BankAccounts[i] = *bankCfg
			return nil
		}
	}
	return fmt.Errorf("client banking details for %s not found, update not applied",
		bankCfg.BankName)
}

// CheckClientBankAccounts checks client bank details
func (c *Config) CheckClientBankAccounts() {
	c.Lock()
	defer c.Unlock()

	if len(c.d.BankAccounts) == 0 {
		c.d.BankAccounts = append(c.d.BankAccounts,
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

	for i := range c.d.BankAccounts {
		if c.d.BankAccounts[i].Enabled {
			err := c.d.BankAccounts[i].Validate()
			if err != nil {
				c.d.BankAccounts[i].Enabled = false
				log.Warn(log.ConfigMgr, err.Error())
			}
		}
	}
}

// PurgeExchangeAPICredentials purges the stored API credentials
func (c *Config) PurgeExchangeAPICredentials() {
	c.Lock()
	defer c.Unlock()
	for x := range c.d.Exchanges {
		if !c.d.Exchanges[x].API.AuthenticatedSupport && !c.d.Exchanges[x].API.AuthenticatedWebsocketSupport {
			continue
		}
		c.d.Exchanges[x].API.AuthenticatedSupport = false
		c.d.Exchanges[x].API.AuthenticatedWebsocketSupport = false

		if c.d.Exchanges[x].API.CredentialsValidator.RequiresKey {
			c.d.Exchanges[x].API.Credentials.Key = DefaultAPIKey
		}

		if c.d.Exchanges[x].API.CredentialsValidator.RequiresSecret {
			c.d.Exchanges[x].API.Credentials.Secret = DefaultAPISecret
		}

		if c.d.Exchanges[x].API.CredentialsValidator.RequiresClientID {
			c.d.Exchanges[x].API.Credentials.ClientID = DefaultAPIClientID
		}

		c.d.Exchanges[x].API.Credentials.PEMKey = ""
		c.d.Exchanges[x].API.Credentials.OTPSecret = ""
	}
}

// GetCommunicationsConfig returns the communications configuration
func (c *Config) GetCommunicationsConfig() CommunicationsConfig {
	c.Lock()
	comms := c.d.Communications
	c.Unlock()
	return comms
}

// UpdateCommunicationsConfig sets a new updated version of a Communications
// configuration
func (c *Config) UpdateCommunicationsConfig(config *CommunicationsConfig) {
	c.Lock()
	c.d.Communications = *config
	c.Unlock()
}

// GetCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) GetCryptocurrencyProviderConfig() CryptocurrencyProvider {
	c.Lock()
	provider := c.d.Currency.CryptocurrencyProvider
	c.Unlock()
	return provider
}

// UpdateCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) UpdateCryptocurrencyProviderConfig(config CryptocurrencyProvider) {
	c.Lock()
	c.d.Currency.CryptocurrencyProvider = config
	c.Unlock()
}

// CheckCommunicationsConfig checks to see if the variables are set correctly
// from config.json
func (c *Config) CheckCommunicationsConfig() {
	c.Lock()
	defer c.Unlock()

	// If the communications config hasn't been populated, populate
	// with example settings

	if c.d.Communications.SlackConfig.Name == "" {
		c.d.Communications.SlackConfig = SlackConfig{
			Name:              "Slack",
			TargetChannel:     "general",
			VerificationToken: "testtest",
		}
	}

	if c.d.Communications.SMSGlobalConfig.Name == "" {
		if c.d.SMS != nil {
			if c.d.SMS.Contacts != nil {
				c.d.Communications.SMSGlobalConfig = SMSGlobalConfig{
					Name:     "SMSGlobal",
					Enabled:  c.d.SMS.Enabled,
					Verbose:  c.d.SMS.Verbose,
					Username: c.d.SMS.Username,
					Password: c.d.SMS.Password,
					Contacts: c.d.SMS.Contacts,
				}
				// flush old SMS config
				c.d.SMS = nil
			} else {
				c.d.Communications.SMSGlobalConfig = SMSGlobalConfig{
					Name:     "SMSGlobal",
					From:     c.d.Name,
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
			c.d.Communications.SMSGlobalConfig = SMSGlobalConfig{
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
		if c.d.Communications.SMSGlobalConfig.From == "" {
			c.d.Communications.SMSGlobalConfig.From = c.d.Name
		}

		if len(c.d.Communications.SMSGlobalConfig.From) > 11 {
			log.Warnf(log.ConfigMgr, "SMSGlobal config supplied from name exceeds 11 characters, trimming.\n")
			c.d.Communications.SMSGlobalConfig.From = c.d.Communications.SMSGlobalConfig.From[:11]
		}

		if c.d.SMS != nil {
			// flush old SMS config
			c.d.SMS = nil
		}
	}

	if c.d.Communications.SMTPConfig.Name == "" {
		c.d.Communications.SMTPConfig = SMTPConfig{
			Name:            "SMTP",
			Host:            "smtp.google.com",
			Port:            "537",
			AccountName:     "some",
			AccountPassword: "password",
			RecipientList:   "lol123@gmail.com",
		}
	}

	if c.d.Communications.TelegramConfig.Name == "" {
		c.d.Communications.TelegramConfig = TelegramConfig{
			Name:              "Telegram",
			VerificationToken: "testest",
		}
	}

	if c.d.Communications.SlackConfig.Name != "Slack" ||
		c.d.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		c.d.Communications.SMTPConfig.Name != "SMTP" ||
		c.d.Communications.TelegramConfig.Name != "Telegram" {
		log.Warnln(log.ConfigMgr, "Communications config name/s not set correctly")
	}
	if c.d.Communications.SlackConfig.Enabled {
		if c.d.Communications.SlackConfig.TargetChannel == "" ||
			c.d.Communications.SlackConfig.VerificationToken == "" ||
			c.d.Communications.SlackConfig.VerificationToken == "testtest" {
			c.d.Communications.SlackConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "Slack enabled in config but variable data not set, disabling.")
		}
	}
	if c.d.Communications.SMSGlobalConfig.Enabled {
		if c.d.Communications.SMSGlobalConfig.Username == "" ||
			c.d.Communications.SMSGlobalConfig.Password == "" ||
			len(c.d.Communications.SMSGlobalConfig.Contacts) == 0 {
			c.d.Communications.SMSGlobalConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "SMSGlobal enabled in config but variable data not set, disabling.")
		}
	}
	if c.d.Communications.SMTPConfig.Enabled {
		if c.d.Communications.SMTPConfig.Host == "" ||
			c.d.Communications.SMTPConfig.Port == "" ||
			c.d.Communications.SMTPConfig.AccountName == "" ||
			c.d.Communications.SMTPConfig.AccountPassword == "" {
			c.d.Communications.SMTPConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "SMTP enabled in config but variable data not set, disabling.")
		}
	}
	if c.d.Communications.TelegramConfig.Enabled {
		if c.d.Communications.TelegramConfig.VerificationToken == "" {
			c.d.Communications.TelegramConfig.Enabled = false
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

	return exchCfg.CurrencyPairs.GetAssetTypes(), nil
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

	if !exchCfg.CurrencyPairs.GetAssetTypes().Contains(assetType) {
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
	for i := range c.d.Exchanges {
		if c.d.Exchanges[i].Enabled {
			enabledExchs = append(enabledExchs, c.d.Exchanges[i].Name)
		}
	}
	return enabledExchs
}

// GetDisabledExchanges returns a list of disabled exchanges
func (c *Config) GetDisabledExchanges() []string {
	var disabledExchs []string
	for i := range c.d.Exchanges {
		if !c.d.Exchanges[i].Enabled {
			disabledExchs = append(disabledExchs, c.d.Exchanges[i].Name)
		}
	}
	return disabledExchs
}

// CountEnabledExchanges returns the number of exchanges that are enabled.
func (c *Config) CountEnabledExchanges() int {
	counter := 0
	for i := range c.d.Exchanges {
		if c.d.Exchanges[i].Enabled {
			counter++
		}
	}
	return counter
}

// GetCurrencyPairDisplayConfig retrieves the currency pair display preference
func (c *Config) GetCurrencyPairDisplayConfig() *CurrencyPairFormatConfig {
	return c.d.Currency.CurrencyPairFormat
}

// GetAllExchangeConfigs returns all exchange configurations
func (c *Config) GetAllExchangeConfigs() []ExchangeConfig {
	c.Lock()
	defer c.Unlock()
	return append(c.d.Exchanges[:0:0], c.d.Exchanges...)
}

// GetExchangeConfig returns exchange configurations by its indivdual name
func (c *Config) GetExchangeConfig(name string) (*ExchangeConfig, error) {
	c.Lock()
	defer c.Unlock()
	for i := range c.d.Exchanges {
		if strings.EqualFold(c.d.Exchanges[i].Name, name) {
			return &c.d.Exchanges[i], nil
		}
	}
	return nil, fmt.Errorf(ErrExchangeNotFound, name)
}

// GetForexProvider returns a forex provider configuration by its name
func (c *Config) GetForexProvider(name string) (currency.FXSettings, error) {
	c.Lock()
	defer c.Unlock()
	for i := range c.d.Currency.ForexProviders {
		if strings.EqualFold(c.d.Currency.ForexProviders[i].Name, name) {
			return c.d.Currency.ForexProviders[i], nil
		}
	}
	return currency.FXSettings{}, errors.New("provider not found")
}

// GetForexProviders returns a list of available forex providers
func (c *Config) GetForexProviders() []currency.FXSettings {
	c.Lock()
	fxProviders := c.d.Currency.ForexProviders
	c.Unlock()
	return fxProviders
}

// GetPrimaryForexProvider returns the primary forex provider
func (c *Config) GetPrimaryForexProvider() string {
	c.Lock()
	defer c.Unlock()
	for i := range c.d.Currency.ForexProviders {
		if c.d.Currency.ForexProviders[i].PrimaryProvider {
			return c.d.Currency.ForexProviders[i].Name
		}
	}
	return ""
}

// UpdateExchangeConfig updates exchange configurations
func (c *Config) UpdateExchangeConfig(e *ExchangeConfig) error {
	c.Lock()
	defer c.Unlock()
	for i := range c.d.Exchanges {
		if strings.EqualFold(c.d.Exchanges[i].Name, e.Name) {
			c.d.Exchanges[i] = *e
			return nil
		}
	}
	return fmt.Errorf(ErrExchangeNotFound, e.Name)
}

// CheckExchangeConfigValues returns configuation values for all enabled
// exchanges
func (c *Config) CheckExchangeConfigValues() error {
	if len(c.d.Exchanges) == 0 {
		return errors.New("no exchange configs found")
	}

	exchanges := 0
	for i := range c.d.Exchanges {
		if strings.EqualFold(c.d.Exchanges[i].Name, "GDAX") {
			c.d.Exchanges[i].Name = "CoinbasePro"
		}

		// Check to see if the old API storage format is used
		if c.d.Exchanges[i].APIKey != nil {
			// It is, migrate settings to new format
			c.d.Exchanges[i].API.AuthenticatedSupport = *c.d.Exchanges[i].AuthenticatedAPISupport
			if c.d.Exchanges[i].AuthenticatedWebsocketAPISupport != nil {
				c.d.Exchanges[i].API.AuthenticatedWebsocketSupport = *c.d.Exchanges[i].AuthenticatedWebsocketAPISupport
			}
			c.d.Exchanges[i].API.Credentials.Key = *c.d.Exchanges[i].APIKey
			c.d.Exchanges[i].API.Credentials.Secret = *c.d.Exchanges[i].APISecret

			if c.d.Exchanges[i].APIAuthPEMKey != nil {
				c.d.Exchanges[i].API.Credentials.PEMKey = *c.d.Exchanges[i].APIAuthPEMKey
			}

			if c.d.Exchanges[i].APIAuthPEMKeySupport != nil {
				c.d.Exchanges[i].API.PEMKeySupport = *c.d.Exchanges[i].APIAuthPEMKeySupport
			}

			if c.d.Exchanges[i].ClientID != nil {
				c.d.Exchanges[i].API.Credentials.ClientID = *c.d.Exchanges[i].ClientID
			}

			if c.d.Exchanges[i].WebsocketURL != nil {
				c.d.Exchanges[i].API.Endpoints.WebsocketURL = *c.d.Exchanges[i].WebsocketURL
			}

			c.d.Exchanges[i].API.Endpoints.URL = *c.d.Exchanges[i].APIURL
			c.d.Exchanges[i].API.Endpoints.URLSecondary = *c.d.Exchanges[i].APIURLSecondary

			// Flush settings
			c.d.Exchanges[i].AuthenticatedAPISupport = nil
			c.d.Exchanges[i].AuthenticatedWebsocketAPISupport = nil
			c.d.Exchanges[i].APIKey = nil
			c.d.Exchanges[i].APISecret = nil
			c.d.Exchanges[i].ClientID = nil
			c.d.Exchanges[i].APIAuthPEMKeySupport = nil
			c.d.Exchanges[i].APIAuthPEMKey = nil
			c.d.Exchanges[i].APIURL = nil
			c.d.Exchanges[i].APIURLSecondary = nil
			c.d.Exchanges[i].WebsocketURL = nil
		}

		if c.d.Exchanges[i].Features == nil {
			c.d.Exchanges[i].Features = &FeaturesConfig{}
		}

		if c.d.Exchanges[i].SupportsAutoPairUpdates != nil {
			c.d.Exchanges[i].Features.Supports.RESTCapabilities.AutoPairUpdates = *c.d.Exchanges[i].SupportsAutoPairUpdates
			c.d.Exchanges[i].Features.Enabled.AutoPairUpdates = *c.d.Exchanges[i].SupportsAutoPairUpdates
			c.d.Exchanges[i].SupportsAutoPairUpdates = nil
		}

		if c.d.Exchanges[i].Websocket != nil {
			c.d.Exchanges[i].Features.Enabled.Websocket = *c.d.Exchanges[i].Websocket
			c.d.Exchanges[i].Websocket = nil
		}

		if c.d.Exchanges[i].API.Endpoints.URL != APIURLNonDefaultMessage {
			if c.d.Exchanges[i].API.Endpoints.URL == "" {
				// Set default if nothing set
				c.d.Exchanges[i].API.Endpoints.URL = APIURLNonDefaultMessage
			}
		}

		if c.d.Exchanges[i].API.Endpoints.URLSecondary != APIURLNonDefaultMessage {
			if c.d.Exchanges[i].API.Endpoints.URLSecondary == "" {
				// Set default if nothing set
				c.d.Exchanges[i].API.Endpoints.URLSecondary = APIURLNonDefaultMessage
			}
		}

		if c.d.Exchanges[i].API.Endpoints.WebsocketURL != WebsocketURLNonDefaultMessage {
			if c.d.Exchanges[i].API.Endpoints.WebsocketURL == "" {
				c.d.Exchanges[i].API.Endpoints.WebsocketURL = WebsocketURLNonDefaultMessage
			}
		}

		// Check if see if the new currency pairs format is empty and flesh it out if so
		if c.d.Exchanges[i].CurrencyPairs == nil {
			c.d.Exchanges[i].CurrencyPairs = new(currency.PairsManager)
			c.d.Exchanges[i].CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)

			if c.d.Exchanges[i].PairsLastUpdated != nil {
				c.d.Exchanges[i].CurrencyPairs.LastUpdated = *c.d.Exchanges[i].PairsLastUpdated
			}

			c.d.Exchanges[i].CurrencyPairs.ConfigFormat = c.d.Exchanges[i].ConfigCurrencyPairFormat
			c.d.Exchanges[i].CurrencyPairs.RequestFormat = c.d.Exchanges[i].RequestCurrencyPairFormat

			var availPairs, enabledPairs currency.Pairs
			if c.d.Exchanges[i].AvailablePairs != nil {
				availPairs = *c.d.Exchanges[i].AvailablePairs
			}

			if c.d.Exchanges[i].EnabledPairs != nil {
				enabledPairs = *c.d.Exchanges[i].EnabledPairs
			}

			c.d.Exchanges[i].CurrencyPairs.UseGlobalFormat = true
			c.d.Exchanges[i].CurrencyPairs.Store(asset.Spot,
				currency.PairStore{
					AssetEnabled: convert.BoolPtr(true),
					Available:    availPairs,
					Enabled:      enabledPairs,
				},
			)

			// flush old values
			c.d.Exchanges[i].PairsLastUpdated = nil
			c.d.Exchanges[i].ConfigCurrencyPairFormat = nil
			c.d.Exchanges[i].RequestCurrencyPairFormat = nil
			c.d.Exchanges[i].AssetTypes = nil
			c.d.Exchanges[i].AvailablePairs = nil
			c.d.Exchanges[i].EnabledPairs = nil
		} else {
			assets := c.d.Exchanges[i].CurrencyPairs.GetAssetTypes()
			var atLeastOne bool
			for index := range assets {
				err := c.d.Exchanges[i].CurrencyPairs.IsAssetEnabled(assets[index])
				if err != nil {
					// Checks if we have an old config without the ability to
					// enable disable the entire asset
					if err.Error() == "cannot ascertain if asset is enabled, variable is nil" {
						log.Warnf(log.ConfigMgr,
							"Exchange %s: upgrading config for asset type %s and setting enabled.\n",
							c.d.Exchanges[i].Name,
							assets[index])
						err = c.d.Exchanges[i].CurrencyPairs.SetAssetEnabled(assets[index], true)
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
					c.d.Exchanges[i].Enabled = false
					log.Warnf(log.ConfigMgr,
						"%s no assets found, disabling...",
						c.d.Exchanges[i].Name)
					continue
				}

				// turn on an asset if all disabled
				log.Warnf(log.ConfigMgr,
					"%s assets disabled, turning on asset %s",
					c.d.Exchanges[i].Name,
					assets[0])

				err := c.d.Exchanges[i].CurrencyPairs.SetAssetEnabled(assets[0], true)
				if err != nil {
					return err
				}
			}
		}

		if c.d.Exchanges[i].Enabled {
			if c.d.Exchanges[i].Name == "" {
				log.Errorf(log.ConfigMgr, ErrExchangeNameEmpty, i)
				c.d.Exchanges[i].Enabled = false
				continue
			}
			if (c.d.Exchanges[i].API.AuthenticatedSupport || c.d.Exchanges[i].API.AuthenticatedWebsocketSupport) &&
				c.d.Exchanges[i].API.CredentialsValidator != nil {
				var failed bool
				if c.d.Exchanges[i].API.CredentialsValidator.RequiresKey &&
					(c.d.Exchanges[i].API.Credentials.Key == "" || c.d.Exchanges[i].API.Credentials.Key == DefaultAPIKey) {
					failed = true
				}

				if c.d.Exchanges[i].API.CredentialsValidator.RequiresSecret &&
					(c.d.Exchanges[i].API.Credentials.Secret == "" || c.d.Exchanges[i].API.Credentials.Secret == DefaultAPISecret) {
					failed = true
				}

				if c.d.Exchanges[i].API.CredentialsValidator.RequiresClientID &&
					(c.d.Exchanges[i].API.Credentials.ClientID == DefaultAPIClientID || c.d.Exchanges[i].API.Credentials.ClientID == "") {
					failed = true
				}

				if failed {
					c.d.Exchanges[i].API.AuthenticatedSupport = false
					c.d.Exchanges[i].API.AuthenticatedWebsocketSupport = false
					log.Warnf(log.ConfigMgr, WarningExchangeAuthAPIDefaultOrEmptyValues, c.d.Exchanges[i].Name)
				}
			}
			if !c.d.Exchanges[i].Features.Supports.RESTCapabilities.AutoPairUpdates &&
				!c.d.Exchanges[i].Features.Supports.WebsocketCapabilities.AutoPairUpdates {
				lastUpdated := convert.UnixTimestampToTime(c.d.Exchanges[i].CurrencyPairs.LastUpdated)
				lastUpdated = lastUpdated.AddDate(0, 0, pairsLastUpdatedWarningThreshold)
				if lastUpdated.Unix() <= time.Now().Unix() {
					log.Warnf(log.ConfigMgr,
						WarningPairsLastUpdatedThresholdExceeded,
						c.d.Exchanges[i].Name,
						pairsLastUpdatedWarningThreshold)
				}
			}
			if c.d.Exchanges[i].HTTPTimeout <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s HTTP Timeout value not set, defaulting to %v.\n",
					c.d.Exchanges[i].Name,
					defaultHTTPTimeout)
				c.d.Exchanges[i].HTTPTimeout = defaultHTTPTimeout
			}

			if c.d.Exchanges[i].WebsocketResponseCheckTimeout <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket response check timeout value not set, defaulting to %v.",
					c.d.Exchanges[i].Name,
					defaultWebsocketResponseCheckTimeout)
				c.d.Exchanges[i].WebsocketResponseCheckTimeout = defaultWebsocketResponseCheckTimeout
			}

			if c.d.Exchanges[i].WebsocketResponseMaxLimit <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket response max limit value not set, defaulting to %v.",
					c.d.Exchanges[i].Name,
					defaultWebsocketResponseMaxLimit)
				c.d.Exchanges[i].WebsocketResponseMaxLimit = defaultWebsocketResponseMaxLimit
			}
			if c.d.Exchanges[i].WebsocketTrafficTimeout <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket response traffic timeout value not set, defaulting to %v.",
					c.d.Exchanges[i].Name,
					defaultWebsocketTrafficTimeout)
				c.d.Exchanges[i].WebsocketTrafficTimeout = defaultWebsocketTrafficTimeout
			}
			if c.d.Exchanges[i].OrderbookConfig.WebsocketBufferLimit <= 0 {
				log.Warnf(log.ConfigMgr,
					"Exchange %s Websocket orderbook buffer limit value not set, defaulting to %v.",
					c.d.Exchanges[i].Name,
					defaultWebsocketOrderbookBufferLimit)
				c.d.Exchanges[i].OrderbookConfig.WebsocketBufferLimit = defaultWebsocketOrderbookBufferLimit
			}
			err := c.CheckPairConsistency(c.d.Exchanges[i].Name)
			if err != nil {
				log.Errorf(log.ConfigMgr,
					"Exchange %s: CheckPairConsistency error: %s\n",
					c.d.Exchanges[i].Name,
					err)
				c.d.Exchanges[i].Enabled = false
				continue
			}
			for x := range c.d.Exchanges[i].BankAccounts {
				if !c.d.Exchanges[i].BankAccounts[x].Enabled {
					continue
				}
				err := c.d.Exchanges[i].BankAccounts[x].Validate()
				if err != nil {
					c.d.Exchanges[i].BankAccounts[x].Enabled = false
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
	for x := range c.d.BankAccounts {
		if c.d.BankAccounts[x].Enabled {
			err := c.d.BankAccounts[x].Validate()
			if err != nil {
				c.d.BankAccounts[x].Enabled = false
				log.Warn(log.ConfigMgr, err.Error())
			}
		}
	}
	banking.Accounts = c.d.BankAccounts
}

// CheckCurrencyConfigValues checks to see if the currency config values are correct or not
func (c *Config) CheckCurrencyConfigValues() error {
	fxProviders := forexprovider.GetSupportedForexProviders()

	if len(fxProviders) != len(c.d.Currency.ForexProviders) {
		for x := range fxProviders {
			_, err := c.GetForexProvider(fxProviders[x])
			if err != nil {
				log.Warnf(log.Global, "%s forex provider not found, adding to config..\n", fxProviders[x])
				c.d.Currency.ForexProviders = append(c.d.Currency.ForexProviders, currency.FXSettings{
					Name:             fxProviders[x],
					RESTPollingDelay: 600,
					APIKey:           DefaultUnsetAPIKey,
					APIKeyLvl:        -1,
				})
			}
		}
	}

	count := 0
	for i := range c.d.Currency.ForexProviders {
		if c.d.Currency.ForexProviders[i].Enabled {
			if c.d.Currency.ForexProviders[i].Name == "CurrencyConverter" &&
				c.d.Currency.ForexProviders[i].PrimaryProvider &&
				(c.d.Currency.ForexProviders[i].APIKey == "" ||
					c.d.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey) {
				log.Warnln(log.Global, "CurrencyConverter forex provider no longer supports unset API key requests. Switching to ExchangeRates FX provider..")
				c.d.Currency.ForexProviders[i].Enabled = false
				c.d.Currency.ForexProviders[i].PrimaryProvider = false
				c.d.Currency.ForexProviders[i].APIKey = DefaultUnsetAPIKey
				c.d.Currency.ForexProviders[i].APIKeyLvl = -1
				continue
			}
			if c.d.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey &&
				c.d.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf(log.Global, "%s enabled forex provider API key not set. Please set this in your config.json file\n", c.d.Currency.ForexProviders[i].Name)
				c.d.Currency.ForexProviders[i].Enabled = false
				c.d.Currency.ForexProviders[i].PrimaryProvider = false
				continue
			}

			if c.d.Currency.ForexProviders[i].APIKeyLvl == -1 && c.d.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf(log.Global, "%s APIKey Level not set, functions limited. Please set this in your config.json file\n",
					c.d.Currency.ForexProviders[i].Name)
			}
			count++
		}
	}

	if count == 0 {
		for x := range c.d.Currency.ForexProviders {
			if c.d.Currency.ForexProviders[x].Name == DefaultForexProviderExchangeRatesAPI {
				c.d.Currency.ForexProviders[x].Enabled = true
				c.d.Currency.ForexProviders[x].PrimaryProvider = true
				log.Warnln(log.ConfigMgr, "Using ExchangeRatesAPI for default forex provider.")
			}
		}
	}

	if c.d.Currency.CryptocurrencyProvider == (CryptocurrencyProvider{}) {
		c.d.Currency.CryptocurrencyProvider.Name = "CoinMarketCap"
		c.d.Currency.CryptocurrencyProvider.Enabled = false
		c.d.Currency.CryptocurrencyProvider.Verbose = false
		c.d.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
		c.d.Currency.CryptocurrencyProvider.APIkey = DefaultUnsetAPIKey
	}

	if c.d.Currency.CryptocurrencyProvider.Enabled {
		if c.d.Currency.CryptocurrencyProvider.APIkey == "" ||
			c.d.Currency.CryptocurrencyProvider.APIkey == DefaultUnsetAPIKey {
			log.Warnln(log.ConfigMgr, "CryptocurrencyProvider enabled but api key is unset please set this in your config.json file")
		}
		if c.d.Currency.CryptocurrencyProvider.AccountPlan == "" ||
			c.d.Currency.CryptocurrencyProvider.AccountPlan == DefaultUnsetAccountPlan {
			log.Warnln(log.ConfigMgr, "CryptocurrencyProvider enabled but account plan is unset please set this in your config.json file")
		}
	} else {
		if c.d.Currency.CryptocurrencyProvider.APIkey == "" {
			c.d.Currency.CryptocurrencyProvider.APIkey = DefaultUnsetAPIKey
		}
		if c.d.Currency.CryptocurrencyProvider.AccountPlan == "" {
			c.d.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
		}
	}

	if c.d.Currency.Cryptocurrencies.Join() == "" {
		if c.d.Cryptocurrencies != nil {
			c.d.Currency.Cryptocurrencies = *c.d.Cryptocurrencies
			c.d.Cryptocurrencies = nil
		} else {
			c.d.Currency.Cryptocurrencies = currency.GetDefaultCryptocurrencies()
		}
	}

	if c.d.Currency.CurrencyPairFormat == nil {
		if c.d.CurrencyPairFormat != nil {
			c.d.Currency.CurrencyPairFormat = c.d.CurrencyPairFormat
			c.d.CurrencyPairFormat = nil
		} else {
			c.d.Currency.CurrencyPairFormat = &CurrencyPairFormatConfig{
				Delimiter: "-",
				Uppercase: true,
			}
		}
	}

	if c.d.Currency.FiatDisplayCurrency.IsEmpty() {
		if c.d.FiatDisplayCurrency != nil {
			c.d.Currency.FiatDisplayCurrency = *c.d.FiatDisplayCurrency
			c.d.FiatDisplayCurrency = nil
		} else {
			c.d.Currency.FiatDisplayCurrency = currency.USD
		}
	}

	// Flush old setting which still exists
	if c.d.FiatDisplayCurrency != nil {
		c.d.FiatDisplayCurrency = nil
	}

	return nil
}

// RetrieveConfigCurrencyPairs splits, assigns and verifies enabled currency
// pairs either cryptoCurrencies or fiatCurrencies
func (c *Config) RetrieveConfigCurrencyPairs(enabledOnly bool, assetType asset.Item) error {
	cryptoCurrencies := c.d.Currency.Cryptocurrencies
	fiatCurrencies := currency.GetFiatCurrencies()

	for x := range c.d.Exchanges {
		if !c.d.Exchanges[x].Enabled && enabledOnly {
			continue
		}

		err := c.SupportsExchangeAssetType(c.d.Exchanges[x].Name, assetType)
		if err != nil {
			continue
		}

		baseCurrencies := c.d.Exchanges[x].BaseCurrencies
		for y := range baseCurrencies {
			if !fiatCurrencies.Contains(baseCurrencies[y]) {
				fiatCurrencies = append(fiatCurrencies, baseCurrencies[y])
			}
		}
	}

	for x := range c.d.Exchanges {
		err := c.SupportsExchangeAssetType(c.d.Exchanges[x].Name, assetType)
		if err != nil {
			continue
		}

		var pairs []currency.Pair
		if !c.d.Exchanges[x].Enabled && enabledOnly {
			pairs, err = c.GetEnabledPairs(c.d.Exchanges[x].Name, assetType)
		} else {
			pairs, err = c.GetAvailablePairs(c.d.Exchanges[x].Name, assetType)
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
	c.Lock()
	defer c.Unlock()

	if c.d.Logging.Enabled == nil || c.d.Logging.Output == "" {
		c.d.Logging = log.GenDefaultSettings()
	}

	if c.d.Logging.AdvancedSettings.ShowLogSystemName == nil {
		c.d.Logging.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(false)
	}

	if c.d.Logging.LoggerFileConfig != nil {
		if c.d.Logging.LoggerFileConfig.FileName == "" {
			c.d.Logging.LoggerFileConfig.FileName = "log.txt"
		}
		if c.d.Logging.LoggerFileConfig.Rotate == nil {
			c.d.Logging.LoggerFileConfig.Rotate = convert.BoolPtr(false)
		}
		if c.d.Logging.LoggerFileConfig.MaxSize <= 0 {
			log.Warnf(log.Global, "Logger rotation size invalid, defaulting to %v", log.DefaultMaxFileSize)
			c.d.Logging.LoggerFileConfig.MaxSize = log.DefaultMaxFileSize
		}
		log.FileLoggingConfiguredCorrectly = true
	}
	log.RWM.Lock()
	log.GlobalLogConfig = &c.d.Logging
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
	c.Lock()
	defer c.Unlock()

	if c.d.GCTScript.ScriptTimeout <= 0 {
		c.d.GCTScript.ScriptTimeout = gctscript.DefaultTimeoutValue
	}

	if c.d.GCTScript.MaxVirtualMachines == 0 {
		c.d.GCTScript.MaxVirtualMachines = gctscript.DefaultMaxVirtualMachines
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
	c.Lock()
	defer c.Unlock()

	if (c.d.Database == database.Config{}) {
		c.d.Database.Driver = database.DBSQLite3
		c.d.Database.Database = database.DefaultSQLiteDatabase
	}

	if !c.d.Database.Enabled {
		return nil
	}

	if !common.StringDataCompare(database.SupportedDrivers, c.d.Database.Driver) {
		c.d.Database.Enabled = false
		return fmt.Errorf("unsupported database driver %v, database disabled", c.d.Database.Driver)
	}

	if c.d.Database.Driver == database.DBSQLite || c.d.Database.Driver == database.DBSQLite3 {
		databaseDir := c.GetDataPath("database")
		err := common.CreateDir(databaseDir)
		if err != nil {
			return err
		}
		database.DB.DataPath = databaseDir
	}

	database.DB.Config = &c.d.Database

	return nil
}

// CheckNTPConfig checks for missing or incorrectly configured NTPClient and recreates with known safe defaults
func (c *Config) CheckNTPConfig() {
	c.Lock()
	defer c.Unlock()

	if c.d.NTPClient.AllowedDifference == nil || *c.d.NTPClient.AllowedDifference == 0 {
		c.d.NTPClient.AllowedDifference = new(time.Duration)
		*c.d.NTPClient.AllowedDifference = defaultNTPAllowedDifference
	}

	if c.d.NTPClient.AllowedNegativeDifference == nil || *c.d.NTPClient.AllowedNegativeDifference <= 0 {
		c.d.NTPClient.AllowedNegativeDifference = new(time.Duration)
		*c.d.NTPClient.AllowedNegativeDifference = defaultNTPAllowedNegativeDifference
	}

	if len(c.d.NTPClient.Pool) < 1 {
		log.Warnln(log.ConfigMgr, "NTPClient enabled with no servers configured, enabling default pool.")
		c.d.NTPClient.Pool = []string{"pool.ntp.org:123"}
	}
}

// DisableNTPCheck allows the user to change how they are prompted for timesync alerts
func (c *Config) DisableNTPCheck(input io.Reader) (string, error) {
	c.Lock()
	defer c.Unlock()

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
			c.d.NTPClient.Level = 0
			resp = "Time sync has been set to alert"
			answered = true
		case "w":
			c.d.NTPClient.Level = 1
			resp = "Time sync has been set to warn only"
			answered = true
		case "d":
			c.d.NTPClient.Level = -1
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
	c.Lock()
	defer c.Unlock()

	if c.d.ConnectionMonitor.CheckInterval == 0 {
		c.d.ConnectionMonitor.CheckInterval = connchecker.DefaultCheckInterval
	}

	if len(c.d.ConnectionMonitor.DNSList) == 0 {
		c.d.ConnectionMonitor.DNSList = connchecker.DefaultDNSList
	}

	if len(c.d.ConnectionMonitor.PublicDomainList) == 0 {
		c.d.ConnectionMonitor.PublicDomainList = connchecker.DefaultDomainList
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
func GetFilePath(configfile string) (configPath string, isImplicitDefaultPath bool, err error) {
	if configfile != "" {
		return configfile, false, nil
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
			configfile = p
			break
		}
	}
	if configfile == "" {
		return "", false, fmt.Errorf("config.json file not found in %s, please follow README.md in root dir for config generation",
			newDir)
	}

	return configfile, true, nil
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
	wasEncrypted, err := c.ReadConfig(confFile, func() ([]byte, error) { return PromptForConfigKey(false) })
	if err != nil {
		return fmt.Errorf("error reading config %w", err)
	}

	if dryrun || wasEncrypted || c.d.EncryptConfig == fileEncryptionDisabled {
		return nil
	}

	if c.d.EncryptConfig == fileEncryptionPrompt {
		confirm, err := promptForConfigEncryption()
		if err != nil {
			log.Errorf(log.ConfigMgr, "The encryption prompt failed, ignoring for now, next time we will prompt again. Error: %s\n", err)
			return nil
		}
		if confirm {
			c.d.EncryptConfig = fileEncryptionEnabled
			return c.SaveConfigToFile(defaultPath)
		}

		c.d.EncryptConfig = fileEncryptionDisabled
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
func (c *Config) ReadConfig(configReader io.Reader, keyProvider func() ([]byte, error)) (bool, error) {
	reader := bufio.NewReader(configReader)
	pref, err := reader.Peek(len(EncryptConfirmString))
	if err != nil {
		return false, err
	}

	if !ConfirmECS(pref) {
		// Read unencrypted configuration
		decoder := json.NewDecoder(reader)
		err = decoder.Decode(&c.d)
		return false, err
	}

	err = c.readEncryptedConfWithKey(reader, keyProvider)
	return true, err
}

// readEncryptedConf reads encrypted configuration and requests key from provider
func (c *Config) readEncryptedConfWithKey(reader *bufio.Reader, keyProvider func() ([]byte, error)) error {
	fileData, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	for errCounter := 0; errCounter < maxAuthFailures; errCounter++ {
		key, err := keyProvider()
		if err != nil {
			log.Errorf(log.ConfigMgr, "PromptForConfigKey err: %s", err)
			continue
		}

		err = c.readEncryptedConf(bytes.NewReader(fileData), key)
		if err != nil {
			log.Error(log.ConfigMgr, "Could not decrypt and deserialise data with given key. Invalid password?", err)
			continue
		}
		return nil
	}
	return errors.New("failed to decrypt config after 3 attempts")
}

func (c *Config) readEncryptedConf(reader io.Reader, key []byte) error {
	data, err := c.decryptConfigData(reader, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &c.d)
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
			writer.Close()
		}
	}()
	return c.Save(provider, func() ([]byte, error) { return PromptForConfigKey(true) })
}

// Save saves your configuration to the writer as a JSON object
// with encryption, if configured
// If there is an error when preparing the data to store, the writer is never requested
func (c *Config) Save(writerProvider func() (io.Writer, error), keyProvider func() ([]byte, error)) error {
	payload, err := json.MarshalIndent(&c.d, "", " ")
	if err != nil {
		return err
	}

	if c.d.EncryptConfig == fileEncryptionEnabled {
		// Ensure we have the key from session or from user
		if len(c.d.sessionDK) == 0 {
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
			c.d.sessionDK, c.d.storedSalt = sessionDK, storedSalt
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

// CheckRemoteControlConfig checks to see if the old c.d.Webserver field is used
// and migrates the existing settings to the new RemoteControl struct
func (c *Config) CheckRemoteControlConfig() {
	c.Lock()
	defer c.Unlock()

	if c.d.Webserver != nil {
		port := common.ExtractPort(c.d.Webserver.ListenAddress)
		host := common.ExtractHost(c.d.Webserver.ListenAddress)

		c.d.RemoteControl = RemoteControlConfig{
			Username: c.d.Webserver.AdminUsername,
			Password: c.d.Webserver.AdminPassword,

			DeprecatedRPC: DepcrecatedRPCConfig{
				Enabled:       c.d.Webserver.Enabled,
				ListenAddress: host + ":" + strconv.Itoa(port),
			},
		}

		port++
		c.d.RemoteControl.WebsocketRPC = WebsocketRPCConfig{
			Enabled:             c.d.Webserver.Enabled,
			ListenAddress:       host + ":" + strconv.Itoa(port),
			ConnectionLimit:     c.d.Webserver.WebsocketConnectionLimit,
			MaxAuthFailures:     c.d.Webserver.WebsocketMaxAuthFailures,
			AllowInsecureOrigin: c.d.Webserver.WebsocketAllowInsecureOrigin,
		}

		port++
		gRPCProxyPort := port + 1
		c.d.RemoteControl.GRPC = GRPCConfig{
			Enabled:                c.d.Webserver.Enabled,
			ListenAddress:          host + ":" + strconv.Itoa(port),
			GRPCProxyEnabled:       c.d.Webserver.Enabled,
			GRPCProxyListenAddress: host + ":" + strconv.Itoa(gRPCProxyPort),
		}

		// Then flush the old webserver settings
		c.d.Webserver = nil
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
	c.CheckCommunicationsConfig()
	c.CheckClientBankAccounts()
	c.CheckBankAccountConfig()
	c.CheckRemoteControlConfig()

	err = c.CheckCurrencyConfigValues()
	if err != nil {
		return err
	}

	if c.d.GlobalHTTPTimeout <= 0 {
		log.Warnf(log.ConfigMgr,
			"Global HTTP Timeout value not set, defaulting to %v.\n",
			defaultHTTPTimeout)
		c.d.GlobalHTTPTimeout = defaultHTTPTimeout
	}

	if c.d.NTPClient.Level != 0 {
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

	c.d.Name = newCfg.d.Name
	c.d.EncryptConfig = newCfg.d.EncryptConfig
	c.d.Currency = newCfg.d.Currency
	c.d.GlobalHTTPTimeout = newCfg.d.GlobalHTTPTimeout
	c.d.Portfolio = newCfg.d.Portfolio
	c.d.Communications = newCfg.d.Communications
	c.d.Webserver = newCfg.d.Webserver
	c.d.Exchanges = newCfg.d.Exchanges

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
	return config
}

// RemoveExchange removes an exchange config
func (c *Config) RemoveExchange(exchName string) bool {
	c.Lock()
	defer c.Unlock()
	for x := range c.d.Exchanges {
		if strings.EqualFold(c.d.Exchanges[x].Name, exchName) {
			c.d.Exchanges = append(c.d.Exchanges[:x], c.d.Exchanges[x+1:]...)
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
	if c.d.DataDirectory != "" {
		baseDir = c.d.DataDirectory
	} else {
		baseDir = common.GetDefaultDataDir(runtime.GOOS)
	}
	return filepath.Join(append([]string{baseDir}, elem...)...)
}

// GetRemoteControl returns remote control configuration
func (c *Config) GetRemoteControl() RemoteControlConfig {
	c.Lock()
	defer c.Unlock()
	return c.d.RemoteControl
}

// GetNTPClient returns ntp client configuration
func (c *Config) GetNTPClient() NTPClientConfig {
	c.Lock()
	defer c.Unlock()
	return c.d.NTPClient
}

// GetLogging returns logger configuration
func (c *Config) GetLogging() log.Config {
	c.Lock()
	defer c.Unlock()
	return c.d.Logging
}

// GetCurrency returns currency configuration
func (c *Config) GetCurrency() CurrencyConfig {
	c.Lock()
	defer c.Unlock()
	return c.d.Currency
}

// GetDatabase returns currency configuration
func (c *Config) GetDatabase() database.Config {
	c.Lock()
	defer c.Unlock()
	return c.d.Database
}

// SetDatabase sets database configuration
func (c *Config) SetDatabase(d database.Config) {
	c.Lock()
	c.d.Database = d
	c.Unlock()
}

// SetProfiler sets profiler configuration
func (c *Config) SetProfiler(p Profiler) {
	c.Lock()
	c.d.Profiler = p
	c.Unlock()
}

// GetProfiler gets profiler configuration
func (c *Config) GetProfiler() Profiler {
	c.Lock()
	defer c.Unlock()
	return c.d.Profiler
}

// GetPortfolio gets portfolio configuration
func (c *Config) GetPortfolio() portfolio.Base {
	c.Lock()
	defer c.Unlock()
	return c.d.Portfolio
}

// SetPortfolio sets portfolio configuration
func (c *Config) SetPortfolio(p portfolio.Base) {
	c.Lock()
	c.d.Portfolio = p
	c.Unlock()
}

// GetConnectionMonitor gets connection monitor configuration
func (c *Config) GetConnectionMonitor() ConnectionMonitorConfig {
	c.Lock()
	defer c.Unlock()
	return c.d.ConnectionMonitor
}

// AddExchangeConfig adds a new exchange config to list
func (c *Config) AddExchangeConfig(exch ExchangeConfig) {
	c.Lock()
	c.d.Exchanges = append(c.d.Exchanges, exch)
	c.Unlock()
}

// GetGCTScript gets connection monitor configuration
func (c *Config) GetGCTScript() gctscript.Config {
	c.Lock()
	defer c.Unlock()
	return c.d.GCTScript
}

// GetDataDirectory returns data directory string
func (c *Config) GetDataDirectory() string {
	c.Lock()
	defer c.Unlock()
	return c.d.DataDirectory
}

// SetDataDirectory sets new path for data directory
func (c *Config) SetDataDirectory(newDir string) {
	c.Lock()
	c.d.DataDirectory = newDir
	c.Unlock()
}

// GetGlobalHTTPTimeout gets timeout value
func (c *Config) GetGlobalHTTPTimeout() time.Duration {
	c.Lock()
	defer c.Unlock()
	return c.d.GlobalHTTPTimeout
}

// GetName returns name of configuration
func (c *Config) GetName() string {
	c.Lock()
	defer c.Unlock()
	return c.d.Name
}
