package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errMissingEitherAlgoIDOrState = errors.New("either algo ID or order state is required")
	errLimitOrderNumberRequired   = errors.New("limit order number is required")
	errAggressivenessRequired     = errors.New("aggressiveness is required")
)

const (
	orderSmartIceberg = "smart_iceberg"
	// orderIceberg is the iceberg order type OKX still lists, which smart_iceberg replaced at placement
	orderIceberg = "iceberg"
)

// PlaceAlgoOrder places a trigger, OCO, chase, conditional, TWAP, trailing stop or smart iceberg algo order
func (e *Exchange) PlaceAlgoOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	arg.TradeMode = strings.ToLower(arg.TradeMode)
	if arg.TradeMode == "" {
		return nil, fmt.Errorf("%w: the trade mode is required", errInvalidTradeModeValue)
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 && arg.CloseFraction <= 0 {
		return nil, fmt.Errorf("%w: either size or close fraction is required", limits.ErrAmountBelowMin)
	}
	var resp *AlgoOrderActionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, placeAlgoOrderEPL, http.MethodPost, "trade/order-algo", arg, &resp, request.AuthenticatedRequest)
}

// PlaceStopOrder places a conditional take profit stop order
func (e *Exchange) PlaceStopOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.TakeProfitTriggerPrice <= 0 {
		return nil, fmt.Errorf("%w, take profit trigger price is required", limits.ErrPriceBelowMin)
	}
	if arg.TakeProfitTriggerPriceType == "" {
		return nil, order.ErrUnknownPriceType
	}
	return e.PlaceAlgoOrder(ctx, arg)
}

// PlaceTrailingStopOrder places a trailing stop order
func (e *Exchange) PlaceTrailingStopOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType != orderMoveOrderStop {
		return nil, fmt.Errorf("%w: order type value %q is only supported for trailing stop orders", order.ErrTypeIsInvalid, orderMoveOrderStop)
	}
	if arg.CallbackRatio == 0 && arg.CallbackSpread == 0 {
		return nil, fmt.Errorf("%w: either callback ratio or callback spread is required", errPriceTrackingNotSet)
	}
	return e.PlaceAlgoOrder(ctx, arg)
}

// PlaceIcebergOrder places a smart iceberg order
func (e *Exchange) PlaceIcebergOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType != orderSmartIceberg {
		return nil, fmt.Errorf("%w: order type value %q is only supported for iceberg orders", order.ErrTypeIsInvalid, orderSmartIceberg)
	}
	if arg.SizeLimit <= 0 {
		return nil, errMissingSizeLimit
	}
	if arg.LimitOrderNumber == 0 {
		return nil, errLimitOrderNumberRequired
	}
	if arg.Aggressiveness == "" {
		return nil, errAggressivenessRequired
	}
	return e.PlaceAlgoOrder(ctx, arg)
}

// PlaceTWAPOrder places a TWAP algo order
func (e *Exchange) PlaceTWAPOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType != orderTWAP {
		return nil, fmt.Errorf("%w: order type value %q is only supported for TWAP orders", order.ErrTypeIsInvalid, orderTWAP)
	}
	if arg.SizeLimit <= 0 {
		return nil, errMissingSizeLimit
	}
	if arg.PriceLimit <= 0 {
		return nil, errInvalidPriceLimit
	}
	if arg.TimeInterval == 0 {
		return nil, errMissingIntervalValue
	}
	return e.PlaceAlgoOrder(ctx, arg)
}

// PlaceTakeProfitStopLossOrder places a conditional order; with both take-profit and stop-loss parameters set, OKX
// performs the stop-loss only
func (e *Exchange) PlaceTakeProfitStopLossOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType != orderConditional {
		return nil, fmt.Errorf("%w for TPSL: %q", order.ErrTypeIsInvalid, arg.OrderType)
	}
	if arg.StopLossTriggerPrice <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	switch arg.StopLossTriggerPriceType {
	case "", "last", "index", "mark":
	default:
		return nil, fmt.Errorf("%w, only 'last', 'index', and 'mark' are supported", order.ErrUnknownPriceType)
	}
	return e.PlaceAlgoOrder(ctx, arg)
}

// PlaceChaseAlgoOrder places a chase order, which keeps amending a post-only order to the best bid or ask price
func (e *Exchange) PlaceChaseAlgoOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType != orderChase {
		return nil, fmt.Errorf("%w: order type value %q is only supported for chase orders", order.ErrTypeIsInvalid, orderChase)
	}
	if (arg.MaxChaseType == "") != (arg.MaxChaseValue == 0) {
		return nil, fmt.Errorf("%w, either none or both of the maximum chase type and value are required", errPriceTrackingNotSet)
	}
	return e.PlaceAlgoOrder(ctx, arg)
}

// PlaceTriggerAlgoOrder places a trigger order, which places a market or limit order, or a chase order for futures and
// perpetual swaps, once the trigger price is crossed
func (e *Exchange) PlaceTriggerAlgoOrder(ctx context.Context, arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType != orderTrigger {
		return nil, fmt.Errorf("%w for Trigger: %q", order.ErrTypeIsInvalid, arg.OrderType)
	}
	if arg.TriggerPrice <= 0 {
		return nil, fmt.Errorf("%w, trigger price must be greater than 0", limits.ErrPriceBelowMin)
	}
	switch arg.TriggerPriceType {
	case "", "last", "index", "mark":
	default:
		return nil, fmt.Errorf("%w, only last, index and mark trigger price types are allowed", order.ErrUnknownPriceType)
	}
	if arg.AdvanceOrderType == orderChase && len(arg.AdvancedChaseParameters) == 0 {
		return nil, fmt.Errorf("%w: a chase trigger order requires chase parameters", errPriceTrackingNotSet)
	}
	return e.PlaceAlgoOrder(ctx, arg)
}

// CancelAlgoOrder cancels up to 10 unfilled algo orders in one request; a partially successful batch returns its
// per-order results with the error
func (e *Exchange) CancelAlgoOrder(ctx context.Context, args []CancelAlgoOrderRequest) ([]AlgoOrderActionResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(args) > maxCancelAlgosPerRequest {
		return nil, fmt.Errorf("%w, cannot cancel more than %d algo orders", errExceedLimit, maxCancelAlgosPerRequest)
	}
	for x := range args {
		if args[x] == (CancelAlgoOrderRequest{}) {
			return nil, common.ErrEmptyParams
		}
		if args[x].AlgoID == "" && args[x].AlgoClientOrderID == "" {
			return nil, fmt.Errorf("%w, either algo ID or algo client order ID is required", order.ErrOrderIDNotSet)
		}
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
	}
	var resp []AlgoOrderActionResponse
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, orderWeightedEPL(cancelAlgoOrderEPL, len(args)), http.MethodPost, "trade/cancel-algos", args, &resp, request.AuthenticatedRequest)
	if err != nil && !errors.Is(err, errPartialSuccess) {
		return nil, err
	}
	return resp, err
}

// AmendAlgoOrder amends an unfilled stop or trigger algo order
func (e *Exchange) AmendAlgoOrder(ctx context.Context, arg *AmendAlgoOrderRequest) (*AmendAlgoOrderResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoID == "" && arg.AlgoClientOrderID == "" {
		return nil, fmt.Errorf("%w, either algo ID or algo client order ID is required", order.ErrOrderIDNotSet)
	}
	var resp *AmendAlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendAlgoOrderEPL, http.MethodPost, "trade/amend-algos", arg, &resp, request.AuthenticatedRequest)
}

// GetAlgoOrderDetail retrieves an algo order's details by its algo ID or algo client order ID
func (e *Exchange) GetAlgoOrderDetail(ctx context.Context, arg *AlgoOrderDetailRequest) (*AlgoOrderResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.AlgoID == "" && arg.AlgoClientOrderID == "" {
		return nil, fmt.Errorf("%w, either algo ID or algo client order ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if arg.AlgoID != "" {
		params.Set("algoId", arg.AlgoID)
	}
	if arg.AlgoClientOrderID != "" {
		params.Set("algoClOrdId", arg.AlgoClientOrderID)
	}
	var resp *AlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderDetailEPL, http.MethodGet, common.EncodeURLValues("trade/order-algo", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAlgoOrderList retrieves the account's untriggered algo orders
func (e *Exchange) GetAlgoOrderList(ctx context.Context, arg *AlgoOrderListRequest) ([]AlgoOrderResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := url.Values{}
	params.Set("ordType", strings.ToLower(arg.OrderType))
	if arg.AlgoID != "" {
		params.Set("algoId", arg.AlgoID)
	}
	if arg.AlgoClientOrderID != "" {
		params.Set("algoClOrdId", arg.AlgoClientOrderID)
	}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
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
	var resp []AlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderListEPL, http.MethodGet, common.EncodeURLValues("trade/orders-algo-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAlgoOrderHistory retrieves the account's algo orders of the last 3 months
func (e *Exchange) GetAlgoOrderHistory(ctx context.Context, arg *AlgoOrderHistoryRequest) ([]AlgoOrderResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.AlgoID == "" && arg.State == "" {
		return nil, errMissingEitherAlgoIDOrState
	}
	params := url.Values{}
	params.Set("ordType", strings.ToLower(arg.OrderType))
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.AlgoID != "" {
		params.Set("algoId", arg.AlgoID)
	}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
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
	var resp []AlgoOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("trade/orders-algo-history", params), nil, &resp, request.AuthenticatedRequest)
}
