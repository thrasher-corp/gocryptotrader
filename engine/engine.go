package engine

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/utils"
)

// Engine contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Engine struct {
	Config                      *config.Config
	Portfolio                   *portfolio.Base
	Exchanges                   []exchange.IBotExchange
	ExchangeCurrencyPairManager *ExchangeCurrencyPairSyncer
	NTPManager                  ntpManager
	ConnectionManager           connectionManager
	DatabaseManager             databaseManager
	GctScriptManager            gctScriptManager
	OrderManager                orderManager
	PortfolioManager            portfolioManager
	CommsManager                commsManager
	DepositAddressManager       *DepositAddressManager
	Settings                    Settings
	Uptime                      time.Time
	ServicesWG                  sync.WaitGroup
}

// Vars for engine
var (
	Bot *Engine

	// Stores the set flags
	flagSet = make(map[string]bool)
)

// New starts a new engine
func New() (*Engine, error) {
	var b Engine
	b.Config = &config.Cfg

	err := b.Config.LoadConfig("", false)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	return &b, nil
}

// NewFromSettings starts a new engine based on supplied settings
func NewFromSettings(settings *Settings) (*Engine, error) {
	if settings == nil {
		return nil, errors.New("engine: settings is nil")
	}

	var b Engine
	b.Config = &config.Cfg
	filePath, err := config.GetFilePath(settings.ConfigFile)
	if err != nil {
		return nil, err
	}

	log.Debugf(log.Global, "Loading config file %s..\n", filePath)
	err = b.Config.LoadConfig(filePath, settings.EnableDryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	err = common.CreateDir(settings.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open/create data directory: %s. Err: %s", settings.DataDir, err)
	}

	if *b.Config.Logging.Enabled {
		log.SetupGlobalLogger()
		log.SetupSubLoggers(b.Config.Logging.SubLoggers)
	}

	b.Settings.ConfigFile = filePath
	b.Settings.DataDir = settings.DataDir
	b.Settings.CheckParamInteraction = settings.CheckParamInteraction

	err = utils.AdjustGoMaxProcs(settings.GoMaxProcs)
	if err != nil {
		return nil, fmt.Errorf("unable to adjust runtime GOMAXPROCS value. Err: %s", err)
	}

	ValidateSettings(&b, settings)
	return &b, nil
}

// ValidateSettings validates and sets all bot settings
func ValidateSettings(b *Engine, s *Settings) {
	flag.Visit(func(f *flag.Flag) { flagSet[f.Name] = true })

	b.Settings.Verbose = s.Verbose
	b.Settings.EnableDryRun = s.EnableDryRun
	b.Settings.EnableAllExchanges = s.EnableAllExchanges
	b.Settings.EnableAllPairs = s.EnableAllPairs
	b.Settings.EnablePortfolioManager = s.EnablePortfolioManager
	b.Settings.EnableCoinmarketcapAnalysis = s.EnableCoinmarketcapAnalysis
	b.Settings.EnableDatabaseManager = s.EnableDatabaseManager
	b.Settings.EnableGCTScriptManager = s.EnableGCTScriptManager
	b.Settings.MaxVirtualMachines = s.MaxVirtualMachines
	b.Settings.EnableDispatcher = s.EnableDispatcher

	if flagSet["grpc"] {
		b.Settings.EnableGRPC = s.EnableGRPC
	} else {
		b.Settings.EnableGRPC = b.Config.RemoteControl.GRPC.Enabled
	}

	if flagSet["grpcproxy"] {
		b.Settings.EnableGRPCProxy = s.EnableGRPCProxy
	} else {
		b.Settings.EnableGRPCProxy = b.Config.RemoteControl.GRPC.GRPCProxyEnabled
	}

	if flagSet["websocketrpc"] {
		b.Settings.EnableWebsocketRPC = s.EnableWebsocketRPC
	} else {
		b.Settings.EnableWebsocketRPC = b.Config.RemoteControl.WebsocketRPC.Enabled
	}

	if flagSet["deprecatedrpc"] {
		b.Settings.EnableDeprecatedRPC = s.EnableDeprecatedRPC
	} else {
		b.Settings.EnableDeprecatedRPC = b.Config.RemoteControl.DeprecatedRPC.Enabled
	}

	if flagSet["gctscriptmanager"] {
		gctscript.GCTScriptConfig.Enabled = s.EnableGCTScriptManager
	}

	if flagSet["maxvirtualmachines"] {
		gctscript.GCTScriptConfig.MaxVirtualMachines = uint8(s.MaxVirtualMachines)
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
	b.Settings.EnableTradeSyncing = s.EnableTradeSyncing
	b.Settings.SyncWorkers = s.SyncWorkers
	b.Settings.SyncTimeout = s.SyncTimeout
	b.Settings.SyncContinuously = s.SyncContinuously
	b.Settings.EnableDepositAddressManager = s.EnableDepositAddressManager
	b.Settings.EnableExchangeAutoPairUpdates = s.EnableExchangeAutoPairUpdates
	b.Settings.EnableExchangeWebsocketSupport = s.EnableExchangeWebsocketSupport
	b.Settings.EnableExchangeRESTSupport = s.EnableExchangeRESTSupport
	b.Settings.EnableExchangeVerbose = s.EnableExchangeVerbose
	b.Settings.EnableExchangeHTTPRateLimiter = s.EnableExchangeHTTPRateLimiter
	b.Settings.EnableExchangeHTTPDebugging = s.EnableExchangeHTTPDebugging
	b.Settings.DisableExchangeAutoPairUpdates = s.DisableExchangeAutoPairUpdates
	b.Settings.ExchangePurgeCredentials = s.ExchangePurgeCredentials
	b.Settings.EnableWebsocketRoutine = s.EnableWebsocketRoutine

	// Checks if the flag values are different from the defaults
	b.Settings.MaxHTTPRequestJobsLimit = s.MaxHTTPRequestJobsLimit
	if b.Settings.MaxHTTPRequestJobsLimit != int(request.DefaultMaxRequestJobs) &&
		s.MaxHTTPRequestJobsLimit > 0 {
		request.MaxRequestJobs = int32(b.Settings.MaxHTTPRequestJobsLimit)
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
	b.Settings.DispatchMaxWorkerAmount = s.DispatchMaxWorkerAmount
	b.Settings.DispatchJobsLimit = s.DispatchJobsLimit
}

// PrintSettings returns the engine settings
func PrintSettings(s *Settings) {
	log.Debugln(log.Global)
	log.Debugf(log.Global, "ENGINE SETTINGS")
	log.Debugf(log.Global, "- CORE SETTINGS:")
	log.Debugf(log.Global, "\t Verbose mode: %v", s.Verbose)
	log.Debugf(log.Global, "\t Enable dry run mode: %v", s.EnableDryRun)
	log.Debugf(log.Global, "\t Enable all exchanges: %v", s.EnableAllExchanges)
	log.Debugf(log.Global, "\t Enable all pairs: %v", s.EnableAllPairs)
	log.Debugf(log.Global, "\t Enable coinmarketcap analaysis: %v", s.EnableCoinmarketcapAnalysis)
	log.Debugf(log.Global, "\t Enable portfolio manager: %v", s.EnablePortfolioManager)
	log.Debugf(log.Global, "\t Enable gPRC: %v", s.EnableGRPC)
	log.Debugf(log.Global, "\t Enable gRPC Proxy: %v", s.EnableGRPCProxy)
	log.Debugf(log.Global, "\t Enable websocket RPC: %v", s.EnableWebsocketRPC)
	log.Debugf(log.Global, "\t Enable deprecated RPC: %v", s.EnableDeprecatedRPC)
	log.Debugf(log.Global, "\t Enable comms relayer: %v", s.EnableCommsRelayer)
	log.Debugf(log.Global, "\t Enable event manager: %v", s.EnableEventManager)
	log.Debugf(log.Global, "\t Event manager sleep delay: %v", s.EventManagerDelay)
	log.Debugf(log.Global, "\t Enable order manager: %v", s.EnableOrderManager)
	log.Debugf(log.Global, "\t Enable exchange sync manager: %v", s.EnableExchangeSyncManager)
	log.Debugf(log.Global, "\t Enable deposit address manager: %v\n", s.EnableDepositAddressManager)
	log.Debugf(log.Global, "\t Enable websocket routine: %v\n", s.EnableWebsocketRoutine)
	log.Debugf(log.Global, "\t Enable NTP client: %v", s.EnableNTPClient)
	log.Debugf(log.Global, "\t Enable Database manager: %v", s.EnableDatabaseManager)
	log.Debugf(log.Global, "\t Enable dispatcher: %v", s.EnableDispatcher)
	log.Debugf(log.Global, "\t Dispatch package max worker amount: %d", s.DispatchMaxWorkerAmount)
	log.Debugf(log.Global, "\t Dispatch package jobs limit: %d", s.DispatchJobsLimit)
	log.Debugf(log.Global, "- EXCHANGE SYNCER SETTINGS:\n")
	log.Debugf(log.Global, "\t Exchange sync continuously: %v\n", s.SyncContinuously)
	log.Debugf(log.Global, "\t Exchange sync workers: %v\n", s.SyncWorkers)
	log.Debugf(log.Global, "\t Enable ticker syncing: %v\n", s.EnableTickerSyncing)
	log.Debugf(log.Global, "\t Enable orderbook syncing: %v\n", s.EnableOrderbookSyncing)
	log.Debugf(log.Global, "\t Enable trade syncing: %v\n", s.EnableTradeSyncing)
	log.Debugf(log.Global, "\t Exchange sync timeout: %v\n", s.SyncTimeout)
	log.Debugf(log.Global, "- FOREX SETTINGS:")
	log.Debugf(log.Global, "\t Enable currency conveter: %v", s.EnableCurrencyConverter)
	log.Debugf(log.Global, "\t Enable currency layer: %v", s.EnableCurrencyLayer)
	log.Debugf(log.Global, "\t Enable fixer: %v", s.EnableFixer)
	log.Debugf(log.Global, "\t Enable OpenExchangeRates: %v", s.EnableOpenExchangeRates)
	log.Debugf(log.Global, "- EXCHANGE SETTINGS:")
	log.Debugf(log.Global, "\t Enable exchange auto pair updates: %v", s.EnableExchangeAutoPairUpdates)
	log.Debugf(log.Global, "\t Disable all exchange auto pair updates: %v", s.DisableExchangeAutoPairUpdates)
	log.Debugf(log.Global, "\t Enable exchange websocket support: %v", s.EnableExchangeWebsocketSupport)
	log.Debugf(log.Global, "\t Enable exchange verbose mode: %v", s.EnableExchangeVerbose)
	log.Debugf(log.Global, "\t Enable exchange HTTP rate limiter: %v", s.EnableExchangeHTTPRateLimiter)
	log.Debugf(log.Global, "\t Enable exchange HTTP debugging: %v", s.EnableExchangeHTTPDebugging)
	log.Debugf(log.Global, "\t Exchange max HTTP request jobs: %v", s.MaxHTTPRequestJobsLimit)
	log.Debugf(log.Global, "\t Exchange HTTP request timeout retry amount: %v", s.RequestTimeoutRetryAttempts)
	log.Debugf(log.Global, "\t Exchange HTTP timeout: %v", s.ExchangeHTTPTimeout)
	log.Debugf(log.Global, "\t Exchange HTTP user agent: %v", s.ExchangeHTTPUserAgent)
	log.Debugf(log.Global, "\t Exchange HTTP proxy: %v\n", s.ExchangeHTTPProxy)
	log.Debugf(log.Global, "- GCTSCRIPT SETTINGS: ")
	log.Debugf(log.Global, "\t Enable GCTScript manager: %v", s.EnableGCTScriptManager)
	log.Debugf(log.Global, "\t GCTScript max virtual machines: %v", s.MaxVirtualMachines)
	log.Debugf(log.Global, "- COMMON SETTINGS:")
	log.Debugf(log.Global, "\t Global HTTP timeout: %v", s.GlobalHTTPTimeout)
	log.Debugf(log.Global, "\t Global HTTP user agent: %v", s.GlobalHTTPUserAgent)
	log.Debugf(log.Global, "\t Global HTTP proxy: %v", s.ExchangeHTTPProxy)

	log.Debugln(log.Global)
}

// Start starts the engine
func (e *Engine) Start() error {
	if e == nil {
		return errors.New("engine instance is nil")
	}

	if e.Settings.EnableDatabaseManager {
		if err := e.DatabaseManager.Start(); err != nil {
			log.Errorf(log.Global, "Database manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableDispatcher {
		if err := dispatch.Start(e.Settings.DispatchMaxWorkerAmount, e.Settings.DispatchJobsLimit); err != nil {
			log.Errorf(log.DispatchMgr, "Dispatcher unable to start: %v", err)
		}
	}

	// Sets up internet connectivity monitor
	if e.Settings.EnableConnectivityMonitor {
		if err := e.ConnectionManager.Start(); err != nil {
			log.Errorf(log.Global, "Connection manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableNTPClient {
		if err := e.NTPManager.Start(); err != nil {
			log.Errorf(log.Global, "NTP manager unable to start: %v", err)
		}
	}

	e.Uptime = time.Now()
	log.Debugf(log.Global, "Bot '%s' started.\n", e.Config.Name)
	log.Debugf(log.Global, "Using data dir: %s\n", e.Settings.DataDir)
	if *e.Config.Logging.Enabled && strings.Contains(e.Config.Logging.Output, "file") {
		log.Debugf(log.Global, "Using log file: %s\n",
			filepath.Join(log.LogPath, e.Config.Logging.LoggerFileConfig.FileName))
	}
	log.Debugf(log.Global,
		"Using %d out of %d logical processors for runtime performance\n",
		runtime.GOMAXPROCS(-1), runtime.NumCPU())

	enabledExchanges := e.Config.CountEnabledExchanges()
	if e.Settings.EnableAllExchanges {
		enabledExchanges = len(e.Config.Exchanges)
	}

	log.Debugln(log.Global, "EXCHANGE COVERAGE")
	log.Debugf(log.Global, "\t Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(e.Config.Exchanges), enabledExchanges)

	if e.Settings.ExchangePurgeCredentials {
		log.Debugln(log.Global, "Purging exchange API credentials.")
		e.Config.PurgeExchangeAPICredentials()
	}

	log.Debugln(log.Global, "Setting up exchanges..")
	SetupExchanges()
	if len(Bot.Exchanges) == 0 {
		return errors.New("no exchanges are loaded")
	}

	if e.Settings.EnableCommsRelayer {
		if err := e.CommsManager.Start(); err != nil {
			log.Errorf(log.Global, "Communications manager unable to start: %v\n", err)
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
		log.Errorf(log.Global, "currency updater system failed to start %v", err)
	}

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
			log.Errorf(log.Global, "Fund manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableDepositAddressManager {
		e.DepositAddressManager = new(DepositAddressManager)
		go e.DepositAddressManager.Sync()
	}

	if e.Settings.EnableOrderManager {
		if err = e.OrderManager.Start(); err != nil {
			log.Errorf(log.Global, "Order manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableExchangeSyncManager {
		exchangeSyncCfg := CurrencyPairSyncerConfig{
			SyncTicker:       e.Settings.EnableTickerSyncing,
			SyncOrderbook:    e.Settings.EnableOrderbookSyncing,
			SyncTrades:       e.Settings.EnableTradeSyncing,
			SyncContinuously: e.Settings.SyncContinuously,
			NumWorkers:       e.Settings.SyncWorkers,
			Verbose:          e.Settings.Verbose,
		}

		e.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(exchangeSyncCfg)
		if err != nil {
			log.Warnf(log.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			go e.ExchangeCurrencyPairManager.Start()
		}
	}

	if e.Settings.EnableEventManager {
		go EventManger()
	}

	if e.Settings.EnableWebsocketRoutine {
		go WebsocketRoutine()
	}

	if e.Settings.EnableGCTScriptManager {
		if e.Config.GCTScript.Enabled {
			if err := e.GctScriptManager.Start(); err != nil {
				log.Errorf(log.Global, "GCTScript manager unable to start: %v", err)
			}
		}
	}

	return nil
}

// Stop correctly shuts down engine saving configuration files
func (e *Engine) Stop() {
	log.Debugln(log.Global, "Engine shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		e.Config.Portfolio = portfolio.Portfolio
	}

	if e.GctScriptManager.Started() {
		if err := e.GctScriptManager.Stop(); err != nil {
			log.Errorf(log.Global, "GCTScript manager unable to stop. Error: %v", err)
		}
	}
	if e.OrderManager.Started() {
		if err := e.OrderManager.Stop(); err != nil {
			log.Errorf(log.Global, "Order manager unable to stop. Error: %v", err)
		}
	}

	if e.NTPManager.Started() {
		if err := e.NTPManager.Stop(); err != nil {
			log.Errorf(log.Global, "NTP manager unable to stop. Error: %v", err)
		}
	}

	if e.CommsManager.Started() {
		if err := e.CommsManager.Stop(); err != nil {
			log.Errorf(log.Global, "Communication manager unable to stop. Error: %v", err)
		}
	}

	if e.PortfolioManager.Started() {
		if err := e.PortfolioManager.Stop(); err != nil {
			log.Errorf(log.Global, "Fund manager unable to stop. Error: %v", err)
		}
	}

	if e.ConnectionManager.Started() {
		if err := e.ConnectionManager.Stop(); err != nil {
			log.Errorf(log.Global, "Connection manager unable to stop. Error: %v", err)
		}
	}

	if e.DatabaseManager.Started() {
		if err := e.DatabaseManager.Stop(); err != nil {
			log.Errorf(log.Global, "Database manager unable to stop. Error: %v", err)
		}
	}

	if dispatch.IsRunning() {
		if err := dispatch.Stop(); err != nil {
			log.Errorf(log.DispatchMgr, "Dispatch system unable to stop. Error: %v", err)
		}
	}

	if !e.Settings.EnableDryRun {
		err := e.Config.SaveConfig(e.Settings.ConfigFile, false)
		if err != nil {
			log.Errorln(log.Global, "Unable to save config.")
		} else {
			log.Debugln(log.Global, "Config file saved successfully.")
		}
	}

	// Wait for services to gracefully shutdown
	e.ServicesWG.Wait()
	err := log.CloseLogger()
	if err != nil {
		fmt.Printf("Failed to close logger %v", err)
	}
}
