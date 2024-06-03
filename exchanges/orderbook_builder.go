package exchange

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	requiresFetch state = iota
	fetching
	fetched

	// defaultWindow defines a max amount of workers that a concurrently
	// fetching using the request package. This is to prevent a single
	// exchange from being overloaded with requests.
	defaultWindow = 10
)

var (
	errShutdownBuilderServices            = errors.New("orderbook builder services have been shutdown")
	errOrderbookFetchingFunctionUndefined = errors.New("orderbook fetching function undefined")
	errOrderbookCheckingFunctionUndefined = errors.New("orderbook checking function undefined")
	errOrderbookUpdateIsNil               = errors.New("orderbook update is nil")
)

// state is an enum that defines the state of the orderbook tracker
type state uint8

// OrderbookBuilder is a type that helps build the initial state of an orderbook
// from a REST request and then maintains the orderbook state via websocket.
type OrderbookBuilder struct {
	exch     IBotExchange
	store    map[key.PairAsset]*tracker
	fetcher  OrderbookFetcher
	checker  OrderbookChecker
	tm       ticketMachine
	shutdown chan struct{}
	mtx      sync.Mutex
}

// OrderbookFetcher is a wrapper defined function that is used to fetch an
// orderbook from the REST API of an exchange. It is used to retrieve the
// initial orderbook state.
type OrderbookFetcher func(ctx context.Context, pair currency.Pair, a asset.Item) (*orderbook.Base, error)

// OrderbookChecker is a wrapper defined function that is used to check the
// incoming orderbook update against the current orderbook state. It is used
// to determine if the incoming update should be applied to the orderbook.
type OrderbookChecker func(loaded *orderbook.Base, incoming *orderbook.Update, isInitialSync bool) (skip bool, err error)

// tracker is a type that holds the state of the orderbook and the pending
// updates that need to be applied to the orderbook.
type tracker struct {
	state          state
	pendingUpdates []*orderbook.Update
	initFinished   bool
	m              sync.Mutex
}

// NewOrderbookBuilder returns a new instance of OrderbookBuilder. It requires
// an exchange, a fetcher and a checker function to be defined.
func NewOrderbookBuilder(exch IBotExchange, fetcher OrderbookFetcher, checker OrderbookChecker) (*OrderbookBuilder, error) {
	if exch == nil {
		return nil, errExchangeIsNil
	}
	if fetcher == nil {
		return nil, errOrderbookFetchingFunctionUndefined
	}
	if checker == nil {
		return nil, errOrderbookCheckingFunctionUndefined
	}
	return &OrderbookBuilder{
		exch:     exch,
		store:    make(map[key.PairAsset]*tracker),
		fetcher:  fetcher,
		checker:  checker,
		shutdown: make(chan struct{}),
	}, nil
}

// Process processes an incoming orderbook update. If the orderbook tracker
// does not exist, it will create a new tracker and fetch the orderbook from
// the REST API. If the orderbook tracker exists and is in the process of
// fetching the orderbook, it will append the update to the pending updates
// list. If the orderbook tracker exists and has fetched the orderbook, it will
// apply the update to the orderbook inline with this caller thread.
func (o *OrderbookBuilder) Process(ctx context.Context, update *orderbook.Update) error {
	if update == nil {
		return errOrderbookUpdateIsNil
	}

	if update.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}

	if !update.Asset.IsValid() {
		return asset.ErrInvalidAsset
	}

	o.mtx.Lock()
	defer o.mtx.Unlock()

	i := key.PairAsset{Base: update.Pair.Base.Item, Quote: update.Pair.Quote.Item, Asset: update.Asset}
	track, ok := o.store[i]
	if !ok {
		track = &tracker{}
		o.store[i] = track
	}

	track.m.Lock()
	defer track.m.Unlock()

	switch track.state {
	case requiresFetch:
		track.state = fetching
		go func() {
			err := o.fetchAndApplyUpdate(ctx, update.Pair, update.Asset)
			if err != nil {
				log.Warnf(log.ExchangeSys, "OrderbookBuilder fetchAndApplyUpdate error: %v", err)
			}
		}()
		fallthrough
	case fetching:
		track.pendingUpdates = append(track.pendingUpdates, update)
	case fetched:
		ws, err := o.exch.GetWebsocket()
		if err != nil {
			return err
		}
		check, err := ws.Orderbook.GetOrderbook(update.Pair, update.Asset)
		if err != nil {
			return err
		}

		skipUpdate, err := o.checker(check, update, !track.initFinished)
		if err != nil {
			return err
		}

		if skipUpdate {
			return nil
		}

		err = ws.Orderbook.Update(update)
		if err != nil {
			return err
		}
		track.initFinished = true
	}

	return nil
}

// fetchAndApplyUpdate fetches the orderbook and applies the pending updates.
// The fetcher function is defined by the exchange wrapper and is used to
// retrieve the orderbook from the REST API.
func (o *OrderbookBuilder) fetchAndApplyUpdate(ctx context.Context, pair currency.Pair, a asset.Item) error {
	ticket := o.tm.getTicket()
	select {
	case <-ctx.Done():
		o.tm.releaseTicket()
		return ctx.Err()
	case <-o.shutdown:
		o.tm.releaseTicket()
		return errShutdownBuilderServices
	case <-ticket:
	}

	ob, err := o.fetcher(ctx, pair, a)
	o.tm.releaseTicket()
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
	track.state = fetched

	check, err := ws.Orderbook.GetOrderbook(pair, a)
	if err != nil {
		return err
	}

	defer func() { track.pendingUpdates = track.pendingUpdates[:0] }()

	for _, update := range track.pendingUpdates {
		skipUpdate, err := o.checker(check, update, !track.initFinished)
		if err != nil {
			return err
		}

		if skipUpdate {
			continue
		}

		err = ws.Orderbook.Update(update)
		if err != nil {
			return err
		}

		track.initFinished = true
	}

	return nil
}

// Release releases the resources used by the orderbook builder
func (o *OrderbookBuilder) Release() {
	close(o.shutdown)
}

// ticketMachine is a type that is used to limit the amount of concurrent
// requests that can be made to an exchange. This is to prevent a single
// exchange from being overloaded with requests.
type ticketMachine struct {
	pending []chan struct{}
	m       sync.Mutex
}

// getTicket returns a channel that is closed when the caller can proceed with
// their request. If the amount of pending requests is less than the default
// window, the channel is closed immediately.
func (t *ticketMachine) getTicket() chan struct{} {
	t.m.Lock()
	defer t.m.Unlock()
	ticket := make(chan struct{})
	t.pending = append(t.pending, ticket)
	l := len(t.pending)
	if l <= defaultWindow {
		close(ticket)
	}
	return ticket
}

// releaseTicket releases the first ticket in the queue and closes the channel
// if the amount of pending requests is greater than the default window.
func (t *ticketMachine) releaseTicket() {
	t.m.Lock()
	defer t.m.Unlock()
	l := len(t.pending)
	if l == 0 {
		return
	}
	t.pending = t.pending[1:]
	if l > defaultWindow {
		close(t.pending[defaultWindow-1])
	}
}
