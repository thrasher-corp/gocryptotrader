package engine

import (
	"encoding/json"
	"net/http"

	"github.com/thrasher-/gocryptotrader/config"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// RESTfulJSONResponse outputs a JSON response of the response interface
func RESTfulJSONResponse(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
}

// RESTfulError prints the REST method and error
func RESTfulError(method string, err error) {
	log.Errorf("RESTful %s: server failed to send JSON response. Error %s",
		method, err)
}

// RESTGetAllSettings replies to a request with an encoded JSON response about the
// trading Bots configuration.
func RESTGetAllSettings(w http.ResponseWriter, r *http.Request) {
	err := RESTfulJSONResponse(w, Bot.Config)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTSaveAllSettings saves all current settings from request body as a JSON
// document then reloads state and returns the settings
func RESTSaveAllSettings(w http.ResponseWriter, r *http.Request) {
	// Get the data from the request
	decoder := json.NewDecoder(r.Body)
	var responseData config.Post
	err := decoder.Decode(&responseData)
	if err != nil {
		RESTfulError(r.Method, err)
	}
	// Save change the settings
	err = Bot.Config.UpdateConfig(Bot.Settings.ConfigFile, &responseData.Data)
	if err != nil {
		RESTfulError(r.Method, err)
	}

	err = RESTfulJSONResponse(w, Bot.Config)
	if err != nil {
		RESTfulError(r.Method, err)
	}

	SetupExchanges()
}

// GetAllActiveOrderbooks returns all enabled exchanges orderbooks
func GetAllActiveOrderbooks() []EnabledExchangeOrderbooks {
	var orderbookData []EnabledExchangeOrderbooks

	for _, exch := range Bot.Exchanges {
		if !exch.IsEnabled() {
			continue
		}

		assets := exch.GetAssetTypes()
		exchName := exch.GetName()
		var exchangeOB EnabledExchangeOrderbooks
		exchangeOB.ExchangeName = exchName

		for y := range assets {
			currencies := exch.GetEnabledPairs(assets[y])
			for z := range currencies {
				ob, err := exch.FetchOrderbook(currencies[z], assets[y])
				if err != nil {
					log.Errorf("Exchange %s failed to retrieve %s orderbook. Err: %s", exchName,
						currencies[z].String(),
						err)
					continue
				}
				exchangeOB.ExchangeValues = append(exchangeOB.ExchangeValues, ob)
			}
			orderbookData = append(orderbookData, exchangeOB)
		}
		orderbookData = append(orderbookData, exchangeOB)
	}
	return orderbookData
}

// RESTGetAllActiveOrderbooks returns all enabled exchange orderbooks
func RESTGetAllActiveOrderbooks(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeOrderbooks
	response.Data = GetAllActiveOrderbooks()

	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTGetPortfolio returns the Bot portfolio
func RESTGetPortfolio(w http.ResponseWriter, r *http.Request) {
	result := Bot.Portfolio.GetPortfolioSummary()
	err := RESTfulJSONResponse(w, result)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTGetAllActiveTickers returns all active tickers
func RESTGetAllActiveTickers(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	response.Data = GetAllActiveTickers()

	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTGetAllEnabledAccountInfo via get request returns JSON response of account
// info
func RESTGetAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := GetAllEnabledExchangeAccountInfo()
	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}
