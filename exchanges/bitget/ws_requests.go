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
	var result []WebsocketSpotPlaceOrderResponse
	if err := e.sendWebsocketTradeRequest(ctx, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: "SPOT",
		InstrumentID:   req.Pair.String(),
		Channel:        bitgetPlaceOrderRequest,
		Params: WebsocketSpotOrderParams{
			OrderType:           req.OrderType,
			Side:                req.Side,
			Size:                req.Size,
			Force:               req.TimeInForce,
			Price:               req.Price,
			ClientOrderID:       req.ClientOrderID,
			SelfTradePrevention: req.SelfTradePrevention,
		},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrInvalidResponse
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
	if err := e.sendWebsocketTradeRequest(ctx, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: "SPOT",
		InstrumentID:   pair.String(),
		Channel:        bitgetCancelOrderRequest,
		Params:         WebsocketIDs{OrderID: orderID, ClientOrderID: clientOrderID},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrInvalidResponse
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
	var result []WebsocketFuturesPlaceOrderResponse
	if err := e.sendWebsocketTradeRequest(ctx, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: req.InstrumentType,
		InstrumentID:   req.Contract.String(),
		Channel:        bitgetPlaceOrderRequest,
		Params: WebsocketFuturesOrderParams{
			OrderType:              req.OrderType,
			Side:                   req.Side,
			Size:                   req.ContractSize,
			Force:                  req.TimeInForce,
			Price:                  req.Price,
			ClientOrderID:          req.ClientOrderID,
			MarginCoin:             req.MarginCoin.String(),
			MarginMode:             req.MarginMode,
			TradeSide:              req.TradeSide,
			ReduceOnly:             req.ReduceOnly,
			PresetStopSurplusPrice: req.PresetStopSurplusPrice,
			PresetStopLossPrice:    req.PresetStopLossPrice,
			SelfTradePrevention:    req.SelfTradePrevention,
		},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrInvalidResponse
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
	if err := e.sendWebsocketTradeRequest(ctx, &result, &WebsocketTradeRequest{
		ID:             strconv.FormatInt(e.GetBase().MessageSequence(), 10),
		InstrumentType: instrumentType,
		InstrumentID:   pair.String(),
		Channel:        bitgetCancelOrderRequest,
		Params:         WebsocketIDs{OrderID: orderID, ClientOrderID: clientOrderID},
	}); err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, common.ErrInvalidResponse
	}
	return &result[0], nil
}

// sendWebsocketTradeRequest sends a trade request via the private websocket connection
func (e *Exchange) sendWebsocketTradeRequest(ctx context.Context, result any, arg *WebsocketTradeRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}

	conn, err := e.Websocket.GetConnection(privateConnection)
	if err != nil {
		return err
	}

	outbound := struct {
		Operation string                   `json:"op"`
		Args      []*WebsocketTradeRequest `json:"args"`
	}{
		Operation: "trade",
		Args:      []*WebsocketTradeRequest{arg}, // Batch requests are not currently supported by the exchange
	}

	got, err := conn.SendMessageReturnResponse(ctx, request.Unset, arg.ID, outbound)
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
