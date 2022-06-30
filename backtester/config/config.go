package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
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

	fileData, err := os.ReadFile(path)
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
	if c.FundingSettings.UseExchangeLevelFunding && !c.StrategySettings.SimultaneousSignalProcessing {
		return errSimultaneousProcessingRequired
	}
	if len(c.FundingSettings.ExchangeLevelFunding) > 0 && !c.FundingSettings.UseExchangeLevelFunding {
		return errExchangeLevelFundingRequired
	}
	if c.FundingSettings.UseExchangeLevelFunding && len(c.FundingSettings.ExchangeLevelFunding) == 0 {
		return errExchangeLevelFundingDataRequired
	}
	if c.FundingSettings.UseExchangeLevelFunding {
		for i := range c.FundingSettings.ExchangeLevelFunding {
			if c.FundingSettings.ExchangeLevelFunding[i].InitialFunds.IsNegative() {
				return fmt.Errorf("%w for %v %v %v",
					errBadInitialFunds,
					c.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
					c.FundingSettings.ExchangeLevelFunding[i].Asset,
					c.FundingSettings.ExchangeLevelFunding[i].Currency,
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
	var hasFutures, hasSlippage bool
	for i := range c.CurrencySettings {
		if c.CurrencySettings[i].Asset == asset.PerpetualSwap ||
			c.CurrencySettings[i].Asset == asset.PerpetualContract {
			return errPerpetualsUnsupported
		}
		if c.CurrencySettings[i].Asset == asset.Futures &&
			(c.CurrencySettings[i].Quote.String() == "PERP" || c.CurrencySettings[i].Base.String() == "PI") {
			return errPerpetualsUnsupported
		}
		if c.CurrencySettings[i].Asset.IsFutures() {
			hasFutures = true
		}
		if c.CurrencySettings[i].SpotDetails != nil {
			if c.FundingSettings.UseExchangeLevelFunding {
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
		if c.CurrencySettings[i].Base.IsEmpty() {
			return errUnsetCurrency
		}
		if !c.CurrencySettings[i].Asset.IsValid() {
			return fmt.Errorf("%v %w", c.CurrencySettings[i].Asset, asset.ErrNotSupported)
		}
		if c.CurrencySettings[i].ExchangeName == "" {
			return errUnsetExchange
		}
		if !c.CurrencySettings[i].MinimumSlippagePercent.IsZero() ||
			!c.CurrencySettings[i].MaximumSlippagePercent.IsZero() {
			hasSlippage = true
		}
		if c.CurrencySettings[i].MinimumSlippagePercent.LessThan(decimal.Zero) ||
			c.CurrencySettings[i].MaximumSlippagePercent.LessThan(decimal.Zero) ||
			c.CurrencySettings[i].MinimumSlippagePercent.GreaterThan(c.CurrencySettings[i].MaximumSlippagePercent) {
			return errBadSlippageRates
		}
		c.CurrencySettings[i].ExchangeName = strings.ToLower(c.CurrencySettings[i].ExchangeName)
	}
	if hasSlippage && hasFutures {
		return fmt.Errorf("%w futures sizing currently incompatible with slippage", errFeatureIncompatible)
	}
	return nil
}

// PrintSetting prints relevant settings to the console for easy reading
func (c *Config) PrintSetting() {
	log.Info(common.Config, common.ColourH1+"------------------Backtester Settings------------------------"+common.ColourDefault)
	log.Info(common.Config, common.ColourH2+"------------------Strategy Settings--------------------------"+common.ColourDefault)
	log.Infof(common.Config, "Strategy: %s", c.StrategySettings.Name)
	if len(c.StrategySettings.CustomSettings) > 0 {
		log.Info(common.Config, "Custom strategy variables:")
		for k, v := range c.StrategySettings.CustomSettings {
			log.Infof(common.Config, "%s: %v", k, v)
		}
	} else {
		log.Info(common.Config, "Custom strategy variables: unset")
	}
	log.Infof(common.Config, "Simultaneous Signal Processing: %v", c.StrategySettings.SimultaneousSignalProcessing)
	log.Infof(common.Config, "USD value tracking: %v", !c.StrategySettings.DisableUSDTracking)

	if c.FundingSettings.UseExchangeLevelFunding && c.StrategySettings.SimultaneousSignalProcessing {
		log.Info(common.Config, common.ColourH2+"------------------Funding Settings---------------------------"+common.ColourDefault)
		log.Infof(common.Config, "Use Exchange Level Funding: %v", c.FundingSettings.UseExchangeLevelFunding)
		for i := range c.FundingSettings.ExchangeLevelFunding {
			log.Infof(common.Config, "Initial funds for %v %v %v: %v",
				c.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
				c.FundingSettings.ExchangeLevelFunding[i].Asset,
				c.FundingSettings.ExchangeLevelFunding[i].Currency,
				c.FundingSettings.ExchangeLevelFunding[i].InitialFunds.Round(8))
		}
	}

	for i := range c.CurrencySettings {
		currStr := fmt.Sprintf(common.ColourH2+"------------------%v %v-%v Currency Settings---------------------------------------------------------"+common.ColourDefault,
			c.CurrencySettings[i].Asset,
			c.CurrencySettings[i].Base,
			c.CurrencySettings[i].Quote)
		log.Infof(common.Config, currStr[:61])
		log.Infof(common.Config, "Exchange: %v", c.CurrencySettings[i].ExchangeName)
		if !c.FundingSettings.UseExchangeLevelFunding && c.CurrencySettings[i].SpotDetails != nil {
			if c.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil {
				log.Infof(common.Config, "Initial base funds: %v %v",
					c.CurrencySettings[i].SpotDetails.InitialBaseFunds.Round(8),
					c.CurrencySettings[i].Base)
			}
			if c.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil {
				log.Infof(common.Config, "Initial quote funds: %v %v",
					c.CurrencySettings[i].SpotDetails.InitialQuoteFunds.Round(8),
					c.CurrencySettings[i].Quote)
			}
		}
		if c.CurrencySettings[i].TakerFee != nil {
			if c.CurrencySettings[i].UsingExchangeTakerFee {
				log.Infof(common.Config, "Taker fee: Using Exchange's API default taker rate: %v", c.CurrencySettings[i].TakerFee.Round(8))
			} else {
				log.Infof(common.Config, "Taker fee: %v", c.CurrencySettings[i].TakerFee.Round(8))
			}
		}
		if c.CurrencySettings[i].MakerFee != nil {
			if c.CurrencySettings[i].UsingExchangeMakerFee {
				log.Infof(common.Config, "Maker fee: Using Exchange's API default maker rate: %v", c.CurrencySettings[i].MakerFee.Round(8))
			} else {
				log.Infof(common.Config, "Maker fee: %v", c.CurrencySettings[i].MakerFee.Round(8))
			}
		}
		log.Infof(common.Config, "Minimum slippage percent: %v", c.CurrencySettings[i].MinimumSlippagePercent.Round(8))
		log.Infof(common.Config, "Maximum slippage percent: %v", c.CurrencySettings[i].MaximumSlippagePercent.Round(8))
		log.Infof(common.Config, "Buy rules: %+v", c.CurrencySettings[i].BuySide)
		log.Infof(common.Config, "Sell rules: %+v", c.CurrencySettings[i].SellSide)
		if c.CurrencySettings[i].FuturesDetails != nil && c.CurrencySettings[i].Asset == asset.Futures {
			log.Infof(common.Config, "Leverage rules: %+v", c.CurrencySettings[i].FuturesDetails.Leverage)
		}
		log.Infof(common.Config, "Can use exchange defined order execution limits: %+v", c.CurrencySettings[i].CanUseExchangeLimits)
	}

	log.Info(common.Config, common.ColourH2+"------------------Portfolio Settings-------------------------"+common.ColourDefault)
	log.Infof(common.Config, "Buy rules: %+v", c.PortfolioSettings.BuySide)
	log.Infof(common.Config, "Sell rules: %+v", c.PortfolioSettings.SellSide)
	log.Infof(common.Config, "Leverage rules: %+v", c.PortfolioSettings.Leverage)
	if c.DataSettings.LiveData != nil {
		log.Info(common.Config, common.ColourH2+"------------------Live Settings------------------------------"+common.ColourDefault)
		log.Infof(common.Config, "Data type: %v", c.DataSettings.DataType)
		log.Infof(common.Config, "Interval: %v", c.DataSettings.Interval)
		log.Infof(common.Config, "REAL ORDERS: %v", c.DataSettings.LiveData.RealOrders)
		log.Infof(common.Config, "Overriding GCT API settings: %v", c.DataSettings.LiveData.APIClientIDOverride != "")
	}
	if c.DataSettings.APIData != nil {
		log.Info(common.Config, common.ColourH2+"------------------API Settings-------------------------------"+common.ColourDefault)
		log.Infof(common.Config, "Data type: %v", c.DataSettings.DataType)
		log.Infof(common.Config, "Interval: %v", c.DataSettings.Interval)
		log.Infof(common.Config, "Start date: %v", c.DataSettings.APIData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(common.Config, "End date: %v", c.DataSettings.APIData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	if c.DataSettings.CSVData != nil {
		log.Info(common.Config, common.ColourH2+"------------------CSV Settings-------------------------------"+common.ColourDefault)
		log.Infof(common.Config, "Data type: %v", c.DataSettings.DataType)
		log.Infof(common.Config, "Interval: %v", c.DataSettings.Interval)
		log.Infof(common.Config, "CSV file: %v", c.DataSettings.CSVData.FullPath)
	}
	if c.DataSettings.DatabaseData != nil {
		log.Info(common.Config, common.ColourH2+"------------------Database Settings--------------------------"+common.ColourDefault)
		log.Infof(common.Config, "Data type: %v", c.DataSettings.DataType)
		log.Infof(common.Config, "Interval: %v", c.DataSettings.Interval)
		log.Infof(common.Config, "Start date: %v", c.DataSettings.DatabaseData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(common.Config, "End date: %v", c.DataSettings.DatabaseData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
}
