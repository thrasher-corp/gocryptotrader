package okcoin

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	okcoinAPIURL                = "https://www.okcoin.com/api/v1/"
	okcoinAPIURLChina           = "https://www.okcoin.cn/api/v1/"
	okcoinAPIVersion            = "1"
	okcoinWebsocketURL          = "wss://real.okcoin.com:10440/websocket/okcoinapi"
	okcoinWebsocketURLChina     = "wss://real.okcoin.cn:10440/websocket/okcoinapi"
	okcoinTicker                = "ticker.do"
	okcoinDepth                 = "depth.do"
	okcoinTrades                = "trades.do"
	okcoinKline                 = "kline.do"
	okcoinUserInfo              = "userinfo.do"
	okcoinTrade                 = "trade.do"
	okcoinTradeHistory          = "trade_history.do"
	okcoinTradeBatch            = "batch_trade.do"
	okcoinOrderCancel           = "cancel_order.do"
	okcoinOrderInfo             = "order_info.do"
	okcoinOrdersInfo            = "orders_info.do"
	okcoinOrderHistory          = "order_history.do"
	okcoinWithdraw              = "withdraw.do"
	okcoinWithdrawCancel        = "cancel_withdraw.do"
	okcoinWithdrawInfo          = "withdraw_info.do"
	okcoinOrderFee              = "order_fee.do"
	okcoinLendDepth             = "lend_depth.do"
	okcoinBorrowsInfo           = "borrows_info.do"
	okcoinBorrowMoney           = "borrow_money.do"
	okcoinBorrowCancel          = "cancel_borrow.do"
	okcoinBorrowOrderInfo       = "borrow_order_info.do"
	okcoinRepayment             = "repayment.do"
	okcoinUnrepaymentsInfo      = "unrepayments_info.do"
	okcoinAccountRecords        = "account_records.do"
	okcoinFuturesTicker         = "future_ticker.do"
	okcoinFuturesDepth          = "future_depth.do"
	okcoinFuturesTrades         = "future_trades.do"
	okcoinFuturesIndex          = "future_index.do"
	okcoinExchangeRate          = "exchange_rate.do"
	okcoinFuturesEstimatedPrice = "future_estimated_price.do"
	okcoinFuturesKline          = "future_kline.do"
	okcoinFuturesHoldAmount     = "future_hold_amount.do"
	okcoinFuturesUserInfo       = "future_userinfo.do"
	okcoinFuturesPosition       = "future_position.do"
	okcoinFuturesTrade          = "future_trade.do"
	okcoinFuturesTradeHistory   = "future_trades_history.do"
	okcoinFuturesTradeBatch     = "future_batch_trade.do"
	okcoinFuturesCancel         = "future_cancel.do"
	okcoinFuturesOrderInfo      = "future_order_info.do"
	okcoinFuturesOrdersInfo     = "future_orders_info.do"
	okcoinFuturesUserInfo4Fix   = "future_userinfo_4fix.do"
	okcoinFuturesposition4Fix   = "future_position_4fix.do"
	okcoinFuturesExplosive      = "future_explosive.do"
	okcoinFuturesDevolve        = "future_devolve.do"
)

var (
	okcoinDefaultsSet = false
)

// OKCoin is the overarching type across this package
type OKCoin struct {
	exchange.Base
	RESTErrors      map[string]string
	WebsocketErrors map[string]string
	FuturesValues   []string
	WebsocketConn   *websocket.Conn
}

// setCurrencyPairFormats sets currency pair formatting for this package
func (o *OKCoin) setCurrencyPairFormats() {
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = ""
	o.ConfigCurrencyPairFormat.Uppercase = true
}

// SetDefaults sets current default values for this package
func (o *OKCoin) SetDefaults() {
	o.SetErrorDefaults()
	o.SetWebsocketErrorDefaults()
	o.Enabled = false
	o.Verbose = false
	o.Websocket = false
	o.RESTPollingDelay = 10
	o.FuturesValues = []string{"this_week", "next_week", "quarter"}
	o.AssetTypes = []string{ticker.Spot}

	if okcoinDefaultsSet {
		o.AssetTypes = append(o.AssetTypes, o.FuturesValues...)
		o.APIUrl = okcoinAPIURL
		o.Name = "OKCOIN International"
		o.WebsocketURL = okcoinWebsocketURL
		o.setCurrencyPairFormats()
	} else {
		o.APIUrl = okcoinAPIURLChina
		o.Name = "OKCOIN China"
		o.WebsocketURL = okcoinWebsocketURLChina
		okcoinDefaultsSet = true
		o.setCurrencyPairFormats()
	}
}

// Setup sets exchange configuration parameters
func (o *OKCoin) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket = exch.Websocket
		o.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		o.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		o.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := o.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns current fees for the exchange
func (o *OKCoin) GetFee(maker bool) float64 {
	if o.APIUrl == okcoinAPIURL {
		if maker {
			return o.MakerFee
		}
		return o.TakerFee
	}
	// Chinese exchange does not have any trading fees
	return 0
}

// GetTicker returns the current ticker
func (o *OKCoin) GetTicker(symbol string) (Ticker, error) {
	resp := TickerResponse{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	path := common.EncodeURLValues(o.APIUrl+okcoinTicker, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &resp)
	if err != nil {
		return Ticker{}, err
	}
	return resp.Ticker, nil
}

// GetOrderBook returns the current order book by size
func (o *OKCoin) GetOrderBook(symbol string, size int64, merge bool) (Orderbook, error) {
	resp := Orderbook{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if merge {
		vals.Set("merge", "1")
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinDepth, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// GetTrades returns historic trades since a timestamp
func (o *OKCoin) GetTrades(symbol string, since int64) ([]Trades, error) {
	result := []Trades{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	if since != 0 {
		vals.Set("since", strconv.FormatInt(since, 10))
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinTrades, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetKline returns kline data
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

	path := common.EncodeURLValues(o.APIUrl+okcoinKline, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetFuturesTicker returns a current ticker for the futures market
func (o *OKCoin) GetFuturesTicker(symbol, contractType string) (FuturesTicker, error) {
	resp := FuturesTickerResponse{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)
	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesTicker, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &resp)
	if err != nil {
		return FuturesTicker{}, err
	}
	return resp.Ticker, nil
}

// GetFuturesDepth returns current depth for the futures market
func (o *OKCoin) GetFuturesDepth(symbol, contractType string, size int64, merge bool) (Orderbook, error) {
	result := Orderbook{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	if size != 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}
	if merge {
		vals.Set("merge", "1")
	}

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesDepth, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetFuturesTrades returns historic trades for the futures market
func (o *OKCoin) GetFuturesTrades(symbol, contractType string) ([]FuturesTrades, error) {
	result := []FuturesTrades{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesTrades, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetFuturesIndex returns an index for the futures market
func (o *OKCoin) GetFuturesIndex(symbol string) (float64, error) {
	type Response struct {
		Index float64 `json:"future_index"`
	}

	result := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesIndex, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &result)
	if err != nil {
		return 0, err
	}
	return result.Index, nil
}

// GetFuturesExchangeRate returns the exchange rate for the futures market
func (o *OKCoin) GetFuturesExchangeRate() (float64, error) {
	type Response struct {
		Rate float64 `json:"rate"`
	}

	result := Response{}
	err := common.SendHTTPGetRequest(o.APIUrl+okcoinExchangeRate, true, o.Verbose, &result)
	if err != nil {
		return result.Rate, err
	}
	return result.Rate, nil
}

// GetFuturesEstimatedPrice returns a current estimated futures price for a
// currency
func (o *OKCoin) GetFuturesEstimatedPrice(symbol string) (float64, error) {
	type Response struct {
		Price float64 `json:"forecast_price"`
	}

	result := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesEstimatedPrice, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &result)
	if err != nil {
		return result.Price, err
	}
	return result.Price, nil
}

// GetFuturesKline returns kline data for a specific currency on the futures
// market
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

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesKline, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFuturesHoldAmount returns the hold amount for a futures trade
func (o *OKCoin) GetFuturesHoldAmount(symbol, contractType string) ([]FuturesHoldAmount, error) {
	resp := []FuturesHoldAmount{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesHoldAmount, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetFuturesExplosive returns the explosive for a futures contract
func (o *OKCoin) GetFuturesExplosive(symbol, contractType string, status, currentPage, pageLength int64) ([]FuturesExplosive, error) {
	type Response struct {
		Data []FuturesExplosive `json:"data"`
	}
	resp := Response{}
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("contract_type", contractType)
	vals.Set("status", strconv.FormatInt(status, 10))
	vals.Set("current_page", strconv.FormatInt(currentPage, 10))
	vals.Set("page_length", strconv.FormatInt(pageLength, 10))

	path := common.EncodeURLValues(o.APIUrl+okcoinFuturesExplosive, vals)
	err := common.SendHTTPGetRequest(path, true, o.Verbose, &resp)

	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetUserInfo returns user information associated with the calling APIkeys
func (o *OKCoin) GetUserInfo() (UserInfo, error) {
	result := UserInfo{}
	err := o.SendAuthenticatedHTTPRequest(okcoinUserInfo, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// Trade initiates a new trade
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

	err := o.SendAuthenticatedHTTPRequest(okcoinTrade, v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to place order")
	}

	return result.OrderID, nil
}

// GetTradeHistory returns client trade history
func (o *OKCoin) GetTradeHistory(symbol string, TradeID int64) ([]Trades, error) {
	result := []Trades{}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("since", strconv.FormatInt(TradeID, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinTradeHistory, v, &result)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// BatchTrade initiates a trade by batch order
func (o *OKCoin) BatchTrade(orderData string, symbol, orderType string) (BatchTrade, error) {
	v := url.Values{}
	v.Set("orders_data", orderData)
	v.Set("symbol", symbol)
	v.Set("type", orderType)
	result := BatchTrade{}

	err := o.SendAuthenticatedHTTPRequest(okcoinTradeBatch, v, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// CancelOrder cancels a specific order or list of orders by orderID
func (o *OKCoin) CancelOrder(orderID []int64, symbol string) (CancelOrderResponse, error) {
	v := url.Values{}
	orders := []string{}
	result := CancelOrderResponse{}

	orderStr := strconv.FormatInt(orderID[0], 10)

	if len(orderID) > 1 {
		for x := range orderID {
			orders = append(orders, strconv.FormatInt(orderID[x], 10))
		}
		orderStr = common.JoinStrings(orders, ",")
	}

	v.Set("order_id", orderStr)
	v.Set("symbol", symbol)

	return result, o.SendAuthenticatedHTTPRequest(okcoinOrderCancel, v, &result)
}

// GetOrderInfo returns order information by orderID
func (o *OKCoin) GetOrderInfo(orderID int64, symbol string) ([]OrderInfo, error) {
	type Response struct {
		Result bool        `json:"result"`
		Orders []OrderInfo `json:"orders"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinOrderInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if result.Result != true {
		return nil, errors.New("unable to retrieve order info")
	}

	return result.Orders, nil
}

// GetOrderInfoBatch returns order info on a batch of orders
func (o *OKCoin) GetOrderInfoBatch(orderID []int64, symbol string) ([]OrderInfo, error) {
	type Response struct {
		Result bool        `json:"result"`
		Orders []OrderInfo `json:"orders"`
	}

	orders := []string{}
	for x := range orderID {
		orders = append(orders, strconv.FormatInt(orderID[x], 10))
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", common.JoinStrings(orders, ","))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinOrderInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if result.Result != true {
		return nil, errors.New("unable to retrieve order info")
	}

	return result.Orders, nil
}

// GetOrderHistory returns a history of orders
func (o *OKCoin) GetOrderHistory(pageLength, currentPage int64, status, symbol string) (OrderHistory, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("status", status)
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))
	result := OrderHistory{}

	err := o.SendAuthenticatedHTTPRequest(okcoinOrderHistory, v, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// Withdrawal withdraws a cryptocurrency to a supplied address
func (o *OKCoin) Withdrawal(symbol string, fee float64, tradePWD, address string, amount float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)

	if fee != 0 {
		v.Set("chargefee", strconv.FormatFloat(fee, 'f', -1, 64))
	}
	v.Set("trade_pwd", tradePWD)
	v.Set("withdraw_address", address)
	v.Set("withdraw_amount", strconv.FormatFloat(amount, 'f', -1, 64))
	result := WithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinWithdraw, v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to process withdrawal request")
	}

	return result.WithdrawID, nil
}

// CancelWithdrawal cancels a withdrawal
func (o *OKCoin) CancelWithdrawal(symbol string, withdrawalID int64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("withdrawal_id", strconv.FormatInt(withdrawalID, 10))
	result := WithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinWithdrawCancel, v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to process withdrawal cancel request")
	}

	return result.WithdrawID, nil
}

// GetWithdrawalInfo returns withdrawal information
func (o *OKCoin) GetWithdrawalInfo(symbol string, withdrawalID int64) ([]WithdrawInfo, error) {
	type Response struct {
		Result   bool
		Withdraw []WithdrawInfo `json:"withdraw"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("withdrawal_id", strconv.FormatInt(withdrawalID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinWithdrawInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("unable to process withdrawal cancel request")
	}

	return result.Withdraw, nil
}

// GetOrderFeeInfo returns order fee information
func (o *OKCoin) GetOrderFeeInfo(symbol string, orderID int64) (OrderFeeInfo, error) {
	type Response struct {
		Data   OrderFeeInfo `json:"data"`
		Result bool         `json:"result"`
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinOrderFee, v, &result)

	if err != nil {
		return result.Data, err
	}

	if !result.Result {
		return result.Data, errors.New("unable to get order fee info")
	}

	return result.Data, nil
}

// GetLendDepth returns the depth of lends
func (o *OKCoin) GetLendDepth(symbol string) ([]LendDepth, error) {
	type Response struct {
		LendDepth []LendDepth `json:"lend_depth"`
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinLendDepth, v, &result)

	if err != nil {
		return nil, err
	}

	return result.LendDepth, nil
}

// GetBorrowInfo returns borrow information
func (o *OKCoin) GetBorrowInfo(symbol string) (BorrowInfo, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	result := BorrowInfo{}

	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowsInfo, v, &result)

	if err != nil {
		return result, nil
	}

	return result, nil
}

// Borrow initiates a borrow request
func (o *OKCoin) Borrow(symbol, days string, amount, rate float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("days", days)
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	result := BorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowMoney, v, &result)

	if err != nil {
		return 0, err
	}

	if !result.Result {
		return 0, errors.New("unable to borrow")
	}

	return result.BorrowID, nil
}

// CancelBorrow cancels a borrow request
func (o *OKCoin) CancelBorrow(symbol string, borrowID int64) (bool, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := BorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowCancel, v, &result)

	if err != nil {
		return false, err
	}

	if !result.Result {
		return false, errors.New("unable to cancel borrow")
	}

	return true, nil
}

// GetBorrowOrderInfo returns information about a borrow order
func (o *OKCoin) GetBorrowOrderInfo(borrowID int64) (BorrowInfo, error) {
	type Response struct {
		Result      bool       `json:"result"`
		BorrowOrder BorrowInfo `json:"borrow_order"`
	}

	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := Response{}
	err := o.SendAuthenticatedHTTPRequest(okcoinBorrowOrderInfo, v, &result)

	if err != nil {
		return result.BorrowOrder, err
	}

	if !result.Result {
		return result.BorrowOrder, errors.New("unable to get borrow info")
	}

	return result.BorrowOrder, nil
}

// GetRepaymentInfo returns information on a repayment
func (o *OKCoin) GetRepaymentInfo(borrowID int64) (bool, error) {
	v := url.Values{}
	v.Set("borrow_id", strconv.FormatInt(borrowID, 10))
	result := BorrowResponse{}

	err := o.SendAuthenticatedHTTPRequest(okcoinRepayment, v, &result)

	if err != nil {
		return false, err
	}

	if !result.Result {
		return false, errors.New("unable to get repayment info")
	}

	return true, nil
}

// GetUnrepaymentsInfo returns information on an unrepayment
func (o *OKCoin) GetUnrepaymentsInfo(symbol string, currentPage, pageLength int) ([]BorrowOrder, error) {
	type Response struct {
		Unrepayments []BorrowOrder `json:"unrepayments"`
		Result       bool          `json:"result"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))
	result := Response{}
	err := o.SendAuthenticatedHTTPRequest(okcoinUnrepaymentsInfo, v, &result)

	if err != nil {
		return nil, err
	}

	if !result.Result {
		return nil, errors.New("unable to get unrepayments info")
	}

	return result.Unrepayments, nil
}

// GetAccountRecords returns account records
func (o *OKCoin) GetAccountRecords(symbol string, recType, currentPage, pageLength int) ([]AccountRecords, error) {
	type Response struct {
		Records []AccountRecords `json:"records"`
		Symbol  string           `json:"symbol"`
	}
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("type", strconv.Itoa(recType))
	v.Set("current_page", strconv.Itoa(currentPage))
	v.Set("page_length", strconv.Itoa(pageLength))
	result := Response{}

	err := o.SendAuthenticatedHTTPRequest(okcoinAccountRecords, v, &result)

	if err != nil {
		return nil, err
	}

	return result.Records, nil
}

// GetFuturesUserInfo returns information on a users futures
func (o *OKCoin) GetFuturesUserInfo() {
	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesUserInfo, url.Values{}, nil)

	if err != nil {
		log.Println(err)
	}
}

// GetFuturesPosition returns position on a futures contract
func (o *OKCoin) GetFuturesPosition(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesPosition, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// FuturesTrade initiates a new futures trade
func (o *OKCoin) FuturesTrade(amount, price float64, matchPrice, leverage int64, symbol, contractType, orderType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	v.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("type", orderType)
	v.Set("match_price", strconv.FormatInt(matchPrice, 10))
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesTrade, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// FuturesBatchTrade initiates a batch of futures contract trades
func (o *OKCoin) FuturesBatchTrade(orderData, symbol, contractType string, leverage int64, orderType string) {
	v := url.Values{} //to-do batch trade support for orders_data)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("orders_data", orderData)
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesTradeBatch, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// CancelFuturesOrder cancels a futures contract order
func (o *OKCoin) CancelFuturesOrder(orderID int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesCancel, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// GetFuturesOrderInfo returns information on a specfic futures contract order
func (o *OKCoin) GetFuturesOrderInfo(orderID, status, currentPage, pageLength int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("status", strconv.FormatInt(status, 10))
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesOrderInfo, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// GetFutureOrdersInfo returns information on a range of futures orders
func (o *OKCoin) GetFutureOrdersInfo(orderID int64, contractType, symbol string) {
	v := url.Values{}
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("contract_type", contractType)
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesOrdersInfo, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// GetFuturesUserInfo4Fix returns futures user info fix rate
func (o *OKCoin) GetFuturesUserInfo4Fix() {
	v := url.Values{}

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesUserInfo4Fix, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// GetFuturesUserPosition4Fix returns futures user info on a fixed position
func (o *OKCoin) GetFuturesUserPosition4Fix(symbol, contractType string) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("type", strconv.FormatInt(1, 10))

	err := o.SendAuthenticatedHTTPRequest(okcoinFuturesUserInfo4Fix, v, nil)

	if err != nil {
		log.Println(err)
	}
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (o *OKCoin) SendAuthenticatedHTTPRequest(method string, v url.Values, result interface{}) (err error) {
	if !o.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, o.Name)
	}

	v.Set("api_key", o.APIKey)
	hasher := common.GetMD5([]byte(v.Encode() + "&secret_key=" + o.APISecret))
	v.Set("sign", strings.ToUpper(common.HexEncodeToString(hasher)))

	encoded := v.Encode()
	path := o.APIUrl + method

	if o.Verbose {
		log.Printf("Sending POST request to %s with params %s\n", path, encoded)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", path, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	if o.Verbose {
		log.Printf("Received raw: \n%s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}

	return nil
}

// SetErrorDefaults sets default error map
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
