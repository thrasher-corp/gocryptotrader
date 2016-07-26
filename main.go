package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
)

type Exchange struct {
	anx           ANX
	btcc          BTCC
	bitstamp      Bitstamp
	bitfinex      Bitfinex
	brightonpeak  BrightonPeak
	btce          BTCE
	btcmarkets    BTCMarkets
	coinbase      Coinbase
	gemini        Gemini
	okcoinChina   OKCoin
	okcoinIntl    OKCoin
	itbit         ItBit
	lakebtc       LakeBTC
	localbitcoins LocalBitcoins
	poloniex      Poloniex
	huobi         HUOBI
	kraken        Kraken
}

type Bot struct {
	config    Config
	exchange  Exchange
	exchanges []IBotExchange
	shutdown  chan bool
}

var bot Bot

func SetupBotConfiguration(s IBotExchange, exch Exchanges) {
	s.SetDefaults()
	if s.GetName() == exch.Name {
		s.Setup(exch)
		if s.IsEnabled() {
			log.Printf("%s: Exchange support: %s (Authenticated API support: %s - Verbose mode: %s).\n", exch.Name, IsEnabled(exch.Enabled), IsEnabled(exch.AuthenticatedAPISupport), IsEnabled(exch.Verbose))
			s.Start()
		} else {
			log.Printf("%s: Exchange support: %s\n", exch.Name, IsEnabled(exch.Enabled))
		}
	}
}

func main() {
	HandleInterrupt()
	log.Println("Loading config file config.json..")

	err := errors.New("")
	bot.config, err = ReadConfig()
	if err != nil {
		log.Printf("Fatal error opening config.json file. Error: %s", err)
		return
	}
	log.Println("Config file loaded. Checking settings.. ")

	err = CheckExchangeConfigValues()
	if err != nil {
		log.Println("Fatal error checking config values. Error:", err)
		return
	}

	log.Printf("Bot '%s' started.\n", bot.config.Name)
	AdjustGoMaxProcs()

	if bot.config.SMS.Enabled {
		err = CheckSMSGlobalConfigValues()
		if err != nil {
			log.Println(err) // non fatal event
			bot.config.SMS.Enabled = false
		} else {
			log.Printf("SMS support enabled. Number of SMS contacts %d.\n", GetEnabledSMSContacts())
		}
	} else {
		log.Println("SMS support disabled.")
	}

	log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n", len(bot.config.Exchanges), GetEnabledExchanges())
	log.Println("Bot Exchange support:")

	bot.exchange.okcoinIntl.APIUrl = OKCOIN_API_URL
	bot.exchange.okcoinChina.APIUrl = OKCOIN_API_URL_CHINA

	bot.exchanges = []IBotExchange{
		new(ANX),
		new(Kraken),
		new(BTCC),
		new(Bitstamp),
		new(BrightonPeak),
		new(Bitfinex),
		new(BTCE),
		new(BTCMarkets),
		new(Coinbase),
		new(Gemini),
		new(OKCoin),
		new(OKCoin),
		new(ItBit),
		new(LakeBTC),
		new(LocalBitcoins),
		new(Poloniex),
		new(HUOBI),
	}
	for i := 0; i < len(bot.exchanges); i++ {
		if bot.exchanges[i] != nil {
			bot.exchanges[i].SetDefaults()
			log.Printf("Exchange %s successfully set default settings.\n", bot.exchanges[i].GetName())
		}
	}

	err = RetrieveConfigCurrencyPairs(bot.config)

	if err != nil {
		log.Println("Fatal error retrieving config currency AvailablePairs. Error: ", err)
	}

	for _, exch := range bot.config.Exchanges {
		for i := 0; i < len(bot.exchanges); i++ {
			if bot.exchanges[i] != nil {
				SetupBotConfiguration(bot.exchanges[i], exch)
			}
		}
	}

	if bot.config.Webserver.Enabled {
		err := CheckWebserverValues()
		if err != nil {
			log.Println(err) // non fatal event
			//bot.config.Webserver.Enabled = false
		} else {
			log.Println("HTTP Webserver support enabled.")
			router := NewRouter(bot.exchanges)
			log.Fatal(http.ListenAndServe(bot.config.Webserver.ListenAddress, router))
		}
	}
	if !bot.config.Webserver.Enabled {
		log.Println("HTTP Webserver support disabled.")
	}

	<-bot.shutdown
	Shutdown()
}

func AdjustGoMaxProcs() {
	log.Println("Adjusting bot runtime performance..")
	maxProcsEnv := os.Getenv("GOMAXPROCS")
	maxProcs := runtime.NumCPU()
	log.Println("Number of CPU's detected:", maxProcs)

	if maxProcsEnv != "" {
		log.Println("GOMAXPROCS env =", maxProcsEnv)
		env, err := strconv.Atoi(maxProcsEnv)

		if err != nil {
			log.Println("Unable to convert GOMAXPROCS to int, using", maxProcs)
		} else {
			maxProcs = env
		}
	}
	log.Println("Set GOMAXPROCS to:", maxProcs)
	runtime.GOMAXPROCS(maxProcs)
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
