package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidGridNumber                = errors.New("invalid grid quantity (grid number)")
	errRunTypeRequired                  = errors.New("run type is required")
	errMissingRequiredArgumentDirection = errors.New("missing required argument, direction")
	errStopTypeRequired                 = errors.New("stop type is required")
	errMissingSubOrderType              = errors.New("missing sub order type")
	errMarginBalanceTypeRequired        = errors.New("margin balance adjustment type is required")
	errTriggerActionRequired            = errors.New("trigger action is required")
	errTriggerStrategyRequired          = errors.New("trigger strategy is required")
	errSourceAlgoIDRequired             = errors.New("source algo ID is required")
	errInvalidRSIThreshold              = errors.New("RSI threshold must be an integer between 1 and 100")
)

const maxStopGridAlgoOrders = 10

// PlaceGridAlgoOrder places a spot or contract grid algo order
func (e *Exchange) PlaceGridAlgoOrder(ctx context.Context, arg *PlaceGridAlgoOrderRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.MaxPrice <= 0 {
		return nil, fmt.Errorf("%w: max price", limits.ErrPriceBelowMin)
	}
	if arg.MinPrice <= 0 {
		return nil, fmt.Errorf("%w: min price", limits.ErrPriceBelowMin)
	}
	if arg.GridNumber == 0 {
		return nil, errInvalidGridNumber
	}
	switch arg.AlgoOrderType {
	case AlgoOrdTypeGrid:
		if arg.QuoteSize <= 0 && arg.BaseSize <= 0 {
			return nil, fmt.Errorf("%w: either quote size or base size is required", order.ErrAmountMustBeSet)
		}
	case AlgoOrdTypeContractGrid:
		if arg.Size <= 0 {
			return nil, fmt.Errorf("%w: size is required", order.ErrAmountMustBeSet)
		}
		if arg.Direction == "" {
			return nil, errMissingRequiredArgumentDirection
		}
		if arg.Leverage <= 0 {
			return nil, errInvalidLeverage
		}
	}
	if err := validateTradingBotTriggerParameters(arg.TriggerParameters); err != nil {
		return nil, err
	}
	var resp *TradingBotOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, gridTradingEPL, http.MethodPost, "tradingBot/grid/order-algo", arg, &resp, request.AuthenticatedRequest); err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// AmendGridAlgoOrderBasicParameters amends a grid algo order's price range and grid quantity. When OKX rejects the
// amendment for a too small or too large investment the response carries the top up required alongside the error
func (e *Exchange) AmendGridAlgoOrderBasicParameters(ctx context.Context, arg *AmendGridAlgoOrderBasicParametersRequest) (*AmendGridAlgoOrderBasicParametersResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.MinPrice <= 0 {
		return nil, fmt.Errorf("%w: min price", limits.ErrPriceBelowMin)
	}
	if arg.MaxPrice <= 0 {
		return nil, fmt.Errorf("%w: max price", limits.ErrPriceBelowMin)
	}
	if arg.GridNumber == 0 {
		return nil, errInvalidGridNumber
	}
	var resp *AmendGridAlgoOrderBasicParametersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendGridAlgoOrderBasicParametersEPL, http.MethodPost, "tradingBot/grid/amend-algo-basic-param", arg, &resp, request.AuthenticatedRequest)
}

// AmendGridAlgoOrder amends a grid algo order's take profit and stop loss settings, top up amount or trigger
// parameters
func (e *Exchange) AmendGridAlgoOrder(ctx context.Context, arg *AmendGridAlgoOrderRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if err := validateTradingBotTriggerParameters(arg.TriggerParameters); err != nil {
		return nil, err
	}
	var resp *TradingBotOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, amendGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/amend-order-algo", arg, &resp, request.AuthenticatedRequest); err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// StopGridAlgoOrder stops up to 10 grid algo orders. When OKX rejects any of them the rows it returned come back
// beside the error
func (e *Exchange) StopGridAlgoOrder(ctx context.Context, args []StopGridAlgoOrderRequest) ([]TradingBotOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(args) > maxStopGridAlgoOrders {
		return nil, fmt.Errorf("%w: cannot stop more than %d grid algo orders", errExceedLimit, maxStopGridAlgoOrders)
	}
	for i := range args {
		if args[i].AlgoID == "" {
			return nil, errAlgoIDRequired
		}
		if args[i].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[i].AlgoOrderType == "" {
			return nil, errInvalidAlgoOrderType
		}
		if args[i].StopType == "" {
			return nil, errStopTypeRequired
		}
	}
	var resp []TradingBotOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, stopGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/stop-order-algo", args, &resp, request.AuthenticatedRequest); err != nil {
		for i := range resp {
			err = common.AppendError(err, getStatusError(resp[i].StatusCode, resp[i].StatusMessage))
		}
		return resp, err
	}
	return resp, nil
}

// ClosePositionForContractGrid closes a contract grid's position when its stop type keeps positions
func (e *Exchange) ClosePositionForContractGrid(ctx context.Context, arg *ClosePositionForContractGridRequest) (*ContractGridCloseOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if !arg.MarketCloseAllPositions {
		if arg.Size <= 0 {
			return nil, fmt.Errorf("%w: size is required to close part of a position", order.ErrAmountMustBeSet)
		}
		if arg.Price <= 0 {
			return nil, fmt.Errorf("%w: price is required to close part of a position", limits.ErrPriceBelowMin)
		}
	}
	var resp *ContractGridCloseOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, closePositionForForContractGridEPL, http.MethodPost, "tradingBot/grid/close-position", arg, &resp, request.AuthenticatedRequest)
}

// CancelClosePositionOrderForContractGrid cancels a contract grid's close position order
func (e *Exchange) CancelClosePositionOrderForContractGrid(ctx context.Context, arg *CancelClosePositionOrderRequest) (*ContractGridCloseOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *ContractGridCloseOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelClosePositionOrderForContractGridEPL, http.MethodPost, "tradingBot/grid/cancel-close-order", arg, &resp, request.AuthenticatedRequest)
}

// InstantTriggerGridAlgoOrder triggers a grid algo order that is waiting for its trigger condition
func (e *Exchange) InstantTriggerGridAlgoOrder(ctx context.Context, arg *InstantTriggerGridAlgoOrderRequest) (*InstantTriggerGridAlgoOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	var resp *InstantTriggerGridAlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, instantTriggerGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/order-instant-trigger", arg, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoOrdersList retrieves pending grid algo orders
func (e *Exchange) GetGridAlgoOrdersList(ctx context.Context, arg *GridAlgoOrderListRequest) ([]GridAlgoOrderResponse, error) {
	params, err := gridAlgoOrderListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []GridAlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderListEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/orders-algo-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoOrderHistory retrieves stopped grid algo orders
func (e *Exchange) GetGridAlgoOrderHistory(ctx context.Context, arg *GridAlgoOrderListRequest) ([]GridAlgoOrderResponse, error) {
	params, err := gridAlgoOrderListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []GridAlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/orders-algo-history", params), nil, &resp, request.AuthenticatedRequest)
}

// gridAlgoOrderListParams validates a grid algo order list request and returns its query parameters
func gridAlgoOrderListParams(arg *GridAlgoOrderListRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	params := url.Values{}
	params.Set("algoOrdType", arg.AlgoOrderType)
	if arg.AlgoID != "" {
		params.Set("algoId", arg.AlgoID)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.InstrumentType != "" {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return params, nil
}

// GetGridAlgoOrderDetails retrieves a grid algo order's details
func (e *Exchange) GetGridAlgoOrderDetails(ctx context.Context, arg *GridAlgoOrderRequest) (*GridAlgoOrderResponse, error) {
	params, err := gridAlgoOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *GridAlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/orders-algo-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoSubOrders retrieves a grid algo order's live or filled sub orders
func (e *Exchange) GetGridAlgoSubOrders(ctx context.Context, arg *GridAlgoSubOrdersRequest) ([]GridSubOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.SubOrderType == "" {
		return nil, errMissingSubOrderType
	}
	params := url.Values{}
	params.Set("algoOrdType", arg.AlgoOrderType)
	params.Set("algoId", arg.AlgoID)
	params.Set("type", arg.SubOrderType)
	if arg.GroupID != "" {
		params.Set("groupId", arg.GroupID)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []GridSubOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoSubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/sub-orders", params), nil, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoOrderPositions retrieves a contract grid algo order's positions
func (e *Exchange) GetGridAlgoOrderPositions(ctx context.Context, arg *GridAlgoOrderRequest) ([]TradingBotPositionResponse, error) {
	params, err := gridAlgoOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []TradingBotPositionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderPositionsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/positions", params), nil, &resp, request.AuthenticatedRequest)
}

// gridAlgoOrderParams validates a grid algo order identifier and returns its query parameters
func gridAlgoOrderParams(arg *GridAlgoOrderRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoOrdType", arg.AlgoOrderType)
	params.Set("algoId", arg.AlgoID)
	return params, nil
}

// SpotGridWithdrawProfit withdraws a spot grid algo order's profit
func (e *Exchange) SpotGridWithdrawProfit(ctx context.Context, algoID string) (*SpotGridWithdrawIncomeResponse, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	var resp *SpotGridWithdrawIncomeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, spotGridWithdrawIncomeEPL, http.MethodPost, "tradingBot/grid/withdraw-income", &TradingBotAlgoIDRequest{AlgoID: algoID}, &resp, request.AuthenticatedRequest)
}

// ComputeGridMarginBalance computes the margin balance a contract grid can add or reduce
func (e *Exchange) ComputeGridMarginBalance(ctx context.Context, arg *ComputeGridMarginBalanceRequest) (*ComputeGridMarginBalanceResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AdjustMarginBalanceType == "" {
		return nil, errMarginBalanceTypeRequired
	}
	var resp *ComputeGridMarginBalanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, computeMarginBalanceEPL, http.MethodPost, "tradingBot/grid/compute-margin-balance", arg, &resp, request.AuthenticatedRequest)
}

// AdjustGridMarginBalance adds or reduces a contract grid's margin balance
func (e *Exchange) AdjustGridMarginBalance(ctx context.Context, arg *AdjustGridMarginBalanceRequest) (*AdjustGridMarginBalanceResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AdjustMarginBalanceType == "" {
		return nil, errMarginBalanceTypeRequired
	}
	if arg.Amount <= 0 && arg.Percent <= 0 {
		return nil, fmt.Errorf("%w: either amount or percent is required", order.ErrAmountIsInvalid)
	}
	var resp *AdjustGridMarginBalanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, adjustMarginBalanceEPL, http.MethodPost, "tradingBot/grid/margin-balance", arg, &resp, request.AuthenticatedRequest)
}

// AddGridInvestment adds investment to a grid algo order
func (e *Exchange) AddGridInvestment(ctx context.Context, arg *AddGridInvestmentRequest) (*TradingBotAlgoIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountMustBeSet
	}
	var resp *TradingBotAlgoIDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, addGridInvestmentEPL, http.MethodPost, "tradingBot/grid/adjust-investment", arg, &resp, request.AuthenticatedRequest)
}

// GetGridAIParameter retrieves the grid parameters OKX's AI suggests for an instrument
func (e *Exchange) GetGridAIParameter(ctx context.Context, arg *GridAIParameterRequest) ([]GridAIParameterResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoOrderType == AlgoOrdTypeContractGrid && arg.Direction == "" {
		return nil, fmt.Errorf("%w: required for contract grids", errMissingRequiredArgumentDirection)
	}
	params := url.Values{}
	params.Set("algoOrdType", arg.AlgoOrderType)
	params.Set("instId", arg.InstrumentID)
	if arg.Direction != "" {
		params.Set("direction", arg.Direction)
	}
	if arg.Duration != "" {
		params.Set("duration", arg.Duration)
	}
	var resp []GridAIParameterResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getGridAIParameterEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/ai-param", params), nil, &resp, request.UnauthenticatedRequest)
}

// ComputeMinInvestment computes a grid's minimum investment
func (e *Exchange) ComputeMinInvestment(ctx context.Context, arg *ComputeMinInvestmentRequest) (*ComputeMinInvestmentResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.MaxPrice <= 0 {
		return nil, fmt.Errorf("%w: max price", limits.ErrPriceBelowMin)
	}
	if arg.MinPrice <= 0 {
		return nil, fmt.Errorf("%w: min price", limits.ErrPriceBelowMin)
	}
	if arg.GridNumber == 0 {
		return nil, errInvalidGridNumber
	}
	if arg.RunType == "" {
		return nil, errRunTypeRequired
	}
	if arg.AlgoOrderType == AlgoOrdTypeContractGrid {
		if arg.Direction == "" {
			return nil, fmt.Errorf("%w: required for contract grids", errMissingRequiredArgumentDirection)
		}
		if arg.Leverage <= 0 {
			return nil, errInvalidLeverage
		}
	}
	for i := range arg.InvestmentData {
		if arg.InvestmentData[i].Amount <= 0 {
			return nil, fmt.Errorf("%w: investment amount", limits.ErrAmountBelowMin)
		}
		if arg.InvestmentData[i].Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
	}
	var resp *ComputeMinInvestmentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, computeMinInvestmentEPL, http.MethodPost, "tradingBot/grid/min-investment", arg, &resp, request.UnauthenticatedRequest)
}

// RSIBackTesting back tests a relative strength index (RSI) trigger, returning how often it would have fired
func (e *Exchange) RSIBackTesting(ctx context.Context, arg *RSIBackTestingRequest) (*RSIBackTestingResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	timeframe := IntervalFromString(arg.Timeframe, false)
	if timeframe == "" {
		return nil, errMissingIntervalValue
	}
	if arg.Threshold < 1 || arg.Threshold > 100 {
		return nil, errInvalidRSIThreshold
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	params.Set("timeframe", timeframe)
	params.Set("thold", strconv.FormatUint(arg.Threshold, 10))
	if arg.TimePeriod > 0 {
		params.Set("timePeriod", strconv.FormatUint(arg.TimePeriod, 10))
	}
	if arg.TriggerCondition != "" {
		params.Set("triggerCond", arg.TriggerCondition)
	}
	if arg.Duration != "" {
		params.Set("duration", arg.Duration)
	}
	var resp *RSIBackTestingResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, rsiBackTestingEPL, http.MethodGet, common.EncodeURLValues("tradingBot/public/rsi-back-testing", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetMaxGridQuantity retrieves the maximum grid quantity for a price range; the minimum is always 2
func (e *Exchange) GetMaxGridQuantity(ctx context.Context, arg *MaxGridQuantityRequest) (*MaxGridQuantityResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.RunType == "" {
		return nil, errRunTypeRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.MaxPrice <= 0 {
		return nil, fmt.Errorf("%w: max price", limits.ErrPriceBelowMin)
	}
	if arg.MinPrice <= 0 {
		return nil, fmt.Errorf("%w: min price", limits.ErrPriceBelowMin)
	}
	if arg.AlgoOrderType == AlgoOrdTypeContractGrid && arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	params := url.Values{}
	params.Set("instId", arg.InstrumentID)
	params.Set("runType", arg.RunType)
	params.Set("algoOrdType", arg.AlgoOrderType)
	params.Set("maxPx", strconv.FormatFloat(arg.MaxPrice, 'f', -1, 64))
	params.Set("minPx", strconv.FormatFloat(arg.MinPrice, 'f', -1, 64))
	if arg.Leverage > 0 {
		params.Set("lever", strconv.FormatFloat(arg.Leverage, 'f', -1, 64))
	}
	var resp *MaxGridQuantityResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMaxGridQuantityEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/grid-quantity", params), nil, &resp, request.UnauthenticatedRequest)
}

// CopyGridAlgoOrder places a grid algo order copying a lead trader's grid algo order
func (e *Exchange) CopyGridAlgoOrder(ctx context.Context, arg *CopyGridAlgoOrderRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.SourceAlgoID == "" {
		return nil, errSourceAlgoIDRequired
	}
	if arg.AutoReserve != nil {
		if *arg.AutoReserve && arg.Size <= 0 {
			return nil, fmt.Errorf("%w: size is required when margin is reserved automatically", order.ErrAmountMustBeSet)
		}
		if !*arg.AutoReserve && arg.ActualMarginSize <= 0 {
			return nil, fmt.Errorf("%w: actual margin size is required when margin is reserved manually", order.ErrAmountMustBeSet)
		}
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, copyGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/copy-order-algo", arg, &resp, request.AuthenticatedRequest)
}

// validateTradingBotTriggerParameters checks each trigger parameter carries the action and strategy OKX requires
func validateTradingBotTriggerParameters(params []TradingBotTriggerParameter) error {
	for i := range params {
		if params[i].TriggerAction == "" {
			return errTriggerActionRequired
		}
		if params[i].TriggerStrategy == "" {
			return errTriggerStrategyRequired
		}
	}
	return nil
}
