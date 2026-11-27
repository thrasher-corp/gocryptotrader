package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errSignalChannelNameRequired      = errors.New("signal channel name is required")
	errSignalSourceTypeRequired       = errors.New("signal source type is required")
	errSignalChannelIDRequired        = errors.New("signal channel ID is required")
	errTakeProfitStopLossTypeRequired = errors.New("take profit and stop loss type is required")
	errSignalOrderIDRequired          = errors.New("signal order ID is required")
)

const maxCancelSignalBots = 10

// CreateSignal creates a signal channel
func (e *Exchange) CreateSignal(ctx context.Context, arg *CreateSignalRequest) (*CreateSignalResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SignalChannelName == "" {
		return nil, errSignalChannelNameRequired
	}
	var resp *CreateSignalResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createSignalEPL, http.MethodPost, "tradingBot/signal/create-signal", arg, &resp, request.AuthenticatedRequest)
}

// GetSignals retrieves signal channels
func (e *Exchange) GetSignals(ctx context.Context, arg *SignalsRequest) ([]SignalResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SignalSourceType == "" {
		return nil, errSignalSourceTypeRequired
	}
	params, err := signalBotTimeCursorParams(arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	params.Set("signalSourceType", arg.SignalSourceType)
	if arg.SignalChannelID != "" {
		params.Set("signalChanId", arg.SignalChannelID)
	}
	var resp []SignalResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSignalsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/signals", params), nil, &resp, request.AuthenticatedRequest)
}

// CreateSignalBot creates a signal bot trading a signal channel's signals
func (e *Exchange) CreateSignalBot(ctx context.Context, arg *CreateSignalBotRequest) (*TradingBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SignalChannelID == "" {
		return nil, errSignalChannelIDRequired
	}
	if arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	if arg.InvestmentAmount <= 0 {
		return nil, fmt.Errorf("%w: investment amount", order.ErrAmountMustBeSet)
	}
	if arg.SubOrderType == "" {
		return nil, errMissingSubOrderType
	}
	var resp *TradingBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createSignalBotEPL, http.MethodPost, "tradingBot/signal/order-algo", arg, &resp, request.AuthenticatedRequest)
}

// CancelSignalBots stops up to 10 signal bots. When OKX rejects any of them the rows it returned come back beside
// the error
func (e *Exchange) CancelSignalBots(ctx context.Context, algoIDs []string) ([]TradingBotOrderResponse, error) {
	if len(algoIDs) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(algoIDs) > maxCancelSignalBots {
		return nil, fmt.Errorf("%w: cannot cancel more than %d signal bots", errExceedLimit, maxCancelSignalBots)
	}
	args := make([]TradingBotAlgoIDRequest, len(algoIDs))
	for i := range algoIDs {
		if algoIDs[i] == "" {
			return nil, errAlgoIDRequired
		}
		args[i].AlgoID = algoIDs[i]
	}
	var resp []TradingBotOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, cancelSignalBotsEPL, http.MethodPost, "tradingBot/signal/stop-order-algo", args, &resp, request.AuthenticatedRequest); err != nil {
		for i := range resp {
			err = common.AppendError(err, getStatusError(resp[i].StatusCode, resp[i].StatusMessage))
		}
		return resp, err
	}
	return resp, nil
}

// AdjustSignalBotMarginBalance adds or reduces a signal bot's margin balance
func (e *Exchange) AdjustSignalBotMarginBalance(ctx context.Context, arg *AdjustSignalBotMarginBalanceRequest) (*TradingBotAlgoIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AdjustMarginBalanceType == "" {
		return nil, errMarginBalanceTypeRequired
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountMustBeSet
	}
	var resp *TradingBotAlgoIDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, adjustSignalBotMarginBalanceEPL, http.MethodPost, "tradingBot/signal/margin-balance", arg, &resp, request.AuthenticatedRequest)
}

// AmendSignalBotTakeProfitStopLoss amends a signal bot's take profit and stop loss settings
func (e *Exchange) AmendSignalBotTakeProfitStopLoss(ctx context.Context, arg *AmendSignalBotTakeProfitStopLossRequest) (*TradingBotAlgoIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.ExitSetting.TakeProfitStopLossType == "" {
		return nil, errTakeProfitStopLossTypeRequired
	}
	var resp *TradingBotAlgoIDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, amendSignalBotTakeProfitStopLossEPL, http.MethodPost, "tradingBot/signal/amendTPSL", arg, &resp, request.AuthenticatedRequest)
}

// SetSignalBotInstruments sets the instruments a signal bot trades
func (e *Exchange) SetSignalBotInstruments(ctx context.Context, arg *SetSignalBotInstrumentsRequest) (*TradingBotAlgoIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if !arg.IncludeAll && len(arg.InstrumentIDs) == 0 {
		return nil, fmt.Errorf("%w: instrument IDs are required unless every USDT margined contract is included", errMissingInstrumentID)
	}
	var resp *TradingBotAlgoIDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, setSignalBotInstrumentsEPL, http.MethodPost, "tradingBot/signal/set-instruments", arg, &resp, request.AuthenticatedRequest)
}

// GetSignalBotOrderDetails retrieves a signal bot's details
func (e *Exchange) GetSignalBotOrderDetails(ctx context.Context, arg *SignalBotOrderRequest) (*SignalBotOrderResponse, error) {
	params, err := signalBotOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *SignalBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, signalBotOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/orders-algo-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetActiveSignalBots retrieves active signal bots
func (e *Exchange) GetActiveSignalBots(ctx context.Context, arg *SignalBotListRequest) ([]SignalBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := signalBotListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []SignalBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getActiveSignalBotsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/orders-algo-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSignalBotHistory retrieves a stopped signal bot
func (e *Exchange) GetSignalBotHistory(ctx context.Context, arg *SignalBotListRequest) ([]SignalBotOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	params, err := signalBotListParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []SignalBotOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSignalBotHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/orders-algo-history", params), nil, &resp, request.AuthenticatedRequest)
}

// signalBotListParams validates a signal bot list request and returns its query parameters
func signalBotListParams(arg *SignalBotListRequest) (url.Values, error) {
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	params, err := signalBotTimeCursorParams(arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	params.Set("algoOrdType", arg.AlgoOrderType)
	if arg.AlgoID != "" {
		params.Set("algoId", arg.AlgoID)
	}
	return params, nil
}

// GetSignalBotOrderPositions retrieves a signal bot's positions
func (e *Exchange) GetSignalBotOrderPositions(ctx context.Context, arg *SignalBotOrderRequest) ([]TradingBotPositionResponse, error) {
	params, err := signalBotOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []TradingBotPositionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, signalBotOrderPositionsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/positions", params), nil, &resp, request.AuthenticatedRequest)
}

// signalBotOrderParams validates a signal bot identifier and returns its query parameters
func signalBotOrderParams(arg *SignalBotOrderRequest) (url.Values, error) {
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

// GetSignalBotPositionHistory retrieves a signal bot's positions updated in the last three months, latest first
func (e *Exchange) GetSignalBotPositionHistory(ctx context.Context, arg *SignalBotPositionHistoryRequest) ([]SignalBotPositionHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	params, err := signalBotTimeCursorParams(arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	params.Set("algoId", arg.AlgoID)
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []SignalBotPositionHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSignalBotPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/positions-history", params), nil, &resp, request.AuthenticatedRequest)
}

// CloseSignalBotPosition closes a signal bot's position in an instrument at market
func (e *Exchange) CloseSignalBotPosition(ctx context.Context, arg *CloseSignalBotPositionRequest) (*TradingBotAlgoIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp *TradingBotAlgoIDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, closeSignalBotPositionEPL, http.MethodPost, "tradingBot/signal/close-position", arg, &resp, request.AuthenticatedRequest)
}

// PlaceSignalBotSubOrder places a sub order for a signal bot; OKX documents no fields for its reply
func (e *Exchange) PlaceSignalBotSubOrder(ctx context.Context, arg *PlaceSignalBotSubOrderRequest) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	if arg.AlgoID == "" {
		return errAlgoIDRequired
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return order.ErrAmountMustBeSet
	}
	if arg.OrderType == orderLimit && arg.Price <= 0 {
		return limits.ErrPriceBelowMin
	}
	var resp []json.RawMessage
	return e.SendHTTPRequest(ctx, exchange.RestSpot, placeSignalBotSubOrderEPL, http.MethodPost, "tradingBot/signal/sub-order", arg, &resp, request.AuthenticatedRequest)
}

// CancelSignalBotSubOrder cancels a signal bot's incomplete sub order
func (e *Exchange) CancelSignalBotSubOrder(ctx context.Context, arg *CancelSignalBotSubOrderRequest) (*CancelSignalBotSubOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.SignalOrderID == "" {
		return nil, errSignalOrderIDRequired
	}
	var resp *CancelSignalBotSubOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelSignalBotSubOrderEPL, http.MethodPost, "tradingBot/signal/cancel-sub-order", arg, &resp, request.AuthenticatedRequest)
}

// GetSignalBotSubOrders retrieves a signal bot's sub orders by state or by sub order ID
func (e *Exchange) GetSignalBotSubOrders(ctx context.Context, arg *SignalBotSubOrdersRequest) ([]SignalBotSubOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AlgoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if arg.State == "" && arg.SignalOrderID == "" && arg.SubOrderType == "" && arg.ClientOrderID == "" {
		return nil, fmt.Errorf("%w: either a sub order state or a sub order ID is required", order.ErrOrderIDNotSet)
	}
	if !arg.Begin.IsZero() && !arg.End.IsZero() && arg.Begin.After(arg.End) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("algoId", arg.AlgoID)
	params.Set("algoOrdType", arg.AlgoOrderType)
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.SignalOrderID != "" {
		params.Set("signalOrdId", arg.SignalOrderID)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if !arg.Begin.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.SubOrderType != "" {
		params.Set("type", arg.SubOrderType)
	}
	if arg.ClientOrderID != "" {
		params.Set("clOrdId", arg.ClientOrderID)
	}
	var resp []SignalBotSubOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, signalBotSubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/sub-orders", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSignalBotEventHistory retrieves a signal bot's events
func (e *Exchange) GetSignalBotEventHistory(ctx context.Context, arg *SignalBotEventHistoryRequest) ([]SignalBotEventHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	params, err := signalBotTimeCursorParams(arg.After, arg.Before, arg.Limit)
	if err != nil {
		return nil, err
	}
	params.Set("algoId", arg.AlgoID)
	var resp []SignalBotEventHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, signalBotEventHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/event-history", params), nil, &resp, request.AuthenticatedRequest)
}

// signalBotTimeCursorParams returns the paging parameters of a signal bot list that pages by Unix millisecond
// timestamps: after returns earlier records and before newer ones, so before must not be later than after
func signalBotTimeCursorParams(after, before time.Time, limit uint64) (url.Values, error) {
	if !after.IsZero() && !before.IsZero() && before.After(after) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}
