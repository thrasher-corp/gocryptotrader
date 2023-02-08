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

// NewManager returns a new sychronization manager
func NewManager(c *ManagerConfig) (*Manager, error) {
	if c == nil {
		return nil, fmt.Errorf("%T %w", c, common.ErrNilPointer)
	}

	if !c.SynchronizeOrderbook && !c.SynchronizeTicker && !c.SynchronizeTrades {
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

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout REST: %v"+
			" timeout Websocket: %v",
		c.SynchronizeContinuously, c.SynchronizeTicker, c.SynchronizeOrderbook,
		c.SynchronizeTrades, c.NumWorkers, c.Verbose, c.TimeoutREST,
		c.TimeoutWebsocket)

	var tickerBatchTracking map[string]map[asset.Item]time.Time
	var tickerJobsChannel chan RESTJob
	if c.SynchronizeTicker {
		tickerBatchTracking = make(map[string]map[asset.Item]time.Time)
		tickerJobsChannel = make(chan RESTJob, defaultChannelBuffer)
	}

	var orderbookJobsChannel chan RESTJob
	if c.SynchronizeOrderbook {
		orderbookJobsChannel = make(chan RESTJob, defaultChannelBuffer)
	}

	var tradesJobsChannel chan RESTJob
	if c.SynchronizeTrades {
		tradesJobsChannel = make(chan RESTJob, defaultChannelBuffer)
	}

	manager := &Manager{
		ManagerConfig:            *c,
		tickerBatchLastRequested: tickerBatchTracking,
		orderbookJobs:            orderbookJobsChannel,
		tickerJobs:               tickerJobsChannel,
		tradeJobs:                tradesJobsChannel,
	}
	manager.initSyncWG.Add(1)
	return manager, nil
}

// checkAllExchangeAssets checks all exchanges, assets, and enabled pairs
// against the list of configured synchronization agents. If a currency pair is
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
// synchronization. If an assets needs updating via REST it will push the work
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
// synchronization protocols for controller sync.
func (m *Manager) getSmallestTimeout() time.Duration {
	if m.TimeoutREST < m.TimeoutWebsocket {
		return m.TimeoutREST
	}
	return m.TimeoutWebsocket
}

// checkSyncItems checks agent against it's current last update time on all
// individual synchronization items.
func (m *Manager) checkSyncItems(exch exchange.IBotExchange, pair currency.Pair, a asset.Item, usingREST bool, update time.Duration) (smallest time.Duration) {
	agent := m.getAgent(exch.GetName(), pair, a, usingREST)
	if m.SynchronizeOrderbook {
		update = m.checkSyncItem(exch, &agent.Orderbook, agent, subsystem.Orderbook, update)
	}
	if m.SynchronizeTicker {
		update = m.checkSyncItem(exch, &agent.Ticker, agent, subsystem.Ticker, update)
	}
	if m.SynchronizeTrades {
		update = m.checkSyncItem(exch, &agent.Trade, agent, subsystem.Trade, update)
	}
	return update
}

// checkSyncItem checks the individual sync item
func (m *Manager) checkSyncItem(exch exchange.IBotExchange, indv *Base, agent *Agent, sync subsystem.SynchronizationType, update time.Duration) (smallest time.Duration) {
	until, needsUpdate := indv.NeedsUpdate(m.TimeoutREST, m.TimeoutWebsocket)
	if needsUpdate {
		// This needs to set processing in this controller routine so that it
		// can be ignored on next check.
		indv.SetProcessing(agent.Exchange, sync.String(), agent.Pair, agent.AssetType, m.TimeoutWebsocket, true)
		m.sendJob(exch, agent.Pair, agent.AssetType, sync)
	} else if until > 0 && until < update {
		update = until
	}
	return update
}

// sendJob sets agent base as processing for that ticker item then sends the
// REST sync job to the jobs channel for processing.
func (m *Manager) sendJob(exch exchange.IBotExchange, pair currency.Pair, a asset.Item, item subsystem.SynchronizationType) {
	switch item {
	case subsystem.Orderbook:
		select {
		case m.orderbookJobs <- RESTJob{exch: exch, Pair: pair, Asset: a, Item: item}:
		default:
			log.Error(log.SyncMgr, "Jobs channel is at max capacity for orderbooks, data integrity cannot be trusted.")
		}
	case subsystem.Ticker:
		select {
		case m.tickerJobs <- RESTJob{exch: exch, Pair: pair, Asset: a, Item: item}:
		default:
			log.Error(log.SyncMgr, "Jobs channel is at max capacity for tickers, data integrity cannot be trusted.")
		}
	case subsystem.Trade:
		// TODO: add support for trade synchronisation
		// m.tradeJobs <- RESTJob{exch: exch, Pair: pair, Asset: a, Item: item}
	}
}

// getAgent returns an agent and will generate a new agent if not found.
func (m *Manager) getAgent(exch string, pair currency.Pair, a asset.Item, usingREST bool) *Agent {
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
		agent.Ticker = m.deployBase(subsystem.Ticker, agent, usingREST)
	}
	if m.SynchronizeOrderbook {
		agent.Orderbook = m.deployBase(subsystem.Orderbook, agent, usingREST)
	}
	if m.SynchronizeTrades {
		agent.Trade = m.deployBase(subsystem.Trade, agent, usingREST)
	}

	m3[a] = agent
	return agent
}

// deployBase deploys a instance of a base struct for each individual
// synchronization item. If verbose it will display the added item. If state
// is in initial sync it will increment counter and add to the waitgroup.
func (m *Manager) deployBase(service subsystem.SynchronizationType, agent *Agent, usingREST bool) Base {
	if m.Verbose {
		log.Debugf(log.SyncMgr,
			"%s: Added %s sync item %s [%s]: using websocket: %v using REST: %v",
			agent.Exchange,
			service,
			m.formatCurrency(agent.Pair),
			agent.AssetType,
			!usingREST,
			usingREST)
	}
	if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
		m.initSyncWG.Add(1)
		m.createdCounter++
	}
	return Base{IsUsingREST: usingREST, IsUsingWebsocket: !usingREST}
}

// orderbookWorker waits for an orderbook job then executes a REST request.
func (m *Manager) orderbookWorker(ctx context.Context) {
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
		err = m.Update(exchName, subsystem.Rest, j.Pair, j.Asset, j.Item, err)
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}
}

// tickerWorker waits for a ticker job then executes a REST request.
func (m *Manager) tickerWorker(ctx context.Context) {
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
		err = m.Update(exchName, subsystem.Rest, j.Pair, j.Asset, j.Item, err)
		if err != nil {
			log.Error(log.SyncMgr, err)
		}
	}
}

// tradeWorker waits for a trade job then executes a REST request. NOTE: This
// is a POC routine which just blocks because there is currently no support for
// this just yet.
// TODO: Implement trade synchronization.
func (m *Manager) tradeWorker(_ context.Context) {
	for j := range m.tradeJobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		err = m.Update(exchName, subsystem.Rest, j.Pair, j.Asset, j.Item, err)
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

// NeedsUpdate determines if the underlying agent sync base is ready for an
// update via REST.
func (b *Base) NeedsUpdate(timeoutRest, timeoutWS time.Duration) (time.Duration, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.IsProcessing {
		return 0, false
	}
	if b.IsUsingWebsocket {
		added := b.LastUpdated.Add(timeoutWS)
		return time.Until(added), !b.LastUpdated.IsZero() && time.Since(b.LastUpdated) >= timeoutWS
	}
	added := b.LastUpdated.Add(timeoutRest)
	return time.Until(added), b.LastUpdated.IsZero() || time.Since(b.LastUpdated) >= timeoutRest
}

// SetProcessing sets processing for the specific sync agent
func (b *Base) SetProcessing(exch, service string, pair currency.Pair, a asset.Item, wsTimeout time.Duration, processing bool) {
	var switched bool
	b.mu.Lock()
	b.IsProcessing = processing
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
}

// Update updates the underlying sync bases' data and last updated fields.
// If protocol is switched from REST to WEBSOCKET it will display that switch.
func (b *Base) Update(service subsystem.SynchronizationType, exch string, protocol subsystem.ProtocolType, pair currency.Pair, a asset.Item, incomingErr error) (isInitialUpdate bool) {
	b.mu.Lock()
	initialUpdate := b.LastUpdated.IsZero()
	b.LastUpdated = time.Now()
	if incomingErr != nil {
		b.NumErrors++
	}
	b.IsProcessing = false

	var switched bool
	if protocol == subsystem.Websocket {
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
	return initialUpdate
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
