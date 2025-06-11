package alphapoint

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
)

// Alphapoint is the overarching type across the alphapoint package
type Alphapoint struct {
	exchange.Base
	WebsocketConn *gws.Conn
}

// GetTicker returns current ticker information from Alphapoint for a selected
// currency pair ie "BTCUSD"
func (a *Alphapoint) GetTicker(ctx context.Context, currencyPair string) (Ticker, error) {
	req := make(map[string]any)
	req["productPair"] = currencyPair
	response := Ticker{}

	err := a.SendHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointTicker,
		req,
		&response)
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
func (a *Alphapoint) GetTrades(ctx context.Context, currencyPair string, startIndex, count int) (Trades, error) {
	req := make(map[string]any)
	req["ins"] = currencyPair
	req["startIndex"] = startIndex
	req["Count"] = count
	response := Trades{}

	err := a.SendHTTPRequest(ctx,
		exchange.RestSpot, http.MethodPost, alphapointTrades, req, &response)
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
func (a *Alphapoint) GetOrderbook(ctx context.Context, currencyPair string) (Orderbook, error) {
	req := make(map[string]any)
	req["productPair"] = currencyPair
	response := Orderbook{}

	err := a.SendHTTPRequest(ctx,
		exchange.RestSpot, http.MethodPost, alphapointOrderbook, req, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetProductPairs gets the currency pairs currently traded on alphapoint
func (a *Alphapoint) GetProductPairs(ctx context.Context) (ProductPairs, error) {
	response := ProductPairs{}

	err := a.SendHTTPRequest(ctx,
		exchange.RestSpot, http.MethodPost, alphapointProductPairs, nil, &response)
	if err != nil {
		return response, err
	}
	if !response.IsAccepted {
		return response, errors.New(response.RejectReason)
	}
	return response, nil
}

// GetProducts gets the currency products currently supported on alphapoint
func (a *Alphapoint) GetProducts(ctx context.Context) (Products, error) {
	response := Products{}

	err := a.SendHTTPRequest(ctx,
		exchange.RestSpot, http.MethodPost, alphapointProducts, nil, &response)
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
func (a *Alphapoint) CreateAccount(ctx context.Context, firstName, lastName, email, phone, password string) error {
	if len(password) < 8 {
		return errors.New(
			"alphapoint Error - Create account - Password must be 8 characters or more",
		)
	}

	req := make(map[string]any)
	req["firstname"] = firstName
	req["lastname"] = lastName
	req["email"] = email
	req["phone"] = phone
	req["password"] = password
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, http.MethodPost, alphapointCreateAccount, req, &response)
	if err != nil {
		return fmt.Errorf("unable to create account. Reason: %s", err)
	}
	if !response.IsAccepted {
		return errors.New(response.RejectReason)
	}
	return nil
}

// GetUserInfo returns current account user information
func (a *Alphapoint) GetUserInfo(ctx context.Context) (UserInfo, error) {
	response := UserInfo{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointUserInfo,
		map[string]any{},
		&response)
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
func (a *Alphapoint) SetUserInfo(ctx context.Context, firstName, lastName, cell2FACountryCode, cell2FAValue string, useAuthy2FA, use2FAForWithdraw bool) (UserInfoSet, error) {
	response := UserInfoSet{}

	userInfoKVPs := []UserInfoKVP{
		{
			Key:   "FirstName",
			Value: firstName,
		},
		{
			Key:   "LastName",
			Value: lastName,
		},
		{
			Key:   "Cell2FACountryCode",
			Value: cell2FACountryCode,
		},
		{
			Key:   "Cell2FAValue",
			Value: cell2FAValue,
		},
		{
			Key:   "UseAuthy2FA",
			Value: strconv.FormatBool(useAuthy2FA),
		},
		{
			Key:   "Use2FAForWithdraw",
			Value: strconv.FormatBool(use2FAForWithdraw),
		},
	}

	req := make(map[string]any)
	req["userInfoKVP"] = userInfoKVPs

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointUserInfo,
		req,
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

// GetAccountInformation returns account info
func (a *Alphapoint) GetAccountInformation(ctx context.Context) (AccountInfo, error) {
	response := AccountInfo{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointAccountInfo,
		map[string]any{},
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
func (a *Alphapoint) GetAccountTrades(ctx context.Context, currencyPair string, startIndex, count int) (Trades, error) {
	req := make(map[string]any)
	req["ins"] = currencyPair
	req["startIndex"] = startIndex
	req["count"] = count
	response := Trades{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointAccountTrades,
		req,
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
func (a *Alphapoint) GetDepositAddresses(ctx context.Context) ([]DepositAddresses, error) {
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointDepositAddresses,
		map[string]any{},
		&response,
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
func (a *Alphapoint) WithdrawCoins(ctx context.Context, symbol, product, address string, amount float64) error {
	req := make(map[string]any)
	req["ins"] = symbol
	req["product"] = product
	req["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	req["sendToAddress"] = address

	response := Response{}
	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointWithdraw,
		req,
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

func (a *Alphapoint) convertOrderTypeToOrderTypeNumber(orderType string) (orderTypeNumber int64) {
	if orderType == order.Market.String() {
		orderTypeNumber = 1
	}

	return orderTypeNumber
}

// CreateOrder creates a market or limit order
// symbol - Instrument code (ex: “BTCUSD”)
// side - “buy” or “sell”
// orderType - “1” for market orders, “0” for limit orders
// quantity - Quantity
// price - Price in USD
func (a *Alphapoint) CreateOrder(ctx context.Context, symbol, side, orderType string, quantity, price float64) (int64, error) {
	orderTypeNumber := a.convertOrderTypeToOrderTypeNumber(orderType)
	req := make(map[string]any)
	req["ins"] = symbol
	req["side"] = side
	req["orderType"] = orderTypeNumber
	req["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	req["px"] = strconv.FormatFloat(price, 'f', -1, 64)
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointCreateOrder,
		req,
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

// ModifyExistingOrder modifies and existing Order
// OrderID - tracked order id number
// symbol - Instrument code (ex: “BTCUSD”)
// modifyAction - “0” or “1”
// “0” means "Move to top", which will modify the order price to the top of the
// book. A buy order will be modified to the highest bid and a sell order will
// be modified to the lowest ask price. “1” means "Execute now", which will
// convert a limit order into a market order.
func (a *Alphapoint) ModifyExistingOrder(ctx context.Context, symbol string, orderID, action int64) (int64, error) {
	req := make(map[string]any)
	req["ins"] = symbol
	req["serverOrderId"] = orderID
	req["modifyAction"] = action
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointModifyOrder,
		req,
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

// CancelExistingOrder cancels an order that has not been executed.
// symbol - Instrument code (ex: “BTCUSD”)
// OrderID - Order id (ex: 1000)
func (a *Alphapoint) CancelExistingOrder(ctx context.Context, orderID int64, omsid string) (int64, error) {
	req := make(map[string]any)
	req["OrderId"] = orderID
	req["OMSId"] = omsid
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointCancelOrder,
		req,
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

// CancelAllExistingOrders cancels all open orders by symbol.
// symbol - Instrument code (ex: “BTCUSD”)
func (a *Alphapoint) CancelAllExistingOrders(ctx context.Context, omsid string) error {
	req := make(map[string]any)
	req["OMSId"] = omsid
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointCancelAllOrders,
		req,
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
func (a *Alphapoint) GetOrders(ctx context.Context) ([]OpenOrders, error) {
	response := OrderInfo{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointOpenOrders,
		map[string]any{},
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
func (a *Alphapoint) GetOrderFee(ctx context.Context, symbol, side string, quantity, price float64) (float64, error) {
	req := make(map[string]any)
	req["ins"] = symbol
	req["side"] = side
	req["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	req["px"] = strconv.FormatFloat(price, 'f', -1, 64)
	response := Response{}

	err := a.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		alphapointOrderFee,
		req,
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

// SendHTTPRequest sends an unauthenticated HTTP request
func (a *Alphapoint) SendHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, data map[string]any, result any) error {
	endpoint, err := a.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	path = fmt.Sprintf("%s/ajax/v%s/%s", endpoint, alphapointAPIVersion, path)

	PayloadJSON, err := json.Marshal(data)
	if err != nil {
		return errors.New("unable to JSON request")
	}

	item := &request.Item{
		Method:        method,
		Path:          path,
		Headers:       headers,
		Result:        result,
		Verbose:       a.Verbose,
		HTTPDebugging: a.HTTPDebugging,
		HTTPRecording: a.HTTPRecording,
	}

	return a.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		item.Body = bytes.NewBuffer(PayloadJSON)
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated request
func (a *Alphapoint) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, data map[string]any, result any) error {
	creds, err := a.GetCredentials(ctx)
	if err != nil {
		return err
	}

	endpoint, err := a.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	n := a.Requester.GetNonce(nonce.UnixNano)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	data["apiKey"] = creds.Key
	data["apiNonce"] = n

	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(n.String()+creds.ClientID+creds.Key),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}

	data["apiSig"] = strings.ToUpper(hex.EncodeToString(hmac))
	path = fmt.Sprintf("%s/ajax/v%s/%s", endpoint, alphapointAPIVersion, path)

	PayloadJSON, err := json.Marshal(data)
	if err != nil {
		return errors.New("unable to JSON request")
	}

	item := &request.Item{
		Method:        method,
		Path:          path,
		Headers:       headers,
		Result:        result,
		NonceEnabled:  true,
		Verbose:       a.Verbose,
		HTTPDebugging: a.HTTPDebugging,
		HTTPRecording: a.HTTPRecording,
	}

	return a.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		item.Body = bytes.NewBuffer(PayloadJSON)
		return item, nil
	}, request.AuthenticatedRequest)
}
