package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	CRYPTSY_API_URL      = "https://api.cryptsy.com/api/v2/"
	CRYPTSY_API_VERISON  = "2"
	CRYPTSY_MARKETS      = "markets"
	CRYPTSY_VOLUME       = "volume"
	CRYPTSY_TICKER       = "ticker"
	CRYPTSY_FEES         = "fees"
	CRYPSTY_TRIGGERS     = "triggers"
	CRYPTSY_CURRENCIES   = "currencies"
	CRYPTSY_ORDERBOOK    = "orderbook"
	CRYPTSY_TRADEHISTORY = "tradehistory"
	CRYPTSY_OHLC         = "ohlc"
	CRYPTSY_INFO         = "info"
	CRYPTSY_BALANCES     = "balances"
	CRYPTSY_DEPOSITS     = "deposits"
	CRYPTSY_ADDRESSES    = "addresses"
	CRYPTSY_ORDER        = "order"
	CRYPTSY_ORDERS       = "orders"
	CRYPSTY_TRIGGER      = "trigger"
)

type Cryptsy struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey, APISecret       string
	TakerFee, MakerFee      float64
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
	Market                  map[string]CryptsyMarket
	Ticker                  map[string]CryptsyTicker
	Volume                  map[string]CryptsyVolume
	Currencies              []CryptsyCurrency
}

type CryptsyMarket struct {
	DayStats struct {
		PriceHigh float64 `json:"price_high"`
		PriceLow  float64 `json:"price_low"`
		Volume    float64 `json:"volume"`
		VolumeBtc float64 `json:"volume_btc"`
	} `json:"24hr"`
	CoinCurrencyID string `json:"coin_currency_id"`
	ID             string `json:"id"`
	Label          string `json:"label"`
	LastTrade      struct {
		Date      string  `json:"date"`
		Price     float64 `json:"price"`
		Timestamp float64 `json:"timestamp"`
	} `json:"last_trade"`
	MaintenanceMode  string `json:"maintenance_mode"`
	MarketCurrencyID string `json:"market_currency_id"`
	VerifiedOnly     bool   `json:"verifiedonly"`
}

type CryptsyVolume struct {
	ID        string  `json:"id"`
	Volume    float64 `json:"volume"`
	VolumeBtc float64 `json:"volume_btc"`
}

type CryptsyTicker struct {
	ID  string  `json:"id"`
	Bid float64 `json:"bid"`
	Ask float64 `json:"ask"`
}

type CryptsyOrderbook struct {
	BuyOrders []struct {
		Price    float64 `json:"price"`
		Quantity float64 `json:"quantity"`
		Total    float64 `json:"total"`
	} `json:"buyorders"`
	Sellorder []struct {
		Price    float64 `json:"price"`
		Quantity float64 `json:"quantity"`
		Total    float64 `json:"total"`
	} `json:"sellorders"`
}

type CryptsyTradeHistory struct {
	Datetime          string  `json:"datetime"`
	InitiateOrderType string  `json:"initiate_ordertype"`
	Quantity          float64 `json:"quantitiy"`
	Timestamp         float64 `json:"timestamp"`
	Total             float64 `json:"total"`
	TradeID           float64 `json:"tradeid"`
	TradePrice        float64 `json:"tradeprice"`
}

type CryptsyOHLC struct {
	Close float64 `json:"close"`
	Date  string  `json:"date"`
	High  float64 `json:"high"`
}

type CryptsyInfo struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	AccountType string `json:"accounttype"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	TradeKey    string `json:"trade_key"`
}

type CryptsyDeposit struct {
	Currency  string  `json:"currency"`
	Timestamp float64 `json:"timestamp"`
	TRXID     string  `json:"txrid"`
}

type CryptsyCurrency struct {
	Code        string `json:"code"`
	ID          string `json:"id"`
	Maintenance string `json:"maintenance"`
	Name        string `json:"name"`
}

func (c *Cryptsy) SetDefaults() {
	c.Name = "Cryptsy"
	c.Enabled = true
	c.Verbose = false
	c.Websocket = false
	c.TakerFee = 0.33
	c.MakerFee = 0.33
	c.Verbose = false
	c.RESTPollingDelay = 10
	c.Market = make(map[string]CryptsyMarket)
	c.Ticker = make(map[string]CryptsyTicker)
	c.Volume = make(map[string]CryptsyVolume)
}

func (c *Cryptsy) GetName() string {
	return c.Name
}

func (c *Cryptsy) SetEnabled(enabled bool) {
	c.Enabled = enabled
}

func (c *Cryptsy) IsEnabled() bool {
	return c.Enabled
}

func (c *Cryptsy) GetFee(maker bool) float64 {
	if maker {
		return c.MakerFee
	} else {
		return c.TakerFee
	}
}

func (c *Cryptsy) Run() {
	if c.Verbose {
		log.Printf("%s Websocket: %s.", c.GetName(), IsEnabled(c.Websocket))
		log.Printf("%s polling delay: %ds.\n", c.GetName(), c.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.EnabledPairs), c.EnabledPairs)
	}

	if c.Websocket {
		go c.PusherClient()
	}

	err := c.GetMarkets()
	if err != nil {
		log.Println(err)
	} else {
		markets := []string{}
		for x, _ := range c.Market {
			markets = append(markets, x)
		}
		diff := StringSliceDifference(c.AvailablePairs, markets)
		if len(diff) > 0 {
			exch, err := GetExchangeConfig(c.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", c.Name, diff)
				exch.AvailablePairs = JoinStrings(markets, ",")
				UpdateExchangeConfig(exch)
			}
		}
	}

	for c.Enabled {
		err := c.GetMarkets()
		if err != nil {
			log.Println(err)
		} else {
			for _, x := range c.EnabledPairs {
				market := c.Market[x]
				if market.ID != "" {
					log.Printf("Cryptsy %s: Last %f High %f Low %f Volume %f\n", x, market.LastTrade.Price, market.DayStats.PriceHigh, market.DayStats.PriceLow, market.DayStats.Volume)
					AddExchangeInfo(c.GetName(), x[0:3], x[3:], market.LastTrade.Price, market.DayStats.Volume)
				}
			}
		}
		time.Sleep(time.Second * c.RESTPollingDelay)
	}
}

func (c *Cryptsy) SetAPIKeys(apiKey, apiSecret string) {
	c.APIKey = apiKey
	c.APISecret = apiSecret
}

func (c *Cryptsy) GetMarkets() error {
	type Response struct {
		Data    []CryptsyMarket `json:"data"`
		Success bool            `json:"success"`
	}

	response := Response{}
	err := SendHTTPGetRequest(CRYPTSY_API_URL+CRYPTSY_MARKETS, true, &response)

	if err != nil {
		return err
	}

	if !response.Success {
		return errors.New("Unable to retrieve Cryptsy market data.")
	}

	for _, x := range response.Data {
		label := strings.Replace(x.Label, "/", "", -1)
		x.Label = label
		c.Market[label] = x
	}
	return nil
}

func (c *Cryptsy) GetVolume(id string) error {
	type Response struct {
		Data    []CryptsyVolume `json:"data"`
		Success bool            `json:"success"`
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s", CRYPTSY_API_URL+CRYPTSY_MARKETS, CRYPTSY_VOLUME)
	err := SendHTTPGetRequest(path, true, &response)

	if err != nil {
		return err
	}

	if !response.Success {
		return errors.New("Unable to retrieve Cryptsy volume data.")
	}

	for _, x := range response.Data {
		c.Volume[x.ID] = x
	}
	return nil
}

func (c *Cryptsy) GetTickers() error {
	type Response struct {
		Data    []CryptsyTicker `json:"data"`
		Success bool            `json:"success"`
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s", CRYPTSY_API_URL+CRYPTSY_MARKETS, CRYPTSY_TICKER)
	err := SendHTTPGetRequest(path, true, &response)

	if err != nil {
		return err
	}

	if !response.Success {
		return errors.New("Unable to fetch market ticker data.")
	}

	for _, x := range response.Data {
		c.Ticker[x.ID] = x
	}
	return nil
}

func (c *Cryptsy) GetMarketFees(id string) {
	path := fmt.Sprintf("%s/%s/%s", CRYPTSY_API_URL+CRYPTSY_MARKETS, id, CRYPTSY_FEES)
	err := c.SendAuthenticatedHTTPRequest("GET", path, url.Values{})
	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) GetMarketTriggers(id string) {
	path := fmt.Sprintf("%s/%s/%s", CRYPTSY_API_URL+CRYPTSY_MARKETS, id, CRYPSTY_TRIGGERS)
	err := c.SendAuthenticatedHTTPRequest("GET", path, url.Values{})
	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) GetOrderbook(id string) {
	type Response struct {
		Data    CryptsyOrderbook `json:"data"`
		Success bool             `json:"success"`
	}
	response := Response{}
	path := fmt.Sprintf("%s/%s/%s", CRYPTSY_API_URL+CRYPTSY_MARKETS, id, CRYPTSY_ORDERBOOK)
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		log.Println(err)
	}
	log.Println(response)
}

func (c *Cryptsy) GetTradeHistory(id string) {
	type Response struct {
		Data    []CryptsyTradeHistory `json:"data"`
		Success bool                  `json:"success"`
	}
	response := Response{}
	path := fmt.Sprintf("%s/%s/%s", CRYPTSY_API_URL+CRYPTSY_MARKETS, id, CRYPTSY_TRADEHISTORY)
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		log.Println(err)
	}
	log.Println(response)
}

func (c *Cryptsy) GetOHLC(id string) {
	type Response struct {
		Data    []CryptsyOHLC `json:"data"`
		Success bool          `json:"success"`
	}
	response := Response{}
	path := fmt.Sprintf("%s/%s/%s", CRYPTSY_API_URL+CRYPTSY_MARKETS, id, CRYPTSY_OHLC)
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		log.Println(err)
	}
	log.Println(response)
}

func (c *Cryptsy) GetCurrencies() error {
	type Response struct {
		Data    []CryptsyCurrency `json:"data"`
		Success bool              `json:"success"`
	}

	response := Response{}
	err := SendHTTPGetRequest(CRYPTSY_API_URL+CRYPTSY_CURRENCIES, true, &response)
	if err != nil {
		return err
	}

	if !response.Success {
		return errors.New("Unable to get Cryptsy currency data.")
	}
	c.Currencies = response.Data
	return nil
}

func (c *Cryptsy) GetInfo() {
	err := c.SendAuthenticatedHTTPRequest("GET", CRYPTSY_API_URL+CRYPTSY_INFO, url.Values{})
	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) GetBalances(balanceType, id string) {
	req := url.Values{}

	if len(balanceType) > 0 {
		req.Set("type", balanceType)
	}
	err := c.SendAuthenticatedHTTPRequest("GET", CRYPTSY_API_URL+CRYPTSY_BALANCES, req)

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) GetDeposits(limit int, id string) {
	req := url.Values{}

	if limit > 0 {
		req.Set("liimt", strconv.Itoa(limit))
	}

	err := c.SendAuthenticatedHTTPRequest("GET", CRYPTSY_API_URL+CRYPTSY_DEPOSITS, req)

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) CreateOrder(marketid, orderType string, amount, price float64) {
	req := url.Values{}
	req.Set("marketid", marketid)
	req.Set("ordertype", orderType)
	req.Set("quantity", strconv.FormatFloat(amount, 'f', 8, 64))
	req.Set("price", strconv.FormatFloat(amount, 'f', 8, 64))

	err := c.SendAuthenticatedHTTPRequest("POST", CRYPTSY_API_URL+CRYPTSY_ORDER, req)

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) GetOrder(orderID int64) {
	path := fmt.Sprintf("%s/%s", CRYPTSY_API_URL+CRYPTSY_ORDER, strconv.FormatInt(orderID, 10))
	err := c.SendAuthenticatedHTTPRequest("GET", path, url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) DeleteOrder(orderID int64) {
	path := fmt.Sprintf("%s/%s", CRYPTSY_API_URL+CRYPTSY_ORDER, strconv.FormatInt(orderID, 10))
	err := c.SendAuthenticatedHTTPRequest("DELETE", path, url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) CreateTrigger(marketid int64, orderType string, quantity float64, comparison string, price, orderprice float64, expires int64) {
	req := url.Values{}
	req.Set("marketid", strconv.FormatInt(marketid, 10))
	req.Set("type", orderType)
	req.Set("quantity", strconv.FormatFloat(quantity, 'f', 8, 64))
	req.Set("comparison", comparison)
	req.Set("price", strconv.FormatFloat(price, 'f', 8, 64))
	req.Set("orderprice", strconv.FormatFloat(orderprice, 'f', 8, 64))

	if expires > 0 {
		req.Set("expires", strconv.FormatInt(expires, 10))
	}

	err := c.SendAuthenticatedHTTPRequest("POST", CRYPTSY_API_URL+CRYPSTY_TRIGGER, req)

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) GetTrigger(triggerID int64) {
	path := fmt.Sprintf("%s/%s", CRYPTSY_API_URL+CRYPSTY_TRIGGER, strconv.FormatInt(triggerID, 10))
	err := c.SendAuthenticatedHTTPRequest("GET", path, url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) DeleteTrigger(triggerID int64) {
	path := fmt.Sprintf("%s/%s", CRYPTSY_API_URL+CRYPSTY_TRIGGER, strconv.FormatInt(triggerID, 10))
	err := c.SendAuthenticatedHTTPRequest("DELETE", path, url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (c *Cryptsy) SendAuthenticatedHTTPRequest(method, path string, params url.Values) (err error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	params.Set("nonce", nonce)
	encoded := params.Encode()
	hmac := GetHMAC(HASH_SHA512, []byte(encoded), []byte(c.APISecret))
	readStr := ""

	if method == "GET" || method == "DELETE" {
		path += "?" + encoded
	} else if method == "POST" {
		readStr = encoded
	}

	if c.Verbose {
		log.Printf("Sending %s request to %s with params %s\n", method, path, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = c.APIKey
	headers["Sign"] = HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest(method, path, headers, strings.NewReader(readStr))

	if err != nil {
		return err
	}

	if c.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	return nil
}
