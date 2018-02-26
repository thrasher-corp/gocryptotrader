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
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	liquiAPIPublicURL      = "https://api.Liqui.io/api"
	liquiAPIPrivateURL     = "https://api.Liqui.io/tapi"
	liquiAPIPublicVersion  = "3"
	liquiAPIPrivateVersion = "1"
	liquiInfo              = "info"
	liquiTicker            = "ticker"
	liquiDepth             = "depth"
	liquiTrades            = "trades"
	liquiAccountInfo       = "getInfo"
	liquiTrade             = "Trade"
	liquiActiveOrders      = "ActiveOrders"
	liquiOrderInfo         = "OrderInfo"
	liquiCancelOrder       = "CancelOrder"
	liquiTradeHistory      = "TradeHistory"
	liquiWithdrawCoin      = "WithdrawCoin"
)

// Liqui is the overarching type across the liqui package
type Liqui struct {
	exchange.Base
	Ticker map[string]Ticker
	Info   Info
}

// SetDefaults sets current default values for liqui
func (l *Liqui) SetDefaults() {
	l.Name = "Liqui"
	l.Enabled = false
	l.Fee = 0.25
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
	l.Ticker = make(map[string]Ticker)
	l.RequestCurrencyPairFormat.Delimiter = "_"
	l.RequestCurrencyPairFormat.Uppercase = false
	l.RequestCurrencyPairFormat.Separator = "-"
	l.ConfigCurrencyPairFormat.Delimiter = "_"
	l.ConfigCurrencyPairFormat.Uppercase = true
	l.AssetTypes = []string{ticker.Spot}
}

// Setup sets exchange configuration parameters for liqui
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
		err := l.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns a fee for a specific currency
func (l *Liqui) GetFee(currency string) (float64, error) {
	val, ok := l.Info.Pairs[common.StringToLower(currency)]
	if !ok {
		return 0, errors.New("currency does not exist")
	}

	return val.Fee, nil
}

// GetAvailablePairs returns all available pairs
func (l *Liqui) GetAvailablePairs(nonHidden bool) []string {
	var pairs []string
	for x, y := range l.Info.Pairs {
		if nonHidden && y.Hidden == 1 || x == "" {
			continue
		}
		pairs = append(pairs, common.StringToUpper(x))
	}
	return pairs
}

// GetInfo provides all the information about currently active pairs, such as
// the maximum number of digits after the decimal point, the minimum price, the
// maximum price, the minimum transaction size, whether the pair is hidden, the
// commission for each pair.
func (l *Liqui) GetInfo() (Info, error) {
	resp := Info{}
	req := fmt.Sprintf("%s/%s/%s/", liquiAPIPublicURL, liquiAPIPublicVersion, liquiInfo)

	return resp, common.SendHTTPGetRequest(req, true, l.Verbose, &resp)
}

// GetTicker returns information about currently active pairs, such as: the
// maximum price, the minimum price, average price, trade volume, trade volume
// in currency, the last trade, Buy and Sell price. All information is provided
// over the past 24 hours.
//
// currencyPair - example "eth_btc"
func (l *Liqui) GetTicker(currencyPair string) (map[string]Ticker, error) {
	type Response struct {
		Data    map[string]Ticker
		Success int    `json:"success"`
		Error   string `json:"error"`
	}

	response := Response{Data: make(map[string]Ticker)}
	req := fmt.Sprintf("%s/%s/%s/%s", liquiAPIPublicURL, liquiAPIPublicVersion, liquiTicker, currencyPair)

	return response.Data,
		common.SendHTTPGetRequest(req, true, l.Verbose, &response.Data)
}

// GetDepth information about active orders on the pair. Additionally it accepts
// an optional GET-parameter limit, which indicates how many orders should be
// displayed (150 by default). Is set to less than 2000.
func (l *Liqui) GetDepth(currencyPair string) (Orderbook, error) {
	type Response struct {
		Data    map[string]Orderbook
		Success int    `json:"success"`
		Error   string `json:"error"`
	}

	response := Response{Data: make(map[string]Orderbook)}
	req := fmt.Sprintf("%s/%s/%s/%s", liquiAPIPublicURL, liquiAPIPublicVersion, liquiDepth, currencyPair)

	return response.Data[currencyPair],
		common.SendHTTPGetRequest(req, true, l.Verbose, &response.Data)
}

// GetTrades returns information about the last trades. Additionally it accepts
// an optional GET-parameter limit, which indicates how many orders should be
// displayed (150 by default). The maximum allowable value is 2000.
func (l *Liqui) GetTrades(currencyPair string) ([]Trades, error) {
	type Response struct {
		Data    map[string][]Trades
		Success int    `json:"success"`
		Error   string `json:"error"`
	}

	response := Response{Data: make(map[string][]Trades)}
	req := fmt.Sprintf("%s/%s/%s/%s", liquiAPIPublicURL, liquiAPIPublicVersion, liquiTrades, currencyPair)

	return response.Data[currencyPair],
		common.SendHTTPGetRequest(req, true, l.Verbose, &response.Data)
}

// GetAccountInfo returns information about the user’s current balance, API-key
// privileges, the number of open orders and Server Time. To use this method you
// need a privilege of the key info.
func (l *Liqui) GetAccountInfo() (AccountInfo, error) {
	var result AccountInfo

	return result,
		l.SendAuthenticatedHTTPRequest(liquiAccountInfo, url.Values{}, &result)
}

// Trade creates orders on the exchange.
// to-do: convert orderid to int64
func (l *Liqui) Trade(pair, orderType string, amount, price float64) (float64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	var result Trade

	return result.OrderID, l.SendAuthenticatedHTTPRequest(liquiTrade, req, &result)
}

// GetActiveOrders returns the list of your active orders.
func (l *Liqui) GetActiveOrders(pair string) (map[string]ActiveOrders, error) {
	result := make(map[string]ActiveOrders)

	req := url.Values{}
	req.Add("pair", pair)

	return result, l.SendAuthenticatedHTTPRequest(liquiActiveOrders, req, &result)
}

// GetOrderInfo returns the information on particular order.
func (l *Liqui) GetOrderInfo(OrderID int64) (map[string]OrderInfo, error) {
	result := make(map[string]OrderInfo)

	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	return result, l.SendAuthenticatedHTTPRequest(liquiOrderInfo, req, &result)
}

// CancelOrder method is used for order cancelation.
func (l *Liqui) CancelOrder(OrderID int64) (bool, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result CancelOrder

	err := l.SendAuthenticatedHTTPRequest(liquiCancelOrder, req, &result)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetTradeHistory returns trade history
func (l *Liqui) GetTradeHistory(vals url.Values, pair string) (map[string]TradeHistory, error) {
	result := make(map[string]TradeHistory)

	if pair != "" {
		vals.Add("pair", pair)
	}

	return result, l.SendAuthenticatedHTTPRequest(liquiTradeHistory, vals, &result)
}

// WithdrawCoins is designed for cryptocurrency withdrawals.
// API mentions that this isn't active now, but will be soon - you must provide the first 8 characters of the key
// in your ticket to support.
func (l *Liqui) WithdrawCoins(coin string, amount float64, address string) (WithdrawCoins, error) {
	req := url.Values{}
	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	var result WithdrawCoins
	return result, l.SendAuthenticatedHTTPRequest(liquiWithdrawCoin, req, &result)
}

// SendAuthenticatedHTTPRequest sends an authenticated http request to liqui
func (l *Liqui) SendAuthenticatedHTTPRequest(method string, values url.Values, result interface{}) (err error) {
	if !l.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, l.Name)
	}

	if l.Nonce.Get() == 0 {
		l.Nonce.Set(time.Now().Unix())
	} else {
		l.Nonce.Inc()
	}
	values.Set("nonce", l.Nonce.String())
	values.Set("method", method)

	encoded := values.Encode()
	hmac := common.GetHMAC(common.HashSHA512, []byte(encoded), []byte(l.APISecret))

	if l.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", liquiAPIPrivateURL, method, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = l.APIKey
	headers["Sign"] = common.HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", liquiAPIPrivateURL, headers, strings.NewReader(encoded))
	if err != nil {
		return err
	}

	response := Response{}

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
