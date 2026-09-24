package okx

import (
	"net/http"
	"sync"
	"testing"
	"uuid"

	gorillaws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := new(Exchange).MessageID()
	require.Len(t, id, 32, "Must return the correct length of message id")
	u, err := uuid.Parse(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	require.Equal(t, byte(7), u[6]>>4, "MessageID must return a V7 uuid") // RFC 9562 version nibble
	require.Len(t, u.String(), 36, "UUID v7 string representation must be 36 characters long")
}

// 7696807	       153.1 ns/op	      48 B/op	       2 allocs/op
func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

// TestSpreadOrdersUseRESTWithAuthenticatedWebsocket guards the spread order
// flows: OKX accepts spread operations only on its business websocket, and the
// WS spread helpers send on the private one, so SubmitOrder, ModifyOrder and
// CancelOrder must place, amend and cancel spread orders over REST even when
// an authenticated websocket is available.
func TestSpreadOrdersUseRESTWithAuthenticatedWebsocket(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var wsOps []string
	var restPaths []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gorillaws.Conn) error {
				var req struct {
					ID        string `json:"id"`
					Operation string `json:"op"`
				}
				if err := json.Unmarshal(msg, &req); err != nil {
					return err
				}
				mu.Lock()
				wsOps = append(wsOps, req.Operation)
				mu.Unlock()
				// The error live OKX returns when a spread operation reaches the
				// private websocket connection.
				return c.WriteMessage(gorillaws.TextMessage,
					[]byte(`{"id":"`+req.ID+`","op":"`+req.Operation+`","code":"60028","msg":"The current operation is not supported by this URL. Please use the correct WebSocket URL for the operation."}`))
			})
			return
		}
		mu.Lock()
		restPaths = append(restPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":{"sCode":"0","sMsg":"","ordId":"SPRD-1","clOrdId":""}}`))
	})

	e := testexch.MockWsInstance[Exchange](t, handler)
	// MockWsInstance points RestSpotURL at the server without a trailing
	// slash, which breaks the REST path join, so every endpoint is re-pointed
	// at the mock server; the websocket connections are already established.
	mockURL, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "the mock server URL must be resolvable")
	b := e.GetBase()
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, mockURL+"/"), "Setup must point endpoint %s at the mock server", k)
	}
	// A connected websocket with authenticated endpoints available is what the
	// broken code branches on before sending spread operations to the private
	// connection.
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the mock websocket must report availability for the wrapper")

	sub, err := e.SubmitOrder(t.Context(), &order.Submit{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     100,
	})
	require.NoError(t, err, "SubmitOrder must place a spread order over REST")
	require.NotEmpty(t, sub.OrderID, "SubmitOrder must return the spread order ID")

	_, err = e.ModifyOrder(t.Context(), &order.Modify{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		OrderID:   "SPRD-1",
		Amount:    2,
		Price:     101,
	})
	require.NoError(t, err, "ModifyOrder must amend a spread order over REST")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		OrderID:   "SPRD-1",
	})
	require.NoError(t, err, "CancelOrder must cancel a spread order over REST")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"/sprd/order", "/sprd/amend-order", "/sprd/cancel-order"}, restPaths,
		"spread submit, amend and cancel should reach their REST endpoints")
	assert.Empty(t, wsOps, "no spread operation should be sent over the websocket")
}
