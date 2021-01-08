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

func TestLoadConfig(t *testing.T) {
	_, err := LoadConfig([]byte(`{}`))
	if err != nil {
		t.Error(err)
	}
}

func TestReadConfigFromFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Problem creating temp dir at %s: %s\n", tempDir, err)
	}
	defer func() {
		err = os.RemoveAll(tempDir)
		if err != nil {
			t.Error(err)
		}
	}()
	var passFile *os.File
	passFile, err = ioutil.TempFile(tempDir, "*.strat")
	if err != nil {
		t.Fatalf("Problem creating temp file at %v: %s\n", passFile, err)
	}
	_, err = passFile.WriteString("{}")
	if err != nil {
		t.Error(err)
	}
	err = passFile.Close()
	if err != nil {
		t.Error(err)
	}
	_, err = ReadConfigFromFile(passFile.Name())
	if err != nil {
		t.Error(err)
	}
}

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
			MaximumHoldingsRatio: 0,
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
			MaximumHoldingsRatio: 0,
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
				InitialFunds: 1000000,
				BuySide: MinMax{
					MinimumSize:  0,
					MaximumSize:  0,
					MaximumTotal: 1000,
				},
				SellSide: MinMax{
					MinimumSize:  0,
					MaximumSize:  0,
					MaximumTotal: 1000,
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
			StartDate: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2020, 5, 20, 0, 0, 0, 0, time.UTC),
			Interval:  kline.OneHour.Duration(),
			DataType:  common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			MaximumHoldingsRatio: 0,
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
			MaximumHoldingsRatio: 0,
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
			Name: "rsi",
			CustomSettings: map[string]interface{}{
				"rsi-low":    30.0,
				"rsi-high":   70.0,
				"rsi-period": 14,
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "binance",
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: 1000000,
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
			StartDate: time.Date(2018, 5, 1, 0, 0, 0, 0, time.Local),
			EndDate:   time.Date(2020, 5, 1, 0, 0, 0, 0, time.Local),
			Interval:  kline.OneDay.Duration(),
			DataType:  common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			MaximumHoldingsRatio: 0,
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
	err = ioutil.WriteFile(filepath.Join(p, "examples", "rsi.strat"), result, 0770)
	if err != nil {
		t.Error(err)
	}
}
