package main

import (
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

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
	superFinalResponse := []ExchangeResponses{}
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

			// Set the APIKEYS MANNN
			if _, ok := keys[name]; ok {
				base.Config.API.Credentials.Key = keys[name].APIKey
				base.Config.API.Credentials.Secret = keys[name].APISecret
				base.Config.API.Credentials.ClientID = keys[name].ClientID
			}
			authenticated := base.ValidateAPICredentials()
			superFinalResponse = append(superFinalResponse, ExchangeResponses{
				ExchangeName:       name,
				AssetPairResponses: testWrappers(engine.Bot.Exchanges[num], base, authenticated, false),
			})
			wg.Done()
		}(x)
	}
	wg.Wait()
	log.Println("Done.")
	log.Println()

	outputToConsole(superFinalResponse, verbose)
	outputToJSON(superFinalResponse)
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
	butts := make(map[string]Key)
	return butts, common.JSONDecode(keys, &butts)
}

func outputToJSON(superFinalResponse []ExchangeResponses) {
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

func outputToConsole(superFinalResponse []ExchangeResponses, verbose bool) {
	var totalErrors int64
	for i := range superFinalResponse {
		log.Printf("------------%v Results-------------\n", superFinalResponse[i].ExchangeName)
		for j := range superFinalResponse[i].AssetPairResponses {
			for k := range superFinalResponse[i].AssetPairResponses[j].EndpointResponses {
				log.Printf("%v Result: %v", superFinalResponse[j].ExchangeName, k)
				log.Printf("Function:\t%v", superFinalResponse[i].AssetPairResponses[j].EndpointResponses[k].Function)
				log.Printf("AssetType:\t%v", superFinalResponse[i].AssetPairResponses[j].AssetType)
				log.Printf("Currency:\t%v\n", superFinalResponse[i].AssetPairResponses[j].CurrencyPair)
				if superFinalResponse[i].AssetPairResponses[j].EndpointResponses[k].Error != "" {
					totalErrors++
					log.Printf("Error:\t%v", superFinalResponse[i].AssetPairResponses[j].EndpointResponses[k].Error)
				} else {
					log.Print("Error:\tnone")
				}
				if verbose {
					butts, err := common.JSONEncode(superFinalResponse[i].AssetPairResponses[j].EndpointResponses[k].Response)
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
}

func testWrappers(e exchange.IBotExchange, base *exchange.Base, authenticated, verbose bool) []ExchangeAssetPairResponses {
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
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "FetchTicker",
			Error:    msg,
			Response: r1,
		})

		r2, err := e.UpdateTicker(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "UpdateTicker",
			Error:    msg,
			Response: r2,
		})

		r3, err := e.FetchOrderbook(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "FetchOrderbook",
			Error:    msg,
			Response: r3,
		})

		r4, err := e.UpdateOrderbook(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "UpdateOrderbook",
			Error:    msg,
			Response: r4,
		})

		r5, err := e.FetchTradablePairs(assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
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
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "UpdateTradablePairs",
			Error:    msg,
		})

		if !authenticated {
			response = append(response, responseContainer)
			continue
		}

		r7, err := e.GetAccountInfo()
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "GetAccountInfo",
			Error:    msg,
			Response: r7,
		})

		r8, err := e.GetExchangeHistory(p, assetTypes[i])
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "GetExchangeHistory",
			Error:    msg,
			Response: r8,
		})

		r9, err := e.GetFundingHistory()
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
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
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "SubmitOrder",
			Error:    msg,
			Response: r10,
		})

		r16, err := e.GetActiveOrders(&exchange.GetOrdersRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
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
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
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
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "CancelOrder",
			Error:    msg,
		})

		r13, err := e.CancelAllOrders(&exchange.OrderCancellation{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "CancelAllOrders",
			Error:    msg,
			Response: r13,
		})

		r14, err := e.GetOrderInfo(orderID)
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "GetOrderInfo",
			Error:    msg,
			Response: r14,
		})

		r15, err := e.GetOrderHistory(&exchange.GetOrdersRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "GetOrderHistory",
			Error:    msg,
			Response: r15,
		})

		r17, err := e.GetDepositAddress(p.Base, "")
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "GetDepositAddress",
			Error:    msg,
			Response: r17,
		})

		r18, err := e.WithdrawCryptocurrencyFunds(&exchange.CryptoWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "WithdrawCryptocurrencyFunds",
			Error:    msg,
			Response: r18,
		})

		r19, err := e.WithdrawFiatFunds(&exchange.FiatWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "WithdrawFiatFunds",
			Error:    msg,
			Response: r19,
		})
		r20, err := e.WithdrawFiatFundsToInternationalBank(&exchange.FiatWithdrawRequest{})
		msg = ""
		if err != nil {
			msg = err.Error()
		}
		responseContainer.EndpointResponses = append(responseContainer.EndpointResponses, EndpointReponse{
			Function: "WithdrawFiatFundsToInternationalBank",
			Error:    msg,
			Response: r20,
		})
		response = append(response, responseContainer)
	}
	return response
}

func setupKeys() map[string]Key {
	return make(map[string]Key)
}

const (
	totalWrappers = 20
)

type Key struct {
	APIKey, APISecret, ClientID string
}
type ExchangeResponses struct {
	ExchangeName       string                       `json:"exchangeName"`
	AssetPairResponses []ExchangeAssetPairResponses `json:"responses"`
}

type ExchangeAssetPairResponses struct {
	AssetType         asset.Item        `json:"asset"`
	CurrencyPair      currency.Pair     `json:"currency"`
	EndpointResponses []EndpointReponse `json:"responses"`
}

type EndpointReponse struct {
	Function string      `json:"function"`
	Error    string      `json:"error"`
	Response interface{} `json:"response"`
}
