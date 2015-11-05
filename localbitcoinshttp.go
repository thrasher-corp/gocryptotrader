package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"
)

const (
	LOCALBITCOINS_API_URL           = "https://localbitcoins.com/"
	LOCALBITCOINS_API_TICKER        = "bitcoinaverage/ticker-all-currencies/"
	LOCALBITCOINS_API_BITCOINCHARTS = "bitcoincharts/"
)

type LocalBitcoins struct {
	Name                        string
	Enabled                     bool
	Verbose                     bool
	Websocket                   bool
	RESTPollingDelay            time.Duration
	AuthenticatedAPISupport     bool
	Password, APIKey, APISecret string
	TakerFee, MakerFee          float64
	BaseCurrencies              []string
	AvailablePairs              []string
	EnabledPairs                []string
}

func (l *LocalBitcoins) SetDefaults() {
	l.Name = "LocalBitcoins"
	l.Enabled = true
	l.Verbose = false
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
}

func (l *LocalBitcoins) GetName() string {
	return l.Name
}

func (l *LocalBitcoins) SetEnabled(enabled bool) {
	l.Enabled = enabled
}

func (l *LocalBitcoins) IsEnabled() bool {
	return l.Enabled
}

func (l *LocalBitcoins) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	} else {
		return l.TakerFee
	}
}

func (l *LocalBitcoins) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	for l.Enabled {
		ticker, err := l.GetTicker()

		if err != nil {
			log.Println(err)
			goto sleep
		}
		for _, x := range l.EnabledPairs {
			currency := x[3:]
			log.Printf("LocalBitcoins BTC %s: Last %f Average 1h %f Average 24h %f Volume %f\n", currency, ticker[currency].Rates.Last,
				ticker[currency].Avg1h, ticker[currency].Avg24h, ticker[currency].VolumeBTC)
			AddExchangeInfo(l.GetName(), x[0:3], x[3:], ticker[currency].Rates.Last, ticker[currency].VolumeBTC)
		}
	sleep:
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

func (l *LocalBitcoins) SetAPIKeys(apiKey, apiSecret string) {
	if !l.AuthenticatedAPISupport {
		return
	}

	l.APIKey = apiKey
	l.APISecret = apiSecret
}

type LocalBitcoinsTicker struct {
	Avg12h float64 `json:"avg_12h"`
	Avg1h  float64 `json:"avg_1h"`
	Avg24h float64 `json:"avg_24h"`
	Rates  struct {
		Last float64 `json:"last,string"`
	} `json:"rates"`
	VolumeBTC float64 `json:"volume_btc,string"`
}

func (l *LocalBitcoins) GetTicker() (map[string]LocalBitcoinsTicker, error) {
	result := make(map[string]LocalBitcoinsTicker)
	err := SendHTTPGetRequest(LOCALBITCOINS_API_URL+LOCALBITCOINS_API_TICKER, true, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type LocalBitcoinsTrade struct {
	TID    int64   `json:"tid"`
	Date   int64   `json:"date"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
}

func (l *LocalBitcoins) GetTrades(currency string, values url.Values) ([]LocalBitcoinsTrade, error) {
	path := EncodeURLValues(fmt.Sprintf("%s/%s/trades.json", LOCALBITCOINS_API_URL+LOCALBITCOINS_API_BITCOINCHARTS, currency), values)
	result := []LocalBitcoinsTrade{}
	err := SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type LocalBitcoinsOrderbook struct {
	Bids []struct {
		Price  float64
		Amount float64
	}
	Asks []struct {
		Price  float64
		Amount float64
	}
}

func (l *LocalBitcoins) GetOrderbook(currency string) (LocalBitcoinsOrderbook, error) {
	path := fmt.Sprintf("%s/%s/orderbook.json", LOCALBITCOINS_API_URL+LOCALBITCOINS_API_BITCOINCHARTS, currency)

	type response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}

	result := response{}
	err := SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return LocalBitcoinsOrderbook{}, err
	}

	return LocalBitcoinsOrderbook{}, nil
}

func (l *LocalBitcoins) SendAuthenticatedHTTPRequest(method, path string, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	payload := values.Encode()
	message := nonce + l.APIKey + path + payload
	hmac := GetHMAC(HASH_SHA256, []byte(message), []byte(l.APISecret))
	headers := make(map[string]string)
	headers["Apiauth-Key"] = l.APIKey
	headers["Apiauth-Nonce"] = nonce
	headers["Apiauth-Signature"] = StringToUpper(HexEncodeToString(hmac))
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest(method, LOCALBITCOINS_API_URL+"api/"+path, headers, bytes.NewBuffer([]byte(payload)))

	if l.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
