package kraken

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	krakenAPIURL         = "https://api.kraken.com"
	krakenFuturesURL     = "https://futures.kraken.com"
	futuresURL           = "https://futures.kraken.com/derivatives"
	krakenSpotVersion    = "0"
	krakenFuturesVersion = "3"
)

// Kraken is the overarching type across the alphapoint package
type Kraken struct {
	exchange.Base
	wsRequestMtx sync.Mutex
}

// GetServerTime returns current server time
func (k *Kraken) GetServerTime() (TimeResponse, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenServerTime)

	var response struct {
		Error  []string     `json:"error"`
		Result TimeResponse `json:"result"`
	}

	if err := k.SendHTTPRequest(exchange.RestSpot, path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// SeedAssets seeds Kraken's asset list and stores it in the
// asset translator
func (k *Kraken) SeedAssets() error {
	assets, err := k.GetAssets()
	if err != nil {
		return err
	}
	for orig, val := range assets {
		assetTranslator.Seed(orig, val.Altname)
	}

	assetPairs, err := k.GetAssetPairs([]string{}, "")
	if err != nil {
		return err
	}
	for k := range assetPairs {
		assetTranslator.Seed(k, assetPairs[k].Altname)
	}
	return nil
}

// GetAssets returns a full asset list
func (k *Kraken) GetAssets() (map[string]*Asset, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenAssets)

	var response struct {
		Error  []string          `json:"error"`
		Result map[string]*Asset `json:"result"`
	}

	if err := k.SendHTTPRequest(exchange.RestSpot, path, &response); err != nil {
		return response.Result, err
	}
	return response.Result, GetError(response.Error)
}

// GetAssetPairs returns a full asset pair list
// Parameter 'info' only supports 4 strings: "fees", "leverage", "margin", "info" <- (default)
func (k *Kraken) GetAssetPairs(assetPairs []string, info string) (map[string]AssetPairs, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenAssetPairs)
	params := url.Values{}
	var assets string
	if len(assetPairs) != 0 {
		assets = strings.Join(assetPairs, ",")
		params.Set("pair", assets)
	}
	var response struct {
		Error  []string              `json:"error"`
		Result map[string]AssetPairs `json:"result"`
	}
	if info != "" {
		if info != "margin" && info != "leverage" && info != "fees" && info != "info" {
			return response.Result, errors.New("parameter info can only be 'asset', 'margin', 'fees' or 'leverage'")
		}
		params.Set("info", info)
	}
	if err := k.SendHTTPRequest(exchange.RestSpot, path+params.Encode(), &response); err != nil {
		return response.Result, err
	}
	return response.Result, GetError(response.Error)
}

// GetTicker returns ticker information from kraken
func (k *Kraken) GetTicker(symbol currency.Pair) (Ticker, error) {
	tick := Ticker{}
	values := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return tick, err
	}
	values.Set("pair", symbolValue)

	type Response struct {
		Error []interface{}             `json:"error"`
		Data  map[string]TickerResponse `json:"result"`
	}

	resp := Response{}
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenTicker, values.Encode())

	err = k.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return tick, err
	}

	if len(resp.Error) > 0 {
		return tick, fmt.Errorf("%s error: %s", k.Name, resp.Error)
	}

	for i := range resp.Data {
		tick.Ask, _ = strconv.ParseFloat(resp.Data[i].Ask[0], 64)
		tick.Bid, _ = strconv.ParseFloat(resp.Data[i].Bid[0], 64)
		tick.Last, _ = strconv.ParseFloat(resp.Data[i].Last[0], 64)
		tick.Volume, _ = strconv.ParseFloat(resp.Data[i].Volume[1], 64)
		tick.VolumeWeightedAveragePrice, _ = strconv.ParseFloat(resp.Data[i].VolumeWeightedAveragePrice[1], 64)
		tick.Trades = resp.Data[i].Trades[1]
		tick.Low, _ = strconv.ParseFloat(resp.Data[i].Low[1], 64)
		tick.High, _ = strconv.ParseFloat(resp.Data[i].High[1], 64)
		tick.Open, _ = strconv.ParseFloat(resp.Data[i].Open, 64)
	}
	return tick, nil
}

// GetTickers supports fetching multiple tickers from Kraken
// pairList must be in the format pairs separated by commas
// ("LTCUSD,ETCUSD")
func (k *Kraken) GetTickers(pairList string) (map[string]Ticker, error) {
	values := url.Values{}
	values.Set("pair", pairList)

	type Response struct {
		Error []interface{}             `json:"error"`
		Data  map[string]TickerResponse `json:"result"`
	}

	resp := Response{}
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenTicker, values.Encode())

	err := k.SendHTTPRequest(exchange.RestSpot, path, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Error) > 0 {
		return nil, fmt.Errorf("%s error: %s", k.Name, resp.Error)
	}

	tickers := make(map[string]Ticker)

	for i := range resp.Data {
		tick := Ticker{}
		tick.Ask, _ = strconv.ParseFloat(resp.Data[i].Ask[0], 64)
		tick.Bid, _ = strconv.ParseFloat(resp.Data[i].Bid[0], 64)
		tick.Last, _ = strconv.ParseFloat(resp.Data[i].Last[0], 64)
		tick.Volume, _ = strconv.ParseFloat(resp.Data[i].Volume[1], 64)
		tick.VolumeWeightedAveragePrice, _ = strconv.ParseFloat(resp.Data[i].VolumeWeightedAveragePrice[1], 64)
		tick.Trades = resp.Data[i].Trades[1]
		tick.Low, _ = strconv.ParseFloat(resp.Data[i].Low[1], 64)
		tick.High, _ = strconv.ParseFloat(resp.Data[i].High[1], 64)
		tick.Open, _ = strconv.ParseFloat(resp.Data[i].Open, 64)
		tickers[i] = tick
	}
	return tickers, nil
}

// GetOHLC returns an array of open high low close values of a currency pair
func (k *Kraken) GetOHLC(symbol currency.Pair, interval string) ([]OpenHighLowClose, error) {
	values := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	translatedAsset := assetTranslator.LookupCurrency(symbolValue)
	if translatedAsset == "" {
		translatedAsset = symbolValue
	}
	values.Set("pair", translatedAsset)
	values.Set("interval", interval)
	type Response struct {
		Error []interface{}          `json:"error"`
		Data  map[string]interface{} `json:"result"`
	}

	var OHLC []OpenHighLowClose
	var result Response

	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenOHLC, values.Encode())

	err = k.SendHTTPRequest(exchange.RestSpot, path, &result)
	if err != nil {
		return OHLC, err
	}

	if len(result.Error) != 0 {
		return OHLC, fmt.Errorf("getOHLC error: %s", result.Error)
	}

	_, ok := result.Data[translatedAsset].([]interface{})
	if !ok {
		return nil, errors.New("invalid data returned")
	}

	for _, y := range result.Data[translatedAsset].([]interface{}) {
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
				o.VolumeWeightedAveragePrice, _ = strconv.ParseFloat(x.(string), 64)
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
func (k *Kraken) GetDepth(symbol currency.Pair) (Orderbook, error) {
	var result interface{}
	var orderBook Orderbook
	values := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return orderBook, err
	}
	values.Set("pair", symbolValue)
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenDepth, values.Encode())
	err = k.SendHTTPRequest(exchange.RestSpot, path, &result)
	if err != nil {
		return orderBook, err
	}

	if result == nil {
		return orderBook, fmt.Errorf("%s GetDepth result is nil", k.Name)
	}

	data := result.(map[string]interface{})
	if data["result"] == nil {
		return orderBook, fmt.Errorf("%s GetDepth data[result] is nil", k.Name)
	}
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
	return orderBook, err
}

// GetTrades returns current trades on Kraken
func (k *Kraken) GetTrades(symbol currency.Pair) ([]RecentTrades, error) {
	values := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	translatedAsset := assetTranslator.LookupCurrency(symbolValue)
	values.Set("pair", translatedAsset)

	var recentTrades []RecentTrades
	var result interface{}

	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenTrades, values.Encode())

	err = k.SendHTTPRequest(exchange.RestSpot, path, &result)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		return nil, errors.New("unable to parse trade data")
	}
	var dataError interface{}
	dataError, ok = data["error"]
	if ok {
		var dataErrorInterface interface{}
		dataErrorInterface, ok = dataError.(interface{})
		if ok {
			var errorList []interface{}
			errorList, ok = dataErrorInterface.([]interface{})
			if ok {
				var errs common.Errors
				for i := range errorList {
					var errString string
					errString, ok = errorList[i].(string)
					if !ok {
						continue
					}
					errs = append(errs, errors.New(errString))
				}
				if len(errs) > 0 {
					return nil, errs
				}
			}
		}
	}

	var resultField interface{}
	resultField, ok = data["result"]
	if !ok {
		return nil, errors.New("unable to find field 'result'")
	}
	var tradeInfo map[string]interface{}
	tradeInfo, ok = resultField.(map[string]interface{})
	if !ok {
		return nil, errors.New("unable to parse field 'result'")
	}

	var trades []interface{}
	var tradesForSymbol interface{}
	tradesForSymbol, ok = tradeInfo[translatedAsset]
	if !ok {
		return nil, fmt.Errorf("no data returned for symbol %v", symbol)
	}

	trades, ok = tradesForSymbol.([]interface{})
	if !ok {
		return nil, fmt.Errorf("no trades returned for symbol %v", symbol)
	}

	for _, x := range trades {
		r := RecentTrades{}
		var individualTrade []interface{}
		individualTrade, ok = x.([]interface{})
		if !ok {
			return nil, errors.New("unable to parse individual trade data")
		}
		if len(individualTrade) != 6 {
			return nil, errors.New("unrecognised trade data received")
		}
		r.Price, err = strconv.ParseFloat(individualTrade[0].(string), 64)
		if err != nil {
			return nil, err
		}
		r.Volume, err = strconv.ParseFloat(individualTrade[1].(string), 64)
		if err != nil {
			return nil, err
		}
		r.Time, ok = individualTrade[2].(float64)
		if !ok {
			return nil, errors.New("unable to parse time for individual trade data")
		}
		r.BuyOrSell, ok = individualTrade[3].(string)
		if !ok {
			return nil, errors.New("unable to parse order side for individual trade data")
		}
		r.MarketOrLimit, ok = individualTrade[4].(string)
		if !ok {
			return nil, errors.New("unable to parse order type for individual trade data")
		}
		r.Miscellaneous, ok = individualTrade[5].(string)
		if !ok {
			return nil, errors.New("unable to parse misc field for individual trade data")
		}
		recentTrades = append(recentTrades, r)
	}
	return recentTrades, nil
}

// GetSpread returns the full spread on Kraken
func (k *Kraken) GetSpread(symbol currency.Pair) ([]Spread, error) {
	values := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	values.Set("pair", symbolValue)

	var peanutButter []Spread
	var response interface{}

	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenSpread, values.Encode())

	err = k.SendHTTPRequest(exchange.RestSpot, path, &response)
	if err != nil {
		return peanutButter, err
	}

	data := response.(map[string]interface{})
	result := data["result"].(map[string]interface{})

	for _, x := range result[symbolValue].([]interface{}) {
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

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenBalance, url.Values{}, &response); err != nil {
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

// GetWithdrawInfo gets withdrawal fees
func (k *Kraken) GetWithdrawInfo(currency string, amount float64) (WithdrawInformation, error) {
	var response struct {
		Error  []string            `json:"error"`
		Result WithdrawInformation `json:"result"`
	}
	params := url.Values{}
	params.Set("asset", currency)
	params.Set("key", "")
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenWithdrawInfo, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// Withdraw withdraws funds
func (k *Kraken) Withdraw(asset, key string, amount float64) (string, error) {
	var response struct {
		Error       []string `json:"error"`
		ReferenceID string   `json:"refid"`
	}
	params := url.Values{}
	params.Set("asset", asset)
	params.Set("key", key)
	params.Set("amount", fmt.Sprintf("%f", amount))

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenWithdraw, params, &response); err != nil {
		return response.ReferenceID, err
	}

	return response.ReferenceID, GetError(response.Error)
}

// GetDepositMethods gets withdrawal fees
func (k *Kraken) GetDepositMethods(currency string) ([]DepositMethods, error) {
	var response struct {
		Error  []string         `json:"error"`
		Result []DepositMethods `json:"result"`
	}
	params := url.Values{}
	params.Set("asset", currency)

	err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenDepositMethods, params, &response)
	if err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradeBalance returns full information about your trades on Kraken
func (k *Kraken) GetTradeBalance(args ...TradeBalanceOptions) (TradeBalanceInfo, error) {
	params := url.Values{}

	if args != nil {
		if len(args[0].Aclass) > 0 {
			params.Set("aclass", args[0].Aclass)
		}

		if len(args[0].Asset) > 0 {
			params.Set("asset", args[0].Asset)
		}
	}

	var response struct {
		Error  []string         `json:"error"`
		Result TradeBalanceInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenTradeBalance, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetOpenOrders returns all current open orders
func (k *Kraken) GetOpenOrders(args OrderInfoOptions) (OpenOrders, error) {
	params := url.Values{}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	var response struct {
		Error  []string   `json:"error"`
		Result OpenOrders `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenOpenOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetClosedOrders returns a list of closed orders
func (k *Kraken) GetClosedOrders(args GetClosedOrdersOptions) (ClosedOrders, error) {
	params := url.Values{}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	if len(args.Start) > 0 {
		params.Set("start", args.Start)
	}

	if len(args.End) > 0 {
		params.Set("end", args.End)
	}

	if args.Ofs > 0 {
		params.Set("ofs", strconv.FormatInt(args.Ofs, 10))
	}

	if len(args.CloseTime) > 0 {
		params.Set("closetime", args.CloseTime)
	}

	var response struct {
		Error  []string     `json:"error"`
		Result ClosedOrders `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenClosedOrders, params, &response); err != nil {
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

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenQueryOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradesHistory returns trade history information
func (k *Kraken) GetTradesHistory(args ...GetTradesHistoryOptions) (TradesHistory, error) {
	params := url.Values{}

	if args != nil {
		if len(args[0].Type) > 0 {
			params.Set("type", args[0].Type)
		}

		if args[0].Trades {
			params.Set("trades", "true")
		}

		if len(args[0].Start) > 0 {
			params.Set("start", args[0].Start)
		}

		if len(args[0].End) > 0 {
			params.Set("end", args[0].End)
		}

		if args[0].Ofs > 0 {
			params.Set("ofs", strconv.FormatInt(args[0].Ofs, 10))
		}
	}

	var response struct {
		Error  []string      `json:"error"`
		Result TradesHistory `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenTradeHistory, params, &response); err != nil {
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

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenQueryTrades, params, &response); err != nil {
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

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenOpenPositions, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetLedgers returns current ledgers
func (k *Kraken) GetLedgers(args ...GetLedgersOptions) (Ledgers, error) {
	params := url.Values{}

	if args != nil {
		if args[0].Aclass == "" {
			params.Set("aclass", args[0].Aclass)
		}

		if args[0].Asset == "" {
			params.Set("asset", args[0].Asset)
		}

		if args[0].Type == "" {
			params.Set("type", args[0].Type)
		}

		if args[0].Start == "" {
			params.Set("start", args[0].Start)
		}

		if args[0].End == "" {
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

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenLedgers, params, &response); err != nil {
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

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenQueryLedgers, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradeVolume returns your trade volume by currency
func (k *Kraken) GetTradeVolume(feeinfo bool, symbol ...currency.Pair) (TradeVolumeResponse, error) {
	var response struct {
		Error  []string            `json:"error"`
		Result TradeVolumeResponse `json:"result"`
	}
	params := url.Values{}
	var formattedPairs []string
	for x := range symbol {
		symbolValue, err := k.FormatSymbol(symbol[x], asset.Spot)
		if err != nil {
			return response.Result, err
		}
		formattedPairs = append(formattedPairs, symbolValue)
	}
	if symbol != nil {
		params.Set("pair", strings.Join(formattedPairs, ","))
	}

	if feeinfo {
		params.Set("fee-info", "true")
	}

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenTradeVolume, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// AddOrder adds a new order for Kraken exchange
func (k *Kraken) AddOrder(symbol currency.Pair, side, orderType string, volume, price, price2, leverage float64, args *AddOrderOptions) (AddOrderResponse, error) {
	var response struct {
		Error  []string         `json:"error"`
		Result AddOrderResponse `json:"result"`
	}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return response.Result, err
	}
	params := url.Values{
		"pair":      {symbolValue},
		"type":      {strings.ToLower(side)},
		"ordertype": {strings.ToLower(orderType)},
		"volume":    {strconv.FormatFloat(volume, 'f', -1, 64)},
	}

	if orderType == order.Limit.Lower() || price > 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}

	if price2 != 0 {
		params.Set("price2", strconv.FormatFloat(price2, 'f', -1, 64))
	}

	if leverage != 0 {
		params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	}

	if args.OrderFlags != "" {
		params.Set("oflags", args.OrderFlags)
	}

	if args.StartTm != "" {
		params.Set("starttm", args.StartTm)
	}

	if args.ExpireTm != "" {
		params.Set("expiretm", args.ExpireTm)
	}

	if args.CloseOrderType != "" {
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

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenOrderPlace, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// CancelExistingOrder cancels order by orderID
func (k *Kraken) CancelExistingOrder(txid string) (CancelOrderResponse, error) {
	values := url.Values{
		"txid": {txid},
	}

	var response struct {
		Error  []string            `json:"error"`
		Result CancelOrderResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenOrderCancel, values, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetError parse Exchange errors in response and return the first one
// Error format from API doc:
//   error = array of error messages in the format of:
//       <char-severity code><string-error category>:<string-error type>[:<string-extra info>]
//       severity code can be E for error or W for warning
func GetError(apiErrors []string) error {
	const exchangeName = "Kraken"
	for _, e := range apiErrors {
		switch e[0] {
		case 'W':
			log.Warnf(log.ExchangeSys, "%s API warning: %v\n", exchangeName, e[1:])
		default:
			return fmt.Errorf("%s API error: %v", exchangeName, e[1:])
		}
	}

	return nil
}

// SendHTTPRequest sends an unauthenticated HTTP requests
func (k *Kraken) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := k.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return k.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       k.Verbose,
		HTTPDebugging: k.HTTPDebugging,
		HTTPRecording: k.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (k *Kraken) SendAuthenticatedHTTPRequest(ep exchange.URL, method string, params url.Values, result interface{}) error {
	if !k.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", k.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := k.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/%s/private/%s", krakenAPIVersion, method)

	params.Set("nonce", k.Requester.GetNonce(true).String())
	encoded := params.Encode()
	shasum := crypto.GetSHA256([]byte(params.Get("nonce") + encoded))
	signature := crypto.Base64Encode(crypto.GetHMAC(crypto.HashSHA512,
		append([]byte(path), shasum...), []byte(k.API.Credentials.Secret)))

	if k.Verbose {
		log.Debugf(log.ExchangeSys, "Sending POST request to %s, path: %s, params: %s",
			endpoint,
			path,
			encoded)
	}

	headers := make(map[string]string)
	headers["API-Key"] = k.API.Credentials.Key
	headers["API-Sign"] = signature

	interim := json.RawMessage{}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	defer cancel()
	err = k.SendPayload(ctx, &request.Item{
		Method:        http.MethodPost,
		Path:          endpoint + path,
		Headers:       headers,
		Body:          strings.NewReader(encoded),
		Result:        &interim,
		AuthRequest:   true,
		NonceEnabled:  true,
		Verbose:       k.Verbose,
		HTTPDebugging: k.HTTPDebugging,
		HTTPRecording: k.HTTPRecording,
	})
	if err != nil {
		return err
	}
	var errCap SpotAuthError
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if len(errCap.Error) != 0 {
			return errors.New(errCap.Error[0])
		}
	}
	return json.Unmarshal(interim, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (k *Kraken) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePair, err := k.GetTradeVolume(true, feeBuilder.Pair)
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			fee = calculateTradingFee(feePair.Currency,
				feePair.FeesMaker,
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		} else {
			fee = calculateTradingFee(feePair.Currency,
				feePair.Fees,
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		}
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankDepositFee:
		depositMethods, err := k.GetDepositMethods(feeBuilder.FiatCurrency.String())
		if err != nil {
			return 0, err
		}

		for _, i := range depositMethods {
			if feeBuilder.BankTransactionType == exchange.WireTransfer {
				if i.Method == "SynapsePay (US Wire)" {
					fee = i.Fee
					return fee, nil
				}
			}
		}
	case exchange.CryptocurrencyDepositFee:
		fee = getCryptocurrencyDepositFee(feeBuilder.Pair.Base)

	case exchange.InternationalBankWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.FiatCurrency)
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
	return 0.0016 * price * amount
}

func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

func getCryptocurrencyDepositFee(c currency.Code) float64 {
	return DepositFees[c]
}

func calculateTradingFee(currency string, feePair map[string]TradeVolumeFee, purchasePrice, amount float64) float64 {
	return (feePair[currency].Fee / 100) * purchasePrice * amount
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
func (k *Kraken) GetCryptoDepositAddress(method, code string) (string, error) {
	var resp = struct {
		Error  []string         `json:"error"`
		Result []DepositAddress `json:"result"`
	}{}

	values := url.Values{}
	values.Set("asset", code)
	values.Set("method", method)

	err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenDepositAddresses, values, &resp)
	if err != nil {
		return "", err
	}

	for _, a := range resp.Result {
		return a.Address, nil
	}

	return "", errors.New("no addresses returned")
}

// WithdrawStatus gets the status of recent withdrawals
func (k *Kraken) WithdrawStatus(c currency.Code, method string) ([]WithdrawStatusResponse, error) {
	var response struct {
		Error  []string                 `json:"error"`
		Result []WithdrawStatusResponse `json:"result"`
	}

	params := url.Values{}
	params.Set("asset", c.String())
	if method != "" {
		params.Set("method", method)
	}

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenWithdrawStatus, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// WithdrawCancel sends a withdrawal cancelation request
func (k *Kraken) WithdrawCancel(c currency.Code, refID string) (bool, error) {
	var response struct {
		Error  []string `json:"error"`
		Result bool     `json:"result"`
	}

	params := url.Values{}
	params.Set("asset", c.String())
	params.Set("refid", refID)

	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenWithdrawCancel, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetWebsocketToken returns a websocket token
func (k *Kraken) GetWebsocketToken() (string, error) {
	var response WsTokenResponse
	if err := k.SendAuthenticatedHTTPRequest(exchange.RestSpot, krakenWebsocketToken, url.Values{}, &response); err != nil {
		return "", err
	}
	if len(response.Error) > 0 {
		return "", fmt.Errorf("%s - %v", k.Name, response.Error)
	}
	return response.Result.Token, nil
}

// LookupAltname converts a currency into its altname (ZUSD -> USD)
func (a *assetTranslatorStore) LookupAltname(target string) string {
	a.l.RLock()
	alt, ok := a.Assets[target]
	if !ok {
		a.l.RUnlock()
		return ""
	}
	a.l.RUnlock()
	return alt
}

// LookupAltname converts an altname to its original type (USD -> ZUSD)
func (a *assetTranslatorStore) LookupCurrency(target string) string {
	a.l.RLock()
	for k, v := range a.Assets {
		if v == target {
			a.l.RUnlock()
			return k
		}
	}
	a.l.RUnlock()
	return ""
}

// Seed seeds a currency translation pair
func (a *assetTranslatorStore) Seed(orig, alt string) {
	a.l.Lock()
	if a.Assets == nil {
		a.Assets = make(map[string]string)
	}

	_, ok := a.Assets[orig]
	if ok {
		a.l.Unlock()
		return
	}

	a.Assets[orig] = alt
	a.l.Unlock()
}

// Seeded returns whether or not the asset translator has been seeded
func (a *assetTranslatorStore) Seeded() bool {
	a.l.RLock()
	isSeeded := len(a.Assets) > 0
	a.l.RUnlock()
	return isSeeded
}
