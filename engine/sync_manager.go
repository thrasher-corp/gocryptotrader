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
	errNoSyncItemsEnabled         = errors.New("no sync items enabled")
	errUnknownSyncItem            = errors.New("unknown sync item")
	errSyncItemAlreadyAdded       = errors.New("currency sync item already added")
	errSyncerNotFound             = errors.New("sync agent not found")
)

// setupSyncManager starts a new CurrencyPairSyncer
func setupSyncManager(c *SyncManagerConfig, exchangeManager iExchangeManager, remoteConfig *config.RemoteControlConfig, websocketRoutineManagerEnabled bool) (*syncManager, error) {
	if c == nil {
		return nil, fmt.Errorf("%T %w", c, common.ErrNilPointer)
	}

	if !c.SynchronizeOrderbook && !c.SynchronizeTicker && !c.SynchronizeTrades {
		return nil, errNoSyncItemsEnabled
	}
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if remoteConfig == nil {
		return nil, errNilConfig
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
		jobs:                           make(chan syncJob, 10000),
	}

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout REST: %v"+
			" timeout Websocket: %v",
		s.config.SynchronizeContinuously, s.config.SynchronizeTicker, s.config.SynchronizeOrderbook,
		s.config.SynchronizeTrades, s.config.NumWorkers, s.config.Verbose, s.config.TimeoutREST,
		s.config.TimeoutWebsocket)
	s.inService.Add(1)
	return s, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *syncManager) IsRunning() bool {
	return m != nil && atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *syncManager) Start() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return ErrSubSystemAlreadyStarted
	}
	m.initSyncWG.Add(1)
	m.inService.Done()
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
			log.Warnf(log.SyncMgr,
				"Loaded exchange %s does not support REST or Websocket.",
				exchangeName)
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
				if m.exists(exchangeName, enabledPairs[i], assetTypes[y]) {
					continue
				}

				c := &currencyPairSyncAgent{
					AssetType: assetTypes[y],
					Exchange:  exchangeName,
					Pair:      enabledPairs[i],
				}
				sBase := syncBase{
					IsUsingREST:      usingREST || !wsAssetSupported,
					IsUsingWebsocket: usingWebsocket && wsAssetSupported,
				}
				if m.config.SynchronizeTicker {
					c.Ticker = sBase
				}
				if m.config.SynchronizeOrderbook {
					c.Orderbook = sBase
				}
				if m.config.SynchronizeTrades {
					c.Trade = sBase
				}

				m.add(c)
			}
		}
	}

	if atomic.CompareAndSwapInt32(&m.initSyncStarted, 0, 1) {
		log.Debugf(log.SyncMgr,
			"Exchange CurrencyPairSyncer initial sync started. %d items to process.",
			m.createdCounter)
		m.initSyncStartTime = time.Now()
	}

	go func() {
		m.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&m.initSyncCompleted, 0, 1) {
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync is complete.")
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync took %v [%v sync items].",
				time.Since(m.initSyncStartTime), m.createdCounter)

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
		go m.worker(context.TODO())
	}
	m.initSyncWG.Done()
	return nil
}

// Stop shuts down the exchange currency pair syncer
func (m *syncManager) Stop() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrSubSystemNotStarted)
	}
	m.inService.Add(1)
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	return nil
}

func (m *syncManager) get(exchangeName string, p currency.Pair, a asset.Item) (*currencyPairSyncAgent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.currencyPairs[exchangeName][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return nil, fmt.Errorf("%v %v %v %w", exchangeName, a, p, errSyncerNotFound)
	}
	return agent, nil
}

func (m *syncManager) exists(exchangeName string, p currency.Pair, a asset.Item) bool {
	_, err := m.get(exchangeName, p, a)
	return err == nil
}

func (m *syncManager) add(c *currencyPairSyncAgent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config.SynchronizeTicker {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added ticker sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Ticker.IsUsingWebsocket,
				c.Ticker.IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			m.createdCounter++
		}
	}

	if m.config.SynchronizeOrderbook {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added orderbook sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Orderbook.IsUsingWebsocket,
				c.Orderbook.IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			m.createdCounter++
		}
	}

	if m.config.SynchronizeTrades {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added trade sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Trade.IsUsingWebsocket,
				c.Trade.IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			m.createdCounter++
		}
	}

	if m.currencyPairs == nil {
		m.currencyPairs = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
	}

	m1, ok := m.currencyPairs[c.Exchange]
	if !ok {
		m1 = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
		m.currencyPairs[c.Exchange] = m1
	}

	m2, ok := m1[c.Pair.Base.Item]
	if !ok {
		m2 = make(map[*currency.Item]map[asset.Item]*currencyPairSyncAgent)
		m1[c.Pair.Base.Item] = m2
	}

	m3, ok := m2[c.Pair.Quote.Item]
	if !ok {
		m3 = make(map[asset.Item]*currencyPairSyncAgent)
		m2[c.Pair.Quote.Item] = m3
	}

	_, ok = m3[c.AssetType]
	if ok {
		log.Errorf(log.SyncMgr, "%s %s %s %v", c.Exchange, c.Pair, c.AssetType, errSyncItemAlreadyAdded)
		return
	}

	cpy := *c
	m3[c.AssetType] = &cpy
}

func (m *syncManager) isProcessing(exchangeName string, p currency.Pair, a asset.Item, syncType int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.currencyPairs[exchangeName][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return false
	}

	switch syncType {
	case SyncItemTicker:
		return agent.Ticker.IsProcessing
	case SyncItemOrderbook:
		return agent.Orderbook.IsProcessing
	case SyncItemTrade:
		return agent.Trade.IsProcessing
	}

	return false
}

func (m *syncManager) setProcessing(exchangeName string, p currency.Pair, a asset.Item, syncType int, processing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.currencyPairs[exchangeName][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return
	}
	switch syncType {
	case SyncItemTicker:
		agent.Ticker.IsProcessing = processing
	case SyncItemOrderbook:
		agent.Orderbook.IsProcessing = processing
	case SyncItemTrade:
		agent.Trade.IsProcessing = processing
	}
}

// Update notifies the syncManager to change the last updated time for an exchange asset pair
func (m *syncManager) Update(exchangeName string, p currency.Pair, a asset.Item, syncType int, incomingErr error) error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrSubSystemNotStarted)
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

	var specific *syncBase
	var name string
	switch syncType {
	case SyncItemTicker:
		specific = &agent.Ticker
		name = "ticker"
	case SyncItemOrderbook:
		specific = &agent.Orderbook
		name = "orderbook"
	case SyncItemTrade:
		specific = &agent.Trade
		name = "trade"
	}

	origHadData := specific.HaveData
	specific.LastUpdated = time.Now()
	if incomingErr != nil {
		agent.Ticker.NumErrors++
	}
	specific.HaveData = true
	specific.IsProcessing = false
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
		exchanges, err := m.exchangeManager.GetExchanges()
		if err != nil {
			log.Errorf(log.SyncMgr, "Sync manager cannot get exchanges: %v", err)
		}
		for x := range exchanges {
			exchangeName := exchanges[x].GetName()
			supportsREST := exchanges[x].SupportsREST()

			usingWebsocket := exchanges[x].SupportsWebsocket() && exchanges[x].IsWebsocketEnabled()
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
				for i := range enabledPairs {
					if atomic.LoadInt32(&m.started) == 0 {
						return
					}

					// TODO: Fix race issue to agent pointer.
					c, err := m.get(exchangeName, enabledPairs[i], assetTypes[y])
					if err != nil {
						c = &currencyPairSyncAgent{
							AssetType: assetTypes[y],
							Exchange:  exchangeName,
							Pair:      enabledPairs[i],
						}

						wsAssetSupported := exchanges[x].IsAssetWebsocketSupported(assetTypes[y])
						sBase := syncBase{
							IsUsingREST:      usingREST || !wsAssetSupported,
							IsUsingWebsocket: usingWebsocket && wsAssetSupported,
						}

						if m.config.SynchronizeTicker {
							c.Ticker = sBase
						}

						if m.config.SynchronizeOrderbook {
							c.Orderbook = sBase
						}

						if m.config.SynchronizeTrades {
							c.Trade = sBase
						}

						m.add(c)
					}

					if switchedToRest && usingWebsocket {
						log.Warnf(log.SyncMgr,
							"%s %s: Websocket re-enabled, switching from rest to websocket",
							c.Exchange,
							m.FormatCurrency(enabledPairs[i]).String())
						switchedToRest = false
					}

					lastUpdatedIsZero := c.Orderbook.LastUpdated.IsZero()
					if m.config.SynchronizeOrderbook &&
						!m.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemOrderbook) &&
						((lastUpdatedIsZero && !usingWebsocket) ||
							(!lastUpdatedIsZero && time.Since(c.Orderbook.LastUpdated) >= m.config.TimeoutREST && usingREST) ||
							(!lastUpdatedIsZero && time.Since(c.Orderbook.LastUpdated) >= m.config.TimeoutWebsocket && usingWebsocket)) {
						if usingWebsocket && supportsREST {
							c.Orderbook.IsUsingWebsocket = false
							c.Orderbook.IsUsingREST = true
							log.Warnf(log.SyncMgr,
								"%s %s %s: No orderbook update after %s, switching from websocket to rest",
								c.Exchange,
								m.FormatCurrency(c.Pair).String(),
								strings.ToUpper(c.AssetType.String()),
								m.config.TimeoutWebsocket,
							)
							switchedToRest = true
						}
						m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
						m.jobs <- syncJob{
							exch:  exchanges[x],
							Pair:  c.Pair,
							Asset: c.AssetType,
							class: SyncItemOrderbook,
						}
					}

					lastUpdatedIsZero = c.Ticker.LastUpdated.IsZero()
					if m.config.SynchronizeTicker &&
						!m.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTicker) &&
						((lastUpdatedIsZero && !usingWebsocket) ||
							(!lastUpdatedIsZero && time.Since(c.Ticker.LastUpdated) >= m.config.TimeoutREST && usingREST) ||
							(!lastUpdatedIsZero && time.Since(c.Ticker.LastUpdated) >= m.config.TimeoutWebsocket && usingWebsocket)) {
						if usingWebsocket && supportsREST {
							c.Ticker.IsUsingWebsocket = false
							c.Ticker.IsUsingREST = true
							log.Warnf(log.SyncMgr,
								"%s %s %s: No ticker update after %s, switching from websocket to rest",
								c.Exchange,
								m.FormatCurrency(enabledPairs[i]).String(),
								strings.ToUpper(c.AssetType.String()),
								m.config.TimeoutWebsocket,
							)
							switchedToRest = true
						}
						m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
						m.jobs <- syncJob{
							exch:  exchanges[x],
							Pair:  c.Pair,
							Asset: c.AssetType,
							class: SyncItemTicker,
						}
					}

					lastUpdatedIsZero = c.Trade.LastUpdated.IsZero()
					if m.config.SynchronizeTrades &&
						!m.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTrade) &&
						((lastUpdatedIsZero && !usingWebsocket) ||
							(!lastUpdatedIsZero && time.Since(c.Trade.LastUpdated) >= m.config.TimeoutREST) ||
							(!lastUpdatedIsZero && time.Since(c.Ticker.LastUpdated) >= m.config.TimeoutWebsocket && usingWebsocket)) {
						m.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, true)
						m.jobs <- syncJob{
							exch:  exchanges[x],
							Pair:  c.Pair,
							Asset: c.AssetType,
							class: SyncItemTrade,
						}
					}
				}
			}
		}
		timer.Reset(time.Second)
	}
}

func (m *syncManager) worker(ctx context.Context) {
	for j := range m.jobs {
		if atomic.LoadInt32(&m.started) == 0 {
			return
		}
		exchName := j.exch.GetName()
		var err error
		switch j.class {
		case SyncItemOrderbook:
			var result *orderbook.Base
			result, err = j.exch.UpdateOrderbook(ctx, j.Pair, j.Asset)
			m.PrintOrderbookSummary(result, "REST", err)
			if err == nil && m.remoteConfig.WebsocketRPC.Enabled {
				relayWebsocketEvent(result, "orderbook_update", j.Asset.String(), exchName)
			}
		case SyncItemTicker:
			var result *ticker.Price
			if j.exch.SupportsRESTTickerBatchUpdates() {
				m.batchMtx.Lock()
				batchLastDone := m.tickerBatchLastRequested[exchName][j.Asset]
				if batchLastDone.IsZero() || time.Since(batchLastDone) > m.config.TimeoutREST {
					if m.config.Verbose {
						log.Debugf(log.SyncMgr, "Initialising %s REST ticker batching", exchName)
					}
					_ = j.exch.UpdateTickers(ctx, j.Asset)
					maperino, ok := m.tickerBatchLastRequested[exchName]
					if !ok {
						maperino = make(map[asset.Item]time.Time)
						m.tickerBatchLastRequested[exchName] = maperino
					}
					maperino[j.Asset] = time.Now()
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
		case SyncItemTrade:
			// TODO: Add in trade processing.
		}

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
		return fmt.Errorf("sync manager %w", ErrNilSubsystem)
	}

	m.inService.Wait()
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("sync manager %w", ErrSubSystemNotStarted)
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
