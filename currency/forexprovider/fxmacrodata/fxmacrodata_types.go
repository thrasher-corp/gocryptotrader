package fxmacrodata

import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// APIURL is the default FXMacroData API endpoint.
	APIURL = "https://fxmacrodata.com/api/v1/"

	supportedCurrencies = "AUD,BRL,CAD,CHF,CNH,CNY,DKK,EUR,GBP,ILS,JPY,NGN,NOK,NZD,PEN,SEK,THB,USD"
)

// FXMacroData is an FXMacroData foreign exchange and macro data provider.
type FXMacroData struct {
	base.Base
	Requester *request.Requester
	APIURL    string
}

type forexResponse struct {
	Data []struct {
		Val float64 `json:"val"`
	} `json:"data"`
}

// ServiceStatusResponse represents a public FXMacroData service status response.
type ServiceStatusResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}
