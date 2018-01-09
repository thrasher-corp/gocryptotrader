package services

import (
	"log"
	"net/http"

	"github.com/thrasher-/gocryptotrader/bot"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/anx"
	"github.com/thrasher-/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-/gocryptotrader/exchanges/btcc"
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
	"github.com/thrasher-/gocryptotrader/exchanges/wex"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

// DefaultMain is the default logic for the bot
type DefaultMain struct{}

// Run starts the main default defined service for testing purposes
func (dm *DefaultMain) Run() {
	b := bot.GetBotP()

	b.HandleInterrupt()
	b.Config = &config.Cfg
	log.Printf("Loading config file %s..\n", b.ConfigFile)

	err := b.Config.LoadConfig(b.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	b.AdjustGoMaxProcs()

	log.Printf("Bot '%s' started.\n", b.Config.Name)
	log.Printf("Fiat display currency: %s.", b.Config.FiatDisplayCurrency)

	if b.Config.SMS.Enabled {
		b.Smsglobal = smsglobal.New(b.Config.SMS.Username, b.Config.SMS.Password,
			b.Config.Name, b.Config.SMS.Contacts)
		log.Printf(
			"SMS support enabled. Number of SMS contacts %d.\n",
			b.Smsglobal.GetEnabledContacts(),
		)
	} else {
		log.Println("SMS support disabled.")
	}

	log.Printf(
		"Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(b.Config.Exchanges), b.Config.GetConfigEnabledExchanges(),
	)
	log.Println("Bot Exchange support:")

	b.Exchanges = []exchange.IBotExchange{
		new(anx.ANX),
		new(kraken.Kraken),
		new(btcc.BTCC),
		new(bitstamp.Bitstamp),
		new(bitfinex.Bitfinex),
		new(bittrex.Bittrex),
		new(wex.WEX),
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

	for i := 0; i < len(b.Exchanges); i++ {
		if b.Exchanges[i] != nil {
			b.Exchanges[i].SetDefaults()
			log.Printf(
				"Exchange %s successfully set default settings.\n",
				b.Exchanges[i].GetName(),
			)
		}
	}

	b.SetupBotExchanges()

	if b.Config.CurrencyExchangeProvider == "yahoo" {
		currency.SetProvider(true)
	} else {
		currency.SetProvider(false)
	}

	log.Printf("Using %s as currency exchange provider.", b.Config.CurrencyExchangeProvider)

	b.Config.RetrieveConfigCurrencyPairs()
	err = currency.SeedCurrencyData(currency.BaseCurrencies)
	if err != nil {
		currency.SwapProvider()
		log.Printf("'%s' currency exchange provider failed, swapping to %s and testing..",
			b.Config.CurrencyExchangeProvider, currency.GetProvider())
		err = currency.SeedCurrencyData(currency.BaseCurrencies)
		if err != nil {
			log.Fatalf("Fatal error retrieving config currencies. Error: %s", err)
		}
	}

	log.Println("Successfully retrieved config currencies.")

	b.Portfolio = &portfolio.Portfolio
	b.Portfolio.SeedPortfolio(b.Config.Portfolio)
	b.SeedExchangeAccountInfo(b.GetAllEnabledExchangeAccountInfo().Data)
	go portfolio.StartPortfolioWatcher()

	log.Println("Starting websocket handler")
	go b.WebsocketHandler()

	go b.TickerUpdaterRoutine()
	go b.OrderbookUpdaterRoutine()

	if b.Config.Webserver.Enabled {
		listenAddr := b.Config.Webserver.ListenAddress
		log.Printf(
			"HTTP Webserver support enabled. Listen URL: http://%s:%d/\n",
			common.ExtractHost(listenAddr), common.ExtractPort(listenAddr),
		)
		router := b.NewRouter(b.Exchanges)
		log.Fatal(http.ListenAndServe(listenAddr, router))
	} else {
		log.Println("HTTP RESTful Webserver support disabled.")
	}

	<-b.ShutdownC
	b.Shutdown()
}
