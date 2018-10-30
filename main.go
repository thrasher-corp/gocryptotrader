package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/core"
	"github.com/thrasher-/gocryptotrader/engine"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	log "github.com/thrasher-/gocryptotrader/logger"
)

func main() {
	defaultPath, err := config.GetFilePath("")
	if err != nil {
		log.Fatal(err)
	}

	// Handle flags
	var settings engine.Settings
	versionFlag := flag.Bool("version", false, "retrieves current GoCryptoTrader version")

	// Core settings
	flag.StringVar(&settings.ConfigFile, "config", defaultPath, "config file to load")
	flag.StringVar(&settings.DataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	flag.IntVar(&settings.GoMaxProcs, "gomaxprocs", runtime.NumCPU(), "sets the runtime GOMAXPROCS value")
	flag.BoolVar(&settings.EnableDryRun, "dryrun", false, "dry runs bot, doesn't save config file")
	flag.BoolVar(&settings.EnableAllExchanges, "enableallexchanges", false, "enables all exchanges")
	flag.BoolVar(&settings.EnableAllPairs, "enableallpairs", false, "enables all pairs for enabled exchanges")
	flag.BoolVar(&settings.EnablePortfolioWatcher, "portfoliowatcher", true, "enables the portfolio watcher")
	flag.BoolVar(&settings.EnableGRPC, "grpc", true, "enables the grpc server")
	flag.BoolVar(&settings.EnableGRPCProxy, "grpcproxy", true, "enables the grpc proxy server")
	flag.BoolVar(&settings.EnableWebsocketRPC, "websocketrpc", true, "enables the websocket RPC server")
	flag.BoolVar(&settings.EnableDeprecatedRPC, "deprecatedrpc", true, "enables the deprecated RPC server")
	flag.BoolVar(&settings.EnableCommsRelayer, "enablecommsrelayer", true, "enables available communications relayer")
	flag.BoolVar(&settings.Verbose, "verbose", false, "increases logging verbosity for GoCryptoTrader")
	flag.BoolVar(&settings.EnableTickerRoutine, "tickerroutine", true, "enables the ticker routine for all loaded exchanges")
	flag.BoolVar(&settings.EnableOrderbookRoutine, "orderbookroutine", true, "enables the orderbook routine for all loaded exchanges")
	flag.BoolVar(&settings.EnableWebsocketRoutine, "websocketroutine", true, "enables the websocket routine for all loaded exchanges")
	flag.BoolVar(&settings.EnableCoinmarketcapAnalysis, "coinmarketcap", false, "overrides config and runs currency analysis")
	flag.BoolVar(&settings.EnableEventManager, "enableeventmanager", true, "enables the event manager")
	flag.DurationVar(&settings.EventManagerDelay, "eventmanagerdelay", time.Duration(0), "sets the event managers sleep delay between event checking")
	flag.BoolVar(&settings.EnableNTPClient, "ntpclient", true, "enables the NTP client to check system clock drift")

	// Forex provider settings
	flag.BoolVar(&settings.EnableCurrencyConverter, "currencyconverter", false, "overrides config and sets up foreign exchange Currency Converter")
	flag.BoolVar(&settings.EnableCurrencyLayer, "currencylayer", false, "overrides config and sets up foreign exchange Currency Layer")
	flag.BoolVar(&settings.EnableFixer, "fixer", false, "overrides config and sets up foreign exchange Fixer.io")
	flag.BoolVar(&settings.EnableOpenExchangeRates, "openexchangerates", false, "overrides config and sets up foreign exchange Open Exchange Rates")

	// Exchange tuning settings
	flag.BoolVar(&settings.EnableExchangeAutoPairUpdates, "exchangeautopairupdates", true, "enables automatic available currency pair updates for supported exchanges")
	flag.BoolVar(&settings.DisableExchangeAutoPairUpdates, "exchangedisableautopairupdates", false, "disables exchange auto pair updates")
	flag.BoolVar(&settings.EnableExchangeWebsocketSupport, "exchangewebsocketsupport", true, "enables Websocket support for exchanges")
	flag.BoolVar(&settings.EnableExchangeRESTSupport, "exchangerestsupport", true, "enables REST support for exchanges")
	flag.BoolVar(&settings.EnableExchangeVerbose, "exchangeverbose", false, "increases exchange logging verbosity")
	flag.BoolVar(&settings.ExchangePurgeCredentials, "exchangepurgecredentials", false, "purges the stored exchange API credentials")
	flag.BoolVar(&settings.EnableHTTPRateLimiter, "ratelimiter", true, "enables the rate limiter for HTTP requests")
	flag.IntVar(&settings.MaxHTTPRequestJobsLimit, "maxhttprequestjobslimit", request.DefaultMaxRequestJobs, "sets the max amount of jobs the HTTP request package stores")
	flag.IntVar(&settings.RequestTimeoutRetryAttempts, "exchangehttptimeoutretryattempts", request.DefaultTimeoutRetryAttempts, "sets the amount of retry attempts after a HTTP request times out")
	flag.DurationVar(&settings.ExchangeHTTPTimeout, "exchangehttptimeout", time.Duration(0), "sets the exchangs HTTP timeout value for HTTP requests")
	flag.StringVar(&settings.ExchangeHTTPUserAgent, "exchangehttpuseragent", "", "sets the exchanges HTTP user agent")
	flag.StringVar(&settings.ExchangeHTTPProxy, "exchangehttpproxy", "", "sets the exchanges HTTP proxy server")

	// Common tuning settings
	flag.DurationVar(&settings.GlobalHTTPTimeout, "globalhttptimeout", time.Duration(0), "sets common HTTP timeout value for HTTP requests")
	flag.StringVar(&settings.GlobalHTTPUserAgent, "globalhttpuseragent", "", "sets the common HTTP client's user agent")
	flag.StringVar(&settings.GlobalHTTPProxy, "globalhttpproxy", "", "sets the common HTTP client's proxy server")

	flag.Parse()

	if *versionFlag {
		fmt.Print(core.Version(true))
		os.Exit(0)
	}

	fmt.Println(core.Banner)
	fmt.Println(core.Version(false))

	engine.Bot, err = engine.NewFromSettings(&settings)
	if engine.Bot == nil || err != nil {
		log.Fatalf("Unable to initialise bot engine. Err: %s", err)
	}

	engine.PrintSettings(&engine.Bot.Settings)
	engine.Bot.Start()
}
