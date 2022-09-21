package engine

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestSetupLiveDataHandler(t *testing.T) {
	t.Parallel()
	bt := &BackTest{}
	var err error
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.exchangeManager = engine.SetupExchangeManager()
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.DataHolder = &data.HandlerPerCurrency{}
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.Reports = &report.Data{}
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.Funding = &funding.FundManager{}
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	dc, ok := bt.LiveDataHandler.(*dataChecker)
	if !ok {
		t.Fatalf("received '%T' expected '%v'", dc, "dataChecker")
	}
	if dc.eventTimeout != defaultEventTimeout {
		t.Errorf("received '%v' expected '%v'", dc.eventTimeout, defaultEventTimeout)
	}
	if dc.dataCheckInterval != defaultDataCheckInterval {
		t.Errorf("received '%v' expected '%v'", dc.dataCheckInterval, defaultDataCheckInterval)
	}

	bt = nil
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestStart(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		shutdown: make(chan struct{}),
	}
	err := dataHandler.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	close(dataHandler.shutdown)
	err = dataHandler.Start()
	if !errors.Is(err, engine.ErrSubSystemAlreadyStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemAlreadyStarted)
	}

	var dh *dataChecker
	err = dh.Start()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{}
	if dataHandler.IsRunning() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	dataHandler.started = 1

	if !dataHandler.IsRunning() {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	var dh *dataChecker
	if dh.IsRunning() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestLiveHandlerStop(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{}
	err := dataHandler.Stop()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dataHandler.started = 1
	dataHandler.shutdown = make(chan struct{})
	err = dataHandler.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = dataHandler.Stop()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	var dh *dataChecker
	err = dh.Stop()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestDataFetcher(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{}
	dataHandler.wg.Add(1)
	err := dataHandler.DataFetcher()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dataHandler.started = 1
	dataHandler.wg.Add(1)
	err = dataHandler.DataFetcher()
	if !errors.Is(err, ErrLiveDataTimeout) {
		t.Errorf("received '%v' expected '%v'", err, ErrLiveDataTimeout)
	}

	dataHandler.wg.Add(1)
	dataHandler.shutdown = make(chan struct{})
	dataHandler.eventTimeout = time.Nanosecond
	var localWg sync.WaitGroup
	localWg.Add(1)
	go func() {
		defer localWg.Done()
		asyncErr := dataHandler.DataFetcher()
		if !errors.Is(asyncErr, ErrLiveDataTimeout) {
			t.Errorf("received '%v' expected '%v'", asyncErr, ErrLiveDataTimeout)
		}
	}()
	localWg.Wait()

	var dh *dataChecker
	err = dh.DataFetcher()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestUpdated(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		shutdown: make(chan struct{}),
	}
	var wg sync.WaitGroup
	wg.Add(1)
	waitChan := dataHandler.Updated()
	go func() {
		<-waitChan
		wg.Done()
	}()
	dataHandler.notice.Alert()
	wg.Wait()

	dataHandler = nil
	wg.Add(1)
	waitChan = dataHandler.Updated()
	go func() {
		<-waitChan
		wg.Done()
	}()
	wg.Wait()
}

func TestLiveHandlerReset(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		eventTimeout: 1,
	}
	dataHandler.Reset()
	if dataHandler.eventTimeout != 0 {
		t.Errorf("received '%v' expected '%v'", dataHandler.eventTimeout, 0)
	}
	var dh *dataChecker
	dh.Reset()
}

func TestAppendDataSource(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{}
	err := dataHandler.AppendDataSource(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	setup := &liveDataSourceSetup{}
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	setup.exchange = &ftx.FTX{}
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrInvalidDataType)
	}

	setup.dataType = common.DataCandle
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, asset.ErrNotSupported)
	}

	setup.asset = asset.Spot
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	setup.pair = currency.NewPair(currency.BTC, currency.USD)
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Errorf("received '%v' expected '%v'", err, kline.ErrUnsetInterval)
	}

	setup.interval = kline.OneDay
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if len(dataHandler.sourcesToCheck) != 1 {
		t.Errorf("received '%v' expected '%v'", len(dataHandler.sourcesToCheck), 1)
	}

	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, errDataSourceExists) {
		t.Errorf("received '%v' expected '%v'", err, errDataSourceExists)
	}

	dataHandler = nil
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestFetchLatestData(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		report:  &report.Data{},
		funding: &funding.FundManager{},
	}
	_, err := dataHandler.FetchLatestData()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dataHandler.started = 1
	_, err = dataHandler.FetchLatestData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	cp := currency.NewPair(currency.BTC, currency.USD).Format(
		currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		})
	f := &ftx.FTX{}
	f.SetDefaults()
	fb := f.GetBase()
	fbA := fb.CurrencyPairs.Pairs[asset.Spot]
	fbA.Enabled = fbA.Enabled.Add(cp)
	fbA.Available = fbA.Available.Add(cp)
	dataHandler.sourcesToCheck = []*liveDataSourceDataHandler{
		{
			exchange:                  f,
			exchangeName:              "ftx",
			asset:                     asset.Spot,
			pair:                      cp,
			dataRequestRetryWaitTime:  defaultDataRequestWaitTime,
			dataRequestRetryTolerance: 1,
			underlyingPair:            cp,
			pairCandles: datakline.DataFromKline{
				Base: data.Base{},
				Item: kline.Item{
					Interval: kline.OneHour,
				},
			},
			dataType:      common.DataCandle,
			processedData: make(map[int64]struct{}),
		},
	}
	dataHandler.dataHolder = &fakeDataHolder{}
	_, err = dataHandler.FetchLatestData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	var dh *dataChecker
	_, err = dh.FetchLatestData()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestLoadCandleData(t *testing.T) {
	t.Parallel()
	l := &liveDataSourceDataHandler{
		dataRequestRetryTolerance: 1,
		dataRequestRetryWaitTime:  defaultDataRequestWaitTime,
		processedData:             make(map[int64]struct{}),
	}
	_, err := l.loadCandleData(time.Now())
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	exch := &ftx.FTX{}
	exch.SetDefaults()
	cp := currency.NewPair(currency.BTC, currency.USD).Format(
		currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		})
	eba := exch.CurrencyPairs.Pairs[asset.Spot]
	eba.Available = eba.Available.Add(cp)
	eba.Enabled = eba.Enabled.Add(cp)
	eba.AssetEnabled = convert.BoolPtr(true)
	l.exchange = exch
	l.dataType = common.DataCandle
	l.asset = asset.Spot
	l.pair = cp
	l.pairCandles = datakline.DataFromKline{
		Item: kline.Item{
			Exchange:       testExchange,
			Asset:          asset.Spot,
			Pair:           cp,
			UnderlyingPair: cp,
			Interval:       kline.OneHour,
		},
	}
	updated, err := l.loadCandleData(time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !updated {
		t.Errorf("received '%v' expected '%v'", updated, true)
	}

	var ldh *liveDataSourceDataHandler
	_, err = ldh.loadCandleData(time.Now())
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestSetDataForClosingAllPositions(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		report:  &report.Data{},
		funding: &funding.FundManager{},
	}

	dataHandler.started = 1
	cp := currency.NewPair(currency.BTC, currency.USD).Format(
		currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		})
	f := &ftx.FTX{}
	f.SetDefaults()
	fb := f.GetBase()
	fbA := fb.CurrencyPairs.Pairs[asset.Spot]
	fbA.Enabled = fbA.Enabled.Add(cp)
	fbA.Available = fbA.Available.Add(cp)
	dataHandler.sourcesToCheck = []*liveDataSourceDataHandler{
		{
			exchange:                  f,
			exchangeName:              "ftx",
			asset:                     asset.Spot,
			pair:                      cp,
			dataRequestRetryWaitTime:  defaultDataRequestWaitTime,
			dataRequestRetryTolerance: 1,
			underlyingPair:            cp,
			pairCandles: datakline.DataFromKline{
				Base: data.Base{},
				Item: kline.Item{
					Interval: kline.OneHour,
				},
			},
			dataType:      common.DataCandle,
			processedData: make(map[int64]struct{}),
		},
	}
	dataHandler.dataHolder = &fakeDataHolder{}
	_, err := dataHandler.FetchLatestData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = dataHandler.SetDataForClosingAllPositions()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	err = dataHandler.SetDataForClosingAllPositions(nil)
	if !errors.Is(err, errNilData) {
		t.Errorf("received '%v' expected '%v'", err, errNilData)
	}
	err = dataHandler.SetDataForClosingAllPositions(&signal.Signal{
		Base: &event.Base{
			Offset:         3,
			Exchange:       "ftx",
			Time:           time.Now(),
			Interval:       kline.OneHour,
			CurrencyPair:   cp,
			UnderlyingPair: cp,
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
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = dataHandler.SetDataForClosingAllPositions(&signal.Signal{
		Base: &event.Base{
			Offset:         3,
			Exchange:       "binance",
			Time:           time.Now(),
			Interval:       kline.OneHour,
			CurrencyPair:   cp,
			UnderlyingPair: cp,
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
	})
	if !errors.Is(err, errNoDataSetForClosingPositions) {
		t.Errorf("received '%v' expected '%v'", err, errNoDataSetForClosingPositions)
	}

	dataHandler = nil
	err = dataHandler.SetDataForClosingAllPositions()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsRealOrders(t *testing.T) {
	t.Parallel()
	d := &dataChecker{}
	if d.IsRealOrders() {
		t.Error("expected false")
	}
	d.realOrders = true
	if !d.IsRealOrders() {
		t.Error("expected true")
	}
}
