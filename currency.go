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
	currency := StringToUpper(from + to)
	for x, y := range CurrencyStore {
		if x == currency {
			return amount * y.Rate, nil
		}
	}
	return 0, ErrCurrencyNotFound
}

func FetchYahooCurrencyData(currencyPairs []string) error {
	values := url.Values{}
	values.Set("q", fmt.Sprintf("SELECT * from yahoo.finance.xchange WHERE pair in (\"%s\")", JoinStrings(currencyPairs, ",")))
	values.Set("format", "json")
	values.Set("env", YAHOO_DATABASE)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", YAHOO_YQL_URL, headers, strings.NewReader(values.Encode()))

	if err != nil {
		return err
	}

	yahooResp := YahooJSONResponse{}
	err = JSONDecode([]byte(resp), &yahooResp)

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
	currencyPairs := SplitStrings(MakecurrencyPairs(currencies), ",")
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
