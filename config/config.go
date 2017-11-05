package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

// Constants declared here are filename strings and test strings
const (
	ConfigFile                   = "config.dat"
	OldConfigFile                = "config.json"
	ConfigTestFile               = "../testdata/configtest.dat"
	configFileEncryptionPrompt   = 0
	configFileEncryptionEnabled  = 1
	configFileEncryptionDisabled = -1
)

// Variables here are mainly alerts and a configuration object
var (
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
	RenamingConfigFile                              = "Renaming config file %s to %s."
	Cfg                                             Config
)

// WebserverConfig struct holds the prestart variables for the webserver.
type WebserverConfig struct {
	Enabled                      bool
	AdminUsername                string
	AdminPassword                string
	ListenAddress                string
	WebsocketConnectionLimit     int
	WebsocketAllowInsecureOrigin bool
}

// SMSGlobalConfig structure holds all the variables you need for instant
// messaging and broadcast used by SMSGlobal
type SMSGlobalConfig struct {
	Enabled  bool
	Username string
	Password string
	Contacts []smsglobal.Contact
}

// Post holds the bot configuration data
type Post struct {
	Data Config `json:"Data"`
}

// CurrencyPairFormatConfig stores the users preferred currency pair display
type CurrencyPairFormatConfig struct {
	Uppercase bool
	Delimiter string `json:",omitempty"`
	Separator string `json:",omitempty"`
	Index     string `json:",omitempty"`
}

// Config is the overarching object that holds all the information for
// prestart management of portfolio, SMSGlobal, webserver and enabled exchange
type Config struct {
	Name                     string
	EncryptConfig            int
	Cryptocurrencies         string
	CurrencyExchangeProvider string
	CurrencyPairFormat       *CurrencyPairFormatConfig `json:"CurrencyPairFormat"`
	FiatDisplayCurrency      string
	Portfolio                portfolio.Base   `json:"PortfolioAddresses"`
	SMS                      SMSGlobalConfig  `json:"SMSGlobal"`
	Webserver                WebserverConfig  `json:"Webserver"`
	Exchanges                []ExchangeConfig `json:"Exchanges"`
}

// ExchangeConfig holds all the information needed for each enabled Exchange.
type ExchangeConfig struct {
	Name                      string
	Enabled                   bool
	Verbose                   bool
	Websocket                 bool
	UseSandbox                bool
	RESTPollingDelay          time.Duration
	AuthenticatedAPISupport   bool
	APIKey                    string
	APISecret                 string
	ClientID                  string `json:",omitempty"`
	AvailablePairs            string
	EnabledPairs              string
	BaseCurrencies            string
	AssetTypes                string
	ConfigCurrencyPairFormat  *CurrencyPairFormatConfig `json:"ConfigCurrencyPairFormat"`
	RequestCurrencyPairFormat *CurrencyPairFormatConfig `json:"RequestCurrencyPairFormat"`
}

// GetConfigEnabledExchanges returns the number of exchanges that are enabled.
func (c *Config) GetConfigEnabledExchanges() int {
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
	return c.CurrencyPairFormat
}

// GetExchangeConfig returns your exchange configurations by its indivdual name
func (c *Config) GetExchangeConfig(name string) (ExchangeConfig, error) {
	for i := range c.Exchanges {
		if c.Exchanges[i].Name == name {
			return c.Exchanges[i], nil
		}
	}
	return ExchangeConfig{}, fmt.Errorf(ErrExchangeNotFound, name)
}

// UpdateExchangeConfig updates exchange configurations
func (c *Config) UpdateExchangeConfig(e ExchangeConfig) error {
	for i := range c.Exchanges {
		if c.Exchanges[i].Name == e.Name {
			c.Exchanges[i] = e
			return nil
		}
	}
	return fmt.Errorf(ErrExchangeNotFound, e.Name)
}

// CheckSMSGlobalConfigValues checks concurrent SMSGlobal configurations
func (c *Config) CheckSMSGlobalConfigValues() error {
	if c.SMS.Username == "" || c.SMS.Username == "Username" || c.SMS.Password == "" || c.SMS.Password == "Password" {
		return errors.New(WarningSMSGlobalDefaultOrEmptyValues)
	}
	contacts := 0
	for i := range c.SMS.Contacts {
		if c.SMS.Contacts[i].Enabled {
			if c.SMS.Contacts[i].Name == "" || c.SMS.Contacts[i].Number == "" || (c.SMS.Contacts[i].Name == "Bob" && c.SMS.Contacts[i].Number == "12345") {
				log.Printf(WarningSSMSGlobalSMSContactDefaultOrEmptyValues, i)
				continue
			}
			contacts++
		}
	}
	if contacts == 0 {
		return errors.New(WarningSSMSGlobalSMSNoContacts)
	}
	return nil
}

// CheckExchangeConfigValues returns configuation values for all enabled
// exchanges
func (c *Config) CheckExchangeConfigValues() error {
	if c.Cryptocurrencies == "" {
		return errors.New(ErrCryptocurrenciesEmpty)
	}

	exchanges := 0
	for i, exch := range c.Exchanges {
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
					continue
				} else if exch.Name == "ITBIT" || exch.Name == "Bitstamp" || exch.Name == "COINUT" || exch.Name == "GDAX" {
					if exch.ClientID == "" || exch.ClientID == "ClientID" {
						c.Exchanges[i].AuthenticatedAPISupport = false
						log.Printf(WarningExchangeAuthAPIDefaultOrEmptyValues, exch.Name)
						continue
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

// CheckWebserverConfigValues checks information before webserver starts and
// returns an error if values are incorrect.
func (c *Config) CheckWebserverConfigValues() error {
	if c.Webserver.AdminUsername == "" || c.Webserver.AdminPassword == "" {
		return errors.New(WarningWebserverCredentialValuesEmpty)
	}

	if !common.StringContains(c.Webserver.ListenAddress, ":") {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	portStr := common.SplitStrings(c.Webserver.ListenAddress, ":")[1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if port < 1 || port > 65355 {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if c.Webserver.WebsocketConnectionLimit <= 0 {
		c.Webserver.WebsocketConnectionLimit = 1
	}

	return nil
}

// RetrieveConfigCurrencyPairs splits, assigns and verifies enabled currency
// pairs either cryptoCurrencies or fiatCurrencies
func (c *Config) RetrieveConfigCurrencyPairs() error {
	cryptoCurrencies := common.SplitStrings(c.Cryptocurrencies, ",")
	fiatCurrencies := common.SplitStrings(currency.DefaultCurrencies, ",")

	for _, s := range cryptoCurrencies {
		_, err := strconv.Atoi(s)
		if err != nil && common.StringContains(c.Cryptocurrencies, s) {
			continue
		} else {
			return errors.New("RetrieveConfigCurrencyPairs: Incorrect Crypto-Currency")
		}
	}

	for _, exchange := range c.Exchanges {
		if exchange.Enabled {
			baseCurrencies := common.SplitStrings(exchange.BaseCurrencies, ",")
			enabledCurrencies := common.SplitStrings(exchange.EnabledPairs, ",")

			for _, currencyPair := range enabledCurrencies {
				ok, separator := currency.ContainsSeparator(currencyPair)
				if ok {
					pair := common.SplitStrings(currencyPair, separator)
					for _, x := range pair {
						ok, _ = currency.ContainsBaseCurrencyIndex(baseCurrencies, x)
						if !ok {
							cryptoCurrencies = currency.CheckAndAddCurrency(cryptoCurrencies, x)
						}
					}
				} else {
					ok, idx := currency.ContainsBaseCurrencyIndex(baseCurrencies, currencyPair)
					if ok {
						curr := strings.Replace(currencyPair, idx, "", -1)

						if currency.ContainsBaseCurrency(baseCurrencies, curr) {
							fiatCurrencies = currency.CheckAndAddCurrency(fiatCurrencies, curr)
						} else {
							cryptoCurrencies = currency.CheckAndAddCurrency(cryptoCurrencies, curr)
						}

						if currency.ContainsBaseCurrency(baseCurrencies, idx) {
							fiatCurrencies = currency.CheckAndAddCurrency(fiatCurrencies, idx)
						} else {
							cryptoCurrencies = currency.CheckAndAddCurrency(cryptoCurrencies, idx)
						}
					}
				}
			}
		}
	}

	currency.BaseCurrencies = common.JoinStrings(fiatCurrencies, ",")
	if common.StringContains(currency.BaseCurrencies, "RUR") {
		currency.BaseCurrencies = strings.Replace(currency.BaseCurrencies, "RUR", "RUB", -1)
	}
	c.Cryptocurrencies = common.JoinStrings(cryptoCurrencies, ",")
	currency.CryptoCurrencies = c.Cryptocurrencies

	return nil
}

// GetFilePath returns the desired config file or the default config file name
// based on if the application is being run under test or normal mode.
func GetFilePath(file string) string {
	if file != "" {
		return file
	}
	if flag.Lookup("test.v") == nil {
		return ConfigFile
	}
	return ConfigTestFile
}

// CheckConfig checks to see if there is an old configuration filename and path
// if found it will change it to correct filename.
func CheckConfig() error {
	_, err := common.ReadFile(OldConfigFile)
	if err == nil {
		err = os.Rename(OldConfigFile, ConfigFile)
		if err != nil {
			return err
		}
		log.Printf(RenamingConfigFile+"\n", OldConfigFile, ConfigFile)
	}
	return nil
}

// ReadConfig verifies and checks for encryption and verifies the unencrypted
// file contains JSON.
func (c *Config) ReadConfig(configPath string) error {
	defaultPath := GetFilePath(configPath)
	err := CheckConfig()
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
			if c.PromptForConfigEncryption() {
				c.EncryptConfig = configFileEncryptionEnabled
				return c.SaveConfig("")
			}
		}
	} else {
		key, err := PromptForConfigKey()
		if err != nil {
			return err
		}

		data, err := DecryptConfigFile(file, key)
		if err != nil {
			return err
		}

		err = ConfirmConfigJSON(data, &c)
		if err != nil {
			return err
		}
	}
	return nil
}

// SaveConfig saves your configuration to your desired path
func (c *Config) SaveConfig(configPath string) error {
	defaultPath := GetFilePath(configPath)
	payload, err := json.MarshalIndent(c, "", " ")

	if c.EncryptConfig == configFileEncryptionEnabled {
		key, err2 := PromptForConfigKey()
		if err2 != nil {
			return err
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

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig(configPath string) error {
	err := c.ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf(ErrFailureOpeningConfig, configPath, err)
	}

	err = c.CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	if c.SMS.Enabled {
		err = c.CheckSMSGlobalConfigValues()
		if err != nil {
			log.Print(fmt.Errorf(ErrCheckingConfigValues, err))
			c.SMS.Enabled = false
		}
	}

	if c.Webserver.Enabled {
		err = c.CheckWebserverConfigValues()
		if err != nil {
			log.Print(fmt.Errorf(ErrCheckingConfigValues, err))
			c.Webserver.Enabled = false
		}
	}

	if c.CurrencyExchangeProvider == "" {
		c.CurrencyExchangeProvider = "fixer"
	} else {
		if c.CurrencyExchangeProvider != "yahoo" && c.CurrencyExchangeProvider != "fixer" {
			log.Println(WarningCurrencyExchangeProvider)
			c.CurrencyExchangeProvider = "fixer"
		}
	}

	if c.CurrencyPairFormat == nil {
		c.CurrencyPairFormat = &CurrencyPairFormatConfig{
			Delimiter: "-",
			Uppercase: true,
		}
	}

	if c.FiatDisplayCurrency == "" {
		c.FiatDisplayCurrency = "USD"
	}

	return nil
}

// UpdateConfig updates the config with a supplied config file
func (c *Config) UpdateConfig(configPath string, newCfg Config) error {
	if c.Name != newCfg.Name && newCfg.Name != "" {
		c.Name = newCfg.Name
	}

	err := newCfg.CheckExchangeConfigValues()
	if err != nil {
		return err
	}
	c.Exchanges = newCfg.Exchanges

	if c.CurrencyPairFormat != newCfg.CurrencyPairFormat {
		c.CurrencyPairFormat = newCfg.CurrencyPairFormat
	}

	c.Portfolio = newCfg.Portfolio

	err = newCfg.CheckSMSGlobalConfigValues()
	if err != nil {
		return err
	}
	c.SMS = newCfg.SMS

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

// GetConfig returns a pointer to a confiuration object
func GetConfig() *Config {
	return &Cfg
}
