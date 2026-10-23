package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Binance takes TWAP durations from 5 minutes to 24 hours
const (
	minTWAPDuration = 5 * time.Minute
	maxTWAPDuration = 24 * time.Hour
)

// CancelFuturesAlgoOrder calls Cancel Futures Algo Order, identifying the order by its algo ID or its client algo ID
func (e *Exchange) CancelFuturesAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*CancelAlgoOrderResponse, error) {
	return e.cancelAlgoOrder(ctx, "/sapi/v1/algo/futures/order", algoID, clientAlgoID)
}

// GetFuturesCurrentAlgoOpenOrders calls Query Current Futures Algo Open Orders
func (e *Exchange) GetFuturesCurrentAlgoOpenOrders(ctx context.Context) (*FuturesAlgoOrdersResponse, error) {
	var resp *FuturesAlgoOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/algo/futures/openOrders", nil, sapiDefaultRate, &resp)
}

// GetFuturesHistoricalAlgoOrders calls Query Historical Futures Algo Orders
func (e *Exchange) GetFuturesHistoricalAlgoOrders(ctx context.Context, arg *AlgoHistoricalOrdersRequest) (*FuturesAlgoOrdersResponse, error) {
	params, err := e.algoHistoricalOrdersParams(arg, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	var resp *FuturesAlgoOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/algo/futures/historicalOrders", params, sapiDefaultRate, &resp)
}

// GetFuturesSubOrders calls Query Futures Sub Orders
func (e *Exchange) GetFuturesSubOrders(ctx context.Context, algoID, page, pageSize uint64) (*AlgoSubOrdersResponse, error) {
	return e.getAlgoSubOrders(ctx, "/sapi/v1/algo/futures/subOrders", algoID, page, pageSize)
}

// FuturesTWAPOrder calls Time-Weighted Futures Average Price (Twap) New Order, for USDⓈ-M contracts only. A success
// response does not guarantee execution; query the order for its final status
func (e *Exchange) FuturesTWAPOrder(ctx context.Context, arg *FuturesTWAPOrderRequest) (*AlgoOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.algoNewOrderParams(arg.Symbol, asset.USDTMarginedFutures, arg.Side, arg.Quantity, arg.ClientAlgoID, arg.LimitPrice)
	if err != nil {
		return nil, err
	}
	if arg.Duration < minTWAPDuration || arg.Duration > maxTWAPDuration {
		return nil, fmt.Errorf("%w: TWAP duration must be between 5 minutes and 24 hours", errDurationRequired)
	}
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	params.Set("duration", strconv.FormatInt(int64(arg.Duration/time.Second), 10))
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	var resp *AlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/algo/futures/newOrderTwap", params, placeTWAveragePriceNewOrderRate, &resp)
}

// VolumeParticipationNewOrder calls Volume Participation (VP) New Order, for USDⓈ-M contracts only. A success response
// does not guarantee execution; query the order for its final status
func (e *Exchange) VolumeParticipationNewOrder(ctx context.Context, arg *VolumeParticipationOrderRequest) (*AlgoOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.algoNewOrderParams(arg.Symbol, asset.USDTMarginedFutures, arg.Side, arg.Quantity, arg.ClientAlgoID, arg.LimitPrice)
	if err != nil {
		return nil, err
	}
	if arg.Urgency == "" {
		return nil, errPossibleValuesRequired
	}
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	params.Set("urgency", arg.Urgency)
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	var resp *AlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/algo/futures/newOrderVp", params, placeVPOrderRate, &resp)
}

// CancelSpotAlgoOrder calls Cancel Spot Algo Order, identifying the order by its algo ID or its client algo ID
func (e *Exchange) CancelSpotAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*CancelAlgoOrderResponse, error) {
	return e.cancelAlgoOrder(ctx, "/sapi/v1/algo/spot/order", algoID, clientAlgoID)
}

// GetCurrentSpotAlgoOpenOrder calls Query Current Spot Algo Open Orders
func (e *Exchange) GetCurrentSpotAlgoOpenOrder(ctx context.Context) (*SpotAlgoOrdersResponse, error) {
	var resp *SpotAlgoOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/algo/spot/openOrders", nil, sapiDefaultRate, &resp)
}

// GetSpotHistoricalAlgoOrders calls Query Historical Spot Algo Orders
func (e *Exchange) GetSpotHistoricalAlgoOrders(ctx context.Context, arg *AlgoHistoricalOrdersRequest) (*SpotAlgoOrdersResponse, error) {
	params, err := e.algoHistoricalOrdersParams(arg, asset.Spot)
	if err != nil {
		return nil, err
	}
	var resp *SpotAlgoOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/algo/spot/historicalOrders", params, sapiDefaultRate, &resp)
}

// GetSpotSubOrders calls Query Spot Sub Orders
func (e *Exchange) GetSpotSubOrders(ctx context.Context, algoID, page, pageSize uint64) (*AlgoSubOrdersResponse, error) {
	return e.getAlgoSubOrders(ctx, "/sapi/v1/algo/spot/subOrders", algoID, page, pageSize)
}

// SpotTWAPNewOrder calls Time-Weighted Spot Average Price(Twap) New Order
func (e *Exchange) SpotTWAPNewOrder(ctx context.Context, arg *SpotTWAPOrderRequest) (*AlgoOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.algoNewOrderParams(arg.Symbol, asset.Spot, arg.Side, arg.Quantity, arg.ClientAlgoID, arg.LimitPrice)
	if err != nil {
		return nil, err
	}
	if arg.Duration < minTWAPDuration || arg.Duration > maxTWAPDuration {
		return nil, fmt.Errorf("%w: TWAP duration must be between 5 minutes and 24 hours", errDurationRequired)
	}
	params.Set("duration", strconv.FormatInt(int64(arg.Duration/time.Second), 10))
	var resp *AlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/algo/spot/newOrderTwap", params, spotTWAPNewOrderRate, &resp)
}

// algoNewOrderParams validates and builds the parameters every algo new order shares
func (e *Exchange) algoNewOrderParams(symbol currency.Pair, a asset.Item, side string, quantity float64, clientAlgoID string, limitPrice float64) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	symbolValue, err := e.FormatSymbol(symbol, a)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	if clientAlgoID != "" {
		params.Set("clientAlgoId", clientAlgoID)
	}
	if limitPrice > 0 {
		params.Set("limitPrice", strconv.FormatFloat(limitPrice, 'f', -1, 64))
	}
	return params, nil
}

// cancelAlgoOrder cancels a futures or spot algo order; the API requires the algo ID or the client algo ID
func (e *Exchange) cancelAlgoOrder(ctx context.Context, path string, algoID uint64, clientAlgoID string) (*CancelAlgoOrderResponse, error) {
	if algoID == 0 && clientAlgoID == "" {
		return nil, fmt.Errorf("%w: algoId or clientAlgoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if algoID != 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	if clientAlgoID != "" {
		params.Set("clientAlgoId", clientAlgoID)
	}
	var resp *CancelAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, path, params, sapiDefaultRate, &resp)
}

// algoHistoricalOrdersParams builds the parameters of the futures and spot historical algo order queries
func (e *Exchange) algoHistoricalOrdersParams(arg *AlgoHistoricalOrdersRequest, a asset.Item) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !arg.Symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(arg.Symbol, a)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if arg.Side != "" {
		params.Set("side", arg.Side)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	return params, nil
}

// getAlgoSubOrders returns the sub orders of a futures or spot algo order
func (e *Exchange) getAlgoSubOrders(ctx context.Context, path string, algoID, page, pageSize uint64) (*AlgoSubOrdersResponse, error) {
	if algoID == 0 {
		return nil, fmt.Errorf("%w: algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", strconv.FormatUint(algoID, 10))
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *AlgoSubOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, path, params, sapiDefaultRate, &resp)
}
