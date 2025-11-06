package engine

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	require.NoError(t, err)

	err = bt.SetupFromConfig(nil, "", "", false)
	assert.ErrorIs(t, err, errNilConfig)

	cfg := &config.Config{}
	err = bt.SetupFromConfig(cfg, "", "", false)
	assert.ErrorIs(t, err, gctkline.ErrInvalidInterval)

	cfg.DataSettings.Interval = gctkline.OneMonth
	err = bt.SetupFromConfig(cfg, "", "", false)
	assert.ErrorIs(t, err, base.ErrStrategyNotFound)

	const testExchange = "okx"

	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: testExchange,
			Base:         currency.BTC,
			Quote:        currency.USD,
			Asset:        asset.Spot,
		},
	}
	err = bt.SetupFromConfig(cfg, "", "", false)
	assert.ErrorIs(t, err, base.ErrStrategyNotFound)

	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]any{
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
	assert.ErrorIs(t, err, gctcommon.ErrDateUnset)

	cfg.DataSettings.Interval = gctkline.OneMin
	cfg.CurrencySettings[0].MakerFee = &decimal.Zero
	cfg.CurrencySettings[0].TakerFee = &decimal.Zero
	err = bt.SetupFromConfig(cfg, "", "", false)
	assert.ErrorIs(t, err, gctcommon.ErrDateUnset)

	cfg.DataSettings.APIData.StartDate = time.Now().Truncate(gctkline.OneMin.Duration()).Add(-gctkline.OneMin.Duration() * 10)
	cfg.DataSettings.APIData.EndDate = cfg.DataSettings.APIData.StartDate.Add(gctkline.OneMin.Duration() * 5)
	cfg.DataSettings.APIData.InclusiveEndDate = true
	err = bt.SetupFromConfig(cfg, "", "", false)
	assert.ErrorIs(t, err, holdings.ErrInitialFundsZero)

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
	assert.NoError(t, err)
}

func TestLoadDataAPI(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewBTCUSDT()
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
			},
		},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]any{
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
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}

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
	cp := currency.NewBTCUSDT()
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
			},
		},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]any{
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
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
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
	cp := currency.NewBTCUSDT()
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
			},
		},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]any{
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
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
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

	cp := currency.NewBTCUSDT()
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
						Keys: accounts.Credentials{
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
			},
		},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]any{
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
	require.NoError(t, err)

	err = bt.LiveDataHandler.Start()
	assert.NoError(t, err)

	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	assert.ErrorIs(t, err, gctkline.ErrCannotConstructInterval)

	cfg.DataSettings.Interval = gctkline.OneMin
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	assert.NoError(t, err)

	err = bt.Stop()
	assert.NoError(t, err)
}

func TestReset(t *testing.T) {
	t.Parallel()
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, true, false, false)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	if bt.Funding.IsUsingExchangeLevelFunding() {
		t.Error("expected false")
	}

	bt = nil
	err = bt.Reset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestFullCycle(t *testing.T) {
	t.Parallel()
	e := testExchange
	cp := currency.NewBTCUSDT()
	a := asset.Spot
	tt := time.Now()

	stats := &statistics.Statistic{}
	stats.ExchangeAssetPairStatistics = make(map[key.ExchangeAssetPair]*statistics.CurrencyPairStatistic)
	port, err := portfolio.Setup(&size.Size{
		BuySide:  exchange.MinMax{},
		SellSide: exchange.MinMax{},
	}, &risk.Risk{}, decimal.Zero)
	assert.NoError(t, err)

	fx := &binance.Exchange{}
	fx.Name = testExchange
	err = port.SetCurrencySettingsMap(&exchange.Settings{Exchange: fx, Asset: a, Pair: cp})
	assert.NoError(t, err)

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	assert.NoError(t, err)

	b, err := funding.CreateItem(e, a, cp.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quote, err := funding.CreateItem(e, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	assert.NoError(t, err)

	pair, err := funding.CreatePair(b, quote)
	assert.NoError(t, err)

	err = f.AddPair(pair)
	assert.NoError(t, err)

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
			Exchange: e,
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
	assert.NoError(t, err)

	err = bt.DataHolder.SetDataForCurrency(e, a, cp, k)
	assert.NoError(t, err)

	bt.MetaData.DateLoaded = time.Now()
	err = bt.Run()
	assert.NoError(t, err)
}

func TestStop(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown:  make(chan struct{}),
		Statistic: &fakeStats{},
		Reports:   &fakeReport{},
	}
	err := bt.Stop()
	require.NoError(t, err)

	tt := bt.MetaData.DateEnded

	err = bt.Stop()
	assert.ErrorIs(t, err, errAlreadyRan)

	if !tt.Equal(bt.MetaData.DateEnded) {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.DateEnded, tt)
	}

	bt = nil
	err = bt.Stop()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestFullCycleMulti(t *testing.T) {
	t.Parallel()
	e := testExchange
	cp := currency.NewBTCUSDT()
	a := asset.Spot
	tt := time.Now()

	stats := &statistics.Statistic{}
	stats.ExchangeAssetPairStatistics = make(map[key.ExchangeAssetPair]*statistics.CurrencyPairStatistic)

	port, err := portfolio.Setup(&size.Size{
		BuySide:  exchange.MinMax{},
		SellSide: exchange.MinMax{},
	}, &risk.Risk{}, decimal.Zero)
	assert.NoError(t, err)

	err = port.SetCurrencySettingsMap(&exchange.Settings{Exchange: &binance.Exchange{}, Asset: a, Pair: cp})
	assert.NoError(t, err)

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	assert.NoError(t, err)

	b, err := funding.CreateItem(e, a, cp.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quote, err := funding.CreateItem(e, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	assert.NoError(t, err)

	pair, err := funding.CreatePair(b, quote)
	assert.NoError(t, err)

	err = f.AddPair(pair)
	assert.NoError(t, err)

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
	require.NoError(t, err)

	bt.DataHolder = data.NewHandlerHolder()
	k := &kline.DataFromKline{
		Item: &gctkline.Item{
			Exchange: e,
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
	assert.NoError(t, err)

	err = bt.DataHolder.SetDataForCurrency(e, a, cp, k)
	assert.NoError(t, err)

	err = bt.Run()
	assert.ErrorIs(t, err, errNotSetup)

	bt.MetaData.DateLoaded = time.Now()
	err = bt.Run()
	assert.NoError(t, err)
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
	err := bt.triggerLiquidationsForExchange(nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	cp := currency.NewBTCUSDT()
	a := asset.USDTMarginedFutures
	ev := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp,
		},
	}
	err = bt.triggerLiquidationsForExchange(ev, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

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
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

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

	bt.EventQueue = &eventholder.Holder{}
	bt.Funding = &funding.FundManager{}
	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, da)
	assert.NoError(t, err)

	err = bt.Statistic.SetEventForOffset(ev)
	assert.NoError(t, err, "SetEventForOffset should not error")

	pnl.Exchange = ev.Exchange
	pnl.Asset = ev.AssetType
	pnl.Pair = ev.CurrencyPair
	err = bt.triggerLiquidationsForExchange(ev, pnl)
	assert.NoError(t, err, "triggerLiquidationsForExchange should not error")

	ev2 := bt.EventQueue.NextEvent()
	ev2o, ok := ev2.(order.Event)
	require.True(t, ok, "NextEvent must return an order event")
	assert.Equal(t, gctorder.Short, ev2o.GetDirection())
}

func TestUpdateStatsForDataEvent(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		Statistic: &fakeStats{},
		Funding:   &funding.FundManager{},
		Portfolio: &fakeFolio{},
		shutdown:  make(chan struct{}),
	}

	err := bt.updateStatsForDataEvent(nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	cp := currency.NewBTCUSDT()
	a := asset.Futures
	ev := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp,
		},
	}

	err = bt.updateStatsForDataEvent(ev, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	require.NoError(t, err, "SetupFundingManager must not error")

	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	pair, err := funding.CreateCollateral(b, quote)
	require.NoError(t, err, "CreateCollateral must not error")

	bt.Funding = f
	exch := &binance.Exchange{}
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
	assert.NoError(t, err, "TrackFuturesOrder should not error")

	err = bt.updateStatsForDataEvent(ev, pair)
	assert.NoError(t, err, "updateStatsForDataEvent should not error")
}

func TestProcessSignalEvent(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		Statistic:  &fakeStats{},
		Funding:    &funding.FundManager{},
		Portfolio:  &fakeFolio{},
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		shutdown:   make(chan struct{}),
	}
	cp := currency.NewBTCUSDT()
	a := asset.USDTMarginedFutures
	de := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp,
		},
	}
	err := bt.Statistic.SetEventForOffset(de)
	require.NoError(t, err, "SetEventForOffset must not error")

	ev := &signal.Signal{
		Base: de.Base,
	}

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	require.NoError(t, err, "SetupFundingManager must not error")

	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	pair, err := funding.CreateCollateral(b, quote)
	require.NoError(t, err, "CreateCollateral must not error")

	bt.Funding = f
	exch := &binance.Exchange{}
	exch.Name = testExchange
	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	require.NoError(t, err, "SetEventForOffset must not error")

	err = bt.processSignalEvent(ev, pair)
	assert.NoError(t, err, "processSignalEvent should not error")
}

func TestProcessOrderEvent(t *testing.T) {
	t.Parallel()
	pt, err := portfolio.Setup(&size.Size{}, &risk.Risk{}, decimal.Zero)
	require.NoError(t, err, "Setup must not error")

	bt := &BackTest{
		Statistic:  &statistics.Statistic{},
		Funding:    &funding.FundManager{},
		Portfolio:  pt,
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		shutdown:   make(chan struct{}),
	}
	cp := currency.NewBTCUSDT()
	a := asset.USDTMarginedFutures
	de := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp,
		},
	}
	err = bt.Statistic.SetEventForOffset(de)
	require.NoError(t, err, "SetEventForOffset must not error")

	ev := &order.Order{
		Base: de.Base,
	}

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	require.NoError(t, err, "SetupFundingManager must not error")

	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	pair, err := funding.CreateCollateral(b, quote)
	require.NoError(t, err, "CreateCollateral must not error")

	bt.Funding = f
	exch := &binance.Exchange{}
	exch.Name = testExchange
	err = pt.SetCurrencySettingsMap(&exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	require.NoError(t, err, "SetEventForOffset must not error")

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
	assert.NoError(t, err)

	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, k)
	assert.NoError(t, err)

	err = bt.processOrderEvent(ev, pair)
	require.NoError(t, err, "processOrderEvent must not error")

	ev2 := bt.EventQueue.NextEvent()
	_, ok := ev2.(fill.Event)
	require.True(t, ok, "NextEvent must return a fill event")
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
	cp := currency.NewBTCUSDT()
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
	assert.NoError(t, err)

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
	require.NoError(t, err)

	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	assert.NoError(t, err)

	pair, err := funding.CreateCollateral(b, quote)
	assert.NoError(t, err)

	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, k)
	assert.NoError(t, err)

	err = bt.processFillEvent(ev, pair)
	assert.NoError(t, err)
}

func TestProcessFuturesFillEvent(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		Statistic:  &fakeStats{},
		Funding:    &funding.FundManager{},
		Portfolio:  &fakeFolio{},
		Exchange:   &exchange.Exchange{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		shutdown:   make(chan struct{}),
	}
	cp := currency.NewBTCUSDT()
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{
			Exchange:     testExchange,
			AssetType:    a,
			CurrencyPair: cp,
		},
	}
	err := bt.Statistic.SetEventForOffset(de)
	require.NoError(t, err, "SetEventForOffset must note error")

	ev := &fill.Fill{
		Base: de.Base,
	}
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	pair, err := funding.CreateCollateral(b, quote)
	require.NoError(t, err, "CreateCollateral must not error")

	bt.exchangeManager = em
	bt.Exchange.SetExchangeAssetCurrencySettings(a, cp, &exchange.Settings{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
	})
	ev.Direction = gctorder.Short
	err = bt.Statistic.SetEventForOffset(ev)
	require.NoError(t, err, "SetEventForOffset must not error")

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
	require.NoError(t, err)

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
	require.NoError(t, err)

	err = bt.processFuturesFillEvent(ev, pair)
	assert.NoError(t, err, "processFuturesFillEvent should not error")
}

func TestCloseAllPositions(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	assert.NoError(t, err)

	pt := &portfolio.Portfolio{}
	bt.Portfolio = pt
	bt.Strategy = &dollarcostaverage.Strategy{}

	err = bt.CloseAllPositions()
	assert.ErrorIs(t, err, errLiveOnly)

	bt.shutdown = make(chan struct{})
	dc := &dataChecker{
		realOrders: true,
		shutdown:   make(chan bool),
	}
	bt.LiveDataHandler = dc
	err = bt.CloseAllPositions()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt.shutdown = make(chan struct{})
	bt.Strategy = &binancecashandcarry.Strategy{}
	err = bt.CloseAllPositions()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

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
	cp := currency.NewBTCUSD()
	dc.sourcesToCheck = append(dc.sourcesToCheck, &liveDataSourceDataHandler{
		exchange:                  &binance.Exchange{},
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
	assert.NoError(t, err)
}

func TestRunLive(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	assert.NoError(t, err)

	err = bt.RunLive()
	assert.ErrorIs(t, err, errLiveOnly)

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
	assert.NoError(t, err)

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
	cp := currency.NewBTCUSD()
	i := &gctkline.Item{
		Pair:           cp,
		UnderlyingPair: cp,
		Asset:          asset.Spot,
		Interval:       gctkline.FifteenSecond,
	}
	// 	AppendDataSource(exchange gctexchange.IBotExchange, interval gctkline.Interval, asset asset.Asset, pair, underlyingPair currency.Pair, dataType int64) error
	setup := &liveDataSourceSetup{
		exchange:       &binance.Exchange{},
		interval:       i.Interval,
		asset:          i.Asset,
		pair:           i.Pair,
		underlyingPair: i.UnderlyingPair,
		dataType:       common.DataCandle,
	}
	err = dc.AppendDataSource(setup)
	assert.NoError(t, err)

	bt.Reports = &report.Data{}
	bt.Funding = &fakeFunding{}
	bt.Statistic = &fakeStats{}
	dc.started = 0
	err = bt.RunLive()
	assert.NoError(t, err)
}

func TestLiveLoop(t *testing.T) {
	t.Parallel()
	bt, err := NewBacktester()
	assert.NoError(t, err)

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
	wg.Go(func() {
		assert.NoError(t, bt.liveCheck())
	})
	dc.dataUpdated <- true
	dc.shutdown <- true
	wg.Wait()

	// shutdown from error case
	dc.started = 0
	wg.Go(func() {
		assert.NoError(t, bt.liveCheck())
	})
	dc.shutdownErr <- true
	wg.Wait()

	// shutdown case
	dc.started = 1
	bt.shutdown = make(chan struct{})
	wg.Go(func() {
		assert.NoError(t, bt.liveCheck())
	})
	dc.shutdown <- true
	wg.Wait()

	// backtester has shutdown
	wg.Add(1)
	bt.shutdown = make(chan struct{})
	go func() {
		defer wg.Done()
		err = bt.liveCheck()
		assert.NoError(t, err)
	}()
	close(bt.shutdown)
	wg.Wait()
}

func TestSetExchangeCredentials(t *testing.T) {
	t.Parallel()
	err := setExchangeCredentials(nil, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	cfg := &config.Config{}
	f := &binanceus.Exchange{}
	f.SetDefaults()
	b := f.GetBase()
	err = setExchangeCredentials(cfg, b)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	ld := &config.LiveData{}
	cfg.DataSettings = config.DataSettings{
		LiveData: ld,
	}
	err = setExchangeCredentials(cfg, b)
	assert.NoError(t, err)

	ld.RealOrders = true
	err = setExchangeCredentials(cfg, b)
	assert.ErrorIs(t, err, errIntervalUnset)

	cfg.DataSettings.Interval = gctkline.OneMin
	err = setExchangeCredentials(cfg, b)
	assert.ErrorIs(t, err, errNoCredsNoLive)

	cfg.DataSettings.LiveData.ExchangeCredentials = []config.Credentials{{}}
	err = setExchangeCredentials(cfg, b)
	assert.ErrorIs(t, err, gctexchange.ErrCredentialsAreEmpty)

	// requires valid credentials here to get complete coverage
	// enter them here
	cfg.DataSettings.LiveData.ExchangeCredentials = []config.Credentials{{
		Exchange: testExchange,
		Keys: accounts.Credentials{
			Key:    "test",
			Secret: "test",
		},
	}}
	err = setExchangeCredentials(cfg, b)
	assert.NoError(t, err)
}

func TestGetFees(t *testing.T) {
	t.Parallel()
	_, _, err := getFees(t.Context(), nil, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	f := &binance.Exchange{}
	f.SetDefaults()
	_, _, err = getFees(t.Context(), f, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	maker, taker, err := getFees(t.Context(), f, currency.NewBTCUSDT())
	assert.NoError(t, err, "getFees should not error")
	assert.NotZero(t, maker, "getFees should return a non-zero maker fee")
	assert.NotZero(t, taker, "getFees should return a non-zero taker fee")
}

func TestGenerateSummary(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown: make(chan struct{}),
	}
	sum, err := bt.GenerateSummary()
	assert.NoError(t, err)

	if !sum.MetaData.ID.IsNil() {
		t.Errorf("received '%v' expected '%v'", sum.MetaData.ID, "")
	}
	id, err := uuid.NewV4()
	assert.NoError(t, err)

	bt.MetaData.ID = id
	sum, err = bt.GenerateSummary()
	assert.NoError(t, err)

	if sum.MetaData.ID != id {
		t.Errorf("received '%v' expected '%v'", sum.MetaData.ID, id)
	}

	bt = nil
	_, err = bt.GenerateSummary()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestSetupMetaData(t *testing.T) {
	t.Parallel()
	bt := &BackTest{
		shutdown: make(chan struct{}),
	}
	err := bt.SetupMetaData()
	assert.NoError(t, err)

	if bt.MetaData.ID.IsNil() {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.ID, "an ID")
	}
	firstID := bt.MetaData.ID
	err = bt.SetupMetaData()
	assert.NoError(t, err)

	if bt.MetaData.ID != firstID {
		t.Errorf("received '%v' expected '%v'", bt.MetaData.ID, firstID)
	}

	bt = nil
	err = bt.SetupMetaData()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
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
	assert.NoError(t, err)

	bt2.MetaData = bt.MetaData
	if !bt.Equal(bt2) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	if bt.Equal(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	err = bt3.SetupMetaData()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, errNotSetup)

	id, err := uuid.NewV4()
	assert.NoError(t, err)

	bt.m.Lock()
	bt.MetaData.ID = id
	bt.MetaData.DateLoaded = time.Now()
	bt.MetaData.DateStarted = time.Now()
	bt.m.Unlock()
	err = bt.ExecuteStrategy(false)
	assert.ErrorIs(t, err, errTaskIsRunning)

	err = bt.Stop()
	assert.NoError(t, err)

	err = bt.ExecuteStrategy(true)
	assert.ErrorIs(t, err, errAlreadyRan)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Time{}
	bt.MetaData.DateEnded = time.Time{}
	bt.MetaData.Closed = false
	bt.shutdown = make(chan struct{})
	bt.m.Unlock()

	err = bt.ExecuteStrategy(true)
	assert.NoError(t, err)

	bt.m.Lock()
	bt.MetaData.DateStarted = time.Time{}
	bt.MetaData.DateEnded = time.Time{}
	bt.MetaData.Closed = false
	bt.shutdown = make(chan struct{})
	bt.m.Unlock()
	err = bt.ExecuteStrategy(false)
	require.NoError(t, err)

	// Wait for the async goroutine to complete before proceeding
	// to avoid race condition where Stop() is called after we reset metadata
	require.Eventually(t, bt.HasRan, time.Second, 10*time.Millisecond, "async goroutine must complete")

	bt.m.Lock()
	bt.MetaData.LiveTesting = true
	bt.MetaData.DateStarted = time.Time{}
	bt.MetaData.DateEnded = time.Time{}
	bt.MetaData.Closed = false
	bt.shutdown = make(chan struct{})
	bt.m.Unlock()
	err = bt.ExecuteStrategy(true)
	assert.ErrorIs(t, err, errCannotHandleRequest)

	err = bt.ExecuteStrategy(false)
	assert.ErrorIs(t, err, errLiveOnly)

	bt = nil
	err = bt.ExecuteStrategy(false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestNewBacktesterFromConfigs(t *testing.T) {
	t.Parallel()
	_, err := NewBacktesterFromConfigs(nil, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "NewBacktesterFromConfigs should error on nil for both configs")

	cfg, err := config.ReadStrategyConfigFromFile(filepath.Join("..", "config", "strategyexamples", "dca-api-candles.strat"))
	assert.NoError(t, err, "ReadStrategyConfigFromFile should not error")

	dc, err := config.GenerateDefaultConfig()
	assert.NoError(t, err, "GenerateDefaultConfig should not error")

	_, err = NewBacktesterFromConfigs(cfg, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "NewBacktesterFromConfigs should error on nil default config")

	_, err = NewBacktesterFromConfigs(nil, dc)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "NewBacktesterFromConfigs should error on nil config")

	bt, err := NewBacktesterFromConfigs(cfg, dc)
	if assert.NoError(t, err, "NewBacktesterFromConfigs should not error") {
		assert.False(t, bt.MetaData.DateLoaded.IsZero(), "DateLoaded should have a non-zero date")
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
	assert.ErrorIs(t, err, common.ErrNilEvent)

	cp := currency.NewBTCUSDT()
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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	assert.NoError(t, err)

	b, err := funding.CreateItem(testExchange, a, cp.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quote, err := funding.CreateItem(testExchange, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	assert.NoError(t, err)

	collateral, err := funding.CreateCollateral(b, quote)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	err = bt.DataHolder.SetDataForCurrency(testExchange, a, cp, k)
	assert.NoError(t, err)

	err = bt.processSingleDataEvent(ev, collateral)
	assert.NoError(t, err)
}
