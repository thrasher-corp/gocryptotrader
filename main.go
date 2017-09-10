package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/anx"
	"github.com/thrasher-/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-/gocryptotrader/exchanges/btcc"
	"github.com/thrasher-/gocryptotrader/exchanges/btce"
	"github.com/thrasher-/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-/gocryptotrader/exchanges/coinut"
	"github.com/thrasher-/gocryptotrader/exchanges/gdax"
	"github.com/thrasher-/gocryptotrader/exchanges/gemini"
	"github.com/thrasher-/gocryptotrader/exchanges/huobi"
	"github.com/thrasher-/gocryptotrader/exchanges/itbit"
	"github.com/thrasher-/gocryptotrader/exchanges/kraken"
	"github.com/thrasher-/gocryptotrader/exchanges/lakebtc"
	"github.com/thrasher-/gocryptotrader/exchanges/liqui"
	"github.com/thrasher-/gocryptotrader/exchanges/localbitcoins"
	"github.com/thrasher-/gocryptotrader/exchanges/okcoin"
	"github.com/thrasher-/gocryptotrader/exchanges/poloniex"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

// ExchangeMain contains all the necessary exchange packages
type ExchangeMain struct {
	anx           anx.ANX
	btcc          btcc.BTCC
	bitstamp      bitstamp.Bitstamp
	bitfinex      bitfinex.Bitfinex
	bittrex       bittrex.Bittrex
	btce          btce.BTCE
	btcmarkets    btcmarkets.BTCMarkets
	coinut        coinut.COINUT
	gdax          gdax.GDAX
	gemini        gemini.Gemini
	okcoinChina   okcoin.OKCoin
	okcoinIntl    okcoin.OKCoin
	itbit         itbit.ItBit
	lakebtc       lakebtc.LakeBTC
	liqui         liqui.Liqui
	localbitcoins localbitcoins.LocalBitcoins
	poloniex      poloniex.Poloniex
	huobi         huobi.HUOBI
	kraken        kraken.Kraken
}

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	config     *config.Config
	portfolio  *portfolio.Base
	exchange   ExchangeMain
	exchanges  []exchange.IBotExchange
	tickers    []ticker.Ticker
	shutdown   chan bool
	configFile string
}

var bot Bot

func setupBotExchanges() {
	for _, exch := range bot.config.Exchanges {
		for i := 0; i < len(bot.exchanges); i++ {
			if bot.exchanges[i] != nil {
				if bot.exchanges[i].GetName() == exch.Name {
					bot.exchanges[i].Setup(exch)
					if bot.exchanges[i].IsEnabled() {
						log.Printf(
							"%s: Exchange support: %s (Authenticated API support: %s - Verbose mode: %s).\n",
							exch.Name, common.IsEnabled(exch.Enabled),
							common.IsEnabled(exch.AuthenticatedAPISupport),
							common.IsEnabled(exch.Verbose),
						)
						bot.exchanges[i].Start()
					} else {
						log.Printf(
							"%s: Exchange support: %s\n", exch.Name,
							common.IsEnabled(exch.Enabled),
						)
					}
				}
			}
		}
	}
}

func main() {
	HandleInterrupt()

	//Handle flags
	flag.StringVar(&bot.configFile, "config", config.GetFilePath(""), "config file to load")
	flag.Parse()

	bot.config = &config.Cfg
	log.Printf("Loading config file %s..\n", bot.configFile)

	err := bot.config.LoadConfig(bot.configFile)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Bot '%s' started.\n", bot.config.Name)
	AdjustGoMaxProcs()

	if bot.config.SMS.Enabled {
		log.Printf(
			"SMS support enabled. Number of SMS contacts %d.\n",
			smsglobal.GetEnabledSMSContacts(bot.config.SMS),
		)
	} else {
		log.Println("SMS support disabled.")
	}

	log.Printf(
		"Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(bot.config.Exchanges), bot.config.GetConfigEnabledExchanges(),
	)
	log.Println("Bot Exchange support:")

	bot.exchanges = []exchange.IBotExchange{
		new(anx.ANX),
		new(kraken.Kraken),
		new(btcc.BTCC),
		new(bitstamp.Bitstamp),
		new(bitfinex.Bitfinex),
		new(bittrex.Bittrex),
		new(btce.BTCE),
		new(btcmarkets.BTCMarkets),
		new(coinut.COINUT),
		new(gdax.GDAX),
		new(gemini.Gemini),
		new(okcoin.OKCoin),
		new(okcoin.OKCoin),
		new(itbit.ItBit),
		new(lakebtc.LakeBTC),
		new(liqui.Liqui),
		new(localbitcoins.LocalBitcoins),
		new(poloniex.Poloniex),
		new(huobi.HUOBI),
	}

	for i := 0; i < len(bot.exchanges); i++ {
		if bot.exchanges[i] != nil {
			bot.exchanges[i].SetDefaults()
			log.Printf(
				"Exchange %s successfully set default settings.\n",
				bot.exchanges[i].GetName(),
			)
		}
	}

	setupBotExchanges()

	if bot.config.CurrencyExchangeProvider == "yahoo" {
		currency.SetProvider(true)
	} else {
		currency.SetProvider(false)
	}

	log.Printf("Using %s as currency exchange provider.", bot.config.CurrencyExchangeProvider)

	bot.config.RetrieveConfigCurrencyPairs()
	err = currency.SeedCurrencyData(currency.BaseCurrencies)
	if err != nil {
		currency.SwapProvider()
		log.Printf("'%s' currency exchange provider failed, swapping to %s and testing..",
			bot.config.CurrencyExchangeProvider, currency.GetProvider())
		err = currency.SeedCurrencyData(currency.BaseCurrencies)
		if err != nil {
			log.Fatalf("Fatal error retrieving config currencies. Error: %s", err)
		}
	}

	log.Println("Successfully retrieved config currencies.")

	bot.portfolio = &portfolio.Portfolio
	bot.portfolio.SeedPortfolio(bot.config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)
	go portfolio.StartPortfolioWatcher()

	log.Println("Starting websocket handler")
	go WebsocketHandler()

	go TickerUpdaterRoutine()
	go OrderbookUpdaterRoutine()

	if bot.config.Webserver.Enabled {
		listenAddr := bot.config.Webserver.ListenAddress
		log.Printf(
			"HTTP Webserver support enabled. Listen URL: http://%s:%d/\n",
			common.ExtractHost(listenAddr), common.ExtractPort(listenAddr),
		)
		router := NewRouter(bot.exchanges)
		log.Fatal(http.ListenAndServe(listenAddr, router))
	} else {
		log.Println("HTTP RESTful Webserver support disabled.")
	}

	<-bot.shutdown
	Shutdown()
}

// AdjustGoMaxProcs adjusts the maximum processes that the CPU can handle.
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
	if i := runtime.GOMAXPROCS(maxProcs); i != maxProcs {
		log.Fatal("Go Max Procs were not set correctly.")
	}
	log.Println("Set GOMAXPROCS to:", maxProcs)
}

// HandleInterrupt monitors and captures the SIGTERM in a new goroutine then
// shuts down bot
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Captured %v.", sig)
		Shutdown()
	}()
}

// Shutdown correctly shuts down bot saving configuration files
func Shutdown() {
	log.Println("Bot shutting down..")
	bot.config.Portfolio = portfolio.Portfolio
	err := bot.config.SaveConfig(bot.configFile)

	if err != nil {
		log.Println("Unable to save config.")
	} else {
		log.Println("Config file saved successfully.")
	}

	log.Println("Exiting.")
	os.Exit(1)
}

// SeedExchangeAccountInfo seeds account info
func SeedExchangeAccountInfo(data []exchange.AccountInfo) {
	if len(data) == 0 {
		return
	}

	port := portfolio.GetPortfolio()

	for i := 0; i < len(data); i++ {
		exchangeName := data[i].ExchangeName
		for j := 0; j < len(data[i].Currencies); j++ {
			currencyName := data[i].Currencies[j].CurrencyName
			onHold := data[i].Currencies[j].Hold
			avail := data[i].Currencies[j].TotalValue
			total := onHold + avail

			if !port.ExchangeAddressExists(exchangeName, currencyName) {
				if total <= 0 {
					continue
				}
				log.Printf("Portfolio: Adding new exchange address: %s, %s, %f, %s\n",
					exchangeName, currencyName, total, portfolio.PortfolioAddressExchange)
				port.Addresses = append(
					port.Addresses,
					portfolio.Address{Address: exchangeName, CoinType: currencyName,
						Balance: total, Description: portfolio.PortfolioAddressExchange},
				)
			} else {
				if total <= 0 {
					log.Printf("Portfolio: Removing %s %s entry.\n", exchangeName,
						currencyName)
					port.RemoveExchangeAddress(exchangeName, currencyName)
				} else {
					balance, ok := port.GetAddressBalance(exchangeName, currencyName, portfolio.PortfolioAddressExchange)
					if !ok {
						continue
					}
					if balance != total {
						log.Printf("Portfolio: Updating %s %s entry with balance %f.\n",
							exchangeName, currencyName, total)
						port.UpdateExchangeAddressBalance(exchangeName, currencyName, total)
					}
				}
			}
		}
	}
}
