package currencylayer

import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// const declarations consist of endpoints and APIKey privileges
const (
	AccountFree = iota
	AccountBasic
	AccountPro
	AccountEnterprise

	APIEndpointURL        = "http://apilayer.net/api/"
	APIEndpointURLSSL     = "https://apilayer.net/api/"
	APIEndpointList       = "list"
	APIEndpointLive       = "live"
	APIEndpointHistorical = "historical"
	APIEndpointConversion = "convert"
	APIEndpointTimeframe  = "timeframe"
	APIEndpointChange     = "change"
)

// CurrencyLayer is a foreign exchange rate provider at https://currencylayer.com
// NOTE default base currency is USD when using a free account
type CurrencyLayer struct {
	base.Base
	Requester *request.Requester
}

// Error Defines the response error if an error occurred
type Error struct {
	Code int    `json:"code"`
	Type string `json:"type"`
	Info string `json:"info"`
}

// LiveRates is a response type holding rates priced now.
type LiveRates struct {
	Success   bool               `json:"success"`
	Error     Error              `json:"error"`
	Terms     string             `json:"terms"`
	Privacy   string             `json:"privacy"`
	Timestamp int64              `json:"timestamp"`
	Source    string             `json:"source"`
	Quotes    map[string]float64 `json:"quotes"`
}

// SupportedCurrencies holds supported currency information
type SupportedCurrencies struct {
	Success    bool              `json:"success"`
	Error      Error             `json:"error"`
	Terms      string            `json:"terms"`
	Privacy    string            `json:"privacy"`
	Currencies map[string]string `json:"currencies"`
}

// HistoricalRates is a response type holding rates priced from the past.
type HistoricalRates struct {
	Success    bool               `json:"success"`
	Error      Error              `json:"error"`
	Terms      string             `json:"terms"`
	Privacy    string             `json:"privacy"`
	Historical bool               `json:"historical"`
	Date       string             `json:"date"`
	Timestamp  int64              `json:"timestamp"`
	Source     string             `json:"source"`
	Quotes     map[string]float64 `json:"quotes"`
}

// ConversionRate is a response type holding a converted rate.
type ConversionRate struct {
	Success bool   `json:"success"`
	Error   Error  `json:"error"`
	Privacy string `json:"privacy"`
	Terms   string `json:"terms"`
	Query   struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	} `json:"query"`
	Info struct {
		Timestamp int64   `json:"timestamp"`
		Quote     float64 `json:"quote"`
	} `json:"info"`
	Historical bool    `json:"historical"`
	Date       string  `json:"date"`
	Result     float64 `json:"result"`
}

// TimeFrame is a response type holding exchange rates for a time period
type TimeFrame struct {
	Success   bool           `json:"success"`
	Error     Error          `json:"error"`
	Terms     string         `json:"terms"`
	Privacy   string         `json:"privacy"`
	Timeframe bool           `json:"timeframe"`
	StartDate string         `json:"start_date"`
	EndDate   string         `json:"end_date"`
	Source    string         `json:"source"`
	Quotes    map[string]any `json:"quotes"`
}

// ChangeRate is the response type that holds rate change data.
type ChangeRate struct {
	Success   bool               `json:"success"`
	Error     Error              `json:"error"`
	Terms     string             `json:"terms"`
	Privacy   string             `json:"privacy"`
	Change    bool               `json:"change"`
	StartDate string             `json:"start_date"`
	EndDate   string             `json:"end_date"`
	Source    string             `json:"source"`
	Quotes    map[string]Changes `json:"quotes"`
}

// Changes is a sub-type of ChangeRate that holds the actual changes of rates.
type Changes struct {
	StartRate float64 `json:"start_rate"`
	EndRate   float64 `json:"end_rate"`
	Change    float64 `json:"change"`
	ChangePCT float64 `json:"change_pct"`
}
