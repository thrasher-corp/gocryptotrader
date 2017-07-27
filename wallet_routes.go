package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/thrasher-/gocryptotrader/exchanges"
)

// AllEnabledExchangeAccounts holds all enabled accounts info
type AllEnabledExchangeAccounts struct {
	Data []exchange.AccountInfo `json:"data"`
}

// GetCollatedExchangeAccountInfoByCoin collates individual exchange account
// information and turns into into a map string of
// exchange.AccountCurrencyInfo
func GetCollatedExchangeAccountInfoByCoin(accounts []exchange.AccountInfo) map[string]exchange.AccountCurrencyInfo {
	result := make(map[string]exchange.AccountCurrencyInfo)
	for i := 0; i < len(accounts); i++ {
		for j := 0; j < len(accounts[i].Currencies); j++ {
			currencyName := accounts[i].Currencies[j].CurrencyName
			avail := accounts[i].Currencies[j].TotalValue
			onHold := accounts[i].Currencies[j].Hold

			info, ok := result[currencyName]
			if !ok {
				accountInfo := exchange.AccountCurrencyInfo{CurrencyName: currencyName, Hold: onHold, TotalValue: avail}
				result[currencyName] = accountInfo
			} else {
				info.Hold += onHold
				info.TotalValue += avail
				result[currencyName] = info
			}
		}
	}
	return result
}

// GetAccountCurrencyInfoByExchangeName returns info for an exchange
func GetAccountCurrencyInfoByExchangeName(accounts []exchange.AccountInfo, exchangeName string) (exchange.AccountInfo, error) {
	for i := 0; i < len(accounts); i++ {
		if accounts[i].ExchangeName == exchangeName {
			return accounts[i], nil
		}
	}
	return exchange.AccountInfo{}, errors.New(exchange.ErrExchangeNotFound)
}

// GetAllEnabledExchangeAccountInfo returns all the current enabled exchanges
func GetAllEnabledExchangeAccountInfo() AllEnabledExchangeAccounts {
	var response AllEnabledExchangeAccounts
	for _, individualBot := range bot.exchanges {
		if individualBot != nil && individualBot.IsEnabled() {
			individualExchange, err := individualBot.GetExchangeAccountInfo()
			if err != nil {
				log.Println(
					"Error encountered retrieving exchange account for '" + individualExchange.ExchangeName + "'",
				)
			}
			response.Data = append(response.Data, individualExchange)
		}
	}
	return response
}

// SendAllEnabledAccountInfo via get request returns JSON response of account
// info
func SendAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := GetAllEnabledExchangeAccountInfo()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// WalletRoutes are current routes specified for queries.
var WalletRoutes = Routes{
	Route{
		"AllEnabledAccountInfo",
		"GET",
		"/exchanges/enabled/accounts/all",
		SendAllEnabledAccountInfo,
	},
}
