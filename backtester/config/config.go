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
func (c *Config) PrintSetting() {
	log.BackTester.Info("-------------------------------------------------------------")
	log.BackTester.Info("------------------Backtester Settings------------------------")
	log.BackTester.Info("-------------------------------------------------------------")
	log.BackTester.Info("------------------Strategy Settings--------------------------")
	log.BackTester.Info("-------------------------------------------------------------")
	log.BackTester.Infof("Strategy: %s", c.StrategySettings.Name)
	if len(c.StrategySettings.CustomSettings) > 0 {
		log.BackTester.Info("Custom strategy variables:")
		for k, v := range c.StrategySettings.CustomSettings {
			log.BackTester.Infof("%s: %v", k, v)
		}
	} else {
		log.BackTester.Info("Custom strategy variables: unset")
	}
	log.BackTester.Infof("Simultaneous Signal Processing: %v", c.StrategySettings.SimultaneousSignalProcessing)
	for i := range c.CurrencySettings {
		log.BackTester.Info("-------------------------------------------------------------")
		currStr := fmt.Sprintf("------------------%v %v-%v Settings---------------------------------------------------------",
			c.CurrencySettings[i].Asset,
			c.CurrencySettings[i].Base,
			c.CurrencySettings[i].Quote)
		log.BackTester.Infof(currStr[:61])
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Infof("Exchange: %v", c.CurrencySettings[i].ExchangeName)
		log.BackTester.Infof("Initial funds: %.4f", c.CurrencySettings[i].InitialFunds)
		log.BackTester.Infof("Maker fee: %.2f", c.CurrencySettings[i].TakerFee)
		log.BackTester.Infof("Taker fee: %.2f", c.CurrencySettings[i].MakerFee)
		log.BackTester.Infof("Minimum slippage percent %.2f", c.CurrencySettings[i].MinimumSlippagePercent)
		log.BackTester.Infof("Maximum slippage percent: %.2f", c.CurrencySettings[i].MaximumSlippagePercent)
		log.BackTester.Infof("Buy rules: %+v", c.CurrencySettings[i].BuySide)
		log.BackTester.Infof("Sell rules: %+v", c.CurrencySettings[i].SellSide)
		log.BackTester.Infof("Leverage rules: %+v", c.CurrencySettings[i].Leverage)
		log.BackTester.Infof("Can use exchange defined order execution limits: %+v", c.CurrencySettings[i].CanUseExchangeLimits)
	}
	log.BackTester.Info("-------------------------------------------------------------")
	log.BackTester.Info("------------------Portfolio Settings-------------------------")
	log.BackTester.Info("-------------------------------------------------------------")
	log.BackTester.Infof("Buy rules: %+v", c.PortfolioSettings.BuySide)
	log.BackTester.Infof("Sell rules: %+v", c.PortfolioSettings.SellSide)
	log.BackTester.Infof("Leverage rules: %+v", c.PortfolioSettings.Leverage)
	if c.DataSettings.LiveData != nil {
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Info("------------------Live Settings------------------------------")
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Infof("Data type: %v", c.DataSettings.DataType)
		log.BackTester.Infof("Interval: %v", c.DataSettings.Interval)
		log.BackTester.Infof("REAL ORDERS: %v", c.DataSettings.LiveData.RealOrders)
		log.BackTester.Infof("Overriding GCT API settings: %v", c.DataSettings.LiveData.APIClientIDOverride != "")
	}
	if c.DataSettings.APIData != nil {
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Info("------------------API Settings-------------------------------")
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Infof("Data type: %v", c.DataSettings.DataType)
		log.BackTester.Infof("Interval: %v", c.DataSettings.Interval)
		log.BackTester.Infof("Start date: %v", c.DataSettings.APIData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.BackTester.Infof("End date: %v", c.DataSettings.APIData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	if c.DataSettings.CSVData != nil {
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Info("------------------CSV Settings-------------------------------")
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Infof("Data type: %v", c.DataSettings.DataType)
		log.BackTester.Infof("Interval: %v", c.DataSettings.Interval)
		log.BackTester.Infof("CSV file: %v", c.DataSettings.CSVData.FullPath)
	}
	if c.DataSettings.DatabaseData != nil {
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Info("------------------Database Settings--------------------------")
		log.BackTester.Info("-------------------------------------------------------------")
		log.BackTester.Infof("Data type: %v", c.DataSettings.DataType)
		log.BackTester.Infof("Interval: %v", c.DataSettings.Interval)
		log.BackTester.Infof("Start date: %v", c.DataSettings.DatabaseData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.BackTester.Infof("End date: %v", c.DataSettings.DatabaseData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	log.BackTester.Info("-------------------------------------------------------------\n\n")
}

// Validate ensures no one sets bad config values on purpose
func (m *MinMax) Validate() {
	if m.MaximumSize < 0 {
		m.MaximumSize *= -1
		log.BackTester.Warnf("invalid maximum size set to %v", m.MaximumSize)
	}
	if m.MinimumSize < 0 {
		m.MinimumSize *= -1
		log.BackTester.Warnf("invalid minimum size set to %v", m.MinimumSize)
	}
	if m.MaximumSize <= m.MinimumSize && m.MinimumSize != 0 && m.MaximumSize != 0 {
		m.MaximumSize = m.MinimumSize + 1
		log.BackTester.Warnf("invalid maximum size set to %v", m.MaximumSize)
	}
	if m.MaximumTotal < 0 {
		m.MaximumTotal *= -1
		log.BackTester.Warnf("invalid maximum total set to %v", m.MaximumTotal)
	}
}

// ValidateDate checks whether someone has set a date poorly in their config
func (c *Config) ValidateDate() error {
	if c.DataSettings.DatabaseData != nil {
		if c.DataSettings.DatabaseData.StartDate.IsZero() ||
			c.DataSettings.DatabaseData.EndDate.IsZero() {
			return ErrStartEndUnset
		}
		if c.DataSettings.DatabaseData.StartDate.After(c.DataSettings.DatabaseData.EndDate) ||
			c.DataSettings.DatabaseData.StartDate.Equal(c.DataSettings.DatabaseData.EndDate) {
			return ErrBadDate
		}
	}
	if c.DataSettings.APIData != nil {
		if c.DataSettings.APIData.StartDate.IsZero() ||
			c.DataSettings.APIData.EndDate.IsZero() {
			return ErrStartEndUnset
		}
		if c.DataSettings.APIData.StartDate.After(c.DataSettings.APIData.EndDate) ||
			c.DataSettings.APIData.StartDate.Equal(c.DataSettings.APIData.EndDate) {
			return ErrBadDate
		}
	}
	return nil
}

// ValidateCurrencySettings checks whether someone has set invalid currency setting data in their config
func (c *Config) ValidateCurrencySettings() error {
	if len(c.CurrencySettings) == 0 {
		return ErrNoCurrencySettings
	}
	for i := range c.CurrencySettings {
		if c.CurrencySettings[i].InitialFunds <= 0 {
			return ErrBadInitialFunds
		}
		if c.CurrencySettings[i].Base == "" {
			return ErrUnsetCurrency
		}
		if c.CurrencySettings[i].Asset == "" {
			return ErrUnsetAsset
		}
		if c.CurrencySettings[i].ExchangeName == "" {
			return ErrUnsetExchange
		}
		if c.CurrencySettings[i].MinimumSlippagePercent < 0 ||
			c.CurrencySettings[i].MaximumSlippagePercent < 0 ||
			c.CurrencySettings[i].MinimumSlippagePercent > c.CurrencySettings[i].MaximumSlippagePercent {
			return ErrBadSlippageRates
		}
	}
	return nil
}
