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
			if !exch.Enabled {
				bot.exchange.btcchina.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.btcchina.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.btcchina.PollingDelay = exch.PollingDelay
				go bot.exchange.btcchina.Run()

				if exch.Verbose {
					bot.exchange.btcchina.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
					go bot.exchange.btcchina.Run()
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
				bot.exchange.bitstamp.PollingDelay = exch.PollingDelay
				go bot.exchange.bitstamp.Run()

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
				bot.exchange.bitfinex.PollingDelay = exch.PollingDelay
				go bot.exchange.bitfinex.Run()
				
				if exch.Verbose {
					bot.exchange.bitfinex.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.btce.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.btce.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.btce.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.btce.PollingDelay = exch.PollingDelay
				go bot.exchange.btce.Run()

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
				bot.exchange.btcmarkets.PollingDelay = exch.PollingDelay
				go bot.exchange.btcmarkets.Run()

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
				bot.exchange.coinbase.PollingDelay = exch.PollingDelay
				go bot.exchange.coinbase.Run()

				if exch.Verbose {
					bot.exchange.coinbase.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
			}
		} else if bot.exchange.cryptsy.GetName() == exch.Name {
			if !exch.Enabled {
				bot.exchange.cryptsy.SetEnabled(false)
				log.Printf("%s disabled.\n", exch.Name)
			} else {
				log.Printf("%s enabled.\n", exch.Name)
				bot.exchange.cryptsy.SetAPIKeys(exch.APIKey, exch.APISecret)
				bot.exchange.cryptsy.PollingDelay = exch.PollingDelay
				go bot.exchange.cryptsy.Run()

				if exch.Verbose {
					bot.exchange.cryptsy.Verbose = true
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
				bot.exchange.okcoinChina.PollingDelay = exch.PollingDelay
				go bot.exchange.okcoinChina.Run()

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
				bot.exchange.okcoinIntl.PollingDelay = exch.PollingDelay
				go bot.exchange.okcoinIntl.Run()

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
				bot.exchange.itbit.PollingDelay = exch.PollingDelay
				go bot.exchange.itbit.Run()

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
				bot.exchange.kraken.PollingDelay = exch.PollingDelay
				go bot.exchange.kraken.Run()

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
				bot.exchange.lakebtc.PollingDelay = exch.PollingDelay
				go bot.exchange.lakebtc.Run()

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
				bot.exchange.huobi.PollingDelay = exch.PollingDelay
				go bot.exchange.huobi.Run()

				if exch.Verbose {
					bot.exchange.huobi.Verbose = true
					log.Printf("%s Verbose output enabled.\n", exch.Name)
				} else {
					log.Printf("%s Verbose output disabled.\n", exch.Name)
				}
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
