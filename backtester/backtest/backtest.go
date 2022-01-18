package backtest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var futuresEnabled = false

// New returns a new BackTest instance
func New() *BackTest {
	return &BackTest{
		shutdown:   make(chan struct{}),
		Datas:      &data.HandlerPerCurrency{},
		EventQueue: &eventholder.Holder{},
	}
}

// Reset BackTest values to default
func (bt *BackTest) Reset() {
	bt.EventQueue.Reset()
	bt.Datas.Reset()
	bt.Portfolio.Reset()
	bt.Statistic.Reset()
	bt.Exchange.Reset()
	bt.Funding.Reset()
	bt.exchangeManager = nil
	bt.orderManager = nil
	bt.databaseManager = nil
}

// Run will iterate over loaded data events
// save them and then handle the event based on its type
func (bt *BackTest) Run() error {
	log.Info(log.BackTester, "running backtester against pre-defined data")
dataLoadingIssue:
	for ev := bt.EventQueue.NextEvent(); ; ev = bt.EventQueue.NextEvent() {
		if ev == nil {
			dataHandlerMap := bt.Datas.GetAllData()
			for exchangeName, exchangeMap := range dataHandlerMap {
				for assetItem, assetMap := range exchangeMap {
					var hasProcessedData bool
					for currencyPair, dataHandler := range assetMap {
						d := dataHandler.Next()
						if d == nil {
							if !bt.hasHandledEvent {
								log.Errorf(log.BackTester, "Unable to perform `Next` for %v %v %v", exchangeName, assetItem, currencyPair)
							}
							break dataLoadingIssue
						}
						if bt.Strategy.UsingSimultaneousProcessing() && hasProcessedData {
							continue
						}
						bt.EventQueue.AppendEvent(d)
						hasProcessedData = true
					}
				}
			}
		}
		if ev != nil {
			err := bt.handleEvent(ev)
			if err != nil {
				return err
			}
		}
		if !bt.hasHandledEvent {
			bt.hasHandledEvent = true
		}
	}

	return nil
}

// handleEvent is the main processor of data for the backtester
// after data has been loaded and Run has appended a data event to the queue,
// handle event will process events and add further events to the queue if they
// are required
func (bt *BackTest) handleEvent(ev common.EventHandler) error {
	funds, err := bt.Funding.GetFundingForEvent(ev)
	if err != nil {
		return err
	}

	switch eType := ev.(type) {
	case common.DataEventHandler:
		if bt.Strategy.UsingSimultaneousProcessing() {
			err = bt.processSimultaneousDataEvents()
			if err != nil {
				return err
			}
			bt.Funding.CreateSnapshot(ev.GetTime())
			return nil
		}
		err = bt.processSingleDataEvent(eType, funds.FundReleaser())
		if err != nil {
			return err
		}
		bt.Funding.CreateSnapshot(ev.GetTime())
		return nil
	case signal.Event:
		bt.processSignalEvent(eType, funds.FundReserver())
	case order.Event:
		bt.processOrderEvent(eType, funds.FundReleaser())
	case fill.Event:
		bt.processFillEvent(eType, funds.FundReleaser())
	default:
		return fmt.Errorf("%w %v received, could not process",
			errUnhandledDatatype,
			ev)
	}

	return nil
}

func (bt *BackTest) processSingleDataEvent(ev common.DataEventHandler, funds funding.IFundReleaser) error {
	err := bt.updateStatsForDataEvent(ev, funds)
	if err != nil {
		return err
	}
	d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	s, err := bt.Strategy.OnSignal(d, bt.Funding, bt.Portfolio)
	if err != nil {
		if errors.Is(err, base.ErrTooMuchBadData) {
			// too much bad data is a severe error and backtesting must cease
			return err
		}
		log.Error(log.BackTester, err)
		return nil
	}
	err = bt.Statistic.SetEventForOffset(s)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	bt.EventQueue.AppendEvent(s)

	return nil
}

// processSimultaneousDataEvents determines what signal events are generated and appended
// to the event queue based on whether it is running a multi-currency consideration strategy order not
//
// for multi-currency-consideration it will pass all currency datas to the strategy for it to determine what
// currencies to act upon
//
// for non-multi-currency-consideration strategies, it will simply process every currency individually
// against the strategy and generate signals
func (bt *BackTest) processSimultaneousDataEvents() error {
	var dataEvents []data.Handler
	dataHandlerMap := bt.Datas.GetAllData()
	for _, exchangeMap := range dataHandlerMap {
		for _, assetMap := range exchangeMap {
			for _, dataHandler := range assetMap {
				latestData := dataHandler.Latest()
				funds, err := bt.Funding.GetFundingForEAP(latestData.GetExchange(), latestData.GetAssetType(), latestData.Pair())
				if err != nil {
					return err
				}
				err = bt.updateStatsForDataEvent(latestData, funds.FundReleaser())
				if err != nil && err == statistics.ErrAlreadyProcessed {
					continue
				}
				dataEvents = append(dataEvents, dataHandler)
			}
		}
	}
	signals, err := bt.Strategy.OnSimultaneousSignals(dataEvents, bt.Funding, bt.Portfolio)
	if err != nil {
		if errors.Is(err, base.ErrTooMuchBadData) {
			// too much bad data is a severe error and backtesting must cease
			return err
		}
		log.Errorf(log.BackTester, "OnSimultaneousSignals %v", err)
		return nil
	}
	for i := range signals {
		err = bt.Statistic.SetEventForOffset(signals[i])
		if err != nil {
			log.Errorf(log.BackTester, "SetEventForOffset %v %v %v %v", signals[i].GetExchange(), signals[i].GetAssetType(), signals[i].Pair(), err)
		}
		bt.EventQueue.AppendEvent(signals[i])
	}
	return nil
}

// updateStatsForDataEvent makes various systems aware of price movements from
// data events
func (bt *BackTest) updateStatsForDataEvent(ev common.DataEventHandler, funds funding.IFundReleaser) error {
	// update statistics with the latest price
	err := bt.Statistic.SetupEventForTime(ev)
	if err != nil {
		if err == statistics.ErrAlreadyProcessed {
			return err
		}
		log.Errorf(log.BackTester, "SetupEventForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	// update portfolio manager with the latest price
	err = bt.Portfolio.UpdateHoldings(ev, funds)
	if err != nil {
		log.Errorf(log.BackTester, "UpdateHoldings %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	if ev.GetAssetType().IsFutures() {
		var cr funding.ICollateralReleaser
		cr, err = funds.GetCollateralReleaser()
		if err != nil {
			return err
		}
		err = bt.Portfolio.CalculatePNL(ev)
		if err != nil {
			if errors.Is(err, gctorder.ErrPositionLiquidated) {
				cr.Liquidate()
			} else {
				log.Errorf(log.BackTester, "CalculatePNL %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
			}
		}
	}

	return nil
}

// processSignalEvent receives an event from the strategy for processing under the portfolio
func (bt *BackTest) processSignalEvent(ev signal.Event, funds funding.IFundReserver) {
	cs, err := bt.Exchange.GetCurrencySettings(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Errorf(log.BackTester, "GetCurrencySettings %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		return
	}
	var o *order.Order
	o, err = bt.Portfolio.OnSignal(ev, &cs, funds)
	if err != nil {
		log.Errorf(log.BackTester, "OnSignal %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		return
	}
	err = bt.Statistic.SetEventForOffset(o)
	if err != nil {
		log.Errorf(log.BackTester, "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	bt.EventQueue.AppendEvent(o)
}

func (bt *BackTest) processOrderEvent(ev order.Event, funds funding.IFundReleaser) {
	d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	f, err := bt.Exchange.ExecuteOrder(ev, d, bt.orderManager, funds)
	if err != nil {
		if f == nil {
			log.Errorf(log.BackTester, "fill event should always be returned, please fix, %v", err)
			return
		}
		log.Errorf(log.BackTester, "%v %v %v %v", f.GetExchange(), f.GetAssetType(), f.Pair(), err)
	}
	err = bt.Statistic.SetEventForOffset(f)
	if err != nil {
		log.Errorf(log.BackTester, "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	bt.EventQueue.AppendEvent(f)
}

func (bt *BackTest) processFillEvent(ev fill.Event, funds funding.IFundReleaser) {
	t, err := bt.Portfolio.OnFill(ev, funds)
	if err != nil {
		log.Errorf(log.BackTester, "OnFill %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		return
	}

	err = bt.Statistic.SetEventForOffset(t)
	if err != nil {
		log.Errorf(log.BackTester, "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	var holding *holdings.Holding
	holding, err = bt.Portfolio.ViewHoldingAtTimePeriod(ev)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	if holding == nil {
		log.Error(log.BackTester, "why is holdings nill?")
	} else {
		err = bt.Statistic.AddHoldingsForTime(holding)
		if err != nil {
			log.Errorf(log.BackTester, "AddHoldingsForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		}
	}

	var cp *compliance.Manager
	cp, err = bt.Portfolio.GetComplianceManager(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Errorf(log.BackTester, "GetComplianceManager %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	snap := cp.GetLatestSnapshot()
	err = bt.Statistic.AddComplianceSnapshotForTime(snap, ev)
	if err != nil {
		log.Errorf(log.BackTester, "AddComplianceSnapshotForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
}

// RunLive is a proof of concept function that does not yet support multi currency usage
// It runs by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	log.Info(log.BackTester, "running backtester against live data")
	timeoutTimer := time.NewTimer(time.Minute * 5)
	// a frequent timer so that when a new candle is released by an exchange
	// that it can be processed quickly
	processEventTicker := time.NewTicker(time.Second)
	doneARun := false
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-timeoutTimer.C:
			return errLiveDataTimeout
		case <-processEventTicker.C:
			for e := bt.EventQueue.NextEvent(); ; e = bt.EventQueue.NextEvent() {
				if e == nil {
					// as live only supports singular currency, just get the proper reference manually
					var d data.Handler
					dd := bt.Datas.GetAllData()
					for k1, v1 := range dd {
						for k2, v2 := range v1 {
							for k3 := range v2 {
								d = dd[k1][k2][k3]
							}
						}
					}
					de := d.Next()
					if de == nil {
						break
					}

					bt.EventQueue.AppendEvent(de)
					doneARun = true
					continue
				}
				err := bt.handleEvent(e)
				if err != nil {
					return err
				}
			}
			if doneARun {
				timeoutTimer = time.NewTimer(time.Minute * 5)
			}
		}
	}
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func (bt *BackTest) loadLiveDataLoop(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, dataType int64) {
	startDate := time.Now().Add(-cfg.DataSettings.Interval * 2)
	dates, err := gctkline.CalculateCandleDateRanges(
		startDate,
		startDate.AddDate(1, 0, 0),
		gctkline.Interval(cfg.DataSettings.Interval),
		0)
	if err != nil {
		log.Errorf(log.BackTester, "%v. Please check your GoCryptoTrader configuration", err)
		return
	}
	candles, err := live.LoadData(context.TODO(),
		exch,
		dataType,
		cfg.DataSettings.Interval,
		fPair,
		a)
	if err != nil {
		log.Errorf(log.BackTester, "%v. Please check your GoCryptoTrader configuration", err)
		return
	}
	dates.SetHasDataFromCandles(candles.Candles)
	resp.RangeHolder = dates
	resp.Item = *candles

	loadNewDataTimer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-bt.shutdown:
			return
		case <-loadNewDataTimer.C:
			log.Infof(log.BackTester, "fetching data for %v %v %v %v", exch.GetName(), a, fPair, cfg.DataSettings.Interval)
			loadNewDataTimer.Reset(time.Second * 15)
			err = bt.loadLiveData(resp, cfg, exch, fPair, a, dataType)
			if err != nil {
				log.Error(log.BackTester, err)
				return
			}
		}
	}
}

func (bt *BackTest) loadLiveData(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, dataType int64) error {
	if resp == nil {
		return errNilData
	}
	if cfg == nil {
		return errNilConfig
	}
	if exch == nil {
		return errNilExchange
	}
	candles, err := live.LoadData(context.TODO(),
		exch,
		dataType,
		cfg.DataSettings.Interval,
		fPair,
		a)
	if err != nil {
		return err
	}
	if len(candles.Candles) == 0 {
		return nil
	}
	resp.AppendResults(candles)
	bt.Reports.UpdateItem(&resp.Item)
	log.Info(log.BackTester, "sleeping for 30 seconds before checking for new candle data")
	return nil
}

// Stop shuts down the live data loop
func (bt *BackTest) Stop() {
	close(bt.shutdown)
}
