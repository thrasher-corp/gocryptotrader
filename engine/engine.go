package engine

import (
	"errors"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
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
	GctScriptManager            *gctscript.GctScriptManager
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
)

// New starts a new engine
func New() (*Engine, error) {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()
	var b Engine
	b.Config = &config.Cfg

	err := b.Config.LoadConfig("", false)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}
	b.GctScriptManager, err = gctscript.NewManager(&b.Config.GCTScript)
	if err != nil {
		return nil, fmt.Errorf("failed to create script manager. Err: %s", err)
	}

	return &b, nil
}

// NewFromSettings starts a new engine based on supplied settings
func NewFromSettings(settings *Settings, flagSet map[string]bool) (*Engine, error) {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()
	if settings == nil {
		return nil, errors.New("engine: settings is nil")
	}

	var b Engine
	var err error

	b.Config, err = loadConfigWithSettings(settings, flagSet)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	if *b.Config.Logging.Enabled {
		gctlog.SetupGlobalLogger()
		gctlog.SetupSubLoggers(b.Config.Logging.SubLoggers)
		gctlog.Infoln(gctlog.Global, "Logger initialised.")
	}

	b.Settings.ConfigFile = settings.ConfigFile
	b.Settings.DataDir = b.Config.GetDataPath()
	b.Settings.CheckParamInteraction = settings.CheckParamInteraction

	err = utils.AdjustGoMaxProcs(settings.GoMaxProcs)
	if err != nil {
		return nil, fmt.Errorf("unable to adjust runtime GOMAXPROCS value. Err: %s", err)
	}

	b.GctScriptManager, err = gctscript.NewManager(&b.Config.GCTScript)
	if err != nil {
		return nil, fmt.Errorf("failed to create script manager. Err: %s", err)
	}

	validateSettings(&b, settings, flagSet)

	return &b, nil
}

// loadConfigWithSettings creates configuration based on the provided settings
func loadConfigWithSettings(settings *Settings, flagSet map[string]bool) (*config.Config, error) {
	filePath, err := config.GetAndMigrateDefaultPath(settings.ConfigFile)
	if err != nil {
		return nil, err
	}
	log.Printf("Loading config file %s..\n", filePath)

	conf := &config.Config{}
	err = conf.ReadConfigFromFile(filePath, settings.EnableDryRun)
	if err != nil {
		return nil, fmt.Errorf(config.ErrFailureOpeningConfig, filePath, err)
	}
	// Apply overrides from settings
	if flagSet["datadir"] {
		// warn if dryrun isn't enabled
		if !settings.EnableDryRun {
			log.Println("Command line argument '-datadir' induces dry run mode.")
		}
		settings.EnableDryRun = true
		conf.DataDirectory = settings.DataDir
	}

	return conf, conf.CheckConfig()
}

// validateSettings validates and sets all bot settings
func validateSettings(b *Engine, s *Settings, flagSet map[string]bool) {
	b.Settings.Verbose = s.Verbose
	b.Settings.EnableDryRun = s.EnableDryRun
	b.Settings.EnableAllExchanges = s.EnableAllExchanges
	b.Settings.EnableAllPairs = s.EnableAllPairs
	b.Settings.EnableCoinmarketcapAnalysis = s.EnableCoinmarketcapAnalysis
	b.Settings.EnableDatabaseManager = s.EnableDatabaseManager
	b.Settings.EnableGCTScriptManager = s.EnableGCTScriptManager && (flagSet["gctscriptmanager"] || b.Config.GCTScript.Enabled)
	b.Settings.MaxVirtualMachines = s.MaxVirtualMachines
	b.Settings.EnableDispatcher = s.EnableDispatcher
	b.Settings.EnablePortfolioManager = s.EnablePortfolioManager
	b.Settings.WithdrawCacheSize = s.WithdrawCacheSize
	if b.Settings.EnablePortfolioManager {
		if b.Settings.PortfolioManagerDelay == time.Duration(0) && s.PortfolioManagerDelay > 0 {
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

	if flagSet["maxvirtualmachines"] {
		maxMachines := uint8(s.MaxVirtualMachines)
		b.GctScriptManager.MaxVirtualMachines = &maxMachines
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

	b.Settings.TradeBufferProcessingInterval = s.TradeBufferProcessingInterval
	if b.Settings.TradeBufferProcessingInterval != trade.DefaultProcessorIntervalTime {
		if b.Settings.TradeBufferProcessingInterval >= time.Second {
			trade.BufferProcessorIntervalTime = b.Settings.TradeBufferProcessingInterval
		} else {
			b.Settings.TradeBufferProcessingInterval = trade.DefaultProcessorIntervalTime
			gctlog.Warnf(gctlog.Global, "-tradeprocessinginterval must be >= to 1 second, using default value of %v",
				trade.DefaultProcessorIntervalTime)
		}
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
	gctlog.Debugf(gctlog.Global, "\t Trade buffer processing interval: %v", s.TradeBufferProcessingInterval)
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
func (bot *Engine) Start() error {
	if bot == nil {
		return errors.New("engine instance is nil")
	}

	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	if bot.Settings.EnableDatabaseManager {
		if err := bot.DatabaseManager.Start(bot); err != nil {
			gctlog.Errorf(gctlog.Global, "Database manager unable to start: %v", err)
		}
	}

	if bot.Settings.EnableDispatcher {
		if err := dispatch.Start(bot.Settings.DispatchMaxWorkerAmount, bot.Settings.DispatchJobsLimit); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatcher unable to start: %v", err)
		}
	}

	// Sets up internet connectivity monitor
	if bot.Settings.EnableConnectivityMonitor {
		if err := bot.ConnectionManager.Start(&bot.Config.ConnectionMonitor); err != nil {
			gctlog.Errorf(gctlog.Global, "Connection manager unable to start: %v", err)
		}
	}

	if bot.Settings.EnableNTPClient {
		if err := bot.NTPManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to start: %v", err)
		}
	}

	bot.Uptime = time.Now()
	gctlog.Debugf(gctlog.Global, "Bot '%s' started.\n", bot.Config.Name)
	gctlog.Debugf(gctlog.Global, "Using data dir: %s\n", bot.Settings.DataDir)
	if *bot.Config.Logging.Enabled && strings.Contains(bot.Config.Logging.Output, "file") {
		gctlog.Debugf(gctlog.Global, "Using log file: %s\n",
			filepath.Join(gctlog.LogPath, bot.Config.Logging.LoggerFileConfig.FileName))
	}
	gctlog.Debugf(gctlog.Global,
		"Using %d out of %d logical processors for runtime performance\n",
		runtime.GOMAXPROCS(-1), runtime.NumCPU())

	enabledExchanges := bot.Config.CountEnabledExchanges()
	if bot.Settings.EnableAllExchanges {
		enabledExchanges = len(bot.Config.Exchanges)
	}

	gctlog.Debugln(gctlog.Global, "EXCHANGE COVERAGE")
	gctlog.Debugf(gctlog.Global, "\t Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(bot.Config.Exchanges), enabledExchanges)

	if bot.Settings.ExchangePurgeCredentials {
		gctlog.Debugln(gctlog.Global, "Purging exchange API credentials.")
		bot.Config.PurgeExchangeAPICredentials()
	}

	gctlog.Debugln(gctlog.Global, "Setting up exchanges..")
	err := bot.SetupExchanges()
	if err != nil {
		return err
	}

	if bot.Settings.EnableCommsRelayer {
		if err = bot.CommsManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "Communications manager unable to start: %v\n", err)
		}
	}
	if bot.Settings.EnableCoinmarketcapAnalysis ||
		bot.Settings.EnableCurrencyConverter ||
		bot.Settings.EnableCurrencyLayer ||
		bot.Settings.EnableFixer ||
		bot.Settings.EnableOpenExchangeRates {
		err = currency.RunStorageUpdater(currency.BotOverrides{
			Coinmarketcap:       bot.Settings.EnableCoinmarketcapAnalysis,
			FxCurrencyConverter: bot.Settings.EnableCurrencyConverter,
			FxCurrencyLayer:     bot.Settings.EnableCurrencyLayer,
			FxFixer:             bot.Settings.EnableFixer,
			FxOpenExchangeRates: bot.Settings.EnableOpenExchangeRates,
		},
			&currency.MainConfiguration{
				ForexProviders:         bot.Config.GetForexProviders(),
				CryptocurrencyProvider: coinmarketcap.Settings(bot.Config.Currency.CryptocurrencyProvider),
				Cryptocurrencies:       bot.Config.Currency.Cryptocurrencies,
				FiatDisplayCurrency:    bot.Config.Currency.FiatDisplayCurrency,
				CurrencyDelay:          bot.Config.Currency.CurrencyFileUpdateDuration,
				FxRateDelay:            bot.Config.Currency.ForeignExchangeUpdateDuration,
			},
			bot.Settings.DataDir)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "ExchangeSettings updater system failed to start %v", err)
		}
	}

	if bot.Settings.EnableGRPC {
		go StartRPCServer(bot)
	}

	if bot.Settings.EnableDeprecatedRPC {
		go StartRESTServer(bot)
	}

	if bot.Settings.EnableWebsocketRPC {
		go StartWebsocketServer(bot)
		StartWebsocketHandler()
	}

	if bot.Settings.EnablePortfolioManager {
		if err = bot.PortfolioManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "Fund manager unable to start: %v", err)
		}
	}

	if bot.Settings.EnableDepositAddressManager {
		bot.DepositAddressManager = new(DepositAddressManager)
		go bot.DepositAddressManager.Sync()
	}

	if bot.Settings.EnableOrderManager {
		if err = bot.OrderManager.Start(bot); err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to start: %v", err)
		}
	}

	if bot.Settings.EnableExchangeSyncManager {
		exchangeSyncCfg := CurrencyPairSyncerConfig{
			SyncTicker:       bot.Settings.EnableTickerSyncing,
			SyncOrderbook:    bot.Settings.EnableOrderbookSyncing,
			SyncTrades:       bot.Settings.EnableTradeSyncing,
			SyncContinuously: bot.Settings.SyncContinuously,
			NumWorkers:       bot.Settings.SyncWorkers,
			Verbose:          bot.Settings.Verbose,
			SyncTimeout:      bot.Settings.SyncTimeout,
		}

		bot.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(exchangeSyncCfg)
		if err != nil {
			gctlog.Warnf(gctlog.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			go bot.ExchangeCurrencyPairManager.Start()
		}
	}

	if bot.Settings.EnableEventManager {
		go EventManger(bot.Settings.Verbose, &bot.CommsManager)
	}

	if bot.Settings.EnableWebsocketRoutine {
		go bot.WebsocketRoutine()
	}

	if bot.Settings.EnableGCTScriptManager {
		if err := bot.GctScriptManager.Start(&bot.ServicesWG); err != nil {
			gctlog.Errorf(gctlog.Global, "GCTScript manager unable to start: %v", err)
		}
	}

	return nil
}

// Stop correctly shuts down engine saving configuration files
func (bot *Engine) Stop() {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	gctlog.Debugln(gctlog.Global, "Engine shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		bot.Config.Portfolio = portfolio.Portfolio
	}

	if bot.GctScriptManager.Started() {
		if err := bot.GctScriptManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "GCTScript manager unable to stop. Error: %v", err)
		}
	}
	if bot.OrderManager.Started() {
		if err := bot.OrderManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to stop. Error: %v", err)
		}
	}

	if bot.NTPManager.Started() {
		if err := bot.NTPManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to stop. Error: %v", err)
		}
	}

	if bot.CommsManager.Started() {
		if err := bot.CommsManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Communication manager unable to stop. Error: %v", err)
		}
	}

	if bot.PortfolioManager.Started() {
		if err := bot.PortfolioManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Fund manager unable to stop. Error: %v", err)
		}
	}

	if bot.ConnectionManager.Started() {
		if err := bot.ConnectionManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Connection manager unable to stop. Error: %v", err)
		}
	}

	if bot.DatabaseManager.Started() {
		if err := bot.DatabaseManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Database manager unable to stop. Error: %v", err)
		}
	}

	if dispatch.IsRunning() {
		if err := dispatch.Stop(); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatch system unable to stop. Error: %v", err)
		}
	}

	if bot.Settings.EnableCoinmarketcapAnalysis ||
		bot.Settings.EnableCurrencyConverter ||
		bot.Settings.EnableCurrencyLayer ||
		bot.Settings.EnableFixer ||
		bot.Settings.EnableOpenExchangeRates {
		if err := currency.ShutdownStorageUpdater(); err != nil {
			gctlog.Errorf(gctlog.Global, "ExchangeSettings storage system. Error: %v", err)
		}
	}

	if !bot.Settings.EnableDryRun {
		err := bot.Config.SaveConfigToFile(bot.Settings.ConfigFile)
		if err != nil {
			gctlog.Errorln(gctlog.Global, "Unable to save config.")
		} else {
			gctlog.Debugln(gctlog.Global, "Config file saved successfully.")
		}
	}

	// Wait for services to gracefully shutdown
	bot.ServicesWG.Wait()
	err := gctlog.CloseLogger()
	if err != nil {
		log.Printf("Failed to close logger. Error: %v\n", err)
	}
}
