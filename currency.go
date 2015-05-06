package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
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
	YAHOO_YQL_URL      = "http://query.yahooapis.com/v1/public/yql"
	YAHOO_DATABASE     = "store://datatables.org/alltableswithkeys"
	DEFAULT_CURRENCIES = "USD,AUD,EUR,CNY"
)

var (
	CurrencyStore             YahooJSONResponse
	ErrCurrencyDataNotFetched = errors.New("Yahoo currency data has not been fetched yet.")
	ErrCurrencyNotFound       = errors.New("Unable to find specified currency.")
	ErrQueryingYahoo          = errors.New("Unable to query Yahoo currency values.")
)

func RetrieveConfigCurrencyPairs(config Config) error {
	currencyPairs := ""
	for _, exchange := range config.Exchanges {
		if exchange.Enabled {
			var result []string
			if StringContains(exchange.BaseCurrencies, ",") {
				result = SplitStrings(exchange.BaseCurrencies, ",")
			} else {
				if StringContains(DEFAULT_CURRENCIES, exchange.BaseCurrencies) {
					result = SplitStrings(DEFAULT_CURRENCIES, ",")
				} else {
					result = SplitStrings(exchange.BaseCurrencies+","+DEFAULT_CURRENCIES, ",")
				}
			}
			for _, s := range result {
				if !StringContains(currencyPairs, s) {
					currencyPairs += s + ","
				}
			}
		}
	}
	currencyPairs = currencyPairs[0 : len(currencyPairs)-1]
	err := QueryYahooCurrencyValues(currencyPairs)

	if err != nil {
		return ErrQueryingYahoo
	}

	log.Println("Fetched currency value data.")
	return nil
}

func MakecurrencyPairs(supportedCurrencies string) string {
	currencies := SplitStrings(supportedCurrencies, ",")
	pairs := ""
	count := len(currencies)
	for i := 0; i < count; i++ {
		currency := currencies[i]
		for j := 0; j < count; j++ {
			if currency != currencies[j] {
				pairs += currency + currencies[j] + ","
			}
		}
	}
	return pairs[0 : len(pairs)-1]
}

func ConvertCurrency(amount float64, from, to string) (float64, error) {
	if CurrencyStore.Query.YahooJSONResponseInfo.Count == 0 {
		return 0, ErrCurrencyDataNotFetched
	}

	currency := strings.ToUpper(from + to)
	for i := 0; i < CurrencyStore.Query.YahooJSONResponseInfo.Count; i++ {
		if CurrencyStore.Query.Results.Rate[i].Id == currency {
			return amount * CurrencyStore.Query.Results.Rate[i].Rate, nil
		}
	}
	return 0, ErrCurrencyNotFound
}

func QueryYahooCurrencyValues(currencies string) error {
	currencyPairs := MakecurrencyPairs(currencies)
	log.Printf("Supported currency pairs: %s\n", currencyPairs)

	values := url.Values{}
	values.Set("q", fmt.Sprintf("SELECT * from yahoo.finance.xchange WHERE pair in (\"%s\")", currencyPairs))
	values.Set("format", "json")
	values.Set("env", YAHOO_DATABASE)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", YAHOO_YQL_URL, headers, strings.NewReader(values.Encode()))

	if err != nil {
		return err
	}

	err = JSONDecode([]byte(resp), &CurrencyStore)

	if err != nil {
		return err
	}

	return nil
}
