package engine

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/coinmarketcap"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/utils"
)

// Engine contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Engine struct {
	Config                         *config.Config
	Portfolio                      *portfolio.Base
	Exchanges                      []exchange.IBotExchange
	ExchangeCurrencyPairManager    *ExchangeCurrencyPairSyncer
	NTPManager                     ntpManager
	ConnectionManager              connectionManager
	OrderManager                   orderManager
	PortfolioManager               portfolioManager
	CommsManager                   commsManager
	Shutdown                       chan struct{}
	Settings                       Settings
	CryptocurrencyDepositAddresses map[string]map[string]string
	Uptime                         time.Time
	ServicesWG                     sync.WaitGroup
}

// Vars for engine
var (
	Bot *Engine
)

func init() {
	if Bot == nil {
		return
	}
}

// New starts a new engine
func New() (*Engine, error) {
	var b Engine
	b.Config = &config.Cfg

	err := b.Config.LoadConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	b.CryptocurrencyDepositAddresses = make(map[string]map[string]string)

	return &b, nil
}

// NewFromSettings starts a new engine based on supplied settings
func NewFromSettings(settings *Settings) (*Engine, error) {
	if settings == nil {
		return nil, errors.New("engine: settings is nil")
	}

	var b Engine
	b.Config = &config.Cfg
	log.Debugf(log.LogGlobal, "Loading config file %s..\n", settings.ConfigFile)
	err := b.Config.LoadConfig(settings.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	err = common.CreateDir(settings.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open/create data directory: %s. Err: %s", settings.DataDir, err)
	}

	if *b.Config.Logging.Enabled {
		log.SetupGlobalLogger()
		log.SetupSubLogger(b.Config.Logging.SubLoggers)
	}

	b.Settings.ConfigFile = settings.ConfigFile
	b.Settings.DataDir = settings.DataDir

	b.CryptocurrencyDepositAddresses = make(map[string]map[string]string)

	err = utils.AdjustGoMaxProcs(settings.GoMaxProcs)
	if err != nil {
		return nil, fmt.Errorf("unable to adjust runtime GOMAXPROCS value. Err: %s", err)
	}

	b.handleInterrupt()

	ValidateSettings(&b, settings)

	return &b, nil
}

// ValidateSettings validates and sets all bot settings
func ValidateSettings(b *Engine, s *Settings) {
	b.Settings.Verbose = s.Verbose
	b.Settings.EnableDryRun = s.EnableDryRun
	b.Settings.EnableAllExchanges = s.EnableAllExchanges
	b.Settings.EnableAllPairs = s.EnableAllPairs
	b.Settings.EnablePortfolioManager = s.EnablePortfolioManager
	b.Settings.EnableCoinmarketcapAnalysis = s.EnableCoinmarketcapAnalysis

	// TO-DO: FIXME
	if flag.Lookup("grpc") != nil {
		b.Settings.EnableGRPC = s.EnableGRPC
	} else {
		b.Settings.EnableGRPC = b.Config.RemoteControl.GRPC.Enabled
	}

	if flag.Lookup("grpcproxy") != nil {
		b.Settings.EnableGRPCProxy = s.EnableGRPCProxy
	} else {
		b.Settings.EnableGRPCProxy = b.Config.RemoteControl.GRPC.GRPCProxyEnabled
	}

	if flag.Lookup("websocketrpc") != nil {
		b.Settings.EnableWebsocketRPC = s.EnableWebsocketRPC
	} else {
		b.Settings.EnableWebsocketRPC = b.Config.RemoteControl.WebsocketRPC.Enabled
	}

	if flag.Lookup("deprecatedrpc") != nil {
		b.Settings.EnableDeprecatedRPC = s.EnableDeprecatedRPC
	} else {
		b.Settings.EnableDeprecatedRPC = b.Config.RemoteControl.DeprecatedRPC.Enabled
	}

	b.Settings.EnableCommsRelayer = s.EnableCommsRelayer
	b.Settings.EnableEventManager = s.EnableEventManager

	if b.Settings.EnableEventManager {
		if b.Settings.EventManagerDelay != time.Duration(0) && s.EventManagerDelay > 0 {
			b.Settings.EventManagerDelay = s.EventManagerDelay
		} else {
			b.Settings.EventManagerDelay = EventSleepDelay
		}
	}

	b.Settings.EnableConnectivityMonitor = s.EnableConnectivityMonitor
	b.Settings.EnableNTPClient = s.EnableNTPClient
	b.Settings.EnableOrderManager = s.EnableOrderManager
	b.Settings.EnableExchangeSyncManager = s.EnableExchangeSyncManager
	b.Settings.EnableTickerSyncing = s.EnableTickerSyncing
	b.Settings.EnableOrderbookSyncing = s.EnableOrderbookSyncing
	b.Settings.EnableExchangeAutoPairUpdates = s.EnableExchangeAutoPairUpdates
	b.Settings.EnableExchangeWebsocketSupport = s.EnableExchangeWebsocketSupport
	b.Settings.EnableExchangeRESTSupport = s.EnableExchangeRESTSupport
	b.Settings.EnableExchangeVerbose = s.EnableExchangeVerbose
	b.Settings.EnableExchangeHTTPRateLimiter = s.EnableExchangeHTTPDebugging
	b.Settings.EnableExchangeHTTPDebugging = s.EnableExchangeHTTPDebugging
	b.Settings.DisableExchangeAutoPairUpdates = s.DisableExchangeAutoPairUpdates
	b.Settings.ExchangePurgeCredentials = s.ExchangePurgeCredentials

	if !b.Settings.EnableExchangeHTTPRateLimiter {
		request.DisableRateLimiter = true
	}

	// Checks if the flag values are different from the defaults
	b.Settings.MaxHTTPRequestJobsLimit = s.MaxHTTPRequestJobsLimit
	if b.Settings.MaxHTTPRequestJobsLimit != request.DefaultMaxRequestJobs && s.MaxHTTPRequestJobsLimit > 0 {
		request.MaxRequestJobs = b.Settings.MaxHTTPRequestJobsLimit
	}

	b.Settings.RequestTimeoutRetryAttempts = s.RequestTimeoutRetryAttempts
	if b.Settings.RequestTimeoutRetryAttempts != request.DefaultTimeoutRetryAttempts && s.RequestTimeoutRetryAttempts > 0 {
		request.TimeoutRetryAttempts = b.Settings.RequestTimeoutRetryAttempts
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
func PrintSettings(s *Settings) {

	log.Debugln(log.LogGlobal, "ENGINE SETTINGS")
	log.Debugln(log.LogGlobal, "- CORE SETTINGS:")
	log.Debugf(log.LogGlobal, "\t Verbose mode: %v\n", s.Verbose)
	log.Debugf(log.LogGlobal, "\t Enable dry run mode: %v\n", s.EnableDryRun)
	log.Debugf(log.LogGlobal, "\t Enable all exchanges: %v\n", s.EnableAllExchanges)
	log.Debugf(log.LogGlobal, "\t Enable all pairs: %v\n", s.EnableAllPairs)
	log.Debugf(log.LogGlobal, "\t Enable coinmarketcap analaysis: %v\n", s.EnableCoinmarketcapAnalysis)
	log.Debugf(log.LogGlobal, "\t Enable portfolio manager: %v\n", s.EnablePortfolioManager)
	log.Debugf(log.LogGlobal, "\t Enable gPRC: %v\n", s.EnableGRPC)
	log.Debugf(log.LogGlobal, "\t Enable gRPC Proxy: %v\n", s.EnableGRPCProxy)
	log.Debugf(log.LogGlobal, "\t Enable websocket RPC: %v\n", s.EnableWebsocketRPC)
	log.Debugf(log.LogGlobal, "\t Enable deprecated RPC: %v\n", s.EnableDeprecatedRPC)
	log.Debugf(log.LogGlobal, "\t Enable comms relayer: %v\n", s.EnableCommsRelayer)
	log.Debugf(log.LogGlobal, "\t Enable event manager: %v\n", s.EnableEventManager)
	log.Debugf(log.LogGlobal, "\t Event manager sleep delay: %v\n", s.EventManagerDelay)
	log.Debugf(log.LogGlobal, "\t Enable order manager: %v\n", s.EnableOrderManager)
	log.Debugf(log.LogGlobal, "\t Enable exchange sync manager: %v\n", s.EnableExchangeSyncManager)
	log.Debugf(log.LogGlobal, "\t Enable ticker syncing: %v\n", s.EnableTickerSyncing)
	log.Debugf(log.LogGlobal, "\t Enable orderbook syncing: %v\n", s.EnableOrderbookSyncing)
	log.Debugf(log.LogGlobal, "\t Enable websocket routine: %v\n", s.EnableWebsocketRoutine)
	log.Debugf(log.LogGlobal, "\t Enable NTP client: %v\n", s.EnableNTPClient)
	log.Debugln(log.LogGlobal, "- FOREX SETTINGS:")
	log.Debugf(log.LogGlobal, "\t Enable currency conveter: %v\n", s.EnableCurrencyConverter)
	log.Debugf(log.LogGlobal, "\t Enable currency layer: %v\n", s.EnableCurrencyLayer)
	log.Debugf(log.LogGlobal, "\t Enable fixer: %v\n", s.EnableFixer)
	log.Debugf(log.LogGlobal, "\t Enable OpenExchangeRates: %v\n", s.EnableOpenExchangeRates)
	log.Debugln(log.LogGlobal, "- EXCHANGE SETTINGS:")
	log.Debugf(log.LogGlobal, "\t Enable exchange auto pair updates: %v\n", s.EnableExchangeAutoPairUpdates)
	log.Debugf(log.LogGlobal, "\t Disable all exchange auto pair updates: %v\n", s.DisableExchangeAutoPairUpdates)
	log.Debugf(log.LogGlobal, "\t Enable exchange websocket support: %v\n", s.EnableExchangeWebsocketSupport)
	log.Debugf(log.LogGlobal, "\t Enable exchange verbose mode: %v\n", s.EnableExchangeVerbose)
	log.Debugf(log.LogGlobal, "\t Enable exchange HTTP rate limiter: %v\n", s.EnableExchangeHTTPRateLimiter)
	log.Debugf(log.LogGlobal, "\t Enable exchange HTTP debugging: %v\n", s.EnableExchangeHTTPDebugging)
	log.Debugf(log.LogGlobal, "\t Exchange max HTTP request jobs: %v\n", s.MaxHTTPRequestJobsLimit)
	log.Debugf(log.LogGlobal, "\t Exchange HTTP request timeout retry amount: %v\n", s.RequestTimeoutRetryAttempts)
	log.Debugf(log.LogGlobal, "\t Exchange HTTP timeout: %v\n", s.ExchangeHTTPTimeout)
	log.Debugf(log.LogGlobal, "\t Exchange HTTP user agent: %v\n", s.ExchangeHTTPUserAgent)
	log.Debugf(log.LogGlobal, "\t Exchange HTTP proxy: %v\n", s.ExchangeHTTPProxy)
	log.Debugln(log.LogGlobal, "- COMMON SETTINGS:")
	log.Debugf(log.LogGlobal, "\t Global HTTP timeout: %v\n", s.GlobalHTTPTimeout)
	log.Debugf(log.LogGlobal, "\t Global HTTP user agent: %v\n", s.GlobalHTTPUserAgent)
	log.Debugf(log.LogGlobal, "\t Global HTTP proxy: %v\n", s.ExchangeHTTPProxy)
}

// Start starts the engine
func (e *Engine) Start() {
	if e == nil {
		log.Errorln(log.LogGlobal, "Engine instance is nil")
		os.Exit(1)
	}

	// Sets up internet connectivity monitor
	if e.Settings.EnableConnectivityMonitor {
		if err := e.ConnectionManager.Start(); err != nil {
			log.Errorf(log.LogGlobal, "Connection manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableNTPClient {
		if err := e.NTPManager.Start(); err != nil {
			log.Errorf(log.LogGlobal, "NTP manager unable to start: %v", err)
		}
	}

	e.Uptime = time.Now()
	log.Debugf(log.LogGlobal, "Bot '%s' started.\n", e.Config.Name)

	enabledExchanges := e.Config.CountEnabledExchanges()
	if e.Settings.EnableAllExchanges {
		enabledExchanges = len(e.Config.Exchanges)
	}

	log.Debugln(log.LogGlobal, "EXCHANGE COVERAGE")
	log.Debugf(log.LogGlobal, "\t Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(e.Config.Exchanges), enabledExchanges)

	if e.Settings.ExchangePurgeCredentials {
		log.Debugln(log.LogGlobal, "Purging exchange API credentials.")
		e.Config.PurgeExchangeAPICredentials()
	}

	log.Debugln(log.LogGlobal, "Setting up exchanges..")
	SetupExchanges()
	if len(e.Exchanges) == 0 {
		log.Errorln(log.LogGlobal, "No exchanges were able to be loaded. Exiting")
		os.Exit(1)
	}

	if e.Settings.EnableCommsRelayer {
		if err := e.CommsManager.Start(); err != nil {
			log.Errorf(log.LogGlobal, "Communications manager unable to start: %v", err)
		}
	}

	var newFxSettings []currency.FXSettings
	for _, d := range e.Config.Currency.ForexProviders {
		newFxSettings = append(newFxSettings, currency.FXSettings(d))
	}

	err := currency.RunStorageUpdater(currency.BotOverrides{
		Coinmarketcap:       e.Settings.EnableCoinmarketcapAnalysis,
		FxCurrencyConverter: e.Settings.EnableCurrencyConverter,
		FxCurrencyLayer:     e.Settings.EnableCurrencyLayer,
		FxFixer:             e.Settings.EnableFixer,
		FxOpenExchangeRates: e.Settings.EnableOpenExchangeRates,
	},
		&currency.MainConfiguration{
			ForexProviders:         newFxSettings,
			CryptocurrencyProvider: coinmarketcap.Settings(e.Config.Currency.CryptocurrencyProvider),
			Cryptocurrencies:       e.Config.Currency.Cryptocurrencies,
			FiatDisplayCurrency:    e.Config.Currency.FiatDisplayCurrency,
			CurrencyDelay:          e.Config.Currency.CurrencyFileUpdateDuration,
			FxRateDelay:            e.Config.Currency.ForeignExchangeUpdateDuration,
		},
		e.Settings.DataDir,
		e.Settings.Verbose)
	if err != nil {
		log.Errorf(log.LogGlobal, "currency updater system failed to start %v", err)
	}

	e.CryptocurrencyDepositAddresses = GetExchangeCryptocurrencyDepositAddresses()

	if e.Settings.EnableGRPC {
		go StartRPCServer()
	}

	if e.Settings.EnableDeprecatedRPC {
		go StartRESTServer()
	}

	if e.Settings.EnableWebsocketRPC {
		go StartWebsocketServer()
		StartWebsocketHandler()
	}

	if e.Settings.EnablePortfolioManager {
		if err = e.PortfolioManager.Start(); err != nil {
			log.Errorf(log.LogGlobal, "Fund manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableOrderManager {
		if err = e.OrderManager.Start(); err != nil {
			log.Errorf(log.LogGlobal, "Order manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableExchangeSyncManager {
		exchangeSyncCfg := CurrencyPairSyncerConfig{
			SyncTicker:       e.Settings.EnableTickerSyncing,
			SyncOrderbook:    e.Settings.EnableOrderbookSyncing,
			SyncContinuously: true,
			NumWorkers:       15,
		}

		e.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(exchangeSyncCfg)
		if err != nil {
			log.Warnf(log.LogGlobal, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			go e.ExchangeCurrencyPairManager.Start()
		}
	}

	if e.Settings.EnableEventManager {
		go EventManger()
	}

	<-e.Shutdown
	e.Stop()
}

// Stop correctly shuts down engine saving configuration files
func (e *Engine) Stop() {
	log.Debugln(log.LogGlobal, "Engine shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		e.Config.Portfolio = portfolio.Portfolio
	}

	if e.OrderManager.Started() {
		if err := e.OrderManager.Stop(); err != nil {
			log.Errorf(log.LogGlobal, "Order manager unable to stop. Error: %v", err)
		}
	}

	if e.NTPManager.Started() {
		if err := e.NTPManager.Stop(); err != nil {
			log.Errorf(log.LogGlobal, "NTP manager unable to stop. Error: %v", err)
		}
	}

	if e.CommsManager.Started() {
		if err := e.CommsManager.Stop(); err != nil {
			log.Errorf(log.LogGlobal, "Communication manager unable to stop. Error: %v", err)
		}
	}

	if e.PortfolioManager.Started() {
		if err := e.PortfolioManager.Stop(); err != nil {
			log.Errorf(log.LogGlobal, "Fund manager unable to stop. Error: %v", err)
		}
	}

	if e.ConnectionManager.Started() {
		if err := e.ConnectionManager.Stop(); err != nil {
			log.Errorf(log.LogGlobal, "Connection manager unable to stop. Error: %v", err)
		}
	}

	if !e.Settings.EnableDryRun {
		err := e.Config.SaveConfig(e.Settings.ConfigFile)
		if err != nil {
			log.Errorln(log.LogGlobal, "Unable to save config.")
		} else {
			log.Debugln(log.LogGlobal, "Config file saved successfully.")
		}
	}
	// Wait for services to gracefully shutdown
	e.ServicesWG.Wait()
	log.Debugln(log.LogGlobal, "Exiting.")
	log.CloseLogger()
	os.Exit(0)
}

// handleInterrupt monitors and captures the SIGTERM in a new goroutine then
// shuts down the engine instance
func (e *Engine) handleInterrupt() {
	c := make(chan os.Signal, 1)
	e.Shutdown = make(chan struct{})
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Debugf(log.LogGlobal, "Captured %v, shutdown requested.\n", sig)
		close(e.Shutdown)
	}()
}
