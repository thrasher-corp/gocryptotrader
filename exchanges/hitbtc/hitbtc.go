package hitbtc

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// API
	apiURL       = "https://api.hitbtc.com"
	tradeBaseURL = "https://hitbtc.com/"
	tradeFutures = "futures/"

	// Public
	apiV2Trades    = "/api/2/public/trades"
	apiV2Currency  = "/api/2/public/currency"
	apiV2Symbol    = "/api/2/public/symbol"
	apiV2Ticker    = "/api/2/public/ticker"
	apiV2Orderbook = "/api/2/public/orderbook"
	apiV2Candles   = "/api/2/public/candles"

	// Authenticated
	apiV2Balance        = "api/2/trading/balance"
	apiV2CryptoAddress  = "api/2/account/crypto/address"
	apiV2CryptoWithdraw = "api/2/account/crypto/withdraw"
	apiV2TradeHistory   = "api/2/history/trades"
	apiV2OrderHistory   = "api/2/history/order"
	apiv2OpenOrders     = "api/2/order"
	apiV2FeeInfo        = "api/2/trading/fee"
	apiOrder            = "api/2/order"
)

// HitBTC is the overarching type across the hitbtc package
type HitBTC struct {
	exchange.Base
}

// Public Market Data
// https://api.hitbtc.com/?python#market-data

// GetCurrencies returns the actual list of available currencies, tokens, ICO
// etc.
func (h *HitBTC) GetCurrencies(ctx context.Context) (map[string]Currencies, error) {
	type Response struct {
		Data []Currencies
	}
	resp := Response{}
	ret := make(map[string]Currencies)
	err := h.SendHTTPRequest(ctx, exchange.RestSpot, apiV2Currency, &resp.Data)
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
func (h *HitBTC) GetCurrency(ctx context.Context, currency string) (Currencies, error) {
	type Response struct {
		Data Currencies
	}
	resp := Response{}
	path := apiV2Currency + "/" + currency

	return resp.Data, h.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp.Data)
}

// GetSymbols Return the actual list of currency symbols (currency pairs) traded
// on HitBTC exchange. The first listed currency of a symbol is called the base
// currency, and the second currency is called the quote currency. The currency
// pair indicates how much of the quote currency is needed to purchase one unit
// of the base currency.
func (h *HitBTC) GetSymbols(ctx context.Context, symbol string) ([]string, error) {
	var resp []Symbol
	path := apiV2Symbol + "/" + symbol

	ret := make([]string, 0, len(resp))
	err := h.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
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
func (h *HitBTC) GetSymbolsDetailed(ctx context.Context) ([]Symbol, error) {
	var resp []Symbol
	return resp, h.SendHTTPRequest(ctx, exchange.RestSpot, apiV2Symbol, &resp)
}

// GetTicker returns ticker information
func (h *HitBTC) GetTicker(ctx context.Context, symbol string) (TickerResponse, error) {
	var resp TickerResponse
	path := apiV2Ticker + "/" + symbol
	return resp, h.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetTickers returns ticker information
func (h *HitBTC) GetTickers(ctx context.Context) ([]TickerResponse, error) {
	var resp []TickerResponse
	return resp, h.SendHTTPRequest(ctx, exchange.RestSpot, apiV2Ticker, &resp)
}

// GetTrades returns trades from hitbtc
func (h *HitBTC) GetTrades(ctx context.Context, currencyPair, by, sort string, from, till, limit, offset int64) ([]TradeHistory, error) {
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
	path := common.EncodeURLValues(apiV2Trades+"/"+currencyPair, urlValues)
	return resp, h.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetOrderbook an order book is an electronic list of buy and sell orders for a
// specific symbol, organized by price level.
func (h *HitBTC) GetOrderbook(ctx context.Context, currencyPair string, limit int) (*Orderbook, error) {
	// limit Limit of orderbook levels, default 100. Set 0 to view full orderbook levels
	vals := url.Values{}

	if limit != 0 {
		vals.Set("limit", strconv.Itoa(limit))
	}

	var resp Orderbook
	path := common.EncodeURLValues(apiV2Orderbook+"/"+currencyPair, vals)
	err := h.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCandles returns candles which is used for OHLC a specific currency.
// Note: Result contain candles only with non zero volume.
func (h *HitBTC) GetCandles(ctx context.Context, currencyPair, limit, period string, start, end time.Time) ([]ChartData, error) {
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
	path := common.EncodeURLValues(apiV2Candles+"/"+currencyPair, vals)
	return resp, h.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// Authenticated Market Data
// https://api.hitbtc.com/?python#market-data

// GetBalances returns full balance for your account
func (h *HitBTC) GetBalances(ctx context.Context) (map[string]Balance, error) {
	var result []Balance
	err := h.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodGet,
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
func (h *HitBTC) GetDepositAddresses(ctx context.Context, currency string) (DepositCryptoAddresses, error) {
	var resp DepositCryptoAddresses

	return resp,
		h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
			apiV2CryptoAddress+"/"+currency,
			url.Values{},
			otherRequests,
			&resp)
}

// GenerateNewAddress generates a new deposit address for a currency
func (h *HitBTC) GenerateNewAddress(ctx context.Context, currency string) (DepositCryptoAddresses, error) {
	resp := DepositCryptoAddresses{}
	err := h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		apiV2CryptoAddress+"/"+currency,
		url.Values{},
		otherRequests,
		&resp)

	return resp, err
}

// GetTradeHistoryForCurrency returns your trade history
func (h *HitBTC) GetTradeHistoryForCurrency(ctx context.Context, currency, start, end string) (AuthenticatedTradeHistoryResponse, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if end != "" {
		values.Set("end", end)
	}

	values.Set("currencyPair", currency)
	result := AuthenticatedTradeHistoryResponse{}

	return result, h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		apiV2TradeHistory,
		values,
		otherRequests,
		&result.Data)
}

// GetTradeHistoryForAllCurrencies returns your trade history
func (h *HitBTC) GetTradeHistoryForAllCurrencies(ctx context.Context, start, end string) (AuthenticatedTradeHistoryAll, error) {
	values := url.Values{}

	if start != "" {
		values.Set("start", start)
	}

	if end != "" {
		values.Set("end", end)
	}

	values.Set("currencyPair", "all")
	result := AuthenticatedTradeHistoryAll{}

	return result, h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		apiV2TradeHistory,
		values,
		otherRequests,
		&result.Data)
}

// GetOrders List of your order history.
func (h *HitBTC) GetOrders(ctx context.Context, currency string) ([]OrderHistoryResponse, error) {
	values := url.Values{}
	values.Set("symbol", currency)
	var result []OrderHistoryResponse

	return result, h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		apiV2OrderHistory,
		values,
		tradingRequests,
		&result)
}

// GetOpenOrders List of your currently open orders.
func (h *HitBTC) GetOpenOrders(ctx context.Context, currency string) ([]OrderHistoryResponse, error) {
	values := url.Values{}
	values.Set("symbol", currency)
	var result []OrderHistoryResponse

	return result, h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		apiv2OpenOrders,
		values,
		tradingRequests,
		&result)
}

// GetActiveOrderByClientOrderID Get an active order by id
func (h *HitBTC) GetActiveOrderByClientOrderID(ctx context.Context, clientOrderID string) (OrderHistoryResponse, error) {
	var result OrderHistoryResponse

	return result, h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		apiv2OpenOrders+"/"+clientOrderID,
		nil,
		tradingRequests,
		&result)
}

// PlaceOrder places an order on the exchange
func (h *HitBTC) PlaceOrder(ctx context.Context, currency string, rate, amount float64, orderType, side string) (OrderResponse, error) {
	var result OrderResponse
	values := url.Values{}

	values.Set("symbol", currency)
	values.Set("rate", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("quantity", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("side", side)
	values.Set("price", strconv.FormatFloat(rate, 'f', -1, 64))
	values.Set("type", orderType)

	return result, h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
		apiOrder,
		values,
		tradingRequests,
		&result)
}

// CancelExistingOrder cancels a specific order by OrderID
func (h *HitBTC) CancelExistingOrder(ctx context.Context, orderID int64) (bool, error) {
	result := GenericResponse{}
	values := url.Values{}

	err := h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
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
func (h *HitBTC) CancelAllExistingOrders(ctx context.Context) ([]Order, error) {
	var result []Order
	values := url.Values{}
	return result, h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete,
		apiOrder,
		values,
		tradingRequests,
		&result)
}

// Withdraw allows for the withdrawal to a specific address
func (h *HitBTC) Withdraw(ctx context.Context, currency, address string, amount float64) (bool, error) {
	result := Withdraw{}
	values := url.Values{}

	values.Set("currency", currency)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	values.Set("address", address)

	err := h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost,
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
func (h *HitBTC) GetFeeInfo(ctx context.Context, currencyPair string) (Fee, error) {
	result := Fee{}
	err := h.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		apiV2FeeInfo+"/"+currencyPair,
		url.Values{},
		tradingRequests,
		&result)

	return result, err
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (h *HitBTC) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := h.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       h.Verbose,
		HTTPDebugging: h.HTTPDebugging,
		HTTPRecording: h.HTTPRecording,
	}

	return h.SendPayload(ctx, marketRequests, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAuthenticatedHTTPRequest sends an authenticated http request
func (h *HitBTC) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, values url.Values, f request.EndpointLimit, result any) error {
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := h.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(creds.Key+":"+creds.Secret))

	item := &request.Item{
		Method:        method,
		Path:          ePoint + "/" + endpoint,
		Headers:       headers,
		Result:        result,
		Verbose:       h.Verbose,
		HTTPDebugging: h.HTTPDebugging,
		HTTPRecording: h.HTTPRecording,
	}

	return h.SendPayload(ctx, f, func() (*request.Item, error) {
		item.Body = bytes.NewBufferString(values.Encode())
		return item, nil
	}, request.AuthenticatedRequest)
}

// GetFee returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feeInfo, err := h.GetFeeInfo(ctx,
			feeBuilder.Pair.Base.String()+
				feeBuilder.Pair.Delimiter+
				feeBuilder.Pair.Quote.String())
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeInfo, feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		currencyInfo, err := h.GetCurrency(ctx, feeBuilder.Pair.Base.String())
		if err != nil {
			return 0, err
		}
		fee = currencyInfo.PayoutFee.Float64()
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
	if c.Equal(currency.BTC) {
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
