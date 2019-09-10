package main

import (
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

func main() {
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
	keys, err := loadKeys()
	if err != nil {
		log.Fatal(err)
	}

	for x := range engine.Bot.Exchanges {
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
			authenticated := setExchangeAPIKeys(name, keys, base)
			wrapperResult := ExchangeResponses{
				ID:                 fmt.Sprintf("Exchange%v", num),
				ExchangeName:       name,
				APIKeysSet:         authenticated,
				AssetPairResponses: testWrappers(engine.Bot.Exchanges[num], base),
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

	outputToConsole(exchangeResponses)
	outputToJSON(exchangeResponses)
	outputToHTML(exchangeResponses)
}

func setExchangeAPIKeys(name string, keys map[string]Key, base *exchange.Base) bool {
	keyName := strings.ToLower(name)
	var authenticated bool
	if _, ok := keys[keyName]; ok {
		if keys[keyName].APIKey != "" {
			base.API.Credentials.Key = keys[keyName].APIKey
			if keys[keyName].APISecret != "" {
				base.API.Credentials.Secret = keys[keyName].APISecret
			}
			if keys[keyName].ClientID != "" {
				base.API.Credentials.ClientID = keys[keyName].ClientID
			}
			authenticated = base.ValidateAPICredentials()
		}
	}
	return authenticated
}

func loadKeys() (map[string]Key, error) {
	file, err := os.OpenFile("keys.json", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	keys, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	keyMap := make(map[string]Key)
	return keyMap, common.JSONDecode(keys, &keyMap)
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

func testWrappers(e exchange.IBotExchange, base *exchange.Base) []ExchangeAssetPairResponses {
	var response []ExchangeAssetPairResponses
	assetTypes := base.GetAssetTypes()
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
		r1, err := e.FetchTicker(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "FetchTicker",
			Error:    msg,
			Response: r1,
		})

		r2, err := e.UpdateTicker(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "UpdateTicker",
			Error:    msg,
			Response: r2,
		})

		r3, err := e.FetchOrderbook(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++

		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "FetchOrderbook",
			Error:    msg,
			Response: r3,
		})

		r4, err := e.UpdateOrderbook(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++

		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "UpdateOrderbook",
			Error:    msg,
			Response: r4,
		})

		r5, err := e.FetchTradablePairs(assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "FetchTradablePairs",
			Error:    msg,
			Response: r5,
		})
		// r6
		err = e.UpdateTradablePairs(false)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "UpdateTradablePairs",
			Error:    msg,
		})

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
			Function: "GetExchangeHistory",
			Error:    msg,
			Response: r8,
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
			OrderSide: exchange.BuyOrderSide,
			OrderType: exchange.LimitOrderType,
			Amount:    1000000,
			Price:     10000000000,
			ClientID:  base.API.Credentials.ClientID,
		}
		r10, err := e.SubmitOrder(s)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "SubmitOrder",
			Error:    msg,
			Response: r10,
		})

		r16, err := e.GetActiveOrders(&exchange.GetOrdersRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetActiveOrders",
			Error:    msg,
			Response: r16,
		})
		var orderID string
		if len(r16) > 0 {
			orderID = r16[0].ID
		}

		r11, err := e.ModifyOrder(&exchange.ModifyOrder{
			OrderID: orderID,
		})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "ModifyOrder",
			Error:    msg,
			Response: r11,
		})
		// r12
		err = e.CancelOrder(&exchange.OrderCancellation{
			OrderID: orderID,
		})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "CancelOrder",
			Error:    msg,
		})

		r13, err := e.CancelAllOrders(&exchange.OrderCancellation{})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "CancelAllOrders",
			Error:    msg,
			Response: r13,
		})

		r14, err := e.GetOrderInfo(orderID)
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetOrderInfo",
			Error:    msg,
			Response: r14,
		})

		r15, err := e.GetOrderHistory(&exchange.GetOrdersRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetOrderHistory",
			Error:    msg,
			Response: r15,
		})

		r17, err := e.GetDepositAddress(p.Base, "")
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "GetDepositAddress",
			Error:    msg,
			Response: r17,
		})

		r18, err := e.WithdrawCryptocurrencyFunds(&exchange.CryptoWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "WithdrawCryptocurrencyFunds",
			Error:    msg,
			Response: r18,
		})

		r19, err := e.WithdrawFiatFunds(&exchange.FiatWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "WithdrawFiatFunds",
			Error:    msg,
			Response: r19,
		})
		r20, err := e.WithdrawFiatFundsToInternationalBank(&exchange.FiatWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
			responseContainer.ErrorCount++
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointResponse{
			Function: "WithdrawFiatFundsToInternationalBank",
			Error:    msg,
			Response: r20,
		})
		response = append(response, responseContainer)
	}
	return response
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
	Function string      `json:"function"`
	Error    string      `json:"error"`
	Response interface{} `json:"response"`
}
