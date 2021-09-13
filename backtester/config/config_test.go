package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/top2bottom2"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	testExchange = "binance"
	dca          = "dollarcostaverage"
	// change this if you modify a config and want it to save to the example folder
	saveConfig = true
)

var (
	startDate    = time.Date(time.Now().Year()-1, 8, 1, 0, 0, 0, 0, time.Local)
	endDate      = time.Date(time.Now().Year()-1, 12, 1, 0, 0, 0, 0, time.Local)
	tradeEndDate = startDate.Add(time.Hour * 72)
	makerFee     = decimal.NewFromFloat(0.001)
	takerFee     = decimal.NewFromFloat(0.002)
	minMax       = MinMax{
		MinimumSize:  decimal.NewFromFloat(0.1),
		MaximumSize:  decimal.NewFromInt(1),
		MaximumTotal: decimal.NewFromInt(10000),
	}
	initialFunds1 = decimal.NewFromInt(1000000)
	initialFunds2 = decimal.NewFromInt(100000)
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
				InitialFunds: initialFunds1,
				BuySide:      minMax,
				SellSide:     minMax,
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
				APISubAccountOverride: "",
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	cfg.PrintSetting()
}

func TestGenerateConfigForDCAAPICandles(t *testing.T) {
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPICandles",
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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

func TestGenerateConfigForDCAAPICandlesExchangeLevelFunding(t *testing.T) {
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPICandlesExchangeLevelFunding",
		Goal:     "To demonstrate DCA strategy using API candles using a shared pool of funds",
		StrategySettings: StrategySettings{
			Name:                         dca,
			SimultaneousSignalProcessing: true,
			UseExchangeLevelFunding:      true,
			ExchangeLevelFunding: []ExchangeLevelFunding{
				{
					ExchangeName: testExchange,
					Asset:        asset.Spot.String(),
					Currency:     currency.USDT.String(),
					InitialFunds: decimal.NewFromInt(100000),
				},
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.ETH.String(),
				Quote:        currency.USDT.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		err = ioutil.WriteFile(filepath.Join(p, "examples", "dca-api-candles-exchange-level-funding.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPITrades(t *testing.T) {
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPITrades",
		Goal:     "To demonstrate running the DCA strategy using API trade data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "ftx",
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage: Leverage{
					CanUseLeverage: false,
				},
				MakerFee: makerFee,
				TakerFee: takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneHour.Duration(),
			DataType: common.TradeStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          tradeEndDate,
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide: MinMax{
				MinimumSize:  decimal.NewFromFloat(0.1),
				MaximumSize:  decimal.NewFromInt(1),
				MaximumTotal: decimal.NewFromInt(10000),
			},
			SellSide: MinMax{
				MinimumSize:  decimal.NewFromFloat(0.1),
				MaximumSize:  decimal.NewFromInt(1),
				MaximumTotal: decimal.NewFromInt(10000),
			},
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		Nickname: "ExampleStrategyDCAAPICandlesMultipleCurrencies",
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		Nickname: "ExampleStrategyDCAAPICandlesSimultaneousProcessing",
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
				InitialFunds: initialFunds1,
				BuySide:      minMax,
				SellSide:     minMax,
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		Nickname: "ExampleStrategyDCALiveCandles",
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
			LiveData: &LiveData{
				APIKeyOverride:        "",
				APISecretOverride:     "",
				APIClientIDOverride:   "",
				API2FAOverride:        "",
				APISubAccountOverride: "",
				RealOrders:            false,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
				InitialFunds: initialFunds1,
				BuySide:      minMax,
				SellSide:     minMax,
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		Nickname: "ExampleStrategyDCACSVCandles",
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		Nickname: "ExampleStrategyDCACSVTrades",
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
				InitialFunds: initialFunds2,
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
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		Nickname: "ExampleStrategyDCADatabaseCandles",
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
				InitialFunds: initialFunds2,
				BuySide:      minMax,
				SellSide:     minMax,
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
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{
				CanUseLeverage: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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

func TestGenerateConfigForTop2Bottom2(t *testing.T) {
	cfg := Config{
		Nickname: "ExampleStrategyTop2Bottom2",
		Goal:     "To demonstrate complex strategy using exchange level funding and simultaneous processing of data signals",
		StrategySettings: StrategySettings{
			Name:                         top2bottom2.Name,
			UseExchangeLevelFunding:      true,
			SimultaneousSignalProcessing: true,
			ExchangeLevelFunding: []ExchangeLevelFunding{
				{
					ExchangeName: testExchange,
					Asset:        asset.Spot.String(),
					Currency:     currency.BTC.String(),
					InitialFunds: decimal.NewFromFloat(3),
				},
				{
					ExchangeName: testExchange,
					Asset:        asset.Spot.String(),
					Currency:     currency.USDT.String(),
					InitialFunds: decimal.NewFromInt(10000),
				},
			},
			CustomSettings: map[string]interface{}{
				"mfi-low":    32,
				"mfi-high":   68,
				"mfi-period": 14,
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BTC.String(),
				Quote:        currency.USDT.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.DOGE.String(),
				Quote:        currency.USDT.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.ETH.String(),
				Quote:        currency.BTC.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.LTC.String(),
				Quote:        currency.BTC.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.XRP.String(),
				Quote:        currency.USDT.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
			},
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot.String(),
				Base:         currency.BNB.String(),
				Quote:        currency.BTC.String(),
				BuySide:      minMax,
				SellSide:     minMax,
				Leverage:     Leverage{},
				MakerFee:     makerFee,
				TakerFee:     takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay.Duration(),
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide:  minMax,
			SellSide: minMax,
			Leverage: Leverage{},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
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
		err = ioutil.WriteFile(filepath.Join(p, "examples", "t2b2-api-candles-exchange-funding.strat"), result, 0770)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestValidate(t *testing.T) {
	m := MinMax{
		MinimumSize:  decimal.NewFromInt(-1),
		MaximumSize:  decimal.NewFromInt(-1),
		MaximumTotal: decimal.NewFromInt(-1),
	}
	err := m.validate()
	if err == nil {
		t.Error("expected error")
	}
	m.MinimumSize = decimal.Zero
	err = m.validate()
	if err == nil {
		t.Error("expected error")
	}
	m.MaximumSize = decimal.Zero
	err = m.validate()
	if err == nil {
		t.Error("expected error")
	}
	m.MaximumTotal = decimal.Zero
	err = m.validate()
	if err != nil {
		t.Error(err)
	}
}

func TestValidateDate(t *testing.T) {
	c := Config{}
	err := c.validateDate()
	if err != nil {
		t.Error(err)
	}
	c.DataSettings = DataSettings{
		DatabaseData: &DatabaseData{},
	}
	err = c.validateDate()
	if !errors.Is(ErrStartEndUnset, err) {
		t.Errorf("received: %v, expected: %v", err, ErrStartEndUnset)
	}
	c.DataSettings.DatabaseData.StartDate = time.Now()
	c.DataSettings.DatabaseData.EndDate = c.DataSettings.DatabaseData.StartDate
	err = c.validateDate()
	if !errors.Is(ErrBadDate, err) {
		t.Errorf("received: %v, expected: %v", err, ErrBadDate)
	}
	c.DataSettings.DatabaseData.EndDate = c.DataSettings.DatabaseData.StartDate.Add(time.Minute)
	err = c.validateDate()
	if err != nil {
		t.Error(err)
	}
	c.DataSettings.APIData = &APIData{}
	err = c.validateDate()
	if !errors.Is(ErrStartEndUnset, err) {
		t.Errorf("received: %v, expected: %v", err, ErrStartEndUnset)
	}
	c.DataSettings.APIData.StartDate = time.Now()
	c.DataSettings.APIData.EndDate = c.DataSettings.APIData.StartDate
	err = c.validateDate()
	if !errors.Is(ErrBadDate, err) {
		t.Errorf("received: %v, expected: %v", err, ErrBadDate)
	}
	c.DataSettings.APIData.EndDate = c.DataSettings.APIData.StartDate.Add(time.Minute)
	err = c.validateDate()
	if err != nil {
		t.Error(err)
	}
}

func TestValidateCurrencySettings(t *testing.T) {
	c := Config{}
	err := c.validateCurrencySettings()
	if !errors.Is(ErrNoCurrencySettings, err) {
		t.Errorf("received: %v, expected: %v", err, ErrNoCurrencySettings)
	}
	c.CurrencySettings = append(c.CurrencySettings, CurrencySettings{})
	err = c.validateCurrencySettings()
	if !errors.Is(ErrBadInitialFunds, err) {
		t.Errorf("received: %v, expected: %v", err, ErrBadInitialFunds)
	}
	leet := decimal.NewFromInt(1337)
	c.CurrencySettings[0].InitialFunds = leet
	err = c.validateCurrencySettings()
	if !errors.Is(ErrUnsetCurrency, err) {
		t.Errorf("received: %v, expected: %v", err, ErrUnsetCurrency)
	}
	c.CurrencySettings[0].Base = "lol"
	err = c.validateCurrencySettings()
	if !errors.Is(ErrUnsetAsset, err) {
		t.Errorf("received: %v, expected: %v", err, ErrUnsetAsset)
	}
	c.CurrencySettings[0].Asset = "lol"
	err = c.validateCurrencySettings()
	if !errors.Is(ErrUnsetExchange, err) {
		t.Errorf("received: %v, expected: %v", err, ErrUnsetExchange)
	}
	c.CurrencySettings[0].ExchangeName = "lol"
	err = c.validateCurrencySettings()
	if err != nil {
		t.Error(err)
	}
	c.CurrencySettings[0].MinimumSlippagePercent = decimal.NewFromInt(-1)
	err = c.validateCurrencySettings()
	if !errors.Is(ErrBadSlippageRates, err) {
		t.Errorf("received: %v, expected: %v", err, ErrBadSlippageRates)
	}
	c.CurrencySettings[0].MinimumSlippagePercent = decimal.NewFromInt(2)
	c.CurrencySettings[0].MaximumSlippagePercent = decimal.NewFromInt(-1)
	err = c.validateCurrencySettings()
	if !errors.Is(ErrBadSlippageRates, err) {
		t.Errorf("received: %v, expected: %v", err, ErrBadSlippageRates)
	}
	c.CurrencySettings[0].MinimumSlippagePercent = decimal.NewFromInt(2)
	c.CurrencySettings[0].MaximumSlippagePercent = decimal.NewFromInt(1)
	err = c.validateCurrencySettings()
	if !errors.Is(ErrBadSlippageRates, err) {
		t.Errorf("received: %v, expected: %v", err, ErrBadSlippageRates)
	}
}
