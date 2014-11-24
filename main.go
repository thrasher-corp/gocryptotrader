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
	itbit ItBit
	huobi HUOBI
}

func main() {
	log.Println("Bot started")
	exchange := Exchange{}
	exchange.okcoinChina.SetURL(OKCOIN_API_URL_CHINA)
	exchange.okcoinIntl.SetURL(OKCOIN_API_URL)
	err := QueryYahooCurrencyValues() 

	if err != nil {
		log.Fatalln(err)
		return
	}

	//temp until proper asynchronous method of getting pricing/order books is coded
	for {
		go func() {
			BTCChinaBTC := exchange.btcchina.GetTicker("btccny")
			BTCChinaBTCLastUSD, _ := ConvertCurrency(BTCChinaBTC.Last, "CNY", "USD")
			BTCChinaBTCHighUSD, _ := ConvertCurrency(BTCChinaBTC.High, "CNY", "USD")
			BTCChinaBTCLowUSD, _ := ConvertCurrency(BTCChinaBTC.Low, "CNY", "USD")
			log.Printf("BTCChina BTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", BTCChinaBTCLastUSD, BTCChinaBTC.Last,  BTCChinaBTCHighUSD, BTCChinaBTC.High, BTCChinaBTCLowUSD, BTCChinaBTC.Low, BTCChinaBTC.Vol)
		}()

		go func() {
			BTCChinaLTC := exchange.btcchina.GetTicker("ltccny")
			BTCChinaLTCLastUSD, _ := ConvertCurrency(BTCChinaLTC.Last, "CNY", "USD")
			BTCChinaLTCHighUSD, _ := ConvertCurrency(BTCChinaLTC.High, "CNY", "USD")
			BTCChinaLTCLowUSD, _ := ConvertCurrency(BTCChinaLTC.Low, "CNY", "USD")
			log.Printf("BTCChina LTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", BTCChinaLTCLastUSD, BTCChinaLTC.Last, BTCChinaLTCHighUSD, BTCChinaLTC.High, BTCChinaLTCLowUSD, BTCChinaLTC.Low, BTCChinaLTC.Vol)
		}()

		go func() {
			HuobiBTC := exchange.huobi.GetTicker("btc")
			HuobiBTCLastUSD, _ := ConvertCurrency(HuobiBTC.Last, "CNY", "USD")
			HuobiBTCHighUSD, _ := ConvertCurrency(HuobiBTC.High, "CNY", "USD")
			HuobiBTCLowUSD, _ := ConvertCurrency(HuobiBTC.Low, "CNY", "USD")
			log.Printf("Huobi BTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", HuobiBTCLastUSD, HuobiBTC.Last, HuobiBTCHighUSD, HuobiBTC.High, HuobiBTCLowUSD, HuobiBTC.Low, HuobiBTC.Vol)
		}()

		go func() {
			HuobiLTC := exchange.huobi.GetTicker("btc")
			HuobiLTCLastUSD, _ := ConvertCurrency(HuobiLTC.Last, "CNY", "USD")
			HuobiLTCHighUSD, _ := ConvertCurrency(HuobiLTC.High, "CNY", "USD")
			HuobiLTCLowUSD, _ := ConvertCurrency(HuobiLTC.Low, "CNY", "USD")
			log.Printf("Huobi BTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", HuobiLTCLastUSD, HuobiLTC.Last, HuobiLTCHighUSD, HuobiLTC.High, HuobiLTCLowUSD, HuobiLTC.Low, HuobiLTC.Vol)
		}()

		go func() {
			ItbitBTC := exchange.itbit.GetTicker("XBTUSD")
			log.Printf("ItBit BTC: Last %f High %f Low %f Volume %f\n", ItbitBTC.LastPrice, ItbitBTC.High24h, ItbitBTC.Low24h, ItbitBTC.Volume24h)
		}()

		go func() {
			BitstampBTC := exchange.bitstamp.GetTicker()
			log.Printf("Bitstamp BTC: Last %f High %f Low %f Volume %f\n", BitstampBTC.Last, BitstampBTC.High, BitstampBTC.Low, BitstampBTC.Volume)
		}()

		go func() {
			BitfinexLTC := exchange.bitfinex.GetTicker("ltcusd")
			log.Printf("Bitfinex LTC: Last %f High %f Low %f Volume %f\n", BitfinexLTC.Last, BitfinexLTC.High, BitfinexLTC.Low, BitfinexLTC.Volume)
		}()

		go func() {
			BitfinexBTC := exchange.bitfinex.GetTicker("btcusd")
			log.Printf("Bitfinex BTC: Last %f High %f Low %f Volume %f\n", BitfinexBTC.Last, BitfinexBTC.High, BitfinexBTC.Low, BitfinexBTC.Volume)
		}()
		
		go func() {
			BTCeBTC := exchange.btce.GetTicker("btc_usd")
			log.Printf("BTC-e BTC: Last %f High %f Low %f Volume %f\n", BTCeBTC.Last, BTCeBTC.High, BTCeBTC.Low, BTCeBTC.Vol_cur)
		}()

		go func() {
			BTCeLTC := exchange.btce.GetTicker("ltc_usd")
			log.Printf("BTC-e LTC: Last %f High %f Low %f Volume %f\n", BTCeLTC.Last, BTCeLTC.High, BTCeLTC.Low, BTCeLTC.Vol_cur)
		}()

		go func() {
			OKCoinChinaBTC := exchange.okcoinChina.GetTicker("btc_cny")
			OKCoinChinaBTCLastUSD, _ := ConvertCurrency(OKCoinChinaBTC.Last, "CNY", "USD")
			OKCoinChinaBTCHighUSD, _ := ConvertCurrency(OKCoinChinaBTC.High, "CNY", "USD")
			OKCoinChinaBTCLowUSD, _ := ConvertCurrency(OKCoinChinaBTC.Low, "CNY", "USD")
			log.Printf("OKCoin China: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", OKCoinChinaBTCLastUSD, OKCoinChinaBTC.Last, OKCoinChinaBTCHighUSD, OKCoinChinaBTC.High, OKCoinChinaBTCLowUSD, OKCoinChinaBTC.Low, OKCoinChinaBTC.Vol)
		}()

		go func() {
			OKCoinChinaLTC := exchange.okcoinChina.GetTicker("ltc_cny")
			OKCoinChinaLTCLastUSD, _ := ConvertCurrency(OKCoinChinaLTC.Last, "CNY", "USD")
			OKCoinChinaLTCHighUSD, _ := ConvertCurrency(OKCoinChinaLTC.High, "CNY", "USD")
			OKCoinChinaLTCLowUSD, _ := ConvertCurrency(OKCoinChinaLTC.Low, "CNY", "USD")
			log.Printf("OKCoin China: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", OKCoinChinaLTCLastUSD, OKCoinChinaLTC.Last, OKCoinChinaLTCHighUSD, OKCoinChinaLTC.High, OKCoinChinaLTCLowUSD, OKCoinChinaLTC.Low, OKCoinChinaLTC.Vol)
		}()

		go func() {
			OKCoinChinaIntlBTC := exchange.okcoinIntl.GetTicker("btc_usd")
			log.Printf("OKCoin Intl BTC: Last %f High %f Low %f Volume %f\n", OKCoinChinaIntlBTC.Last, OKCoinChinaIntlBTC.High, OKCoinChinaIntlBTC.Low, OKCoinChinaIntlBTC.Vol)
		}()

		go func() {
			OKCoinChinaIntlLTC := exchange.okcoinIntl.GetTicker("ltc_usd")
			log.Printf("OKCoin Intl LTC: Last %f High %f Low %f Volume %f\n", OKCoinChinaIntlLTC.Last, OKCoinChinaIntlLTC.High, OKCoinChinaIntlLTC.Low, OKCoinChinaIntlLTC.Vol)
		}()

		time.Sleep(time.Second * 15)
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
    	cmd.Run()
	}
}
