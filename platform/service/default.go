package service

import (
	"log"
	"os"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/platform"
)

// StartDefault starts the main default GCT platform service
func StartDefault(configFilePath string, verbose, dryRun bool) {
	bot := platform.GetBot(verbose, dryRun, configFilePath)
	bot.SetConfig()
	bot.AdjustGoMaxProcs()

	if bot.Verbose {
		log.Printf("Bot '%s' started.\n", bot.Config.Name)
		log.Printf("Fiat display currency: %s.", bot.Config.FiatDisplayCurrency)
		log.Printf("Bot dry run mode: %v\n", common.IsEnabled(bot.DryRun))
	}

	bot.SetSMSGlobal()

	if bot.Verbose {
		log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n",
			len(bot.Config.Exchanges),
			bot.Config.CountEnabledExchanges())
	}

	bot.SetExchanges()
	bot.SetCurrencyProvider()
	bot.RetrieveCurrencyPairs()
	bot.SetPorfolio()
	bot.SeedExchangeAccountInfo(bot.GetAllEnabledExchangeAccountInfo().Data)

	if bot.Verbose {
		log.Println("Starting websocket handler")
	}

	go bot.WebsocketHandler()
	bot.StartRoutines()
	bot.StartWebserver()

	<-bot.Wait
	os.Exit(0)
}
