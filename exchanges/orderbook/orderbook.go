package orderbook

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/asset"
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
	Orderbooks []Orderbook
	m          sync.Mutex
)

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

// Orderbook holds the orderbook information for a currency pair and type
type Orderbook struct {
	Orderbook    map[*currency.Item]map[*currency.Item]map[asset.Item]Base
	ExchangeName string
}

// TotalBidsAmount returns the total amount of bids and the total orderbook
// bids value
func (o *Base) TotalBidsAmount() (amountCollated, total float64) {
	for _, x := range o.Bids {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

// TotalAsksAmount returns the total amount of asks and the total orderbook
// asks value
func (o *Base) TotalAsksAmount() (amountCollated, total float64) {
	for _, x := range o.Asks {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

// Update updates the bids and asks
func (o *Base) Update(bids, asks []Item) {
	o.Bids = bids
	o.Asks = asks
	o.LastUpdated = time.Now()
}

// Get checks and returns the orderbook given an exchange name and currency pair
// if it exists
func Get(exchange string, p currency.Pair, orderbookType asset.Item) (Base, error) {
	orderbook, err := GetByExchange(exchange)
	if err != nil {
		return Base{}, err
	}

	if !BaseCurrencyExists(exchange, p.Base) {
		return Base{}, errors.New(errBaseCurrencyNotFound)
	}

	if !QuoteCurrencyExists(exchange, p) {
		return Base{}, errors.New(errQuoteCurrencyNotFound)
	}

	return orderbook.Orderbook[p.Base.Item][p.Quote.Item][orderbookType], nil
}

// GetByExchange returns an exchange orderbook
func GetByExchange(exchange string) (*Orderbook, error) {
	m.Lock()
	defer m.Unlock()
	for x := range Orderbooks {
		if Orderbooks[x].ExchangeName == exchange {
			return &Orderbooks[x], nil
		}
	}
	return nil, errors.New(errExchangeOrderbookNotFound)
}

// BaseCurrencyExists checks to see if the base currency of the orderbook map
// exists
func BaseCurrencyExists(exchange string, currency currency.Code) bool {
	m.Lock()
	defer m.Unlock()
	for _, y := range Orderbooks {
		if y.ExchangeName == exchange {
			if _, ok := y.Orderbook[currency.Item]; ok {
				return true
			}
		}
	}
	return false
}

// QuoteCurrencyExists checks to see if the quote currency of the orderbook
// map exists
func QuoteCurrencyExists(exchange string, p currency.Pair) bool {
	m.Lock()
	defer m.Unlock()
	for _, y := range Orderbooks {
		if y.ExchangeName == exchange {
			if _, ok := y.Orderbook[p.Base.Item]; ok {
				if _, ok := y.Orderbook[p.Base.Item][p.Quote.Item]; ok {
					return true
				}
			}
		}
	}
	return false
}

// CreateNewOrderbook creates a new orderbook
func CreateNewOrderbook(exchangeName string, orderbookNew *Base, orderbookType asset.Item) *Orderbook {
	m.Lock()
	defer m.Unlock()
	orderbook := Orderbook{}
	orderbook.ExchangeName = exchangeName
	orderbook.Orderbook = make(map[*currency.Item]map[*currency.Item]map[asset.Item]Base)
	a := make(map[*currency.Item]map[asset.Item]Base)
	b := make(map[asset.Item]Base)
	b[orderbookType] = *orderbookNew
	a[orderbookNew.Pair.Quote.Item] = b
	orderbook.Orderbook[orderbookNew.Pair.Base.Item] = a
	Orderbooks = append(Orderbooks, orderbook)
	return &orderbook
}

// Process processes incoming orderbooks, creating or updating the orderbook
// list
func (o *Base) Process() error {
	if o.Pair.IsEmpty() {
		return errors.New(errPairNotSet)
	}

	if o.AssetType == "" {
		return errors.New(errAssetTypeNotSet)
	}

	if o.LastUpdated.IsZero() {
		o.LastUpdated = time.Now()
	}

	orderbook, err := GetByExchange(o.ExchangeName)
	if err != nil {
		CreateNewOrderbook(o.ExchangeName, o, o.AssetType)
		return nil
	}

	if BaseCurrencyExists(o.ExchangeName, o.Pair.Base) {
		m.Lock()
		a := make(map[asset.Item]Base)
		a[o.AssetType] = *o
		orderbook.Orderbook[o.Pair.Base.Item][o.Pair.Quote.Item] = a
		m.Unlock()
		return nil
	}

	m.Lock()
	a := make(map[*currency.Item]map[asset.Item]Base)
	b := make(map[asset.Item]Base)
	b[o.AssetType] = *o
	a[o.Pair.Quote.Item] = b
	orderbook.Orderbook[o.Pair.Base.Item] = a
	m.Unlock()
	return nil
}
