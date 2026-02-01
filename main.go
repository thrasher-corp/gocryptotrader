package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/gctscript"
	gctscriptVM "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

func main() {
	// Handle flags
	var settings engine.Settings
	versionFlag := flag.Bool("version", false, "retrieves current GoCryptoTrader version")

	// Core settings
	flag.StringVar(&settings.ConfigFile, "config", config.DefaultFilePath(), "config file to load")
	flag.StringVar(&settings.DataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	flag.IntVar(&settings.GoMaxProcs, "gomaxprocs", runtime.GOMAXPROCS(-1), "sets the runtime GOMAXPROCS value")
	flag.BoolVar(&settings.EnableDryRun, "dryrun", false, "dry runs bot, doesn't save config file")
	flag.BoolVar(&settings.EnableAllExchanges, "enableallexchanges", false, "enables all exchanges")
	flag.BoolVar(&settings.EnableAllPairs, "enableallpairs", false, "enables all pairs for enabled exchanges")
	flag.BoolVar(&settings.EnablePortfolioManager, "portfoliomanager", true, "enables the portfolio manager")
	flag.BoolVar(&settings.EnableDataHistoryManager, "datahistorymanager", false, "enables the data history manager")
	flag.DurationVar(&settings.PortfolioManagerDelay, "portfoliomanagerdelay", 0, "sets the portfolio managers sleep delay between updates")
	flag.BoolVar(&settings.EnableGRPC, "grpc", true, "enables the grpc server")
	flag.BoolVar(&settings.EnableGRPCProxy, "grpcproxy", false, "enables the grpc proxy server")
	flag.BoolVar(&settings.EnableGRPCShutdown, "grpcshutdown", false, "enables gRPC bot instance shutdown functionality")
	flag.BoolVar(&settings.EnableCommsRelayer, "enablecommsrelayer", true, "enables available communications relayer")
	flag.BoolVar(&settings.Verbose, "verbose", false, "increases logging verbosity for GoCryptoTrader")
	flag.BoolVar(&settings.EnableFuturesTracking, "enablefuturestracking", true, "tracks futures orders PNL is supported by the exchange")
	flag.BoolVar(&settings.EnableExchangeSyncManager, "syncmanager", false, "enables to exchange sync manager")
	flag.BoolVar(&settings.EnableWebsocketRoutine, "websocketroutine", true, "enables the websocket routine for all loaded exchanges")
	flag.BoolVar(&settings.EnableCoinmarketcapAnalysis, "coinmarketcap", false, "overrides config and runs currency analysis")
	flag.BoolVar(&settings.EnableEventManager, "eventmanager", true, "enables the event manager")
	flag.BoolVar(&settings.EnableOrderManager, "ordermanager", true, "enables the order manager")
	flag.BoolVar(&settings.EnableDepositAddressManager, "depositaddressmanager", true, "enables the deposit address manager")
	flag.BoolVar(&settings.EnableConnectivityMonitor, "connectivitymonitor", true, "enables the connectivity monitor")
	flag.BoolVar(&settings.EnableDatabaseManager, "databasemanager", true, "enables database manager")
	flag.BoolVar(&settings.EnableGCTScriptManager, "gctscriptmanager", true, "enables gctscript manager")
	flag.DurationVar(&settings.EventManagerDelay, "eventmanagerdelay", 0, "sets the event managers sleep delay between event checking")
	flag.BoolVar(&settings.EnableNTPClient, "ntpclient", true, "enables the NTP client to check system clock drift")
	flag.BoolVar(&settings.EnableDispatcher, "dispatch", true, "enables the dispatch system")
	flag.BoolVar(&settings.EnableCurrencyStateManager, "currencystatemanager", true, "enables the currency state manager")
	flag.IntVar(&settings.DispatchMaxWorkerAmount, "dispatchworkers", dispatch.DefaultMaxWorkers, "sets the dispatch package max worker generation limit")
	flag.IntVar(&settings.DispatchJobsLimit, "dispatchjobslimit", dispatch.DefaultJobsLimit, "sets the dispatch package max jobs limit")

	// Exchange syncer settings
	flag.BoolVar(&settings.EnableTickerSyncing, "tickersync", false, "enables ticker syncing for all enabled exchanges, overriding false config value")
	flag.BoolVar(&settings.EnableOrderbookSyncing, "orderbooksync", false, "enables orderbook syncing for all enabled exchanges, overriding false config value")
	flag.BoolVar(&settings.EnableTradeSyncing, "tradesync", false, "enables trade syncing for all enabled exchanges, overriding false config value")
	flag.IntVar(&settings.SyncWorkersCount, "syncworkers", config.DefaultSyncerWorkers, "the amount of workers (goroutines) to use for syncing exchange data")
	flag.BoolVar(&settings.SyncContinuously, "synccontinuously", false, "whether to sync exchange data continuously (ticker, orderbook and trade history info), overriding false config value")
	flag.DurationVar(&settings.SyncTimeoutREST, "synctimeoutrest", config.DefaultSyncerTimeoutREST,
		"the amount of time before the syncer will switch from rest protocol to the streaming protocol (e.g. from REST to websocket)")
	flag.DurationVar(&settings.SyncTimeoutWebsocket, "synctimeoutwebsocket", config.DefaultSyncerTimeoutWebsocket,
		"the amount of time before the syncer will switch from the websocket protocol to REST protocol (e.g. from websocket to REST)")

	// Forex provider settings
	flag.BoolVar(&settings.EnableCurrencyConverter, "currencyconverter", false, "overrides config and sets up foreign exchange Currency Converter")
	flag.BoolVar(&settings.EnableCurrencyLayer, "currencylayer", false, "overrides config and sets up foreign exchange Currency Layer")
	flag.BoolVar(&settings.EnableExchangeRates, "exchangerates", false, "overrides config and sets up foreign exchange exchangeratesapi.io")
	flag.BoolVar(&settings.EnableFixer, "fixer", false, "overrides config and sets up foreign exchange Fixer.io")
	flag.BoolVar(&settings.EnableOpenExchangeRates, "openexchangerates", false, "overrides config and sets up foreign exchange Open Exchange Rates")

	// Exchange tuning settings
	flag.BoolVar(&settings.EnableExchangeAutoPairUpdates, "exchangeautopairupdates", false, "enables automatic available currency pair updates for supported exchanges")
	flag.BoolVar(&settings.DisableExchangeAutoPairUpdates, "exchangedisableautopairupdates", false, "disables exchange auto pair updates")
	flag.BoolVar(&settings.EnableExchangeWebsocketSupport, "exchangewebsocketsupport", false, "enables Websocket support for exchanges")
	flag.BoolVar(&settings.EnableExchangeRESTSupport, "exchangerestsupport", true, "enables REST support for exchanges")
	flag.BoolVar(&settings.EnableExchangeVerbose, "exchangeverbose", false, "increases exchange logging verbosity")
	flag.BoolVar(&settings.ExchangePurgeCredentials, "exchangepurgecredentials", false, "purges the stored exchange API credentials")
	flag.BoolVar(&settings.EnableExchangeHTTPRateLimiter, "ratelimiter", true, "enables the rate limiter for HTTP requests")
	flag.IntVar(&settings.RequestMaxRetryAttempts, "httpmaxretryattempts", request.DefaultMaxRetryAttempts, "sets the number of retry attempts after a retryable HTTP failure")
	flag.DurationVar(&settings.HTTPTimeout, "httptimeout", 0, "sets the HTTP timeout value for HTTP requests")
	flag.StringVar(&settings.HTTPUserAgent, "httpuseragent", "", "sets the HTTP user agent")
	flag.StringVar(&settings.HTTPProxy, "httpproxy", "", "sets the HTTP proxy server")
	flag.BoolVar(&settings.EnableExchangeHTTPDebugging, "exchangehttpdebugging", false, "sets the exchanges HTTP debugging")
	flag.DurationVar(&settings.TradeBufferProcessingInterval, "tradeprocessinginterval", trade.DefaultProcessorIntervalTime, "sets the interval to save trade buffer data to the database")
	flag.IntVar(&settings.AlertSystemPreAllocationCommsBuffer, "alertbuffer", alert.PreAllocCommsDefaultBuffer, "sets the size of the pre-allocation communications buffer")
	flag.DurationVar(&settings.ExchangeShutdownTimeout, "exchangeshutdowntimeout", time.Second*10, "sets the maximum amount of time the program will wait for an exchange to shut down gracefully")
	flag.StringVar(&settings.Exchanges, "exchanges", "", "sets a comma-separated list of exchanges to load. If left empty, all enabled exchanges will be loaded from the config file")

	// Common tuning settings
	flag.DurationVar(&settings.GlobalHTTPTimeout, "globalhttptimeout", 0, "sets common HTTP timeout value for HTTP requests")
	flag.StringVar(&settings.GlobalHTTPUserAgent, "globalhttpuseragent", "", "sets the common HTTP client's user agent")
	flag.StringVar(&settings.GlobalHTTPProxy, "globalhttpproxy", "", "sets the common HTTP client's proxy server")

	// GCTScript tuning settings
	flag.Uint64Var(&settings.MaxVirtualMachines, "maxvirtualmachines", gctscriptVM.DefaultMaxVirtualMachines, "set max virtual machines that can load")

	// Withdraw Cache tuning settings
	flag.Uint64Var(&settings.WithdrawCacheSize, "withdrawcachesize", withdraw.CacheSize, "set cache size for withdrawal requests")

	flag.Parse()

	if *versionFlag {
		fmt.Print(core.Version(true))
		os.Exit(0)
	}

	fmt.Print(core.Banner)
	fmt.Println(core.Version(false))

	var err error
	settings.CheckParamInteraction = true

	// collect flags
	flagSet := make(map[string]bool)
	// Stores the set flags
	flag.Visit(func(f *flag.Flag) { flagSet[f.Name] = true })
	if !flagSet["config"] {
		// If config file is not explicitly set, fall back to default path resolution
		settings.ConfigFile = ""
	}

	settings.Shutdown = make(chan struct{})
	engine.Bot, err = engine.NewFromSettings(&settings, flagSet)
	if engine.Bot == nil || err != nil {
		log.Fatalf("Unable to initialise bot engine. Error: %s\n", err)
	}
	config.SetConfig(engine.Bot.Config)

	gctscript.Setup()

	gctlog.Infof(gctlog.Global, "JSON encoding is set to package %q", json.Implementation)

	engine.Bot.Settings.PrintLoadedSettings()

	if err = engine.Bot.Start(); err != nil {
		errClose := gctlog.CloseLogger()
		if errClose != nil {
			log.Printf("Unable to close logger. Error: %s\n", errClose)
		}
		log.Fatalf("Unable to start bot engine. Error: %s\n", err)
	}

	go waitForInterrupt(settings.Shutdown)
	<-settings.Shutdown
	engine.Bot.Stop()
}

func waitForInterrupt(waiter chan<- struct{}) {
	interrupt := <-signaler.WaitForInterrupt()
	gctlog.Infof(gctlog.Global, "Captured %v, shutdown requested.\n", interrupt)
	waiter <- struct{}{}
}
