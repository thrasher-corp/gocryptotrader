// Powered by 15+ exchange rate data sources, the Fixer API is capable of
// delivering real-time exchange rate data for 170 world currencies. The API
// comes with multiple endpoints, each serving a different use case. Endpoint
// functionalities include getting the latest exchange rate data for all or a
// specific set of currencies, converting amounts from one currency to another,
// retrieving Time-Series data for one or multiple currencies and querying the
// API for daily fluctuation data.

package fixer

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
)

const (
	fixerAPIFree = iota
	fixerAPIBasic
	fixerAPIProfessional
	fixerAPIProfessionalPlus
	fixerAPIEnterprise

	fixerAPI            = "http://data.fixer.io/api/"
	fixerAPISSL         = "https://data.fixer.io/api/"
	fixerAPILatest      = "latest"
	fixerAPIConvert     = "convert"
	fixerAPITimeSeries  = "timeseries"
	fixerAPIFluctuation = "fluctuation"
)

// Fixer is a foreign exchange rate provider at https://fixer.io/
// NOTE DEFAULT BASE CURRENCY IS EUR upgrade to basic to change
type Fixer struct {
	base.Base
}

// Setup sets appropriate values for fixer object
func (f *Fixer) Setup(config base.Settings) {
	f.APIKey = config.APIKey
	f.APIKeyLvl = config.APIKeyLvl
	f.Enabled = config.Enabled
	f.Name = config.Name
	f.RESTPollingDelay = config.RESTPollingDelay
	f.Verbose = config.Verbose
	f.PrimaryProvider = config.PrimaryProvider
}

// GetRates is a wrapper function to return rates
func (f *Fixer) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	rates, err := f.GetLatestRates(baseCurrency, symbols)
	if err != nil {
		return nil, err
	}

	if f.APIKeyLvl == fixerAPIFree {
		baseCurrency = "EUR"
	}

	standardisedRates := make(map[string]float64)
	for k, v := range rates {
		curr := baseCurrency + k
		standardisedRates[curr] = v
	}

	return standardisedRates, nil
}

// GetLatestRates returns real-time exchange rate data for all available or a
// specific set of currencies. NOTE DEFAULT BASE CURRENCY IS EUR
func (f *Fixer) GetLatestRates(base, symbols string) (map[string]float64, error) {
	var resp Rates

	v := url.Values{}

	if f.APIKeyLvl > fixerAPIFree {
		v.Add("base", base)
	}
	v.Add("symbols", symbols)

	err := f.SendOpenHTTPRequest(fixerAPILatest, v, &resp)
	if err != nil {
		return resp.Rates, err
	}

	if !resp.Success {
		return resp.Rates, errors.New(resp.Error.Type + resp.Error.Info)
	}

	return resp.Rates, nil
}

// GetHistoricalRates returns historical exchange rate data for all available or
// a specific set of currencies.
// date - YYYY-MM-DD	[required] A date in the past
func (f *Fixer) GetHistoricalRates(date, base string, symbols []string) (map[string]float64, error) {
	var resp Rates

	v := url.Values{}
	v.Set("symbols", common.JoinStrings(symbols, ","))

	err := f.SendOpenHTTPRequest(date, v, &resp)
	if err != nil {
		return resp.Rates, err
	}

	if !resp.Success {
		return resp.Rates, errors.New(resp.Error.Type + resp.Error.Info)
	}
	return resp.Rates, nil
}

// ConvertCurrency allows for conversion of any amount from one currency to
// another.
// from - The three-letter currency code of the currency you would like to
// convert from.
// to - The three-letter currency code of the currency you would like to convert
// to.
// amount - The amount to be converted.
// date - [optional] Specify a date (format YYYY-MM-DD) to use historical rates
// for this conversion.
func (f *Fixer) ConvertCurrency(from, to, date string, amount float64) (float64, error) {
	if f.APIKeyLvl < fixerAPIBasic {
		return 0, errors.New("insufficient API privileges, upgrade to basic to use this function")
	}

	var resp Conversion

	v := url.Values{}
	v.Set("from", from)
	v.Set("to", to)
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("date", date)

	err := f.SendOpenHTTPRequest(fixerAPIConvert, v, &resp)
	if err != nil {
		return resp.Result, err
	}

	if !resp.Success {
		return resp.Result, errors.New(resp.Error.Type + resp.Error.Info)
	}
	return resp.Result, nil
}

// GetTimeSeriesData returns daily historical exchange rate data between two
// specified dates for all available or a specific set of currencies.
func (f *Fixer) GetTimeSeriesData(startDate, endDate, base string, symbols []string) (map[string]interface{}, error) {
	if f.APIKeyLvl < fixerAPIProfessional {
		return nil, errors.New("insufficient API privileges, upgrade to professional to use this function")
	}

	var resp TimeSeries

	v := url.Values{}
	v.Set("start_date", startDate)
	v.Set("end_date", endDate)
	v.Set("base", base)
	v.Set("symbols", common.JoinStrings(symbols, ","))

	err := f.SendOpenHTTPRequest(fixerAPITimeSeries, v, &resp)
	if err != nil {
		return resp.Rates, err
	}

	if !resp.Success {
		return resp.Rates, errors.New(resp.Error.Type + resp.Error.Info)
	}
	return resp.Rates, nil
}

// GetFluctuationData returns fluctuation data between two specified dates for
// all available or a specific set of currencies.
func (f *Fixer) GetFluctuationData(startDate, endDate, base string, symbols []string) (map[string]Flux, error) {
	if f.APIKeyLvl < fixerAPIProfessionalPlus {
		return nil, errors.New("insufficient API privileges, upgrade to professional plus or enterprise to use this function")
	}

	var resp Fluctuation

	v := url.Values{}
	v.Set("start_date", startDate)
	v.Set("end_date", endDate)
	v.Set("base", base)
	v.Set("symbols", common.JoinStrings(symbols, ","))

	err := f.SendOpenHTTPRequest(fixerAPIFluctuation, v, &resp)
	if err != nil {
		return resp.Rates, err
	}

	if !resp.Success {
		return resp.Rates, errors.New(resp.Error.Type + resp.Error.Info)
	}
	return resp.Rates, nil
}

// SendOpenHTTPRequest sends a typical get request
func (f *Fixer) SendOpenHTTPRequest(endpoint string, v url.Values, result interface{}) error {
	var path string
	v.Set("access_key", f.APIKey)

	if f.APIKeyLvl == fixerAPIFree {
		path = fixerAPI + endpoint + "?" + v.Encode()
	} else {
		path = fixerAPISSL + endpoint + "?" + v.Encode()
	}
	return common.SendHTTPGetRequest(path, true, f.Verbose, result)
}
