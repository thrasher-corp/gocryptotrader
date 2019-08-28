package oex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Oex is the overarching type across this package
type Oex struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
}

const (
	oexAPIURL     = "https://openapi.oex.com"
	oexAPIVersion = ""

	// Public endpoints
	oexGetTicker           = "/open/api/get_ticker"
	oexGetAllTicker        = "/open/api/get_allticker"
	oexGetKline            = "/open/api/get_records"
	oexGetTrades           = "/open/api/get_trades"
	oexGetLatestCurrPrices = "/open/api/market"
	oexGetMarketDepth      = "/open/api/market_dept"
	oexGetAllPairs         = "/open/api/common/symbols"
	// Authenticated endpoints
	oexUserInfo        = "/open/api/user/account"
	oexUserBalanceInfo = "/open/api/user_balance_info"
	oexAllOrders       = "/open/api/v2/all_order"
	oexOrderHistory    = "/open/api/all_trade"
	oexRemoveOrder     = "/open/api/cancel_order"
	oexRemoveAllOrder  = "/open/api/cancel_order_all"
	oexCreateOrder     = "/open/api/create_order"
	oexAllOpenOrders   = "/open/api/v2/new_order"
	oexSelfTrade       = "/open/api/self_trade"
	oexFetchOrderInfo  = "/open/api/order_info"
	oexNoError         = "0"
)

// SetDefaults sets the basic defaults for Oex
func (o *Oex) SetDefaults() {
	o.Name = "Oex"
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.RequestCurrencyPairFormat.Delimiter = ""
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = ""
	o.ConfigCurrencyPairFormat.Uppercase = false
	o.AssetTypes = []string{ticker.Spot}
	o.SupportsAutoPairUpdating = false
	o.SupportsRESTTickerBatching = false
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, 0),
		request.NewRateLimit(time.Second, 0),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	o.APIUrlDefault = oexAPIURL
	o.APIUrl = o.APIUrlDefault
	o.Websocket = wshandler.New()
}

// Setup takes in the supplied exchange configuration details and sets params
func (o *Oex) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		o.SetHTTPClientTimeout(exch.HTTPTimeout)
		o.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket.SetWsStatusAndConnection(exch.Websocket)
		o.BaseCurrencies = exch.BaseCurrencies
		o.AvailablePairs = exch.AvailablePairs
		o.EnabledPairs = exch.EnabledPairs
		err := o.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetTicker returns ticker info for a selected pair
func (o *Oex) GetTicker(symbol string) (TickerResponse, error) {
	var resp TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s?%s", o.APIUrl, oexGetTicker, params.Encode())
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetAllTicker returns ticker info for all trading pairs available
func (o *Oex) GetAllTicker() (AllTickerResponse, error) {
	var resp AllTickerResponse
	path := fmt.Sprintf("%s%s", o.APIUrl, oexGetAllTicker)
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetKline returns kline data
func (o *Oex) GetKline(symbol, period string) (KlineResponse, error) {
	var resp KlineResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("period", period)
	path := fmt.Sprintf("%s%s?%s", o.APIUrl, oexGetKline, params.Encode())
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetTrades gets info about market transaction records of a given pair
func (o *Oex) GetTrades(symbol string) (TradeResponse, error) {
	var resp TradeResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s?%s", o.APIUrl, oexGetTrades, params.Encode())
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// LatestCurrencyPrices gets the latest prices for all currencies
func (o *Oex) LatestCurrencyPrices(time string) (LatestCurrencyPrices, error) {
	var resp LatestCurrencyPrices
	params := url.Values{}
	params.Set("time", time)
	path := fmt.Sprintf("%s%s?%s", o.APIUrl, oexGetLatestCurrPrices, params.Encode())
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetMarketDepth gets market depth data
func (o *Oex) GetMarketDepth(symbol, depthType string) (MarketDepthResponse, error) {
	var resp MarketDepthResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("type", depthType)
	path := fmt.Sprintf("%s%s?%s", o.APIUrl, oexGetMarketDepth, params.Encode())
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetAllPairs gets all pairs supported by the exchange and their accuracy
func (o *Oex) GetAllPairs() (AllPairResponse, error) {
	var resp AllPairResponse
	path := o.APIUrl + oexGetAllPairs
	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetUserInfo gets account information given the API key and Secret
func (o *Oex) GetUserInfo() (UserInfoResponse, error) {
	var resp UserInfoResponse
	path := o.APIUrl + oexUserInfo
	err := o.SendAuthHTTPRequest(http.MethodGet, path, nil, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetAllOrders acquires full delegation
func (o *Oex) GetAllOrders(symbol, startDate, endDate, pageSize, page string) (AllOrderResponse, error) {
	var resp AllOrderResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("startDate", startDate)
	params.Set("endDate", endDate)
	params.Set("pageSize", pageSize)
	params.Set("page", page)
	path := o.APIUrl + oexAllOpenOrders
	err := o.SendAuthHTTPRequest(http.MethodGet, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// FindOrderHistory fetches past orders based on the parameters
func (o *Oex) FindOrderHistory(symbol, startDate, endDate, pageSize, page, sort string) (OrderHistoryResponse, error) {
	var resp OrderHistoryResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("startDate", startDate)
	params.Set("endDate", endDate)
	params.Set("pageSize", pageSize)
	params.Set("page", page)
	params.Set("sort", sort)
	path := o.APIUrl + oexOrderHistory
	err := o.SendAuthHTTPRequest(http.MethodGet, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// RemoveOrder cacncels the order for the provided OrderID
func (o *Oex) RemoveOrder(orderID, symbol string) (RemoveOrderResponse, error) {
	var resp RemoveOrderResponse
	params := url.Values{}
	params.Set("order_id", orderID)
	params.Set("symbol", symbol)
	path := o.APIUrl + oexRemoveOrder
	err := o.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// RemoveAllOrders cancels all orders for a given currency pair
func (o *Oex) RemoveAllOrders(symbol string) (RemoveOrderResponse, error) {
	var resp RemoveOrderResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := o.APIUrl + oexRemoveAllOrder
	err := o.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil

}

// CreateOrder creates a new order
func (o *Oex) CreateOrder(side, orderType, volume, price, symbol, fee string) (CreateOrderResponse, error) {
	var resp CreateOrderResponse
	params := url.Values{}
	params.Set("side", side)
	params.Set("type", orderType)
	params.Set("volume", volume)
	params.Set("symbol", symbol)
	params.Set("fee_is_user_exchange_coin", fee)
	path := o.APIUrl + oexCreateOrder
	err := o.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetOpenOrders gets all the current delegation
func (o *Oex) GetOpenOrders(symbol, pageSize, page string) (OpenOrderResponse, error) {
	var resp OpenOrderResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("pageSize", pageSize)
	params.Set("page", page)
	path := o.APIUrl + oexAllOpenOrders
	err := o.SendAuthHTTPRequest(http.MethodGet, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// SelfTrade stores information about self trades
func (o *Oex) SelfTrade(side, orderType, volume, price, symbol, fee string) (SelfTradeResponse, error) {
	var resp SelfTradeResponse
	params := url.Values{}
	params.Set("side", side)
	params.Set("type", orderType)
	params.Set("volume", volume)
	params.Set("price", price)
	params.Set("symbol", symbol)
	params.Set("fee_is_user_exchange_coin", fee)
	path := o.APIUrl + oexSelfTrade
	err := o.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// GetUserAssetData gets user asset and recharge data
func (o *Oex) GetUserAssetData(uid, mobileNumber, email string) (UserAssetResponse, error) {
	var resp UserAssetResponse
	params := url.Values{}
	params.Set("uid", uid)
	params.Set("mobile_number", mobileNumber)
	params.Set("email", email)
	path := o.APIUrl + oexUserBalanceInfo
	err := o.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// FetchOrderInfo gets data of the order given its orderid
func (o *Oex) FetchOrderInfo(orderID, symbol string) (FetchOrderResponse, error) {
	var resp FetchOrderResponse
	params := url.Values{}
	params.Set("order_id", orderID)
	params.Set("symbol", symbol)
	path := o.APIUrl + oexUserBalanceInfo
	err := o.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != oexNoError {
		return resp, ErrorCapture(resp.ErrCapture.Error, resp.ErrCapture.Msg)
	}

	return resp, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (o *Oex) SendHTTPRequest(path string, result interface{}) error {
	var intermediary json.RawMessage
	err := o.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		&intermediary,
		false,
		false,
		o.Verbose,
		o.HTTPDebugging,
		o.HTTPRecording)
	if err != nil {
		return err
	}
	return json.Unmarshal(intermediary, result)
}

// ErrorCapture deals with errors
func ErrorCapture(code, message string) error {
	var temp []string
	temp = append(temp, code, message)
	return errors.New(strings.Join(temp, ":"))
}

// SendAuthHTTPRequest sends a post request (api keys and sign included)
func (o *Oex) SendAuthHTTPRequest(method, path string, params url.Values, result interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)[:13]
	params.Set("api_key", o.APIKey)
	params.Set("time", timestamp)

	var temp []string
	for a := range params {
		if params[a][0] == "" {
			continue
		}
		temp = append(temp, a)
	}
	sort.Strings(temp)

	var tempPath string

	for x := range temp {
		tempPath = fmt.Sprintf("%s%s%s", tempPath, temp[x], strings.Join(params[temp[x]], ""))
	}

	signMsg := fmt.Sprintf("%s%s", tempPath, o.APISecret)

	md5 := common.GetMD5([]byte(signMsg))
	params.Set("sign", common.HexEncodeToString(md5))

	path = common.EncodeURLValues(path, params)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded;charset=utf-8"

	return o.SendPayload(method,
		path,
		headers,
		nil,
		&result,
		true,
		false,
		o.Verbose,
		o.HTTPDebugging,
		o.HTTPRecording)
}
