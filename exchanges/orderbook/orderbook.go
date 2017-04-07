package orderbook

import (
	"errors"
	"time"
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
	FirstCurrency  string          `json:"first_currency"`
	SecondCurrency string          `json:"second_currency"`
	CurrencyPair   string          `json:"currency_pair"`
	Bids           []OrderbookItem `json:"bids"`
	Asks           []OrderbookItem `json:"asks"`
	LastUpdated    time.Time       `json:"last_updated"`
}

type Orderbook struct {
	Orderbook    map[string]map[string]OrderbookBase
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

func GetOrderbook(exchange, firstCurrency, secondCurrency string) (OrderbookBase, error) {
	orderbook, err := GetOrderbookByExchange(exchange)
	if err != nil {
		return OrderbookBase{}, err
	}

	if !FirstCurrencyExists(exchange, firstCurrency) {
		return OrderbookBase{}, errors.New(ErrPrimaryCurrencyNotFound)
	}

	if !SecondCurrencyExists(exchange, firstCurrency, secondCurrency) {
		return OrderbookBase{}, errors.New(ErrSecondaryCurrencyNotFound)
	}

	return orderbook.Orderbook[firstCurrency][secondCurrency], nil
}

func GetOrderbookByExchange(exchange string) (*Orderbook, error) {
	for _, y := range Orderbooks {
		if y.ExchangeName == exchange {
			return &y, nil
		}
	}
	return nil, errors.New(ErrOrderbookForExchangeNotFound)
}

func FirstCurrencyExists(exchange, currency string) bool {
	for _, y := range Orderbooks {
		if y.ExchangeName == exchange {
			if _, ok := y.Orderbook[currency]; ok {
				return true
			}
		}
	}
	return false
}

func SecondCurrencyExists(exchange, primary, secondary string) bool {
	for _, y := range Orderbooks {
		if y.ExchangeName == exchange {
			if _, ok := y.Orderbook[primary]; ok {
				if _, ok := y.Orderbook[primary][secondary]; ok {
					return true
				}
			}
		}
	}
	return false
}

func CreateNewOrderbook(exchangeName string, firstCurrency, secondCurrency string, orderbookNew OrderbookBase) Orderbook {
	orderbook := Orderbook{}
	orderbook.ExchangeName = exchangeName
	orderbook.Orderbook = make(map[string]map[string]OrderbookBase)
	sMap := make(map[string]OrderbookBase)
	sMap[secondCurrency] = orderbookNew
	orderbook.Orderbook[firstCurrency] = sMap
	return orderbook
}

func ProcessOrderbook(exchangeName string, firstCurrency, secondCurrency string, orderbookNew OrderbookBase) {
	orderbookNew.CurrencyPair = orderbookNew.FirstCurrency + orderbookNew.SecondCurrency

	if len(Orderbooks) == 0 {
		CreateNewOrderbook(exchangeName, firstCurrency, secondCurrency, orderbookNew)
		return
	} else {
		orderbook, err := GetOrderbookByExchange(exchangeName)
		if err != nil {
			CreateNewOrderbook(exchangeName, firstCurrency, secondCurrency, orderbookNew)
			return
		}

		if FirstCurrencyExists(exchangeName, firstCurrency) {
			if !SecondCurrencyExists(exchangeName, firstCurrency, secondCurrency) {
				second := orderbook.Orderbook[firstCurrency]
				second[secondCurrency] = orderbookNew
				orderbook.Orderbook[firstCurrency] = second
				return
			}
		}
		sMap := make(map[string]OrderbookBase)
		sMap[secondCurrency] = orderbookNew
		orderbook.Orderbook[firstCurrency] = sMap
	}
}
