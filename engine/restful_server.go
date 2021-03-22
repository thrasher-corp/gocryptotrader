package engine

import (
	"encoding/json"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// RESTfulJSONResponse outputs a JSON response of the response interface
func RESTfulJSONResponse(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
}

// RESTfulError prints the REST method and error
func RESTfulError(method string, err error) {
	log.Errorf(log.RESTSys, "RESTful %s: server failed to send JSON response. Error %s\n",
		method, err)
}

// RESTGetAllSettings replies to a request with an encoded JSON response about the
// trading Bots configuration.
func RESTGetAllSettings(w http.ResponseWriter, r *http.Request) {
	err := RESTfulJSONResponse(w, config.Cfg)
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
	err = Bot.Config.UpdateConfig(Bot.Settings.ConfigFile, &responseData.Data, false)
	if err != nil {
		RESTfulError(r.Method, err)
	}

	err = RESTfulJSONResponse(w, Bot.Config)
	if err != nil {
		RESTfulError(r.Method, err)
	}

	Bot.SetupExchanges()
}

// GetAllActiveOrderbooks returns all enabled exchanges orderbooks
func GetAllActiveOrderbooks() []EnabledExchangeOrderbooks {
	var orderbookData []EnabledExchangeOrderbooks
	exchanges := Bot.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes()
		exchName := exchanges[x].GetName()
		var exchangeOB EnabledExchangeOrderbooks
		exchangeOB.ExchangeName = exchName

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
	p := portfolio.GetPortfolio()
	result := p.GetPortfolioSummary()
	err := RESTfulJSONResponse(w, result)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTGetAllActiveTickers returns all active tickers
func RESTGetAllActiveTickers(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	response.Data = Bot.GetAllActiveTickers()

	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}

// RESTGetAllEnabledAccountInfo via get request returns JSON response of account
// info
func RESTGetAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := Bot.GetAllEnabledExchangeAccountInfo()
	err := RESTfulJSONResponse(w, response)
	if err != nil {
		RESTfulError(r.Method, err)
	}
}
