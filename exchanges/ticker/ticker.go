package ticker

import (
	"errors"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

var (
	ErrTickerForExchangeNotFound = "Ticker for exchange does not exist."
	ErrPrimaryCurrencyNotFound   = "Error primary currency for ticker not found."
	ErrSecondaryCurrencyNotFound = "Error secondary currency for ticker not found."

	Tickers []Ticker
)

type TickerPrice struct {
	Pair         pair.CurrencyPair `json:"Pair"`
	CurrencyPair string            `json:"CurrencyPair"`
	Last         float64           `json:"Last"`
	High         float64           `json:"High"`
	Low          float64           `json:"Low"`
	Bid          float64           `json:"Bid"`
	Ask          float64           `json:"Ask"`
	Volume       float64           `json:"Volume"`
	PriceATH     float64           `json:"PriceATH"`
}

type Ticker struct {
	Price        map[pair.CurrencyItem]map[pair.CurrencyItem]TickerPrice
	ExchangeName string
}

func (t *Ticker) PriceToString(p pair.CurrencyPair, priceType string) string {
	priceType = common.StringToLower(priceType)
	switch priceType {
	case "last":
		return strconv.FormatFloat(t.Price[p.GetFirstCurrency()][p.GetSecondCurrency()].Last, 'f', -1, 64)
	case "high":
		return strconv.FormatFloat(t.Price[p.GetFirstCurrency()][p.GetSecondCurrency()].High, 'f', -1, 64)
	case "low":
		return strconv.FormatFloat(t.Price[p.GetFirstCurrency()][p.GetSecondCurrency()].Low, 'f', -1, 64)
	case "bid":
		return strconv.FormatFloat(t.Price[p.GetFirstCurrency()][p.GetSecondCurrency()].Bid, 'f', -1, 64)
	case "ask":
		return strconv.FormatFloat(t.Price[p.GetFirstCurrency()][p.GetSecondCurrency()].Ask, 'f', -1, 64)
	case "volume":
		return strconv.FormatFloat(t.Price[p.GetFirstCurrency()][p.GetSecondCurrency()].Volume, 'f', -1, 64)
	case "ath":
		return strconv.FormatFloat(t.Price[p.GetFirstCurrency()][p.GetSecondCurrency()].PriceATH, 'f', -1, 64)
	default:
		return ""
	}
}

func GetTicker(exchange string, p pair.CurrencyPair) (TickerPrice, error) {
	ticker, err := GetTickerByExchange(exchange)
	if err != nil {
		return TickerPrice{}, err
	}

	if !FirstCurrencyExists(exchange, p.GetFirstCurrency()) {
		return TickerPrice{}, errors.New(ErrPrimaryCurrencyNotFound)
	}

	if !SecondCurrencyExists(exchange, p) {
		return TickerPrice{}, errors.New(ErrSecondaryCurrencyNotFound)
	}

	return ticker.Price[p.GetFirstCurrency()][p.GetSecondCurrency()], nil
}

func GetTickerByExchange(exchange string) (*Ticker, error) {
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			return &y, nil
		}
	}
	return nil, errors.New(ErrTickerForExchangeNotFound)
}

func FirstCurrencyExists(exchange string, currency pair.CurrencyItem) bool {
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			if _, ok := y.Price[currency]; ok {
				return true
			}
		}
	}
	return false
}

func SecondCurrencyExists(exchange string, p pair.CurrencyPair) bool {
	for _, y := range Tickers {
		if y.ExchangeName == exchange {
			if _, ok := y.Price[p.GetFirstCurrency()]; ok {
				if _, ok := y.Price[p.GetFirstCurrency()][p.GetSecondCurrency()]; ok {
					return true
				}
			}
		}
	}
	return false
}

func CreateNewTicker(exchangeName string, p pair.CurrencyPair, tickerNew TickerPrice) Ticker {
	ticker := Ticker{}
	ticker.ExchangeName = exchangeName
	ticker.Price = make(map[pair.CurrencyItem]map[pair.CurrencyItem]TickerPrice)
	sMap := make(map[pair.CurrencyItem]TickerPrice)
	sMap[p.GetSecondCurrency()] = tickerNew
	ticker.Price[p.GetFirstCurrency()] = sMap
	Tickers = append(Tickers, ticker)
	return ticker
}

func ProcessTicker(exchangeName string, p pair.CurrencyPair, tickerNew TickerPrice) {
	tickerNew.CurrencyPair = p.Pair().String()
	if len(Tickers) == 0 {
		CreateNewTicker(exchangeName, p, tickerNew)
		//issue - not appending
		return
	} else {
		ticker, err := GetTickerByExchange(exchangeName)
		if err != nil {
			CreateNewTicker(exchangeName, p, tickerNew)
			return
		}

		if FirstCurrencyExists(exchangeName, p.GetFirstCurrency()) {
			if !SecondCurrencyExists(exchangeName, p) {
				second := ticker.Price[p.GetFirstCurrency()]
				second[p.GetSecondCurrency()] = tickerNew
				ticker.Price[p.GetFirstCurrency()] = second
				return
			}
		}
		sMap := make(map[pair.CurrencyItem]TickerPrice)
		sMap[p.GetSecondCurrency()] = tickerNew
		ticker.Price[p.GetFirstCurrency()] = sMap
	}
}
