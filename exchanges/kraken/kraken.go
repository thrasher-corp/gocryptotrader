package kraken

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
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

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Kraken
type Exchange struct {
	exchange.Base
	wsAuthToken string
	wsAuthMtx   sync.RWMutex
}

// GetCurrentServerTime returns current server time
func (e *Exchange) GetCurrentServerTime(ctx context.Context) (*TimeResponse, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenServerTime)

	var result TimeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SeedAssets seeds Kraken's asset list and stores it in the
// asset translator
func (e *Exchange) SeedAssets(ctx context.Context) error {
	assets, err := e.GetAssets(ctx)
	if err != nil {
		return err
	}
	for orig, val := range assets {
		assetTranslator.Seed(orig, val.Altname)
	}

	assetPairs, err := e.GetAssetPairs(ctx, []string{}, "")
	if err != nil {
		return err
	}
	for k, v := range assetPairs {
		assetTranslator.Seed(k, v.Altname)
	}
	return nil
}

// GetAssets returns a full asset list
func (e *Exchange) GetAssets(ctx context.Context) (map[string]*Asset, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenAssets)
	var result map[string]*Asset
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAssetPairs returns a full asset pair list
// Parameter 'info' only supports 4 strings: "fees", "leverage", "margin", "info" <- (default)
func (e *Exchange) GetAssetPairs(ctx context.Context, assetPairs []string, info string) (map[string]*AssetPairs, error) {
	path := fmt.Sprintf("/%s/public/%s", krakenAPIVersion, krakenAssetPairs)
	params := url.Values{}
	var assets string
	if len(assetPairs) != 0 {
		assets = strings.Join(assetPairs, ",")
		params.Set("pair", assets)
	}

	var result map[string]*AssetPairs
	if info != "" {
		if info != "margin" && info != "leverage" && info != "fees" && info != "info" {
			return nil, errors.New("parameter info can only be 'asset', 'margin', 'fees' or 'leverage'")
		}
		params.Set("info", info)
	}
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path+params.Encode(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetTicker returns ticker information from kraken
func (e *Exchange) GetTicker(ctx context.Context, symbol currency.Pair) (*Ticker, error) {
	values := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	values.Set("pair", symbolValue)

	var data map[string]*TickerResponse
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenTicker, values.Encode())
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &data); err != nil {
		return nil, err
	}

	var tick Ticker
	for _, v := range data {
		tick.Ask = v.Ask[0].Float64()
		tick.AskSize = v.Ask[2].Float64()
		tick.Bid = v.Bid[0].Float64()
		tick.BidSize = v.Bid[2].Float64()
		tick.Last = v.Last[0].Float64()
		tick.Volume = v.Volume[1].Float64()
		tick.VolumeWeightedAveragePrice = v.VolumeWeightedAveragePrice[1].Float64()
		tick.Trades = v.Trades[1]
		tick.Low = v.Low[1].Float64()
		tick.High = v.High[1].Float64()
		tick.Open = v.Open.Float64()
	}
	return &tick, nil
}

// GetTickers supports fetching multiple tickers from Kraken
// pairList must be in the format pairs separated by commas
// ("LTCUSD,ETCUSD")
func (e *Exchange) GetTickers(ctx context.Context, pairList string) (map[string]Ticker, error) {
	values := url.Values{}
	if pairList != "" {
		values.Set("pair", pairList)
	}

	var result map[string]*TickerResponse
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenTicker, values.Encode())

	err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
	if err != nil {
		return nil, err
	}

	tickers := make(map[string]Ticker, len(result))
	for k, v := range result {
		tickers[k] = Ticker{
			Ask:                        v.Ask[0].Float64(),
			AskSize:                    v.Ask[2].Float64(),
			Bid:                        v.Bid[0].Float64(),
			BidSize:                    v.Bid[2].Float64(),
			Last:                       v.Last[0].Float64(),
			Volume:                     v.Volume[1].Float64(),
			VolumeWeightedAveragePrice: v.VolumeWeightedAveragePrice[1].Float64(),
			Trades:                     v.Trades[1],
			Low:                        v.Low[1].Float64(),
			High:                       v.High[1].Float64(),
			Open:                       v.Open.Float64(),
		}
	}
	return tickers, nil
}

// GetOHLC returns an array of open high low close values of a currency pair
func (e *Exchange) GetOHLC(ctx context.Context, symbol currency.Pair, interval string) ([]OpenHighLowClose, error) {
	values := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	translatedAsset := assetTranslator.LookupCurrency(symbolValue)
	if translatedAsset == "" {
		translatedAsset = symbolValue
	}
	values.Set("pair", translatedAsset)
	values.Set("interval", interval)

	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenOHLC, values.Encode())

	result := make(map[string]any)
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, path, &result)
	if err != nil {
		return nil, err
	}

	ohlcData, ok := result[translatedAsset].([]any)
	if !ok {
		return nil, errors.New("invalid data returned")
	}

	OHLC := make([]OpenHighLowClose, len(ohlcData))
	for x := range ohlcData {
		subData, ok := ohlcData[x].([]any)
		if !ok {
			return nil, errors.New("unable to type assert subData")
		}

		if len(subData) < 8 {
			return nil, errors.New("unexpected data length returned")
		}

		var o OpenHighLowClose

		tmData, ok := subData[0].(float64)
		if !ok {
			return nil, errors.New("unable to type assert time")
		}
		o.Time = time.Unix(int64(tmData), 0)
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
func (e *Exchange) GetDepth(ctx context.Context, symbol currency.Pair) (*Orderbook, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("pair", symbolValue)
	path := fmt.Sprintf("/%s/public/%s?%s", krakenAPIVersion, krakenDepth, values.Encode())

	type orderbookStructure struct {
		Bids [][3]types.Number `json:"bids"`
		Asks [][3]types.Number `json:"asks"`
	}

	result := make(map[string]*orderbookStructure)
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &result); err != nil {
		return nil, err
	}

	ob := new(Orderbook)
	for _, v := range result {
		ob.Asks = make([]OrderbookBase, len(v.Asks))
		ob.Bids = make([]OrderbookBase, len(v.Bids))

		for x := range v.Asks {
			ob.Asks[x].Price = v.Asks[x][0]
			ob.Asks[x].Amount = v.Asks[x][1]
			ob.Asks[x].Timestamp = time.Unix(v.Asks[x][2].Int64(), 0)
		}

		for x := range v.Bids {
			ob.Bids[x].Price = v.Bids[x][0]
			ob.Bids[x].Amount = v.Bids[x][1]
			ob.Bids[x].Timestamp = time.Unix(v.Bids[x][2].Int64(), 0)
		}
	}

	return ob, nil
}

// GetTrades returns current trades on Kraken
func (e *Exchange) GetTrades(ctx context.Context, symbol currency.Pair, since time.Time, count uint64) (*RecentTradesResponse, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("pair", assetTranslator.LookupCurrency(symbolValue))
	if !since.IsZero() {
		values.Set("since", strconv.FormatInt(since.Unix(), 10))
	}
	if count > 0 {
		values.Set("count", strconv.FormatUint(count, 10))
	}

	path := common.EncodeURLValues("/"+krakenAPIVersion+"/public/"+krakenTrades, values)
	var resp *RecentTradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetSpread returns the full spread on Kraken
func (e *Exchange) GetSpread(ctx context.Context, symbol currency.Pair, since time.Time) (*SpreadResponse, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("pair", symbolValue)
	if !since.IsZero() {
		values.Set("since", strconv.FormatInt(since.Unix(), 10))
	}
	var peanutButter *SpreadResponse
	path := common.EncodeURLValues("/"+krakenAPIVersion+"/public/"+krakenSpread, values)
	return peanutButter, e.SendHTTPRequest(ctx, exchange.RestSpot, path, &peanutButter)
}

// GetBalance returns your balance associated with your keys
func (e *Exchange) GetBalance(ctx context.Context) (map[string]Balance, error) {
	var result map[string]Balance
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenBalance, url.Values{}, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetWithdrawInfo gets withdrawal fees
func (e *Exchange) GetWithdrawInfo(ctx context.Context, withdrawalAsset, withdrawalKey string, amount float64) (*WithdrawInformation, error) {
	params := url.Values{}
	params.Set("asset", withdrawalAsset)
	params.Set("key", withdrawalKey)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result *WithdrawInformation
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdrawInfo, params, &result)
}

// Withdraw withdraws funds
func (e *Exchange) Withdraw(ctx context.Context, withdrawalAsset, withdrawalKey string, amount float64) (string, error) {
	params := url.Values{}
	params.Set("asset", withdrawalAsset)
	params.Set("key", withdrawalKey)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var referenceID string
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdraw, params, &referenceID); err != nil {
		return referenceID, err
	}

	return referenceID, nil
}

// GetDepositMethods gets withdrawal fees for a specific asset
func (e *Exchange) GetDepositMethods(ctx context.Context, a string) ([]DepositMethods, error) {
	params := url.Values{}
	params.Set("asset", a)

	var result []DepositMethods
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenDepositMethods, params, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetTradeBalance returns full information about your trades on Kraken
func (e *Exchange) GetTradeBalance(ctx context.Context, args ...TradeBalanceOptions) (*TradeBalanceInfo, error) {
	params := url.Values{}

	if args != nil {
		if args[0].Aclass != "" {
			params.Set("aclass", args[0].Aclass)
		}

		if args[0].Asset != "" {
			params.Set("asset", args[0].Asset)
		}
	}

	var result TradeBalanceInfo
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenTradeBalance, params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetOpenOrders returns all current open orders
func (e *Exchange) GetOpenOrders(ctx context.Context, args OrderInfoOptions) (*OpenOrders, error) {
	params := url.Values{}

	if args.Trades {
		params.Set("trades", "true")
	}

	if args.UserRef != 0 {
		params.Set("userref", strconv.FormatInt(int64(args.UserRef), 10))
	}

	var result OpenOrders
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOpenOrders, params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetClosedOrders returns a list of closed orders
func (e *Exchange) GetClosedOrders(ctx context.Context, args GetClosedOrdersOptions) (*ClosedOrders, error) {
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

	var result ClosedOrders
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenClosedOrders, params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QueryOrdersInfo returns order information
func (e *Exchange) QueryOrdersInfo(ctx context.Context, args OrderInfoOptions, txid string, txids ...string) (map[string]OrderInfo, error) {
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

	var result map[string]OrderInfo
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenQueryOrders, params, &result); err != nil {
		return result, err
	}

	return result, nil
}

// GetTradesHistory returns trade history information
func (e *Exchange) GetTradesHistory(ctx context.Context, args ...GetTradesHistoryOptions) (*TradesHistory, error) {
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

	var result TradesHistory
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenTradeHistory, params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QueryTrades returns information on a specific trade
func (e *Exchange) QueryTrades(ctx context.Context, trades bool, txid string, txids ...string) (map[string]TradeInfo, error) {
	params := url.Values{
		"txid": {txid},
	}

	if trades {
		params.Set("trades", "true")
	}

	if txids != nil {
		params.Set("txid", txid+","+strings.Join(txids, ","))
	}

	var result map[string]TradeInfo
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenQueryTrades, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// OpenPositions returns current open positions
func (e *Exchange) OpenPositions(ctx context.Context, docalcs bool, txids ...string) (map[string]Position, error) {
	params := url.Values{}

	if txids != nil {
		params.Set("txid", strings.Join(txids, ","))
	}

	if docalcs {
		params.Set("docalcs", "true")
	}

	var result map[string]Position
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOpenPositions, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetLedgers returns current ledgers
func (e *Exchange) GetLedgers(ctx context.Context, args ...GetLedgersOptions) (*Ledgers, error) {
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

	var result Ledgers
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenLedgers, params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QueryLedgers queries an individual ledger by ID
func (e *Exchange) QueryLedgers(ctx context.Context, id string, ids ...string) (map[string]LedgerInfo, error) {
	params := url.Values{
		"id": {id},
	}

	if ids != nil {
		params.Set("id", id+","+strings.Join(ids, ","))
	}

	var result map[string]LedgerInfo
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenQueryLedgers, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetTradeVolume returns your trade volume by currency
func (e *Exchange) GetTradeVolume(ctx context.Context, feeinfo bool, symbol ...currency.Pair) (*TradeVolumeResponse, error) {
	params := url.Values{}
	formattedPairs := make([]string, len(symbol))
	for x := range symbol {
		symbolValue, err := e.FormatSymbol(symbol[x], asset.Spot)
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

	var result *TradeVolumeResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenTradeVolume, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// AddOrder adds a new order for Kraken exchange
func (e *Exchange) AddOrder(ctx context.Context, symbol currency.Pair, side, orderType string, volume, price, price2, leverage float64, args *AddOrderOptions) (*AddOrderResponse, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
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
		params.Set("timeinforce", args.TimeInForce)
	}

	var result AddOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOrderPlace, params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CancelExistingOrder cancels order by orderID
func (e *Exchange) CancelExistingOrder(ctx context.Context, txid string) (*CancelOrderResponse, error) {
	values := url.Values{
		"txid": {txid},
	}

	var result CancelOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenOrderCancel, values, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SendHTTPRequest sends an unauthenticated HTTP requests
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	var rawMessage json.RawMessage
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 &rawMessage,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}

	isSpot := ep == exchange.RestSpot
	if isSpot {
		genResponse := genericRESTResponse{
			Result: result,
		}

		if err := json.Unmarshal(rawMessage, &genResponse); err != nil {
			return err
		}

		if genResponse.Error.Warnings() != "" {
			log.Warnf(log.ExchangeSys, "%v: REST request warning: %v", e.Name, genResponse.Error.Warnings())
		}

		return genResponse.Error.Errors()
	}

	if err := getFuturesErr(rawMessage); err != nil {
		return err
	}

	return json.Unmarshal(rawMessage, result)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method string, params url.Values, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	interim := json.RawMessage{}
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		nonce := e.Requester.GetNonce(nonce.UnixNano).String()
		params.Set("nonce", nonce)
		encoded := params.Encode()

		shasum := sha256.Sum256([]byte(nonce + encoded))
		path := "/" + krakenAPIVersion + "/private/" + method
		hmac, err := crypto.GetHMAC(crypto.HashSHA512, append([]byte(path), shasum[:]...), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		headers["API-Key"] = creds.Key
		headers["API-Sign"] = base64.StdEncoding.EncodeToString(hmac)

		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + path,
			Headers:                headers,
			Body:                   strings.NewReader(encoded),
			Result:                 &interim,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	genResponse := genericRESTResponse{
		Result: result,
	}

	if err := json.Unmarshal(interim, &genResponse); err != nil {
		return fmt.Errorf("%w %w", request.ErrAuthRequestFailed, err)
	}

	if err := genResponse.Error.Errors(); err != nil {
		return fmt.Errorf("%w %w", request.ErrAuthRequestFailed, err)
	}

	if genResponse.Error.Warnings() != "" {
		log.Warnf(log.ExchangeSys, "%v: AUTH REST request warning: %v", e.Name, genResponse.Error.Warnings())
	}

	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePair, err := e.GetTradeVolume(ctx, true, feeBuilder.Pair)
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
		depositMethods, err := e.GetDepositMethods(ctx,
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

func calculateTradingFee(ccy string, feePair map[string]TradeVolumeFee, purchasePrice, amount float64) float64 {
	return (feePair[ccy].Fee / 100) * purchasePrice * amount
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
func (e *Exchange) GetCryptoDepositAddress(ctx context.Context, method, code string, createNew bool) ([]DepositAddress, error) {
	values := url.Values{}
	values.Set("asset", code)
	values.Set("method", method)

	if createNew {
		values.Set("new", "true")
	}

	var result []DepositAddress
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenDepositAddresses, values, &result)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("no addresses returned")
	}
	return result, nil
}

// WithdrawStatus gets the status of recent withdrawals
func (e *Exchange) WithdrawStatus(ctx context.Context, c currency.Code, method string) ([]WithdrawStatusResponse, error) {
	params := url.Values{}
	params.Set("asset", c.String())
	if method != "" {
		params.Set("method", method)
	}

	var result []WithdrawStatusResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdrawStatus, params, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// WithdrawCancel sends a withdrawal cancellation request
func (e *Exchange) WithdrawCancel(ctx context.Context, c currency.Code, refID string) (bool, error) {
	params := url.Values{}
	params.Set("asset", c.String())
	params.Set("refid", refID)

	var result bool
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWithdrawCancel, params, &result); err != nil {
		return result, err
	}

	return result, nil
}

// GetWebsocketToken returns a websocket token
func (e *Exchange) GetWebsocketToken(ctx context.Context) (string, error) {
	var response WsTokenResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, krakenWebsocketToken, url.Values{}, &response); err != nil {
		return "", err
	}
	return response.Token, nil
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
