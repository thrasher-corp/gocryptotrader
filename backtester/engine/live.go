package engine

import (
	"context"
	"errors"
	"fmt"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupLiveDataHandler creates a live data handler to retrieve and append
// live data as it comes in
func (bt *BackTest) SetupLiveDataHandler(eventTimeout, dataCheckInterval time.Duration, verbose bool) error {
	if bt == nil {
		return fmt.Errorf("%w backtester", gctcommon.ErrNilPointer)
	}
	if bt.exchangeManager == nil {
		return fmt.Errorf("%w engine manager", gctcommon.ErrNilPointer)
	}
	if bt.DataHolder == nil {
		return fmt.Errorf("%w data holder", gctcommon.ErrNilPointer)
	}
	if bt.Reports == nil {
		return fmt.Errorf("%w reports", gctcommon.ErrNilPointer)
	}
	if bt.Funding == nil {
		return fmt.Errorf("%w funding manager", gctcommon.ErrNilPointer)
	}
	if eventTimeout <= 0 {
		log.Warnf(common.LiveStrategy, "invalid event timeout '%v', defaulting to '%v'", eventTimeout, defaultEventTimeout)
		eventTimeout = defaultEventTimeout
	}
	if dataCheckInterval <= 0 {
		log.Warnf(common.LiveStrategy, "invalid data check interval '%v', defaulting to '%v'", dataCheckInterval, defaultDataCheckInterval)
		dataCheckInterval = defaultDataCheckInterval
	}
	bt.LiveDataHandler = &dataChecker{
		verboseDataCheck:  verbose,
		exchangeManager:   bt.exchangeManager,
		eventTimeout:      eventTimeout,
		dataCheckInterval: dataCheckInterval,
		dataHolder:        bt.DataHolder,
		shutdown:          make(chan struct{}),
		report:            bt.Reports,
		funding:           bt.Funding,
	}
	return nil
}

// Start begins fetching and appending live data
func (l *dataChecker) Start() error {
	if l == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&l.started, 0, 1) {
		return engine.ErrSubSystemAlreadyStarted
	}
	l.shutdown = make(chan struct{})
	l.wg.Add(1)
	go func() {
		err := l.DataFetcher()
		if err != nil {
			log.Error(common.LiveStrategy, err)
			err2 := l.Stop()
			if err2 != nil {
				log.Error(common.LiveStrategy, err2)
			}
		}
	}()

	return nil
}

// IsRunning verifies whether the live data checker is running
func (l *dataChecker) IsRunning() bool {
	return l != nil && atomic.LoadUint32(&l.started) == 1
}

// Stop ceases fetching and processing live data
func (l *dataChecker) Stop() error {
	if l == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&l.started, 1, 0) {
		return engine.ErrSubSystemNotStarted
	}
	l.m.Lock()
	defer l.m.Unlock()
	l.hasShutdown.Alert()
	close(l.shutdown)
	l.wg.Wait()
	return nil
}

// DataFetcher will fetch and append live data
func (l *dataChecker) DataFetcher() error {
	if l == nil {
		return fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	defer func() {
		l.wg.Done()
	}()
	if atomic.LoadUint32(&l.started) == 0 {
		return engine.ErrSubSystemNotStarted
	}
	checkTimer := time.NewTimer(0)
	timeoutTimer := time.NewTimer(l.eventTimeout)
	var err error
	for {
		select {
		case <-l.shutdown:
			return nil
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
			l.notice.Alert()
			if !timeoutTimer.Stop() {
				// drain to avoid closure
				<-timeoutTimer.C
			}
			timeoutTimer.Reset(l.eventTimeout)
		case <-timeoutTimer.C:
			return fmt.Errorf("%w of %v", ErrLiveDataTimeout, l.eventTimeout)
		}
	}
}

// Updated gives other endpoints the ability to listen to
// when data is updated from live sources
func (l *dataChecker) Updated() <-chan bool {
	if l == nil {
		immediateClosure := make(chan bool)
		defer close(immediateClosure)
		return immediateClosure
	}
	return l.notice.Wait(l.shutdown)
}

// HasShutdown indicates when the live data checker
// has been shutdown
func (l *dataChecker) HasShutdown() <-chan bool {
	if l == nil {
		immediateClosure := make(chan bool)
		defer close(immediateClosure)
		return immediateClosure
	}
	return l.hasShutdown.Wait(nil)
}

// Reset clears all stored data
func (l *dataChecker) Reset() {
	if l == nil {
		return
	}
	l.m.Lock()
	defer l.m.Unlock()
	l.dataCheckInterval = 0
	l.eventTimeout = 0
	l.exchangeManager = nil
	l.sourcesToCheck = nil
	l.exchangeManager = nil
	l.verboseDataCheck = false
	l.wg = sync.WaitGroup{}
}

// AppendDataSource stores params to allow the datachecker to fetch and append live data
func (l *dataChecker) AppendDataSource(dataSource *liveDataSourceSetup) error {
	if l == nil {
		return fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	if dataSource == nil {
		return fmt.Errorf("%w live data source", gctcommon.ErrNilPointer)
	}
	if dataSource.exchange == nil {
		return fmt.Errorf("%w IBotExchange", gctcommon.ErrNilPointer)
	}
	if dataSource.dataType != common.DataCandle && dataSource.dataType != common.DataTrade {
		return fmt.Errorf("%w '%v'", common.ErrInvalidDataType, dataSource.dataType)
	}
	if !dataSource.asset.IsValid() {
		return fmt.Errorf("%w '%v'", asset.ErrNotSupported, dataSource.asset)
	}
	if dataSource.pair.IsEmpty() {
		return fmt.Errorf("main %w", currency.ErrCurrencyPairEmpty)
	}
	if dataSource.interval.Duration() == 0 {
		return gctkline.ErrUnsetInterval
	}
	l.m.Lock()
	defer l.m.Unlock()
	exchName := strings.ToLower(dataSource.exchange.GetName())
	for i := range l.sourcesToCheck {
		if l.sourcesToCheck[i].exchangeName == exchName &&
			l.sourcesToCheck[i].asset == dataSource.asset &&
			l.sourcesToCheck[i].pair.Equal(dataSource.pair) {
			return fmt.Errorf("%w %v %v %v", errDataSourceExists, exchName, dataSource.asset, dataSource.pair)
		}
	}

	d := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange:       exchName,
			Pair:           dataSource.pair,
			UnderlyingPair: dataSource.underlyingPair,
			Asset:          dataSource.asset,
			Interval:       dataSource.interval,
		},
	}
	d.SetLive(true)
	if dataSource.dataRequestRetryTolerance == 0 {
		dataSource.dataRequestRetryTolerance = 1
	}
	if dataSource.dataRequestRetryWaitTime <= 0 {
		dataSource.dataRequestRetryWaitTime = defaultDataRequestWaitTime
	}
	l.sourcesToCheck = append(l.sourcesToCheck, &liveDataSourceDataHandler{
		exchange:                  dataSource.exchange,
		exchangeName:              exchName,
		asset:                     dataSource.asset,
		pair:                      dataSource.pair,
		underlyingPair:            dataSource.underlyingPair,
		pairCandles:               d,
		dataType:                  dataSource.dataType,
		processedData:             make(map[int64]struct{}),
		dataRequestRetryTolerance: dataSource.dataRequestRetryTolerance,
		dataRequestRetryWaitTime:  dataSource.dataRequestRetryWaitTime,
		verboseExchangeRequest:    dataSource.verboseExchangeRequest,
	})

	return nil
}

// FetchLatestData loads the latest data for all stored data sources
func (l *dataChecker) FetchLatestData() (bool, error) {
	if l == nil {
		return false, fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	if atomic.LoadUint32(&l.started) == 0 {
		return false, engine.ErrSubSystemNotStarted
	}
	l.m.Lock()
	defer l.m.Unlock()
	var err error

	var results []bool
	// timeToRetrieve ensures consistent data retrieval
	// in the event of a candle rollover mid-loop
	timeToRetrieve := time.Now()
	for i := range l.sourcesToCheck {
		if l.verboseDataCheck {
			log.Infof(common.LiveStrategy, "%v %v %v checking for new data", l.sourcesToCheck[i].exchangeName, l.sourcesToCheck[i].asset, l.sourcesToCheck[i].pair)
		}
		var updated bool
		updated, err = l.sourcesToCheck[i].loadCandleData(timeToRetrieve)
		if err != nil {
			return false, err
		}
		results = append(results, updated)
	}
	allUpdated := true
	for i := range results {
		if !results[i] {
			allUpdated = false
		}
	}
	if !allUpdated {
		return false, nil
	}
	for i := range l.sourcesToCheck {
		if l.verboseDataCheck {
			log.Infof(common.LiveStrategy, "%v %v %v found new data", l.sourcesToCheck[i].exchangeName, l.sourcesToCheck[i].asset, l.sourcesToCheck[i].pair)
		}
		l.sourcesToCheck[i].pairCandles.AppendResults(l.sourcesToCheck[i].candlesToAppend)
		l.sourcesToCheck[i].candlesToAppend.Candles = nil
		l.dataHolder.SetDataForCurrency(l.sourcesToCheck[i].exchangeName, l.sourcesToCheck[i].asset, l.sourcesToCheck[i].pair, &l.sourcesToCheck[i].pairCandles)
		err = l.report.SetKlineData(&l.sourcesToCheck[i].pairCandles.Item)
		if err != nil {
			log.Errorf(common.LiveStrategy, "%v %v %v issue processing kline data: %v", l.sourcesToCheck[i].exchangeName, l.sourcesToCheck[i].asset, l.sourcesToCheck[i].pair, err)
		}
		err = l.funding.AddUSDTrackingData(&l.sourcesToCheck[i].pairCandles)
		if err != nil && !errors.Is(err, funding.ErrUSDTrackingDisabled) {
			log.Errorf(common.LiveStrategy, "%v %v %v issue processing USD tracking data: %v", l.sourcesToCheck[i].exchangeName, l.sourcesToCheck[i].asset, l.sourcesToCheck[i].pair, err)
		}
	}
	return true, nil
}

var errNoDataSetForClosingPositions = errors.New("no data was set for closing positions")

// SetDataForClosingAllPositions is triggered on a live data run
// when closing all positions on close is true.
// it will ensure all data is set such as USD tracking data
func (l *dataChecker) SetDataForClosingAllPositions(s ...signal.Event) error {
	if l == nil {
		return fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	if len(s) == 0 {
		return fmt.Errorf("%w signal events", common.ErrNilArguments)
	}
	l.m.Lock()
	defer l.m.Unlock()
	var err error

	setData := false
	for x := range s {
		if s[x] == nil {
			return fmt.Errorf("%w signal events", errNilData)
		}
		for y := range l.sourcesToCheck {
			if s[x].GetExchange() != l.sourcesToCheck[y].exchangeName ||
				s[x].GetAssetType() != l.sourcesToCheck[y].asset ||
				!s[x].Pair().Equal(l.sourcesToCheck[y].pair) {
				continue
			}
			l.sourcesToCheck[y].pairCandles.Item.Candles = append(l.sourcesToCheck[y].pairCandles.Item.Candles, gctkline.Candle{
				Time:   s[x].GetTime(),
				Open:   s[x].GetOpenPrice().InexactFloat64(),
				High:   s[x].GetHighPrice().InexactFloat64(),
				Low:    s[x].GetLowPrice().InexactFloat64(),
				Close:  s[x].GetClosePrice().InexactFloat64(),
				Volume: s[x].GetVolume().InexactFloat64(),
			})
			l.dataHolder.SetDataForCurrency(l.sourcesToCheck[y].exchangeName, l.sourcesToCheck[y].asset, l.sourcesToCheck[y].pair, &l.sourcesToCheck[y].pairCandles)
			err = l.report.SetKlineData(&l.sourcesToCheck[y].pairCandles.Item)
			if err != nil {
				log.Errorf(common.LiveStrategy, "%v %v %v issue processing kline data: %v", l.sourcesToCheck[y].exchangeName, l.sourcesToCheck[y].asset, l.sourcesToCheck[y].pair, err)
			}
			err = l.funding.AddUSDTrackingData(&l.sourcesToCheck[y].pairCandles)
			if err != nil && !errors.Is(err, funding.ErrUSDTrackingDisabled) {
				log.Errorf(common.LiveStrategy, "%v %v %v issue processing USD tracking data: %v", l.sourcesToCheck[y].exchangeName, l.sourcesToCheck[y].asset, l.sourcesToCheck[y].pair, err)
			}
			setData = true
		}
	}
	if !setData {
		return errNoDataSetForClosingPositions
	}

	return nil
}

// loadCandleData fetches data from the exchange API and appends it
// to the candles to be added to the backtester event queue
func (c *liveDataSourceDataHandler) loadCandleData(timeToRetrieve time.Time) (bool, error) {
	if c == nil {
		return false, gctcommon.ErrNilPointer
	}
	var candles *gctkline.Item
	var err error
	for i := int64(1); i <= c.dataRequestRetryTolerance; i++ {
		candles, err = live.LoadData(context.TODO(),
			timeToRetrieve,
			c.exchange,
			c.dataType,
			c.pairCandles.Item.Interval.Duration(),
			c.pair,
			c.underlyingPair,
			c.asset,
			c.verboseExchangeRequest)
		if err != nil {
			if i < c.dataRequestRetryTolerance {
				log.Errorf(common.Data, "%v %v %v failed to retrieve data %v of %v attempts: %v", c.exchangeName, c.asset, c.pair, i, c.dataRequestRetryTolerance, err)
				continue
			} else {
				return false, err
			}
		}
		break
	}
	if candles == nil {
		return false, fmt.Errorf("%w kline Asset", gctcommon.ErrNilPointer)
	}
	if len(candles.Candles) == 0 {
		return false, nil
	}
	var unprocessedCandles []gctkline.Candle
	for i := range candles.Candles {
		if _, ok := c.processedData[candles.Candles[i].Time.UnixNano()]; !ok {
			unprocessedCandles = append(unprocessedCandles, candles.Candles[i])
			c.processedData[candles.Candles[i].Time.UnixNano()] = struct{}{}
		}
	}
	if len(unprocessedCandles) > 0 {
		if c.candlesToAppend == nil {
			c.candlesToAppend = candles
			c.candlesToAppend.Candles = unprocessedCandles
		} else {
			c.candlesToAppend.Candles = append(c.candlesToAppend.Candles, unprocessedCandles...)
		}
		return true, nil
	}
	return false, nil
}
