package main

import (
	"errors"
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
	Contract_ID int64
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
	BorrowDate  int64   `json:"borrow_date"`
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
	o.Enabled = false
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

func (o *OKCoin) Setup(exch Exchanges) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket = exch.Websocket
		o.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		o.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		o.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (o *OKCoin) Start() {
	go o.Run()
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
						ticker, err := o.GetFuturesTicker(currency, futuresValue)
						if err != nil {
							log.Println(err)
							return
						}
						log.Printf("OKCoin Intl Futures %s (%s): Last %f High %f Low %f Volume %f\n", currency, futuresValue, ticker.Last, ticker.High, ticker.Low, ticker.Vol)
						AddExchangeInfo(o.GetName(), StringToUpper(currency[0:3]), StringToUpper(currency[4:]), ticker.Last, ticker.Vol)
					}()
				}
				go func() {
					ticker, err := o.GetTicker(currency)
					if err != nil {
						log.Println(err)
						return
					}
					log.Printf("OKCoin Intl Spot %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Vol)
					AddExchangeInfo(o.GetName(), StringToUpper(currency[0:3]), StringToUpper(currency[4:]), ticker.Last, ticker.Vol)
				}()
			} else {
				go func() {
					ticker, err := o.GetTicker(currency)
					if err != nil {
						log.Println(err)
						return
					}
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

func (o *OKCoin) GetTicker(symbol string) (OKCoinTicker, error) {
	resp := OKCoinTickerResponse{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	path := EncodeURLValues(o.APIUrl+"ticker.do", vals)
	err := SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return OKCoinTicker{}, err
	}
	return resp.Ticker, nil
}

func (o *OKCoin) GetOrderBook(symbol string, size int64, merge bool) (OKCoinOrderbook, error) {
	resp := OKCoinOrderbook{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if merge {
		vals.Set("merge", "1")
	}

	path := EncodeURLValues(o.APIUrl+"depth.do", vals)
	err := SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

type OKCoinTrades struct {
	Amount  float64 `json:"amount,string"`
	Date    int64   `json:"date`
	DateMS  int64   `json:"date_ms"`
	Price   float64 `json:"price,string"`
	TradeID int64   `json:"tid"`
	Type    string  `json:"type"`
}

func (o *OKCoin) GetTrades(symbol string, since int64) ([]OKCoinTrades, error) {
	result := []OKCoinTrades{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	if since != 0 {
		vals.Set("since", strconv.FormatInt(since, 10))
	}

	path := EncodeURLValues(o.APIUrl+"trades.do", vals)
	err := SendHTTPGetRequest(path, true, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (o *OKCoin) GetKline(symbol, klineType string, size, since int64) ([]interface{}, error) {
	resp := []interface{}{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("type", klineType)

	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}

	if since != 0 {
		vals.Set("since", strconv.FormatInt(since, 10))
	}

	path := EncodeURLValues(o.APIUrl+"kline.do", vals)
	err := SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (o *OKCoin) GetFuturesTicker(symbol, contractType string) (OKCoinFuturesTicker, error) {
	resp := OKCoinFuturesTickerResponse{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)
	path := EncodeURLValues(o.APIUrl+"future_ticker.do", vals)
	err := SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return OKCoinFuturesTicker{}, err
	}
	return resp.Ticker, nil
}

func (o *OKCoin) GetFuturesDepth(symbol, contractType string, size int64, merge bool) (OKCoinOrderbook, error) {
	result := OKCoinOrderbook{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if merge {
		vals.Set("merge", "1")
	}

	path := EncodeURLValues(o.APIUrl+"future_depth.do", vals)
	err := SendHTTPGetRequest(path, true, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

type OKCoinFuturesTrades struct {
	Amount  float64 `json:"amount"`
	Date    int64   `json:"date"`
	DateMS  int64   `json:"date_ms"`
	Price   float64 `json:"price"`
	TradeID int64   `json:"tid"`
	Type    string  `json:"type"`
}

func (o *OKCoin) GetFuturesTrades(symbol, contractType string) ([]OKCoinFuturesTrades, error) {
	result := []OKCoinFuturesTrades{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	path := EncodeURLValues(o.APIUrl+"future_trades.do", vals)
	err := SendHTTPGetRequest(path, true, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (o *OKCoin) GetFuturesIndex(symbol string) (float64, error) {
	type Response struct {
		Index float64 `json:"future_index"`
	}

	result := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)

	path := EncodeURLValues(o.APIUrl+"future_index.do", vals)
	err := SendHTTPGetRequest(path, true, &result)
	if err != nil {
		return 0, err
	}
	return result.Index, nil
}

func (o *OKCoin) GetFuturesExchangeRate() (float64, error) {
	type Response struct {
		Rate float64 `json:"rate"`
	}

	result := Response{}
	err := SendHTTPGetRequest(o.APIUrl+"exchange_rate.do", true, &result)
	if err != nil {
		return result.Rate, err
	}
	return result.Rate, nil
}

func (o *OKCoin) GetFuturesEstimatedPrice(symbol string) (float64, error) {
	type Response struct {
		Price float64 `json:"forecast_price"`
	}

	result := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	path := EncodeURLValues(o.APIUrl+"future_estimated_price.do", vals)
	err := SendHTTPGetRequest(path, true, &result)
	if err != nil {
		return result.Price, err
	}
	return result.Price, nil
}

func (o *OKCoin) GetFuturesKline(symbol, klineType, contractType string, size, since int64) ([]interface{}, error) {
	resp := []interface{}{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("type", klineType)
	vals.Set("contract_type", contractType)

	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if since != 0 {
		vals.Set("since", strconv.FormatInt(since, 10))
	}

	path := EncodeURLValues(o.APIUrl+"future_kline.do", vals)
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (o *OKCoin) GetFuturesHoldAmount(symbol, contractType string) ([]OKCoinFuturesHoldAmount, error) {
	resp := []OKCoinFuturesHoldAmount{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	path := EncodeURLValues(o.APIUrl+"future_hold_amount.do", vals)
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
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

type OKCoinUserInfo struct {
	Info struct {
		Funds struct {
			Asset struct {
				Net   float64 `json:"net,string"`
				Total float64 `json:"total,string"`
			} `json:"asset"`
			Borrow struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
			} `json:"borrow"`
			Free struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
			} `json:"free"`
			Freezed struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
			} `json:"freezed"`
			UnionFund struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
			} `json:"union_fund"`
		} `json:"funds"`
	} `json:"info"`
	Result bool `json:"result"`
}

func (o *OKCoin) GetUserInfo() (OKCoinUserInfo, error) {
	result := OKCoinUserInfo{}
	err := o.SendAuthenticatedHTTPRequest("userinfo.do", url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (o *OKCoin) Trade(amount, price float64, symbol, orderType string) (int64, error) {
	type Response struct {
		Result  bool  `json:"result"`
		OrderID int64 `json:"order_id"`
	}
	v := url.Values{}
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	v.Set("symbol", symbol)
	v.Set("type", orderType)

	result := Response{}

	err := o.SendAuthenticatedHTTPRequest("trade.do", v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("Unable to place order.")
	}

	return result.OrderID, nil
}

func (o *OKCoin) GetTradeHistory(symbol string, TradeID int64) ([]OKCoinTrades, error) {
	result := []OKCoinTrades{}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("since", strconv.FormatInt(TradeID, 10))

	err := o.SendAuthenticatedHTTPRequest("trade_history.do", v, &result)

	if err != nil {
		return nil, err
	}

	return result, nil
}

type OKCoinBatchTrade struct {
	OrderInfo []struct {
		OrderID   int64 `json:"order_id"`
		ErrorCode int64 `json:"error_code"`
	} `json:"order_info"`
	Result bool `json:"result"`
}

func (o *OKCoin) BatchTrade(orderData string, symbol, orderType string) (OKCoinBatchTrade, error) {
	v := url.Values{}
	v.Set("orders_data", orderData)
	v.Set("symbol", symbol)
	v.Set("type", orderType)
	result := OKCoinBatchTrade{}

	err := o.SendAuthenticatedHTTPRequest("batch_trade.do", v, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type OKCoinCancelOrderResponse struct {
	Success string
	Error   string
}

func (o *OKCoin) CancelOrder(orderID []int64, symbol string) (OKCoinCancelOrderResponse, error) {
	v := url.Values{}
	orders := []string{}
	orderStr := ""
	result := OKCoinCancelOrderResponse{}

	if len(orderID) > 1 {
		for x := range orderID {
			orders = append(orders, strconv.FormatInt(orderID[x], 10))
		}
		orderStr = JoinStrings(orders, ",")
	} else {
		orderStr = strconv.FormatInt(orderID[0], 10)
	}

	v.Set("order_id", orderStr)
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("cancel_order.do", v, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type OKCoinOrderInfo struct {
	Amount     float64 `json:"amount"`
	AvgPrice   float64 `json:"avg_price"`
	Created    int64   `json:"create_date"`
	DealAmount float64 `json:"deal_amount"`
	OrderID    int64   `json:"order_id"`
	OrdersID   int64   `json:"orders_id"`
	Price      float64 `json:"price"`
	Status     int     `json:"status"`
	Symbol     string  `json:"symbol"`
	Type       string  `json:"type"`
}

func (o *OKCoin) GetOrderInfo(orderID int64, symbol string) ([]OKCoinOrderInfo, error) {
	type Response struct {
		Result bool              `json:"result"`
		Orders []OKCoinOrderInfo `json:"orders"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest("order_info.do", v, &result)

	if err != nil {
		return nil, err
	}

	if result.Result != true {
		return nil, errors.New("Unable to retrieve order info.")
	}

	return result.Orders, nil
}

func (o *OKCoin) GetOrderInfoBatch(orderID []int64, symbol string) ([]OKCoinOrderInfo, error) {
	type Response struct {
		Result bool              `json:"result"`
		Orders []OKCoinOrderInfo `json:"orders"`
	}

	orders := []string{}
	for x := range orderID {
		orders = append(orders, strconv.FormatInt(orderID[x], 10))
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", JoinStrings(orders, ","))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest("orders_info.do", v, &result)

	if err != nil {
		return nil, err
	}

	if result.Result != true {
		return nil, errors.New("Unable to retrieve order info.")
	}

	return result.Orders, nil
}

type OKCoinOrderHistory struct {
	CurrentPage int               `json:"current_page"`
	Orders      []OKCoinOrderInfo `json:"orders"`
	PageLength  int               `json:"page_length"`
	Result      bool              `json:"result"`
	Total       int               `json:"total"`
}

func (o *OKCoin) GetOrderHistory(pageLength, currentPage int64, status, symbol string) (OKCoinOrderHistory, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("status", status)
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))
	result := OKCoinOrderHistory{}

	err := o.SendAuthenticatedHTTPRequest("order_history.do", v, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type OKCoinWithdrawalResponse struct {
	WithdrawID int  `json:"withdraw_id"`
	Result     bool `json:"result"`
}

func (o *OKCoin) Withdrawal(symbol string, fee float64, tradePWD, address string, amount float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)

	if fee != 0 {
		v.Set("chargefee", strconv.FormatFloat(fee, 'f', -1, 64))
	}
	v.Set("trade_pwd", tradePWD)
	v.Set("withdraw_address", address)
	v.Set("withdraw_amount", strconv.FormatFloat(amount, 'f', -1, 64))
	result := OKCoinWithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest("withdraw.do", v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("Unable to process withdrawal request.")
	}

	return result.WithdrawID, nil
}

func (o *OKCoin) CancelWithdrawal(symbol string, withdrawalID int64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("withdrawal_id", strconv.FormatInt(withdrawalID, 10))
	result := OKCoinWithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest("cancel_withdraw.do", v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("Unable to process withdrawal cancel request.")
	}

	return result.WithdrawID, nil
}

type OKCoinWithdrawInfo struct {
	Address    string  `json:"address"`
	Amount     float64 `json:"amount"`
	Created    int64   `json:"created_date"`
	ChargeFee  float64 `json:"chargefee"`
	Status     int     `json:"status"`
	WithdrawID int64   `json:"withdraw_id"`
}

func (o *OKCoin) GetWithdrawalInfo(symbol string, withdrawalID int64) ([]OKCoinWithdrawInfo, error) {
	type Response struct {
		Result   bool
		Withdraw []OKCoinWithdrawInfo `json:"withdraw"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("withdrawal_id", strconv.FormatInt(withdrawalID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest("withdraw_info.do", v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("Unable to process withdrawal cancel request.")
	}

	return result.Withdraw, nil
}

type OKCoinOrderFeeInfo struct {
	Fee     float64 `json:"fee,string"`
	OrderID int64   `json:"order_id"`
	Type    string  `json:"type"`
}

func (o *OKCoin) GetOrderFeeInfo(symbol string, orderID int64) (OKCoinOrderFeeInfo, error) {
	type Response struct {
		Data   OKCoinOrderFeeInfo `json:"data"`
		Result bool               `json:"result"`
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest("order_fee.do", v, &result)

	if err != nil {
		return result.Data, err
	}

	if !result.Result {
		return result.Data, errors.New("Unable to get order fee info.")
	}

	return result.Data, nil
}

type OKCoinLendDepth struct {
	Amount float64 `json:"amount"`
	Days   string  `json:"days"`
	Num    int64   `json:"num"`
	Rate   float64 `json:"rate,string"`
}

func (o *OKCoin) GetLendDepth(symbol string) ([]OKCoinLendDepth, error) {
	type Response struct {
		LendDepth []OKCoinLendDepth `json:"lend_depth"`
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest("lend_depth.do", v, &result)

	if err != nil {
		return nil, err
	}

	return result.LendDepth, nil
}

func (o *OKCoin) GetBorrowInfo(symbol string) (OKCoinBorrowInfo, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	result := OKCoinBorrowInfo{}

	err := o.SendAuthenticatedHTTPRequest("borrows_info.do", v, &result)

	if err != nil {
		return result, nil
	}

	return result, nil
}

type OKCoinBorrowResponse struct {
	Result   bool `json:"result"`
	BorrowID int  `json:"borrow_id"`
}

func (o *OKCoin) Borrow(symbol, days string, amount, rate float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("days", days)
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	result := OKCoinBorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest("borrow_money.do", v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("Unable to borrow.")
	}

	return result.BorrowID, nil
}

func (o *OKCoin) CancelBorrow(symbol string, borrowID int64) (bool, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := OKCoinBorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest("cancel_borrow.do", v, &result)

	if err != nil {
		return false, err
	}

	if !result.Result {
		return false, errors.New("Unable to cancel borrow.")
	}

	return true, nil
}

func (o *OKCoin) GetBorrowOrderInfo(borrowID int64) (OKCoinBorrowInfo, error) {
	type Response struct {
		Result      bool             `json:"result"`
		BorrowOrder OKCoinBorrowInfo `json:"borrow_order"`
	}

	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := Response{}
	err := o.SendAuthenticatedHTTPRequest("borrow_order_info.do", v, &result)

	if err != nil {
		return result.BorrowOrder, err
	}

	if !result.Result {
		return result.BorrowOrder, errors.New("Unable to get borrow info.")
	}

	return result.BorrowOrder, nil
}

func (o *OKCoin) GetRepaymentInfo(borrowID int64) (bool, error) {
	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := OKCoinBorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest("repayment.do", v, &result)

	if err != nil {
		return false, err
	}

	if !result.Result {
		return false, errors.New("Unable to get repayment info.")
	}

	return true, nil
}

func (o *OKCoin) GetUnrepaymentsInfo(symbol string, currentPage, pageLength int) ([]OKCoinBorrowOrder, error) {
	type Response struct {
		Unrepayments []OKCoinBorrowOrder `json:"unrepayments"`
		Result       bool                `json:"result"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))
	result := Response{}
	err := o.SendAuthenticatedHTTPRequest("unrepayments_info.do", v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("Unable to get unrepayments info.")
	}

	return result.Unrepayments, nil
}

func (o *OKCoin) GetAccountRecords(symbol string, recType, currentPage, pageLength int) ([]OKCoinAccountRecords, error) {
	type Response struct {
		Records []OKCoinAccountRecords `json:"records"`
		Symbol  string                 `json:"symbol"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("type", strconv.Itoa(recType))
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest("account_records.do", v, &result)

	if err != nil {
		return nil, err
	}

	return result.Records, nil
}

func (o *OKCoin) GetFuturesUserInfo() {
	err := o.SendAuthenticatedHTTPRequest("future_userinfo.do", url.Values{}, nil)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesPosition(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	err := o.SendAuthenticatedHTTPRequest("future_userinfo.do", v, nil)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) FuturesTrade(amount, price float64, matchPrice, leverage int64, symbol, contractType, orderType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("type", orderType)
	v.Set("match_price", strconv.FormatInt(matchPrice, 10))
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest("future_trade.do", v, nil)

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

	err := o.SendAuthenticatedHTTPRequest("future_batch_trade.do", v, nil)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) CancelFuturesOrder(orderID int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := o.SendAuthenticatedHTTPRequest("future_cancel.do", v, nil)

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

	err := o.SendAuthenticatedHTTPRequest("future_order_info.do", v, nil)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFutureOrdersInfo(orderID int64, contractType, symbol string) {
	v := url.Values{}
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("contract_type", contractType)
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("future_orders_info.do", v, nil)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesUserInfo4Fix() {
	v := url.Values{}

	err := o.SendAuthenticatedHTTPRequest("future_userinfo_4fix.do", v, nil)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) GetFuturesUserPosition4Fix(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("type", strconv.FormatInt(1, 10))

	err := o.SendAuthenticatedHTTPRequest("future_position_4fix.do", v, nil)

	if err != nil {
		log.Println(err)
	}
}

func (o *OKCoin) SendAuthenticatedHTTPRequest(method string, v url.Values, result interface{}) (err error) {
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

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
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
