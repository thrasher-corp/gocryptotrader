package engine

import (
	"errors"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
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
	err = bt.SetupLiveDataHandler(-1, -1, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.exchangeManager = engine.SetupExchangeManager()
	err = bt.SetupLiveDataHandler(-1, -1, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.DataHolder = &data.HandlerPerCurrency{}
	err = bt.SetupLiveDataHandler(-1, -1, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.Reports = &report.Data{}
	err = bt.SetupLiveDataHandler(-1, -1, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	bt.Funding = &funding.FundManager{}
	err = bt.SetupLiveDataHandler(-1, -1, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	dc, ok := bt.LiveDataHandler.(*DataChecker)
	if !ok {
		t.Fatalf("received '%T' expected '%v'", dc, "DataChecker")
	}
	if dc.eventTimeout != defaultEventTimeout {
		t.Errorf("received '%v' expected '%v'", dc.eventTimeout, defaultEventTimeout)
	}
	if dc.dataCheckInterval != defaultDataCheckInterval {
		t.Errorf("received '%v' expected '%v'", dc.dataCheckInterval, defaultDataCheckInterval)
	}

	bt = nil
	err = bt.SetupLiveDataHandler(-1, -1, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestStart(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{
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

	var dh *DataChecker
	err = dh.Start()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{}
	if dataHandler.IsRunning() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}

	dataHandler.started = 1

	if !dataHandler.IsRunning() {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	var dh *DataChecker
	if dh.IsRunning() {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
}

func TestLiveHandlerStop(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{}
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

	var dh *DataChecker
	err = dh.Stop()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestDataFetcher(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{}
	dataHandler.wg.Add(1)
	err := dataHandler.DataFetcher()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dataHandler.started = 1
	dataHandler.wg.Add(1)
	dataHandler.updated = make(chan struct{})
	err = dataHandler.DataFetcher()
	if !errors.Is(err, ErrLiveDataTimeout) {
		t.Errorf("received '%v' expected '%v'", err, ErrLiveDataTimeout)
	}

	dataHandler.wg.Add(1)
	dataHandler.shutdown = make(chan struct{})
	dataHandler.updated = make(chan struct{})
	dataHandler.eventTimeout = time.Second
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

	dataHandler.wg.Add(1)
	close(dataHandler.shutdown)
	localWg.Add(1)
	go func() {
		defer localWg.Done()
		asyncErr := dataHandler.DataFetcher()
		if !errors.Is(asyncErr, nil) {
			t.Errorf("received '%v' expected '%v'", asyncErr, nil)
		}
	}()
	localWg.Wait()

	var dh *DataChecker
	err = dh.DataFetcher()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestUpdated(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{
		updated: make(chan struct{}),
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		select {
		case <-dataHandler.Updated():
			wg.Done()
		}
	}()

	close(dataHandler.updated)
	wg.Wait()

	var dh *DataChecker
	wg.Add(1)
	go func() {
		select {
		case <-dh.Updated():
			wg.Done()
		}
	}()

	wg.Wait()
}

func TestLiveHandlerReset(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{
		eventTimeout: 1,
	}
	dataHandler.Reset()
	if dataHandler.eventTimeout != 0 {
		t.Errorf("received '%v' expected '%v'", dataHandler.eventTimeout, 0)
	}
	var dh *DataChecker
	dh.Reset()
}

func TestAppendDataSource(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{}
	err := dataHandler.AppendDataSource(nil, 0, 0, currency.EMPTYPAIR, currency.EMPTYPAIR, 0)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	err = dataHandler.AppendDataSource(&ftx.FTX{}, 0, 0, currency.EMPTYPAIR, currency.EMPTYPAIR, 0)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrInvalidDataType)
	}

	dt := common.DataCandle
	err = dataHandler.AppendDataSource(&ftx.FTX{}, 0, 0, currency.EMPTYPAIR, currency.EMPTYPAIR, dt)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, asset.ErrNotSupported)
	}

	err = dataHandler.AppendDataSource(&ftx.FTX{}, 0, asset.Spot, currency.EMPTYPAIR, currency.EMPTYPAIR, dt)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	err = dataHandler.AppendDataSource(&ftx.FTX{}, 0, asset.Spot, currency.NewPair(currency.BTC, currency.USD), currency.EMPTYPAIR, dt)
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Errorf("received '%v' expected '%v'", err, kline.ErrUnsetInterval)
	}

	err = dataHandler.AppendDataSource(&ftx.FTX{}, kline.OneHour, asset.Spot, currency.NewPair(currency.BTC, currency.USD), currency.EMPTYPAIR, dt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if len(dataHandler.exchangesToCheck) != 1 {
		t.Errorf("received '%v' expected '%v'", len(dataHandler.exchangesToCheck), 1)
	}

	err = dataHandler.AppendDataSource(&ftx.FTX{}, kline.OneHour, asset.Spot, currency.NewPair(currency.BTC, currency.USD), currency.EMPTYPAIR, dt)
	if !errors.Is(err, errDataSourceExists) {
		t.Errorf("received '%v' expected '%v'", err, errDataSourceExists)
	}

	dataHandler = nil
	err = dataHandler.AppendDataSource(nil, 0, 0, currency.EMPTYPAIR, currency.EMPTYPAIR, 0)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestFetchLatestData(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{
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
	cp := currency.NewPair(currency.BTC, currency.USD).Format("/", true)
	f := &ftx.FTX{}
	f.SetDefaults()
	fb := f.GetBase()
	fbA := fb.CurrencyPairs.Pairs[asset.Spot]
	fbA.Enabled = fbA.Enabled.Add(cp)
	fbA.Available = fbA.Available.Add(cp)
	dataHandler.exchangesToCheck = []liveExchangeDataHandler{
		{
			m:              sync.Mutex{},
			exchange:       f,
			exchangeName:   "ftx",
			asset:          asset.Spot,
			pair:           cp,
			underlyingPair: cp,
			pairCandles: datakline.DataFromKline{
				Base: data.Base{},
				Item: kline.Item{
					Interval: kline.OneHour,
				},
			},
			dataType: common.DataCandle,
		},
	}
	dataHandler.dataHolder = &fakeDataHolder{}
	_, err = dataHandler.FetchLatestData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	var dh *DataChecker
	_, err = dh.FetchLatestData()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestLoadCandleData(t *testing.T) {
	t.Parallel()
	l := &liveExchangeDataHandler{}
	err := l.loadCandleData(time.Now())
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	exch := &ftx.FTX{}
	exch.SetDefaults()
	cp := currency.NewPair(currency.BTC, currency.USD).Format("/", true)
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
	err = l.loadCandleData(time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	var ldh *liveExchangeDataHandler
	err = ldh.loadCandleData(time.Now())
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}
