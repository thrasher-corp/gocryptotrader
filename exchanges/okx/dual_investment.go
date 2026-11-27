package okx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetDualInvestmentCurrencyPairs returns the currency pairs dual investment products are offered on
func (e *Exchange) GetDualInvestmentCurrencyPairs(ctx context.Context) ([]DualInvestmentCurrencyPairResponse, error) {
	var resp []DualInvestmentCurrencyPairResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getDualInvestmentCurrencyPairsEPL, http.MethodGet, "finance/sfp/dcd/currency-pair", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetDualInvestmentProducts returns the dual investment products of a currency pair and option type
func (e *Exchange) GetDualInvestmentProducts(ctx context.Context, arg *DualInvestmentProductsRequest) ([]DualInvestmentProductResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.BaseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: baseCcy", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: quoteCcy", currency.ErrCurrencyCodeEmpty)
	}
	if arg.OptionType == "" {
		return nil, fmt.Errorf("%w: optType", errMissingRequiredParameter)
	}
	params := url.Values{}
	params.Set("baseCcy", arg.BaseCurrency.String())
	params.Set("quoteCcy", arg.QuoteCurrency.String())
	params.Set("optType", arg.OptionType)
	var resp []DualInvestmentProductResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getDualInvestmentProductsEPL, http.MethodGet, common.EncodeURLValues("finance/sfp/dcd/products", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// RequestDualInvestmentQuote requests a quote for investing in a dual investment product, which expires at its
// ValidUntil time
func (e *Exchange) RequestDualInvestmentQuote(ctx context.Context, arg *DualInvestmentQuoteRequest) (*DualInvestmentQuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ProductID == "" {
		return nil, fmt.Errorf("%w: productId", errMissingRequiredParameter)
	}
	if arg.NotionalSize <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.NotionalCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: notionalCcy", currency.ErrCurrencyCodeEmpty)
	}
	var resp *DualInvestmentQuoteResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, requestDualInvestmentQuoteEPL, http.MethodPost, "finance/sfp/dcd/quote", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// TradeDualInvestment places a dual investment order with an unexpired quote
func (e *Exchange) TradeDualInvestment(ctx context.Context, quoteID string) (*DualInvestmentTradeResponse, error) {
	if quoteID == "" {
		return nil, fmt.Errorf("%w: quoteId", errMissingRequiredParameter)
	}
	var resp *DualInvestmentTradeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradeDualInvestmentEPL, http.MethodPost, "finance/sfp/dcd/trade", &DualInvestmentTradeRequest{QuoteID: quoteID}, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// RequestDualInvestmentRedeemQuote requests an early redemption quote for a live dual investment order, the first step
// of an early redemption
func (e *Exchange) RequestDualInvestmentRedeemQuote(ctx context.Context, orderID string) (*DualInvestmentRedeemQuoteResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *DualInvestmentRedeemQuoteResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, requestDualInvestmentRedeemQuoteEPL, http.MethodPost, "finance/sfp/dcd/redeem-quote", &DualInvestmentRedeemQuoteRequest{OrderID: orderID}, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// RedeemDualInvestment redeems a dual investment order early with an unexpired redeem quote, the second step of an early
// redemption
func (e *Exchange) RedeemDualInvestment(ctx context.Context, arg *DualInvestmentRedeemRequest) (*DualInvestmentRedeemResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.QuoteID == "" {
		return nil, fmt.Errorf("%w: quoteId", errMissingRequiredParameter)
	}
	var resp *DualInvestmentRedeemResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, redeemDualInvestmentEPL, http.MethodPost, "finance/sfp/dcd/redeem", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetDualInvestmentOrderState returns a dual investment order's state
func (e *Exchange) GetDualInvestmentOrderState(ctx context.Context, orderID string) (*DualInvestmentOrderStateResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordId", orderID)
	var resp *DualInvestmentOrderStateResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getDualInvestmentOrderStateEPL, http.MethodGet, common.EncodeURLValues("finance/sfp/dcd/order-status", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetDualInvestmentOrderHistory returns the account's dual investment orders; OKX ignores the other filters when an
// order ID is set
func (e *Exchange) GetDualInvestmentOrderHistory(ctx context.Context, arg *DualInvestmentOrderHistoryRequest) ([]DualInvestmentOrderHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if !arg.Begin.IsZero() && !arg.End.IsZero() && arg.Begin.After(arg.End) {
		return nil, fmt.Errorf("%w: begin %s is after end %s", common.ErrStartAfterEnd, arg.Begin, arg.End)
	}
	params := url.Values{}
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	if arg.ProductID != "" {
		params.Set("productId", arg.ProductID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if !arg.Begin.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []DualInvestmentOrderHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getDualInvestmentOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/sfp/dcd/order-history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}
