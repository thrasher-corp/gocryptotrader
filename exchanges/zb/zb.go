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
	// zbBalances    = "private/balances"
	zbCancelOrder = "cancelOrder"
	zbTicker      = "ticker"

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
func (h *ZB) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	var result SpotNewOrderResponse

	//
	vals := url.Values{}
	vals.Set("accesskey", h.APIKey)
	vals.Set("method", "order")
	vals.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	vals.Set("currency", arg.Symbol)
	vals.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	vals.Set("tradeType", string(arg.Type))

	err := h.SendAuthenticatedHTTPRequest("GET", zbOrder, vals, &result)
	if err != nil {
		return 0, err
	}
	if result.Code != 1000 {

	}
	newOrderID, err := strconv.ParseInt(result.ID, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "转换订单ID出错")
	}
	return newOrderID, nil
}

// CancelOrder cancels an order on Huobi
func (h *ZB) CancelOrder(orderID int64) error {
	type response struct {
		Code    int    `json:"code"`    //返回代码
		Message string `json:"message"` //提示信息
	}

	//
	vals := url.Values{}
	vals.Set("accesskey", h.APIKey)
	vals.Set("method", "cancelOrder")
	vals.Set("id", strconv.FormatInt(orderID, 10))
	vals.Set("currency", h.GetSymbol())

	var result response

	err := h.SendAuthenticatedHTTPRequest("GET", zbCancelOrder, vals, &result)
	if err != nil {
		return err
	}

	if result.Code != 1000 {
		return errors.New(result.Message)
	}
	return nil
}

// GetAccountInfo 获取已开启的市场信息，包括价格、数量小数点位数
func (h *ZB) GetAccountInfo() (AccountsResponse, error) {
	var result AccountsResponse

	// var res interface{}
	vals := url.Values{}
	vals.Set("accesskey", h.APIKey)
	vals.Set("method", "getAccountInfo")

	err := h.SendAuthenticatedHTTPRequest("GET", zbAccountInfo, vals, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// GetMarkets 获取已开启的市场信息，包括价格、数量小数点位数
func (h *ZB) GetMarkets() (map[string]MarketResponseItem, error) {

	url := fmt.Sprintf("%s/%s/%s", zbTradeURL, zbAPIVersion, zbMarkets)

	var res interface{}
	err := h.SendHTTPRequest(url, &res)
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
func (h *ZB) GetLatestSpotPrice(symbol string) (float64, error) {
	res, err := h.GetTicker(symbol)

	if err != nil {
		return 0, err
	}

	return common.FloatFromString(res.Ticket.Last)
}

// GetTicker K 线
func (h *ZB) GetTicker(symbol string) (TicketResponse, error) {

	url := fmt.Sprintf("%s/%s/%s?market=%s", zbTradeURL, zbAPIVersion, zbTicker, symbol)

	var res TicketResponse

	err := h.SendHTTPRequest(url, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

// GetSpotKline K 线
func (h *ZB) GetSpotKline(arg KlinesRequestParams) (KLineResponse, error) {

	// var res interface{}
	vals := url.Values{}
	vals.Set("type", string(arg.Type))
	vals.Set("market", arg.Symbol)
	if arg.Since != "" {
		vals.Set("since", arg.Since)
	}
	if arg.Size != 0 {
		vals.Set("size", fmt.Sprintf("%d", arg.Size))
	}

	// url := fmt.Sprintf("%s/%s/%s?market=%s", zbTradeURL, zbAPIVersion, zbKline, arg.Symbol)
	url := fmt.Sprintf("%s/%s/%s?%s", zbTradeURL, zbAPIVersion, zbKline, vals.Encode())

	var res KLineResponse
	var rawKlines map[string]interface{}
	err := h.SendHTTPRequest(url, &rawKlines)
	if err != nil {
		return res, err
	}
	if rawKlines == nil || rawKlines["symbol"] == nil {
		return res, errors.Wrap(err, "rawKlines is nil")
	}

	res.Symbol = rawKlines["symbol"].(string)
	res.MoneyType = rawKlines["moneyType"].(string)

	//对于 Data数据，再次解析
	rawKlineDatasString, _ := json.Marshal(rawKlines["data"].([]interface{}))
	rawKlineDatas := [][]interface{}{}
	if err := json.Unmarshal(rawKlineDatasString, &rawKlineDatas); err != nil {
		return res, errors.Wrap(err, "rawKlines unmarshal failed")
	}
	for _, k := range rawKlineDatas {
		// s := strconv.FormatFloat(k[0].(float64), 'E', -1, 64)
		//time.Unix(_item.Timestamp, 0).Format("2006-01-02 15:04:05")
		ot, err := common.TimeFromUnixTimestampFloat(k[0])
		if err != nil {
			return res, errors.Wrap(err, "cannot parse Kline.OpenTime")
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
func (h *ZB) SendHTTPRequest(path string, result interface{}) error {
	return h.SendPayload("GET", path, nil, nil, result, false, h.Verbose)
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the zb API
func (h *ZB) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}

	headers := make(map[string]string)
	headers["User-Agent"] = "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36"

	mapParams2Sign := url.Values{}
	mapParams2Sign.Set("accesskey", h.APIKey)
	mapParams2Sign.Set("method", values.Get("method"))
	values.Set("sign", common.HexEncodeToString(common.GetHMAC(common.MD5New, []byte(values.Encode()), []byte(common.Sha1ToHex(h.APISecret)))))
	values.Set("reqTime", fmt.Sprintf("%d", time.Now().UnixNano()/1e6))

	url := fmt.Sprintf("%s/%s?%s", zbMarketURL, endpoint, values.Encode())

	return h.SendPayload(method, url, headers, strings.NewReader(""), result, true, h.Verbose)
}
