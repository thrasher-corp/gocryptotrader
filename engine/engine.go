package engine

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/connchecker"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/engine/events"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/ntpclient"
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
	OrderManager                   *OrderManager
	CommsRelayer                   *communications.Communications
	Connectivity                   *connchecker.Checker
	Shutdown                       chan bool
	Settings                       Settings
	CryptocurrencyDepositAddresses map[string]map[string]string
	Uptime                         time.Time
	DB                             database.Databaser
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
	log.Debugf("Loading config file %s..\n", settings.ConfigFile)
	err := b.Config.LoadConfig(settings.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	err = common.CreateDir(settings.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open/create data directory: %s. Err: %s", settings.DataDir, err)
	}

	err = b.Config.CheckLoggerConfig()
	if err != nil {
		log.Errorf("Failed to configure logger. Err: %s", err)
	}

	err = log.SetupLogger()
	if err != nil {
		log.Errorf("Failed to setup logger. Err: %s", err)
	}

	b.Settings.ConfigFile = settings.ConfigFile
	b.Settings.DataDir = settings.DataDir
	b.Settings.LogFile = path.Join(log.LogPath, log.Logger.File)
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
	b.Settings.EnablePortfolioWatcher = s.EnablePortfolioWatcher
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
		events.Verbose = b.Settings.Verbose
		if b.Settings.EventManagerDelay != time.Duration(0) && s.EventManagerDelay > 0 {
			b.Settings.EventManagerDelay = s.EventManagerDelay
		} else {
			b.Settings.EventManagerDelay = events.SleepDelay
		}
	}

	b.Settings.EnableNTPClient = s.EnableNTPClient
	b.Settings.EnableTickerRoutine = s.EnableTickerRoutine
	b.Settings.EnableOrderbookRoutine = s.EnableOrderbookRoutine
	b.Settings.EnableWebsocketRoutine = s.EnableWebsocketRoutine
	b.Settings.EnableExchangeAutoPairUpdates = s.EnableExchangeAutoPairUpdates
	b.Settings.EnableExchangeWebsocketSupport = s.EnableExchangeWebsocketSupport
	b.Settings.EnableExchangeRESTSupport = s.EnableExchangeRESTSupport
	b.Settings.EnableExchangeVerbose = s.EnableExchangeVerbose
	b.Settings.EnableHTTPRateLimiter = s.EnableHTTPRateLimiter
	b.Settings.DisableExchangeAutoPairUpdates = s.DisableExchangeAutoPairUpdates
	b.Settings.ExchangePurgeCredentials = s.ExchangePurgeCredentials

	if !b.Settings.EnableHTTPRateLimiter {
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

	if s.EnableSqliteDatabase {
		b.Settings.EnableSqliteDatabase = s.EnableSqliteDatabase
	} else {
		b.Settings.EnableSqliteDatabase = b.Config.Databases.Sqlite3.Enabled
	}

	if s.EnablePostgresDatabase {
		b.Settings.EnablePostgresDatabase = s.EnablePostgresDatabase
	} else {
		b.Settings.EnablePostgresDatabase = b.Config.Databases.Postgres.Enabled
	}

	b.Settings.InsertNewDatabaseClient = s.InsertNewDatabaseClient

	b.Settings.DatabaseDirectory = s.DatabaseDirectory

	b.Settings.SqliteDatabasePath = common.GetDefaultSQLitePath()
	if b.Config.Databases.Sqlite3.PathToDb != "" {
		b.Settings.SqliteDatabasePath = b.Config.Databases.Sqlite3.PathToDb
	}

	if s.SqliteDatabasePath != common.GetDefaultSQLitePath() {
		b.Settings.SqliteDatabasePath = s.SqliteDatabasePath
	}

	if s.DatabaseConnectionHostName != "" {
		b.Settings.DatabaseConnectionHostName = s.DatabaseConnectionHostName
	} else {
		b.Settings.DatabaseConnectionHostName = b.Config.Databases.Postgres.Host
	}

	if s.DatabaseConnectionUserName != "" {
		b.Settings.DatabaseConnectionUserName = s.DatabaseConnectionUserName
	} else {
		b.Settings.DatabaseConnectionUserName = b.Config.Databases.Postgres.Username
	}

	if s.DatabaseConnectionPassword != "" {
		b.Settings.DatabaseConnectionPassword = s.DatabaseConnectionPassword
	} else {
		b.Settings.DatabaseConnectionPassword = b.Config.Databases.Postgres.Password
	}

	if s.DatabaseConnectionName != "" {
		b.Settings.DatabaseConnectionName = s.DatabaseConnectionName
	} else {
		b.Settings.DatabaseConnectionName = b.Config.Databases.Postgres.DatabaseName
	}

	if s.DatabaseConnectionPort != "" {
		b.Settings.DatabaseConnectionPort = s.DatabaseConnectionPort
	} else {
		b.Settings.DatabaseConnectionPort = b.Config.Databases.Postgres.Port
	}

	if s.DatabaseConnectionSSLMode != "" {
		b.Settings.DatabaseConnectionSSLMode = s.DatabaseConnectionSSLMode
	} else {
		b.Settings.DatabaseConnectionSSLMode = b.Config.Databases.Postgres.SSLMode
	}
}

// PrintSettings returns the engine settings
func PrintSettings(s *Settings) {
	log.Debugln()
	log.Debugf("ENGINE SETTINGS")
	log.Debugf("- CORE SETTINGS:")
	log.Debugf("\t Verbose mode: %v", s.Verbose)
	log.Debugf("\t Enable dry run mode: %v", s.EnableDryRun)
	log.Debugf("\t Enable all exchanges: %v", s.EnableAllExchanges)
	log.Debugf("\t Enable all pairs: %v", s.EnableAllPairs)
	log.Debugf("\t Enable coinmarketcap analaysis: %v", s.EnableCoinmarketcapAnalysis)
	log.Debugf("\t Enable portfolio watcher: %v", s.EnablePortfolioWatcher)
	log.Debugf("\t Enable gPRC: %v", s.EnableGRPC)
	log.Debugf("\t Enable gRPC Proxy: %v", s.EnableGRPCProxy)
	log.Debugf("\t Enable websocket RPC: %v", s.EnableWebsocketRPC)
	log.Debugf("\t Enable deprecated RPC: %v", s.EnableDeprecatedRPC)
	log.Debugf("\t Enable comms relayer: %v", s.EnableCommsRelayer)
	log.Debugf("\t Enable event manager: %v", s.EnableEventManager)
	log.Debugf("\t Event manager sleep delay: %v", s.EventManagerDelay)
	log.Debugf("\t Enable ticker routine: %v", s.EnableTickerRoutine)
	log.Debugf("\t Enable orderbook routine: %v", s.EnableOrderbookRoutine)
	log.Debugf("\t Enable websocket routine: %v\n", s.EnableWebsocketRoutine)
	log.Debugf("\t Enable NTP client: %v", s.EnableNTPClient)
	log.Debugf("- FOREX SETTINGS:")
	log.Debugf("\t Enable currency conveter: %v", s.EnableCurrencyConverter)
	log.Debugf("\t Enable currency layer: %v", s.EnableCurrencyLayer)
	log.Debugf("\t Enable fixer: %v", s.EnableFixer)
	log.Debugf("\t Enable OpenExchangeRates: %v", s.EnableOpenExchangeRates)
	log.Debugf("- EXCHANGE SETTINGS:")
	log.Debugf("\t Enable exchange auto pair updates: %v", s.EnableExchangeAutoPairUpdates)
	log.Debugf("\t Disable all exchange auto pair updates: %v", s.DisableExchangeAutoPairUpdates)
	log.Debugf("\t Enable exchange websocket support: %v", s.EnableExchangeWebsocketSupport)
	log.Debugf("\t Enable exchange verbose mode: %v", s.EnableExchangeVerbose)
	log.Debugf("\t Enable exchange HTTP rate limiter: %v", s.EnableHTTPRateLimiter)
	log.Debugf("\t Exchange max HTTP request jobs: %v", s.MaxHTTPRequestJobsLimit)
	log.Debugf("\t Exchange HTTP request timeout retry amount: %v", s.RequestTimeoutRetryAttempts)
	log.Debugf("\t Exchange HTTP timeout: %v", s.ExchangeHTTPTimeout)
	log.Debugf("\t Exchange HTTP user agent: %v", s.ExchangeHTTPUserAgent)
	log.Debugf("\t Exchange HTTP proxy: %v\n", s.ExchangeHTTPProxy)
	log.Debugf("- COMMON SETTINGS:")
	log.Debugf("\t Global HTTP timeout: %v", s.GlobalHTTPTimeout)
	log.Debugf("\t Global HTTP user agent: %v", s.GlobalHTTPUserAgent)
	log.Debugf("\t Global HTTP proxy: %v", s.ExchangeHTTPProxy)
	log.Debugf("- DATABASE SETTINGS:")
	log.Debugf("\t Enable SQLite3 database: %v", s.EnableSqliteDatabase)
	log.Debugf("\t Enable PostgreSQL database: %v", s.EnablePostgresDatabase)
	log.Debugf("\t Database directory: %v", s.DatabaseDirectory)
	log.Debugf("- DATABASE CONNECTION DETAILS:")
	log.Debugf("\t SQLite3 database path: %v", s.SqliteDatabasePath)
	log.Debugf("\t Database connection hostname: %v", s.DatabaseConnectionHostName)
	log.Debugf("\t Database connection username: %v", s.DatabaseConnectionUserName)
	log.Debugf("\t Database connection name: %v", s.DatabaseConnectionName)
	log.Debugf("\t Database connection port: %v", s.DatabaseConnectionPort)
	log.Debugf("\t Database connection SSLMode: %v", s.DatabaseConnectionSSLMode)
	log.Debugln()
}

// Start starts the engine
func (e *Engine) Start() {
	if e == nil {
		log.Fatal("Engine instance is nil")
	}

	// Sets up internet connectivity monitor
	e.Connectivity = connchecker.New(e.Config.ConnectionMonitor.DNSList,
		e.Config.ConnectionMonitor.PublicDomainList,
		e.Config.ConnectionMonitor.CheckInterval)

	if e.Settings.EnableNTPClient {
		if e.Config.NTPClient.Level != -1 {
			e.Config.CheckNTPConfig()
			NTPTime, errNTP := ntpclient.NTPClient(e.Config.NTPClient.Pool)
			currentTime := time.Now()
			if errNTP != nil {
				log.Warnf("NTPClient failed to create: %v", errNTP)
			} else {
				NTPcurrentTimeDifference := NTPTime.Sub(currentTime)
				configNTPTime := *e.Config.NTPClient.AllowedDifference
				configNTPNegativeTime := (*e.Config.NTPClient.AllowedNegativeDifference - (*e.Config.NTPClient.AllowedNegativeDifference * 2))
				if NTPcurrentTimeDifference > configNTPTime || NTPcurrentTimeDifference < configNTPNegativeTime {
					log.Warnf("Time out of sync (NTP): %v | (time.Now()): %v | (Difference): %v | (Allowed): +%v / %v", NTPTime, currentTime, NTPcurrentTimeDifference, configNTPTime, configNTPNegativeTime)
					if e.Config.NTPClient.Level == 0 {
						disable, errNTP := e.Config.DisableNTPCheck(os.Stdin)
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

	e.Uptime = time.Now()
	log.Debugf("Bot '%s' started.\n", e.Config.Name)

	enabledExchanges := e.Config.CountEnabledExchanges()
	if e.Settings.EnableAllExchanges {
		enabledExchanges = len(e.Config.Exchanges)
	}

	log.Debugln()
	log.Debugln("EXCHANGE COVERAGE")
	log.Debugf("\t Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(e.Config.Exchanges), enabledExchanges)

	if e.Settings.ExchangePurgeCredentials {
		log.Debugln("Purging exchange API credentials.")
		e.Config.PurgeExchangeAPICredentials()
	}

	log.Debugln("Setting up exchanges..")
	SetupExchanges()
	if len(e.Exchanges) == 0 {
		log.Fatalf("No exchanges were able to be loaded. Exiting")
	}

	// Database start
	var databaseConnected bool
	if e.Settings.EnableSqliteDatabase || e.Settings.EnablePostgresDatabase {
		if e.Settings.EnableSqliteDatabase && e.Settings.EnablePostgresDatabase {
			log.Fatal("Can only run one database at a time, please check config and flags")
		}

		if e.Settings.EnablePostgresDatabase {
			e.DB = database.GetPostgresInstance()
			err := e.DB.Setup(&base.ConnDetails{Verbose: e.Settings.Verbose,
				DirectoryPath: e.Settings.DatabaseDirectory,
				Host:          e.Settings.DatabaseConnectionHostName,
				User:          e.Settings.DatabaseConnectionUserName,
				Pass:          e.Settings.DatabaseConnectionPassword,
				DBName:        e.Settings.DatabaseConnectionName,
				Port:          e.Settings.DatabaseConnectionPort,
				SSLMode:       e.Settings.DatabaseConnectionSSLMode,
				MemCacheSize:  e.Config.Databases.MemoryAllocationInBytes,
			})
			if err != nil {
				log.Fatal("PostgreSQL setup error", err)
			}
		}

		if e.Settings.EnableSqliteDatabase {
			e.DB = database.GetSQLite3Instance()
			err := e.DB.Setup(&base.ConnDetails{
				DirectoryPath: e.Settings.DatabaseDirectory,
				SQLPath:       e.Settings.SqliteDatabasePath,
				Verbose:       e.Settings.Verbose,
				MemCacheSize:  e.Config.Databases.MemoryAllocationInBytes,
			})
			if err != nil {
				log.Fatal("SQLite3 setup error", err)
			}
		}

		err := e.DB.Connect()
		if err != nil {
			disconnectErr := e.DB.Disconnect()
			if disconnectErr != nil {
				log.Error(disconnectErr)
			}
			log.Error("Database connection error", err)
		}

		if e.DB.IsConnected() {
			if e.Settings.Verbose {
				log.Debugf("Bot is now connected to a %s database",
					e.DB.GetName())
			}
			databaseConnected = true
		}
	} else {
		// Dummy instance so DB is not nil
		e.DB = database.GetSQLite3Instance()
	}

	if e.Settings.EnableCommsRelayer {
		log.Debugln("Starting communication mediums..")
		commsCfg := e.Config.GetCommunicationsConfig()
		e.CommsRelayer = communications.NewComm(&commsCfg)
		e.CommsRelayer.GetEnabledCommunicationMediums()
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
		log.Warn("currency updater system failed to start", err)
	}

	e.Portfolio = &portfolio.Portfolio
	e.Portfolio.Seed(e.Config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)

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

	if e.Settings.EnablePortfolioWatcher {
		go portfolio.StartPortfolioWatcher()
	}

	/*
		exchangeSyncCfg := CurrencyPairSyncerConfig{
			SyncTicker:       true,
			SyncOrderbook:    true,
			SyncContinuously: true,
			NumWorkers:       15,
		}


			e.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(exchangeSyncCfg)
			if err != nil {
				log.Warnf("Unable to initialise exchange currency pair syncer. Err: %s", err)
			} else {
				e.ExchangeCurrencyPairManager.Start()
			}
	*/

	go StartOrderManagerRoutine()

	if e.Settings.EnableTickerRoutine {
		go TickerUpdaterRoutine()
	}
	/*

		if e.Settings.EnableOrderbookRoutine {
			go OrderbookUpdaterRoutine()
		}

		if e.Settings.EnableWebsocketRoutine {
			go WebsocketRoutine()
		}
	*/

	if e.Settings.EnableEventManager {
		go events.EventManger()
	}

	if databaseConnected {
		go PlatformTradeUpdaterRoutine()
	}

	<-e.Shutdown
	e.Stop()
}

// Stop correctly shuts down engine saving configuration files
func (e *Engine) Stop() {
	log.Debugln("Engine shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		e.Config.Portfolio = portfolio.Portfolio
	}

	if e.DB != nil {
		if e.DB.IsConnected() {
			err := e.DB.Disconnect()
			if err != nil {
				log.Debug("Unable to disconnect from database", err)
			} else {
				log.Debug("Succesfully shutdown database.")
			}
		}
	}

	if !e.Settings.EnableDryRun {
		err := e.Config.SaveConfig(e.Settings.ConfigFile)

		if err != nil {
			log.Error("Unable to save config.")
		} else {
			log.Debugln("Config file saved successfully.")
		}
	}
	log.Debugln("Exiting.")
	log.CloseLogFile()
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
		log.Debugf("Captured %v, shutdown requested.", sig)
		e.Shutdown <- true
	}()
}
