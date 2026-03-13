package deribit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type deribitMockConn struct {
	websocket.Connection
	subscriptions *subscription.Store
	errByMethod   map[string]error
	respByMethod  map[string][]byte
}

func newDeribitMockConn() *deribitMockConn {
	return &deribitMockConn{
		subscriptions: subscription.NewStore(),
		errByMethod:   make(map[string]error),
		respByMethod:  make(map[string][]byte),
	}
}

func (m *deribitMockConn) Dial(context.Context, *gws.Dialer, http.Header) error { return nil }
func (m *deribitMockConn) ReadMessage() websocket.Response                      { return websocket.Response{} }
func (m *deribitMockConn) SetupPingHandler(request.EndpointLimit, websocket.PingHandler) {
}
func (m *deribitMockConn) Subscriptions() *subscription.Store { return m.subscriptions }
func (m *deribitMockConn) Shutdown() error                    { return nil }
func (m *deribitMockConn) GetURL() string                     { return "mock://deribit" }
func (m *deribitMockConn) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _,
	req any,
) ([]byte, error) {
	switch r := req.(type) {
	case wsInput:
		return []byte(`{"jsonrpc":"2.0","id":"` + r.ID + `","result":"ok"}`), nil
	case WsSubscriptionInput:
		out := wsSubscriptionResponse{
			JSONRPCVersion: rpcVersion,
			ID:             r.ID,
			Method:         r.Method,
			Result:         r.Params["channels"],
		}
		return json.Marshal(out)
	case *WsRequest:
		if err := m.errByMethod[r.Method]; err != nil {
			return nil, err
		}
		if raw, ok := m.respByMethod[r.Method]; ok {
			return raw, nil
		}
		return []byte(`{"jsonrpc":"2.0","id":"` + r.ID + `","result":{}}`), nil
	default:
		return nil, fmt.Errorf("unsupported request type %T", req)
	}
}

func connectWithMockedWebsocket(t *testing.T, ex *Exchange, conn websocket.Connection) {
	t.Helper()
	ex.Websocket.Conn = conn
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(t, ex.Websocket.Connect(t.Context()))
	t.Cleanup(func() {
		_ = ex.Websocket.Shutdown()
	})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	sub := &order.Submit{
		Exchange:    ex.Name,
		Pair:        optionsTradablePair,
		AssetType:   asset.Options,
		Side:        order.Buy,
		Type:        order.Limit,
		Amount:      1,
		Price:       1,
		QuoteAmount: 1,
	}
	_, err := ex.WebsocketSubmitOrder(t.Context(), sub)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	// Wrapper websocket usage requires both authenticated endpoints and an active websocket connection.
	// Setting auth capability alone is insufficient.
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	_, err = ex.WebsocketSubmitOrder(t.Context(), sub)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	modify := &order.Modify{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
		Amount:    1,
	}
	_, err := ex.WebsocketModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	_, err = ex.WebsocketModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))

	cancel := &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
	}
	err := ex.WebsocketCancelOrder(t.Context(), cancel)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	err = ex.WebsocketCancelOrder(t.Context(), cancel)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestSymbolChannelSeparator(t *testing.T) {
	t.Parallel()

	assert.Empty(t, symbolChannelSeparator(&subscription.Subscription{Channel: subscription.MyAccountChannel}))
	assert.Equal(t, ".", symbolChannelSeparator(&subscription.Subscription{Channel: subscription.MyOrdersChannel}))
}

func TestWebsocketSubmitOrderMocked(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	mockConn := newDeribitMockConn()
	connectWithMockedWebsocket(t, ex, mockConn)

	sub := &order.Submit{
		Exchange:  ex.Name,
		Pair:      optionsTradablePair,
		AssetType: asset.Options,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     1,
	}

	_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{})
	require.Error(t, err)

	unsupported := *sub
	unsupported.AssetType = asset.Binary
	_, err = ex.WebsocketSubmitOrder(t.Context(), &unsupported)
	require.ErrorContains(t, err, "orderType binary is not valid")

	badSide := *sub
	badSide.Side = order.AnySide
	_, err = ex.WebsocketSubmitOrder(t.Context(), &badSide)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	mockConn.errByMethod[submitBuy] = errors.New("ws buy failed")
	_, err = ex.WebsocketSubmitOrder(t.Context(), sub)
	require.EqualError(t, err, "ws buy failed")
	delete(mockConn.errByMethod, submitBuy)

	mockConn.respByMethod[submitBuy] = []byte(`{"jsonrpc":"2.0","id":"x","result":null}`)
	_, err = ex.WebsocketSubmitOrder(t.Context(), sub)
	require.ErrorIs(t, err, common.ErrNoResponse)

	mockConn.respByMethod[submitBuy] = []byte(`{"jsonrpc":"2.0","id":"x","result":{"order":{"order_id":"buy-order"}}}`)
	resp, err := ex.WebsocketSubmitOrder(t.Context(), sub)
	require.NoError(t, err)
	require.Equal(t, "buy-order", resp.OrderID)
	require.Equal(t, order.New, resp.Status)

	sell := *sub
	sell.Side = order.Sell
	mockConn.respByMethod[submitSell] = []byte(`{"jsonrpc":"2.0","id":"x","result":{"order":{"order_id":"sell-order"}}}`)
	resp, err = ex.WebsocketSubmitOrder(t.Context(), &sell)
	require.NoError(t, err)
	require.Equal(t, "sell-order", resp.OrderID)
}

func TestWebsocketModifyOrderMocked(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	mockConn := newDeribitMockConn()
	connectWithMockedWebsocket(t, ex, mockConn)

	mod := &order.Modify{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
		Amount:    1,
	}

	_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{})
	require.Error(t, err)

	unsupported := *mod
	unsupported.AssetType = asset.Binary
	_, err = ex.WebsocketModifyOrder(t.Context(), &unsupported)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	mockConn.errByMethod[submitEdit] = errors.New("ws edit failed")
	_, err = ex.WebsocketModifyOrder(t.Context(), mod)
	require.EqualError(t, err, "ws edit failed")
	delete(mockConn.errByMethod, submitEdit)

	mockConn.respByMethod[submitEdit] = []byte(`{"jsonrpc":"2.0","id":"x","result":{"order":{"order_id":"edited-order"}}}`)
	resp, err := ex.WebsocketModifyOrder(t.Context(), mod)
	require.NoError(t, err)
	require.Equal(t, "edited-order", resp.OrderID)
}

func TestWebsocketCancelOrderMocked(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex))
	mockConn := newDeribitMockConn()
	connectWithMockedWebsocket(t, ex, mockConn)

	cancel := &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Options,
		Pair:      optionsTradablePair,
	}

	unsupported := *cancel
	unsupported.AssetType = asset.Binary
	err := ex.WebsocketCancelOrder(t.Context(), &unsupported)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	invalid := *cancel
	invalid.OrderID = ""
	err = ex.WebsocketCancelOrder(t.Context(), &invalid)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	mockConn.errByMethod[submitCancel] = errors.New("ws cancel failed")
	err = ex.WebsocketCancelOrder(t.Context(), cancel)
	require.EqualError(t, err, "ws cancel failed")
	delete(mockConn.errByMethod, submitCancel)

	mockConn.respByMethod[submitCancel] = []byte(`{"jsonrpc":"2.0","id":"x","result":{"order_id":"1"}}`)
	err = ex.WebsocketCancelOrder(t.Context(), cancel)
	require.NoError(t, err)
}
