package liqui

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	LIQUI_API_PUBLIC_URL      = "https://api.Liqui.io/api"
	LIQUI_API_PRIVATE_URL     = "https://api.Liqui.io/tapi"
	LIQUI_API_PUBLIC_VERSION  = "3"
	LIQUI_API_PRIVATE_VERSION = "1"
	LIQUI_INFO                = "info"
	LIQUI_TICKER              = "ticker"
	LIQUI_DEPTH               = "depth"
	LIQUI_TRADES              = "trades"
	LIQUI_ACCOUNT_INFO        = "getInfo"
	LIQUI_TRADE               = "Trade"
	LIQUI_ACTIVE_ORDERS       = "ActiveOrders"
	LIQUI_ORDER_INFO          = "OrderInfo"
	LIQUI_CANCEL_ORDER        = "CancelOrder"
	LIQUI_TRADE_HISTORY       = "TradeHistory"
	LIQUI_WITHDRAW_COIN       = "WithdrawCoin"
)

type Liqui struct {
	exchange.ExchangeBase
	Ticker map[string]LiquiTicker
	Info   LiquiInfo
}

func (l *Liqui) SetDefaults() {
	l.Name = "Liqui"
	l.Enabled = false
	l.Fee = 0.25
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
	l.Ticker = make(map[string]LiquiTicker)
}

func (l *Liqui) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket = exch.Websocket
		l.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		l.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		l.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (l *Liqui) GetFee(currency string) (float64, error) {
	val, ok := l.Info.Pairs[common.StringToLower(currency)]
	if !ok {
		return 0, errors.New("Currency does not exist")
	}

	return val.Fee, nil
}

func (l *Liqui) GetAvailablePairs(nonHidden bool) []string {
	var pairs []string
	for x, y := range l.Info.Pairs {
		if nonHidden && y.Hidden == 1 {
			continue
		}
		pairs = append(pairs, common.StringToUpper(x))
	}
	return pairs
}

func (l *Liqui) GetInfo() (LiquiInfo, error) {
	req := fmt.Sprintf("%s/%s/%s/", LIQUI_API_PUBLIC_URL, LIQUI_API_PUBLIC_VERSION, LIQUI_INFO)
	resp := LiquiInfo{}
	err := common.SendHTTPGetRequest(req, true, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (l *Liqui) GetTicker(symbol string) (map[string]LiquiTicker, error) {
	type Response struct {
		Data map[string]LiquiTicker
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", LIQUI_API_PUBLIC_URL, LIQUI_API_PUBLIC_VERSION, LIQUI_TICKER, symbol)
	err := common.SendHTTPGetRequest(req, true, &response.Data)

	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (l *Liqui) GetDepth(symbol string) (LiquiOrderbook, error) {
	type Response struct {
		Data map[string]LiquiOrderbook
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", LIQUI_API_PUBLIC_URL, LIQUI_API_PUBLIC_VERSION, LIQUI_DEPTH, symbol)

	err := common.SendHTTPGetRequest(req, true, &response.Data)
	if err != nil {
		return LiquiOrderbook{}, err
	}

	depth := response.Data[symbol]
	return depth, nil
}

func (l *Liqui) GetTrades(symbol string) ([]LiquiTrades, error) {
	type Response struct {
		Data map[string][]LiquiTrades
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", LIQUI_API_PUBLIC_URL, LIQUI_API_PUBLIC_VERSION, LIQUI_TRADES, symbol)

	err := common.SendHTTPGetRequest(req, true, &response.Data)
	if err != nil {
		return []LiquiTrades{}, err
	}

	trades := response.Data[symbol]
	return trades, nil
}

func (l *Liqui) GetAccountInfo() (LiquiAccountInfo, error) {
	var result LiquiAccountInfo
	err := l.SendAuthenticatedHTTPRequest(LIQUI_ACCOUNT_INFO, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

//to-do: convert orderid to int64
func (l *Liqui) Trade(pair, orderType string, amount, price float64) (float64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	var result LiquiTrade
	err := l.SendAuthenticatedHTTPRequest(LIQUI_TRADE, req, &result)

	if err != nil {
		return 0, err
	}

	return result.OrderID, nil
}

func (l *Liqui) GetActiveOrders(pair string) (map[string]LiquiActiveOrders, error) {
	req := url.Values{}
	req.Add("pair", pair)

	var result map[string]LiquiActiveOrders
	err := l.SendAuthenticatedHTTPRequest(LIQUI_ACTIVE_ORDERS, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (l *Liqui) GetOrderInfo(OrderID int64) (map[string]LiquiOrderInfo, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result map[string]LiquiOrderInfo
	err := l.SendAuthenticatedHTTPRequest(LIQUI_ORDER_INFO, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (l *Liqui) CancelOrder(OrderID int64) (bool, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result LiquiCancelOrder
	err := l.SendAuthenticatedHTTPRequest(LIQUI_CANCEL_ORDER, req, &result)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (l *Liqui) GetTradeHistory(vals url.Values, pair string) (map[string]LiquiTradeHistory, error) {
	if pair != "" {
		vals.Add("pair", pair)
	}

	var result map[string]LiquiTradeHistory
	err := l.SendAuthenticatedHTTPRequest(LIQUI_TRADE_HISTORY, vals, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// API mentions that this isn't active now, but will be soon - you must provide the first 8 characters of the key
// in your ticket to support.
func (l *Liqui) WithdrawCoins(coin string, amount float64, address string) (LiquiWithdrawCoins, error) {
	req := url.Values{}
	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	var result LiquiWithdrawCoins
	err := l.SendAuthenticatedHTTPRequest(LIQUI_WITHDRAW_COIN, req, &result)

	if err != nil {
		return result, err
	}
	return result, nil
}

func (l *Liqui) SendAuthenticatedHTTPRequest(method string, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	values.Set("nonce", nonce)
	values.Set("method", method)

	encoded := values.Encode()
	hmac := common.GetHMAC(common.HASH_SHA512, []byte(encoded), []byte(l.APISecret))

	if l.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", LIQUI_API_PRIVATE_URL, method, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = l.APIKey
	headers["Sign"] = common.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", LIQUI_API_PRIVATE_URL, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	response := LiquiResponse{}
	err = common.JSONDecode([]byte(resp), &response)

	if err != nil {
		return err
	}

	if response.Success != 1 {
		return errors.New(response.Error)
	}

	jsonEncoded, err := common.JSONEncode(response.Return)

	if err != nil {
		return err
	}

	err = common.JSONDecode(jsonEncoded, &result)

	if err != nil {
		return err
	}
	return nil
}
