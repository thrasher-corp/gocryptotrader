package orderbook

import (
	"fmt"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Get checks and returns the orderbook given an exchange name and currency pair
func Get(exchange string, p currency.Pair, a asset.Item) (*Book, error) {
	return s.Retrieve(exchange, p, a)
}

// GetDepth returns a Depth pointer allowing the caller to stream orderbook changes
func GetDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	return s.GetDepth(exchange, p, a)
}

// DeployDepth sets a depth struct and returns a depth pointer. This allows for
// the loading of a new orderbook snapshot and incremental updates via the
// streaming package.
func DeployDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	return s.DeployDepth(exchange, p, a)
}

// SubscribeToExchangeOrderbooks returns a pipe to an exchange feed
func SubscribeToExchangeOrderbooks(exchange string) (dispatch.Pipe, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	id, ok := s.exchangeRouters[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%w for %s exchange", ErrOrderbookNotFound, exchange)
	}
	return s.signalMux.Subscribe(id)
}

// Update stores orderbook data
func (s *store) Update(b *Book) error {
	s.m.RLock()
	book, ok := s.orderbooks[key.ExchangeAssetPair{Exchange: b.Exchange, Base: b.Pair.Base.Item, Quote: b.Pair.Quote.Item, Asset: b.Asset}]
	s.m.RUnlock()
	if !ok {
		var err error
		book, err = s.track(b)
		if err != nil {
			return err
		}
	}
	b.RestSnapshot = true
	if err := book.Depth.LoadSnapshot(b); err != nil {
		return err
	}
	return s.signalMux.Publish(book.Depth, book.RouterID)
}

func (s *store) track(b *Book) (book, error) {
	s.m.Lock()
	defer s.m.Unlock()
	id, ok := s.exchangeRouters[b.Exchange]
	if !ok {
		exchangeID, err := s.signalMux.GetID()
		if err != nil {
			return book{}, err
		}
		id = exchangeID
		s.exchangeRouters[b.Exchange] = id
	}
	depth := NewDepth(id)
	depth.AssignOptions(b)
	ob := book{RouterID: id, Depth: depth}
	s.orderbooks[key.ExchangeAssetPair{Exchange: b.Exchange, Base: b.Pair.Base.Item, Quote: b.Pair.Quote.Item, Asset: b.Asset}] = ob
	return ob, nil
}

// DeployDepth used for subsystem deployment creates a depth item in the struct then returns a ptr to that Depth item
func (s *store) DeployDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	if exchange == "" {
		return nil, common.ErrExchangeNameNotSet
	}
	if p.IsEmpty() {
		return nil, errPairNotSet
	}
	if !a.IsValid() {
		return nil, errAssetTypeNotSet
	}

	s.m.RLock()
	ob, ok := s.orderbooks[key.ExchangeAssetPair{Exchange: exchange, Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	s.m.RUnlock()
	var err error
	if !ok {
		ob, err = s.track(&Book{Exchange: exchange, Pair: p, Asset: a})
	}
	return ob.Depth, err
}

// GetDepth returns the actual depth struct for potential subsystems and strategies to interact with
func (s *store) GetDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	s.m.RLock()
	ob, ok := s.orderbooks[key.ExchangeAssetPair{Exchange: exchange, Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	s.m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w for %q %q %q", ErrOrderbookNotFound, exchange, p, a)
	}
	return ob.Depth, nil
}

// Retrieve gets orderbook depth data from the stored Levels and returns the
// base equivalent copy
func (s *store) Retrieve(exchange string, p currency.Pair, a asset.Item) (*Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	s.m.RLock()
	ob, ok := s.orderbooks[key.ExchangeAssetPair{Exchange: exchange, Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	s.m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w for %q %q %q", ErrOrderbookNotFound, exchange, p, a)
	}
	return ob.Depth.Retrieve()
}

// GetDepth returns the concrete book allowing the caller to stream orderbook changes
func (b *Book) GetDepth() (*Depth, error) {
	return s.GetDepth(b.Exchange, b.Pair, b.Asset)
}

// TotalBidsAmount returns the total amount of bids and the total orderbook
// bids value
func (b *Book) TotalBidsAmount() (amountCollated, total float64) {
	for x := range b.Bids {
		amountCollated += b.Bids[x].Amount
		total += b.Bids[x].Amount * b.Bids[x].Price
	}
	return amountCollated, total
}

// TotalAsksAmount returns the total amount of asks and the total orderbook
// asks value
func (b *Book) TotalAsksAmount() (amountCollated, total float64) {
	for y := range b.Asks {
		amountCollated += b.Asks[y].Amount
		total += b.Asks[y].Amount * b.Asks[y].Price
	}
	return amountCollated, total
}

// Process processes incoming orderbooks, creating or updating the orderbook
// list
func (b *Book) Process() error {
	if b.Exchange == "" {
		return common.ErrExchangeNameNotSet
	}

	if b.Pair.IsEmpty() {
		return errPairNotSet
	}

	if b.Asset.String() == "" {
		return errAssetTypeNotSet
	}

	if b.LastUpdated.IsZero() { // TODO: Enforce setting this on all exchanges
		b.LastUpdated = time.Now()
	}

	if err := b.Validate(); err != nil {
		return err
	}
	return s.Update(b)
}

// Reverse reverses the order of orderbook items; some bid/asks are
// returned in either ascending or descending order. One bid or ask slice
// depending on what's received can be reversed. This is usually faster than
// using a sort algorithm as the algorithm could be impeded by a worst case time
// complexity when elements are shifted as opposed to just swapping element
// values.
func (l *Levels) Reverse() {
	eLen := len(*l)
	var target int
	for i := eLen/2 - 1; i >= 0; i-- {
		target = eLen - 1 - i
		(*l)[i], (*l)[target] = (*l)[target], (*l)[i]
	}
}

// SortAsks sorts ask items to the correct ascending order if pricing values are
// scattered. If order from exchange is descending consider using the Reverse
// function.
func (l Levels) SortAsks() {
	sort.Slice(l, func(i, j int) bool { return l[i].Price < l[j].Price })
}

// SortBids sorts bid items to the correct descending order if pricing values
// are scattered. If order from exchange is ascending consider using the Reverse
// function.
func (l Levels) SortBids() {
	sort.Slice(l, func(i, j int) bool { return l[i].Price > l[j].Price })
}
