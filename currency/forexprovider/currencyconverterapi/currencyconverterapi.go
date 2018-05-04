package currencyconverter

import (
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
)

// const declarations consist of endpoints
const (
	APIEndpointURL     = "https://currencyconverterapi.com/api/"
	APIEndpointFreeURL = "https://free.currencyconverterapi.com/api/"
	APIEndpointVersion = "v5"

	APIEndpointConvert    = "convert"
	APIEndpointCurrencies = "currencies"
	APIEndpointCountries  = "countries"
	APIEndpointUsage      = "usage"
)

// CurrencyConverter stores the struct for the CurrencyConverter API
type CurrencyConverter struct {
	base.Base
}

// Setup sets appropriate values for CurrencyLayer
func (c *CurrencyConverter) Setup(config base.Settings) {
	c.Name = config.Name
	c.APIKey = config.APIKey
	c.APIKeyLvl = config.APIKeyLvl
	c.Enabled = config.Enabled
	c.RESTPollingDelay = config.RESTPollingDelay
	c.Verbose = config.Verbose
	c.PrimaryProvider = config.PrimaryProvider
}

// GetRates is a wrapper function to return rates
func (c *CurrencyConverter) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	splitSymbols := common.SplitStrings(symbols, ",")

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
				log.Printf("Failed to get batch err: %s", err)
				continue
			}
			for k, v := range result {
				rates[common.ReplaceString(k, "_", "", -1)] = v
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
		rates[common.ReplaceString(k, "_", "", -1)] = v
	}

	return rates, nil
}

// ConvertMany takes 2 or more currencies depending on if using the free
// or paid API
func (c *CurrencyConverter) ConvertMany(currencies []string) (map[string]float64, error) {
	if len(currencies) > 2 && (c.APIKey == "" || c.APIKey == "Key") {
		return nil, errors.New("currency fetching is limited to two currencies per request")
	}

	result := make(map[string]float64)
	v := url.Values{}
	joined := common.JoinStrings(currencies, ",")
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

// GetCurrencies returns a list of the supported currencies
func (c *CurrencyConverter) GetCurrencies() (map[string]CurrencyItem, error) {
	var result Currencies

	err := c.SendHTTPRequest(APIEndpointCurrencies, url.Values{}, &result)
	if err != nil {
		return nil, err
	}

	return result.Results, nil
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

	if c.APIKey == "" || c.APIKey == "Key" {
		path = fmt.Sprintf("%s%s/%s?", APIEndpointFreeURL, APIEndpointVersion, endPoint)
	} else {
		path = fmt.Sprintf("%s%s%s?", APIEndpointURL, APIEndpointVersion, endPoint)
		values.Set("apiKey", c.APIKey)
	}
	path = path + values.Encode()

	return common.SendHTTPGetRequest(path, true, c.Verbose, &result)
}
