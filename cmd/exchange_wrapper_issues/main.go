package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

func main() {
	log.Println("Loading flags...")
	parseCLFlags()
	log.Println("Loading engine...")
	bot, err := engine.New()
	if err != nil {
		log.Fatalf("Failed to initialise engine. Err: %s", err)
	}
	engine.Bot = bot
	bot.ExchangeManager = engine.NewExchangeManager()

	bot.Settings = engine.Settings{
		CoreSettings: engine.CoreSettings{Verbose: verboseOverride},
		ExchangeTuningSettings: engine.ExchangeTuningSettings{
			DisableExchangeAutoPairUpdates: true,
			EnableExchangeHTTPRateLimiter:  true,
		},
	}

	log.Println("Loading config...")
	wrapperConfig, err := loadConfig()
	if err != nil {
		log.Printf("Error loading config: '%v', generating empty config", err)
		wrapperConfig = Config{
			Exchanges: make(map[string]*config.APICredentialsConfig),
		}
	}

	log.Println("Loading exchanges..")

	var wg sync.WaitGroup
	for i := range exchange.Exchanges {
		name := exchange.Exchanges[i]
		if _, ok := wrapperConfig.Exchanges[name]; !ok {
			wrapperConfig.Exchanges[strings.ToLower(name)] = &config.APICredentialsConfig{}
		}
		if shouldLoadExchange(name) {
			wg.Go(func() {
				if err = bot.LoadExchange(name); err != nil {
					log.Printf("Failed to load exchange %s. Err: %s", name, err)
				}
			})
		}
	}
	wg.Wait()
	log.Println("Done.")

	if orderTypeOverride != "LIMIT" {
		wrapperConfig.OrderSubmission.OrderType = orderTypeOverride
	}
	if orderSideOverride != "BUY" {
		wrapperConfig.OrderSubmission.OrderSide = orderSideOverride
	}
	if orderPriceOverride > 0 {
		wrapperConfig.OrderSubmission.Price = orderPriceOverride
	}
	if orderAmountOverride > 0 {
		wrapperConfig.OrderSubmission.Amount = orderAmountOverride
	}

	log.Println("Testing exchange wrappers..")
	var exchangeResponses []ExchangeResponses

	exchs := bot.GetExchanges()
	for x := range exchs {
		exchs[x].SetDefaults()
		base := exchs[x].GetBase()
		if !base.Config.Enabled {
			log.Printf("Exchange %v not enabled, skipping", base.GetName())
			continue
		}
		base.Config.Verbose = verboseOverride
		base.Verbose = verboseOverride
		base.HTTPDebugging = false
		base.Config.HTTPDebugging = false
		wg.Add(1)

		go func(num int) {
			name := exchs[num].GetName()
			authenticated := setExchangeAPIKeys(name, wrapperConfig.Exchanges, base)
			wrapperResult := ExchangeResponses{
				ID:                 fmt.Sprintf("Exchange%v", num),
				ExchangeName:       name,
				APIKeysSet:         authenticated,
				AssetPairResponses: testWrappers(exchs[num], base, &wrapperConfig),
			}
			for i := range wrapperResult.AssetPairResponses {
				wrapperResult.ErrorCount += wrapperResult.AssetPairResponses[i].ErrorCount
			}
			exchangeResponses = append(exchangeResponses, wrapperResult)
			wg.Done()
		}(x)
	}
	wg.Wait()

	log.Println("Done.")
	log.Println()

	sort.Slice(exchangeResponses, func(i, j int) bool {
		return exchangeResponses[i].ExchangeName < exchangeResponses[j].ExchangeName
	})

	if strings.EqualFold(outputOverride, "Console") {
		outputToConsole(exchangeResponses)
	}
	if strings.EqualFold(outputOverride, "JSON") {
		outputToJSON(exchangeResponses)
	}
	if strings.EqualFold(outputOverride, "HTML") {
		outputToHTML(exchangeResponses)
	}

	saveConfig(&wrapperConfig)
}

func parseCLFlags() {
	flag.StringVar(&exchangesToUseOverride, "exchanges", "", "a + delimited list of exchange names to run tests against eg -exchanges=bitfinex+okx")
	flag.StringVar(&exchangesToExcludeOverride, "excluded-exchanges", "", "a + delimited list of exchange names to ignore when they're being temperamental eg -exchangesToExlude=lbank")
	flag.StringVar(&assetTypeOverride, "asset", "", "the asset type to run tests against (where applicable)")
	flag.StringVar(&currencyPairOverride, "currency", "", "the currency to run tests against (where applicable)")
	flag.StringVar(&outputOverride, "output", "HTML", "JSON, HTML or Console")
	flag.BoolVar(&authenticatedOnly, "auth-only", false, "skip any wrapper function that doesn't require auth")
	flag.BoolVar(&verboseOverride, "verbose", false, "verbose CL output - if console output is selected then wrapper response is included")
	flag.StringVar(&orderSideOverride, "orderside", "BUY", "the order type for all order based wrapper tests")
	flag.StringVar(&orderTypeOverride, "ordertype", "LIMIT", "the order type for all order based wrapper tests")
	flag.Float64Var(&orderAmountOverride, "orderamount", 0, "the order amount for all order based wrapper tests")
	flag.Float64Var(&orderPriceOverride, "orderprice", 0, "the order price for all order based wrapper tests")
	flag.StringVar(&withdrawAddressOverride, "withdraw-wallet", "", "withdraw wallet address")
	flag.StringVar(&outputFileName, "filename", "report", "name of the output file eg 'report'.html or 'report'.json")
	flag.Parse()

	if exchangesToUseOverride != "" {
		exchangesToUseList = strings.Split(exchangesToUseOverride, "+")
	}
	if exchangesToExcludeOverride != "" {
		exchangesToExcludeList = strings.Split(exchangesToExcludeOverride, "+")
	}
}

func shouldLoadExchange(name string) bool {
	shouldLoadExchange := true
	if len(exchangesToUseList) > 0 {
		var found bool
		for i := range exchangesToUseList {
			if strings.EqualFold(name, exchangesToUseList[i]) {
				found = true
			}
		}
		if !found {
			shouldLoadExchange = false
		}
	}

	if len(exchangesToExcludeList) > 0 {
		for i := range exchangesToExcludeList {
			if strings.EqualFold(name, exchangesToExcludeList[i]) {
				if shouldLoadExchange {
					shouldLoadExchange = false
				}
			}
		}
	}
	return shouldLoadExchange
}

func setExchangeAPIKeys(name string, keys map[string]*config.APICredentialsConfig, base *exchange.Base) bool {
	lowerExchangeName := strings.ToLower(name)

	creds, ok := keys[lowerExchangeName]
	if !ok {
		log.Printf("%s credentials not found in keys map\n", name)
		return false
	}

	if base.API.CredentialsValidator.RequiresKey && creds.Key == "" {
		creds.Key = config.DefaultAPIKey
	}
	if base.API.CredentialsValidator.RequiresSecret && creds.Secret == "" {
		creds.Secret = config.DefaultAPISecret
	}
	if base.API.CredentialsValidator.RequiresPEM && creds.PEMKey == "" {
		creds.PEMKey = "PEM"
	}
	if base.API.CredentialsValidator.RequiresClientID && creds.ClientID == "" {
		creds.ClientID = config.DefaultAPIClientID
	}
	if creds.OTPSecret == "" {
		creds.OTPSecret = "-" // Ensure OTP is available for use
	}

	base.SetCredentials(creds.Key, creds.Secret, creds.ClientID, creds.Subaccount, creds.PEMKey, creds.OTPSecret)

	base.Config.API.Credentials.Key = creds.Key
	base.Config.API.Credentials.Secret = creds.Secret
	base.Config.API.Credentials.ClientID = creds.ClientID
	base.Config.API.Credentials.Subaccount = creds.Subaccount
	base.Config.API.Credentials.PEMKey = creds.PEMKey
	base.Config.API.Credentials.OTPSecret = creds.OTPSecret

	base.API.AuthenticatedSupport = true
	base.API.AuthenticatedWebsocketSupport = true
	base.Config.API.AuthenticatedSupport = true
	base.Config.API.AuthenticatedWebsocketSupport = true

	return base.VerifyAPICredentials(base.GetDefaultCredentials()) == nil
}

func parseOrderSide(orderSide string) order.Side {
	switch orderSide {
	case order.AnySide.String():
		return order.AnySide
	case order.Buy.String():
		return order.Buy
	case order.Sell.String():
		return order.Sell
	case order.Bid.String():
		return order.Bid
	case order.Ask.String():
		return order.Ask
	default:
		log.Printf("Orderside '%v' not recognised, defaulting to BUY", orderSide)
		return order.Buy
	}
}

func parseOrderType(orderType string) order.Type {
	switch orderType {
	case order.AnyType.String():
		return order.AnyType
	case order.Limit.String():
		return order.Limit
	case order.Market.String():
		return order.Market
	case order.Stop.String():
		return order.Stop
	case order.TrailingStop.String():
		return order.TrailingStop
	case order.UnknownType.String():
		return order.UnknownType
	default:
		log.Printf("OrderType '%v' not recognised, defaulting to LIMIT",
			orderTypeOverride)
		return order.Limit
	}
}

func testWrappers(e exchange.IBotExchange, base *exchange.Base, cfg *Config) []ExchangeAssetPairResponses {
	response := make([]ExchangeAssetPairResponses, 0)
	testOrderSide := parseOrderSide(cfg.OrderSubmission.OrderSide)
	testOrderType := parseOrderType(cfg.OrderSubmission.OrderType)
	assetTypes := base.GetAssetTypes(false)
	if assetTypeOverride != "" {
		a, err := asset.New(assetTypeOverride)
		if err != nil {
			log.Printf("%v Asset Type '%v' not recognised, defaulting to exchange defaults", base.GetName(), assetTypeOverride)
		} else {
			assetTypes = asset.Items{a}
		}
	}
	for i := range assetTypes {
		var msg string
		log.Printf("%v %v", base.GetName(), assetTypes[i])
		storedPairs, ok := base.Config.CurrencyPairs.Pairs[assetTypes[i]]
		if !ok {
			continue
		}

		var p currency.Pair
		var err error
		switch {
		case currencyPairOverride != "":
			p, err = currency.NewPairFromString(currencyPairOverride)
		case len(storedPairs.Enabled) == 0:
			if len(storedPairs.Available) == 0 {
				err = fmt.Errorf("%v has no enabled or available currencies. Skipping", base.GetName())
				break
			}
			p, err = storedPairs.Available.GetRandomPair()
		default:
			p, err = storedPairs.Enabled.GetRandomPair()
		}

		if err != nil {
			log.Printf("%v Encountered error: '%v'", base.GetName(), err)
			continue
		}

		p, err = disruptFormatting(p)
		if err != nil {
			log.Println("failed to disrupt currency pair formatting:", err)
		}

		responseContainer := ExchangeAssetPairResponses{
			AssetType: assetTypes[i],
			Pair:      p,
		}

		log.Printf("Setup config for %v %v %v", base.GetName(), assetTypes[i], p)
		err = e.Setup(base.Config)
		if err != nil {
			log.Printf("%v Encountered error reloading config: '%v'", base.GetName(), err)
		}
		log.Printf("Executing wrappers for %v %v %v", base.GetName(), assetTypes[i], p)

		if !authenticatedOnly {
			var updateTickerResponse *ticker.Price
			updateTickerResponse, err = e.UpdateTicker(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{p, assetTypes[i]}),
				Function:   "UpdateTicker",
				Error:      msg,
				Response:   jsonifyInterface([]any{updateTickerResponse}),
			})

			var GetCachedTickerResponse *ticker.Price
			GetCachedTickerResponse, err = e.GetCachedTicker(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{p, assetTypes[i]}),
				Function:   "GetCachedTicker",
				Error:      msg,
				Response:   jsonifyInterface([]any{GetCachedTickerResponse}),
			})

			var updateOrderbookResponse *orderbook.Book
			updateOrderbookResponse, err = e.UpdateOrderbook(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{p, assetTypes[i]}),
				Function:   "UpdateOrderbook",
				Error:      msg,
				Response:   jsonifyInterface([]any{updateOrderbookResponse}),
			})

			var GetCachedOrderbookResponse *orderbook.Book
			GetCachedOrderbookResponse, err = e.GetCachedOrderbook(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{p, assetTypes[i]}),
				Function:   "GetCachedOrderbook",
				Error:      msg,
				Response:   jsonifyInterface([]any{GetCachedOrderbookResponse}),
			})

			var fetchTradablePairsResponse []currency.Pair
			fetchTradablePairsResponse, err = e.FetchTradablePairs(context.TODO(), assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{assetTypes[i]}),
				Function:   "FetchTradablePairs",
				Error:      msg,
				Response:   jsonifyInterface([]any{fetchTradablePairsResponse}),
			})
			// r6
			err = e.UpdateTradablePairs(context.TODO())
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{false}),
				Function:   "UpdateTradablePairs",
				Error:      msg,
				Response:   jsonifyInterface([]any{nil}),
			})

			var getHistoricTradesResponse []trade.Data
			getHistoricTradesResponse, err = e.GetHistoricTrades(context.TODO(), p, assetTypes[i], time.Now().Add(-time.Hour), time.Now())
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{p, assetTypes[i], time.Now().Add(-time.Hour), time.Now()}),
				Function:   "GetHistoricTrades",
				Error:      msg,
				Response:   jsonifyInterface([]any{getHistoricTradesResponse}),
			})

			var getRecentTradesResponse []trade.Data
			getRecentTradesResponse, err = e.GetRecentTrades(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{p, assetTypes[i]}),
				Function:   "GetRecentTrades",
				Error:      msg,
				Response:   jsonifyInterface([]any{getRecentTradesResponse}),
			})

			var getHistoricCandlesResponse *kline.Item
			startTime, endTime := time.Now().AddDate(0, 0, -1), time.Now()
			getHistoricCandlesResponse, err = e.GetHistoricCandles(context.TODO(), p, assetTypes[i], kline.OneDay, startTime, endTime)
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				Function:   "GetHistoricCandles",
				Error:      msg,
				Response:   getHistoricCandlesResponse,
				SentParams: jsonifyInterface([]any{p, assetTypes[i], startTime, endTime, kline.OneDay}),
			})

			var getHistoricCandlesExtendedResponse *kline.Item
			getHistoricCandlesExtendedResponse, err = e.GetHistoricCandlesExtended(context.TODO(), p, assetTypes[i], kline.OneDay, startTime, endTime)
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				Function:   "GetHistoricCandlesExtended",
				Error:      msg,
				Response:   getHistoricCandlesExtendedResponse,
				SentParams: jsonifyInterface([]any{p, assetTypes[i], startTime, endTime, kline.OneDay}),
			})

			var getServerTimeResponse time.Time
			getServerTimeResponse, err = e.GetServerTime(context.TODO(), assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				Function:   "GetServerTime",
				Error:      msg,
				Response:   getServerTimeResponse,
				SentParams: jsonifyInterface([]any{assetTypes[i]}),
			})

			err = e.UpdateOrderExecutionLimits(context.TODO(), assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}

			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{assetTypes[i]}),
				Function:   "UpdateOrderExecutionLimits",
				Error:      msg,
				Response:   jsonifyInterface([]any{""}),
			})

			fundingRateRequest := &fundingrate.HistoricalRatesRequest{
				Asset:     assetTypes[i],
				Pair:      p,
				StartDate: time.Now().Add(-time.Hour),
				EndDate:   time.Now(),
			}
			var fundingRateResponse *fundingrate.HistoricalRates
			fundingRateResponse, err = e.GetHistoricalFundingRates(context.TODO(), fundingRateRequest)
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}

			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{fundingRateRequest}),
				Function:   "GetFundingRates",
				Error:      msg,
				Response:   jsonifyInterface([]any{fundingRateResponse}),
			})

			var isPerpetualFutures bool
			isPerpetualFutures, err = e.IsPerpetualFutureCurrency(assetTypes[i], p)
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]any{assetTypes[i], p}),
				Function:   "IsPerpetualFutureCurrency",
				Error:      msg,
				Response:   jsonifyInterface([]any{isPerpetualFutures}),
			})
		}

		var GetCachedSubAccountsResponse accounts.SubAccounts
		GetCachedSubAccountsResponse, err = e.GetCachedSubAccounts(context.TODO(), assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetCachedSubAccounts",
			Error:    msg,
			Response: jsonifyInterface([]any{GetCachedSubAccountsResponse}),
		})

		var getFundingHistoryResponse []exchange.FundingHistory
		getFundingHistoryResponse, err = e.GetAccountFundingHistory(context.TODO())
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetAccountFundingHistory",
			Error:    msg,
			Response: jsonifyInterface([]any{getFundingHistoryResponse}),
		})

		feeType := exchange.FeeBuilder{
			FeeType:       exchange.CryptocurrencyTradeFee,
			Pair:          p,
			PurchasePrice: cfg.OrderSubmission.Price,
			Amount:        cfg.OrderSubmission.Amount,
		}
		var getFeeByTypeResponse float64
		getFeeByTypeResponse, err = e.GetFeeByType(context.TODO(), &feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{feeType}),
			Function:   "GetFeeByType-Trade",
			Error:      msg,
			Response:   jsonifyInterface([]any{getFeeByTypeResponse}),
		})

		s := &order.Submit{
			Exchange:  e.GetName(),
			Pair:      p,
			Side:      testOrderSide,
			Type:      testOrderType,
			Amount:    cfg.OrderSubmission.Amount,
			Price:     cfg.OrderSubmission.Price,
			ClientID:  cfg.OrderSubmission.OrderID,
			AssetType: assetTypes[i],
		}
		var submitOrderResponse *order.SubmitResponse
		submitOrderResponse, err = e.SubmitOrder(context.TODO(), s)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{*s}),
			Function:   "SubmitOrder",
			Error:      msg,
			Response:   jsonifyInterface([]any{submitOrderResponse}),
		})

		modifyRequest := order.Modify{
			OrderID:   cfg.OrderSubmission.OrderID,
			Type:      testOrderType,
			Side:      testOrderSide,
			Pair:      p,
			Price:     cfg.OrderSubmission.Price,
			Amount:    cfg.OrderSubmission.Amount,
			AssetType: assetTypes[i],
		}
		modifyOrderResponse, err := e.ModifyOrder(context.TODO(), &modifyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{modifyRequest}),
			Function:   "ModifyOrder",
			Error:      msg,
			Response:   modifyOrderResponse,
		})

		cancelRequest := order.Cancel{
			Side:      testOrderSide,
			Pair:      p,
			OrderID:   cfg.OrderSubmission.OrderID,
			AssetType: assetTypes[i],
		}
		err = e.CancelOrder(context.TODO(), &cancelRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{cancelRequest}),
			Function:   "CancelOrder",
			Error:      msg,
			Response:   jsonifyInterface([]any{nil}),
		})

		var request []order.Cancel
		request = append(request, order.Cancel{
			Side:      testOrderSide,
			Pair:      p,
			OrderID:   cfg.OrderSubmission.OrderID,
			AssetType: assetTypes[i],
		})

		var CancelBatchOrdersResponse *order.CancelBatchResponse
		CancelBatchOrdersResponse, err = e.CancelBatchOrders(context.TODO(), request)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{cancelRequest}),
			Function:   "CancelBatchOrders",
			Error:      msg,
			Response:   jsonifyInterface([]any{CancelBatchOrdersResponse}),
		})

		var cancellAllOrdersResponse order.CancelAllResponse
		cancellAllOrdersResponse, err = e.CancelAllOrders(context.TODO(), &cancelRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{cancelRequest}),
			Function:   "CancelAllOrders",
			Error:      msg,
			Response:   jsonifyInterface([]any{cancellAllOrdersResponse}),
		})

		var r15 *order.Detail
		r15, err = e.GetOrderInfo(context.TODO(), cfg.OrderSubmission.OrderID, p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{cfg.OrderSubmission.OrderID, p, assetTypes[i]}),
			Function:   "GetOrderInfo",
			Error:      msg,
			Response:   jsonifyInterface([]any{r15}),
		})

		historyRequest := order.MultiOrderRequest{
			Type:      testOrderType,
			Side:      testOrderSide,
			Pairs:     []currency.Pair{p},
			AssetType: assetTypes[i],
			StartTime: time.Now().Add(-time.Hour),
			EndTime:   time.Now(),
		}
		var getOrderHistoryResponse []order.Detail
		getOrderHistoryResponse, err = e.GetOrderHistory(context.TODO(), &historyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{historyRequest}),
			Function:   "GetOrderHistory",
			Error:      msg,
			Response:   jsonifyInterface([]any{getOrderHistoryResponse}),
		})

		orderRequest := order.MultiOrderRequest{
			Type:      testOrderType,
			Side:      testOrderSide,
			Pairs:     []currency.Pair{p},
			AssetType: assetTypes[i],
			StartTime: time.Now().Add(-time.Hour),
			EndTime:   time.Now(),
		}
		var getActiveOrdersResponse []order.Detail
		getActiveOrdersResponse, err = e.GetActiveOrders(context.TODO(), &orderRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{orderRequest}),
			Function:   "GetActiveOrders",
			Error:      msg,
			Response:   jsonifyInterface([]any{getActiveOrdersResponse}),
		})

		var getDepositAddressResponse *deposit.Address
		getDepositAddressResponse, err = e.GetDepositAddress(context.TODO(), p.Base, "", "")
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{p.Base, ""}),
			Function:   "GetDepositAddress",
			Error:      msg,
			Response:   getDepositAddressResponse,
		})

		feeType = exchange.FeeBuilder{
			FeeType:       exchange.CryptocurrencyWithdrawalFee,
			Pair:          p,
			PurchasePrice: cfg.OrderSubmission.Price,
			Amount:        cfg.OrderSubmission.Amount,
		}
		var GetFeeByTypeResponse float64
		GetFeeByTypeResponse, err = e.GetFeeByType(context.TODO(), &feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{feeType}),
			Function:   "GetFeeByType-Crypto-Withdraw",
			Error:      msg,
			Response:   jsonifyInterface([]any{GetFeeByTypeResponse}),
		})

		withdrawRequest := withdraw.Request{
			Currency: p.Quote,
			Crypto: withdraw.CryptoRequest{
				Address: withdrawAddressOverride,
			},
			Amount: cfg.OrderSubmission.Amount,
		}
		msg = ""
		err = withdrawRequest.Validate()
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		withdrawCryptocurrencyFundsResponse, err := e.WithdrawCryptocurrencyFunds(context.TODO(), &withdrawRequest)
		if err != nil {
			msg += ", " + err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{withdrawRequest}),
			Function:   "WithdrawCryptocurrencyFunds",
			Error:      msg,
			Response:   withdrawCryptocurrencyFundsResponse,
		})

		feeType = exchange.FeeBuilder{
			FeeType:             exchange.InternationalBankWithdrawalFee,
			Pair:                p,
			PurchasePrice:       cfg.OrderSubmission.Price,
			Amount:              cfg.OrderSubmission.Amount,
			FiatCurrency:        currency.AUD,
			BankTransactionType: exchange.WireTransfer,
		}
		var getFeeByTypeFiatResponse float64
		getFeeByTypeFiatResponse, err = e.GetFeeByType(context.TODO(), &feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{feeType}),
			Function:   "GetFeeByType-FIAT-Withdraw",
			Error:      msg,
			Response:   jsonifyInterface([]any{getFeeByTypeFiatResponse}),
		})

		withdrawRequestFiat := withdraw.Request{
			Currency: p.Quote,
			Amount:   cfg.OrderSubmission.Amount,
			Fiat: withdraw.FiatRequest{
				Bank: banking.Account{
					AccountName:    cfg.BankDetails.BankAccountName,
					AccountNumber:  cfg.BankDetails.BankAccountNumber,
					SWIFTCode:      cfg.BankDetails.SwiftCode,
					IBAN:           cfg.BankDetails.Iban,
					BankPostalCity: cfg.BankDetails.BankCity,
					BankName:       cfg.BankDetails.BankName,
					BankAddress:    cfg.BankDetails.BankAddress,
					BankCountry:    cfg.BankDetails.BankCountry,
					BankPostalCode: cfg.BankDetails.BankPostalCode,
					BankCode:       cfg.BankDetails.BankCode,
				},

				IsExpressWire:                 cfg.BankDetails.IsExpressWire,
				RequiresIntermediaryBank:      cfg.BankDetails.RequiresIntermediaryBank,
				IntermediaryBankName:          cfg.BankDetails.IntermediaryBankName,
				IntermediaryBankAccountNumber: cfg.BankDetails.IntermediaryBankAccountNumber,
				IntermediarySwiftCode:         cfg.BankDetails.IntermediarySwiftCode,
				IntermediaryIBAN:              cfg.BankDetails.IntermediaryIban,
				IntermediaryBankCity:          cfg.BankDetails.IntermediaryBankCity,
				IntermediaryBankAddress:       cfg.BankDetails.IntermediaryBankAddress,
				IntermediaryBankCountry:       cfg.BankDetails.IntermediaryBankCountry,
				IntermediaryBankPostalCode:    cfg.BankDetails.IntermediaryBankPostalCode,
				IntermediaryBankCode:          cfg.BankDetails.IntermediaryBankCode,
			},
		}
		withdrawFiatFundsResponse, err := e.WithdrawFiatFunds(context.TODO(), &withdrawRequestFiat)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{withdrawRequestFiat}),
			Function:   "WithdrawFiatFunds",
			Error:      msg,
			Response:   withdrawFiatFundsResponse,
		})

		withdrawFiatFundsInternationalResponse, err := e.WithdrawFiatFundsToInternationalBank(context.TODO(), &withdrawRequestFiat)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{withdrawRequestFiat}),
			Function:   "WithdrawFiatFundsToInternationalBank",
			Error:      msg,
			Response:   withdrawFiatFundsInternationalResponse,
		})

		marginRateHistoryRequest := &margin.RateHistoryRequest{
			Exchange:           e.GetName(),
			Asset:              assetTypes[i],
			Currency:           p.Base,
			StartDate:          time.Now().Add(-time.Hour * 24),
			EndDate:            time.Now(),
			GetPredictedRate:   true,
			GetLendingPayments: true,
			GetBorrowRates:     true,
			GetBorrowCosts:     true,
		}
		marginRateHistoryResponse, err := e.GetMarginRatesHistory(context.TODO(), marginRateHistoryRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{marginRateHistoryRequest}),
			Function:   "GetMarginRatesHistory",
			Error:      msg,
			Response:   marginRateHistoryResponse,
		})

		positionSummaryRequest := &futures.PositionSummaryRequest{
			Asset: assetTypes[i],
			Pair:  p,
		}
		var positionSummaryResponse *futures.PositionSummary
		positionSummaryResponse, err = e.GetPositionSummary(context.TODO(), positionSummaryRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{positionSummaryRequest}),
			Function:   "GetFuturesPositionSummary",
			Error:      msg,
			Response:   jsonifyInterface([]any{positionSummaryResponse}),
		})

		calculatePNLRequest := &futures.PNLCalculatorRequest{
			Pair:             p,
			Underlying:       p.Base,
			Asset:            assetTypes[i],
			EntryPrice:       decimal.NewFromInt(1337),
			OpeningDirection: testOrderSide,
			OrderDirection:   testOrderSide,
			Time:             time.Now(),
			Exposure:         decimal.NewFromInt(1337),
			EntryAmount:      decimal.NewFromInt(1337),
			PreviousPrice:    decimal.NewFromInt(1337),
		}
		var calculatePNLResponse *futures.PNLResult
		calculatePNLResponse, err = e.CalculatePNL(context.TODO(), calculatePNLRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{calculatePNLRequest}),
			Function:   "CalculatePNL",
			Error:      msg,
			Response:   jsonifyInterface([]any{calculatePNLResponse}),
		})

		collateralCalculator := &futures.CollateralCalculator{
			CollateralCurrency: p.Quote,
			Asset:              assetTypes[i],
			Side:               testOrderSide,
			USDPrice:           decimal.NewFromInt(1337),
			FreeCollateral:     decimal.NewFromInt(1337),
			LockedCollateral:   decimal.NewFromInt(1337),
			UnrealisedPNL:      decimal.NewFromInt(1337),
		}
		var scaleCollateralResponse *collateral.ByCurrency
		scaleCollateralResponse, err = e.ScaleCollateral(context.TODO(), collateralCalculator)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{collateralCalculator}),
			Function:   "ScaleCollateral",
			Error:      msg,
			Response:   jsonifyInterface([]any{scaleCollateralResponse}),
		})

		totalCollateralCalculator := &futures.TotalCollateralCalculator{
			CollateralAssets: []futures.CollateralCalculator{*collateralCalculator},
		}
		var calculateTotalCollateralResponse *futures.TotalCollateralResponse
		calculateTotalCollateralResponse, err = e.CalculateTotalCollateral(context.TODO(), totalCollateralCalculator)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{totalCollateralCalculator}),
			Function:   "CalculateTotalCollateral",
			Error:      msg,
			Response:   jsonifyInterface([]any{calculateTotalCollateralResponse}),
		})

		var futuresPositionsResponse []futures.PositionResponse
		futuresPositionsRequest := &futures.PositionsRequest{
			Asset:     assetTypes[i],
			Pairs:     currency.Pairs{p},
			StartDate: time.Now().Add(-time.Hour),
		}
		futuresPositionsResponse, err = e.GetFuturesPositionOrders(context.TODO(), futuresPositionsRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]any{futuresPositionsRequest}),
			Function:   "GetFuturesPositionOrders",
			Error:      msg,
			Response:   jsonifyInterface([]any{futuresPositionsResponse}),
		})

		response = append(response, responseContainer)
	}
	return response
}

func jsonifyInterface(params []any) json.RawMessage {
	response, _ := json.MarshalIndent(params, "", " ")
	return response
}

func loadConfig() (Config, error) {
	var cfg Config
	keys, err := os.ReadFile("wrapperconfig.json")
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(keys, &cfg)
	return cfg, err
}

func saveConfig(cfg *Config) {
	log.Println("JSONifying config...")
	jsonOutput, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		log.Fatalf("Encountered error encoding JSON: %v", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Printf("Encountered error retrieving output directory: %v", err)
		return
	}

	log.Printf("Outputting to: %v", filepath.Join(dir, "wrapperconfig.json"))
	err = file.Write(filepath.Join(dir, "wrapperconfig.json"), jsonOutput)
	if err != nil {
		log.Printf("Encountered error writing to disk: %v", err)
		return
	}
}

func outputToJSON(exchangeResponses []ExchangeResponses) {
	log.Println("JSONifying results...")
	jsonOutput, err := json.MarshalIndent(exchangeResponses, "", " ")
	if err != nil {
		log.Fatalf("Encountered error encoding JSON: %v", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Printf("Encountered error retrieving output directory: %v", err)
		return
	}

	log.Printf("Outputting to: %v", filepath.Join(dir, fmt.Sprintf("%v.json", outputFileName)))
	err = file.Write(filepath.Join(dir, fmt.Sprintf("%v.json", outputFileName)), jsonOutput)
	if err != nil {
		log.Printf("Encountered error writing to disk: %v", err)
		return
	}
}

func outputToHTML(exchangeResponses []ExchangeResponses) {
	log.Println("Generating HTML report...")
	dir, err := os.Getwd()
	if err != nil {
		log.Print(err)
		return
	}

	tmpl, err := template.New("report.tmpl").ParseFiles(filepath.Join(dir, "report.tmpl"))
	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("Outputting to: %v", filepath.Join(dir, fmt.Sprintf("%v.html", outputFileName)))
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%v.html", outputFileName)))
	if err != nil {
		log.Print(err)
		return
	}

	err = tmpl.Execute(f, exchangeResponses)
	if err != nil {
		log.Print(err)
	}
	err = f.Close()
	if err != nil {
		log.Print(err)
	}
}

func outputToConsole(exchangeResponses []ExchangeResponses) {
	var totalErrors int64
	for i := range exchangeResponses {
		log.Printf("------------%v Results-------------\n", exchangeResponses[i].ExchangeName)
		for j := range exchangeResponses[i].AssetPairResponses {
			for k := range exchangeResponses[i].AssetPairResponses[j].EndpointResponses {
				log.Printf("%v Result: %v", exchangeResponses[i].ExchangeName, k)
				log.Printf("Function:\t%v", exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].Function)
				log.Printf("AssetType:\t%v", exchangeResponses[i].AssetPairResponses[j].AssetType)
				log.Printf("Currency:\t%v\n", exchangeResponses[i].AssetPairResponses[j].Pair)
				log.Printf("Wrapper Params:\t%s\n", exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].SentParams)
				if exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].Error != "" {
					totalErrors++
					log.Printf("Error:\t%v", exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].Error)
				} else {
					log.Print("Error:\tnone")
				}
				if verboseOverride {
					log.Printf("Wrapper Response:\t%s", exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].Response)
				}
				log.Println()
			}
		}
		log.Println()
	}
}

// disruptFormatting adds in an unused delimiter and strange casing features to
// ensure format currency pair is used throughout the code base.
func disruptFormatting(p currency.Pair) (currency.Pair, error) {
	if p.Base.IsEmpty() {
		return currency.EMPTYPAIR, errors.New("cannot disrupt formatting as base is not populated")
	}
	// NOTE: Quote can be empty for margin funding
	return currency.Pair{
		Base:      p.Base.Upper(),
		Quote:     p.Quote.Lower(),
		Delimiter: "-TEST-DELIM-",
	}, nil
}
