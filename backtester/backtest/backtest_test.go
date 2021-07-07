package backtest

import (
	"errors"
	"log"
	"path/filepath"
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
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "Bitstamp"

func newBotWithExchange() *engine.Engine {
	bot := &engine.Engine{
		Config: &gctconfig.Config{
			Exchanges: []gctconfig.ExchangeConfig{
				{
					Name:                    testExchange,
					Enabled:                 true,
					WebsocketTrafficTimeout: time.Second,
					CurrencyPairs: &currency.PairsManager{
						Pairs: map[asset.Item]*currency.PairStore{
							asset.Spot: {
								AssetEnabled:  convert.BoolPtr(true),
								Available:     []currency.Pair{currency.NewPair(currency.BTC, currency.USD)},
								Enabled:       []currency.Pair{currency.NewPair(currency.BTC, currency.USD)},
								ConfigFormat:  &currency.PairFormat{},
								RequestFormat: &currency.PairFormat{},
							},
						},
					},
				},
			},
		},
	}
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		log.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	bot.ExchangeManager = em
	return bot
}

func TestNewFromConfig(t *testing.T) {
	t.Parallel()
	_, err := NewFromConfig(nil, "", "", nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("received %v, expected %v", err, errNilConfig)
	}

	cfg := &config.Config{}
	_, err = NewFromConfig(cfg, "", "", nil)
	if !errors.Is(err, errNilBot) {
		t.Errorf("expected: %v, received %v", errNilBot, err)
	}

	bot := newBotWithExchange()
	_, err = NewFromConfig(cfg, "", "", bot)
	if !errors.Is(err, config.ErrNoCurrencySettings) {
		t.Errorf("expected: %v, received %v", config.ErrNoCurrencySettings, err)
	}

	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: "test",
			Base:         "test",
			Quote:        "test",
		},
	}
	_, err = NewFromConfig(cfg, "", "", bot)
	if !errors.Is(err, config.ErrBadInitialFunds) {
		t.Errorf("expected: %v, received %v", config.ErrBadInitialFunds, err)
	}

	cfg.CurrencySettings[0].InitialFunds = 1337
	_, err = NewFromConfig(cfg, "", "", bot)
	if !errors.Is(err, config.ErrUnsetAsset) {
		t.Errorf("expected: %v, received %v", config.ErrUnsetAsset, err)
	}

	cfg.CurrencySettings[0].Asset = asset.Spot.String()
	_, err = NewFromConfig(cfg, "", "", bot)
	if !errors.Is(err, engine.ErrExchangeNotFound) {
		t.Errorf("expected: %v, received %v", engine.ErrExchangeNotFound, err)
	}

	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]interface{}{
			"hello": "moto",
		},
	}
	cfg.CurrencySettings[0].ExchangeName = testExchange
	_, err = NewFromConfig(cfg, "", "", bot)
	if !errors.Is(err, errNoDataSource) {
		t.Errorf("expected: %v, received %v", errNoDataSource, err)
	}

	cfg.CurrencySettings[0].Base = "BTC"
	cfg.CurrencySettings[0].Quote = "USD"
	cfg.DataSettings.APIData = &config.APIData{
		StartDate: time.Time{},
		EndDate:   time.Time{},
	}

	_, err = NewFromConfig(cfg, "", "", bot)
	if err != nil && !strings.Contains(err.Error(), "unrecognised dataType") {
		t.Error(err)
	}
	cfg.DataSettings.DataType = common.CandleStr
	_, err = NewFromConfig(cfg, "", "", bot)
	if !errors.Is(err, config.ErrStartEndUnset) {
		t.Errorf("expected: %v, received %v", config.ErrStartEndUnset, err)
	}

	cfg.DataSettings.APIData.StartDate = time.Now().Add(-time.Minute)
	cfg.DataSettings.APIData.EndDate = time.Now()
	cfg.DataSettings.APIData.InclusiveEndDate = true
	_, err = NewFromConfig(cfg, "", "", bot)
	if !errors.Is(err, errIntervalUnset) {
		t.Errorf("expected: %v, received %v", errIntervalUnset, err)
	}

	cfg.DataSettings.Interval = gctkline.OneMin.Duration()
	cfg.CurrencySettings[0].MakerFee = 1337
	cfg.CurrencySettings[0].TakerFee = 1337
	_, err = NewFromConfig(cfg, "", "", bot)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataAPI(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
		Bot:     &engine.Engine{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot.String(),
				Base:         cp.Base.String(),
				Quote:        cp.Quote.String(),
				InitialFunds: 1337,
				Leverage:     config.Leverage{},
				BuySide:      config.MinMax{},
				SellSide:     config.MinMax{},
				MakerFee:     1337,
				TakerFee:     1337,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin.Duration(),
			APIData: &config.APIData{
				StartDate: time.Now().Add(-time.Minute),
				EndDate:   time.Now(),
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}

	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataDatabase(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
		Bot: &engine.Engine{
			Config: &gctconfig.Config{Database: database.Config{}},
		},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot.String(),
				Base:         cp.Base.String(),
				Quote:        cp.Quote.String(),
				InitialFunds: 1337,
				Leverage:     config.Leverage{},
				BuySide:      config.MinMax{},
				SellSide:     config.MinMax{},
				MakerFee:     1337,
				TakerFee:     1337,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin.Duration(),
			DatabaseData: &config.DatabaseData{
				ConfigOverride: &database.Config{
					Enabled: true,
					Driver:  "sqlite3",
					ConnectionDetails: drivers.ConnectionDetails{
						Database: "gocryptotrader.db",
					},
				},
				StartDate:        time.Now().Add(-time.Minute),
				EndDate:          time.Now(),
				InclusiveEndDate: true,
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}

	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil && !strings.Contains(err.Error(), "unable to retrieve data from GoCryptoTrader database") {
		t.Error(err)
	}
}

func TestLoadDataCSV(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
		Bot:     &engine.Engine{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot.String(),
				Base:         cp.Base.String(),
				Quote:        cp.Quote.String(),
				InitialFunds: 1337,
				Leverage:     config.Leverage{},
				BuySide:      config.MinMax{},
				SellSide:     config.MinMax{},
				MakerFee:     1337,
				TakerFee:     1337,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin.Duration(),
			CSVData: &config.CSVData{
				FullPath: "test",
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil &&
		!strings.Contains(err.Error(), "The system cannot find the file specified.") &&
		!strings.Contains(err.Error(), "no such file or directory") {
		t.Error(err)
	}
}

func TestLoadDataLive(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports:  &report.Data{},
		Bot:      &engine.Engine{},
		shutdown: make(chan struct{}),
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot.String(),
				Base:         cp.Base.String(),
				Quote:        cp.Quote.String(),
				InitialFunds: 1337,
				Leverage:     config.Leverage{},
				BuySide:      config.MinMax{},
				SellSide:     config.MinMax{},
				MakerFee:     1337,
				TakerFee:     1337,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin.Duration(),
			LiveData: &config.LiveData{
				APIKeyOverride:      "test",
				APISecretOverride:   "test",
				APIClientIDOverride: "test",
				API2FAOverride:      "test",
				RealOrders:          true,
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	bt.Stop()
}

func TestLoadLiveData(t *testing.T) {
	t.Parallel()
	err := loadLiveData(nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	cfg := &config.Config{
		GoCryptoTraderConfigPath: filepath.Join("..", "..", "testdata", "configtest.json"),
	}
	err = loadLiveData(cfg, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	b := &gctexchange.Base{
		Name: testExchange,
		API: gctexchange.API{
			AuthenticatedSupport:          false,
			AuthenticatedWebsocketSupport: false,
			PEMKeySupport:                 false,
			Credentials: struct {
				Key        string
				Secret     string
				ClientID   string
				PEMKey     string
				Subaccount string
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
	if !errors.Is(err, common.ErrNilArguments) {
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
	cfg.DataSettings.LiveData.APISubaccountOverride = "1234"
	err = loadLiveData(cfg, b)
	if err != nil {
		t.Error(err)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	ex := testExchange
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Spot
	tt := time.Now()

	stats := &statistics.Statistic{}
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*currencystatistics.CurrencyStatistic)

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
	bot := newBotWithExchange()

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
		Base: data.Base{},
		Range: &gctkline.IntervalRangeHolder{
			Start: gctkline.CreateIntervalTime(tt),
			End:   gctkline.CreateIntervalTime(tt.Add(gctkline.FifteenMin.Duration())),
			Ranges: []gctkline.IntervalRange{
				{
					Start: gctkline.CreateIntervalTime(tt),
					End:   gctkline.CreateIntervalTime(tt.Add(gctkline.FifteenMin.Duration())),
					Intervals: []gctkline.IntervalData{
						{
							Start:   gctkline.CreateIntervalTime(tt),
							End:     gctkline.CreateIntervalTime(tt.Add(gctkline.FifteenMin.Duration())),
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
	t.Parallel()
	bt := BackTest{shutdown: make(chan struct{})}
	bt.Stop()
}

func TestFullCycleMulti(t *testing.T) {
	t.Parallel()
	ex := testExchange
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Spot
	tt := time.Now()

	stats := &statistics.Statistic{}
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*currencystatistics.CurrencyStatistic)

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
	bot := newBotWithExchange()

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
		Base: data.Base{},
		Range: &gctkline.IntervalRangeHolder{
			Start: gctkline.CreateIntervalTime(tt),
			End:   gctkline.CreateIntervalTime(tt.Add(gctkline.FifteenMin.Duration())),
			Ranges: []gctkline.IntervalRange{
				{
					Start: gctkline.CreateIntervalTime(tt),
					End:   gctkline.CreateIntervalTime(tt.Add(gctkline.FifteenMin.Duration())),
					Intervals: []gctkline.IntervalData{
						{
							Start:   gctkline.CreateIntervalTime(tt),
							End:     gctkline.CreateIntervalTime(tt.Add(gctkline.FifteenMin.Duration())),
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
