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
	linearSwapFunding        = "/linear-swap-api/v1/swap_funding_rate"
	linearSwapBatchFunding   = "/linear-swap-api/v1/swap_batch_funding_rate"
	linearSwapFundingHistory = "/linear-swap-api/v1/swap_historical_funding_rate"
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
