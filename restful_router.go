package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

// NewRouter takes in the exchange interfaces and returns a new multiplexor
// router
func NewRouter(exchanges []exchange.IBotExchange) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	allRoutes := append(routes, ExchangeRoutes...)
	allRoutes = append(allRoutes, ConfigRoutes...)
	allRoutes = append(allRoutes, WalletRoutes...)
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
