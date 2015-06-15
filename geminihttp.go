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
	GEMINI_API_URL     = "https://api.gemini.com"
	GEMINI_API_VERSION = "1"

	GEMINI_SYMBOLS            = "symbols"
	GEMINI_ORDERBOOK          = "book"
	GEMINI_TRADES             = "trades"
	GEMINI_ORDERS             = "orders"
	GEMINI_ORDER_NEW          = "order/new"
	GEMINI_ORDER_CANCEL       = "order/cancel"
	GEMINI_ORDER_CANCEL_MULTI = "order/cancel/multi"
	GEMINI_ORDER_CANCEL_ALL   = "order/cancel/all"
	GEMINI_ORDER_STATUS       = "order/status"
	GEMINI_MYTRADES           = "mytrades"
	GEMINI_BALANCES           = "balances"
)

type Gemini struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey, APISecret       string
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
}

type GeminiOrderbookEntry struct {
	Price    float64 `json:"price,string"`
	Quantity float64 `json:"quantity,string"`
}

type GeminiOrderbook struct {
	Bids []GeminiOrderbookEntry `json:"bids"`
	Asks []GeminiOrderbookEntry `json:"asks"`
}

type GeminiTrade struct {
	Timestamp int64   `json:"timestamp"`
	TID       int64   `json:"tid"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Side      string  `json:"taker"`
}

type GeminiOrder struct {
	OrderID           int64   `json:"order_id"`
	Symbol            string  `json:"symbol"`
	Price             float64 `json:"price,string"`
	AvgExecutionPrice float64 `json:"avg_execution_price,string"`
	Side              string  `json:"side"`
	Type              string  `json:"type"`
	Timestamp         int64   `json:"timestamp"`
	IsLive            bool    `json:"is_live"`
	IsCancelled       bool    `json:"is_cancelled"`
	ExecutedAmount    float64 `json:"executed_amount,string"`
	RemainingAmount   float64 `json:"remaining_amount,string"`
	OriginalAmount    float64 `json:"original_amount,string"`
}

type GeminiOrderResult struct {
	Result bool `json:"result"`
}

type GeminiTradeHistory struct {
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
	Timestamp   int64   `json:"timestamp"`
	Type        string  `json:"type"`
	FeeCurrency string  `json:"fee_currency"`
	FeeAmount   float64 `json:"fee_amount"`
	TID         int64   `json:"tid"`
	OrderID     int64   `json:"order_id"`
}

type GeminiBalance struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	Available float64 `json:"available"`
}

func (g *Gemini) SetDefaults() {
	g.Name = "Gemini"
	g.Enabled = true
	g.Verbose = false
	g.Websocket = false
	g.RESTPollingDelay = 10
}

func (g *Gemini) GetName() string {
	return g.Name
}

func (g *Gemini) SetEnabled(enabled bool) {
	g.Enabled = enabled
}

func (g *Gemini) IsEnabled() bool {
	return g.Enabled
}

func (g *Gemini) SetAPIKeys(apiKey, apiSecret string) {
	g.APIKey = apiKey
	g.APISecret = apiSecret
}

func (g *Gemini) Run() {
	if g.Verbose {
		log.Printf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	exchangeProducts, err := g.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", g.GetName())
	} else {
		exchangeProducts = SplitStrings(StringToUpper(JoinStrings(exchangeProducts, ",")), ",")
		diff := StringSliceDifference(g.AvailablePairs, exchangeProducts)
		if len(diff) > 0 {
			exch, err := GetExchangeConfig(g.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", g.Name, diff)
				exch.AvailablePairs = JoinStrings(exchangeProducts, ",")
				UpdateExchangeConfig(exch)
			}
		}
	}

	for g.Enabled {
		/* Ticker has not been implemented yet
		for _, x := range g.EnabledPairs {
			currency := x
			log.Println(currency)
			go func() {
				//ticker := g.GetTicker(currency)
				//log.Printf("Gemini %s Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				//AddExchangeInfo(g.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		*/
		time.Sleep(time.Second * g.RESTPollingDelay)
	}
}

func (g *Gemini) GetSymbols() ([]string, error) {
	symbols := []string{}
	path := fmt.Sprintf("%s/v%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_SYMBOLS)
	err := SendHTTPGetRequest(path, true, &symbols)
	if err != nil {
		return nil, err
	}
	return symbols, nil
}

func (g *Gemini) GetOrderbook(currency string, params url.Values) (GeminiOrderbook, error) {
	orderbook := GeminiOrderbook{}
	path := fmt.Sprintf("%s/v%s/%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_ORDERBOOK, currency)

	if params.Encode() != "" {
		path += "?" + params.Encode()
	}

	err := SendHTTPGetRequest(path, true, &orderbook)
	if err != nil {
		return GeminiOrderbook{}, err
	}

	return orderbook, nil
}

func (g *Gemini) GetTrades(currency string, params url.Values) ([]GeminiTrade, error) {
	trades := []GeminiTrade{}
	path := fmt.Sprintf("%s/v%s/%s/%s", GEMINI_API_URL, GEMINI_API_VERSION, GEMINI_TRADES, currency)

	if params.Encode() != "" {
		path += "?" + params.Encode()
	}

	err := SendHTTPGetRequest(path, true, &trades)
	if err != nil {
		return []GeminiTrade{}, err
	}

	return trades, nil
}

func (g *Gemini) NewOrder(symbol string, amount, price float64, side, orderType string) (int64, error) {
	request := make(map[string]interface{})
	request["symbol"] = symbol
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	request["side"] = side
	request["type"] = orderType

	response := GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_NEW, request, &response)
	if err != nil {
		return 0, err
	}
	return response.OrderID, nil
}

func (g *Gemini) CancelOrder(OrderID int64) (GeminiOrder, error) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID

	response := GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_CANCEL, request, &response)
	if err != nil {
		return GeminiOrder{}, err
	}
	return response, nil
}

func (g *Gemini) CancelMultipleOrders(orders []int64) ([]GeminiOrderResult, error) {
	request := make(map[string]interface{})
	request["order_ids"] = orders

	response := []GeminiOrderResult{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_CANCEL_MULTI, request, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) CancelAllOrders() ([]GeminiOrderResult, error) {
	response := []GeminiOrderResult{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_CANCEL_ALL, nil, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) GetOrderStatus(orderID int64) (GeminiOrder, error) {
	request := make(map[string]interface{})
	request["order_id"] = orderID

	response := GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDER_STATUS, request, &response)
	if err != nil {
		return GeminiOrder{}, err
	}
	return response, nil
}

func (g *Gemini) GetOrders() ([]GeminiOrder, error) {
	response := []GeminiOrder{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_ORDERS, nil, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) GetTradeHistory(symbol string, timestamp int64) ([]GeminiTradeHistory, error) {
	request := make(map[string]interface{})
	request["symbol"] = symbol
	request["timestamp"] = timestamp

	response := []GeminiTradeHistory{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_MYTRADES, request, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) GetBalances() ([]GeminiBalance, error) {
	response := []GeminiBalance{}
	err := g.SendAuthenticatedHTTPRequest("POST", GEMINI_BALANCES, nil, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (g *Gemini) SendAuthenticatedHTTPRequest(method, path string, params map[string]interface{}, result interface{}) (err error) {
	request := make(map[string]interface{})
	request["request"] = fmt.Sprintf("/v%s/%s", GEMINI_API_VERSION, path)
	request["nonce"] = time.Now().UnixNano()

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJson, err := JSONEncode(request)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	if g.Verbose {
		log.Printf("Request JSON: %s\n", PayloadJson)
	}

	PayloadBase64 := Base64Encode(PayloadJson)
	hmac := GetHMAC(HASH_SHA512_384, []byte(PayloadBase64), []byte(g.APISecret))
	headers := make(map[string]string)
	headers["X-GEMINI-APIKEY"] = g.APIKey
	headers["X-GEMINI-PAYLOAD"] = PayloadBase64
	headers["X-GEMINI-SIGNATURE"] = HexEncodeToString(hmac)

	resp, err := SendHTTPRequest(method, BITFINEX_API_URL+path, headers, strings.NewReader(""))

	if g.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
