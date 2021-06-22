package exchangerates

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	exchangeRatesAPI         = "api.exchangeratesapi.io"
	exchangeRatesLatest      = "latest"
	exchangeRatesTimeSeries  = "timeseries"
	exchangeRatesConvert     = "convert"
	exchangeRatesFluctuation = "fluctuation"

	rateLimitInterval = time.Second * 10
	requestRate       = 10
	timeLayout        = "2006-01-02"

	apiKeyFree = iota
	apiKeyBasic
	apiKeyProfessional
	apiKeyBusiness
)

var (
	errCannotSetBaseCurrencyOnFreePlan = errors.New("base currency cannot be set on the free plan")
	errAPIKeyLevelRestrictedAccess     = errors.New("apiKey level function access denied")
	errStartEndDatesInvalid            = errors.New("startDate and endDate params must be set")
	errStartAfterEnd                   = errors.New("startDate must be before endDate")
)

// ExchangeRates stores the struct for the ExchangeRatesAPI API
type ExchangeRates struct {
	base.Base
	supportedCurrencies []string
	Requester           *request.Requester
}

// Rates holds the latest forex rates info
type Rates struct {
	Base      string             `json:"base"`
	Timestamp int64              `json:"timestamp"`
	Date      string             `json:"date"`
	Rates     map[string]float64 `json:"rates"`
}

// HistoricalRates stores the historical rate info
type HistoricalRates struct {
	Historical bool `json:"historical"`
	Rates
}

// ConvertCurrency stores the converted currency info
type ConvertCurrency struct {
	Query struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	} `json:"query"`
	Info struct {
		Timestamp int64   `json:"timestamp"`
		Rate      float64 `json:"rate"`
	} `json:"info"`
	Historical bool    `json:"historical"`
	Result     float64 `json:"result"`
}

// TimeSeriesRates stores time series rate info
type TimeSeriesRates struct {
	Timeseries bool                          `json:"timeseries"`
	StartDate  string                        `json:"start_date"`
	EndDate    string                        `json:"end_date"`
	Base       string                        `json:"base"`
	Rates      map[string]map[string]float64 `json:"rates"`
}

// FlucutationItem stores an individual rate fluctuation
type FlucutationItem struct {
	StartRate        float64 `json:"start_rate"`
	EndRate          float64 `json:"end_rate"`
	Change           float64 `json:"change"`
	ChangePercentage float64 `json:"change_pct"`
}

// Fluctuations stores a collection of rate fluctuations
type Fluctuations struct {
	Fluctuation bool                       `json:"fluctuation"`
	StartDate   string                     `json:"start_date"`
	EndDate     string                     `json:"end_date"`
	Base        string                     `json:"base"`
	Rates       map[string]FlucutationItem `json:"rates"`
}
