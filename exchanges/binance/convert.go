package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errExactlyOneAmountRequired = errors.New("exactly one of the two amounts is required")

// GetAllConvertPairs calls List All Convert Pairs, returning the convertible pairs and their limits; at least one of
// fromAsset and toAsset is required, and with only one the API returns just the pairs that asset is part of
func (e *Exchange) GetAllConvertPairs(ctx context.Context, fromAsset, toAsset currency.Code) ([]ConvertPair, error) {
	if fromAsset.IsEmpty() && toAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: fromAsset or toAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if !fromAsset.IsEmpty() {
		params.Set("fromAsset", fromAsset.Upper().String())
	}
	if !toAsset.IsEmpty() {
		params.Set("toAsset", toAsset.Upper().String())
	}
	var resp []ConvertPair
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/sapi/v1/convert/exchangeInfo", params), getAllConvertPairsRate, &resp)
}

// GetOrderQuantityPrecisionPerAsset calls Query order quantity precision per asset
func (e *Exchange) GetOrderQuantityPrecisionPerAsset(ctx context.Context) ([]OrderQuantityPrecision, error) {
	var resp []OrderQuantityPrecision
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/convert/assetInfo", nil, getOrderQuantityPrecisionPerAssetRate, &resp)
}

// AcceptQuote calls Accept Quote, converting at a quote that SendQuoteRequest returned
func (e *Exchange) AcceptQuote(ctx context.Context, quoteID string) (*AcceptQuoteResponse, error) {
	if quoteID == "" {
		return nil, errQuoteIDRequired
	}
	params := url.Values{}
	params.Set("quoteId", quoteID)
	var resp *AcceptQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/convert/acceptQuote", params, acceptQuoteRate, &resp)
}

// CancelLimitOrder calls Cancel limit order with an order ID that PlaceLimitOrder returned
func (e *Exchange) CancelLimitOrder(ctx context.Context, orderID uint64) (*ConvertLimitOrderResponse, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatUint(orderID, 10))
	var resp *ConvertLimitOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/convert/limit/cancelOrder", params, cancelLimitOrderRate, &resp)
}

// GetConvertTradeHistory calls Get Convert Trade History for a window of at most 30 days
func (e *Exchange) GetConvertTradeHistory(ctx context.Context, startTime, endTime time.Time, limit uint64) (*ConvertTradeHistoryResponse, error) {
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *ConvertTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/convert/tradeFlow", params, convertTradeFlowHistoryRate, &resp)
}

// GetConvertOrderStatus calls Order status for an order ID or a quote ID; at least one is required
func (e *Exchange) GetConvertOrderStatus(ctx context.Context, orderID, quoteID string) (*ConvertOrderStatusResponse, error) {
	if orderID == "" && quoteID == "" {
		return nil, fmt.Errorf("%w: orderId or quoteId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if quoteID != "" {
		params.Set("quoteId", quoteID)
	}
	var resp *ConvertOrderStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/convert/orderStatus", params, convertOrderStatusRate, &resp)
}

// PlaceLimitOrder calls Place limit order
func (e *Exchange) PlaceLimitOrder(ctx context.Context, arg *ConvertLimitOrderRequest) (*ConvertLimitOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.BaseAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: baseAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: quoteAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.LimitPrice <= 0 {
		return nil, fmt.Errorf("%w: limitPrice is required", limits.ErrPriceBelowMin)
	}
	if (arg.BaseAmount > 0) == (arg.QuoteAmount > 0) {
		return nil, fmt.Errorf("%w: baseAmount or quoteAmount", errExactlyOneAmountRequired)
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.ExpiredType == "" {
		return nil, errExpiredTypeRequired
	}
	params := url.Values{}
	params.Set("baseAsset", arg.BaseAsset.Upper().String())
	params.Set("quoteAsset", arg.QuoteAsset.Upper().String())
	params.Set("limitPrice", strconv.FormatFloat(arg.LimitPrice, 'f', -1, 64))
	if arg.BaseAmount > 0 {
		params.Set("baseAmount", strconv.FormatFloat(arg.BaseAmount, 'f', -1, 64))
	}
	if arg.QuoteAmount > 0 {
		params.Set("quoteAmount", strconv.FormatFloat(arg.QuoteAmount, 'f', -1, 64))
	}
	params.Set("side", arg.Side)
	if arg.WalletType != "" {
		params.Set("walletType", arg.WalletType)
	}
	params.Set("expiredType", arg.ExpiredType)
	var resp *ConvertLimitOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/convert/limit/placeOrder", params, placeLimitOrderRate, &resp)
}

// GetLimitOpenOrders calls Query limit open orders
func (e *Exchange) GetLimitOpenOrders(ctx context.Context) (*ConvertLimitOpenOrdersResponse, error) {
	var resp *ConvertLimitOpenOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/convert/limit/queryOpenOrders", nil, getLimitOpenOrdersRate, &resp)
}

// SendQuoteRequest calls Send Quote Request; the response carries a quote ID only when the account holds enough funds
// to convert
func (e *Exchange) SendQuoteRequest(ctx context.Context, arg *ConvertQuoteRequest) (*ConvertQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.FromAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: fromAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: toAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if (arg.FromAmount > 0) == (arg.ToAmount > 0) {
		return nil, fmt.Errorf("%w: fromAmount or toAmount", errExactlyOneAmountRequired)
	}
	params := url.Values{}
	params.Set("fromAsset", arg.FromAsset.Upper().String())
	params.Set("toAsset", arg.ToAsset.Upper().String())
	if arg.FromAmount > 0 {
		params.Set("fromAmount", strconv.FormatFloat(arg.FromAmount, 'f', -1, 64))
	}
	if arg.ToAmount > 0 {
		params.Set("toAmount", strconv.FormatFloat(arg.ToAmount, 'f', -1, 64))
	}
	if arg.WalletType != "" {
		params.Set("walletType", arg.WalletType)
	}
	if arg.ValidTime != "" {
		params.Set("validTime", arg.ValidTime)
	}
	var resp *ConvertQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/convert/getQuote", params, sendQuoteRequestRate, &resp)
}
