package main

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
	KRAKEN_API_URL        = "https://api.kraken.com"
	KRAKEN_API_VERSION    = "0"
	KRAKEN_SERVER_TIME    = "Time"
	KRAKEN_ASSETS         = "Assets"
	KRAKEN_ASSET_PAIRS    = "AssetPairs"
	KRAKEN_TICKER         = "Ticker"
	KRAKEN_OHLC           = "OHLC"
	KRAKEN_DEPTH          = "Depth"
	KRAKEN_TRADES         = "Trades"
	KRAKEN_SPREAD         = "Spread"
	KRAKEN_BALANCE        = "Balance"
	KRAKEN_TRADE_BALANCE  = "TradeBalance"
	KRAKEN_OPEN_ORDERS    = "OpenOrders"
	KRAKEN_CLOSED_ORDERS  = "ClosedOrders"
	KRAKEN_QUERY_ORDERS   = "QueryOrders"
	KRAKEN_TRADES_HISTORY = "TradesHistory"
	KRAKEN_QUERY_TRADES   = "QueryTrades"
	KRAKEN_OPEN_POSITIONS = "OpenPositions"
	KRAKEN_LEDGERS        = "Ledgers"
	KRAKEN_QUERY_LEDGERS  = "QueryLedgers"
	KRAKEN_TRADE_VOLUME   = "TradeVolume"
	KRAKEN_ORDER_CANCEL   = "CancelOrder"
	KRAKEN_ORDER_PLACE    = "AddOrder"
)

type Kraken struct {
	exchange.ExchangeBase
	CryptoFee, FiatFee float64
	Ticker             map[string]KrakenTicker
}

func (k *Kraken) SetDefaults() {
	k.Name = "Kraken"
	k.Enabled = false
	k.FiatFee = 0.35
	k.CryptoFee = 0.10
	k.Verbose = false
	k.Websocket = false
	k.RESTPollingDelay = 10
	k.Ticker = make(map[string]KrakenTicker)
}

func (k *Kraken) GetName() string {
	return k.Name
}

func (k *Kraken) SetEnabled(enabled bool) {
	k.Enabled = enabled
}

func (k *Kraken) IsEnabled() bool {
	return k.Enabled
}

func (k *Kraken) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		k.SetEnabled(false)
	} else {
		k.Enabled = true
		k.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		k.SetAPIKeys(exch.APIKey, exch.APISecret)
		k.RESTPollingDelay = exch.RESTPollingDelay
		k.Verbose = exch.Verbose
		k.Websocket = exch.Websocket
		k.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		k.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		k.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *Kraken) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (k *Kraken) Start() {
	go k.Run()
}

func (k *Kraken) SetAPIKeys(apiKey, apiSecret string) {
	k.APIKey = apiKey
	k.APISecret = apiSecret
}

func (k *Kraken) GetFee(cryptoTrade bool) float64 {
	if cryptoTrade {
		return k.CryptoFee
	} else {
		return k.FiatFee
	}
}

func (k *Kraken) Run() {
	if k.Verbose {
		log.Printf("%s polling delay: %ds.\n", k.GetName(), k.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", k.GetName(), len(k.EnabledPairs), k.EnabledPairs)
	}

	assetPairs, err := k.GetAssetPairs()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", k.GetName())
	} else {
		var exchangeProducts []string
		for _, v := range assetPairs {
			exchangeProducts = append(exchangeProducts, v.Altname)
		}
		diff := common.StringSliceDifference(k.AvailablePairs, exchangeProducts)
		if len(diff) > 0 {
			exch, err := bot.config.GetExchangeConfig(k.Name)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("%s Updating available pairs. Difference: %s.\n", k.Name, diff)
				exch.AvailablePairs = common.JoinStrings(exchangeProducts, ",")
				bot.config.UpdateExchangeConfig(exch)
			}
		}
	}

	for k.Enabled {
		err := k.GetTicker(common.JoinStrings(k.EnabledPairs, ","))
		if err != nil {
			log.Println(err)
		} else {
			for _, x := range k.EnabledPairs {
				ticker := k.Ticker[x]
				log.Printf("Kraken %s Last %f High %f Low %f Volume %f\n", x, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				AddExchangeInfo(k.GetName(), x[0:3], x[3:], ticker.Last, ticker.Volume)
			}
		}
		time.Sleep(time.Second * k.RESTPollingDelay)
	}
}

func (k *Kraken) GetServerTime() error {
	var result interface{}
	path := fmt.Sprintf("%s/%s/public/%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_SERVER_TIME)
	err := common.SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return err
	}

	log.Println(result)
	return nil
}

func (k *Kraken) GetAssets() error {
	var result interface{}
	path := fmt.Sprintf("%s/%s/public/%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_ASSETS)
	err := common.SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return err
	}

	log.Println(result)
	return nil
}

type KrakenAssetPairs struct {
	Altname           string      `json:"altname"`
	AclassBase        string      `json:"aclass_base"`
	Base              string      `json:"base"`
	AclassQuote       string      `json:"aclass_quote"`
	Quote             string      `json:"quote"`
	Lot               string      `json:"lot"`
	PairDecimals      int         `json:"pair_decimals"`
	LotDecimals       int         `json:"lot_decimals"`
	LotMultiplier     int         `json:"lot_multiplier"`
	LeverageBuy       []int       `json:"leverage_buy"`
	LeverageSell      []int       `json:"leverage_sell"`
	Fees              [][]float64 `json:"fees"`
	FeesMaker         [][]float64 `json:"fees_maker"`
	FeeVolumeCurrency string      `json:"fee_volume_currency"`
	MarginCall        int         `json:"margin_call"`
	MarginStop        int         `json:"margin_stop"`
}

func (k *Kraken) GetAssetPairs() (map[string]KrakenAssetPairs, error) {
	type Response struct {
		Result map[string]KrakenAssetPairs `json:"result"`
		Error  []interface{}               `json:"error"`
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/public/%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_ASSET_PAIRS)
	err := common.SendHTTPGetRequest(path, true, &response)

	if err != nil {
		return nil, err
	}

	return response.Result, nil
}

type KrakenTicker struct {
	Ask    float64
	Bid    float64
	Last   float64
	Volume float64
	VWAP   float64
	Trades int64
	Low    float64
	High   float64
	Open   float64
}

type KrakenTickerResponse struct {
	Ask    []string `json:"a"`
	Bid    []string `json:"b"`
	Last   []string `json:"c"`
	Volume []string `json:"v"`
	VWAP   []string `json:"p"`
	Trades []int64  `json:"t"`
	Low    []string `json:"l"`
	High   []string `json:"h"`
	Open   string   `json:"o"`
}

func (k *Kraken) GetTicker(symbol string) error {
	values := url.Values{}
	values.Set("pair", symbol)

	type Response struct {
		Error []interface{}                   `json:"error"`
		Data  map[string]KrakenTickerResponse `json:"result"`
	}

	resp := Response{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_TICKER, values.Encode())
	err := common.SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return err
	}

	if len(resp.Error) > 0 {
		return errors.New(fmt.Sprintf("Kraken error: %s", resp.Error))
	}

	for x, y := range resp.Data {
		x = x[1:4] + x[5:]
		ticker := KrakenTicker{}
		ticker.Ask, _ = strconv.ParseFloat(y.Ask[0], 64)
		ticker.Bid, _ = strconv.ParseFloat(y.Bid[0], 64)
		ticker.Last, _ = strconv.ParseFloat(y.Last[0], 64)
		ticker.Volume, _ = strconv.ParseFloat(y.Volume[1], 64)
		ticker.VWAP, _ = strconv.ParseFloat(y.VWAP[1], 64)
		ticker.Trades = y.Trades[1]
		ticker.Low, _ = strconv.ParseFloat(y.Low[1], 64)
		ticker.High, _ = strconv.ParseFloat(y.High[1], 64)
		ticker.Open, _ = strconv.ParseFloat(y.Open, 64)
		k.Ticker[x] = ticker
	}
	return nil
}

//This will return the TickerPrice struct when tickers are completed here..
func (k *Kraken) GetTickerPrice(currency string) (TickerPrice, error) {
	var tickerPrice TickerPrice
	/*
		ticker, err := i.GetTicker(currency)
		if err != nil {
			log.Println(err)
			return tickerPrice
		}
		tickerPrice.Ask = ticker.Ask
		tickerPrice.Bid = ticker.Bid
	*/
	return tickerPrice, nil
}

func (k *Kraken) GetOHLC(symbol string) error {
	values := url.Values{}
	values.Set("pair", symbol)

	var result interface{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_OHLC, values.Encode())
	err := common.SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return err
	}

	log.Println(result)
	return nil
}

func (k *Kraken) GetDepth(symbol string) error {
	values := url.Values{}
	values.Set("pair", symbol)

	var result interface{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_DEPTH, values.Encode())
	err := common.SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return err
	}

	log.Println(result)
	return nil
}

func (k *Kraken) GetTrades(symbol string) error {
	values := url.Values{}
	values.Set("pair", symbol)

	var result interface{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_TRADES, values.Encode())
	err := common.SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return err
	}

	log.Println(result)
	return nil
}

func (k *Kraken) GetSpread(symbol string) {
	values := url.Values{}
	values.Set("pair", symbol)

	var result interface{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", KRAKEN_API_URL, KRAKEN_API_VERSION, KRAKEN_SPREAD, values.Encode())
	err := common.SendHTTPGetRequest(path, true, &result)

	if err != nil {
		log.Println(err)
		return
	}
}

func (k *Kraken) GetBalance() {
	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_BALANCE, url.Values{})

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

//TODO: Retrieve Kraken info
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Kraken exchange
func (e *Kraken) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}

func (k *Kraken) GetTradeBalance(symbol, asset string) {
	values := url.Values{}

	if len(symbol) > 0 {
		values.Set("aclass", symbol)
	}

	if len(asset) > 0 {
		values.Set("asset", asset)
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_TRADE_BALANCE, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetOpenOrders(showTrades bool, userref int64) {
	values := url.Values{}

	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_OPEN_ORDERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetClosedOrders(showTrades bool, userref, start, end, offset int64, closetime string) {
	values := url.Values{}

	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	if start != 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end != 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if offset != 0 {
		values.Set("ofs", strconv.FormatInt(offset, 10))
	}

	if len(closetime) > 0 {
		values.Set("closetime", closetime)
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_CLOSED_ORDERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) QueryOrdersInfo(showTrades bool, userref, txid int64) {
	values := url.Values{}

	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	if txid != 0 {
		values.Set("txid", strconv.FormatInt(userref, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_QUERY_ORDERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetTradesHistory(tradeType string, showRelatedTrades bool, start, end, offset int64) {
	values := url.Values{}

	if len(tradeType) > 0 {
		values.Set("aclass", tradeType)
	}

	if showRelatedTrades {
		values.Set("trades", "true")
	}

	if start != 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end != 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if offset != 0 {
		values.Set("offset", strconv.FormatInt(offset, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_TRADES_HISTORY, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) QueryTrades(txid int64, showRelatedTrades bool) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(txid, 10))

	if showRelatedTrades {
		values.Set("trades", "true")
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_QUERY_TRADES, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) OpenPositions(txid int64, showPL bool) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(txid, 10))

	if showPL {
		values.Set("docalcs", "true")
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_OPEN_POSITIONS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetLedgers(symbol, asset, ledgerType string, start, end, offset int64) {
	values := url.Values{}

	if len(symbol) > 0 {
		values.Set("aclass", symbol)
	}

	if len(asset) > 0 {
		values.Set("asset", asset)
	}

	if len(ledgerType) > 0 {
		values.Set("type", ledgerType)
	}

	if start != 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end != 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if offset != 0 {
		values.Set("offset", strconv.FormatInt(offset, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_LEDGERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) QueryLedgers(id string) {
	values := url.Values{}
	values.Set("id", id)

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_QUERY_LEDGERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetTradeVolume(symbol string) {
	values := url.Values{}
	values.Set("pair", symbol)

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_TRADE_VOLUME, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) AddOrder(symbol, side, orderType string, price, price2, volume, leverage, position float64) {
	values := url.Values{}
	values.Set("pairs", symbol)
	values.Set("type", side)
	values.Set("ordertype", orderType)
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	values.Set("price2", strconv.FormatFloat(price, 'f', -1, 64))
	values.Set("volume", strconv.FormatFloat(volume, 'f', -1, 64))
	values.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	values.Set("position", strconv.FormatFloat(position, 'f', -1, 64))

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_ORDER_PLACE, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) CancelOrder(orderID int64) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(orderID, 10))

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_ORDER_CANCEL, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) SendAuthenticatedHTTPRequest(method string, values url.Values) (interface{}, error) {
	path := fmt.Sprintf("/%s/private/%s", KRAKEN_API_VERSION, method)
	values.Set("nonce", strconv.FormatInt(time.Now().UnixNano(), 10))
	secret, err := common.Base64Decode(k.APISecret)

	if err != nil {
		return nil, err
	}

	shasum := common.GetSHA256([]byte(values.Get("nonce") + values.Encode()))
	signature := common.Base64Encode(common.GetHMAC(common.HASH_SHA512, append([]byte(path), shasum...), secret))

	if k.Verbose {
		log.Printf("Sending POST request to %s, path: %s.", KRAKEN_API_URL, path)
	}

	headers := make(map[string]string)
	headers["API-Key"] = k.APIKey
	headers["API-Sign"] = signature

	resp, err := common.SendHTTPRequest("POST", KRAKEN_API_URL+path, headers, strings.NewReader(values.Encode()))

	if err != nil {
		return nil, err
	}

	if k.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	return resp, nil
}
