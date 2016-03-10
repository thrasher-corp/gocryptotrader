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
	BaseCurrencies            string
	ErrCurrencyDataNotFetched = errors.New("Yahoo currency data has not been fetched yet.")
	ErrCurrencyNotFound       = errors.New("Unable to find specified currency.")
	ErrQueryingYahoo          = errors.New("Unable to query Yahoo currency values.")
)

func IsDefaultCurrency(currency string) bool {
	return StringContains(DEFAULT_CURRENCIES, StringToUpper(currency))
}

func IsFiatCurrency(currency string) bool {
	return StringContains(BaseCurrencies, StringToUpper(currency))
}

func IsCryptocurrency(currency string) bool {
	return StringContains(bot.config.Cryptocurrencies, StringToUpper(currency))
}

func RetrieveConfigCurrencyPairs(config Config) error {
	var fiatCurrencies, cryptoCurrencies []string
	for _, exchange := range config.Exchanges {
		if exchange.Enabled {
			enabledPairs := SplitStrings(exchange.EnabledPairs, ",")
			baseCurrencies := SplitStrings(exchange.BaseCurrencies, ",")

			for _, x := range baseCurrencies {
				for _, y := range enabledPairs {
					if StringContains(y, "_") {
						pairs := SplitStrings(y, "_")
						for _, z := range pairs {
							if z != x && !IsCryptocurrency(z) {
								cryptoCurrencies = append(cryptoCurrencies, z)
							} else if z == x && !IsDefaultCurrency(x) {
								fiatCurrencies = append(fiatCurrencies, x)
							}
						}
					} else {
						if StringContains(y, x) {
							if !IsDefaultCurrency(x) {
								fiatCurrencies = append(fiatCurrencies, x)
							}
							currency := TrimString(y, x)
							if !IsCryptocurrency(currency) {
								cryptoCurrencies = append(cryptoCurrencies, currency)
							}
						} else {
							if !IsCryptocurrency(y[0:3]) {
								cryptoCurrencies = append(cryptoCurrencies, y[0:3])
							}
							if !IsCryptocurrency(y[3:]) {
								cryptoCurrencies = append(cryptoCurrencies, y[3:])
							}
						}
					}
				}
			}
		}
	}
	bot.config.Cryptocurrencies = JoinStrings(StringSliceDifference(SplitStrings(bot.config.Cryptocurrencies, ","), cryptoCurrencies), ",")
	BaseCurrencies = JoinStrings(StringSliceDifference(SplitStrings(DEFAULT_CURRENCIES, ","), fiatCurrencies), ",")

	err := QueryYahooCurrencyValues(BaseCurrencies)

	if err != nil {
		return ErrQueryingYahoo
	}

	log.Println("Fetched currency value data.")
	return nil
}

func MakecurrencyPairs(supportedCurrencies string) string {
	currencies := SplitStrings(supportedCurrencies, ",")
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
	return JoinStrings(pairs, ",")
}

func ConvertCurrency(amount float64, from, to string) (float64, error) {
	if CurrencyStore.Query.YahooJSONResponseInfo.Count == 0 {
		return 0, ErrCurrencyDataNotFetched
	}

	currency := StringToUpper(from + to)
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
