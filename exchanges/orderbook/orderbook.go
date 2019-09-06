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
	service = new(Service)
	service.mux = dispatch.GetNewMux()
	service.Books = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Book)
	service.Exchange = make(map[string]uuid.UUID)
}

// Book defines an orderbook with its links to different dispatch outputs
type Book struct {
	b     *Base
	Main  uuid.UUID
	Assoc []uuid.UUID
}

// Service holds orderbook information for each individual exchange
type Service struct {
	Books    map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Book
	Exchange map[string]uuid.UUID
	mux      *dispatch.Mux
	sync.RWMutex
}

// Update stores orderbook data
func (s *Service) Update(b *Base) error {
	var ids []uuid.UUID

	s.Lock()
	switch {
	case s.Books[b.ExchangeName] == nil:
		s.Books[b.ExchangeName] = make(map[*currency.Item]map[*currency.Item]map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item] = make(map[*currency.Item]map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] = make(map[asset.Item]*Book)
		err := s.SetNewData(b)
		if err != nil {
			s.Unlock()
			return err
		}

	case s.Books[b.ExchangeName][b.Pair.Base.Item] == nil:
		s.Books[b.ExchangeName][b.Pair.Base.Item] = make(map[*currency.Item]map[asset.Item]*Book)
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] = make(map[asset.Item]*Book)
		err := s.SetNewData(b)
		if err != nil {
			s.Unlock()
			return err
		}

	case s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] == nil:
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item] = make(map[asset.Item]*Book)
		err := s.SetNewData(b)
		if err != nil {
			s.Unlock()
			return err
		}

	case s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType] == nil:
		err := s.SetNewData(b)
		if err != nil {
			s.Unlock()
			return err
		}

	default:
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType].b.Bids = b.Bids
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType].b.Asks = b.Asks
		s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType].b.LastUpdated = b.LastUpdated
		ids = s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType].Assoc
		ids = append(ids, s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType].Main)
	}
	s.Unlock()
	return s.mux.Publish(ids, b)
}

// SetNewData sets new data
func (s *Service) SetNewData(b *Base) error {
	ids, err := s.GetAssociations(b)
	if err != nil {
		return err
	}
	singleID, err := s.mux.GetID()
	if err != nil {
		return err
	}

	s.Books[b.ExchangeName][b.Pair.Base.Item][b.Pair.Quote.Item][b.AssetType] = &Book{b: b,
		Main:  singleID,
		Assoc: ids}
	return nil
}

// GetAssociations links a singular book with it's dispatch associations
func (s *Service) GetAssociations(b *Base) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	exchangeID, ok := s.Exchange[b.ExchangeName]
	if !ok {
		var err error
		exchangeID, err = s.mux.GetID()
		if err != nil {
			return nil, err
		}
		s.Exchange[b.ExchangeName] = exchangeID
	}

	ids = append(ids, exchangeID)
	return ids, nil
}

// Retrieve gets orderbook data from the slice
func (s *Service) Retrieve(exchange string, p currency.Pair, a asset.Item) (*Base, error) {
	s.RLock()
	defer s.RUnlock()
	if s.Books[exchange] == nil {
		return nil, fmt.Errorf("no orderbooks for %s exchange", exchange)
	}

	if s.Books[exchange][p.Base.Item] == nil {
		return nil, fmt.Errorf("no orderbooks associated with base currency %s",
			p.Base)
	}

	if s.Books[exchange][p.Base.Item][p.Quote.Item] == nil {
		return nil, fmt.Errorf("no orderbooks associated with quote currency %s",
			p.Quote)
	}

	if s.Books[exchange][p.Base.Item][p.Quote.Item][a] == nil {
		return nil, fmt.Errorf("no orderbooks associated with asset type %s",
			a)
	}

	return s.Books[exchange][p.Base.Item][p.Quote.Item][a].b, nil
}

// SubscribeOrderbook subcribes to an orderbook and returns a communication
// channel to stream orderbook data updates
func SubscribeOrderbook(exchange string, p currency.Pair, a asset.Item) (dispatch.Pipe, error) {
	service.RLock()
	defer service.RUnlock()
	if service.Books[exchange][p.Base.Item][p.Quote.Item][a] == nil {
		return dispatch.Pipe{}, fmt.Errorf("orderbook item not found for %s %s %s",
			exchange,
			p,
			a)
	}

	book, ok := service.Books[exchange][p.Base.Item][p.Quote.Item][a]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("orderbook item not found for %s %s %s",
			exchange,
			p,
			a)
	}

	return service.mux.Subscribe(book.Main)
}

// SubscribeToExchangeOrderbooks subcribes to all orderbooks on an exchange
func SubscribeToExchangeOrderbooks(exchange string) (dispatch.Pipe, error) {
	service.RLock()
	defer service.RUnlock()
	id, ok := service.Exchange[exchange]
	if !ok {
		return dispatch.Pipe{}, fmt.Errorf("%s exchange orderbooks not found",
			exchange)
	}

	return service.mux.Subscribe(id)
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
	o, err := service.Retrieve(exchange, p, a)
	if err != nil {
		return Base{}, err
	}
	return *o, nil
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

	return service.Update(b)
}
