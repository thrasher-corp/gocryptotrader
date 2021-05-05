package exchangeratehost

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// A client for the exchangerate.host API. NOTE: The format and callback
// parameters aren't supported as they're not needed for this implementation.
// Furthermore, the source is set to "ECB" as default

const (
	timeLayout          = "2006-01-02"
	exchangeRateHostURL = "https://api.exchangerate.host"
)

var (
	// DefaultSource uses the ecb for forex rates
	DefaultSource = "ecb"
)

// Setup sets up the ExchangeRateHost config
func (e *ExchangeRateHost) Setup(config base.Settings) error {
	e.Name = config.Name
	e.Enabled = config.Enabled
	e.RESTPollingDelay = config.RESTPollingDelay
	e.Verbose = config.Verbose
	e.PrimaryProvider = config.PrimaryProvider
	e.Requester = request.New(e.Name,
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut))
	return nil
}

// GetLatestRates returns a list of forex rates based on the supplied params
func (e *ExchangeRateHost) GetLatestRates(baseCurrency, symbols string, amount float64, places int64, source string) (*LatestRates, error) {
	v := url.Values{}
	if baseCurrency != "" {
		v.Set("base", baseCurrency)
	}

	if symbols != "" {
		v.Set("symbols", symbols)
	}

	if amount != 0 {
		v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	if places != 0 {
		v.Set("places", strconv.FormatInt(places, 10))
	}

	targetSource := DefaultSource
	if source != "" {
		targetSource = source
	}
	v.Set("source", targetSource)

	var l LatestRates
	return &l, e.SendHTTPRequest("latest", v, &l)
}

// ConvertCurrency converts a currency based on the supplied params
func (e *ExchangeRateHost) ConvertCurrency(from, to, baseCurrency, symbols, source string, date time.Time, amount float64, places int64) (*ConvertCurrency, error) {
	v := url.Values{}
	if from != "" {
		v.Set("from", from)
	}
	if to != "" {
		v.Set("to", to)
	}
	if !date.IsZero() {
		v.Set("date", date.UTC().Format(timeLayout))
	}
	if baseCurrency != "" {
		v.Set("base", baseCurrency)
	}
	if symbols != "" {
		v.Set("symbols", symbols)
	}
	if amount != 0 {
		v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if places != 0 {
		v.Set("places", strconv.FormatInt(places, 10))
	}
	targetSource := DefaultSource
	if source != "" {
		targetSource = source
	}
	v.Set("source", targetSource)

	var c ConvertCurrency
	return &c, e.SendHTTPRequest("convert", v, &c)
}

// GetHistoricalRates returns a list of historical rates based on the supplied params
func (e *ExchangeRateHost) GetHistoricalRates(date time.Time, baseCurrency, symbols string, amount float64, places int64, source string) (*HistoricRates, error) {
	v := url.Values{}
	if date.IsZero() {
		date = time.Now()
	}
	fmtDate := date.UTC().Format(timeLayout)
	v.Set("date", fmtDate)
	if baseCurrency != "" {
		v.Set("base", baseCurrency)
	}

	if symbols != "" {
		v.Set("symbols", symbols)
	}

	if amount != 0 {
		v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	if places != 0 {
		v.Set("places", strconv.FormatInt(places, 10))
	}

	targetSource := DefaultSource
	if source != "" {
		targetSource = source
	}
	v.Set("source", targetSource)

	var h HistoricRates
	return &h, e.SendHTTPRequest(fmtDate, v, &h)
}

// GetTimeSeries returns time series forex data based on the supplied params
func (e *ExchangeRateHost) GetTimeSeries(startDate, endDate time.Time, baseCurrency, symbols string, amount float64, places int64, source string) (*TimeSeries, error) {
	if startDate.IsZero() || endDate.IsZero() {
		return nil, errors.New("startDate and endDate must be set")
	}

	if startDate.After(endDate) || startDate.Equal(endDate) {
		return nil, errors.New("startDate and endDate must be set correctly")
	}

	v := url.Values{}
	v.Set("start_date", startDate.UTC().Format(timeLayout))
	v.Set("end_date", endDate.UTC().Format(timeLayout))

	if baseCurrency != "" {
		v.Set("base", baseCurrency)
	}

	if symbols != "" {
		v.Set("symbols", symbols)
	}

	if amount != 0 {
		v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	if places != 0 {
		v.Set("places", strconv.FormatInt(places, 10))
	}

	targetSource := DefaultSource
	if source != "" {
		targetSource = source
	}
	v.Set("source", targetSource)

	var t TimeSeries
	return &t, e.SendHTTPRequest("timeseries", v, &t)
}

// GetFluctuations returns a list of forex price fluctuations based on the supplied params
func (e *ExchangeRateHost) GetFluctuations(startDate, endDate time.Time, baseCurrency, symbols string, amount float64, places int64, source string) (*Fluctuations, error) {
	if startDate.IsZero() || endDate.IsZero() {
		return nil, errors.New("startDate and endDate must be set")
	}

	if startDate.After(endDate) || startDate.Equal(endDate) {
		return nil, errors.New("startDate and endDate must be set correctly")
	}

	v := url.Values{}
	v.Set("start_date", startDate.UTC().Format(timeLayout))
	v.Set("end_date", endDate.UTC().Format(timeLayout))

	if baseCurrency != "" {
		v.Set("base", baseCurrency)
	}

	if symbols != "" {
		v.Set("symbols", symbols)
	}

	if amount != 0 {
		v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	if places != 0 {
		v.Set("places", strconv.FormatInt(places, 10))
	}

	targetSource := DefaultSource
	if source != "" {
		targetSource = source
	}
	v.Set("source", targetSource)

	var f Fluctuations
	return &f, e.SendHTTPRequest("fluctuation", v, &f)
}

// GetSupportedSymbols returns a list of supported symbols
func (e *ExchangeRateHost) GetSupportedSymbols() (*SupportedSymbols, error) {
	var s SupportedSymbols
	return &s, e.SendHTTPRequest("symbols", url.Values{}, &s)
}

// GetSupportedCurrencies returns a list of supported currencies
func (e *ExchangeRateHost) GetSupportedCurrencies() ([]string, error) {
	s, err := e.GetSupportedSymbols()
	if err != nil {
		return nil, err
	}

	var symbols []string
	for x := range s.Symbols {
		symbols = append(symbols, x)
	}
	return symbols, nil
}

// GetRates returns the forex rates based on the supplied base currency and symbols
func (e *ExchangeRateHost) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	l, err := e.GetLatestRates(baseCurrency, symbols, 0, 0, "")
	if err != nil {
		return nil, err
	}

	rates := make(map[string]float64)
	for k, v := range l.Rates {
		rates[baseCurrency+k] = v
	}
	return rates, nil
}

// SendHTTPRequest sends a typical get request
func (e *ExchangeRateHost) SendHTTPRequest(endpoint string, v url.Values, result interface{}) error {
	path := common.EncodeURLValues(exchangeRateHostURL+"/"+endpoint, v)
	return e.Requester.SendPayload(context.Background(), &request.Item{
		Method:  http.MethodGet,
		Path:    path,
		Result:  &result,
		Verbose: e.Verbose,
	})
}
