package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	GDAX_API_URL     = "https://api.gdax.com/"
	GDAX_API_VERISON = "0"
	GDAX_PRODUCTS    = "products"
	GDAX_ORDERBOOK   = "book"
	GDAX_TICKER      = "ticker"
	GDAX_TRADES      = "trades"
	GDAX_HISTORY     = "candles"
	GDAX_STATS       = "stats"
	GDAX_CURRENCIES  = "currencies"
	GDAX_ACCOUNTS    = "accounts"
	GDAX_LEDGER      = "ledger"
	GDAX_HOLDS       = "holds"
	GDAX_ORDERS      = "orders"
	GDAX_FILLS       = "fills"
	GDAX_TRANSFERS   = "transfers"
	GDAX_REPORTS     = "reports"
)

type GDAX struct {
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

type GDAXTicker struct {
	TradeID int64   `json:"trade_id"`
	Price   float64 `json:"price,string"`
	Size    float64 `json:"size,string"`
	Time    string  `json:"time"`
}

type GDAXProduct struct {
	ID             string  `json:"id"`
	BaseCurrency   string  `json:"base_currency"`
	QuoteCurrency  string  `json:"quote_currency"`
	BaseMinSize    float64 `json:"base_min_size,string"`
	BaseMaxSize    int64   `json:"base_max_size,string"`
	QuoteIncrement float64 `json:"quote_increment,string"`
	DisplayName    string  `json:"string"`
}

type GDAXOrderL1L2 struct {
	Price     float64
	Amount    float64
	NumOrders float64
}

type GDAXOrderL3 struct {
	Price   float64
	Amount  float64
	OrderID string
}

type GDAXOrderbookL1L2 struct {
	Sequence int64             `json:"sequence"`
	Bids     [][]GDAXOrderL1L2 `json:"asks"`
	Asks     [][]GDAXOrderL1L2 `json:"asks"`
}

type GDAXOrderbookL3 struct {
	Sequence int64           `json:"sequence"`
	Bids     [][]GDAXOrderL3 `json:"asks"`
	Asks     [][]GDAXOrderL3 `json:"asks"`
}

type GDAXOrderbookResponse struct {
	Sequence int64           `json:"sequence"`
	Bids     [][]interface{} `json:"bids"`
	Asks     [][]interface{} `json:"asks"`
}

type GDAXTrade struct {
	TradeID int64   `json:"trade_id"`
	Price   float64 `json:"price,string"`
	Size    float64 `json:"size,string"`
	Time    string  `json:"time"`
	Side    string  `json:"side"`
}

type GDAXStats struct {
	Open   float64 `json:"open,string"`
	High   float64 `json:"high,string"`
	Low    float64 `json:"low,string"`
	Volume float64 `json:"volume,string"`
}

type GDAXCurrency struct {
	ID      string
	Name    string
	MinSize float64 `json:"min_size,string"`
}

type GDAXHistory struct {
	Time   int64
	Low    float64
	High   float64
	Open   float64
	Close  float64
	Volume float64
}

func (g *GDAX) SetDefaults() {
	g.Name = "GDAX"
	g.Enabled = false
	g.Verbose = false
	g.TakerFee = 0.25
	g.MakerFee = 0
	g.Verbose = false
	g.Websocket = false
	g.RESTPollingDelay = 10
}

func (g *GDAX) GetName() string {
	return g.Name
}

func (g *GDAX) SetEnabled(enabled bool) {
	g.Enabled = enabled
}

func (g *GDAX) IsEnabled() bool {
	return g.Enabled
}

func (g *GDAX) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		g.SetEnabled(false)
	} else {
		g.Enabled = true
		g.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		g.SetAPIKeys(exch.ClientID, exch.APIKey, exch.APISecret)
		g.RESTPollingDelay = exch.RESTPollingDelay
		g.Verbose = exch.Verbose
		g.Websocket = exch.Websocket
		g.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		g.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		g.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *GDAX) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (g *GDAX) Start() {
	go g.Run()
}

func (g *GDAX) GetFee(maker bool) float64 {
	if maker {
		return g.MakerFee
	} else {
		return g.TakerFee
	}
}

func (g *GDAX) Run() {
	if g.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", g.GetName(), common.IsEnabled(g.Websocket), GDAX_WEBSOCKET_URL)
		log.Printf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	if g.Websocket {
		go g.WebsocketClient()
	}

	exchangeProducts, err := g.GetProducts()
	if err != nil {
		log.Printf("%s Failed to get available products.\n", g.GetName())
	} else {
		currencies := []string{}
		for _, x := range exchangeProducts {
			if x.ID != "BTC" && x.ID != "USD" && x.ID != "GBP" {
				currencies = append(currencies, x.ID[0:3]+x.ID[4:])
			}
		}
		diff := common.StringSliceDifference(g.AvailablePairs, currencies)
		if len(diff) > 0 {
			exch, err := bot.config.GetExchangeConfig(g.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", g.Name, diff)
				exch.AvailablePairs = common.JoinStrings(currencies, ",")
				bot.config.UpdateExchangeConfig(exch)
			}
		}
	}

	for g.Enabled {
		for _, x := range g.EnabledPairs {
			currency := x[0:3] + "-" + x[3:]
			go func() {
				ticker, err := g.GetTickerPrice(currency)

				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("GDAX %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				AddExchangeInfo(g.GetName(), currency[0:3], currency[4:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * g.RESTPollingDelay)
	}
}

func (g *GDAX) SetAPIKeys(password, apiKey, apiSecret string) {
	if !g.AuthenticatedAPISupport {
		return
	}

	g.Password = password
	g.APIKey = apiKey
	result, err := common.Base64Decode(apiSecret)

	if err != nil {
		log.Printf("%s unable to decode secret key.", g.GetName())
		g.Enabled = false
		return
	}

	g.APISecret = string(result)
}

func (g *GDAX) GetProducts() ([]GDAXProduct, error) {
	products := []GDAXProduct{}
	err := common.SendHTTPGetRequest(GDAX_API_URL+GDAX_PRODUCTS, true, &products)

	if err != nil {
		return nil, err
	}

	return products, nil
}

func (g *GDAX) GetOrderbook(symbol string, level int) (interface{}, error) {
	orderbook := GDAXOrderbookResponse{}
	path := ""
	if level > 0 {
		levelStr := strconv.Itoa(level)
		path = fmt.Sprintf("%s/%s/%s?level=%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_ORDERBOOK, levelStr)
	} else {
		path = fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_ORDERBOOK)
	}

	err := common.SendHTTPGetRequest(path, true, &orderbook)
	if err != nil {
		return nil, err
	}

	if level == 3 {
		ob := GDAXOrderbookL3{}
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

			order := make([]GDAXOrderL3, 1)
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

			order := make([]GDAXOrderL3, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].OrderID = x[2].(string)
			ob.Bids = append(ob.Bids, order)
		}
		return ob, nil
	} else {
		ob := GDAXOrderbookL1L2{}
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

			order := make([]GDAXOrderL1L2, 1)
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

			order := make([]GDAXOrderL1L2, 1)
			order[0].Price = price
			order[0].Amount = amount
			order[0].NumOrders = x[2].(float64)
			ob.Bids = append(ob.Bids, order)
		}
		return ob, nil
	}
}

func (g *GDAX) GetTicker(symbol string) (GDAXTicker, error) {
	ticker := GDAXTicker{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_TICKER)
	err := common.SendHTTPGetRequest(path, true, &ticker)

	if err != nil {
		return ticker, err
	}
	return ticker, nil
}

func (g *GDAX) GetTickerPrice(currency string) (TickerPrice, error) {
	tickerNew, err := GetTicker(g.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice TickerPrice
	ticker, err := g.GetTicker(currency)
	if err != nil {
		return TickerPrice{}, err
	}

	stats, err := g.GetStats(currency)

	if err != nil {
		return TickerPrice{}, err
	}

	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[4:]
	tickerPrice.CurrencyPair = tickerPrice.FirstCurrency + "_" + tickerPrice.SecondCurrency
	tickerPrice.Volume = stats.Volume
	tickerPrice.Last = ticker.Price
	tickerPrice.High = stats.High
	tickerPrice.Low = stats.Low
	ProcessTicker(g.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (g *GDAX) GetTrades(symbol string) ([]GDAXTrade, error) {
	trades := []GDAXTrade{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_TRADES)
	err := common.SendHTTPGetRequest(path, true, &trades)

	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (g *GDAX) GetHistoricRates(symbol string, start, end, granularity int64) ([]GDAXHistory, error) {
	history := []GDAXHistory{}
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

	path := common.EncodeURLValues(fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_HISTORY), values)
	err := common.SendHTTPGetRequest(path, true, &history)

	if err != nil {
		return nil, err
	}
	return history, nil
}

func (g *GDAX) GetStats(symbol string) (GDAXStats, error) {
	stats := GDAXStats{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_API_URL+GDAX_PRODUCTS, symbol, GDAX_STATS)
	err := common.SendHTTPGetRequest(path, true, &stats)

	if err != nil {
		return stats, err
	}
	return stats, nil
}

func (g *GDAX) GetCurrencies() ([]GDAXCurrency, error) {
	currencies := []GDAXCurrency{}
	err := common.SendHTTPGetRequest(GDAX_API_URL+GDAX_CURRENCIES, true, &currencies)

	if err != nil {
		return nil, err
	}
	return currencies, nil
}

type GDAXAccountResponse struct {
	ID        string  `json:"id"`
	Balance   float64 `json:"balance,string"`
	Hold      float64 `json:"hold,string"`
	Available float64 `json:"available,string"`
	Currency  string  `json:"currency"`
}

func (g *GDAX) GetAccounts() ([]GDAXAccountResponse, error) {
	resp := []GDAXAccountResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+GDAX_ACCOUNTS, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (g *GDAX) GetAccount(account string) (GDAXAccountResponse, error) {
	resp := GDAXAccountResponse{}
	path := fmt.Sprintf("%s/%s", GDAX_ACCOUNTS, account)
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the GDAX exchange
func (e *GDAX) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccounts()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].Hold

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

type GDAXAccountLedgerResponse struct {
	ID        string      `json:"id"`
	CreatedAt string      `json:"created_at"`
	Amount    float64     `json:"amount,string"`
	Balance   float64     `json:"balance,string"`
	Type      string      `json:"type"`
	details   interface{} `json:"details"`
}

func (g *GDAX) GetAccountHistory(accountID string) ([]GDAXAccountLedgerResponse, error) {
	resp := []GDAXAccountLedgerResponse{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_ACCOUNTS, accountID, GDAX_LEDGER)
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type GDAXAccountHolds struct {
	ID        string  `json:"id"`
	AccountID string  `json:"account_id"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	Amount    float64 `json:"amount,string"`
	Type      string  `json:"type"`
	Reference string  `json:"ref"`
}

func (g *GDAX) GetHolds(accountID string) ([]GDAXAccountHolds, error) {
	resp := []GDAXAccountHolds{}
	path := fmt.Sprintf("%s/%s/%s", GDAX_ACCOUNTS, accountID, GDAX_HOLDS)
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (g *GDAX) PlaceOrder(clientRef string, price, amount float64, side string, productID, stp string) (string, error) {
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
	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+GDAX_ORDERS, request, &resp)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (g *GDAX) CancelOrder(orderID string) error {
	path := fmt.Sprintf("%s/%s", GDAX_ORDERS, orderID)
	err := g.SendAuthenticatedHTTPRequest("DELETE", GDAX_API_URL+path, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

type GDAXOrdersResponse struct {
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

func (g *GDAX) GetOrders(params url.Values) ([]GDAXOrdersResponse, error) {
	path := common.EncodeURLValues(GDAX_API_URL+GDAX_ORDERS, params)
	resp := []GDAXOrdersResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type GDAXOrderResponse struct {
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

func (g *GDAX) GetOrder(orderID string) (GDAXOrderResponse, error) {
	path := fmt.Sprintf("%s/%s", GDAX_ORDERS, orderID)
	resp := GDAXOrderResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

type GDAXFillResponse struct {
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

func (g *GDAX) GetFills(params url.Values) ([]GDAXFillResponse, error) {
	path := common.EncodeURLValues(GDAX_API_URL+GDAX_FILLS, params)
	resp := []GDAXFillResponse{}
	err := g.SendAuthenticatedHTTPRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (g *GDAX) Transfer(transferType string, amount float64, accountID string) error {
	request := make(map[string]interface{})
	request["type"] = transferType
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["GDAX_account_id"] = accountID

	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+GDAX_TRANSFERS, request, nil)
	if err != nil {
		return err
	}
	return nil
}

type GDAXReportResponse struct {
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

func (g *GDAX) GetReport(reportType, startDate, endDate string) (GDAXReportResponse, error) {
	request := make(map[string]interface{})
	request["type"] = reportType
	request["start_date"] = startDate
	request["end_date"] = endDate

	resp := GDAXReportResponse{}
	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+GDAX_REPORTS, request, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (g *GDAX) GetReportStatus(reportID string) (GDAXReportResponse, error) {
	path := fmt.Sprintf("%s/%s", GDAX_REPORTS, reportID)
	resp := GDAXReportResponse{}
	err := g.SendAuthenticatedHTTPRequest("POST", GDAX_API_URL+path, nil, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (g *GDAX) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	payload := []byte("")

	if params != nil {
		payload, err = common.JSONEncode(params)

		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
		}

		if g.Verbose {
			log.Printf("Request JSON: %s\n", payload)
		}
	}

	message := timestamp + method + path + string(payload)
	hmac := common.GetHMAC(common.HASH_SHA256, []byte(message), []byte(g.APISecret))
	headers := make(map[string]string)
	headers["CB-ACCESS-SIGN"] = common.Base64Encode([]byte(hmac))
	headers["CB-ACCESS-TIMESTAMP"] = timestamp
	headers["CB-ACCESS-KEY"] = g.APIKey
	headers["CB-ACCESS-PASSPHRASE"] = g.Password
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest(method, GDAX_API_URL+path, headers, bytes.NewBuffer(payload))

	if g.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
