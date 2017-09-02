package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

// GetSpecificOrderbook returns a specific orderbook given the currency,
// exchangeName and assetType
func GetSpecificOrderbook(currency, exchangeName, assetType string) (orderbook.Base, error) {
	var specificOrderbook orderbook.Base
	var err error
	for i := 0; i < len(bot.exchanges); i++ {
		if bot.exchanges[i] != nil {
			if bot.exchanges[i].IsEnabled() && bot.exchanges[i].GetName() == exchangeName {
				specificOrderbook, err = bot.exchanges[i].GetOrderbookEx(
					pair.NewCurrencyPairFromString(currency),
					assetType,
				)
				break
			}
		}
	}
	return specificOrderbook, err
}

func jsonOrderbookResponse(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	exchange := vars["exchangeName"]
	assetType := vars["assetType"]

	if assetType == "" {
		assetType = orderbook.Spot
	}

	response, err := GetSpecificOrderbook(currency, exchange, assetType)
	if err != nil {
		log.Printf("Failed to fetch orderbook for %s currency: %s\n", exchange,
			currency)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// AllEnabledExchangeOrderbooks holds the enabled exchange orderbooks
type AllEnabledExchangeOrderbooks struct {
	Data []EnabledExchangeOrderbooks `json:"data"`
}

// EnabledExchangeOrderbooks is a sub type for singular exchanges and respective
// orderbooks
type EnabledExchangeOrderbooks struct {
	ExchangeName   string           `json:"exchangeName"`
	ExchangeValues []orderbook.Base `json:"exchangeValues"`
}

// GetAllActiveOrderbooks returns all enabled exchanges orderbooks
func GetAllActiveOrderbooks() []EnabledExchangeOrderbooks {
	var orderbookData []EnabledExchangeOrderbooks

	for _, individualBot := range bot.exchanges {
		if individualBot != nil && individualBot.IsEnabled() {
			var individualExchange EnabledExchangeOrderbooks
			exchangeName := individualBot.GetName()
			individualExchange.ExchangeName = exchangeName
			currencies := individualBot.GetEnabledCurrencies()
			assetTypes, err := exchange.GetExchangeAssetTypes(exchangeName)
			if err != nil {
				log.Printf("failed to get %s exchange asset types. Error: %s",
					exchangeName, err)
				continue
			}
			for _, x := range currencies {
				currency := x

				var ob orderbook.Base
				if len(assetTypes) > 1 {
					for y := range assetTypes {
						ob, err = individualBot.UpdateOrderbook(currency,
							assetTypes[y])
					}
				} else {
					ob, err = individualBot.UpdateOrderbook(currency,
						assetTypes[0])
				}

				if err != nil {
					log.Printf("failed to get %s %s orderbook. Error: %s",
						currency.Pair().String(),
						exchangeName,
						err)
					continue
				}

				individualExchange.ExchangeValues = append(
					individualExchange.ExchangeValues, ob,
				)
			}
			orderbookData = append(orderbookData, individualExchange)
		}
	}
	return orderbookData
}

func getAllActiveOrderbooksResponse(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeOrderbooks
	response.Data = GetAllActiveOrderbooks()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// OrderbookRoutes denotes the current exchange orderbook routes
var OrderbookRoutes = Routes{
	Route{
		"AllActiveExchangesAndOrderbooks",
		"GET",
		"/exchanges/orderbook/latest/all",
		getAllActiveOrderbooksResponse,
	},
	Route{
		"IndividualExchangeOrderbook",
		"GET",
		"/exchanges/{exchangeName}/orderbook/latest/{currency}",
		jsonOrderbookResponse,
	},
}
