package hitbtc

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
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
	orderBuy            = "buy"
	orderSell           = "sell"
	orderCancel         = "cancelOrder"
	orderMove           = "moveOrder"
	tradableBalances    = "returnTradableBalances"
	transferBalance     = "transferBalance"
)

// HitBTC is the overarching type across the hitbtc package
type HitBTC struct {
	exchange.Base
}

// SetDefaults sets default settings for hitbtc
func (p *HitBTC) SetDefaults() {
	p.Name = "HitBTC"
	p.Enabled = false
	p.Fee = 0
	p.Verbose = false
	p.Websocket = false
	p.RESTPollingDelay = 10
	p.RequestCurrencyPairFormat.Delimiter = ""
	p.RequestCurrencyPairFormat.Uppercase = true
	p.ConfigCurrencyPairFormat.Delimiter = "-"
	p.ConfigCurrencyPairFormat.Uppercase = true
	p.AssetTypes = []string{ticker.Spot}
}

// Setup sets user exchange configuration settings
func (p *HitBTC) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		p.SetEnabled(false)
	} else {
		p.Enabled = true
		p.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		p.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		p.RESTPollingDelay = exch.RESTPollingDelay // Max 60000ms
		p.Verbose = exch.Verbose
		p.Websocket = exch.Websocket
		p.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		p.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		p.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := p.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = p.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetFee returns the fee for hitbtc
func (p *HitBTC) GetFee() float64 {
	return p.Fee
}

// Public Market Data
// https://api.hitbtc.com/?python#market-data

// GetCurrencies returns the actual list of available currencies, tokens, ICO
// etc.
func (p *HitBTC) GetCurrencies(currency string) (map[string]Currencies, error) {
	type Response struct {
		Data []Currencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/%s/%s", apiURL, apiV2Currency, currency)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp.Data)
	ret := make(map[string]Currencies)
	for _, id := range resp.Data {
		ret[id.ID] = id
	}

	return ret, err
}

// GetSymbols Return the actual list of currency symbols (currency pairs) traded
// on HitBTC exchange. The first listed currency of a symbol is called the base
// currency, and the second currency is called the quote currency. The currency
// pair indicates how much of the quote currency is needed to purchase one unit
// of the base currency.
func (p *HitBTC) GetSymbols(symbol string) ([]string, error) {
	resp := []Symbol{}
	path := fmt.Sprintf("%s/%s/%s", apiURL, apiV2Symbol, symbol)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
	ret := make([]string, 0, len(resp))
	for _, x := range resp {
		ret = append(ret, x.ID)
	}

	return ret, err
}

// GetSymbolsDetailed is the same as above but returns an array of symbols with
// all ther details.
func (p *HitBTC) GetSymbolsDetailed() ([]Symbol, error) {
	resp := []Symbol{}
	path := fmt.Sprintf("%s/%s", apiURL, apiV2Symbol)
	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// GetTicker returns ticker information
func (p *HitBTC) GetTicker(symbol string) (map[string]Ticker, error) {
	resp1 := []TickerResponse{}
	resp2 := TickerResponse{}
	ret := make(map[string]TickerResponse)
	result := make(map[string]Ticker)
	path := fmt.Sprintf("%s/%s/%s", apiURL, apiV2Ticker, symbol)
	var err error

	if symbol == "" {
		err = common.SendHTTPGetRequest(path, true, false, &resp1)
		if err != nil {
			return nil, err
		}

		for _, item := range resp1 {
			if item.Symbol != "" {
				ret[item.Symbol] = item
			}
		}
	} else {
		err = common.SendHTTPGetRequest(path, true, false, &resp2)
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
func (p *HitBTC) GetTrades(currencyPair, from, till, limit, offset, by, sort string) ([]TradeHistory, error) {
	// start   Number or Datetime
	// end     Number or Datetime
	// limit   Number
	// offset  Number
	// by      Filtration definition. Accepted values: id, timestamp. Default timestamp
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
	path := fmt.Sprintf("%s/%s/%s?%s", apiURL, apiV2Trades, currencyPair, vals.Encode())

	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// GetOrderbook an order book is an electronic list of buy and sell orders for a
// specific symbol, organized by price level.
func (p *HitBTC) GetOrderbook(currencyPair string, limit int) (Orderbook, error) {
	// limit Limit of orderbook levels, default 100. Set 0 to view full orderbook levels
	vals := url.Values{}

	if limit != 0 {
		vals.Set("limit", strconv.Itoa(limit))
	}

	resp := OrderbookResponse{}
	path := fmt.Sprintf("%s/%s/%s?%s", apiURL, apiV2Orderbook, currencyPair, vals.Encode())

	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
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
func (p *HitBTC) GetCandles(currencyPair, limit, period string) ([]ChartData, error) {
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
	path := fmt.Sprintf("%s/%s/%s?%s", apiURL, apiV2Candles, currencyPair, vals.Encode())

	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// Authenticated Market Data
// https://api.hitbtc.com/?python#market-data

// GetBalances returns full balance for your account
func (p *HitBTC) GetBalances() (map[string]Balance, error) {
	result := []Balance{}
	err := p.SendAuthenticatedHTTPRequest("GET", apiV2Balance, url.Values{}, &result)
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
func (p *HitBTC) GetDepositAddresses(currency string) (DepositCryptoAddresses, error) {
	resp := DepositCryptoAddresses{}
	err := p.SendAuthenticatedHTTPRequest("GET", apiV2CryptoAddress+"/"+currency, url.Values{}, &resp)

	return resp, err
}

// GenerateNewAddress generates a new deposit address for a currency
func (p *HitBTC) GenerateNewAddress(currency string) (DepositCryptoAddresses, error) {
	resp := DepositCryptoAddresses{}
	err := p.SendAuthenticatedHTTPRequest("POST", apiV2CryptoAddress+"/"+currency, url.Values{}, &resp)

	return resp, err
}

// GetActiveorders returns all your active orders
func (p *HitBTC) GetActiveorders(currency string) ([]Order, error) {
	resp := []Order{}
	err := p.SendAuthenticatedHTTPRequest("GET", orders+"?symbol="+currency, url.Values{}, &resp)

	return resp, err
}

// GetAuthenticatedTradeHistory returns your trade history
func (p *HitBTC) GetAuthenticatedTradeHistory(currency, start, end string) (interface{}, error) {
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

		return result, p.SendAuthenticatedHTTPRequest("POST", apiV2TradeHistory, values, &result.Data)
	}

	values.Set("currencyPair", "all")
	result := AuthenticatedTradeHistoryAll{}

	return result, p.SendAuthenticatedHTTPRequest("POST", apiV2TradeHistory, values, &result.Data)
}

// PlaceOrder places an order on the exchange
func (p *HitBTC) PlaceOrder(currency string, rate, amount float64, immediate, fillOrKill, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = orderBuy
	} else {
		orderType = orderSell
	}

	values.Set("currencyPair", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	if immediate {
		values.Set("immediateOrCancel", "1")
	}

	if fillOrKill {
		values.Set("fillOrKill", "1")
	}

	err := p.SendAuthenticatedHTTPRequest("POST", orderType, values, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

// CancelOrder cancels a specific order by OrderID
func (p *HitBTC) CancelOrder(orderID int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))

	err := p.SendAuthenticatedHTTPRequest("POST", orderCancel, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// MoveOrder generates a new move order
func (p *HitBTC) MoveOrder(orderID int64, rate, amount float64) (MoveOrderResponse, error) {
	result := MoveOrderResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))

	if amount != 0 {
		values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	err := p.SendAuthenticatedHTTPRequest("POST", orderMove, values, &result)

	if err != nil {
		return result, err
	}

	if result.Success != 1 {
		return result, errors.New(result.Error)
	}

	return result, nil
}

// Withdraw allows for the withdrawal to a specific address
func (p *HitBTC) Withdraw(currency, address string, amount float64) (bool, error) {
	result := Withdraw{}
	values := url.Values{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := p.SendAuthenticatedHTTPRequest("POST", apiV2CryptoWithdraw, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// GetFeeInfo returns current fee information
func (p *HitBTC) GetFeeInfo(currencyPair string) (Fee, error) {
	result := Fee{}
	err := p.SendAuthenticatedHTTPRequest("GET", apiV2FeeInfo+"/"+currencyPair, url.Values{}, &result)

	return result, err
}

// GetTradableBalances returns current tradable balances
func (p *HitBTC) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", tradableBalances, url.Values{}, &result.Data)

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
func (p *HitBTC) TransferBalance(currency, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := GenericResponse{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := p.SendAuthenticatedHTTPRequest("POST", transferBalance, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

// SendAuthenticatedHTTPRequest sends an authenticated http request
func (p *HitBTC) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !p.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, p.Name)
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(p.APIKey+":"+p.APISecret))

	path := fmt.Sprintf("%s/%s", apiURL, endpoint)

	resp, err := common.SendHTTPRequest(method, path, headers, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)
	if err != nil {
		return err
	}

	return nil
}
