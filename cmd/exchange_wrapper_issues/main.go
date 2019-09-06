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

type Response struct {
	Function string      `json:"function"`
	Error    error       `json:"error"`
	Response interface{} `json:"response"`
}

type ExchangeWrapperResponse struct {
	AssetType    asset.Item    `json:"asset"`
	CurrencyPair currency.Pair `json:"currency"`
	Responses    []Response    `json:"responses"`
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
	results := make(map[string][]ExchangeWrapperResponse)
	wg = sync.WaitGroup{}
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
			results[name] = testWrappers(engine.Bot.Exchanges[num], base, authenticated, false)
			wg.Done()
		}(x)
	}
	wg.Wait()
	log.Println("Done.")
	log.Println()
	var totalErrors int64
	for name, funcs := range results {
		log.Printf("------------%v Results-------------\n", name)
		for x := range funcs {
			for i := range funcs[x].Responses {
				log.Printf("%v Result: %v", name, i)
				log.Printf("Function:\t%v", funcs[x].Responses[i].Function)
				log.Printf("AssetType:\t%v", funcs[x].AssetType)
				log.Printf("Currency:\t%v\n", funcs[x].CurrencyPair)
				if funcs[x].Responses[i].Error != nil {
					totalErrors++
					log.Printf("Error:\t%v", funcs[x].Responses[i].Error)
				} else {
					log.Print("Error:\tnone")
				}
				if verbose {
					butts, err := common.JSONEncode(funcs[x].Responses[i].Response)
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
	json, err := common.JSONEncode(results)
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
		butts.Responses = append(butts.Responses, Response{
			Function: "FetchTicker",
			Error:    err,
			Response: r1,
		})

		r2, err := e.UpdateTicker(p, assetTypes[i])
		butts.Responses = append(butts.Responses, Response{
			Function: "UpdateTicker",
			Error:    err,
			Response: r2,
		})

		r3, err := e.FetchOrderbook(p, assetTypes[i])
		butts.Responses = append(butts.Responses, Response{
			Function: "FetchOrderbook",
			Error:    err,
			Response: r3,
		})

		r4, err := e.UpdateOrderbook(p, assetTypes[i])
		butts.Responses = append(butts.Responses, Response{
			Function: "UpdateOrderbook",
			Error:    err,
			Response: r4,
		})

		r5, err := e.FetchTradablePairs(asset.Spot)
		butts.Responses = append(butts.Responses, Response{
			Function: "FetchTradablePairs",
			Error:    err,
			Response: r5,
		})
		// r6
		err = e.UpdateTradablePairs(false)
		butts.Responses = append(butts.Responses, Response{
			Function: "UpdateTradablePairs",
			Error:    err,
		})

		if !authenticated {
			response = append(response, butts)
			continue
		}

		r7, err := e.GetAccountInfo()
		butts.Responses = append(butts.Responses, Response{
			Function: "GetAccountInfo",
			Error:    err,
			Response: r7,
		})

		r8, err := e.GetExchangeHistory(p, assetTypes[i])
		butts.Responses = append(butts.Responses, Response{
			Function: "GetExchangeHistory",
			Error:    err,
			Response: r8,
		})

		r9, err := e.GetFundingHistory()
		butts.Responses = append(butts.Responses, Response{
			Function: "GetFundingHistory",
			Error:    err,
			Response: r9,
		})

		s := &exchange.OrderSubmission{
			Pair:      p,
			OrderSide: exchange.BuyOrderSide,
			OrderType: exchange.LimitOrderType,
			Amount:    1000000,
			Price:     10000000000,
			ClientID:  "meow",
		}
		r10, err := e.SubmitOrder(s)
		butts.Responses = append(butts.Responses, Response{
			Function: "SubmitOrder",
			Error:    err,
			Response: r10,
		})

		r11, err := e.ModifyOrder(&exchange.ModifyOrder{})
		butts.Responses = append(butts.Responses, Response{
			Function: "ModifyOrder",
			Error:    err,
			Response: r11,
		})
		// r12
		err = e.CancelOrder(&exchange.OrderCancellation{})
		butts.Responses = append(butts.Responses, Response{
			Function: "CancelOrder",
			Error:    err,
		})

		r13, err := e.CancelAllOrders(&exchange.OrderCancellation{})
		butts.Responses = append(butts.Responses, Response{
			Function: "CancelAllOrders",
			Error:    err,
			Response: r13,
		})

		r14, err := e.GetOrderInfo("1")
		butts.Responses = append(butts.Responses, Response{
			Function: "GetOrderInfo",
			Error:    err,
			Response: r14,
		})

		r15, err := e.GetOrderHistory(&exchange.GetOrdersRequest{})
		butts.Responses = append(butts.Responses, Response{
			Function: "GetOrderHistory",
			Error:    err,
			Response: r15,
		})

		r16, err := e.GetActiveOrders(&exchange.GetOrdersRequest{})
		butts.Responses = append(butts.Responses, Response{
			Function: "GetActiveOrders",
			Error:    err,
			Response: r16,
		})

		r17, err := e.GetDepositAddress(currency.BTC, "")
		butts.Responses = append(butts.Responses, Response{
			Function: "GetDepositAddress",
			Error:    err,
			Response: r17,
		})

		r18, err := e.WithdrawCryptocurrencyFunds(&exchange.CryptoWithdrawRequest{})
		butts.Responses = append(butts.Responses, Response{
			Function: "WithdrawCryptocurrencyFunds",
			Error:    err,
			Response: r18,
		})

		r19, err := e.WithdrawFiatFunds(&exchange.FiatWithdrawRequest{})
		butts.Responses = append(butts.Responses, Response{
			Function: "WithdrawFiatFunds",
			Error:    err,
			Response: r19,
		})
		r20, err := e.WithdrawFiatFundsToInternationalBank(&exchange.FiatWithdrawRequest{})
		butts.Responses = append(butts.Responses, Response{
			Function: "WithdrawFiatFundsToInternationalBank",
			Error:    err,
			Response: r20,
		})
		response = append(response, butts)
	}
	return response
}
