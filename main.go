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
	"time"

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
	log.Printf("Bot dry run mode: %v\n", common.IsEnabled(bot.dryRun))

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

	SetupExchanges()
	if len(bot.exchanges) == 0 {
		log.Fatalf("No exchanges were able to be loaded. Exiting")
	}
	// TODO: Fix hack, allow 5 seconds to update exchange settings
	time.Sleep(time.Second * 5)

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

	if !bot.dryRun {
		err := bot.config.SaveConfig(bot.configFile)

		if err != nil {
			log.Println("Unable to save config.")
		} else {
			log.Println("Config file saved successfully.")
		}
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
