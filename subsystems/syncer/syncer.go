package syncer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems/apiserver"
)

// const holds the sync item types
const (
	SyncItemTicker = iota
	SyncItemOrderbook
	SyncItemTrade

	DefaultSyncerWorkers = 15
	DefaultSyncerTimeout = time.Second * 15
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
		c.NumWorkers = DefaultSyncerWorkers
	}

	if c.SyncTimeout <= time.Duration(0) {
		c.SyncTimeout = DefaultSyncerTimeout
	}

	s := ExchangeCurrencyPairSyncer{
		Cfg: CurrencyPairSyncerConfig{
			SyncTicker:       c.SyncTicker,
			SyncOrderbook:    c.SyncOrderbook,
			SyncTrades:       c.SyncTrades,
			SyncContinuously: c.SyncContinuously,
			SyncTimeout:      c.SyncTimeout,
			NumWorkers:       c.NumWorkers,
		},
	}

	s.tickerBatchLastRequested = make(map[string]time.Time)

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout: %v\n",
		s.Cfg.SyncContinuously, s.Cfg.SyncTicker, s.Cfg.SyncOrderbook,
		s.Cfg.SyncTrades, s.Cfg.NumWorkers, s.Cfg.Verbose, s.Cfg.SyncTimeout)
	return &s, nil
}

func (e *ExchangeCurrencyPairSyncer) get(exchangeName string, p currency.Pair, a asset.Item) (*CurrencyPairSyncAgent, error) {
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

func (e *ExchangeCurrencyPairSyncer) exists(exchangeName string, p currency.Pair, a asset.Item) bool {
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
		if e.Cfg.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added ticker sync item %v: using websocket: %v using REST: %v\n",
				c.Exchange, engine.Bot.FormatCurrency(c.Pair).String(), c.Ticker.IsUsingWebsocket,
				c.Ticker.IsUsingREST)
		}
		if atomic.LoadInt32(&e.initSyncCompleted) != 1 {
			e.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if e.Cfg.SyncOrderbook {
		if e.Cfg.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added orderbook sync item %v: using websocket: %v using REST: %v\n",
				c.Exchange, engine.Bot.FormatCurrency(c.Pair).String(), c.Orderbook.IsUsingWebsocket,
				c.Orderbook.IsUsingREST)
		}
		if atomic.LoadInt32(&e.initSyncCompleted) != 1 {
			e.initSyncWG.Add(1)
			createdCounter++
		}
	}

	if e.Cfg.SyncTrades {
		if e.Cfg.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added trade sync item %v: using websocket: %v using REST: %v\n",
				c.Exchange, engine.Bot.FormatCurrency(c.Pair).String(), c.Trade.IsUsingWebsocket,
				c.Trade.IsUsingREST)
		}
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

func (e *ExchangeCurrencyPairSyncer) isProcessing(exchangeName string, p currency.Pair, a asset.Item, syncType int) bool {
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

func (e *ExchangeCurrencyPairSyncer) setProcessing(exchangeName string, p currency.Pair, a asset.Item, syncType int, processing bool) {
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

// Update notifies the ExchangeCurrencyPairSyncer to change the last updated time for a exchange asset pair
func (e *ExchangeCurrencyPairSyncer) Update(exchangeName string, p currency.Pair, a asset.Item, syncType int, err error) {
	if atomic.LoadInt32(&e.initSyncStarted) != 1 {
		return
	}

	switch syncType {
	case SyncItemOrderbook:
		if !e.Cfg.SyncOrderbook {
			return
		}
	case SyncItemTicker:
		if !e.Cfg.SyncTicker {
			return
		}
	case SyncItemTrade:
		if !e.Cfg.SyncTrades {
			return
		}
	default:
		log.Warnf(log.SyncMgr, "ExchangeCurrencyPairSyncer: unknown sync item %v\n", syncType)
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
					removedCounter++
					log.Debugf(log.SyncMgr, "%s ticker sync complete %v [%d/%d].\n",
						exchangeName,
						engine.Bot.FormatCurrency(p).String(),
						removedCounter,
						createdCounter)
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
					removedCounter++
					log.Debugf(log.SyncMgr, "%s orderbook sync complete %v [%d/%d].\n",
						exchangeName,
						engine.Bot.FormatCurrency(p).String(),
						removedCounter,
						createdCounter)
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
					removedCounter++
					log.Debugf(log.SyncMgr, "%s trade sync complete %v [%d/%d].\n",
						exchangeName,
						engine.Bot.FormatCurrency(p).String(),
						removedCounter,
						createdCounter)
					e.initSyncWG.Done()
				}
			}
		}
	}
}

func (e *ExchangeCurrencyPairSyncer) worker() {
	cleanup := func() {
		log.Debugln(log.SyncMgr,
			"Exchange CurrencyPairSyncer worker shutting down.")
	}
	defer cleanup()

	for atomic.LoadInt32(&e.shutdown) != 1 {
		exchanges := engine.Bot.GetExchanges()
		for x := range exchanges {
			exchangeName := exchanges[x].GetName()
			assetTypes := exchanges[x].GetAssetTypes()
			supportsREST := exchanges[x].SupportsREST()
			supportsRESTTickerBatching := exchanges[x].SupportsRESTTickerBatchUpdates()
			var usingREST bool
			var usingWebsocket bool
			var switchedToRest bool
			if exchanges[x].SupportsWebsocket() && exchanges[x].IsWebsocketEnabled() {
				ws, err := exchanges[x].GetWebsocket()
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s unable to get websocket pointer. Err: %s\n",
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

			for y := range assetTypes {
				if exchanges[x].GetBase().CurrencyPairs.IsAssetEnabled(assetTypes[y]) != nil {
					continue
				}
				enabledPairs, err := exchanges[x].GetEnabledPairs(assetTypes[y])
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s failed to get enabled pairs. Err: %s\n",
						exchangeName,
						err)
					continue
				}
				for i := range enabledPairs {
					if atomic.LoadInt32(&e.shutdown) == 1 {
						return
					}

					if !e.exists(exchangeName, enabledPairs[i], assetTypes[y]) {
						c := CurrencyPairSyncAgent{
							AssetType: assetTypes[y],
							Exchange:  exchangeName,
							Pair:      enabledPairs[i],
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

					c, err := e.get(exchangeName, enabledPairs[i], assetTypes[y])
					if err != nil {
						log.Errorf(log.SyncMgr, "failed to get item. Err: %s\n", err)
						continue
					}
					if switchedToRest && usingWebsocket {
						log.Warnf(log.SyncMgr,
							"%s %s: Websocket re-enabled, switching from rest to websocket\n",
							c.Exchange, engine.Bot.FormatCurrency(enabledPairs[i]).String())
						switchedToRest = false
					}
					if e.Cfg.SyncTicker {
						if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTicker) {
							if c.Ticker.LastUpdated.IsZero() || time.Since(c.Ticker.LastUpdated) > e.Cfg.SyncTimeout {
								if c.Ticker.IsUsingWebsocket {
									if time.Since(c.Created) < e.Cfg.SyncTimeout {
										continue
									}

									if supportsREST {
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
										c.Ticker.IsUsingWebsocket = false
										c.Ticker.IsUsingREST = true
										log.Warnf(log.SyncMgr,
											"%s %s %s: No ticker update after %s, switching from websocket to rest\n",
											c.Exchange,
											engine.Bot.FormatCurrency(enabledPairs[i]).String(),
											strings.ToUpper(c.AssetType.String()),
											e.Cfg.SyncTimeout,
										)
										switchedToRest = true
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, false)
									}
								}

								if c.Ticker.IsUsingREST {
									e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
									var result *ticker.Price
									var err error

									if supportsRESTTickerBatching {
										e.mux.Lock()
										batchLastDone, ok := e.tickerBatchLastRequested[exchangeName]
										if !ok {
											e.tickerBatchLastRequested[exchangeName] = time.Time{}
										}
										e.mux.Unlock()

										if batchLastDone.IsZero() || time.Since(batchLastDone) > e.Cfg.SyncTimeout {
											e.mux.Lock()
											if e.Cfg.Verbose {
												log.Debugf(log.SyncMgr, "%s Init'ing REST ticker batching\n", exchangeName)
											}
											result, err = exchanges[x].UpdateTicker(c.Pair, c.AssetType)
											e.tickerBatchLastRequested[exchangeName] = time.Now()
											e.mux.Unlock()
										} else {
											if e.Cfg.Verbose {
												log.Debugf(log.SyncMgr, "%s Using recent batching cache\n", exchangeName)
											}
											result, err = exchanges[x].FetchTicker(c.Pair, c.AssetType)
										}
									} else {
										result, err = exchanges[x].UpdateTicker(c.Pair, c.AssetType)
									}
									PrintTickerSummary(result, "REST", err)
									if err == nil {
										if engine.Bot.Config.RemoteControl.WebsocketRPC.Enabled {
											relayWebsocketEvent(result, "ticker_update", c.AssetType.String(), exchangeName)
										}
									}
									e.Update(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, err)
								}
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}
					}

					if e.Cfg.SyncOrderbook {
						if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemOrderbook) {
							if c.Orderbook.LastUpdated.IsZero() || time.Since(c.Orderbook.LastUpdated) > e.Cfg.SyncTimeout {
								if c.Orderbook.IsUsingWebsocket {
									if time.Since(c.Created) < e.Cfg.SyncTimeout {
										continue
									}
									if supportsREST {
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
										c.Orderbook.IsUsingWebsocket = false
										c.Orderbook.IsUsingREST = true
										log.Warnf(log.SyncMgr,
											"%s %s %s: No orderbook update after %s, switching from websocket to rest\n",
											c.Exchange,
											engine.Bot.FormatCurrency(c.Pair).String(),
											strings.ToUpper(c.AssetType.String()),
											e.Cfg.SyncTimeout,
										)
										switchedToRest = true
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, false)
									}
								}

								e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
								result, err := exchanges[x].UpdateOrderbook(c.Pair, c.AssetType)
								PrintOrderbookSummary(result, "REST", engine.Bot, err)
								if err == nil {
									if engine.Bot.Config.RemoteControl.WebsocketRPC.Enabled {
										relayWebsocketEvent(result, "orderbook_update", c.AssetType.String(), exchangeName)
									}
								}
								e.Update(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, err)
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}
						if e.Cfg.SyncTrades {
							if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTrade) {
								if c.Trade.LastUpdated.IsZero() || time.Since(c.Trade.LastUpdated) > e.Cfg.SyncTimeout {
									e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, true)
									e.Update(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, nil)
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
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer started.")
	exchanges := engine.Bot.GetExchanges()
	for x := range exchanges {
		exchangeName := exchanges[x].GetName()
		supportsWebsocket := exchanges[x].SupportsWebsocket()
		assetTypes := exchanges[x].GetAssetTypes()
		supportsREST := exchanges[x].SupportsREST()

		if !supportsREST && !supportsWebsocket {
			log.Warnf(log.SyncMgr,
				"Loaded exchange %s does not support REST or Websocket.\n",
				exchangeName)
			continue
		}

		var usingWebsocket bool
		var usingREST bool
		if supportsWebsocket && exchanges[x].IsWebsocketEnabled() {
			ws, err := exchanges[x].GetWebsocket()
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s failed to get websocket. Err: %s\n",
					exchangeName,
					err)
				usingREST = true
			}

			if !ws.IsConnected() && !ws.IsConnecting() {
				go engine.Bot.WebsocketDataReceiver(ws)

				err = ws.Connect()
				if err == nil {
					err = ws.FlushChannels()
				}
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s websocket failed to connect. Err: %s\n",
						exchangeName,
						err)
					usingREST = true
				} else {
					usingWebsocket = true
				}
			} else {
				usingWebsocket = true
			}
		} else if supportsREST {
			usingREST = true
		}

		for y := range assetTypes {
			if exchanges[x].GetBase().CurrencyPairs.IsAssetEnabled(assetTypes[y]) != nil {
				log.Warnf(log.SyncMgr,
					"%s asset type %s is disabled, fetching enabled pairs is paused",
					exchangeName,
					assetTypes[y])
				continue
			}

			enabledPairs, err := exchanges[x].GetEnabledPairs(assetTypes[y])
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s failed to get enabled pairs. Err: %s\n",
					exchangeName,
					err)
				continue
			}
			for i := range enabledPairs {
				if e.exists(exchangeName, enabledPairs[i], assetTypes[y]) {
					continue
				}

				c := CurrencyPairSyncAgent{
					AssetType: assetTypes[y],
					Exchange:  exchangeName,
					Pair:      enabledPairs[i],
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
		log.Debugf(log.SyncMgr,
			"Exchange CurrencyPairSyncer initial sync started. %d items to process.\n",
			createdCounter)
		e.initSyncStartTime = time.Now()
	}

	go func() {
		e.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&e.initSyncCompleted, 0, 1) {
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync is complete.\n")
			completedTime := time.Now()
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync took %v [%v sync items].\n",
				completedTime.Sub(e.initSyncStartTime), createdCounter)

			if !e.Cfg.SyncContinuously {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
				e.Stop()
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
		log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	}
}

func printCurrencyFormat(price float64, displayCurrency currency.Code) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get display symbol: %s\n", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency currency.Code, origPrice float64, displayCurrency currency.Code) string {
	conv, err := currency.ConvertCurrency(origPrice,
		origCurrency,
		displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to convert currency: %s\n", err)
	}

	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get display symbol: %s\n", err)
	}

	origSymbol, err := currency.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Errorf(log.Global, "Failed to get original currency symbol for %s: %s\n",
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
func PrintTickerSummary(result *ticker.Price, protocol string, err error) {
	if err != nil {
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.Ticker, "Failed to get %s ticker. Error: %s\n",
				protocol,
				err)
			return
		}
		log.Errorf(log.Ticker, "Failed to get %s ticker. Error: %s\n",
			protocol,
			err)
		return
	}

	err = stats.Add(result.ExchangeName, result.Pair, result.AssetType, result.Last, result.Volume)
	if err != nil {
		log.Error(log.SyncMgr, err)
	}
	if result.Pair.Quote.IsFiatCurrency() &&
		engine.Bot != nil &&
		result.Pair.Quote != engine.Bot.Config.Currency.FiatDisplayCurrency {
		origCurrency := result.Pair.Quote.Upper()
		log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
			result.ExchangeName,
			protocol,
			engine.Bot.FormatCurrency(result.Pair),
			strings.ToUpper(result.AssetType.String()),
			printConvertCurrencyFormat(origCurrency, result.Last, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Ask, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Bid, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.High, engine.Bot.Config.Currency.FiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Low, engine.Bot.Config.Currency.FiatDisplayCurrency),
			result.Volume)
	} else {
		if result.Pair.Quote.IsFiatCurrency() &&
			engine.Bot != nil &&
			result.Pair.Quote == engine.Bot.Config.Currency.FiatDisplayCurrency {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f\n",
				result.ExchangeName,
				protocol,
				engine.Bot.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				printCurrencyFormat(result.Last, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.Ask, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.Bid, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.High, engine.Bot.Config.Currency.FiatDisplayCurrency),
				printCurrencyFormat(result.Low, engine.Bot.Config.Currency.FiatDisplayCurrency),
				result.Volume)
		} else {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f\n",
				result.ExchangeName,
				protocol,
				engine.Bot.FormatCurrency(result.Pair),
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

const (
	book = "%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s\n"
)

// PrintOrderbookSummary outputs orderbook results
func PrintOrderbookSummary(result *orderbook.Base, protocol string, bot *engine.Engine, err error) {
	if err != nil {
		if result == nil {
			log.Errorf(log.OrderBook, "Failed to get %s orderbook. Error: %s\n",
				protocol,
				err)
			return
		}
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s\n",
				protocol,
				result.ExchangeName,
				result.Pair,
				result.AssetType,
				err)
			return
		}
		log.Errorf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s\n",
			protocol,
			result.ExchangeName,
			result.Pair,
			result.AssetType,
			err)
		return
	}

	bidsAmount, bidsValue := result.TotalBidsAmount()
	asksAmount, asksValue := result.TotalAsksAmount()

	var bidValueResult, askValueResult string
	switch {
	case result.Pair.Quote.IsFiatCurrency() && bot != nil && result.Pair.Quote != bot.Config.Currency.FiatDisplayCurrency:
		origCurrency := result.Pair.Quote.Upper()
		bidValueResult = printConvertCurrencyFormat(origCurrency, bidsValue, bot.Config.Currency.FiatDisplayCurrency)
		askValueResult = printConvertCurrencyFormat(origCurrency, asksValue, bot.Config.Currency.FiatDisplayCurrency)
	case result.Pair.Quote.IsFiatCurrency() && bot != nil && result.Pair.Quote == bot.Config.Currency.FiatDisplayCurrency:
		bidValueResult = printCurrencyFormat(bidsValue, bot.Config.Currency.FiatDisplayCurrency)
		askValueResult = printCurrencyFormat(asksValue, bot.Config.Currency.FiatDisplayCurrency)
	default:
		bidValueResult = strconv.FormatFloat(bidsValue, 'f', -1, 64)
		askValueResult = strconv.FormatFloat(asksValue, 'f', -1, 64)
	}

	log.Infof(log.OrderBook, book,
		result.ExchangeName,
		protocol,
		bot.FormatCurrency(result.Pair),
		strings.ToUpper(result.AssetType.String()),
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

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	evt := apiserver.WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	}
	err := apiserver.BroadcastWebsocketMessage(evt)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "Failed to broadcast websocket event %v. Error: %s\n",
			event, err)
	}
}
