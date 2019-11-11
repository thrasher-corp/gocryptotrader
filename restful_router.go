package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/thrasher-corp/gocryptotrader/common"
	log "github.com/thrasher-corp/gocryptotrader/logger"

	_ "net/http/pprof" // nolint: gosec
)

// RESTLogger logs the requests internally
func RESTLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Debugf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

// Route is a sub type that holds the request routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes is an array of all the registered routes
type Routes []Route

var routes = Routes{}

// NewRouter takes in the exchange interfaces and returns a new multiplexor
// router
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	var listenAddr string

	if common.ExtractPort(bot.config.Webserver.ListenAddress) == 80 {
		listenAddr = common.ExtractHost(bot.config.Webserver.ListenAddress)
	} else {
		listenAddr = common.JoinStrings([]string{common.ExtractHost(bot.config.Webserver.ListenAddress),
			strconv.Itoa(common.ExtractPort(bot.config.Webserver.ListenAddress))}, ":")
	}

	routes = Routes{
		Route{
			"",
			http.MethodGet,
			"/",
			getIndex,
		},
		Route{
			"GetAllSettings",
			http.MethodGet,
			"/config/all",
			RESTGetAllSettings,
		},
		Route{
			"SaveAllSettings",
			http.MethodPost,
			"/config/all/save",
			RESTSaveAllSettings,
		},
		Route{
			"AllEnabledAccountInfo",
			http.MethodGet,
			"/exchanges/enabled/accounts/all",
			RESTGetAllEnabledAccountInfo,
		},
		Route{
			"AllActiveExchangesAndCurrencies",
			http.MethodGet,
			"/exchanges/enabled/latest/all",
			RESTGetAllActiveTickers,
		},
		Route{
			"IndividualExchangeAndCurrency",
			http.MethodGet,
			"/exchanges/{exchangeName}/latest/{currency}",
			RESTGetTicker,
		},
		Route{
			"GetPortfolio",
			http.MethodGet,
			"/portfolio/all",
			RESTGetPortfolio,
		},
		Route{
			"AllActiveExchangesAndOrderbooks",
			http.MethodGet,
			"/exchanges/orderbook/latest/all",
			RESTGetAllActiveOrderbooks,
		},
		Route{
			"IndividualExchangeOrderbook",
			http.MethodGet,
			"/exchanges/{exchangeName}/orderbook/latest/{currency}",
			RESTGetOrderbook,
		},
		Route{
			"ws",
			http.MethodGet,
			"/ws",
			WebsocketClientHandler,
		},
	}

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(RESTLogger(route.HandlerFunc, route.Name)).
			Host(listenAddr)
	}

	if bot.config.Profiler.Enabled {
		log.Debugln("Profiler enabled")
		router.PathPrefix("/debug").Handler(http.DefaultServeMux)
	}

	return router
}

func getIndex(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "<html>GoCryptoTrader RESTful interface. For the web GUI, please visit the <a href=https://github.com/thrasher-corp/gocryptotrader/blob/master/web/README.md>web GUI readme.</a></html>")
}
