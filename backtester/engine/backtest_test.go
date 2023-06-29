package engine

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/binancecashandcarry"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binanceus"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binanceus"

var leet = decimal.NewFromInt(1337)

func TestSetupFromConfig(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
	err = bt.SetupFromConfig(nil, "", "", false)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("received %v, expected %v", err, errNilConfig)
	}
	cfg := &config.Config{}
	err = bt.SetupFromConfig(cfg, "", "", false)
	if !errors.Is(err, gctkline.ErrInvalidInterval) {
		t.Errorf("received: %v, expected: %v", err, gctkline.ErrInvalidInterval)
	}

	cfg.DataSettings.Interval = gctkline.OneMonth
	err = bt.SetupFromConfig(cfg, "", "", false)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}

	const testExchange = "bitfinex"

	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: testExchange,
			Base:         currency.BTC,
			Quote:        currency.USD,
			Asset:        asset.Spot,
		},
	}
	err = bt.SetupFromConfig(cfg, "", "", false)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}

	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]interface{}{
			"hello": "moto",
		},
	}
	cfg.DataSettings.APIData = &config.APIData{}

	err = bt.SetupFromConfig(cfg, "", "", false)
	if err != nil && !strings.Contains(err.Error(), "unrecognised dataType") {
		t.Error(err)
	}
	cfg.DataSettings.DataType = common.CandleStr
	err = bt.SetupFromConfig(cfg, "", "", false)
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrDateUnset)
	}
	cfg.DataSettings.Interval = gctkline.OneMin
	cfg.CurrencySettings[0].MakerFee = &decimal.Zero
	cfg.CurrencySettings[0].TakerFee = &decimal.Zero
	err = bt.SetupFromConfig(cfg, "", "", false)
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrDateUnset)
	}

	cfg.DataSettings.APIData.StartDate = time.Now().Truncate(gctkline.OneMin.Duration()).Add(-gctkline.OneMin.Duration() * 10)
	cfg.DataSettings.APIData.EndDate = cfg.DataSettings.APIData.StartDate.Add(gctkline.OneMin.Duration() * 5)
	cfg.DataSettings.APIData.InclusiveEndDate = true
	err = bt.SetupFromConfig(cfg, "", "", false)
	if !errors.Is(err, holdings.ErrInitialFundsZero) {
		t.Errorf("received: %v, expected: %v", err, holdings.ErrInitialFundsZero)
	}
	cfg.FundingSettings.UseExchangeLevelFunding = true
	cfg.FundingSettings.ExchangeLevelFunding = []config.ExchangeLevelFunding{
		{
			ExchangeName: testExchange,
			Asset:        asset.Spot,
			Currency:     currency.USD,
			InitialFunds: leet,
			TransferFee:  leet,
		},
	}
	err = bt.SetupFromConfig(cfg, "", "", false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestLoadDataAPI(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
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
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
			APIData: &config.APIData{
				StartDate: time.Now().Truncate(gctkline.OneMin.Duration()).Add(-time.Minute * 5),
				EndDate:   time.Now().Truncate(gctkline.OneMin.Duration()),
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
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataCSV(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
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

func TestLoadDataDatabase(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports:  &report.Data{},
		shutdown: make(chan struct{}),
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
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

func TestLoadDataLive(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports:         &fakeReport{},
		Funding:         &funding.FundManager{},
		DataHolder:      &data.HandlerHolder{},
		Statistic:       &fakeStats{},
		exchangeManager: engine.NewExchangeManager(),
		shutdown:        make(chan struct{}),
	}

	cp := currency.NewPair(currency.BTC, currency.USDT)
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
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: 1234,
			LiveData: &config.LiveData{
				ExchangeCredentials: []config.Credentials{
					{
						Exchange: testExchange,
						Keys: account.Credentials{
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
	exch, err := bt.exchangeManager.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = bt.SetupLiveDataHandler(0, 0, false, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}
	err = bt.LiveDataHandler.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if !errors.Is(err, gctkline.ErrCannotConstructInterval) {
		t.Errorf("received: %v, expected: %v", err, gctkline.ErrCannotConstructInterval)
	}

	cfg.DataSettings.Interval = gctkline.OneMin
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = bt.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, true, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	bt := &BackTest{
		shutdown:   make(chan struct{}),
		DataHolder: &data.HandlerHolder{},
		Strategy:   &dollarcostaverage.Strategy{},
		Portfolio:  &portfolio.Portfolio{},
		Exchange:   &exchange.Exchange{},
		Statistic:  &statistics.Statistic{},
		EventQueue: &eventholder.Holder{},
		Reports:    &report.Data{},
		Funding:    f,
	}
	err = bt.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if bt.Funding.IsUsingExchangeLevelFunding() {
		t.Error("expected false")
	}

	bt = nil
	err = bt.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestFullCycle(t *testing.T) {
	t.Parallel()
	ex := testExchange
	cp := currency.NewPair(currency.BTC, currency.USDT)
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
	fx := &binance.Binance{}
	fx.Name = testExchange
	err = port.SetCurrencySettingsMap(&exchange.Settings{Exchange: fx, Asset: a, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
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
		DataHolder:               &data.HandlerHolder{},
		Strategy:                 &dollarcostaverage.Strategy{},
		Portfolio:                port,
		Exchange:                 &exchange.Exchange{},
		Statistic:                stats,
		EventQueue:               &eventholder.Holder{},
		Reports:                  &report.Data{},
		hasProcessedDataAtOffset: make(map[int64]bool),
		Funding:                  f,
		shutdown:                 make(chan struct{}),
	}

	bt.DataHolder = data.NewHandlerHolder()
	k := &kline.DataFromKline{
		Item: &gctkline.Item{
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
		Base: &data.Base{},
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
	err = bt.DataHolder.SetDataForCurrency(ex, a, cp, k)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	bt.MetaData.DateLoaded = time.Now()
	err = bt.Run()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestStop(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown:  make(chan struct{}),
		Statistic: &fakeStats{},
		Reports:   &fakeReport{},
	}
	err := bt.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	tt := bt.MetaData.DateEnded

	err = bt.Stop()
	if !errors.Is(err, errAlreadyRan) {
		t.Errorf("received: %v, expected: %v", err, errAlreadyRan)
	}
	if !tt.Equal(bt.MetaData.DateEnded) {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.DateEnded, tt)
	}

	bt = nil
	err = bt.Stop()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
}

func TestFullCycleMulti(t *testing.T) {
	t.Parallel()
	ex := testExchange
	cp := currency.NewPair(currency.BTC, currency.USDT)
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
	err = port.SetCurrencySettingsMap(&exchange.Settings{Exchange: &binance.Binance{}, Asset: a, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
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
		DataHolder:               &data.HandlerHolder{},
		Portfolio:                port,
		Exchange:                 &exchange.Exchange{},
		Statistic:                stats,
		EventQueue:               &eventholder.Holder{},
		Reports:                  &report.Data{},
		Funding:                  f,
		hasProcessedDataAtOffset: make(map[int64]bool),
		shutdown:                 make(chan struct{}),
	}

	bt.Strategy, err = strategies.LoadStrategyByName(dollarcostaverage.Name, true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	bt.DataHolder = data.NewHandlerHolder()
	k := &kline.DataFromKline{
		Item: &gctkline.Item{
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
		Base: &data.Base{},
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

	err = bt.DataHolder.SetDataForCurrency(ex, a, cp, k)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = bt.Run()
	if !errors.Is(err, errNotSetup) {
		t.Errorf("received: %v, expected: %v", err, errNotSetup)
	}

	bt.MetaData.DateLoaded = time.Now()
	err = bt.Run()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

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

func TestTriggerLiquidationsForExchange(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		shutdown: make(chan struct{}),
	}
	expectedError := common.ErrNilEvent
	err := bt.triggerLiquidationsForExchange(nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.USDTMarginedFutures
	expectedError = gctcommon.ErrNilPointer
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
	bt.DataHolder = &data.HandlerHolder{}
	d := &data.Base{}
	err = d.SetStream([]data.Event{&evkline.Kline{
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
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	da := &kline.DataFromKline{
		Item: &gctkline.Item{
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
	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, da)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = bt.Statistic.SetEventForOffset(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	pnl.Exchange = ev.Exchange
	pnl.Asset = ev.AssetType
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
	bt := &BackTest{
		Statistic: &fakeStats{},
		Funding:   &funding.FundManager{},
		Portfolio: &fakeFolio{},
		shutdown:  make(chan struct{}),
	}
	expectedError := common.ErrNilEvent
	err := bt.updateStatsForDataEvent(nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Futures
	ev := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp,
		},
	}

	expectedError = gctcommon.ErrNilPointer
	err = bt.updateStatsForDataEvent(ev, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	expectedError = nil
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
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
	exch := &binance.Binance{}
	exch.Name = testExchange
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
	_, err = bt.Portfolio.TrackFuturesOrder(fl, pair)
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
	bt := &BackTest{
		Statistic:  &fakeStats{},
		Funding:    &funding.FundManager{},
		Portfolio:  &fakeFolio{},
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		shutdown:   make(chan struct{}),
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.USDTMarginedFutures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err := bt.Statistic.SetEventForOffset(de)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &signal.Signal{
		Base: de.Base,
	}

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
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
	exch := &binance.Binance{}
	exch.Name = testExchange
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
		DataHolder: &data.HandlerHolder{},
		shutdown:   make(chan struct{}),
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.USDTMarginedFutures
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

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
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
	exch := &binance.Binance{}
	exch.Name = testExchange
	err = pt.SetCurrencySettingsMap(&exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNotYetImplemented)
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
	bt.DataHolder = data.NewHandlerHolder()
	k := &kline.DataFromKline{
		Item: &gctkline.Item{
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
		Base: &data.Base{},
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

	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, k)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	bt := &BackTest{
		Statistic:  &fakeStats{},
		Funding:    &fakeFunding{},
		Portfolio:  &fakeFolio{},
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		shutdown:   make(chan struct{}),
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Futures
	tt := time.Now()
	de := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp,
			Time:         tt,
		},
	}
	err := bt.Statistic.SetEventForOffset(de)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	ev := &fill.Fill{
		Base: de.Base,
	}
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pair, err := funding.CreateCollateral(b, quote)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.DataHolder = data.NewHandlerHolder()
	k := &kline.DataFromKline{
		Item: &gctkline.Item{
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
		Base: &data.Base{},
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

	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, k)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = bt.processFillEvent(ev, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestProcessFuturesFillEvent(t *testing.T) {
	t.Parallel()
	var expectedError error
	bt := &BackTest{
		Statistic:  &fakeStats{},
		Funding:    &funding.FundManager{},
		Portfolio:  &fakeFolio{},
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		shutdown:   make(chan struct{}),
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err := bt.Statistic.SetEventForOffset(de)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &fill.Fill{
		Base: de.Base,
	}
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
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

	bt.exchangeManager = em
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
	bt.DataHolder = data.NewHandlerHolder()
	k := &kline.DataFromKline{
		Item: &gctkline.Item{
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
		Base: &data.Base{},
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
	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, k)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = bt.processFuturesFillEvent(ev, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
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
	dc := &dataChecker{
		realOrders: true,
		shutdown:   make(chan bool),
	}
	bt.LiveDataHandler = dc
	err = bt.CloseAllPositions()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.shutdown = make(chan struct{})
	bt.Strategy = &binancecashandcarry.Strategy{}
	err = bt.CloseAllPositions()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.shutdown = make(chan struct{})
	bt.Portfolio = &fakeFolio{}
	bt.Strategy = &fakeStrat{}
	bt.Exchange = &exchange.Exchange{}
	bt.Statistic = &fakeStats{}
	bt.Reports = &fakeReport{}
	bt.Funding = &fakeFunding{}
	bt.DataHolder = &fakeDataHolder{}
	dc.dataHolder = bt.DataHolder
	dc.report = &report.Data{}
	dc.funding = bt.Funding
	cp := currency.NewPair(currency.BTC, currency.USD)
	dc.sourcesToCheck = append(dc.sourcesToCheck, &liveDataSourceDataHandler{
		exchange:                  &binance.Binance{},
		exchangeName:              testExchange,
		asset:                     asset.Spot,
		pair:                      cp,
		underlyingPair:            cp,
		dataType:                  common.DataCandle,
		dataRequestRetryTolerance: 1,
		pairCandles: &kline.DataFromKline{
			Base: &data.Base{},
			Item: &gctkline.Item{
				Exchange:       testExchange,
				Pair:           cp,
				UnderlyingPair: cp,
				Asset:          asset.Spot,
				Interval:       gctkline.OneMin,
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
			},
		},
	})
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

	bt.Funding = &funding.FundManager{}
	bt.Reports = &report.Data{}

	dc := &dataChecker{
		exchangeManager:   bt.exchangeManager,
		eventTimeout:      defaultEventTimeout,
		dataCheckInterval: defaultDataCheckInterval,
		dataHolder:        bt.DataHolder,
		report:            bt.Reports,
		funding:           bt.Funding,
		shutdown:          make(chan bool),
	}
	bt.LiveDataHandler = dc
	err = bt.RunLive()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	close(bt.shutdown)
	bt.wg.Wait()
	bt.shutdown = make(chan struct{})

	dc = &dataChecker{
		exchangeManager:   bt.exchangeManager,
		eventTimeout:      defaultEventTimeout,
		dataCheckInterval: defaultDataCheckInterval,
		dataHolder:        bt.DataHolder,
		report:            bt.Reports,
		shutdown:          make(chan bool),
		dataUpdated:       make(chan bool),
		shutdownErr:       make(chan bool),
		funding:           bt.Funding,
	}
	bt.LiveDataHandler = dc
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
	// 	AppendDataSource(exchange gctexchange.IBotExchange, interval gctkline.Interval, asset asset.Asset, pair, underlyingPair currency.Pair, dataType int64) error
	setup := &liveDataSourceSetup{
		exchange:       &binance.Binance{},
		interval:       i.Interval,
		asset:          i.Asset,
		pair:           i.Pair,
		underlyingPair: i.UnderlyingPair,
		dataType:       common.DataCandle,
	}
	err = dc.AppendDataSource(setup)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.Reports = &report.Data{}
	bt.Funding = &fakeFunding{}
	bt.Statistic = &fakeStats{}
	dc.started = 0
	err = bt.RunLive()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestLiveLoop(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.Reports = &fakeReport{}
	bt.Funding = &fakeFunding{}
	bt.Statistic = &fakeStats{}

	dc := &dataChecker{
		dataUpdated: make(chan bool),
		shutdownErr: make(chan bool),
		shutdown:    make(chan bool),
	}
	bt.LiveDataHandler = dc

	// dataUpdated case
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err = bt.liveCheck()
		if !errors.Is(err, nil) {
			t.Errorf("received '%v' expected '%v'", err, nil)
		}
		wg.Done()
	}()
	dc.dataUpdated <- true
	dc.shutdown <- true
	wg.Wait()

	// shutdown from error case
	wg.Add(1)
	dc.started = 0
	go func() {
		defer wg.Done()
		err = bt.liveCheck()
		if !errors.Is(err, nil) {
			t.Errorf("received '%v' expected '%v'", err, nil)
		}
	}()
	dc.shutdownErr <- true
	wg.Wait()

	// shutdown case
	dc.started = 1
	bt.shutdown = make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = bt.liveCheck()
		if !errors.Is(err, nil) {
			t.Errorf("received '%v' expected '%v'", err, nil)
		}
	}()
	dc.shutdown <- true
	wg.Wait()

	// backtester has shutdown
	wg.Add(1)
	bt.shutdown = make(chan struct{})
	go func() {
		defer wg.Done()
		err = bt.liveCheck()
		if !errors.Is(err, nil) {
			t.Errorf("received '%v' expected '%v'", err, nil)
		}
	}()
	close(bt.shutdown)
	wg.Wait()
}

func TestSetExchangeCredentials(t *testing.T) {
	t.Parallel()
	err := setExchangeCredentials(nil, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	cfg := &config.Config{}
	f := &binanceus.Binanceus{}
	f.SetDefaults()
	b := f.GetBase()
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	ld := &config.LiveData{}
	cfg.DataSettings = config.DataSettings{
		LiveData: ld,
	}
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	ld.RealOrders = true
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, errIntervalUnset) {
		t.Errorf("received '%v' expected '%v'", err, errIntervalUnset)
	}

	cfg.DataSettings.Interval = gctkline.OneMin
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, errNoCredsNoLive) {
		t.Errorf("received '%v' expected '%v'", err, errNoCredsNoLive)
	}

	cfg.DataSettings.LiveData.ExchangeCredentials = []config.Credentials{{}}
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, gctexchange.ErrCredentialsAreEmpty) {
		t.Errorf("received '%v' expected '%v'", err, gctexchange.ErrCredentialsAreEmpty)
	}

	// requires valid credentials here to get complete coverage
	// enter them here
	cfg.DataSettings.LiveData.ExchangeCredentials = []config.Credentials{{
		Exchange: testExchange,
		Keys: account.Credentials{
			Key:    "test",
			Secret: "test",
		},
	}}
	err = setExchangeCredentials(cfg, b)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestGetFees(t *testing.T) {
	t.Parallel()
	_, _, err := getFees(context.Background(), nil, currency.EMPTYPAIR)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	f := &binance.Binance{}
	f.SetDefaults()
	_, _, err = getFees(context.Background(), f, currency.EMPTYPAIR)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	maker, taker, err := getFees(context.Background(), f, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}
	if maker.IsZero() || taker.IsZero() {
		t.Error("expected maker and taker fees")
	}
}

func TestGenerateSummary(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown: make(chan struct{}),
	}
	sum, err := bt.GenerateSummary()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !sum.MetaData.ID.IsNil() {
		t.Errorf("received '%v' expected '%v'", sum.MetaData.ID, "")
	}
	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.MetaData.ID = id
	sum, err = bt.GenerateSummary()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if sum.MetaData.ID != id {
		t.Errorf("received '%v' expected '%v'", sum.MetaData.ID, id)
	}

	bt = nil
	_, err = bt.GenerateSummary()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestSetupMetaData(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown: make(chan struct{}),
	}
	err := bt.SetupMetaData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if bt.MetaData.ID.IsNil() {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.ID, "an ID")
	}
	firstID := bt.MetaData.ID
	err = bt.SetupMetaData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if bt.MetaData.ID != firstID {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.ID, firstID)
	}

	bt = nil
	err = bt.SetupMetaData()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown: make(chan struct{}),
	}
	if bt.IsRunning() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	if !bt.IsRunning() {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	bt.m.Lock()
	bt.MetaData.Closed = true
	bt.m.Unlock()
	if bt.IsRunning() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	bt = nil
	if bt.IsRunning() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestHasRan(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown: make(chan struct{}),
	}
	if bt.HasRan() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	if bt.HasRan() {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	bt.m.Lock()
	bt.MetaData.Closed = true
	bt.m.Unlock()
	if !bt.HasRan() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	bt = nil
	if bt.HasRan() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestEqual(t *testing.T) {
	t.Parallel()
	bt := &BackTest{}
	bt2 := &BackTest{}
	bt3 := &BackTest{}
	if !bt.Equal(bt2) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	err := bt.SetupMetaData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt2.MetaData = bt.MetaData
	if !bt.Equal(bt2) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	if bt.Equal(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	err = bt3.SetupMetaData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if bt.Equal(bt3) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	bt = nil
	if bt.Equal(bt2) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestMatchesID(t *testing.T) {
	t.Parallel()
	bt := &BackTest{}
	if bt.MatchesID(uuid.Nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	err := bt.SetupMetaData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if bt.MatchesID(uuid.Nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	if !bt.MatchesID(bt.MetaData.ID) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	id := bt.MetaData.ID
	bt.MetaData.ID = uuid.Nil
	if bt.MatchesID(id) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	bt = nil
	if bt.MatchesID(id) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestExecuteStrategy(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		DataHolder: &fakeDataHolder{},
		Strategy:   &fakeStrat{},
		Portfolio:  &fakeFolio{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		Funding:    &fakeFunding{},
		EventQueue: &eventholder.Holder{},
		shutdown:   make(chan struct{}),
	}
	err := bt.ExecuteStrategy(false)
	if !errors.Is(err, errNotSetup) {
		t.Errorf("received '%v' expected '%v'", err, errNotSetup)
	}
	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.m.Lock()
	bt.MetaData.ID = id
	bt.MetaData.DateLoaded = time.Now()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	err = bt.ExecuteStrategy(false)
	if !errors.Is(err, errTaskIsRunning) {
		t.Errorf("received '%v' expected '%v'", err, errTaskIsRunning)
	}

	err = bt.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = bt.ExecuteStrategy(true)
	if !errors.Is(err, errAlreadyRan) {
		t.Errorf("received '%v' expected '%v'", err, errAlreadyRan)
	}

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Time{}
	bt.MetaData.DateEnded = time.Time{}
	bt.MetaData.Closed = false
	bt.shutdown = make(chan struct{})
	bt.m.Unlock()

	err = bt.ExecuteStrategy(true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Time{}
	bt.MetaData.DateEnded = time.Time{}
	bt.MetaData.Closed = false
	bt.shutdown = make(chan struct{})
	bt.m.Unlock()
	err = bt.ExecuteStrategy(false)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}

	bt.m.Lock()
	bt.MetaData.LiveTesting = true
	bt.MetaData.DateStarted = time.Time{}
	bt.MetaData.DateEnded = time.Time{}
	bt.MetaData.Closed = false
	bt.shutdown = make(chan struct{})
	bt.m.Unlock()
	err = bt.ExecuteStrategy(true)
	if !errors.Is(err, errCannotHandleRequest) {
		t.Errorf("received '%v' expected '%v'", err, errCannotHandleRequest)
	}

	err = bt.ExecuteStrategy(false)
	if !errors.Is(err, errLiveOnly) {
		t.Errorf("received '%v' expected '%v'", err, errLiveOnly)
	}

	bt = nil
	err = bt.ExecuteStrategy(false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestNewBacktesterFromConfigs(t *testing.T) {
	t.Parallel()
	_, err := NewBacktesterFromConfigs(nil, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	strat1 := filepath.Join("..", "config", "strategyexamples", "dca-api-candles.strat")
	cfg, err := config.ReadStrategyConfigFromFile(strat1)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	dc, err := config.GenerateDefaultConfig()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = NewBacktesterFromConfigs(cfg, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	_, err = NewBacktesterFromConfigs(nil, dc)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt, err := NewBacktesterFromConfigs(cfg, dc)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	if bt.MetaData.DateLoaded.IsZero() {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.DateLoaded, "a date")
	}
}

func TestProcessSingleDataEvent(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		Strategy:   &fakeStrat{},
		Portfolio:  &fakeFolio{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		Funding:    &fakeFunding{},
		DataHolder: &data.HandlerHolder{},
		EventQueue: &eventholder.Holder{},
	}

	err := bt.processSingleDataEvent(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilEvent)
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Spot
	ev := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			Time:         time.Now(),
			Interval:     gctkline.FifteenMin,
			CurrencyPair: cp,
			AssetType:    a,
		},
	}
	err = bt.processSingleDataEvent(ev, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	collateral, err := funding.CreateCollateral(b, quote)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	bt.Funding = f
	tt := time.Now()
	bt.DataHolder = data.NewHandlerHolder()
	k := &kline.DataFromKline{
		Item: &gctkline.Item{
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
		Base: &data.Base{},
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
	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, k)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = bt.processSingleDataEvent(ev, collateral)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}
