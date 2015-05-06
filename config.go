package main

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

const (
	CONFIG_FILE = "config.json"
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
	Name             string
	Enabled          bool
	Verbose          bool
	Websocket        bool
	RESTPollingDelay time.Duration
	APIKey           string
	APISecret        string
	ClientID         string
	AvailablePairs   string
	EnabledPairs     string
	BaseCurrencies   string
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
