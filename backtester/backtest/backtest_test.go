package backtest

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

func TestNewFromConfig(t *testing.T) {
	_, err := NewFromConfig(nil, "", "")
	if err == nil {
		t.Error("expected error for nil config")
	}

	cfg := &config.Config{}
	_, err = NewFromConfig(cfg, "", "")
	if err == nil {
		t.Error("expected error for nil config")
	}
	if err != nil && err.Error() != "expected at least one currency in the config" {
		t.Error(err)
	}

	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: "test",
			Asset:        "test",
			Base:         "test",
			Quote:        "test",
		},
	}
	_, err = NewFromConfig(cfg, "", "")
	if err != nil && err.Error() != "exchange not found" {
		t.Error(err)
	}

	cfg.CurrencySettings[0].ExchangeName = testExchange
	_, err = NewFromConfig(cfg, "", "")
	if err != nil && !strings.Contains(err.Error(), "cannot create new asset") {
		t.Error(err)
	}

	cfg.CurrencySettings[0].Asset = asset.Spot.String()
	cfg.CurrencySettings[0].Base = "BTC"
	cfg.CurrencySettings[0].Quote = "USDT"
	_, err = NewFromConfig(cfg, "", "")
	if err != nil && !strings.Contains(err.Error(), "initial funds unset") {
		t.Error(err)
	}

	cfg.CurrencySettings[0].InitialFunds = 1337

	_, err = NewFromConfig(cfg, "", "")
	if err != nil && err.Error() != "no data settings set in config" {
		t.Error(err)
	}

	cfg.DataSettings.APIData = &config.APIData{
		StartDate: time.Time{},
		EndDate:   time.Time{},
	}

	_, err = NewFromConfig(cfg, "", "")
	if err != nil && err.Error() != "api data start and end dates must be set" {
		t.Error(err)
	}

	cfg.DataSettings.APIData.StartDate = time.Now().Add(-time.Hour)
	cfg.DataSettings.APIData.EndDate = time.Now()
	_, err = NewFromConfig(cfg, "", "")
	if err != nil && err.Error() != "api data interval unset" {
		t.Error(err)
	}

	cfg.DataSettings.Interval = gctkline.FifteenMin.Duration()
	_, err = NewFromConfig(cfg, "", "")
	if err != nil && err.Error() != "unrecognised api datatype received: ''" {
		t.Error(err)
	}

	cfg.DataSettings.DataType = common.CandleStr
	_, err = NewFromConfig(cfg, "", "")
	if err != nil && err.Error() != "strategy '' not found" {
		t.Error(err)
	}

	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]interface{}{
			"hello": "moto",
		},
	}
	cfg.CurrencySettings[0].MakerFee = 1337
	cfg.CurrencySettings[0].TakerFee = 1337
	_, err = NewFromConfig(cfg, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestLoadData(t *testing.T) {
	cfg := &config.Config{}
	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: "test",
			Asset:        "test",
			Base:         "test",
			Quote:        "test",
		},
	}
	cfg.CurrencySettings[0].ExchangeName = testExchange
	cfg.CurrencySettings[0].Asset = asset.Spot.String()
	cfg.CurrencySettings[0].Base = "BTC"
	cfg.CurrencySettings[0].Quote = "USDT"
	cfg.CurrencySettings[0].InitialFunds = 1337
	cfg.DataSettings.APIData = &config.APIData{
		StartDate: time.Time{},
		EndDate:   time.Time{},
	}
	cfg.DataSettings.APIData.StartDate = time.Now().Add(-time.Hour)
	cfg.DataSettings.APIData.EndDate = time.Now()
	cfg.DataSettings.Interval = gctkline.FifteenMin.Duration()
	cfg.DataSettings.DataType = common.CandleStr
	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]interface{}{
			"hello": "moto",
		},
	}
	cfg.CurrencySettings[0].MakerFee = 1337
	cfg.CurrencySettings[0].TakerFee = 1337
	_, err := NewFromConfig(cfg, "", "")
	if err != nil {
		t.Error(err)
	}
	bt := BackTest{
		Reports: &report.Data{},
	}
	bot := &engine.Engine{
		Config: &gctconfig.Config{
			Exchanges: []gctconfig.ExchangeConfig{
				{
					Name:    testExchange,
					Enabled: true,
					API: gctconfig.APIConfig{
						Endpoints: gctconfig.APIEndpointsConfig{
							URL:          "https://api.binance.com",
							URLSecondary: "https://test.test",
							WebsocketURL: "wss://test.test",
						},
					},
					HTTPTimeout:                   time.Hour,
					WebsocketResponseCheckTimeout: time.Hour,
					WebsocketTrafficTimeout:       time.Hour,
					Websocket:                     convert.BoolPtr(false),
					CurrencyPairs: &currency.PairsManager{
						Pairs: map[asset.Item]*currency.PairStore{
							asset.Spot: {AssetEnabled: convert.BoolPtr(true)},
						},
					},
				},
			},
		},
	}
	err = bot.LoadExchange(testExchange, false, nil)
	if err != nil {
		t.Error(err)
	}
	exch := bot.GetExchangeByName(testExchange)
	if exch == nil {
		t.Error("expected not nil")
	}

	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	cfg.DataSettings.APIData = nil
	cfg.DataSettings.DatabaseData = &config.DatabaseData{
		StartDate:      time.Now().Add(-time.Hour),
		EndDate:        time.Now(),
		ConfigOverride: nil,
	}
	cfg.DataSettings.DataType = common.CandleStr
	cfg.DataSettings.Interval = gctkline.FifteenMin.Duration()
	bt.Bot = bot
	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil && err.Error() != "database support is disabled" {
		t.Error(err)
	}

	cfg.DataSettings.DatabaseData = nil
	cfg.DataSettings.CSVData = &config.CSVData{
		FullPath: "test",
	}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil && !strings.Contains(err.Error(), "The system cannot find the file specified.") {
		t.Error(err)
	}
	cfg.DataSettings.CSVData = nil
	cfg.DataSettings.LiveData = &config.LiveData{
		APIKeyOverride:      "test",
		APISecretOverride:   "test",
		APIClientIDOverride: "test",
		API2FAOverride:      "test",
		RealOrders:          true,
	}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDatabaseData(t *testing.T) {
	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err := loadDatabaseData(nil, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "nil config data received") {
		t.Error(err)
	}
	cfg := &config.Config{
		DataSettings: config.DataSettings{
			DatabaseData: &config.DatabaseData{
				StartDate:      time.Time{},
				EndDate:        time.Time{},
				ConfigOverride: nil,
			},
		},
	}
	_, err = loadDatabaseData(cfg, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "database data start and end dates must be set") {
		t.Error(err)
	}
	cfg.DataSettings.DatabaseData.StartDate = time.Now().Add(-time.Hour)
	cfg.DataSettings.DatabaseData.EndDate = time.Now()
	_, err = loadDatabaseData(cfg, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "unexpected database datatype: ''") {
		t.Error(err)
	}

	cfg.DataSettings.DataType = common.CandleStr
	_, err = loadDatabaseData(cfg, "", cp, "")
	if err != nil && !strings.Contains(err.Error(), "exchange, base, quote, asset, interval, start & end cannot be empty") {
		t.Error(err)
	}
	cfg.DataSettings.Interval = gctkline.OneDay.Duration()
	_, err = loadDatabaseData(cfg, testExchange, cp, asset.Spot)
	if err != nil && !strings.Contains(err.Error(), "database support is disabled") {
		t.Error(err)
	}
}

func TestLoadLiveData(t *testing.T) {
	err := loadLiveData(nil, nil)
	if err != nil && err.Error() != "received nil argument(s)" {
		t.Error(err)
	}
	cfg := &config.Config{}
	err = loadLiveData(cfg, nil)
	if err != nil && err.Error() != "received nil argument(s)" {
		t.Error(err)
	}
	b := &gctexchange.Base{
		Name: testExchange,
		API: gctexchange.API{
			AuthenticatedSupport:          false,
			AuthenticatedWebsocketSupport: false,
			PEMKeySupport:                 false,
			Endpoints: struct {
				URL                 string
				URLDefault          string
				URLSecondary        string
				URLSecondaryDefault string
				WebsocketURL        string
			}{},
			Credentials: struct {
				Key      string
				Secret   string
				ClientID string
				PEMKey   string
			}{},
			CredentialsValidator: struct {
				RequiresPEM                bool
				RequiresKey                bool
				RequiresSecret             bool
				RequiresClientID           bool
				RequiresBase64DecodeSecret bool
			}{
				RequiresPEM:                true,
				RequiresKey:                true,
				RequiresSecret:             true,
				RequiresClientID:           true,
				RequiresBase64DecodeSecret: true,
			},
		},
	}
	err = loadLiveData(cfg, b)
	if err != nil && err.Error() != "received nil argument(s)" {
		t.Error(err)
	}
	cfg.DataSettings.LiveData = &config.LiveData{

		RealOrders: true,
	}
	cfg.DataSettings.Interval = gctkline.OneDay.Duration()
	cfg.DataSettings.DataType = common.CandleStr
	err = loadLiveData(cfg, b)
	if err != nil {
		t.Error(err)
	}

	cfg.DataSettings.LiveData.APIKeyOverride = "1234"
	cfg.DataSettings.LiveData.APISecretOverride = "1234"
	cfg.DataSettings.LiveData.APIClientIDOverride = "1234"
	cfg.DataSettings.LiveData.API2FAOverride = "1234"
	err = loadLiveData(cfg, b)
	if err != nil {
		t.Error(err)
	}
}

func TestReset(t *testing.T) {
	bt := BackTest{
		Bot:        &engine.Engine{},
		shutdown:   make(chan struct{}),
		Datas:      &data.HandlerPerCurrency{},
		Strategy:   &dollarcostaverage.Strategy{},
		Portfolio:  &portfolio.Portfolio{},
		Exchange:   &exchange.Exchange{},
		Statistic:  &statistics.Statistic{},
		EventQueue: &eventholder.Holder{},
		Reports:    &report.Data{},
	}
	bt.Reset()
	if bt.Bot != nil {
		t.Error("expected nil")
	}
}

func TestFullCycle(t *testing.T) {
	ex := testExchange
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Spot
	tt := time.Now()

	stats := &statistics.Statistic{}
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*currencystatstics.CurrencyStatistic)

	port, err := portfolio.Setup(&size.Size{
		BuySide:  config.MinMax{},
		SellSide: config.MinMax{},
	}, &risk.Risk{}, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = port.SetupCurrencySettingsMap(ex, a, cp)
	if err != nil {
		t.Error(err)
	}
	err = port.SetInitialFunds(ex, a, cp, 1333337)
	if err != nil {
		t.Error(err)
	}

	bot := &engine.Engine{
		Config: &gctconfig.Config{
			Exchanges: []gctconfig.ExchangeConfig{
				{
					Name:    testExchange,
					Enabled: true,
					API: gctconfig.APIConfig{
						Endpoints: gctconfig.APIEndpointsConfig{
							URL:          "https://api.binance.com",
							URLSecondary: "https://test.test",
							WebsocketURL: "wss://test.test",
						},
					},
					HTTPTimeout:                   time.Hour,
					WebsocketResponseCheckTimeout: time.Hour,
					WebsocketTrafficTimeout:       time.Hour,
					Websocket:                     convert.BoolPtr(false),
					CurrencyPairs: &currency.PairsManager{
						Pairs: map[asset.Item]*currency.PairStore{
							asset.Spot: {AssetEnabled: convert.BoolPtr(true)},
						},
					},
				},
			},
		},
	}
	err = bot.OrderManager.Start(bot)
	if err != nil {
		t.Error(err)
	}
	err = bot.LoadExchange(ex, false, nil)
	if err != nil {
		t.Error(err)
	}

	bt := BackTest{
		Bot:        bot,
		shutdown:   nil,
		Datas:      &data.HandlerPerCurrency{},
		Strategy:   &dollarcostaverage.Strategy{},
		Portfolio:  port,
		Exchange:   &exchange.Exchange{},
		Statistic:  stats,
		EventQueue: &eventholder.Holder{},
		Reports:    &report.Data{},
	}

	bt.Datas.Setup()
	k := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: ex,
			Pair:     cp,
			Asset:    a,
			Interval: gctkline.FifteenMin,
			Candles: []gctkline.Candle{{
				Time:   tt,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			}},
		},
		Data: data.Data{},
		Range: gctkline.IntervalRangeHolder{
			Start: tt,
			End:   tt.Add(gctkline.FifteenMin.Duration()),
			Ranges: []gctkline.IntervalRange{
				{
					Start: tt,
					End:   tt.Add(gctkline.FifteenMin.Duration()),
					Intervals: []gctkline.IntervalData{
						{
							Start:   tt,
							End:     tt.Add(gctkline.FifteenMin.Duration()),
							HasData: true,
						},
					},
				},
			},
		},
	}
	err = k.Load()
	if err != nil {
		t.Error(err)
	}
	bt.Datas.SetDataForCurrency(ex, a, cp, &k)

	err = bt.Run()
	if err != nil {
		t.Error(err)
	}
}

func TestStop(t *testing.T) {
	bt := BackTest{shutdown: make(chan struct{})}
	bt.Stop()
}

func TestFullCycleMulti(t *testing.T) {
	ex := testExchange
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Spot
	tt := time.Now()

	stats := &statistics.Statistic{}
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*currencystatstics.CurrencyStatistic)

	port, err := portfolio.Setup(&size.Size{
		BuySide:  config.MinMax{},
		SellSide: config.MinMax{},
	}, &risk.Risk{}, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = port.SetupCurrencySettingsMap(ex, a, cp)
	if err != nil {
		t.Error(err)
	}
	err = port.SetInitialFunds(ex, a, cp, 1333337)
	if err != nil {
		t.Error(err)
	}

	bot := &engine.Engine{
		Config: &gctconfig.Config{
			Exchanges: []gctconfig.ExchangeConfig{
				{
					Name:    testExchange,
					Enabled: true,
					API: gctconfig.APIConfig{
						Endpoints: gctconfig.APIEndpointsConfig{
							URL:          "https://api.binance.com",
							URLSecondary: "https://test.test",
							WebsocketURL: "wss://test.test",
						},
					},
					HTTPTimeout:                   time.Hour,
					WebsocketResponseCheckTimeout: time.Hour,
					WebsocketTrafficTimeout:       time.Hour,
					Websocket:                     convert.BoolPtr(false),
					CurrencyPairs: &currency.PairsManager{
						Pairs: map[asset.Item]*currency.PairStore{
							asset.Spot: {AssetEnabled: convert.BoolPtr(true)},
						},
					},
				},
			},
		},
	}
	err = bot.OrderManager.Start(bot)
	if err != nil {
		t.Error(err)
	}
	err = bot.LoadExchange(ex, false, nil)
	if err != nil {
		t.Error(err)
	}

	bt := BackTest{
		Bot:        bot,
		shutdown:   nil,
		Datas:      &data.HandlerPerCurrency{},
		Portfolio:  port,
		Exchange:   &exchange.Exchange{},
		Statistic:  stats,
		EventQueue: &eventholder.Holder{},
		Reports:    &report.Data{},
	}

	bt.Strategy, err = strategies.LoadStrategyByName(dollarcostaverage.Name, true)
	if err != nil {
		t.Error(err)
	}

	bt.Datas.Setup()
	k := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: ex,
			Pair:     cp,
			Asset:    a,
			Interval: gctkline.FifteenMin,
			Candles: []gctkline.Candle{{
				Time:   tt,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			}},
		},
		Data: data.Data{},
		Range: gctkline.IntervalRangeHolder{
			Start: tt,
			End:   tt.Add(gctkline.FifteenMin.Duration()),
			Ranges: []gctkline.IntervalRange{
				{
					Start: tt,
					End:   tt.Add(gctkline.FifteenMin.Duration()),
					Intervals: []gctkline.IntervalData{
						{
							Start:   tt,
							End:     tt.Add(gctkline.FifteenMin.Duration()),
							HasData: true,
						},
					},
				},
			},
		},
	}
	err = k.Load()
	if err != nil {
		t.Error(err)
	}

	bt.Datas.SetDataForCurrency(ex, a, cp, &k)

	err = bt.Run()
	if err != nil {
		t.Error(err)
	}
}
