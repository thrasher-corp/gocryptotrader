package htx

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsHandleUSDTMarginedPrivateMessage decodes authenticated V5 USDT notifications.
func (e *Exchange) wsHandleUSDTMarginedPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	var response any
	switch sub.Channel {
	case subscription.MyOrdersChannel:
		response = new(V5WsOrderUpdate)
	case wsTradeUpdatesChannel:
		response = new(V5WsTradeUpdate)
	case wsExecutionDetailsChannel:
		response = new(V5WsTradeDetailUpdate)
	case wsPositionsChannel:
		response = new(V5WsPositionUpdate)
	case subscription.MyAccountChannel:
		response = new(V5WsAccountUpdate)
	case subscription.MyTradesChannel:
		response = new(V5WsMatchOrderUpdate)
	case wsTriggerOrdersChannel:
		response = new(V5WsAlgoOrderUpdate)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, response)
}

// WSPlaceV5Order places an order through the dedicated V5 trade connection.
func (e *Exchange) WSPlaceV5Order(ctx context.Context, req *V5OrderRequest) (*V5WsOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	conn, err := e.Websocket.GetConnection(exchange.WebsocketTrade)
	if err != nil {
		return nil, err
	}
	cid := e.MessageID()
	raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, cid, &V5WsTradeRequest{Operation: "place_order", CID: cid, Data: req})
	if err != nil {
		return nil, err
	}
	if err := getErrResp(raw); err != nil {
		return nil, err
	}
	var resp *V5WsOrderResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSPlaceV5BatchOrders places multiple orders through the dedicated V5 trade connection.
func (e *Exchange) WSPlaceV5BatchOrders(ctx context.Context, req []*V5OrderRequest) (*V5WsBatchOrderResponse, error) {
	if len(req) == 0 {
		return nil, common.ErrEmptyParams
	}
	conn, err := e.Websocket.GetConnection(exchange.WebsocketTrade)
	if err != nil {
		return nil, err
	}
	cid := e.MessageID()
	raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, cid, &V5WsTradeRequest{Operation: "place_batch_orders", CID: cid, Data: req})
	if err != nil {
		return nil, err
	}
	if err := getErrResp(raw); err != nil {
		return nil, err
	}
	var resp *V5WsBatchOrderResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSCancelV5Order cancels an order through the dedicated V5 trade connection.
func (e *Exchange) WSCancelV5Order(ctx context.Context, req *V5CancelOrderRequest) (*V5WsOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	conn, err := e.Websocket.GetConnection(exchange.WebsocketTrade)
	if err != nil {
		return nil, err
	}
	cid := e.MessageID()
	raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, cid, &V5WsTradeRequest{Operation: "cancel_order", CID: cid, Data: req})
	if err != nil {
		return nil, err
	}
	if err := getErrResp(raw); err != nil {
		return nil, err
	}
	var resp *V5WsOrderResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSCancelV5BatchOrders cancels multiple orders through the dedicated V5 trade connection.
func (e *Exchange) WSCancelV5BatchOrders(ctx context.Context, req *V5CancelBatchOrdersRequest) (*V5WsBatchOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	conn, err := e.Websocket.GetConnection(exchange.WebsocketTrade)
	if err != nil {
		return nil, err
	}
	cid := e.MessageID()
	raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, cid, &V5WsTradeRequest{Operation: "cancel_batch_orders", CID: cid, Data: req})
	if err != nil {
		return nil, err
	}
	if err := getErrResp(raw); err != nil {
		return nil, err
	}
	var resp *V5WsBatchOrderResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// WSCancelAllV5Orders cancels all matching orders through the dedicated V5 trade connection.
func (e *Exchange) WSCancelAllV5Orders(ctx context.Context, req *V5CancelAllOrdersRequest) (*V5WsBatchOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	conn, err := e.Websocket.GetConnection(exchange.WebsocketTrade)
	if err != nil {
		return nil, err
	}
	cid := e.MessageID()
	raw, err := conn.SendMessageReturnResponse(ctx, request.Unset, cid, &V5WsTradeRequest{Operation: "cancel_all_orders", CID: cid, Data: req})
	if err != nil {
		return nil, err
	}
	if err := getErrResp(raw); err != nil {
		return nil, err
	}
	var resp *V5WsBatchOrderResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
