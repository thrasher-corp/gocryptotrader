package bithumb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// maxWSUpdateBuffer defines max websocket updates to apply when an
	// orderbook is initially fetched
	maxWSUpdateBuffer = 150
	// maxWSOrderbookJobs defines max websocket orderbook jobs in queue to fetch
	// an orderbook snapshot via REST
	maxWSOrderbookJobs = 2000
	// maxWSOrderbookWorkers defines a max amount of workers allowed to execute
	// jobs from the job channel
	maxWSOrderbookWorkers = 10
)

func (b *Bithumb) processBooks(updates *WsOrderbooks) error {
	bids := make([]orderbook.Level, 0, len(updates.List))
	asks := make([]orderbook.Level, 0, len(updates.List))
	for x := range updates.List {
		i := orderbook.Level{Price: updates.List[x].Price, Amount: updates.List[x].Quantity}
		if updates.List[x].OrderSide == "bid" {
			bids = append(bids, i)
			continue
		}
		asks = append(asks, i)
	}
	return b.Websocket.Orderbook.Update(&orderbook.Update{
		Pair:       updates.List[0].Symbol,
		Asset:      asset.Spot,
		Bids:       bids,
		Asks:       asks,
		UpdateTime: updates.DateTime.Time(),
	})
}

// UpdateLocalBuffer updates and returns the most recent iteration of the orderbook
func (b *Bithumb) UpdateLocalBuffer(wsdp *WsOrderbooks) (bool, error) {
	if len(wsdp.List) < 1 {
		return false, errors.New("insufficient data to process")
	}
	err := b.obm.stageWsUpdate(wsdp, wsdp.List[0].Symbol, asset.Spot)
	if err != nil {
		init, err2 := b.obm.checkIsInitialSync(wsdp.List[0].Symbol)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = b.applyBufferUpdate(wsdp.List[0].Symbol)
	if err != nil {
		b.invalidateAndCleanupOrderbook(wsdp.List[0].Symbol)
	}
	return false, err
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (b *Bithumb) applyBufferUpdate(pair currency.Pair) error {
	fetching, needsFetching, err := b.obm.handleFetchingBook(pair)
	if err != nil {
		return err
	}
	if fetching {
		return nil
	}

	if needsFetching {
		if b.Verbose {
			log.Debugf(log.WebsocketMgr, "%s Orderbook: Fetching via REST\n", b.Name)
		}
		return b.obm.fetchBookViaREST(pair)
	}

	recent, err := b.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "%s error fetching recent orderbook when applying updates: %s\n", b.Name, err)
	}

	if recent != nil {
		err = b.obm.checkAndProcessOrderbookUpdate(b.processBooks, pair, recent)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%s error processing update - initiating new orderbook sync via REST: %s\n", b.Name, err)
			err = b.obm.setNeedsFetchingBook(pair)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SynchroniseWebsocketOrderbook synchronises full orderbook for currency pair
// asset
func (b *Bithumb) SynchroniseWebsocketOrderbook(ctx context.Context) {
	b.Websocket.Wg.Add(1)
	go func() {
		defer b.Websocket.Wg.Done()
		for {
			select {
			case <-b.Websocket.ShutdownC:
				for {
					select {
					case <-b.obm.jobs:
					default:
						return
					}
				}
			case j := <-b.obm.jobs:
				err := b.processJob(ctx, j.Pair)
				if err != nil {
					log.Errorf(log.WebsocketMgr, "%s processing websocket orderbook error %v", b.Name, err)
				}
			}
		}
	}()
}

// processJob fetches and processes orderbook updates
func (b *Bithumb) processJob(ctx context.Context, p currency.Pair) error {
	err := b.SeedLocalCache(ctx, p)
	if err != nil {
		return fmt.Errorf("%s %s seeding local cache for orderbook error: %v",
			p, asset.Spot, err)
	}

	err = b.obm.stopFetchingBook(p)
	if err != nil {
		return err
	}

	// Immediately apply the buffer updates so we don't wait for a
	// new update to initiate this.
	err = b.applyBufferUpdate(p)
	if err != nil {
		b.invalidateAndCleanupOrderbook(p)
		return err
	}
	return nil
}

// invalidateAndCleanupOrderbook invalidates orderbook and cleans local cache
func (b *Bithumb) invalidateAndCleanupOrderbook(p currency.Pair) {
	if err := b.Websocket.Orderbook.InvalidateOrderbook(p, asset.Spot); err != nil {
		log.Errorf(log.WebsocketMgr, "%s invalidate orderbook websocket error: %v", b.Name, err)
	}
	if err := b.obm.cleanup(p); err != nil {
		log.Errorf(log.WebsocketMgr, "%s cleanup websocket error: %v", b.Name, err)
	}
}

func (b *Bithumb) setupOrderbookManager(ctx context.Context) {
	if b.obm.state == nil {
		b.obm.state = make(map[currency.Code]map[currency.Code]map[asset.Item]*update)
		b.obm.jobs = make(chan job, maxWSOrderbookJobs)
	} else {
		// Change state on reconnect for initial sync.
		for _, m1 := range b.obm.state {
			for _, m2 := range m1 {
				for _, update := range m2 {
					update.initialSync = true
					update.needsFetchingBook = true
					update.lastUpdated = time.Time{}
				}
			}
		}
	}

	for range maxWSOrderbookWorkers {
		// 10 workers for synchronising book
		b.SynchroniseWebsocketOrderbook(ctx)
	}
}

// stageWsUpdate stages websocket update to roll through updates that need to
// be applied to a fetched orderbook via REST.
func (o *orderbookManager) stageWsUpdate(u *WsOrderbooks, pair currency.Pair, a asset.Item) error {
	o.Lock()
	defer o.Unlock()
	m1, ok := o.state[pair.Base]
	if !ok {
		m1 = make(map[currency.Code]map[asset.Item]*update)
		o.state[pair.Base] = m1
	}

	m2, ok := m1[pair.Quote]
	if !ok {
		m2 = make(map[asset.Item]*update)
		m1[pair.Quote] = m2
	}

	state, ok := m2[a]
	if !ok {
		state = &update{
			buffer:            make(chan *WsOrderbooks, maxWSUpdateBuffer),
			fetchingBook:      false,
			initialSync:       true,
			needsFetchingBook: true,
		}
		m2[a] = state
	}

	if !state.lastUpdated.IsZero() && u.DateTime.Time().Before(state.lastUpdated) {
		return fmt.Errorf("websocket orderbook synchronisation failure for pair %s and asset %s", pair, a)
	}
	state.lastUpdated = u.DateTime.Time()

	select {
	// Put update in the channel buffer to be processed
	case state.buffer <- u:
		return nil
	default:
		<-state.buffer    // pop one element
		state.buffer <- u // to shift buffer on fail
		return fmt.Errorf("channel blockage for %s, asset %s and connection",
			pair, a)
	}
}

// handleFetchingBook checks if a full book is being fetched or needs to be
// fetched
func (o *orderbookManager) handleFetchingBook(pair currency.Pair) (fetching, needsFetching bool, err error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false,
			false,
			fmt.Errorf("check is fetching book cannot match currency pair %s asset type %s",
				pair,
				asset.Spot)
	}

	if state.fetchingBook {
		return true, false, nil
	}

	if state.needsFetchingBook {
		state.needsFetchingBook = false
		state.fetchingBook = true
		return false, true, nil
	}
	return false, false, nil
}

// stopFetchingBook completes the book fetching.
func (o *orderbookManager) stopFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	if !state.fetchingBook {
		return fmt.Errorf("fetching book already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.fetchingBook = false
	return nil
}

// completeInitialSync sets if an asset type has completed its initial sync
func (o *orderbookManager) completeInitialSync(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("complete initial sync cannot match currency pair %s asset type %s",
			pair,
			asset.Spot)
	}
	if !state.initialSync {
		return fmt.Errorf("initial sync already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.initialSync = false
	return nil
}

// checkIsInitialSync checks status if the book is Initial Sync being via the REST
// protocol.
func (o *orderbookManager) checkIsInitialSync(pair currency.Pair) (bool, error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false,
			fmt.Errorf("checkIsInitialSync of orderbook cannot match currency pair %s asset type %s",
				pair,
				asset.Spot)
	}
	return state.initialSync, nil
}

// fetchBookViaREST pushes a job of fetching the orderbook via the REST protocol
// to get an initial full book that we can apply our buffered updates too.
func (o *orderbookManager) fetchBookViaREST(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()

	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("fetch book via rest cannot match currency pair %s asset type %s",
			pair,
			asset.Spot)
	}

	state.initialSync = true
	state.fetchingBook = true

	select {
	case o.jobs <- job{pair}:
		return nil
	default:
		return fmt.Errorf("%s %s book synchronisation channel blocked up",
			pair,
			asset.Spot)
	}
}

func (o *orderbookManager) checkAndProcessOrderbookUpdate(processor func(*WsOrderbooks) error, pair currency.Pair, recent *orderbook.Book) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair [%s] asset type [%s] in hash table to process websocket orderbook update",
			pair, asset.Spot)
	}

	// This will continuously remove updates from the buffered channel and
	// apply them to the current orderbook.
buffer:
	for {
		select {
		case d := <-state.buffer:
			if !state.validate(d, recent) {
				continue
			}
			err := processor(d)
			if err != nil {
				return fmt.Errorf("%s %s processing update error: %w",
					pair, asset.Spot, err)
			}
		default:
			break buffer
		}
	}
	return nil
}

// validate checks for correct update alignment
func (u *update) validate(updt *WsOrderbooks, recent *orderbook.Book) bool {
	return updt.DateTime.Time().After(recent.LastUpdated)
}

// cleanup cleans up buffer and reset fetch and init
func (o *orderbookManager) cleanup(pair currency.Pair) error {
	o.Lock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		o.Unlock()
		return fmt.Errorf("cleanup cannot match %s %s to hash table",
			pair,
			asset.Spot)
	}

bufferEmpty:
	for {
		select {
		case <-state.buffer:
			// bleed and discard buffer
		default:
			break bufferEmpty
		}
	}
	o.Unlock()
	// disable rest orderbook synchronisation
	_ = o.stopFetchingBook(pair)
	_ = o.completeInitialSync(pair)
	_ = o.stopNeedsFetchingBook(pair)
	return nil
}

// SeedLocalCache seeds depth data
func (b *Bithumb) SeedLocalCache(ctx context.Context, p currency.Pair) error {
	ob, err := b.GetOrderBook(ctx, p.String())
	if err != nil {
		return err
	}
	return b.SeedLocalCacheWithBook(p, ob)
}

// SeedLocalCacheWithBook seeds the local orderbook cache
func (b *Bithumb) SeedLocalCacheWithBook(p currency.Pair, o *Orderbook) error {
	ob := &orderbook.Book{
		Pair:              p,
		Asset:             asset.Spot,
		Exchange:          b.Name,
		LastUpdated:       o.Data.Timestamp.Time(),
		ValidateOrderbook: b.ValidateOrderbook,
		Bids:              make(orderbook.Levels, len(o.Data.Bids)),
		Asks:              make(orderbook.Levels, len(o.Data.Asks)),
	}
	for i := range o.Data.Bids {
		ob.Bids[i].Price = o.Data.Bids[i].Price
		ob.Bids[i].Amount = o.Data.Bids[i].Quantity
	}
	for i := range o.Data.Asks {
		ob.Asks[i].Price = o.Data.Asks[i].Price
		ob.Asks[i].Amount = o.Data.Asks[i].Quantity
	}
	return b.Websocket.Orderbook.LoadSnapshot(ob)
}

// setNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) setNeedsFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	state.needsFetchingBook = true
	return nil
}

// stopNeedsFetchingBook completes the book fetching initiation.
func (o *orderbookManager) stopNeedsFetchingBook(pair currency.Pair) error {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return fmt.Errorf("could not match pair %s and asset type %s in hash table",
			pair,
			asset.Spot)
	}
	if !state.needsFetchingBook {
		return fmt.Errorf("needs fetching book already set to false for %s %s",
			pair,
			asset.Spot)
	}
	state.needsFetchingBook = false
	return nil
}
