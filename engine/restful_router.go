package engine

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// RESTLogger logs the requests internally
func RESTLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		inner.ServeHTTP(w, r)

		log.Debugf(log.RESTSys,
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

// StartRESTServer starts a REST server
func StartRESTServer() {
	listenAddr := Bot.Config.RemoteControl.DeprecatedRPC.ListenAddress
	log.Debugf(log.RESTSys,
		"Deprecated RPC server support enabled. Listen URL: http://%s:%d\n",
		common.ExtractHost(listenAddr), common.ExtractPort(listenAddr))
	err := http.ListenAndServe(listenAddr, newRouter(true))
	if err != nil {
		log.Errorf(log.RESTSys, "Failed to start deprecated RPC server. Err: %s", err)
	}
}

// StartWebsocketServer starts a Websocket server
func StartWebsocketServer() {
	listenAddr := Bot.Config.RemoteControl.WebsocketRPC.ListenAddress
	log.Debugf(log.RESTSys,
		"Websocket RPC support enabled. Listen URL: ws://%s:%d/ws\n",
		common.ExtractHost(listenAddr), common.ExtractPort(listenAddr))
	err := http.ListenAndServe(listenAddr, newRouter(false))
	if err != nil {
		log.Errorf(log.RESTSys, "Failed to start websocket RPC server. Err: %s", err)
	}
}

// newRouter takes in the exchange interfaces and returns a new multiplexor
// router
func newRouter(isREST bool) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	var routes []Route
	var listenAddr string

	if isREST {
		listenAddr = Bot.Config.RemoteControl.DeprecatedRPC.ListenAddress
	} else {
		listenAddr = Bot.Config.RemoteControl.WebsocketRPC.ListenAddress
	}

	if common.ExtractPort(listenAddr) == 80 {
		listenAddr = common.ExtractHost(listenAddr)
	} else {
		listenAddr = strings.Join([]string{common.ExtractHost(listenAddr),
			strconv.Itoa(common.ExtractPort(listenAddr))}, ":")
	}

	if isREST {
		routes = []Route{
			{"", http.MethodGet, "/", getIndex},
			{"GetAllSettings", http.MethodGet, "/config/all", RESTGetAllSettings},
			{"SaveAllSettings", http.MethodPost, "/config/all/save", RESTSaveAllSettings},
			{"AllEnabledAccountInfo", http.MethodGet, "/exchanges/enabled/accounts/all", RESTGetAllEnabledAccountInfo},
			{"AllActiveExchangesAndCurrencies", http.MethodGet, "/exchanges/enabled/latest/all", RESTGetAllActiveTickers},
			{"GetPortfolio", http.MethodGet, "/portfolio/all", RESTGetPortfolio},
			{"AllActiveExchangesAndOrderbooks", http.MethodGet, "/exchanges/orderbook/latest/all", RESTGetAllActiveOrderbooks},
		}

		if Bot.Config.Profiler.Enabled {
			if Bot.Config.Profiler.MutexProfileFraction > 0 {
				runtime.SetMutexProfileFraction(Bot.Config.Profiler.MutexProfileFraction)
			}
			log.Debugf(log.RESTSys,
				"HTTP Go performance profiler (pprof) endpoint enabled: http://%s:%d/debug/pprof/\n",
				common.ExtractHost(listenAddr),
				common.ExtractPort(listenAddr))
			router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		}
	} else {
		routes = []Route{
			{"ws", http.MethodGet, "/ws", WebsocketClientHandler},
		}
	}

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(RESTLogger(route.HandlerFunc, route.Name)).
			Host(listenAddr)
	}
	return router
}

func getIndex(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "<html>GoCryptoTrader RESTful interface. For the web GUI, please visit the <a href=https://github.com/thrasher-corp/gocryptotrader/blob/master/web/README.md>web GUI readme.</a></html>")
	w.WriteHeader(http.StatusOK)
}
