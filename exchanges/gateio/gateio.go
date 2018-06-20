package gateio

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/config"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/request"
	"github.com/idoall/gocryptotrader/exchanges/ticker"
	"github.com/pkg/errors"
)

const (
	gateioTradeURL   = "https://api.gateio.io"
	gateioMarketURL  = "https://data.gateio.io"
	gateioAPIVersion = "api2/1"

	gateioSymbol              = "pairs"
	gateioMarketInfo          = "marketinfo"
	gateioKline               = "candlestick2"
	huobiMarketDepth          = "market/depth"
	huobiMarketTrade          = "market/trade"
	huobiMarketTradeHistory   = "market/history/trade"
	huobiSymbols              = "common/symbols"
	huobiCurrencies           = "common/currencys"
	huobiTimestamp            = "common/timestamp"
	huobiAccounts             = "account/accounts"
	huobiAccountBalance       = "account/accounts/%s/balance"
	huobiOrderPlace           = "order/orders/place"
	huobiOrderCancel          = "order/orders/%s/submitcancel"
	huobiOrderCancelBatch     = "order/orders/batchcancel"
	huobiGetOrder             = "order/orders/%s"
	huobiGetOrderMatch        = "order/orders/%s/matchresults"
	huobiGetOrders            = "order/orders"
	huobiGetOrdersMatch       = "orders/matchresults"
	huobiMarginTransferIn     = "dw/transfer-in/margin"
	huobiMarginTransferOut    = "dw/transfer-out/margin"
	huobiMarginOrders         = "margin/orders"
	huobiMarginRepay          = "margin/orders/%s/repay"
	huobiMarginLoanOrders     = "margin/loan-orders"
	huobiMarginAccountBalance = "margin/accounts/balance"
	huobiWithdrawCreate       = "dw/withdraw/api/create"
	huobiWithdrawCancel       = "dw/withdraw-virtual/%s/cancel"

	huobiAuthRate   = 100
	huobiUnauthRate = 100
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
	h.RequestCurrencyPairFormat.Uppercase = false
	h.ConfigCurrencyPairFormat.Delimiter = "-"
	h.ConfigCurrencyPairFormat.Uppercase = true
	h.AssetTypes = []string{ticker.Spot}
	h.SupportsAutoPairUpdating = true
	h.SupportsRESTTickerBatching = false
	h.Requester = request.New(h.Name, request.NewRateLimit(time.Second*10, huobiAuthRate), request.NewRateLimit(time.Second*10, huobiUnauthRate), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
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
		h.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		h.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		h.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")

		h.RequestCurrencyPairFormat = config.CurrencyPairFormatConfig{
			Delimiter: exch.RequestCurrencyPairFormat.Delimiter,
			Uppercase: exch.RequestCurrencyPairFormat.Uppercase,
			Separator: exch.RequestCurrencyPairFormat.Separator,
			Index:     exch.RequestCurrencyPairFormat.Index,
		}

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

// GetKline 返回市场最近时间段内的K先数据
func (h *Gateio) GetKline(arg GateioKlinesRequestParams) ([]*GateioKLineReturn, error) {

	url := fmt.Sprintf("%s/%s/%s/%s?group_sec=%d&range_hour=%d", gateioMarketURL, gateioAPIVersion, gateioKline, arg.Symbol, arg.GroupSec, arg.HourSize)

	var rawKlines map[string]interface{}
	err := h.SendHTTPRequest(url, &rawKlines)
	if err != nil {
		return nil, err
	}

	var result []*GateioKLineReturn

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
		result = append(result, &GateioKLineReturn{
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

// SendHTTPRequest sends an unauthenticated HTTP request
func (h *Gateio) SendHTTPRequest(path string, result interface{}) error {
	return h.SendPayload("GET", path, nil, nil, result, false, h.Verbose)
}
