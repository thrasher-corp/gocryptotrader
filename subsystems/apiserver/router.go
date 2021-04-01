package apiserver

import (
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// newRouter takes in the exchange interfaces and returns a new multiplexor
// router
func (h *handler) newRouter(isREST bool) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	var routes []Route
	if common.ExtractPort(h.listenAddress) == 80 {
		h.listenAddress = common.ExtractHost(h.listenAddress)
	} else {
		h.listenAddress = strings.Join([]string{common.ExtractHost(h.listenAddress),
			strconv.Itoa(common.ExtractPort(h.listenAddress))}, ":")
	}

	if isREST {
		routes = []Route{
			{"", http.MethodGet, "/", h.getIndex},
			{"GetAllSettings", http.MethodGet, "/config/all", h.restGetAllSettings},
			{"SaveAllSettings", http.MethodPost, "/config/all/save", h.restSaveAllSettings},
			{"AllEnabledAccountInfo", http.MethodGet, "/exchanges/enabled/accounts/all", h.restGetAllEnabledAccountInfo},
			{"AllActiveExchangesAndCurrencies", http.MethodGet, "/exchanges/enabled/latest/all", h.restGetAllActiveTickers},
			{"GetPortfolio", http.MethodGet, "/portfolio/all", h.restGetPortfolio},
			{"AllActiveExchangesAndOrderbooks", http.MethodGet, "/exchanges/orderbook/latest/all", h.restGetAllActiveOrderbooks},
		}

		if h.pprofConfig.Enabled {
			if h.pprofConfig.MutexProfileFraction > 0 {
				runtime.SetMutexProfileFraction(h.pprofConfig.MutexProfileFraction)
			}
			log.Debugf(log.RESTSys,
				"HTTP Go performance profiler (pprof) endpoint enabled: http://%h:%d/debug/pprof/\n",
				common.ExtractHost(h.listenAddress),
				common.ExtractPort(h.listenAddress))
			router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		}
	} else {
		routes = []Route{
			{"ws", http.MethodGet, "/ws", h.WebsocketClientHandler},
		}
	}

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(restLogger(route.HandlerFunc, route.Name)).
			Host(h.listenAddress)
	}
	return router
}
