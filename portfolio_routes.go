package main

import (
	"encoding/json"
	"net/http"
)

// RESTGetPortfolio replies to a request with an encoded JSON response of the
// portfolio
func RESTGetPortfolio(w http.ResponseWriter, r *http.Request) {
	result := bot.portfolio.GetPortfolioSummary()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		panic(err)
	}
}

// PortfolioRoutes declares the current routes for config_routes.go
var PortfolioRoutes = Routes{
	Route{
		"GetPortfolio",
		"GET",
		"/portfolio/all",
		RESTGetPortfolio,
	},
}
