package coinbase

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetBestBidAsk returns the best bid and ask of each product named, or of every product when productIDs is empty, as
// a book holding the top level of each side
func (e *Exchange) GetBestBidAsk(ctx context.Context, productIDs []string) ([]PriceBook, error) {
	params := url.Values{}
	for _, id := range productIDs {
		params.Add("product_ids", id)
	}
	var resp *BestBidAskResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"best_bid_ask", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.PriceBooks, nil
}

// GetMarketTrades returns a product's latest trades, at most 100 whatever the limit, and its best bid and ask; it is
// GetPublicMarketTrades authenticated, so it counts against the API key's rate limit instead of the IP's
func (e *Exchange) GetMarketTrades(ctx context.Context, req *MarketTradesRequest) (*MarketTradesResponse, error) {
	params, err := marketTradesParams(req)
	if err != nil {
		return nil, err
	}
	var resp *MarketTradesResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"products/"+req.ProductID+"/ticker", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetProduct returns a product, with Watched set when it is on the user's watchlist and, when GetTradabilityStatus is
// set, a spot product's ViewOnly set when the account cannot trade it
func (e *Exchange) GetProduct(ctx context.Context, req *ProductRequest) (*ProductResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	params := url.Values{}
	if req.GetTradabilityStatus {
		params.Set("get_tradability_status", "true")
	}
	var resp *ProductResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"products/"+req.ProductID, params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetProductBook returns a product's book, with its last trade price, midpoint and spread; it is GetPublicProductBook
// authenticated, so it counts against the API key's rate limit instead of the IP's
func (e *Exchange) GetProductBook(ctx context.Context, req *ProductBookRequest) (*ProductBookResponse, error) {
	params, err := productBookParams(req)
	if err != nil {
		return nil, err
	}
	var resp *ProductBookResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"product_book", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetProductCandles returns a product's candles over a window spanning at most 350 of them, newest first; it is
// GetPublicProductCandles authenticated, so it counts against the API key's rate limit instead of the IP's
func (e *Exchange) GetProductCandles(ctx context.Context, req *ProductCandlesRequest) ([]Candle, error) {
	params, err := productCandlesParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CandlesResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"products/"+req.ProductID+"/candles", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Candles, nil
}

// ListProducts returns a page of the products Coinbase lists, spot by default, with each product's Watched set when it
// is on the user's watchlist and, when GetTradabilityStatus is set, a spot product's ViewOnly set when the account
// cannot trade it
func (e *Exchange) ListProducts(ctx context.Context, req *ListProductsRequest) (*ProductsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp *ProductsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"products", productsParams(req, true), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
