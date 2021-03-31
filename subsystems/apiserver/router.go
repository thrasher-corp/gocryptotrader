package apiserver

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"

	"github.com/gorilla/mux"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
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

// StartRESTServer starts a REST handler
func StartRESTServer(remoteConfig *config.RemoteControlConfig, pprofConfig *config.Profiler) {
	s := handler{
		remoteConfig:  remoteConfig,
		pprofConfig:   pprofConfig,
		listenAddress: remoteConfig.DeprecatedRPC.ListenAddress,
	}
	log.Debugf(log.RESTSys,
		"Deprecated RPC handler support enabled. Listen URL: http://%s:%d\n",
		common.ExtractHost(s.listenAddress), common.ExtractPort(s.listenAddress))
	err := http.ListenAndServe(s.listenAddress, s.newRouter(true))
	if err != nil {
		log.Errorf(log.RESTSys, "Failed to start deprecated RPC handler. Err: %s", err)
	}
}

type handler struct {
	remoteConfig    *config.RemoteControlConfig
	pprofConfig     *config.Profiler
	exchangeManager *exchangemanager.Manager
	listenAddress   string
}

// StartWebsocketServer starts a Websocket handler
func StartWebsocketServer(remoteConfig *config.RemoteControlConfig, pprofConfig *config.Profiler, exchangeManager *exchangemanager.Manager) {
	s := handler{
		remoteConfig:    remoteConfig,
		pprofConfig:     pprofConfig,
		listenAddress:   remoteConfig.WebsocketRPC.ListenAddress,
		exchangeManager: exchangeManager,
	}
	log.Debugf(log.RESTSys,
		"Websocket RPC support enabled. Listen URL: ws://%s:%d/ws\n",
		common.ExtractHost(s.listenAddress), common.ExtractPort(s.listenAddress))
	err := http.ListenAndServe(s.listenAddress, s.newRouter(false))
	if err != nil {
		log.Errorf(log.RESTSys, "Failed to start websocket RPC handler. Err: %s", err)
	}
}

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
			{"GetAllSettings", http.MethodGet, "/config/all", h.RESTGetAllSettings},
			{"SaveAllSettings", http.MethodPost, "/config/all/save", h.RESTSaveAllSettings},
			{"AllEnabledAccountInfo", http.MethodGet, "/exchanges/enabled/accounts/all", h.RESTGetAllEnabledAccountInfo},
			{"AllActiveExchangesAndCurrencies", http.MethodGet, "/exchanges/enabled/latest/all", h.RESTGetAllActiveTickers},
			{"GetPortfolio", http.MethodGet, "/portfolio/all", h.RESTGetPortfolio},
			{"AllActiveExchangesAndOrderbooks", http.MethodGet, "/exchanges/orderbook/latest/all", h.RESTGetAllActiveOrderbooks},
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
			Handler(RESTLogger(route.HandlerFunc, route.Name)).
			Host(h.listenAddress)
	}
	return router
}

func (h *handler) getIndex(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprint(w, "<html>GoCryptoTrader RESTful interface. For the web GUI, please visit the <a href=https://github.com/thrasher-corp/gocryptotrader/blob/master/web/README.md>web GUI readme.</a></html>")
	if err != nil {
		log.Error(log.CommunicationMgr, err)
	}
	w.WriteHeader(http.StatusOK)
}
