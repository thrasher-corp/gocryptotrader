package engine

import (
	"context"
	"errors"
	"fmt"
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
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupLiveDataHandler creates a live data handler to retrieve and append
// live data as it comes in
func (bt *BackTest) SetupLiveDataHandler(eventCheckInterval, eventTimeout, dataCheckInterval time.Duration, verbose bool) (Handler, error) {
	if bt.exchangeManager == nil {
		return nil, fmt.Errorf("%w engine manager", gctcommon.ErrNilPointer)
	}
	if bt.DataHolder == nil {
		return nil, fmt.Errorf("%w data holder", gctcommon.ErrNilPointer)
	}
	if bt.Reports == nil {
		return nil, fmt.Errorf("%w reports", gctcommon.ErrNilPointer)
	}
	if bt.Funding == nil {
		return nil, fmt.Errorf("%w funding manager", gctcommon.ErrNilPointer)
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
		verbose:            verbose,
		exchangeManager:    bt.exchangeManager,
		eventCheckInterval: eventCheckInterval,
		eventTimeout:       eventTimeout,
		dataCheckInterval:  dataCheckInterval,
		dataHolder:         bt.DataHolder,
		updated:            make(chan struct{}),
		shutdown:           make(chan struct{}),
		report:             bt.Reports,
		funding:            bt.Funding,
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
	l.shutdown = make(chan struct{})
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
	return nil
}

// DataFetcher will fetch and append live data
func (l *DataChecker) DataFetcher() error {
	if l == nil {
		return fmt.Errorf("%w DataChecker", gctcommon.ErrNilPointer)
	}
	defer l.wg.Done()
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
			close(l.updated)
			l.updated = make(chan struct{})
			timeoutTimer.Reset(l.eventTimeout)
		case <-timeoutTimer.C:
			return fmt.Errorf("%w of %v", ErrLiveDataTimeout, l.eventTimeout)
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
func (l *DataChecker) AppendDataSource(exch gctexchange.IBotExchange, interval gctkline.Interval, item asset.Item, curr, underlying currency.Pair, dataType int64) error {
	if l == nil {
		return fmt.Errorf("%w DataChecker", gctcommon.ErrNilPointer)
	}
	if exch == nil {
		return fmt.Errorf("%w IBotExchange", gctcommon.ErrNilPointer)
	}
	if dataType != common.DataCandle && dataType != common.DataTrade {
		return fmt.Errorf("%w '%v'", common.ErrInvalidDataType, dataType)
	}
	if !item.IsValid() {
		return fmt.Errorf("%w '%v'", asset.ErrNotSupported, item)
	}
	if curr.IsEmpty() {
		return fmt.Errorf("main %w", currency.ErrCurrencyPairEmpty)
	}
	l.m.Lock()
	defer l.m.Unlock()
	exchName := strings.ToLower(exch.GetName())
	for i := range l.exchangesToCheck {
		if l.exchangesToCheck[i].exchangeName == exchName &&
			l.exchangesToCheck[i].asset == item &&
			l.exchangesToCheck[i].pair.Equal(curr) {
			return funding.ErrAlreadyExists
		}
	}

	d := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange:       exchName,
			Pair:           curr,
			UnderlyingPair: underlying,
			Asset:          item,
			Interval:       interval,
		},
	}
	d.SetLive(true)
	l.exchangesToCheck = append(l.exchangesToCheck, liveExchangeDataHandler{
		exchange:       exch,
		exchangeName:   exchName,
		asset:          item,
		pair:           curr,
		pairCandles:    d,
		dataType:       dataType,
		underlyingPair: underlying,
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
		err = l.report.SetKlineData(&l.exchangesToCheck[i].pairCandles.Item)
		if err != nil {
			log.Errorf(common.Livetester, "issue processing kline data: %v", err)
		}
		err = l.funding.AddUSDTrackingData(&l.exchangesToCheck[i].pairCandles)
		if err != nil && !errors.Is(err, funding.ErrUSDTrackingDisabled) {
			log.Errorf(common.Livetester, "issue processing USD tracking data: %v", err)
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
