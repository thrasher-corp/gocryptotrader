package orderbook

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/dispatch"
)

// const values for orderbook package
const (
	errExchangeOrderbookNotFound = "orderbook for exchange does not exist"
	errPairNotSet                = "orderbook currency pair not set"
	errAssetTypeNotSet           = "orderbook asset type not set"
	errBaseCurrencyNotFound      = "orderbook base currency not found"
	errQuoteCurrencyNotFound     = "orderbook quote currency not found"
)

// Vars for the orderbook package
var (
	service *Service
)

func init() {
	service = &Service{
		Books: make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Book),
	}
}

// Service holds orderbook information for each individual exchange
type Service struct {
	Books map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Book
	sync.RWMutex
}

// Update stores orderbook data
func (s *Service) Update(b Base) error {
	s.Lock()
	if s.Books[b.ExchangeName] == nil {
		s.Books[b.ExchangeName] = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item] = make(map[*currency.Item]map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] = make(map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType] = &Book{Base: b}
		s.Unlock()
		return nil
	}

	if s.Books[b.ExchangeName][b.Pair.Base.Item] == nil {
		s.Books[b.ExchangeName][b.Pair.Base.Item] = make(map[*currency.Item]map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] = make(map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType] = &Book{Base: b}
		s.Unlock()
		return nil
	}

	if s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] == nil {
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] = make(map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType] = &Book{Base: b}
		s.Unlock()
		return nil
	}

	if s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType] == nil {
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType] = &Book{Base: b}
		s.Unlock()
		return nil
	}

	// Update cache and acquire publish ID
	s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType].Base = b
	id := s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType].ID
	s.Unlock()

	if id == (uuid.UUID{}) {
		return nil
	}

	// Publish update to dispatch system
	return dispatch.Publish(id, &b)
}

// Retrieve gets orderbook data from the slice
func (s *Service) Retrieve(exchange string, p currency.Pair, a asset.Item) (Base, error) {
	s.RLock()
	defer s.RUnlock()
	if s.Books[exchange] == nil {
		return Base{}, errors.New("no orderbooks for exchange")
	}

	if s.Books[exchange][p.Base.Item] == nil {
		return Base{}, errors.New("no orderbooks associated with base currency")
	}

	if s.Books[exchange][p.Base.Item][p.Quote.Item] == nil {
		return Base{}, errors.New("no orderbooks associated with quote currency")
	}

	if s.Books[exchange][p.Base.Item][p.Quote.Item][a] == nil {
		return Base{}, errors.New("no orderbooks associated with asset type")
	}

	return s.Books[exchange][p.Base.Item][p.Quote.Item][a].Base, nil
}

// GetID returns the uuid for the singulare orderbook
func (s *Service) GetID(exchange string, p currency.Pair, a asset.Item) (uuid.UUID, error) {
	s.Lock()
	defer s.Unlock()

	if s.Books[exchange] == nil {
		return uuid.UUID{}, errors.New("no orderbooks for exchange")
	}

	if s.Books[exchange][p.Base.Item] == nil {
		return uuid.UUID{}, errors.New("no orderbooks associated with base currency")
	}

	if s.Books[exchange][p.Base.Item][p.Quote.Item] == nil {
		return uuid.UUID{}, errors.New("no orderbooks associated with quote currency")
	}

	if s.Books[exchange][p.Base.Item][p.Quote.Item][a] == nil {
		return uuid.UUID{}, errors.New("no orderbooks associated with asset type")
	}

	if s.Books[exchange][p.Base.Item][p.Quote.Item][a].ID == (uuid.UUID{}) {
		id, err := dispatch.SetAndGetNewID()
		if err != nil {
			return uuid.UUID{}, err
		}
		s.Books[exchange][p.Base.Item][p.Quote.Item][a].ID = id
		return id, nil
	}
	return s.Books[exchange][p.Base.Item][p.Quote.Item][a].ID, nil
}

// Comms bra
type Comms struct {
	C  <-chan interface{}
	id uuid.UUID
}

// Release allows the channel to be released when the routine has finished
func (c *Comms) Release() error {
	return dispatch.Unsubscribe(c.id, c.C)
}

// SubscribeOrderbook subcribes to an orderbook and returns a communication
// channel to stream orderbook data updates
func SubscribeOrderbook(exchange string, p currency.Pair, a asset.Item) (*Comms, error) {
	id, err := service.GetID(exchange, p, a)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	c, err := dispatch.Subscribe(id)
	if err != nil {
		return nil, err
	}

	newComms := &Comms{
		C:  c.(<-chan interface{}),
		id: id,
	}

	return newComms, nil
}

// Book defines the full orderbook for an exchange, included is a dispatch ID
// to push updates as they come available
type Book struct {
	ID uuid.UUID
	Base
}

// Item stores the amount and price values
type Item struct {
	Amount float64
	Price  float64
	ID     int64
}

// Base holds the fields for the orderbook base
type Base struct {
	Pair         currency.Pair `json:"pair"`
	Bids         []Item        `json:"bids"`
	Asks         []Item        `json:"asks"`
	LastUpdated  time.Time     `json:"lastUpdated"`
	AssetType    asset.Item    `json:"assetType"`
	ExchangeName string        `json:"exchangeName"`
}

// TotalBidsAmount returns the total amount of bids and the total orderbook
// bids value
func (b *Base) TotalBidsAmount() (amountCollated, total float64) {
	for _, x := range b.Bids {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

// TotalAsksAmount returns the total amount of asks and the total orderbook
// asks value
func (b *Base) TotalAsksAmount() (amountCollated, total float64) {
	for _, x := range b.Asks {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

// Update updates the bids and asks
func (b *Base) Update(bids, asks []Item) {
	b.Bids = bids
	b.Asks = asks
	b.LastUpdated = time.Now()
}

type byOBPrice []Item

func (a byOBPrice) Len() int           { return len(a) }
func (a byOBPrice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byOBPrice) Less(i, j int) bool { return a[i].Price < a[j].Price }

// Verify ensures that the orderbook items are correctly sorted
// Bids should always go from a high price to a low price and
// asks should always go from a low price to a higher price
func (b *Base) Verify() {
	var lastPrice float64
	var sortBids, sortAsks bool
	for x := range b.Bids {
		if lastPrice != 0 && b.Bids[x].Price >= lastPrice {
			sortBids = true
			break
		}
		lastPrice = b.Bids[x].Price
	}

	lastPrice = 0
	for x := range b.Asks {
		if lastPrice != 0 && b.Asks[x].Price <= lastPrice {
			sortAsks = true
			break
		}
		lastPrice = b.Asks[x].Price
	}

	if sortBids {
		sort.Sort(sort.Reverse(byOBPrice(b.Bids)))
	}

	if sortAsks {
		sort.Sort((byOBPrice(b.Asks)))
	}
}

// Get checks and returns the orderbook given an exchange name and currency pair
// if it exists
func Get(exchange string, p currency.Pair, a asset.Item) (Base, error) {
	return service.Retrieve(exchange, p, a)
}

// Process processes incoming orderbooks, creating or updating the orderbook
// list
func (b *Base) Process() error {
	if b.ExchangeName == "" {
		return errors.New("exchange name unset")
	}
	if b.Pair.IsEmpty() {
		return errors.New("pair unset")
	}

	if b.AssetType.String() == "" {
		return errors.New("asset unset")
	}

	if len(b.Asks) == 0 && len(b.Bids) == 0 {
		return errors.New("no orderbook info")
	}

	if b.LastUpdated.IsZero() {
		b.LastUpdated = time.Now()
	}

	b.Verify()

	return service.Update(*b)
}
