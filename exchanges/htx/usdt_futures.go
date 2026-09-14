package htx

import (
	"context"
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
