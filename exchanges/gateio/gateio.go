package gateio

import (
	"encoding/json"
	"fmt"
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
	gateioTradeURL   = "https://api.gateio.io"
	gateioMarketURL  = "https://data.gateio.io"
	gateioAPIVersion = "api2/1"

	gateioSymbol      = "pairs"
	gateioMarketInfo  = "marketinfo"
	gateioKline       = "candlestick2"
	gateioOrder       = "private"
	gateioBalances    = "private/balances"
	gateioCancelOrder = "private/cancelOrder"
	gateioTicker      = "ticker"

	gateioAuthRate   = 100
	gateioUnauthRate = 100
)

// Gateio is the overarching type across this package
type Gateio struct {
	exchange.Base
}

// SetDefaults sets default values for the exchange
func (h *Gateio) SetDefaults() {
	h.Name = "Gateio"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.Websocket = false
	h.RESTPollingDelay = 10
	h.RequestCurrencyPairFormat.Delimiter = "_"
	h.RequestCurrencyPairFormat.Uppercase = true
	authRateLimit := request.NewRateLimit(time.Second*10, gateioUnauthRate)
	authRateLimit.SetRequests(3)
	h.Requester = request.New(h.Name, request.NewRateLimit(time.Second*10, gateioAuthRate), authRateLimit, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	// h.Requester = request.New(h.Name, request.NewRateLimit(time.Second*10, gateioAuthRate), request.NewRateLimit(time.Second*10, gateioUnauthRate), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup sets user configuration
func (h *Gateio) Setup(exch config.ExchangeConfig) {
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

// GetSymbols 返回所有系统支持的交易对
func (h *Gateio) GetSymbols() ([]string, error) {
	var result []string

	url := fmt.Sprintf("%s/%s/%s", gateioMarketURL, gateioAPIVersion, gateioSymbol)

	err := h.SendHTTPRequest(url, &result)
	if err != nil {
		return nil, nil
	}
	return result, err
}

// GetMarketInfo 返回所有系统支持的交易市场的参数信息，包括交易费，最小下单量，价格精度等。
func (h *Gateio) GetMarketInfo() (MarketInfoResponse, error) {
	type response struct {
		Result string        `json:"result"`
		Pairs  []interface{} `json:"pairs"`
	}

	url := fmt.Sprintf("%s/%s/%s", gateioMarketURL, gateioAPIVersion, gateioMarketInfo)

	var res response
	var result MarketInfoResponse
	err := h.SendHTTPRequest(url, &res)
	if err != nil {
		return result, err
	}

	result.Result = res.Result
	for _, v := range res.Pairs {
		item := v.(map[string]interface{})
		for itemk, itemv := range item {
			pairv := itemv.(map[string]interface{})
			result.Pairs = append(result.Pairs, MarketInfoPairsResponse{
				Symbol:        itemk,
				DecimalPlaces: pairv["decimal_places"].(float64),
				MinAmount:     pairv["min_amount"].(float64),
				Fee:           pairv["fee"].(float64),
			})
		}
	}
	return result, nil
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
// 获取最新价格，官方每10秒钟更新一次
func (h *Gateio) GetLatestSpotPrice(symbol string) (float64, error) {
	res, err := h.GetTicker(symbol)
	if err != nil {
		return 0, err
	}
	price, err := strconv.ParseFloat(res.Last, 64)
	if err != nil {
		return 0, err
	}
	return price, nil
}

// GetTicker returns 单项交易行情，官方每10秒钟更新一次
func (h *Gateio) GetTicker(symbol string) (TickerResponse, error) {
	url := fmt.Sprintf("%s/%s/%s/%s", gateioMarketURL, gateioAPIVersion, gateioTicker, symbol)

	var res TickerResponse
	err := h.SendHTTPRequest(url, &res)
	if err != nil {
		return res, err
	}
	return res, nil
}

// GetSpotKline 返回市场最近时间段内的K先数据
func (h *Gateio) GetSpotKline(arg KlinesRequestParams) ([]*KLineResponse, error) {

	url := fmt.Sprintf("%s/%s/%s/%s?group_sec=%d&range_hour=%d", gateioMarketURL, gateioAPIVersion, gateioKline, arg.Symbol, arg.GroupSec, arg.HourSize)

	var rawKlines map[string]interface{}
	err := h.SendHTTPRequest(url, &rawKlines)
	if err != nil {
		return nil, err
	}

	var result []*KLineResponse

	if rawKlines == nil || rawKlines["data"] == nil {
		return nil, errors.Wrap(err, "rawKlines is nil")
	}

	//对于 Data数据，再次解析
	rawKlineDatasString, _ := json.Marshal(rawKlines["data"].([]interface{}))
	rawKlineDatas := [][]interface{}{}
	if err := json.Unmarshal(rawKlineDatasString, &rawKlineDatas); err != nil {
		return nil, errors.Wrap(err, "rawKlineDatas unmarshal failed")
	}
	for _, k := range rawKlineDatas {
		otString, _ := strconv.ParseFloat(k[0].(string), 64)
		ot, err := common.TimeFromUnixTimestampFloat(otString)
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse Kline.OpenTime")
		}
		_vol, err := common.FloatFromString(k[1])
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse Kline.Volume")
		}
		_id, err := common.FloatFromString(k[0])
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse Kline.Id")
		}
		_close, err := common.FloatFromString(k[2])
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse Kline.Close")
		}
		_high, err := common.FloatFromString(k[3])
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse Kline.High")
		}
		_low, err := common.FloatFromString(k[4])
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse Kline.Low")
		}
		_open, err := common.FloatFromString(k[5])
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse Kline.Open")
		}
		result = append(result, &KLineResponse{
			ID:        _id,
			KlineTime: ot,
			Volume:    _vol,   //成交量
			Close:     _close, //收盘价
			High:      _high,  //最高
			Low:       _low,   //最低
			Open:      _open,  //开盘价
		})
	}

	return result, nil
}

// GetBalances 获取帐号资金余额
// 通过以下API，用户可以使用程序控制自动进行账号资金查询，下单交易，取消挂单。
// 请注意：请在您的程序中设置的HTTP请求头参数 Content-Type 为 application/x-www-form-urlencoded
// 用户首先要通过这个链接获取API接口身份认证用到的Key和Secret。 然后在程序中用Secret作为密码，通过SHA512加密方式签名需要POST给服务器的数据得到Sign，并在HTTPS请求的Header部分传回Key和Sign。请参考以下接口说明和例子程序进行设置。
func (h *Gateio) GetBalances() (BalancesResponse, error) {

	var result BalancesResponse

	err := h.SendAuthenticatedHTTPRequest("POST", gateioBalances, "", &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// SpotNewOrder 下订单
func (h *Gateio) SpotNewOrder(arg SpotNewOrderRequestParams) (SpotNewOrderResponse, error) {
	var result SpotNewOrderResponse

	//获取交易对的价格精度格式
	params := fmt.Sprintf("currencyPair=%s&rate=%s&amount=%s",
		h.GetSymbol(),
		strconv.FormatFloat(arg.Price, 'f', -1, 64),
		strconv.FormatFloat(arg.Amount, 'f', -1, 64),
	)

	strRequestURL := fmt.Sprintf("%s/%s", gateioOrder, arg.Type)

	err := h.SendAuthenticatedHTTPRequest("POST", strRequestURL, params, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

// CancelOrder 取消订单
// @orderID 下单单号
// @symbol 交易币种对(如 ltc_btc)
func (h *Gateio) CancelOrder(orderID int64, symbol string) (bool, error) {
	type response struct {
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	var result response
	//获取交易对的价格精度格式
	params := fmt.Sprintf("orderNumber=%d&currencyPair=%s",
		orderID,
		symbol,
	)
	err := h.SendAuthenticatedHTTPRequest("POST", gateioCancelOrder, params, &result)
	if err != nil {
		return false, err
	}
	if !result.Result {
		return false, fmt.Errorf("code:%d message:%s", result.Code, result.Message)
	}

	return true, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (h *Gateio) SendHTTPRequest(path string, result interface{}) error {
	return h.SendPayload("GET", path, nil, nil, result, false, h.Verbose)
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
func (h *Gateio) SendAuthenticatedHTTPRequest(method, endpoint, param string, result interface{}) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["key"] = h.APIKey

	hmac := common.GetHMAC(common.HashSHA512, []byte(param), []byte(h.APISecret))
	headers["sign"] = common.ByteArrayToString(hmac)

	url := fmt.Sprintf("%s/%s/%s", gateioTradeURL, gateioAPIVersion, endpoint)

	return h.SendPayload(method, url, headers, strings.NewReader(param), result, true, h.Verbose)
}
