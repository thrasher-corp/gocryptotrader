package main

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	HUOBI_API_URL     = "https://api.huobi.com/apiv2.php"
	HUOBI_API_VERSION = "2"
)

type HUOBI struct {
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

type HuobiTicker struct {
	High float64
	Low  float64
	Last float64
	Vol  float64
	Buy  float64
	Sell float64
}

type HuobiTickerResponse struct {
	Time   string
	Ticker HuobiTicker
}

func (h *HUOBI) SetDefaults() {
	h.Name = "Huobi"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.Websocket = false
	h.RESTPollingDelay = 10
}

func (h *HUOBI) GetName() string {
	return h.Name
}

func (h *HUOBI) SetEnabled(enabled bool) {
	h.Enabled = enabled
}

func (h *HUOBI) IsEnabled() bool {
	return h.Enabled
}

func (h *HUOBI) Setup(exch Exchanges) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret)
		h.RESTPollingDelay = exch.RESTPollingDelay
		h.Verbose = exch.Verbose
		h.Websocket = exch.Websocket
		h.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		h.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		h.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *HUOBI) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (h *HUOBI) Start() {
	go h.Run()
}

func (h *HUOBI) SetAPIKeys(apiKey, apiSecret string) {
	h.AccessKey = apiKey
	h.SecretKey = apiSecret
}

func (h *HUOBI) GetFee() float64 {
	return h.Fee
}

func (h *HUOBI) Run() {
	if h.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", h.GetName(), IsEnabled(h.Websocket), HUOBI_SOCKETIO_ADDRESS)
		log.Printf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	if h.Websocket {
		go h.WebsocketClient()
	}

	for h.Enabled {
		for _, x := range h.EnabledPairs {
			currency := StringToLower(x[0:3])
			go func() {
				ticker := h.GetTicker(currency)
				HuobiLastUSD, _ := ConvertCurrency(ticker.Last, "CNY", "USD")
				HuobiHighUSD, _ := ConvertCurrency(ticker.High, "CNY", "USD")
				HuobiLowUSD, _ := ConvertCurrency(ticker.Low, "CNY", "USD")
				log.Printf("Huobi %s: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", currency, HuobiLastUSD, ticker.Last, HuobiHighUSD, ticker.High, HuobiLowUSD, ticker.Low, ticker.Vol)
				AddExchangeInfo(h.GetName(), StringToUpper(currency[0:3]), StringToUpper(currency[3:]), ticker.Last, ticker.Vol)
				AddExchangeInfo(h.GetName(), StringToUpper(currency[0:3]), "USD", HuobiLastUSD, ticker.Vol)
			}()
		}
		time.Sleep(time.Second * h.RESTPollingDelay)
	}
}

func (h *HUOBI) GetTicker(symbol string) HuobiTicker {
	resp := HuobiTickerResponse{}
	path := fmt.Sprintf("http://api.huobi.com/staticmarket/ticker_%s_json.js", symbol)
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		log.Println(err)
		return HuobiTicker{}
	}
	return resp.Ticker
}

func (h *HUOBI) GetTickerPrice(currency string) TickerPrice {
	var tickerPrice TickerPrice
	ticker := h.GetTicker(currency)
	tickerPrice.Ask = ticker.Sell
	tickerPrice.Bid = ticker.Buy
	tickerPrice.CryptoCurrency = currency
	tickerPrice.Low = ticker.Low
	tickerPrice.Last = ticker.Last
	tickerPrice.Volume = ticker.Vol
	tickerPrice.High = ticker.High

	return tickerPrice
}

func (h *HUOBI) GetOrderBook(symbol string) bool {
	path := fmt.Sprintf("http://api.huobi.com/staticmarket/depth_%s_json.js", symbol)
	err := SendHTTPGetRequest(path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (h *HUOBI) GetAccountInfo() {
	err := h.SendAuthenticatedRequest("get_account_info", url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) GetOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_orders", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) GetOrderInfo(orderID, coinType int) {
	values := url.Values{}
	values.Set("id", strconv.Itoa(orderID))
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("order_info", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) Trade(orderType string, coinType int, price, amount float64) {
	values := url.Values{}
	if orderType != "buy" {
		orderType = "sell"
	}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	err := h.SendAuthenticatedRequest(orderType, values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) MarketTrade(orderType string, coinType int, price, amount float64) {
	values := url.Values{}
	if orderType != "buy_market" {
		orderType = "sell_market"
	}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	err := h.SendAuthenticatedRequest(orderType, values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) CancelOrder(orderID, coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("cancel_order", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) ModifyOrder(orderType string, coinType, orderID int, price, amount float64) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("id", strconv.Itoa(orderID))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	err := h.SendAuthenticatedRequest("modify_order", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) GetNewDealOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_new_deal_orders", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) GetOrderIDByTradeID(coinType, orderID int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("trade_id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("get_order_id_by_trade_id", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) SendAuthenticatedRequest(method string, v url.Values) error {
	v.Set("access_key", h.AccessKey)
	v.Set("created", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("method", method)
	hash := GetMD5([]byte(v.Encode() + "&secret_key=" + h.SecretKey))
	v.Set("sign", strings.ToLower(HexEncodeToString(hash)))
	encoded := v.Encode()

	if h.Verbose {
		log.Printf("Sending POST request to %s with params %s\n", HUOBI_API_URL, encoded)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", HUOBI_API_URL, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	if h.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	return nil
}

//TODO: retrieve HUOBI balance info
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the HUOBI exchange
func (e *HUOBI) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}
