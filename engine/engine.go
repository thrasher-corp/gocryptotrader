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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/utils"
)

// Engine contains configuration, portfolio manager, exchange & ticker data and is the
// overarching type across this code base.
type Engine struct {
	Config                  *config.Config
	apiServer               *apiServerManager
	CommunicationsManager   *CommunicationManager
	connectionManager       *connectionManager
	currencyPairSyncer      *syncManager
	DatabaseManager         *DatabaseConnectionManager
	DepositAddressManager   *DepositAddressManager
	eventManager            *eventManager
	ExchangeManager         *ExchangeManager
	ntpManager              *ntpManager
	OrderManager            *OrderManager
	portfolioManager        *portfolioManager
	gctScriptManager        *gctscript.GctScriptManager
	websocketRoutineManager *websocketRoutineManager
	WithdrawManager         *WithdrawManager
	dataHistoryManager      *DataHistoryManager
	Settings                Settings
	uptime                  time.Time
	ServicesWG              sync.WaitGroup
}

// Bot is a happy global engine to allow various areas of the application
// to access its setup services and functions
var Bot *Engine

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

	b.gctScriptManager, err = gctscript.NewManager(&b.Config.GCTScript)
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
	b.Settings = *s

	b.Settings.EnableDataHistoryManager = (flagSet["datahistorymanager"] && b.Settings.EnableDatabaseManager) || b.Config.DataHistoryManager.Enabled

	b.Settings.EnableGCTScriptManager = b.Settings.EnableGCTScriptManager &&
		(flagSet["gctscriptmanager"] || b.Config.GCTScript.Enabled)

	if b.Settings.EnablePortfolioManager {
		if b.Settings.PortfolioManagerDelay <= 0 {
			b.Settings.PortfolioManagerDelay = PortfolioSleepDelay
		}
	}

	if !flagSet["grpc"] {
		b.Settings.EnableGRPC = b.Config.RemoteControl.GRPC.Enabled
	}

	if !flagSet["grpcproxy"] {
		b.Settings.EnableGRPCProxy = b.Config.RemoteControl.GRPC.GRPCProxyEnabled
	}

	if !flagSet["websocketrpc"] {
		b.Settings.EnableWebsocketRPC = b.Config.RemoteControl.WebsocketRPC.Enabled
	}

	if !flagSet["deprecatedrpc"] {
		b.Settings.EnableDeprecatedRPC = b.Config.RemoteControl.DeprecatedRPC.Enabled
	}

	if flagSet["maxvirtualmachines"] {
		maxMachines := uint8(b.Settings.MaxVirtualMachines)
		b.gctScriptManager.MaxVirtualMachines = &maxMachines
	}

	if flagSet["withdrawcachesize"] {
		withdraw.CacheSize = b.Settings.WithdrawCacheSize
	}

	if b.Settings.EnableEventManager && b.Settings.EventManagerDelay <= 0 {
		b.Settings.EventManagerDelay = EventSleepDelay
	}

	// Checks if the flag values are different from the defaults
	if b.Settings.MaxHTTPRequestJobsLimit != int(request.DefaultMaxRequestJobs) &&
		b.Settings.MaxHTTPRequestJobsLimit > 0 {
		request.MaxRequestJobs = int32(b.Settings.MaxHTTPRequestJobsLimit)
	}

	if b.Settings.TradeBufferProcessingInterval != trade.DefaultProcessorIntervalTime {
		if b.Settings.TradeBufferProcessingInterval >= time.Second {
			trade.BufferProcessorIntervalTime = b.Settings.TradeBufferProcessingInterval
		} else {
			b.Settings.TradeBufferProcessingInterval = trade.DefaultProcessorIntervalTime
			gctlog.Warnf(gctlog.Global, "-tradeprocessinginterval must be >= to 1 second, using default value of %v",
				trade.DefaultProcessorIntervalTime)
		}
	}

	if b.Settings.RequestMaxRetryAttempts != request.DefaultMaxRetryAttempts &&
		b.Settings.RequestMaxRetryAttempts > 0 {
		request.MaxRetryAttempts = b.Settings.RequestMaxRetryAttempts
	}

	if b.Settings.HTTPTimeout <= 0 {
		b.Settings.HTTPTimeout = b.Config.GlobalHTTPTimeout
	}

	if b.Settings.GlobalHTTPTimeout <= 0 {
		b.Settings.GlobalHTTPTimeout = b.Config.GlobalHTTPTimeout
	}
	common.SetHTTPClientWithTimeout(b.Settings.GlobalHTTPTimeout)

	if b.Settings.GlobalHTTPUserAgent != "" {
		common.HTTPUserAgent = b.Settings.GlobalHTTPUserAgent
	}
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
	gctlog.Debugf(gctlog.Global, "\t Enable data history manager: %v", s.EnableDataHistoryManager)
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
	gctlog.Debugf(gctlog.Global, "\t Exchange REST sync timeout: %v\n", s.SyncTimeoutREST)
	gctlog.Debugf(gctlog.Global, "\t Exchange Websocket sync timeout: %v\n", s.SyncTimeoutWebsocket)
	gctlog.Debugf(gctlog.Global, "- FOREX SETTINGS:")
	gctlog.Debugf(gctlog.Global, "\t Enable currency conveter: %v", s.EnableCurrencyConverter)
	gctlog.Debugf(gctlog.Global, "\t Enable currency layer: %v", s.EnableCurrencyLayer)
	gctlog.Debugf(gctlog.Global, "\t Enable fixer: %v", s.EnableFixer)
	gctlog.Debugf(gctlog.Global, "\t Enable OpenExchangeRates: %v", s.EnableOpenExchangeRates)
	gctlog.Debugf(gctlog.Global, "\t Enable ExchangeRateHost: %v", s.EnableExchangeRateHost)
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
	var err error
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	if bot.Settings.EnableDatabaseManager {
		bot.DatabaseManager, err = SetupDatabaseConnectionManager(&bot.Config.Database)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Database manager unable to setup: %v", err)
		} else {
			err = bot.DatabaseManager.Start(&bot.ServicesWG)
			if err != nil {
				gctlog.Errorf(gctlog.Global, "Database manager unable to start: %v", err)
			}
		}
	}

	if bot.Settings.EnableDispatcher {
		if err = dispatch.Start(bot.Settings.DispatchMaxWorkerAmount, bot.Settings.DispatchJobsLimit); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatcher unable to start: %v", err)
		}
	}

	// Sets up internet connectivity monitor
	if bot.Settings.EnableConnectivityMonitor {
		bot.connectionManager, err = setupConnectionManager(&bot.Config.ConnectionMonitor)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Connection manager unable to setup: %v", err)
		} else {
			err = bot.connectionManager.Start()
			if err != nil {
				gctlog.Errorf(gctlog.Global, "Connection manager unable to start: %v", err)
			}
		}
	}

	if bot.Settings.EnableNTPClient {
		if bot.Config.NTPClient.Level == 0 {
			var responseMessage string
			responseMessage, err = bot.Config.SetNTPCheck(os.Stdin)
			if err != nil {
				return fmt.Errorf("unable to set NTP check: %w", err)
			}
			gctlog.Info(gctlog.TimeMgr, responseMessage)
		}
		bot.ntpManager, err = setupNTPManager(&bot.Config.NTPClient, *bot.Config.Logging.Enabled)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to start: %s", err)
		}
	}

	bot.uptime = time.Now()
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

	bot.ExchangeManager = SetupExchangeManager()
	gctlog.Debugln(gctlog.Global, "Setting up exchanges..")
	err = bot.SetupExchanges()
	if err != nil {
		return err
	}

	if bot.Settings.EnableCommsRelayer {
		bot.CommunicationsManager, err = SetupCommunicationManager(&bot.Config.Communications)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Communications manager unable to setup: %s", err)
		} else {
			err = bot.CommunicationsManager.Start()
			if err != nil {
				gctlog.Errorf(gctlog.Global, "Communications manager unable to start: %s", err)
			}
		}
	}
	if bot.Settings.EnableCoinmarketcapAnalysis ||
		bot.Settings.EnableCurrencyConverter ||
		bot.Settings.EnableCurrencyLayer ||
		bot.Settings.EnableFixer ||
		bot.Settings.EnableOpenExchangeRates ||
		bot.Settings.EnableExchangeRateHost {
		err = currency.RunStorageUpdater(currency.BotOverrides{
			Coinmarketcap:       bot.Settings.EnableCoinmarketcapAnalysis,
			FxCurrencyConverter: bot.Settings.EnableCurrencyConverter,
			FxCurrencyLayer:     bot.Settings.EnableCurrencyLayer,
			FxFixer:             bot.Settings.EnableFixer,
			FxOpenExchangeRates: bot.Settings.EnableOpenExchangeRates,
			FxExchangeRateHost:  bot.Settings.EnableExchangeRateHost,
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
			gctlog.Errorf(gctlog.Global, "ExchangeSettings updater system failed to start %s", err)
		}
	}

	if bot.Settings.EnableGRPC {
		go StartRPCServer(bot)
	}

	if bot.Settings.EnablePortfolioManager {
		if bot.portfolioManager == nil {
			bot.portfolioManager, err = setupPortfolioManager(bot.ExchangeManager, bot.Settings.PortfolioManagerDelay, &bot.Config.Portfolio)
			if err != nil {
				gctlog.Errorf(gctlog.Global, "portfolio manager unable to setup: %s", err)
			} else {
				err = bot.portfolioManager.Start(&bot.ServicesWG)
				if err != nil {
					gctlog.Errorf(gctlog.Global, "portfolio manager unable to start: %s", err)
				}
			}
		}
	}

	if bot.Settings.EnableDataHistoryManager {
		if bot.dataHistoryManager == nil {
			bot.dataHistoryManager, err = SetupDataHistoryManager(bot.ExchangeManager, bot.DatabaseManager, &bot.Config.DataHistoryManager)
			if err != nil {
				gctlog.Errorf(gctlog.Global, "database history manager unable to setup: %s", err)
			} else {
				err = bot.dataHistoryManager.Start()
				if err != nil {
					gctlog.Errorf(gctlog.Global, "database history manager unable to start: %s", err)
				}
			}
		}
	}

	bot.WithdrawManager, err = SetupWithdrawManager(bot.ExchangeManager, bot.portfolioManager, bot.Settings.EnableDryRun)
	if err != nil {
		return err
	}

	if bot.Settings.EnableDeprecatedRPC ||
		bot.Settings.EnableWebsocketRPC {
		var filePath string
		filePath, err = config.GetAndMigrateDefaultPath(bot.Settings.ConfigFile)
		if err != nil {
			return err
		}
		bot.apiServer, err = setupAPIServerManager(&bot.Config.RemoteControl, &bot.Config.Profiler, bot.ExchangeManager, bot, bot.portfolioManager, filePath)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "API Server unable to start: %s", err)
		} else {
			if bot.Settings.EnableDeprecatedRPC {
				err = bot.apiServer.StartRESTServer()
				if err != nil {
					gctlog.Errorf(gctlog.Global, "could not start REST API server: %s", err)
				}
			}
			if bot.Settings.EnableWebsocketRPC {
				err = bot.apiServer.StartWebsocketServer()
				if err != nil {
					gctlog.Errorf(gctlog.Global, "could not start websocket API server: %s", err)
				}
			}
		}
	}

	if bot.Settings.EnableDepositAddressManager {
		bot.DepositAddressManager = SetupDepositAddressManager()
		go func() {
			err = bot.DepositAddressManager.Sync(bot.GetExchangeCryptocurrencyDepositAddresses())
			if err != nil {
				gctlog.Errorf(gctlog.Global, "Deposit address manager unable to setup: %s", err)
			}
		}()
	}

	if bot.Settings.EnableOrderManager {
		bot.OrderManager, err = SetupOrderManager(
			bot.ExchangeManager,
			bot.CommunicationsManager,
			&bot.ServicesWG,
			bot.Settings.Verbose)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to setup: %s", err)
		} else {
			err = bot.OrderManager.Start()
			if err != nil {
				gctlog.Errorf(gctlog.Global, "Order manager unable to start: %s", err)
			}
		}
	}

	if bot.Settings.EnableExchangeSyncManager {
		exchangeSyncCfg := &Config{
			SyncTicker:           bot.Settings.EnableTickerSyncing,
			SyncOrderbook:        bot.Settings.EnableOrderbookSyncing,
			SyncTrades:           bot.Settings.EnableTradeSyncing,
			SyncContinuously:     bot.Settings.SyncContinuously,
			NumWorkers:           bot.Settings.SyncWorkers,
			Verbose:              bot.Settings.Verbose,
			SyncTimeoutREST:      bot.Settings.SyncTimeoutREST,
			SyncTimeoutWebsocket: bot.Settings.SyncTimeoutWebsocket,
		}

		bot.currencyPairSyncer, err = setupSyncManager(
			exchangeSyncCfg,
			bot.ExchangeManager,
			bot.websocketRoutineManager,
			&bot.Config.RemoteControl)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			go func() {
				err = bot.currencyPairSyncer.Start()
				if err != nil {
					gctlog.Errorf(gctlog.Global, "failed to start exchange currency pair manager. Err: %s", err)
				}
			}()
		}
	}

	if bot.Settings.EnableEventManager {
		bot.eventManager, err = setupEventManager(bot.CommunicationsManager, bot.ExchangeManager, bot.Settings.EventManagerDelay, bot.Settings.EnableDryRun)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise event manager. Err: %s", err)
		} else {
			err = bot.eventManager.Start()
			if err != nil {
				gctlog.Errorf(gctlog.Global, "failed to start event manager. Err: %s", err)
			}
		}
	}

	if bot.Settings.EnableWebsocketRoutine {
		bot.websocketRoutineManager, err = setupWebsocketRoutineManager(bot.ExchangeManager, bot.OrderManager, bot.currencyPairSyncer, &bot.Config.Currency, bot.Settings.Verbose)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise websocket routine manager. Err: %s", err)
		} else {
			err = bot.websocketRoutineManager.Start()
			if err != nil {
				gctlog.Errorf(gctlog.Global, "failed to start websocket routine manager. Err: %s", err)
			}
		}
	}

	if bot.Settings.EnableGCTScriptManager {
		bot.gctScriptManager, err = gctscript.NewManager(&bot.Config.GCTScript)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "failed to create script manager. Err: %s", err)
		}
		if err := bot.gctScriptManager.Start(&bot.ServicesWG); err != nil {
			gctlog.Errorf(gctlog.Global, "GCTScript manager unable to start: %s", err)
		}
	}

	return nil
}

// Stop correctly shuts down engine saving configuration files
func (bot *Engine) Stop() {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	gctlog.Debugln(gctlog.Global, "Engine shutting down..")

	if len(bot.portfolioManager.GetAddresses()) != 0 {
		bot.Config.Portfolio = *bot.portfolioManager.GetPortfolio()
	}

	if bot.gctScriptManager.IsRunning() {
		if err := bot.gctScriptManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "GCTScript manager unable to stop. Error: %v", err)
		}
	}
	if bot.OrderManager.IsRunning() {
		if err := bot.OrderManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to stop. Error: %v", err)
		}
	}

	if bot.eventManager.IsRunning() {
		if err := bot.eventManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "event manager unable to stop. Error: %v", err)
		}
	}

	if bot.ntpManager.IsRunning() {
		if err := bot.ntpManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to stop. Error: %v", err)
		}
	}

	if bot.CommunicationsManager.IsRunning() {
		if err := bot.CommunicationsManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Communication manager unable to stop. Error: %v", err)
		}
	}

	if bot.portfolioManager.IsRunning() {
		if err := bot.portfolioManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Fund manager unable to stop. Error: %v", err)
		}
	}

	if bot.connectionManager.IsRunning() {
		if err := bot.connectionManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Connection manager unable to stop. Error: %v", err)
		}
	}

	if bot.apiServer.IsRESTServerRunning() {
		if err := bot.apiServer.StopRESTServer(); err != nil {
			gctlog.Errorf(gctlog.Global, "API Server unable to stop REST server. Error: %s", err)
		}
	}

	if bot.apiServer.IsWebsocketServerRunning() {
		if err := bot.apiServer.StopWebsocketServer(); err != nil {
			gctlog.Errorf(gctlog.Global, "API Server unable to stop websocket server. Error: %s", err)
		}
	}

	if bot.dataHistoryManager.IsRunning() {
		if err := bot.dataHistoryManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.DataHistory, "data history manager unable to stop. Error: %v", err)
		}
	}

	if bot.DatabaseManager.IsRunning() {
		if err := bot.DatabaseManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "Database manager unable to stop. Error: %v", err)
		}
	}

	if dispatch.IsRunning() {
		if err := dispatch.Stop(); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatch system unable to stop. Error: %v", err)
		}
	}
	if bot.websocketRoutineManager.IsRunning() {
		if err := bot.websocketRoutineManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "websocket routine manager unable to stop. Error: %v", err)
		}
	}

	if bot.Settings.EnableCoinmarketcapAnalysis ||
		bot.Settings.EnableCurrencyConverter ||
		bot.Settings.EnableCurrencyLayer ||
		bot.Settings.EnableFixer ||
		bot.Settings.EnableOpenExchangeRates ||
		bot.Settings.EnableExchangeRateHost {
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
	return bot.ExchangeManager.GetExchangeByName(exchName)
}

// UnloadExchange unloads an exchange by name
func (bot *Engine) UnloadExchange(exchName string) error {
	exchCfg, err := bot.Config.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	err = bot.ExchangeManager.RemoveExchange(exchName)
	if err != nil {
		return err
	}

	exchCfg.Enabled = false
	return nil
}

// GetExchanges retrieves the loaded exchanges
func (bot *Engine) GetExchanges() []exchange.IBotExchange {
	return bot.ExchangeManager.GetExchanges()
}

// LoadExchange loads an exchange by name. Optional wait group can be added for
// external synchronization.
func (bot *Engine) LoadExchange(name string, wg *sync.WaitGroup) error {
	exch, err := bot.ExchangeManager.NewExchangeByName(name)
	if err != nil {
		return err
	}
	if exch.GetBase() == nil {
		return ErrExchangeFailedToLoad
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
		assets := exchCfg.CurrencyPairs.GetAssetTypes(false)
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

	bot.ExchangeManager.Add(exch)
	base := exch.GetBase()
	if base.API.AuthenticatedSupport ||
		base.API.AuthenticatedWebsocketSupport {
		assetTypes := base.GetAssetTypes(false)
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

	if wg != nil {
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
		go func(c config.ExchangeConfig) {
			defer wg.Done()
			err := bot.LoadExchange(c.Name, &wg)
			if err != nil {
				gctlog.Errorf(gctlog.ExchangeSys, "LoadExchange %s failed: %s\n", c.Name, err)
				return
			}
			gctlog.Debugf(gctlog.ExchangeSys,
				"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
				c.Name,
				common.IsEnabled(c.API.AuthenticatedSupport),
				common.IsEnabled(c.Verbose),
			)
		}(configs[x])
	}
	wg.Wait()
	if len(bot.ExchangeManager.GetExchanges()) == 0 {
		return ErrNoExchangesLoaded
	}
	return nil
}

// WaitForInitialCurrencySync allows for a routine to wait for the initial sync
// of the currency pair syncer management system.
func (bot *Engine) WaitForInitialCurrencySync() error {
	return bot.currencyPairSyncer.WaitForInitialSync()
}
