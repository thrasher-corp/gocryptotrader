package main

import (
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

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var orderType string
var output string
var orderSide string
var currencyPair string
var wrapperName string
var assetType string
var orderPrice float64
var orderAmount float64
var withdrawAddress string
var authenticatedOnly bool
var exchangesToUse []string

func parseCLFlags() {
	exchangesFlag := flag.String("exchanges", "", "A + delimited list of exchange names to run tests against eg -exchanges=bitfinex+anx")
	assetTypeFlag := flag.String("asset", "", "The asset type to run tests against (where applicable)")
	currencyPairFlag := flag.String("currency", "", "The currency to run tests against (where applicable)")
	outputFlag := flag.String("output", "HTML", "JSON, HTML or Console")
	authenticatedOnlyFlag := flag.Bool("authonly", false, "Skip any wrapper function that doesn't require auth")
	orderSideFlag := flag.String("orderSide", "BUY", "The order type for all order based wrapper tests")
	orderTypeFlag := flag.String("orderType", "LIMIT", "The order type for all order based wrapper tests")
	orderAmountFlag := flag.Float64("orderAmount", 100000000, "The order amount for all order based wrapper tests")
	orderPriceFlag := flag.Float64("orderPrice", 100000000, "The order price for all order based wrapper tests")
	wrapperFlag := flag.String("wrapper", "", "Specify a singular wrapper to run against. eg -wrapper=SubmitOrder")
	withdrawAddressFlag := flag.String("withdrawWallet", "", "Withdraw wallet address")
	flag.Parse()
	if *exchangesFlag != "" {
		exchangesToUse = strings.Split(*exchangesFlag, "+")
	}
	currencyPair = *currencyPairFlag
	assetType = *assetTypeFlag
	wrapperName = *wrapperFlag
	orderType = strings.ToUpper(*orderTypeFlag)
	orderSide = strings.ToUpper(*orderSideFlag)
	authenticatedOnly = *authenticatedOnlyFlag
	output = *outputFlag
	orderPrice = *orderPriceFlag
	orderAmount = *orderAmountFlag
	withdrawAddress = *withdrawAddressFlag
}

func main() {
	parseCLFlags()
	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		log.Fatalf("Failed to initialise engine. Err: %s", err)
	}

	engine.Bot.Settings = engine.Settings{
		DisableExchangeAutoPairUpdates: true,
		Verbose:                        false,
	}

	log.Printf("Loading exchanges..")

	var wg sync.WaitGroup
	for x := range exchange.Exchanges {
		name := exchange.Exchanges[x]
		err = engine.LoadExchange(name, true, &wg)
		if err != nil {
			log.Printf("Failed to load exchange %s. Err: %s", name, err)
			continue
		}
	}
	wg.Wait()

	log.Println("Done.")
	log.Printf("Testing exchange wrappers..")

	wg = sync.WaitGroup{}
	var exchangeResponses []ExchangeResponses
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if config.WalletAddress != "" && withdrawAddress == "" {
		withdrawAddress = config.WalletAddress
	}

	for x := range engine.Bot.Exchanges {
		if len(exchangesToUse) > 0 {
			var found bool
			for i := range exchangesToUse {
				if strings.EqualFold(engine.Bot.Exchanges[x].GetName(), exchangesToUse[i]) {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		base := engine.Bot.Exchanges[x].GetBase()
		if !base.Config.Enabled {
			log.Printf("Exchange %v not enabled, skipping", base.GetName())
			continue
		}
		base.Config.Verbose = false
		base.Config.HTTPDebugging = false
		base.Verbose = false
		base.HTTPDebugging = false
		wg.Add(1)

		go func(num int) {
			name := engine.Bot.Exchanges[num].GetName()
			authenticated := setExchangeAPIKeys(name, config.APIKEys, base)
			wrapperResult := ExchangeResponses{
				ID:                 fmt.Sprintf("Exchange%v", num),
				ExchangeName:       name,
				APIKeysSet:         authenticated,
				AssetPairResponses: testWrappers(engine.Bot.Exchanges[num], base, config.BankDetails),
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

	if strings.EqualFold(output, "Console") {
		outputToConsole(exchangeResponses)
	}
	if strings.EqualFold(output, "JSON") {
		outputToJSON(exchangeResponses)
	}
	if strings.EqualFold(output, "HTML") {
		outputToHTML(exchangeResponses)
	}
}

func setExchangeAPIKeys(name string, keys map[string]Key, base *exchange.Base) bool {
	keyName := strings.ToLower(name)
	var authenticated bool
	if _, ok := keys[keyName]; ok {
		if keys[keyName].APIKey != "" {
			base.API.Credentials.Key = keys[keyName].APIKey
			base.Config.API.Credentials.Key = keys[keyName].APIKey
			base.API.AuthenticatedSupport = true
			base.API.AuthenticatedWebsocketSupport = true
			base.Config.API.AuthenticatedSupport = true
			base.Config.API.AuthenticatedWebsocketSupport = true
			if keys[keyName].APISecret != "" {
				base.API.Credentials.Secret = keys[keyName].APISecret
				base.Config.API.Credentials.Secret = keys[keyName].APISecret
			}
			if keys[keyName].ClientID != "" {
				base.API.Credentials.ClientID = keys[keyName].ClientID
				base.Config.API.Credentials.ClientID = keys[keyName].ClientID
			}
			authenticated = base.ValidateAPICredentials()
		}
	}
	return authenticated
}

func parseOrderSide() exchange.OrderSide {
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

func parseOrderType() exchange.OrderType {
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
		log.Printf("OrderType '%v' not recognised, defaulting to MARKET", orderType)
		return exchange.MarketOrderType
	}
}
func testWrappers(e exchange.IBotExchange, base *exchange.Base, bankDetails Bank) []ExchangeAssetPairResponses {
	var response []ExchangeAssetPairResponses
	assetTypes := base.GetAssetTypes()
	testOrderSide := parseOrderSide()
	testOrderType := parseOrderType()

	for i := range assetTypes {
		var msg string
		var p currency.Pair
		log.Printf("%v %v", base.GetName(), assetTypes[i])
		if _, ok := base.Config.CurrencyPairs.Pairs[assetTypes[i]]; !ok {
			continue
		}
		if len(base.Config.CurrencyPairs.Pairs[assetTypes[i]].Enabled) == 0 {
			if len(base.Config.CurrencyPairs.Pairs[assetTypes[i]].Available) == 0 {
				continue
			}
			p = base.Config.CurrencyPairs.Pairs[assetTypes[i]].Available[0]
		} else {
			p = base.Config.CurrencyPairs.Pairs[assetTypes[i]].Enabled[0]
		}
		responseContainer := ExchangeAssetPairResponses{
			AssetType:    assetTypes[i],
			CurrencyPair: p,
		}
		e.Setup(base.Config)
		if !authenticatedOnly {
			r1, err := e.FetchTicker(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "FetchTicker",
				Error:      msg,
				Response:   r1,
			})

			r2, err := e.UpdateTicker(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "UpdateTicker",
				Error:      msg,
				Response:   r2,
			})

			r3, err := e.FetchOrderbook(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++

			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "FetchOrderbook",
				Error:      msg,
				Response:   r3,
			})

			r4, err := e.UpdateOrderbook(p, assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++

			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
				Function:   "UpdateOrderbook",
				Error:      msg,
				Response:   r4,
			})

			r5, err := e.FetchTradablePairs(assetTypes[i])
			msg = ""
			if err != nil {
				msg = err.Error()
				responseContainer.ErrorCount++
			}
			responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
				SentParams: jsonifyInterface([]interface{}{assetTypes[i]}),
				Function:   "FetchTradablePairs",
				Error:      msg,
				Response:   r5,
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
			})
		}
		r7, err := e.GetAccountInfo()
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetAccountInfo",
			Error:    msg,
			Response: r7,
		})

		r8, err := e.GetExchangeHistory(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{p, assetTypes[i]}),
			Function:   "GetExchangeHistory",
			Error:      msg,
			Response:   r8,
		})

		r9, err := e.GetFundingHistory()
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetFundingHistory",
			Error:    msg,
			Response: r9,
		})

		s := &exchange.OrderSubmission{
			Pair:      p,
			OrderSide: testOrderSide,
			OrderType: testOrderType,
			Amount:    orderAmount,
			Price:     orderPrice,
			ClientID:  base.API.Credentials.ClientID,
		}
		r10, err := e.SubmitOrder(s)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{*s}),
			Function:   "SubmitOrder",
			Error:      msg,
			Response:   r10,
		})

		var orderID string
		if r10.IsOrderPlaced {
			orderID = r10.OrderID
		}

		orderRequest := exchange.GetOrdersRequest{
			OrderType:  testOrderType,
			OrderSide:  testOrderSide,
			Currencies: []currency.Pair{p},
		}
		r16, err := e.GetActiveOrders(&orderRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{orderRequest}),
			Function:   "GetActiveOrders",
			Error:      msg,
			Response:   r16,
		})
		if len(r16) > 0 {
			orderID = r16[0].ID
		}

		modifyRequest := exchange.ModifyOrder{
			OrderID:      orderID,
			OrderType:    testOrderType,
			OrderSide:    testOrderSide,
			CurrencyPair: p,
			Price:        orderPrice,
			Amount:       orderAmount,
		}
		r11, err := e.ModifyOrder(&modifyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{modifyRequest}),
			Function:   "ModifyOrder",
			Error:      msg,
			Response:   r11,
		})
		// r12
		cancelRequest := exchange.OrderCancellation{
			Side:         testOrderSide,
			CurrencyPair: p,
			OrderID:      orderID,
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
		})
		r13, err := e.CancelAllOrders(&cancelRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{cancelRequest}),
			Function:   "CancelAllOrders",
			Error:      msg,
			Response:   r13,
		})

		r14, err := e.GetOrderInfo(orderID)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{orderID}),
			Function:   "GetOrderInfo",
			Error:      msg,
			Response:   r14,
		})
		historyRequest := exchange.GetOrdersRequest{
			OrderType:  testOrderType,
			OrderSide:  testOrderSide,
			Currencies: []currency.Pair{p},
		}
		r15, err := e.GetOrderHistory(&historyRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{historyRequest}),
			Function:   "GetOrderHistory",
			Error:      msg,
			Response:   r15,
		})

		r17, err := e.GetDepositAddress(p.Base, "")
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{p.Base, ""}),
			Function:   "GetDepositAddress",
			Error:      msg,
			Response:   r17,
		})

		genericWithdrawRequest := exchange.GenericWithdrawRequestInfo{
			Amount:   orderAmount,
			Currency: p.Quote,
		}
		withdrawRequest := exchange.CryptoWithdrawRequest{
			GenericWithdrawRequestInfo: genericWithdrawRequest,
			Address:                    withdrawAddress,
		}
		r18, err := e.WithdrawCryptocurrencyFunds(&withdrawRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{withdrawRequest}),
			Function:   "WithdrawCryptocurrencyFunds",
			Error:      msg,
			Response:   r18,
		})
		fiatWithdrawRequest := exchange.FiatWithdrawRequest{
			GenericWithdrawRequestInfo:    genericWithdrawRequest,
			BankAccountName:               bankDetails.BankAccountName,
			BankAccountNumber:             bankDetails.BankAccountNumber,
			SwiftCode:                     bankDetails.SwiftCode,
			IBAN:                          bankDetails.Iban,
			BankCity:                      bankDetails.BankCity,
			BankName:                      bankDetails.BankName,
			BankAddress:                   bankDetails.BankAddress,
			BankCountry:                   bankDetails.BankCountry,
			BankPostalCode:                bankDetails.BankPostalCode,
			BankCode:                      bankDetails.BankCode,
			IsExpressWire:                 bankDetails.IsExpressWire,
			RequiresIntermediaryBank:      bankDetails.RequiresIntermediaryBank,
			IntermediaryBankName:          bankDetails.IntermediaryBankName,
			IntermediaryBankAccountNumber: bankDetails.IntermediaryBankAccountNumber,
			IntermediarySwiftCode:         bankDetails.IntermediarySwiftCode,
			IntermediaryIBAN:              bankDetails.IntermediaryIban,
			IntermediaryBankCity:          bankDetails.IntermediaryBankCity,
			IntermediaryBankAddress:       bankDetails.IntermediaryBankAddress,
			IntermediaryBankCountry:       bankDetails.IntermediaryBankCountry,
			IntermediaryBankPostalCode:    bankDetails.IntermediaryBankPostalCode,
			IntermediaryBankCode:          bankDetails.IntermediaryBankCode,
		}
		r19, err := e.WithdrawFiatFunds(&fiatWithdrawRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{fiatWithdrawRequest}),
			Function:   "WithdrawFiatFunds",
			Error:      msg,
			Response:   r19,
		})

		r20, err := e.WithdrawFiatFundsToInternationalBank(&fiatWithdrawRequest)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			SentParams: jsonifyInterface([]interface{}{fiatWithdrawRequest}),
			Function:   "WithdrawFiatFundsToInternationalBank",
			Error:      msg,
			Response:   r20,
		})
		response = append(response, responseContainer)
	}
	return response
}

func jsonifyInterface(params []interface{}) string {
	response, _ := common.JSONEncode(params)
	return string(response)
}

func loadConfig() (Config, error) {
	var config Config
	file, err := os.OpenFile("wrapperconfig.json", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return config, err
	}

	keys, err := ioutil.ReadAll(file)
	if err != nil {
		return config, err
	}

	return config, common.JSONDecode(keys, &config)
}

func outputToJSON(exchangeResponses []ExchangeResponses) {
	log.Println("JSONifying results...")
	json, err := common.JSONEncode(exchangeResponses)
	if err != nil {
		log.Fatalf("Encountered error encoding JSON: %v", err)
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Printf("Encounted error retrieving output directory: %v", err)
		return
	}

	log.Printf("Outputting to: %v", dir+"\\output.json")
	err = common.WriteFile(filepath.Join(dir, "output.json"), json)
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

	fileName := "report.html"
	tmpl, err := template.New("report.tmpl").ParseFiles(filepath.Join(dir, "report.tmpl"))
	if err != nil {
		log.Print(err)
		return
	}

	file, err := os.Create(filepath.Join(dir, fileName))
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

type Config struct {
	WalletAddress string         `json:"walletAddress"`
	BankDetails   Bank           `json:"bankAccount"`
	APIKEys       map[string]Key `json:"exchanges"`
}

type Key struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
	ClientID  string `json:"clientId"`
}

type ExchangeResponses struct {
	ID                 string
	ExchangeName       string                       `json:"exchangeName"`
	AssetPairResponses []ExchangeAssetPairResponses `json:"responses"`
	ErrorCount         int64                        `json:"errorCount"`
	APIKeysSet         bool                         `json:"apiKeysSet"`
}

type ExchangeAssetPairResponses struct {
	ErrorCount        int64              `json:"errorCount"`
	AssetType         asset.Item         `json:"asset"`
	CurrencyPair      currency.Pair      `json:"currency"`
	EndpointResponses []EndpointResponse `json:"responses"`
}

type EndpointResponse struct {
	Function   string      `json:"function"`
	Error      string      `json:"error"`
	Response   interface{} `json:"response"`
	SentParams string      `json:"sentParams"`
}

type Bank struct {
	BankAccountName               string  `json:"bankAccountName"`
	BankAccountNumber             float64 `json:"bankAccountNumber"`
	BankAddress                   string  `json:"bankAddress"`
	BankCity                      string  `json:"bankCity"`
	BankCountry                   string  `json:"bankCountry"`
	BankName                      string  `json:"bankName"`
	BankPostalCode                string  `json:"bankPostalCode"`
	Iban                          string  `json:"iban"`
	IntermediaryBankAccountName   string  `json:"intermediaryBankAccountName"`
	IntermediaryBankAccountNumber float64 `json:"intermediaryBankAccountNumber"`
	IntermediaryBankAddress       string  `json:"intermediaryBankAddress"`
	IntermediaryBankCity          string  `json:"intermediaryBankCity"`
	IntermediaryBankCountry       string  `json:"intermediaryBankCountry"`
	IntermediaryBankName          string  `json:"intermediaryBankName"`
	IntermediaryBankPostalCode    string  `json:"intermediaryBankPostalCode"`
	IntermediaryIban              string  `json:"intermediaryIban"`
	IntermediaryIsExpressWire     bool    `json:"intermediaryIsExpressWire"`
	IntermediarySwiftCode         string  `json:"intermediarySwiftCode"`
	IsExpressWire                 bool    `json:"isExpressWire"`
	RequiresIntermediaryBank      bool    `json:"requiresIntermediaryBank"`
	SwiftCode                     string  `json:"swiftCode"`
	BankCode                      float64 `json:"bankCode"`
	IntermediaryBankCode          float64 `json:"intermediaryBankCode"`
}
