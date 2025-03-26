// Powered by 15+ exchange rate data sources, the Fixer API is capable of
// delivering real-time exchange rate data for 170 world currencies. The API
// comes with multiple endpoints, each serving a different use case. Endpoint
// functionalities include getting the latest exchange rate data for all or a
// specific set of currencies, converting amounts from one currency to another,
// retrieving Time-Series data for one or multiple currencies and querying the
// API for daily fluctuation data.

package fixer

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup sets appropriate values for fixer object
func (f *Fixer) Setup(config base.Settings) error {
	if config.APIKeyLvl < 0 || config.APIKeyLvl > 4 {
		log.Errorf(log.Global,
			"apikey incorrectly set in config.json for %s, please set appropriate account levels\n",
			config.Name)
		return errors.New("apikey set failure")
	}
	f.APIKey = config.APIKey
	f.APIKeyLvl = config.APIKeyLvl
	f.Enabled = config.Enabled
	f.Name = config.Name
	f.Verbose = config.Verbose
	f.PrimaryProvider = config.PrimaryProvider
	var err error
	f.Requester, err = request.New(f.Name,
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut))
	return err
}

// GetSupportedCurrencies returns supported currencies
func (f *Fixer) GetSupportedCurrencies() ([]string, error) {
	var resp Symbols

	err := f.SendOpenHTTPRequest(fixerSupportedCurrencies, nil, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.Error.Type + resp.Error.Info)
	}

	currencies := make([]string, 0, len(resp.Map))
	for key := range resp.Map {
		currencies = append(currencies, key)
	}

	return currencies, nil
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
func (f *Fixer) GetLatestRates(baseCurrency, symbols string) (map[string]float64, error) {
	var resp Rates

	v := url.Values{}

	if f.APIKeyLvl > fixerAPIFree {
		v.Add("base", baseCurrency)
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
// base - USD 			[optional]
// symbols - the desired symbols
func (f *Fixer) GetHistoricalRates(date, baseCurrency string, symbols []string) (map[string]float64, error) {
	var resp Rates

	v := url.Values{}
	v.Set("symbols", strings.Join(symbols, ","))

	if baseCurrency != "" {
		v.Set("base", baseCurrency)
	}

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
func (f *Fixer) GetTimeSeriesData(startDate, endDate, baseCurrency string, symbols []string) (map[string]any, error) {
	if f.APIKeyLvl < fixerAPIProfessional {
		return nil, errors.New("insufficient API privileges, upgrade to professional to use this function")
	}

	var resp TimeSeries

	v := url.Values{}
	v.Set("start_date", startDate)
	v.Set("end_date", endDate)
	v.Set("base", baseCurrency)
	v.Set("symbols", strings.Join(symbols, ","))

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
func (f *Fixer) GetFluctuationData(startDate, endDate, baseCurrency string, symbols []string) (map[string]Flux, error) {
	if f.APIKeyLvl < fixerAPIProfessionalPlus {
		return nil, errors.New("insufficient API privileges, upgrade to professional plus or enterprise to use this function")
	}

	var resp Fluctuation

	v := url.Values{}
	v.Set("start_date", startDate)
	v.Set("end_date", endDate)
	v.Set("base", baseCurrency)
	v.Set("symbols", strings.Join(symbols, ","))

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
func (f *Fixer) SendOpenHTTPRequest(endpoint string, v url.Values, result any) error {
	if v == nil {
		v = url.Values{}
	}
	v.Set("access_key", f.APIKey)

	var auth request.AuthType
	var path string
	if f.APIKeyLvl == fixerAPIFree {
		path = fixerAPI + endpoint + "?" + v.Encode()
		auth = request.UnauthenticatedRequest
	} else {
		path = fixerAPISSL + endpoint + "?" + v.Encode()
		auth = request.AuthenticatedRequest
	}
	item := &request.Item{
		Method:  http.MethodGet,
		Path:    path,
		Result:  &result,
		Verbose: f.Verbose,
	}
	return f.Requester.SendPayload(context.Background(), request.Unset, func() (*request.Item, error) {
		return item, nil
	}, auth)
}
