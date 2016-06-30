package main

import (
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
)

func jsonTickerResponse(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    currency := vars["currency"]
    exchangeName := vars["exchangeName"]
    var response TickerPrice;
    for i := 0; i < len(bot.exchanges); i++ {
        if bot.exchanges[i] != nil {
            if(bot.exchanges[i].IsEnabled() && bot.exchanges[i].GetName() == exchangeName) {
              response =  bot.exchanges[i].GetTickerPrice(currency)
            }
        }
    }

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(response); err != nil {
        panic(err)
    }
}

func getAllActiveTickersResponse(w http.ResponseWriter, r *http.Request) {
    var response[] TickerPrice
    //bot.config.Cryptocurrencies
    for i := 0; i < len(bot.exchanges); i++ {
        if bot.exchanges[i] != nil {
            if(bot.exchanges[i].IsEnabled()) {
                for _, exch := range bot.config.Exchanges {
                    if(bot.exchanges[i].GetName() == exch.Name) {
                        for _, enabledPair := range exch.BaseCurrencies {
                            response = append(response, bot.exchanges[i].GetTickerPrice(enabledPair))
                        }
                    }
                }
            }
        }
    }

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