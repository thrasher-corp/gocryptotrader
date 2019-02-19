package exchangerates

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	exchangeRatesAPI                 = "https://api.exchangeratesapi.io"
	exchangeRatesLatest              = "latest"
	exchangeRatesSupportedCurrencies = "USD,ISK,CAD,MXN,CHF,AUD,CNY,GBP,SEK,NOK,TRY,IDR,ZAR," +
		"HRK,EUR,HKD,ILS,NZD,MYR,JPY,CZK,JPY,CZK,SGD,RUB,RON,HUF,BGN,INR,KRW," +
		"DKK,THB,PHP,PLN,BRL"
)

// ExchangeRates stores the struct for the ExchangeRatesAPI API
type ExchangeRates struct {
	base.Base
}

// Setup sets appropriate values for CurrencyLayer
func (e *ExchangeRates) Setup(config base.Settings) {
	e.Name = config.Name
	e.Enabled = config.Enabled
	e.RESTPollingDelay = config.RESTPollingDelay
	e.Verbose = config.Verbose
	e.PrimaryProvider = config.PrimaryProvider
}

func cleanCurrencies(baseCurrency, symbols string) string {
	var cleanedCurrencies []string
	symbols = strings.Replace(symbols, "RUR", "RUB", -1)
	var s = strings.Split(symbols, ",")
	for _, x := range s {
		// first make sure that the baseCurrency is not in the symbols list
		// if it is set
		if baseCurrency != "" {
			if x == baseCurrency {
				continue
			}
		} else {
			// otherwise since the baseCurrency is empty, make sure that it
			// does not exist in the symbols list
			if x == "EUR" {
				continue
			}
		}

		// remove and warn about any unsupported currencies
		if !common.StringContains(exchangeRatesSupportedCurrencies, x) {
			log.Warnf("Forex provider ExchangeRatesAPI does not support currency %s, removing from forex rates query.", x)
			continue
		}
		cleanedCurrencies = append(cleanedCurrencies, x)
	}
	return strings.Join(cleanedCurrencies, ",")
}

// GetLatestRates returns a map of forex rates based on the supplied params
// baseCurrency [optional] - the base currency to use for forex rates, defaults to EUR
// symbols [optional] - the symbols to query the forex rates for, default is
// all supported currencies
func (e *ExchangeRates) GetLatestRates(baseCurrency, symbols string) (Latest, error) {
	vals := url.Values{}

	if len(baseCurrency) > 0 {
		vals.Set("base", baseCurrency)
	}

	if len(symbols) > 0 {
		symbols = cleanCurrencies(baseCurrency, symbols)
		vals.Set("symbols", symbols)
	}

	var result Latest
	return result, e.SendHTTPRequest(exchangeRatesLatest, vals, &result)
}

// GetRates is a wrapper function to return forex rates
func (e *ExchangeRates) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	result, err := e.GetLatestRates(baseCurrency, symbols)
	if err != nil {
		return nil, err
	}

	standardisedRates := make(map[string]float64)
	for k, v := range result.Rates {
		curr := baseCurrency + k
		standardisedRates[curr] = v
	}

	return standardisedRates, nil
}

// SendHTTPRequest sends a HTTPS request to the desired endpoint and returns the result
func (e *ExchangeRates) SendHTTPRequest(endPoint string, values url.Values, result interface{}) error {
	path := common.EncodeURLValues(exchangeRatesAPI+"/"+endPoint, values)
	err := common.SendHTTPGetRequest(path, true, e.Verbose, &result)
	if err != nil {
		return fmt.Errorf("ExchangeRatesAPI SendHTTPRequest error %s with path %s",
			err,
			path)
	}
	return nil
}
