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
	APIURL = "https://api.hitbtc.com"

	// Public
	APIv2Trades    = "api/2/public/trades"
	APIv2Currency  = "api/2/public/currency"
	APIv2Symbol    = "api/2/public/symbol"
	APIv2Ticker    = "api/2/public/ticker"
	APIv2OrderBook = "api/2/public/orderbook"
	APIv2Candles   = "api/2/public/candles"

	// Authenticated
	APIv2Balance        = "api/2/trading/balance"
	APIv2CryptoAddress  = "api/2/account/crypto/address"
	APIv2CryptoWithdraw = "api/2/account/crypto/withdraw"
	APIv2TradeHistory   = "api/2/history/trades"
	APIv2FeeInfo        = "api/2/trading/fee"
	Orders              = "order"
	OrderBuy            = "buy"
	OrderSell           = "sell"
	OrderCancel         = "cancelOrder"
	OrderMove           = "moveOrder"
	TradableBalances    = "returnTradableBalances"
	TransferBalance     = "transferBalance"
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

// GetCurrencies
// Return the actual list of available currencies, tokens, ICO etc.
func (p *HitBTC) GetCurrencies(currency string) (map[string]Currencies, error) {
	type Response struct {
		Data []Currencies
	}
	resp := Response{}
	path := fmt.Sprintf("%s/%s/%s", APIURL, APIv2Currency, currency)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp.Data)
	ret := make(map[string]Currencies)
	for _, id := range resp.Data {
		ret[id.Id] = id
	}

	return ret, err
}

// GetSymbols
// Return the actual list of currency symbols (currency pairs) traded on HitBTC exchange.
// The first listed currency of a symbol is called the base currency, and the second currency
// is called the quote currency. The currency pair indicates how much of the quote currency
// is needed to purchase one unit of the base currency.
func (p *HitBTC) GetSymbols(symbol string) ([]string, error) {

	resp := []Symbol{}
	path := fmt.Sprintf("%s/%s/%s", APIURL, APIv2Symbol, symbol)
	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
	ret := make([]string, 0, len(resp))
	for _, x := range resp {
		ret = append(ret, x.Id)
	}

	return ret, err
}

// GetSymbolsDetailed is the same as above but returns an array of symbols with
// all ther details.
func (p *HitBTC) GetSymbolsDetailed() ([]Symbol, error) {
	resp := []Symbol{}
	path := fmt.Sprintf("%s/%s", APIURL, APIv2Symbol)
	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// GetTicker
// Return ticker information
func (p *HitBTC) GetTicker(symbol string) (map[string]Ticker, error) {

	resp1 := []TickerResponse{}
	resp2 := TickerResponse{}
	ret := make(map[string]TickerResponse)
	result := make(map[string]Ticker)
	path := fmt.Sprintf("%s/%s/%s", APIURL, APIv2Ticker, symbol)
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
	path := fmt.Sprintf("%s/%s/%s?%s", APIURL, APIv2Trades, currencyPair, vals.Encode())

	return resp, common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
}

// GetOrderbook
// An order book is an electronic list of buy and sell orders for a specific
// symbol, organized by price level.
func (p *HitBTC) GetOrderbook(currencyPair string, limit int) (Orderbook, error) {
	// limit Limit of orderbook levels, default 100. Set 0 to view full orderbook levels
	vals := url.Values{}

	if limit != 0 {
		vals.Set("limit", strconv.Itoa(limit))
	}

	resp := OrderbookResponse{}
	path := fmt.Sprintf("%s/%s/%s?%s", APIURL, APIv2OrderBook, currencyPair, vals.Encode())

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

// GetCandles
// A candles used for OHLC a specific symbol.
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
	path := fmt.Sprintf("%s/%s/%s?%s", APIURL, APIv2Candles, currencyPair, vals.Encode())

	err := common.SendHTTPGetRequest(path, true, p.Verbose, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Authenticated Market Data
// https://api.hitbtc.com/?python#market-data

// GetBalances
func (p *HitBTC) GetBalances() (map[string]Balance, error) {

	result := []Balance{}
	err := p.SendAuthenticatedHTTPRequest("GET", APIv2Balance, url.Values{}, &result)
	ret := make(map[string]Balance)

	if err != nil {
		return ret, err
	}

	for _, item := range result {
		ret[item.Currency] = item
	}

	return ret, nil
}

func (p *HitBTC) GetDepositAddresses(currency string) (DepositCryptoAddresses, error) {
	resp := DepositCryptoAddresses{}
	err := p.SendAuthenticatedHTTPRequest("GET", APIv2CryptoAddress+"/"+currency, url.Values{}, &resp)

	return resp, err
}

func (p *HitBTC) GenerateNewAddress(currency string) (DepositCryptoAddresses, error) {

	resp := DepositCryptoAddresses{}
	err := p.SendAuthenticatedHTTPRequest("POST", APIv2CryptoAddress+"/"+currency, url.Values{}, &resp)

	return resp, err
}

// Get Active orders
func (p *HitBTC) GetActiveOrders(currency string) ([]Order, error) {

	resp := []Order{}
	err := p.SendAuthenticatedHTTPRequest("GET", Orders+"?symbol="+currency, url.Values{}, &resp)

	return resp, err
}

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
		err := p.SendAuthenticatedHTTPRequest("POST", APIv2TradeHistory, values, &result.Data)

		if err != nil {
			return result, err
		}

		return result, nil
	} else {
		values.Set("currencyPair", "all")
		result := AuthenticatedTradeHistoryAll{}
		err := p.SendAuthenticatedHTTPRequest("POST", APIv2TradeHistory, values, &result.Data)

		if err != nil {
			return result, err
		}

		return result, nil
	}
}

func (p *HitBTC) PlaceOrder(currency string, rate, amount float64, immediate, fillOrKill, buy bool) (OrderResponse, error) {
	result := OrderResponse{}
	values := url.Values{}

	var orderType string
	if buy {
		orderType = OrderBuy
	} else {
		orderType = OrderSell
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

func (p *HitBTC) CancelOrder(orderID int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))

	err := p.SendAuthenticatedHTTPRequest("POST", OrderCancel, values, &result)

	if err != nil {
		return false, err
	}

	if result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

func (p *HitBTC) MoveOrder(orderID int64, rate, amount float64) (MoveOrderResponse, error) {
	result := MoveOrderResponse{}
	values := url.Values{}
	values.Set("orderNumber", strconv.FormatInt(orderID, 10))
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))

	if amount != 0 {
		values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}

	err := p.SendAuthenticatedHTTPRequest("POST", OrderMove, values, &result)

	if err != nil {
		return result, err
	}

	if result.Success != 1 {
		return result, errors.New(result.Error)
	}

	return result, nil
}

func (p *HitBTC) Withdraw(currency, address string, amount float64) (bool, error) {
	result := Withdraw{}
	values := url.Values{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := p.SendAuthenticatedHTTPRequest("POST", APIv2CryptoWithdraw, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, errors.New(result.Error)
	}

	return true, nil
}

func (p *HitBTC) GetFeeInfo(currencyPair string) (Fee, error) {
	result := Fee{}
	err := p.SendAuthenticatedHTTPRequest("GET", APIv2FeeInfo+"/"+currencyPair, url.Values{}, &result)

	return result, err
}

func (p *HitBTC) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := p.SendAuthenticatedHTTPRequest("POST", TradableBalances, url.Values{}, &result.Data)

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

func (p *HitBTC) TransferBalance(currency, from, to string, amount float64) (bool, error) {
	values := url.Values{}
	result := GenericResponse{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("fromAccount", from)
	values.Set("toAccount", to)

	err := p.SendAuthenticatedHTTPRequest("POST", TransferBalance, values, &result)

	if err != nil {
		return false, err
	}

	if result.Error != "" && result.Success != 1 {
		return false, errors.New(result.Error)
	}

	return true, nil
}

func (p *HitBTC) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, result interface{}) error {
	if !p.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, p.Name)
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(p.APIKey+":"+p.APISecret))

	path := fmt.Sprintf("%s/%s", APIURL, endpoint)
	resp, err := common.SendHTTPRequest(method, path, headers, bytes.NewBufferString(values.Encode()))

	if err != nil {
		return err
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}
	return nil
}
