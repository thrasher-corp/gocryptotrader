package bittrex

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
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

func (b *Bittrex) setupOrderbookManager() {
	if b.obm == nil {
		b.obm = &orderbookManager{
			state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
			jobs:  make(chan job, maxWSOrderbookJobs),
		}
	} else {
		// Change state on reconnect for initial sync.
		for _, m1 := range b.obm.state {
			for _, m2 := range m1 {
				for _, update := range m2 {
					update.initialSync = true
					update.needsFetchingBook = true
				}
			}
		}
	}

	for i := 0; i < maxWSOrderbookWorkers; i++ {
		// 10 workers for synchronising book
		b.SynchroniseWebsocketOrderbook()
	}
}

// ProcessUpdateOB processes the websocket orderbook update
func (b *Bittrex) ProcessUpdateOB(pair currency.Pair, message *OrderbookUpdateMessage) error {
	updateBids := make([]orderbook.Item, len(message.BidDeltas))
	for x := range message.BidDeltas {
		updateBids[x] = orderbook.Item{
			Price:  message.BidDeltas[x].Rate,
			Amount: message.BidDeltas[x].Quantity,
		}
	}
	updateAsks := make([]orderbook.Item, len(message.AskDeltas))
	for x := range message.AskDeltas {
		updateAsks[x] = orderbook.Item{
			Price:  message.AskDeltas[x].Rate,
			Amount: message.AskDeltas[x].Quantity,
		}
	}

	return b.Websocket.Orderbook.Update(&orderbook.Update{
		Asset:    asset.Spot,
		Pair:     pair,
		UpdateID: message.Sequence,
		Bids:     updateBids,
		Asks:     updateAsks,
	})
}

// UpdateLocalOBBuffer updates and returns the most recent iteration of the orderbook
func (b *Bittrex) UpdateLocalOBBuffer(update *OrderbookUpdateMessage) (bool, error) {
	enabledPairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return false, err
	}

	format, err := b.GetPairFormat(asset.Spot, true)
	if err != nil {
		return false, err
	}

	currencyPair, err := currency.NewPairFromFormattedPairs(update.MarketSymbol,
		enabledPairs,
		format)
	if err != nil {
		return false, err
	}

	err = b.obm.stageWsUpdate(update, currencyPair, asset.Spot)
	if err != nil {
		init, err2 := b.obm.checkIsInitialSync(currencyPair)
		if err2 != nil {
			return false, err2
		}
		return init, err
	}

	err = b.applyBufferUpdate(currencyPair)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "%s websocket UpdateLocalOBBuffer: Could not apply buffer update\n", b.Name)
	}

	return false, err
}

// SeedLocalOBCache seeds depth data
func (b *Bittrex) SeedLocalOBCache(ctx context.Context, p currency.Pair) error {
	ob, sequence, err := b.GetOrderbook(ctx, p.String(), orderbookDepth)
	if err != nil {
		return err
	}
	return b.SeedLocalCacheWithOrderBook(p, sequence, ob, orderbookDepth)
}

// SeedLocalCacheWithOrderBook seeds the local orderbook cache
func (b *Bittrex) SeedLocalCacheWithOrderBook(p currency.Pair, sequence int64, orderbookNew *OrderbookData, maxDepth int) error {
	newOrderBook := orderbook.Base{
		Pair:            p,
		Asset:           asset.Spot,
		Exchange:        b.Name,
		LastUpdateID:    sequence,
		VerifyOrderbook: b.CanVerifyOrderbook,
		Bids:            make(orderbook.Items, len(orderbookNew.Bid)),
		Asks:            make(orderbook.Items, len(orderbookNew.Ask)),
		MaxDepth:        maxDepth,
	}

	for i := range orderbookNew.Bid {
		newOrderBook.Bids[i] = orderbook.Item{
			Amount: orderbookNew.Bid[i].Quantity,
			Price:  orderbookNew.Bid[i].Rate,
		}
	}
	for i := range orderbookNew.Ask {
		newOrderBook.Asks[i] = orderbook.Item{
			Amount: orderbookNew.Ask[i].Quantity,
			Price:  orderbookNew.Ask[i].Rate,
		}
	}

	return b.Websocket.Orderbook.LoadSnapshot(&newOrderBook)
}

// applyBufferUpdate applies the buffer to the orderbook or initiates a new
// orderbook sync by the REST protocol which is off handed to go routine.
func (b *Bittrex) applyBufferUpdate(pair currency.Pair) error {
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
		log.Errorf(
			log.WebsocketMgr,
			"%s error fetching recent orderbook when applying updates: %s\n",
			b.Name,
			err)
	}

	if recent != nil {
		err = b.obm.checkAndProcessUpdate(b.ProcessUpdateOB, pair, recent)
		if err != nil {
			log.Errorf(
				log.WebsocketMgr,
				"%s error processing update - initiating new orderbook sync via REST: %s\n",
				b.Name,
				err)
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
func (b *Bittrex) SynchroniseWebsocketOrderbook() {
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
				err := b.processJob(j.Pair)
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"%s processing websocket orderbook error %v",
						b.Name, err)
				}
			}
		}
	}()
}

// processJob fetches and processes orderbook updates
func (b *Bittrex) processJob(p currency.Pair) error {
	err := b.SeedLocalOBCache(context.TODO(), p)
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
	return b.applyBufferUpdate(p)
}

// stageWsUpdate stages websocket update to roll through updates that need to
// be applied to a fetched orderbook via REST.
func (o *orderbookManager) stageWsUpdate(u *OrderbookUpdateMessage, pair currency.Pair, a asset.Item) error {
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
			// 100ms update assuming we might have up to a 10 second delay.
			// There could be a potential 100 updates for the currency.
			buffer:            make(chan *OrderbookUpdateMessage, maxWSUpdateBuffer),
			fetchingBook:      false,
			initialSync:       true,
			needsFetchingBook: true,
		}
		m2[a] = state
	}

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

// handleFetchingBook checks if a full book is being fetched or needs to be
// fetched
func (o *orderbookManager) handleFetchingBook(pair currency.Pair) (fetching, needsFetching bool, err error) {
	o.Lock()
	defer o.Unlock()
	state, ok := o.state[pair.Base][pair.Quote][asset.Spot]
	if !ok {
		return false, false,
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

func (o *orderbookManager) checkAndProcessUpdate(processor func(currency.Pair, *OrderbookUpdateMessage) error, pair currency.Pair, recent *orderbook.Base) error {
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
			process, err := state.validate(d, recent)
			if err != nil {
				return err
			}
			if process {
				err := processor(pair, d)
				if err != nil {
					return fmt.Errorf("%s %s processing update error: %w",
						pair, asset.Spot, err)
				}
				recent.LastUpdateID = d.Sequence
			}
		default:
			break buffer
		}
	}
	return nil
}

// validate checks for correct update alignment
func (u *update) validate(updt *OrderbookUpdateMessage, recent *orderbook.Base) (bool, error) {
	if updt.Sequence <= recent.LastUpdateID {
		// Drop any event where u is <= lastUpdateId in the snapshot.
		return false, nil
	}

	id := recent.LastUpdateID + 1
	if u.initialSync {
		// The first processed event should have U <= lastUpdateId+1 AND
		// u >= lastUpdateId+1.
		if updt.Sequence > id {
			return false, fmt.Errorf("initial websocket orderbook sync failure for pair %s and asset %s",
				recent.Pair,
				asset.Spot)
		}
		u.initialSync = false
	} else if updt.Sequence != id {
		// While listening to the stream, each new event's U should be
		// equal to the previous event's u+1.
		return false, fmt.Errorf("websocket orderbook synchronisation failure for pair %s and asset %s",
			recent.Pair,
			asset.Spot)
	}
	return true, nil
}
