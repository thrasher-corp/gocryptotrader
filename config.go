package main

import (
	"io/ioutil"
	"encoding/json"
)

type Config struct {
	Exchanges []Exchanges
}

type Exchanges struct {
	Name string
	Enabled bool
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
