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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	krakenAPIURL                  = "https://api.kraken.com"
	krakenFuturesURL              = "https://futures.kraken.com/derivatives"
	krakenFuturesSupplementaryURL = "https://futures.kraken.com/api/"
	tradeBaseURL                  = "https://pro.kraken.com/app/trade/"
	tradeFuturesURL               = "https://futures.kraken.com/trade/futures/"
	krakenSpotVersion             = "0"
	krakenFuturesVersion          = "3"
)

// Kraken is the overarching type across the kraken package
type Kraken struct {
	exchange.Base
	wsRequestMtx sync.Mutex
}

// GetCurrentServerTime returns current server time
func (k *Kraken) GetCurrentServerTime(ctx context.Context) (TimeResponse, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenServerTime)

	var response struct {
		Error  []string     `json:"error"`
		Result TimeResponse `json:"result"`
	}

	if err := k.SendHTTPRequest(ctx, exchange.RestSpot, path, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// SeedAssets seeds Kraken's asset list and stores it in the
// asset translator
func (k *Kraken) SeedAssets(ctx context.Context) error {
	assets, err := k.GetAssets(ctx)
	if err != nil {
		return err
	}
	for orig, val := range assets {
		assetTranslator.Seed(orig, val.Altname)
	}

	assetPairs, err := k.GetAssetPairs(ctx, []string{}, "")
	if err != nil {
		return err
	}
	for k := range assetPairs {
		assetTranslator.Seed(k, assetPairs[k].Altname)
	}
	return nil
}

// GetAssets returns a full asset list
func (k *Kraken) GetAssets(ctx context.Context) (map[string]*Asset, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenAssets)

	var response struct {
		Error  []string          `json:"error"`
		Result map[string]*Asset `json:"result"`
	}

	if err := k.SendHTTPRequest(ctx, exchange.RestSpot, path, &response); err != nil {
		return response.Result, err
	}
	return response.Result, GetError(response.Error)
}

// GetAssetPairs returns a full asset pair list
// Parameter 'info' only supports 4 strings: "fees", "leverage", "margin", "info" <- (default)
func (k *Kraken) GetAssetPairs(ctx context.Context, assetPairs []string, info string) (map[string]*AssetPairs, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenAssetPairs)
	params := url.Values{}
	var assets string
	if len(assetPairs) != 0 {
		assets = strings.Join(assetPairs, ",")
		params.Set("pair", assets)
	}
	var response struct {
		Error  []string               `json:"error"`
		Result map[string]*AssetPairs `json:"result"`
	}
	if info != "" {
		if info != "margin" && info != "leverage" && info != "fees" && info != "info" {
			return response.Result, errors.New("parameter info can only be 'asset', 'margin', 'fees' or 'leverage'")
		}
		params.Set("info", info)
	}
	if err := k.SendHTTPRequest(ctx, exchange.RestSpot, path+params.Encode(), &response); err != nil {
		return response.Result, err
	}
	return response.Result, GetError(response.Error)
}

// GetTicker returns ticker information from kraken
func (k *Kraken) GetTicker(ctx context.Context, symbol currency.Pair) (Ticker, error) {
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

	err = k.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	if err != nil {
		return tick, err
	}

	if len(resp.Error) > 0 {
		return tick, fmt.Errorf("%s error: %s", k.Name, resp.Error)
	}
	for i := range resp.Data {
		tick.Ask = resp.Data[i].Ask[0].Float64()
		tick.AskSize = resp.Data[i].Ask[2].Float64()
		tick.Bid = resp.Data[i].Bid[0].Float64()
		tick.BidSize = resp.Data[i].Bid[2].Float64()
		tick.Last = resp.Data[i].Last[0].Float64()
		tick.Volume = resp.Data[i].Volume[1].Float64()
		tick.VolumeWeightedAveragePrice = resp.Data[i].VolumeWeightedAveragePrice[1].Float64()
		tick.Trades = resp.Data[i].Trades[1]
		tick.Low = resp.Data[i].Low[1].Float64()
		tick.High = resp.Data[i].High[1].Float64()
		tick.Open = resp.Data[i].Open.Float64()
	}
	return tick, nil
}

// GetTickers supports fetching multiple tickers from Kraken
// pairList must be in the format pairs separated by commas
// ("LTCUSD,ETCUSD")
func (k *Kraken) GetTickers(ctx context.Context, pairList string) (map[string]Ticker, error) {
	values := url.Values{}
	if pairList != "" {
		values.Set("pair", pairList)
	}

	type Response struct {
		Error []interface{}             `json:"error"`
		Data  map[string]TickerResponse `json:"result"`
	}

	resp := Response{}
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenTicker, values.Encode())

	err := k.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Error) > 0 {
		return nil, fmt.Errorf("%s error: %s", k.Name, resp.Error)
	}

	tickers := make(map[string]Ticker, len(resp.Data))
	for i := range resp.Data {
		tickers[i] = Ticker{
			Ask:                        resp.Data[i].Ask[0].Float64(),
			AskSize:                    resp.Data[i].Ask[2].Float64(),
			Bid:                        resp.Data[i].Bid[0].Float64(),
			BidSize:                    resp.Data[i].Bid[2].Float64(),
			Last:                       resp.Data[i].Last[0].Float64(),
			Volume:                     resp.Data[i].Volume[1].Float64(),
			VolumeWeightedAveragePrice: resp.Data[i].VolumeWeightedAveragePrice[1].Float64(),
			Trades:                     resp.Data[i].Trades[1],
			Low:                        resp.Data[i].Low[1].Float64(),
			High:                       resp.Data[i].High[1].Float64(),
			Open:                       resp.Data[i].Open.Float64(),
		}
	}
	return tickers, nil
}

// GetOHLC returns an array of open high low close values of a currency pair
func (k *Kraken) GetOHLC(ctx context.Context, symbol currency.Pair, interval string) ([]OpenHighLowClose, error) {
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

	var result Response

	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenOHLC, values.Encode())

	err = k.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
	if err != nil {
		return nil, err
	}

	if len(result.Error) != 0 {
		return nil, fmt.Errorf("getOHLC error: %s", result.Error)
	}

	ohlcData, ok := result.Data[translatedAsset].([]interface{})
	if !ok {
		return nil, errors.New("invalid data returned")
	}

	OHLC := make([]OpenHighLowClose, len(ohlcData))
	for x := range ohlcData {
		subData, ok := ohlcData[x].([]interface{})
		if !ok {
			return nil, errors.New("unable to type assert subData")
		}

		if len(subData) < 8 {
			return nil, errors.New("unexpected data length returned")
		}

		var o OpenHighLowClose
		if o.Time, ok = subData[0].(float64); !ok {
			return nil, errors.New("unable to type assert time")
		}
		if o.Open, err = convert.FloatFromString(subData[1]); err != nil {
			return nil, err
		}
		if o.High, err = convert.FloatFromString(subData[2]); err != nil {
			return nil, err
		}
		if o.Low, err = convert.FloatFromString(subData[3]); err != nil {
			return nil, err
		}
		if o.Close, err = convert.FloatFromString(subData[4]); err != nil {
			return nil, err
		}
		if o.VolumeWeightedAveragePrice, err = convert.FloatFromString(subData[5]); err != nil {
			return nil, err
		}
		if o.Volume, err = convert.FloatFromString(subData[6]); err != nil {
			return nil, err
		}
		if o.Count, ok = subData[7].(float64); !ok {
			return nil, errors.New("unable to type assert count")
		}
		OHLC[x] = o
	}
	return OHLC, nil
}

// GetDepth returns the orderbook for a particular currency
func (k *Kraken) GetDepth(ctx context.Context, symbol currency.Pair) (*Orderbook, error) {
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("pair", symbolValue)
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenDepth, values.Encode())
	var result interface{}
	err = k.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("%s GetDepth result is nil", k.Name)
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		return nil, errors.New("unable to type assert data")
	}
	orderbookData, ok := data["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s GetDepth data[result] is nil", k.Name)
	}
	var bidsData []interface{}
	var asksData []interface{}
	for _, y := range orderbookData {
		yData, ok := y.(map[string]interface{})
		if !ok {
			return nil, errors.New("unable to type assert yData")
		}
		if bidsData, ok = yData["bids"].([]interface{}); !ok {
			return nil, errors.New("unable to type assert bidsData")
		}
		if asksData, ok = yData["asks"].([]interface{}); !ok {
			return nil, errors.New("unable to type assert asksData")
		}
	}

	processOrderbook := func(data []interface{}) ([]OrderbookBase, error) {
		result := make([]OrderbookBase, len(data))
		for x := range data {
			entry, ok := data[x].([]interface{})
			if !ok {
				return nil, errors.New("unable to type assert entry")
			}

			if len(entry) < 2 {
				return nil, errors.New("unexpected entry length")
			}

			price, priceErr := strconv.ParseFloat(entry[0].(string), 64)
			if priceErr != nil {
				return nil, priceErr
			}

			amount, amountErr := strconv.ParseFloat(entry[1].(string), 64)
			if amountErr != nil {
				return nil, amountErr
			}

			result[x] = OrderbookBase{Price: price, Amount: amount}
		}
		return result, nil
	}

	var orderBook Orderbook
	orderBook.Bids, err = processOrderbook(bidsData)
	if err != nil {
		return nil, err
	}

	orderBook.Asks, err = processOrderbook(asksData)
	return &orderBook, err
}

// GetTrades returns current trades on Kraken
func (k *Kraken) GetTrades(ctx context.Context, symbol currency.Pair) ([]RecentTrades, error) {
	values := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	translatedAsset := assetTranslator.LookupCurrency(symbolValue)
	values.Set("pair", translatedAsset)

	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenTrades, values.Encode())

	var data RecentTradesResponse
	err = k.SendHTTPRequest(ctx, exchange.RestSpot, path, &data)
	if err != nil {
		return nil, err
	}

	if len(data.Error) > 0 {
		var errs error
		for x := range data.Error {
			errString, ok := data.Error[x].(string)
			if !ok {
				continue
			}
			errs = common.AppendError(errs, errors.New(errString))
		}
		if errs != nil {
			return nil, errs
		}
	}

	trades, ok := data.Result[translatedAsset].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no data returned for symbol %v", symbol)
	}

	var individualTrade []interface{}
	recentTrades := make([]RecentTrades, len(trades))
	for x := range trades {
		individualTrade, ok = trades[x].([]interface{})
		if !ok {
			return nil, errors.New("unable to parse individual trade data")
		}
		if len(individualTrade) != 7 {
			return nil, errors.New("unrecognised trade data received")
		}
		var r RecentTrades
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
		tradeID, ok := individualTrade[6].(float64)
		if !ok {
			return nil, errors.New("unable to parse TradeID field for individual trade data")
		}
		r.TradeID = int64(tradeID)
		recentTrades[x] = r
	}
	return recentTrades, nil
}

// GetSpread returns the full spread on Kraken
func (k *Kraken) GetSpread(ctx context.Context, symbol currency.Pair) ([]Spread, error) {
	values := url.Values{}
	symbolValue, err := k.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	values.Set("pair", symbolValue)

	resp := struct {
		SpreadData map[string]interface{} `json:"result"`
	}{}
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenSpread, values.Encode())
	err = k.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	if err != nil {
		return nil, err
	}

	data, ok := resp.SpreadData[symbolValue]
	if !ok {
		return nil, fmt.Errorf("unable to find %s in spread data", symbolValue)
	}

	spreadData, ok := data.([]interface{})
	if !ok {
		return nil, errors.New("unable to type assert spreadData")
	}

	peanutButter := make([]Spread, len(spreadData))
	for x := range spreadData {
		subData, ok := spreadData[x].([]interface{})
		if !ok {
			return nil, errors.New("unable to type assert subData")
		}

		if len(subData) < 3 {
			return nil, errors.New("unexpected data length")
		}

		var s Spread
		timeData, ok := subData[0].(float64)
		if !ok {
			return nil, common.GetTypeAssertError("float64", subData[0], "timeData")
		}
		s.Time = time.Unix(int64(timeData), 0)

		if s.Bid, err = convert.FloatFromString(subData[1]); err != nil {
			return nil, err
		}
		if s.Ask, err = convert.FloatFromString(subData[2]); err != nil {
			return nil, err
		}
		peanutButter[x] = s
	}
	return peanutButter, nil
}

// GetBalance returns your balance associated with your keys
func (k *Kraken) GetBalance(ctx context.Context) (map[string]Balance, error) {
	var response struct {
		Error  []string           `json:"error"`
		Result map[string]Balance `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenBalance, url.Values{}, &response); err != nil {
		return nil, err
	}

	return response.Result, GetError(response.Error)
}

// GetWithdrawInfo gets withdrawal fees
func (k *Kraken) GetWithdrawInfo(ctx context.Context, currency string, amount float64) (WithdrawInformation, error) {
	var response struct {
		Error  []string            `json:"error"`
		Result WithdrawInformation `json:"result"`
	}
	params := url.Values{}
	params.Set("asset", currency)
	params.Set("key", "")
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdrawInfo, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// Withdraw withdraws funds
func (k *Kraken) Withdraw(ctx context.Context, asset, key string, amount float64) (string, error) {
	var response struct {
		Error       []string `json:"error"`
		ReferenceID string   `json:"refid"`
	}
	params := url.Values{}
	params.Set("asset", asset)
	params.Set("key", key)
	params.Set("amount", fmt.Sprintf("%f", amount))

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdraw, params, &response); err != nil {
		return response.ReferenceID, err
	}

	return response.ReferenceID, GetError(response.Error)
}

// GetDepositMethods gets withdrawal fees
func (k *Kraken) GetDepositMethods(ctx context.Context, currency string) ([]DepositMethods, error) {
	var response struct {
		Error  []string         `json:"error"`
		Result []DepositMethods `json:"result"`
	}
	params := url.Values{}
	params.Set("asset", currency)

	err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenDepositMethods, params, &response)
	if err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradeBalance returns full information about your trades on Kraken
func (k *Kraken) GetTradeBalance(ctx context.Context, args ...TradeBalanceOptions) (TradeBalanceInfo, error) {
	params := url.Values{}

	if args != nil {
		if args[0].Aclass != "" {
			params.Set("aclass", args[0].Aclass)
		}

		if args[0].Asset != "" {
			params.Set("asset", args[0].Asset)
		}
	}

	var response struct {
		Error  []string         `json:"error"`
		Result TradeBalanceInfo `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenTradeBalance, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetOpenOrders returns all current open orders
func (k *Kraken) GetOpenOrders(ctx context.Context, args OrderInfoOptions) (OpenOrders, error) {
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

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOpenOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetClosedOrders returns a list of closed orders
func (k *Kraken) GetClosedOrders(ctx context.Context, args GetClosedOrdersOptions) (ClosedOrders, error) {
	params := url.Values{}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	if args.Start != "" {
		params.Set("start", args.Start)
	}

	if args.End != "" {
		params.Set("end", args.End)
	}

	if args.Ofs > 0 {
		params.Set("ofs", strconv.FormatInt(args.Ofs, 10))
	}

	if args.CloseTime != "" {
		params.Set("closetime", args.CloseTime)
	}

	var response struct {
		Error  []string     `json:"error"`
		Result ClosedOrders `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenClosedOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryOrdersInfo returns order information
func (k *Kraken) QueryOrdersInfo(ctx context.Context, args OrderInfoOptions, txid string, txids ...string) (map[string]OrderInfo, error) {
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

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenQueryOrders, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradesHistory returns trade history information
func (k *Kraken) GetTradesHistory(ctx context.Context, args ...GetTradesHistoryOptions) (TradesHistory, error) {
	params := url.Values{}

	if args != nil {
		if args[0].Type != "" {
			params.Set("type", args[0].Type)
		}

		if args[0].Trades {
			params.Set("trades", "true")
		}

		if args[0].Start != "" {
			params.Set("start", args[0].Start)
		}

		if args[0].End != "" {
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

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenTradeHistory, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryTrades returns information on a specific trade
func (k *Kraken) QueryTrades(ctx context.Context, trades bool, txid string, txids ...string) (map[string]TradeInfo, error) {
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

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenQueryTrades, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// OpenPositions returns current open positions
func (k *Kraken) OpenPositions(ctx context.Context, docalcs bool, txids ...string) (map[string]Position, error) {
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

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOpenPositions, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetLedgers returns current ledgers
func (k *Kraken) GetLedgers(ctx context.Context, args ...GetLedgersOptions) (Ledgers, error) {
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

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenLedgers, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// QueryLedgers queries an individual ledger by ID
func (k *Kraken) QueryLedgers(ctx context.Context, id string, ids ...string) (map[string]LedgerInfo, error) {
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

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenQueryLedgers, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetTradeVolume returns your trade volume by currency
func (k *Kraken) GetTradeVolume(ctx context.Context, feeinfo bool, symbol ...currency.Pair) (*TradeVolumeResponse, error) {
	var response struct {
		Error  []string             `json:"error"`
		Result *TradeVolumeResponse `json:"result"`
	}
	params := url.Values{}
	formattedPairs := make([]string, len(symbol))
	for x := range symbol {
		symbolValue, err := k.FormatSymbol(symbol[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		formattedPairs[x] = symbolValue
	}
	if symbol != nil {
		params.Set("pair", strings.Join(formattedPairs, ","))
	}

	if feeinfo {
		params.Set("fee-info", "true")
	}

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenTradeVolume, params, &response); err != nil {
		return nil, err
	}

	return response.Result, GetError(response.Error)
}

// AddOrder adds a new order for Kraken exchange
func (k *Kraken) AddOrder(ctx context.Context, symbol currency.Pair, side, orderType string, volume, price, price2, leverage float64, args *AddOrderOptions) (AddOrderResponse, error) {
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

	if args.TimeInForce != "" {
		params.Set("timeinforce", string(args.TimeInForce))
	}

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOrderPlace, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// CancelExistingOrder cancels order by orderID
func (k *Kraken) CancelExistingOrder(ctx context.Context, txid string) (CancelOrderResponse, error) {
	values := url.Values{
		"txid": {txid},
	}

	var response struct {
		Error  []string            `json:"error"`
		Result CancelOrderResponse `json:"result"`
	}

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOrderCancel, values, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetError parse Exchange errors in response and return the first one.
//
// Error format from API doc:
//   - error = array of error messages in the format of:
//     <char-severity code><string-error category>:<string-error type>[:<string-extra info>]
//     severity code can be E for error or W for warning
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
func (k *Kraken) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result interface{}) error {
	endpoint, err := k.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       k.Verbose,
		HTTPDebugging: k.HTTPDebugging,
		HTTPRecording: k.HTTPRecording,
	}

	return k.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (k *Kraken) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method string, params url.Values, result interface{}) error {
	creds, err := k.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := k.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/%s/private/%s", krakenAPIVersion, method)

	interim := json.RawMessage{}
	err = k.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		nonce := k.Requester.GetNonce(nonce.UnixNano).String()
		params.Set("nonce", nonce)
		encoded := params.Encode()
		var shasum []byte
		shasum, err = crypto.GetSHA256([]byte(nonce + encoded))
		if err != nil {
			return nil, err
		}

		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA512,
			append([]byte(path), shasum...),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		signature := crypto.Base64Encode(hmac)

		headers := make(map[string]string)
		headers["API-Key"] = creds.Key
		headers["API-Sign"] = signature

		return &request.Item{
			Method:        http.MethodPost,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          strings.NewReader(encoded),
			Result:        &interim,
			NonceEnabled:  true,
			Verbose:       k.Verbose,
			HTTPDebugging: k.HTTPDebugging,
			HTTPRecording: k.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	var errCap SpotAuthError
	if err = json.Unmarshal(interim, &errCap); err == nil && errCap.Error != nil {
		switch e := errCap.Error.(type) {
		case []string:
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e[0])
		case []interface{}: // no error will be an empty []interface{}
			if len(e) > 0 {
				return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e)
			}
		default:
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e)
		}
	}
	err = json.Unmarshal(interim, result)
	if err != nil {
		return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, err)
	}
	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (k *Kraken) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePair, err := k.GetTradeVolume(ctx, true, feeBuilder.Pair)
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
		depositMethods, err := k.GetDepositMethods(ctx,
			feeBuilder.FiatCurrency.String())
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
func (k *Kraken) GetCryptoDepositAddress(ctx context.Context, method, code string, createNew bool) ([]DepositAddress, error) {
	var resp = struct {
		Error  []string         `json:"error"`
		Result []DepositAddress `json:"result"`
	}{}

	values := url.Values{}
	values.Set("asset", code)
	values.Set("method", method)

	if createNew {
		values.Set("new", "1")
	}

	err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenDepositAddresses, values, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, errors.New("no addresses returned")
	}
	return resp.Result, nil
}

// WithdrawStatus gets the status of recent withdrawals
func (k *Kraken) WithdrawStatus(ctx context.Context, c currency.Code, method string) ([]WithdrawStatusResponse, error) {
	var response struct {
		Error  []string                 `json:"error"`
		Result []WithdrawStatusResponse `json:"result"`
	}

	params := url.Values{}
	params.Set("asset", c.String())
	if method != "" {
		params.Set("method", method)
	}

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdrawStatus, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// WithdrawCancel sends a withdrawal cancellation request
func (k *Kraken) WithdrawCancel(ctx context.Context, c currency.Code, refID string) (bool, error) {
	var response struct {
		Error  []string `json:"error"`
		Result bool     `json:"result"`
	}

	params := url.Values{}
	params.Set("asset", c.String())
	params.Set("refid", refID)

	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdrawCancel, params, &response); err != nil {
		return response.Result, err
	}

	return response.Result, GetError(response.Error)
}

// GetWebsocketToken returns a websocket token
func (k *Kraken) GetWebsocketToken(ctx context.Context) (string, error) {
	var response WsTokenResponse
	if err := k.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWebsocketToken, url.Values{}, &response); err != nil {
		return "", err
	}
	if len(response.Error) > 0 {
		return "", fmt.Errorf("%s - %v", k.Name, response.Error)
	}
	return response.Result.Token, nil
}

// LookupAltName converts a currency into its altName (ZUSD -> USD)
func (a *assetTranslatorStore) LookupAltName(target string) string {
	a.l.RLock()
	alt, ok := a.Assets[target]
	if !ok {
		a.l.RUnlock()
		return ""
	}
	a.l.RUnlock()
	return alt
}

// LookupCurrency converts an altName to its original type (USD -> ZUSD)
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

	if _, ok := a.Assets[orig]; ok {
		a.l.Unlock()
		return
	}

	a.Assets[orig] = alt
	a.l.Unlock()
}

// Seeded checks if assets have been seeded
func (a *assetTranslatorStore) Seeded() bool {
	a.l.RLock()
	isSeeded := len(a.Assets) > 0
	a.l.RUnlock()
	return isSeeded
}
