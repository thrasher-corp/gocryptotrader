package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errTriggerParametersRequired    = errors.New("trigger parameters are required")
	errTakeProfitPercentageRequired = errors.New("take profit percentage is required")
	errStopLossPercentageRequired   = errors.New("stop loss percentage is required")
	errSafetyOrderSettingRequired   = errors.New("safety order setting is required")
)

// PlaceDCAAlgoOrder places a spot or contract DCA (martingale) algo order
func (e *Exchange) PlaceDCAAlgoOrder(ctx context.Context, arg *PlaceDCAAlgoOrderRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.InitialOrderAmount <= 0 {
		return nil, fmt.Errorf("%w: initial order amount", order.ErrAmountMustBeSet)
	}
	if arg.MaxSafetyOrders > 0 {
		if err := validateDCASafetyOrderSettings(arg.SafetyOrderAmount, arg.PriceSteps, arg.PriceStepsMultiplier, arg.VolumeMultiplier); err != nil {
			return nil, err
		}
	}
	if arg.TakeProfitPercentage <= 0 {
		return nil, errTakeProfitPercentageRequired
	}
	if arg.AlgoOrderType == AlgoOrdTypeContractDCA && arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	if len(arg.TriggerParameters) == 0 {
		return nil, errTriggerParametersRequired
	}
	if err := validateTradingBotTriggerParameters(arg.TriggerParameters); err != nil {
		return nil, err
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, placeDCAAlgoOrderEPL, http.MethodPost, "tradingBot/dca/create", arg, &resp, request.AuthenticatedRequest)
}

// AmendSpotDCABasicParameters amends a spot DCA algo order's settings
func (e *Exchange) AmendSpotDCABasicParameters(ctx context.Context, arg *AmendSpotDCABasicParametersRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if err := validateDCASafetyOrderSettings(arg.SafetyOrderAmount, arg.PriceSteps, arg.PriceStepsMultiplier, arg.VolumeMultiplier); err != nil {
		return nil, err
	}
	if arg.TakeProfitPercentage <= 0 {
		return nil, errTakeProfitPercentageRequired
	}
	if arg.StopLossPercentage <= 0 {
		return nil, errStopLossPercentageRequired
	}
	if arg.InitialOrderAmount <= 0 {
		return nil, fmt.Errorf("%w: initial order amount", order.ErrAmountMustBeSet)
	}
	if len(arg.TriggerParameters) == 0 {
		return nil, errTriggerParametersRequired
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendSpotDCABasicParametersEPL, http.MethodPost, "tradingBot/dca/amend-order-algo", arg, &resp, request.AuthenticatedRequest)
}

// validateDCASafetyOrderSettings checks the settings OKX requires when a DCA algo order places safety orders
func validateDCASafetyOrderSettings(safetyOrderAmount, priceSteps, priceStepsMultiplier, volumeMultiplier float64) error {
	if safetyOrderAmount <= 0 {
		return fmt.Errorf("%w: safety order amount", errSafetyOrderSettingRequired)
	}
	if priceSteps <= 0 {
		return fmt.Errorf("%w: price steps", errSafetyOrderSettingRequired)
	}
	if priceStepsMultiplier <= 0 {
		return fmt.Errorf("%w: price steps multiplier", errSafetyOrderSettingRequired)
	}
	if volumeMultiplier <= 0 {
		return fmt.Errorf("%w: volume multiplier", errSafetyOrderSettingRequired)
	}
	return nil
}

// StopDCAAlgoOrder stops a DCA algo order
func (e *Exchange) StopDCAAlgoOrder(ctx context.Context, arg *StopDCAAlgoOrderRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.StopType == "" {
		return nil, errStopTypeRequired
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, stopDCAAlgoOrderEPL, http.MethodPost, "tradingBot/dca/stop", arg, &resp, request.AuthenticatedRequest)
}

// GetDCAAlgoOrderList retrieves ongoing DCA algo orders
func (e *Exchange) GetDCAAlgoOrderList(ctx context.Context, arg *DCAAlgoOrderListRequest) ([]DCAAlgoOrderResponse, error) {
	params, err := dcaAlgoOrderListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []DCAAlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDCAAlgoOrderListEPL, http.MethodGet, common.EncodeURLValues("tradingBot/dca/ongoing-list", params), nil, &resp, request.AuthenticatedRequest)
}

// GetDCAAlgoOrderHistory retrieves stopped DCA algo orders
func (e *Exchange) GetDCAAlgoOrderHistory(ctx context.Context, arg *DCAAlgoOrderListRequest) ([]DCAAlgoOrderResponse, error) {
	params, err := dcaAlgoOrderListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []DCAAlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDCAAlgoOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/dca/history-list", params), nil, &resp, request.AuthenticatedRequest)
}

// dcaAlgoOrderListParams validates a DCA algo order list request and returns its query parameters
func dcaAlgoOrderListParams(arg *DCAAlgoOrderListRequest) (url.Values, error) {
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

// GetDCASubOrders retrieves a DCA algo order's sub orders
func (e *Exchange) GetDCASubOrders(ctx context.Context, arg *DCASubOrdersRequest) ([]DCASubOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	params := url.Values{}
	params.Set("algoId", arg.AlgoID)
	params.Set("algoOrdType", arg.AlgoOrderType)
	if arg.CycleID != "" {
		params.Set("cycleId", arg.CycleID)
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
	var resp []DCASubOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDCASubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/dca/orders", params), nil, &resp, request.AuthenticatedRequest)
}

// AddDCAInvestment adds investment to a DCA algo order with a manual buy
func (e *Exchange) AddDCAInvestment(ctx context.Context, arg *AddDCAInvestmentRequest) (*AddDCAInvestmentResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountMustBeSet
	}
	var resp *AddDCAInvestmentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, addDCAInvestmentEPL, http.MethodPost, "tradingBot/dca/orders/manual-buy", arg, &resp, request.AuthenticatedRequest)
}

// AmendDCAReinvestment sets whether a contract DCA algo order reinvests its profit
func (e *Exchange) AmendDCAReinvestment(ctx context.Context, arg *AmendDCAReinvestmentRequest) (*DCASettingResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	var resp *DCASettingResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendDCAReinvestmentEPL, http.MethodPost, "tradingBot/dca/settings/reinvestment", arg, &resp, request.AuthenticatedRequest)
}

// AmendDCATakeProfit sets a contract DCA algo order's take profit price
func (e *Exchange) AmendDCATakeProfit(ctx context.Context, arg *AmendDCATakeProfitRequest) (*DCASettingResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.TakeProfitPrice <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	var resp *DCASettingResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendDCATakeProfitEPL, http.MethodPost, "tradingBot/dca/settings/take-profit", arg, &resp, request.AuthenticatedRequest)
}

// GetDCAPositionDetails retrieves a DCA algo order's position in its current cycle
func (e *Exchange) GetDCAPositionDetails(ctx context.Context, arg *DCAAlgoOrderRequest) (*DCAPositionDetailsResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	params := url.Values{}
	params.Set("algoId", arg.AlgoID)
	params.Set("algoOrdType", arg.AlgoOrderType)
	var resp *DCAPositionDetailsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDCAPositionDetailsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/dca/position-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetDCACycleList retrieves a DCA algo order's cycles
func (e *Exchange) GetDCACycleList(ctx context.Context, arg *DCACycleListRequest) ([]DCACycleResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	params := url.Values{}
	params.Set("algoId", arg.AlgoID)
	params.Set("algoOrdType", arg.AlgoOrderType)
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
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
	var resp []DCACycleResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDCACycleListEPL, http.MethodGet, common.EncodeURLValues("tradingBot/dca/cycle-list", params), nil, &resp, request.AuthenticatedRequest)
}

// AddDCAMargin adds margin to a contract DCA algo order
func (e *Exchange) AddDCAMargin(ctx context.Context, arg *DCAMarginRequest) (*DCASettingResponse, error) {
	if err := validateDCAMarginRequest(arg); err != nil {
		return nil, err
	}
	var resp *DCASettingResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, addDCAMarginEPL, http.MethodPost, "tradingBot/dca/margin/add", arg, &resp, request.AuthenticatedRequest)
}

// ReduceDCAMargin reduces a contract DCA algo order's margin
func (e *Exchange) ReduceDCAMargin(ctx context.Context, arg *DCAMarginRequest) (*DCASettingResponse, error) {
	if err := validateDCAMarginRequest(arg); err != nil {
		return nil, err
	}
	var resp *DCASettingResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, reduceDCAMarginEPL, http.MethodPost, "tradingBot/dca/margin/reduce", arg, &resp, request.AuthenticatedRequest)
}

// validateDCAMarginRequest checks a DCA margin request carries its algo ID and amount
func validateDCAMarginRequest(arg *DCAMarginRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return errAlgoIDRequired
	}
	if arg.Amount <= 0 {
		return order.ErrAmountMustBeSet
	}
	return nil
}
