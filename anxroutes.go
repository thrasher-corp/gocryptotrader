package main

import (
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
)

func getLatestAnxTicker(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    currency := vars["currency"]
    response := bot.exchange.anx.GetTicker(currency)
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(http.StatusOK)
    if err := json.NewEncoder(w).Encode(response); err != nil {
        panic(err)
    }
}

var anxRoutes = Routes{
    Route{
        "Index",
        "GET",
        "/exchanges/anx/latest/{currency}",
        getLatestAnxTicker,
    },
}