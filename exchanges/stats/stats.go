package stats

import (
	"sort"

	"github.com/thrasher-/gocryptotrader/currency"
)

type ExchangeInfo struct {
	Exchange      string
	FirstCurrency string
	FiatCurrency  string
	Price         float64
	Volume        float64
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

func AddExchangeInfo(exchange, crypto, fiat string, price, volume float64) {
	if currency.BaseCurrencies == "" {
		currency.BaseCurrencies = currency.DEFAULT_CURRENCIES
	}

	if !currency.IsFiatCurrency(fiat) {
		return
	}
	AppendExchangeInfo(exchange, crypto, fiat, price, volume)

}

func AppendExchangeInfo(exchange, crypto, fiat string, price, volume float64) {
	if ExchangeInfoAlreadyExists(exchange, crypto, fiat, price, volume) {
		return
	}

	exch := ExchangeInfo{}
	exch.Exchange = exchange
	exch.FirstCurrency = crypto
	exch.FiatCurrency = fiat
	exch.Price = price
	exch.Volume = volume
	ExchInfo = append(ExchInfo, exch)
}

func ExchangeInfoAlreadyExists(exchange, crypto, fiat string, price, volume float64) bool {
	for i, _ := range ExchInfo {
		if ExchInfo[i].Exchange == exchange && ExchInfo[i].FirstCurrency == crypto && ExchInfo[i].FiatCurrency == fiat {
			ExchInfo[i].Price, ExchInfo[i].Volume = price, volume
			return true
		}
	}
	return false
}

func SortExchangesByVolume(crypto, fiat string, reverse bool) []ExchangeInfo {
	info := []ExchangeInfo{}

	for _, x := range ExchInfo {
		if x.FirstCurrency == crypto && x.FiatCurrency == fiat {
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

func SortExchangesByPrice(crypto, fiat string, reverse bool) []ExchangeInfo {
	info := []ExchangeInfo{}

	for _, x := range ExchInfo {
		if x.FirstCurrency == crypto && x.FiatCurrency == fiat {
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
