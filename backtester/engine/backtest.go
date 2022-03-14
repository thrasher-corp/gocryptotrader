package engine

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

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
	log.Info(common.SubLoggers[common.Backtester], "running backtester against pre-defined data")
dataLoadingIssue:
	for ev := bt.EventQueue.NextEvent(); ; ev = bt.EventQueue.NextEvent() {
		if ev == nil {
			dataHandlerMap := bt.Datas.GetAllData()
			var hasProcessedData bool
			for exchangeName, exchangeMap := range dataHandlerMap {
				for assetItem, assetMap := range exchangeMap {
					for currencyPair, dataHandler := range assetMap {
						d := dataHandler.Next()
						if d == nil {
							if !bt.hasHandledEvent {
								log.Errorf(common.SubLoggers[common.Backtester], "Unable to perform `Next` for %v %v %v", exchangeName, assetItem, currencyPair)
							}
							break dataLoadingIssue
						}
						if bt.Strategy.UsingSimultaneousProcessing() && hasProcessedData {
							// only append one event, as simultaneous processing
							// will retrieve all relevant events to process under
							// processSimultaneousDataEvents()
							continue
						}
						bt.EventQueue.AppendEvent(d)
						hasProcessedData = true
					}
				}
			}
		} else {
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
	if ev == nil {
		return fmt.Errorf("cannot handle event %w", errNilData)
	}
	funds, err := bt.Funding.GetFundingForEvent(ev)
	if err != nil {
		return err
	}

	if ev.GetAssetType().IsFutures() {
		// hardcoded fix
		err = bt.Funding.UpdateCollateral(ev)
		if err != nil {
			return err
		}
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
		return fmt.Errorf("handleEvent %w %T received, could not process",
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
		log.Errorf(common.SubLoggers[common.Backtester], "OnSignal %v", err)
		return nil
	}
	err = bt.Statistic.SetEventForOffset(s)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "SetEventForOffset %v", err)
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
				funds, err := bt.Funding.GetFundingForEvent(latestData)
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
		log.Errorf(common.SubLoggers[common.Backtester], "OnSimultaneousSignals %v", err)
		return nil
	}
	for i := range signals {
		err = bt.Statistic.SetEventForOffset(signals[i])
		if err != nil {
			log.Errorf(common.SubLoggers[common.Backtester], "SetEventForOffset %v %v %v %v", signals[i].GetExchange(), signals[i].GetAssetType(), signals[i].Pair(), err)
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
		log.Errorf(common.SubLoggers[common.Backtester], "SetupEventForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	// update portfolio manager with the latest price
	err = bt.Portfolio.UpdateHoldings(ev, funds)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "UpdateHoldings %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	if ev.GetAssetType().IsFutures() {
		var cr funding.ICollateralReleaser
		cr, err = funds.GetCollateralReleaser()
		if err != nil {
			return err
		}

		err = bt.Portfolio.UpdatePNL(ev, ev.GetClosePrice())
		if err != nil {
			if errors.Is(err, gctorder.ErrPositionLiquidated) {
				cr.Liquidate()
			} else {
				log.Errorf(common.SubLoggers[common.Backtester], "UpdatePNL %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
				return nil
			}
		}
		var pnl *portfolio.PNLSummary
		pnl, err = bt.Portfolio.GetLatestPNLForEvent(ev)
		if err != nil {
			return err
		}
		return bt.Statistic.AddPNLForTime(pnl)
	}

	return nil
}

// processSignalEvent receives an event from the strategy for processing under the portfolio
func (bt *BackTest) processSignalEvent(ev signal.Event, funds funding.IFundReserver) {
	cs, err := bt.Exchange.GetCurrencySettings(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "GetCurrencySettings %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		return
	}
	var o *order.Order
	o, err = bt.Portfolio.OnSignal(ev, &cs, funds)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "OnSignal %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		return
	}
	err = bt.Statistic.SetEventForOffset(o)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	bt.EventQueue.AppendEvent(o)
}

func (bt *BackTest) processOrderEvent(ev order.Event, funds funding.IFundReleaser) {
	d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	f, err := bt.Exchange.ExecuteOrder(ev, d, bt.orderManager, funds)
	if err != nil {
		if f == nil {
			log.Errorf(common.SubLoggers[common.Backtester], "ExecuteOrder fill event should always be returned, please fix, %v", err)
			return
		}
		if !errors.Is(err, exchange.ErrDoNothing) {
			log.Errorf(common.SubLoggers[common.Backtester], "ExecuteOrder %v %v %v %v", f.GetExchange(), f.GetAssetType(), f.Pair(), err)
		}
	}
	err = bt.Statistic.SetEventForOffset(f)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	bt.EventQueue.AppendEvent(f)
}

func (bt *BackTest) processFillEvent(ev fill.Event, funds funding.IFundReleaser) {
	t, err := bt.Portfolio.OnFill(ev, funds)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "OnFill %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		return
	}

	err = bt.Statistic.SetEventForOffset(t)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	var holding *holdings.Holding
	holding, err = bt.Portfolio.ViewHoldingAtTimePeriod(ev)
	if err != nil {
		log.Error(common.SubLoggers[common.Backtester], err)
	}
	if holding == nil {
		log.Error(common.SubLoggers[common.Backtester], "ViewHoldingAtTimePeriod why is holdings nil?")
	} else {
		err = bt.Statistic.AddHoldingsForTime(holding)
		if err != nil {
			log.Errorf(common.SubLoggers[common.Backtester], "AddHoldingsForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		}
	}

	var cp *compliance.Manager
	cp, err = bt.Portfolio.GetComplianceManager(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "GetComplianceManager %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	snap := cp.GetLatestSnapshot()
	err = bt.Statistic.AddComplianceSnapshotForTime(snap, ev)
	if err != nil {
		log.Errorf(common.SubLoggers[common.Backtester], "AddComplianceSnapshotForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	fde := ev.GetFillDependentEvent()
	if !fde.IsNil() {
		// some events can only be triggered on a successful fill event
		fde.SetOffset(ev.GetOffset())
		err = bt.Statistic.SetEventForOffset(fde)
		if err != nil {
			log.Errorf(common.SubLoggers[common.Backtester], "SetEventForOffset %v %v %v %v", fde.GetExchange(), fde.GetAssetType(), fde.Pair(), err)
		}
		bt.EventQueue.AppendEvent(fde)
	}
	if ev.GetAssetType().IsFutures() {
		if ev.GetOrder() != nil {
			pnl, err := bt.Portfolio.TrackFuturesOrder(ev, funds)
			if err != nil && !errors.Is(err, gctorder.ErrSubmissionIsNil) {
				log.Errorf(common.SubLoggers[common.Backtester], "TrackFuturesOrder %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
				return
			}
			err = bt.Statistic.AddPNLForTime(pnl)
			if err != nil {
				log.Errorf(common.SubLoggers[common.Backtester], "AddHoldingsForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
			}
		}
		err = bt.Funding.UpdateCollateral(ev)
		if err != nil {
			log.Errorf(common.SubLoggers[common.Backtester], "UpdateCollateral %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
			return
		}
	}
}

// Stop shuts down the live data loop
func (bt *BackTest) Stop() {
	close(bt.shutdown)
}
