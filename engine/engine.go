package engine

import (
	"errors"
	"fmt"
	"log"
	"os"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/subsystems/apiserver"
	"github.com/thrasher-corp/gocryptotrader/subsystems/connectionmanager"
	"github.com/thrasher-corp/gocryptotrader/subsystems/database"
	"github.com/thrasher-corp/gocryptotrader/subsystems/depositaddress"
	"github.com/thrasher-corp/gocryptotrader/subsystems/events"
	"github.com/thrasher-corp/gocryptotrader/subsystems/events/communicationmanager"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
	"github.com/thrasher-corp/gocryptotrader/subsystems/ntp"
	"github.com/thrasher-corp/gocryptotrader/subsystems/ordermanager"
	"github.com/thrasher-corp/gocryptotrader/subsystems/portfoliosync"
	"github.com/thrasher-corp/gocryptotrader/subsystems/rpcserver"
	"github.com/thrasher-corp/gocryptotrader/subsystems/syncer"
	"github.com/thrasher-corp/gocryptotrader/subsystems/withdrawalmanager"
	"github.com/thrasher-corp/gocryptotrader/utils"
)

// Engine contains configuration, portfoliomanager, exchange & ticker data and is the
// overarching type across this code base.
type Engine struct {
	Config                      *config.Config
	Portfolio                   *portfolio.Base
	ExchangeCurrencyPairManager *syncer.ExchangeCurrencyPairSyncer
	NTPManager                  ntp.Manager
	ConnectionManager           connectionmanager.ConnectionManager
	DatabaseManager             database.Manager
	GctScriptManager            *gctscript.GctScriptManager
	OrderManager                ordermanager.Manager
	PortfolioManager            portfoliosync.Manager
	CommsManager                communicationmanager.Manager
	exchangeManager             exchangemanager.Manager
	eventManager                *events.Manager
	DepositAddressManager       *depositaddress.Manager
	WithdrawalManager           *withdrawalmanager.Manager
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
			b.Settings.PortfolioManagerDelay = portfoliosync.PortfolioSleepDelay
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
			b.Settings.EventManagerDelay = events.EventSleepDelay
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
		if err := bot.DatabaseManager.Start(&bot.Config.Database, &bot.ServicesWG); err != nil {
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
		if bot.Config.NTPClient.Level == 0 {
			responseMessage, err := bot.Config.SetNTPCheck(os.Stdin)
			if err != nil {
				return fmt.Errorf("unable to disable NTP check: %w", err)
			}
			gctlog.Info(gctlog.TimeMgr, responseMessage)
		}
		if err := bot.NTPManager.Start(&bot.Config.NTPClient, *bot.Config.Logging.Enabled); err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to start: %w", err)
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

	bot.WithdrawalManager, err = withdrawalmanager.Setup(&bot.exchangeManager, bot.Settings.EnableDryRun)
	if err != nil {
		return err
	}

	if bot.Settings.EnableCommsRelayer {
		if err = bot.CommsManager.nStart(); err != nil {
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
		err = checkCerts(utils.GetTLSDir(bot.Settings.DataDir))
		if err != nil {
			return err
		}
		go rpcserver.StartRPCServer(bot)
	}

	if bot.Settings.EnableDeprecatedRPC {
		go apiserver.StartRESTServer(bot.Config.RemoteControl, bot.Config.Profiler)
	}

	if bot.Settings.EnableWebsocketRPC {
		go apiserver.StartWebsocketServer(bot.Config.RemoteControl, bot.Config.Profiler)
		apiserver.StartWebsocketHandler()
	}

	if bot.Settings.EnablePortfolioManager {
		if err = bot.PortfolioManager.Start(); err != nil {
			gctlog.Errorf(gctlog.Global, "Fund manager unable to start: %v", err)
		}
	}

	if bot.Settings.EnableDepositAddressManager {
		bot.DepositAddressManager = new(depositaddress.Manager)
		go bot.DepositAddressManager.Sync(bot.GetExchangeCryptocurrencyDepositAddresses())
	}

	if bot.Settings.EnableOrderManager {
		if err = bot.OrderManager.Start(bot); err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to start: %v", err)
		}
	}

	if bot.Settings.EnableExchangeSyncManager {
		exchangeSyncCfg := syncer.CurrencyPairSyncerConfig{
			SyncTicker:       bot.Settings.EnableTickerSyncing,
			SyncOrderbook:    bot.Settings.EnableOrderbookSyncing,
			SyncTrades:       bot.Settings.EnableTradeSyncing,
			SyncContinuously: bot.Settings.SyncContinuously,
			NumWorkers:       bot.Settings.SyncWorkers,
			Verbose:          bot.Settings.Verbose,
			SyncTimeout:      bot.Settings.SyncTimeout,
		}

		bot.ExchangeCurrencyPairManager, err = syncer.NewCurrencyPairSyncer(exchangeSyncCfg)
		if err != nil {
			gctlog.Warnf(gctlog.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			go bot.ExchangeCurrencyPairManager.Start()
		}
	}

	if bot.Settings.EnableEventManager {
		bot.eventManager = events.Setup(&bot.CommsManager, bot.Settings.EnableDryRun)
		go bot.eventManager.Start()
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

// GetExchangeByName returns an exchange given an exchange name
func (bot *Engine) GetExchangeByName(exchName string) exchange.IBotExchange {
	return bot.exchangeManager.GetExchangeByName(exchName)
}

// UnloadExchange unloads an exchange by name
func (bot *Engine) UnloadExchange(exchName string) error {
	exchCfg, err := bot.Config.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	err = bot.exchangeManager.RemoveExchange(exchName)
	if err != nil {
		return err
	}

	exchCfg.Enabled = false
	return nil
}

// GetExchanges retrieves the loaded exchanges
func (bot *Engine) GetExchanges() []exchange.IBotExchange {
	return bot.exchangeManager.GetExchanges()
}

// LoadExchange loads an exchange by name
func (bot *Engine) LoadExchange(name string, useWG bool, wg *sync.WaitGroup) error {
	exch, err := bot.exchangeManager.NewExchangeByName(name)
	if err != nil {
		return err
	}
	if exch.GetBase() == nil {
		return exchangemanager.ErrExchangeFailedToLoad
	}

	var localWG sync.WaitGroup
	localWG.Add(1)
	go func() {
		exch.SetDefaults()
		localWG.Done()
	}()
	exchCfg, err := bot.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	if bot.Settings.EnableAllPairs &&
		exchCfg.CurrencyPairs != nil {
		assets := exchCfg.CurrencyPairs.GetAssetTypes()
		for x := range assets {
			var pairs currency.Pairs
			pairs, err = exchCfg.CurrencyPairs.GetPairs(assets[x], false)
			if err != nil {
				return err
			}
			exchCfg.CurrencyPairs.StorePairs(assets[x], pairs, true)
		}
	}

	if bot.Settings.EnableExchangeVerbose {
		exchCfg.Verbose = true
	}
	if exchCfg.Features != nil {
		if bot.Settings.EnableExchangeWebsocketSupport &&
			exchCfg.Features.Supports.Websocket {
			exchCfg.Features.Enabled.Websocket = true
		}
		if bot.Settings.EnableExchangeAutoPairUpdates &&
			exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
			exchCfg.Features.Enabled.AutoPairUpdates = true
		}
		if bot.Settings.DisableExchangeAutoPairUpdates {
			if exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
				exchCfg.Features.Enabled.AutoPairUpdates = false
			}
		}
	}
	if bot.Settings.HTTPUserAgent != "" {
		exchCfg.HTTPUserAgent = bot.Settings.HTTPUserAgent
	}
	if bot.Settings.HTTPProxy != "" {
		exchCfg.ProxyAddress = bot.Settings.HTTPProxy
	}
	if bot.Settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		exchCfg.HTTPTimeout = bot.Settings.HTTPTimeout
	}
	if bot.Settings.EnableExchangeHTTPDebugging {
		exchCfg.HTTPDebugging = bot.Settings.EnableExchangeHTTPDebugging
	}

	localWG.Wait()
	if !bot.Settings.EnableExchangeHTTPRateLimiter {
		gctlog.Warnf(gctlog.ExchangeSys,
			"Loaded exchange %s rate limiting has been turned off.\n",
			exch.GetName(),
		)
		err = exch.DisableRateLimiter()
		if err != nil {
			gctlog.Errorf(gctlog.ExchangeSys,
				"Loaded exchange %s rate limiting cannot be turned off: %s.\n",
				exch.GetName(),
				err,
			)
		}
	}

	exchCfg.Enabled = true
	err = exch.Setup(exchCfg)
	if err != nil {
		exchCfg.Enabled = false
		return err
	}

	bot.exchangeManager.Add(exch)
	base := exch.GetBase()
	if base.API.AuthenticatedSupport ||
		base.API.AuthenticatedWebsocketSupport {
		assetTypes := base.GetAssetTypes()
		var useAsset asset.Item
		for a := range assetTypes {
			err = base.CurrencyPairs.IsAssetEnabled(assetTypes[a])
			if err != nil {
				continue
			}
			useAsset = assetTypes[a]
			break
		}
		err = exch.ValidateCredentials(useAsset)
		if err != nil {
			gctlog.Warnf(gctlog.ExchangeSys,
				"%s: Cannot validate credentials, authenticated support has been disabled, Error: %s\n",
				base.Name,
				err)
			base.API.AuthenticatedSupport = false
			base.API.AuthenticatedWebsocketSupport = false
			exchCfg.API.AuthenticatedSupport = false
			exchCfg.API.AuthenticatedWebsocketSupport = false
		}
	}

	if useWG {
		exch.Start(wg)
	} else {
		tempWG := sync.WaitGroup{}
		exch.Start(&tempWG)
		tempWG.Wait()
	}

	return nil
}

func (bot *Engine) dryRunParamInteraction(param string) {
	if !bot.Settings.CheckParamInteraction {
		return
	}

	if !bot.Settings.EnableDryRun {
		gctlog.Warnf(gctlog.Global,
			"Command line argument '-%s' induces dry run mode."+
				" Set -dryrun=false if you wish to override this.",
			param)
		bot.Settings.EnableDryRun = true
	}
}

// SetupExchanges sets up the exchanges used by the Bot
func (bot *Engine) SetupExchanges() error {
	var wg sync.WaitGroup
	configs := bot.Config.GetAllExchangeConfigs()
	if bot.Settings.EnableAllPairs {
		bot.dryRunParamInteraction("enableallpairs")
	}
	if bot.Settings.EnableAllExchanges {
		bot.dryRunParamInteraction("enableallexchanges")
	}
	if bot.Settings.EnableExchangeVerbose {
		bot.dryRunParamInteraction("exchangeverbose")
	}
	if bot.Settings.EnableExchangeWebsocketSupport {
		bot.dryRunParamInteraction("exchangewebsocketsupport")
	}
	if bot.Settings.EnableExchangeAutoPairUpdates {
		bot.dryRunParamInteraction("exchangeautopairupdates")
	}
	if bot.Settings.DisableExchangeAutoPairUpdates {
		bot.dryRunParamInteraction("exchangedisableautopairupdates")
	}
	if bot.Settings.HTTPUserAgent != "" {
		bot.dryRunParamInteraction("httpuseragent")
	}
	if bot.Settings.HTTPProxy != "" {
		bot.dryRunParamInteraction("httpproxy")
	}
	if bot.Settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		bot.dryRunParamInteraction("httptimeout")
	}
	if bot.Settings.EnableExchangeHTTPDebugging {
		bot.dryRunParamInteraction("exchangehttpdebugging")
	}

	for x := range configs {
		if !configs[x].Enabled && !bot.Settings.EnableAllExchanges {
			gctlog.Debugf(gctlog.ExchangeSys, "%s: Exchange support: Disabled\n", configs[x].Name)
			continue
		}
		wg.Add(1)
		cfg := configs[x]
		go func(currCfg config.ExchangeConfig) {
			defer wg.Done()
			err := bot.LoadExchange(currCfg.Name, true, &wg)
			if err != nil {
				gctlog.Errorf(gctlog.ExchangeSys, "LoadExchange %s failed: %s\n", currCfg.Name, err)
				return
			}
			gctlog.Debugf(gctlog.ExchangeSys,
				"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
				currCfg.Name,
				common.IsEnabled(currCfg.API.AuthenticatedSupport),
				common.IsEnabled(currCfg.Verbose),
			)
		}(cfg)
	}
	wg.Wait()
	if len(bot.exchangeManager.GetExchanges()) == 0 {
		return exchangemanager.ErrNoExchangesLoaded
	}
	return nil
}

// WebsocketRoutine Initial routine management system for websocket
func (bot *Engine) WebsocketRoutine() {
	if bot.Settings.Verbose {
		gctlog.Debugln(gctlog.WebsocketMgr, "Connecting exchange websocket services...")
	}

	exchanges := bot.GetExchanges()
	for i := range exchanges {
		go func(i int) {
			if exchanges[i].SupportsWebsocket() {
				if bot.Settings.Verbose {
					gctlog.Debugf(gctlog.WebsocketMgr,
						"Exchange %s websocket support: Yes Enabled: %v\n",
						exchanges[i].GetName(),
						common.IsEnabled(exchanges[i].IsWebsocketEnabled()),
					)
				}

				ws, err := exchanges[i].GetWebsocket()
				if err != nil {
					gctlog.Errorf(
						gctlog.WebsocketMgr,
						"Exchange %s GetWebsocket error: %s\n",
						exchanges[i].GetName(),
						err,
					)
					return
				}

				// Exchange sync manager might have already started ws
				// service or is in the process of connecting, so check
				if ws.IsConnected() || ws.IsConnecting() {
					return
				}

				// Data handler routine
				go bot.WebsocketDataReceiver(ws)

				if ws.IsEnabled() {
					err = ws.Connect()
					if err != nil {
						gctlog.Errorf(gctlog.WebsocketMgr, "%v\n", err)
					}
					err = ws.FlushChannels()
					if err != nil {
						gctlog.Errorf(gctlog.WebsocketMgr, "Failed to subscribe: %v\n", err)
					}
				}
			} else if bot.Settings.Verbose {
				gctlog.Debugf(gctlog.WebsocketMgr,
					"Exchange %s websocket support: No\n",
					exchanges[i].GetName(),
				)
			}
		}(i)
	}
}

var shutdowner = make(chan struct{}, 1)
var wg sync.WaitGroup

// WebsocketDataReceiver handles websocket data coming from a websocket feed
// associated with an exchange
func (bot *Engine) WebsocketDataReceiver(ws *stream.Websocket) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-shutdowner:
			return
		case data := <-ws.ToRoutine:
			err := bot.WebsocketDataHandler(ws.GetName(), data)
			if err != nil {
				gctlog.Error(gctlog.WebsocketMgr, err)
			}
		}
	}
}

// WebsocketDataHandler is a central point for exchange websocket implementations to send
// processed data. WebsocketDataHandler will then pass that to an appropriate handler
func (bot *Engine) WebsocketDataHandler(exchName string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("routines.go - exchange %s nil data sent to websocket",
			exchName)
	}

	switch d := data.(type) {
	case string:
		gctlog.Info(gctlog.WebsocketMgr, d)
	case error:
		return fmt.Errorf("routines.go exchange %s websocket error - %s", exchName, data)
	case stream.FundingData:
		if bot.Settings.Verbose {
			gctlog.Infof(gctlog.WebsocketMgr, "%s websocket %s %s funding updated %+v",
				exchName,
				bot.FormatCurrency(d.CurrencyPair),
				d.AssetType,
				d)
		}
	case *ticker.Price:
		if bot.Settings.EnableExchangeSyncManager && bot.ExchangeCurrencyPairManager != nil {
			bot.ExchangeCurrencyPairManager.Update(exchName,
				d.Pair,
				d.AssetType,
				syncer.SyncItemTicker,
				nil)
		}
		err := ticker.ProcessTicker(d)
		syncer.PrintTickerSummary(d, "websocket", err)
	case stream.KlineData:
		if bot.Settings.Verbose {
			gctlog.Infof(gctlog.WebsocketMgr, "%s websocket %s %s kline updated %+v",
				exchName,
				bot.FormatCurrency(d.Pair),
				d.AssetType,
				d)
		}
	case *orderbook.Base:
		if bot.Settings.EnableExchangeSyncManager && bot.ExchangeCurrencyPairManager != nil {
			bot.ExchangeCurrencyPairManager.Update(exchName,
				d.Pair,
				d.AssetType,
				syncer.SyncItemOrderbook,
				nil)
		}
		syncer.PrintOrderbookSummary(d, "websocket", bot, nil)
	case *order.Detail:
		if !bot.OrderManager.Exists(d) {
			err := bot.OrderManager.Add(d)
			if err != nil {
				return err
			}
		} else {
			od, err := bot.OrderManager.GetByExchangeAndID(d.Exchange, d.ID)
			if err != nil {
				return err
			}
			od.UpdateOrderFromDetail(d)
		}
	case *order.Cancel:
		return bot.OrderManager.Cancel(d)
	case *order.Modify:
		od, err := bot.OrderManager.GetByExchangeAndID(d.Exchange, d.ID)
		if err != nil {
			return err
		}
		od.UpdateOrderFromModify(d)
	case order.ClassificationError:
		return errors.New(d.Error())
	case stream.UnhandledMessageWarning:
		gctlog.Warn(gctlog.WebsocketMgr, d.Message)
	default:
		if bot.Settings.Verbose {
			gctlog.Warnf(gctlog.WebsocketMgr,
				"%s websocket Unknown type: %+v",
				exchName,
				d)
		}
	}
	return nil
}
