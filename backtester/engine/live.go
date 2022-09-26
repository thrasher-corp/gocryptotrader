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
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
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
func (bt *BackTest) SetupLiveDataHandler(eventTimeout, dataCheckInterval time.Duration, realOrders, verbose bool) error {
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
		realOrders:        realOrders,
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
func (d *dataChecker) Start() error {
	if d == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&d.started, 0, 1) {
		return engine.ErrSubSystemAlreadyStarted
	}
	d.shutdown = make(chan struct{})

	d.wg.Add(1)
	go func() {
		err := d.DataFetcher()
		if err != nil {
			log.Error(common.LiveStrategy, err)
			stopErr := d.Stop()
			if stopErr != nil {
				log.Error(common.LiveStrategy, stopErr)
			}
		}
	}()

	return nil
}

// IsRunning verifies whether the live data checker is running
func (d *dataChecker) IsRunning() bool {
	return d != nil && atomic.LoadUint32(&d.started) == 1
}

// Stop ceases fetching and processing live data
func (d *dataChecker) Stop() error {
	if d == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&d.started, 1, 0) {
		return engine.ErrSubSystemNotStarted
	}
	d.m.Lock()
	defer d.m.Unlock()
	d.hasShutdown.Alert()
	close(d.shutdown)
	d.wg.Wait()
	return nil
}

// DataFetcher will fetch and append live data
func (d *dataChecker) DataFetcher() error {
	if d == nil {
		return fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	defer func() {
		d.wg.Done()
	}()
	if atomic.LoadUint32(&d.started) == 0 {
		return engine.ErrSubSystemNotStarted
	}
	checkTimer := time.NewTimer(0)
	timeoutTimer := time.NewTimer(d.eventTimeout)
	for {
		select {
		case <-d.shutdown:
			return nil
		case <-checkTimer.C:
			err := d.checkData(checkTimer, timeoutTimer)
			if err != nil {
				return err
			}
		case <-timeoutTimer.C:
			return fmt.Errorf("%w of %v", ErrLiveDataTimeout, d.eventTimeout)
		}
	}
}

func (d *dataChecker) checkData(checkTimer, timeoutTimer *time.Timer) error {
	if checkTimer == nil || timeoutTimer == nil {
		return fmt.Errorf("%w timer", gctcommon.ErrNilPointer)
	}
	defer func() {
		checkTimer.Reset(d.dataCheckInterval)
		if !timeoutTimer.Stop() {
			// drain to avoid closure
			<-timeoutTimer.C
		}
		timeoutTimer.Reset(d.eventTimeout)
	}()
	updated, err := d.FetchLatestData()
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	d.notice.Alert()
	if d.realOrders {
		go func() {
			err = d.UpdateFunding(false)
			if err != nil {
				log.Errorf(common.LiveStrategy, "could not update funding %v", err)
			}
		}()
	}
	return nil
}

// UpdateFunding requests and updates funding levels
func (d *dataChecker) UpdateFunding(force bool) error {
	if d == nil || d.funding == nil {
		return gctcommon.ErrNilPointer
	}
	if force {
		atomic.StoreUint32(&d.updatingFunding, 1)
	} else if !atomic.CompareAndSwapUint32(&d.updatingFunding, 0, 1) {
		// already processing funding and can't go any faster
		return nil
	}
	defer atomic.StoreUint32(&d.updatingFunding, 0)

	var err error
	if d.funding.HasFutures() {
		err = d.funding.UpdateAllCollateral(d.realOrders, d.hasUpdatedFunding)
		if err != nil {
			return err
		}
	}

	if d.realOrders {
		// TODO: design a more sophisticated way of keeping funds up to date
		// with current data type retrieval, this still functions appropriately
		err = d.funding.UpdateFundingFromLiveData(d.hasUpdatedFunding)
		if err != nil {
			return err
		}
	}
	if !d.hasUpdatedFunding {
		d.hasUpdatedFunding = true
	}

	return nil
}

// Updated gives other endpoints the ability to listen to
// when data is updated from live sources
func (d *dataChecker) Updated() <-chan bool {
	if d == nil {
		immediateClosure := make(chan bool)
		defer close(immediateClosure)
		return immediateClosure
	}
	return d.notice.Wait(d.shutdown)
}

// HasShutdown indicates when the live data checker
// has been shutdown
func (d *dataChecker) HasShutdown() <-chan bool {
	if d == nil {
		immediateClosure := make(chan bool)
		defer close(immediateClosure)
		return immediateClosure
	}
	return d.hasShutdown.Wait(nil)
}

// Reset clears all stored data
func (d *dataChecker) Reset() {
	if d == nil {
		return
	}
	d.m.Lock()
	defer d.m.Unlock()
	d.dataCheckInterval = 0
	d.eventTimeout = 0
	d.exchangeManager = nil
	d.sourcesToCheck = nil
	d.exchangeManager = nil
	d.verboseDataCheck = false
	d.wg = sync.WaitGroup{}
}

// AppendDataSource stores params to allow the datachecker to fetch and append live data
func (d *dataChecker) AppendDataSource(dataSource *liveDataSourceSetup) error {
	if d == nil {
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
	d.m.Lock()
	defer d.m.Unlock()
	exchName := strings.ToLower(dataSource.exchange.GetName())
	for i := range d.sourcesToCheck {
		if d.sourcesToCheck[i].exchangeName == exchName &&
			d.sourcesToCheck[i].asset == dataSource.asset &&
			d.sourcesToCheck[i].pair.Equal(dataSource.pair) {
			return fmt.Errorf("%w %v %v %v", errDataSourceExists, exchName, dataSource.asset, dataSource.pair)
		}
	}

	k := kline.DataFromKline{
		Item: gctkline.Item{
			Exchange:       exchName,
			Pair:           dataSource.pair,
			UnderlyingPair: dataSource.underlyingPair,
			Asset:          dataSource.asset,
			Interval:       dataSource.interval,
		},
	}
	k.SetLive(true)
	if dataSource.dataRequestRetryTolerance == 0 {
		dataSource.dataRequestRetryTolerance = 1
	}
	if dataSource.dataRequestRetryWaitTime <= 0 {
		dataSource.dataRequestRetryWaitTime = defaultDataRequestWaitTime
	}
	d.sourcesToCheck = append(d.sourcesToCheck, &liveDataSourceDataHandler{
		exchange:                  dataSource.exchange,
		exchangeName:              exchName,
		asset:                     dataSource.asset,
		pair:                      dataSource.pair,
		underlyingPair:            dataSource.underlyingPair,
		pairCandles:               k,
		dataType:                  dataSource.dataType,
		processedData:             make(map[int64]struct{}),
		dataRequestRetryTolerance: dataSource.dataRequestRetryTolerance,
		dataRequestRetryWaitTime:  dataSource.dataRequestRetryWaitTime,
		verboseExchangeRequest:    dataSource.verboseExchangeRequest,
	})

	return nil
}

// FetchLatestData loads the latest data for all stored data sources
func (d *dataChecker) FetchLatestData() (bool, error) {
	if d == nil {
		return false, fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	if atomic.LoadUint32(&d.started) == 0 {
		return false, engine.ErrSubSystemNotStarted
	}
	d.m.Lock()
	defer d.m.Unlock()
	var err error

	results := make([]bool, len(d.sourcesToCheck))
	// timeToRetrieve ensures consistent data retrieval
	// in the event of a candle rollover mid-loop
	timeToRetrieve := time.Now()
	for i := range d.sourcesToCheck {
		if d.verboseDataCheck {
			log.Infof(common.LiveStrategy, "%v %v %v checking for new data", d.sourcesToCheck[i].exchangeName, d.sourcesToCheck[i].asset, d.sourcesToCheck[i].pair)
		}
		var updated bool
		updated, err = d.sourcesToCheck[i].loadCandleData(timeToRetrieve)
		if err != nil {
			return false, err
		}
		results[i] = updated
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
	for i := range d.sourcesToCheck {
		if d.verboseDataCheck {
			log.Infof(common.LiveStrategy, "%v %v %v found new data", d.sourcesToCheck[i].exchangeName, d.sourcesToCheck[i].asset, d.sourcesToCheck[i].pair)
		}
		d.sourcesToCheck[i].pairCandles.AppendResults(d.sourcesToCheck[i].candlesToAppend)
		d.sourcesToCheck[i].candlesToAppend.Candles = nil
		d.dataHolder.SetDataForCurrency(d.sourcesToCheck[i].exchangeName, d.sourcesToCheck[i].asset, d.sourcesToCheck[i].pair, &d.sourcesToCheck[i].pairCandles)
		err = d.report.SetKlineData(&d.sourcesToCheck[i].pairCandles.Item)
		if err != nil {
			log.Errorf(common.LiveStrategy, "%v %v %v issue processing kline data: %v", d.sourcesToCheck[i].exchangeName, d.sourcesToCheck[i].asset, d.sourcesToCheck[i].pair, err)
		}
		err = d.funding.AddUSDTrackingData(&d.sourcesToCheck[i].pairCandles)
		if err != nil && !errors.Is(err, funding.ErrUSDTrackingDisabled) {
			log.Errorf(common.LiveStrategy, "%v %v %v issue processing USD tracking data: %v", d.sourcesToCheck[i].exchangeName, d.sourcesToCheck[i].asset, d.sourcesToCheck[i].pair, err)
		}
	}
	if !d.hasUpdatedFunding {
		err = d.UpdateFunding(false)
		if err != nil {
			if err != nil {
				log.Error(common.LiveStrategy, err)
			}
		}
	}

	return true, nil
}

// SetDataForClosingAllPositions is triggered on a live data run
// when closing all positions on close is true.
// it will ensure all data is set such as USD tracking data
func (d *dataChecker) SetDataForClosingAllPositions(s ...signal.Event) error {
	if d == nil {
		return fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	if len(s) == 0 {
		return fmt.Errorf("%w signal events", gctcommon.ErrNilPointer)
	}
	d.m.Lock()
	defer d.m.Unlock()
	var err error

	setData := false
	for x := range s {
		if s[x] == nil {
			return fmt.Errorf("%w signal events", errNilData)
		}
		for y := range d.sourcesToCheck {
			if s[x].GetExchange() != d.sourcesToCheck[y].exchangeName ||
				s[x].GetAssetType() != d.sourcesToCheck[y].asset ||
				!s[x].Pair().Equal(d.sourcesToCheck[y].pair) {
				continue
			}
			d.sourcesToCheck[y].pairCandles.Item.Candles = append(d.sourcesToCheck[y].pairCandles.Item.Candles, gctkline.Candle{
				Time:   s[x].GetTime(),
				Open:   s[x].GetOpenPrice().InexactFloat64(),
				High:   s[x].GetHighPrice().InexactFloat64(),
				Low:    s[x].GetLowPrice().InexactFloat64(),
				Close:  s[x].GetClosePrice().InexactFloat64(),
				Volume: s[x].GetVolume().InexactFloat64(),
			})
			d.sourcesToCheck[y].pairCandles.AppendResults(&d.sourcesToCheck[y].pairCandles.Item)
			err = d.report.SetKlineData(&d.sourcesToCheck[y].pairCandles.Item)
			if err != nil {
				log.Errorf(common.LiveStrategy, "%v %v %v issue processing kline data: %v", d.sourcesToCheck[y].exchangeName, d.sourcesToCheck[y].asset, d.sourcesToCheck[y].pair, err)
			}
			err = d.funding.AddUSDTrackingData(&d.sourcesToCheck[y].pairCandles)
			if err != nil && !errors.Is(err, funding.ErrUSDTrackingDisabled) {
				log.Errorf(common.LiveStrategy, "%v %v %v issue processing USD tracking data: %v", d.sourcesToCheck[y].exchangeName, d.sourcesToCheck[y].asset, d.sourcesToCheck[y].pair, err)
			}
			setData = true
		}
	}
	if !setData {
		return errNoDataSetForClosingPositions
	}

	return nil
}

// IsRealOrders is a quick check for if the strategy is using real orders
func (d *dataChecker) IsRealOrders() bool {
	return d.realOrders
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
