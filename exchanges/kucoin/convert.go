package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Convert request validation errors
var (
	errOneCurrencySizeRequired = errors.New("exactly one of from currency size and to currency size is required")
	errQuoteIDRequired         = errors.New("quote ID is required")
)

// GetConvertSymbol returns the trade size limits of converting one currency to another (Get Convert Symbol)
func (e *Exchange) GetConvertSymbol(ctx context.Context, arg *ConvertSymbolRequest) (*ConvertSymbolResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.FromCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: from currency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: to currency", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("fromCurrency", arg.FromCurrency.Upper().String())
	params.Set("toCurrency", arg.ToCurrency.Upper().String())
	if arg.OrderType != "" {
		params.Set("orderType", arg.OrderType)
	}
	var resp *ConvertSymbolResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getConvertSymbolEPL, common.EncodeURLValues("/v1/convert/symbol", params), &resp)
}

// GetConvertCurrencies returns the currencies that can be converted and their size limits (Get Convert Currencies)
func (e *Exchange) GetConvertCurrencies(ctx context.Context) (*ConvertCurrenciesResponse, error) {
	var resp *ConvertCurrenciesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getConvertCurrenciesEPL, "/v1/convert/currencies", &resp)
}

// GetConvertQuote returns a quote to place a convert market order with (Get Convert Quote)
func (e *Exchange) GetConvertQuote(ctx context.Context, arg *ConvertQuoteRequest) (*ConvertQuoteResponse, error) {
	params, err := convertQuoteParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *ConvertQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getConvertQuoteEPL, http.MethodGet, common.EncodeURLValues("/v1/convert/quote", params), nil, &resp)
}

// AddConvertOrder places a convert market order at a quote from GetConvertQuote (Add Convert Order)
func (e *Exchange) AddConvertOrder(ctx context.Context, arg *AddConvertOrderRequest) (*AddConvertOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if arg.QuoteID == "" {
		return nil, errQuoteIDRequired
	}
	body := make(map[string]any)
	body["clientOrderId"] = arg.ClientOrderID
	body["quoteId"] = arg.QuoteID
	if arg.AccountType != "" {
		body["accountType"] = arg.AccountType
	}
	var resp *AddConvertOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, addConvertOrderEPL, http.MethodPost, "/v1/convert/order", body, &resp)
}

// GetConvertOrderDetail returns a convert market order by its order ID or client order ID (Get Convert Order Detail)
func (e *Exchange) GetConvertOrderDetail(ctx context.Context, orderID, clientOrderID string) (*ConvertOrderDetailResponse, error) {
	params, err := convertOrderIDParams(orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *ConvertOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getConvertOrderDetailEPL, http.MethodGet, common.EncodeURLValues("/v1/convert/order/detail", params), nil, &resp)
}

// GetConvertOrderHistory returns a page of convert market orders (Get Convert Order History)
func (e *Exchange) GetConvertOrderHistory(ctx context.Context, arg *ConvertOrdersRequest) (*ConvertOrderHistoryResponse, error) {
	params, err := convertOrdersParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *ConvertOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getConvertOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/convert/order/history", params), nil, &resp)
}

// AddConvertLimitOrder places a convert limit order; its price must be at least the protection price
// GetConvertLimitQuote returns (Add Convert Limit Order)
func (e *Exchange) AddConvertLimitOrder(ctx context.Context, arg *AddConvertLimitOrderRequest) (*AddConvertOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if arg.FromCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: from currency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: to currency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FromCurrencySize <= 0 {
		return nil, fmt.Errorf("%w: from currency size", limits.ErrAmountBelowMin)
	}
	if arg.ToCurrencySize <= 0 {
		return nil, fmt.Errorf("%w: to currency size", limits.ErrAmountBelowMin)
	}
	body := make(map[string]any)
	body["clientOrderId"] = arg.ClientOrderID
	body["fromCurrency"] = arg.FromCurrency.Upper().String()
	body["toCurrency"] = arg.ToCurrency.Upper().String()
	body["fromCurrencySize"] = arg.FromCurrencySize
	body["toCurrencySize"] = arg.ToCurrencySize
	if arg.AccountType != "" {
		body["accountType"] = arg.AccountType
	}
	var resp *AddConvertOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, addConvertLimitOrderEPL, http.MethodPost, "/v1/convert/limit/order", body, &resp)
}

// GetConvertLimitQuote returns the protection price that a convert limit order's price must be at least (Get Convert
// Limit Quote)
func (e *Exchange) GetConvertLimitQuote(ctx context.Context, arg *ConvertQuoteRequest) (*ConvertLimitQuoteResponse, error) {
	params, err := convertQuoteParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *ConvertLimitQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getConvertLimitQuoteEPL, http.MethodGet, common.EncodeURLValues("/v1/convert/limit/quote", params), nil, &resp)
}

// GetConvertLimitOrderDetail returns a convert limit order by its order ID or client order ID (Get Convert Limit Order
// Detail)
func (e *Exchange) GetConvertLimitOrderDetail(ctx context.Context, orderID, clientOrderID string) (*ConvertLimitOrderDetailResponse, error) {
	params, err := convertOrderIDParams(orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *ConvertLimitOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getConvertLimitOrderDetailEPL, http.MethodGet, common.EncodeURLValues("/v1/convert/limit/order/detail", params), nil, &resp)
}

// GetConvertLimitOrders returns a page of active and historical convert limit orders (Get Convert Limit Orders)
func (e *Exchange) GetConvertLimitOrders(ctx context.Context, arg *ConvertOrdersRequest) (*ConvertLimitOrdersResponse, error) {
	params, err := convertOrdersParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *ConvertLimitOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getConvertLimitOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/convert/limit/orders", params), nil, &resp)
}

// CancelConvertLimitOrder cancels a convert limit order by its order ID or client order ID; KuCoin takes them in the
// request body even though the method is DELETE (Cancel Convert Limit Order)
func (e *Exchange) CancelConvertLimitOrder(ctx context.Context, orderID, clientOrderID string) error {
	if orderID == "" && clientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	body := make(map[string]any, 1)
	if orderID != "" {
		body["orderId"] = orderID
	}
	if clientOrderID != "" {
		body["clientOrderId"] = clientOrderID
	}
	var resp string
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelConvertLimitOrderEPL, http.MethodDelete, "/v1/convert/limit/order/cancel", body, &resp)
}

// convertQuoteParams checks a Get Convert Quote or Get Convert Limit Quote request and returns its query parameters
func convertQuoteParams(arg *ConvertQuoteRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.FromCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: from currency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: to currency", currency.ErrCurrencyCodeEmpty)
	}
	if (arg.FromCurrencySize != 0) == (arg.ToCurrencySize != 0) {
		return nil, errOneCurrencySizeRequired
	}
	params := url.Values{}
	params.Set("fromCurrency", arg.FromCurrency.Upper().String())
	params.Set("toCurrency", arg.ToCurrency.Upper().String())
	if arg.FromCurrencySize != 0 {
		params.Set("fromCurrencySize", strconv.FormatFloat(arg.FromCurrencySize, 'f', -1, 64))
	} else {
		params.Set("toCurrencySize", strconv.FormatFloat(arg.ToCurrencySize, 'f', -1, 64))
	}
	return params, nil
}

// convertOrderIDParams returns the query parameters that select a convert order by order ID or client order ID
func convertOrderIDParams(orderID, clientOrderID string) (url.Values, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clientOrderId", clientOrderID)
	}
	return params, nil
}

// convertOrdersParams returns the query parameters of a Get Convert Order History or Get Convert Limit Orders request
func convertOrdersParams(arg *ConvertOrdersRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRange(params, "startAt", "endAt", arg.StartAt, arg.EndAt); err != nil {
		return nil, err
	}
	if arg.Page != 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	return params, nil
}
