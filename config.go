package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	CONFIG_FILE     = "config.dat"
	OLD_CONFIG_FILE = "config.json"

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
	ErrFailureOpeningConfig                         = "Fatal error opening config.json file. Error: %s"
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
)

type Webserver struct {
	Enabled       bool
	AdminUsername string
	AdminPassword string
	ListenAddress string
}

type SMSGlobal struct {
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
	SMS              SMSGlobal `json:"SMSGlobal"`
	Webserver        Webserver `json:"Webserver"`
	Exchanges        []Exchanges
}

type Exchanges struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey                  string
	APISecret               string
	ClientID                string
	AvailablePairs          string
	EnabledPairs            string
	BaseCurrencies          string
}

func GetEnabledExchanges() int {
	counter := 0
	for i := range bot.config.Exchanges {
		if bot.config.Exchanges[i].Enabled {
			counter++
		}
	}
	return counter
}

func GetExchangeConfig(name string) (Exchanges, error) {
	for i, _ := range bot.config.Exchanges {
		if bot.config.Exchanges[i].Name == name {
			return bot.config.Exchanges[i], nil
		}
	}
	return Exchanges{}, fmt.Errorf(ErrExchangeNotFound, name)
}

func UpdateExchangeConfig(e Exchanges) error {
	for i, _ := range bot.config.Exchanges {
		if bot.config.Exchanges[i].Name == e.Name {
			bot.config.Exchanges[i] = e
			return nil
		}
	}
	return fmt.Errorf(ErrExchangeNotFound, e.Name)
}

func CheckSMSGlobalConfigValues() error {
	if bot.config.SMS.Username == "" || bot.config.SMS.Username == "Username" || bot.config.SMS.Password == "" || bot.config.SMS.Password == "Password" {
		return errors.New(WarningSMSGlobalDefaultOrEmptyValues)
	}
	contacts := 0
	for i := range bot.config.SMS.Contacts {
		if bot.config.SMS.Contacts[i].Enabled {
			if bot.config.SMS.Contacts[i].Name == "" || bot.config.SMS.Contacts[i].Number == "" || (bot.config.SMS.Contacts[i].Name == "Bob" && bot.config.SMS.Contacts[i].Number == "12345") {
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

func CheckExchangeConfigValues() error {
	if bot.config.Cryptocurrencies == "" {
		return errors.New(ErrCryptocurrenciesEmpty)
	}

	exchanges := 0
	for i, exch := range bot.config.Exchanges {
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
					bot.config.Exchanges[i].AuthenticatedAPISupport = false
					log.Printf(WarningExchangeAuthAPIDefaultOrEmptyValues, exch.Name)
					continue
				} else if exch.Name == "ITBIT" || exch.Name == "Bitstamp" || exch.Name == "Coinbase" {
					if exch.ClientID == "" || exch.ClientID == "ClientID" {
						bot.config.Exchanges[i].AuthenticatedAPISupport = false
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

func CheckWebserverValues() error {
	if bot.config.Webserver.AdminUsername == "" || bot.config.Webserver.AdminPassword == "" {
		return errors.New(WarningWebserverCredentialValuesEmpty)
	}

	if !StringContains(bot.config.Webserver.ListenAddress, ":") {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	portStr := SplitStrings(bot.config.Webserver.ListenAddress, ":")[1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if port < 1 || port > 65355 {
		return errors.New(WarningWebserverListenAddressInvalid)
	}
	return nil
}

func ReadConfig() error {
	_, err := ioutil.ReadFile(OLD_CONFIG_FILE)
	if err == nil {
		err = os.Rename(OLD_CONFIG_FILE, CONFIG_FILE)
		if err != nil {
			return err
		}
		log.Printf(RenamingConfigFile+"\n", OLD_CONFIG_FILE, CONFIG_FILE)
	}

	file, err := ioutil.ReadFile(CONFIG_FILE)
	if err != nil {
		return err
	}

	if !ConfirmECS(file) {
		err = ConfirmConfigJSON(file, &bot.config)
		if err != nil {
			return err
		}

		if bot.config.EncryptConfig == CONFIG_FILE_ENCRYPTION_DISABLED {
			return nil
		}

		if bot.config.EncryptConfig == CONFIG_FILE_ENCRYPTION_PROMPT {
			if PromptForConfigEncryption() {
				bot.config.EncryptConfig = CONFIG_FILE_ENCRYPTION_ENABLED
				SaveConfig()
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

		err = ConfirmConfigJSON(data, &bot.config)
		if err != nil {
			return err
		}
	}
	return nil
}

func SaveConfig() error {
	payload, err := json.MarshalIndent(bot.config, "", " ")

	if bot.config.EncryptConfig == CONFIG_FILE_ENCRYPTION_ENABLED {
		key, err := PromptForConfigKey()
		if err != nil {
			return err
		}

		payload, err = EncryptConfigFile(payload, key)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(CONFIG_FILE, payload, 0644)
	if err != nil {
		return err
	}
	return nil
}

func LoadConfig() error {
	err := ReadConfig()
	if err != nil {
		return fmt.Errorf(ErrFailureOpeningConfig, err)
	}

	err = CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	return nil
}
