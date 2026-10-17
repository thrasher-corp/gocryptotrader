package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// The types of the input requests, and of the notifications answering them
const (
	wsOrderNew                  = "on"
	wsOrderUpdate               = "ou"
	wsOrderCancel               = "oc"
	wsOrderCancelMulti          = "oc_multi"
	wsFundingOfferNew           = "fon"
	wsFundingOfferCancel        = "foc"
	wsCalc                      = "calc"
	wsOrderNewRequest           = wsOrderNew + "-req"
	wsOrderUpdateRequest        = wsOrderUpdate + "-req"
	wsOrderCancelRequest        = wsOrderCancel + "-req"
	wsOrderCancelMultiRequest   = wsOrderCancelMulti + "-req"
	wsFundingOfferNewRequest    = wsFundingOfferNew + "-req"
	wsFundingOfferCancelRequest = wsFundingOfferCancel + "-req"

	// wsMaxCalcRequests is how many calculations one calc request may ask for
	wsMaxCalcRequests = 30
)

var (
	errWebsocketNotAuthenticated = errors.New("websocket not authenticated")
	errCalcRequestsRequired      = errors.New("calc requests required")
	errTooManyCalcRequests       = errors.New("too many calc requests")
)

// wsInput is an input request: [0, TYPE, null, DATA]
type wsInput struct {
	ChannelID int64  `tuple:"0"`
	Type      string `tuple:"1"`
	Data      any    `tuple:"3"`
}

// MarshalJSON encodes an input request array
func (w *wsInput) MarshalJSON() ([]byte, error) { return marshalTuple(w) }

// WebsocketPlaceOrder places an order through the authenticated websocket, returning the order Bitfinex placed. A
// request without a client order ID is given one, which the notification answering it is matched by
func (e *Exchange) WebsocketPlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*Order, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	placed := *req
	if placed.ClientOrderID == 0 {
		placed.ClientOrderID = uint64(e.wsClientOrderID.GetAndIncrement(func() int64 { return time.Now().UnixMilli() })) //nolint:gosec // A millisecond time is positive
	}
	body, err := placed.body()
	if err != nil {
		return nil, err
	}
	n, err := wsSendInput[Order](ctx, e, wsOrderNew, wsInputSignature(wsOrderNewRequest, strconv.FormatUint(placed.ClientOrderID, 10)), body)
	if err != nil {
		return nil, err
	}
	return &n.Data, nil
}

// WebsocketUpdateOrder updates an order through the authenticated websocket, returning the order as Bitfinex updated it
func (e *Exchange) WebsocketUpdateOrder(ctx context.Context, req *UpdateOrderRequest) (*Order, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	n, err := wsSendInput[Order](ctx, e, wsOrderUpdate, wsInputSignature(wsOrderUpdateRequest, strconv.FormatUint(req.ID, 10)), body)
	if err != nil {
		return nil, err
	}
	return &n.Data, nil
}

// WebsocketCancelExistingOrder submits an order for cancellation through the authenticated websocket; Bitfinex confirms
// the cancellation afterwards with the order's close event
func (e *Exchange) WebsocketCancelExistingOrder(ctx context.Context, orderID uint64) (*Order, error) {
	if orderID == 0 {
		return nil, errOrderIDRequired
	}
	n, err := wsSendInput[Order](ctx, e, wsOrderCancel, wsInputSignature(wsOrderCancelRequest, strconv.FormatUint(orderID, 10)), &cancelOrderBody{ID: orderID})
	if err != nil {
		return nil, err
	}
	return &n.Data, nil
}

// WebsocketCancelOrders submits orders for cancellation through the authenticated websocket, by their IDs, group IDs or
// client order IDs, or every open order, trading and derivative alike; Bitfinex confirms each cancellation afterwards
// with the order's close event
func (e *Exchange) WebsocketCancelOrders(ctx context.Context, req *CancelOrdersRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	body, err := req.body()
	if err != nil {
		return err
	}
	// Bitfinex's notification identifies none of the orders, so one request is sent at a time
	e.wsInputMu.Lock()
	defer e.wsInputMu.Unlock()
	_, err = wsSendInput[json.RawMessage](ctx, e, wsOrderCancelMulti, wsOrderCancelMultiRequest, body)
	return err
}

// WebsocketSubmitFundingOffer submits a funding offer through the authenticated websocket, returning the offer as
// Bitfinex accepted it, which holds its ID, amount, rate and period
func (e *Exchange) WebsocketSubmitFundingOffer(ctx context.Context, req *SubmitFundingOfferRequest) (*FundingOffer, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	// Bitfinex numbers the offer itself, so its notification is matched by type, one request at a time
	e.wsInputMu.Lock()
	defer e.wsInputMu.Unlock()
	n, err := wsSendInput[FundingOffer](ctx, e, wsFundingOfferNew, wsFundingOfferNewRequest, body)
	if err != nil {
		return nil, err
	}
	return &n.Data, nil
}

// WebsocketCancelFundingOffer cancels a funding offer through the authenticated websocket
func (e *Exchange) WebsocketCancelFundingOffer(ctx context.Context, offerID uint64) (*FundingOffer, error) {
	if offerID == 0 {
		return nil, errFundingOfferIDRequired
	}
	n, err := wsSendInput[FundingOffer](ctx, e, wsFundingOfferCancel, wsInputSignature(wsFundingOfferCancelRequest, strconv.FormatUint(offerID, 10)), &fundingIDBody{ID: offerID})
	if err != nil {
		return nil, err
	}
	return &n.Data, nil
}

// WebsocketCalc asks Bitfinex to recalculate values it sends as null until calculated, such as wallets' available
// balances; the results arrive as account events. Each request names a calculation, such as margin_base,
// margin_sym_tBTCUSD, funding_sym_fUSD, position_tBTCUSD, wallet_exchange_BTC or balance
func (e *Exchange) WebsocketCalc(ctx context.Context, requests ...string) error {
	if len(requests) == 0 {
		return errCalcRequestsRequired
	}
	if len(requests) > wsMaxCalcRequests {
		return fmt.Errorf("%w: %d, at most %d", errTooManyCalcRequests, len(requests), wsMaxCalcRequests)
	}
	conn, err := e.wsAccountConn()
	if err != nil {
		return err
	}
	calcs := make([][1]string, len(requests))
	for i := range requests {
		calcs[i] = [1]string{requests[i]}
	}
	return conn.SendJSONMessage(ctx, request.Unset, &wsInput{Type: wsCalc, Data: calcs})
}

// wsSendInput sends an input request through the account connection, returning the notification answering it, which
// is matched by signature
func wsSendInput[T any](ctx context.Context, e *Exchange, inputType, signature string, data any) (*Notification[T], error) {
	conn, err := e.wsAccountConn()
	if err != nil {
		return nil, err
	}
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, signature, &wsInput{Type: inputType, Data: data})
	if err != nil {
		return nil, err
	}
	var f wsTypedFrame[Notification[T]]
	if err := f.UnmarshalJSON(resp); err != nil {
		return nil, err
	}
	if err := f.Data.Err(); err != nil {
		return nil, err
	}
	return &f.Data, nil
}

// wsAccountConn returns the account connection, once authenticated
func (e *Exchange) wsAccountConn() (websocket.Connection, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, errWebsocketNotAuthenticated
	}
	return e.Websocket.GetConnection(wsAccountConnection)
}
