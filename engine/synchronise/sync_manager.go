package synchronise

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// NewManager returns a new synchronisation manager
func NewManager(c *ManagerConfig) (*Manager, error) {
	if c == nil {
		return nil, fmt.Errorf("%T %w", c, common.ErrNilPointer)
	}

	if !c.SynchroniseOrderbook && !c.SynchroniseTicker {
		return nil, ErrNoItemsEnabled
	}

	if c.ExchangeManager == nil {
		return nil, subsystem.ErrNilExchangeManager
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

	log.Debugf(log.SyncMgr, "Exchange currency pair syncer config: continuous: %v ticker: %v orderbook: %v workers: %v verbose: %v timeout REST: %v timeout Websocket: %v",
		c.SynchroniseContinuously,
		c.SynchroniseTicker,
		c.SynchroniseOrderbook,
		c.NumWorkers,
		c.Verbose,
		c.TimeoutREST,
		c.TimeoutWebsocket)

	var tickerBatchTracking map[string]map[asset.Item]time.Time
	if c.SynchroniseTicker {
		tickerBatchTracking = make(map[string]map[asset.Item]time.Time)
	}

	manager := &Manager{
		ManagerConfig:            *c,
		tickerBatchLastRequested: tickerBatchTracking,
		// TODO: Link up defaultChannelBuffer to config.
	}
	manager.initSyncWG.Add(1)
	return manager, nil
}

// checkAllExchangeAssets checks all exchanges, assets, and enabled pairs
// against the list of configured Synchronisation agents. If a currency pair is
// disabled during operation, it is ignored. If a currency pair is not found in
// the list of loaded agents, it is added. The function returns duration until
// the next update is required.
func (m *Manager) checkAllExchangeAssets() (time.Duration, error) {
	exchs, err := m.ExchangeManager.GetExchanges()
	if err != nil {
		return 0, err
	}
	wait := m.getSmallestTimeout()
	for x := range exchs {
		usingWebsocket := exchs[x].SupportsWebsocket() && exchs[x].IsWebsocketEnabled()
		if usingWebsocket {
			var ws *stream.Websocket
			ws, err = exchs[x].GetWebsocket()
			if err != nil {
				return 0, err
			}
			usingWebsocket = ws.IsConnected() || ws.IsConnecting()
		}
		usingREST := !usingWebsocket

		assetTypes := exchs[x].GetAssetTypes(true)
		for y := range assetTypes {
			enabledPairs, err := exchs[x].GetEnabledPairs(assetTypes[y])
			if err != nil {
				return 0, err
			}
			wsAssetSupported := exchs[x].IsAssetWebsocketSupported(assetTypes[y])
			assetUsingREST := (usingREST || !wsAssetSupported) && !(usingWebsocket && wsAssetSupported)
			for i := range enabledPairs {
				wait = m.checkSyncItems(exchs[x], enabledPairs[i], assetTypes[y], assetUsingREST, wait)
			}
		}
	}
	return wait, nil
}

// controller checks all enabled assets on all enabled exchanges for correct
// Synchronisation. If an assets needs updating via REST it will push the work
// to worker routines.
func (m *Manager) controller() error {
	// Pre-load all items for initial sync. The controller routine will take
	// over this functionality to keep assets updated via REST only if needed.
	wait, err := m.checkAllExchangeAssets()
	if err != nil {
		return err
	}

	if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
		log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync started. %d items to process.",
			m.createdCounter)
		m.initSyncStartTime = time.Now()
	}

	go func(timer *time.Timer) {
		for range timer.C {
			if atomic.LoadInt32(&m.started) == 0 {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer worker shutting down.")
				return
			}
			wait, err = m.checkAllExchangeAssets()
			if err != nil {
				log.Errorf(log.SyncMgr, "Sync manager checking all exchange assets. Error: %s", err)
				wait = m.getSmallestTimeout()
			}
			timer.Reset(wait)
		}
	}(time.NewTimer(wait))
	return nil
}

// getSmallestTimeout returns the smallest configured timeout for all supported
// Synchronisation protocols for controller sync.
func (m *Manager) getSmallestTimeout() time.Duration {
	if m.TimeoutREST < m.TimeoutWebsocket {
		return m.TimeoutREST
	}
	return m.TimeoutWebsocket
}

// checkSyncItems checks agent against its current last update time on all
// individual Synchronisation items.
func (m *Manager) checkSyncItems(exch exchange.IBotExchange, pair currency.Pair, a asset.Item, usingREST bool, update time.Duration) time.Duration {
	if m.SynchroniseOrderbook {
		agent := m.getAgent(exch.GetName(), pair, a, subsystem.Orderbook, usingREST)
		update = m.checkSyncItem(exch, agent, update)
	}
	if m.SynchroniseTicker {
		agent := m.getAgent(exch.GetName(), pair, a, subsystem.Ticker, usingREST)
		update = m.checkSyncItem(exch, agent, update)
	}
	return update
}

// checkSyncItem checks the individual sync item
func (m *Manager) checkSyncItem(exch exchange.IBotExchange, agent *Agent, update time.Duration) time.Duration {
	until := agent.NextUpdate(m.TimeoutREST, m.TimeoutWebsocket)
	if until == 0 {
		// This needs to set processing in this controller routine so that it
		// can be ignored on next check.
		agent.SetProcessingViaREST(m.TimeoutWebsocket)
		m.sendJob(exch, agent)
	} else if until > 0 && until < update {
		update = until
	}
	return update
}

// sendJob sets agent base as processing for that ticker item then sends the
// REST sync job to the jobs channel for processing.
func (m *Manager) sendJob(exch exchange.IBotExchange, agent *Agent) {
	agent.mu.Lock()
	switch agent.SynchronisationType {
	case subsystem.Orderbook:
		select {
		case m.orderbookJobs <- RESTJob{exch: exch, Pair: agent.Pair, Asset: agent.Asset}:
		default:
			log.Errorln(log.SyncMgr, "Jobs channel is at max capacity for orderbooks, data integrity cannot be trusted.")
		}
	case subsystem.Ticker:
		select {
		case m.tickerJobs <- RESTJob{exch: exch, Pair: agent.Pair, Asset: agent.Asset}:
		default:
			log.Errorln(log.SyncMgr, "Jobs channel is at max capacity for tickers, data integrity cannot be trusted.")
		}
	}
	agent.mu.Unlock()
}

// getAgent returns an agent and will generate a new agent if not found.
func (m *Manager) getAgent(exch string, pair currency.Pair, a asset.Item, syncType subsystem.SynchronisationType, usingREST bool) *Agent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currencyPairs == nil {
		m.currencyPairs = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]map[subsystem.SynchronisationType]*Agent)
	}

	m1, ok := m.currencyPairs[exch]
	if !ok {
		m1 = make(map[*currency.Item]map[*currency.Item]map[asset.Item]map[subsystem.SynchronisationType]*Agent)
		m.currencyPairs[exch] = m1
	}

	m2, ok := m1[pair.Base.Item]
	if !ok {
		m2 = make(map[*currency.Item]map[asset.Item]map[subsystem.SynchronisationType]*Agent)
		m1[pair.Base.Item] = m2
	}

	m3, ok := m2[pair.Quote.Item]
	if !ok {
		m3 = make(map[asset.Item]map[subsystem.SynchronisationType]*Agent)
		m2[pair.Quote.Item] = m3
	}

	m4, ok := m3[a]
	if !ok {
		m4 = make(map[subsystem.SynchronisationType]*Agent)
		m3[a] = m4
	}

	agent := m4[syncType]
	if agent != nil {
		return agent
	}

	agent = &Agent{
		Exchange:            exch,
		Pair:                pair,
		Asset:               a,
		SynchronisationType: syncType,
		IsUsingREST:         usingREST,
		IsUsingWebsocket:    !usingREST,
	}

	m.loadAgent(agent, syncType)

	m4[syncType] = agent
	return agent
}

// loadAgent loads an agent for each individual synchronisation item. If verbose
// it will display the added item. If state is in initial sync it will increment
// counter and add to the waitgroup.
func (m *Manager) loadAgent(agent *Agent, syncItem subsystem.SynchronisationType) {
	if m.Verbose {
		log.Debugf(log.SyncMgr,
			"%s: Added %s sync item %s [%s]: using websocket: %v using REST: %v",
			agent.Exchange,
			syncItem,
			m.formatCurrency(agent.Pair),
			strings.ToUpper(agent.Asset.String()),
			agent.IsUsingREST,
			agent.IsUsingWebsocket)
	}
	if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
		m.initSyncWG.Add(1)
		m.createdCounter++
	}
}

// orderbookWorker waits for an orderbook job then executes a REST request.
func (m *Manager) orderbookWorker(ctx context.Context) {
	defer m.workerWG.Done()
	for j := range m.orderbookJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		result, err := j.exch.UpdateOrderbook(ctx, j.Pair, j.Asset)
		m.PrintOrderbookSummary(result, subsystem.Rest, err)
		if err == nil && m.WebsocketRPCEnabled {
			m.relayWebsocketEvent(result, "orderbook_update", j.Asset.String(), exchName)
		}
		err = m.Update(exchName, subsystem.Rest, j.Pair, j.Asset, subsystem.Orderbook, err)
		if err != nil {
			log.Errorln(log.SyncMgr, err)
		}
	}
}

// tickerWorker waits for a ticker job then executes a REST request.
func (m *Manager) tickerWorker(ctx context.Context) {
	defer m.workerWG.Done()
	for j := range m.tickerJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		var result *ticker.Price
		if j.exch.SupportsRESTTickerBatchUpdates(j.Asset) {
			m.batchMtx.Lock()
			batchLastDone := m.tickerBatchLastRequested[exchName][j.Asset]
			if batchLastDone.IsZero() || time.Since(batchLastDone) >= m.TimeoutREST {
				if m.Verbose {
					log.Debugf(log.SyncMgr, "Initialising %s REST ticker batching", exchName)
				}
				err = j.exch.UpdateTickers(ctx, j.Asset)
				if err != nil {
					log.Errorf(log.SyncMgr, "%s failed batch cache: %v", exchName, err)
				} else {
					assetMap, ok := m.tickerBatchLastRequested[exchName]
					if !ok {
						assetMap = make(map[asset.Item]time.Time)
						m.tickerBatchLastRequested[exchName] = assetMap
					}
					assetMap[j.Asset] = time.Now()
				}
			}
			m.batchMtx.Unlock()
			if m.Verbose {
				log.Debugf(log.SyncMgr, "%s Using recent batching cache", exchName)
			}
			// NOTE: Use ticker package `GetTicker()` as this should be updated
			// via batch and executing exchange `FetchTicker` here will call
			// `UpdateTicker` on any errors. Which:
			// 1) Nullifies all future price updates when `FetchTicker`
			// functionality is used and once updated `UpdateTicker` will never
			// be called again.
			// 2) Suppress error when `FetchTicker` is called.
			// 3) Ultimately misuse resources.
			result, err = ticker.GetTicker(exchName, j.Pair, j.Asset)
		} else {
			result, err = j.exch.UpdateTicker(ctx, j.Pair, j.Asset)
		}
		m.PrintTickerSummary(result, subsystem.Rest, err)
		if err == nil && m.WebsocketRPCEnabled {
			m.relayWebsocketEvent(result, "ticker_update", j.Asset.String(), exchName)
		}
		err = m.Update(exchName, subsystem.Rest, j.Pair, j.Asset, subsystem.Ticker, err)
		if err != nil {
			log.Errorln(log.SyncMgr, err)
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

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *Manager) formatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return p
	}
	return p.Format(m.PairFormatDisplay)
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

// relayWebsocketEvent relays websocket event.
func (m *Manager) relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	if m.APIServerManager == nil || !m.APIServerManager.IsWebsocketServerRunning() {
		return
	}
	err := m.APIServerManager.BroadcastWebsocketMessage(subsystem.WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	})
	if !errors.Is(err, subsystem.ErrWebsocketServiceNotRunning) {
		// TODO: Fix globals in apiserver.go file and possibly deprecate this
		// link.
		log.Errorf(log.APIServerMgr, "Failed to broadcast websocket event %v. Error: %s",
			event, err)
	}
}

// NextUpdate returns the next update time, if the underlying agent sync base is
// ready for an update via REST it will return 0. -1 indicates there is no update
// needed.
func (a *Agent) NextUpdate(timeoutRest, timeoutWS time.Duration) time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.IsProcessing {
		// Update not needed
		return -1
	}

	protocolTimeout := timeoutRest
	if a.IsUsingWebsocket {
		if a.LastUpdated.IsZero() {
			// Update not needed, stream connection established waiting for
			// initial update.
			return -1
		}
		protocolTimeout = timeoutWS
	} else if a.LastUpdated.IsZero() {
		// Update immediately, via REST request
		return 0
	}

	timeout := time.Until(a.LastUpdated.Add(protocolTimeout))
	if timeout <= 0 {
		// Update immediately
		return 0
	}
	return timeout
}

// SetProcessingViaREST sets processing for the specific sync agent when it is
// specifically fetching via REST.
func (a *Agent) SetProcessingViaREST(wsTimeout time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.IsProcessing = true
	if !a.IsUsingWebsocket {
		return
	}
	a.IsUsingWebsocket = false
	a.IsUsingREST = true
	log.Warnf(log.SyncMgr, "%s %s %s: No %s update after %s, switching from WEBSOCKET to REST",
		a.Exchange,
		a.Pair,
		strings.ToUpper(a.Asset.String()),
		a.SynchronisationType,
		wsTimeout)
}

// Update updates the underlying agent fields. If protocol is switched from REST
// to WEBSOCKET it will display that switch.
func (a *Agent) Update(protocol subsystem.ProtocolType, incomingErr error) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if incomingErr != nil {
		a.NumErrors++
	}

	a.IsProcessing = false
	initialUpdate := a.LastUpdated.IsZero()
	a.LastUpdated = time.Now()

	if protocol == subsystem.Websocket && a.IsUsingREST {
		a.IsUsingREST = false
		a.IsUsingWebsocket = true
		log.Warnf(log.SyncMgr, "%s %s %s: %s update received, switching from REST to WEBSOCKET",
			a.Exchange,
			a.Pair,
			strings.ToUpper(a.Asset.String()),
			a.SynchronisationType)
	}
	return initialUpdate
}
