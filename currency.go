package main

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
)

var (
	CurrencyStore             map[string]Rate
	BaseCurrencies            string
	ErrCurrencyDataNotFetched = errors.New("Yahoo currency data has not been fetched yet.")
	ErrCurrencyNotFound       = errors.New("Unable to find specified currency.")
	ErrQueryingYahoo          = errors.New("Unable to query Yahoo currency values.")
	ErrQueryingYahooZeroCount = errors.New("Yahoo returned zero currency data.")
)

func IsDefaultCurrency(currency string) bool {
	return common.StringContains(DEFAULT_CURRENCIES, common.StringToUpper(currency))
}

func IsFiatCurrency(currency string) bool {
	return common.StringContains(BaseCurrencies, common.StringToUpper(currency))
}

func IsCryptocurrency(currency string) bool {
	return common.StringContains(bot.config.Cryptocurrencies, common.StringToUpper(currency))
}

func ContainsSeparator(input string) (bool, string) {
	separators := []string{"-", "_"}
	for _, x := range separators {
		if common.StringContains(input, x) {
			return true, x
		}
	}
	return false, ""
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
		if check == x {
			return input
		}
	}
	if IsCryptocurrency(check) && IsFiatCurrency(check) {
		return input
	}

	input = append(input, check)
	return input
}

func RetrieveConfigCurrencyPairs() error {
	cryptoCurrencies := common.SplitStrings(bot.config.Cryptocurrencies, ",")
	fiatCurrencies := common.SplitStrings(DEFAULT_CURRENCIES, ",")

	for _, exchange := range bot.config.Exchanges {
		if exchange.Enabled {
			baseCurrencies := common.SplitStrings(exchange.BaseCurrencies, ",")
			enabledCurrencies := common.SplitStrings(exchange.EnabledPairs, ",")

			for _, currencyPair := range enabledCurrencies {
				ok, separator := ContainsSeparator(currencyPair)
				if ok {
					pair := common.SplitStrings(currencyPair, separator)
					for _, x := range pair {
						ok, _ = ContainsBaseCurrencyIndex(baseCurrencies, x)
						if !ok {
							cryptoCurrencies = CheckAndAddCurrency(cryptoCurrencies, x)
						}
					}
				} else {
					ok, idx := ContainsBaseCurrencyIndex(baseCurrencies, currencyPair)
					if ok {
						currency := strings.Replace(currencyPair, idx, "", -1)

						if ContainsBaseCurrency(baseCurrencies, currency) {
							fiatCurrencies = CheckAndAddCurrency(fiatCurrencies, currency)
						} else {
							cryptoCurrencies = CheckAndAddCurrency(cryptoCurrencies, currency)
						}

						if ContainsBaseCurrency(baseCurrencies, idx) {
							fiatCurrencies = CheckAndAddCurrency(fiatCurrencies, idx)
						} else {
							cryptoCurrencies = CheckAndAddCurrency(cryptoCurrencies, idx)
						}
					}
				}
			}
		}
	}

	BaseCurrencies = common.JoinStrings(fiatCurrencies, ",")
	BaseCurrencies = strings.Replace(BaseCurrencies, "RUR", "RUB", -1)
	bot.config.Cryptocurrencies = common.JoinStrings(cryptoCurrencies, ",")

	err := QueryYahooCurrencyValues(BaseCurrencies)

	if err != nil {
		return ErrQueryingYahoo
	}

	log.Println("Fetched currency value data.")
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
	if common.StringContains(currency, "RUB") {
		currency = strings.Replace(currency, "RUB", "RUR", -1)
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
