package engine

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/utils"
)

// Engine contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Engine struct {
	Config       *config.Config
	Portfolio    *portfolio.Base
	Exchanges    []exchange.IBotExchange
	CommsRelayer *communications.Communications
	Shutdown     chan bool
	Settings     Settings
}

// Vars for engine
var (
	Bot *Engine
)

// NewFromSettings starts a new engine based on supplied settings
func NewFromSettings(settings *Settings) (*Engine, error) {
	if settings == nil {
		return nil, errors.New("engine: settings is nil")
	}

	var b Engine
	b.Config = &config.Cfg
	log.Printf("Loading config file %s..\n", settings.ConfigFile)
	err := b.Config.LoadConfig(settings.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	err = common.CheckDir(settings.DataDir, true)
	if err != nil {
		return nil, fmt.Errorf("failed to open/create data directory: %s. Err: %s", settings.DataDir, err)
	}

	b.Settings.ConfigFile = settings.ConfigFile
	b.Settings.DataDir = settings.DataDir
	b.Settings.LogFile = utils.GetLogFile(settings.DataDir)

	err = utils.AdjustGoMaxProcs(settings.GoMaxProcs)
	if err != nil {
		return nil, fmt.Errorf("unable to adjust runtime GOMAXPROCS value. Err: %s", err)
	}

	err = utils.InitLogFile(b.Settings.LogFile)
	if err != nil {
		log.Printf("failed to create log file writer. Err: %s", err)
	} else {
		log.Printf("Using log file: %s.\n", b.Settings.LogFile)
	}

	b.handleInterrupt()

	ValidateSettings(&b, settings)

	return &b, nil
}

// ValidateSettings validates and sets all bot settings
func ValidateSettings(b *Engine, s *Settings) {
	b.Settings.EnableDryRun = s.EnableDryRun
	b.Settings.EnableAllExchanges = s.EnableAllExchanges
	b.Settings.EnableAllPairs = s.EnableAllPairs
	b.Settings.EnablePortfolioWatcher = s.EnablePortfolioWatcher

	// TO-DO: FIXME
	if flag.Lookup("websocketserver") != nil {
		b.Settings.EnableWebsocketServer = s.EnableWebsocketServer
	} else {
		b.Settings.EnableWebsocketServer = b.Config.WebsocketServer.Enabled
	}

	if flag.Lookup("restserver") != nil {
		b.Settings.EnableRESTServer = s.EnableRESTServer
	} else {
		b.Settings.EnableRESTServer = b.Config.RESTServer.Enabled
	}

	b.Settings.EnableCommsRelayer = s.EnableCommsRelayer
	b.Settings.Verbose = s.Verbose
	b.Settings.EnableTickerRoutine = s.EnableTickerRoutine
	b.Settings.EnableOrderbookRoutine = s.EnableOrderbookRoutine
	b.Settings.EnableWebsocketRoutine = s.EnableWebsocketRoutine
	b.Settings.EnableExchangeAutoPairUpdates = s.EnableExchangeAutoPairUpdates
	b.Settings.EnableExchangeWebsocketSupport = s.EnableExchangeWebsocketSupport
	b.Settings.EnableExchangeRESTSupport = s.EnableExchangeRESTSupport
	b.Settings.EnableExchangeVerbose = s.EnableExchangeVerbose
	b.Settings.EnableHTTPRateLimiter = s.EnableHTTPRateLimiter

	if !b.Settings.EnableHTTPRateLimiter {
		request.DisableRateLimiter = true
	}

	// Checks if the flag values are different from the defaults
	b.Settings.MaxHTTPRequestJobsLimit = s.MaxHTTPRequestJobsLimit
	if b.Settings.MaxHTTPRequestJobsLimit != request.DefaultMaxRequestJobs && s.MaxHTTPRequestJobsLimit > 0 {
		request.MaxRequestJobs = b.Settings.MaxHTTPRequestJobsLimit
	}

	b.Settings.ExchangeHTTPTimeout = s.ExchangeHTTPTimeout
	if s.ExchangeHTTPTimeout != time.Duration(0) && s.ExchangeHTTPTimeout > 0 {
		b.Settings.ExchangeHTTPTimeout = s.ExchangeHTTPTimeout
	} else {
		b.Settings.ExchangeHTTPTimeout = b.Config.GlobalHTTPTimeout
	}

	b.Settings.ExchangeHTTPUserAgent = s.ExchangeHTTPUserAgent
	b.Settings.ExchangeHTTPProxy = s.ExchangeHTTPProxy

	if s.GlobalHTTPTimeout != time.Duration(0) && s.GlobalHTTPTimeout > 0 {
		b.Settings.GlobalHTTPTimeout = s.GlobalHTTPTimeout
	} else {
		b.Settings.GlobalHTTPTimeout = b.Config.GlobalHTTPTimeout
	}
	common.HTTPClient = common.NewHTTPClientWithTimeout(b.Settings.GlobalHTTPTimeout)

	b.Settings.GlobalHTTPUserAgent = s.GlobalHTTPUserAgent
	if b.Settings.GlobalHTTPUserAgent != "" {
		common.HTTPUserAgent = b.Settings.GlobalHTTPUserAgent
	}

	b.Settings.GlobalHTTPProxy = s.GlobalHTTPProxy
}

// PrintSettings returns the engine settings
func PrintSettings(s Settings) {
	log.Println()
	log.Println("ENGINE SETTINGS")
	log.Printf("- CORE SETTINGS:")
	log.Printf("\t Verbose mode: %v", s.Verbose)
	log.Printf("\t Enable dry run mode: %v", s.EnableDryRun)
	log.Printf("\t Enable all exchanges: %v", s.EnableAllExchanges)
	log.Printf("\t Enable all pairs: %v", s.EnableAllPairs)
	log.Printf("\t Enable portfolio watcher: %v", s.EnablePortfolioWatcher)
	log.Printf("\t Enable websocket server: %v", s.EnableWebsocketServer)
	log.Printf("\t Enable REST server: %v", s.EnableRESTServer)
	log.Printf("\t Enable comms relayer: %v", s.EnableCommsRelayer)
	log.Printf("\t Enable ticker routine: %v", s.EnableTickerRoutine)
	log.Printf("\t Enable orderbook routine: %v", s.EnableOrderbookRoutine)
	log.Printf("\t Enable websocket routine: %v\n", s.EnableWebsocketRoutine)
	log.Printf("- EXCHANGE SETTINGS:")
	log.Printf("\t Enable exchange auto pair updates: %v", s.EnableExchangeAutoPairUpdates)
	log.Printf("\t Enable exchange websocket support: %v", s.EnableExchangeWebsocketSupport)
	log.Printf("\t Enable exchange verbose mode: %v", s.EnableExchangeVerbose)
	log.Printf("\t Enable exchange HTTP rate limiter: %v", s.EnableHTTPRateLimiter)
	log.Printf("\t Exchange max HTTP request jobs: %v", s.MaxHTTPRequestJobsLimit)
	log.Printf("\t Exchange HTTP timeout: %v", s.ExchangeHTTPTimeout)
	log.Printf("\t Exchange HTTP user agent: %v", s.ExchangeHTTPUserAgent)
	log.Printf("\t Exchange HTTP proxy: %v\n", s.ExchangeHTTPProxy)
	log.Printf("- COMMON SETTINGS:")
	log.Printf("\t Global HTTP timeout: %v", s.GlobalHTTPTimeout)
	log.Printf("\t Global HTTP user agent: %v", s.GlobalHTTPUserAgent)
	log.Printf("\t Global HTTP proxy: %v", s.ExchangeHTTPProxy)
	log.Println()
}

// Start starts the engine
func (e *Engine) Start() {
	if e == nil {
		log.Fatal("Engine instance is nil")
	}

	log.Printf("Bot '%s' started.\n", e.Config.Name)

	enabledExchanges := e.Config.CountEnabledExchanges()
	if e.Settings.EnableAllExchanges {
		enabledExchanges = len(e.Config.Exchanges)
	}

	log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(e.Config.Exchanges), enabledExchanges)

	SetupExchanges()
	if len(e.Exchanges) == 0 {
		log.Fatalf("No exchanges were able to be loaded. Exiting")
	}

	if e.Settings.EnableCommsRelayer {
		log.Println("Starting communication mediums..")
		e.CommsRelayer = communications.NewComm(e.Config.GetCommunicationsConfig())
		e.CommsRelayer.GetEnabledCommunicationMediums()
	}

	log.Printf("Fiat display currency: %s.", e.Config.Currency.FiatDisplayCurrency)
	currency.BaseCurrency = e.Config.Currency.FiatDisplayCurrency
	currency.FXProviders = forexprovider.StartFXService(e.Config.GetCurrencyConfig().ForexProviders)
	log.Printf("Primary forex conversion provider: %s.\n", e.Config.GetPrimaryForexProvider())
	err := e.Config.RetrieveConfigCurrencyPairs(true)
	if err != nil {
		log.Fatalf("Failed to retrieve config currency pairs. Error: %s", err)
	}
	log.Println("Successfully retrieved config currencies.")
	log.Println("Fetching currency data from forex provider..")
	err = currency.Seed(common.JoinStrings(currency.FiatCurrencies, ","))
	if err != nil {
		log.Fatalf("Unable to fetch forex data. Error: %s", err)
	}

	e.Portfolio = &portfolio.Portfolio
	e.Portfolio.Seed(e.Config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)

	if e.Settings.EnableRESTServer {
		go StartRESTServer()
	}

	if e.Settings.EnableWebsocketServer {
		go StartWebsocketServer()
		StartWebsocketHandler()
	}

	if e.Settings.EnablePortfolioWatcher {
		go portfolio.StartPortfolioWatcher()
	}

	if e.Settings.EnableTickerRoutine {
		go TickerUpdaterRoutine()
	}

	if e.Settings.EnableOrderbookRoutine {
		go OrderbookUpdaterRoutine()
	}

	if e.Settings.EnableWebsocketRoutine {
		go WebsocketRoutine()
	}

	<-e.Shutdown
	e.Stop()
}

// Stop correctly shuts down engine saving configuration files
func (e *Engine) Stop() {
	log.Println("Engine shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		e.Config.Portfolio = portfolio.Portfolio
	}

	if !e.Settings.EnableDryRun {
		err := e.Config.SaveConfig(e.Settings.ConfigFile)

		if err != nil {
			log.Println("Unable to save config.")
		} else {
			log.Println("Config file saved successfully.")
		}
	}

	log.Println("Exiting.")

	if utils.LogFileHandle != nil {
		utils.LogFileHandle.Close()
	}
	os.Exit(0)
}

// handleInterrupt monitors and captures the SIGTERM in a new goroutine then
// shuts down the engine instance
func (e *Engine) handleInterrupt() {
	c := make(chan os.Signal, 1)
	e.Shutdown = make(chan bool)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Captured %v, shutdown requested.", sig)
		e.Shutdown <- true
	}()
}
