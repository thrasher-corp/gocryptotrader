package main

import (
	"encoding/json"
	"net/http"
)

func getAllSettings(w http.ResponseWriter, r *http.Request) {

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
}
