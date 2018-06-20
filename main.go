package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/thrasher-/gocryptotrader/communications"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/binance"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	config     *config.Config
	portfolio  *portfolio.Base
	exchanges  []exchange.IBotExchange
	comms      *communications.Communications
	shutdown   chan bool
	dryRun     bool
	configFile string
}

const banner = `
   ______        ______                     __        ______                  __
  / ____/____   / ____/_____ __  __ ____   / /_ ____ /_  __/_____ ______ ____/ /___   _____
 / / __ / __ \ / /    / ___// / / // __ \ / __// __ \ / /  / ___// __  // __  // _ \ / ___/
/ /_/ // /_/ // /___ / /   / /_/ // /_/ // /_ / /_/ // /  / /   / /_/ // /_/ //  __// /
\____/ \____/ \____//_/    \__, // .___/ \__/ \____//_/  /_/    \__,_/ \__,_/ \___//_/
                          /____//_/
`

var bot Bot

// getDefaultConfig 获取默认配置
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "binance",
		Enabled:                 true,
		Verbose:                 true,
		Websocket:               false,
		BaseAsset:               "eth",
		QuoteAsset:              "usdt",
		UseSandbox:              false,
		RESTPollingDelay:        10,
		HTTPTimeout:             15000000000,
		AuthenticatedAPISupport: true,
		APIKey:                  "",
		APISecret:               "",
		ClientID:                "",
		AvailablePairs:          "BTC-USDT,BCH-USDT",
		EnabledPairs:            "BTC-USDT",
		BaseCurrencies:          "USD",
		AssetTypes:              "SPOT",
		SupportsAutoPairUpdates: false,
		ConfigCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: true,
			Delimiter: "-",
		},
		RequestCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: true,
		},
	}
}

// getExchangeInfo 获取交易所的规则和交易对信息
func getExchangeInfo(b binance.Binance) {
	info, err := b.GetExchangeInfo()
	if err != nil {
		fmt.Println(err)
	} else {
		b, _ := json.Marshal(info)
		fmt.Printf("%s \n", b)
	}
}

// getOrderBook 获取交易深度
func getOrderBook(b binance.Binance) {
	model, err := b.GetOrderBook(b.GetSymbol(), 10)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("--------获取交易深度--------")
		b, _ := json.Marshal(model)
		fmt.Printf("%s \n", b)
	}
}

// getRecentTrades 获取最近的交易记录
func getRecentTrades(b binance.Binance) {
	list, err := b.GetRecentTrades(b.GetSymbol(), 100)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("--------获取最近的交易记录--------")
		for _, v := range list {
			b, _ := json.Marshal(v)
			fmt.Printf("%s \n", b)
		}
	}
}

// getCandleStickData 获取 k 线数据
func getCandleStickData(b binance.Binance) {
	list, err := b.GetCandleStickData(b.GetSymbol(), binance.BinanceIntervalFiveMinutes, 10)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("--------获取 k 线数据--------")
		for _, v := range list {
			b, _ := json.Marshal(v)
			fmt.Printf("%s \n", b)
		}
	}
}

// getLatestSpotPrice 获取最新价格
func getLatestSpotPrice(b binance.Binance) {
	model, err := b.GetLatestSpotPrice(b.GetSymbol())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("--------获取最新价格--------")
		b, _ := json.Marshal(model)
		fmt.Printf("%s \n", b)
	}
}

// newOrder 新订单
func newOrder(b binance.Binance) {
	res, err := b.NewOrder(binance.NewOrderRequest{
		Symbol:      b.GetSymbol(),
		Side:        binance.BinanceRequestParamsSideSell,
		TradeType:   binance.BinanceRequestParamsOrderLimit,
		TimeInForce: binance.BinanceRequestParamsTimeGTC,
		Quantity:    0.01,
		Price:       1536.1,
	})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("--------下订单--------")
		b, _ := json.Marshal(res)
		fmt.Printf("%s \n", b)
	}
}

// cancelOrder 取消订单
func cancelOrder(b binance.Binance, orderID int64) {
	res, err := b.CancelOrder(b.GetSymbol(), orderID, "")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("--------取消订单--------")
		b, _ := json.Marshal(res)
		fmt.Printf("%s \n", b)
	}
}

func main() {
	fmt.Println(time.Now())
	exchange := binance.Binance{}
	defaultConfig := getDefaultConfig()
	exchange.SetDefaults()
	fmt.Println("----------setup-------")
	exchange.Setup(defaultConfig)

	//获取交易所的规则和交易对信息
	// getExchangeInfo(exchange)

	//获取交易深度
	// getOrderBook(exchange)

	//获取最近的交易记录
	// getRecentTrades(exchange)

	//获取 k 线数据
	// getCandleStickData(exchange)

	//获取最新价格
	// getLatestSpotPrice(exchange)

	//新订单
	// newOrder(exchange)

	//取消订单
	// cancelOrder(exchange, 82584683)

	fmt.Println(exchange.GetAccount())

	// fmt.Println(exchange.GetSymbol())

	// bot.shutdown = make(chan bool)
	// HandleInterrupt()

	// defaultPath, err := config.GetFilePath("")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// //Handle flags
	// flag.StringVar(&bot.configFile, "config", defaultPath, "config file to load")
	// dryrun := flag.Bool("dryrun", false, "dry runs bot, doesn't save config file")
	// version := flag.Bool("version", false, "retrieves current GoCryptoTrader version")
	// flag.Parse()

	// if *version {
	// 	fmt.Printf(BuildVersion(true))
	// 	os.Exit(0)
	// }

	// if *dryrun {
	// 	bot.dryRun = true
	// }

	// bot.config = &config.Cfg
	// fmt.Println(banner)
	// fmt.Println(BuildVersion(false))
	// log.Printf("Loading config file %s..\n", bot.configFile)

	// err = bot.config.LoadConfig(bot.configFile)
	// if err != nil {
	// 	log.Fatalf("Failed to load config. Err: %s", err)
	// }

	// AdjustGoMaxProcs()
	// log.Printf("Bot '%s' started.\n", bot.config.Name)
	// log.Printf("Bot dry run mode: %v.\n", common.IsEnabled(bot.dryRun))

	// log.Printf("Available Exchanges: %d. Enabled Exchanges: %d.\n",
	// 	len(bot.config.Exchanges),
	// 	bot.config.CountEnabledExchanges())

	// common.HTTPClient = common.NewHTTPClientWithTimeout(bot.config.GlobalHTTPTimeout)
	// log.Printf("Global HTTP request timeout: %v.\n", common.HTTPClient.Timeout)

	// SetupExchanges()
	// if len(bot.exchanges) == 0 {
	// 	log.Fatalf("No exchanges were able to be loaded. Exiting")
	// }

	// log.Println("Starting communication mediums..")
	// bot.comms = communications.NewComm(bot.config.GetCommunicationsConfig())
	// bot.comms.GetEnabledCommunicationMediums()

	// log.Printf("Fiat display currency: %s.", bot.config.Currency.FiatDisplayCurrency)
	// currency.BaseCurrency = bot.config.Currency.FiatDisplayCurrency
	// currency.FXProviders = forexprovider.StartFXService(bot.config.GetCurrencyConfig().ForexProviders)
	// log.Printf("Primary forex conversion provider: %s.\n", bot.config.GetPrimaryForexProvider())
	// err = bot.config.RetrieveConfigCurrencyPairs(true)
	// if err != nil {
	// 	log.Fatalf("Failed to retrieve config currency pairs. Error: %s", err)
	// }
	// log.Println("Successfully retrieved config currencies.")
	// log.Println("Fetching currency data from forex provider..")
	// err = currency.SeedCurrencyData(common.JoinStrings(currency.FiatCurrencies, ","))
	// if err != nil {
	// 	log.Fatalf("Unable to fetch forex data. Error: %s", err)
	// }

	// bot.portfolio = &portfolio.Portfolio
	// bot.portfolio.SeedPortfolio(bot.config.Portfolio)
	// SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)

	// go portfolio.StartPortfolioWatcher()
	// go TickerUpdaterRoutine()
	// go OrderbookUpdaterRoutine()

	// if bot.config.Webserver.Enabled {
	// 	listenAddr := bot.config.Webserver.ListenAddress
	// 	log.Printf(
	// 		"HTTP Webserver support enabled. Listen URL: http://%s:%d/\n",
	// 		common.ExtractHost(listenAddr), common.ExtractPort(listenAddr),
	// 	)

	// 	router := NewRouter(bot.exchanges)
	// 	go func() {
	// 		err = http.ListenAndServe(listenAddr, router)
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 	}()

	// 	log.Println("HTTP Webserver started successfully.")
	// 	log.Println("Starting websocket handler.")
	// 	StartWebsocketHandler()
	// } else {
	// 	log.Println("HTTP RESTful Webserver support disabled.")
	// }

	// <-bot.shutdown
	// Shutdown()
}

// AdjustGoMaxProcs adjusts the maximum processes that the CPU can handle.
func AdjustGoMaxProcs() {
	log.Println("Adjusting bot runtime performance..")
	maxProcsEnv := os.Getenv("GOMAXPROCS")
	maxProcs := runtime.NumCPU()
	log.Println("Number of CPU's detected:", maxProcs)

	if maxProcsEnv != "" {
		log.Println("GOMAXPROCS env =", maxProcsEnv)
		env, err := strconv.Atoi(maxProcsEnv)
		if err != nil {
			log.Println("Unable to convert GOMAXPROCS to int, using", maxProcs)
		} else {
			maxProcs = env
		}
	}
	if i := runtime.GOMAXPROCS(maxProcs); i != maxProcs {
		log.Fatal("Go Max Procs were not set correctly.")
	}
	log.Println("Set GOMAXPROCS to:", maxProcs)
}

// HandleInterrupt monitors and captures the SIGTERM in a new goroutine then
// shuts down bot
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Captured %v, shutdown requested.", sig)
		bot.shutdown <- true
	}()
}

// Shutdown correctly shuts down bot saving configuration files
func Shutdown() {
	log.Println("Bot shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		bot.config.Portfolio = portfolio.Portfolio
	}

	if !bot.dryRun {
		err := bot.config.SaveConfig(bot.configFile)

		if err != nil {
			log.Println("Unable to save config.")
		} else {
			log.Println("Config file saved successfully.")
		}
	}

	log.Println("Exiting.")
	os.Exit(0)
}
