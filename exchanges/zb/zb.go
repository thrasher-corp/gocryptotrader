package zb

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/config"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/request"
	"github.com/pkg/errors"
)

const (
	zbTradeURL   = "http://api.zb.com/data"
	zbMarketURL  = "https://trade.zb.com/api"
	zbAPIVersion = "v1"

	zbAccountInfo = "getAccountInfo"
	zbMarkets     = "markets"
	zbKline       = "kline"
	zbOrder       = "order"
	zbCancelOrder = "cancelOrder"
	zbTicker      = "ticker"
	zbTickers     = "allTicker"
	zbDepth       = "depth"

	zbAuthRate   = 100
	zbUnauthRate = 100
)

// ZB is the overarching type across this package
// 47.91.169.147 api.zb.com
// 47.52.55.212 trade.zb.com
type ZB struct {
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (h *ZB) SetDefaults() {
	h.Name = "ZB"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.Websocket = false
	h.RESTPollingDelay = 10
	h.RequestCurrencyPairFormat.Delimiter = "_"
	h.RequestCurrencyPairFormat.Uppercase = false
	authRateLimit := request.NewRateLimit(time.Second*10, zbUnauthRate)
	authRateLimit.SetRequests(3)
	h.Requester = request.New(h.Name, request.NewRateLimit(time.Second*10, zbAuthRate), authRateLimit, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	// h.Requester = request.New(h.Name, request.NewRateLimit(time.Second*10, zbAuthRate), request.NewRateLimit(time.Second*10, zbUnauthRate), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup sets user configuration
func (h *ZB) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.BaseAsset = exch.BaseAsset
		h.QuoteAsset = exch.QuoteAsset
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		h.SetHTTPClientTimeout(exch.HTTPTimeout)
		h.RESTPollingDelay = exch.RESTPollingDelay
		h.Verbose = exch.Verbose
		h.Websocket = exch.Websocket
		// h.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		// h.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		// h.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")

		// h.RequestCurrencyPairFormat = config.CurrencyPairFormatConfig{
		// 	Delimiter: exch.RequestCurrencyPairFormat.Delimiter,
		// 	Uppercase: exch.RequestCurrencyPairFormat.Uppercase,
		// 	Separator: exch.RequestCurrencyPairFormat.Separator,
		// 	Index:     exch.RequestCurrencyPairFormat.Index,
		// }

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

	err := z.SendAuthenticatedHTTPRequest("GET", zbOrder, vals, &result)
	if err != nil {
		return 0, err
	}
	if result.Code != 1000 {

	}
	newOrderID, err := strconv.ParseInt(result.ID, 10, 64)
	if err != nil {
		return 0, err
	}
	return newOrderID, nil
}

// CancelOrder cancels an order on Huobi
func (z *ZB) CancelOrder(orderID int64, symbol string) error {
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
	err := z.SendAuthenticatedHTTPRequest("GET", zbCancelOrder, vals, &result)
	if err != nil {
		return err
	}

	if result.Code != 1000 {
		return errors.New(result.Message)
	}
	return nil
}

// GetAccountInfo returns account information including coin information
// and pricing
func (z *ZB) GetAccountInfo() (AccountsResponse, error) {
	var result AccountsResponse

	vals := url.Values{}
	vals.Set("accesskey", z.APIKey)
	vals.Set("method", "getAccountInfo")

	err := z.SendAuthenticatedHTTPRequest("GET", zbAccountInfo, vals, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetMarkets returns market information including pricing, symbols and
// each symbols decimal precision
func (z *ZB) GetMarkets() (map[string]MarketResponseItem, error) {
	url := fmt.Sprintf("%s/%s/%s", z.APIUrl, zbAPIVersion, zbMarkets)

	var res interface{}
	err := z.SendHTTPRequest(url, &res)
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
	url := fmt.Sprintf("%s/%s/%s?market=%s", z.APIUrl, zbAPIVersion, zbTicker, symbol)
	var res TickerResponse

	err := z.SendHTTPRequest(url, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

// GetTickers returns ticker data for all supported symbols
func (z *ZB) GetTickers() (map[string]TickerChildResponse, error) {
	url := fmt.Sprintf("%s/%s/%s", z.APIUrl, zbAPIVersion, zbTickers)
	resp := make(map[string]TickerChildResponse)

	err := z.SendHTTPRequest(url, &resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetOrderbook returns the orderbook for a given symbol
func (z *ZB) GetOrderbook(symbol string) (OrderbookResponse, error) {
	url := fmt.Sprintf("%s/%s/%s?market=%s", z.APIUrl, zbAPIVersion, zbDepth, symbol)
	var res OrderbookResponse

	err := z.SendHTTPRequest(url, &res)
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

	url := fmt.Sprintf("%s/%s/%s?%s", z.APIUrl, zbAPIVersion, zbKline, vals.Encode())

	var res KLineResponse
	var rawKlines map[string]interface{}
	err := z.SendHTTPRequest(url, &rawKlines)
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

// SendHTTPRequest sends an unauthenticated HTTP request
func (z *ZB) SendHTTPRequest(path string, result interface{}) error {
	return z.SendPayload("GET", path, nil, nil, result, false, z.Verbose)
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the zb API
func (z *ZB) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !z.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, z.Name)
	}

	mapParams2Sign := url.Values{}
	mapParams2Sign.Set("accesskey", z.APIKey)
	mapParams2Sign.Set("method", values.Get("method"))

	values.Set("sign",
		common.HexEncodeToString(common.GetHMAC(common.HashMD5,
			[]byte(values.Encode()),
			[]byte(common.Sha1ToHex(z.APISecret)))))

	values.Set("reqTime", fmt.Sprintf("%d", time.Now().UnixNano()/1e6))

	url := fmt.Sprintf("%s/%s?%s",
		z.APIUrlSecondaryDefault,
		endpoint,
		values.Encode())

	return z.SendPayload(method,
		url,
		nil,
		strings.NewReader(""),
		result,
		true,
		z.Verbose)
}
