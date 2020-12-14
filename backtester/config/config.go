package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// ReadConfigFromFile will take a config from a path
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

// LoadConfig unmarshalls byte data into a config struct
func LoadConfig(data []byte) (resp *Config, err error) {
	err = json.Unmarshal(data, &resp)
	return resp, err
}

// PrintSetting prints relevant settings to the console for easy reading
func (cfg *Config) PrintSetting() {
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Backtester Settings------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Strategy Settings--------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Strategy: %s", cfg.StrategySettings.Name)
	if len(cfg.StrategySettings.CustomSettings) > 0 {
		log.Info(log.BackTester, "Custom strategy variables:")
		for k, v := range cfg.StrategySettings.CustomSettings {
			log.Infof(log.BackTester, "%s: %v", k, v)
		}
	} else {
		log.Info(log.BackTester, "Custom strategy variables: unset")
	}
	log.Infof(log.BackTester, "MultiCurrency Assessment: %v", cfg.StrategySettings.IsMultiCurrency)
	for i := range cfg.CurrencySettings {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		currStr := fmt.Sprintf("------------------%v %v-%v Settings--------------------------",
			cfg.CurrencySettings[i].Asset,
			cfg.CurrencySettings[i].Base,
			cfg.CurrencySettings[i].Quote)
		log.Infof(log.BackTester, currStr[:61])
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Exchange: %v", cfg.CurrencySettings[i].ExchangeName)
		log.Infof(log.BackTester, "Initial funds: %v", cfg.CurrencySettings[i].InitialFunds)
		log.Infof(log.BackTester, "Maker fee: %v", cfg.CurrencySettings[i].TakerFee)
		log.Infof(log.BackTester, "Taker fee: %v", cfg.CurrencySettings[i].MakerFee)
		log.Infof(log.BackTester, "Buy rules: %+v", cfg.CurrencySettings[i].BuySide)
		log.Infof(log.BackTester, "Sell rules: %+v", cfg.CurrencySettings[i].SellSide)
		log.Infof(log.BackTester, "Leverage rules: %+v", cfg.CurrencySettings[i].Leverage)
	}
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Portfolio Settings-------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Buy rules: %+v", cfg.PortfolioSettings.BuySide)
	log.Infof(log.BackTester, "Sell rules: %+v", cfg.PortfolioSettings.SellSide)
	log.Infof(log.BackTester, "Leverage rules: %+v", cfg.PortfolioSettings.Leverage)
	if cfg.LiveData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Live Settings------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.LiveData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.LiveData.Interval)
		log.Infof(log.BackTester, "REAL ORDERS: %v", cfg.LiveData.RealOrders)
		log.Infof(log.BackTester, "Overriding GCT API settings: %v", cfg.LiveData.APIClientIDOverride != "")
	}
	if cfg.APIData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------API Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.APIData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.APIData.Interval)
		log.Infof(log.BackTester, "Start date: %v", cfg.APIData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", cfg.APIData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	if cfg.CSVData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------CSV Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.CSVData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.CSVData.Interval)
	}
	if cfg.DatabaseData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Database Settings--------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.DatabaseData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.DatabaseData.Interval)
		log.Infof(log.BackTester, "Start date: %v", cfg.DatabaseData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", cfg.DatabaseData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	log.Info(log.BackTester, "-------------------------------------------------------------\n\n")
}
