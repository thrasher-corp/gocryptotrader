package engine

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// const holds the sync item types
const (
	SyncItemTicker = iota
	SyncItemOrderbook
	SyncItemTrade
	SyncManagerName = "exchange_syncer"
)

var (
	// DefaultSyncerWorkers limits the number of sync workers
	DefaultSyncerWorkers = 15
	// DefaultSyncerTimeoutREST the default time to switch from REST to websocket protocols without a response
	DefaultSyncerTimeoutREST = time.Second * 15
	// DefaultSyncerTimeoutWebsocket the default time to switch from websocket to REST protocols without a response
	DefaultSyncerTimeoutWebsocket = time.Minute

	defaultChannelBuffer  = 10000
	errNoSyncItemsEnabled = errors.New("no sync items enabled")
	errUnknownSyncItem    = errors.New("unknown sync item")
	errSyncerNotFound     = errors.New("sync agent not found")
)

// setupSyncManager starts a new CurrencyPairSyncer
func setupSyncManager(c *SyncManagerConfig, exchangeManager subsystem.ExchangeManager, remoteConfig *config.RemoteControlConfig, websocketRoutineManagerEnabled bool) (*syncManager, error) {
	if c == nil {
		return nil, fmt.Errorf("%T %w", c, common.ErrNilPointer)
	}

	if !c.SynchronizeOrderbook && !c.SynchronizeTicker && !c.SynchronizeTrades {
		return nil, errNoSyncItemsEnabled
	}
	if exchangeManager == nil {
		return nil, subsystem.ErrNilExchangeManager
	}
	if remoteConfig == nil {
		return nil, subsystem.ErrNilConfig
	}

	if c.NumWorkers <= 0 {
		c.NumWorkers = DefaultSyncerWorkers
	}

	if c.TimeoutREST <= time.Duration(0) {
		c.TimeoutREST = DefaultSyncerTimeoutREST
	}

	if c.TimeoutWebsocket <= time.Duration(0) {
		c.TimeoutWebsocket = DefaultSyncerTimeoutWebsocket
	}

	if c.FiatDisplayCurrency.IsEmpty() {
		return nil, fmt.Errorf("FiatDisplayCurrency %w", currency.ErrCurrencyCodeEmpty)
	}

	if !c.FiatDisplayCurrency.IsFiatCurrency() {
		return nil, fmt.Errorf("%s %w", c.FiatDisplayCurrency, currency.ErrFiatDisplayCurrencyIsNotFiat)
	}

	if c.PairFormatDisplay == nil {
		return nil, fmt.Errorf("%T %w", c.PairFormatDisplay, common.ErrNilPointer)
	}

	s := &syncManager{
		config:                         *c,
		remoteConfig:                   remoteConfig,
		exchangeManager:                exchangeManager,
		websocketRoutineManagerEnabled: websocketRoutineManagerEnabled,
		fiatDisplayCurrency:            c.FiatDisplayCurrency,
		format:                         *c.PairFormatDisplay,
		tickerBatchLastRequested:       make(map[string]map[asset.Item]time.Time),
		orderbookJobs:                  make(chan syncJob, defaultChannelBuffer),
		tickerJobs:                     make(chan syncJob, defaultChannelBuffer),
		// TODO: Implement trade synchronization.
	}

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout REST: %v"+
			" timeout Websocket: %v",
		s.config.SynchronizeContinuously, s.config.SynchronizeTicker, s.config.SynchronizeOrderbook,
		s.config.SynchronizeTrades, s.config.NumWorkers, s.config.Verbose, s.config.TimeoutREST,
		s.config.TimeoutWebsocket)
	s.initSyncWG.Add(1)
	return s, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *syncManager) IsRunning() bool {
	return m != nil && atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *syncManager) Start() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNil)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return subsystem.ErrAlreadyStarted
	}
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer started.")
	exchanges, err := m.exchangeManager.GetExchanges()
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

		usingWebsocket := m.websocketRoutineManagerEnabled && supportsWebsocket && exchanges[x].IsWebsocketEnabled()
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

			if !m.config.SynchronizeContinuously {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
				err := m.Stop()
				if err != nil {
					log.Error(log.SyncMgr, err)
				}
				return
			}
		}
	}()

	if atomic.LoadInt32(&m.initSyncCompleted) == 1 && !m.config.SynchronizeContinuously {
		return nil
	}

	go m.controller()

	for i := 0; i < m.config.NumWorkers; i++ {
		go m.orderbookWorker(context.TODO())
		go m.tickerWorker(context.TODO())
		go m.tradeWorker(context.TODO())
	}
	m.initSyncWG.Done()
	return nil
}

// Stop shuts down the exchange currency pair syncer
func (m *syncManager) Stop() error {
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
func (m *syncManager) Update(exchangeName string, p currency.Pair, a asset.Item, syncType int, incomingErr error) error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNil)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", subsystem.ErrNotStarted)
	}

	if atomic.LoadInt32(&m.initSyncStarted) != 1 {
		return nil
	}

	switch syncType {
	case SyncItemOrderbook:
		if !m.config.SynchronizeOrderbook {
			return nil
		}
	case SyncItemTicker:
		if !m.config.SynchronizeTicker {
			return nil
		}
	case SyncItemTrade:
		if !m.config.SynchronizeTrades {
			return nil
		}
	default:
		return fmt.Errorf("%v %w", syncType, errUnknownSyncItem)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.currencyPairs[exchangeName][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return fmt.Errorf("%v %v %v %w", exchangeName, a, p, errSyncerNotFound)
	}

	var specificAgentBase *syncBase
	var name string
	switch syncType {
	case SyncItemTicker:
		specificAgentBase = &agent.Ticker
		name = "ticker"
	case SyncItemOrderbook:
		specificAgentBase = &agent.Orderbook
		name = "orderbook"
	case SyncItemTrade:
		specificAgentBase = &agent.Trade
		name = "trade"
	}

	origHadData := specificAgentBase.HaveData
	specificAgentBase.LastUpdated = time.Now()
	if incomingErr != nil {
		agent.Ticker.NumErrors++
	}
	specificAgentBase.HaveData = true
	specificAgentBase.IsProcessing = false
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
	if m.removedCounter <= m.createdCounter {
		m.initSyncWG.Done()
	}
	return nil
}

// controller checks all enabled assets on all enabled exchanges for correct
// synchronization. If an assets needs updating via REST it will push the work
// to worker routines.
func (m *syncManager) controller() {
	defer log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer worker shutting down.")
	timer := time.NewTimer(0) // Fire immediately for initial synchronization
	for range timer.C {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		wait := m.GetSmallestTimeout()
		exchanges, err := m.exchangeManager.GetExchanges()
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
func (m *syncManager) GetSmallestTimeout() time.Duration {
	if m.config.TimeoutREST < m.config.TimeoutWebsocket {
		return m.config.TimeoutREST
	}
	return m.config.TimeoutWebsocket
}

// CheckSyncItems checks agent against it's current last update time on all
func (m *syncManager) CheckSyncItems(exch exchange.IBotExchange, pair currency.Pair, a asset.Item, usingREST, usingWS, supportsREST, switchedToRest bool) time.Duration {
	agent := m.getAgent(exch.GetName(), pair, a, usingREST, usingWS)

	untilUpdate := m.GetSmallestTimeout()
	until, update := agent.Orderbook.NeedsUpdate(m.config.SynchronizeOrderbook, m.config.TimeoutREST, m.config.TimeoutWebsocket)
	if update {
		// TODO: Fix this
		if usingWS && supportsREST {
			agent.Orderbook.SwitchToREST(exch.GetName(), "Orderbook", agent.Pair, agent.AssetType, m.config.TimeoutWebsocket)
			switchedToRest = true
		}
		err := agent.Orderbook.SetProcessing(true)
		if err != nil {
			panic(err)
		}
		m.sendJob(exch, agent.Pair, agent.AssetType, SyncItemOrderbook)
	} else if until > 0 && until < untilUpdate {
		untilUpdate = until
	}

	until, update = agent.Ticker.NeedsUpdate(m.config.SynchronizeTicker, m.config.TimeoutREST, m.config.TimeoutWebsocket)
	if update {
		// TODO: Fix this
		if usingWS && supportsREST {
			agent.Ticker.SwitchToREST(exch.GetName(), "Ticker", agent.Pair, agent.AssetType, m.config.TimeoutWebsocket)
			switchedToRest = true
		}
		err := agent.Ticker.SetProcessing(true)
		if err != nil {
			panic(err)
		}
		m.sendJob(exch, agent.Pair, agent.AssetType, SyncItemTicker)
	} else if until > 0 && until < untilUpdate {
		untilUpdate = until
	}

	until, update = agent.Trade.NeedsUpdate(m.config.SynchronizeTrades, m.config.TimeoutREST, m.config.TimeoutWebsocket)
	if update {
		if usingWS && supportsREST {
			agent.Trade.SwitchToREST(exch.GetName(), "Trade", agent.Pair, agent.AssetType, m.config.TimeoutWebsocket)
			switchedToRest = true
		}
		err := agent.Trade.SetProcessing(true)
		if err != nil {
			panic(err)
		}
		m.sendJob(exch, agent.Pair, agent.AssetType, SyncItemTrade)
	} else if until > 0 && until < untilUpdate {
		untilUpdate = until
	}

	return untilUpdate
}

// NeedsUpdate determines if the underlying agent sync base is ready for an
// update via REST.
func (s *syncBase) NeedsUpdate(configAllowed bool, timeOutRest, timeOutWS time.Duration) (time.Duration, bool) {
	if !configAllowed {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.IsProcessing {
		return 0, false
	}
	if s.IsUsingWebsocket {
		added := s.LastUpdated.Add(timeOutWS)
		return time.Until(added), !s.LastUpdated.IsZero() && time.Since(s.LastUpdated) >= timeOutWS
	}
	added := s.LastUpdated.Add(timeOutRest)
	return time.Until(added), s.LastUpdated.IsZero() || time.Since(s.LastUpdated) >= timeOutRest
}

// SwitchToREST switches current protocol update from websocket to rest, in the
// event it exceeds the websocket timeout duration.
func (s *syncBase) SwitchToREST(exch, service string, pair currency.Pair, a asset.Item, wsTimeout time.Duration) {
	s.mu.Lock()
	s.IsUsingWebsocket = false
	s.IsUsingREST = true
	s.mu.Unlock()

	log.Warnf(log.SyncMgr, "%s %s %s: No %s update after %s, switching from WEBSOCKET to REST",
		exch,
		pair,
		strings.ToUpper(a.String()),
		service,
		wsTimeout)
	// switchedToRest = true
}

var errProcessingConflict = errors.New("processing conflict")

// SetProcessing sets processing for the specific sync agent
func (s *syncBase) SetProcessing(processing bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.IsProcessing && processing {
		return errProcessingConflict
	}
	s.IsProcessing = processing
	return nil
}

// sendJob sets agent base as processing for that ticker item then sends the
// REST sync job to the jobs channel for processing.
func (m *syncManager) sendJob(exch exchange.IBotExchange, pair currency.Pair, a asset.Item, class int) {
	// NOTE: This is blocking, if there are no receivers and the buffer size is
	// full then this will hang the controller.
	switch class {
	case SyncItemOrderbook:
		m.orderbookJobs <- syncJob{exch: exch, Pair: pair, Asset: a, class: class}
	case SyncItemTicker:
		m.tickerJobs <- syncJob{exch: exch, Pair: pair, Asset: a, class: class}
	case SyncItemTrade:
		m.tradeJobs <- syncJob{exch: exch, Pair: pair, Asset: a, class: class}
	}
}

// getAgent returns an agent and will generate a new agent if not found.
func (m *syncManager) getAgent(exch string, pair currency.Pair, a asset.Item, usingRest, usingWS bool) *currencyPairSyncAgent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currencyPairs == nil {
		m.currencyPairs = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
	}

	m1, ok := m.currencyPairs[exch]
	if !ok {
		m1 = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
		m.currencyPairs[exch] = m1
	}

	m2, ok := m1[pair.Base.Item]
	if !ok {
		m2 = make(map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
		m1[pair.Base.Item] = m2
	}

	m3, ok := m2[pair.Quote.Item]
	if !ok {
		m3 = make(map[asset.Item]*currencyPairSyncAgent)
		m2[pair.Quote.Item] = m3
	}

	agent := m3[a]
	if agent != nil {
		return agent
	}

	agent = &currencyPairSyncAgent{Exchange: exch, Pair: pair, AssetType: a}
	if m.config.SynchronizeTicker {
		agent.Ticker = m.deployBase("ticker", agent, usingWS, usingRest)
	}
	if m.config.SynchronizeOrderbook {
		agent.Orderbook = m.deployBase("orderbook", agent, usingWS, usingRest)
	}
	if m.config.SynchronizeTrades {
		agent.Trade = m.deployBase("trade", agent, usingWS, usingRest)
	}

	m3[a] = agent
	return agent
}

// deployBase deploys a instance of a base struct for each individiual
// synchronization item. If verbose it will display the added item. If state
// is in initial sync it will increment counter and add to the waitgroup.
func (m *syncManager) deployBase(service string, agent *currencyPairSyncAgent, usingWS, usingREST bool) syncBase {
	if m.config.Verbose {
		log.Debugf(log.SyncMgr,
			"%s: Added trade sync item %s: using websocket: %v using REST: %v",
			agent.Exchange, m.FormatCurrency(agent.Pair), usingWS, usingREST)
	}
	if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
		m.initSyncWG.Add(1)
		m.createdCounter++
	}
	return syncBase{IsUsingREST: usingREST, IsUsingWebsocket: usingWS}
}

func (m *syncManager) orderbookWorker(ctx context.Context) {
	for j := range m.orderbookJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		var result *orderbook.Base
		result, err = j.exch.UpdateOrderbook(ctx, j.Pair, j.Asset)
		m.PrintOrderbookSummary(result, "REST", err)
		if err == nil && m.remoteConfig.WebsocketRPC.Enabled {
			relayWebsocketEvent(result, "orderbook_update", j.Asset.String(), exchName)
		}
		err = m.Update(exchName, j.Pair, j.Asset, j.class, err)
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}
}

func (m *syncManager) tickerWorker(ctx context.Context) {
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
			if batchLastDone.IsZero() || time.Since(batchLastDone) > m.config.TimeoutREST {
				if m.config.Verbose {
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
			if m.config.Verbose {
				log.Debugf(log.SyncMgr, "%s Using recent batching cache", exchName)
			}
			result, err = j.exch.FetchTicker(ctx, j.Pair, j.Asset)
		} else {
			result, err = j.exch.UpdateTicker(ctx, j.Pair, j.Asset)
		}
		m.PrintTickerSummary(result, "REST", err)
		if err == nil && m.remoteConfig.WebsocketRPC.Enabled {
			relayWebsocketEvent(result, "ticker_update", j.Asset.String(), exchName)
		}
		err = m.Update(exchName, j.Pair, j.Asset, j.class, err)
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}
}

func (m *syncManager) tradeWorker(ctx context.Context) {
	for j := range m.tradeJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		// TODO: Implement trade synchronization.
		err = m.Update(exchName, j.Pair, j.Asset, j.class, err)
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
func (m *syncManager) PrintTickerSummary(result *ticker.Price, protocol string, err error) {
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
		!result.Pair.Quote.Equal(m.fiatDisplayCurrency) &&
		!m.fiatDisplayCurrency.IsEmpty() {
		origCurrency := result.Pair.Quote.Upper()
		log.Infof(log.SyncMgr, "%s %s %s %s TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
			result.ExchangeName,
			protocol,
			m.FormatCurrency(result.Pair),
			strings.ToUpper(result.AssetType.String()),
			printConvertCurrencyFormat(result.Last, origCurrency, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(result.Ask, origCurrency, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(result.Bid, origCurrency, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(result.High, origCurrency, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(result.Low, origCurrency, m.fiatDisplayCurrency),
			result.Volume)
	} else {
		if result.Pair.Quote.IsFiatCurrency() &&
			result.Pair.Quote.Equal(m.fiatDisplayCurrency) &&
			!m.fiatDisplayCurrency.IsEmpty() {
			log.Infof(log.SyncMgr, "%s %s %s %s TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
				result.ExchangeName,
				protocol,
				m.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				printCurrencyFormat(result.Last, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Ask, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Bid, m.fiatDisplayCurrency),
				printCurrencyFormat(result.High, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Low, m.fiatDisplayCurrency),
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
func (m *syncManager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return p
	}
	return p.Format(m.format)
}

const (
	book = "%s %s %s %s ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s"
)

// PrintOrderbookSummary outputs orderbook results
func (m *syncManager) PrintOrderbookSummary(result *orderbook.Base, protocol string, err error) {
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
	case result.Pair.Quote.IsFiatCurrency() && !result.Pair.Quote.Equal(m.fiatDisplayCurrency) && !m.fiatDisplayCurrency.IsEmpty():
		origCurrency := result.Pair.Quote.Upper()
		if bidsValue > 0 {
			bidValueResult = printConvertCurrencyFormat(bidsValue, origCurrency, m.fiatDisplayCurrency)
		}
		if asksValue > 0 {
			askValueResult = printConvertCurrencyFormat(asksValue, origCurrency, m.fiatDisplayCurrency)
		}
	case result.Pair.Quote.IsFiatCurrency() && result.Pair.Quote.Equal(m.fiatDisplayCurrency) && !m.fiatDisplayCurrency.IsEmpty():
		bidValueResult = printCurrencyFormat(bidsValue, m.fiatDisplayCurrency)
		askValueResult = printCurrencyFormat(asksValue, m.fiatDisplayCurrency)
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
func (m *syncManager) WaitForInitialSync() error {
	if m == nil {
		return fmt.Errorf("sync manager %w", subsystem.ErrNil)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("sync manager %w", subsystem.ErrNotStarted)
	}
	m.initSyncWG.Wait()
	return nil
}

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	evt := WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	}
	err := BroadcastWebsocketMessage(evt)
	if !errors.Is(err, ErrWebsocketServiceNotRunning) {
		log.Errorf(log.APIServerMgr, "Failed to broadcast websocket event %v. Error: %s",
			event, err)
	}
}
