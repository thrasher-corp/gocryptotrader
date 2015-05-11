package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

const (
	CONFIG_FILE = "config.json"
)

var (
	ErrExchangeNameEmpty                   = "Exchange #%d in config: Exchange name is empty."
	ErrExchangeAvailablePairsEmpty         = "Exchange %s: Available pairs is empty."
	ErrExchangeEnabledPairsEmpty           = "Exchange %s: Enabled pairs is empty."
	ErrExchangeBaseCurrenciesEmpty         = "Exchange %s: Base currencies is empty."
	ErrExchangeAuthAPIDefaultOrEmptyValues = "WARNING -- Exchange %s: Authenticated API support disabled due to default/empty APIKey/Secret/ClientID values."
)

type SMSContacts struct {
	Name    string
	Number  string
	Enabled bool
}

type Config struct {
	Name              string
	SMSGlobalUsername string
	SMSGlobalPassword string
	SMSContacts       []SMSContacts
	Exchanges         []Exchanges
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

func CheckConfigValues() error {
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
					log.Printf(ErrExchangeAuthAPIDefaultOrEmptyValues, exch.Name)
					continue
				} else if exch.Name == "ITBIT" || exch.Name == "Bitstamp" || exch.Name == "Coinbase" {
					if exch.ClientID == "" || exch.ClientID == "ClientID" {
						bot.config.Exchanges[i].AuthenticatedAPISupport = false
						log.Printf(ErrExchangeAuthAPIDefaultOrEmptyValues, exch.Name)
						continue
					}
				}
			}
		}
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
