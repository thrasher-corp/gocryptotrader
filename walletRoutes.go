package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/thrasher-/gocryptotrader/exchanges"
)

type AllEnabledExchangeAccounts struct {
	Data []exchange.ExchangeAccountInfo `json:"data"`
}

func GetAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
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
		GetAllEnabledAccountInfo,
	},
}
