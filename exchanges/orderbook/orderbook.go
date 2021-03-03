package orderbook

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
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

// GetDepth returns depth
func DeployDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	return service.DeployDepth(exchange, p, a)
}

// Update stores orderbook data
func (s *Service) Update(b *Base) error {
	name := strings.ToLower(b.Exchange)
	s.Lock()
	m1, ok := s.books[name]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Depth)
		s.books[name] = m1
	}

	m2, ok := m1[b.Asset]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*Depth)
		m1[b.Asset] = m2
	}

	m3, ok := m2[b.Pair.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*Depth)
		m2[b.Pair.Base.Item] = m3
	}

	book, ok := m3[b.Pair.Quote.Item]
	if !ok {
		book = new(Depth)
		book.Exchange = b.Exchange
		book.Asset = b.Asset
		book.Pair = b.Pair
		book.RestSnapshot = b.RestSnapshot
		book.IsFundingRate = b.IsFundingRate
		book.HasChecksumValidation = b.HasChecksumValidation
		book.NotAggregated = b.NotAggregated
		m3[b.Pair.Quote.Item] = book
	}
	book.LastUpdated = b.LastUpdated
	book.LastUpdateID = b.LastUpdateID
	book.RestSnapshot = true
	err := book.Process(b.Bids, b.Asks)
	s.Unlock()
	return err
}

// DeployDepth used for subsystem deployment creates a depth item in the struct
// then returns a ptr to that Depth item
func (s *Service) DeployDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	s.Lock()
	defer s.Unlock()
	m1, ok := s.books[strings.ToLower(exchange)]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Depth)
		s.books[strings.ToLower(exchange)] = m1
	}
	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*Depth)
		m1[a] = m2
	}
	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*Depth)
		m2[p.Base.Item] = m3
	}
	book, ok := m3[p.Quote.Item]
	if !ok {
		book = new(Depth)
		m3[p.Quote.Item] = book
	}
	return book, nil
}

// GetDepth returns the actual depth struct for potential subsystems and
// strategies to interract with
func (s *Service) GetDepth(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	s.Lock()
	defer s.Unlock()
	m1, ok := s.books[strings.ToLower(exchange)]
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
	return book, nil
}

// Retrieve gets orderbook depth data from the associated linked list and
// returns the base equivalent copy
func (s *Service) Retrieve(exchange string, p currency.Pair, a asset.Item) (*Base, error) {
	s.Lock()
	defer s.Unlock()
	m1, ok := s.books[strings.ToLower(exchange)]
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
	err := checkAlignment(b.Bids, b.IsFundingRate, b.NotAggregated, dsc)
	if err != nil {
		return fmt.Errorf(bidLoadBookFailure, b.Exchange, b.Pair, b.Asset, err)
	}
	err = checkAlignment(b.Asks, b.IsFundingRate, b.NotAggregated, asc)
	if err != nil {
		return fmt.Errorf(askLoadBookFailure, b.Exchange, b.Pair, b.Asset, err)
	}
	return nil
}

// checker defines specific functionality to determine ascending/descending
// validation
type checker func(current Item, previous Item) error

// asc specfically defines ascending price check
var asc = func(current Item, previous Item) error {
	if current.Price < current.Price {
		return errPriceOutOfOrder
	}
	return nil
}

// dsc specfically defines descending price check
var dsc = func(current Item, previous Item) error {
	if current.Price > current.Price {
		return errPriceOutOfOrder
	}
	return nil
}

// checkAlignment validates full orderbook
func checkAlignment(depth Items, fundingRate, notAggregated bool, c checker) error {
	for i := range depth {
		if depth[i].Price == 0 {
			return errPriceNotSet
		}
		if depth[i].Amount <= 0 {
			return errAmountInvalid
		}
		if fundingRate && depth[i].Period == 0 {
			return errPeriodUnset
		}
		if i != 0 {
			prev := i - 1
			if err := c(depth[i], depth[prev]); err != nil {
				return err
			}
			if depth[i].ID < depth[prev].ID {
				return errIDOutOfOrder
			}
			if !notAggregated && depth[i].Price == depth[prev].Price {
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
