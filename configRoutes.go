package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func getAllSettings(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(bot.config); err != nil {
		panic(err)
	}
}

func saveAllSettings(w http.ResponseWriter, r *http.Request) {
	//Get the data from the request
	log.Println(r.Body)
	decoder := json.NewDecoder(r.Body)
	var responseData ConfigPost
	jsonerr := decoder.Decode(&responseData)
	if jsonerr != nil {
		log.Println(jsonerr)
		panic(jsonerr)
	}
	//Save change the settings
	for _, exch := range bot.config.Exchanges {
		for i := 0; i < len(responseData.Data.Exchanges); i++ {
			if responseData.Data.Exchanges[i].Name == exch.Name {
				log.Println("Looking at exchange " + exch.Name)
				log.Println("Enabled  %s", responseData.Data.Exchanges[i].Enabled)
				log.Println("Key " + responseData.Data.Exchanges[i].APIKey)
				exch.Enabled = responseData.Data.Exchanges[i].Enabled
				exch.APIKey = responseData.Data.Exchanges[i].APIKey
				exch.APISecret = responseData.Data.Exchanges[i].APISecret
				exch.EnabledPairs = responseData.Data.Exchanges[i].EnabledPairs
			}
		}
	}
	//Reload the configuration
	err := SaveConfig()
	if err != nil {
		log.Println("Fatal error checking config values. Error:", err)
		return
	}
	bot.config, err = ReadConfig()
	if err != nil {
		log.Println("Fatal error checking config values. Error:", err)
		return
	}
	//Return response status
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(bot.config); err != nil {
		panic(err)
	}
}

var configRoutes = Routes{
	Route{
		"GetAllSettings",
		"GET",
		"/config/all",
		getAllSettings,
	},

	Route{
		"SaveAllSettings",
		"POST",
		"/config/all/save",
		saveAllSettings,
	},
}
