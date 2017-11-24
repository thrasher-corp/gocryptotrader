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
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	btccAPIUrl                 = "https://api.btcchina.com/"
	btccAPIAuthenticatedMethod = "api_trade_v1.php"
	btccAPIVersion             = "2.0.1.3"
	btccOrderBuy               = "buyOrder2"
	btccOrderSell              = "sellOrder2"
	btccOrderCancel            = "cancelOrder"
	btccIcebergBuy             = "buyIcebergOrder"
	btccIcebergSell            = "sellIcebergOrder"
	btccIcebergOrder           = "getIcebergOrder"
	btccIcebergOrders          = "getIcebergOrders"
	btccIcebergCancel          = "cancelIcebergOrder"
	btccAccountInfo            = "getAccountInfo"
	btccDeposits               = "getDeposits"
	btccMarketdepth            = "getMarketDepth2"
	btccOrder                  = "getOrder"
	btccOrders                 = "getOrders"
	btccTransactions           = "getTransactions"
	btccWithdrawal             = "getWithdrawal"
	btccWithdrawals            = "getWithdrawals"
	btccWithdrawalRequest      = "requestWithdrawal"
	btccStoporderBuy           = "buyStopOrder"
	btccStoporderSell          = "sellStopOrder"
	btccStoporderCancel        = "cancelStopOrder"
	btccStoporder              = "getStopOrder"
	btccStoporders             = "getStopOrders"
)

// BTCC is the main overaching type across the BTCC package
type BTCC struct {
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (b *BTCC) SetDefaults() {
	b.Name = "BTCC"
	b.Enabled = false
	b.Fee = 0
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = false
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
}

// Setup is run on startup to setup exchange with config values
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
		err := b.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the fees associated with transactions
func (b *BTCC) GetFee() float64 {
	return b.Fee
}

// GetTicker returns ticker information
// currencyPair - Example "btccny", "ltccny" or "ltcbtc"
func (b *BTCC) GetTicker(currencyPair string) (Ticker, error) {
	resp := Response{}
	req := fmt.Sprintf("%sdata/ticker?market=%s", btccAPIUrl, currencyPair)

	return resp.Ticker, common.SendHTTPGetRequest(req, true, b.Verbose, &resp)
}

// GetTradesLast24h returns the trades executed on the exchange over the past
// 24 hours by currency pair
// currencyPair - Example "btccny", "ltccny" or "ltcbtc"
func (b *BTCC) GetTradesLast24h(currencyPair string) ([]Trade, error) {
	trades := []Trade{}
	req := fmt.Sprintf("%sdata/trades?market=%s", btccAPIUrl, currencyPair)

	return trades, common.SendHTTPGetRequest(req, true, b.Verbose, &trades)
}

// GetTradeHistory returns trade history data
// currencyPair - Example "btccny", "ltccny" or "ltcbtc"
// limit - limits the returned trades example "10"
// sinceTid - returns trade records starting from id supplied example "5000"
// time - returns trade records starting from unix time 1406794449
func (b *BTCC) GetTradeHistory(currencyPair string, limit, sinceTid int64, time time.Time) ([]Trade, error) {
	trades := []Trade{}

	req := fmt.Sprintf("%sdata/historydata?market=%s", btccAPIUrl, currencyPair)

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

	return trades, common.SendHTTPGetRequest(req, true, b.Verbose, &trades)
}

// GetOrderBook returns current market order book
// currencyPair - Example "btccny", "ltccny" or "ltcbtc"
// limit - limits the returned trades example "10" if 0 will return full
// orderbook
func (b *BTCC) GetOrderBook(currencyPair string, limit int) (Orderbook, error) {
	result := Orderbook{}

	req := fmt.Sprintf("%sdata/orderbook?market=%s&limit=%d", btccAPIUrl, currencyPair, limit)
	if limit == 0 {
		req = fmt.Sprintf("%sdata/orderbook?market=%s", btccAPIUrl, currencyPair)
	}

	return result, common.SendHTTPGetRequest(req, true, b.Verbose, &result)
}

func (b *BTCC) GetAccountInfo(infoType string) error {
	params := make([]interface{}, 0)

	if len(infoType) > 0 {
		params = append(params, infoType)
	}

	return b.SendAuthenticatedHTTPRequest(btccAccountInfo, params)
}

func (b *BTCC) PlaceOrder(buyOrder bool, price, amount float64, market string) {
	params := make([]interface{}, 0)
	params = append(params, strconv.FormatFloat(price, 'f', -1, 64))
	params = append(params, strconv.FormatFloat(amount, 'f', -1, 64))

	if len(market) > 0 {
		params = append(params, market)
	}

	req := btccOrderBuy
	if !buyOrder {
		req = btccOrderSell
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

	err := b.SendAuthenticatedHTTPRequest(btccOrderCancel, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccDeposits, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccMarketdepth, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccOrder, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccOrders, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccTransactions, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccWithdrawal, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccWithdrawals, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) RequestWithdrawal(currency string, amount float64) {
	params := make([]interface{}, 0)
	params = append(params, currency)
	params = append(params, amount)

	err := b.SendAuthenticatedHTTPRequest(btccWithdrawalRequest, params)

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

	req := btccIcebergBuy
	if !buyOrder {
		req = btccIcebergSell
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

	err := b.SendAuthenticatedHTTPRequest(btccIcebergOrder, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccIcebergOrders, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccIcebergCancel, params)

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

	req := btccStoporderBuy
	if !buyOder {
		req = btccStoporderSell
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

	err := b.SendAuthenticatedHTTPRequest(btccStoporder, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccStoporders, params)

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

	err := b.SendAuthenticatedHTTPRequest(btccStoporderCancel, params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCC) SendAuthenticatedHTTPRequest(method string, params []interface{}) (err error) {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	if b.Nonce.Get() == 0 {
		b.Nonce.Set(time.Now().UnixNano())
	} else {
		b.Nonce.Inc()
	}
	encoded := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=%d&method=%s&params=", b.Nonce.String()[0:16], b.APIKey, 1, method)

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

	hmac := common.GetHMAC(common.HashSHA1, []byte(encoded), []byte(b.APISecret))
	postData := make(map[string]interface{})
	postData["method"] = method
	postData["params"] = params
	postData["id"] = 1
	apiURL := btccAPIUrl + btccAPIAuthenticatedMethod
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
	headers["Json-Rpc-Tonce"] = b.Nonce.String()

	resp, err := common.SendHTTPRequest("POST", apiURL, headers, strings.NewReader(string(data)))

	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recv'd :%s\n", resp)
	}

	return nil
}
