// Package currencyconverter package
// https://free.currencyconverterapi.com/
package currencyconverter

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Setup sets appropriate values for CurrencyLayer
func (c *CurrencyConverter) Setup(config base.Settings) error {
	c.Name = config.Name
	c.APIKey = config.APIKey
	c.APIKeyLvl = config.APIKeyLvl
	c.Enabled = config.Enabled
	c.RESTPollingDelay = config.RESTPollingDelay
	c.Verbose = config.Verbose
	c.PrimaryProvider = config.PrimaryProvider
	c.Requester = request.New(c.Name,
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut),
		request.WithLimiter(request.NewBasicRateLimit(rateInterval, requestRate)))
	return nil
}

// GetRates is a wrapper function to return rates
func (c *CurrencyConverter) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	splitSymbols := strings.Split(symbols, ",")

	if len(splitSymbols) == 1 {
		return c.Convert(baseCurrency, symbols)
	}

	var completedStrings []string
	for x := range splitSymbols {
		completedStrings = append(completedStrings, baseCurrency+"_"+splitSymbols[x])
	}

	if (c.APIKey != "" && c.APIKey != "Key") || len(completedStrings) == 2 {
		return c.ConvertMany(completedStrings)
	}

	rates := make(map[string]float64)
	processBatch := func(length int) {
		for i := 0; i < length; i += 2 {
			batch := completedStrings[i : i+2]
			result, err := c.ConvertMany(batch)
			if err != nil {
				log.Errorf(log.Global, "Failed to get batch err: %s\n", err)
				continue
			}
			for k, v := range result {
				rates[strings.Replace(k, "_", "", -1)] = v
			}
		}
	}

	currLen := len(completedStrings)
	mod := currLen % 2
	if mod == 0 {
		processBatch(currLen)
		return rates, nil
	}

	processBatch(currLen - 1)
	result, err := c.ConvertMany(completedStrings[currLen-1:])
	if err != nil {
		return nil, err
	}

	for k, v := range result {
		rates[strings.Replace(k, "_", "", -1)] = v
	}

	return rates, nil
}

// ConvertMany takes 2 or more currencies depending on if using the free
// or paid API
func (c *CurrencyConverter) ConvertMany(currencies []string) (map[string]float64, error) {
	if len(currencies) > 2 && (c.APIKey == "" || c.APIKey == defaultAPIKey) {
		return nil, errors.New("currency fetching is limited to two currencies per request")
	}

	result := make(map[string]float64)
	v := url.Values{}
	joined := strings.Join(currencies, ",")
	v.Set("q", joined)
	v.Set("compact", "ultra")

	err := c.SendHTTPRequest(APIEndpointConvert, v, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Convert gets the conversion rate for the supplied currencies
func (c *CurrencyConverter) Convert(from, to string) (map[string]float64, error) {
	result := make(map[string]float64)
	v := url.Values{}
	v.Set("q", from+"_"+to)
	v.Set("compact", "ultra")

	err := c.SendHTTPRequest(APIEndpointConvert, v, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetSupportedCurrencies returns a list of the supported currencies
func (c *CurrencyConverter) GetSupportedCurrencies() ([]string, error) {
	var result Currencies

	err := c.SendHTTPRequest(APIEndpointCurrencies, url.Values{}, &result)
	if err != nil {
		return nil, err
	}

	var currencies []string
	for key := range result.Results {
		currencies = append(currencies, key)
	}

	return currencies, nil
}

// GetCountries returns a list of the supported countries and
// their symbols
func (c *CurrencyConverter) GetCountries() (map[string]CountryItem, error) {
	var result Countries

	err := c.SendHTTPRequest(APIEndpointCountries, url.Values{}, &result)
	if err != nil {
		return nil, err
	}

	return result.Results, nil
}

// SendHTTPRequest sends a HTTP request, if account is not free it automatically
// upgrades request to SSL.
func (c *CurrencyConverter) SendHTTPRequest(endPoint string, values url.Values, result interface{}) error {
	var path string
	var auth bool
	if c.APIKey == "" || c.APIKey == defaultAPIKey {
		path = fmt.Sprintf("%s%s/%s?", APIEndpointFreeURL, APIEndpointVersion, endPoint)
		auth = true
	} else {
		path = fmt.Sprintf("%s%s%s?", APIEndpointURL, APIEndpointVersion, endPoint)
		values.Set("apiKey", c.APIKey)
	}
	path += values.Encode()

	err := c.Requester.SendPayload(context.Background(), &request.Item{
		Method:      path,
		Result:      result,
		AuthRequest: auth,
		Verbose:     c.Verbose})

	if err != nil {
		return fmt.Errorf("currency converter API SendHTTPRequest error %s with path %s",
			err,
			path)
	}
	return nil
}
