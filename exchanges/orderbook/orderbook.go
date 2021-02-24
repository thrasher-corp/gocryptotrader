package orderbook

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Get checks and returns the orderbook given an exchange name and currency pair
func Get(exchange string, p currency.Pair, a asset.Item) (*Base, error) {
	return service.Retrieve(exchange, p, a)
}

// GetDepth returns depth
func GetDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	return service.GetDepth(exchange, p, a)
}

// SubscribeOrderbook subcribes to an orderbook and returns a communication
// channel to stream orderbook data updates
func SubscribeOrderbook(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	exchange = strings.ToLower(exchange)
	service.Lock()
	defer service.Unlock()
	book, ok := service.Books[exchange][a][p.Base.Item][p.Quote.Item]
	if !ok {
		return dispatch.Pipe{},
			fmt.Errorf("orderbook item not found for %s %s %s",
				exchange,
				p,
				a)
	}
	return service.mux.Subscribe(book.main)
}

// SubscribeToExchangeOrderbooks subcribes to all orderbooks on an exchange
func SubscribeToExchangeOrderbooks(exchange string) (dispatch.Pipe, error) {
	service.Lock()
	defer service.Unlock()
	id, ok := service.Exchange[strings.ToLower(exchange)]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%s exchange orderbooks not found",
			exchange)
	}
	return service.mux.Subscribe(id)
}

// Update stores orderbook data
func (s *Service) Update(b *Base) error {
	name := strings.ToLower(b.Exchange)
	s.Lock()
	m1, ok := s.Books[name]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Book)
		s.Books[name] = m1
	}

	m2, ok := m1[b.Asset]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*Book)
		m1[b.Asset] = m2
	}

	m3, ok := m2[b.Pair.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*Book)
		m2[b.Pair.Base.Item] = m3
	}

	book, ok := m3[b.Pair.Quote.Item]
	if !ok {
		book = new(Book)
		m3[b.Pair.Quote.Item] = book
		err := s.SetNewData(b, book, name)
		s.Unlock()
		return err
	}

	book.Process(b.Bids, b.Asks)
	book.LastUpdated = b.LastUpdated
	book.RestSnapshot = true
	ids := append(book.assoc, book.main)
	s.Unlock()
	return s.mux.Publish(ids, b)
}

// SetNewData sets new data
func (s *Service) SetNewData(ob *Base, book *Book, exch string) error {
	var err error
	book.assoc, err = s.getAssociations(strings.ToLower(exch))
	if err != nil {
		return err
	}
	book.main, err = s.mux.GetID()
	if err != nil {
		return err
	}

	book.Process(ob.Bids, ob.Asks)
	book.Exchange = ob.Exchange
	book.Asset = ob.Asset
	book.Pair = ob.Pair
	book.RestSnapshot = ob.RestSnapshot
	book.IsFundingRate = ob.IsFundingRate
	book.LastUpdateID = ob.LastUpdateID
	book.HasChecksumValidation = ob.HasChecksumValidation
	book.NotAggregated = ob.NotAggregated
	return nil
}

// GetAssociations links a singular book with it's dispatch associations
func (s *Service) getAssociations(exch string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	exchangeID, ok := s.Exchange[exch]
	if !ok {
		var err error
		exchangeID, err = s.mux.GetID()
		if err != nil {
			return nil, err
		}
		s.Exchange[exch] = exchangeID
	}

	ids = append(ids, exchangeID)
	return ids, nil
}

// GetDepth returns the actual depth struct for potential subsystems and
// strategies to interract with
func (s *Service) GetDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	s.Lock()
	defer s.Unlock()
	m1, ok := s.Books[strings.ToLower(exchange)]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Book)
		s.Books[strings.ToLower(exchange)] = m1
	}
	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*Book)
		m1[a] = m2
	}
	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*Book)
		m2[p.Base.Item] = m3
	}
	book, ok := m3[p.Quote.Item]
	if !ok {
		book = &Book{}
		m3[p.Quote.Item] = book
	}
	return &book.Depth, nil
}

// Retrieve gets orderbook depth data from the associated linked list and
// returns the base equivalent copy
func (s *Service) Retrieve(exchange string, p currency.Pair, a asset.Item) (*Base, error) {
	s.Lock()
	defer s.Unlock()
	m1, ok := s.Books[strings.ToLower(exchange)]
	if !ok {
		return nil, fmt.Errorf("no orderbooks for %s exchange", exchange)
	}

	m2, ok := m1[a]
	if !ok {
		return nil, fmt.Errorf("no orderbooks associated with asset type %s",
			a)
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		return nil, fmt.Errorf("no orderbooks associated with base currency %s",
			p.Base)
	}

	book, ok := m3[p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("no orderbooks associated with base currency %s",
			p.Quote)
	}

	return book.Retrieve(), nil
}

// TotalBidsAmount returns the total amount of bids and the total orderbook
// bids value
func (b *Base) TotalBidsAmount() (amountCollated, total float64) {
	for x := range b.Bids {
		amountCollated += b.Bids[x].Amount
		total += b.Bids[x].Amount * b.Bids[x].Price
	}
	return amountCollated, total
}

// TotalAsksAmount returns the total amount of asks and the total orderbook
// asks value
func (b *Base) TotalAsksAmount() (amountCollated, total float64) {
	for y := range b.Asks {
		amountCollated += b.Asks[y].Amount
		total += b.Asks[y].Amount * b.Asks[y].Price
	}
	return amountCollated, total
}

// // Update updates the bids and asks
// func (b *Base) Update(bids, asks []Item) {
// 	b.Bids = bids
// 	b.Asks = asks
// 	b.LastUpdated = time.Now()
// }

// Verify ensures that the orderbook items are correctly sorted prior to being
// set and will reject any book with incorrect values.
// Bids should always go from a high price to a low price and
// Asks should always go from a low price to a higher price
func (b *Base) Verify() error {
	// Checking for both ask and bid lengths being zero has been removed and
	// a warning has been put in place some exchanges e.g. LakeBTC return zero
	// level books. In the event that there is a massive liquidity change where
	// a book dries up, this will still update so we do not traverse potential
	// incorrect old data.
	if len(b.Asks) == 0 || len(b.Bids) == 0 {
		log.Warnf(log.OrderBook,
			bookLengthIssue,
			b.Exchange,
			b.Pair,
			b.Asset,
			len(b.Bids),
			len(b.Asks))
	}
	for i := range b.Bids {
		if b.Bids[i].Price == 0 {
			return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, errPriceNotSet)
		}
		if b.Bids[i].Amount <= 0 {
			return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, errAmountInvalid)
		}
		if b.IsFundingRate && b.Bids[i].Period == 0 {
			return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, errPeriodUnset)
		}
		if i != 0 {
			if b.Bids[i].Price > b.Bids[i-1].Price {
				return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, errOutOfOrder)
			}

			if !b.NotAggregated && b.Bids[i].Price == b.Bids[i-1].Price {
				return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, errDuplication)
			}

			if b.Bids[i].ID != 0 && b.Bids[i].ID == b.Bids[i-1].ID {
				return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, errIDDuplication)
			}
		}
	}

	for i := range b.Asks {
		if b.Asks[i].Price == 0 {
			return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, errPriceNotSet)
		}
		if b.Asks[i].Amount <= 0 {
			return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, errAmountInvalid)
		}
		if b.IsFundingRate && b.Asks[i].Period == 0 {
			return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, errPeriodUnset)
		}
		if i != 0 {
			if b.Asks[i].Price < b.Asks[i-1].Price {
				return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, errOutOfOrder)
			}

			if !b.NotAggregated && b.Asks[i].Price == b.Asks[i-1].Price {
				return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, errDuplication)
			}

			if b.Asks[i].ID != 0 && b.Asks[i].ID == b.Asks[i-1].ID {
				return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, errIDDuplication)
			}
		}
	}
	return nil
}

// Process processes incoming orderbooks, creating or updating the orderbook
// list
func (b *Base) Process() error {
	if b.Exchange == "" {
		return errExchangeNameUnset
	}

	if b.Pair.IsEmpty() {
		return errPairNotSet
	}

	if b.Asset.String() == "" {
		return errAssetTypeNotSet
	}

	if b.LastUpdated.IsZero() {
		b.LastUpdated = time.Now()
	}

	if b.CanVerify() {
		err := b.Verify()
		if err != nil {
			return err
		}
	}
	return service.Update(b)
}

// CanVerify checks to see if orderbook should be verified or it is not required
func (b *Base) CanVerify() bool {
	return !b.VerificationBypass && !b.HasChecksumValidation
}

// Reverse reverses the order of orderbook items; some bid/asks are
// returned in either ascending or descending order. One bid or ask slice
// depending on whats received can be reversed. This is usually faster than
// using a sort algorithm as the algorithm could be impeded by a worst case time
// complexity when elements are shifted as opposed to just swapping element
// values.
func (elem *Items) Reverse() {
	eLen := len(*elem)
	var target int
	for i := eLen/2 - 1; i >= 0; i-- {
		target = eLen - 1 - i
		(*elem)[i], (*elem)[target] = (*elem)[target], (*elem)[i]
	}
}

// SortAsks sorts ask items to the correct ascending order if pricing values are
// scattered. If order from exchange is descending consider using the Reverse
// function.
func (elem *Items) SortAsks() {
	sort.Sort(byOBPrice(*elem))
}

// SortBids sorts bid items to the correct descending order if pricing values
// are scattered. If order from exchange is ascending consider using the Reverse
// function.
func (elem *Items) SortBids() {
	sort.Sort(sort.Reverse(byOBPrice(*elem)))
}
