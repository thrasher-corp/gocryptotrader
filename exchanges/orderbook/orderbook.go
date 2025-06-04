package orderbook

import (
	"fmt"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Get checks and returns the orderbook given an exchange name and currency pair
func Get(exchange string, p currency.Pair, a asset.Item) (*Snapshot, error) {
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
func (s *store) Update(ss *Snapshot) error {
	s.m.RLock()
	book, ok := s.orderbooks[key.ExchangePairAsset{Exchange: ss.Exchange, Base: ss.Pair.Base.Item, Quote: ss.Pair.Quote.Item, Asset: ss.Asset}]
	s.m.RUnlock()
	if !ok {
		var err error
		book, err = s.track(ss)
		if err != nil {
			return err
		}
	}
	if err := book.Depth.LoadSnapshot(ss.Bids, ss.Asks, ss.LastUpdateID, ss.LastUpdated, ss.UpdatePushedAt, true); err != nil {
		return err
	}
	return s.signalMux.Publish(book.Depth, book.RouterID)
}

func (s *store) track(ss *Snapshot) (book, error) {
	s.m.Lock()
	defer s.m.Unlock()
	id, ok := s.exchangeRouters[ss.Exchange]
	if !ok {
		exchangeID, err := s.signalMux.GetID()
		if err != nil {
			return book{}, err
		}
		id = exchangeID
		s.exchangeRouters[ss.Exchange] = id
	}
	depth := NewDepth(id)
	depth.AssignOptions(ss)
	ob := book{RouterID: id, Depth: depth}
	s.orderbooks[key.ExchangePairAsset{Exchange: ss.Exchange, Base: ss.Pair.Base.Item, Quote: ss.Pair.Quote.Item, Asset: ss.Asset}] = ob
	return ob, nil
}

// DeployDepth used for subsystem deployment creates a depth item in the struct then returns a ptr to that Depth item
func (s *store) DeployDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	if exchange == "" {
		return nil, errExchangeNameUnset
	}
	if p.IsEmpty() {
		return nil, errPairNotSet
	}
	if !a.IsValid() {
		return nil, errAssetTypeNotSet
	}

	s.m.RLock()
	ob, ok := s.orderbooks[key.ExchangePairAsset{Exchange: exchange, Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	s.m.RUnlock()
	var err error
	if !ok {
		ob, err = s.track(&Snapshot{Exchange: exchange, Pair: p, Asset: a})
	}
	return ob.Depth, err
}

// GetDepth returns the actual depth struct for potential subsystems and strategies to interact with
func (s *store) GetDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	s.m.RLock()
	ob, ok := s.orderbooks[key.ExchangePairAsset{Exchange: exchange, Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	s.m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w for %s %s %s", ErrOrderbookNotFound, exchange, p, a)
	}
	return ob.Depth, nil
}

// Retrieve gets orderbook depth data from the stored tranches and returns the
// base equivalent copy
func (s *store) Retrieve(exchange string, p currency.Pair, a asset.Item) (*Snapshot, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	s.m.RLock()
	ob, ok := s.orderbooks[key.ExchangePairAsset{Exchange: exchange, Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}]
	s.m.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w for %s %s %s", ErrOrderbookNotFound, exchange, p, a)
	}
	return ob.Depth.Retrieve()
}

// GetDepth returns the concrete book allowing the caller to stream orderbook changes
func (ss *Snapshot) GetDepth() (*Depth, error) {
	return s.GetDepth(ss.Exchange, ss.Pair, ss.Asset)
}

// TotalBidsAmount returns the total amount of bids and the total orderbook
// bids value
func (ss *Snapshot) TotalBidsAmount() (amountCollated, total float64) {
	for x := range ss.Bids {
		amountCollated += ss.Bids[x].Amount
		total += ss.Bids[x].Amount * ss.Bids[x].Price
	}
	return amountCollated, total
}

// TotalAsksAmount returns the total amount of asks and the total orderbook
// asks value
func (ss *Snapshot) TotalAsksAmount() (amountCollated, total float64) {
	for y := range ss.Asks {
		amountCollated += ss.Asks[y].Amount
		total += ss.Asks[y].Amount * ss.Asks[y].Price
	}
	return amountCollated, total
}

// Verify ensures that the orderbook items are correctly sorted prior to being
// set and will reject any book with incorrect values.
// Bids should always go from a high price to a low price and
// Asks should always go from a low price to a higher price
func (ss *Snapshot) Verify() error {
	if !ss.VerifyOrderbook {
		return nil
	}

	// Checking for both ask and bid lengths being zero has been removed and
	// a warning has been put in place for some exchanges that return zero
	// level books. In the event that there is a massive liquidity change where
	// a book dries up, this will still update so we do not traverse potential
	// incorrect old data.
	if (len(ss.Asks) == 0 || len(ss.Bids) == 0) && !ss.Asset.IsOptions() {
		log.Warnf(log.OrderBook,
			bookLengthIssue,
			ss.Exchange,
			ss.Pair,
			ss.Asset,
			len(ss.Bids),
			len(ss.Asks))
	}
	err := checkAlignment(ss.Bids, ss.IsFundingRate, ss.PriceDuplication, ss.IDAlignment, ss.ChecksumStringRequired, dsc, ss.Exchange)
	if err != nil {
		return fmt.Errorf(bidLoadBookFailure, ss.Exchange, ss.Pair, ss.Asset, err)
	}
	err = checkAlignment(ss.Asks, ss.IsFundingRate, ss.PriceDuplication, ss.IDAlignment, ss.ChecksumStringRequired, asc, ss.Exchange)
	if err != nil {
		return fmt.Errorf(askLoadBookFailure, ss.Exchange, ss.Pair, ss.Asset, err)
	}
	return nil
}

// checker defines specific functionality to determine ascending/descending
// validation
type checker func(current, previous Tranche) error

// asc specifically defines ascending price check
var asc = func(current, previous Tranche) error {
	if current.Price < previous.Price {
		return errPriceOutOfOrder
	}
	return nil
}

// dsc specifically defines descending price check
var dsc = func(current, previous Tranche) error {
	if current.Price > previous.Price {
		return errPriceOutOfOrder
	}
	return nil
}

// checkAlignment validates full orderbook
func checkAlignment(depth Tranches, fundingRate, priceDuplication, isIDAligned, requiresChecksumString bool, c checker, exch string) error {
	for i := range depth {
		if depth[i].Price == 0 {
			switch {
			case exch == "Bitfinex" && fundingRate: /* funding rate can be 0 it seems on Bitfinex */
			default:
				return errPriceNotSet
			}
		}

		if depth[i].Amount <= 0 {
			return errAmountInvalid
		}
		if fundingRate && depth[i].Period == 0 {
			return errPeriodUnset
		}
		if requiresChecksumString && (depth[i].StrAmount == "" || depth[i].StrPrice == "") {
			return errChecksumStringNotSet
		}

		if i != 0 {
			prev := i - 1
			if err := c(depth[i], depth[prev]); err != nil {
				return err
			}
			if isIDAligned && depth[i].ID < depth[prev].ID {
				return errIDOutOfOrder
			}
			if !priceDuplication && depth[i].Price == depth[prev].Price {
				return errDuplication
			}
			if depth[i].ID != 0 && depth[i].ID == depth[prev].ID {
				return errIDDuplication
			}
		}
	}
	return nil
}

// Process processes incoming orderbooks, creating or updating the orderbook
// list
func (ss *Snapshot) Process() error {
	if ss.Exchange == "" {
		return errExchangeNameUnset
	}

	if ss.Pair.IsEmpty() {
		return errPairNotSet
	}

	if ss.Asset.String() == "" {
		return errAssetTypeNotSet
	}

	if ss.LastUpdated.IsZero() { // TODO: Enforce setting this on all exchanges
		ss.LastUpdated = time.Now()
	}

	if err := ss.Verify(); err != nil {
		return err
	}
	return s.Update(ss)
}

// Reverse reverses the order of orderbook items; some bid/asks are
// returned in either ascending or descending order. One bid or ask slice
// depending on what's received can be reversed. This is usually faster than
// using a sort algorithm as the algorithm could be impeded by a worst case time
// complexity when elements are shifted as opposed to just swapping element
// values.
func (ts *Tranches) Reverse() {
	eLen := len(*ts)
	var target int
	for i := eLen/2 - 1; i >= 0; i-- {
		target = eLen - 1 - i
		(*ts)[i], (*ts)[target] = (*ts)[target], (*ts)[i]
	}
}

// SortAsks sorts ask items to the correct ascending order if pricing values are
// scattered. If order from exchange is descending consider using the Reverse
// function.
func (ts Tranches) SortAsks() {
	sort.Slice(ts, func(i, j int) bool { return ts[i].Price < ts[j].Price })
}

// SortBids sorts bid items to the correct descending order if pricing values
// are scattered. If order from exchange is ascending consider using the Reverse
// function.
func (ts Tranches) SortBids() {
	sort.Slice(ts, func(i, j int) bool { return ts[i].Price > ts[j].Price })
}
