package main

import (
	"log"
	"time"
	"os"
	"errors"
	"os/exec"
	"os/signal"
	"syscall"
)

type Exchange struct {
	btcchina BTCChina
	bitstamp Bitstamp
	bitfinex Bitfinex
	btce BTCE
	btcmarkets BTCMarkets
	coinbase Coinbase
	okcoinChina OKCoin
	okcoinIntl OKCoin
	itbit ItBit
	lakebtc LakeBTC
	huobi HUOBI
	kraken Kraken
}

type Bot struct {
	config Config
	exchange Exchange
}

var bot Bot

func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Captured %v.", sig)
		Shutdown()
		log.Println("Exiting.")
		os.Exit(1)
	}()
}

func Shutdown() {
	err := SaveConfig()

	if err != nil {
		log.Println("Unable to save config.")
	}

	log.Println("Config file saved successfully.")
}

func main() {
	HandleInterrupt()
	log.Println("Loading config file config.json..")

	err := errors.New("")
	bot.config, err = ReadConfig()

	if err != nil {
		log.Println("Fatal error opening config.json file. Error: ", err)
		return
	}

	log.Println("Config file loaded.")
	log.Printf("Bot '%s' started.\n", bot.config.Name)

	enabledExchanges := 0
	for _, exch := range bot.config.Exchanges {
		if exch.Enabled {
			enabledExchanges++
		}
	}

	if enabledExchanges == 0 {
		log.Println("Bot started with no exchanges supported. Exiting.")
		return
	}

	smsSupport := false
	smsContacts := 0

	for _, sms := range bot.config.SMSContacts {
		if sms.Enabled {
			smsSupport = true
			smsContacts++
		}
	}

	if smsSupport {
		log.Printf("SMS support enabled. Number of SMS contacts %d.\n", smsContacts)
	} else {
		log.Println("SMS support disabled.")
	}

	log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n", len(bot.config.Exchanges), enabledExchanges)
	log.Println("Bot Exchange support:")

	bot.exchange.kraken.SetDefaults()
	bot.exchange.btcchina.SetDefaults()
	bot.exchange.bitstamp.SetDefaults()
	bot.exchange.bitfinex.SetDefaults()
	bot.exchange.btce.SetDefaults()
	bot.exchange.btcmarkets.SetDefaults()
	bot.exchange.coinbase.SetDefaults()
	bot.exchange.okcoinChina.SetURL(OKCOIN_API_URL_CHINA)
	bot.exchange.okcoinChina.SetDefaults()
	bot.exchange.okcoinIntl.SetURL(OKCOIN_API_URL)
	bot.exchange.okcoinIntl.SetDefaults()
	bot.exchange.itbit.SetDefaults()
	bot.exchange.lakebtc.SetDefaults()
	bot.exchange.huobi.SetDefaults()

	for _, exch := range bot.config.Exchanges {
		if bot.exchange.btcchina.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.btcchina.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.btcchina.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.btcchina.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.bitstamp.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.bitstamp.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.bitstamp.SetAPIKeys(exch.ClientID, exch.APIKey, exch.APISecret)
				bot.exchange.bitstamp.GetBalance()

				if exch.Verbose {
					bot.exchange.bitstamp.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.bitfinex.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.bitfinex.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.bitfinex.SetAPIKeys(exch.APIKey, exch.APISecret)
				
				if exch.Verbose {
					bot.exchange.bitfinex.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
				bot.exchange.bitfinex.GetAccountFeeInfo()
				bot.exchange.bitfinex.GetAccountBalance()
			}
		} else if bot.exchange.btce.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.btce.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.btce.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.btce.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.btcmarkets.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.btcmarkets.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.btcmarkets.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.btcmarkets.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.coinbase.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.coinbase.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.coinbase.SetAPIKeys(exch.ClientID, exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.coinbase.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.okcoinChina.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.okcoinChina.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.okcoinChina.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.okcoinChina.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.okcoinIntl.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.okcoinIntl.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.okcoinIntl.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.okcoinIntl.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.itbit.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.itbit.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.itbit.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.itbit.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.kraken.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.kraken.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.kraken.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.kraken.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.lakebtc.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.lakebtc.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.lakebtc.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.lakebtc.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.huobi.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.huobi.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.huobi.SetAPIKeys(exch.APIKey, exch.APISecret)

				if exch.Verbose {
					bot.exchange.huobi.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		}
	}

	err = RetrieveConfigCurrencyPairs(bot.config)

	if err != nil {
		log.Println("Fatal error retrieving config currency pairs. Error: ", err)
	}

	//temp until proper asynchronous method of getting pricing/order books is coded
	for {
		//spot 
		if bot.exchange.coinbase.IsEnabled() {
			go func() {
				CoinbaseStats := bot.exchange.coinbase.GetStats("BTC-USD")
				CoinbaseTicker := bot.exchange.coinbase.GetTicker("BTC-USD")
				log.Printf("Coinbase BTC: Last %f High %f Low %f Volume %f\n", CoinbaseTicker.Price, CoinbaseStats.High, CoinbaseStats.Low, CoinbaseStats.Volume)
			}()
		}
		if bot.exchange.kraken.IsEnabled() {
			go func() {
				KrakenBTC := bot.exchange.kraken.GetTicker("XBTUSD")
				log.Printf("Kraken BTC: %v\n", KrakenBTC)
			}()
			go func() {
				KrakenLTC := bot.exchange.kraken.GetTicker("LTCUSD")
				log.Printf("Kraken LTC: %v\n", KrakenLTC)
			}()
		}
		if bot.exchange.lakebtc.IsEnabled() {
			go func() {
				LakeBTCTickerResponse := bot.exchange.lakebtc.GetTicker()
				log.Printf("LakeBTC USD: Last %f (%f) High %f (%f) Low %f (%f)\n", LakeBTCTickerResponse.USD.Last, LakeBTCTickerResponse.CNY.Last, LakeBTCTickerResponse.USD.High, LakeBTCTickerResponse.CNY.High, LakeBTCTickerResponse.USD.Low, LakeBTCTickerResponse.CNY.Low)
			}()
		}
		if bot.exchange.btcchina.IsEnabled() {
			go func() {
				BTCChinaBTC := bot.exchange.btcchina.GetTicker("btccny")
				BTCChinaBTCLastUSD, _ := ConvertCurrency(BTCChinaBTC.Last, "CNY", "USD")
				BTCChinaBTCHighUSD, _ := ConvertCurrency(BTCChinaBTC.High, "CNY", "USD")
				BTCChinaBTCLowUSD, _ := ConvertCurrency(BTCChinaBTC.Low, "CNY", "USD")
				log.Printf("BTCChina BTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", BTCChinaBTCLastUSD, BTCChinaBTC.Last,  BTCChinaBTCHighUSD, BTCChinaBTC.High, BTCChinaBTCLowUSD, BTCChinaBTC.Low, BTCChinaBTC.Vol)
			}()


			go func() {
				BTCChinaLTC := bot.exchange.btcchina.GetTicker("ltccny")
				BTCChinaLTCLastUSD, _ := ConvertCurrency(BTCChinaLTC.Last, "CNY", "USD")
				BTCChinaLTCHighUSD, _ := ConvertCurrency(BTCChinaLTC.High, "CNY", "USD")
				BTCChinaLTCLowUSD, _ := ConvertCurrency(BTCChinaLTC.Low, "CNY", "USD")
				log.Printf("BTCChina LTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", BTCChinaLTCLastUSD, BTCChinaLTC.Last, BTCChinaLTCHighUSD, BTCChinaLTC.High, BTCChinaLTCLowUSD, BTCChinaLTC.Low, BTCChinaLTC.Vol)
			}()
		}

		if bot.exchange.huobi.IsEnabled() {
			go func() {
				HuobiBTC := bot.exchange.huobi.GetTicker("btc")
				HuobiBTCLastUSD, _ := ConvertCurrency(HuobiBTC.Last, "CNY", "USD")
				HuobiBTCHighUSD, _ := ConvertCurrency(HuobiBTC.High, "CNY", "USD")
				HuobiBTCLowUSD, _ := ConvertCurrency(HuobiBTC.Low, "CNY", "USD")
				log.Printf("Huobi BTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", HuobiBTCLastUSD, HuobiBTC.Last, HuobiBTCHighUSD, HuobiBTC.High, HuobiBTCLowUSD, HuobiBTC.Low, HuobiBTC.Vol)
			}()

			go func() {
				HuobiLTC := bot.exchange.huobi.GetTicker("ltc")
				HuobiLTCLastUSD, _ := ConvertCurrency(HuobiLTC.Last, "CNY", "USD")
				HuobiLTCHighUSD, _ := ConvertCurrency(HuobiLTC.High, "CNY", "USD")
				HuobiLTCLowUSD, _ := ConvertCurrency(HuobiLTC.Low, "CNY", "USD")
				log.Printf("Huobi LTC: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", HuobiLTCLastUSD, HuobiLTC.Last, HuobiLTCHighUSD, HuobiLTC.High, HuobiLTCLowUSD, HuobiLTC.Low, HuobiLTC.Vol)
			}()
		}

		if bot.exchange.itbit.IsEnabled() {
			go func() {
				ItbitBTC := bot.exchange.itbit.GetTicker("XBTUSD")
				log.Printf("ItBit BTC: Last %f High %f Low %f Volume %f\n", ItbitBTC.LastPrice, ItbitBTC.High24h, ItbitBTC.Low24h, ItbitBTC.Volume24h)
			}()
		}

		if bot.exchange.bitstamp.IsEnabled() {
			go func() {
				BitstampBTC := bot.exchange.bitstamp.GetTicker()
				log.Printf("Bitstamp BTC: Last %f High %f Low %f Volume %f\n", BitstampBTC.Last, BitstampBTC.High, BitstampBTC.Low, BitstampBTC.Volume)
			}()
		}

		if bot.exchange.bitfinex.IsEnabled() {
			go func() {
				BitfinexLTC := bot.exchange.bitfinex.GetTicker("ltcusd")
				log.Printf("Bitfinex LTC: Last %f High %f Low %f Volume %f\n", BitfinexLTC.Last, BitfinexLTC.High, BitfinexLTC.Low, BitfinexLTC.Volume)
			}()

			go func() {
				BitfinexBTC := bot.exchange.bitfinex.GetTicker("btcusd")
				log.Printf("Bitfinex BTC: Last %f High %f Low %f Volume %f\n", BitfinexBTC.Last, BitfinexBTC.High, BitfinexBTC.Low, BitfinexBTC.Volume)
			}()
		}

		if bot.exchange.btce.IsEnabled() {
			go func() {
				BTCeBTC := bot.exchange.btce.GetTicker("btc_usd")
				log.Printf("BTC-e BTC: Last %f High %f Low %f Volume %f\n", BTCeBTC.Last, BTCeBTC.High, BTCeBTC.Low, BTCeBTC.Vol_cur)
			}()

			go func() {
				BTCeLTC := bot.exchange.btce.GetTicker("ltc_usd")
				log.Printf("BTC-e LTC: Last %f High %f Low %f Volume %f\n", BTCeLTC.Last, BTCeLTC.High, BTCeLTC.Low, BTCeLTC.Vol_cur)
			}()
		}

		if bot.exchange.btcmarkets.IsEnabled() {
			go func() {
				BTCMarketsBTC := bot.exchange.btcmarkets.GetTicker("BTC")
				BTCMarketsBTCLastUSD, _ := ConvertCurrency(BTCMarketsBTC.LastPrice, "AUD", "USD")
				BTCMarketsBTCBestBidUSD, _ := ConvertCurrency(BTCMarketsBTC.BestBID, "AUD", "USD")
				BTCMarketsBTCBestAskUSD, _ := ConvertCurrency(BTCMarketsBTC.BestAsk, "AUD", "USD")
				log.Printf("BTC Markets BTC: Last %f (%f) Bid %f (%f) Ask %f (%f)\n", BTCMarketsBTCLastUSD, BTCMarketsBTC.LastPrice, BTCMarketsBTCBestBidUSD, BTCMarketsBTC.BestBID, BTCMarketsBTCBestAskUSD, BTCMarketsBTC.BestAsk)
			}()

			go func() {
				BTCMarketsLTC := bot.exchange.btcmarkets.GetTicker("LTC")
				BTCMarketsLTCLastUSD, _ := ConvertCurrency(BTCMarketsLTC.LastPrice, "AUD", "USD")
				BTCMarketsLTCBestBidUSD, _ := ConvertCurrency(BTCMarketsLTC.BestBID, "AUD", "USD")
				BTCMarketsLTCBestAskUSD, _ := ConvertCurrency(BTCMarketsLTC.BestAsk, "AUD", "USD")
				log.Printf("BTC Markets LTC: Last %f (%f) Bid %f (%f) Ask %f (%f)", BTCMarketsLTCLastUSD, BTCMarketsLTC.LastPrice, BTCMarketsLTCBestBidUSD, BTCMarketsLTC.BestBID, BTCMarketsLTCBestAskUSD, BTCMarketsLTC.BestAsk)
			}()
		}

		if bot.exchange.okcoinChina.IsEnabled() {
			go func() {
				OKCoinChinaBTC := bot.exchange.okcoinChina.GetTicker("btc_cny")
				OKCoinChinaBTCLastUSD, _ := ConvertCurrency(OKCoinChinaBTC.Last, "CNY", "USD")
				OKCoinChinaBTCHighUSD, _ := ConvertCurrency(OKCoinChinaBTC.High, "CNY", "USD")
				OKCoinChinaBTCLowUSD, _ := ConvertCurrency(OKCoinChinaBTC.Low, "CNY", "USD")
				log.Printf("OKCoin China: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", OKCoinChinaBTCLastUSD, OKCoinChinaBTC.Last, OKCoinChinaBTCHighUSD, OKCoinChinaBTC.High, OKCoinChinaBTCLowUSD, OKCoinChinaBTC.Low, OKCoinChinaBTC.Vol)
			}()

			go func() {
				OKCoinChinaLTC := bot.exchange.okcoinChina.GetTicker("ltc_cny")
				OKCoinChinaLTCLastUSD, _ := ConvertCurrency(OKCoinChinaLTC.Last, "CNY", "USD")
				OKCoinChinaLTCHighUSD, _ := ConvertCurrency(OKCoinChinaLTC.High, "CNY", "USD")
				OKCoinChinaLTCLowUSD, _ := ConvertCurrency(OKCoinChinaLTC.Low, "CNY", "USD")
				log.Printf("OKCoin China: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", OKCoinChinaLTCLastUSD, OKCoinChinaLTC.Last, OKCoinChinaLTCHighUSD, OKCoinChinaLTC.High, OKCoinChinaLTCLowUSD, OKCoinChinaLTC.Low, OKCoinChinaLTC.Vol)
			}()
		}

		if bot.exchange.okcoinIntl.IsEnabled() {
			go func() {
				OKCoinChinaIntlBTC := bot.exchange.okcoinIntl.GetTicker("btc_usd")
				log.Printf("OKCoin Intl BTC: Last %f High %f Low %f Volume %f\n", OKCoinChinaIntlBTC.Last, OKCoinChinaIntlBTC.High, OKCoinChinaIntlBTC.Low, OKCoinChinaIntlBTC.Vol)
			}()

			go func() {
				OKCoinChinaIntlLTC := bot.exchange.okcoinIntl.GetTicker("ltc_usd")
				log.Printf("OKCoin Intl LTC: Last %f High %f Low %f Volume %f\n", OKCoinChinaIntlLTC.Last, OKCoinChinaIntlLTC.High, OKCoinChinaIntlLTC.Low, OKCoinChinaIntlLTC.Vol)
			}()
		
		// futures
			go func() {
				OKCoinFuturesBTC := bot.exchange.okcoinIntl.GetFuturesTicker("btc_usd", "this_week")
				log.Printf("OKCoin BTC Futures (weekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := bot.exchange.okcoinIntl.GetFuturesTicker("ltc_usd", "this_week")
				log.Printf("OKCoin LTC Futures (weekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := bot.exchange.okcoinIntl.GetFuturesTicker("btc_usd", "next_week")
				log.Printf("OKCoin BTC Futures (biweekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := bot.exchange.okcoinIntl.GetFuturesTicker("ltc_usd", "next_week")
				log.Printf("OKCoin LTC Futures (biweekly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := bot.exchange.okcoinIntl.GetFuturesTicker("btc_usd", "quarter")
				log.Printf("OKCoin BTC Futures (quarterly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()

			go func() {
				OKCoinFuturesBTC := bot.exchange.okcoinIntl.GetFuturesTicker("ltc_usd", "quarter")
				log.Printf("OKCoin LTC Futures (quarterly): Last %f High %f Low %f Volume %f\n", OKCoinFuturesBTC.Last, OKCoinFuturesBTC.High, OKCoinFuturesBTC.Low, OKCoinFuturesBTC.Vol)
			}()
		}

		time.Sleep(time.Second * 15)
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}