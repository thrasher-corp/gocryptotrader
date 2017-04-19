package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

const (
	CONFIG_FILE     = "config.dat"
	OLD_CONFIG_FILE = "config.json"
	CONFIG_TEST     = "../testdata/configtest.dat"

	CONFIG_FILE_ENCRYPTION_PROMPT   = 0
	CONFIG_FILE_ENCRYPTION_ENABLED  = 1
	CONFIG_FILE_ENCRYPTION_DISABLED = -1
)

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
	RenamingConfigFile                              = "Renaming config file %s to %s."
	Cfg                                             Config
)

type WebserverConfig struct {
	Enabled       bool
	AdminUsername string
	AdminPassword string
	ListenAddress string
}

type SMSGlobalConfig struct {
	Enabled  bool
	Username string
	Password string
	Contacts []struct {
		Name    string
		Number  string
		Enabled bool
	}
}

type ConfigPost struct {
	Data Config `json:"Data"`
}

type Config struct {
	Name             string
	EncryptConfig    int
	Cryptocurrencies string
	Portfolio        portfolio.PortfolioBase `json:"PortfolioAddresses"`
	SMS              SMSGlobalConfig         `json:"SMSGlobal"`
	Webserver        WebserverConfig         `json:"Webserver"`
	Exchanges        []ExchangeConfig        `json:"Exchanges"`
}

type ExchangeConfig struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey                  string
	APISecret               string
	ClientID                string `json:",omitempty"`
	AvailablePairs          string
	EnabledPairs            string
	BaseCurrencies          string
}

func (c *Config) GetConfigEnabledExchanges() int {
	counter := 0
	for i := range c.Exchanges {
		if c.Exchanges[i].Enabled {
			counter++
		}
	}
	return counter
}

func (c *Config) GetExchangeConfig(name string) (ExchangeConfig, error) {
	for i, _ := range c.Exchanges {
		if c.Exchanges[i].Name == name {
			return c.Exchanges[i], nil
		}
	}
	return ExchangeConfig{}, fmt.Errorf(ErrExchangeNotFound, name)
}

func (c *Config) UpdateExchangeConfig(e ExchangeConfig) error {
	for i, _ := range c.Exchanges {
		if c.Exchanges[i].Name == e.Name {
			c.Exchanges[i] = e
			return nil
		}
	}
	return fmt.Errorf(ErrExchangeNotFound, e.Name)
}

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
				} else if exch.Name == "ITBIT" || exch.Name == "Bitstamp" || exch.Name == "Coinbase" {
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
	return nil
}

func (c *Config) RetrieveConfigCurrencyPairs() error {
	cryptoCurrencies := common.SplitStrings(c.Cryptocurrencies, ",")
	fiatCurrencies := common.SplitStrings(currency.DEFAULT_CURRENCIES, ",")

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

func CheckConfig() error {
	_, err := common.ReadFile(OLD_CONFIG_FILE)
	if err == nil {
		err = os.Rename(OLD_CONFIG_FILE, CONFIG_FILE)
		if err != nil {
			return err
		}
		log.Printf(RenamingConfigFile+"\n", OLD_CONFIG_FILE, CONFIG_FILE)
	}
	return nil
}

func (c *Config) ReadConfig(configPath string) error {
	defaultPath := ""
	if configPath == "" {
		defaultPath = CONFIG_FILE
	} else {
		defaultPath = configPath
	}

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

		if c.EncryptConfig == CONFIG_FILE_ENCRYPTION_DISABLED {
			return nil
		}

		if c.EncryptConfig == CONFIG_FILE_ENCRYPTION_PROMPT {
			if c.PromptForConfigEncryption() {
				c.EncryptConfig = CONFIG_FILE_ENCRYPTION_ENABLED
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

func (c *Config) SaveConfig(configPath string) error {
	defaultPath := ""
	if configPath == "" {
		defaultPath = CONFIG_FILE
	} else {
		defaultPath = configPath
	}

	payload, err := json.MarshalIndent(c, "", " ")

	if c.EncryptConfig == CONFIG_FILE_ENCRYPTION_ENABLED {
		key, err := PromptForConfigKey()
		if err != nil {
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

func (c *Config) LoadConfig(configPath string) error {
	err := c.ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf(ErrFailureOpeningConfig, CONFIG_FILE, err)
	}

	err = c.CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	return nil
}

func GetConfig() *Config {
	return &Cfg
}
