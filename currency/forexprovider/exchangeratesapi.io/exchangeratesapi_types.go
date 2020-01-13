package exchangerates

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	exchangeRatesAPI                 = "https://api.exchangeratesapi.io"
	exchangeRatesLatest              = "latest"
	exchangeRatesHistory             = "history"
	exchangeRatesSupportedCurrencies = "EUR,CHF,USD,BRL,ISK,PHP,KRW,BGN,MXN," +
		"RON,CAD,SGD,NZD,THB,HKD,JPY,NOK,HRK,ILS,GBP,DKK,HUF,MYR,RUB,TRY,IDR," +
		"ZAR,INR,AUD,CZK,SEK,CNY,PLN"

	rateLimitInterval = time.Second * 10
	requestRate       = 10
)

// ExchangeRates stores the struct for the ExchangeRatesAPI API
type ExchangeRates struct {
	base.Base
	Requester *request.Requester
}

// Rates holds the latest forex rates info
type Rates struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// HistoricalRates stores the historical rate info
type HistoricalRates Rates

// TimeSeriesRates stores time series rate info
type TimeSeriesRates struct {
	Base    string                 `json:"base"`
	StartAt string                 `json:"start_at"`
	EndAt   string                 `json:"end_at"`
	Rates   map[string]interface{} `json:"rates"`
}
