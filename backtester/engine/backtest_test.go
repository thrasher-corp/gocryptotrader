package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	evkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
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

func (p portfolioOverride) CreateLiquidationOrdersForExchange(ev common.DataEventHandler, _ funding.IFundingManager) ([]order.Event, error) {
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

func TestReset(t *testing.T) {
	t.Parallel()
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, true, false)
	if err != nil {
		t.Error(err)
	}
	bt := BackTest{
		shutdown:   make(chan struct{}),
		Datas:      &data.HandlerPerCurrency{},
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
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*statistics.CurrencyPairStatistic)

	port, err := portfolio.Setup(&size.Size{
		BuySide:  exchange.MinMax{},
		SellSide: exchange.MinMax{},
	}, &risk.Risk{}, decimal.Zero)
	if err != nil {
		t.Error(err)
	}
	fx := &ftx.FTX{}
	fx.Name = testExchange
	err = port.SetupCurrencySettingsMap(&exchange.Settings{Exchange: fx, Asset: a, Pair: cp})
	if err != nil {
		t.Error(err)
	}
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if err != nil {
		t.Error(err)
	}
	b, err := funding.CreateItem(ex, a, cp.Base, decimal.Zero, decimal.Zero)
	if err != nil {
		t.Error(err)
	}
	quote, err := funding.CreateItem(ex, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if err != nil {
		t.Error(err)
	}
	pair, err := funding.CreatePair(b, quote)
	if err != nil {
		t.Error(err)
	}
	err = f.AddPair(pair)
	if err != nil {
		t.Error(err)
	}
	bt := BackTest{
		shutdown:   nil,
		Datas:      &data.HandlerPerCurrency{},
		Strategy:   &dollarcostaverage.Strategy{},
		Portfolio:  port,
		Exchange:   &exchange.Exchange{},
		Statistic:  stats,
		EventQueue: &eventholder.Holder{},
		Reports:    &report.Data{},
		Funding:    f,
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
	if err != nil {
		t.Error(err)
	}
	bt.Datas.SetDataForCurrency(ex, a, cp, &k)

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
	stats.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*statistics.CurrencyPairStatistic)
	stats.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*statistics.CurrencyPairStatistic)

	port, err := portfolio.Setup(&size.Size{
		BuySide:  exchange.MinMax{},
		SellSide: exchange.MinMax{},
	}, &risk.Risk{}, decimal.Zero)
	if err != nil {
		t.Error(err)
	}
	err = port.SetupCurrencySettingsMap(&exchange.Settings{Exchange: &ftx.FTX{}, Asset: a, Pair: cp})
	if err != nil {
		t.Error(err)
	}
	f, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if err != nil {
		t.Error(err)
	}
	b, err := funding.CreateItem(ex, a, cp.Base, decimal.Zero, decimal.Zero)
	if err != nil {
		t.Error(err)
	}
	quote, err := funding.CreateItem(ex, a, cp.Quote, decimal.NewFromInt(1337), decimal.Zero)
	if err != nil {
		t.Error(err)
	}
	pair, err := funding.CreatePair(b, quote)
	if err != nil {
		t.Error(err)
	}
	err = f.AddPair(pair)
	if err != nil {
		t.Error(err)
	}
	bt := BackTest{
		shutdown:   nil,
		Datas:      &data.HandlerPerCurrency{},
		Portfolio:  port,
		Exchange:   &exchange.Exchange{},
		Statistic:  stats,
		EventQueue: &eventholder.Holder{},
		Reports:    &report.Data{},
		Funding:    f,
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
	if err != nil {
		t.Error(err)
	}

	bt.Datas.SetDataForCurrency(ex, a, cp, &k)

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

	cp := currency.NewPair(currency.BTC, currency.USDT)
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
	bt.Datas = &data.HandlerPerCurrency{}
	d := data.Base{}
	d.SetStream([]common.DataEventHandler{&evkline.Kline{
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
	bt.Datas.SetDataForCurrency(testExchange, a, cp, da)
	err = bt.Statistic.SetupEventForTime(ev)
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

	cp := currency.NewPair(currency.BTC, currency.USDT)
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
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetupEventForTime(de)
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
		Datas:      &data.HandlerPerCurrency{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetupEventForTime(de)
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
	bt.Datas.Setup()
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
	if err != nil {
		t.Error(err)
	}

	bt.Datas.SetDataForCurrency(testExchange, a, cp, &k)
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
		Datas:      &data.HandlerPerCurrency{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetupEventForTime(de)
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
	cfg, err := exch.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	err = exch.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}
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
	bt.Datas.Setup()
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
	if err != nil {
		t.Error(err)
	}

	bt.Datas.SetDataForCurrency(testExchange, a, cp, &k)
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
		Datas:      &data.HandlerPerCurrency{},
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Futures
	de := &evkline.Kline{
		Base: &event.Base{Exchange: testExchange,
			AssetType:    a,
			CurrencyPair: cp},
	}
	err = bt.Statistic.SetupEventForTime(de)
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
	cfg, err := exch.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	err = exch.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}
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
	bt.Datas.Setup()
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
	if err != nil {
		t.Error(err)
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
	bt.Datas.SetDataForCurrency(testExchange, a, cp, &k)
	err = bt.processFuturesFillEvent(ev, pair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
}
