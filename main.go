package main

import (
	"flag"
	"fmt"
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
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	config     *config.Config
	smsglobal  *smsglobal.Base
	portfolio  *portfolio.Base
	exchanges  []exchange.IBotExchange
	shutdown   chan bool
	dryRun     bool
	configFile string
}

const banner = `
   ______        ______                     __        ______                  __           
  / ____/____   / ____/_____ __  __ ____   / /_ ____ /_  __/_____ ______ ____/ /___   _____
 / / __ / __ \ / /    / ___// / / // __ \ / __// __ \ / /  / ___// __  // __  // _ \ / ___/
/ /_/ // /_/ // /___ / /   / /_/ // /_/ // /_ / /_/ // /  / /   / /_/ // /_/ //  __// /    
\____/ \____/ \____//_/    \__, // .___/ \__/ \____//_/  /_/    \__,_/ \__,_/ \___//_/     
                          /____//_/                                                        
`

var bot Bot

func main() {
	bot.shutdown = make(chan bool)
	HandleInterrupt()

	//Handle flags
	flag.StringVar(&bot.configFile, "config", config.GetFilePath(""), "config file to load")
	dryrun := flag.Bool("dryrun", false, "dry runs bot, doesn't save config file")
	version := flag.Bool("version", false, "retrieves current GoCryptoTrader version")
	flag.Parse()

	if *version {
		fmt.Printf(BuildVersion(true))
		os.Exit(0)
	}

	if *dryrun {
		bot.dryRun = true
	}

	bot.config = &config.Cfg
	fmt.Println(banner)
	fmt.Println(BuildVersion(false))
	log.Printf("Loading config file %s..\n", bot.configFile)

	err := bot.config.LoadConfig(bot.configFile)
	if err != nil {
		log.Fatal(err)
	}

	AdjustGoMaxProcs()
	log.Printf("Bot '%s' started.\n", bot.config.Name)
	log.Printf("Fiat display currency: %s.", bot.config.FiatDisplayCurrency)
	log.Printf("Bot dry run mode: %v.\n", common.IsEnabled(bot.dryRun))

	if bot.config.SMS.Enabled {
		bot.smsglobal = smsglobal.New(bot.config.SMS.Username, bot.config.SMS.Password,
			bot.config.Name, bot.config.SMS.Contacts)
		log.Printf(
			"SMS support enabled. Number of SMS contacts %d.\n",
			bot.smsglobal.GetEnabledContacts(),
		)
	} else {
		log.Println("SMS support disabled.")
	}

	log.Printf(
		"Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(bot.config.Exchanges), bot.config.CountEnabledExchanges(),
	)

	common.HTTPClient = common.NewHTTPClientWithTimeout(bot.config.GlobalHTTPTimeout)
	log.Printf("Global HTTP request timeout: %v.\n", common.HTTPClient.Timeout)

	SetupExchanges()
	if len(bot.exchanges) == 0 {
		log.Fatalf("No exchanges were able to be loaded. Exiting")
	}

	if bot.config.CurrencyExchangeProvider == "yahoo" {
		currency.SetProvider(true)
	} else {
		currency.SetProvider(false)
	}
	log.Printf("Currency exchange provider: %s.", bot.config.CurrencyExchangeProvider)

	bot.config.RetrieveConfigCurrencyPairs(true)
	err = currency.SeedCurrencyData(common.JoinStrings(currency.BaseCurrencies, ","))
	if err != nil {
		currency.SwapProvider()
		log.Printf("'%s' currency exchange provider failed, swapping to %s and testing..",
			bot.config.CurrencyExchangeProvider, currency.GetProvider())
		err = currency.SeedCurrencyData(common.JoinStrings(currency.BaseCurrencies, ","))
		if err != nil {
			log.Fatalf("Fatal error retrieving config currencies. Error: %s", err)
		}
	}
	log.Println("Successfully retrieved config currencies.")

	bot.portfolio = &portfolio.Portfolio
	bot.portfolio.SeedPortfolio(bot.config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)
	go portfolio.StartPortfolioWatcher()
	go TickerUpdaterRoutine()
	go OrderbookUpdaterRoutine()

	if bot.config.Webserver.Enabled {
		listenAddr := bot.config.Webserver.ListenAddress
		log.Printf(
			"HTTP Webserver support enabled. Listen URL: http://%s:%d/\n",
			common.ExtractHost(listenAddr), common.ExtractPort(listenAddr),
		)

		router := NewRouter(bot.exchanges)
		go func() {
			err = http.ListenAndServe(listenAddr, router)
			if err != nil {
				log.Fatal(err)
			}
		}()

		log.Println("HTTP Webserver started successfully.")
		log.Println("Starting websocket handler.")
		StartWebsocketHandler()
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
		log.Printf("Captured %v, shutdown requested.", sig)
		bot.shutdown <- true
	}()
}

// Shutdown correctly shuts down bot saving configuration files
func Shutdown() {
	log.Println("Bot shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		bot.config.Portfolio = portfolio.Portfolio
	}

	if !bot.dryRun {
		err := bot.config.SaveConfig(bot.configFile)

		if err != nil {
			log.Println("Unable to save config.")
		} else {
			log.Println("Config file saved successfully.")
		}
	}

	log.Println("Exiting.")
	os.Exit(0)
}
