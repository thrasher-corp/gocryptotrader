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

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

func main() {
	log.Println("Loading flags...")
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
		if _, ok := wrapperConfig.Exchanges[name]; !ok {
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

	exchs := engine.GetExchanges()
	for x := range exchs {
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

	return base.ValidateAPICredentials()
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
	assetTypes := base.GetAssetTypes()
	if assetTypeOverride != "" {
		if asset.IsValid(asset.Item(assetTypeOverride)) {
			assetTypes = asset.Items{asset.Item(assetTypeOverride)}
		} else {
			log.Printf("%v Asset Type '%v' not recognised, defaulting to exchange defaults", base.GetName(), assetTypeOverride)
		}
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
			log.Printf("%v Encountered error reloading config: '%v'", base.GetName(), err)
		}
		log.Printf("Executing wrappers for %v %v %v", base.GetName(), assetTypes[i], p)

		if !authenticatedOnly {
			var r1 *ticker.Price
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

			var r2 *ticker.Price
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

			var r3 *orderbook.Base
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

			var r4 *orderbook.Base
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

		var r7 account.Holdings
		r7, err = e.FetchAccountInfo()
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "FetchAccountInfo",
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

		s := &order.Submit{
			Pair:     p,
			Side:     testOrderSide,
			Type:     testOrderType,
			Amount:   config.OrderSubmission.Amount,
			Price:    config.OrderSubmission.Price,
			ClientID: config.OrderSubmission.OrderID,
		}
		var r11 order.SubmitResponse
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

		modifyRequest := order.Modify{
			ID:     config.OrderSubmission.OrderID,
			Type:   testOrderType,
			Side:   testOrderSide,
			Pair:   p,
			Price:  config.OrderSubmission.Price,
			Amount: config.OrderSubmission.Amount,
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
		cancelRequest := order.Cancel{
			Side: testOrderSide,
			Pair: p,
			ID:   config.OrderSubmission.OrderID,
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

		var r14 order.CancelAllResponse
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

		var r15 order.Detail
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

		historyRequest := order.GetOrdersRequest{
			Type:  testOrderType,
			Side:  testOrderSide,
			Pairs: []currency.Pair{p},
		}
		var r16 []order.Detail
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

		orderRequest := order.GetOrdersRequest{
			Type:  testOrderType,
			Side:  testOrderSide,
			Pairs: []currency.Pair{p},
		}
		var r17 []order.Detail
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

		withdrawRequest := withdraw.Request{
			Currency: p.Quote,
			Crypto: &withdraw.CryptoRequest{
				Address: withdrawAddressOverride,
			},
			Amount: config.OrderSubmission.Amount,
		}
		var r20 *withdraw.ExchangeResponse
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

		withdrawRequestFiat := withdraw.Request{
			Currency: p.Quote,
			Amount:   config.OrderSubmission.Amount,
			Fiat: &withdraw.FiatRequest{
				Bank: &banking.Account{
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
		var r22 *withdraw.ExchangeResponse
		r22, err = e.WithdrawFiatFunds(&withdrawRequestFiat)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{withdrawRequestFiat}),
			Function:   "WithdrawFiatFunds",
			Error:      msg,
			Response:   r22,
		})

		var r23 *withdraw.ExchangeResponse
		r23, err = e.WithdrawFiatFundsToInternationalBank(&withdrawRequestFiat)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{withdrawRequestFiat}),
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
				log.Printf("Currency:\t%v\n", exchangeResponses[i].AssetPairResponses[j].CurrencyPair)
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
