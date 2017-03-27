package main

import (
	//	"github.com/gorilla/mux"
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
	"github.com/thrasher-/gocryptotrader/exchanges/btcc"
	"github.com/thrasher-/gocryptotrader/exchanges/btce"
	"github.com/thrasher-/gocryptotrader/exchanges/btcmarkets"
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

type ExchangeMain struct {
	anx           anx.ANX
	btcc          btcc.BTCC
	bitstamp      bitstamp.Bitstamp
	bitfinex      bitfinex.Bitfinex
	btce          btce.BTCE
	btcmarkets    btcmarkets.BTCMarkets
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

type Bot struct {
	config    *config.Config
	portfolio *portfolio.PortfolioBase
	exchange  ExchangeMain
	exchanges []exchange.IBotExchange
	tickers   []ticker.Ticker
	shutdown  chan bool
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
	bot.config = &config.Cfg
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
			log.Printf("SMS support enabled. Number of SMS contacts %d.\n", smsglobal.GetEnabledSMSContacts(bot.config.SMS))
		}
	} else {
		log.Println("SMS support disabled.")
	}

	log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n", len(bot.config.Exchanges), bot.config.GetConfigEnabledExchanges())
	log.Println("Bot Exchange support:")

	bot.exchanges = []exchange.IBotExchange{
		new(anx.ANX),
		new(kraken.Kraken),
		new(btcc.BTCC),
		new(bitstamp.Bitstamp),
		new(bitfinex.Bitfinex),
		new(btce.BTCE),
		new(btcmarkets.BTCMarkets),
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
			log.Printf("Exchange %s successfully set default settings.\n", bot.exchanges[i].GetName())
		}
	}

	setupBotExchanges()

	bot.config.RetrieveConfigCurrencyPairs()

	err = currency.SeedCurrencyData(currency.BaseCurrencies)
	if err != nil {
		log.Fatalf("Fatal error retrieving config currencies. Error: ", err)
	}

	log.Println("Successfully retrieved config currencies.")

	bot.portfolio = &portfolio.Portfolio
	bot.portfolio.SeedPortfolio(bot.config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)
	go portfolio.StartPortfolioWatcher()

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
	bot.config.Portfolio = portfolio.Portfolio
	err := bot.config.SaveConfig()

	if err != nil {
		log.Println("Unable to save config.")
	} else {
		log.Println("Config file saved successfully.")
	}

	log.Println("Exiting.")
	os.Exit(1)
}

func SeedExchangeAccountInfo(data []exchange.ExchangeAccountInfo) {
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

			if total <= 0 {
				continue
			}

			if !port.ExchangeAddressExists(exchangeName, currencyName) {
				port.Addresses = append(port.Addresses, portfolio.PortfolioAddress{Address: exchangeName, CoinType: currencyName, Balance: total, Decscription: portfolio.PORTFOLIO_ADDRESS_EXCHANGE})
			} else {
				port.UpdateExchangeAddressBalance(exchangeName, currencyName, total)
			}
		}
	}

}
