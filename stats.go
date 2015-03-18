package main

import (
	"sort"
)

type ExchangeInfo struct {
	Exchange string
	Currency string
	Price float64
	Volume float64
}

var ExchInfo []ExchangeInfo

type ByPrice []ExchangeInfo

func (this ByPrice) Len() int {
	return len(this)
}

func (this ByPrice) Less(i, j int) bool {
	return this[i].Price < this[j].Price
}

func (this ByPrice) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

type ByVolume []ExchangeInfo

func (this ByVolume) Len() int {
	return len(this)
}

func (this ByVolume) Less(i, j int) bool {
	return this[i].Volume < this[j].Volume
}

func (this ByVolume) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func AddExchangeInfo(exchange, currency string, price, volume float64) {
	if len(ExchInfo) == 0 {
		AppendExchangeInfo(exchange, currency, price, volume)
	} else {
		if ExchangeInfoAlreadyExists(exchange, currency, price, volume) {
			return
		} else {
			AppendExchangeInfo(exchange, currency, price, volume)
		}
	}
}

func AppendExchangeInfo(exchange, currency string, price, volume float64) {
	exch := ExchangeInfo{}
	exch.Exchange = exchange
	exch.Currency = currency
	exch.Price = price
	exch.Volume = volume
	ExchInfo = append(ExchInfo, exch)
}

func ExchangeInfoAlreadyExists(exchange, currency string, price, volume float64) (bool) {
	for i, _ := range ExchInfo {
		if  ExchInfo[i].Exchange == exchange && ExchInfo[i].Currency == currency {
			ExchInfo[i].Price, ExchInfo[i].Volume = price, volume
			return true
		}
	}
	return false
}

func SortExchangesByVolume(currency string, reverse bool) []ExchangeInfo {
	info := []ExchangeInfo{}

	for _, x := range ExchInfo {
		if x.Currency == currency {
			info = append(info, x)
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(ByVolume(info)))
	} else {
		sort.Sort(ByVolume(info))
	}
	return info
}

func SortExchangesByPrice(currency string, reverse bool) []ExchangeInfo {
	info := []ExchangeInfo{}

	for _, x := range ExchInfo {
		if x.Currency == currency {
			info = append(info, x)
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(ByPrice(info)))
	} else {
		sort.Sort(ByPrice(info))
	}
	return info
}
