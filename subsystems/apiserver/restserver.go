package apiserver

import (
	"encoding/json"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

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

// RESTGetAllSettings replies to a request with an encoded JSON response about the
// trading Bots configuration.
func (h *handler) RESTGetAllSettings(w http.ResponseWriter, r *http.Request) {
	err := writeResponse(w, config.GetConfig())
	if err != nil {
		handleError(r.Method, err)
	}
}

// RESTSaveAllSettings saves all current settings from request body as a JSON
// document then reloads state and returns the settings
func (h *handler) RESTSaveAllSettings(w http.ResponseWriter, r *http.Request) {
	// Get the data from the request
	decoder := json.NewDecoder(r.Body)
	var responseData config.Post
	err := decoder.Decode(&responseData)
	if err != nil {
		handleError(r.Method, err)
	}
	// Save change the settings
	cfg := config.GetConfig()
	err = cfg.UpdateConfig(h.configPath, &responseData.Data, false)
	if err != nil {
		handleError(r.Method, err)
	}

	err = writeResponse(w, cfg)
	if err != nil {
		handleError(r.Method, err)
	}
	err = h.lBot.SetupExchanges()
	if err != nil {
		handleError(r.Method, err)
	}
}

// RESTGetAllActiveOrderbooks returns all enabled exchange orderbooks
func (h *handler) RESTGetAllActiveOrderbooks(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeOrderbooks
	response.Data = h.getAllActiveOrderbooks()
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// RESTGetPortfolio returns the Bot portfoliomanager
func (h *handler) RESTGetPortfolio(w http.ResponseWriter, r *http.Request) {
	p := portfolio.GetPortfolio()
	result := p.GetPortfolioSummary()
	err := writeResponse(w, result)
	if err != nil {
		handleError(r.Method, err)
	}
}

// RESTGetAllActiveTickers returns all active tickers
func (h *handler) RESTGetAllActiveTickers(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	response.Data = h.getAllActiveTickers()
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// RESTGetAllEnabledAccountInfo via get request returns JSON response of account
// info
func (h *handler) RESTGetAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := h.getAllActiveAccounts()
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// getAllActiveOrderbooks returns all enabled exchanges orderbooks
func (h *handler) getAllActiveOrderbooks() []EnabledExchangeOrderbooks {
	var orderbookData []EnabledExchangeOrderbooks
	exchanges := h.exchangeManager.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes()
		exchName := exchanges[x].GetName()
		var exchangeOB EnabledExchangeOrderbooks
		exchangeOB.ExchangeName = exchName

		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.RESTSys,
					"Exchange %h could not retrieve enabled currencies. Err: %h\n",
					exchName,
					err)
				continue
			}
			for z := range currencies {
				ob, err := exchanges[x].FetchOrderbook(currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.RESTSys,
						"Exchange %h failed to retrieve %h orderbook. Err: %h\n", exchName,
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
func (h *handler) getAllActiveTickers() []EnabledExchangeCurrencies {
	var tickers []EnabledExchangeCurrencies
	exchanges := h.exchangeManager.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes()
		exchName := exchanges[x].GetName()
		var exchangeTickers EnabledExchangeCurrencies
		exchangeTickers.ExchangeName = exchName

		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.RESTSys,
					"Exchange %h could not retrieve enabled currencies. Err: %h\n",
					exchName,
					err)
				continue
			}
			for z := range currencies {
				t, err := exchanges[x].FetchTicker(currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.RESTSys,
						"Exchange %h failed to retrieve %h ticker. Err: %h\n", exchName,
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
func (h *handler) getAllActiveAccounts() []AllEnabledExchangeAccounts {
	var accounts []AllEnabledExchangeAccounts
	exchanges := h.exchangeManager.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes()
		exchName := exchanges[x].GetName()
		var exchangeAccounts AllEnabledExchangeAccounts
		for y := range assets {
			a, err := exchanges[x].FetchAccountInfo(assets[y])
			if err != nil {
				log.Errorf(log.RESTSys,
					"Exchange %h failed to retrieve %h ticker. Err: %h\n",
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
