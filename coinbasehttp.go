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
	COINBASE_API_URL     = "https://api.exchange.coinbase.com/"
	COINBASE_API_VERISON = "0"
	COINBASE_PRODUCTS    = "products"
	COINBASE_ORDERBOOK   = "book"
	COINBASE_TICKER      = "ticker"
	COINBASE_TRADES      = "trades"
	COINBASE_HISTORY     = "candles"
	COINBASE_STATS       = "stats"
	COINBASE_CURRENCIES  = "currencies"
	COINBASE_ACCOUNTS    = "accounts"
	COINBASE_LEDGER      = "ledger"
	COINBASE_HOLDS       = "holds"
	COINBASE_ORDERS      = "orders"
	COINBASE_FILLS       = "fills"
	COINBASE_TRANSFERS   = "transfers"
	COINBASE_REPORTS     = "reports"
)

type Coinbase struct {
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

type CoinbaseTicker struct {
	TradeID int64   `json:"trade_id"`
	Price   float64 `json:"price,string"`
	Size    float64 `json:"size,string"`
	Time    string  `json:"time"`
}

type CoinbaseProduct struct {
	ID             string  `json:"id"`
	BaseCurrency   string  `json:"base_currency"`
	QuoteCurrency  string  `json:"quote_currency"`
	BaseMinSize    float64 `json:"base_min_size"`
	BaseMaxSize    int64   `json:"base_max_size"`
	QuoteIncrement float64 `json:"quote_increment"`
	DisplayName    string  `json:"string"`
}

type CoinbaseOrderL1L2 struct {
	Price     float64
	Amount    float64
	NumOrders float64
}

type CoinbaseOrderL3 struct {
	Price   float64
	Amount  float64
	OrderID string
}

type CoinbaseOrderbookL1L2 struct {
	Sequence int64                 `json:"sequence"`
	Bids     [][]CoinbaseOrderL1L2 `json:"asks"`
	Asks     [][]CoinbaseOrderL1L2 `json:"asks"`
}

type CoinbaseOrderbookL3 struct {
	Sequence int64               `json:"sequence"`
	Bids     [][]CoinbaseOrderL3 `json:"asks"`
	Asks     [][]CoinbaseOrderL3 `json:"asks"`
}

type CoinbaseOrderbookResponse struct {
	Sequence int64           `json:"sequence"`
	Bids     [][]interface{} `json:"bids"`
	Asks     [][]interface{} `json:"asks"`
}

type CoinbaseTrade struct {
	TradeID int64   `json:"trade_id"`
	Price   float64 `json:"price,string"`
	Size    float64 `json:"size,string"`
	Time    string  `json:"time"`
	Side    string  `json:"side"`
}

type CoinbaseStats struct {
	Open   float64 `json:"open,string"`
	High   float64 `json:"high,string"`
	Low    float64 `json:"low,string"`
	Volume float64 `json:"volume,string"`
}

type CoinbaseCurrency struct {
	ID      string
	Name    string
	MinSize float64 `json:"min_size,string"`
}

type CoinbaseHistory struct {
	Time   int64
	Low    float64
	High   float64
	Open   float64
	Close  float64
	Volume float64
}

func (c *Coinbase) SetDefaults() {
	c.Name = "Coinbase"
	c.Enabled = true
	c.Verbose = false
	c.TakerFee = 0.25
	c.MakerFee = 0
	c.Verbose = false
	c.Websocket = false
	c.RESTPollingDelay = 10
}

func (c *Coinbase) GetName() string {
	return c.Name
}

func (c *Coinbase) SetEnabled(enabled bool) {
	c.Enabled = enabled
}

func (c *Coinbase) IsEnabled() bool {
	return c.Enabled
}

func (c *Coinbase) GetFee(maker bool) float64 {
	if maker {
		return c.MakerFee
	} else {
		return c.TakerFee
	}
}

func (c *Coinbase) Run() {
	if c.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", c.GetName(), IsEnabled(c.Websocket), COINBASE_WEBSOCKET_URL)
		log.Printf("%s polling delay: %ds.\n", c.GetName(), c.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", c.GetName(), len(c.EnabledPairs), c.EnabledPairs)
	}

	if c.Websocket {
		go c.WebsocketClient()
	}

	exchangeProducts, err := c.GetProducts()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", c.GetName())
	} else {
		currencies := []string{}
		for _, x := range exchangeProducts {
			if x.ID != "BTC" && x.ID != "USD" && x.ID != "GBP" {
				currencies = append(currencies, x.ID[0:3]+x.ID[4:])
			}
		}
		diff := StringSliceDifference(c.AvailablePairs, currencies)
		if len(diff) > 0 {
			exch, err := GetExchangeConfig(c.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", c.Name, diff)
				exch.AvailablePairs = JoinStrings(currencies, ",")
				UpdateExchangeConfig(exch)
			}
		}
	}

	for c.Enabled {
		for _, x := range c.EnabledPairs {
			currency := x[0:3] + "-" + x[3:]
			go func() {
				stats := c.GetStats(currency)
				ticker := c.GetTicker(currency)
				log.Printf("Coinbase %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Price, stats.High, stats.Low, stats.Volume)
				AddExchangeInfo(c.GetName(), currency[0:3], currency[4:], ticker.Price, stats.Volume)
			}()
		}
		time.Sleep(time.Second * c.RESTPollingDelay)
	}
}

func (c *Coinbase) SetAPIKeys(password, apiKey, apiSecret string) {
	if !c.AuthenticatedAPISupport {
		return
	}

	c.Password = password
	c.APIKey = apiKey
	result, err := Base64Decode(apiSecret)

	if err != nil {
		log.Printf("%s unable to decode secret key.", c.GetName())
		c.Enabled = false
		return
	}

	c.APISecret = string(result)
}

func (c *Coinbase) GetProducts() ([]CoinbaseProduct, error) {
	products := []CoinbaseProduct{}
	err := SendHTTPGetRequest(COINBASE_API_URL+COINBASE_PRODUCTS, true, &products)

	if err != nil {
		return nil, err
	}

	return products, nil
}

func (c *Coinbase) GetOrderbook(symbol string, level int) (interface{}, error) {
	orderbook := CoinbaseOrderbookResponse{}
	path := ""
	if level > 0 {
		levelStr := strconv.Itoa(level)
		path = fmt.Sprintf("%s/%s/%s?level=%s", COINBASE_API_URL+COINBASE_PRODUCTS, symbol, COINBASE_ORDERBOOK, levelStr)
	} else {
		path = fmt.Sprintf("%s/%s/%s", COINBASE_API_URL+COINBASE_PRODUCTS, symbol, COINBASE_ORDERBOOK)
	}

	err := SendHTTPGetRequest(path, true, &orderbook)
	if err != nil {
		return nil, err
	}

	if level == 3 {
		ob := CoinbaseOrderbookL3{}
		ob.Sequence = orderbook.Sequence
		for _, x := range orderbook.Asks {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]CoinbaseOrderL3, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].OrderID = x[2].(string)
			ob.Asks = append(ob.Asks, order)
		}
		for _, x := range orderbook.Bids {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]CoinbaseOrderL3, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].OrderID = x[2].(string)
			ob.Bids = append(ob.Bids, order)
		}
		return ob, nil
	} else {
		ob := CoinbaseOrderbookL1L2{}
		ob.Sequence = orderbook.Sequence
		for _, x := range orderbook.Asks {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]CoinbaseOrderL1L2, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].NumOrders = x[2].(float64)
			ob.Asks = append(ob.Asks, order)
		}
		for _, x := range orderbook.Bids {
			price, err := strconv.ParseFloat((x[0].(string)), 64)
			if err != nil {
				continue
			}
			amount, err := strconv.ParseFloat((x[1].(string)), 64)
			if err != nil {
				continue
			}

			order := make([]CoinbaseOrderL1L2, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].NumOrders = x[2].(float64)
			ob.Bids = append(ob.Bids, order)
		}
		return ob, nil
	}
}

func (c *Coinbase) GetTicker(symbol string) (CoinbaseTicker, error) {
	ticker := CoinbaseTicker{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL+COINBASE_PRODUCTS, symbol, COINBASE_TICKER)
	err := SendHTTPGetRequest(path, true, &ticker)

	if err != nil {
		return ticker, err
	}
	return ticker, nil
}

func (c *Coinbase) GetTrades(symbol string) ([]CoinbaseTrade, error) {
	trades := []CoinbaseTrade{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL+COINBASE_PRODUCTS, symbol, COINBASE_TRADES)
	err := SendHTTPGetRequest(path, true, &trades)

	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (c *Coinbase) GetHistoricRates(symbol string, start, end, granularity int64) ([]CoinbaseHistory, error) {
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

	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL+COINBASE_PRODUCTS, symbol, COINBASE_HISTORY)
	encoded := values.Encode()

	if len(encoded) > 0 {
		path += encoded
	}

	err := SendHTTPGetRequest(path, true, &history)

	if err != nil {
		return nil, err
	}
	return history, nil
}

func (c *Coinbase) GetStats(symbol string) (CoinbaseStats, error) {
	stats := CoinbaseStats{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_API_URL+COINBASE_PRODUCTS, symbol, COINBASE_STATS)
	err := SendHTTPGetRequest(path, true, &stats)

	if err != nil {
		return stats, err
	}
	return stats, nil
}

func (c *Coinbase) GetCurrencies() ([]CoinbaseCurrency, error) {
	currencies := []CoinbaseCurrency{}
	err := SendHTTPGetRequest(COINBASE_API_URL+COINBASE_CURRENCIES, true, &currencies)

	if err != nil {
		return nil, err
	}
	return currencies, nil
}

type CoinbaseAccountResponse struct {
	ID        string  `json:"id"`
	Balance   float64 `json:"balance,string"`
	Hold      float64 `json:"hold,string"`
	Available float64 `json:"available,string"`
	Currency  string  `json:"currency"`
}

func (c *Coinbase) GetAccounts() ([]CoinbaseAccountResponse, error) {
	resp := []CoinbaseAccountResponse{}
	err := c.SendAuthenticatedHTTPRequest("GET", COINBASE_API_URL+COINBASE_ACCOUNTS, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Coinbase) GetAccount(account string) (CoinbaseAccountResponse, error) {
	resp := CoinbaseAccountResponse{}
	path := fmt.Sprintf("%s/%s", COINBASE_ACCOUNTS, account)
	err := c.SendAuthenticatedHTTPRequest("GET", COINBASE_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

type CoinbaseAccountLedgerResponse struct {
	ID        string      `json:"id"`
	CreatedAt string      `json:"created_at"`
	Amount    float64     `json:"amount,string"`
	Balance   float64     `json:"balance,string"`
	Type      string      `json:"type"`
	details   interface{} `json:"details"`
}

func (c *Coinbase) GetAccountHistory(accountID string) ([]CoinbaseAccountLedgerResponse, error) {
	resp := []CoinbaseAccountLedgerResponse{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_ACCOUNTS, accountID, COINBASE_LEDGER)
	err := c.SendAuthenticatedHTTPRequest("GET", COINBASE_API_URL+path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type CoinbaseAccountHolds struct {
	ID        string  `json:"id"`
	AccountID string  `json:"account_id"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	Amount    float64 `json:"amount,string"`
	Type      string  `json:"type"`
	Reference string  `json:"ref"`
}

func (c *Coinbase) GetHolds(accountID string) ([]CoinbaseAccountHolds, error) {
	resp := []CoinbaseAccountHolds{}
	path := fmt.Sprintf("%s/%s/%s", COINBASE_ACCOUNTS, accountID, COINBASE_HOLDS)
	err := c.SendAuthenticatedHTTPRequest("GET", COINBASE_API_URL+path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Coinbase) PlaceOrder(clientRef string, price, amount float64, side string, productID, stp string) (string, error) {
	request := make(map[string]interface{})

	if clientRef != "" {
		request["client_oid"] = clientRef
	}

	request["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	request["size"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["side"] = side
	request["product_id"] = productID

	if stp != "" {
		request["stp"] = stp
	}

	type OrderResponse struct {
		ID string `json:"id"`
	}

	resp := OrderResponse{}
	err := c.SendAuthenticatedHTTPRequest("POST", COINBASE_API_URL+COINBASE_ORDERS, request, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (c *Coinbase) CancelOrder(orderID string) error {
	path := fmt.Sprintf("%s/%s", COINBASE_ORDERS, orderID)
	err := c.SendAuthenticatedHTTPRequest("DELETE", COINBASE_API_URL+path, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

type CoinbaseOrdersResponse struct {
	ID         string  `json:"id"`
	Size       float64 `json:"size,string"`
	Price      float64 `json:"price,string"`
	ProductID  string  `json:"product_id"`
	Status     string  `json:"status"`
	FilledSize float64 `json:"filled_size,string"`
	FillFees   float64 `json:"fill_fees,string"`
	Settled    bool    `json:"settled"`
	Side       string  `json:"side"`
	CreatedAt  string  `json:"created_at"`
}

func (c *Coinbase) GetOrders(params url.Values) ([]CoinbaseOrdersResponse, error) {
	path := COINBASE_API_URL + COINBASE_ORDERS

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp := []CoinbaseOrdersResponse{}
	err := c.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type CoinbaseOrderResponse struct {
	ID         string  `json:"id"`
	Size       float64 `json:"size,string"`
	Price      float64 `json:"price,string"`
	DoneReason string  `json:"done_reason"`
	Status     string  `json:"status"`
	Settled    bool    `json:"settled"`
	FilledSize float64 `json:"filled_size,string"`
	ProductID  string  `json:"product_id"`
	FillFees   float64 `json:"fill_fees,string"`
	Side       string  `json:"side"`
	CreatedAt  string  `json:"created_at"`
	DoneAt     string  `json:"done_at"`
}

func (c *Coinbase) GetOrder(orderID string) (CoinbaseOrderResponse, error) {
	path := fmt.Sprintf("%s/%s", COINBASE_ORDERS, orderID)
	resp := CoinbaseOrderResponse{}
	err := c.SendAuthenticatedHTTPRequest("GET", COINBASE_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

type CoinbaseFillResponse struct {
	TradeID   int     `json:"trade_id"`
	ProductID string  `json:"product_id"`
	Price     float64 `json:"price,string"`
	Size      float64 `json:"size,string"`
	OrderID   string  `json:"order_id"`
	CreatedAt string  `json:"created_at"`
	Liquidity string  `json:"liquidity"`
	Fee       float64 `json:"fee,string"`
	Settled   bool    `json:"settled"`
	Side      string  `json:"side"`
}

func (c *Coinbase) GetFills(params url.Values) ([]CoinbaseFillResponse, error) {
	path := COINBASE_API_URL + COINBASE_FILLS

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp := []CoinbaseFillResponse{}
	err := c.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Coinbase) Transfer(transferType string, amount float64, accountID string) error {
	request := make(map[string]interface{})
	request["type"] = transferType
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["coinbase_account_id"] = accountID

	err := c.SendAuthenticatedHTTPRequest("POST", COINBASE_API_URL+COINBASE_TRANSFERS, request, nil)
	if err != nil {
		return err
	}
	return nil
}

type CoinbaseReportResponse struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at"`
	ExpiresAt   string `json:"expires_at"`
	FileURL     string `json:"file_url"`
	Params      struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:params"`
}

func (c *Coinbase) GetReport(reportType, startDate, endDate string) (CoinbaseReportResponse, error) {
	request := make(map[string]interface{})
	request["type"] = reportType
	request["start_date"] = startDate
	request["end_date"] = endDate

	resp := CoinbaseReportResponse{}
	err := c.SendAuthenticatedHTTPRequest("POST", COINBASE_API_URL+COINBASE_REPORTS, request, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (c *Coinbase) GetReportStatus(reportID string) (CoinbaseReportResponse, error) {
	path := fmt.Sprintf("%s/%s", COINBASE_REPORTS, reportID)
	resp := CoinbaseReportResponse{}
	err := c.SendAuthenticatedHTTPRequest("POST", COINBASE_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (c *Coinbase) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	payload := []byte("")

	if params != nil {
		payload, err = JSONEncode(params)

		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
		}

		if c.Verbose {
			log.Printf("Request JSON: %s\n", payload)
		}
	}

	message := timestamp + method + path + string(payload)
	hmac := GetHMAC(HASH_SHA256, []byte(message), []byte(c.APISecret))
	headers := make(map[string]string)
	headers["CB-ACCESS-SIGN"] = Base64Encode([]byte(hmac))
	headers["CB-ACCESS-TIMESTAMP"] = timestamp
	headers["CB-ACCESS-KEY"] = c.APIKey
	headers["CB-ACCESS-PASSPHRASE"] = c.Password
	headers["Content-Type"] = "application/json"

	resp, err := SendHTTPRequest(method, COINBASE_API_URL+path, headers, bytes.NewBuffer(payload))

	if c.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
