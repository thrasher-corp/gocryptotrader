package htx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

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
	linearSwapKline          = "/linear-swap-ex/market/history/kline"
	linearSwapBatchTrades    = "/linear-swap-ex/market/history/trade"
	linearSwapFunding        = "/linear-swap-api/v1/swap_funding_rate"
	linearSwapBatchFunding   = "/linear-swap-api/v1/swap_batch_funding_rate"
	linearSwapFundingHistory = "/linear-swap-api/v1/swap_historical_funding_rate"
	linearSwapSwitchLeverage = "/linear-swap-api/v1/swap_switch_lever_rate"
	linearSwapCrossLeverage  = "/linear-swap-api/v1/swap_cross_switch_lever_rate"
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
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapMarkets, params), &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
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
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapMarketDepth, params), &resp); err != nil {
		return resp, err
	}
	return resp, nil
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
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapMarketOverview, params), &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetLinearSwapKlineData gets USDT-margined contract candlesticks.
func (e *Exchange) GetLinearSwapKlineData(ctx context.Context, code currency.Pair, period string, size int64, startTime, endTime time.Time) (SwapKlineData, error) {
	var resp SwapKlineData
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errStartTimeAfterEndTime
		}
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("to", strconv.FormatInt(endTime.Unix(), 10))
	}
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapKline, params), &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetLinearSwapBatchTrades gets recent trades for a USDT-margined contract.
func (e *Exchange) GetLinearSwapBatchTrades(ctx context.Context, code currency.Pair, size int64) (BatchTradesData, error) {
	var resp BatchTradesData
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatInt(size, 10))
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapBatchTrades, params), &resp); err != nil {
		return resp, err
	}
	return resp, nil
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
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapFunding, params), &resp); err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

// GetLinearSwapFundingRates gets current funding rates for USDT-margined contracts.
func (e *Exchange) GetLinearSwapFundingRates(ctx context.Context) (SwapFundingRatesResponse, error) {
	var resp SwapFundingRatesResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, linearSwapBatchFunding, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetLinearSwapHistoricalFundingRates gets historical funding rates for a USDT-margined contract.
func (e *Exchange) GetLinearSwapHistoricalFundingRates(ctx context.Context, code currency.Pair, pageSize, pageIndex int64) (HistoricalFundingRateData, error) {
	var resp HistoricalFundingRateData
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(linearSwapFundingHistory, params), &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SwitchLinearSwapLeverage changes the leverage used by a USDT-margined contract.
func (e *Exchange) SwitchLinearSwapLeverage(ctx context.Context, code currency.Pair, leverage uint64, crossMargin bool) error {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	req := &SwitchLinearSwapLeverageRequest{
		ContractCode: codeValue,
		LeverageRate: leverage,
	}
	endpoint := linearSwapSwitchLeverage
	if crossMargin {
		endpoint = linearSwapCrossLeverage
	}
	var resp *SwitchLinearSwapLeverageResponse
	return e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, endpoint, nil, req, &resp)
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, v5AccountBalance, nil, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceV5Order places a migrated USDT-margined unified-margin order.
func (e *Exchange) PlaceV5Order(ctx context.Context, req *V5OrderRequest) (*V5OrderResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w V5OrderRequest", common.ErrNilPointer)
	}
	var resp *V5OrderResponse
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, v5TradeOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, v5TradeCancelOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, v5TradeCancelAllOrders, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, v5TradeOrder, params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, v5TradeOrderOpens, params, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}
