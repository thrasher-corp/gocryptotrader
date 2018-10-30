package engine

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// const holds the sync item types
const (
	SyncItemTicker = iota
	SyncItemOrderbook
	SyncItemTrade

	defaultSyncerWorkers = 30
	defaultSyncerTimeout = time.Second * 15
)

var (
	createdCounter = 0
	removedCounter = 0
)

// NewCurrencyPairSyncer starts a new CurrencyPairSyncer
func NewCurrencyPairSyncer(c CurrencyPairSyncerConfig) (*ExchangeCurrencyPairSyncer, error) {
	if !c.SyncOrderbook && !c.SyncTicker && !c.SyncTrades {
		return nil, errors.New("no sync items enabled")
	}

	if c.NumWorkers <= 0 {
		c.NumWorkers = defaultSyncerWorkers
	}

	s := ExchangeCurrencyPairSyncer{
		Cfg: CurrencyPairSyncerConfig{
			SyncTicker:       c.SyncTicker,
			SyncOrderbook:    c.SyncOrderbook,
			SyncTrades:       c.SyncTrades,
			SyncContinuously: c.SyncContinuously,
			NumWorkers:       c.NumWorkers,
		},
	}

	s.tickerBatchLastRequested = make(map[string]time.Time)

	log.Debugf("Exchange currency pair syncer config:")
	log.Debugf("SyncContinuously: %v", s.Cfg.SyncContinuously)
	log.Debugf("SyncTicker: %v", s.Cfg.SyncTicker)
	log.Debugf("SyncOrderbook: %v", s.Cfg.SyncOrderbook)
	log.Debugf("SyncTrades: %v", s.Cfg.SyncTrades)
	log.Debugf("NumWorkers: %v", s.Cfg.NumWorkers)

	return &s, nil
}

func (e *ExchangeCurrencyPairSyncer) get(exchangeName string, p currency.Pair, a assets.AssetType) (*CurrencyPairSyncAgent, error) {
	e.mux.Lock()
	defer e.mux.Unlock()

	for x := range e.CurrencyPairs {
		if e.CurrencyPairs[x].Exchange == exchangeName &&
			e.CurrencyPairs[x].Pair.Equal(p) &&
			e.CurrencyPairs[x].AssetType == a {
			return &e.CurrencyPairs[x], nil
		}
	}

	return nil, errors.New("exchange currency pair syncer not found")
}

func (e *ExchangeCurrencyPairSyncer) exists(exchangeName string, p currency.Pair, a assets.AssetType) bool {
	e.mux.Lock()
	defer e.mux.Unlock()

	for x := range e.CurrencyPairs {
		if e.CurrencyPairs[x].Exchange == exchangeName &&
			e.CurrencyPairs[x].Pair.Equal(p) &&
			e.CurrencyPairs[x].AssetType == a {
			return true
		}
	}
	return false
}

func (e *ExchangeCurrencyPairSyncer) add(c *CurrencyPairSyncAgent) {
	e.mux.Lock()
	defer e.mux.Unlock()

	if e.Cfg.SyncTicker {
		log.Debugf("%s: Added ticker sync item %v: using websocket: %v using REST: %v", c.Exchange, c.Pair.String(),
			c.Ticker.IsUsingWebsocket, c.Ticker.IsUsingREST)
		if atomic.LoadInt32(&e.initSyncCompleted) != 1 {
			e.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if e.Cfg.SyncOrderbook {
		log.Debugf("%s: Added orderbook sync item %v: using websocket: %v using REST: %v", c.Exchange, c.Pair.String(),
			c.Orderbook.IsUsingWebsocket, c.Orderbook.IsUsingREST)
		if atomic.LoadInt32(&e.initSyncCompleted) != 1 {
			e.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if e.Cfg.SyncTrades {
		log.Debugf("%s: Added trade sync item %v: using websocket: %v using REST: %v", c.Exchange, c.Pair.String(),
			c.Trade.IsUsingWebsocket, c.Trade.IsUsingREST)
		if atomic.LoadInt32(&e.initSyncCompleted) != 1 {
			e.initSyncWG.Add(1)
			createdCounter++
		}
	}

	c.Created = time.Now()
	e.CurrencyPairs = append(e.CurrencyPairs, *c)
}

func (e *ExchangeCurrencyPairSyncer) remove(c *CurrencyPairSyncAgent) {
	e.mux.Lock()
	defer e.mux.Unlock()

	for x := range e.CurrencyPairs {
		if e.CurrencyPairs[x].Exchange == c.Exchange &&
			e.CurrencyPairs[x].Pair.Equal(c.Pair) &&
			e.CurrencyPairs[x].AssetType == c.AssetType {
			e.CurrencyPairs = append(e.CurrencyPairs[:x], e.CurrencyPairs[x+1:]...)
			return
		}
	}
}

func (e *ExchangeCurrencyPairSyncer) isProcessing(exchangeName string, p currency.Pair, a assets.AssetType, syncType int) bool {
	e.mux.Lock()
	defer e.mux.Unlock()

	for x := range e.CurrencyPairs {
		if e.CurrencyPairs[x].Exchange == exchangeName &&
			e.CurrencyPairs[x].Pair.Equal(p) &&
			e.CurrencyPairs[x].AssetType == a {
			switch syncType {
			case SyncItemTicker:
				return e.CurrencyPairs[x].Ticker.IsProcessing
			case SyncItemOrderbook:
				return e.CurrencyPairs[x].Orderbook.IsProcessing
			case SyncItemTrade:
				return e.CurrencyPairs[x].Trade.IsProcessing
			}
		}
	}

	return false
}

func (e *ExchangeCurrencyPairSyncer) setProcessing(exchangeName string, p currency.Pair, a assets.AssetType, syncType int, processing bool) {
	e.mux.Lock()
	defer e.mux.Unlock()

	for x := range e.CurrencyPairs {
		if e.CurrencyPairs[x].Exchange == exchangeName &&
			e.CurrencyPairs[x].Pair.Equal(p) &&
			e.CurrencyPairs[x].AssetType == a {
			switch syncType {
			case SyncItemTicker:
				e.CurrencyPairs[x].Ticker.IsProcessing = processing
			case SyncItemOrderbook:
				e.CurrencyPairs[x].Orderbook.IsProcessing = processing
			case SyncItemTrade:
				e.CurrencyPairs[x].Trade.IsProcessing = processing
			}
		}
	}
}

func (e *ExchangeCurrencyPairSyncer) update(exchangeName string, p currency.Pair, a assets.AssetType, syncType int, err error) {
	if atomic.LoadInt32(&e.initSyncStarted) != 1 {
		return
	}

	switch syncType {
	case SyncItemOrderbook, SyncItemTrade, SyncItemTicker:
		if !e.Cfg.SyncOrderbook && syncType == SyncItemOrderbook {
			return
		}

		if !e.Cfg.SyncTicker && syncType == SyncItemTicker {
			return
		}

		if !e.Cfg.SyncTrades && syncType == SyncItemTrade {
			return
		}
	default:
		log.Warnf("ExchangeCurrencyPairSyncer: unkown sync item %v", syncType)
		return
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	for x := range e.CurrencyPairs {
		if e.CurrencyPairs[x].Exchange == exchangeName &&
			e.CurrencyPairs[x].Pair.Equal(p) &&
			e.CurrencyPairs[x].AssetType == a {
			switch syncType {
			case SyncItemTicker:
				origHadData := e.CurrencyPairs[x].Ticker.HaveData
				e.CurrencyPairs[x].Ticker.LastUpdated = time.Now()
				if err != nil {
					e.CurrencyPairs[x].Ticker.NumErrors++
				}
				e.CurrencyPairs[x].Ticker.HaveData = true
				e.CurrencyPairs[x].Ticker.IsProcessing = false
				if atomic.LoadInt32(&e.initSyncCompleted) != 1 && !origHadData {
					log.Debugf("%s ticker sync complete %v [%d/%d].", exchangeName, p, removedCounter, createdCounter)
					removedCounter++
					e.initSyncWG.Done()
				}

			case SyncItemOrderbook:
				origHadData := e.CurrencyPairs[x].Orderbook.HaveData
				e.CurrencyPairs[x].Orderbook.LastUpdated = time.Now()
				if err != nil {
					e.CurrencyPairs[x].Orderbook.NumErrors++
				}
				e.CurrencyPairs[x].Orderbook.HaveData = true
				e.CurrencyPairs[x].Orderbook.IsProcessing = false
				if atomic.LoadInt32(&e.initSyncCompleted) != 1 && !origHadData {
					log.Debugf("%s orderbook sync complete %v [%d/%d].", exchangeName, p, removedCounter, createdCounter)
					removedCounter++
					e.initSyncWG.Done()
				}

			case SyncItemTrade:
				origHadData := e.CurrencyPairs[x].Trade.HaveData
				e.CurrencyPairs[x].Trade.LastUpdated = time.Now()
				if err != nil {
					e.CurrencyPairs[x].Trade.NumErrors++
				}
				e.CurrencyPairs[x].Trade.HaveData = true
				e.CurrencyPairs[x].Trade.IsProcessing = false
				if atomic.LoadInt32(&e.initSyncCompleted) != 1 && !origHadData {
					log.Debugf("%s trade sync complete %v [%d/%d].", exchangeName, p, removedCounter, createdCounter)
					removedCounter++
					e.initSyncWG.Done()
				}
			}
		}
	}
}

func (e *ExchangeCurrencyPairSyncer) worker() {
	cleanup := func() {
		log.Debugf("Exchange CurrencyPairSyncer worker shutting down.")
	}
	defer cleanup()

	for atomic.LoadInt32(&e.shutdown) != 1 {
		for x := range Bot.Exchanges {
			if !Bot.Exchanges[x].IsEnabled() {
				continue
			}

			if Bot.Exchanges[x].GetName() == "BTCC" {
				continue
			}

			exchangeName := Bot.Exchanges[x].GetName()
			assetTypes := Bot.Exchanges[x].GetAssetTypes()
			supportsREST := Bot.Exchanges[x].SupportsREST()
			supportsRESTTickerBatching := Bot.Exchanges[x].SupportsRESTTickerBatchUpdates()
			var usingREST bool
			var usingWebsocket bool

			if Bot.Exchanges[x].SupportsWebsocket() && Bot.Exchanges[x].IsWebsocketEnabled() {
				ws, err := Bot.Exchanges[x].GetWebsocket()
				if err != nil {
					log.Debugf("%s unable to get websocket pointer. Err: %s", exchangeName, err)
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

			for y := range assetTypes {
				for _, p := range Bot.Exchanges[x].GetEnabledPairs(assetTypes[y]) {
					if atomic.LoadInt32(&e.shutdown) == 1 {
						return
					}

					if !e.exists(exchangeName, p, assetTypes[y]) {
						c := CurrencyPairSyncAgent{
							AssetType: assetTypes[y],
							Exchange:  exchangeName,
							Pair:      p,
						}

						if e.Cfg.SyncTicker {
							c.Ticker = SyncBase{
								IsUsingREST:      usingREST,
								IsUsingWebsocket: usingWebsocket,
							}
						}

						if e.Cfg.SyncOrderbook {
							c.Orderbook = SyncBase{
								IsUsingREST:      usingREST,
								IsUsingWebsocket: usingWebsocket,
							}
						}

						if e.Cfg.SyncTrades {
							c.Trade = SyncBase{
								IsUsingREST:      usingREST,
								IsUsingWebsocket: usingWebsocket,
							}
						}

						e.add(&c)
					}

					c, err := e.get(exchangeName, p, assetTypes[y])
					if err != nil {
						log.Errorf("failed to get item. Err: %s", err)
						continue
					}

					if e.Cfg.SyncTicker {
						if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTicker) {
							if c.Ticker.LastUpdated.IsZero() || time.Since(c.Ticker.LastUpdated) > defaultSyncerTimeout {
								if c.Ticker.IsUsingWebsocket {
									if time.Since(c.Created) < defaultSyncerTimeout {
										continue
									}

									if supportsREST {
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
										c.Ticker.IsUsingWebsocket = false
										c.Ticker.IsUsingREST = true
										log.Warnf("%s %s: No ticker update after 10 seconds, switching from websocket to rest",
											c.Exchange, c.Pair.String())
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, false)
									}
								}

								if c.Ticker.IsUsingREST {
									e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
									var result ticker.Price
									var err error

									if supportsRESTTickerBatching {
										e.mux.Lock()
										batchLastDone, ok := e.tickerBatchLastRequested[exchangeName]
										if !ok {
											e.tickerBatchLastRequested[exchangeName] = time.Time{}
										}
										e.mux.Unlock()

										if batchLastDone.IsZero() || time.Since(batchLastDone) > defaultSyncerTimeout {
											e.mux.Lock()
											if e.Cfg.Verbose {
												log.Debugf("%s Init'ing REST ticker batching", exchangeName)
											}
											result, err = Bot.Exchanges[x].UpdateTicker(c.Pair, c.AssetType)
											e.tickerBatchLastRequested[exchangeName] = time.Now()
											e.mux.Unlock()
										} else {
											if e.Cfg.Verbose {
												log.Debugf("%s Using recent batching cache", exchangeName)
											}
											result, err = Bot.Exchanges[x].FetchTicker(c.Pair, c.AssetType)
										}
									} else {
										result, err = Bot.Exchanges[x].FetchTicker(c.Pair, c.AssetType)
									}
									printTickerSummary(&result, c.Pair, c.AssetType, exchangeName, err)
									if err == nil {
										//nolint:gocritic Bot.CommsRelayer.StageTickerData(exchangeName, c.AssetType, result)
										if Bot.Config.RemoteControl.WebsocketRPC.Enabled {
											relayWebsocketEvent(result, "ticker_update", c.AssetType.String(), exchangeName)
										}
									}
									e.update(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, err)
								}
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}
					}

					if e.Cfg.SyncOrderbook {
						if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemOrderbook) {
							if c.Orderbook.LastUpdated.IsZero() || time.Since(c.Orderbook.LastUpdated) > defaultSyncerTimeout {
								if c.Orderbook.IsUsingWebsocket {
									if time.Since(c.Created) < defaultSyncerTimeout {
										continue
									}
									if supportsREST {
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
										c.Orderbook.IsUsingWebsocket = false
										c.Orderbook.IsUsingREST = true
										log.Warnf("%s %s: No orderbook update after 15 seconds, switching from websocket to rest",
											c.Exchange, c.Pair.String())
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, false)
									}
								}

								e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
								result, err := Bot.Exchanges[x].UpdateOrderbook(c.Pair, c.AssetType)
								printOrderbookSummary(&result, c.Pair, c.AssetType, exchangeName, err)
								if err == nil {
									//nolint:gocritic Bot.CommsRelayer.StageOrderbookData(exchangeName, c.AssetType, result)
									if Bot.Config.RemoteControl.WebsocketRPC.Enabled {
										relayWebsocketEvent(result, "orderbook_update", c.AssetType.String(), exchangeName)
									}
								}
								e.update(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, err)
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}
						if e.Cfg.SyncTrades {
							if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTrade) {
								if c.Trade.LastUpdated.IsZero() || time.Since(c.Trade.LastUpdated) > defaultSyncerTimeout {
									e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, true)
									e.update(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, nil)
								}
							}
						}
					}
				}
			}
		}
	}

}

// Start starts an exchange currency pair syncer
func (e *ExchangeCurrencyPairSyncer) Start() {
	log.Debugf("Exchange CurrencyPairSyncer started.")

	for x := range Bot.Exchanges {
		if !Bot.Exchanges[x].IsEnabled() {
			continue
		}

		if Bot.Exchanges[x].GetName() == "BTCC" {
			continue
		}

		exchangeName := Bot.Exchanges[x].GetName()
		supportsWebsocket := Bot.Exchanges[x].SupportsWebsocket()
		assetTypes := Bot.Exchanges[x].GetAssetTypes()
		supportsREST := Bot.Exchanges[x].SupportsREST()

		if !supportsREST && !supportsWebsocket {
			log.Warnf("Loaded exchange %s does not support REST or Websocket", exchangeName)
			continue
		}

		var usingWebsocket bool
		var usingREST bool

		if supportsWebsocket {
			ws, err := Bot.Exchanges[x].GetWebsocket()
			if err != nil {
				log.Errorf("%s failed to get websocket. Err: %s", exchangeName, err)
				usingREST = true
			}

			if !ws.IsEnabled() {
				usingREST = true
			}

			if !ws.IsConnected() {
				go WebsocketDataHandler(ws)

				err = ws.Connect()
				if err != nil {
					log.Errorf("%s websocket failed to connect. Err: %s", exchangeName, err)
					usingREST = true
				} else {
					usingWebsocket = true
				}
			}
		} else if supportsREST {
			usingREST = true
		}

		for y := range assetTypes {
			for _, p := range Bot.Exchanges[x].GetEnabledPairs(assetTypes[y]) {
				if e.exists(exchangeName, p, assetTypes[y]) {
					continue
				}
				c := CurrencyPairSyncAgent{
					AssetType: assetTypes[y],
					Exchange:  exchangeName,
					Pair:      p,
				}

				if e.Cfg.SyncTicker {
					c.Ticker = SyncBase{
						IsUsingREST:      usingREST,
						IsUsingWebsocket: usingWebsocket,
					}
				}

				if e.Cfg.SyncOrderbook {
					c.Orderbook = SyncBase{
						IsUsingREST:      usingREST,
						IsUsingWebsocket: usingWebsocket,
					}
				}

				if e.Cfg.SyncTrades {
					c.Trade = SyncBase{
						IsUsingREST:      usingREST,
						IsUsingWebsocket: usingWebsocket,
					}
				}

				e.add(&c)
			}
		}
	}

	if atomic.CompareAndSwapInt32(&e.initSyncStarted, 0, 1) {
		log.Debugln("Exchange CurrencyPairSyncer initial sync started.")
		e.initSyncStartTime = time.Now()
		log.Debugln(createdCounter)
		log.Debugln(removedCounter)
	}

	go func() {
		e.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&e.initSyncCompleted, 0, 1) {
			log.Debugf("Exchange CurrencyPairSyncer initial sync is complete.")
			completedTime := time.Now()
			log.Debugf("Exchange CurrencyPairSyncer initiial sync took %v [%v sync items].", completedTime.Sub(e.initSyncStartTime), createdCounter)

			if !e.Cfg.SyncContinuously {
				log.Debugf("Exchange CurrencyPairSyncer stopping.")
				e.Stop()
				Bot.Stop()
				return
			}
		}
	}()

	if atomic.LoadInt32(&e.initSyncCompleted) == 1 && !e.Cfg.SyncContinuously {
		return
	}

	for i := 0; i < e.Cfg.NumWorkers; i++ {
		go e.worker()
	}
}

// Stop shuts down the exchange currency pair syncer
func (e *ExchangeCurrencyPairSyncer) Stop() {
	stopped := atomic.CompareAndSwapInt32(&e.shutdown, 0, 1)
	if stopped {
		log.Debugf("Exchange CurrencyPairSyncer stopped.")
	}
}
