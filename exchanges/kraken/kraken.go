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
		k.SetHTTPClientUserAgent(exch.HTTPUserAgent)
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
func (k *Kraken) GetServerTime() (TimeResponse, error) {
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenServerTime)

	var response struct {
		Error  []string     `json:"error"`
		Result TimeResponse `json:"result"`
	}

	if err := k.SendHTTPRequest(path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetAssets returns a full asset list
func (k *Kraken) GetAssets() (map[string]Asset, error) {
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenAssets)

	var response struct {
		Error  []string         `json:"error"`
		Result map[string]Asset `json:"result"`
	}

	if err := k.SendHTTPRequest(path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetAssetPairs returns a full asset pair list
func (k *Kraken) GetAssetPairs() (map[string]AssetPairs, error) {
	path := fmt.Sprintf("%s/%s/public/%s", krakenAPIURL, krakenAPIVersion, krakenAssetPairs)

	var response struct {
		Error  []string              `json:"error"`
		Result map[string]AssetPairs `json:"result"`
	}

	if err := k.SendHTTPRequest(path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
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
func (k *Kraken) GetBalance() (map[string]float64, error) {
	var response struct {
		Error  []string          `json:"error"`
		Result map[string]string `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenBalance, url.Values{}, &response); err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for curency, balance := range response.Result {
		var err error
		if result[curency], err = strconv.ParseFloat(balance, 64); err != nil {
			return nil, err
		}
	}

	return result, GetError(response.Error)
}

// GetTradeBalance returns full information about your trades on Kraken
func (k *Kraken) GetTradeBalance(args ...TradeBalanceOptions) (TradeBalanceInfo, error) {
	params := url.Values{}

	if args != nil {
		if len(args[0].Aclass) != 0 {
			params.Set("aclass", args[0].Aclass)
		}

		if len(args[0].Asset) != 0 {
			params.Set("asset", args[0].Asset)
		}

	}

	var response struct {
		Error  []string         `json:"error"`
		Result TradeBalanceInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenTradeBalance, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetOpenOrders returns all current open orders
func (k *Kraken) GetOpenOrders(args ...OrderInfoOptions) (OpenOrders, error) {
	params := url.Values{}

	if args[0].Trades {
		params.Set("trades", "true")
	}

	if args[0].UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args[0].UserRef), 10))
	}

	var response struct {
		Error  []string   `json:"error"`
		Result OpenOrders `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOpenOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetClosedOrders returns a list of closed orders
func (k *Kraken) GetClosedOrders(args ...GetClosedOrdersOptions) (ClosedOrders, error) {
	params := url.Values{}

	if args != nil {
		if args[0].Trades {
			params.Set("trades", "true")
		}

		if args[0].UserRef != 0 {
			params.Set("userref", strconv.FormatInt(int64(args[0].UserRef), 10))
		}

		if len(args[0].Start) != 0 {
			params.Set("start", args[0].Start)
		}

		if len(args[0].End) != 0 {
			params.Set("end", args[0].End)
		}

		if args[0].Ofs != 0 {
			params.Set("ofs", strconv.FormatInt(args[0].Ofs, 10))
		}

		if len(args[0].CloseTime) != 0 {
			params.Set("closetime", args[0].CloseTime)
		}
	}

	var response struct {
		Error  []string     `json:"error"`
		Result ClosedOrders `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenClosedOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryOrdersInfo returns order information
func (k *Kraken) QueryOrdersInfo(args OrderInfoOptions, txid string, txids ...string) (map[string]OrderInfo, error) {
	params := url.Values{
		"txid": {txid},
	}

	if txids != nil {
		params.Set("txid", txid+","+strings.Join(txids, ","))
	}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	var response struct {
		Error  []string             `json:"error"`
		Result map[string]OrderInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenQueryOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradesHistory returns trade history information
func (k *Kraken) GetTradesHistory(args ...GetTradesHistoryOptions) (TradesHistory, error) {
	params := url.Values{}

	if args != nil {
		if len(args[0].Type) != 0 {
			params.Set("type", args[0].Type)
		}

		if args[0].Trades {
			params.Set("trades", "true")
		}

		if len(args[0].Start) != 0 {
			params.Set("start", args[0].Start)
		}

		if len(args[0].End) != 0 {
			params.Set("end", args[0].End)
		}

		if args[0].Ofs != 0 {
			params.Set("ofs", strconv.FormatInt(args[0].Ofs, 10))
		}
	}

	var response struct {
		Error  []string      `json:"error"`
		Result TradesHistory `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenTradeHistory, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryTrades returns information on a specific trade
func (k *Kraken) QueryTrades(trades bool, txid string, txids ...string) (map[string]TradeInfo, error) {
	params := url.Values{
		"txid": {txid},
	}

	if trades {
		params.Set("trades", "true")
	}

	if txids != nil {
		params.Set("txid", txid+","+strings.Join(txids, ","))
	}

	var response struct {
		Error  []string             `json:"error"`
		Result map[string]TradeInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenQueryTrades, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// OpenPositions returns current open positions
func (k *Kraken) OpenPositions(docalcs bool, txids ...string) (map[string]Position, error) {
	params := url.Values{}

	if txids != nil {
		params.Set("txid", strings.Join(txids, ","))
	}

	if docalcs {
		params.Set("docalcs", "true")
	}

	var response struct {
		Error  []string            `json:"error"`
		Result map[string]Position `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOpenPositions, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetLedgers returns current ledgers
func (k *Kraken) GetLedgers(args ...GetLedgersOptions) (Ledgers, error) {
	params := url.Values{}

	if args != nil {
		if len(args[0].Aclass) != 0 {
			params.Set("aclass", args[0].Aclass)
		}

		if len(args[0].Asset) != 0 {
			params.Set("asset", args[0].Asset)
		}

		if len(args[0].Type) != 0 {
			params.Set("type", args[0].Type)
		}

		if len(args[0].Start) != 0 {
			params.Set("start", args[0].Start)
		}

		if len(args[0].End) != 0 {
			params.Set("end", args[0].End)
		}

		if args[0].Ofs != 0 {
			params.Set("ofs", strconv.FormatInt(args[0].Ofs, 10))
		}
	}

	var response struct {
		Error  []string `json:"error"`
		Result Ledgers  `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenLedgers, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryLedgers queries an individual ledger by ID
func (k *Kraken) QueryLedgers(id string, ids ...string) (map[string]LedgerInfo, error) {
	params := url.Values{
		"id": {id},
	}

	if ids != nil {
		params.Set("id", id+","+strings.Join(ids, ","))
	}

	var response struct {
		Error  []string              `json:"error"`
		Result map[string]LedgerInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenQueryLedgers, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradeVolume returns your trade volume by currency
func (k *Kraken) GetTradeVolume(feeinfo bool, symbol ...string) (TradeVolumeResponse, error) {
	params := url.Values{}

	if symbol != nil {
		params.Set("pair", strings.Join(symbol, ","))
	}

	if feeinfo {
		params.Set("fee-info", "true")
	}

	var response struct {
		Error  []string            `json:"error"`
		Result TradeVolumeResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenTradeVolume, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// AddOrder adds a new order for Kraken exchange
func (k *Kraken) AddOrder(symbol, side, orderType string, volume, price, price2, leverage float64, args AddOrderOptions) (AddOrderResponse, error) {
	params := url.Values{
		"pair":      {symbol},
		"type":      {side},
		"ordertype": {orderType},
		"volume":    {strconv.FormatFloat(volume, 'f', -1, 64)},
	}

	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}

	if price2 != 0 {
		params.Set("price2", strconv.FormatFloat(price2, 'f', -1, 64))
	}

	if leverage != 0 {
		params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	}

	if len(args.Oflags) != 0 {
		params.Set("oflags", args.Oflags)
	}

	if len(args.StartTm) != 0 {
		params.Set("starttm", args.StartTm)
	}

	if len(args.ExpireTm) != 0 {
		params.Set("expiretm", args.ExpireTm)
	}

	if len(args.CloseOrderType) != 0 {
		params.Set("close[ordertype]", args.ExpireTm)
	}

	if args.ClosePrice != 0 {
		params.Set("close[price]", strconv.FormatFloat(args.ClosePrice, 'f', -1, 64))
	}

	if args.ClosePrice2 != 0 {
		params.Set("close[price2]", strconv.FormatFloat(args.ClosePrice2, 'f', -1, 64))
	}

	if args.Validate {
		params.Set("validate", "true")
	}

	var response struct {
		Error  []string         `json:"error"`
		Result AddOrderResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOrderPlace, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// CancelOrder cancels order by orderID
func (k *Kraken) CancelOrder(txid string) (CancelOrderResponse, error) {
	values := url.Values{
		"txid": {txid},
	}

	var response struct {
		Error  []string            `json:"error"`
		Result CancelOrderResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(krakenOrderCancel, values, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetError parse Exchange errors in response and return the first one
// Error format from API doc:
//   error = array of error messages in the format of:
//       <char-severity code><string-error category>:<string-error type>[:<string-extra info>]
//       severity code can be E for error or W for warning
func GetError(errors []string) error {

	for _, e := range errors {
		switch e[0] {
		case 'W':
			log.Printf("Kraken API warning: %v\n", e[1:])
		default:
			return fmt.Errorf("Kraken API error: %v", e[1:])
		}
	}

	return nil
}

// SendHTTPRequest sends an unauthenticated HTTP requests
func (k *Kraken) SendHTTPRequest(path string, result interface{}) error {
	return k.SendPayload("GET", path, nil, nil, result, false, k.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (k *Kraken) SendAuthenticatedHTTPRequest(method string, params url.Values, result interface{}) (err error) {
	if !k.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, k.Name)
	}

	path := fmt.Sprintf("/%s/private/%s", krakenAPIVersion, method)
	if k.Nonce.Get() == 0 {
		k.Nonce.Set(time.Now().UnixNano())
	} else {
		k.Nonce.Inc()
	}

	params.Set("nonce", k.Nonce.String())

	secret, err := common.Base64Decode(k.APISecret)
	if err != nil {
		return err
	}

	encoded := params.Encode()
	shasum := common.GetSHA256([]byte(params.Get("nonce") + encoded))
	signature := common.Base64Encode(common.GetHMAC(common.HashSHA512, append([]byte(path), shasum...), secret))

	if k.Verbose {
		log.Printf("Sending POST request to %s, path: %s, params: %s", krakenAPIURL, path, encoded)
	}

	headers := make(map[string]string)
	headers["API-Key"] = k.APIKey
	headers["API-Sign"] = signature

	return k.SendPayload("POST", krakenAPIURL+path, headers, strings.NewReader(encoded), result, true, k.Verbose)
}
