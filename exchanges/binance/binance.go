package binance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Binance is the overarching type across the Bithumb package
type Binance struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection

	// Valid string list that is required by the exchange
	validLimits    []int
	validIntervals []TimeInterval
}

const (
	apiURL = "https://api.binance.com"

	// Public endpoints
	exchangeInfo     = "/api/v1/exchangeInfo"
	orderBookDepth   = "/api/v1/depth"
	recentTrades     = "/api/v1/trades"
	historicalTrades = "/api/v1/historicalTrades"
	aggregatedTrades = "/api/v1/aggTrades"
	candleStick      = "/api/v1/klines"
	averagePrice     = "/api/v3/avgPrice"
	priceChange      = "/api/v1/ticker/24hr"
	symbolPrice      = "/api/v3/ticker/price"
	bestPrice        = "/api/v3/ticker/bookTicker"
	accountInfo      = "/api/v3/account"

	// Authenticated endpoints
	newOrderTest = "/api/v3/order/test"
	newOrder     = "/api/v3/order"
	cancelOrder  = "/api/v3/order"
	queryOrder   = "/api/v3/order"
	openOrders   = "/api/v3/openOrders"
	allOrders    = "/api/v3/allOrders"

	// Withdraw API endpoints
	withdraw          = "/wapi/v3/withdraw.html"
	depositHistory    = "/wapi/v3/depositHistory.html"
	withdrawalHistory = "/wapi/v3/withdrawHistory.html"
	depositAddress    = "/wapi/v3/depositAddress.html"
	accountStatus     = "/wapi/v3/accountStatus.html"
	systemStatus      = "/wapi/v3/systemStatus.html"
	dustLog           = "/wapi/v3/userAssetDribbletLog.html"
	tradeFee          = "/wapi/v3/tradeFee.html"
	assetDetail       = "/wapi/v3/assetDetail.html"

	// binance authenticated and unauthenticated limit rates
	// to-do
	binanceAuthRate   = 0
	binanceUnauthRate = 0
)

// SetDefaults sets the basic defaults for Binance
func (b *Binance) SetDefaults() {
	b.Name = "Binance"
	b.Enabled = false
	b.Verbose = false
	b.RESTPollingDelay = 10
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = "-"
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
	b.SupportsAutoPairUpdating = true
	b.SupportsRESTTickerBatching = true
	b.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	b.SetValues()
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, binanceAuthRate),
		request.NewRateLimit(time.Second, binanceUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.APIUrlDefault = apiURL
	b.APIUrl = b.APIUrlDefault
	b.Websocket = wshandler.New()
	b.WebsocketURL = binanceDefaultWebsocketURL
	b.Websocket.Functionality = wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketTickerSupported |
		wshandler.WebsocketKlineSupported |
		wshandler.WebsocketOrderbookSupported
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Binance) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.SetHTTPClientTimeout(exch.HTTPTimeout)
		b.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.HTTPDebugging = exch.HTTPDebugging
		b.Websocket.SetWsStatusAndConnection(exch.Websocket)
		b.BaseCurrencies = exch.BaseCurrencies
		b.AvailablePairs = exch.AvailablePairs
		b.EnabledPairs = exch.EnabledPairs
		err := b.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = b.Websocket.Setup(b.WSConnect,
			nil,
			nil,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			binanceDefaultWebsocketURL,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		b.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         b.Name,
			URL:                  b.Websocket.GetWebsocketURL(),
			ProxyURL:             b.Websocket.GetProxyAddress(),
			Verbose:              b.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
		b.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			true,
			false,
			exch.Name)
	}
}

// GetExchangeValidCurrencyPairs returns the full pair list from the exchange
// at the moment do not integrate with config currency pairs automatically
func (b *Binance) GetExchangeValidCurrencyPairs() ([]string, error) {
	var validCurrencyPairs []string

	info, err := b.GetExchangeInfo()
	if err != nil {
		return nil, err
	}

	for i := range info.Symbols {
		if info.Symbols[i].Status == "TRADING" {
			validCurrencyPairs = append(validCurrencyPairs, info.Symbols[i].BaseAsset+"-"+info.Symbols[i].QuoteAsset)
		}
	}
	return validCurrencyPairs, nil
}

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (b *Binance) GetExchangeInfo() (ExchangeInfo, error) {
	var resp ExchangeInfo
	path := b.APIUrl + exchangeInfo

	return resp, b.SendHTTPRequest(path, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetOrderBook(obd OrderBookDataRequestParams) (OrderBook, error) {
	orderbook, resp := OrderBook{}, OrderBookData{}

	if err := b.CheckLimit(obd.Limit); err != nil {
		return orderbook, err
	}
	if err := b.CheckSymbol(obd.Symbol); err != nil {
		return orderbook, err
	}

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(obd.Symbol))
	params.Set("limit", fmt.Sprintf("%d", obd.Limit))

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, orderBookDepth, params.Encode())

	if err := b.SendHTTPRequest(path, &resp); err != nil {
		return orderbook, err
	}

	for _, asks := range resp.Asks {
		var ASK struct {
			Price    float64
			Quantity float64
		}
		for i, ask := range asks.([]interface{}) {
			switch i {
			case 0:
				ASK.Price, _ = strconv.ParseFloat(ask.(string), 64)
			case 1:
				ASK.Quantity, _ = strconv.ParseFloat(ask.(string), 64)
				orderbook.Asks = append(orderbook.Asks, ASK)
			}
		}
	}

	for _, bids := range resp.Bids {
		var BID struct {
			Price    float64
			Quantity float64
		}
		for i, bid := range bids.([]interface{}) {
			switch i {
			case 0:
				BID.Price, _ = strconv.ParseFloat(bid.(string), 64)
			case 1:
				BID.Quantity, _ = strconv.ParseFloat(bid.(string), 64)
				orderbook.Bids = append(orderbook.Bids, BID)
			}
		}
	}

	orderbook.LastUpdateID = resp.LastUpdateID
	return orderbook, nil
}

// GetRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (b *Binance) GetRecentTrades(rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	var resp []RecentTrade

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(rtr.Symbol))
	params.Set("limit", fmt.Sprintf("%d", rtr.Limit))

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, recentTrades, params.Encode())

	return resp, b.SendHTTPRequest(path, &resp)
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
	if err := b.CheckSymbol(symbol); err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(symbol))
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, aggregatedTrades, params.Encode())

	return resp, b.SendHTTPRequest(path, &resp)
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
	var kline []CandleStick

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

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, candleStick, params.Encode())

	if err := b.SendHTTPRequest(path, &resp); err != nil {
		return kline, err
	}

	for _, responseData := range resp.([]interface{}) {
		var candle CandleStick
		for i, individualData := range responseData.([]interface{}) {
			switch i {
			case 0:
				candle.OpenTime = individualData.(float64)
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
				candle.CloseTime = individualData.(float64)
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
		kline = append(kline, candle)
	}
	return kline, nil
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binance) GetAveragePrice(symbol string) (AveragePrice, error) {
	resp := AveragePrice{}

	if err := b.CheckSymbol(symbol); err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, averagePrice, params.Encode())

	return resp, b.SendHTTPRequest(path, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binance) GetPriceChangeStats(symbol string) (PriceChangeStats, error) {
	resp := PriceChangeStats{}

	if err := b.CheckSymbol(symbol); err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, priceChange, params.Encode())

	return resp, b.SendHTTPRequest(path, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binance) GetTickers() ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	path := fmt.Sprintf("%s%s", b.APIUrl, priceChange)
	return resp, b.SendHTTPRequest(path, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Binance) GetLatestSpotPrice(symbol string) (SymbolPrice, error) {
	resp := SymbolPrice{}

	if err := b.CheckSymbol(symbol); err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, symbolPrice, params.Encode())

	return resp, b.SendHTTPRequest(path, &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binance) GetBestPrice(symbol string) (BestPrice, error) {
	resp := BestPrice{}

	if err := b.CheckSymbol(symbol); err != nil {
		return resp, err
	}

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(symbol))

	path := fmt.Sprintf("%s%s?%s", b.APIUrl, bestPrice, params.Encode())

	return resp, b.SendHTTPRequest(path, &resp)
}

// NewOrder sends a new order to Binance
func (b *Binance) NewOrder(o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse

	path := fmt.Sprintf("%s%s", b.APIUrl, newOrder)

	params := url.Values{}
	params.Set("symbol", o.Symbol)
	params.Set("side", string(o.Side))
	params.Set("type", string(o.TradeType))
	params.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	if o.TradeType == "LIMIT" {
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

	if err := b.SendAuthHTTPRequest(http.MethodPost, path, params, &resp); err != nil {
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

	path := fmt.Sprintf("%s%s", b.APIUrl, cancelOrder)

	params := url.Values{}
	params.Set("symbol", symbol)

	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}

	return resp, b.SendAuthHTTPRequest(http.MethodDelete, path, params, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted against the rate limiter
// is equal to the number of symbols currently trading on the exchange.
func (b *Binance) OpenOrders(symbol string) ([]QueryOrderData, error) {
	var resp []QueryOrderData
	path := fmt.Sprintf("%s%s", b.APIUrl, openOrders)
	params := url.Values{}

	if symbol != "" {
		params.Set("symbol", common.StringToUpper(symbol))
	}

	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// AllOrders Get all account orders; active, canceled, or filled.
// orderId optional param
// limit optional param, default 500; max 500
func (b *Binance) AllOrders(symbol, orderID, limit string) ([]QueryOrderData, error) {
	var resp []QueryOrderData

	path := fmt.Sprintf("%s%s", b.APIUrl, allOrders)

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(symbol))
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// QueryOrder returns information on a past order
func (b *Binance) QueryOrder(symbol, origClientOrderID string, orderID int64) (QueryOrderData, error) {
	var resp QueryOrderData

	path := fmt.Sprintf("%s%s", b.APIUrl, queryOrder)

	params := url.Values{}
	params.Set("symbol", common.StringToUpper(symbol))
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, &resp); err != nil {
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

	path := fmt.Sprintf("%s%s", b.APIUrl, accountInfo)
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(http.MethodGet, path, params, &resp); err != nil {
		return &resp.Account, err
	}

	if resp.Code != 0 {
		return &resp.Account, errors.New(resp.Msg)
	}

	return &resp.Account, nil
}

// SendHTTPRequest sends an unauthenticated request
func (b *Binance) SendHTTPRequest(path string, result interface{}) error {
	return b.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binance) SendAuthHTTPRequest(method, path string, params url.Values, result interface{}) error {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	if params == nil {
		params = url.Values{}
	}
	params.Set("recvWindow", strconv.FormatInt(common.RecvWindow(5*time.Second), 10))
	params.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))

	signature := params.Encode()
	hmacSigned := common.GetHMAC(common.HashSHA256, []byte(signature), []byte(b.APISecret))
	hmacSignedStr := common.HexEncodeToString(hmacSigned)

	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.APIKey

	if b.Verbose {
		log.Debugf("sent path: %s", path)
	}

	path = common.EncodeURLValues(path, params)
	path += fmt.Sprintf("&signature=%s", hmacSignedStr)

	interim := json.RawMessage{}

	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
	}{}

	err := b.SendPayload(method,
		path,
		headers,
		bytes.NewBuffer(nil),
		&interim,
		true,
		false,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
	if err != nil {
		return err
	}

	if err := common.JSONDecode(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" {
			return errors.New(errCap.Message)
		}
	}

	return common.JSONDecode(interim, result)
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
func (b *Binance) CheckSymbol(symbol string) error {
	enPairs := b.GetAvailableCurrencies()
	for x := range enPairs {
		if exchange.FormatExchangeCurrency(b.Name, enPairs[x]).String() == symbol {
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
	b.validLimits = []int{5, 10, 20, 50, 100, 500, 1000}
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
	path := fmt.Sprintf("%s%s", b.APIUrl, withdraw)

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

	if err := b.SendAuthHTTPRequest(http.MethodPost, path, params, &resp); err != nil {
		return "", err
	}

	if !resp.Success {
		return resp.ID, errors.New(resp.Msg)
	}

	return resp.ID, nil
}

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (b *Binance) GetDepositAddressForCurrency(currency string) (string, error) {
	path := fmt.Sprintf("%s%s", b.APIUrl, depositAddress)

	resp := struct {
		Address    string `json:"address"`
		Success    bool   `json:"success"`
		AddressTag string `json:"addressTag"`
	}{}

	params := url.Values{}
	params.Set("asset", currency)
	params.Set("status", "true")

	return resp.Address,
		b.SendAuthHTTPRequest(http.MethodGet, path, params, &resp)
}
