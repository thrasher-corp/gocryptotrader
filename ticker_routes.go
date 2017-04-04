package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func GetSpecificTicker(currency, exchangeName string) (ticker.TickerPrice, error) {
	var specificTicker ticker.TickerPrice
	var err error
	for i := 0; i < len(bot.exchanges); i++ {
		if bot.exchanges[i] != nil {
			if bot.exchanges[i].IsEnabled() && bot.exchanges[i].GetName() == exchangeName {
				specificTicker, err = bot.exchanges[i].GetTickerPrice(
					pair.NewCurrencyPairFromString(currency),
				)
				break
			}
		}
	}
	return specificTicker, err
}

func jsonTickerResponse(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	exchange := vars["exchangeName"]
	response, err := GetSpecificTicker(currency, exchange)
	if err != nil {
		log.Printf("Failed to fetch ticker for %s currency: %s\n", exchange,
			currency)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// AllEnabledExchangeCurrencies holds the enabled exchange currencies
type AllEnabledExchangeCurrencies struct {
	Data []EnabledExchangeCurrencies `json:"data"`
}

// EnabledExchangeCurrencies is a sub type for singular exchanges and respective
// currencies
type EnabledExchangeCurrencies struct {
	ExchangeName   string               `json:"exchangeName"`
	ExchangeValues []ticker.TickerPrice `json:"exchangeValues"`
}

func GetAllActiveTickers() []EnabledExchangeCurrencies {
	var tickerData []EnabledExchangeCurrencies

	for _, individualBot := range bot.exchanges {
		if individualBot != nil && individualBot.IsEnabled() {
			var individualExchange EnabledExchangeCurrencies
			individualExchange.ExchangeName = individualBot.GetName()
			log.Println(
				"Getting enabled currencies for '" + individualBot.GetName() + "'",
			)
			currencies := individualBot.GetEnabledCurrencies()
			for x := range currencies {
				currency := currencies[x]

				tickerPrice, err := individualBot.GetTickerPrice(currency)
				if err != nil {
					continue
				}

				individualExchange.ExchangeValues = append(
					individualExchange.ExchangeValues, tickerPrice,
				)
			}
			tickerData = append(tickerData, individualExchange)
		}
	}
	return tickerData
}

func getAllActiveTickersResponse(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	response.Data = GetAllActiveTickers()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// ExchangeRoutes denotes the current exchange routes
var ExchangeRoutes = Routes{
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
