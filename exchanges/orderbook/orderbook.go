package orderbook

import (
	"errors"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

// Const values for orderbook package
const (
	ErrOrderbookForExchangeNotFound = "Ticker for exchange does not exist."
	ErrPrimaryCurrencyNotFound      = "Error primary currency for orderbook not found."
	ErrSecondaryCurrencyNotFound    = "Error secondary currency for orderbook not found."

	Spot = "SPOT"
)

// Stores the order books, and provides helper methods
type Orderbooks struct {
	orderbooks []Orderbook
}

// Item stores the amount and price values
type Item struct {
	Amount float64
	Price  float64
}

// Base holds the fields for the orderbook base
type Base struct {
	Pair         pair.CurrencyPair `json:"pair"`
	CurrencyPair string            `json:"CurrencyPair"`
	Bids         []Item            `json:"bids"`
	Asks         []Item            `json:"asks"`
	LastUpdated  time.Time         `json:"last_updated"`
}

// Orderbook holds the orderbook information for a currency pair and type
type Orderbook struct {
	Orderbook    map[pair.CurrencyItem]map[pair.CurrencyItem]map[string]Base
	ExchangeName string
}

// CalculateTotalBids returns the total amount of bids and the total orderbook
// bids value
func (o *Base) CalculateTotalBids() (float64, float64) {
	amountCollated := float64(0)
	total := float64(0)
	for _, x := range o.Bids {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

// CalculateTotalAsks returns the total amount of asks and the total orderbook
// asks value
func (o *Base) CalculateTotalAsks() (float64, float64) {
	amountCollated := float64(0)
	total := float64(0)
	for _, x := range o.Asks {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

// Update updates the bids and asks
func (o *Base) Update(Bids, Asks []Item) {
	o.Bids = Bids
	o.Asks = Asks
	o.LastUpdated = time.Now()
}

// GetOrderbook checks and returns the orderbook given an exchange name and
// currency pair if it exists
func (o *Orderbooks) GetOrderbook(exchange string, p pair.CurrencyPair, orderbookType string) (Base, error) {
	orderbook, err := o.GetOrderbookByExchange(exchange)
	if err != nil {
		return Base{}, err
	}

	if !o.FirstCurrencyExists(exchange, p.GetFirstCurrency()) {
		return Base{}, errors.New(ErrPrimaryCurrencyNotFound)
	}

	if !o.SecondCurrencyExists(exchange, p) {
		return Base{}, errors.New(ErrSecondaryCurrencyNotFound)
	}

	return orderbook.Orderbook[p.GetFirstCurrency()][p.GetSecondCurrency()][orderbookType], nil
}

// GetOrderbookByExchange returns an exchange orderbook
func (o *Orderbooks) GetOrderbookByExchange(exchange string) (*Orderbook, error) {
	for _, y := range o.orderbooks {
		if y.ExchangeName == exchange {
			return &y, nil
		}
	}
	return nil, errors.New(ErrOrderbookForExchangeNotFound)
}

// FirstCurrencyExists checks to see if the first currency of the orderbook map
// exists
func (o *Orderbooks) FirstCurrencyExists(exchange string, currency pair.CurrencyItem) bool {
	for _, y := range o.orderbooks {
		if y.ExchangeName == exchange {
			if _, ok := y.Orderbook[currency]; ok {
				return true
			}
		}
	}
	return false
}

// SecondCurrencyExists checks to see if the second currency of the orderbook
// map exists
func (o *Orderbooks) SecondCurrencyExists(exchange string, p pair.CurrencyPair) bool {
	for _, y := range o.orderbooks {
		if y.ExchangeName == exchange {
			if _, ok := y.Orderbook[p.GetFirstCurrency()]; ok {
				if _, ok := y.Orderbook[p.GetFirstCurrency()][p.GetSecondCurrency()]; ok {
					return true
				}
			}
		}
	}
	return false
}

// CreateNewOrderbook creates a new orderbook
func (o *Orderbooks) CreateNewOrderbook(exchangeName string, p pair.CurrencyPair, orderbookNew Base, orderbookType string) Orderbook {
	orderbook := Orderbook{}
	orderbook.ExchangeName = exchangeName
	orderbook.Orderbook = make(map[pair.CurrencyItem]map[pair.CurrencyItem]map[string]Base)
	a := make(map[pair.CurrencyItem]map[string]Base)
	b := make(map[string]Base)
	b[orderbookType] = orderbookNew
	a[p.SecondCurrency] = b
	orderbook.Orderbook[p.FirstCurrency] = a
	o.orderbooks = append(o.orderbooks, orderbook)
	return orderbook
}

// ProcessOrderbook processes incoming orderbooks, creating or updating the
// Orderbook list
func (o *Orderbooks) ProcessOrderbook(exchangeName string, p pair.CurrencyPair, orderbookNew Base, orderbookType string) {
	orderbookNew.CurrencyPair = p.Pair().String()
	orderbookNew.LastUpdated = time.Now()

	if len(o.orderbooks) == 0 {
		o.CreateNewOrderbook(exchangeName, p, orderbookNew, orderbookType)
		return
	}

	orderbook, err := o.GetOrderbookByExchange(exchangeName)
	if err != nil {
		o.CreateNewOrderbook(exchangeName, p, orderbookNew, orderbookType)
		return
	}

	if o.FirstCurrencyExists(exchangeName, p.GetFirstCurrency()) {
		if !o.SecondCurrencyExists(exchangeName, p) {
			a := orderbook.Orderbook[p.FirstCurrency]
			b := make(map[string]Base)
			b[orderbookType] = orderbookNew
			a[p.SecondCurrency] = b
			orderbook.Orderbook[p.FirstCurrency] = a
			return
		}
	}

	a := make(map[pair.CurrencyItem]map[string]Base)
	b := make(map[string]Base)
	b[orderbookType] = orderbookNew
	a[p.SecondCurrency] = b
	orderbook.Orderbook[p.FirstCurrency] = a
}

// Init creates a new set of Orderbooks
func Init() Orderbooks {
	return Orderbooks{}
}
