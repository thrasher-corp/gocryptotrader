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
	log.Infof(log.BackTester, "Simultaneous Signal Processing: %v", cfg.StrategySettings.SimultaneousSignalProcessing)
	for i := range cfg.CurrencySettings {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		currStr := fmt.Sprintf("------------------%v %v-%v Settings---------------------------------------------------------",
			cfg.CurrencySettings[i].Asset,
			cfg.CurrencySettings[i].Base,
			cfg.CurrencySettings[i].Quote)
		log.Infof(log.BackTester, currStr[:61])
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Exchange: %v", cfg.CurrencySettings[i].ExchangeName)
		log.Infof(log.BackTester, "Initial funds: %v", cfg.CurrencySettings[i].InitialFunds)
		log.Infof(log.BackTester, "Maker fee: %v", cfg.CurrencySettings[i].TakerFee)
		log.Infof(log.BackTester, "Taker fee: %v", cfg.CurrencySettings[i].MakerFee)
		log.Infof(log.BackTester, "Minimum slippage percent %v", cfg.CurrencySettings[i].MinimumSlippagePercent)
		log.Infof(log.BackTester, "Maximum slippage percent: %v", cfg.CurrencySettings[i].MaximumSlippagePercent)
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
	if cfg.DataSettings.LiveData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Live Settings------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.DataSettings.Interval)
		log.Infof(log.BackTester, "REAL ORDERS: %v", cfg.DataSettings.LiveData.RealOrders)
		log.Infof(log.BackTester, "Overriding GCT API settings: %v", cfg.DataSettings.LiveData.APIClientIDOverride != "")
	}
	if cfg.DataSettings.APIData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------API Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.DataSettings.Interval)
		log.Infof(log.BackTester, "Start date: %v", cfg.DataSettings.APIData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", cfg.DataSettings.APIData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	if cfg.DataSettings.CSVData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------CSV Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.DataSettings.Interval)
		log.Infof(log.BackTester, "CSV file: %v", cfg.DataSettings.CSVData.FullPath)
	}
	if cfg.DataSettings.DatabaseData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Database Settings--------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.DataSettings.Interval)
		log.Infof(log.BackTester, "Start date: %v", cfg.DataSettings.DatabaseData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", cfg.DataSettings.DatabaseData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	log.Info(log.BackTester, "-------------------------------------------------------------\n\n")
}

// Validate ensures no one sets bad config values on purpose
func (m *MinMax) Validate() {
	if m.MaximumSize < 0 {
		m.MaximumSize *= -1
		log.Warnf(log.BackTester, "invalid maximum size set to %v", m.MaximumSize)
	}
	if m.MinimumSize < 0 {
		m.MinimumSize *= -1
		log.Warnf(log.BackTester, "invalid minimum size set to %v", m.MinimumSize)
	}
	if m.MaximumSize <= m.MinimumSize && m.MinimumSize != 0 && m.MaximumSize != 0 {
		m.MaximumSize = m.MinimumSize + 1
		log.Warnf(log.BackTester, "invalid maximum size set to %v", m.MaximumSize)
	}
	if m.MaximumTotal < 0 {
		m.MaximumTotal *= -1
		log.Warnf(log.BackTester, "invalid maximum total set to %v", m.MaximumTotal)
	}
}
