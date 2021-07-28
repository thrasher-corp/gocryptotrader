package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

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
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Backtester Settings------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Strategy Settings--------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Strategy: %s", c.StrategySettings.Name)
	if len(c.StrategySettings.CustomSettings) > 0 {
		log.Info(log.BackTester, "Custom strategy variables:")
		for k, v := range c.StrategySettings.CustomSettings {
			log.Infof(log.BackTester, "%s: %v", k, v)
		}
	} else {
		log.Info(log.BackTester, "Custom strategy variables: unset")
	}
	log.Infof(log.BackTester, "Simultaneous Signal Processing: %v", c.StrategySettings.SimultaneousSignalProcessing)
	for i := range c.CurrencySettings {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		currStr := fmt.Sprintf("------------------%v %v-%v Settings---------------------------------------------------------",
			c.CurrencySettings[i].Asset,
			c.CurrencySettings[i].Base,
			c.CurrencySettings[i].Quote)
		log.Infof(log.BackTester, currStr[:61])
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Exchange: %v", c.CurrencySettings[i].ExchangeName)
		log.Infof(log.BackTester, "Initial funds: %.4f", c.CurrencySettings[i].InitialFunds)
		log.Infof(log.BackTester, "Maker fee: %.2f", c.CurrencySettings[i].TakerFee)
		log.Infof(log.BackTester, "Taker fee: %.2f", c.CurrencySettings[i].MakerFee)
		log.Infof(log.BackTester, "Minimum slippage percent %.2f", c.CurrencySettings[i].MinimumSlippagePercent)
		log.Infof(log.BackTester, "Maximum slippage percent: %.2f", c.CurrencySettings[i].MaximumSlippagePercent)
		log.Infof(log.BackTester, "Buy rules: %+v", c.CurrencySettings[i].BuySide)
		log.Infof(log.BackTester, "Sell rules: %+v", c.CurrencySettings[i].SellSide)
		log.Infof(log.BackTester, "Leverage rules: %+v", c.CurrencySettings[i].Leverage)
		log.Infof(log.BackTester, "Can use exchange defined order execution limits: %+v", c.CurrencySettings[i].CanUseExchangeLimits)
	}
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Portfolio Settings-------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Buy rules: %+v", c.PortfolioSettings.BuySide)
	log.Infof(log.BackTester, "Sell rules: %+v", c.PortfolioSettings.SellSide)
	log.Infof(log.BackTester, "Leverage rules: %+v", c.PortfolioSettings.Leverage)
	if c.DataSettings.LiveData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Live Settings------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", c.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", c.DataSettings.Interval)
		log.Infof(log.BackTester, "REAL ORDERS: %v", c.DataSettings.LiveData.RealOrders)
		log.Infof(log.BackTester, "Overriding GCT API settings: %v", c.DataSettings.LiveData.APIClientIDOverride != "")
	}
	if c.DataSettings.APIData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------API Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", c.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", c.DataSettings.Interval)
		log.Infof(log.BackTester, "Start date: %v", c.DataSettings.APIData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", c.DataSettings.APIData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	if c.DataSettings.CSVData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------CSV Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", c.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", c.DataSettings.Interval)
		log.Infof(log.BackTester, "CSV file: %v", c.DataSettings.CSVData.FullPath)
	}
	if c.DataSettings.DatabaseData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Database Settings--------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", c.DataSettings.DataType)
		log.Infof(log.BackTester, "Interval: %v", c.DataSettings.Interval)
		log.Infof(log.BackTester, "Start date: %v", c.DataSettings.DatabaseData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", c.DataSettings.DatabaseData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	log.Info(log.BackTester, "-------------------------------------------------------------\n\n")
}

// Validate ensures no one sets bad config values on purpose
func (m *MinMax) Validate() {
	if m.MaximumSize < 0 {
		m.MaximumSize *= -1
		log.Warnf(log.BackTester, "invalid maximum size set to %f", m.MaximumSize)
	}
	if m.MinimumSize < 0 {
		m.MinimumSize *= -1
		log.Warnf(log.BackTester, "invalid minimum size set to %f", m.MinimumSize)
	}
	if m.MaximumSize <= m.MinimumSize && m.MinimumSize != 0 && m.MaximumSize != 0 {
		m.MaximumSize = m.MinimumSize + 1
		log.Warnf(log.BackTester, "invalid maximum size set to %f", m.MaximumSize)
	}
	if m.MaximumTotal < 0 {
		m.MaximumTotal *= -1
		log.Warnf(log.BackTester, "invalid maximum total set to %f", m.MaximumTotal)
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
		c.CurrencySettings[i].ExchangeName = strings.ToLower(c.CurrencySettings[i].ExchangeName)
	}
	return nil
}
