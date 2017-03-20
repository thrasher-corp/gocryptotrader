package ticker

import (
	"errors"
	"strconv"
)

var (
	ErrTickerForExchangeNotFound = "Ticker for exchange does not exist."
	ErrPrimaryCurrencyNotFound   = "Error primary currency for ticker not found."
	ErrSecondaryCurrencyNotFound = "Error secondary currency for ticker not found."

	Tickers []Ticker
)

type TickerPrice struct {
	FirstCurrency  string  `json:"FirstCurrency"`
	SecondCurrency string  `json:"SecondCurrency"`
	CurrencyPair   string  `json:"CurrencyPair"`
	Last           float64 `json:"Last"`
	High           float64 `json:"High"`
	Low            float64 `json:"Low"`
	Bid            float64 `json:"Bid"`
	Ask            float64 `json:"Ask"`
	Volume         float64 `json:"Volume"`
	PriceATH       float64 `json:"PriceATH"`
}

type Ticker struct {
	Price        map[string]map[string]TickerPrice
	ExchangeName string
}

func (t *Ticker) PriceToString(firstCurrency, secondCurrency, priceType string) string {
	switch priceType {
	case "last":
		return strconv.FormatFloat(t.Price[firstCurrency][secondCurrency].Last, 'f', -1, 64)
	case "high":
		return strconv.FormatFloat(t.Price[firstCurrency][secondCurrency].High, 'f', -1, 64)
	case "low":
		return strconv.FormatFloat(t.Price[firstCurrency][secondCurrency].Low, 'f', -1, 64)
	case "bid":
		return strconv.FormatFloat(t.Price[firstCurrency][secondCurrency].Bid, 'f', -1, 64)
	case "ask":
		return strconv.FormatFloat(t.Price[firstCurrency][secondCurrency].Ask, 'f', -1, 64)
	case "volume":
		return strconv.FormatFloat(t.Price[firstCurrency][secondCurrency].Volume, 'f', -1, 64)
	case "ath":
		return strconv.FormatFloat(t.Price[firstCurrency][secondCurrency].PriceATH, 'f', -1, 64)
	default:
		return ""
	}
}

func GetTicker(exchange, firstCurrency, secondCurrency string) (TickerPrice, error) {
	ticker, err := GetTickerByExchange(exchange)
	if err != nil {
		return TickerPrice{}, err
	}

	if !FirstCurrencyExists(exchange, firstCurrency) {
		return TickerPrice{}, errors.New(ErrPrimaryCurrencyNotFound)
	}

	if !SecondCurrencyExists(exchange, firstCurrency, secondCurrency) {
		return TickerPrice{}, errors.New(ErrSecondaryCurrencyNotFound)
	}

	return ticker.Price[firstCurrency][secondCurrency], nil
}

func GetTickerByExchange(exchange string) (*Ticker, error) {
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			return &y, nil
		}
	}
	return nil, errors.New(ErrTickerForExchangeNotFound)
}

func FirstCurrencyExists(exchange, currency string) bool {
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			if _, ok := y.Price[currency]; ok {
				return true
			}
		}
	}
	return false
}

func SecondCurrencyExists(exchange, primary, secondary string) bool {
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			if _, ok := y.Price[primary]; ok {
				if _, ok := y.Price[primary][secondary]; ok {
					return true
				}
			}
		}
	}
	return false
}

func CreateNewTicker(exchangeName string, firstCurrency, secondCurrency string, tickerNew TickerPrice) Ticker {
	ticker := Ticker{}
	ticker.ExchangeName = exchangeName
	ticker.Price = make(map[string]map[string]TickerPrice)
	sMap := make(map[string]TickerPrice)
	sMap[secondCurrency] = tickerNew
	ticker.Price[firstCurrency] = sMap
	return ticker
}

func ProcessTicker(exchangeName string, firstCurrency, secondCurrency string, tickerNew TickerPrice) {
	tickerNew.CurrencyPair = tickerNew.FirstCurrency + tickerNew.SecondCurrency

	if len(Tickers) == 0 {
		CreateNewTicker(exchangeName, firstCurrency, secondCurrency, tickerNew)
		return
	} else {
		ticker, err := GetTickerByExchange(exchangeName)
		if err != nil {
			CreateNewTicker(exchangeName, firstCurrency, secondCurrency, tickerNew)
			return
		}

		if FirstCurrencyExists(exchangeName, firstCurrency) {
			if !SecondCurrencyExists(exchangeName, firstCurrency, secondCurrency) {
				second := ticker.Price[firstCurrency]
				second[secondCurrency] = tickerNew
				ticker.Price[firstCurrency] = second
				return
			}
		}
		sMap := make(map[string]TickerPrice)
		sMap[secondCurrency] = tickerNew
		ticker.Price[firstCurrency] = sMap
	}
}
