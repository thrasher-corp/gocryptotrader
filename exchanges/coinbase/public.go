package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

const (
	// maximumCandles is the most candles a candles request's window may span
	maximumCandles = 350
	// maximumMarketTradesLimit is the largest market trades limit Coinbase accepts; it returns at most 100 trades
	// whatever the limit
	maximumMarketTradesLimit = 1000
)

var (
	errInvalidGranularity  = errors.New("invalid candle granularity")
	errInvalidDuration     = errors.New("invalid duration")
	errCandleWindowUnset   = errors.New("candle window start and end must be set")
	errCandleWindowTooLong = errors.New("candle window spans more than 350 candles")
)

// GetPublicMarketTrades returns a product's latest trades, at most 100, and its best bid and ask; Coinbase caches
// public responses for a second
func (e *Exchange) GetPublicMarketTrades(ctx context.Context, req *MarketTradesRequest) (*MarketTradesResponse, error) {
	params, err := marketTradesParams(req)
	if err != nil {
		return nil, err
	}
	var resp *MarketTradesResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"market/products/"+req.ProductID+"/ticker", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPublicProduct returns a product
func (e *Exchange) GetPublicProduct(ctx context.Context, productID string) (*ProductResponse, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	var resp *ProductResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"market/products/"+productID, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPublicProductBook returns a product's book, 1000 levels a side by default and at most
func (e *Exchange) GetPublicProductBook(ctx context.Context, req *ProductBookRequest) (*ProductBookResponse, error) {
	params, err := productBookParams(req)
	if err != nil {
		return nil, err
	}
	var resp *ProductBookResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"market/product_book", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPublicProductCandles returns a product's candles over a window spanning at most 350 of them, newest first
func (e *Exchange) GetPublicProductCandles(ctx context.Context, req *ProductCandlesRequest) ([]Candle, error) {
	params, err := productCandlesParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CandlesResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"market/products/"+req.ProductID+"/candles", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Candles, nil
}

// GetPublicServerTime returns the Advanced Trade API's time
func (e *Exchange) GetPublicServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	var resp *ServerTimeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"time", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ListPublicProducts returns a page of the products Coinbase lists, spot by default
func (e *Exchange) ListPublicProducts(ctx context.Context, req *ListProductsRequest) (*ProductsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp *ProductsResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"market/products", productsParams(req, false), &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// productsParams returns a product list request's query; only the authenticated list takes the tradability flag
func productsParams(req *ListProductsRequest, authenticated bool) url.Values {
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.ProductType != "" {
		params.Set("product_type", req.ProductType)
	}
	for _, id := range req.ProductIDs {
		params.Add("product_ids", id)
	}
	if req.ContractExpiryType != "" {
		params.Set("contract_expiry_type", req.ContractExpiryType)
	}
	if req.ExpiringContractStatus != "" {
		params.Set("expiring_contract_status", req.ExpiringContractStatus)
	}
	if req.GetAllProducts {
		params.Set("get_all_products", "true")
	}
	if req.ProductsSortOrder != "" {
		params.Set("products_sort_order", req.ProductsSortOrder)
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.FuturesUnderlyingType != "" {
		params.Set("futures_underlying_type", req.FuturesUnderlyingType)
	}
	if req.UserCountryCode != "" {
		params.Set("user_country_code", req.UserCountryCode)
	}
	if req.ProductVenue != "" {
		params.Set("product_venue", req.ProductVenue)
	}
	if req.Expired {
		params.Set("expired", "true")
	}
	if authenticated && req.GetTradabilityStatus {
		params.Set("get_tradability_status", "true")
	}
	return params
}

// productBookParams validates a book request and returns its query
func productBookParams(req *ProductBookRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.AggregationPriceIncrement < 0 {
		return nil, fmt.Errorf("%w: aggregation price increment %v", errInvalidPrice, req.AggregationPriceIncrement)
	}
	params := url.Values{}
	params.Set("product_id", req.ProductID)
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.AggregationPriceIncrement != 0 {
		params.Set("aggregation_price_increment", formatFloat(req.AggregationPriceIncrement))
	}
	return params, nil
}

// productCandlesParams validates a candles request and returns its query
func productCandlesParams(req *ProductCandlesRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.Start.IsZero() || req.End.IsZero() {
		return nil, errCandleWindowUnset
	}
	if err := common.StartEndTimeCheck(req.Start, req.End); err != nil {
		return nil, err
	}
	interval, err := candleDuration(req.Granularity)
	if err != nil {
		return nil, err
	}
	// Coinbase refuses a window spanning more candles than a response holds, whatever the limit
	if req.End.Sub(req.Start) > maximumCandles*interval {
		return nil, fmt.Errorf("%w: %s to %s at %s", errCandleWindowTooLong, req.Start, req.End, req.Granularity)
	}
	if req.Limit > maximumCandles {
		return nil, fmt.Errorf("%w: %d candles, at most %d", errInvalidLimit, req.Limit, maximumCandles)
	}
	params := url.Values{}
	params.Set("start", strconv.FormatInt(req.Start.Unix(), 10))
	params.Set("end", strconv.FormatInt(req.End.Unix(), 10))
	params.Set("granularity", req.Granularity)
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}

// marketTradesParams validates a market trades request and returns its query
func marketTradesParams(req *MarketTradesRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if req.Limit == 0 || req.Limit > maximumMarketTradesLimit {
		return nil, fmt.Errorf("%w: %d trades, want 1 to %d", errInvalidLimit, req.Limit, maximumMarketTradesLimit)
	}
	if err := checkWindow(req.Start, req.End); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("limit", strconv.FormatUint(req.Limit, 10))
	if !req.Start.IsZero() {
		params.Set("start", strconv.FormatInt(req.Start.Unix(), 10))
	}
	if !req.End.IsZero() {
		params.Set("end", strconv.FormatInt(req.End.Unix(), 10))
	}
	return params, nil
}

// candleDuration returns a granularity's interval
func candleDuration(granularity string) (time.Duration, error) {
	switch granularity {
	case GranularityOneMinute:
		return time.Minute, nil
	case GranularityFiveMinute:
		return 5 * time.Minute, nil
	case GranularityFifteenMinute:
		return 15 * time.Minute, nil
	case GranularityThirtyMinute:
		return 30 * time.Minute, nil
	case GranularityOneHour:
		return time.Hour, nil
	case GranularityTwoHour:
		return 2 * time.Hour, nil
	case GranularityFourHour:
		return 4 * time.Hour, nil
	case GranularitySixHour:
		return 6 * time.Hour, nil
	case GranularityOneDay:
		return 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("%w: %q", errInvalidGranularity, granularity)
}
