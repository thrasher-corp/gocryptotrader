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

// GetLinearSwapMarkets gets current USDT-margined contract metadata.
func (e *Exchange) GetLinearSwapMarkets(ctx context.Context, code currency.Pair, supportMarginMode, contractType, businessType string) ([]LinearSwapMarket, error) {
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
	resp := new(LinearSwapMarketsResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/linear-swap-api/v1/swap_contract_info", params), resp)
	return resp.Markets, err
}

// GetLinearSwapMarketDepth gets current USDT-margined market depth.
func (e *Exchange) GetLinearSwapMarketDepth(ctx context.Context, code currency.Pair, dataType string) (*SwapMarketDepthResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("type", dataType)
	resp := new(SwapMarketDepthResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/linear-swap-ex/market/depth", params), resp)
	return resp, err
}

// GetLinearSwapMarketOverview gets current USDT-margined market overview.
func (e *Exchange) GetLinearSwapMarketOverview(ctx context.Context, code currency.Pair) (*SwapMarketOverviewResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	resp := new(SwapMarketOverviewResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/linear-swap-ex/market/detail/merged", params), resp)
	return resp, err
}

// GetLinearSwapKlineData gets USDT-margined contract candlesticks.
func (e *Exchange) GetLinearSwapKlineData(ctx context.Context, code currency.Pair, period string, size uint64, startTime, endTime time.Time) (*SwapKlineResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	if !startTime.IsZero() || !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("to", strconv.FormatInt(endTime.Unix(), 10))
	}
	resp := new(SwapKlineResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/linear-swap-ex/market/history/kline", params), resp)
	return resp, err
}

// GetLinearSwapBatchTrades gets recent trades for a USDT-margined contract.
func (e *Exchange) GetLinearSwapBatchTrades(ctx context.Context, code currency.Pair, size uint64) (*SwapTradesResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatUint(size, 10))
	resp := new(SwapTradesResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/linear-swap-ex/market/history/trade", params), resp)
	return resp, err
}
