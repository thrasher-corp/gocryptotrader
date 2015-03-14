package main

import (
	"log"
	"fmt"
	"strconv"
	"net/url"
)

const (
	COINBASE_API_URL = "https://api.exchange.coinbase.com/"
	COINBASE_API_VERISON = "0"
	COINBASE_PRODUCTS = "products"
	COINBASE_ORDERBOOK = "book"
	COINBASE_TICKER = "ticker"
	COINBASE_TRADES = "trades"
	COINBASE_HISTORY = "candles"
	COINBASE_STATS = "stats"
	COINBASE_CURRENCIES = "currencies"
)

type Coinbase struct {
	Name string
	Enabled bool
	Verbose bool
	Password, APIKey, APISecret string
	TakerFee, MakerFee float64
}

type CoinbaseTicker struct {
	TradeID int64 `json:"trade_id"`
	Price float64 `json:"price,string"`
	Size float64 `json:"size,string"`
	Time string `json:"time"`
}

type CoinbaseProduct struct {
	ID string `json:"id"`
	BaseCurrency string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	BaseMinSize float64 `json:"base_min_size"`
	BaseMaxSize int64 `json:"base_max_size"`
	QuoteIncrement float64 `json:"quote_increment"`
	DisplayName string `json:"string"`
}

type CoinbaseOrderbook struct {
	Asks [][]interface{} `json:"ask"`
	Bids [][]interface{} `json:"bids"`
	Sequence int64 `json:"sequence"`
}

type CoinbaseTrade struct {
	TradeID int64 `json:"trade_id"`
	Price float64 `json:"price,string"`
	Size float64 `json:"size,string"`
	Time string `json:"time"`
	Side string `json:"side"`
}

type CoinbaseStats struct {
	Open float64 `json:"open,string"`
	High float64 `json:"high,string"`
	Low float64 `json:"low,string"`
	Volume float64 `json:"volume,string"`
}

type CoinbaseCurrency struct {
	ID string
	Name string
	MinSize float64 `json:"min_size,string"`
}

type CoinbaseHistory struct {
	Time int64
	Low float64
	High float64
	Open float64
	Close float64
	Volume float64
}

func (c *Coinbase) SetDefaults() {
	c.Name = "Coinbase"
	c.Enabled = true
	c.Verbose = false
	c.TakerFee = 0.25
	c.MakerFee = 0
}

func (c *Coinbase) GetName() (string) {
	return c.Name
}

func (c *Coinbase) SetEnabled(enabled bool) {
	c.Enabled = enabled
}

func (c *Coinbase) IsEnabled() (bool) {
	return c.Enabled
}

func (c *Coinbase) GetFee(maker bool) (float64) {
	if maker {
		return c.MakerFee
	} else {
		return c.TakerFee
	}
}

func (c *Coinbase) SetAPIKeys(password, apiKey, apiSecret string) {
	c.Password = password
	c.APIKey = apiKey
	c.APISecret = apiSecret
}

func (c *Coinbase) GetProducts() {
	products := []CoinbaseProduct{}
	err := SendHTTPGetRequest(COINBASE_API_URL + COINBASE_PRODUCTS, true, &products)

	if err != nil {
		log.Println(err)
	}

	log.Println(products)
}

func (c *Coinbase) GetOrderbook(symbol string, level int) {
	orderbook := CoinbaseOrderbook{}
	path := ""
	if level > 0 {
		levelStr := strconv.Itoa(level)
		path = fmt.Sprintf("%s/%s/%s?level=%s", COINBASE_API_URL + COINBASE_PRODUCTS, symbol, COINBASE_ORDERBOOK, levelStr)
	} else {
		path = fmt.Sprintf("%s/%s/%s", COINBASE_API_URL + COINBASE_PRODUCTS, symbol, COINBASE_ORDERBOOK)
	}
		
	err := SendHTTPGetRequest(path, true, &orderbook)

	if err != nil {
		log.Println(err)
	}
	log.Println(orderbook)
} 

func (c *Coinbase) GetTicker(symbol string) (CoinbaseTicker) {
	ticker := CoinbaseTicker{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL + COINBASE_PRODUCTS, symbol, COINBASE_TICKER)
	err := SendHTTPGetRequest(path, true, &ticker)

	if err != nil {
		log.Println(err)
		return CoinbaseTicker{}
	}
	return ticker
}

func (c *Coinbase) GetTrades(symbol string) {
	trades := []CoinbaseTrade{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL + COINBASE_PRODUCTS, symbol, COINBASE_TRADES)
	err := SendHTTPGetRequest(path, true, &trades)

	if err != nil {
		log.Println(err)
	}
	log.Println(trades)
}

func (c *Coinbase) GetHistoricRates(symbol string, start, end, granularity int64) {
	history := []CoinbaseHistory{}
	values := url.Values{}

	if start > 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end > 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if granularity > 0 {
		values.Set("granularity", strconv.FormatInt(granularity, 10))
	}

	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL + COINBASE_PRODUCTS, symbol, COINBASE_HISTORY)
	encoded := values.Encode()

	if (len(encoded) > 0) {
		path += encoded
	}

	err := SendHTTPGetRequest(path, true, &history)

	if err != nil {
		log.Println(err)
	}
	log.Println(history)
}

func (c *Coinbase) GetStats(symbol string) (CoinbaseStats) {
	stats := CoinbaseStats{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL + COINBASE_PRODUCTS, symbol, COINBASE_STATS)
	err := SendHTTPGetRequest(path, true, &stats)

	if err != nil {
		log.Println(err)
		return CoinbaseStats{}
	}
	return stats
}

func (c *Coinbase) GetCurrencies() {
	currencies := []CoinbaseCurrency{}
	err := SendHTTPGetRequest(COINBASE_API_URL + COINBASE_CURRENCIES, true, &currencies)

	if err != nil {
		log.Println(err)
	}
	log.Println(currencies)
}