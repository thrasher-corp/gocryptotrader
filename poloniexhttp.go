package main

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"
)

const (
	POLONIEX_API_URL           = "https://poloniex.com"
	POLONIEX_WEBSOCKET_ADDRESS = "wss://api.poloniex.com"
	POLONIEX_API_VERSION       = "1"
)

type Poloniex struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	AccessKey, SecretKey    string
	Fee                     float64
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
}

type PoloniexTicker struct {
	Last          float64 `json:"last,string"`
	LowestAsk     float64 `json:"lowestAsk,string"`
	HighestBid    float64 `json:"highestBid,string"`
	PercentChange float64 `json:"percentChange,string"`
	BaseVolume    float64 `json:"baseVolume,string"`
	QuoteVolume   float64 `json:"quoteVolume,string"`
	IsFrozen      int     `json:"isFrozen,string"`
	High24Hr      float64 `json:"high24hr,string"`
	Low24Hr       float64 `json:"low24hr,string"`
}

func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = true
	p.Fee = 0
	p.Verbose = false
	p.Websocket = false
	p.RESTPollingDelay = 10
}

func (p *Poloniex) GetName() string {
	return p.Name
}

func (p *Poloniex) SetEnabled(enabled bool) {
	p.Enabled = enabled
}

func (p *Poloniex) IsEnabled() bool {
	return p.Enabled
}

func (p *Poloniex) SetAPIKeys(apiKey, apiSecret string) {
	p.AccessKey = apiKey
	p.SecretKey = apiSecret
}

func (p *Poloniex) GetFee() float64 {
	return p.Fee
}

func (p *Poloniex) Run() {
	if p.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", p.GetName(), IsEnabled(p.Websocket), POLONIEX_WEBSOCKET_ADDRESS)
		log.Printf("%s polling delay: %ds.\n", p.GetName(), p.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", p.GetName(), len(p.EnabledPairs), p.EnabledPairs)
	}

	if p.Websocket {
		//go p.WebsocketClient()
	}

	for p.Enabled {
		for _, x := range p.EnabledPairs {
			currency := x
			go func() {
				ticker, err := p.GetTicker()
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Poloniex %s Last %f High %f Low %f Volume %f\n", currency, ticker[currency].Last, ticker[currency].High24Hr, ticker[currency].Low24Hr, ticker[currency].QuoteVolume)
				//AddExchangeInfo(p.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * p.RESTPollingDelay)
	}
}

func (p *Poloniex) GetTicker() (map[string]PoloniexTicker, error) {
	type response struct {
		Data map[string]PoloniexTicker
	}

	resp := response{}
	path := fmt.Sprintf("%s/public?command=returnTicker", POLONIEX_API_URL)
	err := SendHTTPGetRequest(path, true, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

func (p *Poloniex) GetVolume() (interface{}, error) {
	var resp interface{}
	path := fmt.Sprintf("%s/public?command=return24hVolume", POLONIEX_API_URL)
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return resp, err
	}
	return resp, nil
}

type PoloniexOrderbook struct {
	Asks     [][]interface{} `json:"asks"`
	Bids     [][]interface{} `json:"bids"`
	IsFrozen string          `json:"isFrozen"`
}

//TO-DO: add support for individual pair depth fetching
func (p *Poloniex) GetOrderbook(currencyPair string, depth int) (map[string]PoloniexOrderbook, error) {
	type Response struct {
		Data map[string]PoloniexOrderbook
	}

	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if depth != 0 {
		vals.Set("depth", strconv.Itoa(depth))
	}

	resp := Response{}
	path := fmt.Sprintf("%s/public?command=returnOrderBook&%s", POLONIEX_API_URL, vals.Encode())
	err := SendHTTPGetRequest(path, true, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

type PoloniexTradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID"`
	Date          string  `json:"date"`
	Type          string  `json:"type"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
}

func (p *Poloniex) GetTradeHistory(currencyPair, start, end string) ([]PoloniexTradeHistory, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	resp := []PoloniexTradeHistory{}
	path := fmt.Sprintf("%s/public?command=returnTradeHistory&%s", POLONIEX_API_URL, vals.Encode())
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PoloniexChartData struct {
	Date            int     `json:"date"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Open            float64 `json:"open"`
	Close           float64 `json:"close"`
	Volume          float64 `json:"volume"`
	QuoteVolume     float64 `json:"quoteVolume"`
	WeightedAverage float64 `json:"weightedAverage"`
}

func (p *Poloniex) GetChartData(currencyPair, start, end, period string) ([]PoloniexChartData, error) {
	vals := url.Values{}
	vals.Set("currencyPair", currencyPair)

	if start != "" {
		vals.Set("start", start)
	}

	if end != "" {
		vals.Set("end", end)
	}

	if period != "" {
		vals.Set("period", period)
	}

	resp := []PoloniexChartData{}
	path := fmt.Sprintf("%s/public?command=returnChartData&%s", POLONIEX_API_URL, vals.Encode())
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

type PoloniexCurrencies struct {
	Name               string      `json:"name"`
	MaxDailyWithdrawal string      `json:"maxDailyWithdrawal"`
	TxFee              float64     `json:"txFee,string"`
	MinConfirmations   int         `json:"minConf"`
	DepositAddresses   interface{} `json:"depositAddress"`
	Disabled           int         `json:"disabled"`
	Delisted           int         `json:"delisted"`
	Frozen             int         `json:"frozen"`
}

func (p *Poloniex) GetCurrencies() (map[string]PoloniexCurrencies, error) {
	type Response struct {
		Data map[string]PoloniexCurrencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/public?command=returnCurrencies", POLONIEX_API_URL)
	err := SendHTTPGetRequest(path, true, &resp.Data)

	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

type PoloniexLoanOrder struct {
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	RangeMin int     `json:"rangeMin"`
	RangeMax int     `json:"rangeMax"`
}

type PoloniexLoanOrders struct {
	Offers  []PoloniexLoanOrder `json:"offers"`
	Demands []PoloniexLoanOrder `json:"demands"`
}

func (p *Poloniex) GetLoanOrders(currency string) (PoloniexLoanOrders, error) {
	resp := PoloniexLoanOrders{}
	path := fmt.Sprintf("%s/public?command=returnLoanOrders&currency=%s", POLONIEX_API_URL, currency)
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return resp, err
	}
	return resp, nil
}
