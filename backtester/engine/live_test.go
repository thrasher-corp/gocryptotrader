package engine

import (
	"errors"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestSetupLiveDataHandler(t *testing.T) {
	t.Parallel()
	_, err := SetupLiveDataHandler(nil, nil, -1, -1, -1, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	em := engine.SetupExchangeManager()
	_, err = SetupLiveDataHandler(em, nil, -1, -1, -1, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	holder := &data.HandlerPerCurrency{}
	dataHandler, err := SetupLiveDataHandler(em, holder, -1, -1, -1, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	dataChecker, ok := dataHandler.(*DataChecker)
	if !ok {
		t.Error(gctcommon.GetAssertError("*DataChecker", dataChecker))
	}
	if dataChecker.eventCheckInterval != defaultEventCheckInterval {
		t.Errorf("received '%v' expected '%v'", dataChecker.eventCheckInterval, defaultEventCheckInterval)
	}
	if dataChecker.eventTimeout != defaultEventTimeout {
		t.Errorf("received '%v' expected '%v'", dataChecker.eventTimeout, defaultEventTimeout)
	}
	if dataChecker.dataCheckInterval != defaultDataCheckInterval {
		t.Errorf("received '%v' expected '%v'", dataChecker.dataCheckInterval, defaultDataCheckInterval)
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
	dataHandler.eventTimeout = time.Minute
	go func() {
		asyncErr := dataHandler.DataFetcher()
		if !errors.Is(asyncErr, nil) {
			t.Errorf("received '%v' expected '%v'", asyncErr, nil)
		}
	}()
	close(dataHandler.shutdown)
	dataHandler.wg.Wait()

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
	err := dataHandler.AppendDataSource(nil, nil, -1)
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dataHandler.started = 1
	err = dataHandler.AppendDataSource(nil, nil, -1)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	item := &kline.Item{}
	err = dataHandler.AppendDataSource(item, nil, -1)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	err = dataHandler.AppendDataSource(item, &ftx.FTX{}, -1)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrInvalidDataType)
	}

	err = dataHandler.AppendDataSource(item, &ftx.FTX{}, common.DataTrade)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, asset.ErrNotSupported)
	}

	item.Asset = asset.Futures
	err = dataHandler.AppendDataSource(item, &ftx.FTX{}, common.DataTrade)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	item.Pair = currency.NewPair(currency.AAC, currency.ACE)
	err = dataHandler.AppendDataSource(item, &ftx.FTX{}, common.DataTrade)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = dataHandler.AppendDataSource(item, &ftx.FTX{}, common.DataTrade)
	if !errors.Is(err, funding.ErrAlreadyExists) {
		t.Errorf("received '%v' expected '%v'", err, funding.ErrAlreadyExists)
	}

	var dh *DataChecker
	err = dh.AppendDataSource(item, &ftx.FTX{}, common.DataTrade)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestFetchLatestData(t *testing.T) {
	t.Parallel()
	dataHandler := &DataChecker{}
	_, err := dataHandler.FetchLatestData()
	if !errors.Is(err, engine.ErrSubSystemNotStarted) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrSubSystemNotStarted)
	}

	dataHandler.started = 1
	_, err = dataHandler.FetchLatestData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	f := &ftx.FTX{}
	f.SetDefaults()
	cfg, err := f.GetDefaultConfig()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.SetupDefaults(cfg)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.CurrencyPairs.EnablePair(asset.Spot, cp)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	err := l.loadCandleData()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	exch := &ftx.FTX{}
	exch.SetDefaults()
	exch.Config, err = exch.GetDefaultConfig()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = exch.SetPairs(currency.Pairs{currency.NewPair(currency.BTC, currency.USD)}, asset.Spot, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = exch.SetPairs(currency.Pairs{currency.NewPair(currency.BTC, currency.USD)}, asset.Spot, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = l.loadCandleData()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	var ldh *liveExchangeDataHandler
	err = ldh.loadCandleData()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}
