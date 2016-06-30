package main

import (
    "net/http"
    "github.com/gorilla/mux"
)

func NewRouter(exchanges []IBotExchange) *mux.Router {
    router := mux.NewRouter().StrictSlash(true)
    allRoutes := append(routes,exchangeRoutes...)
    for _, route := range allRoutes {
        var handler http.Handler
        handler = route.HandlerFunc
        handler = Logger(handler, route.Name)

        router.
            Methods(route.Method).
            Path(route.Pattern).
            Name(route.Name).
            Handler(handler)

    }
    return router
}