package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func jsonTickerResponse(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	exchangeName := vars["exchangeName"]
	var response TickerPrice
	for i := 0; i < len(bot.exchanges); i++ {
		if bot.exchanges[i] != nil {
			if bot.exchanges[i].IsEnabled() && bot.exchanges[i].GetName() == exchangeName {
				response = bot.exchanges[i].GetTickerPrice(currency)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)

	if err := encoder.Encode(response); err != nil {
		panic(err)
	}
}

type AllEnabledExchangeCurrencies struct {
	Data []EnabledExchangeCurrencies `json:"data"`
}

type EnabledExchangeCurrencies struct {
	ExchangeName   string        `json:"exchangeName"`
	ExchangeValues []TickerPrice `json:"exchangeValues"`
}

func getAllActiveTickersResponse(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	log.Println(bot.exchanges[15])

	for _, individualBot := range bot.exchanges {
		if individualBot != nil && individualBot.IsEnabled() {
			var individualExchange EnabledExchangeCurrencies
			individualExchange.ExchangeName = individualBot.GetName()
			log.Println("Getting enabled currencies for '" + individualBot.GetName() + "'")
			currencies := individualBot.GetEnabledCurrencies()
			log.Println(currencies)
			for _, currency := range currencies {
				tickerPrice := individualBot.GetTickerPrice(currency)
				log.Println(tickerPrice)

				individualExchange.ExchangeValues = append(individualExchange.ExchangeValues, tickerPrice)
			}
			response.Data = append(response.Data, individualExchange)
		}
	}

	log.Println(response)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

var exchangeRoutes = Routes{
	Route{
		"AllActiveExchangesAndCurrencies",
		"GET",
		"/exchanges/enabled/latest/all",
		getAllActiveTickersResponse,
	},
	Route{
		"IndividualExchangeAndCurrency",
		"GET",
		"/exchanges/{exchangeName}/latest/{currency}",
		jsonTickerResponse,
	},
}
