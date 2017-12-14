package bot

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	Config     *config.Config
	Smsglobal  *smsglobal.Base
	Portfolio  *portfolio.Base
	Exchange   ExchangeMain
	Exchanges  []exchange.IBotExchange
	Tickers    []ticker.Ticker
	ShutdownC  chan bool
	ConfigFile string
}

// GetBotP returns a pointer to the Bot struct
func GetBotP() *Bot {
	b := Bot{}
	return &b
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
		b.Shutdown()
	}()
}

// Shutdown correctly shuts down bot saving configuration files
func (b *Bot) Shutdown() {
	log.Println("Bot shutting down..")
	b.Config.Portfolio = portfolio.Portfolio
	err := b.Config.SaveConfig(b.ConfigFile)

	if err != nil {
		log.Println("Unable to save config.")
	} else {
		log.Println("Config file saved successfully.")
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
