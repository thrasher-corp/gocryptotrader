package hitbtc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
	apiOrder            = "api/2/order"
	orderMove           = "moveOrder"
	tradableBalances    = "returnTradableBalances"
	transferBalance     = "transferBalance"
)

// HitBTC is the overarching type across the hitbtc package
type HitBTC struct {
	exchange.Base
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
	path := fmt.Sprintf("/%s", apiV2Currency)

	ret := make(map[string]Currencies)
	err := h.SendHTTPRequest(exchange.RestSpot, path, &resp.Data)
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
	path := fmt.Sprintf("/%s/%s", apiV2Currency, currency)

	return resp.Data, h.SendHTTPRequest(exchange.RestSpot, path, &resp.Data)
}

// GetSymbols Return the actual list of currency symbols (currency pairs) traded
// on HitBTC exchange. The first listed currency of a symbol is called the base
// currency, and the second currency is called the quote currency. The currency
// pair indicates how much of the quote currency is needed to purchase one unit
// of the base currency.
func (h *HitBTC) GetSymbols(symbol string) ([]string, error) {
	var resp []Symbol
	path := fmt.Sprintf("/%s/%s", apiV2Symbol, symbol)

	ret := make([]string, 0, len(resp))
	err := h.SendHTTPRequest(exchange.RestSpot, path, &resp)
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
	path := fmt.Sprintf("/%s", apiV2Symbol)
	return resp, h.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// GetTicker returns ticker information
func (h *HitBTC) GetTicker(symbol string) (TickerResponse, error) {
	var resp TickerResponse
	path := fmt.Sprintf("/%s/%s", apiV2Ticker, symbol)
	return resp, h.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// GetTickers returns ticker information
func (h *HitBTC) GetTickers() ([]TickerResponse, error) {
	var resp []TickerResponse
	path := fmt.Sprintf("/%s/", apiV2Ticker)
	return resp, h.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// GetTrades returns trades from hitbtc
func (h *HitBTC) GetTrades(currencyPair, by, sort string, from, till, limit, offset int64) ([]TradeHistory, error) {
	urlValues := url.Values{}
	if from > 0 {
		urlValues.Set("from", strconv.FormatInt(from, 10))
	}
	if till > 0 {
		urlValues.Set("till", strconv.FormatInt(till, 10))
	}
	if limit > 0 {
		urlValues.Set("limit", strconv.FormatInt(limit, 10))
	}
	if offset > 0 {
		urlValues.Set("offset", strconv.FormatInt(offset, 10))
	}
	if by != "" {
		urlValues.Set("by", by)
	}
	if sort != "" {
		urlValues.Set("sort", sort)
	}

	var resp []TradeHistory
	path := fmt.Sprintf("/%s/%s?%s",
		apiV2Trades,
		currencyPair,
		urlValues.Encode())
	return resp, h.SendHTTPRequest(exchange.RestSpot, path, &resp)
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
	path := fmt.Sprintf("/%s/%s?%s",
		apiV2Orderbook,
		currencyPair,
		vals.Encode())

	err := h.SendHTTPRequest(exchange.RestSpot, path, &resp)
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
func (h *HitBTC) GetCandles(currencyPair, limit, period string, start, end time.Time) ([]ChartData, error) {
	// limit   Limit of candles, default 100.
	// period  One of: M1 (one minute), M3, M5, M15, M30, H1, H4, D1, D7, 1M (one month). Default is M30 (30 minutes).
	vals := url.Values{}
	if limit != "" {
		vals.Set("limit", limit)
	}

	if period != "" {
		vals.Set("period", period)
	}

	if !end.IsZero() && start.After(end) {
		return nil, errors.New("start time cannot be after end time")
	}

	if !start.IsZero() {
		vals.Set("from", start.Format(time.RFC3339))
	}

	if !end.IsZero() {
		vals.Set("till", end.Format(time.RFC3339))
	}

	var resp []ChartData
	path := fmt.Sprintf("/%s/%s?%s", apiV2Candles, currencyPair, vals.Encode())
	return resp, h.SendHTTPRequest(exchange.RestSpot, path, &resp)
}

// Authenticated Market Data
// https://api.hitbtc.com/?python#market-data

// GetBalances returns full balance for your account
func (h *HitBTC) GetBalances() (map[string]Balance, error) {
	var result []Balance
	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet,
		apiV2Balance,
		url.Values{},
		otherRequests,
		&result)
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
		h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet,
			apiV2CryptoAddress+"/"+currency,
			url.Values{},
			otherRequests,
			&resp)
}

// GenerateNewAddress generates a new deposit address for a currency
func (h *HitBTC) GenerateNewAddress(currency string) (DepositCryptoAddresses, error) {
	resp := DepositCryptoAddresses{}
	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		apiV2CryptoAddress+"/"+currency,
		url.Values{},
		otherRequests,
		&resp)

	return resp, err
}

// GetActiveorders returns all your active orders
func (h *HitBTC) GetActiveorders(currency string) ([]Order, error) {
	var resp []Order
	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet,
		orders+"?symbol="+currency,
		url.Values{},
		tradingRequests,
		&resp)

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

	return result, h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		apiV2TradeHistory,
		values,
		otherRequests,
		&result.Data)
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

	return result, h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		apiV2TradeHistory,
		values,
		otherRequests,
		&result.Data)
}

// GetOrders List of your order history.
func (h *HitBTC) GetOrders(currency string) ([]OrderHistoryResponse, error) {
	values := url.Values{}
	values.Set("symbol", currency)
	var result []OrderHistoryResponse

	return result, h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet,
		apiV2OrderHistory,
		values,
		tradingRequests,
		&result)
}

// GetOpenOrders List of your currently open orders.
func (h *HitBTC) GetOpenOrders(currency string) ([]OrderHistoryResponse, error) {
	values := url.Values{}
	values.Set("symbol", currency)
	var result []OrderHistoryResponse

	return result, h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet,
		apiv2OpenOrders,
		values,
		tradingRequests,
		&result)
}

// PlaceOrder places an order on the exchange
func (h *HitBTC) PlaceOrder(currency string, rate, amount float64, orderType, side string) (OrderResponse, error) {
	var result OrderResponse
	values := url.Values{}

	values.Set("symbol", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("quantity", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("side", side)
	values.Set("price", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("type", orderType)

	return result, h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		apiOrder,
		values,
		tradingRequests,
		&result)
}

// CancelExistingOrder cancels a specific order by OrderID
func (h *HitBTC) CancelExistingOrder(orderID int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}

	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodDelete,
		apiOrder+"/"+strconv.FormatInt(orderID, 10),
		values,
		tradingRequests,
		&result)

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
	return result, h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodDelete,
		apiOrder,
		values,
		tradingRequests,
		&result)
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

	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		orderMove,
		values,
		tradingRequests,
		&result)

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

	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		apiV2CryptoWithdraw,
		values,
		otherRequests,
		&result)

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
	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodGet,
		apiV2FeeInfo+"/"+currencyPair,
		url.Values{},
		tradingRequests,
		&result)

	return result, err
}

// GetTradableBalances returns current tradable balances
func (h *HitBTC) GetTradableBalances() (map[string]map[string]float64, error) {
	type Response struct {
		Data map[string]map[string]interface{}
	}
	result := Response{}

	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		tradableBalances,
		url.Values{},
		tradingRequests,
		&result.Data)

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

	err := h.SendAuthenticatedHTTPRequest(exchange.RestSpot, http.MethodPost,
		transferBalance,
		values,
		otherRequests,
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
func (h *HitBTC) SendHTTPRequest(ep exchange.URL, path string, result interface{}) error {
	endpoint, err := h.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return h.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       h.Verbose,
		HTTPDebugging: h.HTTPDebugging,
		HTTPRecording: h.HTTPRecording,
		Endpoint:      marketRequests,
	})
}

// SendAuthenticatedHTTPRequest sends an authenticated http request
func (h *HitBTC) SendAuthenticatedHTTPRequest(ep exchange.URL, method, endpoint string, values url.Values, f request.EndpointLimit, result interface{}) error {
	if !h.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", h.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	ePoint, err := h.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Basic " + crypto.Base64Encode([]byte(h.API.Credentials.Key+":"+h.API.Credentials.Secret))

	path := fmt.Sprintf("%s/%s", ePoint, endpoint)

	return h.SendPayload(context.Background(), &request.Item{
		Method:        method,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBufferString(values.Encode()),
		Result:        result,
		AuthRequest:   true,
		Verbose:       h.Verbose,
		HTTPDebugging: h.HTTPDebugging,
		HTTPRecording: h.HTTPRecording,
		Endpoint:      f,
	})
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
	case exchange.CryptocurrencyDepositFee:
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
