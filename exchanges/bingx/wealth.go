package bingx

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errDualCurrencySKURequired         = errors.New("SKU is required")
	errDualCurrencyProductIDRequired   = errors.New("product ID is required")
	errDualCurrencyStrikePriceRequired = errors.New("strike price is required")
	errDualCurrencyQuoteIDRequired     = errors.New("quote ID is required")
)

// GetDualCurrencyProducts returns the dual-currency products, from Dual-Currency Product List
func (e *Exchange) GetDualCurrencyProducts(ctx context.Context, arg *DualCurrencyProductsRequest) (*DualCurrencyProductsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	if arg.BeginDuration > 0 {
		params.Set("beginDuration", strconv.FormatUint(arg.BeginDuration, 10))
	}
	if arg.EndDuration > 0 {
		params.Set("endDuration", strconv.FormatUint(arg.EndDuration, 10))
	}
	if arg.Duration > 0 {
		params.Set("duration", strconv.FormatUint(arg.Duration, 10))
	}
	if !arg.InvestmentAsset.IsEmpty() {
		params.Set("investmentAsset", arg.InvestmentAsset.Upper().String())
	}
	if !arg.ExerciseAsset.IsEmpty() {
		params.Set("exerciseAsset", arg.ExerciseAsset.Upper().String())
	}
	if arg.Direction != "" {
		params.Set("direction", string(arg.Direction))
	}
	var resp *DualCurrencyProductsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/wealth/v1/product/dual-currency/invest-asset-list", params, dualCurrencyProductsRate, &resp)
}

// GetDualCurrencyPreOrderQuote returns a dual-currency product's quote, which an order needs, from Dual-Currency
// Pre-Order Quote
func (e *Exchange) GetDualCurrencyPreOrderQuote(ctx context.Context, arg *DualCurrencyPreOrderQuoteRequest) (*DualCurrencyPreOrderQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SKU == "" {
		return nil, errDualCurrencySKURequired
	}
	if arg.StrikePrice <= 0 {
		return nil, errDualCurrencyStrikePriceRequired
	}
	if arg.ProductID == 0 {
		return nil, errDualCurrencyProductIDRequired
	}
	params := url.Values{}
	params.Set("sku", arg.SKU)
	params.Set("strikePrice", strconv.FormatFloat(arg.StrikePrice, 'f', -1, 64))
	params.Set("productId", strconv.FormatUint(arg.ProductID, 10))
	var resp *DualCurrencyPreOrderQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wealth/v1/product/dual-currency/pre-order", params, dualCurrencyPreOrderQuoteRate, &resp)
}

// PlaceDualCurrencyOrder places a dual-currency order for a quote from GetDualCurrencyPreOrderQuote, from
// Dual-Currency Place Order. The reply's BusinessCode reports whether BingX handled the order normally, and its Message
// why not
func (e *Exchange) PlaceDualCurrencyOrder(ctx context.Context, arg *DualCurrencyOrderRequest) (*DualCurrencyOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.SKU == "" {
		return nil, errDualCurrencySKURequired
	}
	if arg.ProductID == 0 {
		return nil, errDualCurrencyProductIDRequired
	}
	if arg.StrikePrice <= 0 {
		return nil, errDualCurrencyStrikePriceRequired
	}
	if arg.QuoteID == "" {
		return nil, errDualCurrencyQuoteIDRequired
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("sku", arg.SKU)
	params.Set("productId", strconv.FormatUint(arg.ProductID, 10))
	params.Set("strikePrice", strconv.FormatFloat(arg.StrikePrice, 'f', -1, 64))
	params.Set("quoteId", arg.QuoteID)
	if !arg.InvestmentAsset.IsEmpty() {
		params.Set("investmentAsset", arg.InvestmentAsset.Upper().String())
	}
	if !arg.ExerciseAsset.IsEmpty() {
		params.Set("exerciseAsset", arg.ExerciseAsset.Upper().String())
	}
	if len(arg.SelectAccounts) > 0 {
		// BingX takes structured query values as JSON
		accounts, err := json.Marshal(arg.SelectAccounts)
		if err != nil {
			return nil, err
		}
		params.Set("selectAccounts", string(accounts))
	}
	if arg.SettleTargetAccount != "" {
		params.Set("settleTargetAccount", string(arg.SettleTargetAccount))
	}
	var resp *DualCurrencyOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/wealth/v1/product/dual-currency/order", params, dualCurrencyPlaceOrderRate, &resp)
}

// GetDualCurrencyPositions returns the dual-currency positions, or with an order number that order's, from
// Dual-Currency Position Query
func (e *Exchange) GetDualCurrencyPositions(ctx context.Context, orderNumber string) ([]DualCurrencyPositionResponse, error) {
	params := url.Values{}
	if orderNumber != "" {
		params.Set("orderNo", orderNumber)
	}
	var resp []DualCurrencyPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wealth/v1/product/dual-currency/position", params, dualCurrencyPositionsRate, &resp)
}

// GetDualCurrencyOrderRecords returns a page of the dual-currency orders, from Dual-Currency Order Records
func (e *Exchange) GetDualCurrencyOrderRecords(ctx context.Context, arg *DualCurrencyOrderRecordsRequest) (*DualCurrencyOrderRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PageID == 0 {
		return nil, errPageIndexRequired
	}
	if arg.PageSize == 0 {
		return nil, errPageSizeRequired
	}
	params := url.Values{}
	params.Set("pageId", strconv.FormatUint(arg.PageID, 10))
	params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	if arg.OrderNumber != "" {
		params.Set("orderNo", arg.OrderNumber)
	}
	var resp *DualCurrencyOrderRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/wealth/v1/product/dual-currency/order-records", params, dualCurrencyOrderRecordsRate, &resp)
}
