package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/connchecker"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/coinmarketcap"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/ntpclient"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	config              *config.Config
	portfolio           *portfolio.Base
	exchanges           []exchange.IBotExchange
	comms               *communications.Communications
	shutdown            chan bool
	dryRun              bool
	configFile          string
	dataDir             string
	connectivity        *connchecker.Checker
	verbosity           bool
	coinMarketCap       bool
	fxCurrencyConverter bool
	fxCurrencyLayer     bool
	fxFixer             bool
	fxOpenExchangeRates bool
	sync.Mutex
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
	// Activate and setup Bots requirements. Note order is important.
	ActivateShutdownChan()
	HandleInterrupt()
	PrintBanner()
	ActivateFlags()
	PrintVersion()
	LoadConfigFile()
	CreateDataDir()
	ActivateLogger()
	ActivateNTP()
	ActivateConnectivityMonitor()
	AdjustGoMaxProcs()
	ActivateHTTPClient()
	ActivateCommunicationMediums()
	SetupExchanges()
	ActivateStorageUpdater()
	PrintBotSummary()
	ActivatePortfolioWatcher()
	ActivateWebServer()

	go TickerUpdaterRoutine()
	go OrderbookUpdaterRoutine()
	go WebsocketRoutine(bot.verbosity)

	<-bot.shutdown
	Shutdown()
}

// ActivateFlags Sets up all the cli flags
func ActivateFlags() {
	defaultPath, err := config.GetFilePath("")
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&bot.configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&bot.dataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	flag.BoolVar(&bot.dryRun, "dryrun", false, "dry runs bot, doesn't save config file")
	flag.BoolVar(&bot.verbosity, "verbose", false, "increases logging verbosity for GoCryptoTrader")
	flag.BoolVar(&bot.coinMarketCap, "c", false, "overrides config and runs currency analaysis")
	flag.BoolVar(&bot.fxCurrencyConverter, "fxa", false, "overrides config and sets up foreign exchange Currency Converter")
	flag.BoolVar(&bot.fxCurrencyLayer, "fxb", false, "overrides config and sets up foreign exchange Currency Layer")
	flag.BoolVar(&bot.fxFixer, "fxc", false, "overrides config and sets up foreign exchange Fixer.io")
	flag.BoolVar(&bot.fxOpenExchangeRates, "fxd", false, "overrides config and sets up foreign exchange Open Exchange Rates")

	version := flag.Bool("version", false, "retrieves current GoCryptoTrader version")

	flag.Parse()

	if *version {
		fmt.Print(BuildVersion(true))
		os.Exit(0)
	}
}

// ActivateWebServer Sets up a local web server
func ActivateWebServer() {
	if bot.config.Webserver.Enabled {
		listenAddr := bot.config.Webserver.ListenAddress
		log.Debugf(
			"HTTP Webserver support enabled. Listen URL: http://%s:%d/\n",
			common.ExtractHost(listenAddr), common.ExtractPort(listenAddr),
		)

		router := NewRouter()
		go func() {
			err := http.ListenAndServe(listenAddr, router)
			if err != nil {
				log.Fatal(err)
			}
		}()

		log.Debugln("HTTP Webserver started successfully.")
		log.Debugln("Starting websocket handler.")
		StartWebsocketHandler()
	} else {
		log.Debugln("HTTP RESTful Webserver support disabled.")
	}
}

// ActivateConnectivityMonitor Sets up internet connectivity monitor
func ActivateConnectivityMonitor() {
	var err error
	bot.connectivity, err = connchecker.New(bot.config.ConnectionMonitor.DNSList,
		bot.config.ConnectionMonitor.PublicDomainList,
		bot.config.ConnectionMonitor.CheckInterval)
	if err != nil {
		log.Fatalf("Connectivity checker failure: %s", err)
	}
}

// ActivateNTP Sets up NTP client
func ActivateNTP() {
	if bot.config.NTPClient.Level != -1 {
		bot.config.CheckNTPConfig()
		NTPTime, errNTP := ntpclient.NTPClient(bot.config.NTPClient.Pool)
		currentTime := time.Now()
		if errNTP != nil {
			log.Warnf("NTPClient failed to create: %v", errNTP)
		} else {
			NTPcurrentTimeDifference := NTPTime.Sub(currentTime)
			configNTPTime := *bot.config.NTPClient.AllowedDifference
			configNTPNegativeTime := (*bot.config.NTPClient.AllowedNegativeDifference - (*bot.config.NTPClient.AllowedNegativeDifference * 2))
			if NTPcurrentTimeDifference > configNTPTime || NTPcurrentTimeDifference < configNTPNegativeTime {
				log.Warnf("Time out of sync (NTP): %v | (time.Now()): %v | (Difference): %v | (Allowed): +%v / %v", NTPTime, currentTime, NTPcurrentTimeDifference, configNTPTime, configNTPNegativeTime)
				if bot.config.NTPClient.Level == 0 {
					disable, errNTP := bot.config.DisableNTPCheck(os.Stdin)
					if errNTP != nil {
						log.Errorf("failed to disable ntp time check reason: %v", errNTP)
					} else {
						log.Info(disable)
					}
				}
			}
		}
	}
}

// AdjustGoMaxProcs adjusts the maximum processes that the CPU can handle.
func AdjustGoMaxProcs() {
	log.Debugln("Adjusting bot runtime performance..")
	maxProcsEnv := os.Getenv("GOMAXPROCS")
	maxProcs := runtime.NumCPU()
	log.Debugln("Number of CPU's detected:", maxProcs)

	if maxProcsEnv != "" {
		log.Debugln("GOMAXPROCS env =", maxProcsEnv)
		env, err := strconv.Atoi(maxProcsEnv)
		if err != nil {
			log.Debugf("Unable to convert GOMAXPROCS to int, using %d", maxProcs)
		} else {
			maxProcs = env
		}
	}
	if i := runtime.GOMAXPROCS(maxProcs); i != maxProcs {
		log.Error("Go Max Procs were not set correctly.")
	}
	log.Debugln("Set GOMAXPROCS to:", maxProcs)
}

// HandleInterrupt monitors and captures the SIGTERM in a new goroutine then
// shuts down bot
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Debugf("Captured %v, shutdown requested.", sig)
		close(bot.shutdown)
	}()
}

// Shutdown correctly shuts down bot saving configuration files
func Shutdown() {
	log.Debugln("Bot shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		bot.config.Portfolio = portfolio.Portfolio
	}

	if !bot.dryRun {
		err := bot.config.SaveConfig(bot.configFile)

		if err != nil {
			log.Warn("Unable to save config.")
		} else {
			log.Debugln("Config file saved successfully.")
		}
	}

	log.Debugln("Exiting.")

	log.CloseLogFile()
	os.Exit(0)
}

// PrintBanner Prints banner to stdout
func PrintBanner() {
	fmt.Print(banner)
}

// PrintVersion Print the version to stdout
func PrintVersion() {
	fmt.Println(BuildVersion(false))
}

// LoadConfigFile loads the config
func LoadConfigFile() {
	bot.config = &config.Cfg
	log.Debugf("Loading config file %s..\n", bot.configFile)
	err := bot.config.LoadConfig(bot.configFile)
	if err != nil {
		log.Fatalf("Failed to load config. Err: %s", err)
	}
}

// ActivateShutdownChan initialises a global shutdown channel
func ActivateShutdownChan() {
	bot.shutdown = make(chan bool)
}

// CreateDataDir creates the data directory
func CreateDataDir() {
	err := common.CreateDir(bot.dataDir)
	if err != nil {
		log.Fatalf("Failed to open/create data directory: %s. Err: %s", bot.dataDir, err)
	}
	log.Debugf("Using data directory: %s.\n", bot.dataDir)
}

// ActivateLogger check the config and setups the logger
func ActivateLogger() {
	err := bot.config.CheckLoggerConfig()
	if err != nil {
		log.Errorf("Failed to configure logger reason: %s", err)
	}

	err = log.SetupLogger()
	if err != nil {
		log.Errorf("Failed to setup logger reason: %s", err)
	}
}

// PrintBotSummary prints any summary information out to stdout
func PrintBotSummary() {
	log.Debugf("Bot '%s' started.\n", bot.config.Name)
	log.Debugf("Bot dry run mode: %v.\n", common.IsEnabled(bot.dryRun))

	log.Debugf("Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(bot.config.Exchanges),
		bot.config.CountEnabledExchanges())
}

func ActivatePortfolioWatcher() {
	bot.portfolio = &portfolio.Portfolio
	bot.portfolio.SeedPortfolio(bot.config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)
	go portfolio.StartPortfolioWatcher()
}

// ActivateHTTPClient sets up a global HTTPClient
func ActivateHTTPClient() {
	common.HTTPClient = common.NewHTTPClientWithTimeout(bot.config.GlobalHTTPTimeout)
	log.Debugf("Global HTTP request timeout: %v.\n", common.HTTPClient.Timeout)
}

// ActivateCommunicationMediums
func ActivateCommunicationMediums() {
	log.Debugf("Starting communication mediums..")
	cfg := bot.config.GetCommunicationsConfig()
	bot.comms = communications.NewComm(&cfg)
	bot.comms.GetEnabledCommunicationMediums()
}

//
func ActivateStorageUpdater() {
	var newFxSettings []currency.FXSettings
	for _, d := range bot.config.Currency.ForexProviders {
		newFxSettings = append(newFxSettings, currency.FXSettings(d))
	}

	err := currency.RunStorageUpdater(currency.BotOverrides{
		Coinmarketcap:       bot.coinMarketCap,
		FxCurrencyConverter: bot.fxCurrencyConverter,
		FxCurrencyLayer:     bot.fxCurrencyLayer,
		FxFixer:             bot.fxFixer,
		FxOpenExchangeRates: bot.fxOpenExchangeRates,
	},
		&currency.MainConfiguration{
			ForexProviders:         newFxSettings,
			CryptocurrencyProvider: coinmarketcap.Settings(bot.config.Currency.CryptocurrencyProvider),
			Cryptocurrencies:       bot.config.Currency.Cryptocurrencies,
			FiatDisplayCurrency:    bot.config.Currency.FiatDisplayCurrency,
			CurrencyDelay:          bot.config.Currency.CurrencyFileUpdateDuration,
			FxRateDelay:            bot.config.Currency.ForeignExchangeUpdateDuration,
		},
		bot.dataDir,
		bot.verbosity)
	if err != nil {
		log.Fatalf("currency updater system failed to start %v", err)
	}
}
