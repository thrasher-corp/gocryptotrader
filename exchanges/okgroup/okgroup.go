package okgroup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	// just your average return type from okex
	returnTypeOne = "map[string]interface {}"

	okexAuthRate   = 0
	okexUnauthRate = 0

	okGroupAPIPath       = "api/"
	okGroupGetCurrencies = "currencies"
)

var errMissValue = errors.New("warning - resp value is missing from exchange")

// Okex is okgroup implementation for OKEX
var Okex = OKGroup{
	APIURL:       fmt.Sprintf("%v%v", "https://www.okex.com/", okGroupAPIPath),
	APIVersion:   "/v3/",
	ExchangeName: "OKEX",
}

// Okcoin is okgroup implementation for OKEX
var Okcoin = OKGroup{
	APIURL:       fmt.Sprintf("%v%v", "https://www.okcoin.com/", okGroupAPIPath),
	APIVersion:   "/v3/",
	ExchangeName: "OKCoin",
}

// OKGroup is the overaching type across the all of OKEx's exchange methods
type OKGroup struct {
	exchange.Base
	ExchangeName  string
	WebsocketConn *websocket.Conn
	mu            sync.Mutex

	// Spot and contract market error codes as per https://www.okex.com/rest_request.html
	ErrorCodes map[string]error

	// Stores for corresponding variable checks
	ContractTypes    []string
	CurrencyPairs    []string
	ContractPosition []string
	Types            []string

	// URLs to be overridden by implementations of OKGroup
	APIURL     string
	APIVersion string
}

// SetDefaults method assignes the default values for Bittrex
func (o *OKGroup) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = o.ExchangeName
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = "_"
	o.ConfigCurrencyPairFormat.Uppercase = true
	o.SupportsAutoPairUpdating = true
	o.SupportsRESTTickerBatching = false
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okexAuthRate),
		request.NewRateLimit(time.Second, okexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	o.APIUrlDefault = o.APIURL
	o.APIUrl = o.APIUrlDefault
	o.AssetTypes = []string{ticker.Spot}
	o.WebsocketInit()
	o.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketKlineSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup method sets current configuration details if enabled
func (o *OKGroup) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		o.SetEnabled(false)
	} else {
		o.Name = exch.Name
		o.Enabled = true
		o.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		o.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID, false)
		o.SetHTTPClientTimeout(exch.HTTPTimeout)
		o.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		o.RESTPollingDelay = exch.RESTPollingDelay
		o.Verbose = exch.Verbose
		o.Websocket.SetEnabled(exch.Websocket)
		o.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		o.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		o.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := o.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = o.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = o.WebsocketSetup(o.WsConnect,
			exch.Name,
			exch.Websocket,
			okexDefaultWebsocketURL,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// -------------------------------------------------------------------------------------
// Public endpoints
// -------------------------------------------------------------------------------------

// GetSpotInstruments returns a list of tradable spot instruments and their properties
func (o *OKGroup) GetSpotInstruments() ([]SpotInstrument, error) {
	var resp []SpotInstrument

	path := fmt.Sprintf("%sspot/v3/%s", o.APIUrl, "instruments")
	err := o.SendHTTPRequest(path, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetCurrencies returns a list of tradable spot instruments and their properties
func (o *OKGroup) GetCurrencies() ([]SpotInstrument, error) {
	var resp []SpotInstrument

	err := o.SendAuthenticatedHTTPRequest("GET", "account", okGroupGetCurrencies, url.Values{}, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetContractPrice returns current contract prices
//
// symbol e.g. "btc_usd"
// contractType e.g. "this_week" "next_week" "quarter"
func (o *OKGroup) GetContractPrice(symbol, contractType string) (ContractPrice, error) {
	resp := ContractPrice{}

	if err := o.CheckContractType(contractType); err != nil {
		return resp, err
	}
	if err := o.CheckSymbol(symbol); err != nil {
		return resp, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	values.Set("contract_type", common.StringToLower(contractType))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractPrice", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if !resp.Result {
		if resp.Error != nil {
			return resp, o.GetErrorCode(resp.Error)
		}
	}
	return resp, nil
}

// GetContractMarketDepth returns contract market depth
//
// symbol e.g. "btc_usd"
// contractType e.g. "this_week" "next_week" "quarter"
func (o *OKGroup) GetContractMarketDepth(symbol, contractType string) (ActualContractDepth, error) {
	resp := ContractDepth{}
	fullDepth := ActualContractDepth{}

	if err := o.CheckContractType(contractType); err != nil {
		return fullDepth, err
	}
	if err := o.CheckSymbol(symbol); err != nil {
		return fullDepth, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	values.Set("contract_type", common.StringToLower(contractType))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractFutureDepth", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return fullDepth, err
	}

	if !resp.Result {
		if resp.Error != nil {
			return fullDepth, o.GetErrorCode(resp.Error)
		}
	}

	for _, ask := range resp.Asks {
		var askdepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range ask.([]interface{}) {
			if i == 0 {
				askdepth.Price = depth.(float64)
			}
			if i == 1 {
				askdepth.Volume = depth.(float64)
			}
		}
		fullDepth.Asks = append(fullDepth.Asks, askdepth)
	}

	for _, bid := range resp.Bids {
		var bidDepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range bid.([]interface{}) {
			if i == 0 {
				bidDepth.Price = depth.(float64)
			}
			if i == 1 {
				bidDepth.Volume = depth.(float64)
			}
		}
		fullDepth.Bids = append(fullDepth.Bids, bidDepth)
	}

	return fullDepth, nil
}

// GetContractTradeHistory returns trade history for the contract market
func (o *OKGroup) GetContractTradeHistory(symbol, contractType string) ([]ActualContractTradeHistory, error) {
	actualTradeHistory := []ActualContractTradeHistory{}
	var resp interface{}

	if err := o.CheckContractType(contractType); err != nil {
		return actualTradeHistory, err
	}
	if err := o.CheckSymbol(symbol); err != nil {
		return actualTradeHistory, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	values.Set("contract_type", common.StringToLower(contractType))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractTradeHistory", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return actualTradeHistory, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return actualTradeHistory, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, tradeHistory := range resp.([]interface{}) {
		quickHistory := ActualContractTradeHistory{}
		tradeHistoryM := tradeHistory.(map[string]interface{})
		quickHistory.Date = tradeHistoryM["date"].(float64)
		quickHistory.DateInMS = tradeHistoryM["date_ms"].(float64)
		quickHistory.Amount = tradeHistoryM["amount"].(float64)
		quickHistory.Price = tradeHistoryM["price"].(float64)
		quickHistory.Type = tradeHistoryM["type"].(string)
		quickHistory.TID = tradeHistoryM["tid"].(float64)
		actualTradeHistory = append(actualTradeHistory, quickHistory)
	}
	return actualTradeHistory, nil
}

// GetContractIndexPrice returns the current index price
//
// symbol e.g. btc_usd
func (o *OKGroup) GetContractIndexPrice(symbol string) (float64, error) {
	if err := o.CheckSymbol(symbol); err != nil {
		return 0, err
	}

	values := url.Values{}
	values.Set("symbol", common.StringToLower(symbol))
	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractFutureIndex", values.Encode())
	var resp interface{}

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return 0, err
	}

	futureIndex := resp.(map[string]interface{})
	if i, ok := futureIndex["error_code"].(float64); ok {
		return 0, o.GetErrorCode(i)
	}

	if _, ok := futureIndex["future_index"].(float64); ok {
		return futureIndex["future_index"].(float64), nil
	}
	return 0, errMissValue
}

// GetContractExchangeRate returns the current exchange rate for the currency
// pair
// USD-CNY exchange rate used by OKGroup, updated weekly
func (o *OKGroup) GetContractExchangeRate() (float64, error) {
	path := fmt.Sprintf("%s%s%s.do?", o.APIUrl, o.APIVersion, "contractExchangeRate")
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return 0, err
	}

	exchangeRate := resp.(map[string]interface{})
	if i, ok := exchangeRate["error_code"].(float64); ok {
		return 0, o.GetErrorCode(i)
	}

	if _, ok := exchangeRate["rate"].(float64); ok {
		return exchangeRate["rate"].(float64), nil
	}
	return 0, errMissValue
}

// GetContractFutureEstimatedPrice returns futures estimated price
//
// symbol e.g btc_usd
func (o *OKGroup) GetContractFutureEstimatedPrice(symbol string) (float64, error) {
	if err := o.CheckSymbol(symbol); err != nil {
		return 0, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractFutureIndex", values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return 0, err
	}

	futuresEstPrice := resp.(map[string]interface{})
	if i, ok := futuresEstPrice["error_code"].(float64); ok {
		return 0, o.GetErrorCode(i)
	}

	if _, ok := futuresEstPrice["future_index"].(float64); ok {
		return futuresEstPrice["future_index"].(float64), nil
	}
	return 0, errMissValue
}

// GetContractCandlestickData returns CandleStickData
//
// symbol e.g. btc_usd
// type e.g. 1min or 1 minute candlestick data
// contract_type e.g. this_week
// size: specify data size to be acquired
// since: timestamp(eg:1417536000000). data after the timestamp will be returned
func (o *OKGroup) GetContractCandlestickData(symbol, typeInput, contractType string, size, since int) ([]CandleStickData, error) {
	var candleData []CandleStickData
	if err := o.CheckSymbol(symbol); err != nil {
		return candleData, err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return candleData, err
	}
	if err := o.CheckType(typeInput); err != nil {
		return candleData, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("type", typeInput)
	values.Set("contract_type", contractType)
	values.Set("size", strconv.FormatInt(int64(size), 10))
	values.Set("since", strconv.FormatInt(int64(since), 10))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractCandleStick", values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return candleData, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return candleData, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, candleStickData := range resp.([]interface{}) {
		var quickCandle CandleStickData

		for i, datum := range candleStickData.([]interface{}) {
			switch i {
			case 0:
				quickCandle.Timestamp = datum.(float64)
			case 1:
				quickCandle.Open = datum.(float64)
			case 2:
				quickCandle.High = datum.(float64)
			case 3:
				quickCandle.Low = datum.(float64)
			case 4:
				quickCandle.Close = datum.(float64)
			case 5:
				quickCandle.Volume = datum.(float64)
			case 6:
				quickCandle.Amount = datum.(float64)
			default:
				return candleData, errors.New("incoming data out of range")
			}
		}
		candleData = append(candleData, quickCandle)
	}

	return candleData, nil
}

// GetContractHoldingsNumber returns current number of holdings
func (o *OKGroup) GetContractHoldingsNumber(symbol, contractType string) (number float64, contract string, err error) {
	if err = o.CheckSymbol(symbol); err != nil {
		return number, contract, err
	}
	if err = o.CheckContractType(contractType); err != nil {
		return number, contract, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractFutureHoldAmount", values.Encode())
	var resp interface{}

	if err = o.SendHTTPRequest(path, &resp); err != nil {
		return number, contract, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return number, contract, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, holdings := range resp.([]interface{}) {
		if reflect.TypeOf(holdings).String() == returnTypeOne {
			holdingMap := holdings.(map[string]interface{})
			number = holdingMap["amount"].(float64)
			contract = holdingMap["contract_name"].(string)
		}
	}
	return
}

// GetContractlimit returns upper and lower price limit
func (o *OKGroup) GetContractlimit(symbol, contractType string) (map[string]float64, error) {
	contractLimits := make(map[string]float64)
	if err := o.CheckSymbol(symbol); err != nil {
		return contractLimits, err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return contractLimits, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "contractFutureLimits", values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return contractLimits, err
	}

	contractLimitMap := resp.(map[string]interface{})
	if i, ok := contractLimitMap["error_code"].(float64); ok {
		return contractLimits, o.GetErrorCode(i)
	}

	contractLimits["high"] = contractLimitMap["high"].(float64)
	contractLimits["usdCnyRate"] = contractLimitMap["usdCnyRate"].(float64)
	contractLimits["low"] = contractLimitMap["low"].(float64)
	return contractLimits, nil
}

// GetSpotTicker returns Price Ticker
func (o *OKGroup) GetSpotTicker(symbol string) (SpotPrice, error) {
	var resp SpotPrice

	values := url.Values{}
	values.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "ticker", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return resp, err
	}

	if resp.Error != nil {
		return resp, o.GetErrorCode(resp.Error.(float64))
	}
	return resp, nil
}

//GetSpotMarketDepth returns Market Depth
func (o *OKGroup) GetSpotMarketDepth(asd ActualSpotDepthRequestParams) (ActualSpotDepth, error) {
	resp := SpotDepth{}
	fullDepth := ActualSpotDepth{}

	values := url.Values{}
	values.Set("symbol", asd.Symbol)
	values.Set("size", fmt.Sprintf("%d", asd.Size))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "depth", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return fullDepth, err
	}

	if !resp.Result {
		if resp.Error != nil {
			return fullDepth, o.GetErrorCode(resp.Error)
		}
	}

	for _, ask := range resp.Asks {
		var askdepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range ask.([]interface{}) {
			if i == 0 {
				askdepth.Price = depth.(float64)
			}
			if i == 1 {
				askdepth.Volume = depth.(float64)
			}
		}
		fullDepth.Asks = append(fullDepth.Asks, askdepth)
	}

	for _, bid := range resp.Bids {
		var bidDepth struct {
			Price  float64
			Volume float64
		}
		for i, depth := range bid.([]interface{}) {
			if i == 0 {
				bidDepth.Price = depth.(float64)
			}
			if i == 1 {
				bidDepth.Volume = depth.(float64)
			}
		}
		fullDepth.Bids = append(fullDepth.Bids, bidDepth)
	}

	return fullDepth, nil
}

// GetSpotRecentTrades returns recent trades
func (o *OKGroup) GetSpotRecentTrades(ast ActualSpotTradeHistoryRequestParams) ([]ActualSpotTradeHistory, error) {
	actualTradeHistory := []ActualSpotTradeHistory{}
	var resp interface{}

	values := url.Values{}
	values.Set("symbol", ast.Symbol)
	values.Set("since", fmt.Sprintf("%d", ast.Since))

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "trades", values.Encode())

	err := o.SendHTTPRequest(path, &resp)
	if err != nil {
		return actualTradeHistory, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return actualTradeHistory, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, tradeHistory := range resp.([]interface{}) {
		quickHistory := ActualSpotTradeHistory{}
		tradeHistoryM := tradeHistory.(map[string]interface{})
		quickHistory.Date = tradeHistoryM["date"].(float64)
		quickHistory.DateInMS = tradeHistoryM["date_ms"].(float64)
		quickHistory.Amount = tradeHistoryM["amount"].(float64)
		quickHistory.Price = tradeHistoryM["price"].(float64)
		quickHistory.Type = tradeHistoryM["type"].(string)
		quickHistory.TID = tradeHistoryM["tid"].(float64)
		actualTradeHistory = append(actualTradeHistory, quickHistory)
	}
	return actualTradeHistory, nil
}

// GetSpotKline returns candlestick data
func (o *OKGroup) GetSpotKline(arg KlinesRequestParams) ([]CandleStickData, error) {
	var candleData []CandleStickData

	values := url.Values{}
	values.Set("symbol", arg.Symbol)
	values.Set("type", string(arg.Type))
	if arg.Size != 0 {
		values.Set("size", strconv.FormatInt(int64(arg.Size), 10))
	}
	if arg.Since != 0 {
		values.Set("since", strconv.FormatInt(int64(arg.Since), 10))
	}

	path := fmt.Sprintf("%s%s%s.do?%s", o.APIUrl, o.APIVersion, "spotKline", values.Encode())
	var resp interface{}

	if err := o.SendHTTPRequest(path, &resp); err != nil {
		return candleData, err
	}

	if reflect.TypeOf(resp).String() == returnTypeOne {
		errorMap := resp.(map[string]interface{})
		return candleData, o.GetErrorCode(errorMap["error_code"].(float64))
	}

	for _, candleStickData := range resp.([]interface{}) {
		var quickCandle CandleStickData

		for i, datum := range candleStickData.([]interface{}) {
			switch i {
			case 0:
				quickCandle.Timestamp = datum.(float64)
			case 1:
				quickCandle.Open, _ = strconv.ParseFloat(datum.(string), 64)
			case 2:
				quickCandle.High, _ = strconv.ParseFloat(datum.(string), 64)
			case 3:
				quickCandle.Low, _ = strconv.ParseFloat(datum.(string), 64)
			case 4:
				quickCandle.Close, _ = strconv.ParseFloat(datum.(string), 64)
			case 5:
				quickCandle.Volume, _ = strconv.ParseFloat(datum.(string), 64)
			case 6:
				quickCandle.Amount, _ = strconv.ParseFloat(datum.(string), 64)
			default:
				return candleData, errors.New("incoming data out of range")
			}
		}
		candleData = append(candleData, quickCandle)
	}

	return candleData, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (o *OKGroup) SendHTTPRequest(path string, result interface{}) error {
	return o.SendPayload("GET", path, nil, nil, result, false, o.Verbose)
}

// -------------------------------------------------------------------------------------
// Private endpoints
// -------------------------------------------------------------------------------------

// GetContractUserInfo returns OKGroup Contract Account Info（Cross-Margin Mode）
func (o *OKGroup) GetContractUserInfo() error {
	var resp interface{}
	if err := o.SendAuthenticatedHTTPRequest("POST", "", "contractFutureUserInfo", url.Values{}, &resp); err != nil {
		return err
	}

	userInfoMap := resp.(map[string]interface{})
	if code, ok := userInfoMap["error_code"]; ok {
		return o.GetErrorCode(code)
	}
	return nil
}

// GetContractPosition returns User Contract Positions （Cross-Margin Mode）
func (o *OKGroup) GetContractPosition(symbol, contractType string) error {
	var resp interface{}

	if err := o.CheckSymbol(symbol); err != nil {
		return err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)

	if err := o.SendAuthenticatedHTTPRequest("POST", "", "contractFuturePosition", values, &resp); err != nil {
		return err
	}

	userInfoMap := resp.(map[string]interface{})
	if code, ok := userInfoMap["error_code"]; ok {
		return o.GetErrorCode(code)
	}
	return nil
}

// PlaceContractOrders places orders
func (o *OKGroup) PlaceContractOrders(symbol, contractType, position string, leverageRate int, price, amount float64, matchPrice bool) (float64, error) {
	var resp interface{}

	if err := o.CheckSymbol(symbol); err != nil {
		return 0, err
	}
	if err := o.CheckContractType(contractType); err != nil {
		return 0, err
	}
	if err := o.CheckContractPosition(position); err != nil {
		return 0, err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("contract_type", contractType)
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("type", position)
	if matchPrice {
		values.Set("match_price", "1")
	} else {
		values.Set("match_price", "0")
	}

	if leverageRate != 10 && leverageRate != 20 {
		return 0, errors.New("leverage rate can only be 10 or 20")
	}
	values.Set("lever_rate", strconv.FormatInt(int64(leverageRate), 10))

	if err := o.SendAuthenticatedHTTPRequest("POST", "", "contractFutureTrade", values, &resp); err != nil {
		return 0, err
	}

	contractMap := resp.(map[string]interface{})
	if code, ok := contractMap["error_code"]; ok {
		return 0, o.GetErrorCode(code)
	}

	if orderID, ok := contractMap["order_id"]; ok {
		return orderID.(float64), nil
	}

	return 0, errors.New("orderID returned nil")
}

// GetContractFuturesTradeHistory returns OKGroup Contract Trade History (Not for Personal)
func (o *OKGroup) GetContractFuturesTradeHistory(symbol, date string, since int) error {
	var resp interface{}

	if err := o.CheckSymbol(symbol); err != nil {
		return err
	}

	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("date", date)
	values.Set("since", strconv.FormatInt(int64(since), 10))

	if err := o.SendAuthenticatedHTTPRequest("POST", "", "contractFutureTradeHistory", values, &resp); err != nil {
		return err
	}

	respMap := resp.(map[string]interface{})
	if code, ok := respMap["error_code"]; ok {
		return o.GetErrorCode(code)
	}
	return nil
}

// GetTokenOrders returns details for a single orderID or all open orders when orderID == -1
func (o *OKGroup) GetTokenOrders(symbol string, orderID int64) (TokenOrdersResponse, error) {
	var resp TokenOrdersResponse
	values := url.Values{}
	values.Set("symbol", symbol)
	values.Set("order_id", strconv.FormatInt(orderID, 10))

	if err := o.SendAuthenticatedHTTPRequest("POST", "", "contractFutureTradeHistory", values, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// GetUserInfo returns the user info
func (o *OKGroup) GetUserInfo() (SpotUserInfo, error) {
	var resp SpotUserInfo
	err := o.SendAuthenticatedHTTPRequest("POST", "", "spotUserInfo", url.Values{}, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// SpotNewOrder creates a new spot order
func (o *OKGroup) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	type response struct {
		Result  bool  `json:"result"`
		OrderID int64 `json:"order_id"`
	}

	var res response
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("type", string(arg.Type))
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))

	err := o.SendAuthenticatedHTTPRequest("POST", "", "spotTrade", params, &res)
	if err != nil {
		return res.OrderID, err
	}

	return res.OrderID, nil
}

// SpotCancelOrder cancels a spot order
// symbol such as ltc_btc
// orderID orderID
// returns orderID or an error
func (o *OKGroup) SpotCancelOrder(symbol string, argOrderID int64) (int64, error) {
	var res = struct {
		Result    bool   `json:"result"`
		OrderID   string `json:"order_id"`
		ErrorCode int    `json:"error_code"`
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("order_id", strconv.FormatInt(argOrderID, 10))
	var returnOrderID int64

	err := o.SendAuthenticatedHTTPRequest("POST", "", "spotCancelTrade"+".do", params, &res)
	if err != nil {
		return returnOrderID, err
	}

	if res.ErrorCode != 0 {
		return returnOrderID, fmt.Errorf("ErrCode:%d ErrMsg:%s", res.ErrorCode, o.ErrorCodes[strconv.Itoa(res.ErrorCode)])
	}

	returnOrderID, _ = common.Int64FromString(res.OrderID)
	return returnOrderID, nil
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (o *OKGroup) GetLatestSpotPrice(symbol string) (float64, error) {
	spotPrice, err := o.GetSpotTicker(symbol)

	if err != nil {
		return 0, err
	}

	return spotPrice.Ticker.Last, nil
}

// GetErrorCode finds the associated error code and returns its corresponding
// string
func (o *OKGroup) GetErrorCode(code interface{}) error {
	var assertedCode string

	switch reflect.TypeOf(code).String() {
	case "float64":
		assertedCode = strconv.FormatFloat(code.(float64), 'f', -1, 64)
	case "string":
		assertedCode = code.(string)
	default:
		return errors.New("unusual type returned")
	}

	if i, ok := o.ErrorCodes[assertedCode]; ok {
		return i
	}
	return errors.New("unable to find SPOT error code")
}

// SendAuthenticatedHTTPRequest sends an authenticated http request to a desired
// path
func (o *OKGroup) SendAuthenticatedHTTPRequest(httpMethod, requestType, requestPath string, values url.Values, result interface{}) (err error) {
	if !o.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, o.Name)
	}

	utcTime := time.Now().UTC()
	iso := utcTime.String()
	isoBytes := []byte(iso)
	iso = string(isoBytes[:10]) + "T" + string(isoBytes[11:23]) + "Z"

	encoded := values.Encode()
	path := o.APIUrl + requestType + o.APIVersion + requestPath
	signPath := fmt.Sprintf("/%v%v%v%v", okGroupAPIPath, requestType, o.APIVersion, requestPath)
	hmac := common.GetHMAC(common.HashSHA256, []byte(iso+httpMethod+signPath), []byte(o.APISecret))
	base64 := common.Base64Encode(hmac)

	if o.Verbose {
		log.Debugf("Sending %v request to %s with params %s\n", requestType, path, encoded)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["OK-ACCESS-KEY"] = o.APIKey
	headers["OK-ACCESS-SIGN"] = base64
	headers["OK-ACCESS-TIMESTAMP"] = iso
	headers["OK-ACCESS-PASSPHRASE"] = o.ClientID

	var intermediary json.RawMessage

	errCap := struct {
		Result bool  `json:"result"`
		Error  int64 `json:"error_code"`
	}{}

	err = o.SendPayload(strings.ToUpper(httpMethod), path, headers, strings.NewReader(encoded), &intermediary, true, o.Verbose)
	if err != nil {
		return err
	}

	err = common.JSONDecode(intermediary, &errCap)
	if err == nil {
		if !errCap.Result {
			return fmt.Errorf("SendAuthenticatedHTTPRequest error - %s",
				o.ErrorCodes[strconv.FormatInt(errCap.Error, 10)])
		}
	}

	return common.JSONDecode(intermediary, result)
}

// SetCheckVarDefaults sets main variables that will be used in requests because
// api does not return an error if there are misspellings in strings. So better
// to check on this, this end.
func (o *OKGroup) SetCheckVarDefaults() {
	o.ContractTypes = []string{"this_week", "next_week", "quarter"}
	o.CurrencyPairs = []string{"btc_usd", "ltc_usd", "eth_usd", "etc_usd", "bch_usd"}
	o.Types = []string{"1min", "3min", "5min", "15min", "30min", "1day", "3day",
		"1week", "1hour", "2hour", "4hour", "6hour", "12hour"}
	o.ContractPosition = []string{"1", "2", "3", "4"}
}

// CheckContractPosition checks to see if the string is a valid position for OKGroup
func (o *OKGroup) CheckContractPosition(position string) error {
	if !common.StringDataCompare(o.ContractPosition, position) {
		return errors.New("invalid position string - e.g. 1 = open long position, 2 = open short position, 3 = liquidate long position, 4 = liquidate short position")
	}
	return nil
}

// CheckSymbol checks to see if the string is a valid symbol for OKGroup
func (o *OKGroup) CheckSymbol(symbol string) error {
	if !common.StringDataCompare(o.CurrencyPairs, symbol) {
		return errors.New("invalid symbol string")
	}
	return nil
}

// CheckContractType checks to see if the string is a correct asset
func (o *OKGroup) CheckContractType(contractType string) error {
	if !common.StringDataCompare(o.ContractTypes, contractType) {
		return errors.New("invalid contract type string")
	}
	return nil
}

// CheckType checks to see if the string is a correct type
func (o *OKGroup) CheckType(typeInput string) error {
	if !common.StringDataCompare(o.Types, typeInput) {
		return errors.New("invalid type string")
	}
	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FirstCurrency)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
	// TODO volume based fees
	if isMaker {
		fee = 0.001
	} else {
		fee = 0.0015
	}
	return fee * amount * purchasePrice
}

func getWithdrawalFee(currency string) float64 {
	return WithdrawalFees[currency]
}

// GetBalance returns the full balance across all wallets
func (o *OKGroup) GetBalance() ([]FullBalance, error) {
	var resp Balance
	var balances []FullBalance

	err := o.SendAuthenticatedHTTPRequest("POST", "", "myWalletInfo", url.Values{}, &resp)
	if err != nil {
		return balances, err
	}

	for key, available := range resp.Info.Funds.Free {
		free, err := strconv.ParseFloat(available, 64)
		if err != nil {
			return balances, err
		}

		inUse, ok := resp.Info.Funds.Holds[key]
		if !ok {
			return balances, fmt.Errorf("hold currency %s not found in map", key)
		}

		hold, err := strconv.ParseFloat(inUse, 64)
		if err != nil {
			return balances, err
		}

		balances = append(balances, FullBalance{
			Currency:  key,
			Available: free,
			Hold:      hold,
		})
	}

	return balances, nil
}

// Withdrawal withdraws a cryptocurrency to a supplied address
func (o *OKGroup) Withdrawal(symbol string, fee float64, tradePWD, address string, amount float64) (int, error) {
	v := url.Values{}
	v.Set("symbol", symbol)

	if fee != 0 {
		v.Set("chargefee", strconv.FormatFloat(fee, 'f', -1, 64))
	}
	v.Set("trade_pwd", tradePWD)
	v.Set("withdraw_address", address)
	v.Set("withdraw_amount", strconv.FormatFloat(amount, 'f', -1, 64))
	v.Set("target", "address")
	resp := WithdrawalResponse{}

	err := o.SendAuthenticatedHTTPRequest("POST", "", "spotWithdraw", v, &resp)
	if err != nil {
		return 0, err
	}

	if !resp.Result {
		return 0, errors.New("unable to process withdrawal request")
	}

	return resp.WithdrawID, nil
}

// SetErrorDefaults sets the full error default list
func (o *OKGroup) SetErrorDefaults() {
	o.ErrorCodes = map[string]error{
		//Spot Errors
		"10000": errors.New("Required field, can not be null"),
		"10001": errors.New("Request frequency too high to exceed the limit allowed"),
		"10002": errors.New("System error"),
		"10004": errors.New("Request failed - Your API key might need to be recreated"),
		"10005": errors.New("'SecretKey' does not exist"),
		"10006": errors.New("'Api_key' does not exist"),
		"10007": errors.New("Signature does not match"),
		"10008": errors.New("Illegal parameter"),
		"10009": errors.New("Order does not exist"),
		"10010": errors.New("Insufficient funds"),
		"10011": errors.New("Amount too low"),
		"10012": errors.New("Only btc_usd ltc_usd supported"),
		"10013": errors.New("Only support https request"),
		"10014": errors.New("Order price must be between 0 and 1,000,000"),
		"10015": errors.New("Order price differs from current market price too much"),
		"10016": errors.New("Insufficient coins balance"),
		"10017": errors.New("API authorization error"),
		"10018": errors.New("borrow amount less than lower limit [usd:100,btc:0.1,ltc:1]"),
		"10019": errors.New("loan agreement not checked"),
		"10020": errors.New("rate cannot exceed 1%"),
		"10021": errors.New("rate cannot less than 0.01%"),
		"10023": errors.New("fail to get latest ticker"),
		"10024": errors.New("balance not sufficient"),
		"10025": errors.New("quota is full, cannot borrow temporarily"),
		"10026": errors.New("Loan (including reserved loan) and margin cannot be withdrawn"),
		"10027": errors.New("Cannot withdraw within 24 hrs of authentication information modification"),
		"10028": errors.New("Withdrawal amount exceeds daily limit"),
		"10029": errors.New("Account has unpaid loan, please cancel/pay off the loan before withdraw"),
		"10031": errors.New("Deposits can only be withdrawn after 6 confirmations"),
		"10032": errors.New("Please enabled phone/google authenticator"),
		"10033": errors.New("Fee higher than maximum network transaction fee"),
		"10034": errors.New("Fee lower than minimum network transaction fee"),
		"10035": errors.New("Insufficient BTC/LTC"),
		"10036": errors.New("Withdrawal amount too low"),
		"10037": errors.New("Trade password not set"),
		"10040": errors.New("Withdrawal cancellation fails"),
		"10041": errors.New("Withdrawal address not exsit or approved"),
		"10042": errors.New("Admin password error"),
		"10043": errors.New("Account equity error, withdrawal failure"),
		"10044": errors.New("fail to cancel borrowing order"),
		"10047": errors.New("this function is disabled for sub-account"),
		"10048": errors.New("withdrawal information does not exist"),
		"10049": errors.New("User can not have more than 50 unfilled small orders (amount<0.15BTC)"),
		"10050": errors.New("can't cancel more than once"),
		"10051": errors.New("order completed transaction"),
		"10052": errors.New("not allowed to withdraw"),
		"10064": errors.New("after a USD deposit, that portion of assets will not be withdrawable for the next 48 hours"),
		"10100": errors.New("User account frozen"),
		"10101": errors.New("order type is wrong"),
		"10102": errors.New("incorrect ID"),
		"10103": errors.New("the private otc order's key incorrect"),
		"10216": errors.New("Non-available API"),
		"1002":  errors.New("The transaction amount exceed the balance"),
		"1003":  errors.New("The transaction amount is less than the minimum requirement"),
		"1004":  errors.New("The transaction amount is less than 0"),
		"1007":  errors.New("No trading market information"),
		"1008":  errors.New("No latest market information"),
		"1009":  errors.New("No order"),
		"1010":  errors.New("Different user of the cancelled order and the original order"),
		"1011":  errors.New("No documented user"),
		"1013":  errors.New("No order type"),
		"1014":  errors.New("No login"),
		"1015":  errors.New("No market depth information"),
		"1017":  errors.New("Date error"),
		"1018":  errors.New("Order failed"),
		"1019":  errors.New("Undo order failed"),
		"1024":  errors.New("Currency does not exist"),
		"1025":  errors.New("No chart type"),
		"1026":  errors.New("No base currency quantity"),
		"1027":  errors.New("Incorrect parameter may exceeded limits"),
		"1028":  errors.New("Reserved decimal failed"),
		"1029":  errors.New("Preparing"),
		"1030":  errors.New("Account has margin and futures, transactions can not be processed"),
		"1031":  errors.New("Insufficient Transferring Balance"),
		"1032":  errors.New("Transferring Not Allowed"),
		"1035":  errors.New("Password incorrect"),
		"1036":  errors.New("Google Verification code Invalid"),
		"1037":  errors.New("Google Verification code incorrect"),
		"1038":  errors.New("Google Verification replicated"),
		"1039":  errors.New("Message Verification Input exceed the limit"),
		"1040":  errors.New("Message Verification invalid"),
		"1041":  errors.New("Message Verification incorrect"),
		"1042":  errors.New("Wrong Google Verification Input exceed the limit"),
		"1043":  errors.New("Login password cannot be same as the trading password"),
		"1044":  errors.New("Old password incorrect"),
		"1045":  errors.New("2nd Verification Needed"),
		"1046":  errors.New("Please input old password"),
		"1048":  errors.New("Account Blocked"),
		"1201":  errors.New("Account Deleted at 00: 00"),
		"1202":  errors.New("Account Not Exist"),
		"1203":  errors.New("Insufficient Balance"),
		"1204":  errors.New("Invalid currency"),
		"1205":  errors.New("Invalid Account"),
		"1206":  errors.New("Cash Withdrawal Blocked"),
		"1207":  errors.New("Transfer Not Support"),
		"1208":  errors.New("No designated account"),
		"1209":  errors.New("Invalid api"),
		"1216":  errors.New("Market order temporarily suspended. Please send limit order"),
		"1217":  errors.New("Order was sent at ±5% of the current market price. Please resend"),
		"1218":  errors.New("Place order failed. Please try again later"),
		// Errors for both
		"HTTP ERROR CODE 403": errors.New("Too many requests, IP is shielded"),
		"Request Timed Out":   errors.New("Too many requests, IP is shielded"),
		// contract errors
		"405":   errors.New("method not allowed"),
		"20001": errors.New("User does not exist"),
		"20002": errors.New("Account frozen"),
		"20003": errors.New("Account frozen due to liquidation"),
		"20004": errors.New("Contract account frozen"),
		"20005": errors.New("User contract account does not exist"),
		"20006": errors.New("Required field missing"),
		"20007": errors.New("Illegal parameter"),
		"20008": errors.New("Contract account balance is too low"),
		"20009": errors.New("Contract status error"),
		"20010": errors.New("Risk rate ratio does not exist"),
		"20011": errors.New("Risk rate lower than 90%/80% before opening BTC position with 10x/20x leverage. or risk rate lower than 80%/60% before opening LTC position with 10x/20x leverage"),
		"20012": errors.New("Risk rate lower than 90%/80% after opening BTC position with 10x/20x leverage. or risk rate lower than 80%/60% after opening LTC position with 10x/20x leverage"),
		"20013": errors.New("Temporally no counter party price"),
		"20014": errors.New("System error"),
		"20015": errors.New("Order does not exist"),
		"20016": errors.New("Close amount bigger than your open positions"),
		"20017": errors.New("Not authorized/illegal operation"),
		"20018": errors.New("Order price cannot be more than 103% or less than 97% of the previous minute price"),
		"20019": errors.New("IP restricted from accessing the resource"),
		"20020": errors.New("secretKey does not exist"),
		"20021": errors.New("Index information does not exist"),
		"20022": errors.New("Wrong API interface (Cross margin mode shall call cross margin API, fixed margin mode shall call fixed margin API)"),
		"20023": errors.New("Account in fixed-margin mode"),
		"20024": errors.New("Signature does not match"),
		"20025": errors.New("Leverage rate error"),
		"20026": errors.New("API Permission Error"),
		"20027": errors.New("no transaction record"),
		"20028": errors.New("no such contract"),
		"20029": errors.New("Amount is large than available funds"),
		"20030": errors.New("Account still has debts"),
		"20038": errors.New("Due to regulation, this function is not available in the country/region your currently reside in"),
		"20049": errors.New("Request frequency too high"),
	}
}
