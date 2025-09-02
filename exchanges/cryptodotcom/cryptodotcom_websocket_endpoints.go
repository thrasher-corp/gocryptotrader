package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// WsSetCancelOnDisconnect cancel on Disconnect is an optional feature that will cancel all open orders created by the connection upon loss of connectivity between client or server.
func (e *Exchange) WsSetCancelOnDisconnect(scope string) (*CancelOnDisconnectScope, error) {
	if scope != "ACCOUNT" && scope != "CONNECTION" {
		return nil, errInvalidOrderCancellationScope
	}
	params := make(map[string]any)
	params["scope"] = scope
	var resp *CancelOnDisconnectScope
	return resp, e.SendWebsocketRequest("private/set-cancel-on-disconnect", params, &resp, true)
}

// WsRetriveCancelOnDisconnect retrieves cancel-on-disconnect scope information.
func (e *Exchange) WsRetriveCancelOnDisconnect() (*CancelOnDisconnectScope, error) {
	var resp *CancelOnDisconnectScope
	return resp, e.SendWebsocketRequest("private/get-cancel-on-disconnect", nil, &resp, true)
}

// WsCreateWithdrawal creates a withdrawal request. Withdrawal setting must be enabled for your API Key. If you do not see the option when viewing your API Key, this feature is not yet available for you.
// Withdrawal addresses must first be whitelisted in your account’s Withdrawal Whitelist page.
// Withdrawal fees and minimum withdrawal amount can be found on the Fees & Limits page on the Exchange website.
func (e *Exchange) WsCreateWithdrawal(ccy currency.Code, amount float64, address, addressTag, networkID, clientWithdrawalID string) (*WithdrawalItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w, withdrawal amount provided: %f", order.ErrAmountIsInvalid, amount)
	}
	if address == "" {
		return nil, errAddressRequired
	}
	params := make(map[string]any)
	params["currency"] = ccy.String()
	params["amount"] = amount
	params["address"] = address
	if clientWithdrawalID != "" {
		params["client_wid"] = clientWithdrawalID
	}
	if addressTag != "" {
		params["address_tag"] = addressTag
	}
	if networkID != "" {
		params["network_id"] = networkID
	}
	var resp *WithdrawalItem
	return resp, e.SendWebsocketRequest(privateWithdrawal, params, &resp, true)
}

// WsRetriveWithdrawalHistory retrieves accounts withdrawal history through the websocket connection
func (e *Exchange) WsRetriveWithdrawalHistory() (*WithdrawalResponse, error) {
	var resp *WithdrawalResponse
	return resp, e.SendWebsocketRequest(privateGetWithdrawalHistory, nil, &resp, true)
}

// WsPlaceOrder created a new BUY or SELL order on the Exchange through the websocket connection.
func (e *Exchange) WsPlaceOrder(arg *OrderParam) (*CreateOrderResponse, error) {
	params, err := arg.getCreateParamMap()
	if err != nil {
		return nil, err
	}
	var resp *CreateOrderResponse
	return resp, e.SendWebsocketRequest(privateCreateOrder, params, &resp, true)
}

// WsCancelExistingOrder cancels and existing open order through the websocket connection
func (e *Exchange) WsCancelExistingOrder(symbol, orderID string) error {
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	params["order_id"] = orderID
	return e.SendWebsocketRequest(privateCancelOrder, params, nil, true)
}

// WsCreateOrderList create a list of orders on the Exchange.
// contingency_type must be LIST, for list of orders creation.
// This call is asynchronous, so the response is simply a confirmation of the request.
func (e *Exchange) WsCreateOrderList(contingencyType string, arg []OrderParam) (*OrderCreationResponse, error) {
	if len(arg) == 0 {
		return nil, common.ErrNilPointer
	}
	orderParams := make([]map[string]any, len(arg))
	for x := range arg {
		p, err := arg[x].getCreateParamMap()
		if err != nil {
			return nil, err
		}
		orderParams[x] = p
	}
	if contingencyType == "" {
		contingencyType = "LIST"
	}
	params := make(map[string]any)
	params["order_list"] = orderParams
	params["contingency_type"] = contingencyType
	var resp *OrderCreationResponse
	return resp, e.SendWebsocketRequest(privateCreateOrderList, params, &resp, true)
}

// WsCancelOrderList cancel a list of orders on the Exchange through the websocket connection.
func (e *Exchange) WsCancelOrderList(args []CancelOrderParam) (*CancelOrdersResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	cancelOrderList := []map[string]any{}
	for x := range args {
		if args[x].InstrumentName == "" && args[x].OrderID == "" {
			return nil, errInstrumentNameOrOrderIDRequired
		}
		result := make(map[string]any)
		if args[x].InstrumentName != "" {
			result["instrument_name"] = args[x].InstrumentName
		}
		if args[x].OrderID != "" {
			result["order_id"] = args[x].OrderID
		}
		cancelOrderList = append(cancelOrderList, result)
	}
	params := make(map[string]any)
	params["order_list"] = cancelOrderList
	var resp *CancelOrdersResponse
	return resp, e.SendWebsocketRequest(privateCancelOrderList, params, &resp, true)
}

// WsCancelAllPersonalOrders cancels all orders for a particular instrument/pair (asynchronous)
// This call is asynchronous, so the response is simply a confirmation of the request.
func (e *Exchange) WsCancelAllPersonalOrders(symbol string) error {
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	return e.SendWebsocketRequest(privateCancelAllOrders, params, nil, true)
}

// WsRetrivePersonalOrderHistory gets the order history for a particular instrument
//
// If paging is used, enumerate each page (starting with 0) until an empty order_list array appears in the response.
func (e *Exchange) WsRetrivePersonalOrderHistory(instrumentName string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalOrdersResponse, error) {
	params := make(map[string]any)
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
	}
	if !startTimestamp.IsZero() {
		params["start_ts"] = strconv.FormatInt(startTimestamp.UnixMilli(), 10)
	}
	if !endTimestamp.IsZero() {
		params["end_ts"] = strconv.FormatInt(endTimestamp.UnixMilli(), 10)
	}
	if pageSize > 0 {
		params["page_size"] = pageSize
	}
	if page > 0 {
		params["page"] = page
	}
	var resp *PersonalOrdersResponse
	return resp, e.SendWebsocketRequest(privateGetOrderHistory, params, &resp, true)
}

// WsRetrivePersonalOpenOrders retrieves all open orders of particular instrument through the websocket connection
func (e *Exchange) WsRetrivePersonalOpenOrders(instrumentName string, pageSize, page int64) (*PersonalOrdersResponse, error) {
	params := make(map[string]any)
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
	}
	if pageSize > 0 {
		params["page_size"] = pageSize
	}
	params["page"] = page
	var resp *PersonalOrdersResponse
	return resp, e.SendWebsocketRequest(privateGetOpenOrders, params, &resp, true)
}

// WsRetriveOrderDetail retrieves details on a particular order ID through the websocket connection
func (e *Exchange) WsRetriveOrderDetail(orderID string) (*OrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := make(map[string]any)
	params["order_id"] = orderID
	var resp *OrderDetail
	return resp, e.SendWebsocketRequest(privateGetOrderDetail, params, &resp, true)
}

// WsRetrivePrivateTrades gets all executed trades for a particular instrument.
//
// If paging is used, enumerate each page (starting with 0) until an empty trade_list array appears in the response.
// Users should use user.trade to keep track of real-time trades, and private/get-trades should primarily be used for recovery; typically when the websocket is disconnected.
func (e *Exchange) WsRetrivePrivateTrades(instrumentName string, startTimestamp, endTimestamp time.Time, pageSize, page int64) (*PersonalTrades, error) {
	params := make(map[string]any)
	if instrumentName != "" {
		params["instrument_name"] = instrumentName
	}
	if !startTimestamp.IsZero() {
		params["start_ts"] = strconv.FormatInt(startTimestamp.UnixMilli(), 10)
	}
	if !endTimestamp.IsZero() {
		params["end_ts"] = strconv.FormatInt(endTimestamp.UnixMilli(), 10)
	}
	if pageSize != 0 {
		params["page_size"] = pageSize
	}
	params["page"] = page
	var resp *PersonalTrades
	return resp, e.SendWebsocketRequest(privateGetTrades, params, &resp, true)
}

// WsRetriveAccountSummary returns the account balance of a user for a particular token through the websocket connection.
func (e *Exchange) WsRetriveAccountSummary(ccy currency.Code) (*Accounts, error) {
	params := make(map[string]any)
	if !ccy.IsEmpty() {
		params["currency"] = ccy.String()
	}
	var resp *Accounts
	return resp, e.SendWebsocketRequest(privateGetAccountSummary, params, &resp, true)
}

// SendWebsocketRequest pushed a request data through the websocket data for authenticated and public messages.
func (e *Exchange) SendWebsocketRequest(method string, arg map[string]any, result any, authenticated bool) error {
	if authenticated && !e.Websocket.CanUseAuthenticatedEndpoints() {
		return errors.New("can not send authenticated websocket request")
	}
	timestamp := time.Now()
	req := &WsRequestPayload{
		Method: method,
		Nonce:  timestamp.UnixMilli(),
		Params: arg,
	}
	var payload []byte
	var err error
	if authenticated {
		req.ID = e.Websocket.AuthConn.GenerateMessageID(false)
		payload, err = e.Websocket.AuthConn.SendMessageReturnResponse(context.Background(), request.UnAuth, req.ID, req)
	} else {
		req.ID = e.Websocket.Conn.GenerateMessageID(false)
		payload, err = e.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, req.ID, req)
	}
	if err != nil {
		return err
	}
	response := &WSRespData{
		Result: &result,
	}
	err = json.Unmarshal(payload, response)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		mes := fmt.Sprintf("error code: %d Message: %s", response.Code, response.Message)
		if response.DetailCode != "0" && response.DetailCode != "" {
			mes = fmt.Sprintf("%s Detail: %s %s", mes, response.DetailCode, response.DetailMessage)
		}
		return errors.New(mes)
	}
	return nil
}
