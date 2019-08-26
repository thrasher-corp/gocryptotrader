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

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/currency/coinmarketcap"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/ntpclient"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	config       *config.Config
	portfolio    *portfolio.Base
	exchanges    []exchange.IBotExchange
	comms        *communications.Communications
	shutdown     chan bool
	dryRun       bool
	configFile   string
	dataDir      string
	connectivity *connchecker.Checker
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
	bot.shutdown = make(chan bool)
	HandleInterrupt()

	defaultPath, err := config.GetFilePath("")
	if err != nil {
		log.Fatal(err)
	}

	// Handle flags
	flag.StringVar(&bot.configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&bot.dataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	dryrun := flag.Bool("dryrun", false, "dry runs bot, doesn't save config file")
	version := flag.Bool("version", false, "retrieves current GoCryptoTrader version")
	verbosity := flag.Bool("verbose", false, "increases logging verbosity for GoCryptoTrader")

	Coinmarketcap := flag.Bool("c", false, "overrides config and runs currency analaysis")
	FxCurrencyConverter := flag.Bool("fxa", false, "overrides config and sets up foreign exchange Currency Converter")
	FxCurrencyLayer := flag.Bool("fxb", false, "overrides config and sets up foreign exchange Currency Layer")
	FxFixer := flag.Bool("fxc", false, "overrides config and sets up foreign exchange Fixer.io")
	FxOpenExchangeRates := flag.Bool("fxd", false, "overrides config and sets up foreign exchange Open Exchange Rates")

	flag.Parse()

	if *version {
		fmt.Print(BuildVersion(true))
		os.Exit(0)
	}

	if *dryrun {
		bot.dryRun = true
	}

	fmt.Println(banner)
	fmt.Println(BuildVersion(false))

	bot.config = &config.Cfg
	log.Debugf("Loading config file %s..\n", bot.configFile)
	err = bot.config.LoadConfig(bot.configFile)
	if err != nil {
		log.Fatalf("Failed to load config. Err: %s", err)
	}

	err = common.CreateDir(bot.dataDir)
	if err != nil {
		log.Fatalf("Failed to open/create data directory: %s. Err: %s", bot.dataDir, err)
	}
	log.Debugf("Using data directory: %s.\n", bot.dataDir)

	err = bot.config.CheckLoggerConfig()
	if err != nil {
		log.Errorf("Failed to configure logger reason: %s", err)
	}

	err = log.SetupLogger()
	if err != nil {
		log.Errorf("Failed to setup logger reason: %s", err)
	}

	ActivateNTP()
	ActivateConnectivityMonitor()
	AdjustGoMaxProcs()

	log.Debugf("Bot '%s' started.\n", bot.config.Name)
	log.Debugf("Bot dry run mode: %v.\n", common.IsEnabled(bot.dryRun))

	log.Debugf("Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(bot.config.Exchanges),
		bot.config.CountEnabledExchanges())

	common.HTTPClient = common.NewHTTPClientWithTimeout(bot.config.GlobalHTTPTimeout)
	log.Debugf("Global HTTP request timeout: %v.\n", common.HTTPClient.Timeout)

	SetupExchanges()
	if len(bot.exchanges) == 0 {
		log.Fatal("No exchanges were able to be loaded. Exiting")
	}

	log.Debugf("Starting communication mediums..")
	cfg := bot.config.GetCommunicationsConfig()
	bot.comms = communications.NewComm(&cfg)
	bot.comms.GetEnabledCommunicationMediums()

	var newFxSettings []currency.FXSettings
	for _, d := range bot.config.Currency.ForexProviders {
		newFxSettings = append(newFxSettings, currency.FXSettings(d))
	}

	err = currency.RunStorageUpdater(currency.BotOverrides{
		Coinmarketcap:       *Coinmarketcap,
		FxCurrencyConverter: *FxCurrencyConverter,
		FxCurrencyLayer:     *FxCurrencyLayer,
		FxFixer:             *FxFixer,
		FxOpenExchangeRates: *FxOpenExchangeRates,
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
		*verbosity)
	if err != nil {
		log.Fatalf("currency updater system failed to start %v", err)

	}

	bot.portfolio = &portfolio.Portfolio
	bot.portfolio.SeedPortfolio(bot.config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)

	ActivateWebServer()

	go portfolio.StartPortfolioWatcher()

	go TickerUpdaterRoutine()
	go OrderbookUpdaterRoutine()
	go WebsocketRoutine(*verbosity)

	<-bot.shutdown
	Shutdown()
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
				if *bot.config.Logging.Enabled && bot.config.NTPClient.Level == 0 {
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
