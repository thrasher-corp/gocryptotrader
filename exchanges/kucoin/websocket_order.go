package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	kucoinOrderWebsocketURL = "wss://wsapi.kucoin.com/v1/private"
	orderConnection         = "order"
)

var (
	errOrderSessionNotWelcomed = errors.New("websocket order session not welcomed")
	errWsOrderRequestFailed    = errors.New("websocket order request failed")
	errNoOrders                = errors.New("no orders")
)

// newOrderConnectionSetup returns the setup of the websocket order API connection, which places and cancels Classic
// spot, margin and futures orders. It needs no subscriptions, and it signs its URL, so it cannot connect without
// credentials and sits out while authenticated use is off, as the engine turns it off when it finds the credentials
// refused
func (e *Exchange) newOrderConnectionSetup(exch *config.Exchange, wsURL string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         exch.WebsocketResponseMaxLimit,
		URL:                      wsURL,
		Connector:                e.WebsocketConnectOrderAPI,
		Handler:                  e.wsHandleOrderData,
		MessageFilter:            orderConnection,
		SubscriptionsNotRequired: true,
		Optional:                 true,
		RequiresAuthentication:   true,
	}
}

// WebsocketConnectOrderAPI connects to the websocket order API and authenticates the session
func (e *Exchange) WebsocketConnectOrderAPI(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	values := orderAPIQuery(creds, time.Now())
	if err := conn.Dial(ctx, e.websocketDialer(), nil, values); err != nil {
		return fmt.Errorf("%w: %w", errWebsocketDialFailed, err)
	}
	welcome, err := e.authenticateOrderSession(ctx, conn, creds.Secret)
	if err != nil {
		return err
	}
	// The order API takes websocket ping frames as heartbeats
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{MessageType: gws.PingMessage, Delay: pingDelay(welcome.PingInterval)})
	return nil
}

// orderAPIQuery returns the query that signs a websocket order API connection: the key, the time, the signature of
// the key and time, and the signed passphrase
func orderAPIQuery(creds *accounts.Credentials, now time.Time) url.Values {
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	values := url.Values{}
	values.Set("apikey", creds.Key)
	values.Set("timestamp", timestamp)
	values.Set("sign", hmacSign(creds.Key+timestamp, creds.Secret))
	values.Set("passphrase", hmacSign(creds.ClientID, creds.Secret))
	return values
}

// authenticateOrderSession signs the session message the order API sends on connecting and waits for its welcome,
// before the connection's reader starts
func (e *Exchange) authenticateOrderSession(ctx context.Context, conn websocket.Connection, secret string) (*WebsocketOrderWelcome, error) {
	session, err := readWebsocketMessage(ctx, conn, e.WebsocketResponseMaxLimit)
	if err != nil {
		return nil, err
	}
	var s *WebsocketOrderSession
	if err := json.Unmarshal(session, &s); err != nil {
		return nil, err
	}
	if s == nil {
		return nil, common.ErrNoResponse
	}
	if s.SessionID == "" {
		return nil, fmt.Errorf("%w: %s", errOrderSessionNotWelcomed, session)
	}
	if err := conn.SendRawMessage(ctx, request.Unset, gws.TextMessage, []byte(hmacSign(string(session), secret))); err != nil {
		return nil, err
	}
	data, err := readWebsocketMessage(ctx, conn, e.WebsocketResponseMaxLimit)
	if err != nil {
		return nil, err
	}
	var welcome *WebsocketOrderWelcome
	if err := json.Unmarshal(data, &welcome); err != nil {
		return nil, err
	}
	if welcome == nil {
		return nil, common.ErrNoResponse
	}
	if welcome.Data != "welcome" || welcome.SessionID != s.SessionID {
		return nil, fmt.Errorf("%w: %s", errOrderSessionNotWelcomed, data)
	}
	return welcome, nil
}

// wsHandleOrderData routes a message from the websocket order API connection to the request it answers
func (e *Exchange) wsHandleOrderData(_ context.Context, conn websocket.Connection, respData []byte) error {
	var resp *WebsocketOrderResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return err
	}
	switch {
	case resp == nil:
		return common.ErrNoResponse
	case resp.Operation == "pong":
		return nil
	case resp.ID != "":
		return conn.RequireMatchWithData(resp.ID, respData)
	}
	// Requests failing validation before their ID is read are answered without it
	return fmt.Errorf("%w: %s %s", errWsOrderRequestFailed, resp.Code, resp.Message)
}

// WebsocketAddSpotOrder places a spot order through the websocket order API (Add Order, op spot.order)
func (e *Exchange) WebsocketAddSpotOrder(ctx context.Context, req *PlaceHFOrderRequest) (*WebsocketAddOrderResponse, error) {
	args, err := req.Payload()
	if err != nil {
		return nil, err
	}
	var resp *WebsocketAddOrderResponse
	return resp, e.sendWsOrderRequest(ctx, placeHFOrderEPL, "spot.order", args, &resp)
}

// WebsocketAddSpotOrderSync places a spot order through the websocket order API and returns the order's state once placed
// (Add Order, op spot.sync_order)
func (e *Exchange) WebsocketAddSpotOrderSync(ctx context.Context, req *PlaceHFOrderRequest) (*WebsocketAddSpotOrderSyncResponse, error) {
	args, err := req.Payload()
	if err != nil {
		return nil, err
	}
	var resp *WebsocketAddSpotOrderSyncResponse
	return resp, e.sendWsOrderRequest(ctx, placeHFOrderSyncEPL, "spot.sync_order", args, &resp)
}

// WebsocketAddMarginOrder places a margin order through the websocket order API (Add Order, op margin.order)
func (e *Exchange) WebsocketAddMarginOrder(ctx context.Context, req *PlaceMarginHFOrderRequest) (*WebsocketAddMarginOrderResponse, error) {
	args, err := req.Payload()
	if err != nil {
		return nil, err
	}
	var resp *WebsocketAddMarginOrderResponse
	return resp, e.rememberMarginOrder(req.ClientOrderID, func() (string, error) {
		err := e.sendWsOrderRequest(ctx, placeMarginHFOrderEPL, "margin.order", args, &resp)
		if resp == nil {
			return "", err
		}
		return resp.OrderID, err
	})
}

// WebsocketAddFuturesOrder places a futures order through the websocket order API (Add Order, op futures.order)
func (e *Exchange) WebsocketAddFuturesOrder(ctx context.Context, req *FuturesOrderRequest) (*WebsocketAddOrderResponse, error) {
	args, err := req.Payload()
	if err != nil {
		return nil, err
	}
	var resp *WebsocketAddOrderResponse
	return resp, e.sendWsOrderRequest(ctx, postFuturesOrderEPL, "futures.order", args, &resp)
}

// WebsocketBatchAddFuturesOrders places futures orders through the websocket order API and returns each order's result (Add
// Order, op futures.multi_order)
func (e *Exchange) WebsocketBatchAddFuturesOrders(ctx context.Context, reqs []FuturesOrderRequest) ([]WebsocketFuturesOrderResponse, error) {
	args, err := futuresOrderListBody(reqs)
	if err != nil {
		return nil, err
	}
	var resp []WebsocketFuturesOrderResponse
	return resp, e.sendWsOrderRequest(ctx, placeMultipleFuturesOrdersEPL, "futures.multi_order", args, &resp)
}

// WebsocketCancelSpotOrder cancels a spot order through the websocket order API (Cancel Order, op spot.cancel)
func (e *Exchange) WebsocketCancelSpotOrder(ctx context.Context, req *WebsocketCancelOrderRequest) (*WebsocketCancelOrderResponse, error) {
	args, err := req.args(true)
	if err != nil {
		return nil, err
	}
	var resp *WebsocketCancelOrderResponse
	return resp, e.sendWsOrderRequest(ctx, cancelHFOrderByOrderIDEPL, "spot.cancel", args, &resp)
}

// WebsocketCancelSpotOrderSync cancels a spot order through the websocket order API and returns the order's state once
// cancelled (Cancel Order, op spot.sync_cancel)
func (e *Exchange) WebsocketCancelSpotOrderSync(ctx context.Context, req *WebsocketCancelOrderRequest) (*WebsocketCancelSpotOrderSyncResponse, error) {
	args, err := req.args(true)
	if err != nil {
		return nil, err
	}
	var resp *WebsocketCancelSpotOrderSyncResponse
	return resp, e.sendWsOrderRequest(ctx, cancelHFOrderByOrderIDSyncEPL, "spot.sync_cancel", args, &resp)
}

// WebsocketCancelMarginOrder cancels a margin order through the websocket order API (Cancel Order, op margin.cancel)
func (e *Exchange) WebsocketCancelMarginOrder(ctx context.Context, req *WebsocketCancelOrderRequest) (*WebsocketCancelOrderResponse, error) {
	args, err := req.args(true)
	if err != nil {
		return nil, err
	}
	var resp *WebsocketCancelOrderResponse
	return resp, e.sendWsOrderRequest(ctx, cancelMarginHFOrderByOrderIDEPL, "margin.cancel", args, &resp)
}

// WebsocketCancelFuturesOrder cancels a futures order through the websocket order API (Cancel Order, op futures.cancel). The
// symbol is only required with a client order ID
func (e *Exchange) WebsocketCancelFuturesOrder(ctx context.Context, req *WebsocketCancelOrderRequest) (*WebsocketCancelFuturesOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	args, err := req.args(req.ClientOrderID != "")
	if err != nil {
		return nil, err
	}
	var resp *WebsocketCancelFuturesOrderResponse
	return resp, e.sendWsOrderRequest(ctx, cancelFuturesOrderByOrderIDEPL, "futures.cancel", args, &resp)
}

// args returns the arguments of a cancel request, requiring the symbol when told to
func (r *WebsocketCancelOrderRequest) args(symbolRequired bool) (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if symbolRequired && r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.OrderID == "" && r.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	args := make(map[string]any, 3)
	if r.Symbol != "" {
		args["symbol"] = r.Symbol
	}
	if r.OrderID != "" {
		args["orderId"] = r.OrderID
	}
	if r.ClientOrderID != "" {
		args["clientOid"] = r.ClientOrderID
	}
	return args, nil
}

// WebsocketBatchCancelFuturesOrders cancels futures orders through the websocket order API and returns each order's result
// (Cancel Order, op futures.multi_cancel)
func (e *Exchange) WebsocketBatchCancelFuturesOrders(ctx context.Context, req *WebsocketBatchCancelFuturesOrdersRequest) ([]WebsocketFuturesOrderResponse, error) {
	args, err := req.args()
	if err != nil {
		return nil, err
	}
	var resp []WebsocketFuturesOrderResponse
	return resp, e.sendWsOrderRequest(ctx, batchCancelFuturesOrdersEPL, "futures.multi_cancel", args, &resp)
}

// args returns the arguments of a batch cancel request
func (r *WebsocketBatchCancelFuturesOrdersRequest) args() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if len(r.OrderIDs) == 0 && len(r.ClientOrders) == 0 {
		return nil, errNoOrders
	}
	args := make(map[string]any, 2)
	if len(r.OrderIDs) != 0 {
		args["orderIdsList"] = r.OrderIDs
	}
	if len(r.ClientOrders) != 0 {
		clientOrders := make([]map[string]any, len(r.ClientOrders))
		for i := range r.ClientOrders {
			if r.ClientOrders[i].Symbol == "" {
				return nil, currency.ErrSymbolStringEmpty
			}
			if r.ClientOrders[i].ClientOrderID == "" {
				return nil, order.ErrClientOrderIDMustBeSet
			}
			clientOrders[i] = map[string]any{"symbol": r.ClientOrders[i].Symbol, "clientOid": r.ClientOrders[i].ClientOrderID}
		}
		args["clientOidsList"] = clientOrders
	}
	return args, nil
}

// sendWsOrderRequest sends a request to the websocket order API and decodes its result. Order requests share the
// rate limits of the REST order endpoints, so they wait on the REST limiter
func (e *Exchange) sendWsOrderRequest(ctx context.Context, epl request.EndpointLimit, op string, args, result any) error {
	conn, err := e.Websocket.GetConnection(orderConnection)
	if err != nil {
		return err
	}
	if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
		return err
	}
	return e.sendWsOrderRequestOnConnection(ctx, conn, op, args, result)
}

// sendWsOrderRequestOnConnection sends a request on a websocket order API connection and decodes its result
func (e *Exchange) sendWsOrderRequestOnConnection(ctx context.Context, conn websocket.Connection, op string, args, result any) error {
	req := &WebsocketOrderRequest{ID: e.MessageID(), Operation: op, Args: args}
	respRaw, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return err
	}
	var resp *WebsocketOrderResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%w: %s", common.ErrNoResponse, op)
	}
	if resp.Code != "200000" {
		return fmt.Errorf("%w: %s code %s: %s", errWsOrderRequestFailed, op, resp.Code, resp.Message)
	}
	return json.Unmarshal(resp.Data, result)
}
