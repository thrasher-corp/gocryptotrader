package main

import (
	"log"
	"os"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	totalWrappers = 20
)

type Key struct {
	ExchangeName, APIKey, APISecret, ClientID string
}
type SuperDuperResponse struct {
	ExchangeName string                    `json:"exchangeName"`
	Responses    []ExchangeWrapperResponse `json:"responses"`
}

type ExchangeWrapperResponse struct {
	AssetType    asset.Item    `json:"asset"`
	CurrencyPair currency.Pair `json:"currency"`
	Responses    []Response    `json:"responses"`
}

type Response struct {
	Function string      `json:"function"`
	Error    string      `json:"error"`
	Response interface{} `json:"response"`
}

func SetupKeys() map[string]Key {
	return make(map[string]Key)
}

func main() {
	//keys := SetupKeys()
	var verbose bool
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
		err := engine.LoadExchange(name, true, &wg)
		if err != nil {
			log.Printf("Failed to load exchange %s. Err: %s", name, err)
			continue
		}
	}
	wg.Wait()
	log.Println("Done.")
	log.Printf("Testing exchange wrappers..")
	wg = sync.WaitGroup{}
	superFinalResponse := []SuperDuperResponse{}
	for x := range engine.Bot.Exchanges {
		base := engine.Bot.Exchanges[x].GetBase()
		if !base.Config.Enabled {
			log.Printf("Exchange %v not enabled, skipping", base.GetName())
			continue
		}
		base.Config.Verbose = false
		base.Verbose = false
		base.Config.HTTPDebugging = false
		base.HTTPDebugging = false
		wg.Add(1)
		go func(num int) {
			name := engine.Bot.Exchanges[num].GetName()
			// Set the APIKEYS MANNN
			// base.Config.API.Credentials.Key = keys[name].APIKey
			// base.Config.API.Credentials.Secret = keys[name].APISecret
			// base.Config.API.Credentials.ClientID = keys[name].ClientID
			authenticated := base.ValidateAPICredentials()
			superFinalResponse = append(superFinalResponse, SuperDuperResponse{
				ExchangeName: name,
				Responses:    testWrappers(engine.Bot.Exchanges[num], base, authenticated, false),
			})
			wg.Done()
		}(x)
	}
	wg.Wait()
	log.Println("Done.")
	log.Println()
	var totalErrors int64
	for i := range superFinalResponse {
		log.Printf("------------%v Results-------------\n", superFinalResponse[i].ExchangeName)
		for j := range superFinalResponse[i].Responses {
			for k := range superFinalResponse[i].Responses[j].Responses {
				log.Printf("%v Result: %v", superFinalResponse[j].ExchangeName, k)
				log.Printf("Function:\t%v", superFinalResponse[i].Responses[j].Responses[k].Function)
				log.Printf("AssetType:\t%v", superFinalResponse[i].Responses[j].AssetType)
				log.Printf("Currency:\t%v\n", superFinalResponse[i].Responses[j].CurrencyPair)
				if superFinalResponse[i].Responses[j].Responses[k].Error != "" {
					totalErrors++
					log.Printf("Error:\t%v", superFinalResponse[i].Responses[j].Responses[k].Error)
				} else {
					log.Print("Error:\tnone")
				}
				if verbose {
					butts, err := common.JSONEncode(superFinalResponse[i].Responses[j].Responses[k].Response)
					if err != nil {
						log.Printf("JSON Error:\t%v", err)
					}
					log.Printf("Response:\t%s", butts)
				}
				log.Println()

			}
		}
		log.Println()

	}
	log.Println("JSONifying results...")
	json, err := common.JSONEncode(superFinalResponse)
	if err != nil {
		log.Println("WOAH NELLY, JSON STUFFED UP")
		return
	}
	dir, err := os.Getwd()
	if err != nil {
		log.Println("WOAH NELLY, DIRECTORY STUFFED UP")
		return
	}
	log.Printf("Outputting to: %v", dir+"\\output.json")

	err = common.WriteFile(dir+"\\output.json", json)
	if err != nil {
		log.Println("WOAH NELLY, OUTPUT STUFFED UP")
		return
	}
}

func testWrappers(e exchange.IBotExchange, base *exchange.Base, authenticated, verbose bool) []ExchangeWrapperResponse {
	var response []ExchangeWrapperResponse
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
		butts := ExchangeWrapperResponse{
			AssetType:    assetTypes[i],
			CurrencyPair: p,
		}
		r1, err := e.FetchTicker(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "FetchTicker",
			Error:    msg,
			Response: r1,
		})

		r2, err := e.UpdateTicker(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "UpdateTicker",
			Error:    msg,
			Response: r2,
		})

		r3, err := e.FetchOrderbook(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "FetchOrderbook",
			Error:    msg,
			Response: r3,
		})

		r4, err := e.UpdateOrderbook(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "UpdateOrderbook",
			Error:    msg,
			Response: r4,
		})

		r5, err := e.FetchTradablePairs(assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "FetchTradablePairs",
			Error:    msg,
			Response: r5,
		})
		// r6
		err = e.UpdateTradablePairs(false)
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "UpdateTradablePairs",
			Error:    msg,
		})

		if !authenticated {
			response = append(response, butts)
			continue
		}

		r7, err := e.GetAccountInfo()
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "GetAccountInfo",
			Error:    msg,
			Response: r7,
		})

		r8, err := e.GetExchangeHistory(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "GetExchangeHistory",
			Error:    msg,
			Response: r8,
		})

		r9, err := e.GetFundingHistory()
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
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
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "SubmitOrder",
			Error:    msg,
			Response: r10,
		})

		r16, err := e.GetActiveOrders(&exchange.GetOrdersRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
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
		}
		butts.Responses = append(butts.Responses, Response{
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
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "CancelOrder",
			Error:    msg,
		})

		r13, err := e.CancelAllOrders(&exchange.OrderCancellation{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "CancelAllOrders",
			Error:    msg,
			Response: r13,
		})

		r14, err := e.GetOrderInfo(orderID)
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "GetOrderInfo",
			Error:    msg,
			Response: r14,
		})

		r15, err := e.GetOrderHistory(&exchange.GetOrdersRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "GetOrderHistory",
			Error:    msg,
			Response: r15,
		})

		r17, err := e.GetDepositAddress(p.Base, "")
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "GetDepositAddress",
			Error:    msg,
			Response: r17,
		})

		r18, err := e.WithdrawCryptocurrencyFunds(&exchange.CryptoWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "WithdrawCryptocurrencyFunds",
			Error:    msg,
			Response: r18,
		})

		r19, err := e.WithdrawFiatFunds(&exchange.FiatWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "WithdrawFiatFunds",
			Error:    msg,
			Response: r19,
		})
		r20, err := e.WithdrawFiatFundsToInternationalBank(&exchange.FiatWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		butts.Responses = append(butts.Responses, Response{
			Function: "WithdrawFiatFundsToInternationalBank",
			Error:    msg,
			Response: r20,
		})
		response = append(response, butts)
	}
	return response
}
