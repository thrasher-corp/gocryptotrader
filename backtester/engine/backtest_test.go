package engine

import (
	"errors"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/ftxcashandcarry"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
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
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	evkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "ftx"

var leet = decimal.NewFromInt(1337)

type portfolioOverride struct {
	Err error
	portfolio.Portfolio
}

func (p portfolioOverride) CreateLiquidationOrdersForExchange(ev data.Event, _ funding.IFundingManager) ([]order.Event, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	return []order.Event{
		&order.Order{
			Base:      ev.GetBase(),
			ID:        "1",
			Direction: gctorder.Short,
		},
	}, nil
}

func TestNewFromConfig(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = bt.NewFromConfig(nil, "", "", false)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("received %v, expected %v", err, errNilConfig)
	}
	cfg := &config.Config{}
	err = bt.NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}

	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: testExchange,
			Base:         currency.BTC,
			Quote:        currency.NewCode("0624"),
			Asset:        asset.Spot,
		},
	}
	err = bt.NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}

	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]interface{}{
			"hello": "moto",
		},
	}
	cfg.CurrencySettings[0].Base = currency.BTC
	cfg.CurrencySettings[0].Quote = currency.USD
	cfg.DataSettings.APIData = &config.APIData{
		StartDate: time.Time{},
		EndDate:   time.Time{},
	}

	err = bt.NewFromConfig(cfg, "", "", false)
	if err != nil && !strings.Contains(err.Error(), "unrecognised dataType") {
		t.Error(err)
	}
	cfg.DataSettings.DataType = common.CandleStr
	err = bt.NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, errIntervalUnset) {
		t.Errorf("received: %v, expected: %v", err, errIntervalUnset)
	}
	cfg.DataSettings.Interval = gctkline.OneMin
	cfg.CurrencySettings[0].MakerFee = &decimal.Zero
	cfg.CurrencySettings[0].TakerFee = &decimal.Zero
	err = bt.NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrDateUnset)
	}

	cfg.DataSettings.APIData.StartDate = time.Now().Add(-time.Minute)
	cfg.DataSettings.APIData.EndDate = time.Now()
	cfg.DataSettings.APIData.InclusiveEndDate = true
	err = bt.NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	cfg.FundingSettings.UseExchangeLevelFunding = true
	cfg.FundingSettings.ExchangeLevelFunding = []config.ExchangeLevelFunding{
		{
			ExchangeName: testExchange,
			Asset:        asset.Spot,
			Currency:     currency.BTC,
			InitialFunds: leet,
			TransferFee:  leet,
		},
		{
			ExchangeName: testExchange,
			Asset:        asset.Futures,
			Currency:     currency.BTC,
			InitialFunds: leet,
			TransferFee:  leet,
		},
	}
	err = bt.NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

}

func TestLoadDataAPI(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
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
	exch, err := em.NewExchangeByName(testExchange)
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
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: "/"}}

	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestLoadDataDatabase(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
			DatabaseData: &config.DatabaseData{
				Config: database.Config{
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
	exch, err := em.NewExchangeByName(testExchange)
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
	bt.databaseManager, err = engine.SetupDatabaseConnectionManager(&cfg.DataSettings.DatabaseData.Config)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if err != nil && !strings.Contains(err.Error(), "unable to retrieve data from GoCryptoTrader database") {
		t.Error(err)
	}
}

func TestLoadDataCSV(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
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
	exch, err := em.NewExchangeByName(testExchange)
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
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
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
		shutdown: make(chan struct{}),
	}

	cp := currency.NewPair(currency.BTC, currency.USD)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: testExchange,
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
			LiveData: &config.LiveData{
				ExchangeCredentials: []config.Credentials{
					{
						Exchange: testExchange,
						Credentials: account.Credentials{
							Key:             "test",
							Secret:          "test",
							ClientID:        "test",
							PEMKey:          "test",
							SubAccount:      "test",
							OneTimePassword: "test",
						},
					},
				},
				RealOrders: true,
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	bt.LiveDataHandler, err = SetupLiveDataHandler(&em, &data.HandlerPerCurrency{}, 0, 0, 0, false)
	err = bt.LiveDataHandler.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: "/"}}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	bt.Stop()
}

func TestLoadLiveData(t *testing.T) {
	t.Parallel()
	err := setExchangeCredentials(nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	cfg := &config.Config{}
	err = setExchangeCredentials(cfg, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	b := &gctexchange.Base{
		Name: testExchange,
		API: gctexchange.API{
			AuthenticatedSupport:          false,
			AuthenticatedWebsocketSupport: false,
			PEMKeySupport:                 false,
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

	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	cfg.DataSettings.LiveData = &config.LiveData{
		RealOrders: true,
	}
	cfg.DataSettings.Interval = gctkline.OneDay
	cfg.DataSettings.DataType = common.CandleStr
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	cfg.DataSettings.LiveData.ExchangeCredentials = []config.Credentials{
		{
			Exchange: testExchange,
			Credentials: account.Credentials{
				Key:             "1234",
				Secret:          "1234",
				ClientID:        "1234",
				PEMKey:          "1234",
				SubAccount:      "1234",
				OneTimePassword: "1234",
			},
		},
	}
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	bt := BackTest{
		shutdown:   make(chan struct{}),
		DataHolder: &data.HandlerPerCurrency{},
		Strategy:   &dollarcostaverage.Strategy{},
		Portfolio:  &portfolio.Portfolio{},
		Exchange:   &exchange.Exchange{},
		Statistic:  &statistics.Statistic{},
		EventQueue: &eventholder.Holder{},
		Reports:    &report.Data{},
		Funding:    f,
	}
	bt.Reset()
	if bt.Funding.IsUsingExchangeLevelFunding() {
		t.Error("expected false")
	}
}

func TestFullCycle(t *testing.T) {
	t.Parallel()
	ex := testExchange
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Spot
	tt := time.Now()

	stats := &statistics.Statistic{}
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex][a][cp.Base.Item] = make(map[*currency.Item]*statistics.CurrencyPairStatistic)

	port, err := portfolio.Setup(&size.Size{
		BuySide:  exchange.MinMax{},
		SellSide: exchange.MinMax{},
	}, &risk.Risk{}, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	fx := &ftx.FTX{}
	fx.Name = testExchange
	err = port.SetupCurrencySettingsMap(&exchange.Settings{Exchange: fx, Asset: a, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b, err := funding.CreateItem(ex, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	quote, err := funding.CreateItem(ex, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	pair, err := funding.CreatePair(b, quote)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = f.AddPair(pair)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	bt := BackTest{
		DataHolder:               &data.HandlerPerCurrency{},
		Strategy:                 &dollarcostaverage.Strategy{},
		Portfolio:                port,
		Exchange:                 &exchange.Exchange{},
		Statistic:                stats,
		EventQueue:               &eventholder.Holder{},
		Reports:                  &report.Data{},
		hasProcessedDataAtOffset: make(map[int64]bool),
		Funding:                  f,
	}

	bt.DataHolder.Setup()
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
		RangeHolder: &gctkline.IntervalRangeHolder{
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
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	bt.DataHolder.SetDataForCurrency(ex, a, cp, &k)

	bt.Run()
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
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex][a][cp.Base.Item] = make(map[*currency.Item]*statistics.CurrencyPairStatistic)

	port, err := portfolio.Setup(&size.Size{
		BuySide:  exchange.MinMax{},
		SellSide: exchange.MinMax{},
	}, &risk.Risk{}, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = port.SetupCurrencySettingsMap(&exchange.Settings{Exchange: &ftx.FTX{}, Asset: a, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b, err := funding.CreateItem(ex, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	quote, err := funding.CreateItem(ex, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	pair, err := funding.CreatePair(b, quote)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = f.AddPair(pair)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	bt := BackTest{
		shutdown:                 nil,
		DataHolder:               &data.HandlerPerCurrency{},
		Portfolio:                port,
		Exchange:                 &exchange.Exchange{},
		Statistic:                stats,
		EventQueue:               &eventholder.Holder{},
		Reports:                  &report.Data{},
		Funding:                  f,
		hasProcessedDataAtOffset: make(map[int64]bool),
	}

	bt.Strategy, err = strategies.LoadStrategyByName(dollarcostaverage.Name, true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	bt.DataHolder.Setup()
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
		RangeHolder: &gctkline.IntervalRangeHolder{
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
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	bt.DataHolder.SetDataForCurrency(ex, a, cp, &k)

	bt.Run()
}

func TestTriggerLiquidationsForExchange(t *testing.T) {
	t.Parallel()
	bt := BackTest{}
	expectedError := common.ErrNilEvent
	err := bt.triggerLiquidationsForExchange(nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	expectedError = common.ErrNilArguments
	ev := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.triggerLiquidationsForExchange(ev, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	bt.Portfolio = &portfolioOverride{}
	pnl := &portfolio.PNLSummary{}
	bt.DataHolder = &data.HandlerPerCurrency{}
	d := data.Base{}
	d.SetStream([]data.Event{&evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.OneDay,
			CurrencyPair: cp,
			AssetType:    a,
		},
		Open:   leet,
		Close:  leet,
		Low:    leet,
		High:   leet,
		Volume: leet,
	}})
	d.Next()
	da := &kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: testExchange,
			Asset:    a,
			Pair:     cp,
		},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	bt.Statistic = &statistics.Statistic{}
	expectedError = nil

	bt.EventQueue = &eventholder.Holder{}
	bt.Funding = &funding.FundManager{}
	bt.DataHolder.SetDataForCurrency(testExchange, a, cp, da)
	err = bt.Statistic.SetEventForOffset(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	pnl.Exchange = ev.Exchange
	pnl.Item = ev.AssetType
	pnl.Pair = ev.CurrencyPair
	err = bt.triggerLiquidationsForExchange(ev, pnl)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev2 := bt.EventQueue.NextEvent()
	ev2o, ok := ev2.(order.Event)
	if !ok {
		t.Fatal("expected order event")
	}
	if ev2o.GetDirection() != gctorder.Short {
		t.Error("expected liquidation order")
	}
}

func TestUpdateStatsForDataEvent(t *testing.T) {
	t.Parallel()
	pt := &portfolio.Portfolio{}
	bt := &BackTest{
		Statistic: &statistics.Statistic{},
		Funding:   &funding.FundManager{},
		Portfolio: pt,
	}
	expectedError := common.ErrNilEvent
	err := bt.updateStatsForDataEvent(nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	ev := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}

	expectedError = common.ErrNilArguments
	err = bt.updateStatsForDataEvent(ev, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	expectedError = nil
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	pair, err := funding.CreateCollateral(b, quote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt.Funding = f
	exch := &ftx.FTX{}
	exch.Name = testExchange
	err = pt.SetupCurrencySettingsMap(&exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev.Time = time.Now()
	fl := &fill.Fill{
		Base:                ev.Base,
		Direction:           gctorder.Short,
		Amount:              decimal.NewFromInt(1),
		ClosePrice:          decimal.NewFromInt(1),
		VolumeAdjustedPrice: decimal.NewFromInt(1),
		PurchasePrice:       decimal.NewFromInt(1),
		Total:               decimal.NewFromInt(1),
		Slippage:            decimal.NewFromInt(1),
		Order: &gctorder.Detail{
			Exchange:  testExchange,
			AssetType: ev.AssetType,
			Pair:      cp,
			Amount:    1,
			Price:     1,
			Side:      gctorder.Short,
			OrderID:   "1",
			Date:      time.Now(),
		},
	}
	_, err = pt.TrackFuturesOrder(fl, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	err = bt.updateStatsForDataEvent(ev, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
}

func TestProcessSignalEvent(t *testing.T) {
	t.Parallel()
	var expectedError error
	pt, err := portfolio.Setup(&size.Size{}, &risk.Risk{}, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt := &BackTest{
		Statistic:  &statistics.Statistic{},
		Funding:    &funding.FundManager{},
		Portfolio:  pt,
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetEventForOffset(de)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &signal.Signal{
		Base: de.Base,
	}

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	pair, err := funding.CreateCollateral(b, quote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt.Funding = f
	exch := &ftx.FTX{}
	exch.Name = testExchange
	err = pt.SetupCurrencySettingsMap(&exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = bt.processSignalEvent(ev, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
}

func TestProcessOrderEvent(t *testing.T) {
	t.Parallel()
	var expectedError error
	pt, err := portfolio.Setup(&size.Size{}, &risk.Risk{}, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt := &BackTest{
		Statistic:  &statistics.Statistic{},
		Funding:    &funding.FundManager{},
		Portfolio:  pt,
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerPerCurrency{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetEventForOffset(de)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &order.Order{
		Base: de.Base,
	}

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	pair, err := funding.CreateCollateral(b, quote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt.Funding = f
	exch := &ftx.FTX{}
	exch.Name = testExchange
	err = pt.SetupCurrencySettingsMap(&exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	tt := time.Now()
	bt.DataHolder.Setup()
	k := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: testExchange,
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
		RangeHolder: &gctkline.IntervalRangeHolder{
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
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	bt.DataHolder.SetDataForCurrency(testExchange, a, cp, &k)
	err = bt.processOrderEvent(ev, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev2 := bt.EventQueue.NextEvent()
	if _, ok := ev2.(fill.Event); !ok {
		t.Fatal("expected fill event")
	}
}

func TestProcessFillEvent(t *testing.T) {
	t.Parallel()
	var expectedError error
	pt, err := portfolio.Setup(&size.Size{}, &risk.Risk{}, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt := &BackTest{
		Statistic:  &statistics.Statistic{},
		Funding:    &funding.FundManager{},
		Portfolio:  pt,
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerPerCurrency{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetEventForOffset(de)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &fill.Fill{
		Base: de.Base,
	}
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	f, err := funding.SetupFundingManager(em, false, true)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	pair, err := funding.CreateCollateral(b, quote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	err = f.AddItem(b)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = f.AddItem(quote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	spotBase, err := funding.CreateItem(testExchange, asset.Spot, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	spotQuote, err := funding.CreateItem(testExchange, asset.Spot, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	spotPair, err := funding.CreatePair(spotBase, spotQuote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = f.AddPair(spotPair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	bt.Funding = f
	err = pt.SetupCurrencySettingsMap(&exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	tt := time.Now()
	bt.DataHolder.Setup()
	k := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: testExchange,
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
		RangeHolder: &gctkline.IntervalRangeHolder{
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
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	bt.DataHolder.SetDataForCurrency(testExchange, a, cp, &k)
	err = bt.processFillEvent(ev, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
}

func TestProcessFuturesFillEvent(t *testing.T) {
	t.Parallel()
	var expectedError error
	pt, err := portfolio.Setup(&size.Size{}, &risk.Risk{}, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt := &BackTest{
		Statistic:  &statistics.Statistic{},
		Funding:    &funding.FundManager{},
		Portfolio:  pt,
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerPerCurrency{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetEventForOffset(de)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &fill.Fill{
		Base: de.Base,
	}
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	em.Add(exch)
	f, err := funding.SetupFundingManager(em, false, true)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	pair, err := funding.CreateCollateral(b, quote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	err = f.AddItem(b)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = f.AddItem(quote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	spotBase, err := funding.CreateItem(testExchange, asset.Spot, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	spotQuote, err := funding.CreateItem(testExchange, asset.Spot, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	spotPair, err := funding.CreatePair(spotBase, spotQuote)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = f.AddPair(spotPair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	bt.exchangeManager = em
	bt.Funding = f
	err = pt.SetupCurrencySettingsMap(&exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	tt := time.Now()
	bt.DataHolder.Setup()
	k := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: testExchange,
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
		RangeHolder: &gctkline.IntervalRangeHolder{
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
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	ev.Order = &gctorder.Detail{
		Exchange:  testExchange,
		AssetType: ev.AssetType,
		Pair:      cp,
		Amount:    1,
		Price:     1,
		Side:      gctorder.Short,
		OrderID:   "1",
		Date:      time.Now(),
	}
	bt.DataHolder.SetDataForCurrency(testExchange, a, cp, &k)
	err = bt.processFuturesFillEvent(ev, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	hi := make(chan struct{})
	t.Log(hi)
	hi = make(chan struct{})
}

func TestCloseAllPositions(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pt := &portfolio.Portfolio{}
	bt.Portfolio = pt
	bt.Strategy = &dollarcostaverage.Strategy{}

	err = bt.CloseAllPositions()
	if !errors.Is(err, errLiveOnly) {
		t.Errorf("received '%v' expected '%v'", err, errLiveOnly)
	}

	bt.shutdown = make(chan struct{})
	bt.LiveDataHandler = &DataChecker{}
	err = bt.CloseAllPositions()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.shutdown = make(chan struct{})
	bt.Strategy = &ftxcashandcarry.Strategy{}
	err = bt.CloseAllPositions()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.shutdown = make(chan struct{})
	bt.Portfolio = &fakeFolio{}
	bt.Strategy = &fakeStrat{}
	bt.Exchange = &exchange.Exchange{}
	bt.Statistic = &statistics.Statistic{}
	bt.Funding = &fakeFunding{}
	bt.DataHolder = &fakeDataHolder{}
	err = bt.CloseAllPositions()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestRunLive(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = bt.RunLive()
	if !errors.Is(err, errLiveOnly) {
		t.Errorf("received '%v' expected '%v'", err, errLiveOnly)
	}

	em := engine.SetupExchangeManager()
	holder := &data.HandlerPerCurrency{}
	bt.LiveDataHandler, err = SetupLiveDataHandler(em, holder, -1, -1, -1, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = bt.LiveDataHandler.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = bt.RunLive()
		if !errors.Is(err, nil) {
			t.Errorf("received '%v' expected '%v'", err, nil)
		}
	}()
	close(bt.shutdown)
	wg.Wait()

	bt.shutdown = make(chan struct{})
	err = bt.LiveDataHandler.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	cp := currency.NewPair(currency.BTC, currency.USD)
	i := &gctkline.Item{
		Exchange:       testExchange,
		Pair:           cp,
		UnderlyingPair: cp,
		Asset:          asset.Spot,
		Interval:       gctkline.FifteenSecond,
		Candles: []gctkline.Candle{
			{
				Time:   time.Now(),
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err = bt.LiveDataHandler.AppendDataSource(i, &ftx.FTX{}, common.DataCandle)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.Reports = &report.Data{}
	bt.Funding = &fakeFunding{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = bt.RunLive()
		if !errors.Is(err, nil) {
			t.Errorf("received '%v' expected '%v'", err, nil)
		}
	}()
	bt.LiveDataHandler.Updated() <- struct{}{}
	close(bt.shutdown)
	wg.Wait()
}

// Overriding functions
// these are designed to override interface implementations
// so there is less requirement gathering per test as the functions are
// tested in their own package

type fakeDataHolder struct{}

func (f fakeDataHolder) Setup() {
}

func (f fakeDataHolder) SetDataForCurrency(s string, item asset.Item, pair currency.Pair, handler data.Handler) {
}

func (f fakeDataHolder) GetAllData() map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]data.Handler {
	return map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]data.Handler{
		testExchange: {
			asset.Spot: {
				currency.BTC.Item: map[*currency.Item]data.Handler{
					currency.USD.Item: &kline.DataFromKline{},
				},
			},
		},
	}
}

func (f fakeDataHolder) GetDataForCurrency(ev common.Event) (data.Handler, error) {
	return nil, nil
}

func (f fakeDataHolder) Reset() {}

type fakeFunding struct{}

func (f fakeFunding) Reset() {
}

func (f fakeFunding) IsUsingExchangeLevelFunding() bool {
	return true
}

func (f fakeFunding) GetFundingForEvent(c common.Event) (funding.IFundingPair, error) {
	return &funding.SpotPair{}, nil
}

func (f fakeFunding) Transfer(d decimal.Decimal, item *funding.Item, item2 *funding.Item, b bool) error {
	return nil
}

func (f fakeFunding) GenerateReport() *funding.Report {
	return nil
}

func (f fakeFunding) AddUSDTrackingData(fromKline *kline.DataFromKline) error {
	return nil
}

func (f fakeFunding) CreateSnapshot(t time.Time) {
}

func (f fakeFunding) USDTrackingDisabled() bool {
	return false
}

func (f fakeFunding) Liquidate(c common.Event) {
}

func (f fakeFunding) GetAllFunding() []funding.BasicItem {
	return nil
}

func (f fakeFunding) UpdateCollateral(c common.Event) error {
	return nil
}

func (f fakeFunding) HasFutures() bool {
	return false
}

func (f fakeFunding) HasExchangeBeenLiquidated(handler common.Event) bool {
	return false
}

func (f fakeFunding) RealisePNL(receivingExchange string, receivingAsset asset.Item, receivingCurrency currency.Code, realisedPNL decimal.Decimal) error {
	return nil
}

type fakeStrat struct{}

func (f fakeStrat) Name() string {
	return "fake"
}

func (f fakeStrat) Description() string {
	return "fake"
}

func (f fakeStrat) OnSignal(handler data.Handler, transferer funding.IFundingTransferer, handler2 portfolio.Handler) (signal.Event, error) {
	return nil, nil
}

func (f fakeStrat) OnSimultaneousSignals(handlers []data.Handler, transferer funding.IFundingTransferer, handler portfolio.Handler) ([]signal.Event, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeStrat) UsingSimultaneousProcessing() bool {
	return true
}

func (f fakeStrat) SupportsSimultaneousProcessing() bool {
	return true
}

func (f fakeStrat) SetSimultaneousProcessing(b bool) {}

func (f fakeStrat) SetCustomSettings(m map[string]interface{}) error {
	return nil
}

func (f fakeStrat) SetDefaults() {}

func (f fakeStrat) CloseAllPositions(i []holdings.Holding, events []data.Event) ([]signal.Event, error) {
	return []signal.Event{
		&signal.Signal{
			Base: &event.Base{
				Offset:         1,
				Exchange:       testExchange,
				Time:           time.Now(),
				Interval:       gctkline.FifteenSecond,
				CurrencyPair:   currency.NewPair(currency.BTC, currency.USD),
				UnderlyingPair: currency.NewPair(currency.BTC, currency.USD),
				AssetType:      asset.Spot,
			},
			OpenPrice:  leet,
			HighPrice:  leet,
			LowPrice:   leet,
			ClosePrice: leet,
			Volume:     leet,
			BuyLimit:   leet,
			SellLimit:  leet,
			Amount:     leet,
			Direction:  gctorder.Buy,
		},
	}, nil
}

type fakeFolio struct{}

func (f fakeFolio) SetHoldingsForOffset(holding *holdings.Holding, b bool) error {
	return nil
}

func (f fakeFolio) OnSignal(s signal.Event, settings *exchange.Settings, reserver funding.IFundReserver) (*order.Order, error) {
	return nil, nil
}

func (f fakeFolio) OnFill(f2 fill.Event, releaser funding.IFundReleaser) (fill.Event, error) {
	return nil, nil
}

func (f fakeFolio) GetLatestOrderSnapshotForEvent(c common.Event) (compliance.Snapshot, error) {
	return compliance.Snapshot{}, nil
}

func (f fakeFolio) GetLatestOrderSnapshots() ([]compliance.Snapshot, error) {
	return nil, nil
}

func (f fakeFolio) ViewHoldingAtTimePeriod(c common.Event) (*holdings.Holding, error) {
	return nil, nil
}

func (f fakeFolio) setHoldingsForOffset(holding *holdings.Holding, b bool) error {
	return nil
}

func (f fakeFolio) UpdateHoldings(d data.Event, releaser funding.IFundReleaser) error {
	return nil
}

func (f fakeFolio) GetComplianceManager(s string, item asset.Item, pair currency.Pair) (*compliance.Manager, error) {
	return nil, nil
}

func (f fakeFolio) GetPositions(c common.Event) ([]gctorder.PositionStats, error) {
	return nil, nil
}

func (f fakeFolio) TrackFuturesOrder(f2 fill.Event, releaser funding.IFundReleaser) (*portfolio.PNLSummary, error) {
	return nil, nil
}

func (f fakeFolio) UpdatePNL(c common.Event, d decimal.Decimal) error {
	return nil
}

func (f fakeFolio) GetLatestPNLForEvent(c common.Event) (*portfolio.PNLSummary, error) {
	return nil, nil
}

func (f fakeFolio) GetLatestPNLs() []portfolio.PNLSummary {
	return nil
}

func (f fakeFolio) CheckLiquidationStatus(d data.Event, reader funding.ICollateralReader, summary *portfolio.PNLSummary) error {
	return nil
}

func (f fakeFolio) CreateLiquidationOrdersForExchange(d data.Event, manager funding.IFundingManager) ([]order.Event, error) {
	return nil, nil
}

func (f fakeFolio) GetLatestHoldingsForAllCurrencies() []holdings.Holding {
	return nil
}

func (f fakeFolio) Reset() {}
