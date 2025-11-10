package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
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
	CommunicationsManager   *CommunicationManager
	connectionManager       *connectionManager
	currencyPairSyncer      *SyncManager
	DatabaseManager         *DatabaseConnectionManager
	DepositAddressManager   *DepositAddressManager
	eventManager            *eventManager
	ExchangeManager         *ExchangeManager
	ntpManager              *ntpManager
	OrderManager            *OrderManager
	portfolioManager        *portfolioManager
	gctScriptManager        *gctscript.GctScriptManager
	WebsocketRoutineManager *WebsocketRoutineManager
	WithdrawManager         *WithdrawManager
	dataHistoryManager      *DataHistoryManager
	currencyStateManager    *CurrencyStateManager
	Settings                Settings
	uptime                  time.Time
	GRPCShutdownSignal      chan struct{}
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
	b.Config = config.GetConfig()

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
		return nil, fmt.Errorf("failed to load config. Err: %w", err)
	}

	if *b.Config.Logging.Enabled {
		err = gctlog.SetupGlobalLogger(b.Config.Name, b.Config.Logging.AdvancedSettings.StructuredLogging)
		if err != nil {
			return nil, fmt.Errorf("failed to setup global logger. %w", err)
		}
		err = gctlog.SetupSubLoggers(b.Config.Logging.SubLoggers)
		if err != nil {
			return nil, fmt.Errorf("failed to setup sub loggers. %w", err)
		}
		gctlog.Infoln(gctlog.Global, "Logger initialised.")
	}

	b.Settings.ConfigFile = settings.ConfigFile
	b.Settings.DataDir = b.Config.GetDataPath()
	b.Settings.CheckParamInteraction = settings.CheckParamInteraction

	err = utils.AdjustGoMaxProcs(settings.GoMaxProcs)
	if err != nil {
		return nil, fmt.Errorf("unable to adjust runtime GOMAXPROCS value. Err: %w", err)
	}

	b.gctScriptManager, err = gctscript.NewManager(&b.Config.GCTScript)
	if err != nil {
		return nil, fmt.Errorf("failed to create script manager. Err: %w", err)
	}

	b.ExchangeManager = NewExchangeManager()

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
		return nil, fmt.Errorf("%w %s: %w", config.ErrFailureOpeningConfig, filePath, err)
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

// FlagSet defines set flags from command line args for comparison methods
type FlagSet map[string]bool

// WithBool checks the supplied flag. If set it will override the config boolean
// value as a command line takes precedence. If not set will fall back to config
// options.
func (f FlagSet) WithBool(key string, flagValue *bool, configValue bool) {
	isSet := f[key]
	*flagValue = !isSet && configValue || isSet && *flagValue
}

// validateSettings validates and sets all bot settings
func validateSettings(b *Engine, s *Settings, flagSet FlagSet) {
	b.Settings = *s

	flagSet.WithBool("coinmarketcap", &b.Settings.EnableCoinmarketcapAnalysis, b.Config.Currency.CryptocurrencyProvider.Enabled)
	flagSet.WithBool("ordermanager", &b.Settings.EnableOrderManager, b.Config.OrderManager.Enabled)

	flagSet.WithBool("currencyconverter", &b.Settings.EnableCurrencyConverter, b.Config.Currency.ForexProviders.IsEnabled("currencyconverter"))

	flagSet.WithBool("currencylayer", &b.Settings.EnableCurrencyLayer, b.Config.Currency.ForexProviders.IsEnabled("currencylayer"))
	flagSet.WithBool("exchangerates", &b.Settings.EnableExchangeRates, b.Config.Currency.ForexProviders.IsEnabled("exchangerates"))
	flagSet.WithBool("fixer", &b.Settings.EnableFixer, b.Config.Currency.ForexProviders.IsEnabled("fixer"))
	flagSet.WithBool("openexchangerates", &b.Settings.EnableOpenExchangeRates, b.Config.Currency.ForexProviders.IsEnabled("openexchangerates"))

	flagSet.WithBool("datahistorymanager", &b.Settings.EnableDataHistoryManager, b.Config.DataHistoryManager.Enabled)
	flagSet.WithBool("currencystatemanager", &b.Settings.EnableCurrencyStateManager, b.Config.CurrencyStateManager.Enabled != nil && *b.Config.CurrencyStateManager.Enabled)
	flagSet.WithBool("gctscriptmanager", &b.Settings.EnableGCTScriptManager, b.Config.GCTScript.Enabled)

	flagSet.WithBool("tickersync", &b.Settings.EnableTickerSyncing, b.Config.SyncManagerConfig.SynchronizeTicker)
	flagSet.WithBool("orderbooksync", &b.Settings.EnableOrderbookSyncing, b.Config.SyncManagerConfig.SynchronizeOrderbook)
	flagSet.WithBool("tradesync", &b.Settings.EnableTradeSyncing, b.Config.SyncManagerConfig.SynchronizeTrades)
	flagSet.WithBool("synccontinuously", &b.Settings.SyncContinuously, b.Config.SyncManagerConfig.SynchronizeContinuously)
	flagSet.WithBool("syncmanager", &b.Settings.EnableExchangeSyncManager, b.Config.SyncManagerConfig.Enabled)

	if b.Settings.EnablePortfolioManager &&
		b.Settings.PortfolioManagerDelay <= 0 {
		b.Settings.PortfolioManagerDelay = PortfolioSleepDelay
	}

	flagSet.WithBool("grpc", &b.Settings.EnableGRPC, b.Config.RemoteControl.GRPC.Enabled)
	flagSet.WithBool("grpcproxy", &b.Settings.EnableGRPCProxy, b.Config.RemoteControl.GRPC.GRPCProxyEnabled)

	flagSet.WithBool("grpcshutdown", &b.Settings.EnableGRPCShutdown, b.Config.RemoteControl.GRPC.GRPCAllowBotShutdown)
	if b.Settings.EnableGRPCShutdown {
		b.GRPCShutdownSignal = make(chan struct{})
		go b.waitForGPRCShutdown()
	}

	if flagSet["maxvirtualmachines"] {
		maxMachines := b.Settings.MaxVirtualMachines
		b.gctScriptManager.MaxVirtualMachines = &maxMachines
	}

	if flagSet["withdrawcachesize"] {
		withdraw.CacheSize = b.Settings.WithdrawCacheSize
	}

	if b.Settings.EnableEventManager && b.Settings.EventManagerDelay <= 0 {
		b.Settings.EventManagerDelay = EventSleepDelay
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

	err := common.SetHTTPClientWithTimeout(b.Settings.GlobalHTTPTimeout)
	if err != nil {
		gctlog.Errorf(gctlog.Global,
			"Could not set new HTTP Client with timeout %s error: %v",
			b.Settings.GlobalHTTPTimeout,
			err)
	}

	if b.Settings.GlobalHTTPUserAgent != "" {
		err = common.SetHTTPUserAgent(b.Settings.GlobalHTTPUserAgent)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Could not set HTTP User Agent for %s error: %v",
				b.Settings.GlobalHTTPUserAgent,
				err)
		}
	}

	if b.Settings.AlertSystemPreAllocationCommsBuffer != alert.PreAllocCommsDefaultBuffer {
		err = alert.SetPreAllocationCommsBuffer(b.Settings.AlertSystemPreAllocationCommsBuffer)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Could not set alert pre-allocation comms buffer to %v: %v",
				b.Settings.AlertSystemPreAllocationCommsBuffer,
				err)
		}
	}
}

// PrintLoadedSettings logs loaded settings.
func (s *Settings) PrintLoadedSettings() {
	if s == nil {
		return
	}
	gctlog.Debugln(gctlog.Global)
	gctlog.Debugf(gctlog.Global, "ENGINE SETTINGS")
	settings := reflect.ValueOf(*s)
	for x := range settings.NumField() {
		field := settings.Field(x)
		if field.Kind() != reflect.Struct {
			continue
		}

		fieldName := field.Type().Name()
		gctlog.Debugln(gctlog.Global, "- "+common.AddPaddingOnUpperCase(fieldName)+":")
		for y := range field.NumField() {
			indvSetting := field.Field(y)
			indvName := field.Type().Field(y).Name
			if indvSetting.Kind() == reflect.String && indvSetting.IsZero() {
				indvSetting = reflect.ValueOf("Undefined")
			}
			gctlog.Debugln(gctlog.Global, "\t", common.AddPaddingOnUpperCase(indvName)+":", indvSetting)
		}
	}
	gctlog.Debugln(gctlog.Global)
}

// Start starts the engine
func (bot *Engine) Start() error {
	if bot == nil {
		return errors.New("engine instance is nil")
	}
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	if bot.Config.Profiler.Enabled {
		if err := StartPPROF(context.TODO(), &bot.Config.Profiler); err != nil {
			gctlog.Errorf(gctlog.Global, "Failed to start pprof: %v", err)
		}
	}

	if bot.Settings.EnableDatabaseManager {
		if d, err := SetupDatabaseConnectionManager(&bot.Config.Database); err != nil {
			gctlog.Errorf(gctlog.Global, "Database manager unable to setup: %v", err)
		} else {
			bot.DatabaseManager = d
			if err := bot.DatabaseManager.Start(&bot.ServicesWG); err != nil && !errors.Is(err, database.ErrDatabaseSupportDisabled) {
				gctlog.Errorf(gctlog.Global, "Database manager unable to start: %v", err)
			}
		}
	}

	if bot.Settings.EnableDispatcher {
		if err := dispatch.Start(bot.Settings.DispatchMaxWorkerAmount, bot.Settings.DispatchJobsLimit); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatcher unable to start: %v", err)
		}
	}

	// Sets up internet connectivity monitor
	if bot.Settings.EnableConnectivityMonitor {
		if c, err := setupConnectionManager(&bot.Config.ConnectionMonitor); err != nil {
			gctlog.Errorf(gctlog.Global, "Connection manager unable to setup: %v", err)
		} else {
			bot.connectionManager = c
			if err := bot.connectionManager.Start(); err != nil {
				gctlog.Errorf(gctlog.Global, "Connection manager unable to start: %v", err)
			}
		}
	}

	if bot.Settings.EnableNTPClient {
		if bot.Config.NTPClient.Level == 0 {
			responseMessage, err := bot.Config.SetNTPCheck(os.Stdin)
			if err != nil {
				return fmt.Errorf("unable to set NTP check: %w", err)
			}
			gctlog.Infoln(gctlog.TimeMgr, responseMessage)
		}
		if n, err := setupNTPManager(&bot.Config.NTPClient, *bot.Config.Logging.Enabled); err != nil {
			gctlog.Errorf(gctlog.Global, "NTP manager unable to start: %s", err)
		} else {
			bot.ntpManager = n
		}
	}

	bot.uptime = time.Now()
	gctlog.Debugf(gctlog.Global, "Bot %q started.\n", bot.Config.Name)
	gctlog.Debugf(gctlog.Global, "Using data dir: %s\n", bot.Settings.DataDir)
	if *bot.Config.Logging.Enabled && strings.Contains(bot.Config.Logging.Output, "file") {
		gctlog.Debugf(gctlog.Global,
			"Using log file: %s\n",
			filepath.Join(gctlog.GetLogPath(),
				bot.Config.Logging.LoggerFileConfig.FileName),
		)
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
	if err := bot.SetupExchanges(); err != nil {
		return err
	}

	if bot.Settings.EnableCommsRelayer {
		if c, err := SetupCommunicationManager(&bot.Config.Communications); err != nil {
			gctlog.Errorf(gctlog.Global, "Communications manager unable to setup: %s", err)
		} else {
			bot.CommunicationsManager = c
			if err := bot.CommunicationsManager.Start(); err != nil {
				gctlog.Errorf(gctlog.Global, "Communications manager unable to start: %s", err)
			}
		}
	}

	if err := currency.RunStorageUpdater(
		currency.BotOverrides{
			Coinmarketcap:     bot.Settings.EnableCoinmarketcapAnalysis,
			CurrencyConverter: bot.Settings.EnableCurrencyConverter,
			CurrencyLayer:     bot.Settings.EnableCurrencyLayer,
			ExchangeRates:     bot.Settings.EnableExchangeRates,
			Fixer:             bot.Settings.EnableFixer,
			OpenExchangeRates: bot.Settings.EnableOpenExchangeRates,
		},
		&bot.Config.Currency,
		bot.Settings.DataDir,
	); err != nil {
		gctlog.Errorf(gctlog.Global, "Currency Converter system failed to start %s", err)
	}

	if bot.Settings.EnableGRPC {
		go StartRPCServer(bot)
	}

	if bot.Settings.EnablePortfolioManager {
		if bot.portfolioManager == nil {
			if p, err := setupPortfolioManager(bot.ExchangeManager, bot.Settings.PortfolioManagerDelay, bot.Config.Portfolio); err != nil {
				gctlog.Errorf(gctlog.Global, "portfolio manager unable to setup: %s", err)
			} else {
				bot.portfolioManager = p
				if err := bot.portfolioManager.Start(&bot.ServicesWG); err != nil {
					gctlog.Errorf(gctlog.Global, "portfolio manager unable to start: %s", err)
				}
			}
		}
	}

	if bot.Settings.EnableDataHistoryManager {
		if bot.dataHistoryManager == nil {
			if d, err := SetupDataHistoryManager(bot.ExchangeManager, bot.DatabaseManager, &bot.Config.DataHistoryManager); err != nil {
				gctlog.Errorf(gctlog.Global, "database history manager unable to setup: %s", err)
			} else {
				bot.dataHistoryManager = d
				if err := bot.dataHistoryManager.Start(); err != nil {
					gctlog.Errorf(gctlog.Global, "database history manager unable to start: %s", err)
				}
			}
		}
	}

	if w, err := SetupWithdrawManager(bot.ExchangeManager, bot.portfolioManager, bot.Settings.EnableDryRun); err != nil {
		return err
	} else { //nolint:revive // TODO: revive false positive, see https://github.com/mgechev/revive/pull/832 for more information
		bot.WithdrawManager = w
	}

	if bot.Settings.EnableDepositAddressManager {
		bot.DepositAddressManager = SetupDepositAddressManager()
		go func() {
			if err := bot.DepositAddressManager.Sync(bot.GetAllExchangeCryptocurrencyDepositAddresses()); err != nil {
				gctlog.Errorf(gctlog.Global, "Deposit address manager unable to setup: %s", err)
			}
		}()
	}

	if bot.Settings.EnableOrderManager {
		if o, err := SetupOrderManager(
			bot.ExchangeManager,
			bot.CommunicationsManager,
			&bot.ServicesWG,
			&bot.Config.OrderManager); err != nil {
			gctlog.Errorf(gctlog.Global, "Order manager unable to setup: %s", err)
		} else {
			bot.OrderManager = o
			if err = bot.OrderManager.Start(); err != nil {
				gctlog.Errorf(gctlog.Global, "Order manager unable to start: %s", err)
			}
		}
	}

	if bot.Settings.EnableExchangeSyncManager {
		cfg := bot.Config.SyncManagerConfig
		cfg.SynchronizeTicker = bot.Settings.EnableTickerSyncing
		cfg.SynchronizeOrderbook = bot.Settings.EnableOrderbookSyncing
		cfg.SynchronizeContinuously = bot.Settings.SyncContinuously
		cfg.SynchronizeTrades = bot.Settings.EnableTradeSyncing
		cfg.Verbose = bot.Settings.Verbose || cfg.Verbose

		if cfg.TimeoutREST != bot.Settings.SyncTimeoutREST &&
			bot.Settings.SyncTimeoutREST != config.DefaultSyncerTimeoutREST {
			cfg.TimeoutREST = bot.Settings.SyncTimeoutREST
		}
		if cfg.TimeoutWebsocket != bot.Settings.SyncTimeoutWebsocket &&
			bot.Settings.SyncTimeoutWebsocket != config.DefaultSyncerTimeoutWebsocket {
			cfg.TimeoutWebsocket = bot.Settings.SyncTimeoutWebsocket
		}
		if cfg.NumWorkers != bot.Settings.SyncWorkersCount &&
			bot.Settings.SyncWorkersCount != config.DefaultSyncerWorkers {
			cfg.NumWorkers = bot.Settings.SyncWorkersCount
		}
		if s, err := SetupSyncManager(
			&cfg,
			bot.ExchangeManager,
			&bot.Config.RemoteControl,
			bot.Settings.EnableWebsocketRoutine,
		); err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			bot.currencyPairSyncer = s
			go func() {
				if err := bot.currencyPairSyncer.Start(); err != nil {
					gctlog.Errorf(gctlog.Global, "failed to start exchange currency pair manager. Err: %s", err)
				}
			}()
		}
	}

	if bot.Settings.EnableEventManager {
		if e, err := setupEventManager(bot.CommunicationsManager, bot.ExchangeManager, bot.Settings.EventManagerDelay, bot.Settings.EnableDryRun); err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise event manager. Err: %s", err)
		} else {
			bot.eventManager = e
			if err = bot.eventManager.Start(); err != nil {
				gctlog.Errorf(gctlog.Global, "failed to start event manager. Err: %s", err)
			}
		}
	}

	if bot.Settings.EnableWebsocketRoutine {
		if w, err := setupWebsocketRoutineManager(bot.ExchangeManager, bot.OrderManager, bot.currencyPairSyncer, &bot.Config.Currency, bot.Settings.Verbose); err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise websocket routine manager. Err: %s", err)
		} else {
			bot.WebsocketRoutineManager = w
			if err = bot.WebsocketRoutineManager.Start(); err != nil {
				gctlog.Errorf(gctlog.Global, "failed to start websocket routine manager. Err: %s", err)
			}
		}
	}

	if bot.Settings.EnableGCTScriptManager {
		if g, err := gctscript.NewManager(&bot.Config.GCTScript); err != nil {
			gctlog.Errorf(gctlog.Global, "failed to create script manager. Err: %s", err)
		} else {
			bot.gctScriptManager = g
			if err := bot.gctScriptManager.Start(&bot.ServicesWG); err != nil {
				gctlog.Errorf(gctlog.Global, "GCTScript manager unable to start: %s", err)
			}
		}
	}

	if bot.Settings.EnableCurrencyStateManager {
		if c, err := SetupCurrencyStateManager(
			bot.Config.CurrencyStateManager.Delay,
			bot.ExchangeManager,
		); err != nil {
			gctlog.Errorf(gctlog.Global,
				"%s unable to setup: %s",
				CurrencyStateManagementName,
				err)
		} else {
			bot.currencyStateManager = c
			if err := bot.currencyStateManager.Start(); err != nil {
				gctlog.Errorf(gctlog.Global,
					"%s unable to start: %s",
					CurrencyStateManagementName,
					err)
			}
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
		bot.Config.Portfolio = bot.portfolioManager.GetPortfolio()
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
	if bot.WebsocketRoutineManager.IsRunning() {
		if err := bot.WebsocketRoutineManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "websocket routine manager unable to stop. Error: %v", err)
		}
	}
	if bot.currencyStateManager.IsRunning() {
		if err := bot.currencyStateManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global,
				"currency state manager unable to stop. Error: %v",
				err)
		}
	}

	err := bot.ExchangeManager.Shutdown(bot.Settings.ExchangeShutdownTimeout)
	if err != nil {
		gctlog.Errorf(gctlog.Global, "Exchange manager unable to stop. Error: %v", err)
	}

	err = currency.ShutdownStorageUpdater()
	if err != nil {
		gctlog.Errorf(gctlog.Global, "Currency Converter unable to stop. Error: %v", err)
	}

	if !bot.Settings.EnableDryRun {
		err = bot.Config.SaveConfigToFile(bot.Settings.ConfigFile)
		if err != nil {
			gctlog.Errorln(gctlog.Global, "Unable to save config.")
		} else {
			gctlog.Debugln(gctlog.Global, "Config file saved successfully.")
		}
	}

	// Wait for services to gracefully shutdown
	bot.ServicesWG.Wait()
	gctlog.Infoln(gctlog.Global, "Exiting.")
	if err := gctlog.CloseLogger(); err != nil {
		log.Printf("Failed to close logger. Error: %v\n", err)
	}
}

// GetExchangeByName returns an exchange given an exchange name
func (bot *Engine) GetExchangeByName(exchName string) (exchange.IBotExchange, error) {
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
	exch, err := bot.ExchangeManager.GetExchanges()
	if err != nil {
		gctlog.Warnf(gctlog.ExchangeSys, "Cannot get exchanges: %v", err)
		return []exchange.IBotExchange{}
	}
	return exch
}

// LoadExchange loads an exchange by name. Optional wait group can be added for
// external synchronization.
func (bot *Engine) LoadExchange(name string) error {
	exch, err := bot.ExchangeManager.NewExchangeByName(name)
	if err != nil {
		return err
	}
	if exch.GetBase() == nil {
		return ErrExchangeFailedToLoad
	}

	exch.SetDefaults()

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
			err = exchCfg.CurrencyPairs.StorePairs(assets[x], pairs, true)
			if err != nil {
				return err
			}
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

	if !bot.Settings.EnableExchangeHTTPRateLimiter {
		err = exch.DisableRateLimiter()
		if err != nil {
			gctlog.Errorf(gctlog.ExchangeSys, "%s error disabling rate limiter: %v", exch.GetName(), err)
		} else {
			gctlog.Warnf(gctlog.ExchangeSys, "%s rate limiting has been turned off", exch.GetName())
		}
	}

	// NOTE: This will standardize name to default and apply it to the config.
	exchCfg.Name = exch.GetName()

	exchCfg.Enabled = true
	err = exch.Setup(exchCfg)
	if err != nil {
		exchCfg.Enabled = false
		return err
	}

	err = bot.ExchangeManager.Add(exch)
	if err != nil {
		return err
	}

	b := exch.GetBase()
	if b.API.AuthenticatedSupport || b.API.AuthenticatedWebsocketSupport {
		err = exch.ValidateAPICredentials(context.TODO(), asset.Spot)
		if err != nil {
			gctlog.Warnf(gctlog.ExchangeSys, "%s: Error validating credentials: %v", b.Name, err)
			b.API.AuthenticatedSupport = false
			b.API.AuthenticatedWebsocketSupport = false
			exchCfg.API.AuthenticatedSupport = false
			exchCfg.API.AuthenticatedWebsocketSupport = false
			if b.Websocket != nil {
				b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			}
		}
	}

	return exchange.Bootstrap(context.TODO(), exch)
}

func (bot *Engine) dryRunParamInteraction(param string) {
	if !bot.Settings.CheckParamInteraction {
		return
	}

	gctlog.Warnf(gctlog.Global, "Command line argument '-%s' induces dry run mode. Set -dryrun=false if you wish to override this.", param)

	if !bot.Settings.EnableDryRun {
		bot.Settings.EnableDryRun = true
	}
}

// SetupExchanges sets up the exchanges used by the Bot
func (bot *Engine) SetupExchanges() error {
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

	var exchangesOverride []string
	if bot.Settings.Exchanges != "" {
		bot.dryRunParamInteraction("exchanges")
		exchangesOverride = strings.Split(bot.Settings.Exchanges, ",")
		for x := range exchangesOverride {
			if !common.StringSliceCompareInsensitive(exchange.Exchanges, exchangesOverride[x]) {
				return fmt.Errorf("exchange %s not found", exchangesOverride[x])
			}
		}
	}

	if bot.Settings.EnableAllExchanges && len(exchangesOverride) > 0 {
		return errors.New("cannot enable all exchanges and specific exchanges concurrently")
	}

	var wg sync.WaitGroup
	for x := range configs {
		shouldLoad := false
		if len(exchangesOverride) > 0 {
			for y := range exchangesOverride {
				if strings.EqualFold(configs[x].Name, exchangesOverride[y]) {
					shouldLoad = true
					break
				}
			}
		} else {
			shouldLoad = configs[x].Enabled || bot.Settings.EnableAllExchanges
		}

		if !shouldLoad {
			gctlog.Debugf(gctlog.ExchangeSys, "%s: Exchange support: Disabled\n", configs[x].Name)
			continue
		}

		wg.Add(1)
		go func(c config.Exchange) {
			defer wg.Done()
			if err := bot.LoadExchange(c.Name); err != nil {
				gctlog.Errorf(gctlog.ExchangeSys, "LoadExchange %s failed: %s\n", c.Name, err)
			} else {
				gctlog.Debugf(gctlog.ExchangeSys,
					"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
					c.Name,
					common.IsEnabled(c.API.AuthenticatedSupport),
					common.IsEnabled(c.Verbose),
				)
			}
		}(configs[x])
	}
	wg.Wait()
	if len(bot.GetExchanges()) == 0 {
		return ErrNoExchangesLoaded
	}
	return nil
}

// WaitForInitialCurrencySync allows for a routine to wait for the initial sync
// of the currency pair syncer management system.
func (bot *Engine) WaitForInitialCurrencySync() error {
	return bot.currencyPairSyncer.WaitForInitialSync()
}

// RegisterWebsocketDataHandler registers an externally defined data handler
// for diverting and handling websocket notifications across all enabled
// exchanges. InterceptorOnly as true will purge all other registered handlers
// (including default) bypassing all other handling.
func (bot *Engine) RegisterWebsocketDataHandler(fn WebsocketDataHandler, interceptorOnly bool) error {
	if bot == nil {
		return errNilBot
	}
	return bot.WebsocketRoutineManager.registerWebsocketDataHandler(fn, interceptorOnly)
}

// SetDefaultWebsocketDataHandler sets the default websocket handler and
// removing all pre-existing handlers
func (bot *Engine) SetDefaultWebsocketDataHandler() error {
	if bot == nil {
		return errNilBot
	}
	return bot.WebsocketRoutineManager.setWebsocketDataHandler(bot.WebsocketRoutineManager.websocketDataHandler)
}

// waitForGPRCShutdown routines waits for a signal from the grpc server to
// send a shutdown signal.
func (bot *Engine) waitForGPRCShutdown() {
	<-bot.GRPCShutdownSignal
	gctlog.Warnln(gctlog.Global, "Captured gRPC shutdown request.")
	bot.Settings.Shutdown <- struct{}{}
}
