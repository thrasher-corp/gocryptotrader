package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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
	bot.ExchangeManager = engine.SetupExchangeManager()

	bot.Settings = engine.Settings{
		DisableExchangeAutoPairUpdates: true,
		Verbose:                        verboseOverride,
		EnableExchangeHTTPRateLimiter:  true,
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
	for x := range exchange.Exchanges {
		name := exchange.Exchanges[x]
		if _, ok := wrapperConfig.Exchanges[name]; !ok {
			wrapperConfig.Exchanges[strings.ToLower(name)] = &config.APICredentialsConfig{}
		}
		if shouldLoadExchange(name) {
			err = bot.LoadExchange(name, &wg)
			if err != nil {
				log.Printf("Failed to load exchange %s. Err: %s", name, err)
				continue
			}
		}
	}
	wg.Wait()
	log.Println("Done.")

	if withdrawAddressOverride != "" {
		wrapperConfig.WalletAddress = withdrawAddressOverride
	}
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
	flag.StringVar(&exchangesToUseOverride, "exchanges", "", "a + delimited list of exchange names to run tests against eg -exchanges=bitfinex+okex")
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

	return base.ValidateAPICredentials(base.GetDefaultCredentials()) == nil
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
	case order.ImmediateOrCancel.String():
		return order.ImmediateOrCancel
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

func testWrappers(e exchange.IBotExchange, base *exchange.Base, config *Config) []ExchangeAssetPairResponses {
	var response []ExchangeAssetPairResponses
	testOrderSide := parseOrderSide(config.OrderSubmission.OrderSide)
	testOrderType := parseOrderType(config.OrderSubmission.OrderType)
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
		if _, ok := base.Config.CurrencyPairs.Pairs[assetTypes[i]]; !ok {
			continue
		}

		var p currency.Pair
		switch {
		case currencyPairOverride != "":
			var err error
			p, err = currency.NewPairFromString(currencyPairOverride)
			if err != nil {
				log.Printf("%v Encountered error: '%v'", base.GetName(), err)
				continue
			}
		case len(base.Config.CurrencyPairs.Pairs[assetTypes[i]].Enabled) == 0:
			if len(base.Config.CurrencyPairs.Pairs[assetTypes[i]].Available) == 0 {
				log.Printf("%v has no enabled or available currencies. Skipping",
					base.GetName())
				continue
			}
			p = base.Config.CurrencyPairs.Pairs[assetTypes[i]].Available.GetRandomPair()
		default:
			p = base.Config.CurrencyPairs.Pairs[assetTypes[i]].Enabled.GetRandomPair()
		}

		var err error
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
			var fetchTickerResponse *ticker.Price
			fetchTickerResponse, err = e.FetchTicker(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "FetchTicker",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{fetchTickerResponse}),
			})

			var updateTickerResponse *ticker.Price
			updateTickerResponse, err = e.UpdateTicker(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "UpdateTicker",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{updateTickerResponse}),
			})

			var fetchOrderbookResponse *orderbook.Base
			fetchOrderbookResponse, err = e.FetchOrderbook(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "FetchOrderbook",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{fetchOrderbookResponse}),
			})

			var updateOrderbookResponse *orderbook.Base
			updateOrderbookResponse, err = e.UpdateOrderbook(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "UpdateOrderbook",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{updateOrderbookResponse}),
			})

			var fetchTradablePairsResponse []string
			fetchTradablePairsResponse, err = e.FetchTradablePairs(context.TODO(), assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{assetTypes[i]}),
				Function:   "FetchTradablePairs",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{fetchTradablePairsResponse}),
			})
			// r6
			err = e.UpdateTradablePairs(context.TODO(), false)
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{false}),
				Function:   "UpdateTradablePairs",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{nil}),
			})

			var getHistoricTradesResponse []trade.Data
			getHistoricTradesResponse, err = e.GetHistoricTrades(context.TODO(), p, assetTypes[i], time.Now().Add(-time.Hour), time.Now())
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i], time.Now().Add(-time.Hour), time.Now()}),
				Function:   "GetHistoricTrades",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{getHistoricTradesResponse}),
			})

			var getRecentTradesResponse []trade.Data
			getRecentTradesResponse, err = e.GetRecentTrades(context.TODO(), p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "GetRecentTrades",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{getRecentTradesResponse}),
			})

			var getHistoricCandlesResponse kline.Item
			startTime, endTime := time.Now().AddDate(0, 0, -1), time.Now()
			getHistoricCandlesResponse, err = e.GetHistoricCandles(context.TODO(), p, assetTypes[i], startTime, endTime, kline.OneDay)
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				Function:   "GetHistoricCandles",
				Error:      msg,
				Response:   getHistoricCandlesResponse,
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i], startTime, endTime, kline.OneDay}),
			})

			var getHisotirCandlesExtendedResponse kline.Item
			getHisotirCandlesExtendedResponse, err = e.GetHistoricCandlesExtended(context.TODO(), p, assetTypes[i], startTime, endTime, kline.OneDay)
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				Function:   "GetHistoricCandlesExtended",
				Error:      msg,
				Response:   getHisotirCandlesExtendedResponse,
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i], startTime, endTime, kline.OneDay}),
			})

			err = e.UpdateOrderExecutionLimits(context.TODO(), assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}

			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{assetTypes[i]}),
				Function:   "UpdateOrderExecutionLimits",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{""}),
			})
		}

		var fetchAccountInfoResponse account.Holdings
		fetchAccountInfoResponse, err = e.FetchAccountInfo(context.TODO(), assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "FetchAccountInfo",
			Error:    msg,
			Response: jsonifyInterface([]interface{}{fetchAccountInfoResponse}),
		})

		var getFundingHistoryResponse []exchange.FundHistory
		getFundingHistoryResponse, err = e.GetFundingHistory(context.TODO())
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetFundingHistory",
			Error:    msg,
			Response: jsonifyInterface([]interface{}{getFundingHistoryResponse}),
		})

		feeType := exchange.FeeBuilder{
			FeeType:       exchange.CryptocurrencyTradeFee,
			Pair:          p,
			PurchasePrice: config.OrderSubmission.Price,
			Amount:        config.OrderSubmission.Amount,
		}
		var getFeeByTypeResponse float64
		getFeeByTypeResponse, err = e.GetFeeByType(context.TODO(), &feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{feeType}),
			Function:   "GetFeeByType-Trade",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{getFeeByTypeResponse}),
		})

		s := &order.Submit{
			Pair:      p,
			Side:      testOrderSide,
			Type:      testOrderType,
			Amount:    config.OrderSubmission.Amount,
			Price:     config.OrderSubmission.Price,
			ClientID:  config.OrderSubmission.OrderID,
			AssetType: assetTypes[i],
		}
		var submitOrderResponse order.SubmitResponse
		submitOrderResponse, err = e.SubmitOrder(context.TODO(), s)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{*s}),
			Function:   "SubmitOrder",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{submitOrderResponse}),
		})

		modifyRequest := order.Modify{
			ID:     config.OrderSubmission.OrderID,
			Type:   testOrderType,
			Side:   testOrderSide,
			Pair:   p,
			Price:  config.OrderSubmission.Price,
			Amount: config.OrderSubmission.Amount,
		}
		modifyOrderResponse, err := e.ModifyOrder(context.TODO(), &modifyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{modifyRequest}),
			Function:   "ModifyOrder",
			Error:      msg,
			Response:   modifyOrderResponse,
		})

		cancelRequest := order.Cancel{
			Side:      testOrderSide,
			Pair:      p,
			ID:        config.OrderSubmission.OrderID,
			AssetType: assetTypes[i],
		}
		err = e.CancelOrder(context.TODO(), &cancelRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{cancelRequest}),
			Function:   "CancelOrder",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{nil}),
		})

		var request []order.Cancel
		request = append(request, order.Cancel{
			Side:      testOrderSide,
			Pair:      p,
			ID:        config.OrderSubmission.OrderID,
			AssetType: assetTypes[i],
		})

		var CancelBatchOrdersResponse order.CancelBatchResponse
		CancelBatchOrdersResponse, err = e.CancelBatchOrders(context.TODO(), request)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{cancelRequest}),
			Function:   "CancelBatchOrders",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{CancelBatchOrdersResponse}),
		})

		var cancellAllOrdersResponse order.CancelAllResponse
		cancellAllOrdersResponse, err = e.CancelAllOrders(context.TODO(), &cancelRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{cancelRequest}),
			Function:   "CancelAllOrders",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{cancellAllOrdersResponse}),
		})

		var r15 order.Detail
		r15, err = e.GetOrderInfo(context.TODO(), config.OrderSubmission.OrderID, p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{config.OrderSubmission.OrderID}),
			Function:   "GetOrderInfo",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r15}),
		})

		historyRequest := order.GetOrdersRequest{
			Type:  testOrderType,
			Side:  testOrderSide,
			Pairs: []currency.Pair{p},
		}
		var getOrderHistoryResponse []order.Detail
		getOrderHistoryResponse, err = e.GetOrderHistory(context.TODO(), &historyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{historyRequest}),
			Function:   "GetOrderHistory",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{getOrderHistoryResponse}),
		})

		orderRequest := order.GetOrdersRequest{
			Type:  testOrderType,
			Side:  testOrderSide,
			Pairs: []currency.Pair{p},
		}
		var getActiveOrdersResponse []order.Detail
		getActiveOrdersResponse, err = e.GetActiveOrders(context.TODO(), &orderRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{orderRequest}),
			Function:   "GetActiveOrders",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{getActiveOrdersResponse}),
		})

		var getDepositAddressResponse *deposit.Address
		getDepositAddressResponse, err = e.GetDepositAddress(context.TODO(), p.Base, "", "")
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{p.Base, ""}),
			Function:   "GetDepositAddress",
			Error:      msg,
			Response:   getDepositAddressResponse,
		})

		feeType = exchange.FeeBuilder{
			FeeType:       exchange.CryptocurrencyWithdrawalFee,
			Pair:          p,
			PurchasePrice: config.OrderSubmission.Price,
			Amount:        config.OrderSubmission.Amount,
		}
		var GetFeeByTypeResponse float64
		GetFeeByTypeResponse, err = e.GetFeeByType(context.TODO(), &feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{feeType}),
			Function:   "GetFeeByType-Crypto-Withdraw",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{GetFeeByTypeResponse}),
		})

		withdrawRequest := withdraw.Request{
			Currency: p.Quote,
			Crypto: withdraw.CryptoRequest{
				Address: withdrawAddressOverride,
			},
			Amount: config.OrderSubmission.Amount,
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
			SentParams: jsonifyInterface([]interface{}{withdrawRequest}),
			Function:   "WithdrawCryptocurrencyFunds",
			Error:      msg,
			Response:   withdrawCryptocurrencyFundsResponse,
		})

		feeType = exchange.FeeBuilder{
			FeeType:             exchange.InternationalBankWithdrawalFee,
			Pair:                p,
			PurchasePrice:       config.OrderSubmission.Price,
			Amount:              config.OrderSubmission.Amount,
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
			SentParams: jsonifyInterface([]interface{}{feeType}),
			Function:   "GetFeeByType-FIAT-Withdraw",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{getFeeByTypeFiatResponse}),
		})

		withdrawRequestFiat := withdraw.Request{
			Currency: p.Quote,
			Amount:   config.OrderSubmission.Amount,
			Fiat: withdraw.FiatRequest{
				Bank: banking.Account{
					AccountName:    config.BankDetails.BankAccountName,
					AccountNumber:  config.BankDetails.BankAccountNumber,
					SWIFTCode:      config.BankDetails.SwiftCode,
					IBAN:           config.BankDetails.Iban,
					BankPostalCity: config.BankDetails.BankCity,
					BankName:       config.BankDetails.BankName,
					BankAddress:    config.BankDetails.BankAddress,
					BankCountry:    config.BankDetails.BankCountry,
					BankPostalCode: config.BankDetails.BankPostalCode,
					BankCode:       config.BankDetails.BankCode,
				},

				IsExpressWire:                 config.BankDetails.IsExpressWire,
				RequiresIntermediaryBank:      config.BankDetails.RequiresIntermediaryBank,
				IntermediaryBankName:          config.BankDetails.IntermediaryBankName,
				IntermediaryBankAccountNumber: config.BankDetails.IntermediaryBankAccountNumber,
				IntermediarySwiftCode:         config.BankDetails.IntermediarySwiftCode,
				IntermediaryIBAN:              config.BankDetails.IntermediaryIban,
				IntermediaryBankCity:          config.BankDetails.IntermediaryBankCity,
				IntermediaryBankAddress:       config.BankDetails.IntermediaryBankAddress,
				IntermediaryBankCountry:       config.BankDetails.IntermediaryBankCountry,
				IntermediaryBankPostalCode:    config.BankDetails.IntermediaryBankPostalCode,
				IntermediaryBankCode:          config.BankDetails.IntermediaryBankCode,
			},
		}
		withdrawFiatFundsResponse, err := e.WithdrawFiatFunds(context.TODO(), &withdrawRequestFiat)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{withdrawRequestFiat}),
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
			SentParams: jsonifyInterface([]interface{}{withdrawRequestFiat}),
			Function:   "WithdrawFiatFundsToInternationalBank",
			Error:      msg,
			Response:   withdrawFiatFundsInternationalResponse,
		})
		response = append(response, responseContainer)
	}
	return response
}

func jsonifyInterface(params []interface{}) json.RawMessage {
	response, _ := json.MarshalIndent(params, "", " ")
	return response
}

func loadConfig() (Config, error) {
	var config Config
	keys, err := ioutil.ReadFile("wrapperconfig.json")
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(keys, &config)
	return config, err
}

func saveConfig(config *Config) {
	log.Println("JSONifying config...")
	jsonOutput, err := json.MarshalIndent(config, "", " ")
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
	file, err := os.Create(filepath.Join(dir, fmt.Sprintf("%v.html", outputFileName)))
	if err != nil {
		log.Print(err)
		return
	}

	err = tmpl.Execute(file, exchangeResponses)
	if err != nil {
		log.Print(err)
	}
	file.Close()
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
	if p.Quote.IsEmpty() {
		return currency.EMPTYPAIR, errors.New("cannot disrupt formatting as quote is not populated")
	}

	return currency.Pair{
		Base:      p.Base.Upper(),
		Quote:     p.Quote.Lower(),
		Delimiter: "-TEST-DELIM-",
	}, nil
}
