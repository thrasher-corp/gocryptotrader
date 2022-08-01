package engine

import (
	"context"
	"errors"
	"fmt"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding/trackingcurrencies"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// LiveHandler is all the functionality required in order to
// run a backtester with live data
type LiveHandler interface {
	AppendDataSource(item *gctkline.Item, exch gctexchange.IBotExchange, dataType int64) error
	FetchLatestData() error
	Start() error
	IsRunning() bool
	DataFetcher() error
	Stop() error
	Reset()
	// AppendUSDTrackingData ??
}

// LiveDataChecker is responsible for managing all data retrieval
// for a live data option
type LiveDataChecker struct {
	m                  sync.Mutex
	wg                 sync.WaitGroup
	started            uint32
	exchangeManager    *engine.ExchangeManager
	exchangesToCheck   []liveExchangeDataHandler
	eventCheckInterval time.Duration
	eventTimeout       time.Duration
	dataCheckInterval  time.Duration
	shutdown           chan struct{}
}

type LiveExchangeHandler interface {
	UpdateData() error
}

type liveExchangeDataHandler struct {
	m              sync.Mutex
	exchange       gctexchange.IBotExchange
	asset          asset.Item
	pair           currency.Pair
	underlyingPair currency.Pair
	pairCandles    kline.DataFromKline
	dataType       int64
}

func SetupLiveDataHandler(em *engine.ExchangeManager, evenCheckInterval, eventTimeout, dataCheckInterval time.Duration) (LiveHandler, error) {
	if em == nil {
		return nil, fmt.Errorf("%w engine manager", gctcommon.ErrNilPointer)
	}

	return &LiveDataChecker{
		exchangeManager:    em,
		eventCheckInterval: evenCheckInterval,
		eventTimeout:       eventTimeout,
		dataCheckInterval:  dataCheckInterval,
	}, nil
}

func (l *LiveDataChecker) Start() error {
	if l == nil {
		return gctcommon.ErrNilPointer
	}
	if atomic.CompareAndSwapUint32(&l.started, 0, 1) {
		return engine.ErrSubSystemAlreadyStarted
	}
	var err error
	l.wg.Add(1)
	go func() {
		err := l.DataFetcher()
		if err != nil {
			return
		}
	}()

	return err
}

func (l *LiveDataChecker) IsRunning() bool {
	return l != nil && atomic.LoadUint32(&l.started) == 1
}

func (l *LiveDataChecker) Stop() error {
	if l == nil {
		return gctcommon.ErrNilPointer
	}
	if atomic.CompareAndSwapUint32(&l.started, 0, 1) {
		return engine.ErrSubSystemAlreadyStarted
	}
	l.m.Lock()
	defer l.m.Unlock()
	close(l.shutdown)
	l.wg.Wait()
	l.shutdown = make(chan struct{})
	return nil
}

func (l *LiveDataChecker) DataFetcher() error {
	checkTimer := time.NewTimer(0)
	timeoutTimer := time.NewTimer(l.eventTimeout)
	var err error
	for {
		select {
		case <-l.shutdown:
			return nil
		case <-timeoutTimer.C:
			return errLiveDataTimeout
		case <-checkTimer.C:
			err = l.FetchLatestData()
			if err != nil {
				return err
			}
		}
	}
}

func (l *LiveDataChecker) Reset() {
	if l == nil {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()
	l.dataCheckInterval = 0
	l.eventCheckInterval = 0
	l.eventTimeout = 0
	l.exchangeManager = nil
	l.exchangesToCheck = nil
}

func (l *LiveDataChecker) AppendDataSource(item *gctkline.Item, exch gctexchange.IBotExchange, dataType int64) error {
	if l == nil {
		return fmt.Errorf("%w LiveDataChecker", gctcommon.ErrNilPointer)
	}
	l.m.Lock()
	defer l.m.Unlock()

	for i := range l.exchangesToCheck {
		if strings.ToLower(l.exchangesToCheck[i].exchange.GetName()) == item.Exchange &&
			l.exchangesToCheck[i].asset == item.Asset &&
			l.exchangesToCheck[i].pair.Equal(item.Pair) {
			return funding.ErrAlreadyExists
		}
	}

	dataeroo := kline.DataFromKline{
		Item: *item,
	}
	underlying := currency.EMPTYPAIR
	if item.Asset.IsFutures() {
		curr, _, err := exch.GetCollateralCurrencyForContract(item.Asset, item.Pair)
		if err != nil {
			return err
		}
		underlying.Base = item.Pair.Base
		underlying.Quote = curr
	}

	l.exchangesToCheck = append(l.exchangesToCheck, liveExchangeDataHandler{
		exchange:       exch,
		asset:          item.Asset,
		pair:           item.Pair,
		pairCandles:    dataeroo,
		dataType:       dataType,
		underlyingPair: underlying,
	})
	return nil
}

func (l *LiveDataChecker) FetchLatestData() error {
	if l == nil {
		return fmt.Errorf("%w LiveDataChecker", gctcommon.ErrNilPointer)
	}
	l.m.Lock()
	defer l.m.Unlock()
	var err error
	for i := range l.exchangesToCheck {
		err = l.exchangesToCheck[i].loadCandleData()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *liveExchangeDataHandler) loadCandleData() error {
	if c == nil {
		return gctcommon.ErrNilPointer
	}
	c.m.Lock()
	defer c.m.Unlock()
	candles, err := live.LoadData(context.TODO(),
		c.exchange,
		c.dataType,
		c.pairCandles.Item.Interval.Duration(),
		c.pair,
		c.asset)
	if err != nil {
		return err
	}
	if len(candles.Candles) == 0 {
		return nil
	}
	c.pairCandles.AppendResults(candles)
	return c.pairCandles.Load()
}

// RunLive is a proof of concept function that does not yet support multi currency usage
// It runs by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	log.Info(common.Livetester, "running backtester against live data")
	timer := bt.EventQueue.GetRunTimer()
	timeout := bt.EventQueue.GetNewEventTimeout()
	timeoutTimer := time.NewTimer(timeout)
	// a frequent timer so that when a new candle is released by an exchange
	// that it can be processed quickly
	processEventTicker := time.NewTimer(0)
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-timeoutTimer.C:
			return fmt.Errorf("%w %v", errLiveDataTimeout, timeout)
		case <-processEventTicker.C:
			bt.Run()
			processEventTicker.Reset(timer)
			timeoutTimer.Reset(timeout)
		}
	}
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func (bt *BackTest) loadLiveDataLoop(cfg *config.Config, verboseDataCheck bool) {
	loadNewDataTimer := time.NewTimer(0)
	dataCheckInterval := bt.EventQueue.GetDataCheckTimer()
	bt.EventQueue.GetRunTimer()
	exchangeMap := bt.Datas.GetAllData()
	dataType, err := common.DataTypeToInt(cfg.DataSettings.DataType)
	if err != nil {
		log.Error(common.Livetester, err)
		return
	}

	for {
		select {
		case <-bt.shutdown:
			return
		case <-loadNewDataTimer.C:
			for exchName, assetMap := range exchangeMap {
				var exch gctexchange.IBotExchange
				exch, err = bt.exchangeManager.GetExchangeByName(exchName)
				if err != nil {
					log.Error(common.Livetester, err)
				}

				for a, pairMap := range assetMap {
					for pair, handler := range pairMap {
						if verboseDataCheck {
							log.Infof(common.Livetester, "%v has passed, fetching data for %v %v %v ", dataCheckInterval, exch.GetName(), a, pair)
						}
						err = bt.loadLiveData(handler, exch, a, pair, cfg.DataSettings.Interval, dataType)
						if err != nil {
							log.Error(common.Livetester, err)
						}
					}
				}
			}
			loadNewDataTimer.Reset(dataCheckInterval)
		}
	}
}

func (bt *BackTest) loadLiveData(handler data.Handler, exch gctexchange.IBotExchange, a asset.Item, cp currency.Pair, interval gctkline.Interval, dataType int64) error {
	if exch == nil {
		return errNilExchange
	}
	appendKline := kline.DataFromKline{
		Base: handler.GetBase(),
		Item: gctkline.Item{
			Exchange:        "",
			Pair:            currency.Pair{},
			UnderlyingPair:  currency.Pair{},
			Asset:           0,
			Interval:        0,
			Candles:         nil,
			SourceJobID:     uuid.UUID{},
			ValidationJobID: uuid.UUID{},
		},
		RangeHolder: nil,
	}
	candles, err := live.LoadData(context.TODO(),
		exch,
		dataType,
		interval.Duration(),
		cp,
		a)
	if err != nil {
		return err
	}
	if len(candles.Candles) == 0 {
		return nil
	}

	if a.IsFutures() {
		// returning the collateral currency along with using the
		// cp base creates a pair that links the futures contract to
		// is underlying pair
		// eg BTC-PERP on FTX has a collateral currency of USD
		// taking the BTC base and USD as quote, allows linking
		// BTC-USD and BTC-PERP
		var curr currency.Code
		curr, _, err = exch.GetCollateralCurrencyForContract(a, cp)
		if err != nil {
			return err
		}
		resp.Item.UnderlyingPair = currency.NewPair(cp.Base, curr)
	}
	handler.AppendStream()
	resp.AppendResults(candles)
	err = bt.Reports.UpdateItem(&resp.Item)
	if err != nil {
		return err
	}

	err = bt.Funding.AddUSDTrackingData(resp)
	if err != nil &&
		!errors.Is(err, trackingcurrencies.ErrCurrencyDoesNotContainsUSD) &&
		!errors.Is(err, funding.ErrUSDTrackingDisabled) {
		return err
	}
	return nil
}
