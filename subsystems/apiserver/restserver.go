package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// StartRESTServer starts a REST handler
func (m *Manager) StartRESTServer() error {
	if !atomic.CompareAndSwapInt32(&m.restStarted, 0, 1) {
		return fmt.Errorf("rest server %w", errAlreadyRuning)
	}
	if !m.remoteConfig.DeprecatedRPC.Enabled {
		atomic.StoreInt32(&m.restStarted, 0)
		return fmt.Errorf("rest %w", errServerDisabled)
	}
	atomic.StoreInt32(&m.started, 1)
	log.Debugf(log.RESTSys,
		"Deprecated RPC handler support enabled. Listen URL: http://%s:%d\n",
		common.ExtractHost(m.restListenAddress), common.ExtractPort(m.restListenAddress))
	m.restRouter = m.newRouter(true)
	m.restHttpServer = &http.Server{
		Addr:    m.restListenAddress,
		Handler: m.restRouter,
	}
	err := m.restHttpServer.ListenAndServe()
	if err != nil {
		atomic.StoreInt32(&m.restStarted, 0)
		atomic.StoreInt32(&m.started, 0)
		return err
	}
	return nil
}

// restLogger logs the requests internally
func restLogger(inner http.Handler, name string) http.Handler {
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

// writeResponse outputs a JSON response of the response interface
func writeResponse(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
}

// handleError prints the REST method and error
func handleError(method string, err error) {
	log.Errorf(log.RESTSys, "RESTful %s: handler failed to send JSON response. Error %s\n",
		method, err)
}

// restGetAllSettings replies to a request with an encoded JSON response about the
// trading Bots configuration.
func (m *Manager) restGetAllSettings(w http.ResponseWriter, r *http.Request) {
	err := writeResponse(w, config.GetConfig())
	if err != nil {
		handleError(r.Method, err)
	}
}

// restSaveAllSettings saves all current settings from request body as a JSON
// document then reloads state and returns the settings
func (m *Manager) restSaveAllSettings(w http.ResponseWriter, r *http.Request) {
	// Get the data from the request
	decoder := json.NewDecoder(r.Body)
	var responseData config.Post
	err := decoder.Decode(&responseData)
	if err != nil {
		handleError(r.Method, err)
	}
	// Save change the settings
	cfg := config.GetConfig()
	err = cfg.UpdateConfig(m.gctConfigPath, &responseData.Data, false)
	if err != nil {
		handleError(r.Method, err)
	}

	err = writeResponse(w, cfg)
	if err != nil {
		handleError(r.Method, err)
	}
	err = m.bot.SetupExchanges()
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetAllActiveOrderbooks returns all enabled exchange orderbooks
func (m *Manager) restGetAllActiveOrderbooks(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeOrderbooks
	response.Data = getAllActiveOrderbooks(m.exchangeManager)
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetPortfolio returns the Bot portfolio manager
func (m *Manager) restGetPortfolio(w http.ResponseWriter, r *http.Request) {
	p := portfolio.GetPortfolio()
	result := p.GetPortfolioSummary()
	err := writeResponse(w, result)
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetAllActiveTickers returns all active tickers
func (m *Manager) restGetAllActiveTickers(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	response.Data = getAllActiveTickers(m.exchangeManager)
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetAllEnabledAccountInfo via get request returns JSON response of account
// info
func (m *Manager) restGetAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := getAllActiveAccounts(m.exchangeManager)
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

func (m *Manager) getIndex(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprint(w, restIndexResponse)
	if err != nil {
		log.Error(log.CommunicationMgr, err)
	}
	w.WriteHeader(http.StatusOK)
}

// getAllActiveOrderbooks returns all enabled exchanges orderbooks
func getAllActiveOrderbooks(m iExchangeManager) []EnabledExchangeOrderbooks {
	var orderbookData []EnabledExchangeOrderbooks
	exchanges := m.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes()
		exchName := exchanges[x].GetName()
		var exchangeOB EnabledExchangeOrderbooks
		exchangeOB.ExchangeName = exchName

		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.RESTSys,
					"Exchange %s could not retrieve enabled currencies. Err: %h\n",
					exchName,
					err)
				continue
			}
			for z := range currencies {
				ob, err := exchanges[x].FetchOrderbook(currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.RESTSys,
						"Exchange %s failed to retrieve %s orderbook. Err: %s\n", exchName,
						currencies[z].String(),
						err)
					continue
				}
				exchangeOB.ExchangeValues = append(exchangeOB.ExchangeValues, *ob)
			}
			orderbookData = append(orderbookData, exchangeOB)
		}
		orderbookData = append(orderbookData, exchangeOB)
	}
	return orderbookData
}

// getAllActiveTickers returns all enabled exchanges tickers
func getAllActiveTickers(m iExchangeManager) []EnabledExchangeCurrencies {
	var tickers []EnabledExchangeCurrencies
	exchanges := m.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes()
		exchName := exchanges[x].GetName()
		var exchangeTickers EnabledExchangeCurrencies
		exchangeTickers.ExchangeName = exchName

		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.RESTSys,
					"Exchange %s could not retrieve enabled currencies. Err: %s\n",
					exchName,
					err)
				continue
			}
			for z := range currencies {
				t, err := exchanges[x].FetchTicker(currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.RESTSys,
						"Exchange %s failed to retrieve %s ticker. Err: %s\n", exchName,
						currencies[z].String(),
						err)
					continue
				}
				exchangeTickers.ExchangeValues = append(exchangeTickers.ExchangeValues, *t)
			}
			tickers = append(tickers, exchangeTickers)
		}
		tickers = append(tickers, exchangeTickers)
	}
	return tickers
}

// getAllActiveAccounts returns all enabled exchanges accounts
func getAllActiveAccounts(m iExchangeManager) []AllEnabledExchangeAccounts {
	var accounts []AllEnabledExchangeAccounts
	exchanges := m.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes()
		exchName := exchanges[x].GetName()
		var exchangeAccounts AllEnabledExchangeAccounts
		for y := range assets {
			a, err := exchanges[x].FetchAccountInfo(assets[y])
			if err != nil {
				log.Errorf(log.RESTSys,
					"Exchange %s failed to retrieve %s ticker. Err: %s\n",
					exchName,
					assets[y],
					err)
				continue
			}
			exchangeAccounts.Data = append(exchangeAccounts.Data, a)
		}
		accounts = append(accounts, exchangeAccounts)
	}
	return accounts
}
