package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-/gocryptotrader/currency/pair"
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
	configDefaultHTTPTimeout               = time.Duration(time.Second * 15)
	configMaxAuthFailres                   = 3
)

// Constants here hold some messages
const (
	ErrExchangeNameEmpty                            = "Exchange #%d in config: Exchange name is empty."
	ErrExchangeAvailablePairsEmpty                  = "Exchange %s: Available pairs is empty."
	ErrExchangeEnabledPairsEmpty                    = "Exchange %s: Enabled pairs is empty."
	ErrExchangeBaseCurrenciesEmpty                  = "Exchange %s: Base currencies is empty."
	ErrExchangeNotFound                             = "Exchange %s: Not found."
	ErrNoEnabledExchanges                           = "No Exchanges enabled."
	ErrCryptocurrenciesEmpty                        = "Cryptocurrencies variable is empty."
	ErrFailureOpeningConfig                         = "Fatal error opening %s file. Error: %s"
	ErrCheckingConfigValues                         = "Fatal error checking config values. Error: %s"
	ErrSavingConfigBytesMismatch                    = "Config file %q bytes comparison doesn't match, read %s expected %s."
	WarningSMSGlobalDefaultOrEmptyValues            = "WARNING -- SMS Support disabled due to default or empty Username/Password values."
	WarningSSMSGlobalSMSContactDefaultOrEmptyValues = "WARNING -- SMS contact #%d Name/Number disabled due to default or empty values."
	WarningSSMSGlobalSMSNoContacts                  = "WARNING -- SMS Support disabled due to no enabled contacts."
	WarningWebserverCredentialValuesEmpty           = "WARNING -- Webserver support disabled due to empty Username/Password values."
	WarningWebserverListenAddressInvalid            = "WARNING -- Webserver support disabled due to invalid listen address."
	WarningWebserverRootWebFolderNotFound           = "WARNING -- Webserver support disabled due to missing web folder."
	WarningExchangeAuthAPIDefaultOrEmptyValues      = "WARNING -- Exchange %s: Authenticated API support disabled due to default/empty APIKey/Secret/ClientID values."
	WarningCurrencyExchangeProvider                 = "WARNING -- Currency exchange provider invalid valid. Reset to Fixer."
	WarningPairsLastUpdatedThresholdExceeded        = "WARNING -- Exchange %s: Last manual update of available currency pairs has exceeded %d days. Manual update required!"
	APIURLNonDefaultMessage                         = "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"
	WebsocketURLNonDefaultMessage                   = "NON_DEFAULT_HTTP_LINK_TO_WEBSOCKET_EXCHANGE_API"
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
func (c *Config) GetExchangeBankAccounts(exchangeName string, depositingCurrency string) (BankAccount, error) {
	m.Lock()
	defer m.Unlock()

	for _, exch := range c.Exchanges {
		if exch.Name == exchangeName {
			for _, account := range exch.BankAccounts {
				if common.StringContains(account.SupportedCurrencies, depositingCurrency) {
					return account, nil
				}
			}
		}
	}
	return BankAccount{}, fmt.Errorf("Exchange %s bank details not found for %s",
		exchangeName,
		depositingCurrency)
}

// UpdateExchangeBankAccounts updates the configuration for the associated
// exchange bank
func (c *Config) UpdateExchangeBankAccounts(exchangeName string, bankCfg []BankAccount) error {
	m.Lock()
	defer m.Unlock()

	for i := range c.Exchanges {
		if c.Exchanges[i].Name == exchangeName {
			c.Exchanges[i].BankAccounts = bankCfg
			return nil
		}
	}
	return fmt.Errorf("UpdateExchangeBankAccounts() error exchange %s not found",
		exchangeName)
}

// GetClientBankAccounts returns banking details used for a given exchange
// and currency
func (c *Config) GetClientBankAccounts(exchangeName string, targetCurrency string) (BankAccount, error) {
	m.Lock()
	defer m.Unlock()

	for _, bank := range c.BankAccounts {
		if (common.StringContains(bank.SupportedExchanges, exchangeName) || bank.SupportedExchanges == "ALL") && common.StringContains(bank.SupportedCurrencies, targetCurrency) {
			return bank, nil

		}
	}
	return BankAccount{}, fmt.Errorf("client banking details not found for %s and currency %s",
		exchangeName,
		targetCurrency)
}

// UpdateClientBankAccounts updates the configuration for a bank
func (c *Config) UpdateClientBankAccounts(bankCfg BankAccount) error {
	m.Lock()
	defer m.Unlock()

	for i := range c.BankAccounts {
		if c.BankAccounts[i].BankName == bankCfg.BankName && c.BankAccounts[i].AccountNumber == bankCfg.AccountNumber {
			c.BankAccounts[i] = bankCfg
			return nil
		}
	}
	return fmt.Errorf("client banking details for %s not found, update not applied",
		bankCfg.BankName)
}

// CheckClientBankAccounts checks client bank details
func (c *Config) CheckClientBankAccounts() error {
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
		return nil
	}

	for i := range c.BankAccounts {
		if c.BankAccounts[i].Enabled == true {
			if c.BankAccounts[i].BankName == "" || c.BankAccounts[i].BankAddress == "" {
				return fmt.Errorf("banking details for %s is enabled but variables not set correctly",
					c.BankAccounts[i].BankName)
			}

			if c.BankAccounts[i].AccountName == "" || c.BankAccounts[i].AccountNumber == "" {
				return fmt.Errorf("banking account details for %s variables not set correctly",
					c.BankAccounts[i].BankName)
			}
			if c.BankAccounts[i].IBAN == "" && c.BankAccounts[i].SWIFTCode == "" && c.BankAccounts[i].BSBNumber == "" {
				return fmt.Errorf("critical banking numbers not set for %s in %s account",
					c.BankAccounts[i].BankName,
					c.BankAccounts[i].AccountName)
			}

			if c.BankAccounts[i].SupportedExchanges == "" {
				c.BankAccounts[i].SupportedExchanges = "ALL"
			}
		}
	}
	return nil
}

// GetCommunicationsConfig returns the communications configuration
func (c *Config) GetCommunicationsConfig() CommunicationsConfig {
	m.Lock()
	defer m.Unlock()
	return c.Communications
}

// UpdateCommunicationsConfig sets a new updated version of a Communications
// configuration
func (c *Config) UpdateCommunicationsConfig(config CommunicationsConfig) {
	m.Lock()
	c.Communications = config
	m.Unlock()
}

// CheckCommunicationsConfig checks to see if the variables are set correctly
// from config.json
func (c *Config) CheckCommunicationsConfig() error {
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
		return errors.New("Communications config name/s not set correctly")
	}
	if c.Communications.SlackConfig.Enabled {
		if c.Communications.SlackConfig.TargetChannel == "" ||
			c.Communications.SlackConfig.VerificationToken == "" ||
			c.Communications.SlackConfig.VerificationToken == "testtest" {
			return errors.New("Slack enabled in config but variable data not set")
		}
	}
	if c.Communications.SMSGlobalConfig.Enabled {
		if c.Communications.SMSGlobalConfig.Username == "" ||
			c.Communications.SMSGlobalConfig.Password == "" ||
			len(c.Communications.SMSGlobalConfig.Contacts) == 0 {
			return errors.New("SMSGlobal enabled in config but variable data not set")
		}
	}
	if c.Communications.SMTPConfig.Enabled {
		if c.Communications.SMTPConfig.Host == "" ||
			c.Communications.SMTPConfig.Port == "" ||
			c.Communications.SMTPConfig.AccountName == "" ||
			c.Communications.SMTPConfig.AccountPassword == "" {
			return errors.New("SMTP enabled in config but variable data not set")
		}
	}
	if c.Communications.TelegramConfig.Enabled {
		if c.Communications.TelegramConfig.VerificationToken == "" {
			return errors.New("Telegram enabled in config but variable data not set")
		}
	}
	return nil
}

// CheckPairConsistency checks to see if the enabled pair exists in the
// available pairs list
func (c *Config) CheckPairConsistency(exchName string) error {
	enabledPairs, err := c.GetEnabledPairs(exchName)
	if err != nil {
		return err
	}

	availPairs, err := c.GetAvailablePairs(exchName)
	if err != nil {
		return err
	}

	var pairs, pairsRemoved []pair.CurrencyPair
	update := false
	for x := range enabledPairs {
		if !pair.Contains(availPairs, enabledPairs[x], true) {
			update = true
			pairsRemoved = append(pairsRemoved, enabledPairs[x])
			continue
		}
		pairs = append(pairs, enabledPairs[x])
	}

	if !update {
		return nil
	}

	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	if len(pairs) == 0 {
		exchCfg.EnabledPairs = pair.RandomPairFromPairs(availPairs).Pair().String()
		log.Printf("Exchange %s: No enabled pairs found in available pairs, randomly added %v\n", exchName, exchCfg.EnabledPairs)
	} else {
		exchCfg.EnabledPairs = common.JoinStrings(pair.PairsToStringArray(pairs), ",")
	}

	err = c.UpdateExchangeConfig(exchCfg)
	if err != nil {
		return err
	}

	log.Printf("Exchange %s: Removing enabled pair(s) %v from enabled pairs as it isn't an available pair", exchName, pair.PairsToStringArray(pairsRemoved))
	return nil
}

// SupportsPair returns true or not whether the exchange supports the supplied
// pair
func (c *Config) SupportsPair(exchName string, p pair.CurrencyPair) (bool, error) {
	pairs, err := c.GetAvailablePairs(exchName)
	if err != nil {
		return false, err
	}
	return pair.Contains(pairs, p, false), nil
}

// GetAvailablePairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetAvailablePairs(exchName string) ([]pair.CurrencyPair, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairs := pair.FormatPairs(common.SplitStrings(exchCfg.AvailablePairs, ","),
		exchCfg.ConfigCurrencyPairFormat.Delimiter,
		exchCfg.ConfigCurrencyPairFormat.Index)
	return pairs, nil
}

// GetEnabledPairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetEnabledPairs(exchName string) ([]pair.CurrencyPair, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairs := pair.FormatPairs(common.SplitStrings(exchCfg.EnabledPairs, ","),
		exchCfg.ConfigCurrencyPairFormat.Delimiter,
		exchCfg.ConfigCurrencyPairFormat.Index)
	return pairs, nil
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
func (c *Config) GetConfigCurrencyPairFormat(exchName string) (*CurrencyPairFormatConfig, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}
	return exchCfg.ConfigCurrencyPairFormat, nil
}

// GetRequestCurrencyPairFormat returns the request currency pair format
// for a specific exchange
func (c *Config) GetRequestCurrencyPairFormat(exchName string) (*CurrencyPairFormatConfig, error) {
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
func (c *Config) GetExchangeConfig(name string) (ExchangeConfig, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if c.Exchanges[i].Name == name {
			return c.Exchanges[i], nil
		}
	}
	return ExchangeConfig{}, fmt.Errorf(ErrExchangeNotFound, name)
}

// GetForexProviderConfig returns a forex provider configuration by its name
func (c *Config) GetForexProviderConfig(name string) (base.Settings, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Currency.ForexProviders {
		if c.Currency.ForexProviders[i].Name == name {
			return c.Currency.ForexProviders[i], nil
		}
	}
	return base.Settings{}, errors.New("provider not found")
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
func (c *Config) UpdateExchangeConfig(e ExchangeConfig) error {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if c.Exchanges[i].Name == e.Name {
			c.Exchanges[i] = e
			return nil
		}
	}
	return fmt.Errorf(ErrExchangeNotFound, e.Name)
}

// CheckExchangeConfigValues returns configuation values for all enabled
// exchanges
func (c *Config) CheckExchangeConfigValues() error {
	exchanges := 0
	for i, exch := range c.Exchanges {
		if exch.Name == "GDAX" {
			c.Exchanges[i].Name = "CoinbasePro"
		}

		if exch.WebsocketURL != WebsocketURLNonDefaultMessage {
			if exch.WebsocketURL == "" {
				c.Exchanges[i].WebsocketURL = WebsocketURLNonDefaultMessage
			}
		}

		if exch.APIURL != APIURLNonDefaultMessage {
			if exch.APIURL == "" {
				// Set default if nothing set
				c.Exchanges[i].APIURL = APIURLNonDefaultMessage
			}
		}

		if exch.APIURLSecondary != APIURLNonDefaultMessage {
			if exch.APIURLSecondary == "" {
				// Set default if nothing set
				c.Exchanges[i].APIURLSecondary = APIURLNonDefaultMessage
			}
		}

		if exch.Enabled {
			if exch.Name == "" {
				return fmt.Errorf(ErrExchangeNameEmpty, i)
			}
			if exch.AvailablePairs == "" {
				return fmt.Errorf(ErrExchangeAvailablePairsEmpty, exch.Name)
			}
			if exch.EnabledPairs == "" {
				return fmt.Errorf(ErrExchangeEnabledPairsEmpty, exch.Name)
			}
			if exch.BaseCurrencies == "" {
				return fmt.Errorf(ErrExchangeBaseCurrenciesEmpty, exch.Name)
			}
			if exch.AuthenticatedAPISupport { // non-fatal error
				if exch.APIKey == "" || exch.APISecret == "" || exch.APIKey == "Key" || exch.APISecret == "Secret" {
					c.Exchanges[i].AuthenticatedAPISupport = false
					log.Printf(WarningExchangeAuthAPIDefaultOrEmptyValues, exch.Name)
				} else if exch.Name == "ITBIT" || exch.Name == "Bitstamp" || exch.Name == "COINUT" || exch.Name == "CoinbasePro" {
					if exch.ClientID == "" || exch.ClientID == "ClientID" {
						c.Exchanges[i].AuthenticatedAPISupport = false
						log.Printf(WarningExchangeAuthAPIDefaultOrEmptyValues, exch.Name)
					}
				}
			}
			if !exch.SupportsAutoPairUpdates {
				lastUpdated := common.UnixTimestampToTime(exch.PairsLastUpdated)
				lastUpdated = lastUpdated.AddDate(0, 0, configPairsLastUpdatedWarningThreshold)
				if lastUpdated.Unix() <= time.Now().Unix() {
					log.Printf(WarningPairsLastUpdatedThresholdExceeded, exch.Name, configPairsLastUpdatedWarningThreshold)
				}
			}

			if exch.HTTPTimeout <= 0 {
				log.Printf("Exchange %s HTTP Timeout value not set, defaulting to %v.", exch.Name, configDefaultHTTPTimeout)
				c.Exchanges[i].HTTPTimeout = configDefaultHTTPTimeout
			}

			err := c.CheckPairConsistency(exch.Name)
			if err != nil {
				log.Printf("Exchange %s: CheckPairConsistency error: %s", exch.Name, err)
			}

			if len(exch.BankAccounts) == 0 {
				c.Exchanges[i].BankAccounts = append(c.Exchanges[i].BankAccounts, BankAccount{})
			} else {
				for _, bankAccount := range exch.BankAccounts {
					if bankAccount.Enabled == true {
						if bankAccount.BankName == "" || bankAccount.BankAddress == "" {
							return fmt.Errorf("banking details for %s is enabled but variables not set",
								exch.Name)
						}

						if bankAccount.AccountName == "" || bankAccount.AccountNumber == "" {
							return fmt.Errorf("banking account details for %s variables not set",
								exch.Name)
						}

						if bankAccount.SupportedCurrencies == "" {
							return fmt.Errorf("banking account details for %s acceptable funding currencies not set",
								exch.Name)
						}

						if bankAccount.BSBNumber == "" && bankAccount.IBAN == "" &&
							bankAccount.SWIFTCode == "" {
							return fmt.Errorf("banking account details for %s critical banking numbers not set",
								exch.Name)
						}
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

// CheckRESTServerConfigValues checks information before the REST server starts
// and returns an error if values are incorrect
func (c *Config) CheckRESTServerConfigValues() error {
	if c.RESTServer.AdminUsername == "" || c.RESTServer.AdminPassword == "" {
		return errors.New(WarningWebserverCredentialValuesEmpty)
	}

	if !common.StringContains(c.RESTServer.ListenAddress, ":") {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	portStr := common.SplitStrings(c.RESTServer.ListenAddress, ":")[1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if port < 1 || port > 65355 {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	return nil
}

// CheckWebsocketServerConfigValues checks information before the websocket server
// starts and returns an error if values are incorrect
func (c *Config) CheckWebsocketServerConfigValues() error {
	if c.WebsocketServer.AdminUsername == "" || c.WebsocketServer.AdminPassword == "" {
		return errors.New(WarningWebserverCredentialValuesEmpty)
	}

	if !common.StringContains(c.WebsocketServer.ListenAddress, ":") {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	portStr := common.SplitStrings(c.WebsocketServer.ListenAddress, ":")[1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if port < 1 || port > 65355 {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if c.RESTServer.Enabled && c.RESTServer.ListenAddress == c.WebsocketServer.ListenAddress {
		port++
		log.Printf("Config: Updating Websocket server port to %v prevent duplicate listen port", port)
		c.WebsocketServer.ListenAddress = common.ExtractHost(c.WebsocketServer.ListenAddress) + ":" + strconv.Itoa(port)
	}

	if c.WebsocketServer.WebsocketConnectionLimit <= 0 {
		c.WebsocketServer.WebsocketConnectionLimit = 1
	}

	if c.WebsocketServer.WebsocketMaxAuthFailures <= 0 {
		c.WebsocketServer.WebsocketMaxAuthFailures = 3
	}

	return nil
}

// CheckCurrencyConfigValues checks to see if the currency config values are correct or not
func (c *Config) CheckCurrencyConfigValues() error {
	if len(c.Currency.ForexProviders) == 0 {
		if len(forexprovider.GetAvailableForexProviders()) == 0 {
			return errors.New("no forex providers available")
		}
		var providers []base.Settings
		availProviders := forexprovider.GetAvailableForexProviders()
		for x := range availProviders {
			providers = append(providers,
				base.Settings{
					Name:             availProviders[x],
					Enabled:          false,
					Verbose:          false,
					RESTPollingDelay: 600,
					APIKey:           "Key",
					APIKeyLvl:        -1,
					PrimaryProvider:  false,
				},
			)
		}
		c.Currency.ForexProviders = providers
	}

	count := 0
	for i := range c.Currency.ForexProviders {
		if c.Currency.ForexProviders[i].Enabled == true {
			if c.Currency.ForexProviders[i].APIKey == "Key" {
				log.Printf("WARNING -- %s forex provider API key not set. Please set this in your config.json file", c.Currency.ForexProviders[i].Name)
				c.Currency.ForexProviders[i].Enabled = false
				c.Currency.ForexProviders[i].PrimaryProvider = false
				continue
			}
			if c.Currency.ForexProviders[i].APIKeyLvl == -1 && c.Currency.ForexProviders[i].Name != "CurrencyConverter" {
				log.Printf("WARNING -- %s APIKey Level not set, functions limited. Please set this in your config.json file",
					c.Currency.ForexProviders[i].Name)
			}
			count++
		}
	}

	if count == 0 {
		for x := range c.Currency.ForexProviders {
			if c.Currency.ForexProviders[x].Name == "CurrencyConverter" {
				c.Currency.ForexProviders[x].Enabled = true
				c.Currency.ForexProviders[x].APIKey = ""
				c.Currency.ForexProviders[x].PrimaryProvider = true
				log.Printf("WARNING -- No forex providers set, defaulting to free provider CurrencyConverterAPI.")
			}
		}
	}

	if len(c.Currency.Cryptocurrencies) == 0 {
		if len(c.Cryptocurrencies) != 0 {
			c.Currency.Cryptocurrencies = c.Cryptocurrencies
			c.Cryptocurrencies = ""
		} else {
			c.Currency.Cryptocurrencies = currency.DefaultCryptoCurrencies
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

	if c.Currency.FiatDisplayCurrency == "" {
		if c.FiatDisplayCurrency != "" {
			c.Currency.FiatDisplayCurrency = c.FiatDisplayCurrency
			c.FiatDisplayCurrency = ""
		} else {
			c.Currency.FiatDisplayCurrency = "USD"
		}
	}
	return nil
}

// RetrieveConfigCurrencyPairs splits, assigns and verifies enabled currency
// pairs either cryptoCurrencies or fiatCurrencies
func (c *Config) RetrieveConfigCurrencyPairs(enabledOnly bool) error {
	cryptoCurrencies := common.SplitStrings(c.Cryptocurrencies, ",")
	fiatCurrencies := common.SplitStrings(currency.DefaultCurrencies, ",")

	for x := range c.Exchanges {
		if !c.Exchanges[x].Enabled && enabledOnly {
			continue
		}

		baseCurrencies := common.SplitStrings(c.Exchanges[x].BaseCurrencies, ",")
		for y := range baseCurrencies {
			if !common.StringDataCompare(fiatCurrencies, common.StringToUpper(baseCurrencies[y])) {
				fiatCurrencies = append(fiatCurrencies, common.StringToUpper(baseCurrencies[y]))
			}
		}
	}

	for x := range c.Exchanges {
		var pairs []pair.CurrencyPair
		var err error
		if !c.Exchanges[x].Enabled && enabledOnly {
			pairs, err = c.GetEnabledPairs(c.Exchanges[x].Name)
		} else {
			pairs, err = c.GetAvailablePairs(c.Exchanges[x].Name)
		}

		if err != nil {
			return err
		}

		for y := range pairs {
			if !common.StringDataCompare(fiatCurrencies, pairs[y].FirstCurrency.Upper().String()) &&
				!common.StringDataCompare(cryptoCurrencies, pairs[y].FirstCurrency.Upper().String()) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].FirstCurrency.Upper().String())
			}

			if !common.StringDataCompare(fiatCurrencies, pairs[y].SecondCurrency.Upper().String()) &&
				!common.StringDataCompare(cryptoCurrencies, pairs[y].SecondCurrency.Upper().String()) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].SecondCurrency.Upper().String())
			}
		}
	}

	currency.Update(fiatCurrencies, false)
	currency.Update(cryptoCurrencies, true)
	return nil
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

	oldDir := exePath + common.GetOSPathSlash()
	oldDirs := []string{oldDir + ConfigFile, oldDir + EncryptedConfigFile}

	newDir := common.GetDefaultDataDir(runtime.GOOS) + common.GetOSPathSlash()
	err = common.CheckDir(newDir, true)
	if err != nil {
		return "", err
	}
	newDirs := []string{newDir + ConfigFile, newDir + EncryptedConfigFile}

	// First upgrade the old dir config file if it exists to the corresponding new one
	for x := range oldDirs {
		_, err := os.Stat(oldDirs[x])
		if os.IsNotExist(err) {
			continue
		} else {
			if path.Ext(oldDirs[x]) == ".json" {
				err = os.Rename(oldDirs[x], newDirs[0])
				if err != nil {
					return "", err
				}
				log.Printf("Renamed old config file %s to %s", oldDirs[x], newDirs[0])
			} else {
				err = os.Rename(oldDirs[x], newDirs[1])
				if err != nil {
					return "", err
				}
				log.Printf("Renamed old config file %s to %s", oldDirs[x], newDirs[1])
			}
		}
	}

	// Secondly check to see if the new config file extension is correct or not
	for x := range newDirs {
		_, err := os.Stat(newDirs[x])
		if os.IsNotExist(err) {
			continue
		}

		data, err := common.ReadFile(newDirs[x])
		if ConfirmECS(data) {
			if path.Ext(newDirs[x]) == ".dat" {
				return newDirs[x], nil
			}
			err = os.Rename(newDirs[x], newDirs[1])
			if err != nil {
				return "", err
			}
			return newDirs[1], nil
		}

		if path.Ext(newDirs[x]) == ".json" {
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

	file, err := common.ReadFile(defaultPath)
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
			if errCounter >= configMaxAuthFailres {
				return errors.New("failed to decrypt config after 3 attempts")
			}
			key, err := PromptForConfigKey(IsInitialSetup)
			if err != nil {
				log.Printf("PromptForConfigKey err: %s", err)
				errCounter++
				continue
			}

			var f []byte
			f = append(f, file...)
			data, err := DecryptConfigFile(f, key)
			if err != nil {
				log.Printf("DecryptConfigFile err: %s", err)
				errCounter++
				continue
			}

			err = ConfirmConfigJSON(data, &c)
			if err != nil {
				if errCounter < configMaxAuthFailres {
					log.Printf("Invalid password.")
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

	err = common.WriteFile(defaultPath, payload)
	if err != nil {
		return err
	}
	return nil
}

// CheckConfig checks all config settings
func (c *Config) CheckConfig() error {
	err := c.CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	if err = c.CheckCommunicationsConfig(); err != nil {
		log.Fatal(err)
	}

	if c.Webserver != nil {
		// Migrate to new settings for REST and Websocket
		c.RESTServer.AdminUsername = c.Webserver.AdminUsername
		c.RESTServer.AdminPassword = c.Webserver.AdminPassword
		c.RESTServer.Enabled = c.Webserver.Enabled
		c.RESTServer.ListenAddress = c.Webserver.ListenAddress

		c.WebsocketServer.AdminUsername = c.RESTServer.AdminUsername
		c.WebsocketServer.AdminPassword = c.RESTServer.AdminPassword
		c.WebsocketServer.Enabled = c.RESTServer.Enabled
		c.WebsocketServer.ListenAddress = c.Webserver.ListenAddress

		// Then flush the old webserver settings
		c.Webserver = nil
	}

	if c.RESTServer.Enabled {
		err = c.CheckRESTServerConfigValues()
		if err != nil {
			log.Print(fmt.Errorf(ErrCheckingConfigValues, err))
			c.RESTServer.Enabled = false
		}
	}

	if c.WebsocketServer.Enabled {
		err = c.CheckWebsocketServerConfigValues()
		if err != nil {
			log.Print(fmt.Errorf(ErrCheckingConfigValues, err))
			c.WebsocketServer.Enabled = false
		}
	}

	err = c.CheckCurrencyConfigValues()
	if err != nil {
		return err
	}

	if c.GlobalHTTPTimeout <= 0 {
		log.Printf("Global HTTP Timeout value not set, defaulting to %v.", configDefaultHTTPTimeout)
		c.GlobalHTTPTimeout = configDefaultHTTPTimeout
	}

	err = c.CheckClientBankAccounts()
	if err != nil {
		return err
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
func (c *Config) UpdateConfig(configPath string, newCfg Config) error {
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

	err = c.LoadConfig(configPath)
	if err != nil {
		return err
	}

	return nil
}

// GetConfig returns a pointer to a configuration object
func GetConfig() *Config {
	return &Cfg
}
