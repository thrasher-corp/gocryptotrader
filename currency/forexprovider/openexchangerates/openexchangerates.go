// Open Exchange Rates provides a simple, lightweight and portable JSON API with
// live and historical foreign exchange (forex) rates, via a simple and
// easy-to-integrate API, in JSON format. Data are tracked and blended
// algorithmically from multiple reliable sources, ensuring fair and unbiased
// consistency.
// End-of-day rates are available historically for all days going back to
// 1st January, 1999.

package openexchangerates

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// These consts contain endpoint information
const (
	APIDeveloperAccess = iota
	APIEnterpriseAccess
	APIUnlimitedAccess

	APIURL                = "https://openexchangerates.org/api/"
	APIEndpointLatest     = "latest.json"
	APIEndpointHistorical = "historical/%s.json"
	APIEndpointCurrencies = "currencies.json"
	APIEndpointTimeSeries = "time-series.json"
	APIEndpointConvert    = "convert/%s/%s/%s"
	APIEndpointOHLC       = "ohlc.json"
	APIEndpointUsage      = "usage.json"

	oxrSupportedCurrencies = "AED,AFN,ALL,AMD,ANG,AOA,ARS,AUD,AWG,AZN,BAM,BBD," +
		"BDT,BGN,BHD,BIF,BMD,BND,BOB,BRL,BSD,BTC,BTN,BWP,BYN,BYR,BZD,CAD,CDF," +
		"CHF,CLF,CLP,CNH,CNY,COP,CRC,CUC,CUP,CVE,CZK,DJF,DKK,DOP,DZD,EEK,EGP," +
		"ERN,ETB,EUR,FJD,FKP,GBP,GEL,GGP,GHS,GIP,GMD,GNF,GTQ,GYD,HKD,HNL,HRK," +
		"HTG,HUF,IDR,ILS,IMP,INR,IQD,IRR,ISK,JEP,JMD,JOD,JPY,KES,KGS,KHR,KMF," +
		"KPW,KRW,KWD,KYD,KZT,LAK,LBP,LKR,LRD,LSL,LYD,MAD,MDL,MGA,MKD,MMK,MNT," +
		"MOP,MRO,MRU,MTL,MUR,MVR,MWK,MXN,MYR,MZN,NAD,NGN,NIO,NOK,NPR,NZD,OMR," +
		"PAB,PEN,PGK,PHP,PKR,PLN,PYG,QAR,RON,RSD,RUB,RWF,SAR,SBD,SCR,SDG,SEK," +
		"SGD,SHP,SLL,SOS,SRD,SSP,STD,STN,SVC,SYP,SZL,THB,TJS,TMT,TND,TOP,TRY," +
		"TTD,TWD,TZS,UAH,UGX,USD,UYU,UZS,VEF,VND,VUV,WST,XAF,XAG,XAU,XCD,XDR," +
		"XOF,XPD,XPF,XPT,YER,ZAR,ZMK,ZMW"

	authRate   = 0
	unAuthRate = 0
)

// OXR is a foreign exchange rate provider at https://openexchangerates.org/
// this is the overarching type across this package
// DOCs : https://docs.openexchangerates.org/docs
type OXR struct {
	base.Base
	Requester *request.Requester
}

// Setup sets values for the OXR object
func (o *OXR) Setup(config base.Settings) error {
	if config.APIKeyLvl < 0 || config.APIKeyLvl > 2 {
		log.Errorf("apikey incorrectly set in config.json for %s, please set appropriate account levels",
			config.Name)
		return errors.New("apikey set failure")
	}
	o.APIKey = config.APIKey
	o.APIKeyLvl = config.APIKeyLvl
	o.Enabled = config.Enabled
	o.Name = config.Name
	o.RESTPollingDelay = config.RESTPollingDelay
	o.Verbose = config.Verbose
	o.PrimaryProvider = config.PrimaryProvider
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second*10, authRate),
		request.NewRateLimit(time.Second*10, unAuthRate),
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut))
	return nil
}

// GetRates is a wrapper function to return rates
func (o *OXR) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	rates, err := o.GetLatest(baseCurrency, symbols, false, false)
	if err != nil {
		return nil, err
	}

	standardisedRates := make(map[string]float64)
	for k, v := range rates {
		curr := baseCurrency + k
		standardisedRates[curr] = v
	}

	return standardisedRates, nil
}

// GetLatest returns the latest exchange rates available from the Open Exchange
// Rates
func (o *OXR) GetLatest(baseCurrency, symbols string, prettyPrint, showAlternative bool) (map[string]float64, error) {
	var resp Latest

	v := url.Values{}
	v.Set("base", baseCurrency)
	v.Set("symbols", symbols)
	v.Set("prettyprint", strconv.FormatBool(prettyPrint))
	v.Set("show_alternative", strconv.FormatBool(showAlternative))

	if err := o.SendHTTPRequest(APIEndpointLatest, v, &resp); err != nil {
		return nil, err
	}

	if resp.Error {
		return nil, errors.New(resp.Message)
	}
	return resp.Rates, nil
}

// GetHistoricalRates returns historical exchange rates for any date available
// from the Open Exchange Rates API.
func (o *OXR) GetHistoricalRates(date, baseCurrency string, symbols []string, prettyPrint, showAlternative bool) (map[string]float64, error) {
	var resp Latest

	v := url.Values{}
	v.Set("base", baseCurrency)
	v.Set("symbols", common.JoinStrings(symbols, ","))
	v.Set("prettyprint", strconv.FormatBool(prettyPrint))
	v.Set("show_alternative", strconv.FormatBool(showAlternative))
	endpoint := fmt.Sprintf(APIEndpointHistorical, date)

	if err := o.SendHTTPRequest(endpoint, v, &resp); err != nil {
		return nil, err
	}

	if resp.Error {
		return nil, errors.New(resp.Message)
	}
	return resp.Rates, nil
}

// GetCurrencies returns a list of all currency symbols available from the Open
// Exchange Rates API,
func (o *OXR) GetCurrencies(showInactive, prettyPrint, showAlternative bool) (map[string]string, error) {
	resp := make(map[string]string)

	v := url.Values{}
	v.Set("show_inactive", strconv.FormatBool(showInactive))
	v.Set("prettyprint", strconv.FormatBool(prettyPrint))
	v.Set("show_alternative", strconv.FormatBool(showAlternative))

	return resp, o.SendHTTPRequest(APIEndpointCurrencies, v, &resp)
}

// GetSupportedCurrencies returns a list of supported currencies
func (o *OXR) GetSupportedCurrencies() ([]string, error) {
	return common.SplitStrings(oxrSupportedCurrencies, ","), nil
}

// GetTimeSeries returns historical exchange rates for a given time period,
// where available.
func (o *OXR) GetTimeSeries(baseCurrency, startDate, endDate string, symbols []string, prettyPrint, showAlternative bool) (map[string]interface{}, error) {
	if o.APIKeyLvl < APIEnterpriseAccess {
		return nil, errors.New("upgrade account, insufficient access")
	}

	var resp TimeSeries

	v := url.Values{}
	v.Set("base", baseCurrency)
	v.Set("start", startDate)
	v.Set("end", endDate)
	v.Set("symbols", common.JoinStrings(symbols, ","))
	v.Set("prettyprint", strconv.FormatBool(prettyPrint))
	v.Set("show_alternative", strconv.FormatBool(showAlternative))

	if err := o.SendHTTPRequest(APIEndpointTimeSeries, v, &resp); err != nil {
		return nil, err
	}

	if resp.Error {
		return nil, errors.New(resp.Message)
	}
	return resp.Rates, nil
}

// ConvertCurrency converts any money value from one currency to another at the
// latest API rates
func (o *OXR) ConvertCurrency(amount float64, from, to string) (float64, error) {
	if o.APIKeyLvl < APIUnlimitedAccess {
		return 0, errors.New("upgrade account, insufficient access")
	}

	var resp Convert

	endPoint := fmt.Sprintf(APIEndpointConvert, strconv.FormatFloat(amount, 'f', -1, 64), from, to)
	if err := o.SendHTTPRequest(endPoint, url.Values{}, &resp); err != nil {
		return 0, err
	}

	if resp.Error {
		return 0, errors.New(resp.Message)
	}
	return resp.Response, nil
}

// GetOHLC returns historical Open, High Low, Close (OHLC) and Average exchange
// rates for a given time period, ranging from 1 month to 1 minute, where
// available.
func (o *OXR) GetOHLC(startTime, period, baseCurrency string, symbols []string, prettyPrint bool) (map[string]interface{}, error) {
	if o.APIKeyLvl < APIUnlimitedAccess {
		return nil, errors.New("upgrade account, insufficient access")
	}

	var resp OHLC

	v := url.Values{}
	v.Set("start_time", startTime)
	v.Set("period", period)
	v.Set("base", baseCurrency)
	v.Set("symbols", common.JoinStrings(symbols, ","))
	v.Set("prettyprint", strconv.FormatBool(prettyPrint))

	if err := o.SendHTTPRequest(APIEndpointOHLC, v, &resp); err != nil {
		return nil, err
	}

	if resp.Error {
		return nil, errors.New(resp.Message)
	}
	return resp.Rates, nil
}

// GetUsageStats returns basic plan information and usage statistics for an Open
// Exchange Rates App ID
func (o *OXR) GetUsageStats(prettyPrint bool) (Usage, error) {
	var resp Usage

	v := url.Values{}
	v.Set("prettyprint", strconv.FormatBool(prettyPrint))

	if err := o.SendHTTPRequest(APIEndpointUsage, v, &resp); err != nil {
		return resp, err
	}

	if resp.Error {
		return resp, errors.New(resp.Message)
	}
	return resp, nil
}

// SendHTTPRequest sends a HTTP request
func (o *OXR) SendHTTPRequest(endpoint string, values url.Values, result interface{}) error {
	headers := make(map[string]string)
	headers["Authorization"] = "Token " + o.APIKey
	path := APIURL + endpoint + "?" + values.Encode()

	return o.Requester.SendPayload(http.MethodGet,
		path,
		headers,
		nil,
		result,
		false,
		false,
		o.Verbose,
		false,
		false)
}
