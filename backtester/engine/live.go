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
		log.Warnf(common.LiveStrategy, "Invalid event timeout '%v', defaulting to '%v'", eventTimeout, defaultEventTimeout)
		eventTimeout = defaultEventTimeout
	}
	if dataCheckInterval <= 0 {
		log.Warnf(common.LiveStrategy, "Invalid data check interval '%v', defaulting to '%v'", dataCheckInterval, defaultDataCheckInterval)
		dataCheckInterval = defaultDataCheckInterval
	}
	bt.LiveDataHandler = &dataChecker{
		verboseDataCheck:  verbose,
		realOrders:        realOrders,
		hasUpdatedFunding: false,
		exchangeManager:   bt.exchangeManager,
		sourcesToCheck:    nil,
		eventTimeout:      eventTimeout,
		dataCheckInterval: dataCheckInterval,
		dataHolder:        bt.DataHolder,
		shutdownErr:       make(chan bool),
		dataUpdated:       make(chan bool),
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
	d.wg.Add(1)
	d.shutdown = make(chan bool)
	d.dataUpdated = make(chan bool)
	d.shutdownErr = make(chan bool)
	go func() {
		err := d.DataFetcher()
		if err != nil {
			stopErr := d.SignalStopFromError(err)
			if stopErr != nil {
				log.Errorln(common.LiveStrategy, stopErr)
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
	close(d.shutdown)
	return nil
}

// SignalStopFromError ceases fetching and processing live data
func (d *dataChecker) SignalStopFromError(err error) error {
	if err == nil {
		return errNilError
	}
	if d == nil {
		return gctcommon.ErrNilPointer
	}
	if !atomic.CompareAndSwapUint32(&d.started, 1, 0) {
		return engine.ErrSubSystemNotStarted
	}
	log.Errorln(common.LiveStrategy, err)
	d.shutdownErr <- true
	return nil
}

// DataFetcher will fetch and append live data
func (d *dataChecker) DataFetcher() error {
	if d == nil {
		return fmt.Errorf("%w dataChecker", gctcommon.ErrNilPointer)
	}
	d.wg.Done()
	if atomic.LoadUint32(&d.started) == 0 {
		return engine.ErrSubSystemNotStarted
	}
	checkTimer := time.NewTimer(0)
	timeoutTimer := time.NewTimer(d.eventTimeout)
	for {
		select {
		case <-d.shutdown:
			return nil
		case <-timeoutTimer.C:
			return fmt.Errorf("%w of %v", ErrLiveDataTimeout, d.eventTimeout)
		case <-checkTimer.C:
			err := d.checkData()
			if err != nil {
				return err
			}
			checkTimer.Reset(d.dataCheckInterval)
			if !timeoutTimer.Stop() {
				<-timeoutTimer.C
			}
			timeoutTimer.Reset(d.eventTimeout)
		}
	}
}

func (d *dataChecker) checkData() error {
	hasDataUpdated, err := d.FetchLatestData()
	if err != nil {
		return err
	}
	if !hasDataUpdated {
		return nil
	}
	d.dataUpdated <- hasDataUpdated
	if d.realOrders {
		go func() {
			err = d.UpdateFunding(false)
			if err != nil {
				log.Errorf(common.LiveStrategy, "Could not update funding: %v", err)
			}
		}()
	}
	return nil
}

// UpdateFunding requests and updates funding levels
func (d *dataChecker) UpdateFunding(force bool) error {
	switch {
	case d == nil:
		return fmt.Errorf("%w datachecker", gctcommon.ErrNilPointer)
	case d.funding == nil:
		return fmt.Errorf("%w datachecker funding manager", gctcommon.ErrNilPointer)
	case force:
		atomic.StoreUint32(&d.updatingFunding, 1)
	case !atomic.CompareAndSwapUint32(&d.updatingFunding, 0, 1):
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

func closedChan() chan bool {
	immediateClosure := make(chan bool)
	close(immediateClosure)
	return immediateClosure
}

// Updated gives other endpoints the ability to listen to
// when data is dataUpdated from live sources
func (d *dataChecker) Updated() chan bool {
	if d == nil {
		return closedChan()
	}
	return d.dataUpdated
}

// HasShutdown indicates when the live data checker
// has been shutdown
func (d *dataChecker) HasShutdown() chan bool {
	if d == nil {
		return closedChan()
	}

	return d.shutdown
}

// HasShutdownFromError indicates when the live data checker
// has been shutdown from encountering an error
func (d *dataChecker) HasShutdownFromError() chan bool {
	if d == nil {
		return closedChan()
	}
	return d.shutdownErr
}

// Reset clears all stored data
func (d *dataChecker) Reset() error {
	if d == nil {
		return gctcommon.ErrNilPointer
	}
	d.m.Lock()
	defer d.m.Unlock()
	d.wg = sync.WaitGroup{}
	d.started = 0
	d.updatingFunding = 0
	d.verboseDataCheck = false
	d.realOrders = false
	d.hasUpdatedFunding = false
	d.exchangeManager = nil
	d.sourcesToCheck = nil
	d.eventTimeout = 0
	d.dataCheckInterval = 0
	d.dataHolder = nil
	d.report = nil
	d.funding = nil

	return nil
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
		return gctkline.ErrInvalidInterval
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
	k := kline.NewDataFromKline()
	k.Item = &gctkline.Item{
		Exchange:       exchName,
		Pair:           dataSource.pair,
		UnderlyingPair: dataSource.underlyingPair,
		Asset:          dataSource.asset,
		Interval:       dataSource.interval,
	}

	err := k.SetLive(true)
	if err != nil {
		return err
	}
	if dataSource.dataRequestRetryTolerance <= 0 {
		log.Warnf(common.LiveStrategy, "Invalid data retry tolerance, setting %v to %v", dataSource.dataRequestRetryTolerance, defaultDataRetryAttempts)
		dataSource.dataRequestRetryTolerance = defaultDataRetryAttempts
	}
	if dataSource.dataRequestRetryWaitTime <= 0 {
		log.Warnf(common.LiveStrategy, "Invalid data request wait time, setting %v to %v", dataSource.dataRequestRetryWaitTime, defaultDataRequestWaitTime)
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
	for i := range results {
		if !results[i] {
			return false, nil
		}
	}
	for i := range d.sourcesToCheck {
		if d.verboseDataCheck {
			log.Infof(common.LiveStrategy, "%v %v %v found new data", d.sourcesToCheck[i].exchangeName, d.sourcesToCheck[i].asset, d.sourcesToCheck[i].pair)
		}
		err = d.sourcesToCheck[i].pairCandles.AppendResults(d.sourcesToCheck[i].candlesToAppend)
		if err != nil {
			return false, err
		}
		d.sourcesToCheck[i].candlesToAppend.Candles = nil
		err = d.dataHolder.SetDataForCurrency(d.sourcesToCheck[i].exchangeName, d.sourcesToCheck[i].asset, d.sourcesToCheck[i].pair, d.sourcesToCheck[i].pairCandles)
		if err != nil {
			return false, err
		}
		err = d.report.SetKlineData(d.sourcesToCheck[i].pairCandles.Item)
		if err != nil {
			return false, err
		}
		err = d.funding.AddUSDTrackingData(d.sourcesToCheck[i].pairCandles)
		if err != nil && !errors.Is(err, funding.ErrUSDTrackingDisabled) {
			return false, err
		}
	}
	if !d.hasUpdatedFunding {
		err = d.UpdateFunding(false)
		if err != nil {
			log.Errorln(common.LiveStrategy, err)
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
			err = d.sourcesToCheck[y].pairCandles.AppendResults(d.sourcesToCheck[y].pairCandles.Item)
			if err != nil {
				log.Errorf(common.LiveStrategy, "%v %v %v issue appending kline data: %v", d.sourcesToCheck[y].exchangeName, d.sourcesToCheck[y].asset, d.sourcesToCheck[y].pair, err)
				continue
			}
			err = d.report.SetKlineData(d.sourcesToCheck[y].pairCandles.Item)
			if err != nil {
				log.Errorf(common.LiveStrategy, "%v %v %v issue processing kline data: %v", d.sourcesToCheck[y].exchangeName, d.sourcesToCheck[y].asset, d.sourcesToCheck[y].pair, err)
				continue
			}
			err = d.funding.AddUSDTrackingData(d.sourcesToCheck[y].pairCandles)
			if err != nil && !errors.Is(err, funding.ErrUSDTrackingDisabled) {
				log.Errorf(common.LiveStrategy, "%v %v %v issue processing USD tracking data: %v", d.sourcesToCheck[y].exchangeName, d.sourcesToCheck[y].asset, d.sourcesToCheck[y].pair, err)
				continue
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
		return false, fmt.Errorf("%w live data source data handler", gctcommon.ErrNilPointer)
	}
	if c.pairCandles == nil {
		return false, fmt.Errorf("%w pair candles", gctcommon.ErrNilPointer)
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
			}
			return false, err
		}
		break
	}
	if candles == nil {
		return false, fmt.Errorf("%w kline Asset", gctcommon.ErrNilPointer)
	}
	if len(candles.Candles) == 0 {
		return false, nil
	}

	unprocessedCandles := make([]gctkline.Candle, 0, len(candles.Candles))
	for i := range candles.Candles {
		if _, ok := c.processedData[candles.Candles[i].Time.UnixNano()]; !ok {
			unprocessedCandles = append(unprocessedCandles, candles.Candles[i])
			c.processedData[candles.Candles[i].Time.UnixNano()] = struct{}{}
		}
	}
	if len(unprocessedCandles) > 0 {
		if c.candlesToAppend == nil {
			c.candlesToAppend = candles
		}
		c.candlesToAppend.Candles = append(c.candlesToAppend.Candles, unprocessedCandles...)
		return true, nil
	}
	return false, nil
}
