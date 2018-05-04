package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/smsglobal"
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
	WarningPairsLastUpdatedThresholdExceeded        = "WARNING -- Exchange %s: Last manual update of available currency pairs has exceeded %d days. Manual update required!"
	Cfg                                             Config
	IsInitialSetup                                  bool
	testBypass                                      bool
	m                                               sync.Mutex
)

// WebserverConfig struct holds the prestart variables for the webserver.
type WebserverConfig struct {
	Enabled                      bool
	AdminUsername                string
	AdminPassword                string
	ListenAddress                string
	WebsocketConnectionLimit     int
	WebsocketMaxAuthFailures     int
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
	Name                string
	EncryptConfig       int
	Cryptocurrencies    string                    `json:"Cryptocurrencies,omitempty"`
	Currency            CurrencyConfig            `json:"CurrencyConfig,omitempty"`
	CurrencyPairFormat  *CurrencyPairFormatConfig `json:"CurrencyPairFormat,omitempty"`
	FiatDisplayCurrency string                    `json:"FiatDispayCurrency,omitempty"`
	GlobalHTTPTimeout   time.Duration
	Portfolio           portfolio.Base   `json:"PortfolioAddresses"`
	SMS                 SMSGlobalConfig  `json:"SMSGlobal"`
	Webserver           WebserverConfig  `json:"Webserver"`
	Exchanges           []ExchangeConfig `json:"Exchanges"`
}

// ExchangeConfig holds all the information needed for each enabled Exchange.
type ExchangeConfig struct {
	Name                      string
	Enabled                   bool
	Verbose                   bool
	Websocket                 bool
	UseSandbox                bool
	RESTPollingDelay          time.Duration
	HTTPTimeout               time.Duration
	AuthenticatedAPISupport   bool
	APIKey                    string
	APISecret                 string
	ClientID                  string `json:",omitempty"`
	AvailablePairs            string
	EnabledPairs              string
	BaseCurrencies            string
	AssetTypes                string
	SupportsAutoPairUpdates   bool
	PairsLastUpdated          int64                     `json:",omitempty"`
	ConfigCurrencyPairFormat  *CurrencyPairFormatConfig `json:"ConfigCurrencyPairFormat"`
	RequestCurrencyPairFormat *CurrencyPairFormatConfig `json:"RequestCurrencyPairFormat"`
}

// CurrencyConfig holds all the information needed for currency related manipulation
type CurrencyConfig struct {
	ForexProviders      []base.Settings           `json:"ForexProviders"`
	Cryptocurrencies    string                    `json:"Cryptocurrencies"`
	CurrencyPairFormat  *CurrencyPairFormatConfig `json:"CurrencyPairFormat"`
	FiatDisplayCurrency string
}

// GetCurrencyConfig returns currency configurations
func (c *Config) GetCurrencyConfig() CurrencyConfig {
	return c.Currency
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
				} else if exch.Name == "ITBIT" || exch.Name == "Bitstamp" || exch.Name == "COINUT" || exch.Name == "GDAX" {
					if exch.ClientID == "" || exch.ClientID == "ClientID" {
						c.Exchanges[i].AuthenticatedAPISupport = false
						log.Printf(WarningExchangeAuthAPIDefaultOrEmptyValues, exch.Name)
					}
				}
			}
			if !exch.SupportsAutoPairUpdates {
				lastUpdated := common.UnixTimestampToTime(exch.PairsLastUpdated)
				lastUpdated.AddDate(0, 0, configPairsLastUpdatedWarningThreshold)
				if lastUpdated.Unix() <= time.Now().Unix() {
					log.Printf(WarningPairsLastUpdatedThresholdExceeded, exch.Name, configPairsLastUpdatedWarningThreshold)
				}
			}

			if exch.HTTPTimeout <= 0 {
				log.Printf("Exchange %s HTTP Timeout value not set, defaulting to %v.", exch.Name, configDefaultHTTPTimeout)
				c.Exchanges[i].HTTPTimeout = configDefaultHTTPTimeout
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

	if c.Webserver.WebsocketMaxAuthFailures <= 0 {
		c.Webserver.WebsocketMaxAuthFailures = 3
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
			if c.Currency.ForexProviders[i].APIKeyLvl == -1 {
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
		log.Fatalf("Unable to get executable path: %s", err)
		return "", err
	}

	tempPath := exePath + common.GetOSPathSlash()
	encPath := tempPath + EncryptedConfigFile
	cfgPath := tempPath + ConfigFile

	data, err := common.ReadFile(encPath)
	if err == nil {
		if ConfirmECS(data) {
			return encPath, nil
		}
		err = os.Rename(encPath, cfgPath)
		if err != nil {
			log.Fatalf("Unable to rename config file: %s", err)
			return "", err
		}
		log.Printf("Renaming non-encrypted config file from %s to %s",
			encPath, cfgPath)
		return cfgPath, nil
	}
	if !ConfirmECS(data) {
		return cfgPath, nil
	}
	err = os.Rename(cfgPath, encPath)
	if err != nil {
		log.Fatalf("Unable to rename config file: %s", err)
		return "", err
	}
	log.Printf("Renamed encrypted config file from %s to %s", cfgPath,
		encPath)
	return encPath, nil
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
		var err error

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

	err = c.CheckCurrencyConfigValues()
	if err != nil {
		return err
	}

	if c.GlobalHTTPTimeout <= 0 {
		log.Printf("Global HTTP Timeout value not set, defaulting to %v.", configDefaultHTTPTimeout)
		c.GlobalHTTPTimeout = configDefaultHTTPTimeout
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
	c.SMS = newCfg.SMS
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
