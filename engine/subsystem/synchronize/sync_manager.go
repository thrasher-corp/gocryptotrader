package synchronize

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// DefaultWorkers limits the number of sync workers
	DefaultWorkers = 15
	// DefaultTimeoutREST the default time to switch from REST to websocket
	// protocols without a response.
	DefaultTimeoutREST = time.Second * 15
	// DefaultTimeoutWebsocket the default time to switch from websocket to REST
	// protocols without a response.
	DefaultTimeoutWebsocket = time.Minute

	defaultChannelBuffer  = 10000
	errNoSyncItemsEnabled = errors.New("no sync items enabled")
	errUnknownSyncType    = errors.New("unknown sync type")
	errSyncerNotFound     = errors.New("sync agent not found")
)

const (
	WebsocketUpdate = "WEBSOCKET"
	RESTUpdate      = "REST"
)

// NewManager returns a new sychronization manager
func NewManager(c *ManagerConfig) (*Manager, error) {
	if c == nil {
		return nil, fmt.Errorf("%T %w", c, common.ErrNilPointer)
	}

	if !c.SynchronizeOrderbook && !c.SynchronizeTicker && !c.SynchronizeTrades {
		return nil, errNoSyncItemsEnabled
	}

	if c.ExchangeManager == nil {
		return nil, subsystem.ErrNilExchangeManager
	}

	if c.RemoteConfig == nil {
		return nil, fmt.Errorf("remote control: %w", subsystem.ErrNilConfig)
	}

	if c.NumWorkers <= 0 {
		c.NumWorkers = DefaultWorkers
	}

	if c.TimeoutREST <= 0 {
		c.TimeoutREST = DefaultTimeoutREST
	}

	if c.TimeoutWebsocket <= time.Duration(0) {
		c.TimeoutWebsocket = DefaultTimeoutWebsocket
	}

	if c.FiatDisplayCurrency.IsEmpty() {
		return nil, fmt.Errorf("FiatDisplayCurrency %w", currency.ErrCurrencyCodeEmpty)
	}

	if !c.FiatDisplayCurrency.IsFiatCurrency() {
		return nil, fmt.Errorf("%s %w", c.FiatDisplayCurrency, currency.ErrFiatDisplayCurrencyIsNotFiat)
	}

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout REST: %v"+
			" timeout Websocket: %v",
		c.SynchronizeContinuously, c.SynchronizeTicker, c.SynchronizeOrderbook,
		c.SynchronizeTrades, c.NumWorkers, c.Verbose, c.TimeoutREST,
		c.TimeoutWebsocket)

	manager := &Manager{
		ManagerConfig:            *c,
		tickerBatchLastRequested: make(map[string]map[asset.Item]time.Time),
		orderbookJobs:            make(chan RESTJob, defaultChannelBuffer),
		tickerJobs:               make(chan RESTJob, defaultChannelBuffer),
		tradeJobs:                make(chan RESTJob, defaultChannelBuffer),
	}
	manager.initSyncWG.Add(1)
	return manager, nil
}

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
	exchanges, err := m.ExchangeManager.GetExchanges()
	if err != nil {
		return err
	}
	for x := range exchanges {
		exchangeName := exchanges[x].GetName()
		supportsWebsocket := exchanges[x].SupportsWebsocket()
		supportsREST := exchanges[x].SupportsREST()

		if !supportsREST && !supportsWebsocket {
			// TODO: Should error?
			log.Warnf(log.SyncMgr, "Loaded exchange %s does not support REST or Websocket.", exchangeName)
			continue
		}

		usingWebsocket := m.WebsocketRoutineManagerEnabled && supportsWebsocket && exchanges[x].IsWebsocketEnabled()
		usingREST := supportsREST && !usingWebsocket

		assetTypes := exchanges[x].GetAssetTypes(false)
		for y := range assetTypes {
			if exchanges[x].GetBase().CurrencyPairs.IsAssetEnabled(assetTypes[y]) != nil {
				log.Warnf(log.SyncMgr,
					"%s asset type %s is disabled, fetching enabled pairs is paused",
					exchangeName,
					assetTypes[y])
				continue
			}

			wsAssetSupported := exchanges[x].IsAssetWebsocketSupported(assetTypes[y])
			if !wsAssetSupported {
				log.Warnf(log.SyncMgr,
					"%s asset type %s websocket functionality is unsupported, REST fetching only.",
					exchangeName,
					assetTypes[y])
			}
			enabledPairs, err := exchanges[x].GetEnabledPairs(assetTypes[y])
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s failed to get enabled pairs. Err: %s",
					exchangeName,
					err)
				continue
			}
			for i := range enabledPairs {
				_ = m.getAgent(exchangeName,
					enabledPairs[i],
					assetTypes[y],
					usingREST || !wsAssetSupported,
					usingWebsocket && wsAssetSupported)
			}
		}
	}

	if atomic.CompareAndSwapInt32(&m.initSyncStarted, 0, 1) {
		log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync started. %d items to process.",
			m.createdCounter)
		m.initSyncStartTime = time.Now()
	}

	go func() {
		m.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&m.initSyncCompleted, 0, 1) {
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial completed. Sync took %v [%v sync items].",
				time.Since(m.initSyncStartTime),
				m.createdCounter)

			if !m.SynchronizeContinuously {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
				err := m.Stop()
				if err != nil {
					log.Error(log.SyncMgr, err)
				}
				return
			}
		}
	}()

	if atomic.LoadInt32(&m.initSyncCompleted) == 1 && !m.SynchronizeContinuously {
		return nil
	}

	go m.controller()

	for i := 0; i < m.NumWorkers; i++ {
		go m.orderbookWorker(context.TODO())
		go m.tickerWorker(context.TODO())
		go m.tradeWorker(context.TODO())
	}
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
	m.initSyncWG.Add(1)
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	return nil
}

// Update notifies the syncManager to change the last updated time for an exchange asset pair
func (m *Manager) Update(exchangeName, protocol string, p currency.Pair, a asset.Item, item int, incomingErr error) error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNil)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNotStarted)
	}

	if atomic.LoadInt32(&m.initSyncStarted) != 1 {
		return nil
	}

	switch SyncType(item) {
	case Orderbook:
		if !m.SynchronizeOrderbook {
			return nil
		}
	case Ticker:
		if !m.SynchronizeTicker {
			return nil
		}
	case Trade:
		if !m.SynchronizeTrades {
			return nil
		}
	default:
		return fmt.Errorf("%v %w", item, errUnknownSyncType)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.currencyPairs[exchangeName][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return fmt.Errorf("%v %v %v %w", exchangeName, a, p, errSyncerNotFound)
	}

	var name string
	var origHadData bool
	switch SyncType(item) {
	case Ticker:
		name = "ticker"
		origHadData = agent.Ticker.Update(name, "", exchangeName, p, a, incomingErr)
	case Orderbook:
		name = "orderbook"
		origHadData = agent.Orderbook.Update(name, "", exchangeName, p, a, incomingErr)
	case Trade:
		name = "trade"
		origHadData = agent.Trade.Update(name, "", exchangeName, p, a, incomingErr)
	}

	if atomic.LoadInt32(&m.initSyncCompleted) == 1 && origHadData {
		return nil
	}
	m.removedCounter++
	log.Debugf(log.SyncMgr, "%s %s sync complete %v [%d/%d].",
		exchangeName,
		name,
		m.FormatCurrency(p).String(),
		m.removedCounter,
		m.createdCounter)
	m.initSyncWG.Done()
	return nil
}

// controller checks all enabled assets on all enabled exchanges for correct
// synchronization. If an assets needs updating via REST it will push the work
// to worker routines.
func (m *Manager) controller() {
	defer log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer worker shutting down.")
	timer := time.NewTimer(0) // Fire immediately for initial synchronization
	for range timer.C {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		wait := m.GetSmallestTimeout()
		exchanges, err := m.ExchangeManager.GetExchanges()
		if err != nil {
			log.Errorf(log.SyncMgr, "Sync manager cannot get exchanges: %v", err)
		}
		for x := range exchanges {
			exchangeName := exchanges[x].GetName()
			supportsREST := exchanges[x].SupportsREST()
			usingWebsocket := exchanges[x].SupportsWebsocket() &&
				exchanges[x].IsWebsocketEnabled()
			if usingWebsocket {
				ws, err := exchanges[x].GetWebsocket()
				if err != nil {
					log.Errorf(log.SyncMgr, "%s unable to get websocket pointer. Err: %s", exchangeName, err)
				}
				usingWebsocket = ws.IsConnected() || ws.IsConnecting()
			}
			usingREST := !usingWebsocket

			var switchedToRest bool
			assetTypes := exchanges[x].GetAssetTypes(true)
			for y := range assetTypes {
				enabledPairs, err := exchanges[x].GetEnabledPairs(assetTypes[y])
				if err != nil {
					log.Errorf(log.SyncMgr, "%s failed to get enabled pairs. Err: %s",
						exchangeName,
						err)
					continue
				}
				wsAssetSupported := exchanges[x].IsAssetWebsocketSupported(assetTypes[y])
				for i := range enabledPairs {
					if atomic.LoadInt32(&m.started) == 0 {
						return
					}
					untilTimeout := m.CheckSyncItems(exchanges[x],
						enabledPairs[i],
						assetTypes[y],
						(usingREST || !wsAssetSupported),
						(usingWebsocket && wsAssetSupported),
						supportsREST,
						switchedToRest)
					if untilTimeout < wait {
						wait = untilTimeout
					}
				}
			}
		}
		timer.Reset(wait)
	}
}

// GetSmallestTimeout returns the smallest configured timeout for all supported
// synchronization protocols for controller sync.
func (m *Manager) GetSmallestTimeout() time.Duration {
	if m.TimeoutREST < m.TimeoutWebsocket {
		return m.TimeoutREST
	}
	return m.TimeoutWebsocket
}

// CheckSyncItems checks agent against it's current last update time on all
func (m *Manager) CheckSyncItems(exch exchange.IBotExchange, pair currency.Pair, a asset.Item, usingREST, usingWS, supportsREST, switchedToRest bool) time.Duration {
	agent := m.getAgent(exch.GetName(), pair, a, usingREST, usingWS)

	untilUpdate := m.GetSmallestTimeout()
	until, update := agent.Orderbook.NeedsUpdate(m.SynchronizeOrderbook, m.TimeoutREST, m.TimeoutWebsocket)
	if update {
		err := agent.Orderbook.SetProcessing(exch.GetName(), "Orderbook", agent.Pair, agent.AssetType, m.TimeoutWebsocket, true)
		if err != nil {
			panic(err)
		}
		m.sendJob(exch, agent.Pair, agent.AssetType, Orderbook)
	} else if until > 0 && until < untilUpdate {
		untilUpdate = until
	}

	until, update = agent.Ticker.NeedsUpdate(m.SynchronizeTicker, m.TimeoutREST, m.TimeoutWebsocket)
	if update {
		err := agent.Ticker.SetProcessing(exch.GetName(), "Ticker", agent.Pair, agent.AssetType, m.TimeoutWebsocket, true)
		if err != nil {
			panic(err)
		}
		m.sendJob(exch, agent.Pair, agent.AssetType, Ticker)
	} else if until > 0 && until < untilUpdate {
		untilUpdate = until
	}

	until, update = agent.Trade.NeedsUpdate(m.SynchronizeTrades, m.TimeoutREST, m.TimeoutWebsocket)
	if update {
		err := agent.Trade.SetProcessing(exch.GetName(), "Trade", agent.Pair, agent.AssetType, m.TimeoutWebsocket, true)
		if err != nil {
			panic(err)
		}
		m.sendJob(exch, agent.Pair, agent.AssetType, Trade)
	} else if until > 0 && until < untilUpdate {
		untilUpdate = until
	}

	return untilUpdate
}

// sendJob sets agent base as processing for that ticker item then sends the
// REST sync job to the jobs channel for processing.
func (m *Manager) sendJob(exch exchange.IBotExchange, pair currency.Pair, a asset.Item, item SyncType) {
	// NOTE: This is blocking, if there are no receivers and the buffer size is
	// full then this will hang the controller.
	switch item {
	case Orderbook:
		m.orderbookJobs <- RESTJob{exch: exch, Pair: pair, Asset: a, Item: item}
	case Ticker:
		m.tickerJobs <- RESTJob{exch: exch, Pair: pair, Asset: a, Item: item}
	case Trade:
		m.tradeJobs <- RESTJob{exch: exch, Pair: pair, Asset: a, Item: item}
	}
}

// getAgent returns an agent and will generate a new agent if not found.
func (m *Manager) getAgent(exch string, pair currency.Pair, a asset.Item, usingRest, usingWS bool) *Agent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currencyPairs == nil {
		m.currencyPairs = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Agent)
	}

	m1, ok := m.currencyPairs[exch]
	if !ok {
		m1 = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*Agent)
		m.currencyPairs[exch] = m1
	}

	m2, ok := m1[pair.Base.Item]
	if !ok {
		m2 = make(map[*currency.Item]map[asset.Item]*Agent)
		m1[pair.Base.Item] = m2
	}

	m3, ok := m2[pair.Quote.Item]
	if !ok {
		m3 = make(map[asset.Item]*Agent)
		m2[pair.Quote.Item] = m3
	}

	agent := m3[a]
	if agent != nil {
		return agent
	}

	agent = &Agent{Exchange: exch, Pair: pair, AssetType: a}
	if m.SynchronizeTicker {
		agent.Ticker = m.deployBase("ticker", agent, usingWS, usingRest)
	}
	if m.SynchronizeOrderbook {
		agent.Orderbook = m.deployBase("orderbook", agent, usingWS, usingRest)
	}
	if m.SynchronizeTrades {
		agent.Trade = m.deployBase("trade", agent, usingWS, usingRest)
	}

	m3[a] = agent
	return agent
}

// deployBase deploys a instance of a base struct for each individiual
// synchronization item. If verbose it will display the added item. If state
// is in initial sync it will increment counter and add to the waitgroup.
func (m *Manager) deployBase(service string, agent *Agent, usingWS, usingREST bool) Base {
	if m.Verbose {
		log.Debugf(log.SyncMgr,
			"%s: Added trade sync item %s: using websocket: %v using REST: %v",
			agent.Exchange, m.FormatCurrency(agent.Pair), usingWS, usingREST)
	}
	if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
		m.initSyncWG.Add(1)
		m.createdCounter++
	}
	return Base{IsUsingREST: usingREST, IsUsingWebsocket: usingWS}
}

func (m *Manager) orderbookWorker(ctx context.Context) {
	for j := range m.orderbookJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		var result *orderbook.Base
		result, err = j.exch.UpdateOrderbook(ctx, j.Pair, j.Asset)
		m.PrintOrderbookSummary(result, RESTUpdate, err)
		if err == nil && m.RemoteConfig.WebsocketRPC.Enabled {
			relayWebsocketEvent(result, "orderbook_update", j.Asset.String(), exchName)
		}
		err = m.Update(exchName, RESTUpdate, j.Pair, j.Asset, int(j.Item), err)
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}
}

func (m *Manager) tickerWorker(ctx context.Context) {
	for j := range m.orderbookJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		var result *ticker.Price
		if j.exch.SupportsRESTTickerBatchUpdates() {
			m.batchMtx.Lock()
			batchLastDone := m.tickerBatchLastRequested[exchName][j.Asset]
			if batchLastDone.IsZero() || time.Since(batchLastDone) > m.TimeoutREST {
				if m.Verbose {
					log.Debugf(log.SyncMgr, "Initialising %s REST ticker batching", exchName)
				}
				_ = j.exch.UpdateTickers(ctx, j.Asset)
				assetMap, ok := m.tickerBatchLastRequested[exchName]
				if !ok {
					assetMap = make(map[asset.Item]time.Time)
					m.tickerBatchLastRequested[exchName] = assetMap
				}
				assetMap[j.Asset] = time.Now()
			}
			m.batchMtx.Unlock()
			if m.Verbose {
				log.Debugf(log.SyncMgr, "%s Using recent batching cache", exchName)
			}
			result, err = j.exch.FetchTicker(ctx, j.Pair, j.Asset)
		} else {
			result, err = j.exch.UpdateTicker(ctx, j.Pair, j.Asset)
		}
		m.PrintTickerSummary(result, RESTUpdate, err)
		if err == nil && m.RemoteConfig.WebsocketRPC.Enabled {
			relayWebsocketEvent(result, "ticker_update", j.Asset.String(), exchName)
		}
		err = m.Update(exchName, RESTUpdate, j.Pair, j.Asset, int(j.Item), err)
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}
}

func (m *Manager) tradeWorker(ctx context.Context) {
	for j := range m.tradeJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		// TODO: Implement trade synchronization.
		err = m.Update(exchName, RESTUpdate, j.Pair, j.Asset, int(j.Item), err)
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}
}

func printCurrencyFormat(price float64, displayCurrency currency.Code) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.SyncMgr, "Failed to get display symbol: %s", err)
	}
	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origPrice float64, origCurrency, displayCurrency currency.Code) string {
	var conv float64
	if origPrice > 0 {
		var err error
		conv, err = currency.ConvertFiat(origPrice, origCurrency, displayCurrency)
		if err != nil {
			log.Errorf(log.SyncMgr, "Failed to convert currency: %s", err)
		}
	}

	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.SyncMgr, "Failed to get display symbol: %s", err)
	}

	origSymbol, err := currency.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Errorf(log.SyncMgr, "Failed to get original currency symbol for %s: %s",
			origCurrency,
			err)
	}

	return fmt.Sprintf("%s%.2f %s (%s%.2f %s)",
		displaySymbol,
		conv,
		displayCurrency,
		origSymbol,
		origPrice,
		origCurrency,
	)
}

// PrintTickerSummary outputs the ticker results
func (m *Manager) PrintTickerSummary(result *ticker.Price, protocol string, err error) {
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
			m.FormatCurrency(result.Pair),
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
				m.FormatCurrency(result.Pair),
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
				m.FormatCurrency(result.Pair),
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

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *Manager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return p
	}
	return p.Format(m.PairFormatDisplay)
}

const (
	book = "%s %s %s %s ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s"
)

// PrintOrderbookSummary outputs orderbook results
func (m *Manager) PrintOrderbookSummary(result *orderbook.Base, protocol string, err error) {
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
		m.FormatCurrency(result.Pair),
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

// WaitForInitialSync allows for a routine to wait for an initial sync to be
// completed without exposing the underlying type. This needs to be called in a
// separate routine.
func (m *Manager) WaitForInitialSync() error {
	if m == nil {
		return fmt.Errorf("sync manager %w", subsystem.ErrNil)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("sync manager %w", subsystem.ErrNotStarted)
	}
	m.initSyncWG.Wait()
	return nil
}

// NeedsUpdate determines if the underlying agent sync base is ready for an
// update via REST.
func (b *Base) NeedsUpdate(configAllowed bool, timeOutRest, timeOutWS time.Duration) (time.Duration, bool) {
	if !configAllowed {
		return 0, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsProcessing {
		return 0, false
	}
	if b.IsUsingWebsocket {
		added := b.LastUpdated.Add(timeOutWS)
		return time.Until(added), !b.LastUpdated.IsZero() && time.Since(b.LastUpdated) >= timeOutWS
	}
	added := b.LastUpdated.Add(timeOutRest)
	return time.Until(added), b.LastUpdated.IsZero() || time.Since(b.LastUpdated) >= timeOutRest
}

var errProcessingConflict = errors.New("processing conflict")

// SetProcessing sets processing for the specific sync agent
func (b *Base) SetProcessing(exch, service string, pair currency.Pair, a asset.Item, wsTimeout time.Duration, processing bool) error {
	b.mu.Lock()
	if b.IsProcessing && processing {
		b.mu.Unlock()
		return errProcessingConflict
	}
	b.IsProcessing = processing

	var switched bool
	if b.IsUsingWebsocket {
		b.IsUsingWebsocket = true
		b.IsUsingREST = true
		switched = true
	}
	b.mu.Unlock()

	if switched {
		log.Warnf(log.SyncMgr, "%s %s %s: No %s update after %s, switching from WEBSOCKET to REST",
			exch,
			pair,
			strings.ToUpper(a.String()),
			service,
			wsTimeout)
	}

	return nil
}

func (b *Base) Update(service, protocol, exch string, pair currency.Pair, a asset.Item, incomingErr error) bool {
	b.mu.Lock()
	origHadData := b.HaveData
	b.LastUpdated = time.Now()
	if incomingErr != nil {
		b.NumErrors++
	}
	b.HaveData = true
	b.IsProcessing = false

	var switched bool
	if protocol == WebsocketUpdate {
		if b.IsUsingREST {
			switched = true
			b.IsUsingREST = false
			b.IsUsingWebsocket = true
		}
	}
	b.mu.Unlock()

	if switched {
		log.Warnf(log.SyncMgr, "%s %s %s: %s update received, switching from REST to WEBSOCKET",
			exch,
			pair,
			strings.ToUpper(a.String()),
			service)
	}

	return origHadData
}

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	// evt := WebsocketEvent{
	// 	Data:      result,
	// 	Event:     event,
	// 	AssetType: assetType,
	// 	Exchange:  exchangeName,
	// }
	// err := BroadcastWebsocketMessage(evt)
	// if !errors.Is(err, ErrWebsocketServiceNotRunning) {
	// 	log.Errorf(log.APIServerMgr, "Failed to broadcast websocket event %v. Error: %s",
	// 		event, err)
	// }

	// TODO: Package tech
}
