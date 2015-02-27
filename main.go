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
	btcmarkets BTCMarkets
	okcoinChina OKCoin
	okcoinIntl OKCoin
	itbit ItBit
	lakebtc LakeBTC
	huobi HUOBI
}

func main() {
	log.Println("Bot started")
	log.Println("Loading config file config.json..")
	config, err := ReadConfig("config.json")

	if err != nil {
		log.Println("Fatal error opening config.json file. Error: ", err)
		return
	}

	log.Println("Config file loaded.")

	enabledExchanges := 0
	for _, exch := range config.Exchanges {
		if exch.Enabled {
			enabledExchanges++
		}
	}

	if enabledExchanges == 0 {
		log.Println("Bot started with no exchanges supported. Exiting.")
		return
	}

	log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n", len(config.Exchanges), enabledExchanges)
	log.Println("Bot exchange support:")

	exchange := Exchange{}
	exchange.btcchina.SetDefaults()
	exchange.bitstamp.SetDefaults()
	exchange.bitfinex.SetDefaults()
	exchange.btce.SetDefaults()
	exchange.btcmarkets.SetDefaults()
	exchange.okcoinChina.SetURL(OKCOIN_API_URL_CHINA)
	exchange.okcoinChina.SetDefaults()
	exchange.okcoinIntl.SetURL(OKCOIN_API_URL)
	exchange.okcoinIntl.SetDefaults()
	exchange.itbit.SetDefaults()
	exchange.lakebtc.SetDefaults()
	exchange.huobi.SetDefaults()

	for _, exch := range config.Exchanges {
		if exchange.btcchina.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.btcchina.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.btcchina.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.btcchina.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.bitstamp.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.bitstamp.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.bitstamp.SetAPIKeys(exch.ClientID, exch.APIKey, exch.APISecret)
				exchange.bitstamp.GetBalance()

				if exch.Verbose {
					exchange.bitstamp.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.bitfinex.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.bitfinex.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.bitfinex.SetAPIKeys(exch.APIKey, exch.APISecret)
				
				if exch.Verbose {
					exchange.bitfinex.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
				exchange.bitfinex.GetAccountFeeInfo()
				exchange.bitfinex.GetActiveOrders()
			}
		} else if exchange.btce.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.btce.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.btce.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.btce.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.btcmarkets.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.btcmarkets.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.btcmarkets.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.btcmarkets.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.okcoinChina.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.okcoinChina.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.okcoinChina.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.okcoinChina.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.okcoinIntl.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.okcoinIntl.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.okcoinIntl.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.okcoinIntl.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.itbit.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.itbit.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.itbit.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.itbit.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.lakebtc.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.lakebtc.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.lakebtc.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.lakebtc.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if exchange.huobi.GetName() == exch.Name {
			if !exch.Enabled {
				exchange.huobi.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				exchange.huobi.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					exchange.huobi.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		}
	}
	
	err = RetrieveConfigCurrencyPairs(config)

	if err != nil {
		log.Println("Fatal error retrieving config currency pairs. Error: ", err)
	}

	//temp until proper asynchronous method of getting pricing/order books is coded
	for {

		//spot 
		if exchange.lakebtc.IsEnabled() {
			go func() {
				LakeBTCTickerResponse := exchange.lakebtc.GetTicker()
				log.Printf("LakeBTC USD: Last %f (%f) High %f (%f) Low %f (%f)\n", LakeBTCTickerResponse.USD.Last, LakeBTCTickerResponse.CNY.Last, LakeBTCTickerResponse.USD.High, LakeBTCTickerResponse.CNY.High, LakeBTCTickerResponse.USD.Low, LakeBTCTickerResponse.CNY.Low)
			}()
		}

		if exchange.btcchina.IsEnabled() {
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
		}

		if exchange.huobi.IsEnabled() {
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
		}

		if exchange.itbit.IsEnabled() {
			go func() {
				ItbitBTC := exchange.itbit.GetTicker("XBTUSD")
				log.Printf("ItBit BTC: Last %f High %f Low %f Volume %f\n", ItbitBTC.LastPrice, ItbitBTC.High24h, ItbitBTC.Low24h, ItbitBTC.Volume24h)
			}()
		}

		if exchange.bitstamp.IsEnabled() {
			go func() {
				BitstampBTC := exchange.bitstamp.GetTicker()
				log.Printf("Bitstamp BTC: Last %f High %f Low %f Volume %f\n", BitstampBTC.Last, BitstampBTC.High, BitstampBTC.Low, BitstampBTC.Volume)
			}()
		}

		if exchange.bitfinex.IsEnabled() {
			go func() {
				BitfinexLTC := exchange.bitfinex.GetTicker("ltcusd")
				log.Printf("Bitfinex LTC: Last %f High %f Low %f Volume %f\n", BitfinexLTC.Last, BitfinexLTC.High, BitfinexLTC.Low, BitfinexLTC.Volume)
			}()

			go func() {
				BitfinexBTC := exchange.bitfinex.GetTicker("btcusd")
				log.Printf("Bitfinex BTC: Last %f High %f Low %f Volume %f\n", BitfinexBTC.Last, BitfinexBTC.High, BitfinexBTC.Low, BitfinexBTC.Volume)
			}()
		}

		if exchange.btce.IsEnabled() {
			go func() {
				BTCeBTC := exchange.btce.GetTicker("btc_usd")
				log.Printf("BTC-e BTC: Last %f High %f Low %f Volume %f\n", BTCeBTC.Last, BTCeBTC.High, BTCeBTC.Low, BTCeBTC.Vol_cur)
			}()

			go func() {
				BTCeLTC := exchange.btce.GetTicker("ltc_usd")
				log.Printf("BTC-e LTC: Last %f High %f Low %f Volume %f\n", BTCeLTC.Last, BTCeLTC.High, BTCeLTC.Low, BTCeLTC.Vol_cur)
			}()
		}

		if exchange.btcmarkets.IsEnabled() {
			go func() {
				BTCMarketsBTC := exchange.btcmarkets.GetTicker("BTC")
				BTCMarketsBTCLastUSD, _ := ConvertCurrency(BTCMarketsBTC.LastPrice, "AUD", "USD")
				BTCMarketsBTCBestBidUSD, _ := ConvertCurrency(BTCMarketsBTC.BestBID, "AUD", "USD")
				BTCMarketsBTCBestAskUSD, _ := ConvertCurrency(BTCMarketsBTC.BestAsk, "AUD", "USD")
				log.Printf("BTC Markets BTC: Last %f (%f) Bid %f (%f) Ask %f (%f)\n", BTCMarketsBTCLastUSD, BTCMarketsBTC.LastPrice, BTCMarketsBTCBestBidUSD, BTCMarketsBTC.BestBID, BTCMarketsBTCBestAskUSD, BTCMarketsBTC.BestAsk)
			}()

			go func() {
				BTCMarketsLTC := exchange.btcmarkets.GetTicker("LTC")
				BTCMarketsLTCLastUSD, _ := ConvertCurrency(BTCMarketsLTC.LastPrice, "AUD", "USD")
				BTCMarketsLTCBestBidUSD, _ := ConvertCurrency(BTCMarketsLTC.BestBID, "AUD", "USD")
				BTCMarketsLTCBestAskUSD, _ := ConvertCurrency(BTCMarketsLTC.BestAsk, "AUD", "USD")
				log.Printf("BTC Markets LTC: Last %f (%f) Bid %f (%f) Ask %f (%f)", BTCMarketsLTCLastUSD, BTCMarketsLTC.LastPrice, BTCMarketsLTCBestBidUSD, BTCMarketsLTC.BestBID, BTCMarketsLTCBestAskUSD, BTCMarketsLTC.BestAsk)
			}()
		}

		if exchange.okcoinChina.IsEnabled() {
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
		}

		if exchange.okcoinIntl.IsEnabled() {
			go func() {
				OKCoinChinaIntlBTC := exchange.okcoinIntl.GetTicker("btc_usd")
				log.Printf("OKCoin Intl BTC: Last %f High %f Low %f Volume %f\n", OKCoinChinaIntlBTC.Last, OKCoinChinaIntlBTC.High, OKCoinChinaIntlBTC.Low, OKCoinChinaIntlBTC.Vol)
			}()

			go func() {
				OKCoinChinaIntlLTC := exchange.okcoinIntl.GetTicker("ltc_usd")
				log.Printf("OKCoin Intl LTC: Last %f High %f Low %f Volume %f\n", OKCoinChinaIntlLTC.Last, OKCoinChinaIntlLTC.High, OKCoinChinaIntlLTC.Low, OKCoinChinaIntlLTC.Vol)
			}()
		
		// futures
			go func() {
				OKCoinFuturesBTC := exchange.okcoinIntl.GetFuturesTicker("btc_usd", "this_week")
				log.Printf("OKCoin BTC Futures (weekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := exchange.okcoinIntl.GetFuturesTicker("ltc_usd", "this_week")
				log.Printf("OKCoin LTC Futures (weekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := exchange.okcoinIntl.GetFuturesTicker("btc_usd", "next_week")
				log.Printf("OKCoin BTC Futures (biweekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := exchange.okcoinIntl.GetFuturesTicker("ltc_usd", "next_week")
				log.Printf("OKCoin LTC Futures (biweekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := exchange.okcoinIntl.GetFuturesTicker("btc_usd", "quarter")
				log.Printf("OKCoin BTC Futures (quarterly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := exchange.okcoinIntl.GetFuturesTicker("ltc_usd", "quarter")
				log.Printf("OKCoin LTC Futures (quarterly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()
		}

		time.Sleep(time.Second * 15)
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
    	cmd.Run()
	}
}
