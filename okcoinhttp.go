package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	OKCOIN_API_URL             = "https://www.okcoin.com/api/v1/"
	OKCOIN_API_URL_CHINA       = "https://www.okcoin.cn/api/v1/"
	OKCOIN_API_VERSION         = "1"
	OKCOIN_WEBSOCKET_URL       = "wss://real.okcoin.com:10440/websocket/okcoinapi"
	OKCOIN_WEBSOCKET_URL_CHINA = "wss://real.okcoin.cn:10440/websocket/okcoinapi"
)

type OKCoin struct {
	Name                         string
	Enabled                      bool
	Verbose                      bool
	Websocket                    bool
	WebsocketURL                 string
	RESTPollingDelay             time.Duration
	AuthenticatedAPISupport      bool
	APIUrl, PartnerID, SecretKey string
	TakerFee, MakerFee           float64
	RESTErrors                   map[string]string
	WebsocketErrors              map[string]string
	BaseCurrencies               []string
	AvailablePairs               []string
	EnabledPairs                 []string
	FuturesValues                []string
	WebsocketConn                *websocket.Conn
}

type OKCoinTicker struct {
	Buy  float64 `json:",string"`
	High float64 `json:",string"`
	Last float64 `json:",string"`
	Low  float64 `json:",string"`
	Sell float64 `json:",string"`
	Vol  float64 `json:",string"`
}

type OKCoinTickerResponse struct {
	Date   string
	Ticker OKCoinTicker
}
type OKCoinFuturesTicker struct {
	Last        float64
	Buy         float64
	Sell        float64
	High        float64
	Low         float64
	Vol         float64
	Contract_ID float64
	Unit_Amount float64
}

type OKCoinOrderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

type OKCoinFuturesTickerResponse struct {
	Date   string
	Ticker OKCoinFuturesTicker
}

type OKCoinBorrowInfo struct {
	BorrowBTC        float64 `json:"borrow_btc"`
	BorrowLTC        float64 `json:"borrow_ltc"`
	BorrowCNY        float64 `json:"borrow_cny"`
	CanBorrow        float64 `json:"can_borrow"`
	InterestBTC      float64 `json:"interest_btc"`
	InterestLTC      float64 `json:"interest_ltc"`
	Result           bool    `json:"result"`
	DailyInterestBTC float64 `json:"today_interest_btc"`
	DailyInterestLTC float64 `json:"today_interest_ltc"`
	DailyInterestCNY float64 `json:"today_interest_cny"`
}

type OKCoinBorrowOrder struct {
	Amount      float64 `json:"amount"`
	BorrowDate  float64 `json:"borrow_date"`
	BorrowID    int64   `json:"borrow_id"`
	Days        int64   `json:"days"`
	TradeAmount float64 `json:"deal_amount"`
	Rate        float64 `json:"rate"`
	Status      int64   `json:"status"`
	Symbol      string  `json:"symbol"`
}

type OKCoinRecord struct {
	Address            string  `json:"addr"`
	Account            int64   `json:"account,string"`
	Amount             float64 `json:"amount"`
	Bank               string  `json:"bank"`
	BenificiaryAddress string  `json:"benificiary_addr"`
	TransactionValue   float64 `json:"transaction_value"`
	Fee                float64 `json:"fee"`
	Date               float64 `json:"date"`
}

type OKCoinAccountRecords struct {
	Records []OKCoinRecord `json:"records"`
	Symbol  string         `json:"symbol"`
}

type OKCoinFuturesOrder struct {
	Amount       float64 `json:"amount"`
	ContractName string  `json:"contract_name"`
	DateCreated  float64 `json:"create_date"`
	TradeAmount  float64 `json:"deal_amount"`
	Fee          float64 `json:"fee"`
	LeverageRate float64 `json:"lever_rate"`
	OrderID      int64   `json:"order_id"`
	Price        float64 `json:"price"`
	AvgPrice     float64 `json:"avg_price"`
	Status       float64 `json:"status"`
	Symbol       string  `json:"symbol"`
	Type         int64   `json:"type"`
	UnitAmount   int64   `json:"unit_amount"`
}

type OKCoinFuturesHoldAmount struct {
	Amount       float64 `json:"amount"`
	ContractName string  `json:"contract_name"`
}

type OKCoinLendDepth struct {
	Amount float64 `json:"amount"`
	Days   string  `json:"days"`
	Num    int64   `json:"num"`
	Rate   float64 `json:"rate,string"`
}

type OKCoinFuturesExplosive struct {
	Amount      float64 `json:"amount,string"`
	DateCreated string  `json:"create_date"`
	Loss        float64 `json:"loss,string"`
	Type        int64   `json:"type"`
}

func (o *OKCoin) SetDefaults() {
	o.SetErrorDefaults()
	o.SetWebsocketErrorDefaults()

	if o.APIUrl == OKCOIN_API_URL {
		o.Name = "OKCOIN International"
		o.WebsocketURL = OKCOIN_WEBSOCKET_URL
	} else if o.APIUrl == OKCOIN_API_URL_CHINA {
		o.Name = "OKCOIN China"
		o.WebsocketURL = OKCOIN_WEBSOCKET_URL_CHINA
	}
	o.Enabled = true
	o.Verbose = false
	o.Websocket = false
	o.RESTPollingDelay = 10
	o.FuturesValues = []string{"this_week", "next_week", "quarter"}
}

func (o *OKCoin) GetName() string {
	return o.Name
}

func (o *OKCoin) SetEnabled(enabled bool) {
	o.Enabled = enabled
}

func (o *OKCoin) IsEnabled() bool {
	return o.Enabled
}

func (o *OKCoin) SetURL(url string) {
	o.APIUrl = url
}

func (o *OKCoin) SetAPIKeys(apiKey, apiSecret string) {
	o.PartnerID = apiKey
	o.SecretKey = apiSecret
}

func (o *OKCoin) GetFee(maker bool) float64 {
	if o.APIUrl == OKCOIN_API_URL {
		if maker {
			return o.MakerFee
		} else {
			return o.TakerFee
		}
	}
	// Chinese exchange does not have any trading fees
	return 0
}

func (o *OKCoin) Run() {
	if o.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", o.GetName(), IsEnabled(o.Websocket), o.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	if o.Websocket {
		go o.WebsocketClient()
	}

	for o.Enabled {
		for _, x := range o.EnabledPairs {
			currency := StringToLower(x[0:3] + "_" + x[3:])
			if o.APIUrl == OKCOIN_API_URL {
				for _, y := range o.FuturesValues {
					futuresValue := y
					go func() {
						ticker := o.GetFuturesTicker(currency, futuresValue)
						log.Printf("OKCoin Intl Futures %s (%s): Last %f High %f Low %f Volume %f\n", currency, futuresValue, ticker.Last, ticker.High, ticker.Low, ticker.Vol)
						AddExchangeInfo(o.GetName(), StringToUpper(currency[0:3]), StringToUpper(currency[4:]), ticker.Last, ticker.Vol)
					}()
				}
				go func() {
					ticker := o.GetTicker(currency)
					log.Printf("OKCoin Intl Spot %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Vol)
					AddExchangeInfo(o.GetName(), StringToUpper(currency[0:3]), StringToUpper(currency[4:]), ticker.Last, ticker.Vol)
				}()
			} else {
				go func() {
					ticker := o.GetTicker(currency)
					tickerLastUSD, _ := ConvertCurrency(ticker.Last, "CNY", "USD")
					tickerHighUSD, _ := ConvertCurrency(ticker.High, "CNY", "USD")
					tickerLowUSD, _ := ConvertCurrency(ticker.Low, "CNY", "USD")
					log.Printf("OKCoin China %s: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", currency, tickerLastUSD, ticker.Last, tickerHighUSD, ticker.High, tickerLowUSD, ticker.Low, ticker.Vol)
					AddExchangeInfo(o.GetName(), StringToUpper(currency[0:3]), StringToUpper(currency[4:]), ticker.Last, ticker.Vol)
					AddExchangeInfo(o.GetName(), StringToUpper(currency[0:3]), "USD", tickerLastUSD, ticker.Vol)
				}()
			}
		}
		time.Sleep(time.Second * o.RESTPollingDelay)
	}
}

func (o *OKCoin) GetTicker(symbol string) OKCoinTicker {
	resp := OKCoinTickerResponse{}
	path := fmt.Sprintf("ticker.do?symbol=%s&ok=1", symbol)
	err := SendHTTPGetRequest(o.APIUrl+path, true, &resp)

	if err != nil {
		log.Println(err)
		return OKCoinTicker{}
	}
	return resp.Ticker
}

func (o *OKCoin) GetKline(symbol, klineType string, size, since int64) []interface{} {
	resp := []interface{}{}
	path := fmt.Sprintf("kline.do?symbol=%stype=%s&size=%d&since=%d&ok=1", symbol, klineType, size, since)
	err := SendHTTPGetRequest(o.APIUrl+path, true, &resp)

	if err != nil {
		log.Println(err)
		return nil
	}
	return resp
}

func (o *OKCoin) GetLendDepth(symbol string) []OKCoinLendDepth {
	type Response struct {
		LendDepth []OKCoinLendDepth `json:"lend_depth"`
	}
	resp := Response{}
	path := fmt.Sprintf("lend_depth.do?symbol=%s&ok=1", symbol)
	err := SendHTTPGetRequest(o.APIUrl+path, true, &resp)

	if err != nil {
		log.Println(err)
		return []OKCoinLendDepth{}
	}
	return resp.LendDepth
}

func (o *OKCoin) GetFuturesTicker(symbol, contractType string) OKCoinFuturesTicker {
	resp := OKCoinFuturesTickerResponse{}
	path := fmt.Sprintf("future_ticker.do?symbol=%s&contract_type=%s", symbol, contractType)
	err := SendHTTPGetRequest(o.APIUrl+path, true, &resp)
	if err != nil {
		log.Println(err)
		return OKCoinFuturesTicker{}
	}
	return resp.Ticker
}

func (o *OKCoin) GetOrderBook(symbol string) bool {
	path := "depth.do?symbol=" + symbol
	err := SendHTTPGetRequest(o.APIUrl+path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesDepth(symbol, contractType string) bool {
	path := fmt.Sprintf("future_depth.do?symbol=%s&contract_type=%s", symbol, contractType)
	err := SendHTTPGetRequest(o.APIUrl+path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetTradeHistory(symbol string) bool {
	path := "trades.do?symbol=" + symbol
	err := SendHTTPGetRequest(o.APIUrl+path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesTrades(symbol, contractType string) bool {
	path := fmt.Sprintf("future_trades.do?symbol=%s&contract_type=%s", symbol, contractType)
	err := SendHTTPGetRequest(o.APIUrl+path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesIndex(symbol string) bool {
	path := "future_index.do?symbol=" + symbol
	err := SendHTTPGetRequest(o.APIUrl+path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesExchangeRate() bool {
	err := SendHTTPGetRequest(o.APIUrl+"exchange_rate.do", true, nil)
	if err != nil {
		log.Println(err)
	}
	return true
}

func (o *OKCoin) GetFuturesEstimatedPrice(symbol string) bool {
	path := "future_estimated_price.do?symbol=" + symbol
	err := SendHTTPGetRequest(o.APIUrl+path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesTradeHistory(symbol, date string, since int64) bool {
	path := fmt.Sprintf("future_trades_history.do?symbol=%s&date%s&since=%d", symbol, date, since)
	err := SendHTTPGetRequest(o.APIUrl+path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesKline(symbol, klineType, contractType string, size, since int64) []interface{} {
	resp := []interface{}{}
	path := fmt.Sprintf("future_kline.do?symbol=%s&type=%s&contract_type=%s&size=%d&since=%d", symbol, klineType, contractType, size, since)
	err := SendHTTPGetRequest(o.APIUrl+path, true, &resp)

	if err != nil {
		log.Println(err)
		return nil
	}
	return resp
}

func (o *OKCoin) GetFuturesHoldAmount(symbol, contractType string) []OKCoinFuturesHoldAmount {
	resp := []OKCoinFuturesHoldAmount{}
	path := fmt.Sprintf("future_hold_amount.do?symbol=%s&contract_type=%s", symbol, contractType)
	err := SendHTTPGetRequest(o.APIUrl+path, true, &resp)

	if err != nil {
		log.Println(err)
		return nil
	}
	return resp
}

func (o *OKCoin) GetFuturesExplosive(symbol, contractType string, status, currentPage, pageLength int64) []OKCoinFuturesExplosive {
	type Response struct {
		Data []OKCoinFuturesExplosive `json:"data"`
	}
	resp := Response{}
	path := fmt.Sprintf("future_explosive.do?symbol=%s&contract_type=%s&status=%d&current_page=%d&page_length=%d", symbol, contractType, status, currentPage, pageLength)
	err := SendHTTPGetRequest(o.APIUrl+path, true, &resp)

	if err != nil {
		log.Println(err)
		return nil
	}
	return resp.Data
}

func (o *OKCoin) GetUserInfo() {
	err := o.SendAuthenticatedHTTPRequest("userinfo.do", url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesUserInfo() {
	err := o.SendAuthenticatedHTTPRequest("future_userinfo.do", url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesPosition(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	err := o.SendAuthenticatedHTTPRequest("future_userinfo.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) Trade(amount, price float64, symbol, orderType string) {
	v := url.Values{}
	v.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	v.Set("price", strconv.FormatFloat(price, 'f', 8, 64))
	v.Set("symbol", symbol)
	v.Set("type", orderType)

	err := o.SendAuthenticatedHTTPRequest("trade.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) FuturesTrade(amount, price float64, matchPrice, leverage int64, symbol, contractType, orderType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("price", strconv.FormatFloat(price, 'f', 8, 64))
	v.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	v.Set("type", orderType)
	v.Set("match_price", strconv.FormatInt(matchPrice, 10))
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest("future_trade.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) BatchTrade(orderData string, symbol, orderType string) {
	v := url.Values{} //to-do batch trade support for orders_data
	v.Set("orders_data", orderData)
	v.Set("symbol", symbol)
	v.Set("type", orderType)

	err := o.SendAuthenticatedHTTPRequest("batch_trade.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) FuturesBatchTrade(orderData, symbol, contractType string, leverage int64, orderType string) {
	v := url.Values{} //to-do batch trade support for orders_data)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("orders_data", orderData)
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest("future_batch_trade.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) CancelOrder(orderID int64, symbol string) {
	v := url.Values{}
	v.Set("orders_id", strconv.FormatInt(orderID, 10))
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("cancel_order.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) CancelFuturesOrder(orderID int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := o.SendAuthenticatedHTTPRequest("future_cancel.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetOrderInfo(orderID int64, symbol string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := o.SendAuthenticatedHTTPRequest("order_info.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesOrderInfo(orderID, status, currentPage, pageLength int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("status", strconv.FormatInt(status, 10))
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))

	err := o.SendAuthenticatedHTTPRequest("future_order_info.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetOrdersInfo(orderID int64, orderType string, symbol string) {
	v := url.Values{}
	v.Set("orders_id", strconv.FormatInt(orderID, 10))
	v.Set("type", orderType)
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("orders_info.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFutureOrdersInfo(orderID int64, contractType, symbol string) {
	v := url.Values{}
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("contract_type", contractType)
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("future_orders_info.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetOrderHistory(orderID, pageLength, currentPage int64, orderType string, status, symbol string) {
	v := url.Values{}
	v.Set("orders_id", strconv.FormatInt(orderID, 10))
	v.Set("type", orderType)
	v.Set("symbol", symbol)
	v.Set("status", status)
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))

	err := o.SendAuthenticatedHTTPRequest("order_history.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) Withdrawal(symbol string, fee float64, tradePWD, address string, amount float64) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("chargefee", strconv.FormatFloat(fee, 'f', 8, 64))
	v.Set("trade_pwd", tradePWD)
	v.Set("withdraw_address", address)
	v.Set("withdraw_amount", strconv.FormatFloat(amount, 'f', 8, 64))

	err := o.SendAuthenticatedHTTPRequest("withdraw.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) CancelWithdrawal(withdrawalID int64) {
	v := url.Values{}
	v.Set("withdrawal_id", strconv.FormatInt(withdrawalID, 10))

	err := o.SendAuthenticatedHTTPRequest("cancel_withdraw.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesUserInfo4Fix() {
	v := url.Values{}

	err := o.SendAuthenticatedHTTPRequest("future_userinfo_4fix.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesUserPosition4Fix(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("type", strconv.FormatInt(1, 10))

	err := o.SendAuthenticatedHTTPRequest("future_position_4fix.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetBorrowInfo(symbol string) {
	v := url.Values{}
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("borrows_info.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) Borrow(symbol, days string, amount, rate float64) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("days", days)
	v.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	v.Set("rate", strconv.FormatFloat(rate, 'f', 8, 64))
	err := o.SendAuthenticatedHTTPRequest("borrow_money.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) CancelBorrow(symbol string, borrowID int64) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	err := o.SendAuthenticatedHTTPRequest("cancel_borrow.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetBorrowOrderInfo(borrowID int64) {
	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	err := o.SendAuthenticatedHTTPRequest("borrow_order_info.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetRepaymentInfo(borrowID int64) {
	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	err := o.SendAuthenticatedHTTPRequest("repayment.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetUnrepaymentsInfo(symbol string, currentPage, pageLength int) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))

	err := o.SendAuthenticatedHTTPRequest("unrepayments_info.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetAccountRecords(symbol string, recType, currentPage, pageLength int) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("type", strconv.Itoa(recType))
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))

	err := o.SendAuthenticatedHTTPRequest("account_records.do", v)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) SendAuthenticatedHTTPRequest(method string, v url.Values) (err error) {
	v.Set("api_key", o.PartnerID)
	hasher := GetMD5([]byte(v.Encode() + "&secret_key=" + o.SecretKey))
	v.Set("sign", strings.ToUpper(HexEncodeToString(hasher)))

	encoded := v.Encode()
	path := o.APIUrl + method

	if o.Verbose {
		log.Printf("Sending POST request to %s with params %s\n", path, encoded)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", path, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	if o.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	return nil
}

func (o *OKCoin) SetErrorDefaults() {
	o.RESTErrors = map[string]string{
		"10000": "Required field, can not be null",
		"10001": "Request frequency too high",
		"10002": "System error",
		"10003": "Not in reqest list, please try again later",
		"10004": "IP not allowed to access the resource",
		"10005": "'secretKey' does not exist",
		"10006": "'partner' does not exist",
		"10007": "Signature does not match",
		"10008": "Illegal parameter",
		"10009": "Order does not exist",
		"10010": "Insufficient funds",
		"10011": "Amount too low",
		"10012": "Only btc_usd/btc_cny ltc_usd,ltc_cny supported",
		"10013": "Only support https request",
		"10014": "Order price must be between 0 and 1,000,000",
		"10015": "Order price differs from current market price too much",
		"10016": "Insufficient coins balance",
		"10017": "API authorization error",
		"10018": "Borrow amount less than lower limit [usd/cny:100,btc:0.1,ltc:1]",
		"10019": "Loan agreement not checked",
		"10020": `Rate cannot exceed 1%`,
		"10021": `Rate cannot less than 0.01%`,
		"10023": "Fail to get latest ticker",
		"10024": "Balance not sufficient",
		"10025": "Quota is full, cannot borrow temporarily",
		"10026": "Loan (including reserved loan) and margin cannot be withdrawn",
		"10027": "Cannot withdraw within 24 hrs of authentication information modification",
		"10028": "Withdrawal amount exceeds daily limit",
		"10029": "Account has unpaid loan, please cancel/pay off the loan before withdraw",
		"10031": "Deposits can only be withdrawn after 6 confirmations",
		"10032": "Please enabled phone/google authenticator",
		"10033": "Fee higher than maximum network transaction fee",
		"10034": "Fee lower than minimum network transaction fee",
		"10035": "Insufficient BTC/LTC",
		"10036": "Withdrawal amount too low",
		"10037": "Trade password not set",
		"10040": "Withdrawal cancellation fails",
		"10041": "Withdrawal address not approved",
		"10042": "Admin password error",
		"10043": "Account equity error, withdrawal failure",
		"10044": "fail to cancel borrowing order",
		"10047": "This function is disabled for sub-account",
		"10100": "User account frozen",
		"10216": "Non-available API",
		"20001": "User does not exist",
		"20002": "Account frozen",
		"20003": "Account frozen due to liquidation",
		"20004": "Futures account frozen",
		"20005": "User futures account does not exist",
		"20006": "Required field missing",
		"20007": "Illegal parameter",
		"20008": "Futures account balance is too low",
		"20009": "Future contract status error",
		"20010": "Risk rate ratio does not exist",
		"20011": `Risk rate higher than 90% before opening position`,
		"20012": `Risk rate higher than 90% after opening position`,
		"20013": "Temporally no counter party price",
		"20014": "System error",
		"20015": "Order does not exist",
		"20016": "Close amount bigger than your open positions",
		"20017": "Not authorized/illegal operation",
		"20018": `Order price differ more than 5% from the price in the last minute`,
		"20019": "IP restricted from accessing the resource",
		"20020": "secretKey does not exist",
		"20021": "Index information does not exist",
		"20022": "Wrong API interface (Cross margin mode shall call cross margin API, fixed margin mode shall call fixed margin API)",
		"20023": "Account in fixed-margin mode",
		"20024": "Signature does not match",
		"20025": "Leverage rate error",
		"20026": "API Permission Error",
		"20027": "No transaction record",
		"20028": "No such contract",
	}
}
