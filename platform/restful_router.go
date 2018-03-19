package platform

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

// RESTLogger logs the requests internally
func (b *Bot) RESTLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
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
func (b *Bot) NewRouter(exchanges []exchange.IBotExchange) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	routes = Routes{
		Route{
			"",
			"GET",
			"/",
			getIndex,
		},
		Route{
			"GetAllSettings",
			"GET",
			"/config/all",
			b.RESTGetAllSettings,
		},
		Route{
			"SaveAllSettings",
			"POST",
			"/config/all/save",
			b.RESTSaveAllSettings,
		},
		Route{
			"AllEnabledAccountInfo",
			"GET",
			"/exchanges/enabled/accounts/all",
			b.RESTGetAllEnabledAccountInfo,
		},
		Route{
			"AllActiveExchangesAndCurrencies",
			"GET",
			"/exchanges/enabled/latest/all",
			b.RESTGetAllActiveTickers,
		},
		Route{
			"IndividualExchangeAndCurrency",
			"GET",
			"/exchanges/{exchangeName}/latest/{currency}",
			b.RESTGetTicker,
		},
		Route{
			"GetPortfolio",
			"GET",
			"/portfolio/all",
			b.RESTGetPortfolio,
		},
		Route{
			"AllActiveExchangesAndOrderbooks",
			"GET",
			"/exchanges/orderbook/latest/all",
			b.RESTGetAllActiveOrderbooks,
		},
		Route{
			"IndividualExchangeOrderbook",
			"GET",
			"/exchanges/{exchangeName}/orderbook/latest/{currency}",
			b.RESTGetOrderbook,
		},
		Route{
			"ws",
			"GET",
			"/ws",
			b.WebsocketClientHandler,
		},
	}

	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = b.RESTLogger(handler, route.Name)

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
