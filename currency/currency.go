package currency

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

// Rate holds the current exchange rates for the currency pair.
type Rate struct {
	ID   string  `json:"id"`
	Name string  `json:"Name"`
	Rate float64 `json:",string"`
	Date string  `json:"Date"`
	Time string  `json:"Time"`
	Ask  float64 `json:",string"`
	Bid  float64 `json:",string"`
}

// YahooJSONResponseInfo is a sub type that holds JSON response info
type YahooJSONResponseInfo struct {
	Count   int       `json:"count"`
	Created time.Time `json:"created"`
	Lang    string    `json:"lang"`
}

// YahooJSONResponse holds Yahoo API responses
type YahooJSONResponse struct {
	Query struct {
		YahooJSONResponseInfo
		Results struct {
			Rate []Rate `json:"rate"`
		}
	}
}

// FixerResponse contains the data fields for the Fixer API response
type FixerResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

const (
	maxCurrencyPairsPerRequest = 350
	yahooYQLURL                = "https://query.yahooapis.com/v1/public/yql?"
	yahooDatabase              = "store://datatables.org/alltableswithkeys"
	fixerAPI                   = "http://api.fixer.io/latest"
	// DefaultCurrencies has the default minimum of FIAT values
	DefaultCurrencies = "USD,AUD,EUR,CNY"
	// DefaultCryptoCurrencies has the default minimum of crytpocurrency values
	DefaultCryptoCurrencies = "BTC,LTC,ETH,DOGE,DASH,XRP,XMR"
)

// Variables for package which includes base error strings & exportable
// queries
var (
	CurrencyStore             map[string]Rate
	CurrencyStoreFixer        map[string]float64
	BaseCurrencies            string
	CryptoCurrencies          string
	ErrCurrencyDataNotFetched = errors.New("yahoo currency data has not been fetched yet")
	ErrCurrencyNotFound       = errors.New("unable to find specified currency")
	ErrQueryingYahoo          = errors.New("unable to query Yahoo currency values")
	ErrQueryingYahooZeroCount = errors.New("yahoo returned zero currency data")
	YahooEnabled              = false
)

// IsDefaultCurrency checks if the currency passed in matches the default
// FIAT currency
func IsDefaultCurrency(currency string) bool {
	return common.StringContains(
		DefaultCurrencies, common.StringToUpper(currency),
	)
}

// IsDefaultCryptocurrency checks if the currency passed in matches the default
// CRYPTO currency
func IsDefaultCryptocurrency(currency string) bool {
	return common.StringContains(
		DefaultCryptoCurrencies, common.StringToUpper(currency),
	)
}

// IsFiatCurrency checks if the currency passed is an enabled FIAT currency
func IsFiatCurrency(currency string) bool {
	if BaseCurrencies == "" {
		log.Println("IsFiatCurrency: BaseCurrencies string variable not populated")
		return false
	}
	return common.StringContains(BaseCurrencies, common.StringToUpper(currency))
}

// IsCryptocurrency checks if the currency passed is an enabled CRYPTO currency.
func IsCryptocurrency(currency string) bool {
	if CryptoCurrencies == "" {
		log.Println(
			"IsCryptocurrency: CryptoCurrencies string variable not populated",
		)
		return false
	}
	return common.StringContains(CryptoCurrencies, common.StringToUpper(currency))
}

// ContainsSeparator checks to see if the string passed contains "-" or "_"
// separated strings and returns what the separators were.
func ContainsSeparator(input string) (bool, string) {
	separators := []string{"-", "_"}
	var separatorsContainer []string

	for _, x := range separators {
		if common.StringContains(input, x) {
			separatorsContainer = append(separatorsContainer, x)
		}
	}
	if len(separatorsContainer) == 0 {
		return false, ""
	}
	return true, strings.Join(separatorsContainer, ",")
}

// ContainsBaseCurrencyIndex checks the currency against the baseCurrencies and
// returns a bool and its corresponding basecurrency.
func ContainsBaseCurrencyIndex(baseCurrencies []string, currency string) (bool, string) {
	for _, x := range baseCurrencies {
		if common.StringContains(currency, x) {
			return true, x
		}
	}
	return false, ""
}

// ContainsBaseCurrency checks the currency against the baseCurrencies and
// returns a bool
func ContainsBaseCurrency(baseCurrencies []string, currency string) bool {
	for _, x := range baseCurrencies {
		if common.StringContains(currency, x) {
			return true
		}
	}
	return false
}

// CheckAndAddCurrency checks the string you passed with the input string array,
// if not already added, checks to see if it is part of the default currency
// list and returns the appended string.
func CheckAndAddCurrency(input []string, check string) []string {
	for _, x := range input {
		if IsDefaultCurrency(x) {
			if IsDefaultCurrency(check) {
				if check == x {
					return input
				}
				continue
			}
			return input
		} else if IsDefaultCryptocurrency(x) {
			if IsDefaultCryptocurrency(check) {
				if check == x {
					return input
				}
				continue
			}
			return input
		}
		return input
	}
	input = append(input, check)
	return input
}

// SeedCurrencyData takes the desired FIAT currency string, if not defined the
// function will assign it the default values. The function will query
// yahoo for the currency values and will seed currency data.
func SeedCurrencyData(fiatCurrencies string) error {
	if fiatCurrencies == "" {
		fiatCurrencies = DefaultCurrencies
	}

	if YahooEnabled {
		return QueryYahooCurrencyValues(fiatCurrencies)
	}

	return FetchFixerCurrencyData()
}

// MakecurrencyPairs takes all supported currency and turns them into pairs.
func MakecurrencyPairs(supportedCurrencies string) string {
	currencies := common.SplitStrings(supportedCurrencies, ",")
	pairs := []string{}
	count := len(currencies)
	for i := 0; i < count; i++ {
		currency := currencies[i]
		for j := 0; j < count; j++ {
			if currency != currencies[j] {
				pairs = append(pairs, currency+currencies[j])
			}
		}
	}
	return common.JoinStrings(pairs, ",")
}

// ConvertCurrency for example converts $1 USD to the equivalent Japanese Yen
// or vice versa.
func ConvertCurrency(amount float64, from, to string) (float64, error) {
	from = common.StringToUpper(from)
	to = common.StringToUpper(to)

	if from == to {
		return amount, nil
	}

	if YahooEnabled {
		currency := from + to
		_, ok := CurrencyStore[currency]
		if !ok {
			err := SeedCurrencyData(currency[:len(from)] + "," + currency[len(to):])
			if err != nil {
				return 0, err
			}
		}

		result, ok := CurrencyStore[currency]
		if !ok {
			return 0, ErrCurrencyNotFound
		}
		return amount * result.Rate, nil
	}

	_, ok := CurrencyStoreFixer[from]
	if !ok {
		err := FetchFixerCurrencyData()
		if err != nil {
			return 0, err
		}
	}

	var resultFrom float64
	var resultTo float64

	// First check if we're converting to USD, USD doesn't exist in the rates map
	if to == "USD" {
		resultFrom, ok = CurrencyStoreFixer[from]
		if !ok {
			return 0, ErrCurrencyNotFound
		}
		return amount / resultFrom, nil
	}

	// Check to see if we're converting from USD
	if from == "USD" {
		resultTo, ok = CurrencyStoreFixer[to]
		if !ok {
			return 0, ErrCurrencyNotFound
		}
		return resultTo * amount, nil
	}

	// Otherwise convert to USD, then to the target currency
	resultFrom, ok = CurrencyStoreFixer[from]
	if !ok {
		return 0, ErrCurrencyNotFound
	}

	converted := amount / resultFrom
	resultTo, ok = CurrencyStoreFixer[to]
	if !ok {
		return 0, ErrCurrencyNotFound
	}

	return converted * resultTo, nil
}

// FetchFixerCurrencyData seeds the variable C
func FetchFixerCurrencyData() error {
	var result FixerResponse
	values := url.Values{}
	values.Set("base", "USD")
	url := common.EncodeURLValues(fixerAPI, values)

	CurrencyStoreFixer = make(map[string]float64)

	err := common.SendHTTPGetRequest(url, true, &result)
	if err != nil {
		return err
	}

	CurrencyStoreFixer = result.Rates
	return nil
}

// FetchYahooCurrencyData seeds the variable CurrencyStore; this is a
// map[string]Rate
func FetchYahooCurrencyData(currencyPairs []string) error {
	values := url.Values{}
	values.Set(
		"q", fmt.Sprintf("SELECT * from yahoo.finance.xchange WHERE pair in (\"%s\")",
			common.JoinStrings(currencyPairs, ",")),
	)
	values.Set("format", "json")
	values.Set("env", yahooDatabase)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest(
		"POST", yahooYQLURL, headers, strings.NewReader(values.Encode()),
	)
	if err != nil {
		return err
	}

	log.Printf("Currency recv: %s", resp)

	yahooResp := YahooJSONResponse{}
	err = common.JSONDecode([]byte(resp), &yahooResp)
	if err != nil {
		return err
	}

	if yahooResp.Query.Count == 0 {
		return ErrQueryingYahooZeroCount
	}

	for i := 0; i < yahooResp.Query.YahooJSONResponseInfo.Count; i++ {
		CurrencyStore[yahooResp.Query.Results.Rate[i].ID] = yahooResp.Query.Results.Rate[i]
	}
	return nil
}

// QueryYahooCurrencyValues takes in desired currencies, creates pairs then
// uses FetchYahooCurrencyData to seed CurrencyStore
func QueryYahooCurrencyValues(currencies string) error {
	CurrencyStore = make(map[string]Rate)
	currencyPairs := common.SplitStrings(MakecurrencyPairs(currencies), ",")
	log.Printf(
		"%d fiat currency pairs generated. Fetching Yahoo currency data (this may take a minute)..\n",
		len(currencyPairs),
	)
	return FetchYahooCurrencyData(currencyPairs)
}
