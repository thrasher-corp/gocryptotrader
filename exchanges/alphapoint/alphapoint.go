package alphapoint

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges"
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
	exchange.ExchangeBase
	WebsocketConn *websocket.Conn
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
	if len(password) < 8 {
		return errors.New("Alphapoint Error - Create account - Password must be 8 characters or more.")
	}

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
	hmac := common.GetHMAC(common.HASH_SHA256, []byte(nonceStr+a.ClientID+a.APIKey), []byte(a.APISecret))
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
