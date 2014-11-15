package main

import (
	"log"
	"time"
	"os"
	"os/exec"
)

type Exchange struct {
	btcchina BTCChina
	bitstamp Bitstamp
	bitfinex Bitfinex
	btce BTCE
	okcoinChina OKCoin
	okcoinIntl OKCoin
}

func main() {
	log.Println("Bot started")
	exchange := Exchange{}
	exchange.okcoinChina.SetURL(OKCOIN_API_URL_CHINA)
	exchange.okcoinIntl.SetURL(OKCOIN_API_URL)

	//temp until proper asynchronous method of getting pricing/order books is coded
	for {
		go func() {
			BTCChinaBTC := exchange.btcchina.GetTicker("btccny")
			log.Printf("BTCChina BTC: Last %s High %s Low %s Volume %s\n", BTCChinaBTC.Last, BTCChinaBTC.High, BTCChinaBTC.Low, BTCChinaBTC.Vol)
		}()

		go func() {
			BitstampBTC := exchange.bitstamp.GetTicker()
			log.Printf("Bitstamp BTC: Last %s High %s Low %s Volume %s\n", BitstampBTC.Last, BitstampBTC.High, BitstampBTC.Low, BitstampBTC.Volume)
		}()

		go func() {
			BitfinexBTC := exchange.bitfinex.GetTicker("btcusd")
			log.Printf("Bitfinex BTC: Last %s High %s Low %s Volume %s\n", BitfinexBTC.Last_price, BitfinexBTC.High, BitfinexBTC.Low, BitfinexBTC.Volume)
		}()
		
		go func() {
			BTCeBTC := exchange.btce.GetTicker("btc_usd")
			log.Printf("BTC-e BTC: Last %f High %f Low %f Volume %f\n", BTCeBTC.Last, BTCeBTC.High, BTCeBTC.Low, BTCeBTC.Vol_cur)
		}()

		go func() {
			OKCoinChinaBTC := exchange.okcoinChina.GetTicker("btc_cny")
			log.Printf("OKCoin China BTC: Last %s High %s Low %s Volume %s\n", OKCoinChinaBTC.Last, OKCoinChinaBTC.High, OKCoinChinaBTC.Low, OKCoinChinaBTC.Vol)
		}()

		go func() {
			OKCoinChinaIntlBTC := exchange.okcoinIntl.GetTicker("btc_usd")
			log.Printf("OKCoin Intl BTC: Last %s High %s Low %s Volume %s\n", OKCoinChinaIntlBTC.Last, OKCoinChinaIntlBTC.High, OKCoinChinaIntlBTC.Low, OKCoinChinaIntlBTC.Vol)
		}()
		
		go func() {
			BTCChinaBTC := exchange.btcchina.GetTicker("ltccny")
			log.Printf("BTCChina LTC: Last %s High %s Low %s Volume %s\n", BTCChinaBTC.Last, BTCChinaBTC.High, BTCChinaBTC.Low, BTCChinaBTC.Vol)
		}()

		go func() {
			BitfinexBTC := exchange.bitfinex.GetTicker("ltcusd")
			log.Printf("Bitfinex LTC: Last %s High %s Low %s Volume %s\n", BitfinexBTC.Last_price, BitfinexBTC.High, BitfinexBTC.Low, BitfinexBTC.Volume)
		}()
		
		go func() {
			BTCeBTC := exchange.btce.GetTicker("ltc_usd")
			log.Printf("BTC-e LTC: Last %f High %f Low %f Volume %f\n", BTCeBTC.Last, BTCeBTC.High, BTCeBTC.Low, BTCeBTC.Vol_cur)
		}()

		go func() {
			OKCoinChinaBTC := exchange.okcoinChina.GetTicker("ltc_cny")
			log.Printf("OKCoin China LTC: Last %s High %s Low %s Volume %s\n", OKCoinChinaBTC.Last, OKCoinChinaBTC.High, OKCoinChinaBTC.Low, OKCoinChinaBTC.Vol)
		}()

		go func() {
			OKCoinChinaIntlLTC := exchange.okcoinIntl.GetTicker("ltc_usd")
			log.Printf("OKCoin Intl LTC: Last %s High %s Low %s Volume %s\n", OKCoinChinaIntlLTC.Last, OKCoinChinaIntlLTC.High, OKCoinChinaIntlLTC.Low, OKCoinChinaIntlLTC.Vol)
		}()

		time.Sleep(time.Second * 15)
		cmd := exec.Command("cmd", "/c", "cls")
    	cmd.Stdout = os.Stdout
    	cmd.Run()
	}
}
