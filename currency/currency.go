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

type Rate struct {
	Id   string  `json:"id"`
	Name string  `json:"Name"`
	Rate float64 `json:",string"`
	Date string  `json:"Date"`
	Time string  `json:"Time"`
	Ask  float64 `json:",string"`
	Bid  float64 `json:",string"`
}

type YahooJSONResponseInfo struct {
	Count   int       `json:"count"`
	Created time.Time `json:"created"`
	Lang    string    `json:"lang"`
}

type YahooJSONResponse struct {
	Query struct {
		YahooJSONResponseInfo
		Results struct {
			Rate []Rate `json:"rate"`
		}
	}
}

const (
	MAX_CURRENCY_PAIRS_PER_REQUEST = 350
	YAHOO_YQL_URL                  = "http://query.yahooapis.com/v1/public/yql"
	YAHOO_DATABASE                 = "store://datatables.org/alltableswithkeys"
	DEFAULT_CURRENCIES             = "USD,AUD,EUR,CNY"
	DEFAULT_CRYPTOCURRENCIES       = "BTC,LTC,ETH,DOGE,DASH,XRP,XMR"
)

var (
	CurrencyStore             map[string]Rate
	BaseCurrencies            string
	CryptoCurrencies          string
	ErrCurrencyDataNotFetched = errors.New("Yahoo currency data has not been fetched yet.")
	ErrCurrencyNotFound       = errors.New("Unable to find specified currency.")
	ErrQueryingYahoo          = errors.New("Unable to query Yahoo currency values.")
	ErrQueryingYahooZeroCount = errors.New("Yahoo returned zero currency data.")
)

func IsDefaultCurrency(currency string) bool {
	return common.StringContains(DEFAULT_CURRENCIES, common.StringToUpper(currency))
}

func IsDefaultCryptocurrency(currency string) bool {
	return common.StringContains(DEFAULT_CRYPTOCURRENCIES, common.StringToUpper(currency))
}

func IsFiatCurrency(currency string) bool {
	if BaseCurrencies == "" {
		log.Println("IsFiatCurrency: BaseCurrencies string variable not populated")
	}
	return common.StringContains(BaseCurrencies, common.StringToUpper(currency))
}

func IsCryptocurrency(currency string) bool {
	if CryptoCurrencies == "" {
		log.Println("IsCryptocurrency: CryptoCurrencies string variable not populated")
	}
	return common.StringContains(CryptoCurrencies, common.StringToUpper(currency))
}

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
	} else {
		return true, strings.Join(separatorsContainer, ",")
	}
}

func ContainsBaseCurrencyIndex(baseCurrencies []string, currency string) (bool, string) {
	for _, x := range baseCurrencies {
		if common.StringContains(currency, x) {
			return true, x
		}
	}
	return false, ""
}

func ContainsBaseCurrency(baseCurrencies []string, currency string) bool {
	for _, x := range baseCurrencies {
		if common.StringContains(currency, x) {
			return true
		}
	}
	return false
}

func CheckAndAddCurrency(input []string, check string) []string {
	for _, x := range input {
		if IsDefaultCurrency(x) {
			if IsDefaultCurrency(check) {
				if check == x {
					return input
				}
				continue
			} else {
				return input
			}
		} else if IsDefaultCryptocurrency(x) {
			if IsDefaultCryptocurrency(check) {
				if check == x {
					return input
				}
				continue
			} else {
				return input
			}
		} else {
			return input
		}
	}

	input = append(input, check)
	return input
}

func SeedCurrencyData(fiatCurrencies string) error {
	if fiatCurrencies == "" {
		fiatCurrencies = DEFAULT_CURRENCIES
	}

	err := QueryYahooCurrencyValues(fiatCurrencies)
	if err != nil {
		return ErrQueryingYahoo
	}

	return nil
}

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

func ConvertCurrency(amount float64, from, to string) (float64, error) {
	currency := common.StringToUpper(from + to)

	if CurrencyStore[currency].Name != currency {
		err := SeedCurrencyData(currency[:len(from)] + "," + currency[len(to):])
		if err != nil {
			return 0, err
		}
	}

	for x, y := range CurrencyStore {
		if x == currency {
			return amount * y.Rate, nil
		}
	}
	return 0, ErrCurrencyNotFound
}

func FetchYahooCurrencyData(currencyPairs []string) error {
	values := url.Values{}
	values.Set("q", fmt.Sprintf("SELECT * from yahoo.finance.xchange WHERE pair in (\"%s\")", common.JoinStrings(currencyPairs, ",")))
	values.Set("format", "json")
	values.Set("env", YAHOO_DATABASE)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", YAHOO_YQL_URL, headers, strings.NewReader(values.Encode()))

	if err != nil {
		return err
	}

	yahooResp := YahooJSONResponse{}
	err = common.JSONDecode([]byte(resp), &yahooResp)

	if err != nil {
		return err
	}

	if yahooResp.Query.Count == 0 {
		return ErrQueryingYahooZeroCount
	}

	for i := 0; i < yahooResp.Query.YahooJSONResponseInfo.Count; i++ {
		CurrencyStore[yahooResp.Query.Results.Rate[i].Id] = yahooResp.Query.Results.Rate[i]
	}

	return nil
}

func QueryYahooCurrencyValues(currencies string) error {
	CurrencyStore = make(map[string]Rate)
	currencyPairs := common.SplitStrings(MakecurrencyPairs(currencies), ",")
	log.Printf("%d fiat currency pairs generated. Fetching Yahoo currency data (this may take a minute)..\n", len(currencyPairs))
	var err error
	var pairs []string
	index := 0

	if len(currencyPairs) > MAX_CURRENCY_PAIRS_PER_REQUEST {
		for index < len(currencyPairs) {
			if len(currencyPairs)-index > MAX_CURRENCY_PAIRS_PER_REQUEST {
				pairs = currencyPairs[index : index+MAX_CURRENCY_PAIRS_PER_REQUEST]
				index += MAX_CURRENCY_PAIRS_PER_REQUEST
			} else {
				pairs = currencyPairs[index:len(currencyPairs)]
				index += (len(currencyPairs) - index)
			}
			err = FetchYahooCurrencyData(pairs)
			if err != nil {
				return err
			}
		}
	} else {
		pairs = currencyPairs[index:len(currencyPairs)]
		err = FetchYahooCurrencyData(pairs)

		if err != nil {
			return err
		}
	}
	return nil
}
