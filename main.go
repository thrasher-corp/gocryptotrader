package main

import (
	"log"
	"os"
	"errors"
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
	cryptsy Cryptsy
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
	shutdown chan bool
}

var bot Bot

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
	bot.exchange.cryptsy.SetDefaults()
	bot.exchange.okcoinChina.SetURL(OKCOIN_API_URL_CHINA)
	bot.exchange.okcoinChina.SetDefaults()
	bot.exchange.okcoinIntl.SetURL(OKCOIN_API_URL)
	bot.exchange.okcoinIntl.SetDefaults()
	bot.exchange.itbit.SetDefaults()
	bot.exchange.lakebtc.SetDefaults()
	bot.exchange.huobi.SetDefaults()

	err = RetrieveConfigCurrencyPairs(bot.config)

	if err != nil {
		log.Println("Fatal error retrieving config currency pairs. Error: ", err)
	}

	for _, exch := range bot.config.Exchanges {
		if bot.exchange.btcchina.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.btcchina.SetEnabled(false)
			} else {
				bot.exchange.btcchina.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.btcchina.PollingDelay = exch.PollingDelay
				bot.exchange.btcchina.Verbose = exch.Verbose
				bot.exchange.btcchina.Websocket = exch.Websocket
				go bot.exchange.btcchina.Run()
			}
		} else if bot.exchange.bitstamp.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.bitstamp.SetEnabled(false)
			} else {
				bot.exchange.bitstamp.SetAPIKeys(exch.ClientID, exch.APIKey, exch.APISecret)
				bot.exchange.bitstamp.PollingDelay = exch.PollingDelay
				bot.exchange.bitstamp.Verbose = exch.Verbose
				bot.exchange.bitstamp.Websocket = exch.Websocket
				go bot.exchange.bitstamp.Run()
			}
		} else if bot.exchange.bitfinex.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.bitfinex.SetEnabled(false)
			} else {
				bot.exchange.bitfinex.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.bitfinex.PollingDelay = exch.PollingDelay
				bot.exchange.bitfinex.Verbose = exch.Verbose
				bot.exchange.bitfinex.Websocket = exch.Websocket
				go bot.exchange.bitfinex.Run()
			}
		} else if bot.exchange.btce.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.btce.SetEnabled(false)
			} else {
				bot.exchange.btce.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.btce.PollingDelay = exch.PollingDelay
				bot.exchange.btce.Verbose = exch.Verbose
				bot.exchange.btce.Websocket = exch.Websocket
				go bot.exchange.btce.Run()
			}
		} else if bot.exchange.btcmarkets.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.btcmarkets.SetEnabled(false)
			} else {
				bot.exchange.btcmarkets.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.btcmarkets.PollingDelay = exch.PollingDelay
				bot.exchange.btcmarkets.Verbose = exch.Verbose
				bot.exchange.btcmarkets.Websocket = exch.Websocket
				go bot.exchange.btcmarkets.Run()
			}
		} else if bot.exchange.coinbase.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.coinbase.SetEnabled(false)
			} else {
				bot.exchange.coinbase.SetAPIKeys(exch.ClientID, exch.APIKey, exch.APISecret)
				bot.exchange.coinbase.PollingDelay = exch.PollingDelay
				bot.exchange.coinbase.Verbose = exch.Verbose
				bot.exchange.coinbase.Websocket = exch.Websocket
				go bot.exchange.coinbase.Run()
			}
		} else if bot.exchange.cryptsy.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.cryptsy.SetEnabled(false)
			} else {
				bot.exchange.cryptsy.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.cryptsy.PollingDelay = exch.PollingDelay
				bot.exchange.cryptsy.Verbose = exch.Verbose
				bot.exchange.cryptsy.Websocket = exch.Websocket
				go bot.exchange.cryptsy.Run()
			}
		} else if bot.exchange.okcoinChina.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.okcoinChina.SetEnabled(false)
			} else {
				bot.exchange.okcoinChina.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.okcoinChina.PollingDelay = exch.PollingDelay
				bot.exchange.okcoinChina.Verbose = exch.Verbose
				bot.exchange.okcoinChina.Websocket = exch.Websocket
				go bot.exchange.okcoinChina.Run()
			}
		} else if bot.exchange.okcoinIntl.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.okcoinIntl.SetEnabled(false)
			} else {
				bot.exchange.okcoinIntl.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.okcoinIntl.PollingDelay = exch.PollingDelay
				bot.exchange.okcoinIntl.Verbose = exch.Verbose
				bot.exchange.okcoinIntl.Websocket = exch.Websocket
				go bot.exchange.okcoinIntl.Run()
			}
		} else if bot.exchange.itbit.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.itbit.SetEnabled(false)
			} else {
				bot.exchange.itbit.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.itbit.PollingDelay = exch.PollingDelay
				bot.exchange.itbit.Verbose = exch.Verbose
				bot.exchange.itbit.Websocket = exch.Websocket
				go bot.exchange.itbit.Run()
			}
		} else if bot.exchange.kraken.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.kraken.SetEnabled(false)
			} else {
				bot.exchange.kraken.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.kraken.PollingDelay = exch.PollingDelay
				bot.exchange.kraken.Verbose = exch.Verbose
				bot.exchange.kraken.Websocket = exch.Websocket
				go bot.exchange.kraken.Run()
			}
		} else if bot.exchange.lakebtc.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.lakebtc.SetEnabled(false)
			} else {
				bot.exchange.lakebtc.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.lakebtc.PollingDelay = exch.PollingDelay
				bot.exchange.lakebtc.Verbose = exch.Verbose
				bot.exchange.lakebtc.Websocket = exch.Websocket
				go bot.exchange.lakebtc.Run()
			}
		} else if bot.exchange.huobi.GetName() == exch.Name {
			log.Printf("%s: %s (Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.Verbose))
			if !exch.Enabled {
				bot.exchange.huobi.SetEnabled(false)
			} else {
				bot.exchange.huobi.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.huobi.PollingDelay = exch.PollingDelay
				bot.exchange.huobi.Verbose = exch.Verbose
				bot.exchange.huobi.Websocket = exch.Websocket
				go bot.exchange.huobi.Run()
			}
		}
	}
	<-bot.shutdown
	Shutdown()
}

func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Captured %v.", sig)
		Shutdown()
	}()
}

func Shutdown() {
	log.Println("Bot shutting down..")
	err := SaveConfig()

	if err != nil {
		log.Println("Unable to save config.")
	} else {
		log.Println("Config file saved successfully.")
	}

	log.Println("Exiting.")
	os.Exit(1)
}
