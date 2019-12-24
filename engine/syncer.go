package engine

import (
	"fmt"
	"sync"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// NewSyncManager gets a new exchange synchronisation manager
func NewSyncManager(cfg SyncConfig) (*SyncManager, error) {
	if !cfg.Orderbook && !cfg.Ticker && !cfg.Trades {
		return nil, ErrInvalidItems
	}

	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = DefaultSyncerWorkers
	}

	if cfg.SyncTimeout <= time.Duration(0) {
		cfg.SyncTimeout = DefaultSyncerTimeout
	}

	s := SyncManager{
		SyncConfig: cfg,
		shutdown:   make(chan struct{}),
		synchro:    make(chan struct{}),
		pipe:       make(chan SyncUpdate),
		syncComm:   make(chan time.Time),
	}

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout: %v\n",
		s.SyncConfig.Continuous,
		s.SyncConfig.Ticker,
		s.SyncConfig.Orderbook,
		s.SyncConfig.Trades,
		s.SyncConfig.NumWorkers,
		s.SyncConfig.Verbose,
		s.SyncConfig.SyncTimeout)
	return &s, nil
}

// Start starts an exchange currency pair syncer
func (e *SyncManager) Start() {
	for x := range Bot.Exchanges {
		if !Bot.Exchanges[x].IsEnabled() {
			continue
		}

		// Disables rate limiter by request system so it can be handled by
		// engine sub systems
		Bot.Exchanges[x].GetBase().DisableRateLimit()

		assetTypes := Bot.Exchanges[x].GetAssetTypes()

		if !Bot.Exchanges[x].SupportsREST() && !Bot.Exchanges[x].SupportsWebsocket() {
			log.Warnf(log.SyncMgr,
				"Loaded exchange %s does not support REST or Websocket.\n",
				Bot.Exchanges[x].GetName())
			continue
		}

		var protocol string = syncProtocolREST
		if Bot.Exchanges[x].IsWebsocketEnabled() {
			ws, err := Bot.Exchanges[x].GetWebsocket()
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s failed to get websocket. Err: %s\n",
					Bot.Exchanges[x].GetName(),
					err)
			}

			if !ws.IsConnected() && !ws.IsConnecting() {
				go WebsocketDataHandler(ws)

				err = ws.Connect()
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s websocket failed to connect. Err: %s\n",
						Bot.Exchanges[x].GetName(), err)
				} else {
					protocol = syncProtocolWebsocket
				}
			} else {
				protocol = syncProtocolWebsocket
			}
		}

		var exchangeSyncItems int
		var wg sync.WaitGroup
		auth := Bot.Exchanges[x].GetBase().API.AuthenticatedSupport
		if auth {
			// TODO: Tie in with account config settings
			e.Agents = append(e.Agents, &AccountAgent{
				Exchange: Bot.Exchanges[x],
				Protocol: protocol,
				Pipe:     e.pipe,
				Wg:       &wg,
			})
			exchangeSyncItems++
			// TODO: Tie in with account fee settings
			e.Agents = append(e.Agents, &FeeAgent{
				Exchange: Bot.Exchanges[x],
				Protocol: protocol,
				Pipe:     e.pipe,
				Wg:       &wg,
			})
			exchangeSyncItems++
			// TODO: Tie in with order config settings
			e.Agents = append(e.Agents, &OrderAgent{
				Exchange: Bot.Exchanges[x],
				Protocol: protocol,
				Pipe:     e.pipe,
				Wg:       &wg,
			})
			exchangeSyncItems++
		}

		// TODO: Add Historic trades for an exchange and items to configuration

		// e.Agents = append(e.Agents, &ExchangeTradeHistoryAgent{
		// 	Exchange: Bot.Exchanges[x],
		// 	Protocol: protocol,
		// 	Pipe:     e.pipe,
		// 	Wg:       &wg,
		// })

		// e.Agents = append(e.Agents, &SupportedPairsAgent{
		// 	Exchange: Bot.Exchanges[x],
		// 	Protocol: protocol,
		// 	Pipe:     e.pipe,
		// 	Wg:       &wg,
		// })

		for y := range assetTypes {
			enabledPairs := Bot.Exchanges[x].GetEnabledPairs(assetTypes[y])
			for z := range enabledPairs {
				if e.SyncConfig.Ticker {
					e.Agents = append(e.Agents, &TickerAgent{
						AssetType: assetTypes[y],
						Exchange:  Bot.Exchanges[x],
						Pair:      enabledPairs[z],
						Protocol:  protocol,
						Pipe:      e.pipe,
						Wg:        &wg,
						CancelMe:  make(chan int),
					})
					exchangeSyncItems++
				}

				if e.SyncConfig.Orderbook {
					e.Agents = append(e.Agents, &OrderbookAgent{
						AssetType: assetTypes[y],
						Exchange:  Bot.Exchanges[x],
						Pair:      enabledPairs[z],
						Protocol:  protocol,
						Pipe:      e.pipe,
						Wg:        &wg,
						CancelMe:  make(chan int),
					})
					exchangeSyncItems++
				}

				if e.SyncConfig.Trades {
					e.Agents = append(e.Agents, &TradeAgent{
						AssetType: assetTypes[y],
						Exchange:  Bot.Exchanges[x],
						Pair:      enabledPairs[z],
						Protocol:  protocol,
						Pipe:      e.pipe,
						Wg:        &wg,
						CancelMe:  make(chan int),
					})
					exchangeSyncItems++
				}
			}
		}

		go func(exch string, count int, wg *sync.WaitGroup) {
			wg.Add(count)
			log.Debugf(log.SyncMgr, "Initial sync started for %s with [%d] ---------------------------- \n", exch, count)
			start := time.Now()
			wg.Wait()
			end := time.Now()
			log.Debugf(log.SyncMgr, "Initial sync finished for %s in [%s] ----------------------------- \n", exch, end.Sub(start))
		}(Bot.Exchanges[x].GetName(), exchangeSyncItems, &wg)
	}

	go e.Monitor()
	go e.Worker()
	go e.Processor()
}

// Stop shuts down the exchange currency pair syncer
func (e *SyncManager) Stop() {
	// stopped := atomic.CompareAndSwapInt32(&e.shutdown, 0, 1)
	// if stopped {
	// 	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	// }
}

// Worker iterates across the full agent list and initiates an update by
// detatching in a routine which directly interacts with a work manager.
// The routine generated *should* block until an update has occured then
// it will interact with the processor routine keeping a record of last updated
// and when a new update should occur to keep it in optimal synchronisation on
// the REST protocol.
func (e *SyncManager) Worker() {
	go func() { e.synchro <- struct{}{} }() // Pre-load switch
	for {
		select {
		case <-e.shutdown:
			log.Debugln(log.SyncMgr, "Sync manager worker shutting down.")
			return
		case <-e.synchro: // Synchro chan is switch by the Monitor routine
			// This sleep variable will keep track on the next update time on
			// the agent list, allowing the routine to sleep reducing CPU
			// contention.
			var sleep time.Time
			for i := range e.Agents {
				e.Lock() // TODO: re-structure locks
				nextUpdate := e.Agents[i].GetNextUpdate()
				if nextUpdate.Before(time.Now()) || nextUpdate == (time.Time{}) {
					if !e.Agents[i].IsProcessing() {
						e.Agents[i].SetProcessing(true)
						go func(s Synchroniser) {
							// TODO: Add cancellation on work stack when
							// websocket kicks in
							s.Execute()
						}(e.Agents[i])
					}
				} else {
					if nextUpdate.Before(sleep) || sleep == (time.Time{}) {
						sleep = nextUpdate
					}
				}
				e.Unlock()
			}
			if sleep != (time.Time{}) {
				// Passes the sleep time to the Monitor routine which interdicts
				// a wake up alert to the Worker routine if time < current
				// update time.
				e.syncComm <- sleep
			}
		}
	}
}

// Monitor monitors the last updated to initiate an agent check for the worker
func (e *SyncManager) Monitor() {
	var sleep time.Time
	c := make(chan struct{}) // Main communications within this routine
	i := make(chan struct{}) // Interdiction channel to disregard update

	var wg sync.WaitGroup
	for {
		select {
		case <-e.shutdown:
			log.Debugln(log.SyncMgr, "Sync manager monitor shutting down.")
			return
		case <-c:
			e.synchro <- struct{}{} // Wakes up worker routine
			sleep = time.Time{}     // Reset sleep time, so new update's next
			// time to update will define Worker's sleep rate
		case t := <-e.syncComm:
			if t.Before(sleep) || sleep == (time.Time{}) {
				close(i) // Stop routines from broadcasting
				wg.Wait()
				i = make(chan struct{}) // Reinstate interdiction
				sleep = t
				wg.Add(1)
				// Detach and allow for interdiction
				go func(s time.Duration, wg *sync.WaitGroup) {
					ch := make(chan struct{})
					go func(ch chan struct{}) {
						time.Sleep(s)
						select {
						case ch <- struct{}{}:
						default:
						}
					}(ch)
					select {
					case <-i:
						// Bypass this update
					case <-ch:
						c <- struct{}{}
					}
					wg.Done()
				}(t.Sub(time.Now()), &wg)
			}
		}
	}
}

// SyncUpdate wraps updates for concurrent processing
type SyncUpdate struct {
	Agent    Synchroniser
	Payload  interface{}
	Procotol string
	Err      error
}

// Processor processing all concurrent sync agent updates
func (e *SyncManager) Processor() {
	for {
		select {
		case <-e.shutdown:
			return
		case u := <-e.pipe:
			if u.Err != nil {
				fmt.Println("error occured", u.Err)
				continue
			}
			e.Lock() // TODO - Possibly have a mutex to the agent instead
			lastUpdated := u.Agent.GetLastUpdated()
			if lastUpdated != (time.Time{}) {
				if Bot.Settings.Verbose {
					log.Debugf(log.SyncMgr, "Agent last updated %s ago",
						time.Now().Sub(lastUpdated))
				}
			} else {
				u.Agent.InitialSyncComplete()
			}
			u.Agent.SetLastUpdated(time.Now())
			u.Agent.SetNextUpdate(u.Agent.GetLastUpdated().Add(DefaultSyncerTimeout))
			// TODO: Probably RM this field
			if u.Agent.IsProcessing() &&
				u.Agent.IsUsingProtocol(syncProtocolREST) &&
				u.Procotol != syncProtocolREST {
				// Processing in REST, cancel request
				fmt.Println("Cancelling process", u.Procotol)
				u.Agent.Cancel()
			}
			u.Agent.SetUsingProtocol(u.Procotol)
			u.Agent.SetProcessing(false)

			e.syncComm <- u.Agent.GetNextUpdate() // Send items next update time
			// to the Monitor routine for check
			e.Unlock()
			switch p := u.Payload.(type) {
			case *ticker.Price:
				printTickerSummary(p,
					p.Pair,
					p.AssetType,
					p.ExchangeName,
					u.Procotol,
					u.Err)
			case *orderbook.Base:
				printOrderbookSummary(p,
					p.Pair,
					p.AssetType,
					p.ExchangeName,
					u.Procotol,
					u.Err)
				// TODO: Add all items to the list
			case exchange.AccountInfo:
				fmt.Println("YAY! Account info!")
			case float64: // fee???
				fmt.Println("YAY! Fee updated!")
			case exchange.TradeHistory:
				fmt.Println("YAY! Trade History updated!")
			case []order.Detail:
				fmt.Println("YAY! Open Orders updated!")
			default:
				panic(p) // TODO: Rework this
			}
		}
	}
}

// StreamUpdate passes in a stream object TODO: Need to find a more efficient
// way of matching agent
func (e *SyncManager) StreamUpdate(payload interface{}) {
	for i := range e.Agents {
		if agent := e.Agents[i].Stream(payload); agent != nil {
			go func(s Synchroniser, payload interface{}) {
				e.pipe <- SyncUpdate{
					Agent:    agent,
					Payload:  payload,
					Procotol: syncProtocolWebsocket,
				}
			}(agent, payload)
		}
	}
}
