package zb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	zbTradeURL   = "http://api.zb.com/data"
	zbMarketURL  = "https://trade.zb.com/api"
	zbAPIVersion = "v1"

	zbAccountInfo                     = "getAccountInfo"
	zbMarkets                         = "markets"
	zbKline                           = "kline"
	zbOrder                           = "order"
	zbCancelOrder                     = "cancelOrder"
	zbTicker                          = "ticker"
	zbTickers                         = "allTicker"
	zbDepth                           = "depth"
	zbUnfinishedOrdersIgnoreTradeType = "getUnfinishedOrdersIgnoreTradeType"
	zbGetOrdersGet                    = "getOrders"
	zbWithdraw                        = "withdraw"
	zbDepositAddress                  = "getUserAddress"

	zbAuthRate   = 100
	zbUnauthRate = 100
)

// ZB is the overarching type across this package
// 47.91.169.147 api.zb.com
// 47.52.55.212 trade.zb.com
type ZB struct {
	WebsocketConn *websocket.Conn
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (z *ZB) SetDefaults() {
	z.Name = "ZB"
	z.Enabled = false
	z.Fee = 0
	z.Verbose = false
	z.RESTPollingDelay = 10
	z.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	z.RequestCurrencyPairFormat.Delimiter = "_"
	z.RequestCurrencyPairFormat.Uppercase = false
	z.ConfigCurrencyPairFormat.Delimiter = "_"
	z.ConfigCurrencyPairFormat.Uppercase = true
	z.AssetTypes = []string{ticker.Spot}
	z.SupportsAutoPairUpdating = true
	z.SupportsRESTTickerBatching = true
	z.Requester = request.New(z.Name,
		request.NewRateLimit(time.Second*10, zbAuthRate),
		request.NewRateLimit(time.Second*10, zbUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	z.APIUrlDefault = zbTradeURL
	z.APIUrl = z.APIUrlDefault
	z.APIUrlSecondaryDefault = zbMarketURL
	z.APIUrlSecondary = z.APIUrlSecondaryDefault
	z.WebsocketInit()
	z.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketOrderbookSupported |
		exchange.WebsocketTradeDataSupported
}

// Setup sets user configuration
func (z *ZB) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		z.SetEnabled(false)
	} else {
		z.Enabled = true
		z.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		z.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		z.APIAuthPEMKey = exch.APIAuthPEMKey
		z.SetHTTPClientTimeout(exch.HTTPTimeout)
		z.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		z.RESTPollingDelay = exch.RESTPollingDelay
		z.Verbose = exch.Verbose
		z.Websocket.SetWsStatusAndConnection(exch.Websocket)
		z.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		z.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		z.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := z.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = z.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = z.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = z.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = z.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = z.WebsocketSetup(z.WsConnect,
			exch.Name,
			exch.Websocket,
			zbWebsocketAPI,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// SpotNewOrder submits an order to ZB
func (z *ZB) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	var result SpotNewOrderResponse

	vals := url.Values{}
	vals.Set("accesskey", z.APIKey)
	vals.Set("method", "order")
	vals.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	vals.Set("currency", arg.Symbol)
	vals.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	vals.Set("tradeType", string(arg.Type))

	err := z.SendAuthenticatedHTTPRequest(http.MethodGet, vals, &result)
	if err != nil {
		return 0, err
	}
	if result.Code != 1000 {
		return 0, fmt.Errorf("unsucessful new order, message: %s code: %d", result.Message, result.Code)
	}
	newOrderID, err := strconv.ParseInt(result.ID, 10, 64)
	if err != nil {
		return 0, err
	}
	return newOrderID, nil
}

// CancelExistingOrder cancels an order
func (z *ZB) CancelExistingOrder(orderID int64, symbol string) error {
	type response struct {
		Code    int    `json:"code"`    // Result code
		Message string `json:"message"` // Result Message
	}

	vals := url.Values{}
	vals.Set("accesskey", z.APIKey)
	vals.Set("method", "cancelOrder")
	vals.Set("id", strconv.FormatInt(orderID, 10))
	vals.Set("currency", symbol)

	var result response
	err := z.SendAuthenticatedHTTPRequest(http.MethodGet, vals, &result)
	if err != nil {
		return err
	}

	if result.Code != 1000 {
		return errors.New(result.Message)
	}
	return nil
}

// GetAccountInformation returns account information including coin information
// and pricing
func (z *ZB) GetAccountInformation() (AccountsResponse, error) {
	var result AccountsResponse

	vals := url.Values{}
	vals.Set("accesskey", z.APIKey)
	vals.Set("method", "getAccountInfo")

	return result, z.SendAuthenticatedHTTPRequest(http.MethodGet, vals, &result)
}

// GetUnfinishedOrdersIgnoreTradeType returns unfinished orders
func (z *ZB) GetUnfinishedOrdersIgnoreTradeType(currency string, pageindex, pagesize int64) ([]Order, error) {
	var result []Order
	vals := url.Values{}
	vals.Set("accesskey", z.APIKey)
	vals.Set("method", zbUnfinishedOrdersIgnoreTradeType)
	vals.Set("currency", currency)
	vals.Set("pageIndex", strconv.FormatInt(pageindex, 10))
	vals.Set("pageSize", strconv.FormatInt(pagesize, 10))

	err := z.SendAuthenticatedHTTPRequest(http.MethodGet, vals, &result)
	return result, err
}

// GetOrders returns finished orders
func (z *ZB) GetOrders(currency string, pageindex, side int64) ([]Order, error) {
	var response []Order
	vals := url.Values{}
	vals.Set("accesskey", z.APIKey)
	vals.Set("method", zbGetOrdersGet)
	vals.Set("currency", currency)
	vals.Set("pageIndex", strconv.FormatInt(pageindex, 10))
	vals.Set("tradeType", strconv.FormatInt(side, 10))
	return response, z.SendAuthenticatedHTTPRequest(http.MethodGet, vals, &response)
}

// GetMarkets returns market information including pricing, symbols and
// each symbols decimal precision
func (z *ZB) GetMarkets() (map[string]MarketResponseItem, error) {
	endpoint := fmt.Sprintf("%s/%s/%s", z.APIUrl, zbAPIVersion, zbMarkets)

	var res interface{}
	err := z.SendHTTPRequest(endpoint, &res)
	if err != nil {
		return nil, err
	}

	list := res.(map[string]interface{})
	result := map[string]MarketResponseItem{}
	for k, v := range list {
		item := v.(map[string]interface{})
		result[k] = MarketResponseItem{
			AmountScale: item["amountScale"].(float64),
			PriceScale:  item["priceScale"].(float64),
		}
	}
	return result, nil
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
// 获取最新价格
func (z *ZB) GetLatestSpotPrice(symbol string) (float64, error) {
	res, err := z.GetTicker(symbol)

	if err != nil {
		return 0, err
	}

	return res.Ticker.Last, nil
}

// GetTicker returns a ticker for a given symbol
func (z *ZB) GetTicker(symbol string) (TickerResponse, error) {
	urlPath := fmt.Sprintf("%s/%s/%s?market=%s", z.APIUrl, zbAPIVersion, zbTicker, symbol)
	var res TickerResponse

	err := z.SendHTTPRequest(urlPath, &res)
	return res, err
}

// GetTickers returns ticker data for all supported symbols
func (z *ZB) GetTickers() (map[string]TickerChildResponse, error) {
	urlPath := fmt.Sprintf("%s/%s/%s", z.APIUrl, zbAPIVersion, zbTickers)
	resp := make(map[string]TickerChildResponse)

	err := z.SendHTTPRequest(urlPath, &resp)
	return resp, err
}

// GetOrderbook returns the orderbook for a given symbol
func (z *ZB) GetOrderbook(symbol string) (OrderbookResponse, error) {
	urlPath := fmt.Sprintf("%s/%s/%s?market=%s", z.APIUrl, zbAPIVersion, zbDepth, symbol)
	var res OrderbookResponse

	err := z.SendHTTPRequest(urlPath, &res)
	if err != nil {
		return res, err
	}

	// reverse asks data
	var data [][]float64
	for x := len(res.Asks) - 1; x != 0; x-- {
		data = append(data, res.Asks[x])
	}

	res.Asks = data
	return res, nil
}

// GetSpotKline returns Kline data
func (z *ZB) GetSpotKline(arg KlinesRequestParams) (KLineResponse, error) {
	vals := url.Values{}
	vals.Set("type", string(arg.Type))
	vals.Set("market", arg.Symbol)
	if arg.Since != "" {
		vals.Set("since", arg.Since)
	}
	if arg.Size != 0 {
		vals.Set("size", fmt.Sprintf("%d", arg.Size))
	}

	urlPath := fmt.Sprintf("%s/%s/%s?%s", z.APIUrl, zbAPIVersion, zbKline, vals.Encode())

	var res KLineResponse
	var rawKlines map[string]interface{}
	err := z.SendHTTPRequest(urlPath, &rawKlines)
	if err != nil {
		return res, err
	}
	if rawKlines == nil || rawKlines["symbol"] == nil {
		return res, errors.New("zb GetSpotKline rawKlines is nil")
	}

	res.Symbol = rawKlines["symbol"].(string)
	res.MoneyType = rawKlines["moneyType"].(string)

	rawKlineDatasString, _ := json.Marshal(rawKlines["data"].([]interface{}))
	rawKlineDatas := [][]interface{}{}
	if err := json.Unmarshal(rawKlineDatasString, &rawKlineDatas); err != nil {
		return res, errors.New("zb rawKlines unmarshal failed")
	}
	for _, k := range rawKlineDatas {
		ot, err := common.TimeFromUnixTimestampFloat(k[0])
		if err != nil {
			return res, errors.New("zb cannot parse Kline.OpenTime")
		}
		res.Data = append(res.Data, &KLineResponseData{
			ID:        k[0].(float64),
			KlineTime: ot,
			Open:      k[1].(float64),
			High:      k[2].(float64),
			Low:       k[3].(float64),
			Close:     k[4].(float64),
			Volume:    k[5].(float64),
		})
	}

	return res, nil
}

// GetCryptoAddress fetches and returns the deposit address
// NOTE - PLEASE BE AWARE THAT YOU NEED TO GENERATE A DEPOSIT ADDRESS VIA
// LOGGING IN AND NOT BY USING THIS ENDPOINT OTHERWISE THIS WILL GIVE YOU A
// GENERAL ERROR RESPONSE.
func (z *ZB) GetCryptoAddress(currency pair.CurrencyItem) (UserAddress, error) {
	var resp UserAddress

	vals := url.Values{}
	vals.Set("method", zbDepositAddress)
	vals.Set("currency", currency.Lower().String())

	return resp,
		z.SendAuthenticatedHTTPRequest(http.MethodGet, vals, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (z *ZB) SendHTTPRequest(path string, result interface{}) error {
	return z.SendPayload(http.MethodGet, path, nil, nil, result, false, z.Verbose)
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the zb API
func (z *ZB) SendAuthenticatedHTTPRequest(httpMethod string, params url.Values, result interface{}) error {
	if !z.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, z.Name)
	}

	params.Set("accesskey", z.APIKey)

	hmac := common.GetHMAC(common.HashMD5,
		[]byte(params.Encode()),
		[]byte(common.Sha1ToHex(z.APISecret)))

	params.Set("reqTime", fmt.Sprintf("%d", common.UnixMillis(time.Now())))
	params.Set("sign", fmt.Sprintf("%x", hmac))

	urlPath := fmt.Sprintf("%s/%s?%s",
		z.APIUrlSecondary,
		params.Get("method"),
		params.Encode())

	var intermediary json.RawMessage

	errCap := struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}{}

	err := z.SendPayload(httpMethod,
		urlPath,
		nil,
		strings.NewReader(""),
		&intermediary,
		true,
		z.Verbose)
	if err != nil {
		return err
	}

	err = common.JSONDecode(intermediary, &errCap)
	if err == nil {
		if errCap.Code != 0 {
			return fmt.Errorf("sendAuthenticatedHTTPRequest error code: %d message %s",
				errCap.Code,
				errorCode[errCap.Code])
		}
	}

	return common.JSONDecode(intermediary, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (z *ZB) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FirstCurrency)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(purchasePrice, amount float64) (fee float64) {
	fee = 0.002
	return fee * amount * purchasePrice
}

func getWithdrawalFee(currency string) float64 {
	return WithdrawalFees[currency]
}

var errorCode = map[int64]string{
	1000: "Successful call",
	1001: "General error message",
	1002: "internal error",
	1003: "Verification failed",
	1004: "Financial security password lock",
	1005: "The fund security password is incorrect. Please confirm and re-enter.",
	1006: "Real-name certification is awaiting review or review",
	1009: "This interface is being maintained",
	1010: "Not open yet",
	1012: "Insufficient permissions",
	1013: "Can not trade, if you have any questions, please contact online customer service",
	1014: "Cannot be sold during the pre-sale period",
	2002: "Insufficient balance in Bitcoin account",
	2003: "Insufficient balance of Litecoin account",
	2005: "Insufficient balance in Ethereum account",
	2006: "Insufficient balance in ETC currency account",
	2007: "Insufficient balance of BTS currency account",
	2009: "Insufficient account balance",
	3001: "Pending order not found",
	3002: "Invalid amount",
	3003: "Invalid quantity",
	3004: "User does not exist",
	3005: "Invalid parameter",
	3006: "Invalid IP or inconsistent with the bound IP",
	3007: "Request time has expired",
	3008: "Transaction history not found",
	4001: "API interface is locked",
	4002: "Request too frequently",
}

// Withdraw transfers funds
func (z *ZB) Withdraw(currency, address, safepassword string, amount, fees float64, itransfer bool) (string, error) {
	type response struct {
		Code    int    `json:"code"`    // Result code
		Message string `json:"message"` // Result Message
		ID      string `json:"id"`      // Withdrawal ID
	}

	vals := url.Values{}
	vals.Set("accesskey", z.APIKey)
	vals.Set("amount", fmt.Sprintf("%v", amount))
	vals.Set("currency", currency)
	vals.Set("fees", fmt.Sprintf("%v", fees))
	vals.Set("itransfer", fmt.Sprintf("%v", itransfer))
	vals.Set("method", "withdraw")
	vals.Set("recieveAddr", address)
	vals.Set("safePwd", safepassword)

	var resp response
	err := z.SendAuthenticatedHTTPRequest(http.MethodGet, vals, &resp)
	if err != nil {
		return "", err
	}
	if resp.Code != 1000 {
		return "", errors.New(resp.Message)
	}

	return resp.ID, nil
}
