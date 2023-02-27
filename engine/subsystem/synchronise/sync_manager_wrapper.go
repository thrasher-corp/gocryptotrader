package synchronise

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// IsRunning safely checks whether the subsystem is running
func (m *Manager) IsRunning() bool {
	return m != nil && atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *Manager) Start() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNil)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return subsystem.ErrAlreadyStarted
	}
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer started.")

	if m.SynchroniseTicker {
		m.tickerJobs = make(chan RESTJob, defaultChannelBuffer)
	}

	if m.SynchroniseOrderbook {
		m.orderbookJobs = make(chan RESTJob, defaultChannelBuffer)
	}

	// Set job channel lanes for differing update speeds per exchange. POC;
	// dangly routines will just block.
	// TODO: Add workers for each individual exchange and lane. In future would
	// like to add priority lanes for requests (order management) which will
	// need to be heavily coupled with the rate limit systems.
	for i := 0; i < m.NumWorkers; i++ {
		if m.SynchroniseTicker {
			m.workerWG.Add(1)
			go m.tickerWorker(context.TODO())
		}
		if m.SynchroniseOrderbook {
			m.workerWG.Add(1)
			go m.orderbookWorker(context.TODO())
		}
		// TODO: Implement trade synchronization.
	}

	err := m.controller()
	if err != nil {
		return err
	}

	go func() {
		m.initSyncWG.Wait()
		if !atomic.CompareAndSwapInt32(&m.initSyncCompleted, 0, 1) {
			return
		}

		log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync completed. Sync took %v [%v sync items].",
			time.Since(m.initSyncStartTime),
			m.createdCounter)

		if m.SynchroniseContinuously {
			return
		}

		log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
		err := m.Stop()
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}()
	m.initSyncWG.Done()
	return nil
}

// Stop shuts down the exchange currency pair syncer
func (m *Manager) Stop() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNil)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNotStarted)
	}
	if m.SynchroniseTicker {
		close(m.tickerJobs)
	}
	if m.SynchroniseOrderbook {
		close(m.orderbookJobs)
	}
	m.workerWG.Wait()
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	return nil
}

// Update notifies the syncManager to change the last updated time for an exchange asset pair
func (m *Manager) Update(exchangeName string, updateProtocol subsystem.ProtocolType, p currency.Pair, a asset.Item, syncType subsystem.SynchronisationType, incomingErr error) error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNil)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNotStarted)
	}
	if exchangeName == "" {
		return errExchangeNameUnset
	}
	if updateProtocol == "" {
		return errProtocolUnset
	}
	if p.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return asset.ErrNotSupported
	}

	// NOTE: Switch for a pre-lock check against config field bools.
	switch syncType {
	case subsystem.Orderbook:
		if !m.SynchroniseOrderbook {
			return nil
		}
	case subsystem.Ticker:
		if !m.SynchroniseTicker {
			return nil
		}
	default:
		return fmt.Errorf("%v %w", syncType, errUnknownSyncType)
	}

	agent := m.getAgent(exchangeName, p, a, syncType, updateProtocol == subsystem.Rest)
	if !agent.Update(updateProtocol, incomingErr) {
		return nil
	}

	if atomic.LoadInt32(&m.initSyncCompleted) == 1 {
		return nil
	}

	m.removedCounter++
	log.Debugf(log.SyncMgr, "%s %s sync complete for %s %s via %s [%d/%d].",
		exchangeName,
		syncType,
		m.formatCurrency(p),
		strings.ToUpper(a.String()),
		updateProtocol,
		m.removedCounter,
		m.createdCounter)
	m.initSyncWG.Done()
	return nil
}

// PrintTickerSummary outputs the ticker results
func (m *Manager) PrintTickerSummary(result *ticker.Price, protocol subsystem.ProtocolType, err error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if err != nil {
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.SyncMgr, "Failed to get %s ticker. Error: %s",
				protocol,
				err)
			return
		}
		log.Errorf(log.SyncMgr, "Failed to get %s ticker. Error: %s",
			protocol,
			err)
		return
	}

	// ignoring error as not all tickers have volume populated and error is not actionable
	_ = stats.Add(result.ExchangeName, result.Pair, result.AssetType, result.Last, result.Volume)

	if result.Pair.Quote.IsFiatCurrency() &&
		!result.Pair.Quote.Equal(m.FiatDisplayCurrency) &&
		!m.FiatDisplayCurrency.IsEmpty() {
		origCurrency := result.Pair.Quote.Upper()
		log.Infof(log.SyncMgr, "%s %s %s %s TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
			result.ExchangeName,
			protocol,
			m.formatCurrency(result.Pair),
			strings.ToUpper(result.AssetType.String()),
			printConvertCurrencyFormat(result.Last, origCurrency, m.FiatDisplayCurrency),
			printConvertCurrencyFormat(result.Ask, origCurrency, m.FiatDisplayCurrency),
			printConvertCurrencyFormat(result.Bid, origCurrency, m.FiatDisplayCurrency),
			printConvertCurrencyFormat(result.High, origCurrency, m.FiatDisplayCurrency),
			printConvertCurrencyFormat(result.Low, origCurrency, m.FiatDisplayCurrency),
			result.Volume)
	} else {
		if result.Pair.Quote.IsFiatCurrency() &&
			result.Pair.Quote.Equal(m.FiatDisplayCurrency) &&
			!m.FiatDisplayCurrency.IsEmpty() {
			log.Infof(log.SyncMgr, "%s %s %s %s TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
				result.ExchangeName,
				protocol,
				m.formatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				printCurrencyFormat(result.Last, m.FiatDisplayCurrency),
				printCurrencyFormat(result.Ask, m.FiatDisplayCurrency),
				printCurrencyFormat(result.Bid, m.FiatDisplayCurrency),
				printCurrencyFormat(result.High, m.FiatDisplayCurrency),
				printCurrencyFormat(result.Low, m.FiatDisplayCurrency),
				result.Volume)
		} else {
			log.Infof(log.SyncMgr, "%s %s %s %s TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f",
				result.ExchangeName,
				protocol,
				m.formatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				result.Last,
				result.Ask,
				result.Bid,
				result.High,
				result.Low,
				result.Volume)
		}
	}
}

// PrintOrderbookSummary outputs orderbook results
func (m *Manager) PrintOrderbookSummary(result *orderbook.Base, protocol subsystem.ProtocolType, err error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if err != nil {
		if result == nil {
			log.Errorf(log.OrderBook, "Failed to get %s orderbook. Error: %s",
				protocol,
				err)
			return
		}
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s",
				protocol,
				result.Exchange,
				result.Pair,
				result.Asset,
				err)
			return
		}
		log.Errorf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s",
			protocol,
			result.Exchange,
			result.Pair,
			result.Asset,
			err)
		return
	}

	bidsAmount, bidsValue := result.TotalBidsAmount()
	asksAmount, asksValue := result.TotalAsksAmount()

	var bidValueResult, askValueResult string
	switch {
	case result.Pair.Quote.IsFiatCurrency() && !result.Pair.Quote.Equal(m.FiatDisplayCurrency) && !m.FiatDisplayCurrency.IsEmpty():
		origCurrency := result.Pair.Quote.Upper()
		if bidsValue > 0 {
			bidValueResult = printConvertCurrencyFormat(bidsValue, origCurrency, m.FiatDisplayCurrency)
		}
		if asksValue > 0 {
			askValueResult = printConvertCurrencyFormat(asksValue, origCurrency, m.FiatDisplayCurrency)
		}
	case result.Pair.Quote.IsFiatCurrency() && result.Pair.Quote.Equal(m.FiatDisplayCurrency) && !m.FiatDisplayCurrency.IsEmpty():
		bidValueResult = printCurrencyFormat(bidsValue, m.FiatDisplayCurrency)
		askValueResult = printCurrencyFormat(asksValue, m.FiatDisplayCurrency)
	default:
		bidValueResult = strconv.FormatFloat(bidsValue, 'f', -1, 64)
		askValueResult = strconv.FormatFloat(asksValue, 'f', -1, 64)
	}

	log.Infof(log.SyncMgr, book,
		result.Exchange,
		protocol,
		m.formatCurrency(result.Pair),
		strings.ToUpper(result.Asset.String()),
		len(result.Bids),
		bidsAmount,
		result.Pair.Base,
		bidValueResult,
		len(result.Asks),
		asksAmount,
		result.Pair.Base,
		askValueResult,
	)
}
