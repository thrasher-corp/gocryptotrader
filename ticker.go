package main

import (
	"strconv"
)

type TickerPrice struct {
	CryptoCurrency string  `json:"CryptoCurrency"`
	FiatCurrency   string  `json:"FiatCurrency"`
	Last           float64 `json:"Last"`
	High           float64 `json:"High"`
	Low            float64 `json:"Low"`
	Bid            float64 `json:"Bid"`
	Ask            float64 `json:"Ask"`
	Volume         float64 `json:"Volume"`
}

type Ticker struct {
	Price        map[string]map[string]TickerPrice
	ExchangeName string
}

func (t *Ticker) PriceToString(cryptoCurrency, fiatCurrency, priceType string) string {
	switch priceType {
	case "last":
		return strconv.FormatFloat(t.Price[cryptoCurrency][fiatCurrency].Last, 'f', -1, 64)
	case "high":
		return strconv.FormatFloat(t.Price[cryptoCurrency][fiatCurrency].High, 'f', -1, 64)
	case "low":
		return strconv.FormatFloat(t.Price[cryptoCurrency][fiatCurrency].Low, 'f', -1, 64)
	case "bid":
		return strconv.FormatFloat(t.Price[cryptoCurrency][fiatCurrency].Bid, 'f', -1, 64)
	case "ask":
		return strconv.FormatFloat(t.Price[cryptoCurrency][fiatCurrency].Ask, 'f', -1, 64)
	case "volume":
		return strconv.FormatFloat(t.Price[cryptoCurrency][fiatCurrency].Volume, 'f', -1, 64)
	default:
		return ""
	}
}

func AddTickerPrice(m map[string]map[string]TickerPrice, cryptocurrency, fiatcurrency string, price TickerPrice) {
	mm, ok := m[cryptocurrency]
	if !ok {
		mm = make(map[string]TickerPrice)
		m[cryptocurrency] = mm
	}
	mm[fiatcurrency] = price
}

func NewTicker(exchangeName string, prices []TickerPrice) *Ticker {
	ticker := &Ticker{}
	ticker.ExchangeName = exchangeName
	ticker.Price = make(map[string]map[string]TickerPrice, 0)

	for x, _ := range prices {
		AddTickerPrice(ticker.Price, prices[x].CryptoCurrency, prices[x].FiatCurrency, prices[x])
	}

	return ticker
}
