package main

import (
	"log"
	"time"
)

type Exchange struct {
	btcchina BTCChina
	bitstamp Bitstamp
	bitfinex Bitfinex
	btce BTCE
	lakeBTC LakeBTC
	okcoin OKCoin
	itbit ItBit
}

func main() {
	log.Println("Bot started")
	exchange := Exchange{}

	for {
		log.Println("BTC Pricing:")
		log.Printf("BTC China: %v\n", exchange.btcchina.GetTicker("btccny"))
		log.Printf("Bitfinex: %v\n", exchange.bitfinex.GetTicker("btcusd"))
		log.Printf("BTC-e BTC: %v\n", exchange.btce.GetTicker("btc_usd"))
		log.Printf("OKCoin: %v\n", exchange.okcoin.GetTicker("btc_usd"))

		log.Println("LTC Pricing:")
		log.Printf("BTC China: %v\n", exchange.btcchina.GetTicker("ltccny"))
		log.Printf("Bitfinex: %v\n", exchange.bitfinex.GetTicker("ltcusd"))
		log.Printf("BTC-e: %v\n", exchange.btce.GetTicker("ltc_usd"))
		log.Printf("OKCoin: %v\n", exchange.okcoin.GetTicker("ltc_usd"))
		time.Sleep(time.Second * 15)
	}
}
