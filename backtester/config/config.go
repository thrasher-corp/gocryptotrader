package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
	log.Infof(log.BackTester, "Use Exchange Level Funding: %v", c.StrategySettings.UseExchangeLevelFunding)
	log.Infof(log.BackTester, "USD value tracking: %v", !c.StrategySettings.DisableUSDTracking)
	if c.StrategySettings.UseExchangeLevelFunding && c.StrategySettings.SimultaneousSignalProcessing {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Funding Settings---------------------------")
		for i := range c.StrategySettings.ExchangeLevelFunding {
			log.Infof(log.BackTester, "Initial funds for %v %v %v: %v",
				c.StrategySettings.ExchangeLevelFunding[i].ExchangeName,
				c.StrategySettings.ExchangeLevelFunding[i].Asset,
				c.StrategySettings.ExchangeLevelFunding[i].Currency,
				c.StrategySettings.ExchangeLevelFunding[i].InitialFunds.Round(8))
		}
	}

	for i := range c.CurrencySettings {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		currStr := fmt.Sprintf("------------------%v %v-%v Currency Settings---------------------------------------------------------",
			c.CurrencySettings[i].Asset,
			c.CurrencySettings[i].Base,
			c.CurrencySettings[i].Quote)
		log.Infof(log.BackTester, currStr[:61])
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Exchange: %v", c.CurrencySettings[i].ExchangeName)
		if !c.StrategySettings.UseExchangeLevelFunding && c.CurrencySettings[i].SpotDetails != nil {
			if c.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil {
				log.Infof(log.BackTester, "Initial base funds: %v %v",
					c.CurrencySettings[i].SpotDetails.InitialBaseFunds.Round(8),
					c.CurrencySettings[i].Base)
			}
			if c.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil {
				log.Infof(log.BackTester, "Initial quote funds: %v %v",
					c.CurrencySettings[i].SpotDetails.InitialQuoteFunds.Round(8),
					c.CurrencySettings[i].Quote)
			}
		}
		log.Infof(log.BackTester, "Maker fee: %v", c.CurrencySettings[i].TakerFee.Round(8))
		log.Infof(log.BackTester, "Taker fee: %v", c.CurrencySettings[i].MakerFee.Round(8))
		log.Infof(log.BackTester, "Minimum slippage percent %v", c.CurrencySettings[i].MinimumSlippagePercent.Round(8))
		log.Infof(log.BackTester, "Maximum slippage percent: %v", c.CurrencySettings[i].MaximumSlippagePercent.Round(8))
		log.Infof(log.BackTester, "Buy rules: %+v", c.CurrencySettings[i].BuySide)
		log.Infof(log.BackTester, "Sell rules: %+v", c.CurrencySettings[i].SellSide)
		if c.CurrencySettings[i].FuturesDetails != nil && c.CurrencySettings[i].Asset == asset.Futures.String() {
			log.Infof(log.BackTester, "Leverage rules: %+v", c.CurrencySettings[i].FuturesDetails.Leverage)

		}
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

// Validate checks all config settings
func (c *Config) Validate() error {
	err := c.validateDate()
	if err != nil {
		return err
	}
	err = c.validateStrategySettings()
	if err != nil {
		return err
	}
	err = c.validateCurrencySettings()
	if err != nil {
		return err
	}
	return c.validateMinMaxes()
}

// validate ensures no one sets bad config values on purpose
func (m *MinMax) validate() error {
	if m.MaximumSize.IsNegative() {
		return fmt.Errorf("invalid maximum size %w", errSizeLessThanZero)
	}
	if m.MinimumSize.IsNegative() {
		return fmt.Errorf("invalid minimum size %w", errSizeLessThanZero)
	}
	if m.MaximumTotal.IsNegative() {
		return fmt.Errorf("invalid maximum total set to %w", errSizeLessThanZero)
	}
	if m.MaximumSize.LessThan(m.MinimumSize) && !m.MinimumSize.IsZero() && !m.MaximumSize.IsZero() {
		return fmt.Errorf("%w maximum size %v vs minimum size %v",
			errMaxSizeMinSizeMismatch,
			m.MaximumSize,
			m.MinimumSize)
	}
	if m.MaximumSize.Equal(m.MinimumSize) && !m.MinimumSize.IsZero() && !m.MaximumSize.IsZero() {
		return fmt.Errorf("%w %v",
			errMinMaxEqual,
			m.MinimumSize)
	}

	return nil
}

func (c *Config) validateMinMaxes() (err error) {
	for i := range c.CurrencySettings {
		err = c.CurrencySettings[i].BuySide.validate()
		if err != nil {
			return err
		}
		err = c.CurrencySettings[i].SellSide.validate()
		if err != nil {
			return err
		}
	}
	err = c.PortfolioSettings.BuySide.validate()
	if err != nil {
		return err
	}
	err = c.PortfolioSettings.SellSide.validate()
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) validateStrategySettings() error {
	if c.StrategySettings.UseExchangeLevelFunding && !c.StrategySettings.SimultaneousSignalProcessing {
		return errSimultaneousProcessingRequired
	}
	if len(c.StrategySettings.ExchangeLevelFunding) > 0 && !c.StrategySettings.UseExchangeLevelFunding {
		return errExchangeLevelFundingRequired
	}
	if c.StrategySettings.UseExchangeLevelFunding && len(c.StrategySettings.ExchangeLevelFunding) == 0 {
		return errExchangeLevelFundingDataRequired
	}
	if c.StrategySettings.UseExchangeLevelFunding {
		for i := range c.StrategySettings.ExchangeLevelFunding {
			if c.StrategySettings.ExchangeLevelFunding[i].InitialFunds.IsNegative() {
				return fmt.Errorf("%w for %v %v %v",
					errBadInitialFunds,
					c.StrategySettings.ExchangeLevelFunding[i].ExchangeName,
					c.StrategySettings.ExchangeLevelFunding[i].Asset,
					c.StrategySettings.ExchangeLevelFunding[i].Currency,
				)
			}
		}
	}
	strats := strategies.GetStrategies()
	for i := range strats {
		if strings.EqualFold(strats[i].Name(), c.StrategySettings.Name) {
			return nil
		}
	}

	return fmt.Errorf("strategty %v %w", c.StrategySettings.Name, base.ErrStrategyNotFound)
}

// validateDate checks whether someone has set a date poorly in their config
func (c *Config) validateDate() error {
	if c.DataSettings.DatabaseData != nil {
		if c.DataSettings.DatabaseData.StartDate.IsZero() ||
			c.DataSettings.DatabaseData.EndDate.IsZero() {
			return errStartEndUnset
		}
		if c.DataSettings.DatabaseData.StartDate.After(c.DataSettings.DatabaseData.EndDate) ||
			c.DataSettings.DatabaseData.StartDate.Equal(c.DataSettings.DatabaseData.EndDate) {
			return errBadDate
		}
	}
	if c.DataSettings.APIData != nil {
		if c.DataSettings.APIData.StartDate.IsZero() ||
			c.DataSettings.APIData.EndDate.IsZero() {
			return errStartEndUnset
		}
		if c.DataSettings.APIData.StartDate.After(c.DataSettings.APIData.EndDate) ||
			c.DataSettings.APIData.StartDate.Equal(c.DataSettings.APIData.EndDate) {
			return errBadDate
		}
	}
	return nil
}

// validateCurrencySettings checks whether someone has set invalid currency setting data in their config
func (c *Config) validateCurrencySettings() error {
	if len(c.CurrencySettings) == 0 {
		return errNoCurrencySettings
	}
	for i := range c.CurrencySettings {
		if c.CurrencySettings[i].Asset == asset.PerpetualSwap.String() ||
			c.CurrencySettings[i].Asset == asset.PerpetualContract.String() {
			return errPerpetualsUnsupported
		}
		if c.CurrencySettings[i].SpotDetails == nil && c.CurrencySettings[i].FuturesDetails == nil {
			return fmt.Errorf("%w please add spot or future currency details or create a new config via the config builder", errNoCurrencySettings)
		}
		if c.CurrencySettings[i].FuturesDetails != nil {
			if c.CurrencySettings[i].Quote == "PERP" || c.CurrencySettings[i].Base == "PI" {
				return errPerpetualsUnsupported
			}
		}
		if c.CurrencySettings[i].SpotDetails != nil {
			// if c.CurrencySettings[i].SpotDetails.InitialLegacyFunds > 0 {
			// 	// temporarily migrate legacy start config value
			// 	log.Warn(log.BackTester, "config field 'initial-funds' no longer supported, please use 'initial-quote-funds'")
			// 	log.Warnf(log.BackTester, "temporarily setting 'initial-quote-funds' to 'initial-funds' value of %v", c.CurrencySettings[i].InitialLegacyFunds)
			// 	iqf := decimal.NewFromFloat(c.CurrencySettings[i].SpotDetails.InitialLegacyFunds)
			// 	c.CurrencySettings[i].SpotDetails.InitialQuoteFunds = &iqf
			// }
			if c.StrategySettings.UseExchangeLevelFunding {
				if c.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil &&
					c.CurrencySettings[i].SpotDetails.InitialQuoteFunds.GreaterThan(decimal.Zero) {
					return fmt.Errorf("non-nil quote %w", errBadInitialFunds)
				}
				if c.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil &&
					c.CurrencySettings[i].SpotDetails.InitialBaseFunds.GreaterThan(decimal.Zero) {
					return fmt.Errorf("non-nil base %w", errBadInitialFunds)
				}
			} else {
				if c.CurrencySettings[i].SpotDetails.InitialQuoteFunds == nil &&
					c.CurrencySettings[i].SpotDetails.InitialBaseFunds == nil {
					return fmt.Errorf("nil base and quote %w", errBadInitialFunds)
				}
				if c.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil &&
					c.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil &&
					c.CurrencySettings[i].SpotDetails.InitialBaseFunds.IsZero() &&
					c.CurrencySettings[i].SpotDetails.InitialQuoteFunds.IsZero() {
					return fmt.Errorf("base or quote funds set to zero %w", errBadInitialFunds)
				}
				if c.CurrencySettings[i].SpotDetails.InitialQuoteFunds == nil {
					c.CurrencySettings[i].SpotDetails.InitialQuoteFunds = &decimal.Zero
				}
				if c.CurrencySettings[i].SpotDetails.InitialBaseFunds == nil {
					c.CurrencySettings[i].SpotDetails.InitialBaseFunds = &decimal.Zero
				}
			}
		}
		if c.CurrencySettings[i].Base == "" {
			return errUnsetCurrency
		}
		if c.CurrencySettings[i].Asset == "" {
			return errUnsetAsset
		}
		if c.CurrencySettings[i].ExchangeName == "" {
			return errUnsetExchange
		}
		if c.CurrencySettings[i].MinimumSlippagePercent.LessThan(decimal.Zero) ||
			c.CurrencySettings[i].MaximumSlippagePercent.LessThan(decimal.Zero) ||
			c.CurrencySettings[i].MinimumSlippagePercent.GreaterThan(c.CurrencySettings[i].MaximumSlippagePercent) {
			return errBadSlippageRates
		}
		c.CurrencySettings[i].ExchangeName = strings.ToLower(c.CurrencySettings[i].ExchangeName)
	}
	return nil
}
