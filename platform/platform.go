package platform

// Bot contains configuration, portfolio, exchange & ticker data and is the
import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

// Bot is the overarching type across the entire GoCryptoTrader code base.
type Bot struct {
	Config    *config.Config
	Smsglobal *smsglobal.Base
	Portfolio *portfolio.Base
	Exchanges []exchange.IBotExchange
	Orders    []*Order

	Wait       chan bool
	shutdown   bool
	DryRun     bool
	ConfigFile string
	Verbose    bool

	sync.WaitGroup
}

const banner = `
   ______        ______                     __        ______                  __
  / ____/____   / ____/_____ __  __ ____   / /_ ____ /_  __/_____ ______ ____/ /___   _____
 / / __ / __ \ / /    / ___// / / // __ \ / __// __ \ / /  / ___// __  // __  // _ \ / ___/
/ /_/ // /_/ // /___ / /   / /_/ // /_/ // /_ / /_/ // /  / /   / /_/ // /_/ //  __// /
\____/ \____/ \____//_/    \__, // .___/ \__/ \____//_/  /_/    \__,_/ \__,_/ \___//_/
                          /____//_/
`

// GetBot returns a pointer to the main bot
func GetBot(verbose, dryrun bool, configFile string) *Bot {
	bot := new(Bot)
	fmt.Println(banner)
	fmt.Println(bot.BuildVersion(false))
	bot.HandleInterrupt()
	bot.DryRun = dryrun
	bot.Verbose = verbose

	if configFile != "" {
		bot.ConfigFile = configFile
		return bot
	}
	bot.ConfigFile = config.GetFilePath("")
	return bot
}

// SetConfig sets configuration to bot with default configration values
func (b *Bot) SetConfig() {
	b.Config = config.GetConfig()

	if b.Verbose {
		log.Printf("Loading config file %s..\n", b.ConfigFile)
	}

	err := b.Config.LoadConfig(b.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

}

// SetSMSGlobal sets Smsglobal to bot with default values
func (b *Bot) SetSMSGlobal() {
	if b.Config.SMS.Enabled {
		b.Smsglobal = smsglobal.New(b.Config.SMS.Username,
			b.Config.SMS.Password,
			b.Config.Name,
			b.Config.SMS.Contacts)
		if b.Verbose {
			log.Printf("SMS support enabled. Number of SMS contacts %d.\n",
				b.Smsglobal.GetEnabledContacts())
		}
	} else {
		if b.Verbose {
			log.Println("SMS support disabled.")
		}
	}
}

// SetExchanges sets exchanges to bot with default values
func (b *Bot) SetExchanges() {
	b.SetupExchanges()
	if len(b.Exchanges) == 0 {
		log.Fatalf("No exchanges were able to be loaded. Exiting")
	}
	// TODO: Fix hack, allow 5 seconds to update exchange settings
	time.Sleep(time.Second * 5)
}

// SetCurrencyProvider sets current currency fetching provider
func (b *Bot) SetCurrencyProvider() {
	if b.Config.CurrencyExchangeProvider == "yahoo" {
		currency.SetProvider(true)
	} else {
		currency.SetProvider(false)
	}
	if b.Verbose {
		log.Printf("Currency exchange provider: %s.", b.Config.CurrencyExchangeProvider)
	}
}

// RetrieveCurrencyPairs retrieves current currency pairs for an exchange
func (b *Bot) RetrieveCurrencyPairs() {
	b.Config.RetrieveConfigCurrencyPairs(true)
	err := currency.SeedCurrencyData(common.JoinStrings(currency.BaseCurrencies, ","))
	if err != nil {
		currency.SwapProvider()

		if b.Verbose {
			log.Printf("'%s' currency exchange provider failed, swapping to %s and testing..",
				b.Config.CurrencyExchangeProvider,
				currency.GetProvider())
		}

		err = currency.SeedCurrencyData(common.JoinStrings(currency.BaseCurrencies, ","))
		if err != nil {
			log.Fatalf("Fatal error retrieving config currencies. Error: %s", err)
		}
	}
	if b.Verbose {
		log.Println("Successfully retrieved config currencies.")
	}
}

// SetPorfolio sets Portfolio to bot with default values
func (b *Bot) SetPorfolio() {
	b.Portfolio = portfolio.GetPortfolio()
	b.Portfolio.SeedPortfolio(b.Config.Portfolio)
}

// StartRoutines starts the default bot routines
func (b *Bot) StartRoutines() {
	go b.StartPortfolioWatcherRoutine()
	go b.TickerUpdaterRoutine()
	go b.OrderbookUpdaterRoutine()
}

// StartWebserver starts the default web server for the bot
func (b *Bot) StartWebserver() {
	if b.Config.Webserver.Enabled {
		listenAddr := b.Config.Webserver.ListenAddress

		if b.Verbose {
			log.Printf("HTTP Webserver support enabled. Listen URL: http://%s:%d/\n",
				common.ExtractHost(listenAddr),
				common.ExtractPort(listenAddr))
		}
		router := b.NewRouter(b.Exchanges)
		log.Fatal(http.ListenAndServe(listenAddr, router))
	} else {
		if b.Verbose {
			log.Println("HTTP RESTful Webserver support disabled.")
		}
	}
}

// AdjustGoMaxProcs adjusts the maximum processes that the CPU can handle.
func (b *Bot) AdjustGoMaxProcs() {
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
func (b *Bot) HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Captured %v.", sig)
		b.shutdown = true
		b.WaitGroup.Wait()
		b.Shutdown()
	}()
}

// Shutdown correctly shuts down bot saving configuration files
func (b *Bot) Shutdown() {
	log.Println("Bot shutting down..")

	if !b.DryRun {
		b.Config.Portfolio = *b.Portfolio

		err := b.Config.SaveConfig(b.ConfigFile)
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
func (b *Bot) SeedExchangeAccountInfo(data []exchange.AccountInfo) {
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
