package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt.exchangeManager = engine.NewExchangeManager()
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt.DataHolder = &data.HandlerHolder{}
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt.Reports = &report.Data{}
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt.Funding = &funding.FundManager{}
	err = bt.SetupLiveDataHandler(-1, -1, false, false)
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestStart(t *testing.T) {
	t.Parallel()

	var dc *dataChecker
	err := dc.Start()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	dc = &dataChecker{
		shutdown: make(chan bool),
	}
	err = dc.Start()
	require.NoError(t, err)

	close(dc.shutdown)
	dc.wg.Wait()

	dc = &dataChecker{
		started: 1,
	}
	err = dc.Start()
	assert.ErrorIs(t, err, engine.ErrSubSystemAlreadyStarted)
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
	assert.ErrorIs(t, err, engine.ErrSubSystemNotStarted)

	dc.started = 1
	err = dc.Stop()
	assert.NoError(t, err)

	dc.shutdown = make(chan bool)
	err = dc.Stop()
	assert.ErrorIs(t, err, engine.ErrSubSystemNotStarted)

	var dh *dataChecker
	err = dh.Stop()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestLiveHandlerStopFromError(t *testing.T) {
	t.Parallel()
	dc := &dataChecker{
		shutdownErr: make(chan bool, 10),
	}
	err := dc.SignalStopFromError(errNoCredsNoLive)
	assert.ErrorIs(t, err, engine.ErrSubSystemNotStarted)

	err = dc.SignalStopFromError(nil)
	assert.ErrorIs(t, err, errNilError)

	dc.started = 1
	var wg sync.WaitGroup
	wg.Go(func() {
		assert.NoError(t, dc.SignalStopFromError(errNoCredsNoLive))
	})
	wg.Wait()

	var dh *dataChecker
	err = dh.SignalStopFromError(errNoCredsNoLive)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
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
	assert.ErrorIs(t, err, engine.ErrSubSystemNotStarted)

	dc.started = 1
	dc.wg.Add(1)
	err = dc.DataFetcher()
	assert.ErrorIs(t, err, ErrLiveDataTimeout)

	var dh *dataChecker
	err = dh.DataFetcher()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestUpdated(t *testing.T) {
	t.Parallel()
	dc := &dataChecker{
		dataUpdated: make(chan bool, 10),
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		_ = dc.Updated()
	})
	wg.Wait()

	dc = nil
	wg.Go(func() {
		_ = dc.Updated()
	})
	wg.Wait()
}

func TestLiveHandlerReset(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		eventTimeout: 1,
	}
	err := dataHandler.Reset()
	assert.NoError(t, err)

	if dataHandler.eventTimeout != 0 {
		t.Errorf("received '%v' expected '%v'", dataHandler.eventTimeout, 0)
	}
	var dh *dataChecker
	err = dh.Reset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestAppendDataSource(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{}
	err := dataHandler.AppendDataSource(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	setup := &liveDataSourceSetup{}
	err = dataHandler.AppendDataSource(setup)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	setup.exchange = &binance.Exchange{}
	err = dataHandler.AppendDataSource(setup)
	assert.ErrorIs(t, err, common.ErrInvalidDataType)

	setup.dataType = common.DataCandle
	err = dataHandler.AppendDataSource(setup)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	setup.asset = asset.Spot
	err = dataHandler.AppendDataSource(setup)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	setup.pair = currency.NewBTCUSDT()
	err = dataHandler.AppendDataSource(setup)
	assert.ErrorIs(t, err, kline.ErrInvalidInterval)

	setup.interval = kline.OneDay
	err = dataHandler.AppendDataSource(setup)
	assert.NoError(t, err)

	if len(dataHandler.sourcesToCheck) != 1 {
		t.Errorf("received '%v' expected '%v'", len(dataHandler.sourcesToCheck), 1)
	}

	err = dataHandler.AppendDataSource(setup)
	assert.ErrorIs(t, err, errDataSourceExists)

	dataHandler = nil
	err = dataHandler.AppendDataSource(setup)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestFetchLatestData(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		report:  &report.Data{},
		funding: &fakeFunding{},
	}
	_, err := dataHandler.FetchLatestData()
	require.ErrorIs(t, err, engine.ErrSubSystemNotStarted)

	dataHandler.started = 1
	_, err = dataHandler.FetchLatestData()
	require.NoError(t, err)
	cp := currency.NewBTCUSDT()
	f := &binanceus.Exchange{}
	f.SetDefaults()
	fb := f.GetBase()
	require.NoError(t, fb.CurrencyPairs.SetAssetEnabled(asset.Spot, true), "SetAssetEnabled must not error")
	require.NoError(t, fb.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{cp}, false), "StorePairs must not error")
	require.NoError(t, fb.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{cp}, true), "StorePairs must not error")
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
	require.NoError(t, err)

	var dh *dataChecker
	_, err = dh.FetchLatestData()
	require.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestLoadCandleData(t *testing.T) {
	t.Parallel()
	l := &liveDataSourceDataHandler{
		dataRequestRetryTolerance: 1,
		dataRequestRetryWaitTime:  defaultDataRequestWaitTime,
		processedData:             make(map[int64]struct{}),
	}
	_, err := l.loadCandleData(time.Now())
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	exch := &binanceus.Exchange{}
	exch.SetDefaults()
	cp := currency.NewBTCUSDT().Format(
		currency.PairFormat{
			Uppercase: true,
		})
	eba := exch.CurrencyPairs.Pairs[asset.Spot]
	eba.Available = eba.Available.Add(cp)
	eba.Enabled = eba.Enabled.Add(cp)
	eba.AssetEnabled = true
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
	assert.NoError(t, err)

	if !updated {
		t.Errorf("received '%v' expected '%v'", updated, true)
	}

	var ldh *liveDataSourceDataHandler
	_, err = ldh.loadCandleData(time.Now())
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestSetDataForClosingAllPositions(t *testing.T) {
	t.Parallel()
	dataHandler := &dataChecker{
		report:  &fakeReport{},
		funding: &fakeFunding{},
	}

	dataHandler.started = 1
	cp := currency.NewBTCUSDT()
	f := &binanceus.Exchange{}
	f.SetDefaults()
	fb := f.GetBase()
	err := fb.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{cp}, true)
	require.NoError(t, err, "StorePairs must not error")
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
	assert.NoError(t, err)

	err = dataHandler.SetDataForClosingAllPositions()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	err = dataHandler.SetDataForClosingAllPositions(nil)
	assert.ErrorIs(t, err, errNilData)

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
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	dataHandler = nil
	err = dataHandler.SetDataForClosingAllPositions()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	ff := &fakeFunding{}
	d.funding = ff
	err = d.UpdateFunding(false)
	assert.NoError(t, err)

	err = d.UpdateFunding(true)
	assert.NoError(t, err)

	d.realOrders = true
	err = d.UpdateFunding(true)
	assert.NoError(t, err)

	ff.hasFutures = true
	err = d.UpdateFunding(true)
	assert.NoError(t, err)

	d.updatingFunding = 1
	err = d.UpdateFunding(true)
	assert.NoError(t, err)

	d.updatingFunding = 1
	err = d.UpdateFunding(false)
	assert.NoError(t, err)

	d = nil
	err = d.UpdateFunding(false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
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
