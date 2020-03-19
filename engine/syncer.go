package engine

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
		pipe:       make(chan SyncUpdate, 1000),
		syncComm:   make(chan time.Time),
		jobBuffer:  make(map[string]chan Synchroniser),
	}, nil
}

// Started returns if gctscript manager subsystem is started
func (e *SyncManager) Started() bool {
	return atomic.LoadInt32(&e.started) == 1
}

// Start starts the synchronisation manager
func (e *SyncManager) Start() error {
	if !atomic.CompareAndSwapInt32(&e.started, 0, 1) {
		return errors.New("synchronisation manager already started")
	}

	Bot.ServicesWG.Add(1)

	go func() {
		exchanges := GetExchanges()
		for x := range exchanges {
			if !exchanges[x].IsEnabled() {
				continue
			}

			exchName := exchanges[x].GetName()

			if !exchanges[x].SupportsREST() &&
				!exchanges[x].SupportsWebsocket() {
				log.Warnf(log.SyncMgr,
					"Loaded exchange %s does not support REST or Websocket.\n",
					exchName)
				continue
			}

			// Initial synchronisation count and waitgroup
			var exchangeSyncItems int

			auth := exchanges[x].GetBase().API.AuthenticatedSupport
			if !auth && e.AccountBalance {
				log.Warnf(log.SyncMgr,
					"Loaded exchange %s cannot sync account specific items as functionality is disabled.\n",
					exchName)
			}

			var exchangeWG sync.WaitGroup

			if auth {
				// Fetches account balance for the exchange
				if e.AccountBalance {
					e.Agents = append(e.Agents, &AccountBalanceAgent{
						Agent: Agent{
							Name:            "AccountBalanceAgent",
							Exchange:        exchanges[x],
							RestUpdateDelay: defaultSyncDelay,
							Pipe:            e.pipe,
							Wg:              &exchangeWG,
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
						Wg:              &exchangeWG,
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
								Wg:              &exchangeWG,
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
								Wg:              &exchangeWG,
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
								Wg:              &exchangeWG,
							},
						})
						exchangeSyncItems++
					}
				}
			}

			if exchangeSyncItems != 0 {
				// This is linked up with an executor, which limits the outbound
				// REST requests to not exceed the requester MaxRequestJobs variable
				e.jobBuffer[exchName] = make(chan Synchroniser, exchangeSyncItems)

				go func(exchName string, count int, wg *sync.WaitGroup) {
					wg.Add(count)
					log.Debugf(log.SyncMgr,
						"Initial sync started for %s with [%d] ---------------------------- \n",
						exchName,
						count)
					start := time.Now()
					wg.Wait()
					end := time.Now()
					log.Debugf(log.SyncMgr,
						"Initial sync finished for %s in [%s] ----------------------------- \n",
						exchName,
						end.Sub(start))
				}(exchName, exchangeSyncItems, &exchangeWG)
			} else {
				log.Warnf(log.SyncMgr,
					"Initial sync cannot start for %s as no sync items available",
					exchName)
			}
		}

		for _, channel := range e.jobBuffer {
			e.wg.Add(1)
			go e.executor(channel)
		}

		e.wg.Add(3)
		go e.monitor()
		go e.worker()
		go e.Processor()
	}()
	return nil
}

// Stop shuts down the exchange currency pair syncer
func (e *SyncManager) Stop() error {
	if atomic.LoadInt32(&e.started) == 0 {
		return errors.New("synchronisation manager has not started")
	}

	if !atomic.CompareAndSwapInt32(&e.stopped, 0, 1) {
		return errors.New("synchronisation manager has already stopped")
	}

	log.Debugln(log.SyncMgr, "Synchronisation manager shutting down...")
	close(e.shutdown)
	e.wg.Wait()

	if !atomic.CompareAndSwapInt32(&e.started, 1, 0) {
		fmt.Println(e.started)
		return errors.New("value started not swapped")
	}
	Bot.ServicesWG.Done()
	log.Debugln(log.SyncMgr, "Synchronisation manager has shutdown.")
	return nil
}

// Worker iterates across the full agent list and initiates an update by
// detatching in a routine which directly interacts with a work manager.
// The routine generated will block until an update has occured then it will
// interact with the processor routine keeping a record of last updated and when
// a new update should occur to keep it in optimal synchronisation on the main
// fall-back REST protocol.
func (e *SyncManager) worker() {
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
			e.RLock()
			for i := range e.Agents {
				e.Agents[i].Lock()
				if e.Agents[i].IsRESTDisabled() {
					e.Agents[i].Unlock()
					continue
				}
				nextUpdate := e.Agents[i].GetNextUpdate()
				if nextUpdate.Before(time.Now()) || nextUpdate == (time.Time{}) {
					if !e.Agents[i].IsProcessing() {
						e.Agents[i].SetProcessing(true)
						e.Agents[i].Clear()
						select {
						case e.jobBuffer[e.Agents[i].GetExchangeName()] <- e.Agents[i]:
						default:
							log.Warnf(log.SyncMgr,
								"extreme back log detected shunting %s execution to routine\n",
								e.Agents[i].GetAgentName())
							go func(s Synchroniser) {
								e.jobBuffer[e.Agents[i].GetExchangeName()] <- e.Agents[i]
							}(e.Agents[i])
						}
					}
				} else {
					if nextUpdate.Before(sleep) || sleep == (time.Time{}) {
						sleep = nextUpdate
					}
				}
				e.Agents[i].Unlock()
			}
			e.RUnlock()

			if sleep != (time.Time{}) {
				// Passes the sleep time to the Monitor routine which interdicts
				// a wake up alert to the Worker routine if time < current
				// update time.
				e.syncComm <- sleep
			}
		}
	}
}

// executor monitors job buffer to not exceed max request jobs
// WARNING: This is a temporary WIP work manager, TODO: This will be a
// centralised service that will monitor all outbound calls to not exceed any
// requester limitations.
func (e *SyncManager) executor(sc chan Synchroniser) {
	var wg sync.WaitGroup
	var count int32
	var mtx sync.Mutex

	// This is a temp reduction on this system to allow for other outbound
	// requests currently not linked up with this system.
	var tempReduction = int32(5)

	for {
		select {
		case <-e.shutdown:
			e.wg.Done()
			return
		case s := <-sc:
			if !s.IsCancelled() {
				wg.Wait()
				mtx.Lock()
				if count+tempReduction >= request.MaxRequestJobs {
					count++
					mtx.Unlock()
					wg.Add(1)
					go func(wg *sync.WaitGroup, count *int32, mtx *sync.Mutex) {
						s.Execute()
						mtx.Lock()
						*count--
						mtx.Unlock()
						wg.Done()
					}(&wg, &count, &mtx)
				} else {
					count++
					mtx.Unlock()
					go func(count *int32, mtx *sync.Mutex) {
						s.Execute()
						mtx.Lock()
						*count--
						mtx.Unlock()
					}(&count, &mtx)
				}
			} else {
				if Bot.Settings.Verbose {
					log.Debugf(log.SyncMgr,
						"%s successfully cancelled due to already updated by stream process\n",
						s.GetAgentName())
				}
			}
		}
	}
}

// Monitor monitors the last updated to initiate an agent check for the worker
func (e *SyncManager) monitor() {
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
			if u.Err == nil {
				// Check for last updated
				u.Agent.Lock()
				lastUpdated := u.Agent.GetLastUpdated()
				if lastUpdated != (time.Time{}) {
					if Bot.Settings.Verbose {
						log.Debugf(log.SyncMgr, "Agent last updated %s ago",
							time.Now().Sub(lastUpdated))
					}
				} else {
					// Set initial sync completed on agent
					u.Agent.InitialSyncComplete()
				}

				// Sets new update time
				u.Agent.SetNewUpdate()
				u.Agent.SetProcessing(false)

				// Send items next update time to the Monitor routine for check
				e.syncComm <- u.Agent.GetNextUpdate()
				u.Agent.Unlock()

				switch p := u.Payload.(type) {
				case *ticker.Price:
					printTickerSummary(p, u.Protocol, u.Err)
				case *orderbook.Base:
					printOrderbookSummary(p, u.Protocol, u.Err)
				case *account.Holdings:
					printAccountSummary(p, u.Protocol)
				case []order.TradeHistory: // from REST
					printTradeSummary(p, u.Protocol)
				case order.TradeHistory: // from websocket
					printTradeSummary([]order.TradeHistory{p}, u.Protocol)
				case *kline.Item:
					printKlineSummary(p, u.Protocol)
				case []order.Detail:
					printOrderSummary(p, u.Protocol)
				default:
					log.Warnf(log.SyncMgr,
						"Unexpected payload found %T but cannot print summary",
						u.Payload)
				}
				continue
			}

			u.Agent.Lock()
			if u.Err == common.ErrNotYetImplemented ||
				u.Err == common.ErrFunctionNotSupported {
				u.Agent.DisableREST()
				u.Agent.SetProcessing(false)
				u.Agent.SetNewUpdate()
				u.Agent.InitialSyncComplete()
				log.Warnf(log.SyncMgr, "%s %s %s disabling functionality: %s",
					u.Agent.GetExchangeName(),
					u.Agent.GetAgentName(),
					u.Protocol,
					u.Err)
				u.Agent.Unlock()
			} else {
				log.Errorf(log.SyncMgr, "%s %s %s error %s",
					u.Agent.GetExchangeName(),
					u.Agent.GetAgentName(),
					u.Protocol,
					u.Err)
				u.Agent.Unlock()
			}
		}
	}
}

// DeRegister removes agent from list
func (e *SyncManager) DeRegister(s Synchroniser) error {
	e.Lock()
	for i := range e.Agents {
		if s == e.Agents[i] {
			e.Agents[i] = e.Agents[len(e.Agents)-1]
			e.Agents[len(e.Agents)-1] = nil
			e.Agents = e.Agents[:len(e.Agents)-1]
			e.Unlock()
			return nil
		}
	}
	e.Unlock()
	return fmt.Errorf("agent %s not found", s)
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
	case order.TradeHistory:
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

	e.RLock()
	for i := range e.Agents {
		e.Agents[i].Lock()
		if agent := e.Agents[i].Stream(payload); agent != nil {
			if agent.IsProcessing() {
				agent.Cancel()
				if Bot.Settings.Verbose {
					log.Debugf(log.SyncMgr,
						"%s updated via websocket cancelling REST request on stack",
						agent.GetAgentName())
				}
			} else {
				agent.SetProcessing(true)
			}

			e.pipe <- SyncUpdate{
				Agent:    agent,
				Payload:  payload,
				Protocol: Websocket,
			}
			e.Agents[i].Unlock()
			e.RUnlock()
			return
		}
		e.Agents[i].Unlock()
	}
	e.RUnlock()
	log.Errorf(log.SyncMgr,
		"Cannot match payload %+v with agent",
		payload)
	for i := range e.Agents {
		log.Debugln(log.SyncMgr, e.Agents[i])
	}

	os.Exit(1)
}
