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
}

// Setup sets current exchange configuration
func (k *Kraken) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		k.SetEnabled(false)
	} else {
		k.Enabled = true
		k.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		k.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
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
func (k *Kraken) GetServerTime(unixTime bool) (interface{}, error) {
	var result GeneralResponse
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenServerTime)

	err := common.SendHTTPGetRequest(path, true, k.Verbose, &result)
	if err != nil {
		return nil, fmt.Errorf("getServerTime() error %s", err)
	}

	if unixTime {
		return result.Result["unixtime"], nil
	}
	return result.Result["rfc1123"], nil
}

// GetAssets returns a full asset list
func (k *Kraken) GetAssets() (interface{}, error) {
	var result GeneralResponse
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenAssets)

	return result.Result, common.SendHTTPGetRequest(path, true, k.Verbose, &result)
}

// GetAssetPairs returns a full asset pair list
func (k *Kraken) GetAssetPairs() (map[string]AssetPairs, error) {
	type Response struct {
		Result map[string]AssetPairs `json:"result"`
		Error  []interface{}         `json:"error"`
	}

	response := Response{}
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenAssetPairs)

	return response.Result, common.SendHTTPGetRequest(path, true, k.Verbose, &response)
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

	err := common.SendHTTPGetRequest(path, true, k.Verbose, &resp)
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

	err := common.SendHTTPGetRequest(path, true, k.Verbose, &result)
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

	err := common.SendHTTPGetRequest(path, true, k.Verbose, &result)
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

	err := common.SendHTTPGetRequest(path, true, k.Verbose, &result)
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

	err := common.SendHTTPGetRequest(path, true, k.Verbose, &response)
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
func (k *Kraken) GetBalance() (interface{}, error) {
	return k.SendAuthenticatedHTTPRequest(krakenBalance, url.Values{})
}

// GetTradeBalance returns full information about your trades on Kraken
func (k *Kraken) GetTradeBalance(symbol, asset string) (interface{}, error) {
	values := url.Values{}

	if len(symbol) > 0 {
		values.Set("aclass", symbol)
	}
	if len(asset) > 0 {
		values.Set("asset", asset)
	}

	return k.SendAuthenticatedHTTPRequest(krakenTradeBalance, values)
}

// GetOpenOrders returns all current open orders
func (k *Kraken) GetOpenOrders(showTrades bool, userref int64) (interface{}, error) {
	values := url.Values{}

	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	return k.SendAuthenticatedHTTPRequest(krakenOpenOrders, values)
}

// GetClosedOrders returns a list of closed orders
func (k *Kraken) GetClosedOrders(showTrades bool, userref, start, end, offset int64, closetime string) (interface{}, error) {
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

	return k.SendAuthenticatedHTTPRequest(krakenClosedOrders, values)
}

// QueryOrdersInfo returns order information
func (k *Kraken) QueryOrdersInfo(showTrades bool, userref, txid int64) (interface{}, error) {
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

	return k.SendAuthenticatedHTTPRequest(krakenQueryOrders, values)
}

// GetTradesHistory returns trade history information
func (k *Kraken) GetTradesHistory(tradeType string, showRelatedTrades bool, start, end, offset int64) (interface{}, error) {
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

	return k.SendAuthenticatedHTTPRequest(krakenTradeHistory, values)
}

// QueryTrades returns information on a specific trade
func (k *Kraken) QueryTrades(txid int64, showRelatedTrades bool) (interface{}, error) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(txid, 10))

	if showRelatedTrades {
		values.Set("trades", "true")
	}

	return k.SendAuthenticatedHTTPRequest(krakenQueryTrades, values)
}

// OpenPositions returns current open positions
func (k *Kraken) OpenPositions(txid int64, showPL bool) (interface{}, error) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(txid, 10))

	if showPL {
		values.Set("docalcs", "true")
	}

	return k.SendAuthenticatedHTTPRequest(krakenOpenPositions, values)
}

// GetLedgers returns current ledgers
func (k *Kraken) GetLedgers(symbol, asset, ledgerType string, start, end, offset int64) (interface{}, error) {
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

	return k.SendAuthenticatedHTTPRequest(krakenLedgers, values)
}

// QueryLedgers queries an individual ledger by ID
func (k *Kraken) QueryLedgers(id string) (interface{}, error) {
	values := url.Values{}
	values.Set("id", id)

	return k.SendAuthenticatedHTTPRequest(krakenQueryLedgers, values)
}

// GetTradeVolume returns your trade volume by currency
func (k *Kraken) GetTradeVolume(symbol string) (interface{}, error) {
	values := url.Values{}
	values.Set("pair", symbol)

	return k.SendAuthenticatedHTTPRequest(krakenTradeVolume, values)
}

// AddOrder adds a new order for Kraken exchange
func (k *Kraken) AddOrder(symbol, side, orderType string, price, price2, volume, leverage, position float64) (interface{}, error) {
	values := url.Values{}
	values.Set("pairs", symbol)
	values.Set("type", side)
	values.Set("ordertype", orderType)
	values.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	values.Set("price2", strconv.FormatFloat(price, 'f', -1, 64))
	values.Set("volume", strconv.FormatFloat(volume, 'f', -1, 64))
	values.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	values.Set("position", strconv.FormatFloat(position, 'f', -1, 64))

	return k.SendAuthenticatedHTTPRequest(krakenOrderPlace, values)
}

// CancelOrder cancels order by orderID
func (k *Kraken) CancelOrder(orderID int64) (interface{}, error) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(orderID, 10))

	return k.SendAuthenticatedHTTPRequest(krakenOrderCancel, values)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (k *Kraken) SendAuthenticatedHTTPRequest(method string, values url.Values) (interface{}, error) {
	if !k.AuthenticatedAPISupport {
		return nil, fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, k.Name)
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
		return nil, err
	}

	shasum := common.GetSHA256([]byte(values.Get("nonce") + values.Encode()))
	signature := common.Base64Encode(common.GetHMAC(common.HashSHA512, append([]byte(path), shasum...), secret))

	if k.Verbose {
		log.Printf("Sending POST request to %s, path: %s.", krakenAPIURL, path)
	}

	headers := make(map[string]string)
	headers["API-Key"] = k.APIKey
	headers["API-Sign"] = signature

	rawResp, err := common.SendHTTPRequest("POST", krakenAPIURL+path, headers, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	if k.Verbose {
		log.Printf("Received raw: \n%s\n", rawResp)
	}

	var resp interface{}

	err = common.JSONDecode([]byte(rawResp), &resp)
	if err != nil {
		return nil, err
	}

	data := resp.(map[string]interface{})
	if len(data["error"].([]interface{})) != 0 {
		return nil, fmt.Errorf("kraken AuthenticattedHTTPRequest error: %s", data["error"])
	}

	return data["result"].(interface{}), nil
}
