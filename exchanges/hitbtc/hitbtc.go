package hitbtc

import (
	"bytes"
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

const (
	// API
	apiURL = "https://api.hitbtc.com"

	// Public
	apiV2Trades    = "api/2/public/trades"
	apiV2Currency  = "api/2/public/currency"
	apiV2Symbol    = "api/2/public/symbol"
	apiV2Ticker    = "api/2/public/ticker"
	apiV2Orderbook = "api/2/public/orderbook"
	apiV2Candles   = "api/2/public/candles"

	// Authenticated
	apiV2Balance        = "api/2/trading/balance"
	apiV2CryptoAddress  = "api/2/account/crypto/address"
	apiV2CryptoWithdraw = "api/2/account/crypto/withdraw"
	apiV2TradeHistory   = "api/2/history/trades"
	apiV2OrderHistory   = "api/2/history/order"
	apiv2OpenOrders     = "api/2/order"
	apiV2FeeInfo        = "api/2/trading/fee"
	orders              = "order"
	orderBuy            = "api/2/order"
	orderSell           = "sell"
	orderMove           = "moveOrder"
	tradableBalances    = "returnTradableBalances"
	transferBalance     = "transferBalance"

	hitbtcAuthRate   = 0
	hitbtcUnauthRate = 0
)

// HitBTC is the overarching type across the hitbtc package
type HitBTC struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
}

// SetDefaults sets default settings for hitbtc
func (h *HitBTC) SetDefaults() {
	h.Name = "HitBTC"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.RESTPollingDelay = 10
	h.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	h.RequestCurrencyPairFormat.Delimiter = ""
	h.RequestCurrencyPairFormat.Uppercase = true
	h.ConfigCurrencyPairFormat.Delimiter = "-"
	h.ConfigCurrencyPairFormat.Uppercase = true
	h.AssetTypes = []string{ticker.Spot}
	h.SupportsAutoPairUpdating = true
	h.SupportsRESTTickerBatching = true
	h.Requester = request.New(h.Name,
		request.NewRateLimit(time.Second, hitbtcAuthRate),
		request.NewRateLimit(time.Second, hitbtcUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	h.APIUrlDefault = apiURL
	h.APIUrl = h.APIUrlDefault
	h.Websocket = wshandler.New()
	h.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketSubmitOrderSupported |
		wshandler.WebsocketCancelOrderSupported |
		wshandler.WebsocketMessageCorrelationSupported
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (h *HitBTC) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		h.SetHTTPClientTimeout(exch.HTTPTimeout)
		h.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		h.RESTPollingDelay = exch.RESTPollingDelay // Max 60000ms
		h.Verbose = exch.Verbose
		h.HTTPDebugging = exch.HTTPDebugging
		h.Websocket.SetWsStatusAndConnection(exch.Websocket)
		h.BaseCurrencies = exch.BaseCurrencies
		h.AvailablePairs = exch.AvailablePairs
		h.EnabledPairs = exch.EnabledPairs
		err := h.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = h.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = h.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = h.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = h.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = h.Websocket.Setup(h.WsConnect,
			h.Subscribe,
			h.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			hitbtcWebsocketAddress,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		h.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         h.Name,
			URL:                  h.Websocket.GetWebsocketURL(),
			ProxyURL:             h.Websocket.GetProxyAddress(),
			Verbose:              h.Verbose,
			RateLimit:            rateLimit,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
		h.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			true,
			false,
			exch.Name)
	}
}

// Public Market Data
// https://api.hitbtc.com/?python#market-data

// GetCurrencies returns the actual list of available currencies, tokens, ICO
// etc.
func (h *HitBTC) GetCurrencies() (map[string]Currencies, error) {
	type Response struct {
		Data []Currencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/%s", h.APIUrl, apiV2Currency)

	ret := make(map[string]Currencies)
	err := h.SendHTTPRequest(path, &resp.Data)
	if err != nil {
		return ret, err
	}

	for _, id := range resp.Data {
		ret[id.ID] = id
	}
	return ret, err
}

// GetCurrency returns the actual list of available currencies, tokens, ICO
// etc.
func (h *HitBTC) GetCurrency(currency string) (Currencies, error) {
	type Response struct {
		Data Currencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/%s/%s", h.APIUrl, apiV2Currency, currency)

	return resp.Data, h.SendHTTPRequest(path, &resp.Data)
}

// GetSymbols Return the actual list of currency symbols (currency pairs) traded
// on HitBTC exchange. The first listed currency of a symbol is called the base
// currency, and the second currency is called the quote currency. The currency
// pair indicates how much of the quote currency is needed to purchase one unit
// of the base currency.
func (h *HitBTC) GetSymbols(symbol string) ([]string, error) {
	var resp []Symbol
	path := fmt.Sprintf("%s/%s/%s", h.APIUrl, apiV2Symbol, symbol)

	ret := make([]string, 0, len(resp))
	err := h.SendHTTPRequest(path, &resp)
	if err != nil {
		return ret, err
	}

	for _, x := range resp {
		ret = append(ret, x.ID)
	}
	return ret, err
}

// GetSymbolsDetailed is the same as above but returns an array of symbols with
// all their details.
func (h *HitBTC) GetSymbolsDetailed() ([]Symbol, error) {
	var resp []Symbol
	path := fmt.Sprintf("%s/%s", h.APIUrl, apiV2Symbol)

	return resp, h.SendHTTPRequest(path, &resp)
}

// GetTicker returns ticker information
func (h *HitBTC) GetTicker(symbol string) (map[string]Ticker, error) {
	var resp1 []TickerResponse
	resp2 := TickerResponse{}
	ret := make(map[string]TickerResponse)
	result := make(map[string]Ticker)
	path := fmt.Sprintf("%s/%s/%s", h.APIUrl, apiV2Ticker, symbol)
	var err error

	if symbol == "" {
		err = h.SendHTTPRequest(path, &resp1)
		if err != nil {
			return nil, err
		}

		for i := range resp1 {
			if resp1[i].Symbol != "" {
				ret[resp1[i].Symbol] = resp1[i]
			}
		}
	} else {
		err = h.SendHTTPRequest(path, &resp2)
		ret[resp2.Symbol] = resp2
	}

	if err == nil {
		for i := range ret {
			tick := Ticker{}

			ask, _ := strconv.ParseFloat(ret[i].Ask, 64)
			tick.Ask = ask

			bid, _ := strconv.ParseFloat(ret[i].Bid, 64)
			tick.Bid = bid

			high, _ := strconv.ParseFloat(ret[i].High, 64)
			tick.High = high

			last, _ := strconv.ParseFloat(ret[i].Last, 64)
			tick.Last = last

			low, _ := strconv.ParseFloat(ret[i].Low, 64)
			tick.Low = low

			open, _ := strconv.ParseFloat(ret[i].Open, 64)
			tick.Open = open

			vol, _ := strconv.ParseFloat(ret[i].Volume, 64)
			tick.Volume = vol

			volQuote, _ := strconv.ParseFloat(ret[i].VolumeQuote, 64)
			tick.VolumeQuote = volQuote

			tick.Symbol = ret[i].Symbol
			tick.Timestamp = ret[i].Timestamp
			result[i] = tick
		}
	}

	return result, err
}

// GetTrades returns trades from hitbtc
func (h *HitBTC) GetTrades(currencyPair, from, till, limit, offset, by, sort string) ([]TradeHistory, error) {
	// start   Number or Datetime
	// end     Number or Datetime
	// limit   Number
	// offset  Number
	// by      Filtration definition. Accepted values: id, timestamh. Default timestamp
	// sort    Default DESC
	vals := url.Values{}

	if from != "" {
		vals.Set("from", from)
	}

	if till != "" {
		vals.Set("till", till)
	}

	if limit != "" {
		vals.Set("limit", limit)
	}

	if offset != "" {
		vals.Set("offset", offset)
	}

	if by != "" {
		vals.Set("by", by)
	}

	if sort != "" {
		vals.Set("sort", sort)
	}

	var resp []TradeHistory
	path := fmt.Sprintf("%s/%s/%s?%s", h.APIUrl, apiV2Trades, currencyPair, vals.Encode())

	return resp, h.SendHTTPRequest(path, &resp)
}

// GetOrderbook an order book is an electronic list of buy and sell orders for a
// specific symbol, organized by price level.
func (h *HitBTC) GetOrderbook(currencyPair string, limit int) (Orderbook, error) {
	// limit Limit of orderbook levels, default 100. Set 0 to view full orderbook levels
	vals := url.Values{}

	if limit != 0 {
		vals.Set("limit", strconv.Itoa(limit))
	}

	resp := OrderbookResponse{}
	path := fmt.Sprintf("%s/%s/%s?%s", h.APIUrl, apiV2Orderbook, currencyPair, vals.Encode())

	err := h.SendHTTPRequest(path, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	ob := Orderbook{}
	ob.Asks = append(ob.Asks, resp.Asks...)
	ob.Bids = append(ob.Bids, resp.Bids...)
	return ob, nil
}

// GetCandles returns candles which is used for OHLC a specific currency.
// Note: Result contain candles only with non zero volume.
func (h *HitBTC) GetCandles(currencyPair, limit, period string) ([]ChartData, error) {
	// limit   Limit of candles, default 100.
	// period  One of: M1 (one minute), M3, M5, M15, M30, H1, H4, D1, D7, 1M (one month). Default is M30 (30 minutes).
	vals := url.Values{}

	if limit != "" {
		vals.Set("limit", limit)
	}

	if period != "" {
		vals.Set("period", period)
	}

	var resp []ChartData
	path := fmt.Sprintf("%s/%s/%s?%s", h.APIUrl, apiV2Candles, currencyPair, vals.Encode())

	return resp, h.SendHTTPRequest(path, &resp)
}

// Authenticated Market Data
// https://api.hitbtc.com/?python#market-data

// GetBalances returns full balance for your account
func (h *HitBTC) GetBalances() (map[string]Balance, error) {
	var result []Balance
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, apiV2Balance, url.Values{}, &result)
	ret := make(map[string]Balance)

	if err != nil {
		return ret, err
	}

	for _, item := range result {
		ret[item.Currency] = item
	}

	return ret, nil
}

// GetDepositAddresses returns a deposit address for a specific currency
func (h *HitBTC) GetDepositAddresses(currency string) (DepositCryptoAddresses, error) {
	var resp DepositCryptoAddresses

	return resp,
		h.SendAuthenticatedHTTPRequest(http.MethodGet,
			apiV2CryptoAddress+"/"+currency,
			url.Values{},
			&resp)
}

// GenerateNewAddress generates a new deposit address for a currency
func (h *HitBTC) GenerateNewAddress(currency string) (DepositCryptoAddresses, error) {
	resp := DepositCryptoAddresses{}
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, apiV2CryptoAddress+"/"+currency, url.Values{}, &resp)

	return resp, err
}

// GetActiveorders returns all your active orders
func (h *HitBTC) GetActiveorders(currency string) ([]Order, error) {
	var resp []Order
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, orders+"?symbol="+currency, url.Values{}, &resp)

	return resp, err
}

// GetTradeHistoryForCurrency returns your trade history
func (h *HitBTC) GetTradeHistoryForCurrency(currency, start, end string) (AuthenticatedTradeHistoryResponse, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if end != "" {
		values.Set("end", end)
	}

	values.Set("currencyPair", currency)
	result := AuthenticatedTradeHistoryResponse{}

	return result, h.SendAuthenticatedHTTPRequest(http.MethodPost, apiV2TradeHistory, values, &result.Data)
}

// GetTradeHistoryForAllCurrencies returns your trade history
func (h *HitBTC) GetTradeHistoryForAllCurrencies(start, end string) (AuthenticatedTradeHistoryAll, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if end != "" {
		values.Set("end", end)
	}

	values.Set("currencyPair", "all")
	result := AuthenticatedTradeHistoryAll{}

	return result, h.SendAuthenticatedHTTPRequest(http.MethodPost, apiV2TradeHistory, values, &result.Data)
}

// GetOrders List of your order history.
func (h *HitBTC) GetOrders(currency string) ([]OrderHistoryResponse, error) {
	values := url.Values{}
	values.Set("symbol", currency)
	var result []OrderHistoryResponse

	return result, h.SendAuthenticatedHTTPRequest(http.MethodGet, apiV2OrderHistory, values, &result)
}

// GetOpenOrders List of your currently open orders.
func (h *HitBTC) GetOpenOrders(currency string) ([]OrderHistoryResponse, error) {
	values := url.Values{}
	values.Set("symbol", currency)
	var result []OrderHistoryResponse

	return result, h.SendAuthenticatedHTTPRequest(http.MethodGet, apiv2OpenOrders, values, &result)
}

// PlaceOrder places an order on the exchange
func (h *HitBTC) PlaceOrder(currency string, rate, amount float64, orderType, side string) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	values.Set("symbol", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("quantity", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("side", side)
	values.Set("price", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("type", orderType)

	return result, h.SendAuthenticatedHTTPRequest(http.MethodPost, orderBuy, values, &result)
}

// CancelExistingOrder cancels a specific order by OrderID
func (h *HitBTC) CancelExistingOrder(orderID int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}

	err := h.SendAuthenticatedHTTPRequest(http.MethodDelete, orderBuy+"/"+strconv.FormatInt(orderID, 10), values, &result)

	if err != nil {
		return false, err
	}

	if result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// CancelAllExistingOrders cancels all open orders
func (h *HitBTC) CancelAllExistingOrders() ([]Order, error) {
	var result []Order
	values := url.Values{}
	return result, h.SendAuthenticatedHTTPRequest(http.MethodDelete, orderBuy, values, &result)
}

// MoveOrder generates a new move order
func (h *HitBTC) MoveOrder(orderID int64, rate, amount float64) (MoveOrderResponse, error) {
	result := MoveOrderResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))

	if amount != 0 {
		values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, orderMove, values, &result)

	if err != nil {
		return result, err
	}

	if result.Success != 1 {
		return result, errors.New(result.Error)
	}

	return result, nil
}

// Withdraw allows for the withdrawal to a specific address
func (h *HitBTC) Withdraw(currency, address string, amount float64) (bool, error) {
	result := Withdraw{}
	values := url.Values{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, apiV2CryptoWithdraw, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetFeeInfo returns current fee information
func (h *HitBTC) GetFeeInfo(currencyPair string) (Fee, error) {
	result := Fee{}
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, apiV2FeeInfo+"/"+currencyPair, url.Values{}, &result)

	return result, err
}

// GetTradableBalances returns current tradable balances
func (h *HitBTC) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, tradableBalances, url.Values{}, &result.Data)

	if err != nil {
		return nil, err
	}

	balances := make(map[string]map[string]float64)

	for x, y := range result.Data {
		balances[x] = make(map[string]float64)
		for z, w := range y {
			balances[x][z], _ = strconv.ParseFloat(w.(string), 64)
		}
	}

	return balances, nil
}

// TransferBalance transfers a balance
func (h *HitBTC) TransferBalance(currency, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := GenericResponse{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := h.SendAuthenticatedHTTPRequest(http.MethodPost,
		transferBalance,
		values,
		&result)

	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (h *HitBTC) SendHTTPRequest(path string, result interface{}) error {
	return h.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		result,
		false,
		false,
		h.Verbose,
		h.HTTPDebugging,
		h.HTTPRecording)
}

// SendAuthenticatedHTTPRequest sends an authenticated http request
func (h *HitBTC) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			h.Name)
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(h.APIKey+":"+h.APISecret))

	path := fmt.Sprintf("%s/%s", h.APIUrl, endpoint)

	return h.SendPayload(method,
		path,
		headers,
		bytes.NewBufferString(values.Encode()),
		result,
		true,
		false,
		h.Verbose,
		h.HTTPDebugging,
		h.HTTPRecording)
}

// GetFee returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feeInfo, err := h.GetFeeInfo(feeBuilder.Pair.Base.String() +
			feeBuilder.Pair.Delimiter +
			feeBuilder.Pair.Quote.String())

		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeInfo, feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		currencyInfo, err := h.GetCurrency(feeBuilder.Pair.Base.String())
		if err != nil {
			return 0, err
		}
		fee, err = strconv.ParseFloat(currencyInfo.PayoutFee, 64)
		if err != nil {
			return 0, err
		}
	case exchange.CyptocurrencyDepositFee:
		fee = calculateCryptocurrencyDepositFee(feeBuilder.Pair.Base,
			feeBuilder.Amount)
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

func calculateCryptocurrencyDepositFee(c currency.Code, amount float64) float64 {
	var fee float64
	if c == currency.BTC {
		fee = 0.0006
	}
	return fee * amount
}

func calculateTradingFee(feeInfo Fee, purchasePrice, amount float64, isMaker bool) float64 {
	var volumeFee float64
	if isMaker {
		volumeFee = feeInfo.ProvideLiquidityRate
	} else {
		volumeFee = feeInfo.TakeLiquidityRate
	}

	return volumeFee * amount * purchasePrice
}
