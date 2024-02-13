package exchange

import (
	"context"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// state is an enum that defines the state of the orderbook tracker
type state uint8

const (
	RequiresFetch state = iota
	Fetching
	Fetched

	// maxWSOrderbookWorkers defines a max amount of workers allowed to execute
	// jobs from the job channel
	maxWSOrderbookWorkers = 10
)

// OrderbookFetcher is a wrapper defined function that is used to fetch an
// orderbook from the REST API of an exchange. It is used to retrieve the
// initial orderbook state.
type OrderbookFetcher func(ctx context.Context, pair currency.Pair, a asset.Item) (*orderbook.Base, error)

// OrderbookChecker is a wrapper defined function that is used to check the
// incoming orderbook update against the current orderbook state. It is used
// to determine if the incoming update should be applied to the orderbook.
type OrderbookChecker func(loaded *orderbook.Base, incoming *orderbook.Update) (skip bool, err error)

// OrderbookBuilder is a type that helps build the initial state of an orderbook
// from a REST request and then maintains the orderbook state via websocket.
type OrderbookBuilder struct {
	exch    IBotExchange
	store   map[key.PairAsset]*tracker
	fetcher OrderbookFetcher
	checker OrderbookChecker
	mtx     sync.Mutex
}

// tracker is a type that holds the state of the orderbook and the pending
// updates that need to be applied to the orderbook.
type tracker struct {
	state          state
	pendingUpdates []*orderbook.Update
	m              sync.Mutex
}

func NewOrderbookBuilder(exch IBotExchange, fetcher OrderbookFetcher, checker OrderbookChecker) *OrderbookBuilder {
	return &OrderbookBuilder{
		exch:    exch,
		store:   make(map[key.PairAsset]*tracker),
		fetcher: fetcher,
	}
}

func (o *OrderbookBuilder) Process(ctx context.Context, update *orderbook.Update) error {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	ident := key.PairAsset{
		Base:  update.Pair.Base.Item,
		Quote: update.Pair.Quote.Item,
		Asset: update.Asset,
	}

	track, ok := o.store[ident]
	if !ok {
		track = &tracker{}
		o.store[ident] = track
	}

	track.m.Lock()
	switch track.state {
	case RequiresFetch:
		fmt.Println("fetching orderbook...")
		track.state = Fetching
		go func() {
			err := o.fetchAndApplyUpdate(context.Background(), update.Pair, update.Asset)
			if err != nil {
				fmt.Println(err)
			}
		}()
		fallthrough
	case Fetching:
		fmt.Println("pending update...")
		track.pendingUpdates = append(track.pendingUpdates, update)
	case Fetched:
		fmt.Println("applying update...")
		ws, err := o.exch.GetWebsocket()
		if err != nil {
			track.m.Unlock()
			return err
		}
		check, err := ws.Orderbook.GetOrderbook(update.Pair, update.Asset)
		if err != nil {
			track.m.Unlock()
			return err
		}

		_, err = o.checker(check, update)
		if err != nil {
			track.m.Unlock()
			return err
		}

		err = ws.Orderbook.Update(update)
		if err != nil {
			track.m.Unlock()
			return err
		}
	}
	track.m.Unlock()

	return nil
}

// fetchAndApplyUpdate fetches the orderbook and applies the pending updates.
// The fetcher function is defined by the exchange wrapper and is used to
// retrieve the orderbook from the REST API.
func (o *OrderbookBuilder) fetchAndApplyUpdate(ctx context.Context, pair currency.Pair, a asset.Item) error {
	ob, err := o.fetcher(ctx, pair, a)
	if err != nil {
		return err
	}

	ws, err := o.exch.GetWebsocket()
	if err != nil {
		return err
	}

	err = ws.Orderbook.LoadSnapshot(ob)
	if err != nil {
		return err
	}

	o.mtx.Lock()
	defer o.mtx.Unlock()
	track, ok := o.store[key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}]
	if !ok {
		return fmt.Errorf("orderbook tracker not found for %s %s %s", pair.Base, pair.Quote, a)
	}

	track.m.Lock()
	defer track.m.Unlock()
	track.state = Fetched
	defer func() { track.pendingUpdates = track.pendingUpdates[:0] }()

	fmt.Println("applying pending updates...")
	for _, update := range track.pendingUpdates {
		err := ws.Orderbook.Update(update)
		if err != nil {
			return err
		}
	}

	return nil
}

// func (b *OrderbookBuilder) setupOrderbookManager() {
// 	if b.obm == nil {
// 		b.obm = &orderbookManager{
// 			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
// 			jobs:  make(chan job, maxWSOrderbookJobs),
// 		}
// 	} else {
// 		// Change state on reconnect for initial sync.
// 		for _, m1 := range b.obm.state {
// 			for _, m2 := range m1 {
// 				for _, update := range m2 {
// 					update.initialSync = true
// 					update.needsFetchingBook = true
// 					update.lastUpdateID = 0
// 				}
// 			}
// 		}
// 	}

// 	for i := 0; i < maxWSOrderbookWorkers; i++ {
// 		// 10 workers for synchronising book
// 		b.SynchroniseWebsocketOrderbook()
// 	}
// }

// // setNeedsFetchingBook completes the book fetching initiation.
// func (o *OrderbookBuilder) setNeedsFetchingBook(pair currency.Pair) error {
// 	o.Lock()
// 	defer o.Unlock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
// 			pair,
// 			asset.Spot)
// 	}
// 	state.needsFetchingBook = true
// 	return nil
// }

// // SynchroniseWebsocketOrderbook synchronises full orderbook for currency pair
// // asset
// func (b *OrderbookBuilder) SynchroniseWebsocketOrderbook() {
// 	b.Websocket.Wg.Add(1)
// 	go func() {
// 		defer b.Websocket.Wg.Done()
// 		for {
// 			select {
// 			case <-b.Websocket.ShutdownC:
// 				for {
// 					select {
// 					case <-b.obm.jobs:
// 					default:
// 						return
// 					}
// 				}
// 			case j := <-b.obm.jobs:
// 				err := b.processJob(j.Pair)
// 				if err != nil {
// 					log.Errorf(log.WebsocketMgr,
// 						"%s processing websocket orderbook error %v",
// 						b.Name, err)
// 				}
// 			}
// 		}
// 	}()
// }

// // processJob fetches and processes orderbook updates
// func (b *OrderbookBuilder) processJob(p currency.Pair) error {
// 	err := b.SeedLocalCache(context.TODO(), p)
// 	if err != nil {
// 		return fmt.Errorf("%s %s seeding local cache for orderbook error: %v",
// 			p, asset.Spot, err)
// 	}

// 	err = b.obm.stopFetchingBook(p)
// 	if err != nil {
// 		return err
// 	}

// 	// Immediately apply the buffer updates so we don't wait for a
// 	// new update to initiate this.
// 	err = b.applyBufferUpdate(p)
// 	if err != nil {
// 		b.flushAndCleanup(p)
// 		return err
// 	}
// 	return nil
// }

// // flushAndCleanup flushes orderbook and clean local cache
// func (b *OrderbookBuilder) flushAndCleanup(p currency.Pair) {
// 	errClean := b.Websocket.Orderbook.FlushOrderbook(p, asset.Spot)
// 	if errClean != nil {
// 		log.Errorf(log.WebsocketMgr,
// 			"%s flushing websocket error: %v",
// 			b.Name,
// 			errClean)
// 	}
// 	errClean = b.obm.cleanup(p)
// 	if errClean != nil {
// 		log.Errorf(log.WebsocketMgr, "%s cleanup websocket error: %v",
// 			b.Name,
// 			errClean)
// 	}
// }

// // stageWsUpdate stages websocket update to roll through updates that need to
// // be applied to a fetched orderbook via REST.
// func (o *OrderbookBuilder) stageWsUpdate(u *WebsocketDepthStream, pair currency.Pair, a asset.Item) error {
// 	o.Lock()
// 	defer o.Unlock()
// 	m1, ok := o.state[pair.Base]
// 	if !ok {
// 		m1 = make(map[currency.Code]map[asset.Item]*update)
// 		o.state[pair.Base] = m1
// 	}

// 	m2, ok := m1[pair.Quote]
// 	if !ok {
// 		m2 = make(map[asset.Item]*update)
// 		m1[pair.Quote] = m2
// 	}

// 	state, ok := m2[a]
// 	if !ok {
// 		state = &update{
// 			// 100ms update assuming we might have up to a 10 second delay.
// 			// There could be a potential 100 updates for the currency.
// 			buffer:            make(chan *WebsocketDepthStream, maxWSUpdateBuffer),
// 			fetchingBook:      false,
// 			initialSync:       true,
// 			needsFetchingBook: true,
// 		}
// 		m2[a] = state
// 	}

// 	if state.lastUpdateID != 0 && u.FirstUpdateID != state.lastUpdateID+1 {
// 		// While listening to the stream, each new event's U should be
// 		// equal to the previous event's u+1.
// 		return fmt.Errorf("websocket orderbook synchronisation failure for pair %s and asset %s", pair, a)
// 	}
// 	state.lastUpdateID = u.LastUpdateID

// 	select {
// 	// Put update in the channel buffer to be processed
// 	case state.buffer <- u:
// 		return nil
// 	default:
// 		<-state.buffer    // pop one element
// 		state.buffer <- u // to shift buffer on fail
// 		return fmt.Errorf("channel blockage for %s, asset %s and connection",
// 			pair, a)
// 	}
// }

// // handleFetchingBook checks if a full book is being fetched or needs to be
// // fetched
// func (o *OrderbookBuilder) handleFetchingBook(pair currency.Pair) (fetching, needsFetching bool, err error) {
// 	o.Lock()
// 	defer o.Unlock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return false,
// 			false,
// 			fmt.Errorf("check is fetching book cannot match currency pair %s asset type %s",
// 				pair,
// 				asset.Spot)
// 	}

// 	if state.fetchingBook {
// 		return true, false, nil
// 	}

// 	if state.needsFetchingBook {
// 		state.needsFetchingBook = false
// 		state.fetchingBook = true
// 		return false, true, nil
// 	}
// 	return false, false, nil
// }

// // stopFetchingBook completes the book fetching.
// func (o *OrderbookBuilder) stopFetchingBook(pair currency.Pair) error {
// 	o.Lock()
// 	defer o.Unlock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
// 			pair,
// 			asset.Spot)
// 	}
// 	if !state.fetchingBook {
// 		return fmt.Errorf("fetching book already set to false for %s %s",
// 			pair,
// 			asset.Spot)
// 	}
// 	state.fetchingBook = false
// 	return nil
// }

// // completeInitialSync sets if an asset type has completed its initial sync
// func (o *OrderbookBuilder) completeInitialSync(pair currency.Pair) error {
// 	o.Lock()
// 	defer o.Unlock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return fmt.Errorf("complete initial sync cannot match currency pair %s asset type %s",
// 			pair,
// 			asset.Spot)
// 	}
// 	if !state.initialSync {
// 		return fmt.Errorf("initial sync already set to false for %s %s",
// 			pair,
// 			asset.Spot)
// 	}
// 	state.initialSync = false
// 	return nil
// }

// // checkIsInitialSync checks status if the book is Initial Sync being via the REST
// // protocol.
// func (o *OrderbookBuilder) checkIsInitialSync(pair currency.Pair) (bool, error) {
// 	o.Lock()
// 	defer o.Unlock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return false,
// 			fmt.Errorf("checkIsInitialSync of orderbook cannot match currency pair %s asset type %s",
// 				pair,
// 				asset.Spot)
// 	}
// 	return state.initialSync, nil
// }

// // fetchBookViaREST pushes a job of fetching the orderbook via the REST protocol
// // to get an initial full book that we can apply our buffered updates too.
// func (o *OrderbookBuilder) fetchBookViaREST(pair currency.Pair) error {
// 	o.Lock()
// 	defer o.Unlock()

// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return fmt.Errorf("fetch book via rest cannot match currency pair %s asset type %s",
// 			pair,
// 			asset.Spot)
// 	}

// 	state.initialSync = true
// 	state.fetchingBook = true

// 	select {
// 	case o.jobs <- job{pair}:
// 		return nil
// 	default:
// 		return fmt.Errorf("%s %s book synchronisation channel blocked up",
// 			pair,
// 			asset.Spot)
// 	}
// }

// func (o *OrderbookBuilder) checkAndProcessUpdate(processor func(currency.Pair, asset.Item, *WebsocketDepthStream) error, pair currency.Pair, recent *orderbook.Base) error {
// 	o.Lock()
// 	defer o.Unlock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return fmt.Errorf("could not match pair [%s] asset type [%s] in hash table to process websocket orderbook update",
// 			pair, asset.Spot)
// 	}

// 	// This will continuously remove updates from the buffered channel and
// 	// apply them to the current orderbook.
// buffer:
// 	for {
// 		select {
// 		case d := <-state.buffer:
// 			process, err := state.validate(d, recent)
// 			if err != nil {
// 				return err
// 			}
// 			if process {
// 				err := processor(pair, asset.Spot, d)
// 				if err != nil {
// 					return fmt.Errorf("%s %s processing update error: %w",
// 						pair, asset.Spot, err)
// 				}
// 			}
// 		default:
// 			break buffer
// 		}
// 	}
// 	return nil
// }

// // validate checks for correct update alignment
// func (u *update) validate(updt *WebsocketDepthStream, recent *orderbook.Base) (bool, error) {
// 	if updt.LastUpdateID <= recent.LastUpdateID {
// 		// Drop any event where u is <= lastUpdateId in the snapshot.
// 		return false, nil
// 	}

// 	id := recent.LastUpdateID + 1
// 	if u.initialSync {
// 		// The first processed event should have U <= lastUpdateId+1 AND
// 		// u >= lastUpdateId+1.
// 		if updt.FirstUpdateID > id || updt.LastUpdateID < id {
// 			return false, fmt.Errorf("initial websocket orderbook sync failure for pair %s and asset %s",
// 				recent.Pair,
// 				asset.Spot)
// 		}
// 		u.initialSync = false
// 	}
// 	return true, nil
// }

// // cleanup cleans up buffer and reset fetch and init
// func (o *OrderbookBuilder) cleanup(pair currency.Pair) error {
// 	o.Lock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		o.Unlock()
// 		return fmt.Errorf("cleanup cannot match %s %s to hash table",
// 			pair,
// 			asset.Spot)
// 	}

// bufferEmpty:
// 	for {
// 		select {
// 		case <-state.buffer:
// 			// bleed and discard buffer
// 		default:
// 			break bufferEmpty
// 		}
// 	}
// 	o.Unlock()
// 	// disable rest orderbook synchronisation
// 	_ = o.stopFetchingBook(pair)
// 	_ = o.completeInitialSync(pair)
// 	_ = o.stopNeedsFetchingBook(pair)
// 	return nil
// }

// // stopNeedsFetchingBook completes the book fetching initiation.
// func (o *OrderbookBuilder) stopNeedsFetchingBook(pair currency.Pair) error {
// 	o.Lock()
// 	defer o.Unlock()
// 	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
// 	if !ok {
// 		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
// 			pair,
// 			asset.Spot)
// 	}
// 	if !state.needsFetchingBook {
// 		return fmt.Errorf("needs fetching book already set to false for %s %s",
// 			pair,
// 			asset.Spot)
// 	}
// 	state.needsFetchingBook = false
// 	return nil
// }
