package okx

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestPublicLiquidationSubscriptionCycle(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "setup must succeed")
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: liquidationOrders, Asset: asset.PerpetualSwap}}
	public, err := ex.generateSubscriptions(true)
	require.NoError(t, err, "public subscription expansion must succeed")
	require.Len(t, public, 1, "all enabled swap instruments must share one subscription")
	require.False(t, public[0].Authenticated, "liquidation observations must use public data")
	require.JSONEq(t, `{"channel":"liquidation-orders","instType":"SWAP"}`, public[0].QualifiedChannel, "subscription must match the documented asset scope")
	private, err := ex.generateSubscriptions(false)
	require.NoError(t, err, "private subscription expansion must succeed")
	require.Empty(t, private, "liquidation observations must not require credentials")

	done := make(chan struct{})
	server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		assert.Equal(t, "/ws/v5/public", r.URL.Path, "liquidations should use the public socket")
		conn, err := (&gws.Upgrader{}).Upgrade(w, r, nil)
		if !assert.NoError(t, err, "local websocket upgrade should succeed") {
			return
		}
		defer conn.Close()
		assert.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Second)), "mock reads should be bounded")
		for _, operation := range []string{operationSubscribe, operationUnsubscribe} {
			_, payload, err := conn.ReadMessage()
			if !assert.NoError(t, err, "subscription message should arrive") {
				return
			}
			assert.JSONEq(t, fmt.Sprintf(`{"op":%q,"args":[{"channel":"liquidation-orders","instType":"SWAP"}]}`, operation), string(payload), "wire request should contain only documented arguments")
			if operation == operationSubscribe {
				assert.NoError(t, conn.WriteMessage(gws.TextMessage, []byte(`{"arg":{"channel":"liquidation-orders","instType":"SWAP"},"data":[{"instId":"BTC-USDT-SWAP","instType":"SWAP","details":[{"bkPx":"60000","posSide":"short","side":"buy","sz":"13","ts":"1700000000000"}]}]}`)), "liquidation push should write")
			}
		}
	}))
	conn, err := ex.Websocket.CreateTestConnection(nil)
	require.NoError(t, err, "public connection must be available without authentication")
	require.Contains(t, conn.GetURL(), "/ws/v5/public", "default public connection must use the documented route")
	conn.SetURL("ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v5/public")
	require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "local public socket must connect")
	t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "local socket should close") })
	require.NoError(t, ex.Subscribe(t.Context(), conn, public), "asset subscription must reach the public socket")
	response := conn.ReadMessage()
	require.Equal(t, gws.TextMessage, response.Type, "public socket must return a text push")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, response.Raw), "liquidation push must decode")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "one typed envelope must be relayed")
	event, ok := (<-ex.Websocket.DataHandler.C).Data.(*WsLiquidationOrders)
	require.True(t, ok, "public liquidation envelope must preserve its type")
	require.Equal(t, "SWAP", event.Argument.InstrumentType, "push must retain instrument scope")
	require.Len(t, event.Data, 1, "push must retain the market row")
	require.Equal(t, "BTC-USDT-SWAP", event.Data[0].InstrumentID, "instrument identity must survive decoding")
	require.Len(t, event.Data[0].Details, 1, "push must retain the liquidation detail")
	assert.Equal(t, 13.0, event.Data[0].Details[0].QuantityOfLiquidation.Float64(), "contract quantity should remain unconverted")
	assert.Equal(t, int64(1700000000000), event.Data[0].Details[0].Timestamp.Time().UnixMilli(), "liquidation time should survive decoding")
	require.NoError(t, ex.Unsubscribe(t.Context(), conn, public), "asset unsubscription must reach the public socket")
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "mock subscription cycle must complete")
	}
}
