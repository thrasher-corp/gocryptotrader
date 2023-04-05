package engine

import (
	"errors"
	"sync"
	"sync/atomic"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binanceus"
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

	bt.exchangeManager = engine.NewExchangeManager()
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.DataHolder = &data.HandlerHolder{}
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
	dc := &dataChecker{
		shutdown: make(chan bool),
	}
	err := dc.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	close(dc.shutdown)
	dc.wg.Wait()
	atomic.CompareAndSwapUint32(&dc.started, 0, 1)
	err = dc.Start()
	if !errors.Is(err, engine.ErrSubSystemAlreadyStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemAlreadyStarted)
	}

	var dh *dataChecker
	err = dh.Start()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestDataCheckerIsRunning(t *testing.T) {
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
	dc := &dataChecker{
		shutdown: make(chan bool),
	}
	err := dc.Stop()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dc.started = 1
	err = dc.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	dc.shutdown = make(chan bool)
	err = dc.Stop()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	var dh *dataChecker
	err = dh.Stop()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestLiveHandlerStopFromError(t *testing.T) {
	t.Parallel()
	dc := &dataChecker{
		shutdownErr: make(chan bool, 10),
	}
	err := dc.SignalStopFromError(errNoCredsNoLive)
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	err = dc.SignalStopFromError(nil)
	if !errors.Is(err, errNilError) {
		t.Errorf("received '%v' expected '%v'", err, errNilError)
	}
	dc.started = 1
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = dc.SignalStopFromError(errNoCredsNoLive)
		if !errors.Is(err, nil) {
			t.Errorf("received '%v' expected '%v'", err, nil)
		}
	}()
	wg.Wait()

	var dh *dataChecker
	err = dh.SignalStopFromError(errNoCredsNoLive)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestDataFetcher(t *testing.T) {
	t.Parallel()
	dc := &dataChecker{
		dataCheckInterval: time.Second,
		eventTimeout:      time.Millisecond,
		shutdown:          make(chan bool, 10),
		shutdownErr:       make(chan bool, 10),
		dataUpdated:       make(chan bool, 10),
	}
	dc.wg.Add(1)
	err := dc.DataFetcher()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dc.started = 1
	dc.wg.Add(1)
	err = dc.DataFetcher()
	if !errors.Is(err, ErrLiveDataTimeout) {
		t.Errorf("received '%v' expected '%v'", err, ErrLiveDataTimeout)
	}

	var dh *dataChecker
	err = dh.DataFetcher()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestUpdated(t *testing.T) {
	t.Parallel()
	dc := &dataChecker{
		dataUpdated: make(chan bool, 10),
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = dc.Updated()
		wg.Done()
	}()
	wg.Wait()

	dc = nil
	wg.Add(1)
	go func() {
		_ = dc.Updated()
		wg.Done()
	}()
	wg.Wait()
}

func TestLiveHandlerReset(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		eventTimeout: 1,
	}
	err := dataHandler.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if dataHandler.eventTimeout != 0 {
		t.Errorf("received '%v' expected '%v'", dataHandler.eventTimeout, 0)
	}
	var dh *dataChecker
	err = dh.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
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

	setup.exchange = &binance.Binance{}
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

	setup.pair = currency.NewPair(currency.BTC, currency.USDT)
	err = dataHandler.AppendDataSource(setup)
	if !errors.Is(err, kline.ErrInvalidInterval) {
		t.Errorf("received '%v' expected '%v'", err, kline.ErrInvalidInterval)
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
		funding: &fakeFunding{},
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
	cp := currency.NewPair(currency.BTC, currency.USDT).Format(
		currency.PairFormat{
			Uppercase: true,
		})
	f := &binanceus.Binanceus{}
	f.SetDefaults()
	fb := f.GetBase()
	fbA := fb.CurrencyPairs.Pairs[asset.Spot]
	fbA.Enabled = fbA.Enabled.Add(cp)
	fbA.Available = fbA.Available.Add(cp)
	dataHandler.sourcesToCheck = []*liveDataSourceDataHandler{
		{
			exchange:                  f,
			exchangeName:              testExchange,
			asset:                     asset.Spot,
			pair:                      cp,
			dataRequestRetryWaitTime:  defaultDataRequestWaitTime,
			dataRequestRetryTolerance: 1,
			underlyingPair:            cp,
			pairCandles: &datakline.DataFromKline{
				Base: &data.Base{},
				Item: &kline.Item{
					Exchange:       testExchange,
					Pair:           cp,
					UnderlyingPair: cp,
					Asset:          asset.Spot,
					Interval:       kline.OneHour,
					Candles: []kline.Candle{
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

	exch := &binanceus.Binanceus{}
	exch.SetDefaults()
	cp := currency.NewPair(currency.BTC, currency.USDT).Format(
		currency.PairFormat{
			Uppercase: true,
		})
	eba := exch.CurrencyPairs.Pairs[asset.Spot]
	eba.Available = eba.Available.Add(cp)
	eba.Enabled = eba.Enabled.Add(cp)
	eba.AssetEnabled = convert.BoolPtr(true)
	l.exchange = exch
	l.dataType = common.DataCandle
	l.asset = asset.Spot
	l.pair = cp
	l.pairCandles = &datakline.DataFromKline{
		Base: &data.Base{},
		Item: &kline.Item{
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
		report:  &fakeReport{},
		funding: &fakeFunding{},
	}

	dataHandler.started = 1
	cp := currency.NewPair(currency.BTC, currency.USDT).Format(
		currency.PairFormat{
			Uppercase: true,
		})
	f := &binanceus.Binanceus{}
	f.SetDefaults()
	fb := f.GetBase()
	fbA := fb.CurrencyPairs.Pairs[asset.Spot]
	fbA.Enabled = fbA.Enabled.Add(cp)
	fbA.Available = fbA.Available.Add(cp)
	dataHandler.sourcesToCheck = []*liveDataSourceDataHandler{
		{
			exchange:                  f,
			exchangeName:              testExchange,
			asset:                     asset.Spot,
			pair:                      cp,
			dataRequestRetryWaitTime:  defaultDataRequestWaitTime,
			dataRequestRetryTolerance: 1,
			underlyingPair:            cp,
			pairCandles: &datakline.DataFromKline{
				Base: &data.Base{},
				Item: &kline.Item{
					Exchange:       testExchange,
					Pair:           cp,
					UnderlyingPair: cp,
					Asset:          asset.Spot,
					Interval:       kline.OneHour,
					Candles: []kline.Candle{
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
			Exchange:       testExchange,
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
			Offset:         4,
			Exchange:       testExchange,
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

func TestUpdateFunding(t *testing.T) {
	t.Parallel()
	d := &dataChecker{}
	err := d.UpdateFunding(false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	ff := &fakeFunding{}
	d.funding = ff
	err = d.UpdateFunding(false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = d.UpdateFunding(true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	d.realOrders = true
	err = d.UpdateFunding(true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	ff.hasFutures = true
	err = d.UpdateFunding(true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	d.updatingFunding = 1
	err = d.UpdateFunding(true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	d.updatingFunding = 1
	err = d.UpdateFunding(false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	d = nil
	err = d.UpdateFunding(false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestClosedChan(t *testing.T) {
	t.Parallel()
	chantel := closedChan()
	if chantel == nil {
		t.Errorf("expected channel, received %v", nil)
	}
	<-chantel
	// demonstrate nil channel still functions on a select case
	chantel = nil
	select {
	case <-chantel:
		t.Error("woah")
	default:
	}
}
