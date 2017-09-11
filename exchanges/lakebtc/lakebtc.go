package lakebtc

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	LAKEBTC_API_URL               = "https://api.lakebtc.com/api_v2"
	LAKEBTC_API_VERSION           = "2"
	LAKEBTC_TICKER                = "ticker"
	LAKEBTC_ORDERBOOK             = "bcorderbook"
	LAKEBTC_TRADES                = "bctrades"
	LAKEBTC_GET_ACCOUNT_INFO      = "getAccountInfo"
	LAKEBTC_BUY_ORDER             = "buyOrder"
	LAKEBTC_SELL_ORDER            = "sellOrder"
	LAKEBTC_OPEN_ORDERS           = "openOrders"
	LAKEBTC_GET_ORDERS            = "getOrders"
	LAKEBTC_CANCEL_ORDER          = "cancelOrder"
	LAKEBTC_GET_TRADES            = "getTrades"
	LAKEBTC_GET_EXTERNAL_ACCOUNTS = "getExternalAccounts"
	LAKEBTC_CREATE_WITHDRAW       = "createWithdraw"
)

type LakeBTC struct {
	exchange.Base
}

func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = false
	l.TakerFee = 0.2
	l.MakerFee = 0.15
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
	l.RequestCurrencyPairFormat.Delimiter = ""
	l.RequestCurrencyPairFormat.Uppercase = true
	l.ConfigCurrencyPairFormat.Delimiter = ""
	l.ConfigCurrencyPairFormat.Uppercase = true
	l.AssetTypes = []string{ticker.Spot}
}

func (l *LakeBTC) Setup(exch config.ExchangeConfig) {
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

func (l *LakeBTC) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	} else {
		return l.TakerFee
	}
}

func (l *LakeBTC) GetTicker() (map[string]LakeBTCTicker, error) {
	response := make(map[string]LakeBTCTickerResponse)
	path := fmt.Sprintf("%s/%s", LAKEBTC_API_URL, LAKEBTC_TICKER)
	err := common.SendHTTPGetRequest(path, true, l.Verbose, &response)
	if err != nil {
		return nil, err
	}
	result := make(map[string]LakeBTCTicker)

	var addresses []string
	for k, v := range response {
		var ticker LakeBTCTicker
		key := common.StringToUpper(k)
		if v.Ask != nil {
			ticker.Ask, _ = strconv.ParseFloat(v.Ask.(string), 64)
		}
		if v.Bid != nil {
			ticker.Bid, _ = strconv.ParseFloat(v.Bid.(string), 64)
		}
		if v.High != nil {
			ticker.High, _ = strconv.ParseFloat(v.High.(string), 64)
		}
		if v.Last != nil {
			ticker.Last, _ = strconv.ParseFloat(v.Last.(string), 64)
		}
		if v.Low != nil {
			ticker.Low, _ = strconv.ParseFloat(v.Low.(string), 64)
		}
		if v.Volume != nil {
			ticker.Volume, _ = strconv.ParseFloat(v.Volume.(string), 64)
		}
		result[key] = ticker
		addresses = append(addresses, key)
	}
	return result, nil
}

func (l *LakeBTC) GetOrderBook(currency string) (LakeBTCOrderbook, error) {
	type Response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}
	path := fmt.Sprintf("%s/%s?symbol=%s", LAKEBTC_API_URL, LAKEBTC_ORDERBOOK, common.StringToLower(currency))
	resp := Response{}
	err := common.SendHTTPGetRequest(path, true, l.Verbose, &resp)
	if err != nil {
		return LakeBTCOrderbook{}, err
	}
	orderbook := LakeBTCOrderbook{}

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, LakeBTCOrderbookStructure{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, LakeBTCOrderbookStructure{price, amount})
	}
	return orderbook, nil
}

func (l *LakeBTC) GetTradeHistory(currency string) ([]LakeBTCTradeHistory, error) {
	path := fmt.Sprintf("%s/%s?symbol=%s", LAKEBTC_API_URL, LAKEBTC_TRADES, common.StringToLower(currency))
	resp := []LakeBTCTradeHistory{}
	err := common.SendHTTPGetRequest(path, true, l.Verbose, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (l *LakeBTC) GetAccountInfo() (LakeBTCAccountInfo, error) {
	resp := LakeBTCAccountInfo{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ACCOUNT_INFO, "", &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (l *LakeBTC) Trade(orderType int, amount, price float64, currency string) (LakeBTCTrade, error) {
	resp := LakeBTCTrade{}
	params := strconv.FormatFloat(price, 'f', -1, 64) + "," + strconv.FormatFloat(amount, 'f', -1, 64) + "," + currency
	err := errors.New("")

	if orderType == 1 {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_BUY_ORDER, params, &resp)
	} else {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_SELL_ORDER, params, &resp)
	}

	if err != nil {
		return resp, err
	}

	if resp.Result != "order received" {
		return resp, fmt.Errorf("Unexpected result: %s", resp.Result)
	}

	return resp, nil
}

func (l *LakeBTC) GetOpenOrders() ([]LakeBTCOpenOrders, error) {
	orders := []LakeBTCOpenOrders{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_OPEN_ORDERS, "", &orders)

	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (l *LakeBTC) GetOrders(orders []int64) ([]LakeBTCOrders, error) {
	var ordersStr []string
	for _, x := range orders {
		ordersStr = append(ordersStr, strconv.FormatInt(x, 10))
	}

	resp := []LakeBTCOrders{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ORDERS, common.JoinStrings(ordersStr, ","), &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (l *LakeBTC) CancelOrder(orderID int64) error {
	type Response struct {
		Result bool `json:"Result"`
	}

	resp := Response{}
	params := strconv.FormatInt(orderID, 10)
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_CANCEL_ORDER, params, &resp)
	if err != nil {
		return err
	}

	if resp.Result != true {
		return errors.New("Unable to cancel order.")
	}

	return nil
}

func (l *LakeBTC) GetTrades(timestamp int64) ([]LakeBTCAuthenticaltedTradeHistory, error) {
	params := ""
	if timestamp != 0 {
		params = strconv.FormatInt(timestamp, 10)
	}

	trades := []LakeBTCAuthenticaltedTradeHistory{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_TRADES, params, &trades)
	if err != nil {
		return nil, err
	}

	return trades, nil
}

/* Only for BTC */
func (l *LakeBTC) GetExternalAccounts() ([]LakeBTCExternalAccounts, error) {
	resp := []LakeBTCExternalAccounts{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_EXTERNAL_ACCOUNTS, "", &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

/* Only for BTC */
func (l *LakeBTC) CreateWithdraw(amount float64, accountID int64) (LakeBTCWithdraw, error) {
	resp := LakeBTCWithdraw{}
	params := strconv.FormatFloat(amount, 'f', -1, 64) + ",btc," + strconv.FormatInt(accountID, 10)
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_CREATE_WITHDRAW, params, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (l *LakeBTC) SendAuthenticatedHTTPRequest(method, params string, result interface{}) (err error) {
	if !l.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, l.Name)
	}

	if l.Nonce.Get() == 0 {
		l.Nonce.Set(time.Now().UnixNano())
	} else {
		l.Nonce.Inc()
	}

	req := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=1&method=%s&params=%s", l.Nonce.String(), l.APIKey, method, params)
	hmac := common.GetHMAC(common.HashSHA1, []byte(req), []byte(l.APISecret))

	if l.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", LAKEBTC_API_URL, method, req)
	}

	postData := make(map[string]interface{})
	postData["method"] = method
	postData["id"] = 1
	postData["params"] = common.SplitStrings(params, ",")

	data, err := common.JSONEncode(postData)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["Json-Rpc-Tonce"] = l.Nonce.String()
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(l.APIKey+":"+common.HexEncodeToString(hmac)))
	headers["Content-Type"] = "application/json-rpc"

	resp, err := common.SendHTTPRequest("POST", LAKEBTC_API_URL, headers, strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	if l.Verbose {
		log.Printf("Received raw: %s\n", resp)
	}

	type ErrorResponse struct {
		Error string `json:"error"`
	}

	errResponse := ErrorResponse{}
	err = common.JSONDecode([]byte(resp), &errResponse)
	if err != nil {
		return errors.New("unable to check response for error")
	}

	if errResponse.Error != "" {
		return errors.New(errResponse.Error)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("unable to JSON Unmarshal response")
	}

	return nil
}
