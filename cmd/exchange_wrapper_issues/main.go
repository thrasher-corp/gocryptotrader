package main

import (
	"encoding/json"
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

	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func main() {
	log.Printf("Loading flags...")
	parseCLFlags()
	var err error
	log.Println("Loading engine...")
	engine.Bot, err = engine.New()
	if err != nil {
		log.Fatalf("Failed to initialise engine. Err: %s", err)
	}

	engine.Bot.Settings = engine.Settings{
		DisableExchangeAutoPairUpdates: true,
		Verbose:                        verboseOverride,
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
		if _, ok := wrapperConfig.Exchanges[strings.ToLower(name)]; !ok {
			wrapperConfig.Exchanges[strings.ToLower(name)] = &config.APICredentialsConfig{}
		}
		if shouldLoadExchange(name) {
			err = engine.LoadExchange(name, true, &wg)
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

	for x := range engine.Bot.Exchanges {
		base := engine.Bot.Exchanges[x].GetBase()
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
			name := engine.Bot.Exchanges[num].GetName()
			authenticated := setExchangeAPIKeys(name, wrapperConfig.Exchanges, base)
			wrapperResult := ExchangeResponses{
				ID:                 fmt.Sprintf("Exchange%v", num),
				ExchangeName:       name,
				APIKeysSet:         authenticated,
				AssetPairResponses: testWrappers(engine.Bot.Exchanges[num], base, &wrapperConfig),
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

	saveConfig(&wrapperConfig)

	if strings.EqualFold(outputOverride, "Console") {
		outputToConsole(exchangeResponses)
	}
	if strings.EqualFold(outputOverride, "JSON") {
		outputToJSON(exchangeResponses)
	}
	if strings.EqualFold(outputOverride, "HTML") {
		outputToHTML(exchangeResponses)
	}
}

func parseCLFlags() {
	flag.StringVar(&exchangesToUseOverride, "exchanges", "", "a + delimited list of exchange names to run tests against eg -exchanges=bitfinex+anx")
	flag.StringVar(&exchangesToExcludeOverride, "excluded-exchanges", "", "a + delimited list of exchange names to ignore when they're being temperamental eg -exchangesToExlude=lbank")
	flag.StringVar(&assetTypeOverride, "asset", "", "the asset type to run tests against (where applicable)")
	flag.StringVar(&currencyPairOverride, "currency", "", "the currency to run tests against (where applicable)")
	flag.StringVar(&outputOverride, "output", "HTML", "JSON, HTML or Console")
	flag.BoolVar(&authenticatedOnly, "auth-only", false, "skip any wrapper function that doesn't require auth")
	flag.BoolVar(&verboseOverride, "verbose", false, "verbose CL output")
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
	var authenticated bool

	if base.API.CredentialsValidator.RequiresKey && keys[lowerExchangeName].Key == "" {
		keys[lowerExchangeName].Key = config.DefaultAPIKey
	}
	if base.API.CredentialsValidator.RequiresSecret && keys[lowerExchangeName].Secret == "" {
		keys[lowerExchangeName].Secret = config.DefaultAPISecret
	}
	if base.API.CredentialsValidator.RequiresPEM && keys[lowerExchangeName].PEMKey == "" {
		keys[lowerExchangeName].PEMKey = "PEM"
	}
	if base.API.CredentialsValidator.RequiresClientID && keys[lowerExchangeName].ClientID == "" {
		keys[lowerExchangeName].ClientID = config.DefaultAPIClientID
	}
	if keys[lowerExchangeName].OTPSecret == "" {
		keys[lowerExchangeName].OTPSecret = "-" // Ensure OTP is available for use
	}

	base.API.Credentials.Key = keys[lowerExchangeName].Key
	base.Config.API.Credentials.Key = keys[lowerExchangeName].Key

	base.API.Credentials.Secret = keys[lowerExchangeName].Secret
	base.Config.API.Credentials.Secret = keys[lowerExchangeName].Secret

	base.API.Credentials.ClientID = keys[lowerExchangeName].ClientID
	base.Config.API.Credentials.ClientID = keys[lowerExchangeName].ClientID

	if keys[lowerExchangeName].OTPSecret != "-" {
		base.Config.API.Credentials.OTPSecret = keys[lowerExchangeName].OTPSecret
	}

	base.API.AuthenticatedSupport = true
	base.API.AuthenticatedWebsocketSupport = true
	base.Config.API.AuthenticatedSupport = true
	base.Config.API.AuthenticatedWebsocketSupport = true

	authenticated = base.ValidateAPICredentials()

	return authenticated
}

func parseOrderSide(orderSide string) exchange.OrderSide {
	switch orderSide {
	case exchange.AnyOrderSide.ToString():
		return exchange.AnyOrderSide
	case exchange.BuyOrderSide.ToString():
		return exchange.BuyOrderSide
	case exchange.SellOrderSide.ToString():
		return exchange.SellOrderSide
	case exchange.BidOrderSide.ToString():
		return exchange.BidOrderSide
	case exchange.AskOrderSide.ToString():
		return exchange.AskOrderSide
	default:
		log.Printf("Orderside '%v' not recognised, defaulting to BUY", orderSide)
		return exchange.BuyOrderSide
	}
}

func parseOrderType(orderType string) exchange.OrderType {
	switch orderType {
	case exchange.AnyOrderType.ToString():
		return exchange.AnyOrderType
	case exchange.LimitOrderType.ToString():
		return exchange.LimitOrderType
	case exchange.MarketOrderType.ToString():
		return exchange.MarketOrderType
	case exchange.ImmediateOrCancelOrderType.ToString():
		return exchange.ImmediateOrCancelOrderType
	case exchange.StopOrderType.ToString():
		return exchange.StopOrderType
	case exchange.TrailingStopOrderType.ToString():
		return exchange.TrailingStopOrderType
	case exchange.UnknownOrderType.ToString():
		return exchange.UnknownOrderType
	default:
		log.Printf("OrderType '%v' not recognised, defaulting to LIMIT", orderTypeOverride)
		return exchange.LimitOrderType
	}
}

func testWrappers(e exchange.IBotExchange, base *exchange.Base, config *Config) []ExchangeAssetPairResponses {
	var response []ExchangeAssetPairResponses
	assetTypes := base.GetAssetTypes()
	testOrderSide := parseOrderSide(config.OrderSubmission.OrderSide)
	testOrderType := parseOrderType(config.OrderSubmission.OrderType)
	if assetTypeOverride != "" {
		assetTypes = asset.Items{asset.Item(assetTypeOverride)}
	}
	for i := range assetTypes {
		var msg string
		var p currency.Pair
		log.Printf("%v %v", base.GetName(), assetTypes[i])
		if _, ok := base.Config.CurrencyPairs.Pairs[assetTypes[i]]; !ok {
			continue
		}

		switch {
		case currencyPairOverride != "":
			p = currency.NewPairFromString(currencyPairOverride)
		case len(base.Config.CurrencyPairs.Pairs[assetTypes[i]].Enabled) == 0:
			if len(base.Config.CurrencyPairs.Pairs[assetTypes[i]].Available) == 0 {
				log.Printf("%v has no enabled or available currencies. Skipping", base.GetName())
				continue
			}
			p = base.Config.CurrencyPairs.Pairs[assetTypes[i]].Available.GetRandomPair()
		default:
			p = base.Config.CurrencyPairs.Pairs[assetTypes[i]].Enabled.GetRandomPair()
		}

		responseContainer := ExchangeAssetPairResponses{
			AssetType:    assetTypes[i],
			CurrencyPair: p,
		}

		log.Printf("Setup config for %v %v %v", base.GetName(), assetTypes[i], p)
		err := e.Setup(base.Config)
		if err != nil {
			log.Printf("%v Could not load config, skipping. Error: '%v'", base.GetName(), err)
			continue
		}
		log.Printf("Executing wrappers for %v %v %v", base.GetName(), assetTypes[i], p)

		if !authenticatedOnly {
			var r1 ticker.Price
			r1, err = e.FetchTicker(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "FetchTicker",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{r1}),
			})

			var r2 ticker.Price
			r2, err = e.UpdateTicker(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "UpdateTicker",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{r2}),
			})

			var r3 orderbook.Base
			r3, err = e.FetchOrderbook(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++

			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "FetchOrderbook",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{r3}),
			})

			var r4 orderbook.Base
			r4, err = e.UpdateOrderbook(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++

			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "UpdateOrderbook",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{r4}),
			})

			var r5 []string
			r5, err = e.FetchTradablePairs(assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{assetTypes[i]}),
				Function:   "FetchTradablePairs",
				Error:      msg,
				Response:   jsonifyInterface([]interface{}{r5}),
			})
			// r6
			err = e.UpdateTradablePairs(false)
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
		}

		var r7 exchange.AccountInfo
		r7, err = e.GetAccountInfo()
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetAccountInfo",
			Error:    msg,
			Response: jsonifyInterface([]interface{}{r7}),
		})

		var r8 []exchange.TradeHistory
		r8, err = e.GetExchangeHistory(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
			Function:   "GetExchangeHistory",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r8}),
		})

		var r9 []exchange.FundHistory
		r9, err = e.GetFundingHistory()
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetFundingHistory",
			Error:    msg,
			Response: jsonifyInterface([]interface{}{r9}),
		})

		feeType := exchange.FeeBuilder{
			FeeType:       exchange.CryptocurrencyTradeFee,
			Pair:          p,
			PurchasePrice: config.OrderSubmission.Price,
			Amount:        config.OrderSubmission.Amount,
		}
		var r10 float64
		r10, err = e.GetFeeByType(&feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{feeType}),
			Function:   "GetFeeByType-Trade",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r10}),
		})

		s := &exchange.OrderSubmission{
			Pair:      p,
			OrderSide: testOrderSide,
			OrderType: testOrderType,
			Amount:    config.OrderSubmission.Amount,
			Price:     config.OrderSubmission.Price,
			ClientID:  config.OrderSubmission.OrderID,
		}
		var r11 exchange.SubmitOrderResponse
		r11, err = e.SubmitOrder(s)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{*s}),
			Function:   "SubmitOrder",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r11}),
		})

		modifyRequest := exchange.ModifyOrder{
			OrderID:      config.OrderSubmission.OrderID,
			OrderType:    testOrderType,
			OrderSide:    testOrderSide,
			CurrencyPair: p,
			Price:        config.OrderSubmission.Price,
			Amount:       config.OrderSubmission.Amount,
		}
		var r12 string
		r12, err = e.ModifyOrder(&modifyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{modifyRequest}),
			Function:   "ModifyOrder",
			Error:      msg,
			Response:   r12,
		})
		// r13
		cancelRequest := exchange.OrderCancellation{
			Side:         testOrderSide,
			CurrencyPair: p,
			OrderID:      config.OrderSubmission.OrderID,
		}
		err = e.CancelOrder(&cancelRequest)
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

		var r14 exchange.CancelAllOrdersResponse
		r14, err = e.CancelAllOrders(&cancelRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{cancelRequest}),
			Function:   "CancelAllOrders",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r14}),
		})

		var r15 exchange.OrderDetail
		r15, err = e.GetOrderInfo(config.OrderSubmission.OrderID)
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

		historyRequest := exchange.GetOrdersRequest{
			OrderType:  testOrderType,
			OrderSide:  testOrderSide,
			Currencies: []currency.Pair{p},
		}
		var r16 []exchange.OrderDetail
		r16, err = e.GetOrderHistory(&historyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{historyRequest}),
			Function:   "GetOrderHistory",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r16}),
		})

		orderRequest := exchange.GetOrdersRequest{
			OrderType:  testOrderType,
			OrderSide:  testOrderSide,
			Currencies: []currency.Pair{p},
		}
		var r17 []exchange.OrderDetail
		r17, err = e.GetActiveOrders(&orderRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{orderRequest}),
			Function:   "GetActiveOrders",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r17}),
		})

		var r18 string
		r18, err = e.GetDepositAddress(p.Base, "")
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{p.Base, ""}),
			Function:   "GetDepositAddress",
			Error:      msg,
			Response:   r18,
		})

		feeType = exchange.FeeBuilder{
			FeeType:       exchange.CryptocurrencyWithdrawalFee,
			Pair:          p,
			PurchasePrice: config.OrderSubmission.Price,
			Amount:        config.OrderSubmission.Amount,
		}
		var r19 float64
		r19, err = e.GetFeeByType(&feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{feeType}),
			Function:   "GetFeeByType-Crypto-Withdraw",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r19}),
		})

		genericWithdrawRequest := exchange.GenericWithdrawRequestInfo{
			Amount:   config.OrderSubmission.Amount,
			Currency: p.Quote,
		}
		withdrawRequest := exchange.CryptoWithdrawRequest{
			GenericWithdrawRequestInfo: genericWithdrawRequest,
			Address:                    withdrawAddressOverride,
		}
		var r20 string
		r20, err = e.WithdrawCryptocurrencyFunds(&withdrawRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{withdrawRequest}),
			Function:   "WithdrawCryptocurrencyFunds",
			Error:      msg,
			Response:   r20,
		})

		feeType = exchange.FeeBuilder{
			FeeType:             exchange.InternationalBankWithdrawalFee,
			Pair:                p,
			PurchasePrice:       config.OrderSubmission.Price,
			Amount:              config.OrderSubmission.Amount,
			FiatCurrency:        currency.AUD,
			BankTransactionType: exchange.WireTransfer,
		}
		var r21 float64
		r21, err = e.GetFeeByType(&feeType)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{feeType}),
			Function:   "GetFeeByType-FIAT-Withdraw",
			Error:      msg,
			Response:   jsonifyInterface([]interface{}{r21}),
		})

		fiatWithdrawRequest := exchange.FiatWithdrawRequest{
			GenericWithdrawRequestInfo:    genericWithdrawRequest,
			BankAccountName:               config.BankDetails.BankAccountName,
			BankAccountNumber:             config.BankDetails.BankAccountNumber,
			SwiftCode:                     config.BankDetails.SwiftCode,
			IBAN:                          config.BankDetails.Iban,
			BankCity:                      config.BankDetails.BankCity,
			BankName:                      config.BankDetails.BankName,
			BankAddress:                   config.BankDetails.BankAddress,
			BankCountry:                   config.BankDetails.BankCountry,
			BankPostalCode:                config.BankDetails.BankPostalCode,
			BankCode:                      config.BankDetails.BankCode,
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
		}
		var r22 string
		r22, err = e.WithdrawFiatFunds(&fiatWithdrawRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{fiatWithdrawRequest}),
			Function:   "WithdrawFiatFunds",
			Error:      msg,
			Response:   r22,
		})

		var r23 string
		r23, err = e.WithdrawFiatFundsToInternationalBank(&fiatWithdrawRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{fiatWithdrawRequest}),
			Function:   "WithdrawFiatFundsToInternationalBank",
			Error:      msg,
			Response:   r23,
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
	file, err := os.OpenFile("wrapperconfig.json", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return config, err
	}
	defer file.Close()
	keys, err := ioutil.ReadAll(file)
	if err != nil {
		return config, err
	}

	return config, common.JSONDecode(keys, &config)
}

func saveConfig(config *Config) {
	log.Println("JSONifying config...")
	jsonOutput, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		log.Fatalf("Encountered error encoding JSON: %v", err)
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Printf("Encountered error retrieving output directory: %v", err)
		return
	}

	log.Printf("Outputting to: %v", filepath.Join(dir, "wrapperconfig.json"))
	err = common.WriteFile(filepath.Join(dir, "wrapperconfig.json"), jsonOutput)
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
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Printf("Encountered error retrieving output directory: %v", err)
		return
	}

	log.Printf("Outputting to: %v", filepath.Join(dir, fmt.Sprintf("%v.json", outputFileName)))
	err = common.WriteFile(filepath.Join(dir, fmt.Sprintf("%v.json", outputFileName)), jsonOutput)
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

	defer file.Close()
	err = tmpl.Execute(file, exchangeResponses)
	if err != nil {
		log.Print(err)
		return
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
				log.Printf("Currency:\t%v\n", exchangeResponses[i].AssetPairResponses[j].CurrencyPair)
				log.Printf("Wrapper Params:\t%v\n", exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].SentParams)
				if exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].Error != "" {
					totalErrors++
					log.Printf("Error:\t%v", exchangeResponses[i].AssetPairResponses[j].EndpointResponses[k].Error)
				} else {
					log.Print("Error:\tnone")
				}
				log.Println()
			}
		}
		log.Println()
	}
}
