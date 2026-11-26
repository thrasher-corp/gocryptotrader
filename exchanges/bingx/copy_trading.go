package bingx

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errCopyTradingPositionIDRequired         = errors.New("position ID is required")
	errCopyTradingTakeProfitStopLossRequired = errors.New("a take profit or stop loss mark price is required")
	errCopyTradingCommissionRateRequired     = errors.New("commission rate is required")
	errCopyTradingContractTypeRequired       = errors.New("contract type is required")
)

// SetCopyTradingPerpetualTakeProfitStopLoss sets the take profit or stop loss mark price of a perpetual futures order
// the trader leads, from Traders set take profit and stop loss based on order numbers
func (e *Exchange) SetCopyTradingPerpetualTakeProfitStopLoss(ctx context.Context, arg *CopyTradingTakeProfitStopLossRequest) (*CopyTradingTakeProfitStopLossResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PositionID == 0 {
		return nil, errCopyTradingPositionIDRequired
	}
	if arg.TakeProfitMarkPrice <= 0 && arg.StopLossMarkPrice <= 0 {
		return nil, errCopyTradingTakeProfitStopLossRequired
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatUint(arg.PositionID, 10))
	if arg.TakeProfitMarkPrice > 0 {
		params.Set("takeProfitMarkPrice", strconv.FormatFloat(arg.TakeProfitMarkPrice, 'f', -1, 64))
	}
	if arg.StopLossMarkPrice > 0 {
		params.Set("stopLossMarkPrice", strconv.FormatFloat(arg.StopLossMarkPrice, 'f', -1, 64))
	}
	var resp *CopyTradingTakeProfitStopLossResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/copyTrading/v1/swap/trace/setTPSL", params, copyTradingSetTakeProfitStopLossRate, &resp)
}

// GetCopyTradingPerpetualTraderDetail returns the trader's perpetual futures copy trading overview, from Personal
// Trading Overview. daySize sets the period of the profit percentage, 7, 30, 90 or 180 days, when not zero
func (e *Exchange) GetCopyTradingPerpetualTraderDetail(ctx context.Context, daySize uint64) (*CopyTradingPerpetualTraderDetailResponse, error) {
	params := url.Values{}
	if daySize > 0 {
		params.Set("daySize", strconv.FormatUint(daySize, 10))
	}
	var resp *CopyTradingPerpetualTraderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/PFutures/traderDetail", params, copyTradingPerpetualTraderDetailRate, &resp)
}

// GetCopyTradingPerpetualProfitSummary returns the trader's perpetual futures profit sharing overview, from Profit
// Overview
func (e *Exchange) GetCopyTradingPerpetualProfitSummary(ctx context.Context) (*CopyTradingPerpetualProfitSummaryResponse, error) {
	var resp *CopyTradingPerpetualProfitSummaryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/PFutures/profitHistorySummarys", nil, copyTradingPerpetualProfitSummaryRate, &resp)
}

// GetCopyTradingPerpetualProfitDetail returns a page of the profit shares the trader received from perpetual futures
// copiers, from Profit Details
func (e *Exchange) GetCopyTradingPerpetualProfitDetail(ctx context.Context, arg *CopyTradingProfitDetailRequest) ([]CopyTradingPerpetualProfitDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := copyTradingPageParams(arg.StartTime, arg.EndTime, arg.PageIndex, arg.PageSize)
	if err != nil {
		return nil, err
	}
	var resp []CopyTradingPerpetualProfitDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/PFutures/profitDetail", params, copyTradingPerpetualProfitDetailRate, &resp)
}

// SetCopyTradingPerpetualCommissionRate sets the trader's perpetual futures commission rate, a percentage such as 16
// for 16%, from Set Commission Rate. BingX takes rates above 10% and below the trader's level's own: 16% for Silver,
// 20% for Gold and 32% for Diamond
func (e *Exchange) SetCopyTradingPerpetualCommissionRate(ctx context.Context, commissionRate float64) (*CopyTradingPerpetualCommissionRateResponse, error) {
	if commissionRate <= 0 {
		return nil, errCopyTradingCommissionRateRequired
	}
	params := url.Values{}
	params.Set("newCommission", strconv.FormatFloat(commissionRate, 'f', -1, 64)+"%")
	var resp *CopyTradingPerpetualCommissionRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/openApi/copyTrading/v1/PFutures/setCommission", params, copyTradingSetCommissionRate, &resp)
}

// GetCopyTradingPerpetualTradingPairs returns the pairs a trader can lead copy trading in, from Trader Gets Copy
// Trading Pairs
func (e *Exchange) GetCopyTradingPerpetualTradingPairs(ctx context.Context, contractType CopyTradingContractType) ([]CopyTradingPerpetualTradingPairResponse, error) {
	if contractType == "" {
		return nil, errCopyTradingContractTypeRequired
	}
	params := url.Values{}
	params.Set("contractType", string(contractType))
	var resp []CopyTradingPerpetualTradingPairResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/PFutures/tradingPairs", params, copyTradingPerpetualTradingPairsRate, &resp)
}

// SellCopyTradingSpotOrder sells the spot assets a trader's buy order bought, from Trader sells spot assets based on
// buy order number
func (e *Exchange) SellCopyTradingSpotOrder(ctx context.Context, orderID uint64) (*CopyTradingSpotSellOrderResponse, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatUint(orderID, 10))
	var resp *CopyTradingSpotSellOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/copyTrading/v1/spot/trader/sellOrder", params, copyTradingSpotSellOrderRate, &resp)
}

// GetCopyTradingSpotTraderDetail returns the trader's spot copy trading overview, from Personal Trading Overview
func (e *Exchange) GetCopyTradingSpotTraderDetail(ctx context.Context) (*CopyTradingSpotTraderDetailResponse, error) {
	var resp *CopyTradingSpotTraderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/spot/traderDetail", nil, copyTradingSpotTraderDetailRate, &resp)
}

// GetCopyTradingSpotProfitSummary returns the trader's spot profit sharing overview, from Profit Summary
func (e *Exchange) GetCopyTradingSpotProfitSummary(ctx context.Context) (*CopyTradingSpotProfitSummaryResponse, error) {
	var resp *CopyTradingSpotProfitSummaryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/spot/profitHistorySummarys", nil, copyTradingSpotProfitSummaryRate, &resp)
}

// GetCopyTradingSpotProfitDetail returns a page of the profit shares the trader received from spot copiers, from
// Profit Details
func (e *Exchange) GetCopyTradingSpotProfitDetail(ctx context.Context, arg *CopyTradingProfitDetailRequest) ([]CopyTradingSpotProfitDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := copyTradingPageParams(arg.StartTime, arg.EndTime, arg.PageIndex, arg.PageSize)
	if err != nil {
		return nil, err
	}
	var resp []CopyTradingSpotProfitDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/spot/profitDetail", params, copyTradingSpotProfitDetailRate, &resp)
}

// GetCopyTradingSpotOrderHistory returns a page of the trader's completed spot lead trades, from Query Historical
// Orders
func (e *Exchange) GetCopyTradingSpotOrderHistory(ctx context.Context, arg *CopyTradingSpotOrderHistoryRequest) ([]CopyTradingSpotOrderHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := copyTradingPageParams(arg.StartTime, arg.EndTime, arg.PageIndex, arg.PageSize)
	if err != nil {
		return nil, err
	}
	if !arg.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(arg.Symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	var resp []CopyTradingSpotOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/copyTrading/v1/spot/historyOrder", params, copyTradingSpotOrderHistoryRate, &resp)
}

// copyTradingPageParams returns the parameters of a copy trading query with an optional time range and required page
// parameters
func copyTradingPageParams(start, end time.Time, pageIndex, pageSize uint64) (url.Values, error) {
	if pageIndex == 0 {
		return nil, errPageIndexRequired
	}
	if pageSize == 0 {
		return nil, errPageSizeRequired
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTime", "endTime", start, end); err != nil {
		return nil, err
	}
	params.Set("pageIndex", strconv.FormatUint(pageIndex, 10))
	params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	return params, nil
}
