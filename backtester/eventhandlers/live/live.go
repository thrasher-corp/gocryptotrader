package live

import (
	"context"
	"errors"
	"fmt"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var ErrLiveDataTimeout = errors.New("shutting down due to no data returned in")

// Handler is all the functionality required in order to
// run a backtester with live data
type Handler interface {
	AppendDataSource(item *gctkline.Item, exch gctexchange.IBotExchange, dataType int64) error
	FetchLatestData() error
	Start() error
	IsRunning() bool
	DataFetcher() error
	Stop() error
	Reset()
	Updated() chan struct{}
	// AppendUSDTrackingData ??
}

// DataChecker is responsible for managing all data retrieval
// for a live data option
type DataChecker struct {
	m                  sync.Mutex
	wg                 sync.WaitGroup
	started            uint32
	verbose            bool
	exchangeManager    *engine.ExchangeManager
	exchangesToCheck   []liveExchangeDataHandler
	eventCheckInterval time.Duration
	eventTimeout       time.Duration
	dataCheckInterval  time.Duration
	datas              data.Holder
	updated            chan struct{}
	shutdown           chan struct{}
}

type ExchangeHandler interface {
	UpdateData() error
}

type liveExchangeDataHandler struct {
	m              sync.Mutex
	exchange       gctexchange.IBotExchange
	exchangeName   string
	asset          asset.Item
	pair           currency.Pair
	underlyingPair currency.Pair
	pairCandles    kline.DataFromKline
	dataType       int64
}

func SetupLiveDataHandler(em *engine.ExchangeManager, datas data.Holder, evenCheckInterval, eventTimeout, dataCheckInterval time.Duration, verbose bool) (Handler, error) {
	if em == nil {
		return nil, fmt.Errorf("%w engine manager", gctcommon.ErrNilPointer)
	}
	return &DataChecker{
		exchangeManager:    em,
		eventCheckInterval: evenCheckInterval,
		eventTimeout:       eventTimeout,
		dataCheckInterval:  dataCheckInterval,
		verbose:            verbose,
		datas:              datas,
		updated:            make(chan struct{}),
	}, nil
}

func (l *DataChecker) Updated() chan struct{} {
	return l.updated
}

func (l *DataChecker) Start() error {
	if l == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&l.started, 0, 1) {
		return engine.ErrSubSystemAlreadyStarted
	}
	l.wg.Add(1)
	go func() {
		err := l.DataFetcher()
		if err != nil {
			return
		}
	}()

	return nil
}

func (l *DataChecker) IsRunning() bool {
	return l != nil && atomic.LoadUint32(&l.started) == 1
}

func (l *DataChecker) Stop() error {
	if l == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&l.started, 0, 1) {
		return engine.ErrSubSystemNotStarted
	}
	l.m.Lock()
	defer l.m.Unlock()
	close(l.shutdown)
	l.wg.Wait()
	l.shutdown = make(chan struct{})
	return nil
}

func (l *DataChecker) DataFetcher() error {
	checkTimer := time.NewTimer(0)
	timeoutTimer := time.NewTimer(l.eventTimeout)
	var err error
	for {
		select {
		case <-l.shutdown:
			return nil
		case <-timeoutTimer.C:
			return ErrLiveDataTimeout
		case <-checkTimer.C:
			if !l.verbose {
				log.Info(common.Livetester, "fetching data...")
			}
			err = l.FetchLatestData()
			if err != nil {
				return err
			}
			if !l.verbose {
				log.Info(common.Livetester, "fetching data... complete")
			}
			l.updated <- struct{}{}
			checkTimer.Reset(l.dataCheckInterval)
			timeoutTimer.Reset(l.eventTimeout)
		}
	}
}

func (l *DataChecker) Reset() {
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

func (l *DataChecker) AppendDataSource(item *gctkline.Item, exch gctexchange.IBotExchange, dataType int64) error {
	if l == nil {
		return fmt.Errorf("%w DataChecker", gctcommon.ErrNilPointer)
	}
	l.m.Lock()
	defer l.m.Unlock()

	for i := range l.exchangesToCheck {
		if l.exchangesToCheck[i].exchangeName == item.Exchange &&
			l.exchangesToCheck[i].asset == item.Asset &&
			l.exchangesToCheck[i].pair.Equal(item.Pair) {
			return funding.ErrAlreadyExists
		}
	}

	dataeroo := kline.DataFromKline{
		Item: *item,
	}
	dataeroo.SetLive(true)
	l.exchangesToCheck = append(l.exchangesToCheck, liveExchangeDataHandler{
		exchange:       exch,
		exchangeName:   strings.ToLower(exch.GetName()),
		asset:          item.Asset,
		pair:           item.Pair,
		pairCandles:    dataeroo,
		dataType:       dataType,
		underlyingPair: item.UnderlyingPair,
	})

	return nil
}

func (l *DataChecker) FetchLatestData() error {
	if l == nil {
		return fmt.Errorf("%w DataChecker", gctcommon.ErrNilPointer)
	}
	if atomic.LoadUint32(&l.started) == 0 {
		return engine.ErrSubSystemAlreadyStarted
	}
	l.m.Lock()
	defer l.m.Unlock()
	var err error
	for i := range l.exchangesToCheck {
		err = l.exchangesToCheck[i].loadCandleData()
		if err != nil {
			return err
		}
		l.datas.SetDataForCurrency(l.exchangesToCheck[i].exchangeName, l.exchangesToCheck[i].asset, l.exchangesToCheck[i].pair, &l.exchangesToCheck[i].pairCandles)

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
	candles.UnderlyingPair = c.underlyingPair
	if err != nil {
		return err
	}
	if len(candles.Candles) == 0 {
		return nil
	}
	c.pairCandles.AppendResults(candles)
	return nil
}
