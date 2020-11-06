package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/thrasher-corp/gocryptotrader/common/file"
)

func ReadConfigFromFile(path string) (*Config, error) {
	if !file.Exists(path) {
		return nil, errors.New("file not found")
	}

	fileData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return LoadConfig(fileData)
}

func LoadConfig(data []byte) (resp *Config, err error) {
	err = json.Unmarshal(data, &resp)
	return resp, err
}
