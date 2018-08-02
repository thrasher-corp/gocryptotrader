package kraken

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	krakenAPIURL        = "https://api.kraken.com"
	krakenAPIVersion    = "0"
	krakenServerTime    = "Time"
	krakenAssets        = "Assets"
	krakenAssetPairs    = "AssetPairs"
	krakenTicker        = "Ticker"
	krakenOHLC          = "OHLC"
	krakenDepth         = "Depth"
	krakenTrades        = "Trades"
	krakenSpread        = "Spread"
	krakenBalance       = "Balance"
	krakenTradeBalance  = "TradeBalance"
	krakenOpenOrders    = "OpenOrders"
	krakenClosedOrders  = "ClosedOrders"
	krakenQueryOrders   = "QueryOrders"
	krakenTradeHistory  = "TradesHistory"
	krakenQueryTrades   = "QueryTrades"
	krakenOpenPositions = "OpenPositions"
	krakenLedgers       = "Ledgers"
	krakenQueryLedgers  = "QueryLedgers"
	krakenTradeVolume   = "TradeVolume"
	krakenOrderCancel   = "CancelOrder"
	krakenOrderPlace    = "AddOrder"

	krakenAuthRate   = 0
	krakenUnauthRate = 0
)

// Kraken is the overarching type across the alphapoint package
type Kraken struct {
	exchange.Base
	CryptoFee, FiatFee float64
	Ticker             map[string]Ticker
}

// SetDefaults sets current default settings
func (k *Kraken) SetDefaults() {
	k.Name = "Kraken"
	k.Enabled = false
	k.FiatFee = 0.35
	k.CryptoFee = 0.10
	k.Verbose = false
	k.Websocket = false
	k.RESTPollingDelay = 10
	k.Ticker = make(map[string]Ticker)
	k.RequestCurrencyPairFormat.Delimiter = ""
	k.RequestCurrencyPairFormat.Uppercase = true
	k.RequestCurrencyPairFormat.Separator = ","
	k.ConfigCurrencyPairFormat.Delimiter = "-"
	k.ConfigCurrencyPairFormat.Uppercase = true
	k.AssetTypes = []string{ticker.Spot}
	k.SupportsAutoPairUpdating = true
	k.SupportsRESTTickerBatching = true
	k.Requester = request.New(k.Name, request.NewRateLimit(time.Second, krakenAuthRate), request.NewRateLimit(time.Second, krakenUnauthRate), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup sets current exchange configuration
func (k *Kraken) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		k.SetEnabled(false)
	} else {
		k.Enabled = true
		k.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		k.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		k.SetHTTPClientTimeout(exch.HTTPTimeout)
		k.RESTPollingDelay = exch.RESTPollingDelay
		k.Verbose = exch.Verbose
		k.Websocket = exch.Websocket
		k.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		k.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		k.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := k.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = k.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = k.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns current fee for either crypto or fiat
func (k *Kraken) GetFee(cryptoTrade bool) float64 {
	if cryptoTrade {
		return k.CryptoFee
	}
	return k.FiatFee
}

// GetServerTime returns current server time
func (k *Kraken) GetServerTime(unixTime bool) error {
	var result GeneralResponse
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenServerTime)

	return k.SendHTTPRequest(path, &result)
}

// GetAssets returns a full asset list
func (k *Kraken) GetAssets() error {
	var result GeneralResponse
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenAssets)

	return k.SendHTTPRequest(path, &result)
}

// GetAssetPairs returns a full asset pair list
func (k *Kraken) GetAssetPairs(result map[string]AssetPairs) error {
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenAssetPairs)

	return k.SendHTTPRequest(path, &result)
}

// GetTicker returns ticker information from kraken
func (k *Kraken) GetTicker(symbol string) (Ticker, error) {
	ticker := Ticker{}
	values := url.Values{}
	values.Set("pair", symbol)

	type Response struct {
		Error []interface{}             `json:"error"`
		Data  map[string]TickerResponse `json:"result"`
	}

	resp := Response{}
	path := fmt.Sprintf("%s/%s/public/%s?%s", krakenAPIURL, krakenAPIVersion, krakenTicker, values.Encode())

	err := k.SendHTTPRequest(path, &resp)
	if err != nil {
		return ticker, err
	}

	if len(resp.Error) > 0 {
		return ticker, fmt.Errorf("Kraken error: %s", resp.Error)
	}

	for _, y := range resp.Data {
		ticker.Ask, _ = strconv.ParseFloat(y.Ask[0], 64)
		ticker.Bid, _ = strconv.ParseFloat(y.Bid[0], 64)
		ticker.Last, _ = strconv.ParseFloat(y.Last[0], 64)
		ticker.Volume, _ = strconv.ParseFloat(y.Volume[1], 64)
		ticker.VWAP, _ = strconv.ParseFloat(y.VWAP[1], 64)
		ticker.Trades = y.Trades[1]
		ticker.Low, _ = strconv.ParseFloat(y.Low[1], 64)
		ticker.High, _ = strconv.ParseFloat(y.High[1], 64)
		ticker.Open, _ = strconv.ParseFloat(y.Open, 64)
	}
	return ticker, nil
}

// GetOHLC returns an array of open high low close values of a currency pair
func (k *Kraken) GetOHLC(symbol string) ([]OpenHighLowClose, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	type Response struct {
		Error []interface{}          `json:"error"`
		Data  map[string]interface{} `json:"result"`
	}

	var OHLC []OpenHighLowClose
	var result Response

	path := fmt.Sprintf("%s/%s/public/%s?%s", krakenAPIURL, krakenAPIVersion, krakenOHLC, values.Encode())

	err := k.SendHTTPRequest(path, &result)
	if err != nil {
		return OHLC, err
	}

	if len(result.Error) != 0 {
		return OHLC, fmt.Errorf("GetOHLC error: %s", result.Error)
	}

	for _, y := range result.Data[symbol].([]interface{}) {
		o := OpenHighLowClose{}
		for i, x := range y.([]interface{}) {
			switch i {
			case 0:
				o.Time = x.(float64)
			case 1:
				o.Open, _ = strconv.ParseFloat(x.(string), 64)
			case 2:
				o.High, _ = strconv.ParseFloat(x.(string), 64)
			case 3:
				o.Low, _ = strconv.ParseFloat(x.(string), 64)
			case 4:
				o.Close, _ = strconv.ParseFloat(x.(string), 64)
			case 5:
				o.Vwap, _ = strconv.ParseFloat(x.(string), 64)
			case 6:
				o.Volume, _ = strconv.ParseFloat(x.(string), 64)
			case 7:
				o.Count = x.(float64)
			}
		}
		OHLC = append(OHLC, o)
	}
	return OHLC, nil
}

// GetDepth returns the orderbook for a particular currency
func (k *Kraken) GetDepth(symbol string) (Orderbook, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	var result interface{}
	var orderBook Orderbook

	path := fmt.Sprintf("%s/%s/public/%s?%s", krakenAPIURL, krakenAPIVersion, krakenDepth, values.Encode())

	err := k.SendHTTPRequest(path, &result)
	if err != nil {
		return orderBook, err
	}

	data := result.(map[string]interface{})
	orderbookData := data["result"].(map[string]interface{})

	var bidsData []interface{}
	var asksData []interface{}
	for _, y := range orderbookData {
		yData := y.(map[string]interface{})
		bidsData = yData["bids"].([]interface{})
		asksData = yData["asks"].([]interface{})
	}

	processOrderbook := func(data []interface{}) ([]OrderbookBase, error) {
		var result []OrderbookBase
		for x := range data {
			entry := data[x].([]interface{})

			price, priceErr := strconv.ParseFloat(entry[0].(string), 64)
			if priceErr != nil {
				return nil, priceErr
			}

			amount, amountErr := strconv.ParseFloat(entry[1].(string), 64)
			if amountErr != nil {
				return nil, amountErr
			}

			result = append(result, OrderbookBase{Price: price, Amount: amount})
		}
		return result, nil
	}

	orderBook.Bids, err = processOrderbook(bidsData)
	if err != nil {
		return orderBook, err
	}

	orderBook.Asks, err = processOrderbook(asksData)
	if err != nil {
		return orderBook, err
	}

	return orderBook, nil
}

// GetTrades returns current trades on Kraken
func (k *Kraken) GetTrades(symbol string) ([]RecentTrades, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	var recentTrades []RecentTrades
	var result interface{}

	path := fmt.Sprintf("%s/%s/public/%s?%s", krakenAPIURL, krakenAPIVersion, krakenTrades, values.Encode())

	err := k.SendHTTPRequest(path, &result)
	if err != nil {
		return recentTrades, err
	}

	data := result.(map[string]interface{})
	tradeInfo := data["result"].(map[string]interface{})

	for _, x := range tradeInfo[symbol].([]interface{}) {
		r := RecentTrades{}
		for i, y := range x.([]interface{}) {
			switch i {
			case 0:
				r.Price, _ = strconv.ParseFloat(y.(string), 64)
			case 1:
				r.Volume, _ = strconv.ParseFloat(y.(string), 64)
			case 2:
				r.Time = y.(float64)
			case 3:
				r.BuyOrSell = y.(string)
			case 4:
				r.MarketOrLimit = y.(string)
			case 5:
				r.Miscellaneous = y.(string)
			}
		}
		recentTrades = append(recentTrades, r)
	}
	return recentTrades, nil
}

// GetSpread returns the full spread on Kraken
func (k *Kraken) GetSpread(symbol string) ([]Spread, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	var peanutButter []Spread
	var response interface{}

	path := fmt.Sprintf("%s/%s/public/%s?%s", krakenAPIURL, krakenAPIVersion, krakenSpread, values.Encode())

	err := k.SendHTTPRequest(path, &response)
	if err != nil {
		return peanutButter, err
	}

	data := response.(map[string]interface{})
	result := data["result"].(map[string]interface{})

	for _, x := range result[symbol].([]interface{}) {
		s := Spread{}
		for i, y := range x.([]interface{}) {
			switch i {
			case 0:
				s.Time = y.(float64)
			case 1:
				s.Bid, _ = strconv.ParseFloat(y.(string), 64)
			case 2:
				s.Ask, _ = strconv.ParseFloat(y.(string), 64)
			}
		}
		peanutButter = append(peanutButter, s)
	}
	return peanutButter, nil
}

// GetBalance returns your balance associated with your keys
func (k *Kraken) GetBalance() error {
	response := GeneralResponse{}
	return k.SendAuthenticatedHTTPRequest(krakenBalance, url.Values{}, &response)
}

// GetTradeBalance returns full information about your trades on Kraken
func (k *Kraken) GetTradeBalance(symbol, asset string) error {
	values := url.Values{}
	response := GeneralResponse{}

	if len(symbol) > 0 {
		values.Set("aclass", symbol)
	}
	if len(asset) > 0 {
		values.Set("asset", asset)
	}

	return k.SendAuthenticatedHTTPRequest(krakenTradeBalance, values, &response)
}

// GetOpenOrders returns all current open orders
func (k *Kraken) GetOpenOrders(showTrades bool, userref int64) error {
	values := url.Values{}
	response := GeneralResponse{}

	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	return k.SendAuthenticatedHTTPRequest(krakenOpenOrders, values, &response)
}

// GetClosedOrders returns a list of closed orders
func (k *Kraken) GetClosedOrders(showTrades bool, userref, start, end, offset int64, closetime string) error {
	values := url.Values{}
	response := GeneralResponse{}

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

	return k.SendAuthenticatedHTTPRequest(krakenClosedOrders, values, &response)
}

// QueryOrdersInfo returns order information
func (k *Kraken) QueryOrdersInfo(showTrades bool, userref, txid int64) error {
	values := url.Values{}
	response := GeneralResponse{}

	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	if txid != 0 {
		values.Set("txid", strconv.FormatInt(userref, 10))
	}

	return k.SendAuthenticatedHTTPRequest(krakenQueryOrders, values, &response)
}

// GetTradesHistory returns trade history information
func (k *Kraken) GetTradesHistory(tradeType string, showRelatedTrades bool, start, end, offset int64) error {
	values := url.Values{}
	response := GeneralResponse{}

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

	return k.SendAuthenticatedHTTPRequest(krakenTradeHistory, values, &response)
}

// QueryTrades returns information on a specific trade
func (k *Kraken) QueryTrades(txid int64, showRelatedTrades bool) error {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(txid, 10))
	response := GeneralResponse{}

	if showRelatedTrades {
		values.Set("trades", "true")
	}

	return k.SendAuthenticatedHTTPRequest(krakenQueryTrades, values, &response)
}

// OpenPositions returns current open positions
func (k *Kraken) OpenPositions(showPL bool, txids ...string) (map[string]Position, error) {
	params := url.Values{}
	if txids != nil {
		params.Set("txid", strings.Join(txids, ","))
	}
	if showPL {
		params.Set("docalcs", "true")
	}

	var response struct {
		Error  []string            `json:"error"`
		Result map[string]Position `json:"result"`
	}

	err := k.SendAuthenticatedHTTPRequest(krakenOpenPositions, params, &response)
	if len(response.Error) != 0 {
		return response.Result, fmt.Errorf("OpenPositions error: %v", response.Error[0])
	}

	return response.Result, err
}

// GetLedgers returns current ledgers
func (k *Kraken) GetLedgers(symbol, asset, ledgerType string, start, end, offset int64) error {
	values := url.Values{}
	response := GeneralResponse{}

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

	return k.SendAuthenticatedHTTPRequest(krakenLedgers, values, &response)
}

// QueryLedgers queries an individual ledger by ID
func (k *Kraken) QueryLedgers(id string) error {
	values := url.Values{}
	values.Set("id", id)
	response := GeneralResponse{}

	return k.SendAuthenticatedHTTPRequest(krakenQueryLedgers, values, &response)
}

// GetTradeVolume returns your trade volume by currency
func (k *Kraken) GetTradeVolume(symbol string) error {
	values := url.Values{}
	values.Set("pair", symbol)
	response := GeneralResponse{}

	return k.SendAuthenticatedHTTPRequest(krakenTradeVolume, values, &response)
}

// AddOrder adds a new order for Kraken exchange
func (k *Kraken) AddOrder(symbol, side, orderType string, volume, price, price2, leverage float64, args map[string]string) (AddOrderResponse, error) {
	var response struct {
		Error  []string         `json:"error"`
		Result AddOrderResponse `json:"result"`
	}

	params := url.Values{
		"pair":      {symbol},
		"type":      {side},
		"ordertype": {orderType},
	}

	params.Set("volume", strconv.FormatFloat(volume, 'f', -1, 64))
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("price2", strconv.FormatFloat(price2, 'f', -1, 64))
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))

	if value, ok := args["oflags"]; ok {
		params.Set("oflags", value)
	}
	if value, ok := args["starttm"]; ok {
		params.Set("starttm", value)
	}
	if value, ok := args["expiretm"]; ok {
		params.Set("expiretm", value)
	}
	if value, ok := args["validate"]; ok {
		params.Set("validate", value)
	}
	if value, ok := args["close_order_type"]; ok {
		params.Set("close[ordertype]", value)
	}
	if value, ok := args["close_price"]; ok {
		params.Set("close[price]", value)
	}
	if value, ok := args["close_price2"]; ok {
		params.Set("close[price2]", value)
	}
	if value, ok := args["trading_agreement"]; ok {
		params.Set("trading_agreement", value)
	}

	err := k.SendAuthenticatedHTTPRequest(krakenOrderPlace, params, &response)
	if len(response.Error) != 0 {
		return response.Result, fmt.Errorf("AddOrder error: %v", response.Error[0])
	}

	return response.Result, err
}

// CancelOrder cancels order by orderID
func (k *Kraken) CancelOrder(orderID int64) error {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(orderID, 10))
	response := GeneralResponse{}

	return k.SendAuthenticatedHTTPRequest(krakenOrderCancel, values, &response)
}

// SendHTTPRequest sends an unauthenticated HTTP requests
func (k *Kraken) SendHTTPRequest(path string, result interface{}) error {
	return k.SendPayload("GET", path, nil, nil, result, false, k.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (k *Kraken) SendAuthenticatedHTTPRequest(method string, values url.Values, result interface{}) (err error) {
	if !k.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, k.Name)
	}

	path := fmt.Sprintf("/%s/private/%s", krakenAPIVersion, method)
	if k.Nonce.Get() == 0 {
		k.Nonce.Set(time.Now().UnixNano())
	} else {
		k.Nonce.Inc()
	}

	values.Set("nonce", k.Nonce.String())

	secret, err := common.Base64Decode(k.APISecret)
	if err != nil {
		return err
	}

	shasum := common.GetSHA256([]byte(values.Get("nonce") + values.Encode()))
	signature := common.Base64Encode(common.GetHMAC(common.HashSHA512, append([]byte(path), shasum...), secret))

	if k.Verbose {
		log.Printf("Sending POST request to %s, path: %s.", krakenAPIURL, path)
	}

	headers := make(map[string]string)
	headers["API-Key"] = k.APIKey
	headers["API-Sign"] = signature

	return k.SendPayload("POST", krakenAPIURL+path, headers, strings.NewReader(values.Encode()), result, true, k.Verbose)
}
