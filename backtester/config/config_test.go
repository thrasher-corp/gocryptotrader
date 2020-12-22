package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	makerFee = 0.002
	takerFee = 0.001
)

// these are tests for experimentation more than anything
func TestGenerateDCACandleAPIStrat(t *testing.T) {
	cfg := Config{
		Nickname: "super fun run",
		StrategySettings: StrategySettings{
			Name: "dollarcostaverage",
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage:  false,
					MaximumLeverage: 102,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		APIData: &APIData{
			StartDate: time.Now().Add(-time.Hour * 24 * 365),
			EndDate:   time.Now(),
			Interval:  kline.OneDay.Duration(),
			DataType:  common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	p, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(filepath.Join(p, "examples", "dollar-cost-average.strat"), result, 0770)
	if err != nil {
		t.Error(err)
	}
}

// these are tests for experimentation more than anything
func TestGenerateDCAMultipleCurrencyAPICandleStrat(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateDCAMultipleCurrencyAPICandleStrat",
		StrategySettings: StrategySettings{
			Name: "dollarcostaverage",
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage:  false,
					MaximumLeverage: 102,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.ETH.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage:  false,
					MaximumLeverage: 102,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		APIData: &APIData{
			StartDate: time.Now().Add(-time.Hour * 24 * 7),
			EndDate:   time.Now(),
			Interval:  kline.OneHour.Duration(),
			DataType:  common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	p, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(filepath.Join(p, "examples", "dollar-cost-average-multiple-currencies.strat"), result, 0770)
	if err != nil {
		t.Error(err)
	}
}

// these are tests for experimentation more than anything
func TestGenerateDCAMultiCurrencyAssessmentAPICandleStrat(t *testing.T) {
	cfg := Config{
		Nickname: "hello!",
		StrategySettings: StrategySettings{
			Name:            "dollarcostaverage",
			IsMultiCurrency: true,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.0001,
					MaximumSize:  5,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.0001,
					MaximumSize:  5,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage:  false,
					MaximumLeverage: 102,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			/*{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.ETH.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage:  false,
					MaximumLeverage: 102,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},*/
		},
		APIData: &APIData{
			StartDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			Interval:  kline.OneHour.Duration(),
			DataType:  common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	p, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(filepath.Join(p, "examples", "dollar-cost-average-multi-currency-assessment.strat"), result, 0770)
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateDCALiveCandleStrat(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateDCALiveCandleStrat",
		StrategySettings: StrategySettings{
			Name: "dollarcostaverage",
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage:  false,
					MaximumLeverage: 102,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		LiveData: &LiveData{
			Interval: kline.OneMin.Duration(),
			DataType: common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	p, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(filepath.Join(p, "examples", "dollar-cost-average-live.strat"), result, 0770)
	if err != nil {
		t.Error(err)
	}
}

// these are tests for experimentation more than anything
func TestGenerateRSICandleAPICustomSettingsStrat(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateRSICandleAPICustomSettingsStrat",
		StrategySettings: StrategySettings{
			Name: "rsi420blazeit",
			CustomSettings: map[string]interface{}{
				"rsi-low":    31.0,
				"rsi-high":   69.0,
				"rsi-period": 12,
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 100000,
				BuySide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				SellSide: MinMax{
					MinimumSize:  0.1,
					MaximumSize:  1,
					MaximumTotal: 10000,
				},
				Leverage: Leverage{
					CanUseLeverage:  false,
					MaximumLeverage: 102,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		APIData: &APIData{
			StartDate: time.Date(2017, 5, 1, 0, 0, 0, 0, time.Local),
			EndDate:   time.Date(2018, 5, 1, 0, 0, 0, 0, time.Local),
			Interval:  kline.OneHour.Duration(),
			DataType:  common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	p, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(filepath.Join(p, "examples", "rsi420blazeit.strat"), result, 0770)
	if err != nil {
		t.Error(err)
	}
}
