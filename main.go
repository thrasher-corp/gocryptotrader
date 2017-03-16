package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

type Exchange struct {
	anx           ANX
	btcc          BTCC
	bitstamp      Bitstamp
	bitfinex      Bitfinex
	btce          BTCE
	btcmarkets    BTCMarkets
	gdax          GDAX
	gemini        Gemini
	okcoinChina   OKCoin
	okcoinIntl    OKCoin
	itbit         ItBit
	lakebtc       LakeBTC
	liqui         Liqui
	localbitcoins LocalBitcoins
	poloniex      Poloniex
	huobi         HUOBI
	kraken        Kraken
}

type Bot struct {
	config     config.Config
	exchange   Exchange
	exchanges  []IBotExchange
	tickers    []Ticker
	tickerChan chan Ticker
	shutdown   chan bool
}

var bot Bot

func setupBotExchanges() {
	for _, exch := range bot.config.Exchanges {
		for i := 0; i < len(bot.exchanges); i++ {
			if bot.exchanges[i] != nil {
				if bot.exchanges[i].GetName() == exch.Name {
					bot.exchanges[i].Setup(exch)
					if bot.exchanges[i].IsEnabled() {
						log.Printf("%s: Exchange support: %s (Authenticated API support: %s - Verbose mode: %s).\n", exch.Name, common.IsEnabled(exch.Enabled), common.IsEnabled(exch.AuthenticatedAPISupport), common.IsEnabled(exch.Verbose))
						bot.exchanges[i].Start()
					} else {
						log.Printf("%s: Exchange support: %s\n", exch.Name, common.IsEnabled(exch.Enabled))
					}
				}
			}
		}
	}
}

func main() {
	HandleInterrupt()
	log.Printf("Loading config file %s..\n", config.CONFIG_FILE)

	err := bot.config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Bot '%s' started.\n", bot.config.Name)
	AdjustGoMaxProcs()

	if bot.config.SMS.Enabled {
		err = bot.config.CheckSMSGlobalConfigValues()
		if err != nil {
			log.Println(err) // non fatal event
			bot.config.SMS.Enabled = false
		} else {
			log.Printf("SMS support enabled. Number of SMS contacts %d.\n", GetEnabledSMSContacts())
		}
	} else {
		log.Println("SMS support disabled.")
	}

	log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n", len(bot.config.Exchanges), bot.config.GetConfigEnabledExchanges())
	log.Println("Bot Exchange support:")

	bot.exchanges = []IBotExchange{
		new(ANX),
		new(Kraken),
		new(BTCC),
		new(Bitstamp),
		new(Bitfinex),
		new(BTCE),
		new(BTCMarkets),
		new(GDAX),
		new(Gemini),
		new(OKCoin),
		new(OKCoin),
		new(ItBit),
		new(LakeBTC),
		new(Liqui),
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

	setupBotExchanges()

	err = RetrieveConfigCurrencyPairs()

	if err != nil {
		log.Fatalf("Fatal error retrieving config currency AvailablePairs. Error: ", err)
	}

	go StartPortfolioWatcher()

	if bot.config.Webserver.Enabled {
		err := bot.config.CheckWebserverConfigValues()
		if err != nil {
			log.Println(err) // non fatal event
			//bot.config.Webserver.Enabled = false
		} else {
			listenAddr := bot.config.Webserver.ListenAddress
			log.Printf("HTTP Webserver support enabled. Listen URL: http://%s:%d/\n", common.ExtractHost(listenAddr), common.ExtractPort(listenAddr))
			router := NewRouter(bot.exchanges)
			log.Fatal(http.ListenAndServe(listenAddr, router))
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
	err := bot.config.SaveConfig()

	if err != nil {
		log.Println("Unable to save config.")
	} else {
		log.Println("Config file saved successfully.")
	}

	log.Println("Exiting.")
	os.Exit(1)
}
