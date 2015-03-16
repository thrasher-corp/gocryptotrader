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
	exch := ExchangeInfo{}
	exch.Exchange = exchange
	exch.Currency = currency
	exch.Price = price
	exch.Volume = volume
	ExchInfo = append(ExchInfo, exch)
}

func SortExchangesByVolume(currency string) []ExchangeInfo {
	info := []ExchangeInfo{}

	for _, x := range ExchInfo {
		if x.Currency == currency {
			info = append(info, x)
		}
	}

	sort.Sort(ByVolume(info))
	return info
}

func SortExchangesByPrice(currency string) []ExchangeInfo {
	info := []ExchangeInfo{}

	for _, x := range ExchInfo {
		if x.Currency == currency {
			info = append(info, x)
		}
	}

	sort.Sort(ByPrice(info))
	return info
}

func PopulateExchangeInfo() {
	if bot.exchange.bitfinex.IsEnabled() {
		BitfinexBTC := bot.exchange.bitfinex.GetTicker("BTCUSD")
		BitfinexLTC := bot.exchange.bitfinex.GetTicker("LTCUSD")
		AddExchangeInfo(bot.exchange.bitfinex.GetName(), "BTC", BitfinexBTC.Last, BitfinexBTC.Volume)
		AddExchangeInfo(bot.exchange.bitfinex.GetName(), "LTC", BitfinexLTC.Last, BitfinexLTC.Volume)
	}
	if bot.exchange.bitstamp.IsEnabled() {
		BitstampBTC := bot.exchange.bitstamp.GetTicker()
		AddExchangeInfo(bot.exchange.bitstamp.GetName(), "BTC", BitstampBTC.Last, BitstampBTC.Volume)
	}
	if bot.exchange.btcchina.IsEnabled() {
		BTCChinaBTC := bot.exchange.btcchina.GetTicker("btccny")
		BTCChinaLTC := bot.exchange.btcchina.GetTicker("ltccny")
		BTCChinaBTCLastUSD, _ := ConvertCurrency(BTCChinaBTC.Last, "CNY", "USD")
		BTCChinaLTCLastUSD, _ := ConvertCurrency(BTCChinaLTC.Last, "CNY", "USD")
		AddExchangeInfo(bot.exchange.btcchina.GetName(), "BTC", BTCChinaBTCLastUSD, BTCChinaBTC.Vol)
		AddExchangeInfo(bot.exchange.btcchina.GetName(), "LTC", BTCChinaLTCLastUSD, BTCChinaLTC.Vol)
	}
	if bot.exchange.btce.IsEnabled() {
		BTCEBTC := bot.exchange.btce.GetTicker("btc_usd")
		BTCELTC := bot.exchange.btce.GetTicker("ltc_usd")
		AddExchangeInfo(bot.exchange.btce.GetName(), "BTC", BTCEBTC.Last, BTCEBTC.Vol_cur)
		AddExchangeInfo(bot.exchange.btce.GetName(), "LTC", BTCELTC.Last, BTCELTC.Vol_cur)
	}
	if bot.exchange.btcmarkets.IsEnabled() {
		BTCMarketsBTC := bot.exchange.btcmarkets.GetTicker("BTC")
		BTCMarketsLTC := bot.exchange.btcmarkets.GetTicker("LTC")
		BTCMarketsBTCUSD, _ := ConvertCurrency(BTCMarketsBTC.LastPrice, "AUD", "USD")
		BTCMarketsLTCUSD, _ := ConvertCurrency(BTCMarketsLTC.LastPrice, "AUD", "USD")
		AddExchangeInfo(bot.exchange.btcmarkets.GetName(), "BTC", BTCMarketsBTCUSD, 0)
		AddExchangeInfo(bot.exchange.btcmarkets.GetName(), "LTC", BTCMarketsLTCUSD, 0)
	}
	if bot.exchange.coinbase.IsEnabled() {
		CoinbaseBTC := bot.exchange.coinbase.GetTicker("BTC-USD")
		CoinbaseStats := bot.exchange.coinbase.GetStats("BTC-USD")
		AddExchangeInfo(bot.exchange.coinbase.GetName(), "BTC",  CoinbaseBTC.Price, CoinbaseStats.Volume)
	}
	if bot.exchange.huobi.IsEnabled() {
		HuobiBTC := bot.exchange.huobi.GetTicker("btc")
		HuobiLTC := bot.exchange.huobi.GetTicker("ltc")
		HuobiBTCLastUSD, _ := ConvertCurrency(HuobiBTC.Last,  "CNY", "USD")
		HuobiLTCLastUSD, _ := ConvertCurrency(HuobiLTC.Last,  "CNY", "USD")
		AddExchangeInfo(bot.exchange.huobi.GetName(), "BTC", HuobiBTCLastUSD, HuobiBTC.Vol)
		AddExchangeInfo(bot.exchange.huobi.GetName(), "LTC", HuobiLTCLastUSD, HuobiLTC.Vol)
	}
	if bot.exchange.itbit.IsEnabled() {
		itbitBTC := bot.exchange.itbit.GetTicker("XBTUSD")
		AddExchangeInfo(bot.exchange.itbit.GetName(), "BTC", itbitBTC.LastPrice, itbitBTC.Volume24h)
	}
	if bot.exchange.okcoinIntl.IsEnabled() {
		okcoinIntlBTC := bot.exchange.okcoinIntl.GetTicker("btc_usd")
		okcoinIntlLTC := bot.exchange.okcoinIntl.GetTicker("ltc_usd")
		AddExchangeInfo(bot.exchange.okcoinIntl.GetName(), "BTC", okcoinIntlBTC.Last, okcoinIntlBTC.Vol)
		AddExchangeInfo(bot.exchange.okcoinIntl.GetName(), "LTC", okcoinIntlLTC.Last, okcoinIntlLTC.Vol)
	}
	if bot.exchange.okcoinChina.IsEnabled() {
		okcoinChinaBTC := bot.exchange.okcoinChina.GetTicker("btc_cny")
		okcoinChinaLTC := bot.exchange.okcoinChina.GetTicker("ltc_cny")
		okcoinChinaBTCLast, _ := ConvertCurrency(okcoinChinaBTC.Last, "CNY", "USD")
		okcoinChinaLTCLast, _ := ConvertCurrency(okcoinChinaLTC.Last, "CNY", "USD")
		AddExchangeInfo(bot.exchange.okcoinChina.GetName(), "BTC", okcoinChinaBTCLast, okcoinChinaBTC.Vol)
		AddExchangeInfo(bot.exchange.okcoinChina.GetName(), "LTC", okcoinChinaLTCLast, okcoinChinaLTC.Vol)
	}
	if bot.exchange.lakebtc.IsEnabled() {
		LakeBTC := bot.exchange.lakebtc.GetTicker()
		AddExchangeInfo(bot.exchange.lakebtc.GetName(), "BTC", LakeBTC.USD.Last, 0)
	}
}
