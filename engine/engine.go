package engine

import (
	"errors"
	"flag"
	"fmt"
	"log"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/utils"
)

// Engine contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Engine struct {
	Config                      *config.Config
	Portfolio                   *portfolio.Base
	ExchangeCurrencyPairManager *ExchangeCurrencyPairSyncer
	NTPManager                  ntpManager
	ConnectionManager           connectionManager
	DatabaseManager             databaseManager
	GctScriptManager            gctScriptManager
	OrderManager                orderManager
	PortfolioManager            portfolioManager
	CommsManager                commsManager
	exchangeManager             exchangeManager
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

	log.Printf("Loading config file %s..\n", filePath)
	err = b.Config.LoadConfig(filePath, settings.EnableDryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	err = common.CreateDir(settings.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open/create data directory: %s. Err: %s", settings.DataDir, err)
	}

	if *b.Config.Logging.Enabled {
		gctlog.SetupGlobalLogger()
		gctlog.SetupSubLoggers(b.Config.Logging.SubLoggers)
		gctlog.Infoln(gctlog.Global, "Logger initialised.")
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
	b.Settings.EnableCoinmarketcapAnalysis = s.EnableCoinmarketcapAnalysis
	b.Settings.EnableDatabaseManager = s.EnableDatabaseManager
	b.Settings.EnableGCTScriptManager = s.EnableGCTScriptManager
	b.Settings.MaxVirtualMachines = s.MaxVirtualMachines
	b.Settings.EnableDispatcher = s.EnableDispatcher
	b.Settings.EnablePortfolioManager = s.EnablePortfolioManager
	b.Settings.WithdrawCacheSize = s.WithdrawCacheSize
	if b.Settings.EnablePortfolioManager {
		if b.Settings.PortfolioManagerDelay != time.Duration(0) && s.PortfolioManagerDelay > 0 {
			b.Settings.PortfolioManagerDelay = s.PortfolioManagerDelay
		} else {
			b.Settings.PortfolioManagerDelay = PortfolioSleepDelay
		}
	}

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

	if flagSet["withdrawcachesize"] {
		withdraw.CacheSize = s.WithdrawCacheSize
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

	b.Settings.RequestMaxRetryAttempts = s.RequestMaxRetryAttempts
	if b.Settings.RequestMaxRetryAttempts != request.DefaultMaxRetryAttempts && s.RequestMaxRetryAttempts > 0 {
		request.MaxRetryAttempts = b.Settings.RequestMaxRetryAttempts
	}

	b.Settings.HTTPTimeout = s.HTTPTimeout
	if s.HTTPTimeout != time.Duration(0) && s.HTTPTimeout > 0 {
		b.Settings.HTTPTimeout = s.HTTPTimeout
	} else {
		b.Settings.HTTPTimeout = b.Config.GlobalHTTPTimeout
	}

	b.Settings.HTTPUserAgent = s.HTTPUserAgent
	b.Settings.HTTPProxy = s.HTTPProxy

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
	gctlog.Debugln(gctlog.Global)
	gctlog.Debugf(gctlog.Global, "ENGINE SETTINGS")
	gctlog.Debugf(gctlog.Global, "- CORE SETTINGS:")
	gctlog.Debugf(gctlog.Global, "\t Verbose mode: %v", s.Verbose)
	gctlog.Debugf(gctlog.Global, "\t Enable dry run mode: %v", s.EnableDryRun)
	gctlog.Debugf(gctlog.Global, "\t Enable all exchanges: %v", s.EnableAllExchanges)
	gctlog.Debugf(gctlog.Global, "\t Enable all pairs: %v", s.EnableAllPairs)
	gctlog.Debugf(gctlog.Global, "\t Enable coinmarketcap analaysis: %v", s.EnableCoinmarketcapAnalysis)
	gctlog.Debugf(gctlog.Global, "\t Enable portfolio manager: %v", s.EnablePortfolioManager)
	gctlog.Debugf(gctlog.Global, "\t Portfolio manager sleep delay: %v\n", s.PortfolioManagerDelay)
	gctlog.Debugf(gctlog.Global, "\t Enable gPRC: %v", s.EnableGRPC)
	gctlog.Debugf(gctlog.Global, "\t Enable gRPC Proxy: %v", s.EnableGRPCProxy)
	gctlog.Debugf(gctlog.Global, "\t Enable websocket RPC: %v", s.EnableWebsocketRPC)
	gctlog.Debugf(gctlog.Global, "\t Enable deprecated RPC: %v", s.EnableDeprecatedRPC)
	gctlog.Debugf(gctlog.Global, "\t Enable comms relayer: %v", s.EnableCommsRelayer)
	gctlog.Debugf(gctlog.Global, "\t Enable event manager: %v", s.EnableEventManager)
	gctlog.Debugf(gctlog.Global, "\t Event manager sleep delay: %v", s.EventManagerDelay)
	gctlog.Debugf(gctlog.Global, "\t Enable order manager: %v", s.EnableOrderManager)
	gctlog.Debugf(gctlog.Global, "\t Enable exchange sync manager: %v", s.EnableExchangeSyncManager)
	gctlog.Debugf(gctlog.Global, "\t Enable deposit address manager: %v\n", s.EnableDepositAddressManager)
	gctlog.Debugf(gctlog.Global, "\t Enable websocket routine: %v\n", s.EnableWebsocketRoutine)
	gctlog.Debugf(gctlog.Global, "\t Enable NTP client: %v", s.EnableNTPClient)
	gctlog.Debugf(gctlog.Global, "\t Enable Database manager: %v", s.EnableDatabaseManager)
	gctlog.Debugf(gctlog.Global, "\t Enable dispatcher: %v", s.EnableDispatcher)
	gctlog.Debugf(gctlog.Global, "\t Dispatch package max worker amount: %d", s.DispatchMaxWorkerAmount)
	gctlog.Debugf(gctlog.Global, "\t Dispatch package jobs limit: %d", s.DispatchJobsLimit)
	gctlog.Debugf(gctlog.Global, "- EXCHANGE SYNCER SETTINGS:\n")
	gctlog.Debugf(gctlog.Global, "\t Exchange sync continuously: %v\n", s.SyncContinuously)
	gctlog.Debugf(gctlog.Global, "\t Exchange sync workers: %v\n", s.SyncWorkers)
	gctlog.Debugf(gctlog.Global, "\t Enable ticker syncing: %v\n", s.EnableTickerSyncing)
	gctlog.Debugf(gctlog.Global, "\t Enable orderbook syncing: %v\n", s.EnableOrderbookSyncing)
	gctlog.Debugf(gctlog.Global, "\t Enable trade syncing: %v\n", s.EnableTradeSyncing)
	gctlog.Debugf(gctlog.Global, "\t Exchange sync timeout: %v\n", s.SyncTimeout)
	gctlog.Debugf(gctlog.Global, "- FOREX SETTINGS:")
	gctlog.Debugf(gctlog.Global, "\t Enable currency conveter: %v", s.EnableCurrencyConverter)
	gctlog.Debugf(gctlog.Global, "\t Enable currency layer: %v", s.EnableCurrencyLayer)
	gctlog.Debugf(gctlog.Global, "\t Enable fixer: %v", s.EnableFixer)
	gctlog.Debugf(gctlog.Global, "\t Enable OpenExchangeRates: %v", s.EnableOpenExchangeRates)
	gctlog.Debugf(gctlog.Global, "- EXCHANGE SETTINGS:")
	gctlog.Debugf(gctlog.Global, "\t Enable exchange auto pair updates: %v", s.EnableExchangeAutoPairUpdates)
	gctlog.Debugf(gctlog.Global, "\t Disable all exchange auto pair updates: %v", s.DisableExchangeAutoPairUpdates)
	gctlog.Debugf(gctlog.Global, "\t Enable exchange websocket support: %v", s.EnableExchangeWebsocketSupport)
	gctlog.Debugf(gctlog.Global, "\t Enable exchange verbose mode: %v", s.EnableExchangeVerbose)
	gctlog.Debugf(gctlog.Global, "\t Enable exchange HTTP rate limiter: %v", s.EnableExchangeHTTPRateLimiter)
	gctlog.Debugf(gctlog.Global, "\t Enable exchange HTTP debugging: %v", s.EnableExchangeHTTPDebugging)
	gctlog.Debugf(gctlog.Global, "\t Max HTTP request jobs: %v", s.MaxHTTPRequestJobsLimit)
	gctlog.Debugf(gctlog.Global, "\t HTTP request max retry attempts: %v", s.RequestMaxRetryAttempts)
	gctlog.Debugf(gctlog.Global, "\t HTTP timeout: %v", s.HTTPTimeout)
	gctlog.Debugf(gctlog.Global, "\t HTTP user agent: %v", s.HTTPUserAgent)
	gctlog.Debugf(gctlog.Global, "- GCTSCRIPT SETTINGS: ")
	gctlog.Debugf(gctlog.Global, "\t Enable GCTScript manager: %v", s.EnableGCTScriptManager)
	gctlog.Debugf(gctlog.Global, "\t GCTScript max virtual machines: %v", s.MaxVirtualMachines)
	gctlog.Debugf(gctlog.Global, "- WITHDRAW SETTINGS: ")
	gctlog.Debugf(gctlog.Global, "\t Withdraw Cache size: %v", s.WithdrawCacheSize)
	gctlog.Debugf(gctlog.Global, "- COMMON SETTINGS:")
	gctlog.Debugf(gctlog.Global, "\t Global HTTP timeout: %v", s.GlobalHTTPTimeout)
	gctlog.Debugf(gctlog.Global, "\t Global HTTP user agent: %v", s.GlobalHTTPUserAgent)
	gctlog.Debugf(gctlog.Global, "\t Global HTTP proxy: %v", s.GlobalHTTPProxy)

	gctlog.Debugln(gctlog.Global)
}

// Start starts the engine
func (e *Engine) Start() error {
	if e == nil {
		return errors.New("engine instance is nil")
	}

	if e.Settings.EnableDatabaseManager {
		if err := e.DatabaseManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "Database manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableDispatcher {
		if err := dispatch.Start(e.Settings.DispatchMaxWorkerAmount, e.Settings.DispatchJobsLimit); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatcher unable to start: %v", err)
		}
	}

	// Sets up internet connectivity monitor
	if e.Settings.EnableConnectivityMonitor {
		if err := e.ConnectionManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "Connection manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableNTPClient {
		if err := e.NTPManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to start: %v", err)
		}
	}

	e.Uptime = time.Now()
	gctlog.Debugf(gctlog.Global, "Bot '%s' started.\n", e.Config.Name)
	gctlog.Debugf(gctlog.Global, "Using data dir: %s\n", e.Settings.DataDir)
	if *e.Config.Logging.Enabled && strings.Contains(e.Config.Logging.Output, "file") {
		gctlog.Debugf(gctlog.Global, "Using log file: %s\n",
			filepath.Join(gctlog.LogPath, e.Config.Logging.LoggerFileConfig.FileName))
	}
	gctlog.Debugf(gctlog.Global,
		"Using %d out of %d logical processors for runtime performance\n",
		runtime.GOMAXPROCS(-1), runtime.NumCPU())

	enabledExchanges := e.Config.CountEnabledExchanges()
	if e.Settings.EnableAllExchanges {
		enabledExchanges = len(e.Config.Exchanges)
	}

	gctlog.Debugln(gctlog.Global, "EXCHANGE COVERAGE")
	gctlog.Debugf(gctlog.Global, "\t Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(e.Config.Exchanges), enabledExchanges)

	if e.Settings.ExchangePurgeCredentials {
		gctlog.Debugln(gctlog.Global, "Purging exchange API credentials.")
		e.Config.PurgeExchangeAPICredentials()
	}

	gctlog.Debugln(gctlog.Global, "Setting up exchanges..")
	SetupExchanges()
	if Bot.exchangeManager.Len() == 0 {
		return errors.New("no exchanges are loaded")
	}

	if e.Settings.EnableCommsRelayer {
		if err := e.CommsManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "Communications manager unable to start: %v\n", err)
		}
	}

	err := currency.RunStorageUpdater(currency.BotOverrides{
		Coinmarketcap:       e.Settings.EnableCoinmarketcapAnalysis,
		FxCurrencyConverter: e.Settings.EnableCurrencyConverter,
		FxCurrencyLayer:     e.Settings.EnableCurrencyLayer,
		FxFixer:             e.Settings.EnableFixer,
		FxOpenExchangeRates: e.Settings.EnableOpenExchangeRates,
	},
		&currency.MainConfiguration{
			ForexProviders:         e.Config.GetForexProviders(),
			CryptocurrencyProvider: coinmarketcap.Settings(e.Config.Currency.CryptocurrencyProvider),
			Cryptocurrencies:       e.Config.Currency.Cryptocurrencies,
			FiatDisplayCurrency:    e.Config.Currency.FiatDisplayCurrency,
			CurrencyDelay:          e.Config.Currency.CurrencyFileUpdateDuration,
			FxRateDelay:            e.Config.Currency.ForeignExchangeUpdateDuration,
		},
		e.Settings.DataDir,
		e.Settings.Verbose)
	if err != nil {
		gctlog.Errorf(gctlog.Global, "Currency updater system failed to start %v", err)
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
			gctlog.Errorf(gctlog.Global, "Fund manager unable to start: %v", err)
		}
	}

	if e.Settings.EnableDepositAddressManager {
		e.DepositAddressManager = new(DepositAddressManager)
		go e.DepositAddressManager.Sync()
	}

	if e.Settings.EnableOrderManager {
		if err = e.OrderManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to start: %v", err)
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
			SyncTimeout:      e.Settings.SyncTimeout,
		}

		e.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(exchangeSyncCfg)
		if err != nil {
			gctlog.Warnf(gctlog.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
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
				gctlog.Errorf(gctlog.Global, "GCTScript manager unable to start: %v", err)
			}
		}
	}

	return nil
}

// Stop correctly shuts down engine saving configuration files
func (e *Engine) Stop() {
	gctlog.Debugln(gctlog.Global, "Engine shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		e.Config.Portfolio = portfolio.Portfolio
	}

	if e.GctScriptManager.Started() {
		if err := e.GctScriptManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "GCTScript manager unable to stop. Error: %v", err)
		}
	}
	if e.OrderManager.Started() {
		if err := e.OrderManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to stop. Error: %v", err)
		}
	}

	if e.NTPManager.Started() {
		if err := e.NTPManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to stop. Error: %v", err)
		}
	}

	if e.CommsManager.Started() {
		if err := e.CommsManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Communication manager unable to stop. Error: %v", err)
		}
	}

	if e.PortfolioManager.Started() {
		if err := e.PortfolioManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Fund manager unable to stop. Error: %v", err)
		}
	}

	if e.ConnectionManager.Started() {
		if err := e.ConnectionManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Connection manager unable to stop. Error: %v", err)
		}
	}

	if e.DatabaseManager.Started() {
		if err := e.DatabaseManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Database manager unable to stop. Error: %v", err)
		}
	}

	if dispatch.IsRunning() {
		if err := dispatch.Stop(); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatch system unable to stop. Error: %v", err)
		}
	}

	if !e.Settings.EnableDryRun {
		err := e.Config.SaveConfig(e.Settings.ConfigFile, false)
		if err != nil {
			gctlog.Errorln(gctlog.Global, "Unable to save config.")
		} else {
			gctlog.Debugln(gctlog.Global, "Config file saved successfully.")
		}
	}

	// Wait for services to gracefully shutdown
	e.ServicesWG.Wait()
	err := gctlog.CloseLogger()
	if err != nil {
		log.Printf("Failed to close logger. Error: %v\n", err)
	}
}
