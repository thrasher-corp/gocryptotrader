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

var exchangeRoutes = Routes{
    Route{
        "Index",
        "GET",
        "/exchanges/{exchangeName}/latest/{currency}",
        jsonTickerResponse,
    },
}