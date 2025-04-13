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
	c.Verbose = config.Verbose
	c.PrimaryProvider = config.PrimaryProvider
	var err error
	c.Requester, err = request.New(c.Name,
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut),
		request.WithLimiter(request.NewBasicRateLimit(rateInterval, requestRate, 1)))
	return err
}

// GetRates is a wrapper function to return rates
func (c *CurrencyConverter) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	splitSymbols := strings.Split(symbols, ",")

	if len(splitSymbols) == 1 {
		return c.Convert(baseCurrency, symbols)
	}

	completedStrings := make([]string, len(splitSymbols))
	for x := range splitSymbols {
		completedStrings[x] = baseCurrency + "_" + splitSymbols[x]
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
				rates[strings.ReplaceAll(k, "_", "")] = v
			}
		}
	}

	currLen := len(completedStrings)
	if mod := currLen % 2; mod == 0 {
		processBatch(currLen)
		return rates, nil
	}

	processBatch(currLen - 1)
	result, err := c.ConvertMany(completedStrings[currLen-1:])
	if err != nil {
		return nil, err
	}

	for k, v := range result {
		rates[strings.ReplaceAll(k, "_", "")] = v
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

	currencies := make([]string, 0, len(result.Results))
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
func (c *CurrencyConverter) SendHTTPRequest(endPoint string, values url.Values, result any) error {
	var path string
	var auth request.AuthType
	if c.APIKey == "" || c.APIKey == defaultAPIKey {
		path = fmt.Sprintf("%s%s/%s?", APIEndpointFreeURL, APIEndpointVersion, endPoint)
		auth = request.AuthenticatedRequest
	} else {
		path = fmt.Sprintf("%s%s%s?", APIEndpointURL, APIEndpointVersion, endPoint)
		values.Set("apiKey", c.APIKey)
		auth = request.UnauthenticatedRequest
	}

	path += values.Encode()
	item := &request.Item{
		Method:  path,
		Result:  result,
		Verbose: c.Verbose,
	}
	err := c.Requester.SendPayload(context.TODO(), request.Unset, func() (*request.Item, error) {
		return item, nil
	}, auth)
	if err != nil {
		return fmt.Errorf("currency converter API SendHTTPRequest error %s with path %s",
			err,
			path)
	}
	return nil
}
