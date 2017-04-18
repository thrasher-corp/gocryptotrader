package orderbook

import (
	"errors"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

var (
	ErrOrderbookForExchangeNotFound = "Ticker for exchange does not exist."
	ErrPrimaryCurrencyNotFound      = "Error primary currency for orderbook not found."
	ErrSecondaryCurrencyNotFound    = "Error secondary currency for orderbook not found."

	Orderbooks []Orderbook
)

type OrderbookItem struct {
	Amount float64
	Price  float64
}

type OrderbookBase struct {
	Pair         pair.CurrencyPair `json:"pair"`
	CurrencyPair string            `json:"CurrencyPair"`
	Bids         []OrderbookItem   `json:"bids"`
	Asks         []OrderbookItem   `json:"asks"`
	LastUpdated  time.Time         `json:"last_updated"`
}

type Orderbook struct {
	Orderbook    map[pair.CurrencyItem]map[pair.CurrencyItem]OrderbookBase
	ExchangeName string
}

func (o *OrderbookBase) CalculateTotalBids() (float64, float64) {
	amountCollated := float64(0)
	total := float64(0)
	for _, x := range o.Bids {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

func (o *OrderbookBase) CalculateTotalAsks() (float64, float64) {
	amountCollated := float64(0)
	total := float64(0)
	for _, x := range o.Asks {
		amountCollated += x.Amount
		total += x.Amount * x.Price
	}
	return amountCollated, total
}

func (o *OrderbookBase) Update(Bids, Asks []OrderbookItem) {
	o.Bids = Bids
	o.Asks = Asks
	o.LastUpdated = time.Now()
}

func GetOrderbook(exchange string, p pair.CurrencyPair) (OrderbookBase, error) {
	orderbook, err := GetOrderbookByExchange(exchange)
	if err != nil {
		return OrderbookBase{}, err
	}

	if !FirstCurrencyExists(exchange, p.GetFirstCurrency()) {
		return OrderbookBase{}, errors.New(ErrPrimaryCurrencyNotFound)
	}

	if !SecondCurrencyExists(exchange, p) {
		return OrderbookBase{}, errors.New(ErrSecondaryCurrencyNotFound)
	}

	return orderbook.Orderbook[p.GetFirstCurrency()][p.GetSecondCurrency()], nil
}

func GetOrderbookByExchange(exchange string) (*Orderbook, error) {
	for _, y := range Orderbooks {
		if y.ExchangeName == exchange {
			return &y, nil
		}
	}
	return nil, errors.New(ErrOrderbookForExchangeNotFound)
}

func FirstCurrencyExists(exchange string, currency pair.CurrencyItem) bool {
	for _, y := range Orderbooks {
		if y.ExchangeName == exchange {
			if _, ok := y.Orderbook[currency]; ok {
				return true
			}
		}
	}
	return false
}

func SecondCurrencyExists(exchange string, p pair.CurrencyPair) bool {
	for _, y := range Orderbooks {
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

func CreateNewOrderbook(exchangeName string, p pair.CurrencyPair, orderbookNew OrderbookBase) Orderbook {
	orderbook := Orderbook{}
	orderbook.ExchangeName = exchangeName
	orderbook.Orderbook = make(map[pair.CurrencyItem]map[pair.CurrencyItem]OrderbookBase)
	sMap := make(map[pair.CurrencyItem]OrderbookBase)
	sMap[p.GetSecondCurrency()] = orderbookNew
	orderbook.Orderbook[p.GetFirstCurrency()] = sMap
	Orderbooks = append(Orderbooks, orderbook)
	return orderbook
}

func ProcessOrderbook(exchangeName string, p pair.CurrencyPair, orderbookNew OrderbookBase) {
	orderbookNew.CurrencyPair = p.Pair().String()
	if len(Orderbooks) == 0 {
		CreateNewOrderbook(exchangeName, p, orderbookNew)
		return
	} else {
		orderbook, err := GetOrderbookByExchange(exchangeName)
		if err != nil {
			CreateNewOrderbook(exchangeName, p, orderbookNew)
			return
		}

		if FirstCurrencyExists(exchangeName, p.GetFirstCurrency()) {
			if !SecondCurrencyExists(exchangeName, p) {
				second := orderbook.Orderbook[p.GetFirstCurrency()]
				second[p.GetSecondCurrency()] = orderbookNew
				orderbook.Orderbook[p.GetFirstCurrency()] = second
				return
			}
		}
		sMap := make(map[pair.CurrencyItem]OrderbookBase)
		sMap[p.GetSecondCurrency()] = orderbookNew
		orderbook.Orderbook[p.GetFirstCurrency()] = sMap
	}
}
