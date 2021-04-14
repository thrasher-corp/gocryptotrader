package zb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	zbTradeURL                        = "http://api.zb.live"
	zbMarketURL                       = "https://trade.zb.live/api"
	zbAPIVersion                      = "v1"
	zbData                            = "data"
	zbAccountInfo                     = "getAccountInfo"
	zbMarkets                         = "markets"
	zbKline                           = "kline"
	zbOrder                           = "order"
	zbCancelOrder                     = "cancelOrder"
	zbTicker                          = "ticker"
	zbTrades                          = "trades"
	zbTickers                         = "allTicker"
	zbDepth                           = "depth"
	zbUnfinishedOrdersIgnoreTradeType = "getUnfinishedOrdersIgnoreTradeType"
	zbGetOrdersGet                    = "getOrders"
	zbWithdraw                        = "withdraw"
	zbDepositAddress                  = "getUserAddress"
)

// ZB is the overarching type across this package
// 47.91.169.147 api.zb.com
// 47.52.55.212 trade.zb.com
type ZB struct {
	exchange.Base
}

// SpotNewOrder submits an order to ZB
func (z *ZB) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	var result SpotNewOrderResponse

	vals := url.Values{}
	vals.Set("accesskey", z.API.Credentials.Key)
	vals.Set("method", "order")
	vals.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	vals.Set("currency", arg.Symbol)
	vals.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	vals.Set("tradeType", string(arg.Type))

	err := z.SendAuthenticatedHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, vals, &result, request.Auth)
	if err != nil {
		return 0, err
	}
	if result.Code != 1000 {
		return 0, fmt.Errorf("unsuccessful new order, message: %s code: %d", result.Message, result.Code)
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
	vals.Set("accesskey", z.API.Credentials.Key)
	vals.Set("method", "cancelOrder")
	vals.Set("id", strconv.FormatInt(orderID, 10))
	vals.Set("currency", symbol)

	var result response
	err := z.SendAuthenticatedHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, vals, &result, request.Auth)
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
	vals.Set("accesskey", z.API.Credentials.Key)
	vals.Set("method", "getAccountInfo")

	return result, z.SendAuthenticatedHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, vals, &result, request.Auth)
}

// GetUnfinishedOrdersIgnoreTradeType returns unfinished orders
func (z *ZB) GetUnfinishedOrdersIgnoreTradeType(currency string, pageindex, pagesize int64) ([]Order, error) {
	var result []Order
	vals := url.Values{}
	vals.Set("accesskey", z.API.Credentials.Key)
	vals.Set("method", zbUnfinishedOrdersIgnoreTradeType)
	vals.Set("currency", currency)
	vals.Set("pageIndex", strconv.FormatInt(pageindex, 10))
	vals.Set("pageSize", strconv.FormatInt(pagesize, 10))

	err := z.SendAuthenticatedHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, vals, &result, request.Auth)
	return result, err
}

// GetOrders returns finished orders
func (z *ZB) GetOrders(currency string, pageindex, side int64) ([]Order, error) {
	var response []Order
	vals := url.Values{}
	vals.Set("accesskey", z.API.Credentials.Key)
	vals.Set("method", zbGetOrdersGet)
	vals.Set("currency", currency)
	vals.Set("pageIndex", strconv.FormatInt(pageindex, 10))
	vals.Set("tradeType", strconv.FormatInt(side, 10))
	return response, z.SendAuthenticatedHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, vals, &response, request.Auth)
}

// GetMarkets returns market information including pricing, symbols and
// each symbols decimal precision
func (z *ZB) GetMarkets() (map[string]MarketResponseItem, error) {
	endpoint := fmt.Sprintf("/%s/%s/%s", zbData, zbAPIVersion, zbMarkets)

	var res map[string]MarketResponseItem
	err := z.SendHTTPRequest(exchange.RestSpot, endpoint, &res, request.UnAuth)
	if err != nil {
		return nil, err
	}

	return res, nil
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
	urlPath := fmt.Sprintf("/%s/%s/%s?market=%s", zbData, zbAPIVersion, zbTicker, symbol)
	var res TickerResponse
	err := z.SendHTTPRequest(exchange.RestSpot, urlPath, &res, request.UnAuth)
	return res, err
}

// GetTrades returns trades for a given symbol
func (z *ZB) GetTrades(symbol string) (TradeHistory, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s?market=%s", zbData, zbAPIVersion, zbTrades, symbol)
	var res TradeHistory
	err := z.SendHTTPRequest(exchange.RestSpot, urlPath, &res, request.UnAuth)
	return res, err
}

// GetTickers returns ticker data for all supported symbols
func (z *ZB) GetTickers() (map[string]TickerChildResponse, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s", zbData, zbAPIVersion, zbTickers)
	resp := make(map[string]TickerChildResponse)
	err := z.SendHTTPRequest(exchange.RestSpot, urlPath, &resp, request.UnAuth)
	return resp, err
}

// GetOrderbook returns the orderbook for a given symbol
func (z *ZB) GetOrderbook(symbol string) (OrderbookResponse, error) {
	urlPath := fmt.Sprintf("/%s/%s/%s?market=%s", zbData, zbAPIVersion, zbDepth, symbol)
	var res OrderbookResponse

	err := z.SendHTTPRequest(exchange.RestSpot, urlPath, &res, request.UnAuth)
	if err != nil {
		return res, err
	}

	if len(res.Asks) == 0 {
		return res, fmt.Errorf("ZB GetOrderbook asks is empty")
	}

	if len(res.Bids) == 0 {
		return res, fmt.Errorf("ZB GetOrderbook bids is empty")
	}

	// reverse asks data
	var data [][]float64
	for x := len(res.Asks); x > 0; x-- {
		data = append(data, res.Asks[x-1])
	}

	res.Asks = data
	return res, nil
}

// GetSpotKline returns Kline data
func (z *ZB) GetSpotKline(arg KlinesRequestParams) (KLineResponse, error) {
	vals := url.Values{}
	vals.Set("type", arg.Type)
	vals.Set("market", arg.Symbol)
	if arg.Since > 0 {
		vals.Set("since", strconv.FormatInt(arg.Since, 10))
	}
	if arg.Size != 0 {
		vals.Set("size", fmt.Sprintf("%d", arg.Size))
	}

	urlPath := fmt.Sprintf("/%s/%s/%s?%s", zbData, zbAPIVersion, zbKline, vals.Encode())

	var res KLineResponse
	var rawKlines map[string]interface{}
	err := z.SendHTTPRequest(exchange.RestSpot, urlPath, &rawKlines, klineFunc)
	if err != nil {
		return res, err
	}
	if rawKlines == nil || rawKlines["symbol"] == nil {
		return res, errors.New("zb GetSpotKline rawKlines is nil")
	}

	res.Symbol = rawKlines["symbol"].(string)
	res.MoneyType = rawKlines["moneyType"].(string)

	rawKlineDatasString, _ := json.Marshal(rawKlines["data"].([]interface{}))
	var rawKlineDatas [][]interface{}
	if err := json.Unmarshal(rawKlineDatasString, &rawKlineDatas); err != nil {
		return res, errors.New("zb rawKlines unmarshal failed")
	}
	for _, k := range rawKlineDatas {
		ot, err := convert.TimeFromUnixTimestampFloat(k[0])
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
func (z *ZB) GetCryptoAddress(currency currency.Code) (UserAddress, error) {
	var resp UserAddress

	vals := url.Values{}
	vals.Set("method", zbDepositAddress)
	vals.Set("currency", currency.Lower().String())

	return resp,
		z.SendAuthenticatedHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, vals, &resp, request.Auth)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (z *ZB) SendHTTPRequest(ep exchange.URL, path string, result interface{}, f request.EndpointLimit) error {
	endpoint, err := z.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return z.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       z.Verbose,
		HTTPDebugging: z.HTTPDebugging,
		HTTPRecording: z.HTTPRecording,
		Endpoint:      f,
	})
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the zb API
func (z *ZB) SendAuthenticatedHTTPRequest(ep exchange.URL, httpMethod string, params url.Values, result interface{}, f request.EndpointLimit) error {
	if !z.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", z.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := z.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	params.Set("accesskey", z.API.Credentials.Key)

	hmac := crypto.GetHMAC(crypto.HashMD5,
		[]byte(params.Encode()),
		[]byte(crypto.Sha1ToHex(z.API.Credentials.Secret)))

	now := time.Now()
	params.Set("reqTime", fmt.Sprintf("%d", convert.UnixMillis(now)))
	params.Set("sign", fmt.Sprintf("%x", hmac))

	urlPath := fmt.Sprintf("%s/%s?%s",
		endpoint,
		params.Get("method"),
		params.Encode())

	var intermediary json.RawMessage

	errCap := struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}{}

	// Expiry of timestamp doesn't appear to be documented, so making a reasonable assumption
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(15*time.Second))
	defer cancel()
	err = z.SendPayload(ctx, &request.Item{
		Method:        httpMethod,
		Path:          urlPath,
		Body:          strings.NewReader(""),
		Result:        &intermediary,
		AuthRequest:   true,
		Verbose:       z.Verbose,
		HTTPDebugging: z.HTTPDebugging,
		HTTPRecording: z.HTTPRecording,
		Endpoint:      f,
	})
	if err != nil {
		return err
	}

	err = json.Unmarshal(intermediary, &errCap)
	if err == nil {
		if errCap.Code > 1000 {
			return fmt.Errorf("sendAuthenticatedHTTPRequest error code: %d message %s",
				errCap.Code,
				errorCode[errCap.Code])
		}
	}

	return json.Unmarshal(intermediary, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (z *ZB) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

func calculateTradingFee(purchasePrice, amount float64) (fee float64) {
	fee = 0.002
	return fee * amount * purchasePrice
}

func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
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
	vals.Set("accesskey", z.API.Credentials.Key)
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	vals.Set("currency", currency)
	vals.Set("fees", strconv.FormatFloat(fees, 'f', -1, 64))
	vals.Set("itransfer", strconv.FormatBool(itransfer))
	vals.Set("method", "withdraw")
	vals.Set("receiveAddr", address)
	vals.Set("safePwd", safepassword)

	var resp response
	err := z.SendAuthenticatedHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, vals, &resp, request.Auth)
	if err != nil {
		return "", err
	}
	if resp.Code != 1000 {
		return "", errors.New(resp.Message)
	}

	return resp.ID, nil
}
