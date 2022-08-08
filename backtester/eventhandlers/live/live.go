package live

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
)

// SetupLiveDataHandler creates a live data handler to retrieve and append
// live data as it comes in
func SetupLiveDataHandler(em *engine.ExchangeManager, dataHolder data.Holder, eventCheckInterval, eventTimeout, dataCheckInterval time.Duration, verbose bool) (Handler, error) {
	if em == nil {
		return nil, fmt.Errorf("%w engine manager", gctcommon.ErrNilPointer)
	}
	if dataHolder == nil {
		return nil, fmt.Errorf("%w data holder", gctcommon.ErrNilPointer)
	}
	if eventCheckInterval <= 0 {
		log.Warnf(common.Livetester, "invalid event check interval '%v', defaulting to '%v'", eventCheckInterval, defaultEventCheckInterval)
		eventCheckInterval = defaultEventCheckInterval
	}
	if eventTimeout <= 0 {
		log.Warnf(common.Livetester, "invalid event timeout '%v', defaulting to '%v'", eventTimeout, defaultEventTimeout)
		eventTimeout = defaultEventTimeout
	}
	if dataCheckInterval <= 0 {
		log.Warnf(common.Livetester, "invalid data check interval '%v', defaulting to '%v'", dataCheckInterval, defaultDataCheckInterval)
		dataCheckInterval = defaultEventCheckInterval
	}
	return &DataChecker{
		exchangeManager:    em,
		eventCheckInterval: eventCheckInterval,
		eventTimeout:       eventTimeout,
		dataCheckInterval:  dataCheckInterval,
		verbose:            verbose,
		dataHolder:         dataHolder,
		updated:            make(chan struct{}),
		shutdown:           make(chan struct{}),
	}, nil
}

// Start begins fetching and appending live data
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

// IsRunning verifies whether the live data checker is running
func (l *DataChecker) IsRunning() bool {
	return l != nil && atomic.LoadUint32(&l.started) == 1
}

// Stop ceases fetching and processing live data
func (l *DataChecker) Stop() error {
	if l == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&l.started, 1, 0) {
		return engine.ErrSubSystemNotStarted
	}
	l.m.Lock()
	defer l.m.Unlock()
	close(l.shutdown)
	l.wg.Wait()
	l.shutdown = make(chan struct{})
	return nil
}

// DataFetcher will fetch and append live data
func (l *DataChecker) DataFetcher() error {
	if l == nil {
		return fmt.Errorf("%w DataChecker", gctcommon.ErrNilPointer)
	}
	if atomic.LoadUint32(&l.started) == 0 {
		return engine.ErrSubSystemNotStarted
	}
	defer l.wg.Done()
	checkTimer := time.NewTimer(0)
	timeoutTimer := time.NewTimer(l.eventTimeout)
	var err error
	for {
		select {
		case <-l.shutdown:
			return nil
		case <-timeoutTimer.C:
			return fmt.Errorf("%w of %v", ErrLiveDataTimeout, l.eventTimeout)
		case <-checkTimer.C:
			checkTimer.Reset(l.dataCheckInterval)
			var updated bool
			updated, err = l.FetchLatestData()
			if err != nil {
				return err
			}
			if !updated {
				continue
			}
			close(l.updated)
			l.updated = make(chan struct{})
			timeoutTimer.Reset(l.eventTimeout)
		}
	}
}

// Updated gives other endpoints the ability to listen to
// when data is updated from live sources
func (l *DataChecker) Updated() chan struct{} {
	if l == nil {
		immediateClosure := make(chan struct{})
		defer close(immediateClosure)
		return immediateClosure
	}
	return l.updated
}

// Reset clears all stored data
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
	l.shutdown = nil
	l.updated = nil
	l.exchangeManager = nil
	l.verbose = false
	l.wg = sync.WaitGroup{}
}

// AppendDataSource stores params to allow the datachecker to fetch and append live data
func (l *DataChecker) AppendDataSource(item *gctkline.Item, exch gctexchange.IBotExchange, dataType int64) error {
	if l == nil {
		return fmt.Errorf("%w DataChecker", gctcommon.ErrNilPointer)
	}
	if atomic.LoadUint32(&l.started) == 0 {
		return engine.ErrSubSystemNotStarted
	}
	if item == nil {
		return fmt.Errorf("%w kline item", gctcommon.ErrNilPointer)
	}
	if exch == nil {
		return fmt.Errorf("%w IBotExchange", gctcommon.ErrNilPointer)
	}
	if dataType != common.DataCandle && dataType != common.DataTrade {
		return fmt.Errorf("%w '%v'", common.ErrInvalidDataType, dataType)
	}
	if !item.Asset.IsValid() {
		return fmt.Errorf("%w '%v'", asset.ErrNotSupported, item.Asset)
	}
	if item.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
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

	d := kline.DataFromKline{
		Item: *item,
	}
	d.SetLive(true)
	l.exchangesToCheck = append(l.exchangesToCheck, liveExchangeDataHandler{
		exchange:       exch,
		exchangeName:   strings.ToLower(exch.GetName()),
		asset:          item.Asset,
		pair:           item.Pair,
		pairCandles:    d,
		dataType:       dataType,
		underlyingPair: item.UnderlyingPair,
	})

	return nil
}

// FetchLatestData loads the latest data for all stored data sources
func (l *DataChecker) FetchLatestData() (bool, error) {
	if l == nil {
		return false, fmt.Errorf("%w DataChecker", gctcommon.ErrNilPointer)
	}
	if atomic.LoadUint32(&l.started) == 0 {
		return false, engine.ErrSubSystemNotStarted
	}
	l.m.Lock()
	defer l.m.Unlock()
	var err error
	var updated bool
	for i := range l.exchangesToCheck {
		if !l.verbose {
			log.Infof(common.Livetester, "fetching live data for %v %v %v", l.exchangesToCheck[i].exchangeName, l.exchangesToCheck[i].asset, l.exchangesToCheck[i].pair)
		}
		preCandleLen := len(l.exchangesToCheck[i].pairCandles.Item.Candles)
		err = l.exchangesToCheck[i].loadCandleData()
		if err != nil {
			return false, err
		}
		l.dataHolder.SetDataForCurrency(l.exchangesToCheck[i].exchangeName, l.exchangesToCheck[i].asset, l.exchangesToCheck[i].pair, &l.exchangesToCheck[i].pairCandles)
		if len(l.exchangesToCheck[i].pairCandles.Item.Candles) > preCandleLen {
			updated = true
		}
	}
	return updated, nil
}

func (l *DataChecker) GetKlines() []kline.DataFromKline {
	var response []kline.DataFromKline
	for i := range l.exchangesToCheck {
		response = append(response, l.exchangesToCheck[i].pairCandles)
	}
	return response
}

// loadCandleData fetches data from the exchange API and appends it
// to the candles to be added to the backtester event queue
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
		c.underlyingPair,
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
