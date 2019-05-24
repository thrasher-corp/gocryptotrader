// Currencylayer provides a simple REST API with real-time and historical
// exchange rates for 168 world currencies, delivering currency pairs in
// universally usable JSON format - compatible with any of your applications.
// Spot exchange rate data is retrieved from several major forex data providers
// in real-time, validated, processed and delivered hourly, every 10 minutes, or
// even within the 60-second market window.
// Providing the most representative forex market value available
// ("midpoint" value) for every API request, the currencylayer API powers
// currency converters, mobile applications, financial software components and
// back-office systems all around the world.

package currencylayer

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

	authRate   = 0
	unAuthRate = 0
)

// CurrencyLayer is a foreign exchange rate provider at
// https://currencylayer.com NOTE default base currency is USD when using a free
// account. Has automatic upgrade to a SSL connection.
type CurrencyLayer struct {
	base.Base
	Requester *request.Requester
}

// Setup sets appropriate values for CurrencyLayer
func (c *CurrencyLayer) Setup(config base.Settings) error {
	if config.APIKeyLvl < 0 || config.APIKeyLvl > 3 {
		log.Errorf("apikey incorrectly set in config.json for %s, please set appropriate account levels",
			config.Name)
		return errors.New("apikey set failure")
	}

	c.Name = config.Name
	c.APIKey = config.APIKey
	c.APIKeyLvl = config.APIKeyLvl
	c.Enabled = config.Enabled
	c.RESTPollingDelay = config.RESTPollingDelay
	c.Verbose = config.Verbose
	c.PrimaryProvider = config.PrimaryProvider
	c.Requester = request.New(c.Name,
		request.NewRateLimit(time.Second*10, authRate),
		request.NewRateLimit(time.Second*10, unAuthRate),
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut))

	return nil
}

// GetRates is a wrapper function to return rates for GoCryptoTrader
func (c *CurrencyLayer) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	return c.GetliveData(symbols, baseCurrency)
}

// GetSupportedCurrencies returns supported currencies
func (c *CurrencyLayer) GetSupportedCurrencies() ([]string, error) {
	var resp SupportedCurrencies

	if err := c.SendHTTPRequest(APIEndpointList, url.Values{}, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.Error.Info)
	}

	var currencies []string
	for key := range resp.Currencies {
		currencies = append(currencies, key)
	}

	return currencies, nil
}

// GetliveData returns live quotes for foreign exchange currencies
func (c *CurrencyLayer) GetliveData(currencies, source string) (map[string]float64, error) {
	var resp LiveRates
	v := url.Values{}
	v.Set("currencies", currencies)
	v.Set("source", source)

	err := c.SendHTTPRequest(APIEndpointLive, v, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.Error.Info)
	}

	return resp.Quotes, nil
}

// GetHistoricalData returns historical exchange rate data for every past day of
// the last 16 years.
func (c *CurrencyLayer) GetHistoricalData(date string, currencies []string, source string) (map[string]float64, error) {
	var resp HistoricalRates
	v := url.Values{}
	v.Set("currencies", common.JoinStrings(currencies, ","))
	v.Set("source", source)
	v.Set("date", date)

	err := c.SendHTTPRequest(APIEndpointHistorical, v, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.Error.Info)
	}

	return resp.Quotes, nil
}

// Convert converts one currency amount to another currency amount.
func (c *CurrencyLayer) Convert(from, to, date string, amount float64) (float64, error) {
	if c.APIKeyLvl >= AccountBasic {
		return 0, errors.New("insufficient API privileges, upgrade to basic to use this function")
	}

	var resp ConversionRate

	v := url.Values{}
	v.Set("from", from)
	v.Set("to", to)
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("date", date)

	err := c.SendHTTPRequest(APIEndpointConversion, v, &resp)
	if err != nil {
		return resp.Result, err
	}

	if !resp.Success {
		return resp.Result, errors.New(resp.Error.Info)
	}
	return resp.Result, nil
}

// QueryTimeFrame returns historical exchange rates for a time-period.
// (maximum range: 365 days)
func (c *CurrencyLayer) QueryTimeFrame(startDate, endDate, baseCurrency string, currencies []string) (map[string]interface{}, error) {
	if c.APIKeyLvl >= AccountPro {
		return nil, errors.New("insufficient API privileges, upgrade to basic to use this function")
	}

	var resp TimeFrame

	v := url.Values{}
	v.Set("start_date", startDate)
	v.Set("end_date", endDate)
	v.Set("base", baseCurrency)
	v.Set("currencies", common.JoinStrings(currencies, ","))

	err := c.SendHTTPRequest(APIEndpointTimeframe, v, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.Error.Info)
	}
	return resp.Quotes, nil
}

// QueryCurrencyChange returns the change (both margin and percentage) of one or
// more currencies, relative to a Source Currency, within a specific
// time-frame (optional).
func (c *CurrencyLayer) QueryCurrencyChange(startDate, endDate, baseCurrency string, currencies []string) (map[string]Changes, error) {
	if c.APIKeyLvl != AccountEnterprise {
		return nil, errors.New("insufficient API privileges, upgrade to basic to use this function")
	}
	var resp ChangeRate

	v := url.Values{}
	v.Set("start_date", startDate)
	v.Set("end_date", endDate)
	v.Set("base", baseCurrency)
	v.Set("currencies", common.JoinStrings(currencies, ","))

	err := c.SendHTTPRequest(APIEndpointChange, v, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.Error.Info)
	}
	return resp.Quotes, nil
}

// SendHTTPRequest sends a HTTP request, if account is not free it automatically
// upgrades request to SSL.
func (c *CurrencyLayer) SendHTTPRequest(endPoint string, values url.Values, result interface{}) error {
	var path string
	values.Set("access_key", c.APIKey)

	var auth bool
	if c.APIKeyLvl == AccountFree {
		path = fmt.Sprintf("%s%s%s", APIEndpointURL, endPoint, "?")
	} else {
		auth = true
		path = fmt.Sprintf("%s%s%s", APIEndpointURLSSL, endPoint, "?")
	}
	path += values.Encode()

	return c.Requester.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		&result,
		auth,
		false,
		c.Verbose,
		false,
		false)
}
