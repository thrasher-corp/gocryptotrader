package currencyconverter

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// const declarations consist of endpoints
const (
	APIEndpointURL     = "https://currencyconverterapi.com/api/"
	APIEndpointFreeURL = "https://free.currencyconverterapi.com/api/"
	APIEndpointVersion = "v5"

	APIEndpointConvert    = "convert"
	APIEndpointCurrencies = "currencies"
	APIEndpointCountries  = "countries"
	APIEndpointUsage      = "usage"

	defaultAPIKey = "Key"

	rateInterval = time.Hour
	requestRate  = 100
)

// CurrencyConverter stores the struct for the CurrencyConverter API
type CurrencyConverter struct {
	base.Base
	Requester *request.Requester
}

// Error stores the error message
type Error struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

// CurrencyItem stores variables related to the currency response
type CurrencyItem struct {
	CurrencyName   string `json:"currencyName"`
	CurrencySymbol string `json:"currencySymbol"`
	ID             string `json:"ID"`
}

// Currencies stores the currency result data
type Currencies struct {
	Results map[string]CurrencyItem
}

// CountryItem stores variables related to the country response
type CountryItem struct {
	Alpha3         string `json:"alpha3"`
	CurrencyID     string `json:"currencyId"`
	CurrencyName   string `json:"currencyName"`
	CurrencySymbol string `json:"currencySymbol"`
	ID             string `json:"ID"`
	Name           string `json:"Name"`
}

// Countries stores the country result data
type Countries struct {
	Results map[string]CountryItem
}
