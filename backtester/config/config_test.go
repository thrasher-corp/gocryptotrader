package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	makerFee     = 0.001
	takerFee     = 0.002
	testExchange = "binance"
	dca          = "dollarcostaverage"
	// change this if you modify a config and want it to save to the example folder
	saveConfig = false
)

var (
	startDate time.Time
	endDate   time.Time
)

func TestMain(m *testing.M) {
	startDate = time.Date(time.Now().Year()-1, 11, 1, 0, 0, 0, 0, time.Local)
	endDate = time.Date(time.Now().Year()-1, 12, 1, 0, 0, 0, 0, time.Local)
	os.Exit(m.Run())
}

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
	passFile, err = ioutil.TempFile(tempDir, "*.start")
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

func TestPrintSettings(t *testing.T) {
	cfg := Config{
		Nickname: "super fun run",
		Goal:     "To demonstrate rendering of settings",
		StrategySettings: StrategySettings{
			Name: dca,
			CustomSettings: map[string]interface{}{
				"dca-dummy1": 30.0,
				"dca-dummy2": 30.0,
				"dca-dummy3": 30.0,
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneMin.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate,
				InclusiveEndDate: true,
			},
			CSVData: &CSVData{
				FullPath: "fake",
			},
			LiveData: &LiveData{
				APIKeyOverride:        "",
				APISecretOverride:     "",
				APIClientIDOverride:   "",
				API2FAOverride:        "",
				APISubaccountOverride: "",
				RealOrders:            false,
			},
			DatabaseData: &DatabaseData{
				StartDate:        startDate,
				EndDate:          endDate,
				ConfigOverride:   nil,
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	cfg.PrintSetting()
}

func TestGenerateConfigForDCAAPICandles(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPICandles",
		Goal:     "To demonstrate DCA strategy using API candles",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate,
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPITrades(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPITrades",
		Goal:     "To demonstrate running the DCA strategy using API trade data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.TradeStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate,
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-trades.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPICandlesMultipleCurrencies(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPICandlesMultipleCurrencies",
		Goal:     "To demonstrate running the DCA strategy using the API against multiple currencies candle data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate,
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-candles-multiple-currencies.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPICandlesSimultaneousProcessing(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCAAPICandlesSimultaneousProcessing",
		Goal:     "To demonstrate how simultaneous processing can work",
		StrategySettings: StrategySettings{
			Name:                         dca,
			SimultaneousSignalProcessing: true,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate,
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-candles-simultaneous-processing.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCALiveCandles(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCALiveCandles",
		Goal:     "To demonstrate live trading proof of concept against candle data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneHour.Duration(),
			DataType: common.CandleStr,
			LiveData: &LiveData{
				APIKeyOverride:        "",
				APISecretOverride:     "",
				APIClientIDOverride:   "",
				API2FAOverride:        "",
				APISubaccountOverride: "",
				RealOrders:            false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-candles-live.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForRSIAPICustomSettings(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateRSICandleAPICustomSettingsStrat",
		Goal:     "To demonstrate the RSI strategy using API candle data and custom settings",
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
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate,
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "rsi-api-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCACSVCandles(t *testing.T) {
	fp := filepath.Join("..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv")
	cfg := Config{
		Nickname: "TestGenerateConfigForDCACSVCandles",
		Goal:     "To demonstrate the DCA strategy using CSV candle data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			CSVData: &CSVData{
				FullPath: fp,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-csv-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCACSVTrades(t *testing.T) {
	fp := filepath.Join("..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv")
	cfg := Config{
		Nickname: "TestGenerateConfigForDCACSVTrades",
		Goal:     "To demonstrate the DCA strategy using CSV trade data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneMin.Duration(),
			DataType: common.TradeStr,
			CSVData: &CSVData{
				FullPath: fp,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-csv-trades.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCADatabaseCandles(t *testing.T) {
	cfg := Config{
		Nickname: "TestGenerateConfigForDCADatabaseCandles",
		Goal:     "To demonstrate the DCA strategy using database candle data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
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
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			DatabaseData: &DatabaseData{
				StartDate: startDate,
				EndDate:   endDate,
				ConfigOverride: &database.Config{
					Enabled: true,
					Verbose: false,
					Driver:  "sqlite",
					ConnectionDetails: drivers.ConnectionDetails{
						Host:     "localhost",
						Database: "testsqlite.db",
					},
				},
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
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
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: 0.03,
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Error(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-database-candles.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestValidate(t *testing.T) {
	m := MinMax{
		MinimumSize:  -1,
		MaximumSize:  -1,
		MaximumTotal: -1,
	}
	m.Validate()
	if m.MinimumSize > m.MaximumSize {
		t.Errorf("expected %v > %v", m.MaximumSize, m.MinimumSize)
	}
	if m.MinimumSize < 0 {
		t.Errorf("expected %v > %v", m.MinimumSize, 0)
	}
	if m.MaximumSize < 0 {
		t.Errorf("expected %v > %v", m.MaximumSize, 0)
	}
	if m.MaximumTotal < 0 {
		t.Errorf("expected %v > %v", m.MaximumTotal, 0)
	}
}

func TestValidateDate(t *testing.T) {
	c := Config{}
	err := c.ValidateDate()
	if err != nil {
		t.Error(err)
	}
	c.DataSettings = DataSettings{
		DatabaseData: &DatabaseData{},
	}
	err = c.ValidateDate()
	if !errors.Is(ErrStartEndUnset, err) {
		t.Errorf("expected %v, received %v", ErrStartEndUnset, err)
	}
	c.DataSettings.DatabaseData.StartDate = time.Now()
	c.DataSettings.DatabaseData.EndDate = c.DataSettings.DatabaseData.StartDate
	err = c.ValidateDate()
	if !errors.Is(ErrBadDate, err) {
		t.Errorf("expected %v, received %v", ErrBadDate, err)
	}
	c.DataSettings.DatabaseData.EndDate = c.DataSettings.DatabaseData.StartDate.Add(time.Minute)
	err = c.ValidateDate()
	if err != nil {
		t.Error(err)
	}
	c.DataSettings.APIData = &APIData{}
	err = c.ValidateDate()
	if !errors.Is(ErrStartEndUnset, err) {
		t.Errorf("expected %v, received %v", ErrStartEndUnset, err)
	}
	c.DataSettings.APIData.StartDate = time.Now()
	c.DataSettings.APIData.EndDate = c.DataSettings.APIData.StartDate
	err = c.ValidateDate()
	if !errors.Is(ErrBadDate, err) {
		t.Errorf("expected %v, received %v", ErrBadDate, err)
	}
	c.DataSettings.APIData.EndDate = c.DataSettings.APIData.StartDate.Add(time.Minute)
	err = c.ValidateDate()
	if err != nil {
		t.Error(err)
	}
}

func TestValidateCurrencySettings(t *testing.T) {
	c := Config{}
	err := c.ValidateCurrencySettings()
	if !errors.Is(ErrNoCurrencySettings, err) {
		t.Errorf("expected %v, received %v", ErrNoCurrencySettings, err)
	}
	c.CurrencySettings = append(c.CurrencySettings, CurrencySettings{})
	err = c.ValidateCurrencySettings()
	if !errors.Is(ErrBadInitialFunds, err) {
		t.Errorf("expected %v, received %v", ErrBadInitialFunds, err)
	}
	c.CurrencySettings[0].InitialFunds = 1337
	err = c.ValidateCurrencySettings()
	if !errors.Is(ErrUnsetCurrency, err) {
		t.Errorf("expected %v, received %v", ErrUnsetCurrency, err)
	}
	c.CurrencySettings[0].Base = "lol"
	err = c.ValidateCurrencySettings()
	if !errors.Is(ErrUnsetAsset, err) {
		t.Errorf("expected %v, received %v", ErrUnsetAsset, err)
	}
	c.CurrencySettings[0].Asset = "lol"
	err = c.ValidateCurrencySettings()
	if !errors.Is(ErrUnsetExchange, err) {
		t.Errorf("expected %v, received %v", ErrUnsetExchange, err)
	}
	c.CurrencySettings[0].ExchangeName = "lol"
	err = c.ValidateCurrencySettings()
	if err != nil {
		t.Error(err)
	}
	c.CurrencySettings[0].MinimumSlippagePercent = -1.0
	err = c.ValidateCurrencySettings()
	if !errors.Is(ErrBadSlippageRates, err) {
		t.Errorf("expected %v, received %v", ErrBadSlippageRates, err)
	}
	c.CurrencySettings[0].MinimumSlippagePercent = 2.0
	c.CurrencySettings[0].MaximumSlippagePercent = -1.0
	err = c.ValidateCurrencySettings()
	if !errors.Is(ErrBadSlippageRates, err) {
		t.Errorf("expected %v, received %v", ErrBadSlippageRates, err)
	}
	c.CurrencySettings[0].MinimumSlippagePercent = 2.0
	c.CurrencySettings[0].MaximumSlippagePercent = 1.0
	err = c.ValidateCurrencySettings()
	if !errors.Is(ErrBadSlippageRates, err) {
		t.Errorf("expected %v, received %v", ErrBadSlippageRates, err)
	}
}
