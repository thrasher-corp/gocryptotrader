package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Strategy request validation errors
var (
	errStrategyIDRequired                = errors.New("strategy ID required")
	errStrategyTypeRequired              = errors.New("strategy type required")
	errStrategySizeRequired              = errors.New("strategy size or position value required")
	errStrategySizeWithPositionValue     = errors.New("strategy size and position value are mutually exclusive")
	errStrategyDurationRequired          = errors.New("strategy duration required")
	errStrategyPOVParametersRequired     = errors.New("POV strategy parameters required")
	errStrategyPOVModeRequired           = errors.New("POV strategy mode required")
	errStrategyParticipationRateRequired = errors.New("POV strategy participation rate required")
	errStrategyPOVStopConditionRequired  = errors.New("POV strategy needs a duration, size, position value or one-time interval")
	errStrategyReferenceWindowRequired   = errors.New("POV strategy reference window required for TradedVolume mode")
	errStrategyDepthReferenceRequired    = errors.New("POV strategy depth reference required for liquidity modes")
)

// errStrategyOrderFailed is returned when Bybit accepts a strategy order request but reports that the strategy was not
// created
var errStrategyOrderFailed = errors.New("strategy order failed")

// CreateStrategyOrder creates a TWAP, chase, iceberg or POV strategy order (Create Strategy Order). When the result
// reports a failure, the response is returned with the error so the strategy ID is not lost
func (e *Exchange) CreateStrategyOrder(ctx context.Context, arg *CreateStrategyOrderRequest) (*CreateStrategyOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Category == "" {
		return nil, errCategoryNotSet
	}
	if arg.Symbol == "" {
		return nil, errSymbolMissing
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.StrategyType == "" {
		return nil, errStrategyTypeRequired
	}
	switch arg.StrategyType {
	case "twap", "chaseOrder", "iceberg":
		if arg.Size <= 0 && arg.PositionValue <= 0 {
			return nil, errStrategySizeRequired
		}
		if arg.StrategyType == "twap" && arg.Duration < time.Second {
			return nil, errStrategyDurationRequired
		}
	case "pov":
		if arg.POVParameters == nil {
			return nil, errStrategyPOVParametersRequired
		}
		if arg.POVParameters.Mode == "" {
			return nil, errStrategyPOVModeRequired
		}
		if arg.POVParameters.ParticipationRate <= 0 {
			return nil, errStrategyParticipationRateRequired
		}
		if arg.Size > 0 && arg.PositionValue > 0 {
			return nil, errStrategySizeWithPositionValue
		}
		oneTime := arg.Interval != nil && *arg.Interval < time.Second
		if arg.Duration < time.Second && arg.Size <= 0 && arg.PositionValue <= 0 && !oneTime {
			return nil, errStrategyPOVStopConditionRequired
		}
		switch arg.POVParameters.Mode {
		case "TradedVolume":
			if arg.POVParameters.ReferenceWindow < time.Second {
				return nil, errStrategyReferenceWindowRequired
			}
		case "OppositeSideLiquidity", "SameSideLiquidity":
			if arg.POVParameters.DepthReference == 0 {
				return nil, errStrategyDepthReferenceRequired
			}
		}
	}
	body := make(map[string]any)
	body["category"] = arg.Category
	body["symbol"] = arg.Symbol
	body["side"] = arg.Side
	body["strategyType"] = arg.StrategyType
	if arg.Size != 0 {
		body["size"] = strconv.FormatFloat(arg.Size, 'f', -1, 64)
	}
	if arg.PositionValue != 0 {
		body["positionValue"] = strconv.FormatFloat(arg.PositionValue, 'f', -1, 64)
	}
	if arg.Duration != 0 {
		body["duration"] = int64(arg.Duration / time.Second)
	}
	if arg.ReduceOnly {
		body["reduceOnly"] = true
	}
	if arg.PositionIndex != 0 {
		body["positionIdx"] = arg.PositionIndex
	}
	if arg.LeverageType != 0 {
		body["leverageType"] = arg.LeverageType
	}
	if arg.Interval != nil {
		body["interval"] = int64(*arg.Interval / time.Second)
	}
	if arg.IsRandom {
		body["isRandom"] = true
	}
	if arg.TriggerPrice != 0 {
		body["triggerPrice"] = strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64)
	}
	if arg.MaxChasePrice != 0 {
		body["maxChasePrice"] = strconv.FormatFloat(arg.MaxChasePrice, 'f', -1, 64)
	}
	if arg.ChaseDistance != 0 {
		body["chaseDistance"] = strconv.FormatFloat(arg.ChaseDistance, 'f', -1, 64)
	}
	if arg.ChasePercentE4 != 0 {
		body["chasePercentE4"] = arg.ChasePercentE4
	}
	if arg.SubSize != 0 {
		body["subSize"] = strconv.FormatFloat(arg.SubSize, 'f', -1, 64)
	}
	if arg.OrderCount != 0 {
		body["orderCount"] = arg.OrderCount
	}
	if arg.SubPositionValue != 0 {
		body["subPositionValue"] = strconv.FormatFloat(arg.SubPositionValue, 'f', -1, 64)
	}
	if arg.PostOnly != nil {
		body["postOnly"] = *arg.PostOnly
	}
	if arg.LimitPrice != 0 {
		body["limitPrice"] = strconv.FormatFloat(arg.LimitPrice, 'f', -1, 64)
	}
	if p := arg.POVParameters; p != nil {
		parameters := make(map[string]any)
		parameters["mode"] = p.Mode
		parameters["participationRate"] = strconv.FormatFloat(p.ParticipationRate, 'f', -1, 64)
		if p.ReferenceWindow != 0 {
			parameters["referenceWindow"] = strconv.FormatInt(int64(p.ReferenceWindow/time.Second), 10)
		}
		if p.DepthReference != 0 {
			parameters["depthReference"] = p.DepthReference
		}
		body["povParams"] = parameters
	}
	var resp *CreateStrategyOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/strategy/create", nil, body, &resp, createStrategyOrderEPL); err != nil {
		return nil, err
	}
	if resp.Result != "" {
		return resp, fmt.Errorf("%w: strategy ID %q: %s", errStrategyOrderFailed, resp.StrategyID, resp.Result)
	}
	return resp, nil
}

// setStrategyTimeWindow sets the strategy endpoints' beginTimeE0 and endTimeE0, which are in whole seconds
func setStrategyTimeWindow(params url.Values, begin, end time.Time) error {
	if !begin.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(begin, end); err != nil {
			return err
		}
	}
	if !begin.IsZero() {
		params.Set("beginTimeE0", strconv.FormatInt(begin.Unix(), 10))
	}
	if !end.IsZero() {
		params.Set("endTimeE0", strconv.FormatInt(end.Unix(), 10))
	}
	return nil
}

// GetStrategyList returns strategies matching the filters (Get Strategy List)
func (e *Exchange) GetStrategyList(ctx context.Context, arg *GetStrategyListRequest) (*StrategyListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.StrategyID != "" {
		params.Set("strategyId", arg.StrategyID)
	}
	if arg.Symbol != "" {
		params.Set("symbol", arg.Symbol)
	}
	if arg.Status != 0 {
		params.Set("status", strconv.FormatUint(arg.Status, 10))
	}
	if arg.Category != "" {
		params.Set("category", arg.Category)
	}
	if arg.StrategyType != "" {
		params.Set("strategyType", arg.StrategyType)
	}
	if err := setStrategyTimeWindow(params, arg.BeginTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *StrategyListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/strategy/list", params, nil, &resp, getStrategyListEPL)
}

// GetStrategyOrderList returns the orders a strategy placed (Get Strategy Order List)
func (e *Exchange) GetStrategyOrderList(ctx context.Context, arg *GetStrategyOrderListRequest) (*StrategyOrderListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.StrategyID == "" {
		return nil, errStrategyIDRequired
	}
	params := url.Values{}
	params.Set("strategyId", arg.StrategyID)
	if arg.Status != 0 {
		params.Set("status", strconv.FormatUint(arg.Status, 10))
	}
	if arg.Symbol != "" {
		params.Set("symbol", arg.Symbol)
	}
	if arg.StrategyType != "" {
		params.Set("strategyType", arg.StrategyType)
	}
	if err := setStrategyTimeWindow(params, arg.BeginTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *StrategyOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/strategy/order-list", params, nil, &resp, getStrategyOrderListEPL)
}

// StopStrategy stops a running strategy, cancelling its unfilled orders; a stopped strategy cannot resume (Stop
// Strategy)
func (e *Exchange) StopStrategy(ctx context.Context, strategyID string) (*StopStrategyResponse, error) {
	if strategyID == "" {
		return nil, errStrategyIDRequired
	}
	body := make(map[string]any)
	body["strategyId"] = strategyID
	var resp *StopStrategyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/strategy/stop", nil, body, &resp, stopStrategyEPL)
}
