package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/top2bottom2"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	mainExchange = "binance"
	dca          = "dollarcostaverage"
	// change this if you modify a config and want it to save to the example folder
	saveConfig = false
)

var (
	startDate    = time.Date(time.Now().Year()-1, 8, 1, 0, 0, 0, 0, time.Local)
	endDate      = time.Date(time.Now().Year()-1, 12, 1, 0, 0, 0, 0, time.Local)
	tradeEndDate = startDate.Add(time.Hour * 72)
	makerFee     = decimal.NewFromFloat(0.0002)
	takerFee     = decimal.NewFromFloat(0.0007)
	minMax       = MinMax{
		MinimumSize:  decimal.NewFromFloat(0.005),
		MaximumSize:  decimal.NewFromInt(2),
		MaximumTotal: decimal.NewFromInt(40000),
	}
	// strictMinMax used for live order restrictions
	strictMinMax = MinMax{
		MinimumSize:  decimal.NewFromFloat(0.001),
		MaximumSize:  decimal.NewFromFloat(0.05),
		MaximumTotal: decimal.NewFromInt(100),
	}
	initialFunds1000000 *decimal.Decimal
	initialFunds100000  *decimal.Decimal
	initialFunds10      *decimal.Decimal

	mainCurrencyPair = currency.NewBTCUSDT()
)

func TestMain(m *testing.M) {
	iF1 := decimal.NewFromInt(1000000)
	iF2 := decimal.NewFromInt(100000)
	iBF := decimal.NewFromInt(10)
	initialFunds1000000 = &iF1
	initialFunds100000 = &iF2
	initialFunds10 = &iBF
	os.Exit(m.Run())
}

func TestValidateDate(t *testing.T) {
	t.Parallel()
	c := Config{}
	err := c.validateDate()
	assert.NoError(t, err)

	c.DataSettings = DataSettings{
		DatabaseData: &DatabaseData{},
	}
	err = c.validateDate()
	assert.ErrorIs(t, err, gctcommon.ErrDateUnset)

	c.DataSettings.DatabaseData.StartDate = time.Now()
	c.DataSettings.DatabaseData.EndDate = c.DataSettings.DatabaseData.StartDate
	err = c.validateDate()
	assert.ErrorIs(t, err, gctcommon.ErrStartEqualsEnd)

	c.DataSettings.DatabaseData.EndDate = c.DataSettings.DatabaseData.StartDate.Add(time.Minute)
	err = c.validateDate()
	assert.NoError(t, err)

	c.DataSettings.APIData = &APIData{}
	err = c.validateDate()
	assert.ErrorIs(t, err, gctcommon.ErrDateUnset)

	c.DataSettings.APIData.StartDate = time.Now()
	c.DataSettings.APIData.EndDate = c.DataSettings.APIData.StartDate
	err = c.validateDate()
	assert.ErrorIs(t, err, gctcommon.ErrStartEqualsEnd)

	c.DataSettings.APIData.EndDate = c.DataSettings.APIData.StartDate.Add(time.Minute)
	err = c.validateDate()
	assert.NoError(t, err)
}

func TestValidateCurrencySettings(t *testing.T) {
	t.Parallel()
	c := Config{}
	err := c.validateCurrencySettings()
	assert.ErrorIs(t, err, errNoCurrencySettings)

	c.CurrencySettings = append(c.CurrencySettings, CurrencySettings{})
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errUnsetCurrency)

	leet := decimal.NewFromInt(1337)
	c.CurrencySettings[0].SpotDetails = &SpotDetails{InitialQuoteFunds: &leet}
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errUnsetCurrency)

	c.CurrencySettings[0].Base = currency.NewCode("lol")
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	c.CurrencySettings[0].Asset = asset.Spot
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errUnsetExchange)

	c.CurrencySettings[0].ExchangeName = "lol"
	err = c.validateCurrencySettings()
	assert.NoError(t, err)

	c.CurrencySettings[0].Asset = asset.PerpetualSwap
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errPerpetualsUnsupported)

	c.CurrencySettings[0].Asset = asset.USDTMarginedFutures
	c.CurrencySettings[0].Quote = currency.NewCode("PERP")
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errPerpetualsUnsupported)

	c.CurrencySettings[0].MinimumSlippagePercent = decimal.NewFromInt(2)
	c.CurrencySettings[0].MaximumSlippagePercent = decimal.NewFromInt(3)
	c.CurrencySettings[0].Quote = currency.NewCode("USD")
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errFeatureIncompatible)

	c.CurrencySettings[0].Asset = asset.Spot
	c.CurrencySettings[0].MinimumSlippagePercent = decimal.NewFromInt(-1)
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errBadSlippageRates)

	c.CurrencySettings[0].MinimumSlippagePercent = decimal.NewFromInt(2)
	c.CurrencySettings[0].MaximumSlippagePercent = decimal.NewFromInt(-1)
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errBadSlippageRates)

	c.CurrencySettings[0].MinimumSlippagePercent = decimal.NewFromInt(2)
	c.CurrencySettings[0].MaximumSlippagePercent = decimal.NewFromInt(1)
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errBadSlippageRates)

	c.CurrencySettings[0].SpotDetails = &SpotDetails{}
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errBadInitialFunds)

	z := decimal.Zero
	c.CurrencySettings[0].SpotDetails.InitialQuoteFunds = &z
	c.CurrencySettings[0].SpotDetails.InitialBaseFunds = &z
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errBadInitialFunds)

	c.CurrencySettings[0].SpotDetails.InitialQuoteFunds = &leet
	c.FundingSettings.UseExchangeLevelFunding = true
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errBadInitialFunds)

	c.CurrencySettings[0].SpotDetails.InitialQuoteFunds = &z
	c.CurrencySettings[0].SpotDetails.InitialBaseFunds = &leet
	c.FundingSettings.UseExchangeLevelFunding = true
	err = c.validateCurrencySettings()
	assert.ErrorIs(t, err, errBadInitialFunds)
}

func TestValidateMinMaxes(t *testing.T) {
	t.Parallel()
	c := &Config{}
	err := c.validateMinMaxes()
	assert.NoError(t, err)

	c.CurrencySettings = []CurrencySettings{
		{
			SellSide: MinMax{
				MinimumSize: decimal.NewFromInt(-1),
			},
		},
	}
	err = c.validateMinMaxes()
	assert.ErrorIs(t, err, errSizeLessThanZero)

	c.CurrencySettings = []CurrencySettings{
		{
			SellSide: MinMax{
				MaximumTotal: decimal.NewFromInt(-1),
			},
		},
	}
	err = c.validateMinMaxes()
	assert.ErrorIs(t, err, errSizeLessThanZero)

	c.CurrencySettings = []CurrencySettings{
		{
			SellSide: MinMax{
				MaximumSize: decimal.NewFromInt(-1),
			},
		},
	}
	err = c.validateMinMaxes()
	assert.ErrorIs(t, err, errSizeLessThanZero)

	c.CurrencySettings = []CurrencySettings{
		{
			BuySide: MinMax{
				MinimumSize:  decimal.NewFromInt(2),
				MaximumTotal: decimal.NewFromInt(10),
				MaximumSize:  decimal.NewFromInt(1),
			},
		},
	}
	err = c.validateMinMaxes()
	assert.ErrorIs(t, err, errMaxSizeMinSizeMismatch)

	c.CurrencySettings = []CurrencySettings{
		{
			BuySide: MinMax{
				MinimumSize: decimal.NewFromInt(2),
				MaximumSize: decimal.NewFromInt(2),
			},
		},
	}
	err = c.validateMinMaxes()
	assert.ErrorIs(t, err, errMinMaxEqual)

	c.CurrencySettings = []CurrencySettings{
		{
			BuySide: MinMax{
				MinimumSize:  decimal.NewFromInt(1),
				MaximumTotal: decimal.NewFromInt(10),
				MaximumSize:  decimal.NewFromInt(2),
			},
		},
	}
	c.PortfolioSettings = PortfolioSettings{
		BuySide: MinMax{
			MinimumSize: decimal.NewFromInt(-1),
		},
	}
	err = c.validateMinMaxes()
	assert.ErrorIs(t, err, errSizeLessThanZero)

	c.PortfolioSettings = PortfolioSettings{
		SellSide: MinMax{
			MinimumSize: decimal.NewFromInt(-1),
		},
	}
	err = c.validateMinMaxes()
	assert.ErrorIs(t, err, errSizeLessThanZero)
}

func TestValidateStrategySettings(t *testing.T) {
	t.Parallel()
	c := &Config{}
	err := c.validateStrategySettings()
	assert.ErrorIs(t, err, base.ErrStrategyNotFound)

	c.StrategySettings = StrategySettings{Name: dca}
	err = c.validateStrategySettings()
	assert.NoError(t, err)

	c.StrategySettings.SimultaneousSignalProcessing = true
	err = c.validateStrategySettings()
	assert.NoError(t, err)

	c.FundingSettings = FundingSettings{}
	c.FundingSettings.UseExchangeLevelFunding = true
	err = c.validateStrategySettings()
	assert.ErrorIs(t, err, errExchangeLevelFundingDataRequired)

	c.FundingSettings.ExchangeLevelFunding = []ExchangeLevelFunding{
		{
			InitialFunds: decimal.NewFromInt(-1),
		},
	}
	err = c.validateStrategySettings()
	assert.ErrorIs(t, err, errBadInitialFunds)

	c.StrategySettings.SimultaneousSignalProcessing = false
	err = c.validateStrategySettings()
	assert.ErrorIs(t, err, errSimultaneousProcessingRequired)

	c.FundingSettings.UseExchangeLevelFunding = false
	err = c.validateStrategySettings()
	assert.ErrorIs(t, err, errExchangeLevelFundingRequired)
}

func TestPrintSettings(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Nickname: "super fun run",
		Goal:     "To demonstrate rendering of settings",
		StrategySettings: StrategySettings{
			Name: dca,
			CustomSettings: map[string]any{
				"dca-dummy1": 30.0,
				"dca-dummy2": 30.0,
				"dca-dummy3": 30.0,
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds1000000,
					InitialBaseFunds:  initialFunds1000000,
				},
				BuySide:        minMax,
				SellSide:       minMax,
				MakerFee:       &makerFee,
				TakerFee:       &takerFee,
				FuturesDetails: &FuturesDetails{},
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneMin,
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate,
				InclusiveEndDate: true,
			},
			CSVData: &CSVData{
				FullPath: "fake",
			},
			LiveData: &LiveData{},
			DatabaseData: &DatabaseData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide:  minMax,
			SellSide: minMax,
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	cfg.PrintSetting()
	cfg.FundingSettings = FundingSettings{
		UseExchangeLevelFunding: true,
		ExchangeLevelFunding:    []ExchangeLevelFunding{{}},
	}
	cfg.PrintSetting()
}

func TestValidate(t *testing.T) {
	t.Parallel()
	c := &Config{
		StrategySettings: StrategySettings{Name: dca},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialBaseFunds:  initialFunds10,
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide: MinMax{
					MinimumSize:  decimal.NewFromInt(1),
					MaximumSize:  decimal.NewFromInt(10),
					MaximumTotal: decimal.NewFromInt(10),
				},
			},
		},
	}
	err := c.Validate()
	assert.NoError(t, err)

	c = nil
	err = c.Validate()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestReadStrategyConfigFromFile(t *testing.T) {
	tempDir := t.TempDir()
	passFile, err := os.CreateTemp(tempDir, "*.start")
	if err != nil {
		t.Fatalf("Problem creating temp file at %v: %s\n", passFile, err)
	}
	_, err = passFile.WriteString("{}")
	assert.NoError(t, err)

	err = passFile.Close()
	assert.NoError(t, err)

	_, err = ReadStrategyConfigFromFile(passFile.Name())
	assert.NoError(t, err)

	_, err = ReadStrategyConfigFromFile("test")
	assert.ErrorIs(t, err, common.ErrFileNotFound)
}

func TestGenerateConfigForDCAAPICandles(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPICandles",
		Goal:     "To demonstrate DCA strategy using API candles",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: "bybit",
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
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
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-api-candles.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForPluginStrategy(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExamplePluginStrategy",
		Goal:     "To demonstrate that custom strategies can be used",
		StrategySettings: StrategySettings{
			Name: "custom-strategy",
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds1000000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
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
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "custom-plugin-strategy.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPICandlesExchangeLevelFunding(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPICandlesExchangeLevelFunding",
		Goal:     "To demonstrate DCA strategy using API candles using a shared pool of funds",
		StrategySettings: StrategySettings{
			Name:                         dca,
			SimultaneousSignalProcessing: true,
			DisableUSDTracking:           true,
		},
		FundingSettings: FundingSettings{
			UseExchangeLevelFunding: true,
			ExchangeLevelFunding: []ExchangeLevelFunding{
				{
					ExchangeName: mainExchange,
					Asset:        asset.Spot,
					Currency:     mainCurrencyPair.Quote,
					InitialFunds: decimal.NewFromInt(100000),
				},
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.ETH,
				Quote:        mainCurrencyPair.Quote,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
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
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-api-candles-exchange-level-funding.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPITrades(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPITrades",
		Goal:     "To demonstrate running the DCA strategy using API trade data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:                 minMax,
				SellSide:                minMax,
				MakerFee:                &makerFee,
				TakerFee:                &takerFee,
				SkipCandleVolumeFitting: true,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneHour,
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
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-api-trades.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPICandlesMultipleCurrencies(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPICandlesMultipleCurrencies",
		Goal:     "To demonstrate running the DCA strategy using the API against multiple currencies candle data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.ETH,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
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
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-api-candles-multiple-currencies.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCAAPICandlesSimultaneousProcessing(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyDCAAPICandlesSimultaneousProcessing",
		Goal:     "To demonstrate how simultaneous processing can work",
		StrategySettings: StrategySettings{
			Name:                         dca,
			SimultaneousSignalProcessing: true,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds1000000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.ETH,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
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
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-api-candles-simultaneous-processing.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCALiveCandles(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyDCALiveCandles",
		Goal:     "To demonstrate live trading proof of concept against candle data",
		StrategySettings: StrategySettings{
			Name:               dca,
			DisableUSDTracking: true,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  strictMinMax,
				SellSide: strictMinMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneMin,
			DataType: common.CandleStr,
			LiveData: &LiveData{
				NewEventTimeout:           time.Minute * 2,
				DataCheckTimer:            time.Second,
				RealOrders:                false,
				DataRequestRetryTolerance: 3,
				DataRequestRetryWaitTime:  time.Millisecond * 500,
				ExchangeCredentials: []Credentials{
					{
						Exchange: mainExchange,
						Keys: accounts.Credentials{
							Key:    "",
							Secret: "",
						},
					},
				},
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide:  strictMinMax,
			SellSide: strictMinMax,
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-candles-live.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForRSIAPICustomSettings(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "TestGenerateRSICandleAPICustomSettingsStrat",
		Goal:     "To demonstrate the RSI strategy using API candle data and custom settings",
		StrategySettings: StrategySettings{
			Name: "rsi",
			CustomSettings: map[string]any{
				"rsi-low":    30.0,
				"rsi-high":   70.0,
				"rsi-period": 14,
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.ThreeHour,
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        startDate,
				EndDate:          endDate.Add(time.Hour), // Now divisible by 3 hour candle
				InclusiveEndDate: false,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide:  minMax,
			SellSide: minMax,
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "rsi-api-candles.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCACSVCandles(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	fp := filepath.Join("..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv")
	cfg := Config{
		Nickname: "ExampleStrategyDCACSVCandles",
		Goal:     "To demonstrate the DCA strategy using CSV candle data",
		StrategySettings: StrategySettings{
			Name:               dca,
			DisableUSDTracking: true,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
			DataType: common.CandleStr,
			CSVData: &CSVData{
				FullPath: fp,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide:  minMax,
			SellSide: minMax,
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-csv-candles.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCACSVTrades(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	fp := filepath.Join("..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv")
	cfg := Config{
		Nickname: "ExampleStrategyDCACSVTrades",
		Goal:     "To demonstrate the DCA strategy using CSV trade data",
		StrategySettings: StrategySettings{
			Name:               dca,
			DisableUSDTracking: true,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneMin,
			DataType: common.TradeStr,
			CSVData: &CSVData{
				FullPath: fp,
			},
		},
		PortfolioSettings: PortfolioSettings{},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-csv-trades.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForDCADatabaseCandles(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyDCADatabaseCandles",
		Goal:     "To demonstrate the DCA strategy using database candle data",
		StrategySettings: StrategySettings{
			Name: dca,
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				SpotDetails: &SpotDetails{
					InitialQuoteFunds: initialFunds100000,
				},
				BuySide:  minMax,
				SellSide: minMax,
				MakerFee: &makerFee,
				TakerFee: &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
			DataType: common.CandleStr,
			DatabaseData: &DatabaseData{
				StartDate: startDate,
				EndDate:   endDate,
				Config: database.Config{
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
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "dca-database-candles.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForTop2Bottom2(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleStrategyTop2Bottom2",
		Goal:     "To demonstrate a complex strategy using exchange level funding and simultaneous processing of data signals",
		StrategySettings: StrategySettings{
			Name:                         top2bottom2.Name,
			SimultaneousSignalProcessing: true,

			CustomSettings: map[string]any{
				"mfi-low":    32,
				"mfi-high":   68,
				"mfi-period": 14,
			},
		},
		FundingSettings: FundingSettings{
			UseExchangeLevelFunding: true,
			ExchangeLevelFunding: []ExchangeLevelFunding{
				{
					ExchangeName: mainExchange,
					Asset:        asset.Spot,
					Currency:     mainCurrencyPair.Base,
					InitialFunds: decimal.NewFromFloat(3),
				},
				{
					ExchangeName: mainExchange,
					Asset:        asset.Spot,
					Currency:     mainCurrencyPair.Quote,
					InitialFunds: decimal.NewFromInt(10000),
				},
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.DOGE,
				Quote:        mainCurrencyPair.Quote,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.ETH,
				Quote:        mainCurrencyPair.Base,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.LTC,
				Quote:        mainCurrencyPair.Base,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.XRP,
				Quote:        mainCurrencyPair.Quote,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         currency.BNB,
				Quote:        mainCurrencyPair.Base,
				BuySide:      minMax,
				SellSide:     minMax,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		PortfolioSettings: PortfolioSettings{
			BuySide:  minMax,
			SellSide: minMax,
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "t2b2-api-candles-exchange-funding.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateBinanceCashAndCarryStrategy(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleCashAndCarry",
		Goal:     "To demonstrate a cash and carry strategy",
		StrategySettings: StrategySettings{
			Name:                         "binance-cash-carry",
			SimultaneousSignalProcessing: true,
		},
		FundingSettings: FundingSettings{
			UseExchangeLevelFunding: true,
			ExchangeLevelFunding: []ExchangeLevelFunding{
				{
					ExchangeName: mainExchange,
					Asset:        asset.Spot,
					Currency:     mainCurrencyPair.Quote,
					InitialFunds: *initialFunds100000,
				},
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName: mainExchange,
				Asset:        asset.USDTMarginedFutures,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
				BuySide:      minMax,
				SellSide:     minMax,
			},
			{
				ExchangeName: mainExchange,
				Asset:        asset.Spot,
				Base:         mainCurrencyPair.Base,
				Quote:        mainCurrencyPair.Quote,
				MakerFee:     &makerFee,
				TakerFee:     &takerFee,
				BuySide:      minMax,
				SellSide:     minMax,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.OneDay,
			DataType: common.CandleStr,
			APIData: &APIData{
				StartDate:        time.Date(2021, 1, 14, 0, 0, 0, 0, time.UTC),
				EndDate:          time.Date(2021, 9, 24, 0, 0, 0, 0, time.UTC),
				InclusiveEndDate: false,
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "binance-cash-and-carry.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateConfigForLiveCashAndCarry(t *testing.T) {
	if !saveConfig {
		t.Skip("saveConfig false, skipping")
	}
	cfg := Config{
		Nickname: "ExampleBinanceLiveCashAndCarry",
		Goal:     "To demonstrate a cash and carry strategy using a live data source",
		StrategySettings: StrategySettings{
			Name:                         "binance-cash-carry",
			SimultaneousSignalProcessing: true,
		},
		FundingSettings: FundingSettings{
			UseExchangeLevelFunding: true,
			ExchangeLevelFunding: []ExchangeLevelFunding{
				{
					ExchangeName: mainExchange,
					Asset:        asset.Spot,
					Currency:     mainCurrencyPair.Quote,
					InitialFunds: *initialFunds100000,
				},
			},
		},
		CurrencySettings: []CurrencySettings{
			{
				ExchangeName:            mainExchange,
				Asset:                   asset.USDTMarginedFutures,
				Base:                    mainCurrencyPair.Base,
				Quote:                   mainCurrencyPair.Quote,
				MakerFee:                &makerFee,
				TakerFee:                &takerFee,
				SkipCandleVolumeFitting: true,
				BuySide:                 strictMinMax,
				SellSide:                strictMinMax,
			},
			{
				ExchangeName:            mainExchange,
				Asset:                   asset.Spot,
				Base:                    mainCurrencyPair.Base,
				Quote:                   mainCurrencyPair.Quote,
				MakerFee:                &makerFee,
				TakerFee:                &takerFee,
				SkipCandleVolumeFitting: true,
				BuySide:                 strictMinMax,
				SellSide:                strictMinMax,
			},
		},
		DataSettings: DataSettings{
			Interval: kline.FifteenSecond,
			DataType: common.CandleStr,
			LiveData: &LiveData{
				NewEventTimeout:           time.Minute,
				DataCheckTimer:            time.Second,
				RealOrders:                false,
				DataRequestRetryTolerance: 3,
				ClosePositionsOnStop:      true,
				DataRequestRetryWaitTime:  time.Millisecond * 500,
				ExchangeCredentials: []Credentials{
					{
						Exchange: mainExchange,
						Keys: accounts.Credentials{
							Key:        "",
							Secret:     "",
							SubAccount: "",
						},
					},
				},
			},
		},
		StatisticSettings: StatisticSettings{
			RiskFreeRate: decimal.NewFromFloat(0.03),
		},
	}
	if saveConfig {
		result, err := json.MarshalIndent(cfg, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		p, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(p, "strategyexamples", "binance-live-cash-and-carry.strat"), result, file.DefaultPermissionOctal)
		if err != nil {
			t.Error(err)
		}
	}
}
