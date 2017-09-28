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
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	alphapointDefaultAPIURL    = "https://sim3.alphapoint.com:8400"
	alphapointAPIVersion       = "1"
	alphapointTicker           = "GetTicker"
	alphapointTrades           = "GetTrades"
	alphapointTradesByDate     = "GetTradesByDate"
	alphapointOrderbook        = "GetOrderBook"
	alphapointProductPairs     = "GetProductPairs"
	alphapointProducts         = "GetProducts"
	alphapointCreateAccount    = "CreateAccount"
	alphapointUserInfo         = "GetUserInfo"
	alphapointAccountInfo      = "GetAccountInfo"
	alphapointAccountTrades    = "GetAccountTrades"
	alphapointDepositAddresses = "GetDepositAddresses"
	alphapointWithdraw         = "Withdraw"
	alphapointCreateOrder      = "CreateOrder"
	alphapointModifyOrder      = "ModifyOrder"
	alphapointCancelOrder      = "CancelOrder"
	alphapointCancelAllOrders  = "CancelAllOrders"
	alphapointOpenOrders       = "GetAccountOpenOrders"
	alphapointOrderFee         = "GetOrderFee"

	// Anymore and you get IP banned
	alphapointMaxRequestsPer10minutes = 500
)

// Alphapoint is the overarching type across the alphapoint package
type Alphapoint struct {
	exchange.Base
	WebsocketConn *websocket.Conn
}

// SetDefaults sets current default settings
func (a *Alphapoint) SetDefaults() {
	a.APIUrl = alphapointDefaultAPIURL
	a.WebsocketURL = alphapointDefaultWebsocketURL
	a.AssetTypes = []string{ticker.Spot}
}

// GetTicker returns current ticker information from Alphapoint for a selected
// currency pair ie "BTCUSD"
func (a *Alphapoint) GetTicker(currencyPair string) (Ticker, error) {
	request := make(map[string]interface{})
	request["productPair"] = currencyPair
	response := Ticker{}

	err := a.SendRequest("POST", alphapointTicker, request, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetTrades fetches past trades for the given currency pair
// currencyPair: ie "BTCUSD"
// StartIndex: specifies the index to begin from, -1 being the first trade on
// AlphaPoint Exchange. To begin from the most recent trade, set startIndex to
// 0 (default: 0)
// Count: specifies the number of trades to return (default: 10)
func (a *Alphapoint) GetTrades(currencyPair string, startIndex, count int) (Trades, error) {
	request := make(map[string]interface{})
	request["ins"] = currencyPair
	request["startIndex"] = startIndex
	request["Count"] = count
	response := Trades{}

	err := a.SendRequest("POST", alphapointTrades, request, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetTradesByDate gets trades by date
// CurrencyPair - instrument code (ex: “BTCUSD”)
// StartDate - specifies the starting time in epoch time, type is long
// EndDate - specifies the end time in epoch time, type is long
func (a *Alphapoint) GetTradesByDate(currencyPair string, startDate, endDate int64) (Trades, error) {
	request := make(map[string]interface{})
	request["ins"] = currencyPair
	request["startDate"] = startDate
	request["endDate"] = endDate
	response := Trades{}

	err := a.SendRequest("POST", alphapointTradesByDate, request, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetOrderbook fetches the current orderbook for a given currency pair
// CurrencyPair - trade pair (ex: “BTCUSD”)
func (a *Alphapoint) GetOrderbook(currencyPair string) (Orderbook, error) {
	request := make(map[string]interface{})
	request["productPair"] = currencyPair
	response := Orderbook{}

	err := a.SendRequest("POST", alphapointOrderbook, request, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetProductPairs gets the currency pairs currently traded on alphapoint
func (a *Alphapoint) GetProductPairs() (ProductPairs, error) {
	response := ProductPairs{}

	err := a.SendRequest("POST", alphapointProductPairs, nil, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetProducts gets the currency products currently supported on alphapoint
func (a *Alphapoint) GetProducts() (Products, error) {
	response := Products{}

	err := a.SendRequest("POST", alphapointProducts, nil, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// CreateAccount creates a new account on alphapoint
// FirstName - First name
// LastName - Last name
// Email - Email address
// Phone - Phone number (ex: “+12223334444”)
// Password - Minimum 8 characters
func (a *Alphapoint) CreateAccount(firstName, lastName, email, phone, password string) error {
	if len(password) < 8 {
		return errors.New(
			"alphapoint Error - Create account - Password must be 8 characters or more",
		)
	}

	request := make(map[string]interface{})
	request["firstname"] = firstName
	request["lastname"] = lastName
	request["email"] = email
	request["phone"] = phone
	request["password"] = password
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest("POST", alphapointCreateAccount, request, &response)
	if err != nil {
		log.Println(err)
	}
	if !response.IsAccepted {
		return errors.New(response.RejectReason)
	}
	return nil
}

// GetUserInfo returns current account user information
func (a *Alphapoint) GetUserInfo() (UserInfo, error) {
	response := UserInfo{}

	err := a.SendAuthenticatedHTTPRequest("POST", alphapointUserInfo, map[string]interface{}{}, &response)
	if err != nil {
		return UserInfo{}, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// SetUserInfo changes user name and/or 2FA settings
// userInfoKVP - An array of key value pairs
// FirstName - First name
// LastName - Last name
// UseAuthy2FA - “true” or “false” toggle Authy app
// Cell2FACountryCode - Cell country code (ex: 1), required for Authentication
// Cell2FAValue - Cell phone number, required for Authentication
// Use2FAForWithdraw - “true” or “false” set to true for using 2FA for
// withdrawals
func (a *Alphapoint) SetUserInfo(firstName, lastName, cell2FACountryCode, cell2FAValue string, useAuthy2FA, use2FAForWithdraw bool) (UserInfoSet, error) {
	response := UserInfoSet{}

	var userInfoKVPs = []UserInfoKVP{
		UserInfoKVP{
			Key:   "FirstName",
			Value: firstName,
		},
		UserInfoKVP{
			Key:   "LastName",
			Value: lastName,
		},
		UserInfoKVP{
			Key:   "Cell2FACountryCode",
			Value: cell2FACountryCode,
		},
		UserInfoKVP{
			Key:   "Cell2FAValue",
			Value: cell2FAValue,
		},
		UserInfoKVP{
			Key:   "UseAuthy2FA",
			Value: strconv.FormatBool(useAuthy2FA),
		},
		UserInfoKVP{
			Key:   "Use2FAForWithdraw",
			Value: strconv.FormatBool(use2FAForWithdraw),
		},
	}

	request := make(map[string]interface{})
	request["userInfoKVP"] = userInfoKVPs

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointUserInfo,
		request,
		&response,
	)
	if err != nil {
		return response, err
	}
	if response.IsAccepted != "true" {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetAccountInfo returns account info
func (a *Alphapoint) GetAccountInfo() (AccountInfo, error) {
	response := AccountInfo{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointAccountInfo,
		map[string]interface{}{},
		&response,
	)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetAccountTrades returns the trades executed on the account.
// CurrencyPair - Instrument code (ex: “BTCUSD”)
// StartIndex - Starting index, if less than 0 then start from the beginning
// Count - Returns last trade, (Default: 30)
func (a *Alphapoint) GetAccountTrades(currencyPair string, startIndex, count int) (Trades, error) {
	request := make(map[string]interface{})
	request["ins"] = currencyPair
	request["startIndex"] = startIndex
	request["count"] = count
	response := Trades{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointAccountTrades,
		request,
		&response,
	)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetDepositAddresses generates a deposit address
func (a *Alphapoint) GetDepositAddresses() ([]DepositAddresses, error) {
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest("POST", alphapointDepositAddresses,
		map[string]interface{}{}, &response,
	)
	if err != nil {
		return nil, err
	}
	if !response.IsAccepted {
		return nil, errors.New(response.RejectReason)
	}
	return response.Addresses, nil
}

// WithdrawCoins withdraws a coin to a specific address
// symbol - Instrument name (ex: “BTCUSD”)
// product - Currency name (ex: “BTC”)
// amount - Amount (ex: “.011”)
// address - Withdraw address
func (a *Alphapoint) WithdrawCoins(symbol, product, address string, amount float64) error {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["product"] = product
	request["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	request["sendToAddress"] = address

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointWithdraw,
		request,
		&response,
	)
	if err != nil {
		return err
	}
	if !response.IsAccepted {
		return errors.New(response.RejectReason)
	}
	return nil
}

// CreateOrder creates a market or limit order
// symbol - Instrument code (ex: “BTCUSD”)
// side - “buy” or “sell”
// orderType - “1” for market orders, “0” for limit orders
// quantity - Quantity
// price - Price in USD
func (a *Alphapoint) CreateOrder(symbol, side string, orderType int, quantity, price float64) (int64, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["side"] = side
	request["orderType"] = orderType
	request["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	request["px"] = strconv.FormatFloat(price, 'f', -1, 64)
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointCreateOrder,
		request,
		&response,
	)
	if err != nil {
		return 0, err
	}
	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.ServerOrderID, nil
}

// ModifyOrder modifies and existing Order
// OrderId - tracked order id number
// symbol - Instrument code (ex: “BTCUSD”)
// modifyAction - “0” or “1”
// “0” means "Move to top", which will modify the order price to the top of the
// book. A buy order will be modified to the highest bid and a sell order will
// be modified to the lowest ask price. “1” means "Execute now", which will
// convert a limit order into a market order.
func (a *Alphapoint) ModifyOrder(symbol string, OrderID, action int64) (int64, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["serverOrderId"] = OrderID
	request["modifyAction"] = action
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointModifyOrder,
		request,
		&response,
	)
	if err != nil {
		return 0, err
	}
	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.ModifyOrderID, nil
}

// CancelOrder cancels an order that has not been executed.
// symbol - Instrument code (ex: “BTCUSD”)
// OrderId - Order id (ex: 1000)
func (a *Alphapoint) CancelOrder(symbol string, OrderID int64) (int64, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["serverOrderId"] = OrderID
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointCancelOrder,
		request,
		&response,
	)
	if err != nil {
		return 0, err
	}
	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.CancelOrderID, nil
}

// CancelAllOrders cancels all open orders by symbol
// symbol - Instrument code (ex: “BTCUSD”)
func (a *Alphapoint) CancelAllOrders(symbol string) error {
	request := make(map[string]interface{})
	request["ins"] = symbol
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointCancelAllOrders,
		request,
		&response,
	)
	if err != nil {
		return err
	}
	if !response.IsAccepted {
		return errors.New(response.RejectReason)
	}
	return nil
}

// GetOrders returns all current open orders
func (a *Alphapoint) GetOrders() ([]OpenOrders, error) {
	response := OrderInfo{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointOpenOrders,
		map[string]interface{}{},
		&response,
	)
	if err != nil {
		return nil, err
	}
	if !response.IsAccepted {
		return nil, errors.New(response.RejectReason)
	}
	return response.OpenOrders, nil
}

// GetOrderFee returns a fee associated with an order
// symbol - Instrument code (ex: “BTCUSD”)
// side - “buy” or “sell”
// quantity - Quantity
// price - Price in USD
func (a *Alphapoint) GetOrderFee(symbol, side string, quantity, price float64) (float64, error) {
	request := make(map[string]interface{})
	request["ins"] = symbol
	request["side"] = side
	request["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	request["px"] = strconv.FormatFloat(price, 'f', -1, 64)
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(
		"POST",
		alphapointOrderFee,
		request,
		&response,
	)
	if err != nil {
		return 0, err
	}
	if !response.IsAccepted {
		return 0, errors.New(response.RejectReason)
	}
	return response.Fee, nil
}

// SendRequest sends an unauthenticated request
func (a *Alphapoint) SendRequest(method, path string, data map[string]interface{}, result interface{}) error {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	path = fmt.Sprintf("%s/ajax/v%s/%s", a.APIUrl, alphapointAPIVersion, path)

	PayloadJSON, err := common.JSONEncode(data)
	if err != nil {
		return errors.New("SendHTTPRequest: Unable to JSON request")
	}

	resp, err := common.SendHTTPRequest(
		method,
		path,
		headers,
		bytes.NewBuffer(PayloadJSON),
	)
	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}
	return nil
}

// SendAuthenticatedHTTPRequest sends an authenticated request
func (a *Alphapoint) SendAuthenticatedHTTPRequest(method, path string, data map[string]interface{}, result interface{}) error {
	if !a.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, a.Name)
	}

	if a.Nonce.Get() == 0 {
		a.Nonce.Set(time.Now().UnixNano())
	} else {
		a.Nonce.Inc()
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	data["apiKey"] = a.APIKey
	data["apiNonce"] = a.Nonce.Get()
	hmac := common.GetHMAC(common.HashSHA256, []byte(a.Nonce.String()+a.ClientID+a.APIKey), []byte(a.APISecret))
	data["apiSig"] = common.StringToUpper(common.HexEncodeToString(hmac))
	path = fmt.Sprintf("%s/ajax/v%s/%s", a.APIUrl, alphapointAPIVersion, path)

	PayloadJSON, err := common.JSONEncode(data)
	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	resp, err := common.SendHTTPRequest(
		method, path, headers, bytes.NewBuffer(PayloadJSON),
	)
	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}
	return nil
}
