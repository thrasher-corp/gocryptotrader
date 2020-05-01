package binance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	apiURL = "https://api.binance.com"

	// Public endpoints
	exchangeInfo      = "/api/v3/exchangeInfo"
	orderBookDepth    = "/api/v3/depth"
	recentTrades      = "/api/v3/trades"
	historicalTrades  = "/api/v3/historicalTrades"
	aggregatedTrades  = "/api/v3/aggTrades"
	candleStick       = "/api/v3/klines"
	averagePrice      = "/api/v3/avgPrice"
	priceChange       = "/api/v3/ticker/24hr"
	symbolPrice       = "/api/v3/ticker/price"
	bestPrice         = "/api/v3/ticker/bookTicker"
	accountInfo       = "/api/v3/account"
	userAccountStream = "/api/v3/userDataStream"

	// Authenticated endpoints
	newOrderTest = "/api/v3/order/test"
	newOrder     = "/api/v3/order"
	cancelOrder  = "/api/v3/order"
	queryOrder   = "/api/v3/order"
	openOrders   = "/api/v3/openOrders"
	allOrders    = "/api/v3/allOrders"

	// Withdraw API endpoints
	withdrawEndpoint  = "/wapi/v3/withdraw.html"
	depositHistory    = "/wapi/v3/depositHistory.html"
	withdrawalHistory = "/wapi/v3/withdrawHistory.html"
	depositAddress    = "/wapi/v3/depositAddress.html"
	accountStatus     = "/wapi/v3/accountStatus.html"
	systemStatus      = "/wapi/v3/systemStatus.html"
	dustLog           = "/wapi/v3/userAssetDribbletLog.html"
	tradeFee          = "/wapi/v3/tradeFee.html"
	assetDetail       = "/wapi/v3/assetDetail.html"
)

// Binance is the overarching type across the Bithumb package
type Binance struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection

	// Valid string list that is required by the exchange
	validLimits    []int
	validIntervals []TimeInterval
}

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (b *Binance) GetExchangeInfo() (ExchangeInfo, error) {
	var resp ExchangeInfo
	path := b.API.Endpoints.URL + exchangeInfo

	return resp, b.SendHTTPRequest(path, limitDefault, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetOrderBook(obd OrderBookDataRequestParams) (OrderBook, error) {
	var orderbook OrderBook
	if err := b.CheckLimit(obd.Limit); err != nil {
		return orderbook, err
	}

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(obd.Symbol))
	params.Set("limit", fmt.Sprintf("%d", obd.Limit))

	var resp OrderBookData
	path := common.EncodeURLValues(b.API.Endpoints.URL+orderBookDepth, params)
	if err := b.SendHTTPRequest(path, orderbookLimit(obd.Limit), &resp); err != nil {
		return orderbook, err
	}

	for x := range resp.Bids {
		price, err := strconv.ParseFloat(resp.Bids[x][0], 64)
		if err != nil {
			return orderbook, err
		}

		amount, err := strconv.ParseFloat(resp.Bids[x][1], 64)
		if err != nil {
			return orderbook, err
		}

		orderbook.Bids = append(orderbook.Bids, OrderbookItem{
			Price:    price,
			Quantity: amount,
		})
	}

	for x := range resp.Asks {
		price, err := strconv.ParseFloat(resp.Asks[x][0], 64)
		if err != nil {
			return orderbook, err
		}

		amount, err := strconv.ParseFloat(resp.Asks[x][1], 64)
		if err != nil {
			return orderbook, err
		}

		orderbook.Asks = append(orderbook.Asks, OrderbookItem{
			Price:    price,
			Quantity: amount,
		})
	}

	orderbook.LastUpdateID = resp.LastUpdateID
	return orderbook, nil
}

// GetRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (b *Binance) GetRecentTrades(rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	var resp []RecentTrade

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(rtr.Symbol))
	params.Set("limit", fmt.Sprintf("%d", rtr.Limit))

	path := fmt.Sprintf("%s%s?%s", b.API.Endpoints.URL, recentTrades, params.Encode())

	return resp, b.SendHTTPRequest(path, limitDefault, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
// fromID:
func (b *Binance) GetHistoricalTrades(symbol string, limit int, fromID int64) ([]HistoricalTrade, error) {
	// Dropping support due to response for market data is always
	// {"code":-2014,"msg":"API-key format invalid."}
	// TODO: replace with newer API vs REST endpoint
	return nil, common.ErrFunctionNotSupported
}

// GetAggregatedTrades returns aggregated trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
func (b *Binance) GetAggregatedTrades(symbol string, limit int) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade

	if err := b.CheckLimit(limit); err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("%s%s?%s", b.API.Endpoints.URL, aggregatedTrades, params.Encode())

	return resp, b.SendHTTPRequest(path, limitDefault, &resp)
}

// GetSpotKline returns kline data
//
// KlinesRequestParams supports 5 parameters
// symbol: the symbol to get the kline data for
// limit: optinal
// interval: the interval time for the data
// startTime: startTime filter for kline data
// endTime: endTime filter for the kline data
func (b *Binance) GetSpotKline(arg KlinesRequestParams) ([]CandleStick, error) {
	var resp interface{}
	var klineData []CandleStick

	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("interval", string(arg.Interval))
	if arg.Limit != 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if arg.StartTime != 0 {
		params.Set("startTime", strconv.FormatInt(arg.StartTime, 10))
	}
	if arg.EndTime != 0 {
		params.Set("endTime", strconv.FormatInt(arg.EndTime, 10))
	}

	path := fmt.Sprintf("%s%s?%s", b.API.Endpoints.URL, candleStick, params.Encode())

	if err := b.SendHTTPRequest(path, limitDefault, &resp); err != nil {
		return klineData, err
	}

	for _, responseData := range resp.([]interface{}) {
		var candle CandleStick
		for i, individualData := range responseData.([]interface{}) {
			switch i {
			case 0:
				tempTime := individualData.(float64)
				var err error
				candle.OpenTime, err = convert.TimeFromUnixTimestampFloat(tempTime)
				if err != nil {
					return klineData, err
				}
			case 1:
				candle.Open, _ = strconv.ParseFloat(individualData.(string), 64)
			case 2:
				candle.High, _ = strconv.ParseFloat(individualData.(string), 64)
			case 3:
				candle.Low, _ = strconv.ParseFloat(individualData.(string), 64)
			case 4:
				candle.Close, _ = strconv.ParseFloat(individualData.(string), 64)
			case 5:
				candle.Volume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 6:
				tempTime := individualData.(float64)
				var err error
				candle.CloseTime, err = convert.TimeFromUnixTimestampFloat(tempTime)
				if err != nil {
					return klineData, err
				}
			case 7:
				candle.QuoteAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 8:
				candle.TradeCount = individualData.(float64)
			case 9:
				candle.TakerBuyAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 10:
				candle.TakerBuyQuoteAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			}
		}
		klineData = append(klineData, candle)
	}
	return klineData, nil
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binance) GetAveragePrice(symbol string) (AveragePrice, error) {
	resp := AveragePrice{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.API.Endpoints.URL, averagePrice, params.Encode())

	return resp, b.SendHTTPRequest(path, limitDefault, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binance) GetPriceChangeStats(symbol string) (PriceChangeStats, error) {
	resp := PriceChangeStats{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.API.Endpoints.URL, priceChange, params.Encode())

	return resp, b.SendHTTPRequest(path, limitDefault, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binance) GetTickers() ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	path := b.API.Endpoints.URL + priceChange
	return resp, b.SendHTTPRequest(path, limitPriceChangeAll, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Binance) GetLatestSpotPrice(symbol string) (SymbolPrice, error) {
	resp := SymbolPrice{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.API.Endpoints.URL, symbolPrice, params.Encode())

	return resp, b.SendHTTPRequest(path, symbolPriceLimit(symbol), &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binance) GetBestPrice(symbol string) (BestPrice, error) {
	resp := BestPrice{}
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.API.Endpoints.URL, bestPrice, params.Encode())

	return resp, b.SendHTTPRequest(path, bestPriceLimit(symbol), &resp)
}

// NewOrder sends a new order to Binance
func (b *Binance) NewOrder(o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse

	path := b.API.Endpoints.URL + newOrder

	params := url.Values{}
	params.Set("symbol", o.Symbol)
	params.Set("side", o.Side)
	params.Set("type", string(o.TradeType))
	params.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	if o.TradeType == BinanceRequestParamsOrderLimit {
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.TimeInForce != "" {
		params.Set("timeInForce", string(o.TimeInForce))
	}

	if o.NewClientOrderID != "" {
		params.Set("newClientOrderID", o.NewClientOrderID)
	}

	if o.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(o.StopPrice, 'f', -1, 64))
	}

	if o.IcebergQty != 0 {
		params.Set("icebergQty", strconv.FormatFloat(o.IcebergQty, 'f', -1, 64))
	}

	if o.NewOrderRespType != "" {
		params.Set("newOrderRespType", o.NewOrderRespType)
	}

	if err := b.SendAuthHTTPRequest(http.MethodPost, path, params, limitOrder, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// CancelExistingOrder sends a cancel order to Binance
func (b *Binance) CancelExistingOrder(symbol string, orderID int64, origClientOrderID string) (CancelOrderResponse, error) {
	var resp CancelOrderResponse

	path := b.API.Endpoints.URL + cancelOrder

	params := url.Values{}
	params.Set("symbol", symbol)

	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}

	return resp, b.SendAuthHTTPRequest(http.MethodDelete, path, params, limitOrder, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted against the rate limiter
// is significantly higher
func (b *Binance) OpenOrders(symbol string) ([]QueryOrderData, error) {
	var resp []QueryOrderData

	path := b.API.Endpoints.URL + openOrders

	params := url.Values{}

	if symbol != "" {
		params.Set("symbol", strings.ToUpper(symbol))
	}

	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, openOrdersLimit(symbol), &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// AllOrders Get all account orders; active, canceled, or filled.
// orderId optional param
// limit optional param, default 500; max 500
func (b *Binance) AllOrders(symbol, orderID, limit string) ([]QueryOrderData, error) {
	var resp []QueryOrderData

	path := b.API.Endpoints.URL + allOrders

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, limitOrdersAll, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// QueryOrder returns information on a past order
func (b *Binance) QueryOrder(symbol, origClientOrderID string, orderID int64) (QueryOrderData, error) {
	var resp QueryOrderData

	path := b.API.Endpoints.URL + queryOrder

	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, limitOrder, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// GetAccount returns binance user accounts
func (b *Binance) GetAccount() (*Account, error) {
	type response struct {
		Response
		Account
	}

	var resp response

	path := b.API.Endpoints.URL + accountInfo
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, request.Unset, &resp); err != nil {
		return &resp.Account, err
	}

	if resp.Code != 0 {
		return &resp.Account, errors.New(resp.Msg)
	}

	return &resp.Account, nil
}

// SendHTTPRequest sends an unauthenticated request
func (b *Binance) SendHTTPRequest(path string, f request.EndpointLimit, result interface{}) error {
	return b.SendPayload(&request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f})
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binance) SendAuthHTTPRequest(method, path string, params url.Values, f request.EndpointLimit, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	if params == nil {
		params = url.Values{}
	}
	params.Set("recvWindow", strconv.FormatInt(convert.RecvWindow(5*time.Second), 10))
	params.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))

	signature := params.Encode()
	hmacSigned := crypto.GetHMAC(crypto.HashSHA256, []byte(signature), []byte(b.API.Credentials.Secret))
	hmacSignedStr := crypto.HexEncodeToString(hmacSigned)

	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key

	if b.Verbose {
		log.Debugf(log.ExchangeSys, "sent path: %s", path)
	}

	path = common.EncodeURLValues(path, params)
	path += "&signature=" + hmacSignedStr

	interim := json.RawMessage{}

	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
	}{}

	err := b.SendPayload(&request.Item{
		Method:        method,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		Result:        &interim,
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f})
	if err != nil {
		return err
	}

	if err := json.Unmarshal(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" {
			return errors.New(errCap.Message)
		}
	}

	return json.Unmarshal(interim, result)
}

// CheckLimit checks value against a variable list
func (b *Binance) CheckLimit(limit int) error {
	for x := range b.validLimits {
		if b.validLimits[x] == limit {
			return nil
		}
	}
	return errors.New("incorrect limit values - valid values are 5, 10, 20, 50, 100, 500, 1000")
}

// CheckSymbol checks value against a variable list
func (b *Binance) CheckSymbol(symbol string, assetType asset.Item) error {
	enPairs := b.GetAvailablePairs(assetType)
	for x := range enPairs {
		if b.FormatExchangeCurrency(enPairs[x], assetType).String() == symbol {
			return nil
		}
	}
	return errors.New("incorrect symbol values - please check available pairs in configuration")
}

// CheckIntervals checks value against a variable list
func (b *Binance) CheckIntervals(interval string) error {
	for x := range b.validIntervals {
		if TimeInterval(interval) == b.validIntervals[x] {
			return nil
		}
	}
	return errors.New(`incorrect interval values - valid values are "1m","3m","5m","15m","30m","1h","2h","4h","6h","8h","12h","1d","3d","1w","1M"`)
}

// SetValues sets the default valid values
func (b *Binance) SetValues() {
	b.validLimits = []int{5, 10, 20, 50, 100, 500, 1000, 5000}
	b.validIntervals = []TimeInterval{
		TimeIntervalMinute,
		TimeIntervalThreeMinutes,
		TimeIntervalFiveMinutes,
		TimeIntervalFifteenMinutes,
		TimeIntervalThirtyMinutes,
		TimeIntervalHour,
		TimeIntervalTwoHours,
		TimeIntervalFourHours,
		TimeIntervalSixHours,
		TimeIntervalEightHours,
		TimeIntervalTwelveHours,
		TimeIntervalDay,
		TimeIntervalThreeDays,
		TimeIntervalWeek,
		TimeIntervalMonth,
	}
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Binance) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		multiplier, err := b.getMultiplier(feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, multiplier)
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
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
	return 0.002 * price * amount
}

// getMultiplier retrieves account based taker/maker fees
func (b *Binance) getMultiplier(isMaker bool) (float64, error) {
	var multiplier float64
	account, err := b.GetAccount()
	if err != nil {
		return 0, err
	}
	if isMaker {
		multiplier = float64(account.MakerCommission)
	} else {
		multiplier = float64(account.TakerCommission)
	}
	return multiplier, nil
}

// calculateTradingFee returns the fee for trading any currency on Bittrex
func calculateTradingFee(purchasePrice, amount, multiplier float64) float64 {
	return (multiplier / 100) * purchasePrice * amount
}

// getCryptocurrencyWithdrawalFee returns the fee for withdrawing from the exchange
func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

// WithdrawCrypto sends cryptocurrency to the address of your choosing
func (b *Binance) WithdrawCrypto(asset, address, addressTag, name, amount string) (string, error) {
	var resp WithdrawResponse
	path := b.API.Endpoints.URL + withdrawEndpoint

	params := url.Values{}
	params.Set("asset", asset)
	params.Set("address", address)
	params.Set("amount", amount)
	if len(name) > 0 {
		params.Set("name", name)
	}
	if len(addressTag) > 0 {
		params.Set("addressTag", addressTag)
	}

	if err := b.SendAuthHTTPRequest(http.MethodPost, path, params, request.Unset, &resp); err != nil {
		return "", err
	}

	if !resp.Success {
		return resp.ID, errors.New(resp.Msg)
	}

	return resp.ID, nil
}

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (b *Binance) GetDepositAddressForCurrency(currency string) (string, error) {
	path := b.API.Endpoints.URL + depositAddress

	resp := struct {
		Address    string `json:"address"`
		Success    bool   `json:"success"`
		AddressTag string `json:"addressTag"`
	}{}

	params := url.Values{}
	params.Set("asset", currency)
	params.Set("status", "true")

	return resp.Address,
		b.SendAuthHTTPRequest(http.MethodGet, path, params, request.Unset, &resp)
}

// GetWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (b *Binance) GetWsAuthStreamKey() (string, error) {
	var resp UserAccountStream
	path := b.API.Endpoints.URL + userAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	err := b.SendPayload(&request.Item{
		Method:        http.MethodPost,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		Result:        &resp,
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
	if err != nil {
		return "", err
	}
	return resp.ListenKey, nil
}

// MaintainWsAuthStreamKey will keep the key alive
func (b *Binance) MaintainWsAuthStreamKey() error {
	var err error
	if listenKey == "" {
		listenKey, err = b.GetWsAuthStreamKey()
		return err
	}
	path := b.API.Endpoints.URL + userAccountStream
	params := url.Values{}
	params.Set("listenKey", listenKey)
	path = common.EncodeURLValues(path, params)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	return b.SendPayload(&request.Item{
		Method:        http.MethodPut,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

func parseInterval(in time.Duration) (TimeInterval, error) {
	switch in {
	case kline.OneMin:
		return TimeIntervalMinute, nil
	case kline.ThreeMin:
		return TimeIntervalThreeMinutes, nil
	case kline.FiveMin:
		return TimeIntervalFiveMinutes, nil
	case kline.FifteenMin:
		return TimeIntervalFifteenMinutes, nil
	case kline.ThirtyMin:
		return TimeIntervalThirtyMinutes, nil
	case kline.OneHour:
		return TimeIntervalHour, nil
	case kline.TwoHour:
		return TimeIntervalTwoHours, nil
	case kline.FourHour:
		return TimeIntervalFourHours, nil
	case kline.SixHour:
		return TimeIntervalSixHours, nil
	case kline.OneHour * 8:
		return TimeIntervalEightHours, nil
	case kline.TwelveHour:
		return TimeIntervalTwelveHours, nil
	case kline.OneDay:
		return TimeIntervalDay, nil
	case kline.ThreeDay:
		return TimeIntervalThreeDays, nil
	case kline.OneWeek:
		return TimeIntervalWeek, nil
	default:
		return TimeIntervalMinute, errInvalidInterval
	}
}
