package main

import (
	"fmt"
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
	allRoutes = append(allRoutes, PortfolioRoutes...)
	allRoutes = append(allRoutes, WalletRoutes...)
	allRoutes = append(allRoutes, IndexRoute...)
	allRoutes = append(allRoutes, WebsocketRoutes...)
	allRoutes = append(allRoutes, OrderbookRoutes...)
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

func getIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<html>GoCryptoTrader RESTful interface. For the web GUI, please visit the <a href=https://github.com/thrasher-/gocryptotrader/blob/master/web/README.md>web GUI readme.</a></html>")
	w.WriteHeader(http.StatusOK)
}

// IndexRoute maps the index route to the getIndex function
var IndexRoute = Routes{
	Route{
		"",
		"GET",
		"/",
		getIndex,
	},
}
