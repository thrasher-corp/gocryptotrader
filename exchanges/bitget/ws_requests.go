package bitget

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidInstrumentType = errors.New("invalid instrument type")
	errMarginModeUnset       = errors.New("margin mode must be set")
	errMissingValues         = errors.New("missing values")
	errValuesConflict        = errors.New("values conflict")
)

const privateConnection = "private"

// WebsocketSpotPlaceOrder places an order via the websocket connection
func (e *Exchange) WebsocketSpotPlaceOrder(ctx context.Context, req *WebsocketSpotPlaceOrderRequest) (*WebsocketSpotPlaceOrderResponse, error) {
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Size <= 0 {
		return nil, order.ErrAmountMustBeSet
	}
	if req.TimeInForce == "" {
		return nil, order.ErrInvalidTimeInForce
	}
	if req.Price <= 0 && req.OrderType == "limit" {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}

	var price string
	if req.Price > 0 {
		price = strconv.FormatFloat(req.Price, 'f', -1, 64)
	}

	var result []WebsocketSpotPlaceOrderResponse
	if err := e.sendWebsocketTradeRequest(ctx, request.Unset, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: "SPOT",
		InstrumentID:   req.Pair.String(),
		Channel:        "place-order",
		Params: WebsocketSpotOrderParams{
			OrderType:           req.OrderType,
			Side:                req.Side,
			Size:                strconv.FormatFloat(req.Size, 'f', -1, 64),
			Force:               req.TimeInForce,
			Price:               price,
			ClientOrderID:       req.ClientOrderID,
			SelfTradePrevention: req.SelfTradePrevention,
		},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrNoResponse
	}
	return &result[0], nil
}

// WebsocketSpotCancelOrder cancels an order via the websocket connection
func (e *Exchange) WebsocketSpotCancelOrder(ctx context.Context, pair currency.Pair, orderID, clientOrderID string) (*WebsocketCancelOrderResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	var result []WebsocketCancelOrderResponse
	if err := e.sendWebsocketTradeRequest(ctx, request.Unset, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: "SPOT",
		InstrumentID:   pair.String(),
		Channel:        "cancel-order",
		Params:         WebsocketIDs{OrderID: orderID, ClientOrderID: clientOrderID},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrNoResponse
	}
	return &result[0], nil
}

// WebsocketFuturesPlaceOrder places a futures order via the websocket connection
func (e *Exchange) WebsocketFuturesPlaceOrder(ctx context.Context, req *WebsocketFuturesOrderRequest) (*WebsocketFuturesPlaceOrderResponse, error) {
	if req.Contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.InstrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.ContractSize <= 0 {
		return nil, order.ErrAmountMustBeSet
	}
	if req.TimeInForce == "" {
		return nil, order.ErrInvalidTimeInForce
	}
	if req.Price <= 0 && req.OrderType == "limit" {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if req.MarginCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: margin coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.MarginMode == "" {
		return nil, errMarginModeUnset
	}
	if req.ReduceOnly == "" && req.TradeSide == "" {
		return nil, fmt.Errorf("%w: either reduce only or trade side must be set, but not both", errMissingValues)
	}
	if req.ReduceOnly != "" && req.TradeSide != "" {
		return nil, fmt.Errorf("%w: reduce only and trade side cannot both be set", errValuesConflict)
	}

	var price, takeProfit, stopLoss string
	if req.Price > 0 {
		price = strconv.FormatFloat(req.Price, 'f', -1, 64)
	}
	if req.PresetStopSurplusPrice > 0 {
		takeProfit = strconv.FormatFloat(req.PresetStopSurplusPrice, 'f', -1, 64)
	}
	if req.PresetStopLossPrice > 0 {
		stopLoss = strconv.FormatFloat(req.PresetStopLossPrice, 'f', -1, 64)
	}

	var result []WebsocketFuturesPlaceOrderResponse
	if err := e.sendWebsocketTradeRequest(ctx, request.Unset, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: req.InstrumentType,
		InstrumentID:   req.Contract.String(),
		Channel:        "place-order",
		Params: WebsocketFuturesOrderParams{
			OrderType:              req.OrderType,
			Side:                   req.Side,
			Size:                   strconv.FormatFloat(req.ContractSize, 'f', -1, 64),
			Force:                  req.TimeInForce,
			Price:                  price,
			ClientOrderID:          req.ClientOrderID,
			MarginCoin:             req.MarginCoin.String(),
			MarginMode:             req.MarginMode,
			TradeSide:              req.TradeSide,
			ReduceOnly:             req.ReduceOnly,
			PresetStopSurplusPrice: takeProfit,
			PresetStopLossPrice:    stopLoss,
			SelfTradePrevention:    req.SelfTradePrevention,
		},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrNoResponse
	}
	return &result[0], nil
}

// WebsocketFuturesCancelOrder cancels a futures order via the websocket connection
func (e *Exchange) WebsocketFuturesCancelOrder(ctx context.Context, pair currency.Pair, instrumentType, orderID, clientOrderID string) (*WebsocketCancelOrderResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if instrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}

	var result []WebsocketCancelOrderResponse
	if err := e.sendWebsocketTradeRequest(ctx, request.Unset, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: instrumentType,
		InstrumentID:   pair.String(),
		Channel:        "cancel-order",
		Params:         WebsocketIDs{OrderID: orderID, ClientOrderID: clientOrderID},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrNoResponse
	}
	return &result[0], nil
}

// sendWebsocketTradeRequest sends a trade request via the private websocket connection
func (e *Exchange) sendWebsocketTradeRequest(ctx context.Context, epl request.EndpointLimit, result any, arg *WebsocketTradeRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}

	conn, err := e.Websocket.GetConnection(privateConnection)
	if err != nil {
		return err
	}

	outbound := struct {
		Operation string `json:"op"`
		Args      []any  `json:"args"`
	}{
		Operation: "trade",
		Args:      []any{arg}, // Batch requests are not currently supported
	}

	got, err := conn.SendMessageReturnResponse(ctx, epl, arg.ID, outbound)
	if err != nil {
		return err
	}

	response := WebsocketTradeResponse{
		Result: result,
	}

	if err := json.Unmarshal(got, &response); err != nil {
		return err
	}

	if response.Event == "error" {
		return fmt.Errorf("%s %s %d: %s", e.Name, response.Event, response.Code, response.Message)
	}

	return nil
}
