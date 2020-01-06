package engine

import (
	"fmt"
	"reflect"
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
	if cfg == (SyncConfig{}) {
		return nil, ErrInvalidItems
	}

	return &SyncManager{
		SyncConfig: cfg,
		shutdown:   make(chan struct{}),
		synchro:    make(chan struct{}),
		pipe:       make(chan SyncUpdate),
		syncComm:   make(chan time.Time),
	}, nil
}

// Start starts an exchange currency pair syncer
// TODO: Add in time to update for each agent type
func (e *SyncManager) Start() {
	for x := range Bot.Exchanges {
		if !Bot.Exchanges[x].IsEnabled() {
			continue
		}

		// Disables rate limiter by request system so it can be handled by
		// engine sub systems
		Bot.Exchanges[x].GetBase().DisableRateLimit()

		if !Bot.Exchanges[x].SupportsREST() &&
			!Bot.Exchanges[x].SupportsWebsocket() {
			log.Warnf(log.SyncMgr,
				"Loaded exchange %s does not support REST or Websocket.\n",
				Bot.Exchanges[x].GetName())
			continue
		}

		// Set up initial websocket connection
		if Bot.Exchanges[x].IsWebsocketEnabled() {
			ws, err := Bot.Exchanges[x].GetWebsocket()
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s failed to get websocket. Err: %s\n",
					Bot.Exchanges[x].GetName(),
					err)
			} else if !ws.IsConnected() && !ws.IsConnecting() {
				go WebsocketDataHandler(ws)
				err = ws.Connect()
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s websocket failed to connect. Err: %s\n",
						Bot.Exchanges[x].GetName(), err)
				}
			}
		}

		// Initial synchronisation count and waitgroup
		var exchangeSyncItems int
		var wg sync.WaitGroup
		if e.ExchangeDepositAddresses {
			// TODO:
			// for _, cur := range Bot.Exchanges[x].GetBase().BaseCurrencies {
			// 	if cur.IsFiatCurrency() {
			// 		continue
			// 	}

			// 	e.Agents = append(e.Agents, &DepositAddressAgent{
			// 		Exchange: Bot.Exchanges[x],
			// 		Currency: cur,
			// 		Pipe:     e.pipe,
			// 		Wg:       &wg,
			// 		CancelMe: make(chan int),
			// 	})
			// 	exchangeSyncItems++
			// }
		}

		if Bot.Exchanges[x].GetBase().API.AuthenticatedSupport {
			// Fetches account balance for the exchange
			if e.AccountBalance {
				e.Agents = append(e.Agents, &AccountAgent{
					Agent: Agent{
						Exchange: Bot.Exchanges[x],
						Pipe:     e.pipe,
						Wg:       &wg,
						CancelMe: make(chan int),
					},
				})
				exchangeSyncItems++
			}

			if e.AccountFees {
				// Fetches updated fee structure
				// TODO: REDO current fee retrieval system, package it
				// e.Agents = append(e.Agents, &FeeAgent{
				// 	Exchange: Bot.Exchanges[x],
				// 	Pipe:     e.pipe,
				// 	Wg:       &wg,
				// 	CancelMe: make(chan int),
				// })
				// exchangeSyncItems++
			}

			if e.AccountOrders {
				// Fetches account active orders
				e.Agents = append(e.Agents, &OrderAgent{
					Agent: Agent{
						Exchange: Bot.Exchanges[x],
						Pipe:     e.pipe,
						Wg:       &wg,
						CancelMe: make(chan int),
					},
				})
				exchangeSyncItems++
			}

			if e.AccountFunding {
				// TODO:
				// Fetches funding status and alerts - will be used mostly from
				// streaming
				// e.Agents = append(e.Agents, &FundingAgent{
				// 	Exchange: Bot.Exchanges[x],
				// 	Pipe:     e.pipe,
				// 	Wg:       &wg,
				// 	CancelMe: make(chan int),
				// })
				// exchangeSyncItems++
			}

			if e.AccountPosition {
				// TODO:
				// Fetches position status and alerts - will be used mostly from
				// streaming
				// e.Agents = append(e.Agents, &PositionAgent{
				// 	Exchange: Bot.Exchanges[x],
				// 	Pipe:     e.pipe,
				// 	Wg:       &wg,
				// 	CancelMe: make(chan int),
				// })
				// exchangeSyncItems++
			}
		}

		if e.ExchangeTradeHistory {
			e.Agents = append(e.Agents, &ExchangeTradeHistoryAgent{
				Agent: Agent{
					Exchange: Bot.Exchanges[x],
					Pipe:     e.pipe,
					Wg:       &wg,
					CancelMe: make(chan int),
				},
			})
			exchangeSyncItems++
		}

		if e.ExchangeSupportedPairs {
			// Periodically checks supported pairs list for a persistant bot
			// instance
			e.Agents = append(e.Agents, &SupportedPairsAgent{
				Agent: Agent{
					Exchange: Bot.Exchanges[x],
					Pipe:     e.pipe,
					Wg:       &wg,
					CancelMe: make(chan int),
				},
			})
		}

		assetTypes := Bot.Exchanges[x].GetAssetTypes()
		for y := range assetTypes {
			enabledPairs := Bot.Exchanges[x].GetEnabledPairs(assetTypes[y])
			for z := range enabledPairs {
				if e.ExchangeTicker {
					e.Agents = append(e.Agents, &TickerAgent{
						AssetType: assetTypes[y],
						Pair:      enabledPairs[z],
						Agent: Agent{
							Exchange: Bot.Exchanges[x],
							Pipe:     e.pipe,
							Wg:       &wg,
							CancelMe: make(chan int),
						},
					})
					exchangeSyncItems++
				}

				if e.ExchangeOrderbook {
					e.Agents = append(e.Agents, &OrderbookAgent{
						AssetType: assetTypes[y],
						Pair:      enabledPairs[z],
						Agent: Agent{
							Exchange: Bot.Exchanges[x],
							Pipe:     e.pipe,
							Wg:       &wg,
							CancelMe: make(chan int),
						},
					})
					exchangeSyncItems++
				}

				if e.ExchangeTrades {
					e.Agents = append(e.Agents, &TradeAgent{
						AssetType: assetTypes[y],
						Pair:      enabledPairs[z],
						Agent: Agent{
							Exchange: Bot.Exchanges[x],
							Pipe:     e.pipe,
							Wg:       &wg,
							CancelMe: make(chan int),
						},
					})
					exchangeSyncItems++
				}

				if e.ExchangeKline {
					e.Agents = append(e.Agents, &KlineAgent{
						AssetType: assetTypes[y],
						Pair:      enabledPairs[z],
						Agent: Agent{
							Exchange: Bot.Exchanges[x],
							Pipe:     e.pipe,
							Wg:       &wg,
							CancelMe: make(chan int),
						},
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

// Processor processing all concurrent sync agent updates
func (e *SyncManager) Processor() {
	for {
		select {
		case <-e.shutdown:
			return
		case u := <-e.pipe:
			if u.Err == nil {
				e.Lock()
				// Check for last updated
				lastUpdated := u.Agent.GetLastUpdated()
				if lastUpdated != (time.Time{}) {
					if Bot.Settings.Verbose {
						log.Debugf(log.SyncMgr, "Agent last updated %s ago",
							time.Now().Sub(lastUpdated))
					}
				} else {
					// Set initial sync complete on agent
					u.Agent.InitialSyncComplete()
				}

				if u.Agent.IsProcessing() && u.Protocol != REST {
					// Cancel on job stack if update from stream connection
					// comes through
					u.Agent.Cancel()
				}

				// Sets new update time
				u.Agent.SetNewUpdate()
				u.Agent.SetProcessing(false)

				// Send items next update time to the Monitor routine for check
				e.syncComm <- u.Agent.GetNextUpdate()
				e.Unlock()
			}

			// --- This section can be deprecated in production
			switch p := u.Payload.(type) {
			case *ticker.Price:
				printTickerSummary(p, u.Protocol, u.Err)
			case *orderbook.Base:
				printOrderbookSummary(p, u.Protocol, u.Err)
			case *exchange.AccountInfo:
				printAccountSummary(p, u.Protocol)
			case []order.Trade:
				printTradeSummary(p, u.Protocol)
			case float64: // fee???
				fmt.Println("YAY! Fee updated!")
			case exchange.TradeHistory:
				fmt.Println("YAY! Trade History updated!")
			case []order.Detail:
				printOrderSummary(p, u.Protocol)
			default:
				panic(p) // TODO: Rework this
			}
			// ---
		}
	}
}

// StreamUpdate passes in a stream object
// TODO: Need to find a more efficient way of matching agent
func (e *SyncManager) StreamUpdate(payload interface{}) {
	// Default subscriptions for websocket streaming, this switch will stop
	// interaction with the sync agent [temp]
	switch payload.(type) {
	case *ticker.Price:
		if !e.ExchangeTicker {
			return
		}
	case *orderbook.Base:
		if !e.ExchangeOrderbook {
			return
		}
	case []order.Detail:
		// if !e.AccountOrders {
		// 	return
		// }
	case []order.Trade:
		if !e.ExchangeTrades {
			return
		}
	default:
		fmt.Println("Cannot match to agent", reflect.TypeOf(payload))
		return
	}

	for i := range e.Agents {
		if agent := e.Agents[i].Stream(payload); agent != nil {
			go func(s Synchroniser, payload interface{}) {
				e.pipe <- SyncUpdate{
					Agent:    agent,
					Payload:  payload,
					Protocol: Websocket,
				}
			}(agent, payload)
			return
		}
	}
	fmt.Println("Cannot match to agent", reflect.TypeOf(payload))
}
