package main

import (
	"encoding/json"
	"net/http"

	"github.com/thrasher-/gocryptotrader/config"
)

func GetAllSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(bot.config); err != nil {
		panic(err)
	}
}

func SaveAllSettings(w http.ResponseWriter, r *http.Request) {
	//Get the data from the request
	decoder := json.NewDecoder(r.Body)
	var responseData config.ConfigPost
	jsonerr := decoder.Decode(&responseData)
	if jsonerr != nil {
		panic(jsonerr)
	}
	//Save change the settings
	for x, _ := range bot.config.Exchanges {
		for i := 0; i < len(responseData.Data.Exchanges); i++ {
			if responseData.Data.Exchanges[i].Name == bot.config.Exchanges[x].Name {
				bot.config.Exchanges[x].Enabled = responseData.Data.Exchanges[i].Enabled
				bot.config.Exchanges[x].APIKey = responseData.Data.Exchanges[i].APIKey
				bot.config.Exchanges[x].APISecret = responseData.Data.Exchanges[i].APISecret
				bot.config.Exchanges[x].EnabledPairs = responseData.Data.Exchanges[i].EnabledPairs
			}
		}
	}
	//Reload the configuration
	err := bot.config.SaveConfig("")
	if err != nil {
		panic(err)
	}
	err = bot.config.LoadConfig("")
	if err != nil {
		panic(err)
	}
	setupBotExchanges()
	//Return response status
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(bot.config); err != nil {
		panic(err)
	}
}

var ConfigRoutes = Routes{
	Route{
		"GetAllSettings",
		"GET",
		"/config/all",
		GetAllSettings,
	},

	Route{
		"SaveAllSettings",
		"POST",
		"/config/all/save",
		SaveAllSettings,
	},
}
