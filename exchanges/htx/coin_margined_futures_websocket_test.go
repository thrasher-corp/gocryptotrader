package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWSHandleCoinMarginedPrivateMessage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		channel  string
		expected any
		raw      string
	}{
		{name: "orders", channel: subscription.MyOrdersChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6,"order_id":1,"volume":2,"trade_volume":1}`},
		{name: "matches", channel: subscription.MyTradesChannel, expected: &order.Detail{}, raw: `{"contract_code":"BTC-USD","direction":"buy","order_type":1,"status":6,"order_id":1,"volume":2,"trade_volume":1}`},
		{name: "accounts", channel: subscription.MyAccountChannel, expected: []accounts.Change{}, raw: `{"ts":1603878749908,"data":[{"margin_asset":"BTC","margin_balance":2,"margin_frozen":1,"margin_available":1}]}`},
		{name: "positions", channel: wsPositionsChannel, expected: &SwapWsSubPositionUpdates{}},
		{name: "trigger orders", channel: wsTriggerOrdersChannel, expected: &SwapWsSubTriggerOrderUpdates{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			sub := &subscription.Subscription{Asset: asset.CoinMarginedFutures, Channel: tt.channel, Authenticated: true}
			raw := []byte(tt.raw)
			if len(raw) == 0 {
				raw = []byte(`{"op":"notify","topic":"private.*","ts":1603878749908,"symbol":"BTC","contract_code":"BTC-USD","data":[]}`)
			}
			require.NoError(t, h.wsHandleCoinMarginedPrivateMessage(t.Context(), sub, raw), "private coin-margined notification must be decoded")
			message := <-h.Websocket.DataHandler.C
			assert.IsType(t, tt.expected, message.Data, "notification should use its dedicated response type")
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	err := h.wsHandleCoinMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: "unsupported"}, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "unknown private coin-margined channels must be rejected")
	err = h.wsHandleCoinMarginedPrivateMessage(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel}, []byte(`{`))
	require.Error(t, err, "malformed private coin-margined notifications must be rejected")
	err = h.wsHandleCoinMarginedPrivateMessage(t.Context(), nil, []byte(`{}`))
	require.ErrorIs(t, err, common.ErrNilPointer, "nil private coin-margined subscriptions must be rejected")
}
