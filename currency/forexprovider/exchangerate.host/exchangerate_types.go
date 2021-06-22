package exchangeratehost

import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// ExchangeRateHost stores the struct for the exchangerate.host API
type ExchangeRateHost struct {
	base.Base
	Requester *request.Requester
}

// MessageOfTheDay stores the message of the day
type MessageOfTheDay struct {
	Message     string `json:"msg"`
	DonationURL string `json:"url"`
}

// LatestRates stores the latest forex rates
type LatestRates struct {
	MessageOfTheDay MessageOfTheDay    `json:"motd"`
	Success         bool               `json:"success"`
	Base            string             `json:"base"`
	Date            string             `json:"date"`
	Rates           map[string]float64 `json:"rates"`
}

// ConvertCurrency stores currency conversion data
type ConvertCurrency struct {
	MessageOfTheDay MessageOfTheDay `json:"motd"`
	Query           struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	} `json:"query"`
	Info struct {
		Rate float64 `json:"rate"`
	} `json:"info"`
	Historical bool    `json:"historical"`
	Date       string  `json:"date"`
	Result     float64 `json:"result"`
}

// HistoricRates stores the hostoric rates
type HistoricRates struct {
	LatestRates
	Historical bool `json:"historical"`
}

// TimeSeries stores time series data
type TimeSeries struct {
	MessageOfTheDay MessageOfTheDay               `json:"motd"`
	Success         bool                          `json:"success"`
	TimeSeries      bool                          `json:"timeseries"`
	Base            string                        `json:"base"`
	StartDate       string                        `json:"start_date"`
	EndDate         string                        `json:"end_date"`
	Rates           map[string]map[string]float64 `json:"rates"`
}

// Fluctuation stores an individual rate flucutation
type Fluctuation struct {
	StartRate        float64 `json:"start_rate"`
	EndRate          float64 `json:"end_rate"`
	Change           float64 `json:"change"`
	ChangePercentage float64 `json:"change_pct"`
}

// Fluctuations stores a collection of rate fluctuations
type Fluctuations struct {
	MessageOfTheDay MessageOfTheDay        `json:"motd"`
	Success         bool                   `json:"success"`
	Flucutation     bool                   `json:"fluctuation"`
	StartDate       string                 `json:"start_date"`
	EndDate         string                 `json:"end_date"`
	Rates           map[string]Fluctuation `json:"rate"`
}

// Symbol stores an individual symbol
type Symbol struct {
	Description string `json:"description"`
	Code        string `json:"code"`
}

// SupportedSymbols store a collection of supported symbols
type SupportedSymbols struct {
	MessageOfTheDay MessageOfTheDay   `json:"motd"`
	Success         bool              `json:"success"`
	Symbols         map[string]Symbol `json:"symbols"`
}
