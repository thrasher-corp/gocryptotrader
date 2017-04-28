package btcc

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
	BTCC_API_URL                  = "https://api.btcchina.com/"
	BTCC_API_AUTHENTICATED_METHOD = "api_trade_v1.php"
	BTCC_API_VER                  = "2.0.1.3"
	BTCC_ORDER_BUY                = "buyOrder2"
	BTCC_ORDER_SELL               = "sellOrder2"
	BTCC_ORDER_CANCEL             = "cancelOrder"
	BTCC_ICEBERG_BUY              = "buyIcebergOrder"
	BTCC_ICEBERG_SELL             = "sellIcebergOrder"
	BTCC_ICEBERG_ORDER            = "getIcebergOrder"
	BTCC_ICEBERG_ORDERS           = "getIcebergOrders"
	BTCC_ICEBERG_CANCEL           = "cancelIcebergOrder"
	BTCC_ACCOUNT_INFO             = "getAccountInfo"
	BTCC_DEPOSITS                 = "getDeposits"
	BTCC_MARKETDEPTH              = "getMarketDepth2"
	BTCC_ORDER                    = "getOrder"
	BTCC_ORDERS                   = "getOrders"
	BTCC_TRANSACTIONS             = "getTransactions"
	BTCC_WITHDRAWAL               = "getWithdrawal"
	BTCC_WITHDRAWALS              = "getWithdrawals"
	BTCC_WITHDRAWAL_REQUEST       = "requestWithdrawal"
	BTCC_STOPORDER_BUY            = "buyStopOrder"
	BTCC_STOPORDER_SELL           = "sellStopOrder"
	BTCC_STOPORDER_CANCEL         = "cancelStopOrder"
	BTCC_STOPORDER                = "getStopOrder"
	BTCC_STOPORDERS               = "getStopOrders"
)

type BTCC struct {
	exchange.ExchangeBase
}

func (b *BTCC) SetDefaults() {
	b.Name = "BTCC"
	b.Enabled = false
	b.Fee = 0
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
}

//Setup is run on startup to setup exchange with config values
func (b *BTCC) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (b *BTCC) GetFee() float64 {
	return b.Fee
}

func (b *BTCC) GetTicker(symbol string) (BTCCTicker, error) {
	type Response struct {
		Ticker BTCCTicker
	}

	resp := Response{}
	req := fmt.Sprintf("%sdata/ticker?market=%s", BTCC_API_URL, symbol)
	err := common.SendHTTPGetRequest(req, true, &resp)
	if err != nil {
		return BTCCTicker{}, err
	}
	return resp.Ticker, nil
}

func (b *BTCC) GetTradesLast24h(symbol string) bool {
	req := fmt.Sprintf("%sdata/trades?market=%s", BTCC_API_URL, symbol)
	err := common.SendHTTPGetRequest(req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *BTCC) GetTradeHistory(symbol string, limit, sinceTid int64, time time.Time) bool {
	req := fmt.Sprintf("%sdata/historydata?market=%s", BTCC_API_URL, symbol)
	v := url.Values{}

	if limit > 0 {
		v.Set("limit", strconv.FormatInt(limit, 10))
	}
	if sinceTid > 0 {
		v.Set("since", strconv.FormatInt(sinceTid, 10))
	}
	if !time.IsZero() {
		v.Set("sincetype", strconv.FormatInt(time.Unix(), 10))
	}

	req = common.EncodeURLValues(req, v)
	err := common.SendHTTPGetRequest(req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *BTCC) GetOrderBook(symbol string, limit int) (BTCCOrderbook, error) {
	result := BTCCOrderbook{}
	req := fmt.Sprintf("%sdata/orderbook?market=%s&limit=%d", BTCC_API_URL, symbol, limit)
	err := common.SendHTTPGetRequest(req, true, &result)
	if err != nil {
		return BTCCOrderbook{}, err
	}

	return result, nil
}

func (b *BTCC) GetAccountInfo(infoType string) {
	params := make([]interface{}, 0)

	if len(infoType) > 0 {
		params = append(params, infoType)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_ACCOUNT_INFO, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) PlaceOrder(buyOrder bool, price, amount float64, market string) {
	params := make([]interface{}, 0)
	params = append(params, strconv.FormatFloat(price, 'f', -1, 64))
	params = append(params, strconv.FormatFloat(amount, 'f', -1, 64))

	if len(market) > 0 {
		params = append(params, market)
	}

	req := BTCC_ORDER_BUY
	if !buyOrder {
		req = BTCC_ORDER_SELL
	}

	err := b.SendAuthenticatedHTTPRequest(req, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) CancelOrder(orderID int64, market string) {
	params := make([]interface{}, 0)
	params = append(params, orderID)

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_ORDER_CANCEL, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetDeposits(currency string, pending bool) {
	params := make([]interface{}, 0)
	params = append(params, currency)

	if pending {
		params = append(params, pending)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_DEPOSITS, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetMarketDepth(market string, limit int64) {
	params := make([]interface{}, 0)

	if limit > 0 {
		params = append(params, limit)
	}

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_MARKETDEPTH, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetOrder(orderID int64, market string, detailed bool) {
	params := make([]interface{}, 0)
	params = append(params, orderID)

	if len(market) > 0 {
		params = append(params, market)
	}

	if detailed {
		params = append(params, detailed)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_ORDER, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetOrders(openonly bool, market string, limit, offset, since int64, detailed bool) {
	params := make([]interface{}, 0)

	if openonly {
		params = append(params, openonly)
	}

	if len(market) > 0 {
		params = append(params, market)
	}

	if limit > 0 {
		params = append(params, limit)
	}

	if offset > 0 {
		params = append(params, offset)
	}

	if since > 0 {
		params = append(params, since)
	}

	if detailed {
		params = append(params, detailed)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_ORDERS, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetTransactions(transType string, limit, offset, since int64, sinceType string) {
	params := make([]interface{}, 0)

	if len(transType) > 0 {
		params = append(params, transType)
	}

	if limit > 0 {
		params = append(params, limit)
	}

	if offset > 0 {
		params = append(params, offset)
	}

	if since > 0 {
		params = append(params, since)
	}

	if len(sinceType) > 0 {
		params = append(params, sinceType)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_TRANSACTIONS, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetWithdrawal(withdrawalID int64, currency string) {
	params := make([]interface{}, 0)
	params = append(params, withdrawalID)

	if len(currency) > 0 {
		params = append(params, currency)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_WITHDRAWAL, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetWithdrawals(currency string, pending bool) {
	params := make([]interface{}, 0)
	params = append(params, currency)

	if pending {
		params = append(params, pending)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_WITHDRAWALS, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) RequestWithdrawal(currency string, amount float64) {
	params := make([]interface{}, 0)
	params = append(params, currency)
	params = append(params, amount)

	err := b.SendAuthenticatedHTTPRequest(BTCC_WITHDRAWAL_REQUEST, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) IcebergOrder(buyOrder bool, price, amount, discAmount, variance float64, market string) {
	params := make([]interface{}, 0)
	params = append(params, strconv.FormatFloat(price, 'f', -1, 64))
	params = append(params, strconv.FormatFloat(amount, 'f', -1, 64))
	params = append(params, strconv.FormatFloat(discAmount, 'f', -1, 64))
	params = append(params, strconv.FormatFloat(variance, 'f', -1, 64))

	if len(market) > 0 {
		params = append(params, market)
	}

	req := BTCC_ICEBERG_BUY
	if !buyOrder {
		req = BTCC_ICEBERG_SELL
	}

	err := b.SendAuthenticatedHTTPRequest(req, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetIcebergOrder(orderID int64, market string) {
	params := make([]interface{}, 0)
	params = append(params, orderID)

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_ICEBERG_ORDER, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetIcebergOrders(limit, offset int64, market string) {
	params := make([]interface{}, 0)

	if limit > 0 {
		params = append(params, limit)
	}

	if offset > 0 {
		params = append(params, offset)
	}

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_ICEBERG_ORDERS, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) CancelIcebergOrder(orderID int64, market string) {
	params := make([]interface{}, 0)
	params = append(params, orderID)

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_ICEBERG_CANCEL, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) PlaceStopOrder(buyOder bool, stopPrice, price, amount, trailingAmt, trailingPct float64, market string) {
	params := make([]interface{}, 0)

	if stopPrice > 0 {
		params = append(params, stopPrice)
	}

	params = append(params, strconv.FormatFloat(price, 'f', -1, 64))
	params = append(params, strconv.FormatFloat(amount, 'f', -1, 64))

	if trailingAmt > 0 {
		params = append(params, strconv.FormatFloat(trailingAmt, 'f', -1, 64))
	}

	if trailingPct > 0 {
		params = append(params, strconv.FormatFloat(trailingPct, 'f', -1, 64))
	}

	if len(market) > 0 {
		params = append(params, market)
	}

	req := BTCC_STOPORDER_BUY
	if !buyOder {
		req = BTCC_STOPORDER_SELL
	}

	err := b.SendAuthenticatedHTTPRequest(req, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetStopOrder(orderID int64, market string) {
	params := make([]interface{}, 0)
	params = append(params, orderID)

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_STOPORDER, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) GetStopOrders(status, orderType string, stopPrice float64, limit, offset int64, market string) {
	params := make([]interface{}, 0)

	if len(status) > 0 {
		params = append(params, status)
	}

	if len(orderType) > 0 {
		params = append(params, orderType)
	}

	if stopPrice > 0 {
		params = append(params, stopPrice)
	}

	if limit > 0 {
		params = append(params, limit)
	}

	if offset > 0 {
		params = append(params, limit)
	}

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_STOPORDERS, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) CancelStopOrder(orderID int64, market string) {
	params := make([]interface{}, 0)
	params = append(params, orderID)

	if len(market) > 0 {
		params = append(params, market)
	}

	err := b.SendAuthenticatedHTTPRequest(BTCC_STOPORDER_CANCEL, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) SendAuthenticatedHTTPRequest(method string, params []interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:16]
	encoded := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=%d&method=%s&params=", nonce, b.APIKey, 1, method)

	if len(params) == 0 {
		params = make([]interface{}, 0)
	} else {
		items := make([]string, 0)
		for _, x := range params {
			xType := fmt.Sprintf("%T", x)
			switch xType {
			case "int64", "int":
				{
					items = append(items, fmt.Sprintf("%d", x))
				}
			case "string":
				{
					items = append(items, fmt.Sprintf("%s", x))
				}
			case "float64":
				{
					items = append(items, fmt.Sprintf("%f", x))
				}
			case "bool":
				{
					if x == true {
						items = append(items, "1")
					} else {
						items = append(items, "")
					}
				}
			default:
				{
					items = append(items, fmt.Sprintf("%v", x))
				}
			}
		}
		encoded += common.JoinStrings(items, ",")
	}
	if b.Verbose {
		log.Println(encoded)
	}

	hmac := common.GetHMAC(common.HASH_SHA1, []byte(encoded), []byte(b.APISecret))
	postData := make(map[string]interface{})
	postData["method"] = method
	postData["params"] = params
	postData["id"] = 1
	apiURL := BTCC_API_URL + BTCC_API_AUTHENTICATED_METHOD
	data, err := common.JSONEncode(postData)

	if err != nil {
		return errors.New("Unable to JSON Marshal POST data")
	}

	if b.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", apiURL, method, data)
	}

	headers := make(map[string]string)
	headers["Content-type"] = "application/json-rpc"
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(b.APIKey+":"+common.HexEncodeToString(hmac)))
	headers["Json-Rpc-Tonce"] = nonce

	resp, err := common.SendHTTPRequest("POST", apiURL, headers, strings.NewReader(string(data)))

	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recv'd :%s\n", resp)
	}

	return nil
}
