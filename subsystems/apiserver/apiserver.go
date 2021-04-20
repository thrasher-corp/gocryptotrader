package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gorilla/mux"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

func Setup(remoteConfig *config.RemoteControlConfig, pprofConfig *config.Profiler, exchangeManager iExchangeManager, bot iBot, portfolioManager iPortfolioManager, configPath string) (*Manager, error) {
	if remoteConfig == nil {
		return nil, errNilRemoteConfig
	}
	if pprofConfig == nil {
		return nil, errNilPProfConfig
	}
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if bot == nil {
		return nil, errNilBot
	}
	if configPath == "" {
		return nil, errEmptyConfigPath
	}
	return &Manager{
		remoteConfig:           remoteConfig,
		pprofConfig:            pprofConfig,
		restListenAddress:      remoteConfig.DeprecatedRPC.ListenAddress,
		websocketListenAddress: remoteConfig.WebsocketRPC.ListenAddress,
		exchangeManager:        exchangeManager,
		bot:                    bot,
		gctConfigPath:          configPath,
		portfolioManager:       portfolioManager,
	}, nil
}

func (m *Manager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

func (m *Manager) Stop() error {
	if m == nil {
		return fmt.Errorf("api server %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("api server %w", subsystems.ErrSubSystemNotStarted)
	}
	m.restRouter = nil
	m.websocketRouter = nil
	m.websocketHub = nil
	if m.restHttpServer != nil {
		err := m.restHttpServer.Shutdown(context.Background())
		if err != nil {
			return err
		}
		m.restHttpServer = nil
	}
	if m.websocketHttpServer != nil {
		err := m.websocketHttpServer.Shutdown(context.Background())
		if err != nil {
			return err
		}
		m.websocketHttpServer = nil
	}
	atomic.StoreInt32(&m.websocketStarted, 0)
	atomic.StoreInt32(&m.restStarted, 0)
	return nil
}

// newRouter takes in the exchange interfaces and returns a new multiplexor
// router
func (m *Manager) newRouter(isREST bool) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	var routes []Route
	if common.ExtractPort(m.websocketListenAddress) == 80 {
		m.websocketListenAddress = common.ExtractHost(m.websocketListenAddress)
	} else {
		m.websocketListenAddress = strings.Join([]string{common.ExtractHost(m.websocketListenAddress),
			strconv.Itoa(common.ExtractPort(m.websocketListenAddress))}, ":")
	}

	if isREST {
		routes = []Route{
			{"", http.MethodGet, "/", m.getIndex},
			{"GetAllSettings", http.MethodGet, "/config/all", m.restGetAllSettings},
			{"SaveAllSettings", http.MethodPost, "/config/all/save", m.restSaveAllSettings},
			{"AllEnabledAccountInfo", http.MethodGet, "/exchanges/enabled/accounts/all", m.restGetAllEnabledAccountInfo},
			{"AllActiveExchangesAndCurrencies", http.MethodGet, "/exchanges/enabled/latest/all", m.restGetAllActiveTickers},
			{"GetPortfolio", http.MethodGet, "/portfolio/all", m.restGetPortfolio},
			{"AllActiveExchangesAndOrderbooks", http.MethodGet, "/exchanges/orderbook/latest/all", m.restGetAllActiveOrderbooks},
		}

		if m.pprofConfig.Enabled {
			if m.pprofConfig.MutexProfileFraction > 0 {
				runtime.SetMutexProfileFraction(m.pprofConfig.MutexProfileFraction)
			}
			log.Debugf(log.RESTSys,
				"HTTP Go performance profiler (pprof) endpoint enabled: http://%s:%d/debug/pprof/\n",
				common.ExtractHost(m.websocketListenAddress),
				common.ExtractPort(m.websocketListenAddress))
			router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		}
	} else {
		routes = []Route{
			{"ws", http.MethodGet, "/ws", m.WebsocketClientHandler},
		}
	}

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(restLogger(route.HandlerFunc, route.Name)).
			Host(m.websocketListenAddress)
	}
	return router
}
