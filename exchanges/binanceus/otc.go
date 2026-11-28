package binanceus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var errQuoteIDRequired = errors.New("quote ID required")

// GetSupportedCoinPairs returns the coin pairs over-the-counter orders convert between, filtered on the coin converted
// from and the coin converted to when they are set
func (e *Exchange) GetSupportedCoinPairs(ctx context.Context, fromCoin, toCoin currency.Code) ([]CoinPairResponse, error) {
	params := url.Values{}
	if !fromCoin.IsEmpty() {
		params.Set("fromCoin", fromCoin.Upper().String())
	}
	if !toCoin.IsEmpty() {
		params.Set("toCoin", toCoin.Upper().String())
	}
	var resp []CoinPairResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/otc/coinPairs", params, defaultRate, &resp)
}

// RequestForQuote requests a quote to convert one coin to another. The quote carries the quote ID
// PlaceOverTheCounterTradeOrder takes only when the account holds enough of the coin it converts from; otherwise it is
// informational
func (e *Exchange) RequestForQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.FromCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: fromCoin", currency.ErrCurrencyCodeEmpty)
	}
	if req.ToCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: toCoin", currency.ErrCurrencyCodeEmpty)
	}
	if req.RequestCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: requestCoin", currency.ErrCurrencyCodeEmpty)
	}
	if req.RequestAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("fromCoin", req.FromCoin.Upper().String())
	params.Set("toCoin", req.ToCoin.Upper().String())
	params.Set("requestCoin", req.RequestCoin.Upper().String())
	params.Set("requestAmount", formatFloat(req.RequestAmount))
	var resp *QuoteResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/otc/quotes", params, defaultRate, &resp)
}

// PlaceOverTheCounterTradeOrder places an over-the-counter order from a quote. A placed order settles afterwards, so
// GetOverTheCounterTradeOrder reports whether it succeeded; a quote that cannot be placed, such as an expired one, is
// refused with an error
func (e *Exchange) PlaceOverTheCounterTradeOrder(ctx context.Context, quoteID string) (*PlaceOverTheCounterTradeOrderResponse, error) {
	if quoteID == "" {
		return nil, errQuoteIDRequired
	}
	params := url.Values{}
	params.Set("quoteId", quoteID)
	var resp *PlaceOverTheCounterTradeOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/otc/orders", params, defaultRate, &resp)
}

// GetOverTheCounterTradeOrder returns an over-the-counter order
func (e *Exchange) GetOverTheCounterTradeOrder(ctx context.Context, orderID string) (*OverTheCounterTradeOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OverTheCounterTradeOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/otc/orders/"+url.PathEscape(orderID), nil, defaultRate, &resp)
}

// GetAllOverTheCounterTradeOrders returns a page of the account's over-the-counter orders. A nil request returns the
// first page of 10 orders
func (e *Exchange) GetAllOverTheCounterTradeOrders(ctx context.Context, req *OverTheCounterTradeOrdersRequest) (*OverTheCounterTradeOrdersResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.OrderID != "" {
			params.Set("orderId", req.OrderID)
		}
		if !req.FromCoin.IsEmpty() {
			params.Set("fromCoin", req.FromCoin.Upper().String())
		}
		if !req.ToCoin.IsEmpty() {
			params.Set("toCoin", req.ToCoin.Upper().String())
		}
		if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		if req.Page > 0 {
			params.Set("page", strconv.FormatUint(req.Page, 10))
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
	}
	var resp *OverTheCounterTradeOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/otc/orders", params, defaultRate, &resp)
}

// GetAllOneClickBuySellTradeOrders returns a page of the account's One Click Buy/Sell orders, which buy crypto with
// fiat or sell it for fiat. A nil request returns the first page of 10 orders
func (e *Exchange) GetAllOneClickBuySellTradeOrders(ctx context.Context, req *OneClickBuySellTradeOrdersRequest) (*OneClickBuySellTradeOrdersResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.OrderID != "" {
			params.Set("orderId", req.OrderID)
		}
		if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
		if req.Page > 0 {
			params.Set("page", strconv.FormatUint(req.Page, 10))
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
	}
	var resp *OneClickBuySellTradeOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/ocbs/orders", params, defaultRate, &resp)
}
