package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	ALPHAPOINT_DEFAULT_API_URL   = "https://sim3.alphapoint.com:8400"
	ALPHAPOINT_API_VERSION       = "1"
	ALPHAPOINT_TICKER            = "GetTicker"
	ALPHAPOINT_TRADES            = "GetTrades"
	ALPHAPOINT_TRADESBYDATE      = "GetTradesByDate"
	ALPHAPOINT_ORDERBOOK         = "GetOrderBook"
	ALPHAPOINT_PRODUCT_PAIRS     = "GetProductPairs"
	ALPHAPOINT_PRODUCTS          = "GetProducts"
	ALPHAPOINT_CREATE_ACCOUNT    = "CreateAccount"
	ALPHAPOINT_USERINFO          = "GetUserInfo"
	ALPHAPOINT_ACCOUNT_INFO      = "GetAccountInfo"
	ALPHAPOINT_ACCOUNT_TRADES    = "GetAccountTrades"
	ALPHAPOINT_DEPOSIT_ADDRESSES = "GetDepositAddresses"
	ALPHAPOINT_WITHDRAW          = "Withdraw"
	ALPHAPOINT_CREATE_ORDER      = "CreateOrder"
	ALPHAPOINT_MODIFY_ORDER      = "ModifyOrder"
	ALPHAPOINT_CANCEL_ORDER      = "CancelOrder"
	ALPHAPOINT_CANCEALLORDERS    = "CancelAllOrders"
	ALPHAPOINT_OPEN_ORDERS       = "GetAccountOpenOrders"
	ALPHAPOINT_ORDER_FEE         = "GetOrderFee"
)

type Alphapoint struct {
	WebsocketConn                     *websocket.Conn
	WebsocketURL                      string
	ExchangeName                      string
	ExchangeEnabled                   bool
	WebsocketEnabled                  bool
	Verbose                           bool
	APIUrl, APIKey, UserID, APISecret string
}

type AlphapointTrade struct {
	TID                   int64   `json:"tid"`
	Price                 float64 `json:"px"`
	Quantity              float64 `json:"qty"`
	Unixtime              int     `json:"unixtime"`
	UTCTicks              int64   `json:"utcticks"`
	IncomingOrderSide     int     `json:"incomingOrderSide"`
	IncomingServerOrderID int     `json:"incomingServerOrderId"`
	BookServerOrderID     int     `json:"bookServerOrderId"`
}

type AlphapointTrades struct {
	IsAccepted   bool              `json:"isAccepted"`
	RejectReason string            `json:"rejectReason"`
	DateTimeUTC  int64             `json:"dateTimeUtc"`
	Instrument   string            `json:"ins"`
	StartIndex   int               `json:"startIndex"`
	Count        int               `json:"count"`
	Trades       []AlphapointTrade `json:"trades"`
}

type AlphapointTradesByDate struct {
	IsAccepted   bool              `json:"isAccepted"`
	RejectReason string            `json:"rejectReason"`
	DateTimeUTC  int64             `json:"dateTimeUtc"`
	Instrument   string            `json:"ins"`
	StartDate    int64             `json:"startDate"`
	EndDate      int64             `json:"endDate"`
	Trades       []AlphapointTrade `json:"trades"`
}

type AlphapointTicker struct {
	High               float64 `json:"high"`
	Last               float64 `json:"last"`
	Bid                float64 `json:"bid"`
	Volume             float64 `json:"volume"`
	Low                float64 `json:"low"`
	Ask                float64 `json:"ask"`
	Total24HrQtyTraded float64 `json:"Total24HrQtyTraded"`
	Total24HrNumTrades float64 `json:"Total24HrNumTrades"`
	SellOrderCount     float64 `json:"sellOrderCount"`
	BuyOrderCount      float64 `json:"buyOrderCount"`
	NumOfCreateOrders  float64 `json:"numOfCreateOrders"`
	IsAccepted         bool    `json:"isAccepted"`
	RejectReason       string  `json:"rejectReason"`
}

type AlphapointOrderbookEntry struct {
	Quantity float64 `json:"qty"`
	Price    float64 `json:"px"`
}

type AlphapointOrderbook struct {
	Bids         []AlphapointOrderbookEntry `json:"bids"`
	Asks         []AlphapointOrderbookEntry `json:"asks"`
	IsAccepted   bool                       `json:"isAccepted"`
	RejectReason string                     `json:"rejectReason"`
}

type AlphapointProductPair struct {
	Name                  string `json:"name"`
	Productpaircode       int    `json:"productPairCode"`
	Product1Label         string `json:"product1Label"`
	Product1Decimalplaces int    `json:"product1DecimalPlaces"`
	Product2Label         string `json:"product2Label"`
	Product2Decimalplaces int    `json:"product2DecimalPlaces"`
}

type AlphapointProductPairs struct {
	ProductPairs []AlphapointProductPair `json:"productPairs"`
	IsAccepted   bool                    `json:"isAccepted"`
	RejectReason string                  `json:"rejectReason"`
}

type AlphapointProduct struct {
	Name          string `json:"name"`
	IsDigital     bool   `json:"isDigital"`
	ProductCode   int    `json:"productCode"`
	DecimalPlaces int    `json:"decimalPlaces"`
	FullName      string `json:"fullName"`
}

type AlphapointProducts struct {
	Products     []AlphapointProduct `json:"products"`
	IsAccepted   bool                `json:"isAccepted"`
	RejectReason string              `json:"rejectReason"`
}

type AlphapointUserInfo struct {
	UserInfoKVP []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"userInfoKVP"`
	IsAccepted   bool   `json:"isAccepted"`
	RejectReason string `json:"rejectReason"`
}

type AlphapointAccountInfo struct {
	Currencies []struct {
		Name    string `json:"name"`
		Balance int    `json:"balance"`
		Hold    int    `json:"hold"`
	} `json:"currencies"`
	ProductPairs []struct {
		ProductPairName string `json:"productPairName"`
		ProductPairCode int    `json:"productPairCode"`
		TradeCount      int    `json:"tradeCount"`
		TradeVolume     int    `json:"tradeVolume"`
	} `json:"productPairs"`
	IsAccepted   bool   `json:"isAccepted"`
	RejectReason string `json:"rejectReason"`
}

type AlphapointOrder struct {
	Serverorderid int   `json:"ServerOrderId"`
	AccountID     int   `json:"AccountId"`
	Price         int   `json:"Price"`
	QtyTotal      int   `json:"QtyTotal"`
	QtyRemaining  int   `json:"QtyRemaining"`
	ReceiveTime   int64 `json:"ReceiveTime"`
	Side          int   `json:"Side"`
}

type AlphapointOpenOrders struct {
	Instrument string            `json:"ins"`
	Openorders []AlphapointOrder `json:"openOrders"`
}

type AlphapointOrderInfo struct {
	OpenOrders   []AlphapointOpenOrders `json:"openOrdersInfo"`
	IsAccepted   bool                   `json:"isAccepted"`
	DateTimeUTC  int64                  `json:"dateTimeUtc"`
	RejectReason string                 `json:"rejectReason"`
}

type AlphapointDepositAddresses struct {
	Name           string `json:"name"`
	DepositAddress string `json:"depositAddress"`
}

func (a *Alphapoint) SetDefaults() {
	a.APIUrl = ALPHAPOINT_DEFAULT_API_URL
	a.WebsocketURL = ALPHAPOINT_DEFAULT_WEBSOCKET_URL
}

func (a *Alphapoint) GetTicker(symbol string) (AlphapointTicker, error) {
	request := make(map[string]interface{})
	request["productPair"] = symbol
	response := AlphapointTicker{}
	err := a.SendRequest("POST", ALPHAPOINT_TICKER, request, &response)

	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) GetTickerPrice(currency string) TickerPrice {
	var tickerPrice TickerPrice
	ticker, err := a.GetTicker(currency)
	if err != nil {
		log.Println(err)
		return TickerPrice{}
	}
	tickerPrice.Ask = ticker.Ask
	tickerPrice.Bid = ticker.Bid

	return tickerPrice
}

func (a *Alphapoint) GetTrades(symbol string, startIndex, count int) (AlphapointTrades, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["startIndex"] = startIndex
	request["Count"] = count
	response := AlphapointTrades{}
	err := a.SendRequest("POST", ALPHAPOINT_TRADES, request, &response)

	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) GetTradesByDate(symbol string, startDate, endDate int64) (AlphapointTradesByDate, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["startDate"] = startDate
	request["endDate"] = endDate
	response := AlphapointTradesByDate{}
	err := a.SendRequest("POST", ALPHAPOINT_TRADESBYDATE, request, &response)

	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) GetOrderbook(symbol string) (AlphapointOrderbook, error) {
	request := make(map[string]interface{})
	request["productPair"] = symbol
	response := AlphapointOrderbook{}
	err := a.SendRequest("POST", ALPHAPOINT_ORDERBOOK, request, &response)

	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) GetProductPairs() (AlphapointProductPairs, error) {
	response := AlphapointProductPairs{}
	err := a.SendRequest("POST", ALPHAPOINT_PRODUCT_PAIRS, nil, &response)

	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) GetProducts() (AlphapointProducts, error) {
	response := AlphapointProducts{}
	err := a.SendRequest("POST", ALPHAPOINT_PRODUCTS, nil, &response)

	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) CreateAccount(firstName, lastName, email, phone, password string) error {
	request := make(map[string]interface{})
	request["firstname"] = firstName
	request["lastname"] = lastName
	request["email"] = email
	request["phone"] = phone
	request["password"] = password

	type Response struct {
		IsAccepted   bool   `json:"isAccepted"`
		RejectReason string `json:"rejectReason"`
	}

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_CREATE_ACCOUNT, request, &response)

	if err != nil {
		log.Println(err)
	}

	if !response.IsAccepted {
		return errors.New(response.RejectReason)
	}

	return nil
}

func (a *Alphapoint) GetUserInfo() (AlphapointUserInfo, error) {
	response := AlphapointUserInfo{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_USERINFO, map[string]interface{}{}, &response)
	if err != nil {
		return AlphapointUserInfo{}, err
	}
	return response, nil
}

func (a *Alphapoint) GetAccountInfo() (AlphapointAccountInfo, error) {
	response := AlphapointAccountInfo{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_ACCOUNT_INFO, map[string]interface{}{}, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) GetName() string {
	return a.ExchangeName
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Alphapoint exchange
func (e *Alphapoint) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	account, err := e.GetAccountInfo()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(account.Currencies); i++ {
		var exchangeCurrency ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = account.Currencies[i].Name
		exchangeCurrency.TotalValue = float64(account.Currencies[i].Balance)
		exchangeCurrency.Hold = float64(account.Currencies[i].Hold)

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	//If it all works out
	return response, nil
}

func (a *Alphapoint) GetAccountTrades(symbol string, startIndex, count int) (AlphapointTrades, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["startIndex"] = startIndex
	request["count"] = count

	response := AlphapointTrades{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_ACCOUNT_TRADES, request, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

func (a *Alphapoint) GetDepositAddresses() ([]AlphapointDepositAddresses, error) {
	type Response struct {
		Addresses    []AlphapointDepositAddresses
		IsAccepted   bool   `json:"isAccepted"`
		RejectReason string `json:"rejectReason"`
	}

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_DEPOSIT_ADDRESSES, map[string]interface{}{}, &response)
	if err != nil {
		return nil, err
	}
	if !response.IsAccepted {
		return nil, errors.New(response.RejectReason)
	}
	return response.Addresses, nil
}

func (a *Alphapoint) WithdrawCoins(symbol, product string, amount float64, address string) error {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["product"] = product
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["sendToAddress"] = address

	type Response struct {
		IsAccepted   bool   `json:"isAccepted"`
		RejectReason string `json:"rejectReason"`
	}

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_WITHDRAW, request, &response)
	if err != nil {
		return err
	}

	if !response.IsAccepted {
		return errors.New(response.RejectReason)
	}
	return nil
}

func (a *Alphapoint) CreateOrder(symbol, side string, orderType int, quantity, price float64) (int64, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["side"] = side
	request["orderType"] = orderType
	request["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	request["px"] = strconv.FormatFloat(price, 'f', -1, 64)

	type Response struct {
		ServerOrderID int64   `json:"serverOrderId"`
		DateTimeUTC   float64 `json:"dateTimeUtc"`
		IsAccepted    bool    `json:"isAccepted"`
		RejectReason  string  `json:"rejectReason"`
	}

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_CREATE_ORDER, request, &response)
	if err != nil {
		return 0, err
	}

	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.ServerOrderID, nil
}

func (a *Alphapoint) ModifyOrder(symbol string, OrderID, action int64) (int64, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["serverOrderId"] = OrderID
	request["modifyAction"] = action

	type Response struct {
		ModifyOrderID int64   `json:"modifyOrderId"`
		ServerOrderID int64   `json:"serverOrderId"`
		DateTimeUTC   float64 `json:"dateTimeUtc"`
		IsAccepted    bool    `json:"isAccepted"`
		RejectReason  string  `json:"rejectReason"`
	}

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_MODIFY_ORDER, request, &response)
	if err != nil {
		return 0, err
	}

	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.ModifyOrderID, nil
}

func (a *Alphapoint) CancelOrder(symbol string, OrderID int64) (int64, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["serverOrderId"] = OrderID

	type Response struct {
		CancelOrderID int64   `json:"cancelOrderId"`
		ServerOrderID int64   `json:"serverOrderId"`
		DateTimeUTC   float64 `json:"dateTimeUtc"`
		IsAccepted    bool    `json:"isAccepted"`
		RejectReason  string  `json:"rejectReason"`
	}

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_CANCEL_ORDER, request, &response)
	if err != nil {
		return 0, err
	}

	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.CancelOrderID, nil
}

func (a *Alphapoint) CancelAllOrders(symbol string) error {
	request := make(map[string]interface{})
	request["ins"] = symbol

	type Response struct {
		IsAccepted   bool   `json:"isAccepted"`
		RejectReason string `json:"rejectReason"`
	}

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_CANCEALLORDERS, request, &response)
	if err != nil {
		return err
	}

	if !response.IsAccepted {
		return errors.New(response.RejectReason)
	}
	return nil
}

func (a *Alphapoint) GetOrders() ([]AlphapointOpenOrders, error) {
	response := AlphapointOrderInfo{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_OPEN_ORDERS, map[string]interface{}{}, &response)
	if err != nil {
		return nil, err
	}

	if !response.IsAccepted {
		return nil, errors.New(response.RejectReason)
	}
	return response.OpenOrders, nil
}

func (a *Alphapoint) GetOrderFee(symbol, side string, quantity, price float64) (float64, error) {
	type Response struct {
		IsAccepted   bool    `json:"isAccepted"`
		RejectReason string  `json:"rejectReason"`
		Fee          float64 `json:"fee"`
		FeeProduct   string  `json:"feeProduct"`
	}

	request := make(map[string]interface{})
	request["ins"] = symbol
	request["side"] = side
	request["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	request["px"] = strconv.FormatFloat(price, 'f', -1, 64)

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest("POST", ALPHAPOINT_ORDER_FEE, request, &response)
	if err != nil {
		return 0, err
	}

	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.Fee, nil
}

func (a *Alphapoint) SendRequest(method, path string, data map[string]interface{}, result interface{}) error {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	path = fmt.Sprintf("%s/ajax/v%s/%s", a.APIUrl, ALPHAPOINT_API_VERSION, path)
	PayloadJson, err := common.JSONEncode(data)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	resp, err := common.SendHTTPRequest(method, path, headers, bytes.NewBuffer(PayloadJson))

	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}
	return nil
}

func (a *Alphapoint) SendAuthenticatedHTTPRequest(method, path string, data map[string]interface{}, result interface{}) error {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	data["apiKey"] = a.APIKey
	nonce := time.Now().UnixNano()
	nonceStr := strconv.FormatInt(nonce, 10)
	data["apiNonce"] = nonce
	hmac := common.GetHMAC(common.HASH_SHA256, []byte(nonceStr+a.UserID+a.APIKey), []byte(a.APISecret))
	data["apiSig"] = common.StringToUpper(common.HexEncodeToString(hmac))
	path = fmt.Sprintf("%s/ajax/v%s/%s", a.APIUrl, ALPHAPOINT_API_VERSION, path)
	PayloadJson, err := common.JSONEncode(data)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	resp, err := common.SendHTTPRequest(method, path, headers, bytes.NewBuffer(PayloadJson))

	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}
	return nil
}
