package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/thrasher-/gocryptotrader/exchanges"
)

type AllEnabledExchangeAccounts struct {
	Data []exchange.ExchangeAccountInfo `json:"data"`
}

func GetCollatedExchangeAccountInfoByCoin(accounts []exchange.ExchangeAccountInfo) map[string]exchange.ExchangeAccountCurrencyInfo {
	result := make(map[string]exchange.ExchangeAccountCurrencyInfo)
	for i := 0; i < len(accounts); i++ {
		for j := 0; j < len(accounts[i].Currencies); j++ {
			currencyName := accounts[i].Currencies[j].CurrencyName
			avail := accounts[i].Currencies[j].TotalValue
			onHold := accounts[i].Currencies[j].Hold

			info, ok := result[currencyName]
			if !ok {
				accountInfo := exchange.ExchangeAccountCurrencyInfo{CurrencyName: currencyName, Hold: onHold, TotalValue: avail}
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

func GetAccountCurrencyInfoByExchangeName(accounts []exchange.ExchangeAccountInfo, exchangeName string) (exchange.ExchangeAccountInfo, error) {
	for i := 0; i < len(accounts); i++ {
		if accounts[i].ExchangeName == exchangeName {
			return accounts[i], nil
		}
	}
	return exchange.ExchangeAccountInfo{}, errors.New(exchange.ErrExchangeNotFound)
}

func GetAllEnabledExchangeAccountInfo() AllEnabledExchangeAccounts {
	var response AllEnabledExchangeAccounts
	for _, individualBot := range bot.exchanges {
		if individualBot != nil && individualBot.IsEnabled() {
			individualExchange, err := individualBot.GetExchangeAccountInfo()
			if err != nil {
				log.Println("Error encountered retrieving exchange account for '" + individualExchange.ExchangeName + "'")
			}
			response.Data = append(response.Data, individualExchange)
		}
	}
	return response
}

func SendAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := GetAllEnabledExchangeAccountInfo()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

var WalletRoutes = Routes{
	Route{
		"AllEnabledAccountInfo",
		"GET",
		"/exchanges/enabled/accounts/all",
		SendAllEnabledAccountInfo,
	},
}
