package hitbtc

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
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
	WebsocketConn *websocket.Conn
}

// SetDefaults sets default settings for hitbtc
func (h *HitBTC) SetDefaults() {
	h.Name = "HitBTC"
	h.Enabled = false
	h.Fee = 0
	h.Verbose = false
	h.RESTPollingDelay = 10
	h.APIWithdrawPermissions = exchange.AutoWithdrawCrypto
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
	h.WebsocketInit()
}

// Setup sets user exchange configuration settings
func (h *HitBTC) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		h.SetEnabled(false)
	} else {
		h.Enabled = true
		h.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		h.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		h.SetHTTPClientTimeout(exch.HTTPTimeout)
		h.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		h.RESTPollingDelay = exch.RESTPollingDelay // Max 60000ms
		h.Verbose = exch.Verbose
		h.Websocket.SetEnabled(exch.Websocket)
		h.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		h.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		h.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
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
		err = h.WebsocketSetup(h.WsConnect,
			exch.Name,
			exch.Websocket,
			hitbtcWebsocketAddress,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
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
	resp := []Symbol{}
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
	resp := []Symbol{}
	path := fmt.Sprintf("%s/%s", h.APIUrl, apiV2Symbol)

	return resp, h.SendHTTPRequest(path, &resp)
}

// GetTicker returns ticker information
func (h *HitBTC) GetTicker(symbol string) (map[string]Ticker, error) {
	resp1 := []TickerResponse{}
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

		for _, item := range resp1 {
			if item.Symbol != "" {
				ret[item.Symbol] = item
			}
		}
	} else {
		err = h.SendHTTPRequest(path, &resp2)
		ret[resp2.Symbol] = resp2
	}

	if err == nil {
		for x, y := range ret {
			tick := Ticker{}

			ask, _ := strconv.ParseFloat(y.Ask, 64)
			tick.Ask = ask

			bid, _ := strconv.ParseFloat(y.Bid, 64)
			tick.Bid = bid

			high, _ := strconv.ParseFloat(y.High, 64)
			tick.High = high

			last, _ := strconv.ParseFloat(y.Last, 64)
			tick.Last = last

			low, _ := strconv.ParseFloat(y.Low, 64)
			tick.Low = low

			open, _ := strconv.ParseFloat(y.Open, 64)
			tick.Open = open

			vol, _ := strconv.ParseFloat(y.Volume, 64)
			tick.Volume = vol

			volQuote, _ := strconv.ParseFloat(y.VolumeQuote, 64)
			tick.VolumeQuote = volQuote

			tick.Symbol = y.Symbol
			tick.Timestamp = y.Timestamp
			result[x] = tick
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

	resp := []TradeHistory{}
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
	for _, x := range resp.Asks {
		ob.Asks = append(ob.Asks, x)
	}

	for _, x := range resp.Bids {
		ob.Bids = append(ob.Bids, x)
	}
	return ob, nil
}

// GetCandles returns candles which is used for OHLC a specific symbol.
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

	resp := []ChartData{}
	path := fmt.Sprintf("%s/%s/%s?%s", h.APIUrl, apiV2Candles, currencyPair, vals.Encode())

	return resp, h.SendHTTPRequest(path, &resp)
}

// Authenticated Market Data
// https://api.hitbtc.com/?python#market-data

// GetBalances returns full balance for your account
func (h *HitBTC) GetBalances() (map[string]Balance, error) {
	result := []Balance{}
	err := h.SendAuthenticatedHTTPRequest("GET", apiV2Balance, url.Values{}, &result)
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
	resp := DepositCryptoAddresses{}
	err := h.SendAuthenticatedHTTPRequest("GET", apiV2CryptoAddress+"/"+currency, url.Values{}, &resp)

	return resp, err
}

// GenerateNewAddress generates a new deposit address for a currency
func (h *HitBTC) GenerateNewAddress(currency string) (DepositCryptoAddresses, error) {
	resp := DepositCryptoAddresses{}
	err := h.SendAuthenticatedHTTPRequest("POST", apiV2CryptoAddress+"/"+currency, url.Values{}, &resp)

	return resp, err
}

// GetActiveorders returns all your active orders
func (h *HitBTC) GetActiveorders(currency string) ([]Order, error) {
	resp := []Order{}
	err := h.SendAuthenticatedHTTPRequest("GET", orders+"?symbol="+currency, url.Values{}, &resp)

	return resp, err
}

// GetAuthenticatedTradeHistory returns your trade history
func (h *HitBTC) GetAuthenticatedTradeHistory(currency, start, end string) (interface{}, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if end != "" {
		values.Set("end", end)
	}

	if currency != "" && currency != "all" {
		values.Set("currencyPair", currency)
		result := AuthenticatedTradeHistoryResponse{}

		return result, h.SendAuthenticatedHTTPRequest("POST", apiV2TradeHistory, values, &result.Data)
	}

	values.Set("currencyPair", "all")
	result := AuthenticatedTradeHistoryAll{}

	return result, h.SendAuthenticatedHTTPRequest("POST", apiV2TradeHistory, values, &result.Data)
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

	err := h.SendAuthenticatedHTTPRequest("POST", orderBuy, values, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// CancelExistingOrder cancels a specific order by OrderID
func (h *HitBTC) CancelExistingOrder(orderID int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}

	err := h.SendAuthenticatedHTTPRequest("DELETE", orderBuy+"/"+strconv.FormatInt(orderID, 10), values, &result)

	if err != nil {
		return false, err
	}

	if result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// CancelAllExistingOrders cancels all open orders
func (h *HitBTC) CancelAllExistingOrders() error {
	result := GenericResponse{}
	values := url.Values{}

	err := h.SendAuthenticatedHTTPRequest("DELETE", orderBuy, values, &result)

	if err != nil {
		return err
	}

	return nil
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

	err := h.SendAuthenticatedHTTPRequest("POST", orderMove, values, &result)

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

	err := h.SendAuthenticatedHTTPRequest("POST", apiV2CryptoWithdraw, values, &result)

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
	err := h.SendAuthenticatedHTTPRequest("GET", apiV2FeeInfo+"/"+currencyPair, url.Values{}, &result)

	return result, err
}

// GetTradableBalances returns current tradable balances
func (h *HitBTC) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := h.SendAuthenticatedHTTPRequest("POST", tradableBalances, url.Values{}, &result.Data)

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

	err := h.SendAuthenticatedHTTPRequest("POST", transferBalance, values, &result)

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
	return h.SendPayload("GET", path, nil, nil, result, false, h.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated http request
func (h *HitBTC) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !h.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(h.APIKey+":"+h.APISecret))

	path := fmt.Sprintf("%s/%s", h.APIUrl, endpoint)

	return h.SendPayload(method, path, headers, bytes.NewBufferString(values.Encode()), result, true, h.Verbose)
}

// GetFee returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFee(feeBuilder exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feeInfo, err := h.GetFeeInfo(feeBuilder.FirstCurrency + feeBuilder.Delimiter + feeBuilder.SecondCurrency)
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeInfo, feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		currencyInfo, err := h.GetCurrency(feeBuilder.FirstCurrency)
		if err != nil {
			return 0, err
		}
		fee, err = strconv.ParseFloat(currencyInfo.PayoutFee, 64)
		if err != nil {
			return 0, err
		}
	case exchange.CyptocurrencyDepositFee:
		fee = calculateCryptocurrencyDepositFee(feeBuilder.FirstCurrency, feeBuilder.Amount)
	}

	return fee, nil
}

func calculateCryptocurrencyDepositFee(currency string, amount float64) float64 {
	var fee float64
	if currency == symbol.BTC {
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
