package engine

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type syncItemType int

// const holds the sync item types
const (
	SyncItemTicker syncItemType = iota
	SyncItemOrderbook
	SyncItemTrade
	SyncManagerName = "exchange_syncer"
)

var (
	createdCounter         = 0
	removedCounter         = 0
	errNoSyncItemsEnabled  = errors.New("no sync items enabled")
	errUnknownSyncItem     = errors.New("unknown sync item")
	errCouldNotSyncNewData = errors.New("could not sync new data")
)

// setupSyncManager starts a new CurrencyPairSyncer
func setupSyncManager(c *config.SyncManagerConfig, exchangeManager iExchangeManager, remoteConfig *config.RemoteControlConfig, websocketRoutineManagerEnabled bool) (*syncManager, error) {
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
		c.NumWorkers = config.DefaultSyncerWorkers
	}

	if c.TimeoutREST <= time.Duration(0) {
		c.TimeoutREST = config.DefaultSyncerTimeoutREST
	}

	if c.TimeoutWebsocket <= time.Duration(0) {
		c.TimeoutWebsocket = config.DefaultSyncerTimeoutWebsocket
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
		tickerBatchLastRequested:       make(map[string]time.Time),
		currencyPairs:                  make(map[currencyPairKey]*currencyPairSyncAgent),
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
	if !m.config.SynchronizeTicker &&
		!m.config.SynchronizeOrderbook &&
		!m.config.SynchronizeTrades {
		return errNoSyncItemsEnabled
	}
	m.shutdown = make(chan bool)
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

		var usingWebsocket bool
		var usingREST bool
		if m.websocketRoutineManagerEnabled &&
			supportsWebsocket &&
			exchanges[x].IsWebsocketEnabled() {
			usingWebsocket = true
		} else if supportsREST {
			usingREST = true
		}

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
				k := currencyPairKey{
					AssetType: assetTypes[y],
					Exchange:  exchangeName,
					Pair:      enabledPairs[i].Format(currency.PairFormat{Uppercase: true}),
				}
				if e := m.get(k); e != nil {
					continue
				}

				sBase := syncBase{
					IsUsingREST:      usingREST || !wsAssetSupported,
					IsUsingWebsocket: usingWebsocket && wsAssetSupported,
				}

				m.add(k, sBase)
			}
		}
	}

	if atomic.CompareAndSwapInt32(&m.initSyncStarted, 0, 1) {
		if m.config.LogInitialSyncEvents {
			log.Debugf(log.SyncMgr,
				"Exchange CurrencyPairSyncer initial sync started. %d items to process.",
				createdCounter)
		}
		m.initSyncStartTime = time.Now()
	}

	go func() {
		m.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&m.initSyncCompleted, 0, 1) {
			if m.config.LogInitialSyncEvents {
				log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync is complete.")
				log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync took %v [%v sync items].",
					time.Since(m.initSyncStartTime), createdCounter)
			}

			if !m.config.SynchronizeContinuously {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
				err := m.Stop()
				if err != nil {
					log.Errorln(log.SyncMgr, err)
				}
				return
			}
		}
	}()

	if atomic.LoadInt32(&m.initSyncCompleted) == 1 && !m.config.SynchronizeContinuously {
		return nil
	}

	for i := 0; i < m.config.NumWorkers; i++ {
		go m.worker()
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
	close(m.shutdown)
	m.inService.Add(1)
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	return nil
}

func (m *syncManager) get(k currencyPairKey) *currencyPairSyncAgent {
	m.mux.Lock()
	defer m.mux.Unlock()

	return m.currencyPairs[k]
}

func newCurrencyPairSyncAgent(k currencyPairKey) *currencyPairSyncAgent {
	return &currencyPairSyncAgent{
		currencyPairKey: k,
		Created:         time.Now(),
		locks:           make([]sync.Mutex, SyncItemTrade+1),
		trackers:        make([]*syncBase, SyncItemTrade+1),
	}
}
func (m *syncManager) add(k currencyPairKey, s syncBase) *currencyPairSyncAgent {
	m.mux.Lock()
	defer m.mux.Unlock()

	c := newCurrencyPairSyncAgent(k)

	if m.config.SynchronizeTicker {
		s := s
		c.trackers[SyncItemTicker] = &s
	}

	if m.config.SynchronizeOrderbook {
		s := s
		c.trackers[SyncItemOrderbook] = &s
	}

	if m.config.SynchronizeTrades {
		s := s
		c.trackers[SyncItemTrade] = &s
	}

	if m.config.SynchronizeTicker {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added ticker sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.trackers[SyncItemTicker].IsUsingWebsocket,
				c.trackers[SyncItemTicker].IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if m.config.SynchronizeOrderbook {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added orderbook sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.trackers[SyncItemOrderbook].IsUsingWebsocket,
				c.trackers[SyncItemOrderbook].IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if m.config.SynchronizeTrades {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added trade sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.trackers[SyncItemTrade].IsUsingWebsocket,
				c.trackers[SyncItemTrade].IsUsingREST)
		}
		if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
			m.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if m.currencyPairs == nil {
		m.currencyPairs = make(map[currencyPairKey]*currencyPairSyncAgent)
	}

	m.currencyPairs[k] = c

	return c
}

// WebsocketUpdate notifies the syncManager to change the last updated time for a exchange asset pair
// And set IsUsingWebsocket to true. It should be used externally only from websocket updaters
func (m *syncManager) WebsocketUpdate(exchangeName string, p currency.Pair, a asset.Item, syncType syncItemType, err error) error {
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

	k := currencyPairKey{
		AssetType: a,
		Exchange:  exchangeName,
		Pair:      p.Format(currency.PairFormat{Uppercase: true}),
	}

	c, exists := m.currencyPairs[k]
	if !exists {
		return fmt.Errorf("%w for %s %s %s %s", errCouldNotSyncNewData, k.Exchange, k.Pair, k.AssetType, syncType)
	}

	c.locks[syncType].Lock()
	defer c.locks[syncType].Unlock()

	if c.trackers[syncType] == nil {
		c.trackers[syncType] = &syncBase{}
	}
	s := c.trackers[syncType]

	if !s.IsUsingWebsocket {
		s.IsUsingWebsocket = true
		s.IsUsingREST = false
		if m.config.LogSwitchProtocolEvents {
			log.Warnf(log.SyncMgr,
				"%s %s %s: %s Websocket re-enabled, switching from rest to websocket",
				c.Exchange,
				m.FormatCurrency(c.Pair),
				strings.ToUpper(c.AssetType.String()),
				syncType,
			)
		}
	}

	return m.update(c, syncType, err)
}

// update notifies the syncManager to change the last updated time for a exchange asset pair
func (m *syncManager) update(c *currencyPairSyncAgent, syncType syncItemType, err error) error {
	if syncType < SyncItemTicker || syncType > SyncItemTrade {
		return fmt.Errorf("%v %w", syncType, errUnknownSyncItem)
	}

	s := c.trackers[syncType]

	origHadData := s.HaveData
	s.LastUpdated = time.Now()
	if err != nil {
		s.NumErrors++
	}
	s.HaveData = true
	if atomic.LoadInt32(&m.initSyncCompleted) != 1 && !origHadData {
		removedCounter++
		if m.config.LogInitialSyncEvents {
			log.Debugf(log.SyncMgr, "%s %s sync complete %v [%d/%d].",
				c.Exchange,
				syncType,
				m.FormatCurrency(c.Pair),
				removedCounter,
				createdCounter)
		}
		m.initSyncWG.Done()
	}

	return nil
}

func (m *syncManager) worker() {
	cleanup := func() {
		log.Debugln(log.SyncMgr,
			"Exchange CurrencyPairSyncer worker shutting down.")
	}
	defer cleanup()

	interval := greatestCommonDivisor(m.config.TimeoutWebsocket, m.config.TimeoutREST)
	if interval > time.Second {
		interval = time.Second
	}
	t := time.NewTicker(interval)

	for {
		select {
		case <-m.shutdown:
			return
		case <-t.C:
			exchanges, err := m.exchangeManager.GetExchanges()
			if err != nil {
				log.Errorf(log.SyncMgr, "Sync manager cannot get exchanges: %v", err)
				continue
			}
			for _, e := range exchanges {
				exchangeName := e.GetName()
				supportsREST := e.SupportsREST()
				// TODO: These vars are only used for enabling new pairs, deriving them every cycle is sub-optimal
				var usingREST bool
				var usingWebsocket bool
				if e.SupportsWebsocket() && e.IsWebsocketEnabled() {
					ws, err := e.GetWebsocket()
					if err != nil {
						log.Errorf(log.SyncMgr,
							"%s unable to get websocket pointer. Err: %s",
							exchangeName,
							err)
						usingREST = true
					}

					if ws.IsConnected() {
						usingWebsocket = true
					} else {
						usingREST = true
					}
				} else if supportsREST {
					usingREST = true
				}

				assetTypes := e.GetAssetTypes(true)
				for y := range assetTypes {
					wsAssetSupported := e.IsAssetWebsocketSupported(assetTypes[y])
					enabledPairs, err := e.GetEnabledPairs(assetTypes[y])
					if err != nil {
						log.Errorf(log.SyncMgr,
							"%s failed to get enabled pairs. Err: %s",
							e.GetName(),
							err)
						continue
					}
					for i := range enabledPairs {
						if atomic.LoadInt32(&m.started) == 0 {
							return
						}

						k := currencyPairKey{
							AssetType: assetTypes[y],
							Exchange:  exchangeName,
							Pair:      enabledPairs[i].Format(currency.PairFormat{Uppercase: true}),
						}
						c := m.get(k)
						if c == nil {
							c = m.add(k, syncBase{
								IsUsingREST:      usingREST || !wsAssetSupported,
								IsUsingWebsocket: usingWebsocket && wsAssetSupported,
							})
						}

						if m.config.SynchronizeTicker {
							m.syncTicker(c, e)
						}
						if m.config.SynchronizeOrderbook {
							m.syncOrderbook(c, e)
						}
						if m.config.SynchronizeTrades {
							m.syncTrades(c)
						}
					}
				}
			}
		}
	}
}

func (m *syncManager) syncTicker(c *currencyPairSyncAgent, e exchange.IBotExchange) {
	if !c.locks[SyncItemTicker].TryLock() {
		return
	}
	defer c.locks[SyncItemTicker].Unlock()

	exchangeName := e.GetName()

	s := c.trackers[SyncItemTicker]

	if s.IsUsingWebsocket &&
		e.SupportsREST() &&
		time.Since(s.LastUpdated) > m.config.TimeoutWebsocket &&
		time.Since(c.Created) > m.config.TimeoutWebsocket {
		// Downgrade to REST
		s.IsUsingWebsocket = false
		s.IsUsingREST = true
		if m.config.LogSwitchProtocolEvents {
			log.Warnf(log.SyncMgr,
				"%s %s %s: No ticker update after %s, switching from websocket to rest",
				c.Exchange,
				m.FormatCurrency(c.Pair),
				strings.ToUpper(c.AssetType.String()),
				m.config.TimeoutWebsocket,
			)
		}
	}

	if s.IsUsingREST && time.Since(s.LastUpdated) > m.config.TimeoutREST {
		var result *ticker.Price
		var err error

		if e.SupportsRESTTickerBatchUpdates() {
			m.mux.Lock()
			batchLastDone, ok := m.tickerBatchLastRequested[e.GetName()]
			if !ok {
				m.tickerBatchLastRequested[exchangeName] = time.Time{}
			}
			m.mux.Unlock()

			if batchLastDone.IsZero() || time.Since(batchLastDone) > m.config.TimeoutREST {
				m.mux.Lock()
				if m.config.Verbose {
					log.Debugf(log.SyncMgr, "Initialising %s REST ticker batching", exchangeName)
				}
				err = e.UpdateTickers(context.TODO(), c.AssetType)
				if err == nil {
					result, err = e.FetchTicker(context.TODO(), c.Pair, c.AssetType)
				}
				m.tickerBatchLastRequested[exchangeName] = time.Now()
				m.mux.Unlock()
			} else {
				if m.config.Verbose {
					log.Debugf(log.SyncMgr, "%s Using recent batching cache", exchangeName)
				}
				result, err = e.FetchTicker(context.TODO(),
					c.Pair,
					c.AssetType)
			}
		} else {
			result, err = e.UpdateTicker(context.TODO(),
				c.Pair,
				c.AssetType)
		}
		m.PrintTickerSummary(result, "REST", err)
		if err == nil {
			if m.remoteConfig.WebsocketRPC.Enabled {
				relayWebsocketEvent(result, "ticker_update", c.AssetType.String(), exchangeName)
			}
		}
		updateErr := m.update(c, SyncItemTicker, err)
		if updateErr != nil {
			log.Errorln(log.SyncMgr, updateErr)
		}
	}
}

func (m *syncManager) syncOrderbook(c *currencyPairSyncAgent, e exchange.IBotExchange) {
	if !c.locks[SyncItemOrderbook].TryLock() {
		return
	}
	defer c.locks[SyncItemOrderbook].Unlock()

	s := c.trackers[SyncItemOrderbook]

	if s.IsUsingWebsocket &&
		e.SupportsREST() &&
		time.Since(s.LastUpdated) > m.config.TimeoutWebsocket &&
		time.Since(c.Created) > m.config.TimeoutWebsocket {
		// Downgrade to REST
		s.IsUsingWebsocket = false
		s.IsUsingREST = true
		if m.config.LogSwitchProtocolEvents {
			log.Warnf(log.SyncMgr,
				"%s %s %s: No orderbook update after %s, switching from websocket to rest",
				c.Exchange,
				m.FormatCurrency(c.Pair).String(),
				strings.ToUpper(c.AssetType.String()),
				m.config.TimeoutWebsocket,
			)
		}
	}

	if s.IsUsingREST && time.Since(s.LastUpdated) > m.config.TimeoutREST {
		result, err := e.UpdateOrderbook(context.TODO(),
			c.Pair,
			c.AssetType)
		m.PrintOrderbookSummary(result, "REST", err)
		if err == nil {
			if m.remoteConfig.WebsocketRPC.Enabled {
				relayWebsocketEvent(result, "orderbook_update", c.AssetType.String(), e.GetName())
			}
		}
		updateErr := m.update(c, SyncItemOrderbook, err)
		if updateErr != nil {
			log.Errorln(log.SyncMgr, updateErr)
		}
	}
}

func (m *syncManager) syncTrades(c *currencyPairSyncAgent) {
	if !c.locks[SyncItemTrade].TryLock() {
		return
	}
	defer c.locks[SyncItemTrade].Unlock()

	if time.Since(c.trackers[SyncItemTrade].LastUpdated) > m.config.TimeoutREST {
		err := m.update(c, SyncItemTrade, nil)
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

// PrintTickerSummary outputs the ticker results
func (m *syncManager) PrintTickerSummary(result *ticker.Price, protocol string, err error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if !m.config.SynchronizeTicker {
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
	if !m.config.LogSyncUpdateEvents {
		return
	}

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
	if !m.config.SynchronizeOrderbook {
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
	if !m.config.LogSyncUpdateEvents {
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

func greatestCommonDivisor(a, b time.Duration) time.Duration {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

func (s syncItemType) String() string {
	switch s {
	case SyncItemTicker:
		return "Ticker"
	case SyncItemOrderbook:
		return "Orderbook"
	case SyncItemTrade:
		return "Trade"
	default:
		return fmt.Sprintf("Invalid syncItemType: %d", s)
	}
}
