package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

const (
	CONFIG_FILE = "config.json"
)

var (
	ErrExchangeNameEmpty                            = "Exchange #%d in config: Exchange name is empty."
	ErrExchangeAvailablePairsEmpty                  = "Exchange %s: Available pairs is empty."
	ErrExchangeEnabledPairsEmpty                    = "Exchange %s: Enabled pairs is empty."
	ErrExchangeBaseCurrenciesEmpty                  = "Exchange %s: Base currencies is empty."
	WarningExchangeAuthAPIDefaultOrEmptyValues      = "WARNING -- Exchange %s: Authenticated API support disabled due to default/empty APIKey/Secret/ClientID values."
	ErrExchangeNotFound                             = "Exchange %s: Not found."
	ErrNoEnabledExchanges                           = "No Exchanges enabled."
	ErrCryptocurrenciesEmpty                        = "Cryptocurrencies variable is empty."
	WarningSMSGlobalDefaultOrEmptyValues            = "WARNING -- SMS Support disabled due to default or empty Username/Password values."
	WarningSSMSGlobalSMSContactDefaultOrEmptyValues = "WARNING -- SMS contact #%d Name/Number disabled due to default or empty values."
	WarningSSMSGlobalSMSNoContacts                  = "WARNING -- SMS Support disabled due to no enabled contacts."
	WarningWebserverCredentialValuesEmpty           = "WARNING -- Webserver support disabled due to empty Username/Password values."
	WarningWebserverListenAddressInvalid            = "WARNING -- Webserver support disabled due to invalid listen address."
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

type Config struct {
	Name             string
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

func ReadConfig() (Config, error) {
	file, err := ioutil.ReadFile(CONFIG_FILE)

	if err != nil {
		return Config{}, err
	}

	cfg := Config{}
	err = json.Unmarshal(file, &cfg)
	return cfg, err
}

func SaveConfig() error {
	payload, err := json.MarshalIndent(bot.config, "", " ")

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(CONFIG_FILE, payload, 0644)

	if err != nil {
		return err
	}

	return nil
}
