package main

import (
	"io/ioutil"
	"encoding/json"
)

type SMSContacts struct {
	Name string
	Number string
	Enabled bool
}

type Config struct {
	Name string
	SMSGlobalUsername string
	SMSGlobalPassword string
	SMSContacts []SMSContacts
	Exchanges []Exchanges
}

type Exchanges struct {
	Name string
	Enabled bool
	Verbose bool
	APIKey string
	APISecret string
	ClientID string
	Pairs string
	BaseCurrencies string
}

func ReadConfig(path string) (Config, error) {
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return Config{}, err
	}

	cfg := Config{}
	err = json.Unmarshal(file, &cfg)
	return cfg, err
}
