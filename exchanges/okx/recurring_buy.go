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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errStrategyNameRequired       = errors.New("strategy name required")
	errRecurringRatioRequired     = errors.New("recurring currency ratio is required")
	errRecurringPeriodRequired    = errors.New("recurring buy period is required")
	errRecurringDayRequired       = errors.New("recurring day is required")
	errRecurringHourRequired      = errors.New("recurring hour is required")
	errRecurringBuyTimeRequired   = errors.New("recurring buy time, the value range is an integer with value between 0 and 23")
	errInvalidTimeZone            = errors.New("time zone must be an integer between -12 and 14")
	errRecurringTimeTypeRequired  = errors.New("recurring buy time type is required")
	errRecurringListRequired      = errors.New("recurring list is required")
	errRecurringBuyAmountRequired = errors.New("recurring buy amount is required")
)

const (
	maxStopRecurringBuyOrders = 10

	recurringBuyPeriodMonthly = "monthly"
	recurringBuyPeriodWeekly  = "weekly"
	recurringBuyPeriodHourly  = "hourly"

	recurringBuyTimeTypeCustom = "1"
)

// PlaceRecurringBuyOrder places a recurring buy order, which invests a fixed amount in one or more currencies at
// fixed intervals
func (e *Exchange) PlaceRecurringBuyOrder(ctx context.Context, arg *PlaceRecurringBuyOrderRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.StrategyName == "" {
		return nil, errStrategyNameRequired
	}
	if len(arg.RecurringList) == 0 {
		return nil, errRecurringListRequired
	}
	for i := range arg.RecurringList {
		if arg.RecurringList[i].Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if arg.RecurringList[i].Ratio <= 0 {
			return nil, errRecurringRatioRequired
		}
	}
	if err := validateRecurringBuySchedule(arg.Period, arg.RecurringDay, arg.RecurringHour); err != nil {
		return nil, err
	}
	if arg.RecurringTime > 23 {
		return nil, errRecurringBuyTimeRequired
	}
	if arg.TimeZone < -12 || arg.TimeZone > 14 {
		return nil, errInvalidTimeZone
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountMustBeSet
	}
	if arg.InvestmentCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: investment currency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.TradeMode == "" {
		return nil, errInvalidTradeModeValue
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, placeRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/order-algo", arg, &resp, request.AuthenticatedRequest)
}

// validateRecurringBuySchedule checks a recurring buy period carries the day or hour interval it needs
func validateRecurringBuySchedule(period string, recurringDay, recurringHour uint64) error {
	switch period {
	case "":
		return errRecurringPeriodRequired
	case recurringBuyPeriodMonthly, recurringBuyPeriodWeekly:
		if recurringDay == 0 {
			return errRecurringDayRequired
		}
	case recurringBuyPeriodHourly:
		if recurringHour == 0 {
			return errRecurringHourRequired
		}
	}
	return nil
}

// AmendRecurringBuyOrder renames a recurring buy order
func (e *Exchange) AmendRecurringBuyOrder(ctx context.Context, arg *AmendRecurringBuyOrderRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.StrategyName == "" {
		return nil, errStrategyNameRequired
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/amend-order-algo", arg, &resp, request.AuthenticatedRequest)
}

// StopRecurringBuyOrder stops up to 10 recurring buy orders. When OKX rejects any of them the rows it returned come
// back beside the error
func (e *Exchange) StopRecurringBuyOrder(ctx context.Context, algoIDs []string) ([]TradingBotOrderResponse, error) {
	if len(algoIDs) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(algoIDs) > maxStopRecurringBuyOrders {
		return nil, fmt.Errorf("%w: cannot stop more than %d recurring buy orders", errExceedLimit, maxStopRecurringBuyOrders)
	}
	args := make([]TradingBotAlgoIDRequest, len(algoIDs))
	for i := range algoIDs {
		if algoIDs[i] == "" {
			return nil, errAlgoIDRequired
		}
		args[i].AlgoID = algoIDs[i]
	}
	var resp []TradingBotOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, stopRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/stop-order-algo", args, &resp, request.AuthenticatedRequest); err != nil {
		for i := range resp {
			err = common.AppendError(err, getStatusError(resp[i].StatusCode, resp[i].StatusMessage))
		}
		return resp, err
	}
	return resp, nil
}

// GetRecurringBuyOrderList retrieves pending recurring buy orders
func (e *Exchange) GetRecurringBuyOrderList(ctx context.Context, arg *RecurringBuyOrderListRequest) ([]RecurringBuyOrderResponse, error) {
	params, err := recurringBuyOrderListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []RecurringBuyOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderListEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetRecurringBuyOrderHistory retrieves stopped recurring buy orders
func (e *Exchange) GetRecurringBuyOrderHistory(ctx context.Context, arg *RecurringBuyOrderListRequest) ([]RecurringBuyOrderResponse, error) {
	params, err := recurringBuyOrderListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []RecurringBuyOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-history", params), nil, &resp, request.AuthenticatedRequest)
}

// recurringBuyOrderListParams validates a recurring buy order list request and returns its query parameters
func recurringBuyOrderListParams(arg *RecurringBuyOrderListRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
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

// GetRecurringBuyOrderDetails retrieves a recurring buy order's details
func (e *Exchange) GetRecurringBuyOrderDetails(ctx context.Context, algoID string) (*RecurringOrderDetailResponse, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	var resp *RecurringOrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderDetailEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetRecurringBuySubOrders retrieves a recurring buy order's sub orders
func (e *Exchange) GetRecurringBuySubOrders(ctx context.Context, arg *RecurringBuySubOrdersRequest) ([]RecurringBuySubOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoId", arg.AlgoID)
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
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
	var resp []RecurringBuySubOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuySubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/sub-orders", params), nil, &resp, request.AuthenticatedRequest)
}

// AmendRecurringBuyTime amends a recurring buy order's schedule
func (e *Exchange) AmendRecurringBuyTime(ctx context.Context, arg *AmendRecurringBuyTimeRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.RecurringTimeType == "" {
		return nil, errRecurringTimeTypeRequired
	}
	if arg.TimeZone < -12 || arg.TimeZone > 14 {
		return nil, errInvalidTimeZone
	}
	if arg.Period == "" {
		return nil, errRecurringPeriodRequired
	}
	if arg.Period == recurringBuyPeriodHourly && arg.RecurringHour == 0 {
		return nil, errRecurringHourRequired
	}
	if arg.RecurringTimeType == recurringBuyTimeTypeCustom {
		if (arg.Period == recurringBuyPeriodMonthly || arg.Period == recurringBuyPeriodWeekly) && arg.RecurringDay == 0 {
			return nil, errRecurringDayRequired
		}
		if arg.RecurringTime == nil {
			return nil, errRecurringBuyTimeRequired
		}
	}
	if arg.RecurringTime != nil && *arg.RecurringTime > 23 {
		return nil, errRecurringBuyTimeRequired
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendRecurringBuyTimeEPL, http.MethodPost, "tradingBot/recurring/amend-recurring-time", arg, &resp, request.AuthenticatedRequest)
}

// AmendRecurringBuyAmount amends the amount a recurring buy order invests each cycle
func (e *Exchange) AmendRecurringBuyAmount(ctx context.Context, arg *RecurringBuyAmountRequest) (*TradingBotOrderResponse, error) {
	if err := validateRecurringBuyAmountRequest(arg); err != nil {
		return nil, err
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendRecurringBuyAmountEPL, http.MethodPost, "tradingBot/recurring/amend-recurring-amount", arg, &resp, request.AuthenticatedRequest)
}

// AddRecurringBuyInvestment adds investment to a recurring buy order
func (e *Exchange) AddRecurringBuyInvestment(ctx context.Context, arg *RecurringBuyAmountRequest) (*TradingBotOrderResponse, error) {
	if err := validateRecurringBuyAmountRequest(arg); err != nil {
		return nil, err
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, addRecurringBuyInvestmentEPL, http.MethodPost, "tradingBot/recurring/add-investment", arg, &resp, request.AuthenticatedRequest)
}

// validateRecurringBuyAmountRequest checks a recurring buy amount request carries its algo ID and amount
func validateRecurringBuyAmountRequest(arg *RecurringBuyAmountRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return errAlgoIDRequired
	}
	if arg.Amount <= 0 {
		return errRecurringBuyAmountRequired
	}
	return nil
}

// PauseRecurringBuy pauses a recurring buy order
func (e *Exchange) PauseRecurringBuy(ctx context.Context, algoID string) (*TradingBotOrderResponse, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, pauseRecurringBuyEPL, http.MethodPost, "tradingBot/recurring/pause", &TradingBotAlgoIDRequest{AlgoID: algoID}, &resp, request.AuthenticatedRequest)
}

// RestartRecurringBuy restarts a paused recurring buy order
func (e *Exchange) RestartRecurringBuy(ctx context.Context, algoID string) (*TradingBotOrderResponse, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, restartRecurringBuyEPL, http.MethodPost, "tradingBot/recurring/restart", &TradingBotAlgoIDRequest{AlgoID: algoID}, &resp, request.AuthenticatedRequest)
}

// AmendRecurringBuyPriceRange amends the price ranges a recurring buy order buys its currencies within
func (e *Exchange) AmendRecurringBuyPriceRange(ctx context.Context, arg *AmendRecurringBuyPriceRangeRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if len(arg.RecurringList) == 0 {
		return nil, errRecurringListRequired
	}
	for i := range arg.RecurringList {
		if arg.RecurringList[i].Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendRecurringBuyPriceRangeEPL, http.MethodPost, "tradingBot/recurring/amend-price-range", arg, &resp, request.AuthenticatedRequest)
}
