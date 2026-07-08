package htx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	// USDT-margined futures endpoints.
	linearSwapMarkets        = "/linear-swap-api/v1/swap_contract_info"
	linearSwapMarketDepth    = "/linear-swap-ex/market/depth"
	linearSwapMarketOverview = "/linear-swap-ex/market/detail/merged"
	linearSwapFunding        = "/linear-swap-api/v1/swap_funding_rate"
	linearSwapBatchFunding   = "/linear-swap-api/v1/swap_batch_funding_rate"
	v5AccountBalance         = "/v5/account/balance"
	v5TradeOrder             = "/v5/trade/order"
	v5TradeCancelOrder       = "/v5/trade/cancel_order"
	v5TradeCancelAllOrders   = "/v5/trade/cancel_all_orders"
	v5TradeOrderOpens        = "/v5/trade/order/opens"
	v5MarketOpenInterest     = "/v5/market/open_interest"
)

// GetLinearSwapMarkets gets current USDT-margined contract metadata.
func (e *Exchange) GetLinearSwapMarkets(ctx context.Context, code currency.Pair, supportMarginMode, contractType, businessType string) ([]LinearSwapMarket, error) {
	var resp struct {
		Response
		Data []LinearSwapMarket `json:"data"`
	}
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	if supportMarginMode != "" {
		params.Set("support_margin_mode", supportMarginMode)
	}
	if contractType != "" {
		params.Set("contract_type", contractType)
	}
	if businessType != "" {
		params.Set("business_type", businessType)
	}
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapMarkets, params), &resp)
}

// GetLinearSwapMarketDepth gets current USDT-margined market depth.
func (e *Exchange) GetLinearSwapMarketDepth(ctx context.Context, code currency.Pair, dataType string) (SwapMarketDepthData, error) {
	var resp SwapMarketDepthData
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("type", dataType)
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapMarketDepth, params), &resp)
}

// GetLinearSwapMarketOverview gets current USDT-margined market overview.
func (e *Exchange) GetLinearSwapMarketOverview(ctx context.Context, code currency.Pair) (MarketOverviewData, error) {
	var resp MarketOverviewData
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapMarketOverview, params), &resp)
}

// GetLinearSwapFundingRate gets the current funding rate for a USDT-margined contract.
func (e *Exchange) GetLinearSwapFundingRate(ctx context.Context, code currency.Pair) (FundingRatesData, error) {
	var resp struct {
		Response
		Data FundingRatesData `json:"data"`
	}
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapFunding, params), &resp)
}

// GetLinearSwapFundingRates gets current funding rates for USDT-margined contracts.
func (e *Exchange) GetLinearSwapFundingRates(ctx context.Context) (SwapFundingRatesResponse, error) {
	var resp SwapFundingRatesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, linearSwapBatchFunding, &resp)
}

// GetV5OpenInterest gets the current USDT-margined contract open interest.
func (e *Exchange) GetV5OpenInterest(ctx context.Context, code currency.Pair) (*V5OpenInterestResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	var resp *V5OpenInterestResponse
	err = e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(v5MarketOpenInterest, params), &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != http.StatusOK {
		return nil, fmt.Errorf("error code: %v error message: %s", resp.Code, resp.Message)
	}
	return resp, nil
}

// GetV5AccountBalance gets the migrated USDT-margined unified-margin account balance.
func (e *Exchange) GetV5AccountBalance(ctx context.Context) (*V5AccountBalanceResponse, error) {
	var resp *V5AccountBalanceResponse
	return resp, e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, v5AccountBalance, nil, nil, &resp)
}

// PlaceV5Order places a migrated USDT-margined unified-margin order.
func (e *Exchange) PlaceV5Order(ctx context.Context, req *V5OrderRequest) (*V5OrderResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w V5OrderRequest", common.ErrNilPointer)
	}
	var resp *V5OrderResponse
	return resp, e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, v5TradeOrder, nil, req, &resp)
}

// CancelV5Order cancels a migrated USDT-margined unified-margin order.
func (e *Exchange) CancelV5Order(ctx context.Context, code currency.Pair, orderID, clientOrderID string) (*V5OrderResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	req := &V5CancelOrderRequest{
		ContractCode: codeValue,
	}
	if orderID != "" {
		req.OrderID = orderID
	}
	if clientOrderID != "" {
		req.ClientOrderID = clientOrderID
	}
	var resp *V5OrderResponse
	return resp, e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, v5TradeCancelOrder, nil, req, &resp)
}

// CancelAllV5Orders cancels all migrated USDT-margined unified-margin orders for a contract.
func (e *Exchange) CancelAllV5Orders(ctx context.Context, code currency.Pair, side, positionSide string) (*V5CancelAllOrdersResponse, error) {
	req := &V5CancelAllOrdersRequest{
		Side:         side,
		PositionSide: positionSide,
	}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		req.ContractCode = codeValue
	}
	var resp *V5CancelAllOrdersResponse
	return resp, e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, v5TradeCancelAllOrders, nil, req, &resp)
}

// GetV5Order gets a migrated USDT-margined unified-margin order.
func (e *Exchange) GetV5Order(ctx context.Context, code currency.Pair, marginMode, orderID, clientOrderID string) (*V5OrderQueryResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	var resp *V5OrderQueryResponse
	return resp, e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, v5TradeOrder, params, nil, &resp)
}

// GetV5OpenOrders gets migrated USDT-margined unified-margin open orders.
func (e *Exchange) GetV5OpenOrders(ctx context.Context, code currency.Pair, marginMode, orderID, clientOrderID string, from, limit uint64, direct string) (*V5OrdersQueryResponse, error) {
	params := url.Values{}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	if marginMode != "" {
		params.Set("margin_mode", marginMode)
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if from != 0 {
		params.Set("from", strconv.FormatUint(from, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if direct != "" {
		params.Set("direct", direct)
	}
	var resp *V5OrdersQueryResponse
	return resp, e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, v5TradeOrderOpens, params, nil, &resp)
}
