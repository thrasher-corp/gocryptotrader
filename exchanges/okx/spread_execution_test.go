package okx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestSpreadCumulativeExecution(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
	require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "passphrase"})
	row := func(total, last int, state string) string {
		return fmt.Sprintf(`{"sprdId":"BTC-USDT_BTC-USDT-SWAP","instId":"BTC-USDT_BTC-USDT-SWAP","ordId":"1","side":"buy","ordType":"limit","state":%q,"sz":"3","px":"100","avgPx":"99","fillSz":"%d","accFillSz":"%d","pendingFillSz":"%d","cTime":"1735720637000","uTime":"1735720638000"}`, state, last, total, 3-total)
	}
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprintf(w, `{"code":"0","msg":"","data":[%s]}`, row(2, 1, "partially_filled"))
		assert.NoError(t, err, "mock spread order should be written")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must be installed")
	info, err := ex.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spread)
	require.NoError(t, err, "spread detail lookup must succeed")
	assert.Equal(t, 2.0, info.ExecutedAmount, "detail should use cumulative quantity rather than the last fill")
	assert.Equal(t, 1.0, info.RemainingAmount, "detail should use cumulative quantity to derive the remainder")
	req := &order.MultiOrderRequest{AssetType: asset.Spread, Type: order.Limit, Side: order.AnySide}
	active, err := ex.GetActiveOrders(t.Context(), req)
	require.NoError(t, err, "spread active lookup must succeed")
	require.Len(t, active, 1, "active lookup must retain the mocked order")
	assert.Equal(t, 2.0, active[0].ExecutedAmount, "active order should use cumulative fills")
	assert.Equal(t, 1.0, active[0].RemainingAmount, "active order should derive its remainder from cumulative fills")
	history, err := ex.GetOrderHistory(t.Context(), req)
	require.NoError(t, err, "spread history lookup must succeed")
	require.Len(t, history, 1, "history lookup must retain the mocked order")
	assert.Equal(t, 2.0, history[0].ExecutedAmount, "history should use cumulative fills")
	assert.Equal(t, 1.0, history[0].RemainingAmount, "history should retain the reported pending fill quantity")
	stored := order.Detail{}
	for _, step := range []struct {
		total, last int
		state       string
		status      order.Status
	}{{2, 2, "partially_filled", order.PartiallyFilled}, {3, 1, "filled", order.Filled}} {
		payload := fmt.Appendf(nil, `{"arg":{"sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[%s]}`, row(step.total, step.last, step.state))
		require.NoError(t, ex.wsProcessSpreadOrders(t.Context(), payload), "spread push must succeed")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "spread push must emit one batch")
		got, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
		require.True(t, ok, "spread push must emit order details")
		require.Len(t, got, 1, "spread batch must contain one order")
		assert.Equal(t, float64(step.total), got[0].ExecutedAmount, "push should retain cumulative execution")
		assert.Equal(t, float64(3-step.total), got[0].RemainingAmount, "push should derive remaining quantity from cumulative execution")
		require.NoError(t, stored.UpdateOrderFromDetail(&got[0]), "spread lifecycle must merge")
		assert.Equal(t, step.status, stored.Status, "a smaller final fill should finish the stored order")
		assert.Equal(t, float64(step.total), stored.ExecutedAmount, "stored execution should accumulate each fill")
	}
}
