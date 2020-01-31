package engine

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
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

// Started returns if gctscript manager subsystem is started
func (e *SyncManager) Started() bool {
	return atomic.LoadInt32(&e.started) == 1
}

// Start starts the synchronisation manager
func (e *SyncManager) Start() error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return errors.New("synchronisation manager already started")
	}

	Bot.ServicesWG.Add(1)

	go func() {
		exchanges := GetExchanges()
		for x := range exchanges {
			if !exchanges[x].IsEnabled() {
				continue
			}

			if !exchanges[x].SupportsREST() &&
				!exchanges[x].SupportsWebsocket() {
				log.Warnf(log.SyncMgr,
					"Loaded exchange %s does not support REST or Websocket.\n",
					exchanges[x].GetName())
				continue
			}

			// Initial synchronisation count and waitgroup
			var exchangeSyncItems int
			var wg sync.WaitGroup

			auth := exchanges[x].GetBase().API.AuthenticatedSupport

			if !auth && (e.AccountBalance || e.AccountOrders) {
				log.Warnf(log.SyncMgr,
					"Loaded exchange %s cannot sync account specific items as functionality is disabled.\n",
					exchanges[x].GetName())
			}

			if auth {
				// Fetches account balance for the exchange
				if e.AccountBalance {
					e.Agents = append(e.Agents, &AccountBalanceAgent{
						Agent: Agent{
							Name:            "AccountBalanceAgent",
							Exchange:        exchanges[x],
							RestUpdateDelay: defaultSyncDelay,
							Pipe:            e.pipe,
							Wg:              &wg,
						},
					})
					exchangeSyncItems++
				}
			}

			if e.ExchangeSupportedPairs {
				// Periodically checks supported pairs list for a persistant bot
				// instance
				e.Agents = append(e.Agents, &SupportedPairsAgent{
					Agent: Agent{
						Name:            "ExchangeSupportedPairsAgent",
						Exchange:        exchanges[x],
						RestUpdateDelay: defaultExchangeSupportedPairDelay,
						NextUpdate:      time.Now().Add(defaultExchangeSupportedPairDelay),
						Pipe:            e.pipe,
						Wg:              &wg,
					},
				})
			}

			assetTypes := exchanges[x].GetAssetTypes()
			for y := range assetTypes {
				enabledPairs := exchanges[x].GetEnabledPairs(assetTypes[y])
				for z := range enabledPairs {
					if e.ExchangeTicker {
						e.Agents = append(e.Agents, &TickerAgent{
							AssetType: assetTypes[y],
							Pair:      enabledPairs[z],
							Agent: Agent{
								Name: fmt.Sprintf("TickerAgent %s %s",
									enabledPairs[z],
									assetTypes[y]),
								RestUpdateDelay: defaultSyncDelay,
								Exchange:        exchanges[x],
								Pipe:            e.pipe,
								Wg:              &wg,
							},
						})
						exchangeSyncItems++
					}

					if e.ExchangeOrderbook {
						e.Agents = append(e.Agents, &OrderbookAgent{
							AssetType: assetTypes[y],
							Pair:      enabledPairs[z],
							Agent: Agent{
								Name: fmt.Sprintf("OrderbookAgent %s %s",
									enabledPairs[z],
									assetTypes[y]),
								RestUpdateDelay: defaultSyncDelay,
								Exchange:        exchanges[x],
								Pipe:            e.pipe,
								Wg:              &wg,
							},
						})
						exchangeSyncItems++
					}

					if e.ExchangeTrades {
						e.Agents = append(e.Agents, &TradeAgent{
							AssetType: assetTypes[y],
							Pair:      enabledPairs[z],
							Agent: Agent{
								Name: fmt.Sprintf("TradeAgent %s %s",
									enabledPairs[z],
									assetTypes[y]),
								RestUpdateDelay: defaultSyncDelay,
								Exchange:        exchanges[x],
								Pipe:            e.pipe,
								Wg:              &wg,
							},
						})
						exchangeSyncItems++
					}

					if e.AccountOrders && auth {
						// Fetches account active orders
						e.Agents = append(e.Agents, &OrderAgent{
							Pair:  enabledPairs[z],
							Asset: assetTypes[y],
							Agent: Agent{
								Name:            "AccountOrderAgent",
								Exchange:        exchanges[x],
								RestUpdateDelay: defaultSyncDelay,
								Pipe:            e.pipe,
								Wg:              &wg,
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
			}(exchanges[x].GetName(), exchangeSyncItems, &wg)
		}

		e.wg.Add(3)
		go e.Monitor()
		go e.Worker()
		go e.Processor()
	}()
	return nil
}

// Stop shuts down the exchange currency pair syncer
func (e *SyncManager) Stop() error {
	if atomic.LoadInt32(&e.started) == 0 {
		return errors.New("synchronisation manager has not started")
	}

	if atomic.AddInt32(&e.stopped, 1) != 1 {
		return errors.New("synchronisation manager has already stopped")
	}

	log.Debugln(log.SyncMgr, "Synchronisation manager shutting down...")
	close(e.shutdown)
	e.wg.Wait()
	atomic.CompareAndSwapInt32(&e.stopped, 1, 0)
	atomic.CompareAndSwapInt32(&e.started, 1, 0)
	Bot.ServicesWG.Done()
	log.Debugln(log.SyncMgr, "Synchronisation manager has shutdown.")
	return nil
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
			e.wg.Done()
			return
		case <-e.synchro: // Synchro chan is switched by the Monitor routine
			// This sleep variable will keep track on the next update time on
			// the agent list, allowing the routine to sleep reducing CPU
			// contention.
			if Bot.Settings.Verbose {
				log.Debugln(log.SyncMgr,
					"Synchronisation worker has fired -- Checking agent list for updates...")
			}
			var sleep time.Time
			e.Lock() // TODO: re-structure locks
			for i := range e.Agents {
				if e.Agents[i].IsRESTDisabled() {
					continue
				}
				nextUpdate := e.Agents[i].GetNextUpdate()
				if nextUpdate.Before(time.Now()) || nextUpdate == (time.Time{}) {
					if !e.Agents[i].IsProcessing() {
						e.Agents[i].SetProcessing(true)
						go func(s Synchroniser) {
							s.Execute()
						}(e.Agents[i])
					}
				} else {
					if nextUpdate.Before(sleep) || sleep == (time.Time{}) {
						sleep = nextUpdate
					}
				}
			}
			e.Unlock()

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
			e.wg.Done()
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
			e.wg.Done()
			return
		case u := <-e.pipe:
			e.Lock()
			if u.Err == nil {
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

				// Sets new update time
				u.Agent.SetNewUpdate()
				u.Agent.SetProcessing(false)

				// Send items next update time to the Monitor routine for check
				e.syncComm <- u.Agent.GetNextUpdate()

				switch p := u.Payload.(type) {
				case *ticker.Price:
					printTickerSummary(p, u.Protocol, u.Err)
				case *orderbook.Base:
					printOrderbookSummary(p, u.Protocol, u.Err)
				case *account.Holdings:
					printAccountSummary(p, u.Protocol)
				case []order.TradeHistory:
					printTradeSummary(p, u.Protocol)
				case *kline.Item:
					printKlineSummary(p, u.Protocol)
				case []order.Detail:
					printOrderSummary(p, u.Protocol)
				case nil:
				default:
					log.Warnf(log.SyncMgr,
						"Unexpected payload found %T but cannot print summary",
						u.Payload)
				}
				e.Unlock()
				continue
			}

			if u.Err == common.ErrNotYetImplemented ||
				u.Err == common.ErrFunctionNotSupported {
				u.Agent.DisableREST()
				u.Agent.SetProcessing(false)
				log.Warnf(log.SyncMgr, "%s %s %s disabling functionality: %s",
					u.Agent.GetExchangeName(),
					u.Agent.GetAgentName(),
					u.Protocol,
					u.Err)
			} else {
				log.Errorf(log.SyncMgr, "%s %s %s error %s",
					u.Agent.GetExchangeName(),
					u.Agent.GetAgentName(),
					u.Protocol,
					u.Err)
			}
			e.Unlock()
		}
	}
}

// StreamUpdate passes in a stream object to be processed and matched with
// an agent
func (e *SyncManager) StreamUpdate(payload interface{}) {
	if e == nil || payload == nil {
		return
	}

	if !e.Started() {
		return
	}

	// Default subscriptions for websocket streaming, this switch will stop
	// interaction with the sync agent [temp]
	switch payload.(type) {
	case *ticker.Price:
		if !e.ExchangeTicker {
			if Bot.Settings.Verbose {
				log.Warnln(log.SyncMgr, "Cannot process ticker stream, currently turned off")
			}
			return
		}
	case *orderbook.Base:
		if !e.ExchangeOrderbook {
			if Bot.Settings.Verbose {
				log.Warnln(log.SyncMgr, "Cannot process orderbook stream, currently turned off")
			}
			return
		}
	case []order.Detail:
		if !e.ExchangeTrades {
			if Bot.Settings.Verbose {
				log.Warnln(log.SyncMgr, "Cannot process order stream, currently turned off")
			}
			return
		}
	case []order.TradeHistory:
		if !e.ExchangeTrades {
			if Bot.Settings.Verbose {
				log.Warnln(log.SyncMgr, "Cannot process trade stream, currently turned off")
			}
			return
		}
	default:
		log.Warnf(log.SyncMgr,
			"Unexpected stream payload found %T cannot match with agent",
			payload)
		return
	}

	e.Lock()
	for i := range e.Agents {
		if agent := e.Agents[i].Stream(payload); agent != nil {
			go func(s Synchroniser, payload interface{}) {
				e.pipe <- SyncUpdate{
					Agent:    agent,
					Payload:  payload,
					Protocol: Websocket,
				}
			}(agent, payload)
			e.Unlock()
			return
		}
	}
	e.Unlock()
	log.Errorf(log.SyncMgr,
		"Cannot match payload %T with agent",
		payload)
}
